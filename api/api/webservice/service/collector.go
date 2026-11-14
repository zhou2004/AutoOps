package service

import (
	"context"
	"dodevops-api/api/webservice/model"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sync"
	"time"
)

func (s *Service) lease(key, owner string, ttl time.Duration) bool {
	now := time.Now().UTC()
	s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.Lease{Key: key, ExpiresAt: now.Add(-time.Second)})
	r := s.DB.Model(&model.Lease{}).Where("`key` = ? AND expires_at < ?", key, now).Updates(map[string]any{"owner": owner, "expires_at": now.Add(ttl)})
	return r.Error == nil && r.RowsAffected == 1
}
func (s *Service) RunCollector(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collect(ctx)
		}
	}
}
func (s *Service) collect(ctx context.Context) {
	var instances []model.Resource
	if s.DB.Table(model.Tables["instances"]).Find(&instances).Error != nil {
		return
	}
	var wg sync.WaitGroup
	limit := make(chan struct{}, 4)
	for _, i := range instances {
		if ctx.Err() != nil {
			return
		}
		key := fmt.Sprintf("collect:%d", i.ID)
		if !s.lease(key, "collector", 4*time.Minute) {
			continue
		}
		limit <- struct{}{}
		wg.Add(1)
		go func(i model.Resource) {
			defer wg.Done()
			defer func() { <-limit }()
			defer s.DB.Model(&model.Lease{}).Where("`key` = ?", key).Update("expires_at", time.Now().UTC())
			var n int64
			s.DB.Model(&model.Lock{}).Where("`key` = ?", fmt.Sprintf("cluster:%d", i.ClusterID)).Count(&n)
			if n > 0 {
				return
			}
			src, e := s.Get("sources", i.SourceID)
			if e != nil {
				return
			}
			spec, e := decode[model.SourceSpec](src.Spec)
			if e != nil {
				return
			}
			o, e := (Nginx{}).Discover(ctx, s.Remote, spec)
			now := time.Now().UTC()
			status := "healthy"
			if e != nil {
				status = "unavailable"
				s.collectTraffic(ctx, i, src, spec, &o, e)
				s.DB.Table(model.Tables["instances"]).Where("id = ?", i.ID).Updates(map[string]any{"status": status, "collected_at": now})
				return
			}
			if spec.StatusURL != "" {
				raw, err := s.Remote.Run(ctx, RemoteRequest{Action: "metrics", Spec: spec})
				if err == nil {
					metrics, _ := decode[Observation](raw)
					o.Metrics = metrics.Metrics
				}
			}
			s.collectTraffic(ctx, i, src, spec, &o, nil)
			old, _ := decode[Observation](i.Observed)
			o.ExpectedDigest = old.ExpectedDigest
			if o.ExpectedDigest == "" && i.Management == "operate" {
				o.ExpectedDigest = old.Digest
			}
			if o.ExpectedDigest != "" && o.ExpectedDigest != o.Digest {
				status = "drifted"
			}
			if spec.HealthURL != "" {
				if _, e = s.Remote.Run(ctx, RemoteRequest{Action: "health", Spec: spec}); e != nil {
					status = "unavailable"
				}
			}
			s.DB.Model(&model.Lock{}).Where("`key` = ?", fmt.Sprintf("cluster:%d", i.ClusterID)).Count(&n)
			if n > 0 {
				return
			}
			s.DB.Table(model.Tables["instances"]).Where("id = ? AND revision = ? AND observed = ?", i.ID, i.Revision, i.Observed).Updates(map[string]any{"observed": JSON(o), "status": status, "collected_at": now})
			s.DB.Table(model.Tables["sources"]).Where("id = ?", src.ID).Updates(map[string]any{"status": status, "collected_at": now})
			if i.ClusterID > 0 {
				s.collectCertificates(ctx, i, spec)
			}
		}(i)
	}
	wg.Wait()
	var clusters []model.Resource
	s.DB.Table(model.Tables["clusters"]).Find(&clusters)
	for _, c := range clusters {
		var active int64
		if s.DB.Model(&model.Lock{}).Where("`key` = ?", fmt.Sprintf("cluster:%d", c.ID)).Count(&active).Error != nil || active > 0 {
			continue
		}
		var members []model.Resource
		s.DB.Table(model.Tables["instances"]).Where("cluster_id = ?", c.ID).Find(&members)
		s.refreshPublicationState(c.ID, members)
		status := "healthy"
		if len(members) == 0 {
			status = "unknown"
		}
		for _, m := range members {
			if m.CollectedAt == nil || time.Since(*m.CollectedAt) > 90*time.Second {
				status = "unknown"
				break
			}
			if m.Status != "healthy" {
				status = "degraded"
			}
		}
		spec, _ := decode[model.ClusterSpec](c.Spec)
		entries := []map[string]any{}
		if len(members) > 0 {
			source, err := s.Get("sources", members[0].SourceID)
			if err == nil {
				connector, _ := decode[model.SourceSpec](source.Spec)
				for _, entry := range spec.Entries {
					connector.HealthURL = entry.URL
					connector.HealthHost = entry.Host
					connector.ExpectedStatus = entry.ExpectedStatus
					_, err := s.Remote.Run(ctx, RemoteRequest{Action: "health", Spec: connector})
					entryStatus := "healthy"
					if err != nil {
						entryStatus = "unavailable"
						status = "degraded"
					}
					entries = append(entries, map[string]any{"name": entry.Name, "url": entry.URL, "status": entryStatus, "collectedAt": time.Now().UTC()})
				}
			}
		}
		observed := map[string]any{}
		_ = json.Unmarshal(c.Observed, &observed)
		observed["entries"] = entries
		if spec.VIP != nil {
			child, cancel := context.WithTimeout(ctx, 25*time.Second)
			targets, err := s.vipTopologyTargets(c.ID, spec.VIP, members)
			state := VIPState{Config: spec.VIP, Status: "unknown", Nodes: map[uint]VIPObservation{}, CheckedAt: time.Now().UTC()}
			if err == nil {
				state, err = s.inspectVIPState(child, spec.VIP, targets)
			}
			if err == nil {
				err = s.verifyVIPHTTP(child, spec.VIP, targets)
				if err != nil {
					state.Status = "degraded"
				}
			}
			if err != nil {
				state.Message = err.Error()
			}
			cancel()
			observed["vip"] = state
			if state.Status == "unknown" {
				status = "unknown"
			} else if state.Status != "healthy" && status != "unknown" {
				status = "degraded"
			}
		}
		if s.DB.Model(&model.Lock{}).Where("`key` = ?", fmt.Sprintf("cluster:%d", c.ID)).Count(&active).Error != nil || active > 0 {
			continue
		}
		s.DB.Table(model.Tables["clusters"]).Where("id = ? AND revision = ?", c.ID, c.Revision).Updates(map[string]any{"observed": JSON(observed), "status": status, "collected_at": time.Now().UTC()})
	}
	if s.lease("traffic-cleanup", "collector", time.Hour) {
		s.DB.Where("at < ?", time.Now().UTC().Add(-7*24*time.Hour)).Delete(&model.TrafficSample{})
		s.DB.Where("instance_id NOT IN (?)", s.DB.Table(model.Tables["instances"]).Select("id")).Delete(&model.TrafficState{})
	}
	if s.lease("audit-cleanup", "collector", 24*time.Hour) {
		cutoff := time.Now().UTC().Add(-90 * 24 * time.Hour)
		s.DB.Where("created_at < ?", cutoff).Delete(&model.Event{})
		s.DB.Where("expires_at < ?", cutoff).Delete(&model.Ticket{})
		_ = s.DB.Transaction(func(tx *gorm.DB) error {
			return tx.Where("created_at < ? AND operation_id IN (?)", cutoff, tx.Model(&model.Operation{}).Select("id").Where("status IN ?", []string{"succeeded", "failed", "rolled-back"})).Delete(&model.Step{}).Error
		})
	}
}

