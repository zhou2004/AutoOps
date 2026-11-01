package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"dodevops-api/api/database/model"
)

type ParameterDefinition struct {
	Name        string `json:"name"`
	Mode        string `json:"mode"`
	Description string `json:"description"`
	Min, Max    int64
}

var definitions = map[string][]ParameterDefinition{
	"mysql":      {{"max_connections", "dynamic", "最大连接数", 10, 10000}, {"max_connect_errors", "dynamic", "连接错误上限", 10, 1000000}, {"wait_timeout", "dynamic", "非交互连接空闲超时（秒）", 1, 31536000}, {"interactive_timeout", "dynamic", "交互连接空闲超时（秒）", 1, 31536000}, {"long_query_time", "dynamic", "慢查询阈值（秒）", 0, 3600}, {"slow_query_log", "dynamic", "慢查询开关（0/1）", 0, 1}},
	"postgresql": {{"max_connections", "restart", "最大连接数", 10, 10000}, {"work_mem", "reload", "每个排序或哈希操作内存（kB）", 64, 1048576}, {"log_min_duration_statement", "reload", "慢查询阈值（毫秒，-1 禁用）", -1, 3600000}, {"statement_timeout", "reload", "语句超时（毫秒，0 禁用）", 0, 86400000}},
	"redis":      {{"maxmemory", "dynamic", "最大内存（字节，只允许增加）", 0, 1 << 50}, {"appendfsync", "dynamic", "AOF 刷盘策略（always/everysec）", 0, 0}, {"slowlog-log-slower-than", "dynamic", "慢命令阈值（微秒）", 0, 3600000000}, {"slowlog-max-len", "dynamic", "慢命令最大保留条数", 1, 100000}},
}

