package service

import (
	"context"
	"crypto/sha256"
	"dodevops-api/api/database/model"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gorm.io/gorm/clause"
	"net/http"
	"strings"
	"time"
)

func (s *Service) nativeEvent(c *model.Cluster, member uint, key, message, severity string, details any) {
	if len(message) > 900 {
		message = string([]rune(message)[:min(200, len([]rune(message)))])
	}
	s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.Event{ClusterID: c.ID, MemberID: member, ExternalKey: &key, Kind: "native-ha", Severity: severity, Message: message, Details: JSON(details)})
}
func (s *Service) collectNativeEvents(ctx context.Context, c *model.Cluster) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if c.Engine == "mysql" && c.Topology != "standalone" {
		rows, e := s.mysqlRecoveries(ctx, c)
		if e != nil {
			return
		}
		for _, r := range rows {
			if !r.belongsTo(c) {
				continue
			}
			status := "进行中"
			severity := "warning"
			if r.finished() {
				status = "已结束"
				severity = "info"
			} else if s.recoveryAborted(ctx, c, r) {
				status = "隔离钩子阻止恢复"
			}
			key := fmt.Sprintf("orchestrator:%s:%s", r.UID, status)
			s.nativeEvent(c, 0, key, fmt.Sprintf("Orchestrator 恢复 #%d：%s；分析实例 %s:%d", r.ID, status, r.AnalysisEntry.AnalyzedInstanceKey.Hostname, r.AnalysisEntry.AnalyzedInstanceKey.Port), severity, r)
		}
	} else if c.Topology == "postgresql-patroni" {
		for _, m := range c.Members {
			if m.Kind != "patroni" || m.State != "healthy" {
				continue
			}
			raw, e := s.control(ctx, c, m, http.MethodGet, "/history", nil)
			if e != nil {
				continue
			}
			var entries [][]any
			if json.Unmarshal(raw, &entries) != nil {
				continue
			}
			for _, entry := range entries {
				if len(entry) < 3 {
					continue
				}
				digest := sha256.Sum256(JSON(entry))
				key := "patroni:" + hex.EncodeToString(digest[:])
				s.nativeEvent(c, 0, key, fmt.Sprintf("Patroni timeline %v，LSN %v：%v", entry[0], entry[1], entry[2]), "info", entry)
			}
			break
		}
	} else if c.Engine == "redis" {
		for _, m := range c.Members {
			if !m.Managed || m.Service == "" || m.Kind != "sentinel" && m.Kind != "data" {
				continue
			}
			raw, e := s.remoteJSON(ctx, m.HostID, redisNativeJournal, map[string]any{"service": m.Service})
			if e != nil {
				continue
			}
			var rows []map[string]string
			if json.Unmarshal(raw, &rows) != nil {
				continue
			}
			for _, row := range rows {
				digest := sha256.Sum256([]byte(row["cursor"]))
				key := fmt.Sprintf("redis:%d:%s", m.ID, hex.EncodeToString(digest[:]))
				severity := "info"
				if strings.Contains(row["message"], "+sdown") || strings.Contains(row["message"], "+odown") {
					severity = "warning"
				}
				s.nativeEvent(c, m.ID, key, row["message"], severity, row)
			}
		}
	}
}

const redisNativeJournal = `import json,sys,subprocess,re
p=json.load(sys.stdin);assert re.fullmatch(r'[A-Za-z0-9_.@-]+',p['service'])
r=subprocess.run(['journalctl','-u',p['service'],'--since','10 minutes ago','-n','150','--no-pager','-o','json'],capture_output=True,text=True,timeout=5,check=True)
rows=[]
for line in r.stdout.splitlines():
 try:item=json.loads(line)
 except ValueError:continue
 message=item.get('MESSAGE','')
 if any(marker in message for marker in ('+switch-master','+failover-state','+sdown','-sdown','+odown','-odown','Failover auth granted','Failover election won','Cluster state changed:')):
  rows.append({'cursor':item.get('__CURSOR',''),'timeMicroseconds':item.get('__REALTIME_TIMESTAMP',''),'message':message[:900]})
print(json.dumps(rows))`
