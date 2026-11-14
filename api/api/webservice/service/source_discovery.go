package service

import (
	"context"
	cmdbservice "dodevops-api/api/cmdb/service"
	"dodevops-api/api/webservice/model"
	"errors"
	"time"
)

type SourceDiscoveryInput struct {
	HostID       uint   `json:"hostId"`
	ConnectionID uint   `json:"connectionId"`
	Runtime      string `json:"runtime"`
	Container    string `json:"container"`
	Sudo         bool   `json:"sudo"`
}
type SourceCandidate struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Spec     model.SourceSpec `json:"spec"`
	Identity string           `json:"identity"`
	Version  string           `json:"version"`
	Warnings []string         `json:"warnings"`
}
type SourceDiscoveryResult struct {
	Candidates []SourceCandidate `json:"candidates"`
	Warnings   []string          `json:"warnings"`
}

func (s *Service) DiscoverSources(ctx context.Context, u uint, input SourceDiscoveryInput) (SourceDiscoveryResult, error) {
	result := SourceDiscoveryResult{Candidates: []SourceCandidate{}, Warnings: []string{}}
	if s.Module(ctx, u, "write") != nil || !s.Admin(u) {
		return result, ErrForbidden
	}
	if input.HostID == 0 || input.ConnectionID == 0 || (input.Runtime != "host" && input.Runtime != "docker") || (input.Container != "" && !objectName.MatchString(input.Container)) {
		return result, errors.New("请选择有效主机、SSH 连接及运行环境")
	}
	if err := s.requireHostAccess(ctx, u, input.HostID, "connect_ssh"); err != nil {
		return result, err
	}
	connection, err := cmdbservice.GetAssetConnection(s.DB, "host", input.HostID, input.ConnectionID)
	if err != nil || !connection.Enabled || connection.Protocol != "ssh" {
		return result, errors.New("请选择该主机已启用的 SSH 连接")
	}
	child, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	raw, err := s.Remote.Run(child, RemoteRequest{Action: "source-discovery", Spec: model.SourceSpec{HostID: input.HostID, ConnectionID: input.ConnectionID, Runtime: input.Runtime, Container: input.Container, Sudo: input.Sudo}})
	if err != nil {
		return result, err
	}
	result, err = decode[SourceDiscoveryResult](raw)
	if err != nil {
		return SourceDiscoveryResult{}, err
	}
	// A scan never creates sources or changes existing registrations.
	for _, candidate := range result.Candidates {
		if candidate.Spec.HostID != input.HostID || candidate.Spec.ConnectionID != input.ConnectionID || candidate.Spec.Runtime != input.Runtime {
			return SourceDiscoveryResult{}, errors.New("发现结果目标不匹配")
		}
	}
	return result, nil
}
