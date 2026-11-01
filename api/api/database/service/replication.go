package service

import (
	"context"
	"dodevops-api/api/database/model"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func replicationParent(c *model.Cluster, r model.Member) *model.Member {
	for i := range c.Members {
		p := &c.Members[i]
		if p.Kind != "data" || p.Role != "primary" {
			continue
		}
		m := object(r.Metrics)
		if (p.NativeID != "" && r.ParentID == p.NativeID) || r.ParentID == fmt.Sprintf("%s:%d", p.Address, p.Port) || (text(m["upstreamAddress"]) == p.Address && text(m["upstreamPort"]) == fmt.Sprint(p.Port)) {
			return p
		}
	}
	return nil
}

// Capture a source watermark FIRST, then read the replica. This proves replay
// through that sampled point, not whole-table equality or zero future data loss.
func (s *Service) captureReplicationEvidence(ctx context.Context, c *model.Cluster) {
	for i := range c.Members {
		r := &c.Members[i]
		if r.Kind != "data" || r.Role != "replica" || r.State != "healthy" {
			continue
		}
		p := replicationParent(c, *r)
		if p == nil || p.State != "healthy" {
			continue
		}
		rm, pm := object(r.Metrics), object(p.Metrics)
		switch c.Engine {
		case "mysql":
			probe, cancel := context.WithTimeout(ctx, 3*time.Second)
			conn, e := s.sql(probe, c, *r, "")
			if e == nil {
				var caught int
				watermark := text(pm["gtid_executed"])
				if watermark != "" && conn.QueryRowContext(probe, "SELECT GTID_SUBSET(?,@@global.gtid_executed)", watermark).Scan(&caught) == nil {
					rm["sampleCaughtUp"] = caught == 1
					rm["sampleWatermark"] = watermark
				}
				conn.Close()
			}
			cancel()
		case "postgresql":
			probe, cancel := context.WithTimeout(ctx, 3*time.Second)
			conn, e := s.sql(probe, c, *r, "")
			if e == nil {
				var delta int64
				watermark := text(pm["lsn"])
				if watermark != "" && conn.QueryRowContext(probe, "SELECT pg_wal_lsn_diff($1::pg_lsn,pg_last_wal_replay_lsn())::bigint", watermark).Scan(&delta) == nil {
					rm["sampleCaughtUp"] = delta <= 0
					rm["sampleLagBytes"] = max(delta, 0)
					rm["sampleWatermark"] = watermark
				}
				conn.Close()
			}
			cancel()
		case "redis":
			probe, cancel := context.WithTimeout(ctx, 3*time.Second)
			client, e := s.redis(c, *r)
			if e == nil {
				raw, err := client.Info(probe, "replication").Result()
				if err == nil {
					current := infoMap(raw)
					a, e1 := strconv.ParseInt(text(pm["master_repl_offset"]), 10, 64)
					b, e2 := strconv.ParseInt(text(current["slave_repl_offset"]), 10, 64)
					if text(current["master_replid"]) == text(pm["master_replid"]) && text(pm["master_replid"]) != "" && e1 == nil && e2 == nil {
						rm["sampleCaughtUp"] = b >= a
						rm["sampleLagBytes"] = max(a-b, 0)
						rm["sampleWatermark"] = fmt.Sprint(a)
					}
				}
				client.Close()
			}
			cancel()
		}
		r.Metrics = JSON(rm)
	}
}
func ReplicationLinks(c *model.Cluster) []model.ReplicationLink {
	links := []model.ReplicationLink{}
	for _, r := range c.Members {
		if r.Kind != "data" || r.Role != "replica" {
			continue
		}
		m := object(r.Metrics)
		p := replicationParent(c, r)
		link := model.ReplicationLink{TargetID: r.ID, State: "unknown", Message: "尚未确认上游与复制状态", CollectedAt: r.CollectedAt, Evidence: map[string]any{}}
		if p != nil {
			link.SourceID = p.ID
		}
		fresh := collectedFresh(r.CollectedAt, 90*time.Second) && p != nil && collectedFresh(p.CollectedAt, 90*time.Second)
		if !fresh || c.Status == "stale" {
			link.Message = "采集过期或上游未知"
			links = append(links, link)
			continue
		}
		running := false
		switch c.Engine {
		case "mysql":
			running = m["replicationIO"] == "Yes" && m["replicationSQL"] == "Yes" && text(m["replicationError"]) == "0"
			for _, k := range []string{"replicationIO", "replicationSQL", "replicationError", "lagSeconds", "read_only", "super_read_only"} {
				link.Evidence[k] = m[k]
			}
			if !flag(m["read_only"]) || !flag(m["super_read_only"]) {
				running = false
			}
		case "postgresql":
			running = m["replicationState"] == "streaming" && strings.Split(r.NativeID, "@")[0] == strings.Split(p.NativeID, "@")[0]
			link.Evidence["receiver"] = m["receiver"]
			link.Evidence["streaming"] = m["replicationState"]
		case "redis":
			running = m["master_link_status"] == "up" && text(m["master_sync_in_progress"]) == "0"
			for _, k := range []string{"master_link_status", "master_replid", "slave_repl_offset", "master_last_io_seconds_ago"} {
				link.Evidence[k] = m[k]
			}
		}
		for _, k := range []string{"sampleCaughtUp", "sampleWatermark", "sampleLagBytes"} {
			if v, ok := m[k]; ok {
				link.Evidence[k] = v
			}
		}
		switch {
		case r.State != "healthy" || p.State != "healthy" || !running:
			link.State = "unavailable"
			link.Message = "复制中断、节点不可达或保护配置异常"
		case m["sampleCaughtUp"] == true:
			link.State = "healthy"
			link.Message = "复制正常 · 已追平采样位点"
		case m["sampleCaughtUp"] == false:
			link.State = "degraded"
			link.Message = "复制运行中 · 尚未追平采样位点"
		default:
			link.Message = "复制运行中 · 追平结果尚未核验"
		}
		links = append(links, link)
	}
	return links
}
