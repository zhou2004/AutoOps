package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// VolumeCreate 创建数据卷。
func (s *DockerOperationService) VolumeCreate(ctx context.Context, assetType string, assetID, connectionID uint, name, driver string) (string, error) {
	name = strings.TrimSpace(name)
	if !dockerObjectPattern.MatchString(name) {
		return "", errors.New("数据卷名称无效")
	}
	command := "docker volume create"
	if driver != "" {
		if !dockerObjectPattern.MatchString(driver) {
			return "", errors.New("数据卷驱动无效")
		}
		command += " --driver " + dockerShellQuote(driver)
	}
	command += " " + dockerShellQuote(name)
	out, err := s.run(ctx, assetType, assetID, connectionID, command, 64<<10)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// VolumeInspect 查看数据卷详情。
func (s *DockerOperationService) VolumeInspect(ctx context.Context, assetType string, assetID, connectionID uint, name string) (map[string]any, error) {
	name = strings.TrimSpace(name)
	if !dockerObjectPattern.MatchString(name) {
		return nil, errors.New("数据卷名称无效")
	}
	raw, err := s.run(ctx, assetType, assetID, connectionID, "docker volume inspect "+dockerShellQuote(name), dockerOutputLimit)
	if err != nil {
		return nil, err
	}
	var list []map[string]any
	if json.Unmarshal([]byte(strings.TrimSpace(raw)), &list) != nil || len(list) == 0 {
		return nil, errors.New("数据卷信息解析失败")
	}
	return list[0], nil
}

// VolumeRemove 删除数据卷。
func (s *DockerOperationService) VolumeRemove(ctx context.Context, assetType string, assetID, connectionID uint, name, confirmName string) (string, error) {
	name, confirmName = strings.TrimSpace(name), strings.TrimSpace(confirmName)
	if !dockerObjectPattern.MatchString(name) {
		return "", errors.New("数据卷名称无效")
	}
	if confirmName != name {
		return "", errors.New("请输入完整数据卷名称确认删除")
	}
	out, err := s.run(ctx, assetType, assetID, connectionID, "docker volume rm "+dockerShellQuote(name), 64<<10)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// VolumeContainers 查看使用指定数据卷的容器（映射关系）。
func (s *DockerOperationService) VolumeContainers(ctx context.Context, assetType string, assetID, connectionID uint, name string) ([]map[string]any, error) {
	name = strings.TrimSpace(name)
	if !dockerObjectPattern.MatchString(name) {
		return nil, errors.New("数据卷名称无效")
	}
	raw, err := s.run(ctx, assetType, assetID, connectionID, "docker ps -a --filter volume="+dockerShellQuote(name)+" --format '{{json .}}'", dockerOutputLimit)
	if err != nil {
		return nil, err
	}
	return parseDockerLines(raw)
}
