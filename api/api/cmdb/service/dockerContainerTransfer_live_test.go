package service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// An opt-in integration test using two local, distinct Docker daemons. Only
// uniquely named test resources are changed; SSH uses a loopback test server.
func TestDockerContainerTransferLive(t *testing.T) {
	sourceSocket, targetSocket := os.Getenv("CMDB_TRANSFER_SOURCE_SOCKET"), os.Getenv("CMDB_TRANSFER_TARGET_SOCKET")
	if sourceSocket == "" || targetSocket == "" {
		t.Skip("set CMDB_TRANSFER_SOURCE_SOCKET and CMDB_TRANSFER_TARGET_SOCKET to test two local Docker daemons")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	source := dockerTransferLiveSSH(t, sourceSocket)
	target := dockerTransferLiveSSH(t, targetSocket)
	sourceInfo, err := source.run(ctx, "docker info --format '{{.ID}}'")
	if err != nil {
		t.Fatal(err)
	}
	targetInfo, err := target.run(ctx, "docker info --format '{{.ID}}'")
	if err != nil {
		t.Fatal(err)
	}
	if sourceInfo == targetInfo {
		t.Fatal("test requires distinct Docker daemons")
	}
	suffix := dockerRandomSuffix()
	sourceName, copyName, migrateName := "autoops-transfer-test-"+suffix, "autoops-copy-test-"+suffix, "autoops-migrate-test-"+suffix
	sourceVolume := "autoops-test-data-" + suffix
	targets := []*DockerContainerTransferResult{}
	t.Cleanup(func() {
		clean, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_, _ = source.run(clean, "docker rm -f -v "+dockerShellQuote(sourceName))
		_, _ = source.run(clean, "docker volume rm "+dockerShellQuote(sourceVolume))
		for _, result := range targets {
			inspected, _ := target.run(clean, "docker inspect --format '{{.Config.Image}}' "+dockerShellQuote(result.ContainerID))
			_, _ = target.run(clean, "docker rm -f -v "+dockerShellQuote(result.ContainerID))
			for _, volume := range result.Volumes {
				_, _ = target.run(clean, "docker volume rm "+dockerShellQuote(volume))
			}
			if strings.HasPrefix(strings.TrimSpace(inspected), "autoops-transfer:") {
				_, _ = target.run(clean, "docker rmi "+dockerShellQuote(strings.TrimSpace(inspected)))
			}
		}
	})
	bind := t.TempDir()
	if err := os.WriteFile(filepath.Join(bind, "readonly.txt"), []byte("bound-data"), 0600); err != nil {
		t.Fatal(err)
	}
	image := os.Getenv("CMDB_TRANSFER_TEST_IMAGE")
	if image == "" {
		image = "python:3.12-alpine"
	}
	command := "docker run -d --network none --name " + dockerShellQuote(sourceName) + " --restart unless-stopped --memory 128m -e TRANSFER_SENTINEL=preserved --health-cmd 'python -c \"exit(0)\"' --health-interval 1s --mount " + dockerShellQuote("type=volume,src="+sourceVolume+",dst=/data") + " --mount " + dockerShellQuote("type=bind,src="+bind+",dst=/readonly,readonly") + " " + dockerShellQuote(image) + " python -c 'import time;time.sleep(600)'"
	if _, err := source.run(ctx, command); err != nil {
		t.Fatal(err)
	}
	if _, err := source.run(ctx, "docker exec "+dockerShellQuote(sourceName)+" python -c "+dockerShellQuote("open('/data/data.txt','w').write('volume-data');open('/root/marker','w').write('rootfs-data')")); err != nil {
		t.Fatal(err)
	}
	req := DockerContainerTransferRequest{Name: copyName, ConfirmName: sourceName, Start: true}
	copied, err := transferDockerContainer(ctx, source, target, sourceName, req)
	if err != nil {
		t.Fatal(err)
	}
	targets = append(targets, copied)
	if _, err := source.run(ctx, "docker exec "+dockerShellQuote(sourceName)+" python -c 'exit(0)'"); err != nil {
		t.Fatal("copied source was not resumed", err)
	}
	verify := func(result *DockerContainerTransferResult) {
		t.Helper()
		script := "import os; assert os.getenv('TRANSFER_SENTINEL')=='preserved'; assert open('/data/data.txt').read()=='volume-data'; assert open('/readonly/readonly.txt').read()=='bound-data'; assert open('/root/marker').read()=='rootfs-data'"
		if _, err := target.run(ctx, "docker exec "+dockerShellQuote(result.ContainerID)+" python -c "+dockerShellQuote(script)); err != nil {
			t.Fatal("copied data/config mismatch", err)
		}
		inspect, err := inspectTransferContainer(ctx, target, result.ContainerID)
		if err != nil {
			t.Fatal(err)
		}
		if inspect.HostConfig["Memory"] != float64(128<<20) {
			t.Fatal("memory limit not preserved")
		}
		readonly := false
		for _, mount := range inspect.Mounts {
			if mount.Destination == "/readonly" {
				readonly = !mount.RW && mount.Type == "volume"
			}
		}
		if !readonly {
			t.Fatal("read-only mount not restored as independent volume")
		}
	}
	verify(copied)
	req.Name, req.Migrate = migrateName, true
	migrated, err := transferDockerContainer(ctx, source, target, sourceName, req)
	if err != nil {
		t.Fatal(err)
	}
	targets = append(targets, migrated)
	if !migrated.Migrated {
		t.Fatal("migration not completed")
	}
	verify(migrated)
	if _, err := inspectTransferContainer(ctx, source, sourceName); err == nil {
		t.Fatal("source still exists after migration")
	}
	if _, err := source.run(ctx, "docker volume inspect "+dockerShellQuote(sourceVolume)); err != nil {
		t.Fatal("source data volume was deleted")
	}
}

func dockerTransferLiveSSH(t *testing.T, socket string) *dockerSSHTransferNode {
	t.Helper()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(private)
	if err != nil {
		t.Fatal(err)
	}
	password := dockerRandomSuffix()
	config := &ssh.ServerConfig{PasswordCallback: func(_ ssh.ConnMetadata, secret []byte) (*ssh.Permissions, error) {
		if string(secret) != password {
			return nil, fmt.Errorf("denied")
		}
		return nil, nil
	}}
	config.AddHostKey(signer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		server, channels, requests, err := ssh.NewServerConn(conn, config)
		if err != nil {
			conn.Close()
			return
		}
		defer server.Close()
		go ssh.DiscardRequests(requests)
		for incoming := range channels {
			if incoming.ChannelType() != "session" {
				_ = incoming.Reject(ssh.UnknownChannelType, "session only")
				continue
			}
			channel, requests, err := incoming.Accept()
			if err != nil {
				continue
			}
			go func() {
				defer channel.Close()
				for request := range requests {
					if request.Type != "exec" {
						_ = request.Reply(false, nil)
						continue
					}
					var payload struct{ Command string }
					if ssh.Unmarshal(request.Payload, &payload) != nil {
						_ = request.Reply(false, nil)
						return
					}
					_ = request.Reply(true, nil)
					command := exec.Command("sh", "-c", payload.Command)
					command.Env = append(os.Environ(), "DOCKER_HOST=unix://"+socket)
					command.Stdin, command.Stdout, command.Stderr = channel, channel, channel.Stderr()
					status := uint32(0)
					if command.Run() != nil {
						status = 1
					}
					data := make([]byte, 4)
					binary.BigEndian.PutUint32(data, status)
					_, _ = channel.SendRequest("exit-status", false, data)
					return
				}
			}()
		}
	}()
	client, err := ssh.Dial("tcp", listener.Addr().String(), &ssh.ClientConfig{User: "test", Auth: []ssh.AuthMethod{ssh.Password(password)}, HostKeyCallback: ssh.FixedHostKey(signer.PublicKey()), Timeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return &dockerSSHTransferNode{client: client, socket: socket}
}
