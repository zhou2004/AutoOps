package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"dodevops-api/api/cmdb/model"
	"github.com/go-redis/redis/v8"
)

// The database domain reuses the existing credential and verified SSH machinery.
// These helpers are internal Go APIs and are never serialized by an HTTP controller.
func (s *DatabaseOperationService) OpenSQL(ctx context.Context, id uint, database string) (*sql.DB, error) {
	db, _, err := s.openSQL(ctx, id, database)
	return db, err
}
func (s *DatabaseOperationService) OpenRedis(id uint) (redis.UniversalClient, error) {
	client, _, err := s.redisClient(id)
	return client, err
}
func (s *DatabaseAutomationService) InventoryHosts(ids []uint, dir string) (map[uint]map[string]any, error) {
	raw, _ := json.Marshal(ids)
	inventory, err := s.inventory(model.DatabaseDeployment{Nodes: string(raw), Managers: "[]"}, dir)
	if err != nil {
		return nil, err
	}
	hosts := inventory["all"].(map[string]any)["children"].(map[string]any)["mysql_nodes"].(map[string]any)["hosts"].(map[string]any)
	out := map[uint]map[string]any{}
	for _, id := range ids {
		out[id] = hosts[fmt.Sprintf("node_%d", id)].(map[string]any)
	}
	return out, nil
}
func (s *DatabaseAutomationService) HostProfile(id uint) (*model.CmdbAssetConnection, error) {
	return s.hostConnection(id)
}
func (s *DatabaseAutomationService) OrchestratorControlAccount(id uint) (string, string, error) {
	d, e := s.Deployment(id)
	if e != nil {
		return "", "", e
	}
	sec, e := s.secrets(d)
	if e != nil {
		return "", "", e
	}
	return "autoops", sec.HTTP, nil
}

// DeploymentCredentials is an internal adapter for migrating operational tasks.
// It must never be returned from a controller or written into a task log.
func (s *DatabaseAutomationService) DeploymentCredentials(id uint) (map[string]string, error) {
	d, e := s.Deployment(id)
	if e != nil {
		return nil, e
	}
	v, e := s.secrets(d)
	if e != nil {
		return nil, e
	}
	return map[string]string{"password": v.Password, "replication": v.Replication, "topology": v.Topology, "control": v.HTTP}, nil
}
