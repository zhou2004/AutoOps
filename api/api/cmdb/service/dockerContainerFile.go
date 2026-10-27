package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

// dockerShellPattern 限制容器内执行的 shell 类型，避免任意命令注入。
var dockerShellPattern = map[string]bool{
	"/bin/sh":   true,
	"/bin/bash": true,
	"/bin/ash":  true,
	"/bin/zsh":  true,
	"/bin/dash": true,
}

// ContainerExec 在容器内执行命令（用于文件管理）。
func (s *DockerOperationService) ContainerExec(ctx context.Context, assetType string, assetID, connectionID uint, container, command string, maxBytes int) (string, error) {
	if err := validateDockerObject(container); err != nil {
		return "", err
	}
	if strings.TrimSpace(command) == "" {
		return "", errors.New("容器内命令不能为空")
	}
	return s.run(ctx, assetType, assetID, connectionID, "docker exec "+dockerShellQuote(container)+" "+command, maxBytes)
}

// ContainerInspect 返回容器检查信息。
func (s *DockerOperationService) ContainerInspect(ctx context.Context, assetType string, assetID, connectionID uint, container string) (map[string]any, error) {
	if err := validateDockerObject(container); err != nil {
		return nil, err
	}
	raw, err := s.run(ctx, assetType, assetID, connectionID, "docker inspect --type container "+dockerShellQuote(container), dockerOutputLimit)
	if err != nil {
		return nil, err
	}
	var list []map[string]any
	if json.Unmarshal([]byte(strings.TrimSpace(raw)), &list) != nil || len(list) == 0 {
		return nil, errors.New("容器信息解析失败")
	}
	return list[0], nil
}

// ContainerFileList 获取容器内文件列表。
func (s *DockerOperationService) ContainerFileList(ctx context.Context, assetType string, assetID, connectionID uint, container, path string) ([]map[string]any, error) {
	if path == "" {
		path = "/"
	}
	inner := "ls -al -p " + dockerShellQuote(path)
	output, err := s.ContainerExec(ctx, assetType, assetID, connectionID, container, "sh -c "+dockerShellQuote(inner), 1<<20)
	if err != nil {
		return nil, err
	}
	files := []map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "total ") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 9 {
			continue
		}
		name := strings.Join(parts[8:], " ")
		if name == "." || name == ".." {
			continue
		}
		isDir := strings.HasSuffix(name, "/")
		if isDir {
			name = strings.TrimSuffix(name, "/")
		} else {
			isDir = strings.HasPrefix(parts[0], "d")
		}
		files = append(files, map[string]any{
			"name":    name,
			"isDir":   isDir,
			"size":    parts[4],
			"modTime": parts[5] + " " + parts[6] + " " + parts[7],
			"mode":    parts[0],
			"owner":   parts[2],
			"group":   parts[3],
		})
	}
	return files, nil
}

// ContainerFileContent 获取容器内文件内容。
func (s *DockerOperationService) ContainerFileContent(ctx context.Context, assetType string, assetID, connectionID uint, container, path string) (string, error) {
	if path == "" {
		return "", errors.New("文件路径不能为空")
	}
	return s.ContainerExec(ctx, assetType, assetID, connectionID, container, "cat "+dockerShellQuote(path), 4<<20)
}

// ContainerFileWrite 覆盖写入容器内文件。
func (s *DockerOperationService) ContainerFileWrite(ctx context.Context, assetType string, assetID, connectionID uint, container, path string, content []byte) error {
	if path == "" {
		return errors.New("文件路径不能为空")
	}
	inner := "echo " + dockerShellQuote(base64.StdEncoding.EncodeToString(content)) + " | base64 -d > " + dockerShellQuote(path)
	_, err := s.ContainerExec(ctx, assetType, assetID, connectionID, container, "sh -c "+dockerShellQuote(inner), 64<<10)
	return err
}

// ContainerFileMkdir 在容器内创建目录。
func (s *DockerOperationService) ContainerFileMkdir(ctx context.Context, assetType string, assetID, connectionID uint, container, path string) error {
	if path == "" {
		return errors.New("目录路径不能为空")
	}
	_, err := s.ContainerExec(ctx, assetType, assetID, connectionID, container, "mkdir -p "+dockerShellQuote(path), 64<<10)
	return err
}

