package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

const dockerOutputLimit = 4 << 20

var (
	dockerObjectPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$|^[a-fA-F0-9]{12,64}$`)
	dockerImagePattern  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/:@-]{0,254}$`)
	dockerEnvPattern    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,127}$`)
)

type DockerOperationService struct{ db *gorm.DB }

type DockerCreateRequest struct {
	ConnectionID uint                `json:"connectionId"`
	Name         string              `json:"name"`
	Image        string              `json:"image"`
	Restart      string              `json:"restart"`
	Environment  map[string]string   `json:"environment"`
	Ports        []DockerPort        `json:"ports"`
	Volumes      []DockerVolumeMount `json:"volumes"`
	Command      []string            `json:"command"`
}

type DockerPort struct {
	Host      int    `json:"host"`
	Container int    `json:"container"`
	Protocol  string `json:"protocol"`
}

type DockerVolumeMount struct {
	Name          string `json:"name"`
	ContainerPath string `json:"containerPath"`
	ReadOnly      bool   `json:"readOnly"`
}

func NewDockerOperationService(db *gorm.DB) *DockerOperationService {
	return &DockerOperationService{db: db}
}

func dockerShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

type dockerBoundedBuffer struct {
	bytes.Buffer
	max int
}

func (b *dockerBoundedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.max-b.Len() {
		return 0, errors.New("Docker 命令输出超过限制")
	}
	return b.Buffer.Write(p)
}

func runDockerSSH(parent context.Context, client *ssh.Client, command string, maxBytes int, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	stop := context.AfterFunc(ctx, func() { _ = client.Close() })
	defer stop()
	session, err := client.NewSession()
	if err != nil {
		return "", errors.New("无法建立 Docker SSH 命令会话")
	}
	defer session.Close()
	stdout, stderr := &dockerBoundedBuffer{max: maxBytes}, &dockerBoundedBuffer{max: 64 << 10}
	session.Stdout, session.Stderr = stdout, stderr
	done := make(chan error, 1)
	go func() { done <- session.Run(command) }()
	select {
	case <-ctx.Done():
		_ = client.Close()
		<-done
		return "", errors.New("Docker 操作超时或已取消")
	case err := <-done:
		if err != nil {
			message := strings.TrimSpace(stderr.String())
			if len(message) > 300 {
				message = message[:300]
			}
			if message == "" {
				message = "请检查 Docker 服务和当前 SSH 用户权限"
			}
			return "", fmt.Errorf("Docker 命令失败：%s", message)
		}
		return stdout.String(), nil
	}
}

func (s *DockerOperationService) run(ctx context.Context, assetType string, assetID, connectionID uint, command string, maxBytes int) (string, error) {
	return s.runWithTimeout(ctx, assetType, assetID, connectionID, command, maxBytes, 45*time.Second)
}

// runLong 用于拉取/导入/导出/推送等耗时操作。
func (s *DockerOperationService) runLong(ctx context.Context, assetType string, assetID, connectionID uint, command string, maxBytes int) (string, error) {
	return s.runWithTimeout(ctx, assetType, assetID, connectionID, command, maxBytes, 15*time.Minute)
}

func (s *DockerOperationService) runWithTimeout(ctx context.Context, assetType string, assetID, connectionID uint, command string, maxBytes int, timeout time.Duration) (string, error) {
	if assetType != "host" && assetType != "physical" {
		return "", errors.New("Docker 仅支持主机或物理机资产")
	}
	client, err := ResolveAssetSSH(s.db, assetType, assetID, connectionID)
	if err != nil {
		return "", errors.New("SSH 连接失败，请检查连接档案、凭据和主机指纹")
	}
	defer client.Close()
	return runDockerSSH(ctx, client, command, maxBytes, timeout)
}

func parseDockerLines(raw string) ([]map[string]any, error) {
	items := []map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var item map[string]any
		if json.Unmarshal([]byte(line), &item) != nil {
			return nil, errors.New("Docker 返回了无法解析的数据")
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *DockerOperationService) Info(ctx context.Context, assetType string, assetID, connectionID uint) (map[string]any, error) {
	raw, err := s.run(ctx, assetType, assetID, connectionID, `docker info --format '{{json .}}'`, dockerOutputLimit)
	if err != nil {
		return nil, err
	}
	var source map[string]any
	if json.Unmarshal([]byte(strings.TrimSpace(raw)), &source) != nil {
		return nil, errors.New("Docker 信息解析失败")
	}
	output := map[string]any{}
	for _, key := range []string{"ServerVersion", "OperatingSystem", "OSType", "Architecture", "NCPU", "MemTotal", "DockerRootDir", "Containers", "ContainersRunning", "ContainersPaused", "ContainersStopped", "Images", "Driver", "CgroupDriver", "Name"} {
		output[key] = source[key]
	}
	return output, nil
}

func (s *DockerOperationService) Containers(ctx context.Context, assetType string, assetID, connectionID uint) ([]map[string]any, error) {
	raw, err := s.run(ctx, assetType, assetID, connectionID, `docker ps -a --no-trunc --size --format '{{json .}}'`, dockerOutputLimit)
	if err != nil {
		return nil, err
	}
	return parseDockerLines(raw)
}

func (s *DockerOperationService) Images(ctx context.Context, assetType string, assetID, connectionID uint) ([]map[string]any, error) {
	raw, err := s.run(ctx, assetType, assetID, connectionID, `docker image ls --no-trunc --format '{{json .}}'`, dockerOutputLimit)
	if err != nil {
		return nil, err
	}
	return parseDockerLines(raw)
}

func (s *DockerOperationService) Volumes(ctx context.Context, assetType string, assetID, connectionID uint) ([]map[string]any, error) {
	raw, err := s.run(ctx, assetType, assetID, connectionID, `docker volume ls --format '{{json .}}'`, dockerOutputLimit)
	if err != nil {
		return nil, err
	}
	return parseDockerLines(raw)
}

func (s *DockerOperationService) Networks(ctx context.Context, assetType string, assetID, connectionID uint) ([]map[string]any, error) {
	raw, err := s.run(ctx, assetType, assetID, connectionID, `docker network ls --no-trunc --format '{{json .}}'`, dockerOutputLimit)
	if err != nil {
		return nil, err
	}
	return parseDockerLines(raw)
}

func validateDockerObject(value string) error {
	if !dockerObjectPattern.MatchString(value) {
		return errors.New("容器 ID 或名称无效")
	}
	return nil
}

func (s *DockerOperationService) containerName(ctx context.Context, assetType string, assetID, connectionID uint, container string) (string, error) {
	if err := validateDockerObject(container); err != nil {
		return "", err
	}
	raw, err := s.run(ctx, assetType, assetID, connectionID, "docker inspect --type container --format '{{.Name}}' "+dockerShellQuote(container), 16<<10)
	if err != nil {
		return "", err
	}
	name := strings.TrimPrefix(strings.TrimSpace(raw), "/")
	if !dockerObjectPattern.MatchString(name) {
		return "", errors.New("容器名称无效")
	}
	return name, nil
}

func (s *DockerOperationService) Logs(ctx context.Context, assetType string, assetID, connectionID uint, container string, tail int) (string, error) {
	if err := validateDockerObject(container); err != nil {
		return "", err
	}
	if tail < 1 || tail > 2000 {
		tail = 200
	}
	command := "docker logs --timestamps --tail " + strconv.Itoa(tail) + " " + dockerShellQuote(container) + " 2>&1"
	return s.run(ctx, assetType, assetID, connectionID, command, 1<<20)
}

func (s *DockerOperationService) Action(ctx context.Context, assetType string, assetID, connectionID uint, container, action, confirmName string) (string, error) {
	name, err := s.containerName(ctx, assetType, assetID, connectionID, container)
	if err != nil {
		return "", err
	}
	allowed := map[string]bool{"start": true, "stop": true, "restart": true, "pause": true, "unpause": true}
	if !allowed[action] {
		return "", errors.New("不支持的容器操作")
	}
	if action != "start" && confirmName != name {
		return "", errors.New("请输入完整容器名称确认操作")
	}
	return s.run(ctx, assetType, assetID, connectionID, "docker "+action+" "+dockerShellQuote(container), 64<<10)
}

func (s *DockerOperationService) Remove(ctx context.Context, assetType string, assetID, connectionID uint, container, confirmName string) (string, error) {
	name, err := s.containerName(ctx, assetType, assetID, connectionID, container)
	if err != nil {
		return "", err
	}
	if confirmName != name {
		return "", errors.New("请输入完整容器名称确认删除")
	}
	// No --force: a running container cannot be deleted accidentally.
	return s.run(ctx, assetType, assetID, connectionID, "docker rm "+dockerShellQuote(container), 64<<10)
}

func validateDockerCreate(req *DockerCreateRequest) error {
	req.Name, req.Image, req.Restart = strings.TrimSpace(req.Name), strings.TrimSpace(req.Image), strings.TrimSpace(req.Restart)
	if !dockerObjectPattern.MatchString(req.Name) || !dockerImagePattern.MatchString(req.Image) {
		return errors.New("容器名称或镜像名称无效")
	}
	if req.Restart == "" {
		req.Restart = "no"
	}
	if !map[string]bool{"no": true, "always": true, "unless-stopped": true, "on-failure": true}[req.Restart] {
		return errors.New("重启策略无效")
	}
	if len(req.Environment) > 100 || len(req.Ports) > 50 || len(req.Command) > 100 || len(req.Volumes) > 50 {
		return errors.New("容器参数过多")
	}
	for key, value := range req.Environment {
		if !dockerEnvPattern.MatchString(key) || len(value) > 4096 || strings.ContainsRune(value, 0) {
			return errors.New("环境变量无效")
		}
	}
	for _, port := range req.Ports {
		if port.Host < 0 || port.Host > 65535 || port.Container < 1 || port.Container > 65535 || (port.Protocol != "" && port.Protocol != "tcp" && port.Protocol != "udp") {
			return errors.New("端口映射无效")
		}
	}
	for i := range req.Volumes {
		req.Volumes[i].Name = strings.TrimSpace(req.Volumes[i].Name)
		req.Volumes[i].ContainerPath = strings.TrimSpace(req.Volumes[i].ContainerPath)
		if !dockerObjectPattern.MatchString(req.Volumes[i].Name) || !strings.HasPrefix(req.Volumes[i].ContainerPath, "/") || len(req.Volumes[i].ContainerPath) > 512 || strings.ContainsRune(req.Volumes[i].ContainerPath, 0) {
			return errors.New("数据卷映射无效")
		}
	}
	for _, arg := range req.Command {
		if len(arg) > 4096 || strings.ContainsRune(arg, 0) {
			return errors.New("启动命令无效")
		}
	}
	return nil
}

func (s *DockerOperationService) Create(ctx context.Context, assetType string, assetID uint, req *DockerCreateRequest) (string, error) {
	if err := validateDockerCreate(req); err != nil {
		return "", err
	}
	if _, err := s.run(ctx, assetType, assetID, req.ConnectionID, "docker image inspect --format '{{.Id}}' "+dockerShellQuote(req.Image), 64<<10); err != nil {
		return "", errors.New("镜像在目标主机上不存在，请先由镜像发布流程准备镜像")
	}
	args := []string{"docker create", "--name", dockerShellQuote(req.Name), "--restart", dockerShellQuote(req.Restart)}
	for key, value := range req.Environment {
		args = append(args, "-e", dockerShellQuote(key+"="+value))
	}
	for _, port := range req.Ports {
		protocol := port.Protocol
		if protocol == "" {
			protocol = "tcp"
		}
		mapping := strconv.Itoa(port.Container) + "/" + protocol
		if port.Host > 0 {
			mapping = strconv.Itoa(port.Host) + ":" + mapping
		}
		args = append(args, "-p", dockerShellQuote(mapping))
	}
	for _, volume := range req.Volumes {
		mapping := volume.Name + ":" + volume.ContainerPath
		if volume.ReadOnly {
			mapping += ":ro"
		}
		args = append(args, "-v", dockerShellQuote(mapping))
	}
	args = append(args, dockerShellQuote(req.Image))
	for _, arg := range req.Command {
		args = append(args, dockerShellQuote(arg))
	}
	id, err := s.run(ctx, assetType, assetID, req.ConnectionID, strings.Join(args, " "), 64<<10)
	if err != nil {
		return "", err
	}
	containerID := strings.TrimSpace(id)
	if _, err := s.run(ctx, assetType, assetID, req.ConnectionID, "docker start "+dockerShellQuote(containerID), 64<<10); err != nil {
		return containerID, errors.New("容器已创建但启动失败，请检查容器日志")
	}
	return containerID, nil
}
