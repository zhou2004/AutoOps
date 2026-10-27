package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// NetworkCreate 创建网络。
func (s *DockerOperationService) NetworkCreate(ctx context.Context, assetType string, assetID, connectionID uint, name, driver string) (string, error) {
	name = strings.TrimSpace(name)
	if !dockerObjectPattern.MatchString(name) {
		return "", errors.New("网络名称无效")
	}
	command := "docker network create"
	if driver != "" {
		if !dockerObjectPattern.MatchString(driver) {
			return "", errors.New("网络驱动无效")
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

// NetworkRemove 删除网络。
func (s *DockerOperationService) NetworkRemove(ctx context.Context, assetType string, assetID, connectionID uint, name, confirmName string) (string, error) {
	name, confirmName = strings.TrimSpace(name), strings.TrimSpace(confirmName)
	if !dockerObjectPattern.MatchString(name) {
		return "", errors.New("网络名称无效")
	}
	if confirmName != name {
		return "", errors.New("请输入完整网络名称确认删除")
	}
	out, err := s.run(ctx, assetType, assetID, connectionID, "docker network rm "+dockerShellQuote(name), 64<<10)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// NetworkInspect 查看网络详情。
func (s *DockerOperationService) NetworkInspect(ctx context.Context, assetType string, assetID, connectionID uint, name string) (map[string]any, error) {
	name = strings.TrimSpace(name)
	if !dockerObjectPattern.MatchString(name) {
		return nil, errors.New("网络名称无效")
	}
	raw, err := s.run(ctx, assetType, assetID, connectionID, "docker network inspect "+dockerShellQuote(name), dockerOutputLimit)
	if err != nil {
		return nil, err
	}
	var list []map[string]any
	if json.Unmarshal([]byte(strings.TrimSpace(raw)), &list) != nil || len(list) == 0 {
		return nil, errors.New("网络信息解析失败")
	}
	return list[0], nil
}

// NetworkContainers 查看连接到网络的容器。
func (s *DockerOperationService) NetworkContainers(ctx context.Context, assetType string, assetID, connectionID uint, name string) ([]map[string]any, error) {
	name = strings.TrimSpace(name)
	if !dockerObjectPattern.MatchString(name) {
		return nil, errors.New("网络名称无效")
	}
	raw, err := s.run(ctx, assetType, assetID, connectionID, "docker ps -a --filter network="+dockerShellQuote(name)+" --format '{{json .}}'", dockerOutputLimit)
	if err != nil {
		return nil, err
	}
	return parseDockerLines(raw)
}

// NetworkConnect 将容器连接到网络。
func (s *DockerOperationService) NetworkConnect(ctx context.Context, assetType string, assetID, connectionID uint, network, container string) (string, error) {
	network, container = strings.TrimSpace(network), strings.TrimSpace(container)
	if !dockerObjectPattern.MatchString(network) {
		return "", errors.New("网络名称无效")
	}
	if err := validateDockerObject(container); err != nil {
		return "", err
	}
	out, err := s.run(ctx, assetType, assetID, connectionID, "docker network connect "+dockerShellQuote(network)+" "+dockerShellQuote(container), 64<<10)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// NetworkDisconnect 将容器从网络断开。
func (s *DockerOperationService) NetworkDisconnect(ctx context.Context, assetType string, assetID, connectionID uint, network, container string) (string, error) {
	network, container = strings.TrimSpace(network), strings.TrimSpace(container)
	if !dockerObjectPattern.MatchString(network) {
		return "", errors.New("网络名称无效")
	}
	if err := validateDockerObject(container); err != nil {
		return "", err
	}
	out, err := s.run(ctx, assetType, assetID, connectionID, "docker network disconnect "+dockerShellQuote(network)+" "+dockerShellQuote(container), 64<<10)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
