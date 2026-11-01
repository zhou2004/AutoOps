package service

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	cmdbservice "dodevops-api/api/cmdb/service"
	"dodevops-api/api/database/model"
	"github.com/go-redis/redis/v8"
	mysql "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"gorm.io/gorm"
)

func (s *Service) sql(ctx context.Context, c *model.Cluster, m model.Member, database string) (*sql.DB, error) {
	if e := validateTLS(c.Engine, m.TLSMode); e != nil {
		return nil, e
	}
	if err := s.executionOwned(ctx); err != nil {
		return nil, err
	}
	user, password, e := s.memberAccount(c, m)
	if e != nil {
		return nil, e
	}
	address := net.JoinHostPort(m.Address, strconv.Itoa(m.Port))
	driver, dsn := "mysql", ""
	if c.Engine == "mysql" {
		cfg := mysql.NewConfig()
		cfg.User = user
		cfg.Passwd = password
		cfg.Net = "tcp"
		cfg.Addr = address
		cfg.DBName = database
		cfg.InterpolateParams = true
		switch m.TLSMode {
		case "require":
			cfg.TLSConfig = "skip-verify"
		case "verify-full":
			cfg.TLSConfig = "true"
		case "", "disable":
		default:
			return nil, errors.New("MySQL TLS 模式无效")
		}
		cfg.Timeout = 5 * time.Second
		cfg.ReadTimeout = 15 * time.Second
		cfg.WriteTimeout = 15 * time.Second
		dsn = cfg.FormatDSN()
	} else if c.Engine == "postgresql" {
		driver = "postgres"
		if database == "" {
			database = "postgres"
		}
		u := &url.URL{Scheme: "postgres", Host: address, Path: database, User: url.UserPassword(user, password)}
		q := u.Query()
		mode := m.TLSMode
		if mode == "" {
			mode = "prefer"
		}
		q.Set("sslmode", mode)
		q.Set("connect_timeout", "5")
		u.RawQuery = q.Encode()
		dsn = u.String()
	} else {
		return nil, errors.New("此成员不支持 SQL")
	}
	db, e := sql.Open(driver, dsn)
	if e != nil {
		return nil, errors.New("连接初始化失败")
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(0)
	if e = db.PingContext(ctx); e != nil {
		db.Close()
		return nil, errors.New("数据库连接或认证失败")
	}
	return db, nil
}
func (s *Service) redis(c *model.Cluster, m model.Member) (*redis.Client, error) {
	if e := validateTLS("redis", m.TLSMode); e != nil {
		return nil, e
	}
	user, password, e := s.memberAccount(c, m)
	if e != nil {
		return nil, e
	}
	if m.Kind == "sentinel" && m.AccountID == 0 {
		sec, e := s.secrets(c)
		if e != nil {
			return nil, e
		}
		password = sec.Sentinel
		user = "default"
	}
	options := &redis.Options{Addr: net.JoinHostPort(m.Address, fmt.Sprint(m.Port)), Username: user, Password: password, DialTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, MaxRetries: -1}
	if m.TLSMode == "verify-full" {
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12, ServerName: m.Address}
	}
	return redis.NewClient(options), nil
}
func queryMaps(ctx context.Context, db *sql.DB, q string, args ...any) ([]map[string]any, error) {
	rows, e := db.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, errors.New("数据库状态查询失败，请检查版本与账号权限")
	}
	defer rows.Close()
	names, e := rows.Columns()
	if e != nil {
		return nil, e
	}
	out := []map[string]any{}
	for rows.Next() {
		vals := make([]any, len(names))
		ptrs := make([]any, len(names))
		for i := range ptrs {
			ptrs[i] = &vals[i]
		}
		if e = rows.Scan(ptrs...); e != nil {
			return nil, e
		}
		item := map[string]any{}
		for i, v := range vals {
			if b, ok := v.([]byte); ok {
				v = string(b)
			}
			item[names[i]] = v
		}
		out = append(out, item)
		if len(out) > 1000 {
			return nil, errors.New("状态结果超过限制")
		}
	}
	return out, rows.Err()
}
func text(v any) string {
	if v == nil {
		return ""
	}
	if raw, ok := v.([]byte); ok {
		return string(raw)
	}
	return fmt.Sprint(v)
}
func number(v any) float64 { n, _ := strconv.ParseFloat(text(v), 64); return n }
func flag(v any) bool      { return v == true || text(v) == "1" || text(v) == "on" || text(v) == "true" }
func infoMap(value string) map[string]any {
	out := map[string]any{}
	for _, line := range strings.Split(value, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
		if len(parts) == 2 {
			out[parts[0]] = parts[1]
		}
	}
	return out
}

