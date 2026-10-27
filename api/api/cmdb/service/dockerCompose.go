package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

const composeBaseDir = "/opt/autoops/compose"

func composeFilePath(project string) string {
	return composeBaseDir + "/" + project + "/docker-compose.yml"
}

// ComposeList 列出目标主机上的 Compose 项目。
// 以本面板管理的配置目录为数据源，避免依赖 `docker compose ls` 的版本差异。
func (s *DockerOperationService) ComposeList(ctx context.Context, assetType string, assetID, connectionID uint) ([]map[string]any, error) {
	sc, client, err := s.sftpClient(ctx, assetType, assetID, connectionID)
	if err != nil {
		return nil, err
	}
	defer sc.Close()
	defer client.Close()
	entries, err := sc.ReadDir(composeBaseDir)
	if err != nil {
		return []map[string]any{}, nil
	}
	projects := []map[string]any{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !dockerObjectPattern.MatchString(name) {
			continue
		}
		status := "未部署"
		raw, cerr := s.run(ctx, assetType, assetID, connectionID, "docker ps -a --filter label=com.docker.compose.project="+dockerShellQuote(name)+" --format '{{json .}}'", dockerOutputLimit)
		if cerr == nil {
			if items, perr := parseDockerLines(raw); perr == nil {
				running := 0
				for _, item := range items {
					if item["State"] == "running" {
						running++
					}
				}
				switch {
				case len(items) == 0:
					status = "未部署"
				case running > 0:
					status = fmt.Sprintf("运行中 %d/%d", running, len(items))
				default:
					status = fmt.Sprintf("已停止 (%d)", len(items))
				}
			}
		}
		projects = append(projects, map[string]any{
			"Name":        name,
			"Status":      status,
			"ConfigFiles": composeFilePath(name),
		})
	}
	return projects, nil
}

// ComposeGet 读取项目 yml 配置。
func (s *DockerOperationService) ComposeGet(ctx context.Context, assetType string, assetID, connectionID uint, project string) (map[string]any, error) {
	if !dockerObjectPattern.MatchString(strings.TrimSpace(project)) {
		return nil, errors.New("项目名称无效")
	}
	sc, client, err := s.sftpClient(ctx, assetType, assetID, connectionID)
	if err != nil {
		return nil, err
	}
	defer sc.Close()
	defer client.Close()
	f, err := sc.Open(composeFilePath(project))
	if err != nil {
		return map[string]any{"project": project, "content": "", "path": composeFilePath(project)}, nil
	}
	defer f.Close()
	content, err := io.ReadAll(io.LimitReader(f, 4<<20))
	if err != nil {
		return nil, errors.New("读取 Compose 配置失败")
	}
	return map[string]any{"project": project, "content": string(content), "path": composeFilePath(project)}, nil
}

// ComposeSave 保存 yml 并可选启动。
func (s *DockerOperationService) ComposeSave(ctx context.Context, assetType string, assetID, connectionID uint, project, content string, up bool) (string, error) {
	project = strings.TrimSpace(project)
	if !dockerObjectPattern.MatchString(project) {
		return "", errors.New("项目名称无效")
	}
	if strings.TrimSpace(content) == "" || len(content) > 4<<20 {
		return "", errors.New("Compose 配置内容为空或过大")
	}
	sc, client, err := s.sftpClient(ctx, assetType, assetID, connectionID)
	if err != nil {
		return "", err
	}
	if err := sc.MkdirAll(composeBaseDir + "/" + project); err != nil {
		sc.Close()
		client.Close()
		return "", errors.New("无法创建 Compose 配置目录")
	}
	f, err := sc.Create(composeFilePath(project))
	if err != nil {
		sc.Close()
		client.Close()
		return "", errors.New("无法写入 Compose 配置")
	}
	if _, err := f.Write([]byte(content)); err != nil {
		f.Close()
		sc.Close()
		client.Close()
		return "", errors.New("写入 Compose 配置失败")
	}
	f.Close()
	sc.Close()
	client.Close()

	if up {
		out, err := s.runLong(ctx, assetType, assetID, connectionID, "docker compose -f "+composeFilePath(project)+" -p "+dockerShellQuote(project)+" up -d", 8<<20)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(out), nil
	}
	return "配置已保存", nil
}

var composeActions = map[string]bool{"up": true, "down": true, "restart": true, "pull": true, "ps": true, "config": true, "stop": true, "start": true}

// ComposeAction 对项目执行生命周期操作。
func (s *DockerOperationService) ComposeAction(ctx context.Context, assetType string, assetID, connectionID uint, project, action string) (string, error) {
	project, action = strings.TrimSpace(project), strings.TrimSpace(strings.ToLower(action))
	if !dockerObjectPattern.MatchString(project) {
		return "", errors.New("项目名称无效")
	}
	if !composeActions[action] {
		return "", errors.New("不支持的 Compose 操作")
	}
	command := "docker compose -f " + composeFilePath(project) + " -p " + dockerShellQuote(project) + " " + action
	if action == "up" {
		command += " -d"
	}
	if action == "down" {
		command += " --remove-orphans"
	}
	runFn := s.run
	if action == "up" || action == "pull" || action == "restart" {
		runFn = s.runLong
	}
	out, err := runFn(ctx, assetType, assetID, connectionID, command, 8<<20)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// ComposeRemove 停止并删除 Compose 项目及其配置。
func (s *DockerOperationService) ComposeRemove(ctx context.Context, assetType string, assetID, connectionID uint, project, confirmName string) (string, error) {
	project, confirmName = strings.TrimSpace(project), strings.TrimSpace(confirmName)
	if !dockerObjectPattern.MatchString(project) {
		return "", errors.New("项目名称无效")
	}
	if confirmName != project {
		return "", errors.New("请输入完整项目名称确认删除")
	}
	if _, err := s.runLong(ctx, assetType, assetID, connectionID, "docker compose -f "+composeFilePath(project)+" -p "+dockerShellQuote(project)+" down --remove-orphans", 8<<20); err != nil {
		return "", errors.New("停止项目失败：" + err.Error())
	}
	sc, client, err := s.sftpClient(ctx, assetType, assetID, connectionID)
	if err != nil {
		return "", err
	}
	defer sc.Close()
	defer client.Close()
	_ = sc.RemoveAll(composeBaseDir + "/" + project)
	return fmt.Sprintf("项目 %s 已删除", project), nil
}
