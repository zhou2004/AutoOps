package service

import (
	"context"
	"dodevops-api/api/database/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"sync"
	"time"
)

// Collection is read-only and independent of task execution. Multiple API/worker
// processes coordinate per cluster; a slow/unreachable cluster cannot starve others.
func (s *Service) RunCollector(ctx context.Context) {
	s = New(s.DB.Session(&gorm.Session{Logger: logger.New(log.New(os.Stderr, "database collector: ", log.LstdFlags), logger.Config{LogLevel: logger.Warn, IgnoreRecordNotFoundError: true, SlowThreshold: time.Second})}))
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	for {
		s.collectAll(ctx)
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}
func (s *Service) acquireCollection(id uint, owner string) bool {
	s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.CollectionLease{ClusterID: id, Until: time.Now().Add(-time.Hour)})
	r := s.DB.Model(&model.CollectionLease{}).Where("cluster_id = ? AND `until` < ?", id, time.Now()).Updates(map[string]any{"owner": owner, "until": time.Now().Add(100 * time.Second)})
	return r.Error == nil && r.RowsAffected == 1
}

// Queued jobs have not touched remote state. Isolated recovery never changes the
// source topology. Neither should hide live health behind their mutation lock.
// Preflight still blocks new collectors while the worker drains the prior lease.
// Unknown outcomes retain their lock without disabling read-only health checks.
func (s *Service) collectionAllowed(id uint) bool {
	var blocked int64
	err := s.DB.Model(&model.OperationLock{}).
		Joins("LEFT JOIN database_operation op ON op.id = database_operation_lock.operation_id AND op.cluster_id = database_operation_lock.cluster_id").
		Where("database_operation_lock.cluster_id = ? AND (op.status IS NULL OR NOT (op.status IN ? OR (op.status IN ? AND COALESCE(op.action, '') IN ?)))", id, []string{"queued", "unknown"}, []string{"running", "verifying"}, []string{"restore", "verify-backup"}).
		Count(&blocked).Error
	return err == nil && blocked == 0
}

func (s *Service) collectAll(ctx context.Context) {
	var clusters []model.Cluster
	if s.DB.Where("status <> ?", "planned").Find(&clusters).Error != nil {
		return
	}
	var wg sync.WaitGroup
	limit := make(chan struct{}, 6)
	for _, c := range clusters {
		if ctx.Err() != nil {
			break
		}
		if collectedFresh(c.CollectedAt, 15*time.Second) {
			continue
		}
		owner := token()
		if !s.acquireCollection(c.ID, owner) {
			continue
		}
		wg.Add(1)
		go func(id uint, owner string) {
			defer wg.Done()
			limit <- struct{}{}
			defer func() { <-limit }()
			defer s.DB.Model(&model.CollectionLease{}).Where("cluster_id = ? AND owner = ?", id, owner).Update("until", time.Now())
			if !s.collectionAllowed(id) {
				return
			}
			capture, cancel := context.WithTimeout(ctx, 70*time.Second)
			defer cancel()
			fresh, e := s.Refresh(capture, id)
			if e != nil {
				return
			}
			s.collectVIP(capture, fresh)
			s.collectNativeEvents(capture, fresh)
			var latest model.StorageSnapshot
			s.DB.Where("cluster_id = ?", id).Order("id DESC").First(&latest)
			if latest.ID == 0 || time.Since(latest.CollectedAt) > 5*time.Minute {
				storage, stop := context.WithTimeout(capture, 25*time.Second)
				defer stop()
				_ = s.collectStorageSnapshots(storage, fresh, func(string) {})
				s.collectConfigurationStatus(storage, fresh)
			}
		}(c.ID, owner)
	}
	wg.Wait()
	s.PruneAudit(ctx)
}
