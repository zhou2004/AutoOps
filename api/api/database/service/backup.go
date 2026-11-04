package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"dodevops-api/api/database/model"
	"dodevops-api/common/util"
	"dodevops-api/pkg/apikey"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BackupRecord resolves an individual backup independently of list pagination.
func (s *Service) BackupRecord(ctx context.Context, user, id uint) (*model.Backup, error) {
	var backup model.Backup
	if id == 0 {
		return nil, errors.New("备份 ID 无效")
	}
	if err := s.DB.WithContext(ctx).First(&backup, id).Error; err != nil {
		return nil, err
	}
	if err := s.Authorize(user, backup.ClusterID, "db_cluster_view"); err != nil {
		return nil, err
	}
	return &backup, nil
}

func backupName(c *model.Cluster, p Parameters, at time.Time) (string, error) {
	if !utf8.ValidString(p.BackupName) || utf8.RuneCountInString(p.BackupName) > 100 || strings.IndexFunc(p.BackupName, unicode.IsControl) >= 0 {
		return "", errors.New("备份名称最多 100 个字符，不能包含换行或控制字符")
	}
	if value := strings.TrimSpace(p.BackupName); value != "" {
		return value, nil
	}
	return fmt.Sprintf("%.60s-%s-%s", c.Name, p.Method, at.Format("20060102-150405")), nil
}

