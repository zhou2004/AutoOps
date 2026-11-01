package service

import (
	"context"
	"dodevops-api/api/database/model"
	"dodevops-api/pkg/apikey"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

func (s *Service) AuditPolicy() model.AuditPolicy {
	p := model.AuditPolicy{ID: 1, RetentionDays: 90}
	s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&p)
	s.DB.First(&p, 1)
	return p
}
func (s *Service) auditAdmin(ctx context.Context, user uint) error {
	identity, e := apikey.EffectiveIdentity(ctx, s.DB, user)
	if e != nil || !s.IsAdmin(user) || !identity.Allows("database:write") {
		return ErrForbidden
	}
	return nil
}
func (s *Service) SaveAuditPolicy(ctx context.Context, user uint, days int) (*model.AuditPolicy, error) {
	if e := s.auditAdmin(ctx, user); e != nil {
		return nil, e
	}
	if days < 0 || days > 3650 {
		return nil, errors.New("保存周期应为 0（永久）或 1–3650 天")
	}
	p := model.AuditPolicy{ID: 1, RetentionDays: days, UpdatedBy: user}
	e := s.DB.Save(&p).Error
	return &p, e
}

// Erase log contents, preserving compact operation receipts for idempotency,
// native-event deduplication and backup/restoration references.
func eraseAudit(tx *gorm.DB, kind string, id uint) error {
	now := time.Now()
	if kind == "operations" {
		var op model.Operation
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&op, id).Error; e != nil {
			return e
		}
		if op.AuditDeletedAt != nil {
			return nil
		}
		if op.Status != "succeeded" && op.Status != "failed" && op.Status != "cancelled" {
			return errors.New("运行中或结果待核验的任务不能删除")
		}
		var locks int64
		if e := tx.Model(&model.OperationLock{}).Where("operation_id = ?", id).Count(&locks).Error; e != nil {
			return e
		}
		if locks > 0 {
			return errors.New("任务仍持有集群锁，不能删除")
		}
		if e := tx.Where("operation_id = ?", id).Delete(&model.OperationStep{}).Error; e != nil {
			return e
		}
		return tx.Model(&op).Updates(map[string]any{"audit_deleted_at": now, "message": "", "parameters": JSON(map[string]any{})}).Error
	}
	if kind == "events" {
		var event model.Event
		if e := tx.First(&event, id).Error; e != nil {
			return e
		}
		if event.OperationID > 0 {
			var op model.Operation
			if tx.First(&op, event.OperationID).Error == nil && op.Status != "succeeded" && op.Status != "failed" && op.Status != "cancelled" {
				return errors.New("事件关联任务仍未结束")
			}
		}
		return tx.Model(&event).Updates(map[string]any{"audit_deleted_at": now, "message": "", "details": JSON(map[string]any{})}).Error
	}
	return errors.New("不支持的审计类型")
}
func (s *Service) DeleteAudit(ctx context.Context, user uint, kind string, id uint, confirmed bool) error {
	if e := s.auditAdmin(ctx, user); e != nil {
		return e
	}
	if !confirmed {
		return errors.New("请确认删除日志")
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if e := eraseAudit(tx, kind, id); e != nil {
			return e
		}
		var clusterID uint
		table := "database_event"
		if kind == "operations" {
			table = "database_operation"
		}
		if e := tx.Table(table).Where("id = ?", id).Pluck("cluster_id", &clusterID).Error; e != nil {
			return e
		}
		return tx.Create(&model.Event{ClusterID: clusterID, UserID: user, Kind: "audit_deleted", Severity: "info", Message: "管理员删除审计日志", Details: JSON(map[string]any{"kind": kind, "id": id})}).Error
	})
}
func (s *Service) PruneAudit(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	p := s.AuditPolicy()
	if p.RetentionDays == 0 {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -p.RetentionDays)
	var ops []model.Operation
	s.DB.Where("audit_deleted_at IS NULL AND finished_at < ? AND status IN ?", cutoff, []string{"succeeded", "failed", "cancelled"}).Limit(200).Find(&ops)
	for _, op := range ops {
		if ctx.Err() != nil {
			return
		}
		_ = s.DB.Transaction(func(tx *gorm.DB) error { return eraseAudit(tx, "operations", op.ID) })
	}
	var events []model.Event
	s.DB.Where("audit_deleted_at IS NULL AND created_at < ?", cutoff).Limit(200).Find(&events)
	for _, event := range events {
		if ctx.Err() != nil {
			return
		}
		_ = s.DB.Transaction(func(tx *gorm.DB) error { return eraseAudit(tx, "events", event.ID) })
	}
}
