package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type DockerContainerTransferRequest struct {
	ConnectionID       uint   `json:"connectionId"`
	TargetAssetType    string `json:"targetAssetType" binding:"required"`
	TargetAssetID      uint   `json:"targetAssetId" binding:"required"`
	TargetConnectionID uint   `json:"targetConnectionId" binding:"required"`
	Name               string `json:"name" binding:"required"`
	Migrate            bool   `json:"migrate"`
	Start              bool   `json:"start"`
	PreservePorts      bool   `json:"preservePorts"`
	ConfirmName        string `json:"confirmName" binding:"required"`
}

type DockerContainerTransferResult struct {
	ContainerID string   `json:"containerId"`
	Name        string   `json:"name"`
	Migrated    bool     `json:"migrated"`
	Volumes     []string `json:"volumes"`
}

type dockerTransferInspect struct {
	ID         string `json:"Id"`
	Name       string
	Config     map[string]any
	HostConfig map[string]any
	State      struct {
		Status  string
		Running bool
		Paused  bool
		Health  *struct{ Status string }
	}
	Mounts []struct {
		Type, Name, Source, Destination string
		RW                              bool
	}
	NetworkSettings struct {
		Networks map[string]struct {
			NetworkID  string
			Aliases    []string
			IPAMConfig any
		}
	}
}

// Keep the transport behind a small interface so failure recovery can be tested without live assets.
type dockerTransferNode interface {
	run(context.Context, string) (string, error)
	create(context.Context, string, map[string]any) (string, error)
	stream(context.Context, string, dockerTransferNode, string) error
}

type dockerSSHTransferNode struct {
	client *ssh.Client
	socket string
}

func (n *dockerSSHTransferNode) run(ctx context.Context, command string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	session, err := n.client.NewSession()
	if err != nil {
		return "", errors.New("无法建立 Docker SSH 命令会话")
	}
	defer session.Close()
	out, stderr := &dockerBoundedBuffer{max: dockerOutputLimit}, &dockerBoundedBuffer{max: 64 << 10}
	session.Stdout, session.Stderr = out, stderr
	stop := context.AfterFunc(ctx, func() { _ = session.Close() })
	defer stop()
	if err := session.Run(command); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		message := strings.TrimSpace(stderr.String())
		if len(message) > 300 {
			message = message[:300]
		}
		return "", fmt.Errorf("Docker 命令失败：%s", message)
	}
	return out.String(), nil
}

func (n *dockerSSHTransferNode) create(ctx context.Context, name string, body map[string]any) (string, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	session, err := n.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	stop := context.AfterFunc(ctx, func() { _ = session.Close() })
	defer stop()
	session.Stdin = strings.NewReader(string(payload))
	out, stderr := &dockerBoundedBuffer{max: dockerOutputLimit}, &dockerBoundedBuffer{max: 64 << 10}
	session.Stdout, session.Stderr = out, stderr
	// Configuration (including environment secrets) travels on stdin, never in process arguments.
	command := "curl --silent --show-error --max-time 60 --unix-socket " + dockerShellQuote(n.socket) +
		" -X POST -H 'Content-Type: application/json' --data-binary @- " + dockerShellQuote("http://localhost/containers/create?name="+url.QueryEscape(name))
	if err = session.Run(command); err != nil {
		return "", errors.New("目标 Docker API 请求失败，请检查 curl 和 Docker socket 权限")
	}
	var response struct {
		ID      string `json:"Id"`
		Message string `json:"message"`
	}
	if json.Unmarshal(out.Bytes(), &response) != nil {
		return "", errors.New("目标 Docker API 返回无效响应")
	}
	if !dockerObjectPattern.MatchString(response.ID) {
		// Do not reflect arbitrary daemon messages containing container configuration into logs.
		return "", errors.New("目标容器创建失败，请检查名称冲突、网络、端口和运行时配置")
	}
	return response.ID, nil
}

