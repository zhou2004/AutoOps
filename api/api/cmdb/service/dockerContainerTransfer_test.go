package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func transferFixture() *dockerTransferInspect {
	var x dockerTransferInspect
	_ = json.Unmarshal([]byte(`{"Id":"abcdef0123456789","Name":"/web","Config":{"Image":"app:old","Env":["PASSWORD=preserved"],"Entrypoint":["/entrypoint"],"User":"1000","WorkingDir":"/app","Healthcheck":{"Test":["CMD","true"]},"Labels":{"custom":"kept","com.docker.compose.project":"old"}},"HostConfig":{"AutoRemove":false,"NetworkMode":"network-id","RestartPolicy":{"Name":"unless-stopped"},"Memory":268435456,"PortBindings":{"80/tcp":[{"HostIp":"127.0.0.1","HostPort":"8080"}]},"Binds":["/data:/data:ro"]},"State":{"Status":"running","Running":true},"Mounts":[{"Type":"bind","Source":"/data","Destination":"/data","RW":false}],"NetworkSettings":{"Networks":{"app-net":{"NetworkID":"network-id","Aliases":["web","abcdef012345","shared-alias"],"IPAMConfig":{"IPv4Address":"172.20.0.9"}}}}}`), &x)
	return &x
}
func TestDockerTransferConfiguration(t *testing.T) {
	source := transferFixture()
	body, volumes, err := transferContainerBody(source, DockerContainerTransferRequest{}, "snapshot:new", "test")
	if err != nil {
		t.Fatal(err)
	}
	if body["Image"] != "snapshot:new" || body["User"] != "1000" || body["WorkingDir"] != "/app" || body["Healthcheck"] == nil || body["Env"].([]any)[0] != "PASSWORD=preserved" {
		t.Fatal("lost container configuration")
	}
	labels := body["Labels"].(map[string]any)
	if labels["custom"] != "kept" || labels["com.docker.compose.project"] != nil {
		t.Fatal("invalid labels")
	}
	host := body["HostConfig"].(map[string]any)
	if host["Memory"] != float64(268435456) || host["NetworkMode"] != "app-net" || host["Binds"] != nil {
		t.Fatal("invalid host settings")
	}
	port := host["PortBindings"].(map[string]any)["80/tcp"].([]any)[0].(map[string]any)
	if port["HostPort"] != "" || port["HostIp"] != "" {
		t.Fatal("random ports not applied")
	}
	mount := host["Mounts"].([]map[string]any)[0]
	if len(volumes) != 1 || mount["Type"] != "volume" || mount["Source"] != volumes[0] || mount["ReadOnly"] != true {
		t.Fatal("data not isolated or read-only lost")
	}
	endpoint := body["NetworkingConfig"].(map[string]any)["EndpointsConfig"].(map[string]any)["app-net"].(map[string]any)
	if endpoint["IPAMConfig"] != nil || len(endpoint["Aliases"].([]string)) != 1 {
		t.Fatal("source network identity leaked")
	}
	if source.Config["Image"] != "app:old" || source.HostConfig["Binds"] == nil {
		t.Fatal("source mutated")
	}
}
func TestDockerTransferValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*dockerTransferInspect)
	}{
		{"auto-remove", func(x *dockerTransferInspect) { x.HostConfig["AutoRemove"] = true }},
		{"shared-network", func(x *dockerTransferInspect) { x.HostConfig["NetworkMode"] = "container:other" }},
		{"shared-pid", func(x *dockerTransferInspect) { x.HostConfig["PidMode"] = "container:other" }},
		{"paused", func(x *dockerTransferInspect) { x.State.Paused = true }},
		{"devices", func(x *dockerTransferInspect) {
			x.HostConfig["Devices"] = []any{map[string]any{"PathOnHost": "/dev/kvm"}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := transferFixture()
			tc.change(x)
			if _, _, err := transferContainerBody(x, DockerContainerTransferRequest{}, "snapshot:new", "test"); err == nil {
				t.Fatal("unsafe config accepted")
			}
		})
	}
	req := DockerContainerTransferRequest{TargetAssetType: "host", TargetAssetID: 2, TargetConnectionID: 3, Name: "copy", ConfirmName: "web"}
	if err := validateDockerTransfer("web", req); err != nil {
		t.Fatal(err)
	}
	req.Name = "web;touch /tmp/injected"
	if validateDockerTransfer("web", req) == nil {
		t.Fatal("shell injection accepted")
	}
}

type fakeDockerTransferNode struct {
	name                                string
	events                              *[]string
	inspect                             *dockerTransferInspect
	fail                                string
	failStream, failCreate, failCleanup bool
	created                             bool
}

