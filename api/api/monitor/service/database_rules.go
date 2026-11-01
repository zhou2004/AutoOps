package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	cmdb "dodevops-api/api/cmdb/model"
	database "dodevops-api/api/database/model"
	"dodevops-api/api/database/policy"
	"dodevops-api/api/monitor/model"
	"dodevops-api/common"
	"github.com/go-redis/redis/v8"
)

// Only explicitly enabled rules use the existing notification engine. Collectors
// never send notifications or create a second alert state machine.
func (s *monitorAlertRuleService) evalDatabaseRule(ctx context.Context, rdb *redis.Client, rule *model.MonitorAlertRule) {
	var group model.MonitorAlertGroupRule
	if common.GetDB().WithContext(ctx).First(&group, rule.GroupID).Error != nil {
		return
	}
	q := common.GetDB().WithContext(ctx).Preload("Members").Where("status <> ?", "planned")
	if group.CmdbGroupID > 0 {
		var groups []cmdb.CmdbGroup
		if common.GetDB().Find(&groups).Error != nil {
			return
		}
		ids := map[uint]bool{group.CmdbGroupID: true}
		for changed := true; changed; {
			changed = false
			for _, g := range groups {
				if ids[g.ParentID] && !ids[g.ID] {
					ids[g.ID] = true
					changed = true
				}
			}
		}
		list := []uint{}
		for id := range ids {
			list = append(list, id)
		}
		q = q.Where("group_id IN ?", list)
	}
	var clusters []database.Cluster
	if q.Find(&clusters).Error != nil {
		return
	}
	var results []struct {
		Metric map[string]string `json:"metric"`
		Value  []interface{}     `json:"value"`
	}
	now := time.Now()
	appendResult := func(labels map[string]string, values map[string]float64) {
		value := "0"
		if matchDatabaseRule(rule, labels, values) {
			value = "1"
		}
		results = append(results, struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		}{labels, []interface{}{float64(now.Unix()), value}})
	}
	for _, c := range clusters {
		labels := map[string]string{"monitor_source_id": strconv.FormatUint(uint64(c.ID), 10), "db_cluster_id": strconv.FormatUint(uint64(c.ID), 10), "engine": c.Engine, "business_group_id": strconv.FormatUint(uint64(c.GroupID), 10), "environment": c.Environment, "name": c.Name, "instance_id": "0", "role": "cluster", "metric_kind": "health"}
		values := map[string]float64{"state_unknown": 0}
		stale := c.CollectedAt == nil || now.Sub(*c.CollectedAt) > 90*time.Second
		if stale {
			values["state_unknown"] = 1
		} else {
			values["placement_unsafe"] = 0
			if len(policy.RedisPlacementWarnings(&c)) > 0 {
				values["placement_unsafe"] = 1
			}
			var health map[string]string
			json.Unmarshal(c.Health, &health)
			for _, key := range []string{"database", "replication", "control", "endpoint", "backup"} {
				values[key+"_unhealthy"] = 0
				if health[key] != "healthy" && health[key] != "not-applicable" {
					values[key+"_unhealthy"] = 1
				}
			}
		}
		if c.ConfigurationCollectedAt != nil && now.Sub(*c.ConfigurationCollectedAt) < 10*time.Minute {
			values["configuration_drift"], values["configuration_pending_restart"], values["configuration_unknown"] = 0, 0, 0
			if c.ConfigurationStatus == "drifted" {
				values["configuration_drift"] = 1
			}
			if c.ConfigurationStatus == "pending-restart" {
				values["configuration_pending_restart"] = 1
			}
			if c.ConfigurationStatus == "unknown" {
				values["configuration_unknown"] = 1
			}
		} else {
			values["configuration_unknown"] = 1
		}
		appendResult(labels, values)
		for _, m := range c.Members {
			if m.Kind != "data" {
				continue
			}
			nodeLabels := map[string]string{}
			for k, v := range labels {
				nodeLabels[k] = v
			}
			nodeLabels["instance_id"] = strconv.FormatUint(uint64(m.ID), 10)
			nodeLabels["role"] = m.Role
			nodeLabels["metric_kind"] = "instance"
			values = map[string]float64{"state_unknown": 0}
			if stale || m.CollectedAt == nil || now.Sub(*m.CollectedAt) > 90*time.Second {
				values["state_unknown"] = 1
			} else {
				var metrics map[string]any
				json.Unmarshal(m.Metrics, &metrics)
				values["instance_unavailable"] = 0
				if m.State != "healthy" {
					values["instance_unavailable"] = 1
				}
				for _, key := range []string{"lagSeconds", "used_memory", "used_memory_rss", "maxmemory", "mem_fragmentation_ratio", "connected_clients", "Threads_connected", "max_connections"} {
					if v, ok := metrics[key]; ok {
						raw, _ := json.Marshal(v)
						n, e := strconv.ParseFloat(strings.Trim(string(raw), "\""), 64)
						if e == nil {
							values[key] = n
						}
					}
				}
				values["semisync_degraded"] = 0
				if c.Engine == "mysql" && m.Role == "primary" {
					enabled, active := false, false
					for k, v := range metrics {
						key := strings.ToLower(k)
						if strings.HasPrefix(key, "rpl_semi_sync_") {
							on := strings.EqualFold(fmt.Sprint(v), "ON") || fmt.Sprint(v) == "1"
							if strings.HasSuffix(key, "_enabled") {
								enabled = enabled || on
							}
							if key == "rpl_semi_sync_master_status" || key == "rpl_semi_sync_source_status" {
								active = active || on
							}
						}
					}
					if enabled && !active {
						values["semisync_degraded"] = 1
					}
				}
				values["persistence_failed"] = 0
				for _, key := range []string{"rdb_last_bgsave_status", "aof_last_write_status", "aof_last_bgrewrite_status"} {
					if metrics[key] == "err" {
						values["persistence_failed"] = 1
					}
				}
				if limit := values["maxmemory"]; limit > 0 {
					values["memory_used_percent"] = values["used_memory"] * 100 / limit
				}
			}
			appendResult(nodeLabels, values)
		}
		var snapshots []database.StorageSnapshot
		latest := common.GetDB().Model(&database.StorageSnapshot{}).Select("MAX(id)").Where("cluster_id = ? AND kind = ?", c.ID, "filesystem").Group("filesystem_id")
		if common.GetDB().Where("id IN (?)", latest).Find(&snapshots).Error == nil {
			for _, snapshot := range snapshots {
				fsLabels := map[string]string{}
				for k, v := range labels {
					fsLabels[k] = v
				}
				fsLabels["instance_id"] = strconv.FormatUint(uint64(snapshot.MemberID), 10)
				fsLabels["metric_kind"] = "filesystem"
				fsLabels["filesystem_id"] = snapshot.FilesystemID
				values = map[string]float64{"state_unknown": 0}
				if now.Sub(snapshot.CollectedAt) > 10*time.Minute {
					values["state_unknown"] = 1
				} else if snapshot.TotalBytes > 0 {
					values["disk_used_percent"] = float64(snapshot.UsedBytes) * 100 / float64(snapshot.TotalBytes)
					thresholds := policy.StorageThresholds(common.GetDB(), c.GroupID)
					values["disk_warning"], values["disk_critical"] = 0, 0
					if values["disk_used_percent"] >= float64(thresholds.WarningPercent) {
						values["disk_warning"] = 1
					}
					if values["disk_used_percent"] >= float64(thresholds.CriticalPercent) {
						values["disk_critical"] = 1
					}
					var detail map[string]float64
					json.Unmarshal(snapshot.Details, &detail)
					if detail["inodes"] > 0 {
						values["inode_used_percent"] = (detail["inodes"] - detail["freeInodes"]) * 100 / detail["inodes"]
					}
				}
				appendResult(fsLabels, values)
			}
		}
	}
	duration, _ := time.ParseDuration(rule.ForDuration)
	s.processInternalEval(ctx, rdb, rule, results, duration)
}

func matchDatabaseRule(rule *model.MonitorAlertRule, labels map[string]string, values map[string]float64) bool {
	var constraints map[string]string
	if rule.Constraints != "" && json.Unmarshal([]byte(rule.Constraints), &constraints) != nil {
		return false
	}
	for key, want := range constraints {
		if want != "" && labels[key] != want {
			return false
		}
	}
	parts := strings.Fields(rule.Expr)
	if len(parts) != 3 {
		return false
	}
	value, exists := values[parts[0]]
	if !exists {
		return false
	}
	threshold, e := strconv.ParseFloat(parts[2], 64)
	if e != nil {
		return false
	}
	switch parts[1] {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "=", "==":
		return value == threshold
	case "!=":
		return value != threshold
	}
	return false
}