func (n *dockerSSHTransferNode) stream(ctx context.Context, sourceCommand string, target dockerTransferNode, targetCommand string) error {
	dst, ok := target.(*dockerSSHTransferNode)
	if !ok {
		return errors.New("不支持的 Docker 传输通道")
	}
	source, err := n.client.NewSession()
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := dst.client.NewSession()
	if err != nil {
		return err
	}
	defer destination.Close()
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	source.Stdout, destination.Stdin = writer, reader
	source.Stderr = &dockerBoundedBuffer{max: 64 << 10}
	destination.Stderr = &dockerBoundedBuffer{max: 64 << 10}
	destination.Stdout = &dockerBoundedBuffer{max: dockerOutputLimit}
	stop := context.AfterFunc(ctx, func() {
		_ = source.Close()
		_ = destination.Close()
		_ = reader.CloseWithError(ctx.Err())
		_ = writer.CloseWithError(ctx.Err())
	})
	defer stop()
	if err = destination.Start(targetCommand); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { err := destination.Wait(); _ = reader.CloseWithError(err); done <- err }()
	sourceErr := source.Run(sourceCommand)
	_ = writer.CloseWithError(sourceErr)
	destinationErr := <-done
	if sourceErr != nil || destinationErr != nil {
		return errors.New("Docker 数据流传输失败，请检查节点空间和挂载目录（文件型绑定挂载不支持复制）")
	}
	return nil
}

func (s *DockerOperationService) transferNode(ctx context.Context, kind string, id, connection uint) (*dockerSSHTransferNode, error) {
	client, err := ResolveAssetSSH(s.db, kind, id, connection)
	if err != nil {
		return nil, errors.New("SSH 连接失败，请检查连接档案、凭据和主机指纹")
	}
	node := &dockerSSHTransferNode{client: client}
	endpoint, err := node.run(ctx, `command -v curl >/dev/null && if [ -n "${DOCKER_HOST:-}" ]; then printf '%s' "$DOCKER_HOST"; else docker context inspect --format '{{.Endpoints.docker.Host}}'; fi`)
	endpoint = strings.TrimSpace(endpoint)
	if err != nil || !strings.HasPrefix(endpoint, "unix:///") || strings.ContainsAny(endpoint, "\r\n\x00") {
		client.Close()
		return nil, errors.New("容器传输需要节点安装 curl，并使用本机 Unix socket Docker context")
	}
	node.socket = strings.TrimPrefix(endpoint, "unix://")
	return node, nil
}

var dockerTransferLocks sync.Map

func (s *DockerOperationService) ContainerTransfer(ctx context.Context, kind string, id uint, container string, req DockerContainerTransferRequest) (*DockerContainerTransferResult, error) {
	if err := validateDockerTransfer(container, req); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	source, err := s.transferNode(ctx, kind, id, req.ConnectionID)
	if err != nil {
		return nil, err
	}
	defer source.client.Close()
	target, err := s.transferNode(ctx, req.TargetAssetType, req.TargetAssetID, req.TargetConnectionID)
	if err != nil {
		return nil, err
	}
	defer target.client.Close()
	// Each run/stream observes cancellation. Connections remain available for detached rollback.
	return transferDockerContainer(ctx, source, target, container, req)
}

func validateDockerTransfer(container string, req DockerContainerTransferRequest) error {
	if validateDockerObject(container) != nil || validateDockerObject(req.Name) != nil {
		return errors.New("源容器或目标容器名称无效")
	}
	if (req.TargetAssetType != "host" && req.TargetAssetType != "physical") || req.TargetAssetID == 0 || req.TargetConnectionID == 0 {
		return errors.New("请选择目标节点和 SSH 连接档案")
	}
	if req.ConfirmName == "" {
		return errors.New("请输入源容器名称确认操作")
	}
	return nil
}

func inspectTransferContainer(ctx context.Context, node dockerTransferNode, container string) (*dockerTransferInspect, error) {
	raw, err := node.run(ctx, "docker inspect --type container "+dockerShellQuote(container))
	if err != nil {
		return nil, err
	}
	var items []dockerTransferInspect
	if json.Unmarshal([]byte(raw), &items) != nil || len(items) != 1 || items[0].Config == nil || items[0].HostConfig == nil || validateDockerObject(items[0].ID) != nil {
		return nil, errors.New("容器配置解析失败")
	}
	return &items[0], nil
}

