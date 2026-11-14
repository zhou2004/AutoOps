package controller

import (
	"dodevops-api/api/webservice/model"
	"dodevops-api/api/webservice/service"
	"dodevops-api/common/result"
	"dodevops-api/pkg/websecret"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pmezard/go-difflib/difflib"
	"sort"
	"time"
)

func (c *Controller) RevisionDiff(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	var next, previous model.Revision
	if e := c.S.DB.First(&next, id(ctx)).Error; e != nil {
		fail(ctx, e)
		return
	}
	if e := c.S.Asset(u, "web_cluster", next.ClusterID, "web_config_view"); e != nil {
		fail(ctx, e)
		return
	}
	if next.Kind != "configuration" {
		fail(ctx, errors.New("请选择配置版本"))
		return
	}
	if compare := queryID(ctx, "compareTo"); compare > 0 {
		if e := c.S.DB.First(&previous, compare).Error; e != nil {
			fail(ctx, e)
			return
		}
		if previous.ClusterID != next.ClusterID || previous.Kind != next.Kind {
			fail(ctx, service.ErrForbidden)
			return
		}
	}
	var before, after model.ConfigSpec
	if len(previous.Content) > 0 {
		json.Unmarshal(previous.Content, &before)
	}
	json.Unmarshal(next.Content, &after)
	changes := []gin.H{}
	ids := map[uint]bool{}
	for id := range before.Nodes {
		ids[id] = true
	}
	for id := range after.Nodes {
		ids[id] = true
	}
	for iid := range ids {
		old, newFiles := map[string]string{}, map[string]string{}
		paths := map[string]bool{}
		for _, f := range before.Nodes[iid] {
			old[f.Path] = f.Content
			paths[f.Path] = true
		}
		for _, f := range after.Nodes[iid] {
			newFiles[f.Path] = f.Content
			paths[f.Path] = true
		}
		ordered := []string{}
		for p := range paths {
			ordered = append(ordered, p)
		}
		sort.Strings(ordered)
		for _, p := range ordered {
			if old[p] == newFiles[p] {
				continue
			}
			text, e := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{A: difflib.SplitLines(old[p]), B: difflib.SplitLines(newFiles[p]), FromFile: fmt.Sprintf("%d:%s", previous.ID, p), ToFile: fmt.Sprintf("%d:%s", next.ID, p), Context: 3})
			if e != nil {
				fail(ctx, e)
				return
			}
			changes = append(changes, gin.H{"instanceId": iid, "path": p, "diff": text})
		}
	}
	result.Success(ctx, gin.H{"revisionId": next.ID, "compareTo": previous.ID, "files": changes})
}

func (c *Controller) WorkerStatus(ctx *gin.Context) {
	var workers []model.Lease
	if e := c.S.DB.Where("`key` LIKE ? AND expires_at > ?", "worker:%", time.Now().UTC()).Find(&workers).Error; e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, gin.H{"encryptionReady": websecret.Ready() == nil, "online": len(workers) > 0, "count": len(workers), "checkedAt": time.Now().UTC()})
}

func (c *Controller) Events(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	var rows []model.Event
	q := c.S.DB.Order("id DESC")
	if cid := queryID(ctx, "clusterId"); cid > 0 {
		q = q.Where("cluster_id = ?", cid)
	}
	if cursor := queryID(ctx, "before"); cursor > 0 {
		q = q.Where("id < ?", cursor)
	}
	// Filter before counting and pagination; terminal and query events use source scopes.
	if e := q.Find(&rows).Error; e != nil {
		fail(ctx, e)
		return
	}
	out := []model.Event{}
	for _, event := range rows {
		allowed := false
		if event.SourceID > 0 {
			allowed = c.S.Asset(u, "web_source", event.SourceID, "web_audit") == nil
		} else if event.ClusterID > 0 {
			allowed = c.S.Asset(u, "web_cluster", event.ClusterID, "web_audit") == nil
		} else {
			allowed = c.S.Admin(u)
		}
		if allowed {
			out = append(out, event)
		}
	}
	total := len(out)
	if len(out) > 100 {
		out = out[:100]
	}
	result.Success(ctx, gin.H{"list": out, "total": total})
}

func (c *Controller) PlanPreview(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	var plan model.Plan
	if e := c.S.DB.First(&plan, id(ctx)).Error; e != nil {
		fail(ctx, e)
		return
	}
	if plan.UserID != u {
		fail(ctx, service.ErrForbidden)
		return
	}
	target, e := c.S.Get(plan.TargetType, plan.TargetID)
	if e == nil {
		e = c.S.Authorize(u, plan.TargetType, target, "web_config_view")
	}
	if e != nil {
		fail(ctx, e)
		return
	}
	views, e := c.S.PlanPreview(plan)
	if e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, views)
}
