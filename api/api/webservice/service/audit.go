package service

import (
	"dodevops-api/api/webservice/model"
	"fmt"
	"strings"
	"time"
)

// Audit applies the same resource filters as the pages, and paginates only
// after asset authorization. Unfinished operations are never hidden by age.
func (s *Service) Audit(u uint, f Filter) ([]model.Operation, int, error) {
	var rows []model.Operation
	q := s.DB.Where("created_at >= ? OR status IN ?", time.Now().UTC().Add(-90*24*time.Hour), []string{"queued", "preflight", "running", "verifying", "unknown"})
	if f.ClusterID > 0 {
		q = q.Where("cluster_id = ?", f.ClusterID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if e := q.Order("id DESC").Find(&rows).Error; e != nil {
		return nil, 0, e
	}
	groups := map[uint]bool{}
	if f.GroupID > 0 {
		ids, e := s.GroupIDs(f.GroupID)
		if e != nil {
			return nil, 0, e
		}
		for _, id := range ids {
			groups[id] = true
		}
	}
	filtered := []model.Operation{}
	for _, op := range rows {
		r, e := s.Get(op.TargetType, op.TargetID)
		if e != nil || s.Authorize(u, op.TargetType, r, "web_audit") != nil {
			continue
		}
		if f.GroupID > 0 && !groups[r.GroupID] || f.Engine != "" && f.Engine != r.Engine || f.Environment != "" && f.Environment != r.Environment {
			continue
		}
		if f.Name != "" && !strings.Contains(strings.ToLower(fmt.Sprintf("%s %s %s #%d", r.Name, op.Action, actions[op.Action], op.ID)), strings.ToLower(f.Name)) {
			continue
		}
		op.TargetName = r.Name
		filtered = append(filtered, op)
	}
	total := len(filtered)
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Size < 1 || f.Size > 100 {
		f.Size = 20
	}
	start := min((f.Page-1)*f.Size, total)
	return filtered[start:min(start+f.Size, total)], total, nil
}