func (n *fakeDockerTransferNode) run(_ context.Context, command string) (string, error) {
	*n.events = append(*n.events, n.name+":"+command)
	if n.fail != "" && strings.HasPrefix(command, n.fail) {
		return "", errors.New("injected failure")
	}
	if n.failCleanup && strings.HasPrefix(command, "docker rm -f") {
		return "", errors.New("cleanup failure")
	}
	if strings.HasPrefix(command, "docker ps -a --no-trunc --filter") && n.created {
		return "target123456", nil
	}
	if strings.HasPrefix(command, "docker rm -f") {
		if !n.created {
			return "", errors.New("container already removed")
		}
		n.created = false
	}
	if strings.HasPrefix(command, "docker inspect") {
		raw, _ := json.Marshal([]*dockerTransferInspect{n.inspect})
		return string(raw), nil
	}
	if strings.HasPrefix(command, "docker info") {
		return n.name + "|linux|x86_64", nil
	}
	return "", nil
}
func (n *fakeDockerTransferNode) create(_ context.Context, name string, _ map[string]any) (string, error) {
	*n.events = append(*n.events, n.name+":create "+name)
	if n.failCreate {
		return "", errors.New("create failed")
	}
	n.created = true
	return "target123456", nil
}
func (n *fakeDockerTransferNode) stream(_ context.Context, command string, _ dockerTransferNode, destination string) error {
	*n.events = append(*n.events, n.name+":stream "+command+" -> "+destination)
	if n.failStream {
		return errors.New("stream failed")
	}
	return nil
}
func transferNodes() (*fakeDockerTransferNode, *fakeDockerTransferNode, *[]string) {
	events := []string{}
	source := transferFixture()
	source.Mounts = nil
	target := transferFixture()
	target.ID = "target123456"
	target.Name = "/copy"
	return &fakeDockerTransferNode{name: "source", events: &events, inspect: source}, &fakeDockerTransferNode{name: "target", events: &events, inspect: target}, &events
}
func transferRequest() DockerContainerTransferRequest {
	return DockerContainerTransferRequest{Name: "copy", ConfirmName: "web", Start: true, Migrate: true}
}
func eventIndex(events []string, prefix string) int {
	for i, event := range events {
		if strings.HasPrefix(event, prefix) {
			return i
		}
	}
	return -1
}
func TestDockerTransferFailureRestoresSource(t *testing.T) {
	for _, fail := range []string{"stream", "create", "start", "health"} {
		t.Run(fail, func(t *testing.T) {
			source, target, events := transferNodes()
			switch fail {
			case "stream":
				source.failStream = true
			case "create":
				target.failCreate = true
			case "start":
				target.fail = "docker start"
			case "health":
				target.inspect.State.Running = false
			}
			if _, err := transferDockerContainer(context.Background(), source, target, "web", transferRequest()); err == nil {
				t.Fatal("expected failure")
			}
			if eventIndex(*events, "source:docker rm ") >= 0 {
				t.Fatal("source removed on failure")
			}
			if eventIndex(*events, "source:docker start ") < 0 {
				t.Fatalf("source not restored: %v", *events)
			}
		})
	}
}
func TestDockerTransferMigrationVerification(t *testing.T) {
	source, target, events := transferNodes()
	result, err := transferDockerContainer(context.Background(), source, target, "web", transferRequest())
	if err != nil || !result.Migrated {
		t.Fatalf("migration failed: %v", err)
	}
	checked, deleted := eventIndex(*events, "target:docker inspect"), eventIndex(*events, "source:docker rm ")
	if checked < 0 || deleted <= checked {
		t.Fatalf("unsafe removal order: %v", *events)
	}
	if eventIndex(*events, "source:docker start ") >= 0 {
		t.Fatal("migrated source restarted")
	}
}
func TestDockerTransferCopyKeepsSource(t *testing.T) {
	source, target, events := transferNodes()
	req := transferRequest()
	req.Migrate = false
	req.Start = false
	result, err := transferDockerContainer(context.Background(), source, target, "web", req)
	if err != nil || result.Migrated {
		t.Fatal(err)
	}
	if eventIndex(*events, "source:docker start ") < 0 || eventIndex(*events, "source:docker rm ") >= 0 {
		t.Fatalf("copy damaged source: %v", *events)
	}
}
func TestDockerTransferCleanupFailure(t *testing.T) {
	source, target, events := transferNodes()
	target.fail = "docker start"
	target.failCleanup = true
	_, err := transferDockerContainer(context.Background(), source, target, "web", transferRequest())
	if err == nil || !strings.Contains(err.Error(), "保持停止") {
		t.Fatal("missing recovery guidance")
	}
	if eventIndex(*events, "source:docker start ") >= 0 {
		t.Fatal("source restarted with uncertain target state")
	}
}
func TestDockerTransferPartialMigration(t *testing.T) {
	source, target, events := transferNodes()
	source.fail = "docker rm "
	result, err := transferDockerContainer(context.Background(), source, target, "web", transferRequest())
	if err == nil || result == nil || result.Migrated {
		t.Fatal("partial migration not reported")
	}
	if eventIndex(*events, "target:docker rm -f") >= 0 || eventIndex(*events, "source:docker start ") >= 0 {
		t.Fatal("verified target destroyed or source restarted")
	}
}

func TestDockerTransferSharedMountUsesOneTargetVolume(t *testing.T) {
	source := transferFixture()
	repeated := source.Mounts[0]
	repeated.Destination = "/same-data"
	source.Mounts = append(source.Mounts, repeated)
	body, volumes, err := transferContainerBody(source, DockerContainerTransferRequest{}, "snapshot:new", "shared")
	if err != nil {
		t.Fatal(err)
	}
	mounts := body["HostConfig"].(map[string]any)["Mounts"].([]map[string]any)
	if len(volumes) != 1 || mounts[0]["Source"] != mounts[1]["Source"] {
		t.Fatal("shared mount relationship lost")
	}
}

func TestDockerTransferSameDaemon(t *testing.T) {
	source, target, events := transferNodes()
	target.name = source.name
	req := transferRequest()
	if _, err := transferDockerContainer(context.Background(), source, target, "web", req); err == nil {
		t.Fatal("migration to same daemon accepted")
	}
	if eventIndex(*events, "source:docker stop") >= 0 {
		t.Fatal("source stopped before same-daemon rejection")
	}
	req.Migrate = false
	req.Start = false
	result, err := transferDockerContainer(context.Background(), source, target, "web", req)
	if err != nil || result == nil {
		t.Fatal(err)
	}
	if eventIndex(*events, "source:stream docker save") >= 0 {
		t.Fatal("same-daemon copy unnecessarily streamed image")
	}
	if eventIndex(*events, "source:docker rmi") >= 0 {
		t.Fatal("removed snapshot tag required by the copied container")
	}
}