func validateChanges(engine string, changes map[string]string) error {
	if len(changes) == 0 || len(changes) > 20 {
		return errors.New("配置项数量无效")
	}
	for key, value := range changes {
		found := false
		for _, d := range definitions[engine] {
			if d.Name != key {
				continue
			}
			found = true
			if key == "appendfsync" {
				if value != "always" && value != "everysec" {
					return errors.New("仅支持 always 或 everysec")
				}
			} else {
				n, e := strconv.ParseFloat(value, 64)
				if e != nil || math.IsNaN(n) || math.IsInf(n, 0) || !regexp.MustCompile(`^-?[0-9]+(?:\.[0-9]+)?$`).MatchString(value) || n < float64(d.Min) || n > float64(d.Max) || (key != "long_query_time" && n != math.Trunc(n)) {
					return fmt.Errorf("%s 参数值超出允许范围", key)
				}
			}
		}
		if !found {
			return fmt.Errorf("参数 %s 不在受控模板范围内", key)
		}
	}
	return nil
}
func (s *Service) Configurations(ctx context.Context, c *model.Cluster) (map[string]any, error) {
	out := []map[string]any{}
	for _, m := range c.Members {
		if m.Kind != "data" {
			continue
		}
		values := map[string]any{}
		pending := map[string]bool{}
		pendingValues := map[string]string{}
		var failure string
		if c.Engine == "redis" {
			client, e := s.redis(c, m)
			if e != nil {
				failure = e.Error()
			} else {
				for _, d := range definitions[c.Engine] {
					v, e := client.ConfigGet(ctx, d.Name).Result()
					if e == nil && len(v) == 2 {
						values[d.Name] = v[1]
					} else {
						failure = "参数读取失败"
					}
				}
				client.Close()
			}
		}
		if c.Engine == "mysql" || c.Engine == "postgresql" {
			db, e := s.sql(ctx, c, m, "")
			if e != nil {
				failure = e.Error()
			} else {
				for _, d := range definitions[c.Engine] {
					var value string
					q := "SELECT @@global." + d.Name
					if c.Engine == "postgresql" {
						var value string
						var restart bool
						if e := db.QueryRowContext(ctx, "SELECT setting,pending_restart FROM pg_settings WHERE name=$1", d.Name).Scan(&value, &restart); e == nil {
							values[d.Name] = value
							pending[d.Name] = restart
							if restart {
								var saved string
								if e := db.QueryRowContext(ctx, "SELECT setting FROM pg_file_settings WHERE name=$1 ORDER BY seqno DESC LIMIT 1", d.Name).Scan(&saved); e == nil {
									pendingValues[d.Name] = saved
								}
							}
						} else {
							failure = "参数读取失败"
						}
						continue
					}
					if db.QueryRowContext(ctx, q).Scan(&value) == nil {
						values[d.Name] = value
					} else {
						failure = "参数读取失败"
					}
				}
				db.Close()
			}
		}
		out = append(out, map[string]any{"memberId": m.ID, "name": m.Name, "values": values, "pendingRestart": pending, "pendingValues": pendingValues, "error": failure})
	}
	var revisions []model.ConfigRevision
	s.DB.Where("cluster_id = ?", c.ID).Order("id DESC").Limit(30).Find(&revisions)
	desired := map[string]string{}
	for _, r := range revisions {
		if r.MemberID != 0 {
			continue
		}
		for key, v := range object(r.Desired) {
			if _, seen := desired[key]; !seen {
				desired[key] = text(v)
			}
		}
	}
	for _, row := range out {
		differences := map[string]any{}
		values := row["values"].(map[string]any)
		memberDesired := desiredForMember(revisions, row["memberId"].(uint))
		row["desired"] = memberDesired
		for key, want := range memberDesired {
			if !parameterMatches(key, want, text(values[key])) {
				differences[key] = map[string]any{"actual": values[key], "desired": want, "pendingRestart": row["pendingRestart"].(map[string]bool)[key]}
			}
		}
		row["differences"] = differences
	}
	return map[string]any{"definitions": definitions[c.Engine], "instances": out, "revisions": revisions, "desired": desired}, nil
}
func (s *Service) configure(ctx context.Context, c *model.Cluster, op model.Operation, p Parameters) error {
	if e := validateChanges(c.Engine, p.Changes); e != nil {
		return e
	}
	old, e := s.Configurations(ctx, c)
	if e != nil {
		return e
	}
	revision := model.ConfigRevision{MemberID: p.MemberID, RuntimeOnly: p.RuntimeOnly, ClusterID: c.ID, OperationID: op.ID, Previous: JSON(old["instances"]), Desired: JSON(p.Changes), ApplyMode: "dynamic", Status: "applying", UserID: op.UserID}
	for _, d := range definitions[c.Engine] {
		if _, ok := p.Changes[d.Name]; ok && d.Mode != "dynamic" {
			if d.Mode == "restart" || revision.ApplyMode != "restart" {
				revision.ApplyMode = d.Mode
			}
		}
	}
	if e = s.DB.Create(&revision).Error; e != nil {
		return e
	}
	e = s.step(op, "应用受控配置并记录版本", func(log func(string)) error {
		if c.Topology == "postgresql-patroni" {
			m, e := primary(c)
			if e != nil {
				return e
			}
			control, e := patroniFor(c, *m)
			if e != nil {
				return e
			}
			_, e = s.control(ctx, c, control, http.MethodPatch, "/config", bytes.NewReader(JSON(map[string]any{"postgresql": map[string]any{"parameters": p.Changes}})))
			return e
		}
		// Replicas first, then the primary. Every member is checked before continuing.
		nodes := []model.Member{}
		for _, role := range []string{"replica", "primary"} {
			for _, m := range c.Members {
				if m.Kind == "data" && m.Role == role && (p.MemberID == 0 || p.MemberID == m.ID) {
					nodes = append(nodes, m)
				}
			}
		}
		for _, m := range nodes {
			if e := s.applyConfiguration(ctx, c, m, p.Changes, p.RuntimeOnly); e != nil {
				return e
			}
			if e := s.probeMember(ctx, c, &m); e != nil {
				return e
			}
			log("配置已核验：" + m.Name)
		}
		return nil
	})
	status := "applied"
	if revision.ApplyMode == "restart" {
		status = "pending-restart"
	}
	if e != nil {
		status = "failed"
	}
	if e == nil {
		e = s.waitConfiguration(ctx, configurationScope(c, p.MemberID), p.Changes)
		if e != nil {
			status = "failed"
		}
	}
	s.DB.Model(&revision).Update("status", status)
	return e
}
func (s *Service) applyConfiguration(ctx context.Context, c *model.Cluster, m model.Member, changes map[string]string, runtimeOnly ...bool) error {
	if c.Engine == "redis" {
		client, e := s.redis(c, m)
		if e != nil {
			return e
		}
		defer client.Close()
		if v, ok := changes["maxmemory"]; ok {
			current, e := client.ConfigGet(ctx, "maxmemory").Result()
			if e != nil || len(current) != 2 {
				return errors.New("无法确认当前内存上限")
			}
			if number(v) < number(current[1]) || number(current[1]) == 0 && number(v) > 0 {
				return errors.New("禁止降低当前 maxmemory；无限制到有限制需独立容量评估")
			}
		}
		for k, v := range changes {
			if e = client.ConfigSet(ctx, k, v).Err(); e != nil {
				return errors.New("Redis 参数应用失败")
			}
		}
		if m.Managed {
			return client.ConfigRewrite(ctx).Err()
		}
		return nil
	}
	db, e := s.sql(ctx, c, m, "")
	if e != nil {
		return e
	}
	defer db.Close()
	for k, v := range changes {
		if !regexp.MustCompile(`^[a-z_]+$`).MatchString(k) || strings.ContainsAny(v, "'\\\n") {
			return errors.New("非法参数")
		}
		q := "SET PERSIST " + k + " = " + v
		if c.Engine == "mysql" && len(runtimeOnly) > 0 && runtimeOnly[0] {
			q = "SET GLOBAL " + k + " = " + v
		}
		if c.Engine == "postgresql" {
			q = "ALTER SYSTEM SET " + k + " = '" + v + "'"
		}
		if _, e = db.ExecContext(ctx, q); e != nil {
			return errors.New("数据库拒绝参数变更，请核对版本与权限")
		}
	}
	if c.Engine == "postgresql" {
		_, e = db.ExecContext(ctx, "SELECT pg_reload_conf()")
	}
	return e
}

