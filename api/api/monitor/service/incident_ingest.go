package service

import (
	"crypto/sha256"
	"dodevops-api/api/monitor/dao"
	"dodevops-api/api/monitor/model"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func incidentLevel(level string) string {
	switch strings.ToLower(level) {
	case "critical", "fatal", "error", "p0", "p1", "p2":
		return "critical"
	case "info", "informational", "p4":
		return "info"
	default:
		return "warning"
	}
}

// 在通知路由之前接入真实告警，即使未配置通知渠道也保留故障记录。
func ingestIncidentAlerts(d *dao.IncidentDao, payload map[string]interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var data struct {
		Status string `json:"status"`
		Alerts []struct {
			Status      string            `json:"status"`
			Labels      map[string]string `json:"labels"`
			Annotations map[string]string `json:"annotations"`
			StartsAt    string            `json:"startsAt"`
			EndsAt      string            `json:"endsAt"`
		} `json:"alerts"`
	}
	if err = json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("告警数据格式错误: %w", err)
	}
	for _, alert := range data.Alerts {
		status := alert.Status
		if status == "" {
			status = data.Status
		}
		if status != "firing" && status != "resolved" {
			continue
		}
		title := alert.Labels["alertname"]
		if title == "" {
			continue
		}
		start, err := time.Parse(time.RFC3339Nano, alert.StartsAt)
		if err != nil || start.IsZero() {
			return fmt.Errorf("告警 %s 缺少有效 startsAt", title)
		}
		source := alert.Labels["monitor_source"]
		if source != "domain_cert" && source != "api_endpoint" {
			source = "prometheus"
		}
		sourceID, _ := strconv.ParseUint(alert.Labels["monitor_source_id"], 10, 32)
		identity := alert.Labels
		if source != "prometheus" {
			identity = map[string]string{"source": source, "id": alert.Labels["monitor_source_id"], "rule": alert.Labels["monitor_rule_id"]}
			if sourceID == 0 {
				// Old rule-state entries may predate target IDs in labels.
				for _, field := range []string{"domain", "port", "url", "method"} {
					if value := alert.Labels[field]; value != "" {
						identity[field] = value
					}
				}
			}
		}
		encoded, _ := json.Marshal(identity)
		sum := sha256.Sum256(append(encoded, []byte(start.UTC().Format(time.RFC3339Nano))...))
		key := hex.EncodeToString(sum[:])
		description := alert.Annotations["description"]
		if description == "" {
			description = alert.Annotations["summary"]
		}
		target := alert.Labels["instance"]
		if source == "domain_cert" {
			target = alert.Labels["domain"]
		}
		if source == "api_endpoint" {
			target = alert.Labels["name"]
		}
		if target != "" {
			title += " · " + target
		}
		m := &model.MonitorIncident{EventKey: &key, Title: title, Source: source, SourceID: uint(sourceID), Level: incidentLevel(alert.Labels["severity"]), Status: status, Description: description, AlertTime: start.In(time.Local).Format("2006-01-02 15:04:05")}
		if status == "resolved" {
			end, err := time.Parse(time.RFC3339Nano, alert.EndsAt)
			if err != nil || end.IsZero() {
				end = time.Now()
			}
			m.ResolvedAt = end.In(time.Local).Format("2006-01-02 15:04:05")
		}
		if err := d.Ingest(m); err != nil {
			return fmt.Errorf("保存故障失败: %w", err)
		}
	}
	return nil
}