func (s *Service) control(ctx context.Context, c *model.Cluster, m model.Member, method, path string, body io.Reader) (json.RawMessage, error) {
	if err := s.executionOwned(ctx); err != nil {
		return nil, err
	}
	user, password, e := s.memberAccount(c, m)
	if e != nil {
		return nil, e
	}
	if m.Kind == "orchestrator" && c.DeploymentID != nil {
		user, password, e = cmdbservice.NewDatabaseAutomationService(s.DB).OrchestratorControlAccount(*c.DeploymentID)
		if e != nil {
			return nil, e
		}
	}
	endpoint := "http://" + net.JoinHostPort(m.Address, fmt.Sprint(m.Port)) + path
	req, e := http.NewRequestWithContext(ctx, method, endpoint, body)
	if e != nil {
		return nil, e
	}
	req.SetBasicAuth(user, password)
	req.Header.Set("Content-Type", "application/json")
	// Older etcd HTTP/gRPC gateway listeners can close a reused health-check
	// connection before the status POST. Use a fresh connection, never replay it.
	if m.Kind == "etcd" {
		req.Close = true
	}
	client := http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, e := client.Do(req)
	if e != nil {
		return nil, errors.New("管理端请求未完成，写操作不得自动重试")
	}
	defer res.Body.Close()
	b, e := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if e != nil || res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("管理端返回异常状态 %d", res.StatusCode)
	}
	if !json.Valid(b) {
		if m.Kind == "patroni" && method != http.MethodGet {
			return JSON(map[string]any{"accepted": true, "message": strings.TrimSpace(string(b))}), nil
		}
		return nil, errors.New("管理端返回无效 JSON")
	}
	return b, nil
}