func parameterMatches(key, want, actual string) bool {
	if key == "slow_query_log" {
		actual = strings.ToLower(actual)
		if actual == "on" {
			actual = "1"
		}
		if actual == "off" {
			actual = "0"
		}
	}
	if want == actual {
		return true
	}
	a, e1 := strconv.ParseFloat(actual, 64)
	w, e2 := strconv.ParseFloat(want, 64)
	return e1 == nil && e2 == nil && a == w
}
func (s *Service) waitConfiguration(ctx context.Context, c *model.Cluster, changes map[string]string) error {
	return waitUntil(ctx, 35*time.Second, func() bool {
		actual, e := s.Configurations(ctx, c)
		if e != nil {
			return false
		}
		rows := actual["instances"].([]map[string]any)
		if len(rows) == 0 {
			return false
		}
		for _, row := range rows {
			if text(row["error"]) != "" {
				return false
			}
			values := row["values"].(map[string]any)
			pending := row["pendingRestart"].(map[string]bool)
			pendingValues := row["pendingValues"].(map[string]string)
			for key, want := range changes {
				if !parameterMatches(key, want, text(values[key])) && !(pending[key] && parameterMatches(key, want, pendingValues[key])) {
					return false
				}
			}
		}
		return true
	})
}

func (s *Service) collectConfigurationStatus(ctx context.Context, c *model.Cluster) {
	report, e := s.Configurations(ctx, c)
	status := "healthy"
	if e != nil {
		status = "unknown"
	} else {
		rows := report["instances"].([]map[string]any)
		if len(rows) == 0 {
			status = "unknown"
		}
		for _, row := range rows {
			if text(row["error"]) != "" {
				status = "unknown"
				break
			}
			for _, diff := range row["differences"].(map[string]any) {
				if flag(diff.(map[string]any)["pendingRestart"]) {
					if status == "healthy" {
						status = "pending-restart"
					}
				} else {
					status = "drifted"
				}
			}
		}
		if status == "healthy" {
			desired := report["desired"].(map[string]string)
			for _, r := range report["revisions"].([]model.ConfigRevision) {
				if r.Status != "pending-restart" {
					continue
				}
				next := "applied"
				for k, v := range object(r.Desired) {
					if desired[k] != text(v) {
						next = "superseded"
					}
				}
				s.DB.Model(&r).Update("status", next)
			}
		}
	}
	if c.ConfigurationStatus != "" && c.ConfigurationStatus != status {
		severity := "info"
		if status != "healthy" {
			severity = "warning"
		}
		s.event(c.ID, 0, 0, "configuration-status", severity, "参数核验状态："+c.ConfigurationStatus+" → "+status)
	}
	s.DB.Model(&model.Cluster{}).Where("id = ?", c.ID).Updates(map[string]any{"configuration_status": status, "configuration_collected_at": time.Now().UTC()})
}

func configurationScope(c *model.Cluster, id uint) *model.Cluster {
	if id == 0 {
		return c
	}
	copy := *c
	copy.Members = nil
	for _, m := range c.Members {
		if m.ID == id {
			copy.Members = append(copy.Members, m)
		}
	}
	return &copy
}
func desiredForMember(revisions []model.ConfigRevision, id uint) map[string]string {
	out := map[string]string{}
	for _, r := range revisions {
		if r.MemberID != 0 && r.MemberID != id {
			continue
		}
		for k, v := range object(r.Desired) {
			if _, ok := out[k]; !ok {
				out[k] = text(v)
			}
		}
	}
	return out
}