func (s *Service) backup(ctx context.Context, c *model.Cluster, op model.Operation, p Parameters) error {
	name, e := backupName(c, p, time.Now())
	if e != nil {
		return e
	}
	nodes := []model.Member{}
	for _, m := range c.Members {
		if m.Kind == "data" && m.Role == "primary" {
			nodes = append(nodes, m)
		}
	}
	if len(nodes) == 0 {
		return errors.New("没有可备份的主节点")
	}
	parentLabel := ""
	parentArtifact := map[string]any{}
	if c.Engine == "postgresql" && (p.Method == "incremental" || p.Method == "differential" || p.Method == "wal-archive") {
		var parent model.Backup
		q := s.DB.Where("cluster_id = ? AND status = ? AND method IN ?", c.ID, "complete", []string{"physical", "incremental", "differential", "wal-archive"})
		if p.ParentBackupID > 0 {
			q = q.Where("id = ?", p.ParentBackupID)
		}
		if q.Order("id DESC").First(&parent).Error != nil {
			return errors.New("没有完整物理基线备份")
		}
		artifacts := object(JSON(object(parent.Manifest)["artifacts"]))
		artifact := object(JSON(artifacts[fmt.Sprintf("member_%d", nodes[0].ID)]))
		parentLabel = text(artifact["backupLabel"])
		if parentLabel == "" {
			return errors.New("当前主库与原物理备份成员不一致，请先创建新全量")
		}
		p.ParentBackupID = parent.ID
	}
	if c.Engine == "mysql" && (p.Method == "incremental" || p.Method == "binlog-archive") {
		var parent model.Backup
		q := s.DB.Where("cluster_id = ? AND status = ? AND method IN ?", c.ID, "complete", []string{"physical", "incremental"})
		if p.ParentBackupID > 0 {
			q = q.Where("id = ?", p.ParentBackupID)
		}
		if q.Order("id DESC").First(&parent).Error != nil {
			return errors.New("没有完整的 MySQL 物理基线")
		}
		artifacts := object(JSON(object(parent.Manifest)["artifacts"]))
		parentArtifact = object(JSON(artifacts[fmt.Sprintf("member_%d", nodes[0].ID)]))
		if text(parentArtifact["nativeId"]) != nodes[0].NativeID || text(parentArtifact["file"]) == "" {
			return errors.New("备份基线不是当前主实例，请创建新全量")
		}
		p.ParentBackupID = parent.ID
	}
	b := model.Backup{Name: name, ClusterID: c.ID, OperationID: op.ID, RepositoryID: p.RepositoryID, ParentID: p.ParentBackupID, Method: p.Method, Status: "running", Manifest: JSON(map[string]any{})}
	credentials := map[string]any{}
	for _, m := range nodes {
		user, password, e := s.memberAccount(c, m)
		if e != nil {
			return e
		}
		credentials[fmt.Sprintf("member_%d", m.ID)] = map[string]string{"username": user, "password": password}
	}
	encrypted, e := util.AESEncrypt(string(JSON(credentials)))
	if e != nil {
		return e
	}
	b.Credentials = encrypted
	if e := s.DB.Create(&b).Error; e != nil {
		return e
	}
	manifests := map[string]json.RawMessage{}
	e = s.step(op, "创建一致性备份并校验文件", func(log func(string)) error {
		if c.Engine == "redis" {
			for _, m := range nodes {
				client, e := s.redis(c, m)
				if e != nil {
					return e
				}
				e = s.persistencePreflight(ctx, c, m, client)
				client.Close()
				if e != nil {
					return e
				}
			}
		}
		var e error
		manifests, e = s.ansible(ctx, c, nodes, "backup.yml", map[string]any{"operation_id": op.ID, "backup_id": b.ID, "backup_method": p.Method, "parent_backup_label": parentLabel, "parent_backup_artifact": parentArtifact}, log)
		if e != nil {
			return e
		}
		if len(manifests) != len(nodes) {
			return errors.New("备份清单不完整，不能标记成功")
		}
		for _, raw := range manifests {
			m := object(raw)
			if m["complete"] != true || m["sha256"] == nil || m["file"] == nil {
				return errors.New("备份未生成完整性证据")
			}
		}
		return nil
	})
	manifest := map[string]any{"engine": c.Engine, "topology": c.Topology, "members": nodes, "artifacts": manifests, "localComplete": e == nil, "remoteComplete": false, "crossShardAtomic": false, "createdAt": time.Now().UTC()}
	if e == nil && p.RepositoryID > 0 {
		e = s.storeBackup(ctx, c, &b, manifest, op)
	}
	status := "complete"
	if e != nil {
		status = "failed"
	}
	s.DB.Model(&b).Updates(map[string]any{"status": status, "manifest": JSON(manifest)})
	return e
}
func (s *Service) restore(ctx context.Context, c *model.Cluster, op model.Operation, p Parameters) error {
	var backup model.Backup
	if e := s.DB.Where("id = ? AND cluster_id = ? AND status = ?", p.BackupID, c.ID, "complete").First(&backup).Error; e != nil {
		return errors.New("备份不存在或未完成")
	}
	var manifest struct {
		Members   []model.Member             `json:"members"`
		Artifacts map[string]json.RawMessage `json:"artifacts"`
	}
	if json.Unmarshal(backup.Manifest, &manifest) != nil || len(manifest.Members) == 0 {
		return errors.New("备份清单无效")
	}
	if p.TargetTime != "" && backup.Method == "logical" {
		return errors.New("逻辑备份不具备连续日志归档，不能用于时间点恢复")
	}
	// A restore is a new instance with a reserved port; existing instances are
	// never reused, even when they belong to another business group.
	var used int64
	s.DB.Model(&model.Member{}).Where("host_id = ? AND port >= ? AND port < ?", p.HostID, p.Port, p.Port+len(manifest.Members)).Count(&used)
	if used > 0 {
		return errors.New("恢复端口已被登记实例使用")
	}
	return s.step(op, "隔离恢复、启动并验证备份数据", func(log func(string)) error {
		credentials := map[string]any{}
		if backup.Credentials != "" {
			raw, e := util.AESDecrypt(backup.Credentials)
			if e != nil {
				return errors.New("备份凭据无法解密")
			}
			if json.Unmarshal([]byte(raw), &credentials) != nil {
				return errors.New("备份凭据格式无效")
			}
		}
		transferMembers := append([]model.Member(nil), manifest.Members...)
		transferManifest := object(backup.Manifest)
		if backup.RepositoryID > 0 && transferManifest["remoteComplete"] == true {
			var e error
			transferMembers, transferManifest, e = s.retrieveRepository(ctx, c, &backup, op, transferMembers)
			if e != nil {
				return e
			}
			log("已从独立仓库读取并校验备份；传输不依赖原数据库节点")
		}
		results, e := s.ansible(ctx, c, transferMembers, "restore.yml", map[string]any{"operation_id": op.ID, "backup_manifest": transferManifest, "restore_host_id": p.HostID, "restore_port": p.Port, "target_time": p.TargetTime, "restore_credentials": credentials}, log)
		if e != nil {
			return e
		}
		if len(results) != len(manifest.Members) {
			return errors.New("恢复报告不完整")
		}
		for _, r := range results {
			if object(r)["verified"] != true {
				return errors.New("数据恢复未通过一致性核验")
			}
		}
		verification := object(backup.Manifest)
		verification["restoreReport"] = results
		verification["verifiedAt"] = time.Now().UTC()
		if e = s.DB.Model(&backup).Updates(map[string]any{"verified_at": time.Now().UTC(), "manifest": JSON(verification)}).Error; e != nil {
			return e
		}
		return s.DB.Model(&model.Cluster{}).Where("id = ?", c.ID).Update("restore_verified_at", time.Now().UTC()).Error
	})
}
func (s *Service) scheduleBackups(ctx context.Context) {
	var policies []model.BackupPolicy
	s.DB.Where("enabled = ? AND next_run_at <= ?", true, time.Now()).Find(&policies)
	for _, p := range policies {
		policyCtx := ctx
		if p.APIKeyID > 0 {
			policyCtx = apikey.WithKey(ctx, p.APIKeyID)
		}
		if e := s.keyExecutionAllowed(policyCtx, p.CreatedBy); e != nil {
			s.DB.Model(&p).Update("enabled", false)
			s.event(p.ClusterID, p.CreatedBy, 0, "backup-policy", "warning", "备份策略的执行凭据已失效，策略已暂停，修复授权后重新启用")
			continue
		}
		schedule, e := cron.ParseStandard(p.Cron)
		if e != nil {
			continue
		}
		previous := p.NextRunAt
		next := schedule.Next(time.Now())
		r := s.DB.Model(&model.BackupPolicy{}).Where("id = ? AND next_run_at = ?", p.ID, previous).Update("next_run_at", next)
		if r.Error != nil || r.RowsAffected != 1 {
			continue
		}
		plan, e := s.Plan(policyCtx, p.ClusterID, p.CreatedBy, "backup", Parameters{Method: p.Method, RepositoryID: p.RepositoryID})
		if e != nil || !plan.Executable {
			s.event(p.ClusterID, p.CreatedBy, 0, "backup-policy-skipped", "warning", fmt.Sprintf("备份策略 #%d 本次预检未通过，未执行备份；请检查集群、基线与仓库状态", p.ID))
			continue
		}
		c, e := s.Cluster(p.ClusterID)
		if e != nil {
			continue
		}
		_, _ = s.Submit(policyCtx, c.ID, p.CreatedBy, plan.ID, fmt.Sprintf("policy-%d-%d", p.ID, previous.Unix()), c.Name)
	}
}
func (s *Service) SaveBackupPolicy(ctx context.Context, user uint, p model.BackupPolicy) (*model.BackupPolicy, error) {
	if e := s.keyExecutionAllowed(ctx, user); e != nil {
		return nil, e
	}
	p.APIKeyID = apikey.ID(ctx)
	if e := s.Authorize(user, p.ClusterID, "db_cluster_backup"); e != nil {
		return nil, e
	}
	if p.RetentionDays < 1 || p.RetentionDays > 3650 {
		return nil, errors.New("保留期范围为 1–3650 天")
	}
	schedule, e := cron.ParseStandard(p.Cron)
	if e != nil {
		return nil, errors.New("无效的五段 cron 表达式")
	}
	if p.Method != "logical" && p.Method != "physical" && p.Method != "incremental" && p.Method != "differential" && p.Method != "rdb" && p.Method != "wal-archive" && p.Method != "binlog-archive" {
		return nil, errors.New("备份方式无效")
	}
	cluster, e := s.Cluster(p.ClusterID)
	if e != nil {
		return nil, e
	}
	if !supportsBackupMethod(cluster.Engine, p.Method) {
		return nil, errors.New("备份方式与数据库引擎不匹配")
	}
	if p.RepositoryID > 0 && p.Enabled {
		var repo model.Repository
		if s.DB.First(&repo, p.RepositoryID).Error != nil || repo.GroupID != cluster.GroupID || repo.Status != "healthy" {
			return nil, errors.New("仓库必须属于当前业务分组且检查正常")
		}
	}
	if p.ID > 0 {
		var old model.BackupPolicy
		if s.DB.First(&old, p.ID).Error != nil || old.ClusterID != p.ClusterID {
			return nil, errors.New("策略不存在或不能转移集群")
		}
		p.CreatedAt = old.CreatedAt
	}
	p.CreatedBy = user
	next := schedule.Next(time.Now())
	p.NextRunAt = &next
	e = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if p.RepositoryID > 0 {
			var repo model.Repository
			if tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&repo, p.RepositoryID).Error != nil || repo.GroupID != cluster.GroupID || (p.Enabled && repo.Status != "healthy") {
				return errors.New("仓库不存在、分组不匹配或状态不可用")
			}
		}
		return tx.Save(&p).Error
	})
	if e == nil {
		s.event(p.ClusterID, user, 0, "backup-policy", "info", fmt.Sprintf("备份策略 #%d 已保存，启用=%t", p.ID, p.Enabled))
	}
	return &p, e
}
func (s *Service) storeBackup(ctx context.Context, c *model.Cluster, b *model.Backup, manifest map[string]any, op model.Operation) error {
	var repo model.Repository
	if s.DB.First(&repo, b.RepositoryID).Error != nil {
		return errors.New("备份仓库不存在")
	}
	return s.step(op, "上传并核验独立备份副本", func(log func(string)) error {
		var members []model.Member
		if json.Unmarshal(JSON(manifest["members"]), &members) != nil {
			return errors.New("备份成员无效")
		}
		extra := map[string]any{"operation_id": op.ID, "repository": repo, "backup_manifest": manifest}
		if repo.Kind == "s3" {
			credentials, e := s.RepositoryCredentials(repo.KeyID)
			if e != nil {
				return e
			}
			extra["repository_credentials"] = credentials
		} else {
			extra["restore_host_id"] = repo.HostID
		}
		results, e := s.ansible(ctx, c, members, "repository.yml", extra, log)
		if e != nil {
			return e
		}
		if len(results) != len(members) {
			return errors.New("独立仓库核验未完成，本地暂存不能替代远端成功")
		}
		for _, r := range results {
			if object(r)["verified"] != true {
				return errors.New("仓库对象校验失败")
			}
		}
		manifest["repositoryReport"] = results
		manifest["remoteComplete"] = true
		return nil
	})
}

func supportsBackupMethod(engine, method string) bool {
	switch engine {
	case "mysql":
		return method == "logical" || method == "physical" || method == "incremental" || method == "binlog-archive"
	case "postgresql":
		return method == "logical" || method == "physical" || method == "incremental" || method == "differential" || method == "wal-archive"
	case "redis":
		return method == "rdb"
	}
	return false
}