func (s *Service) probeMember(ctx context.Context, c *model.Cluster, m *model.Member) error {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	metrics := map[string]any{}
	role, native, parent, version, dataDir := m.Role, m.NativeID, m.ParentID, m.Version, m.DataDir
	if m.Kind == "data" && c.Engine == "mysql" {
		db, e := s.sql(ctx, c, *m, "")
		if e != nil {
			return e
		}
		defer db.Close()
		rows, e := queryMaps(ctx, db, "SELECT @@version AS version,@@server_uuid AS uuid,@@global.read_only AS read_only,@@global.super_read_only AS super_read_only,@@global.gtid_executed AS gtid_executed,@@global.gtid_mode AS gtid_mode,@@datadir AS data_dir,@@max_connections AS max_connections")
		if e != nil || len(rows) != 1 {
			return errors.New("无法读取 MySQL 实例身份")
		}
		metrics = rows[0]
		native = text(metrics["uuid"])
		version = text(metrics["version"])
		dataDir = text(metrics["data_dir"])
		role = "primary"
		parent = ""
		replicas, e := queryMaps(ctx, db, "SHOW SLAVE STATUS")
		if e != nil {
			return errors.New("无法核验 MySQL 复制身份，不能将实例认定为主库")
		}
		if e == nil && len(replicas) > 0 {
			role = "replica"
			r := replicas[0]
			parent = text(r["Master_UUID"])
			for out, key := range map[string]string{"replicationIO": "Slave_IO_Running", "replicationSQL": "Slave_SQL_Running", "lagSeconds": "Seconds_Behind_Master", "replicationError": "Last_SQL_Errno", "upstreamAddress": "Master_Host", "upstreamPort": "Master_Port", "retrievedGTID": "Retrieved_Gtid_Set"} {
				metrics[out] = r[key]
			}
		}
		if m.BaselinePending && role == "primary" {
			role = "initializing"
		}
		semi, _ := queryMaps(ctx, db, "SHOW GLOBAL VARIABLES WHERE Variable_name IN ('rpl_semi_sync_master_enabled','rpl_semi_sync_source_enabled')")
		for _, v := range semi {
			metrics[text(v["Variable_name"])] = v["Value"]
		}
		state, _ := queryMaps(ctx, db, "SHOW GLOBAL STATUS WHERE Variable_name IN ('Threads_connected','Threads_running','Uptime','rpl_semi_sync_master_status','rpl_semi_sync_source_status','rpl_semi_sync_master_clients','rpl_semi_sync_slave_status')")
		for _, v := range state {
			metrics[text(v["Variable_name"])] = v["Value"]
		}
	} else if m.Kind == "data" && c.Engine == "postgresql" {
		db, e := s.sql(ctx, c, *m, "")
		if e != nil {
			return e
		}
		defer db.Close()
		rows, e := queryMaps(ctx, db, "SELECT current_setting('server_version') AS version,pg_is_in_recovery() AS read_only,current_setting('max_connections') AS max_connections")
		if e != nil || len(rows) != 1 {
			return errors.New("无法读取 PostgreSQL 状态")
		}
		metrics = rows[0]
		version = text(metrics["version"])
		role = "primary"
		parent = ""
		if flag(metrics["read_only"]) {
			role = "replica"
		}
		id, _ := queryMaps(ctx, db, "SELECT system_identifier::text AS id FROM pg_control_system()")
		if len(id) > 0 {
			native = text(id[0]["id"]) + "@" + net.JoinHostPort(m.Address, fmt.Sprint(m.Port))
		}
		settings, _ := queryMaps(ctx, db, "SELECT current_setting('data_directory') AS data_dir,current_setting('synchronous_standby_names') AS synchronous_standby_names,current_setting('synchronous_commit') AS synchronous_commit")
		if len(settings) > 0 {
			for k, v := range settings[0] {
				metrics[k] = v
			}
			dataDir = text(metrics["data_dir"])
		}
		if role == "primary" {
			rows, _ = queryMaps(ctx, db, "SELECT application_name,client_addr::text,state,sync_state,write_lsn::text,flush_lsn::text,replay_lsn::text,EXTRACT(EPOCH FROM replay_lag) AS lag_seconds FROM pg_stat_replication")
			metrics["replicas"] = rows
			lsn, _ := queryMaps(ctx, db, "SELECT pg_current_wal_lsn()::text AS lsn")
			if len(lsn) > 0 {
				metrics["lsn"] = lsn[0]["lsn"]
			}
		} else {
			rows, _ = queryMaps(ctx, db, "SELECT status,sender_host,sender_port,received_tli,latest_end_lsn::text FROM pg_stat_wal_receiver")
			metrics["receiver"] = rows
			if len(rows) > 0 {
				parent = text(rows[0]["sender_host"]) + ":" + text(rows[0]["sender_port"])
				metrics["replicationState"] = rows[0]["status"]
			}
			lag, _ := queryMaps(ctx, db, "SELECT pg_last_wal_replay_lsn()::text AS replay_lsn,EXTRACT(EPOCH FROM now()-pg_last_xact_replay_timestamp()) AS replay_age_seconds")
			if len(lag) > 0 {
				for k, v := range lag[0] {
					metrics[k] = v
				}
			}
		}
	} else if c.Engine == "redis" && (m.Kind == "data" || m.Kind == "sentinel") {
		client, e := s.redis(c, *m)
		if e != nil {
			return e
		}
		defer client.Close()
		raw, e := client.Info(ctx).Result()
		if e != nil {
			return errors.New("Redis 连接或认证失败")
		}
		metrics = infoMap(raw)
		role = text(metrics["role"])
		version = text(metrics["redis_version"])
		native = text(metrics["run_id"])
		if m.Kind == "sentinel" {
			role = "sentinel"
			masterName := text(object(c.Policy)["masterName"])
			if masterName == "" {
				masterName = c.Name
			}
			sc := redis.NewSentinelClient(&redis.Options{Addr: client.Options().Addr, Username: client.Options().Username, Password: client.Options().Password, MaxRetries: -1, DialTimeout: 5 * time.Second})
			defer sc.Close()
			status, e := sc.Master(ctx, masterName).Result()
			if e != nil {
				return errors.New("Sentinel 无法读取指定主从组")
			}
			metrics["master"] = status
			_, e = sc.CkQuorum(ctx, masterName).Result()
			metrics["quorumHealthy"] = e == nil
		} else {
			if role == "master" {
				role = "primary"
				parent = ""
			} else if role == "slave" {
				role = "replica"
				parent = text(metrics["master_host"]) + ":" + text(metrics["master_port"])
			}
			cfg, _ := client.ConfigGet(ctx, "dir").Result()
			if len(cfg) == 2 {
				dataDir = text(cfg[1])
			}
			for _, key := range []string{"maxmemory-policy", "min-replicas-to-write", "min-replicas-max-lag", "appendonly", "appendfsync"} {
				v, _ := client.ConfigGet(ctx, key).Result()
				if len(v) == 2 {
					metrics[key] = v[1]
				}
			}
			if flag(metrics["cluster_enabled"]) {
				raw, e = client.ClusterNodes(ctx).Result()
				if e != nil {
					return errors.New("无法读取 Redis Cluster 拓扑")
				}
				metrics["clusterNodes"] = parseClusterNodes(raw)
				clusterInfo, _ := client.ClusterInfo(ctx).Result()
				metrics["cluster"] = infoMap(clusterInfo)
				for _, node := range parseClusterNodes(raw) {
					if strings.Contains(text(node["flags"]), "myself") {
						native = text(node["id"])
						parent = text(node["parent"])
						metrics["slots"] = node["slots"]
					}
				}
			}
		}
	} else if m.Kind == "patroni" {
		raw, e := s.control(ctx, c, *m, http.MethodGet, "/patroni", nil)
		if e != nil {
			return e
		}
		metrics = object(raw)
		role = text(metrics["role"])
		version = text(object(JSON(metrics["patroni"]))["version"])
	} else if m.Kind == "orchestrator" {
		raw, e := s.control(ctx, c, *m, http.MethodGet, "/api/health", nil)
		if e != nil {
			return e
		}
		metrics = object(raw)
		if text(metrics["Code"]) != "OK" {
			return errors.New("Orchestrator 健康检查失败")
		}
		_, e = s.control(ctx, c, *m, http.MethodGet, "/api/leader-check", nil)
		role = "follower"
		if e == nil {
			role = "leader"
		}
	} else if m.Kind == "etcd" {
		raw, e := s.control(ctx, c, *m, http.MethodGet, "/health", nil)
		if e != nil {
			return e
		}
		metrics = object(raw)
		if !flag(metrics["health"]) {
			return errors.New("etcd 不健康")
		}
		status, e := s.control(ctx, c, *m, http.MethodPost, "/v3/maintenance/status", strings.NewReader("{}"))
		if e != nil {
			return fmt.Errorf("无法读取 etcd Raft 身份: %w", e)
		}
		detail := object(status)
		header := object(JSON(detail["header"]))
		metrics["clusterId"], metrics["leaderId"], metrics["raftTerm"] = header["cluster_id"], detail["leader"], detail["raftTerm"]
		native = text(header["member_id"])
		version = text(detail["version"])
		leaderID, leaderErr := strconv.ParseUint(text(detail["leader"]), 10, 64)
		if native == "" || text(metrics["clusterId"]) == "" || leaderErr != nil || leaderID == 0 {
			return errors.New("etcd 尚无可确认的集群多数派 Leader")
		}
		role = "follower"
		if native == text(detail["leader"]) {
			role = "leader"
		}
	} else if m.Kind == "proxy" {
		conn, e := net.DialTimeout("tcp", net.JoinHostPort(m.Address, fmt.Sprint(m.Port)), 3*time.Second)
		if e != nil {
			return errors.New("代理端口不可达")
		}
		conn.Close()
		metrics["check"] = "tcp-only"
		role = "proxy"
	} else {
		return errors.New("成员与数据库引擎不匹配")
	}
	m.Role, m.NativeID, m.ParentID, m.Version, m.DataDir = role, native, parent, version, dataDir
	m.Metrics = JSON(metrics)
	m.State = "healthy"
	m.CollectedAt = nowPtr()
	return nil
}
func parseClusterNodes(raw string) []map[string]any {
	out := []map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		p := strings.Fields(line)
		if len(p) < 8 {
			continue
		}
		slots := []string{}
		if len(p) > 8 {
			slots = p[8:]
		}
		out = append(out, map[string]any{"id": p[0], "address": p[1], "flags": p[2], "parent": p[3], "epoch": p[6], "link": p[7], "slots": slots})
	}
	return out
}
func health(c *model.Cluster) json.RawMessage {
	h := map[string]any{"database": "healthy", "replication": "healthy", "control": "not-applicable", "endpoint": "missing", "backup": "unknown"}
	data, primary, control, healthyControl := 0, 0, 0, 0
	etcdClusters := map[string]bool{}
	for _, m := range c.Members {
		metrics := object(m.Metrics)
		if m.Kind == "etcd" && m.State == "healthy" {
			if id := text(metrics["clusterId"]); id != "" {
				etcdClusters[id] = true
			}
		}
		if m.Kind == "data" {
			data++
			if m.Role == "initializing" {
				h["replication"] = "degraded"
			}
			if m.State != "healthy" {
				if h["database"] != "conflict" {
					h["database"] = "degraded"
				}
				h["replication"] = "degraded"
			}
			if m.Role == "primary" && m.State == "healthy" {
				primary++
			}
			if m.Role == "replica" {
				if c.Engine == "mysql" && m.State == "healthy" && (!flag(metrics["read_only"]) || !flag(metrics["super_read_only"])) {
					h["database"] = "conflict"
				}
				if c.Engine == "mysql" && (metrics["replicationIO"] != "Yes" || metrics["replicationSQL"] != "Yes") {
					h["replication"] = "degraded"
				}
				if c.Engine == "redis" && metrics["master_link_status"] != "up" {
					h["replication"] = "degraded"
				}
				if c.Engine == "postgresql" && metrics["replicationState"] != "streaming" {
					h["replication"] = "degraded"
				}
			}
			if c.Topology == "redis-cluster" {
				ci := object(JSON(metrics["cluster"]))
				if ci["cluster_state"] != "ok" || number(ci["cluster_slots_assigned"]) != 16384 {
					h["replication"] = "degraded"
				}
			}
		} else if m.Kind != "proxy" {
			control++
			if m.State == "healthy" && (m.Kind != "sentinel" || flag(metrics["quorumHealthy"])) {
				healthyControl++
			}
		}
	}
	if (data == 0 || primary == 0) && h["database"] != "conflict" {
		h["database"] = "unavailable"
	}
	if c.Topology != "redis-cluster" && primary > 1 {
		h["database"] = "conflict"
	}
	if c.Topology == "standalone" {
		h["replication"] = "not-applicable"
	}
	if control > 0 {
		h["control"] = "healthy"
		if healthyControl != control {
			h["control"] = "degraded"
		}
	}
	if len(etcdClusters) > 1 {
		h["control"] = "conflict"
	}
	if len(c.Endpoints) > 0 {
		h["endpoint"] = "healthy"
		for _, ep := range c.Endpoints {
			if ep.Status != "healthy" {
				h["endpoint"] = ep.Status
			}
		}
	}
	return JSON(h)
}
func healthStatus(c *model.Cluster) string {
	h := object(c.Health)
	if h["database"] == "conflict" || h["database"] == "unavailable" {
		return text(h["database"])
	}
	for _, v := range h {
		if v == "degraded" || v == "conflict" || v == "unavailable" {
			return "degraded"
		}
	}
	return "healthy"
}
func (s *Service) Refresh(ctx context.Context, id uint) (*model.Cluster, error) {
	c, e := s.Cluster(id)
	if e != nil {
		return nil, e
	}
	previous := map[uint]string{}
	identity := []any{}
	var wg sync.WaitGroup
	limit := make(chan struct{}, 8)
	for i := range c.Members {
		m := &c.Members[i]
		previous[m.ID] = m.Role
		wg.Add(1)
		go func(m *model.Member) {
			defer wg.Done()
			limit <- struct{}{}
			defer func() { <-limit }()
			probeCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
			defer cancel()
			if err := s.probeMember(probeCtx, c, m); err != nil {
				m.State = "unreachable"
				m.Metrics = JSON(map[string]any{"error": err.Error()})
				m.CollectedAt = nowPtr()
			}
		}(m)
	}
	wg.Wait()
	s.captureReplicationEvidence(ctx, c)
	for _, m := range c.Members {
		identity = append(identity, []any{m.ID, m.NativeID, m.Role, m.ParentID, object(m.Metrics)["slots"]})
	}
	if len(c.Endpoints) > 0 {
		_ = s.verifyEndpoints(ctx, c)
		s.DB.Where("cluster_id = ?", c.ID).Find(&c.Endpoints)
	}
	hash := sha256.Sum256(JSON(identity))
	nextHash := hex.EncodeToString(hash[:])
	changed := c.TopologyHash != "" && c.TopologyHash != nextHash
	c.TopologyHash = nextHash
	c.CollectedAt = nowPtr()
	c.Health = health(c)
	var latest model.Backup
	if s.DB.Where("cluster_id = ?", c.ID).Order("created_at DESC").First(&latest).Error == nil {
		h := object(c.Health)
		h["backup"] = "degraded"
		if latest.Status == "complete" && time.Since(latest.CreatedAt) < 24*time.Hour {
			h["backup"] = "healthy"
		}
		c.Health = JSON(h)
	}
	c.Status = healthStatus(c)
	e = s.DB.Transaction(func(tx *gorm.DB) error {
		for _, m := range c.Members {
			if e := tx.Model(&model.Member{}).Where("id = ? AND cluster_id = ?", m.ID, c.ID).Select("DataMemberID", "TLSMode", "Role", "NativeID", "ParentID", "Version", "DataDir", "State", "Metrics", "CollectedAt").Updates(&m).Error; e != nil {
				return e
			}
		}
		updates := map[string]any{"topology_hash": c.TopologyHash, "collected_at": c.CollectedAt, "health": c.Health, "status": c.Status}
		if changed {
			updates["revision"] = gorm.Expr("revision + 1")
		}
		if e := tx.Model(&model.Cluster{}).Where("id = ?", c.ID).Updates(updates).Error; e != nil {
			return e
		}
		if changed {
			return tx.Create(&model.Event{ClusterID: id, Kind: "topology_changed", Severity: "warning", Message: "采集到拓扑或角色变化，旧执行计划已失效"}).Error
		}
		return nil
	})
	if e != nil {
		return nil, e
	}
	return s.Cluster(id)
}