func transferContainerBody(source *dockerTransferInspect, req DockerContainerTransferRequest, image, suffix string) (map[string]any, []string, error) {
	if source.HostConfig["AutoRemove"] == true {
		return nil, nil, errors.New("自动删除容器不支持复制或迁移，请先关闭 AutoRemove")
	}
	if source.State.Paused || (source.State.Status != "running" && source.State.Status != "exited" && source.State.Status != "created") {
		return nil, nil, errors.New("仅支持运行中、已停止或已创建的容器")
	}
	for _, field := range []string{"NetworkMode", "PidMode", "IpcMode", "UTSMode"} {
		if strings.HasPrefix(fmt.Sprint(source.HostConfig[field]), "container:") {
			return nil, nil, errors.New("共享其他容器命名空间的容器不支持独立传输")
		}
	}
	for _, field := range []string{"Devices", "DeviceRequests", "DeviceCgroupRules"} {
		if value, ok := source.HostConfig[field].([]any); ok && len(value) > 0 {
			return nil, nil, errors.New("设备或 GPU 绑定容器需要先在目标节点配置设备")
		}
	}
	bodyBytes, _ := json.Marshal(source.Config)
	body := map[string]any{}
	_ = json.Unmarshal(bodyBytes, &body)
	hostBytes, _ := json.Marshal(source.HostConfig)
	host := map[string]any{}
	_ = json.Unmarshal(hostBytes, &host)
	body["Image"] = image
	if body["Labels"] == nil {
		body["Labels"] = map[string]any{}
	}
	body["Labels"].(map[string]any)["autoops.transfer"] = suffix
	// An independent container must not be claimed by the source Compose/Swarm controller.
	if labels, ok := body["Labels"].(map[string]any); ok {
		for key := range labels {
			if strings.HasPrefix(key, "com.docker.compose.") || strings.HasPrefix(key, "com.docker.swarm.") {
				delete(labels, key)
			}
		}
	}
	if body["Hostname"] == source.ID[:min(12, len(source.ID))] {
		body["Hostname"] = ""
	}
	body["Volumes"] = map[string]any{}
	host["Binds"], host["VolumesFrom"], host["ContainerIDFile"] = nil, nil, ""
	host["AutoRemove"] = false
	mounts := []map[string]any{}
	volumes := []string{}
	volumeSources := map[string]string{}
	for i, mount := range source.Mounts {
		if mount.Type == "tmpfs" {
			continue
		}
		if mount.Type != "volume" && mount.Type != "bind" {
			return nil, nil, errors.New("不支持的挂载类型：" + mount.Type)
		}
		if !strings.HasPrefix(mount.Destination, "/") || mount.Destination == "/" || strings.ContainsRune(mount.Destination, 0) {
			return nil, nil, errors.New("挂载目录无效")
		}
		key := mount.Type + ":" + mount.Source
		if mount.Source == "" {
			key = mount.Type + ":" + mount.Name
		}
		name := volumeSources[key]
		if name == "" {
			name = fmt.Sprintf("autoops-%s-%d", suffix, i)
			volumeSources[key] = name
			volumes = append(volumes, name)
		}
		mounts = append(mounts, map[string]any{"Type": "volume", "Source": name, "Target": mount.Destination, "ReadOnly": !mount.RW, "VolumeOptions": map[string]any{"NoCopy": true}})
	}
	// Preserve tmpfs settings, including mounts configured with --mount.
	if original, ok := host["Mounts"].([]any); ok {
		for _, item := range original {
			if mount, ok := item.(map[string]any); ok && mount["Type"] == "tmpfs" {
				mounts = append(mounts, mount)
			}
		}
	}
	host["Mounts"] = mounts
	if !req.PreservePorts {
		if ports, ok := host["PortBindings"].(map[string]any); ok {
			for _, bindings := range ports {
				if list, ok := bindings.([]any); ok {
					for _, entry := range list {
						if binding, ok := entry.(map[string]any); ok {
							binding["HostPort"], binding["HostIp"] = "", ""
						}
					}
				}
			}
		}
	}
	endpoints := map[string]any{}
	for name, endpoint := range source.NetworkSettings.Networks {
		if name == "host" || name == "none" || name == "bridge" {
			continue
		}
		aliases := []string{}
		for _, alias := range endpoint.Aliases {
			if alias != strings.TrimPrefix(source.Name, "/") && alias != source.ID && alias != source.ID[:min(12, len(source.ID))] {
				aliases = append(aliases, alias)
			}
		}
		endpoints[name] = map[string]any{"Aliases": aliases}
		if host["NetworkMode"] == endpoint.NetworkID {
			host["NetworkMode"] = name
		}
	}
	body["HostConfig"] = host
	if len(endpoints) > 0 {
		body["NetworkingConfig"] = map[string]any{"EndpointsConfig": endpoints}
	}
	return body, volumes, nil
}