// ContainerFileDelete 删除容器内文件或目录。
func (s *DockerOperationService) ContainerFileDelete(ctx context.Context, assetType string, assetID, connectionID uint, container, path string) error {
	if path == "" || path == "/" {
		return errors.New("不能删除根目录或空路径")
	}
	_, err := s.ContainerExec(ctx, assetType, assetID, connectionID, container, "rm -rf "+dockerShellQuote(path), 64<<10)
	return err
}

// ContainerFileDownload 下载容器内文件，返回原始字节。
func (s *DockerOperationService) ContainerFileDownload(ctx context.Context, assetType string, assetID, connectionID uint, container, path string) ([]byte, error) {
	if path == "" {
		return nil, errors.New("文件路径不能为空")
	}
	inner := "base64 " + dockerShellQuote(path)
	output, err := s.ContainerExec(ctx, assetType, assetID, connectionID, container, "sh -c "+dockerShellQuote(inner), 64<<20)
	if err != nil {
		return nil, err
	}
	output = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' {
			return -1
		}
		return r
	}, output)
	raw, err := base64.StdEncoding.DecodeString(output)
	if err != nil {
		return nil, errors.New("容器内文件编码解析失败")
	}
	return raw, nil
}

// ContainerTerminal 通过 WebSocket 桥接 docker exec 终端。
func (s *DockerOperationService) ContainerTerminal(ctx context.Context, assetType string, assetID, connectionID uint, container, shell string, ws *websocket.Conn) error {
	if err := validateDockerObject(container); err != nil {
		return err
	}
	if !dockerShellPattern[shell] {
		shell = "/bin/sh"
	}
	client, err := ResolveAssetSSH(s.db, assetType, assetID, connectionID)
	if err != nil {
		return errors.New("SSH 连接失败，请检查连接档案、凭据和主机指纹")
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return errors.New("无法创建 SSH 会话")
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return errors.New("无法建立终端输入管道")
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return errors.New("无法建立终端输出管道")
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return errors.New("无法建立终端错误管道")
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
		ssh.ICRNL:         1,
		ssh.OPOST:         1,
	}
	if err := session.RequestPty("xterm-256color", 40, 160, modes); err != nil {
		return errors.New("终端伪终端创建失败")
	}
	command := "docker exec -it " + dockerShellQuote(container) + " " + shell
	if err := session.Start(command); err != nil {
		return errors.New("容器终端启动失败，请确认容器处于运行状态")
	}

	stop := context.AfterFunc(ctx, func() { _ = session.Close(); _ = client.Close() })
	defer stop()

	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				if werr := ws.WriteMessage(websocket.TextMessage, buf[:n]); werr != nil {
					_ = session.Close()
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				if werr := ws.WriteMessage(websocket.TextMessage, buf[:n]); werr != nil {
					_ = session.Close()
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			break
		}
		var msg map[string]any
		if json.Unmarshal(data, &msg) == nil {
			// 兼容 K8s 终端协议
			if msg["operation"] == "resize" {
				info, _ := msg["data"].(map[string]any)
				cols, _ := strconv.Atoi(fmt.Sprintf("%v", info["cols"]))
				rows, _ := strconv.Atoi(fmt.Sprintf("%v", info["rows"]))
				if cols > 0 && rows > 0 {
					_ = session.WindowChange(rows, cols)
				}
				continue
			}
			if msg["type"] == "resize" {
				cols, _ := strconv.Atoi(fmt.Sprintf("%v", msg["cols"]))
				rows, _ := strconv.Atoi(fmt.Sprintf("%v", msg["rows"]))
				if cols > 0 && rows > 0 {
					_ = session.WindowChange(rows, cols)
				}
				continue
			}
			if msg["operation"] == "stdin" {
				if text, ok := msg["data"].(string); ok {
					_, _ = stdin.Write([]byte(text))
				}
				continue
			}
			continue
		}
		if _, err := stdin.Write(data); err != nil {
			break
		}
	}
	_ = stdin.Close()
	return session.Wait()
}