func (s *Service) collectCertificates(ctx context.Context, i model.Resource, spec model.SourceSpec) {
	var bindings []model.Resource
	if s.DB.Table(model.Tables["certificate-bindings"]).Where("cluster_id = ?", i.ClusterID).Find(&bindings).Error != nil {
		return
	}
	for _, binding := range bindings {
		b, e := decode[model.BindingSpec](binding.Spec)
		if e != nil {
			continue
		}
		raw, e := s.Remote.Run(ctx, RemoteRequest{Action: "tls", Spec: spec, TLS: []TLSProbe{{BindingID: binding.ID, Host: b.VerifyHost, Port: b.VerifyPort, ServerName: b.ServerName}}})
		fingerprint := ""
		if e == nil {
			var result struct {
				Certificates []struct {
					Fingerprint string `json:"fingerprint"`
				} `json:"certificates"`
			}
			if decodeErr := json.Unmarshal(raw, &result); decodeErr == nil && len(result.Certificates) > 0 {
				fingerprint = result.Certificates[0].Fingerprint
			}
		}
		var revision model.Revision
		if fingerprint != "" {
			s.DB.Where("kind = ? AND resource_id = ? AND digest = ?", "certificate", b.CertificateID, fingerprint).First(&revision)
		}
		_ = s.DB.Transaction(func(tx *gorm.DB) error {
			var lock model.Lock
			if tx.Where("`key` = ?", fmt.Sprintf("cluster:%d", i.ClusterID)).First(&lock).Error == nil {
				return nil
			}
			var current model.Resource
			if e := tx.Table(model.Tables["certificate-bindings"]).Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, binding.ID).Error; e != nil {
				return e
			}
			state, _ := decode[map[string]any](current.Observed)
			if state == nil {
				state = map[string]any{}
			}
			nodes, _ := state["nodes"].(map[string]any)
			if nodes == nil {
				nodes = map[string]any{}
			}
			nodes[fmt.Sprint(i.ID)] = map[string]any{"versionId": revision.ID, "fingerprint": fingerprint, "verifiedAt": time.Now().UTC()}
			state["nodes"] = nodes
			var certificate model.Resource
			if e := tx.Table(model.Tables["certificates"]).First(&certificate, b.CertificateID).Error; e != nil {
				return e
			}
			cs, _ := decode[model.CertificateSpec](certificate.Spec)
			expected := b.VersionID
			if expected == 0 {
				expected = cs.CurrentVersionID
			}
			state["desiredVersionId"] = expected
			local := *s
			local.DB = tx
			return tx.Table(model.Tables["certificate-bindings"]).Where("id = ?", binding.ID).Updates(map[string]any{"observed": JSON(state), "status": local.bindingStatus(binding.ClusterID, nodes, expected)}).Error
		})
	}
}