func transferDockerContainer(ctx context.Context, source, target dockerTransferNode, container string, req DockerContainerTransferRequest) (result *DockerContainerTransferResult, err error) {
	original, err := inspectTransferContainer(ctx, source, container)
	if err != nil {
		return nil, err
	}
	name := strings.TrimPrefix(original.Name, "/")
	if req.ConfirmName != name {
		return nil, errors.New("请输入完整源容器名称确认操作")
	}
	if req.Migrate && original.State.Running && !req.Start {
		return nil, errors.New("运行中容器迁移必须启动并验证目标容器")
	}
	suffix := dockerRandomSuffix()
	snapshot := "autoops-transfer:" + suffix
	body, volumes, err := transferContainerBody(original, req, snapshot, suffix)
	if err != nil {
		return nil, err
	}
	sourceInfo, err := source.run(ctx, "docker info --format '{{.ID}}|{{.OSType}}|{{.Architecture}}'")
	if err != nil {
		return nil, err
	}
	targetInfo, err := target.run(ctx, "docker info --format '{{.ID}}|{{.OSType}}|{{.Architecture}}'")
	if err != nil {
		return nil, err
	}
	srcParts, dstParts := strings.Split(strings.TrimSpace(sourceInfo), "|"), strings.Split(strings.TrimSpace(targetInfo), "|")
	if len(srcParts) != 3 || len(dstParts) != 3 || srcParts[0] == "" || dstParts[0] == "" {
		return nil, errors.New("无法确认 Docker 节点标识和架构")
	}
	same := srcParts[0] == dstParts[0]
	if srcParts[1] != dstParts[1] || srcParts[2] != dstParts[2] {
		return nil, errors.New("源节点与目标节点操作系统或架构不一致")
	}
	if same && (req.Migrate || req.Name == name) {
		return nil, errors.New("迁移必须选择其他 Docker 节点，同节点复制必须使用新名称")
	}
	if same && (req.PreservePorts || original.HostConfig["NetworkMode"] == "host") && req.Start && original.State.Running {
		return nil, errors.New("同节点复制启动时请使用随机宿主机端口，host 网络容器请先选择不启动")
	}
	// Lock actual daemon/container identities, including aliases registered as multiple assets.
	for _, key := range []string{srcParts[0] + "/" + original.ID, dstParts[0] + "/name/" + req.Name} {
		if _, loaded := dockerTransferLocks.LoadOrStore(key, true); loaded {
			return nil, errors.New("此容器或目标名称正在传输，请稍后重试")
		}
		defer dockerTransferLocks.Delete(key)
	}
	existing, err := target.run(ctx, "docker ps -a --format '{{.Names}}'")
	if err != nil {
		return nil, err
	}
	for _, existingName := range strings.Split(strings.TrimSpace(existing), "\n") {
		if existingName == req.Name {
			return nil, errors.New("目标节点已存在同名容器")
		}
	}
	for network := range original.NetworkSettings.Networks {
		if _, err = target.run(ctx, "docker network inspect "+dockerShellQuote(network)+" --format '{{.Name}}'"); err != nil {
			return nil, fmt.Errorf("目标节点缺少网络 %s，请先创建对应网络", network)
		}
	}
	stopped, committed, targetLoaded, accepted := false, false, false, false
	restoreSource := true
	targetID := ""
	createdVolumes := []string{}
	// Roll back only resources owned by this transfer. Never delete source volumes.
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		appendFailure := func(label string, cleanupErr error) {
			if cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("%s，请人工检查：%w", label, cleanupErr))
			}
		}
		if !accepted {
			// Also finds containers created just before a lost API response, including helpers.
			owned, listErr := target.run(cleanup, "docker ps -a --no-trunc --filter "+dockerShellQuote("label=autoops.transfer="+suffix)+" --format '{{.ID}}'")
			removed := listErr == nil
			appendFailure("查询待清理目标容器失败", listErr)
			ids := strings.Fields(owned)
			if targetID != "" {
				ids = append(ids, targetID)
			}
			seen := map[string]bool{}
			for _, ownedID := range ids {
				if seen[ownedID] || validateDockerObject(ownedID) != nil {
					continue
				}
				seen[ownedID] = true
				_, e := target.run(cleanup, "docker rm -f -v "+dockerShellQuote(ownedID))
				removed = removed && e == nil
				appendFailure("清理目标容器失败", e)
			}
			if removed {
				for _, volume := range createdVolumes {
					_, e := target.run(cleanup, "docker volume rm "+dockerShellQuote(volume))
					appendFailure("清理目标数据卷失败", e)
				}
			} else if req.Migrate {
				restoreSource = false
				err = errors.Join(err, errors.New("目标状态未确认，源容器保持停止以避免同时写入，请人工核对后恢复"))
			}
			if targetLoaded && !same {
				_, _ = target.run(cleanup, "docker rmi "+dockerShellQuote(snapshot))
			}
		}
		if stopped && restoreSource && (!req.Migrate || !accepted) {
			_, e := source.run(cleanup, "docker start "+dockerShellQuote(original.ID))
			appendFailure("恢复源容器失败", e)
		}
		if committed && (!same || !accepted) {
			_, _ = source.run(cleanup, "docker rmi "+dockerShellQuote(snapshot))
		}
	}()
	if original.State.Running {
		stopped = true
		if _, err = source.run(ctx, "docker stop --time 30 "+dockerShellQuote(original.ID)); err != nil {
			return nil, err
		}
	}
	committed = true
	if _, err = source.run(ctx, "docker commit "+dockerShellQuote(original.ID)+" "+dockerShellQuote(snapshot)); err != nil {
		return nil, err
	}
	if !same {
		targetLoaded = true
		if err = source.stream(ctx, "docker save "+dockerShellQuote(snapshot), target, "docker load"); err != nil {
			return nil, err
		}
	}
	for _, volume := range volumes {
		createdVolumes = append(createdVolumes, volume)
		if _, err = target.run(ctx, "docker volume create "+dockerShellQuote(volume)); err != nil {
			return nil, err
		}
	}
	// A stopped helper gives docker cp a writable mount even when the final mount is read-only.
	for index, volume := range volumes {
		host := body["HostConfig"].(map[string]any)
		mounts := host["Mounts"].([]map[string]any)
		destination := ""
		for _, mount := range mounts {
			if mount["Source"] == volume {
				destination = mount["Target"].(string)
				break
			}
		}
		helper, e := target.create(ctx, "autoops-copy-"+suffix+fmt.Sprint(index), map[string]any{
			"Image": snapshot, "Entrypoint": []string{"/bin/true"}, "Volumes": map[string]any{}, "Labels": map[string]any{"autoops.transfer": suffix},
			"HostConfig": map[string]any{"NetworkMode": "none", "Mounts": []map[string]any{{"Type": "volume", "Source": volume, "Target": "/autoops-data", "VolumeOptions": map[string]any{"NoCopy": true}}}},
		})
		if e != nil {
			return nil, e
		}
		copyErr := source.stream(ctx, "docker cp -a "+dockerShellQuote(original.ID+":"+strings.TrimSuffix(destination, "/")+"/.")+" -", target, "docker cp -a - "+dockerShellQuote(helper+":/autoops-data"))
		cleanup, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		_, removeErr := target.run(cleanup, "docker rm -v "+dockerShellQuote(helper))
		cancel()
		if copyErr != nil || removeErr != nil {
			return nil, errors.Join(copyErr, removeErr)
		}
	}
	// Resume a copied source as soon as its consistent snapshot/data have been transferred.
	if stopped && !req.Migrate {
		if _, err = source.run(ctx, "docker start "+dockerShellQuote(original.ID)); err != nil {
			return nil, err
		}
		stopped = false
	}
	targetID, err = target.create(ctx, req.Name, body)
	if err != nil {
		return nil, err
	}
	result = &DockerContainerTransferResult{ContainerID: targetID, Name: req.Name, Volumes: volumes}
	if req.Start {
		if _, err = target.run(ctx, "docker start "+dockerShellQuote(targetID)); err != nil {
			return nil, err
		}
		if err = waitTransferContainer(ctx, target, targetID); err != nil {
			return nil, err
		}
	} else {
		if _, err = inspectTransferContainer(ctx, target, targetID); err != nil {
			return nil, err
		}
	}
	accepted = true
	if req.Migrate {
		if _, err = source.run(ctx, "docker rm "+dockerShellQuote(original.ID)); err != nil {
			return result, errors.New("目标容器已就绪，但源容器删除失败；源容器保持停止，请人工核对后清理")
		}
		result.Migrated = true
	}
	return result, nil
}

func waitTransferContainer(ctx context.Context, node dockerTransferNode, id string) error {
	deadline := time.NewTimer(60 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errors.New("目标容器健康检查超时，已保留源容器")
		case <-ticker.C:
			item, err := inspectTransferContainer(ctx, node, id)
			if err != nil {
				return err
			}
			if !item.State.Running {
				return errors.New("目标容器启动后退出，已保留源容器")
			}
			if item.State.Health == nil || item.State.Health.Status == "healthy" {
				return nil
			}
			if item.State.Health.Status == "unhealthy" {
				return errors.New("目标容器健康检查失败，已保留源容器")
			}
		}
	}
}
