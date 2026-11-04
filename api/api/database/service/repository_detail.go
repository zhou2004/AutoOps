package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	accounts "dodevops-api/api/configcenter/model"
	"dodevops-api/api/database/model"
	"gorm.io/gorm"
	"time"
)

// RepositoryDetail performs metadata reads only. It never scans an entire bucket
// or decrypts/returns its credential; counts are limited to authorized clusters.
func (s *Service) RepositoryDetail(ctx context.Context, user, id uint) (map[string]any, error) {
	db := s.DB.WithContext(ctx)
	var repo model.Repository
	if e := db.First(&repo, id).Error; e != nil {
		return nil, e
	}
	scope, e := s.Scope(user, "db_cluster_view")
	if e != nil {
		return nil, e
	}
	clusters := db.Model(&model.Cluster{})
	if scope != nil {
		ids := []uint{}
		for id := range scope {
			ids = append(ids, id)
		}
		clusters = clusters.Where("id IN ?", ids).Session(&gorm.Session{})
		var n int64
		if e := clusters.Where("group_id = ?", repo.GroupID).Count(&n).Error; e != nil {
			return nil, e
		}
		if n == 0 {
			return nil, ErrForbidden
		}
	}
	var group cmdb.CmdbGroup
	db.Select("id,name").First(&group, repo.GroupID)
	var host cmdb.CmdbHost
	db.Select("id,host_name,ssh_ip").First(&host, repo.HostID)
	var credential accounts.KeyManage
	if repo.KeyID > 0 {
		db.Select("id,remark").First(&credential, repo.KeyID)
	}
	var rows []model.Backup
	if e := db.Select("id,name,cluster_id,status,method,verified_at,created_at,manifest").Where("repository_id = ? AND cluster_id IN (?)", repo.ID, clusters.Select("id")).Order("id DESC").Find(&rows).Error; e != nil {
		return nil, e
	}
	var count, verified int
	var storedBytes int64
	var latestAt *time.Time
	recent := []map[string]any{}
	for _, b := range rows {
		if b.Status != "complete" {
			continue
		}
		count++
		if latestAt == nil {
			at := b.CreatedAt
			latestAt = &at
		}
		if b.VerifiedAt != nil {
			verified++
		}
		manifest := object(b.Manifest)
		if manifest["remoteComplete"] == true {
			artifacts := object(JSON(manifest["artifacts"]))
			for key, raw := range object(JSON(manifest["repositoryReport"])) {
				report := object(JSON(raw))
				artifact := object(JSON(artifacts[key]))
				if report["verified"] == true && report["sha256"] == artifact["sha256"] {
					storedBytes += int64(number(artifact["size"]))
				}
			}
		}
		if len(recent) < 5 {
			recent = append(recent, map[string]any{"id": b.ID, "name": b.Name, "clusterId": b.ClusterID, "method": b.Method, "createdAt": b.CreatedAt, "verifiedAt": b.VerifiedAt})
		}
	}
	policies := []model.BackupPolicy{}
	if e := db.Where("repository_id = ? AND cluster_id IN (?)", repo.ID, clusters.Select("id")).Order("id DESC").Find(&policies).Error; e != nil {
		return nil, e
	}
	var names []model.Cluster
	if e := clusters.Select("id,name,engine").Find(&names).Error; e != nil {
		return nil, e
	}
	lookup := map[uint]model.Cluster{}
	for _, c := range names {
		lookup[c.ID] = c
	}
	for _, b := range recent {
		c := lookup[b["clusterId"].(uint)]
		b["clusterName"] = c.Name
		b["engine"] = c.Engine
	}
	return map[string]any{"repository": repo, "groupName": group.Name, "host": map[string]any{"id": repo.HostID, "name": host.HostName, "address": host.SSHIP}, "credentialName": credential.Remark, "statistics": map[string]any{"backupCount": count, "verifiedBackupCount": verified, "managedBytes": storedBytes, "lastBackupAt": latestAt, "policyCount": len(policies), "scoped": scope != nil}, "policies": policies, "recentBackups": recent}, nil
}
