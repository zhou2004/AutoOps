package service

import (
	"context"
	"dodevops-api/api/monitor/model"
	web "dodevops-api/api/webservice/model"
	webservice "dodevops-api/api/webservice/service"
	"dodevops-api/common"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"strconv"
	"time"
)

func (s *monitorAlertRuleService) evalWebRule(ctx context.Context, rdb *redis.Client, rule *model.MonitorAlertRule) {
	var group model.MonitorAlertGroupRule
	if common.GetDB().First(&group, rule.GroupID).Error != nil {
		return
	}
	var results []struct {
		Metric map[string]string `json:"metric"`
		Value  []interface{}     `json:"value"`
	}
	scope := []uint{}
	if group.CmdbGroupID > 0 {
		var e error
		scope, e = webservice.New(common.GetDB()).GroupIDs(group.CmdbGroupID)
		if e != nil {
			return
		}
	}
	add := func(r web.Resource, kind string, values map[string]float64) {
		labels := map[string]string{"monitor_source_id": strconv.FormatUint(uint64(r.ID), 10), "web_cluster_id": strconv.FormatUint(uint64(r.ClusterID), 10), "business_group_id": strconv.FormatUint(uint64(r.GroupID), 10), "engine": r.Engine, "name": r.Name, "metric_kind": kind, "environment": r.Environment}
		value := "0"
		if matchDatabaseRule(rule, labels, values) {
			value = "1"
		}
		results = append(results, struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		}{labels, []interface{}{float64(time.Now().Unix()), value}})
	}
	for _, kind := range []string{"instances", "certificates"} {
		var rows []web.Resource
		q := common.GetDB().Table(web.Tables[kind])
		if len(scope) > 0 {
			q = q.Where("group_id IN ?", scope)
		}
		if q.Find(&rows).Error != nil {
			continue
		}
		for _, r := range rows {
			v := map[string]float64{}
			if kind == "instances" {
				v = webInstanceRuleValues(r, time.Now())
			} else {
				v["renewal_failed"] = 0
				v["certificate_deployment_lag"] = 0
				var issuance web.Operation
				if common.GetDB().Where("target_type = ? AND target_id = ? AND action IN ?", "certificates", r.ID, []string{"issue", "renew"}).Order("id DESC").First(&issuance).Error == nil && (issuance.Status == "failed" || issuance.Status == "unknown") {
					v["renewal_failed"] = 1
				}
				var cs web.CertificateSpec
				json.Unmarshal(r.Spec, &cs)
				var bindings []web.Resource
				common.GetDB().Table(web.Tables["certificate-bindings"]).Find(&bindings)
				for _, binding := range bindings {
					var b web.BindingSpec
					json.Unmarshal(binding.Spec, &b)
					if b.CertificateID != r.ID {
						continue
					}
					expected := b.VersionID
					if expected == 0 {
						expected = cs.CurrentVersionID
					}
					var observed struct {
						Nodes map[uint]struct {
							VersionID uint `json:"versionId"`
						} `json:"nodes"`
					}
					json.Unmarshal(binding.Observed, &observed)
					var nodes []web.Resource
					common.GetDB().Table(web.Tables["instances"]).Where("cluster_id = ?", binding.ClusterID).Find(&nodes)
					for _, node := range nodes {
						if observed.Nodes[node.ID].VersionID != expected {
							v["certificate_deployment_lag"] = 1
						}
					}
				}
				var rev web.Revision
				if cs.CurrentVersionID > 0 && common.GetDB().First(&rev, cs.CurrentVersionID).Error == nil {
					var meta struct {
						NotAfter time.Time `json:"notAfter"`
					}
					if json.Unmarshal(rev.Content, &meta) == nil && !meta.NotAfter.IsZero() {
						v["remaining_days"] = time.Until(meta.NotAfter).Hours() / 24
					}
				}
			}
			add(r, kind, v)
		}
	}
	var clusters []web.Resource
	q := common.GetDB().Table(web.Tables["clusters"])
	if len(scope) > 0 {
		q = q.Where("group_id IN ?", scope)
	}
	q.Find(&clusters)
	for _, r := range clusters {
		var op web.Operation
		values := map[string]float64{"publish_failed": 0}
		if common.GetDB().Where("cluster_id = ?", r.ID).Order("id DESC").First(&op).Error == nil && (op.Status == "failed" || op.Status == "rolled-back" || op.Status == "unknown") {
			values["publish_failed"] = 1
		}
		r.ClusterID = r.ID
		add(r, "operation", values)
	}
	duration, _ := time.ParseDuration(rule.ForDuration)
	s.processInternalEval(ctx, rdb, rule, results, duration)
}

// A failed discovery retains its last observation for diagnosis, not for fresh
// traffic alerts. Missing samples remain absent instead of becoming zero.
func webInstanceRuleValues(r web.Resource, now time.Time) map[string]float64 {
	v := map[string]float64{"state_unknown": 0}
	if r.CollectedAt == nil || now.Sub(*r.CollectedAt) > 90*time.Second || r.CollectedAt.After(now.Add(time.Second)) {
		v["state_unknown"] = 1
		return v
	}
	v["instance_unavailable"], v["configuration_drift"] = 0, 0
	if r.Status == "unavailable" {
		v["instance_unavailable"] = 1
		return v
	}
	if r.Status == "drifted" {
		v["configuration_drift"] = 1
	}
	var observation webservice.Observation
	if json.Unmarshal(r.Observed, &observation) == nil {
		for name, value := range observation.Metrics {
			v[name] = value
		}
	}
	return v
}
