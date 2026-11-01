package service

import (
	"context"
	"crypto/tls"
	"database/sql"
	"dodevops-api/pkg/rediscompat"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"dodevops-api/api/cmdb/model"
	configmodel "dodevops-api/api/configcenter/model"
	databaseModel "dodevops-api/api/database/model"
	"github.com/go-redis/redis/v8"
	mysqlDriver "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"gorm.io/gorm"
)

const (
	databaseResultLimit = 500
	redisValueLimit     = 1 << 20
)

type DatabaseOperationService struct{ db *gorm.DB }

type databaseTarget struct {
	Asset    model.CmdbSQL
	Account  configmodel.AccountAuth
	Secret   string
	Endpoint *databaseModel.Endpoint
}

type DatabaseOverview struct {
	Type      string         `json:"type"`
	Address   string         `json:"address"`
	Version   string         `json:"version"`
	Role      string         `json:"role,omitempty"`
	LatencyMS int64          `json:"latencyMs"`
	Metrics   map[string]any `json:"metrics"`
	CheckedAt time.Time      `json:"checkedAt"`
}

type DatabaseTable struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Rows   int64  `json:"rows"`
}

type DatabaseQueryResult struct {
	Columns      []string         `json:"columns,omitempty"`
	Rows         []map[string]any `json:"rows,omitempty"`
	ReturnedRows int              `json:"returnedRows"`
	AffectedRows int64            `json:"affectedRows"`
	Truncated    bool             `json:"truncated"`
	DurationMS   int64            `json:"durationMs"`
}

type RedisKeySummary struct {
	Key  string `json:"key"`
	Type string `json:"type"`
	TTL  int64  `json:"ttl"`
}

func NewDatabaseOperationService(db *gorm.DB) *DatabaseOperationService {
	return &DatabaseOperationService{db: db}
}

func (s *DatabaseOperationService) target(id uint) (*databaseTarget, error) {
	var target databaseTarget
	if id == 0 || s.db.First(&target.Asset, id).Error != nil {
		return nil, errors.New("数据源不存在")
	}
	if target.Asset.Retired {
		return nil, errors.New("该实例已退役，数据源保留用于历史审计")
	}
	if target.Asset.Type < 1 || target.Asset.Type > 3 {
		return nil, errors.New("仅支持 MySQL、PostgreSQL 和 Redis")
	}
	if err := s.db.First(&target.Account, target.Asset.AccountID).Error; err != nil {
		return nil, errors.New("数据源关联的连接凭据不存在")
	}
	secret, err := target.Account.DecryptPassword()
	if err != nil {
		return nil, errors.New("数据源凭据无法解密")
	}
	if target.Asset.ConnectionMode == "cluster" {
		var endpoint databaseModel.Endpoint
		if target.Asset.ClusterID == 0 || s.db.Where("id = ? AND cluster_id = ?", target.Asset.EndpointID, target.Asset.ClusterID).First(&endpoint).Error != nil || endpoint.AccountID != target.Asset.AccountID {
			return nil, errors.New("集群入口与数据源凭据关联无效")
		}
		target.Endpoint = &endpoint
	}
	target.Secret = secret
	if strings.TrimSpace(target.Account.Host) == "" || target.Account.Port < 1 || target.Account.Port > 65535 {
		return nil, errors.New("数据源地址或端口无效")
	}
	return &target, nil
}

func databaseTypeName(kind int) string {
	return map[int]string{1: "mysql", 2: "postgresql", 3: "redis"}[kind]
}

func normalizeTLS(kind int, value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = "disable"
	}
	allowed := map[int]map[string]bool{
		1: {"disable": true, "verify-full": true},
		2: {"disable": true, "require": true, "verify-ca": true, "verify-full": true},
		3: {"disable": true, "verify-full": true},
	}
	if !allowed[kind][value] {
		return "", errors.New("TLS 模式不受支持")
	}
	return value, nil
}

func (s *DatabaseOperationService) openSQL(ctx context.Context, id uint, database string) (*sql.DB, *databaseTarget, error) {
	target, err := s.target(id)
	if err != nil {
		return nil, nil, err
	}
	if target.Endpoint != nil {
		var addresses []string
		if json.Unmarshal(target.Endpoint.Addresses, &addresses) != nil || len(addresses) == 0 {
			return nil, nil, errors.New("集群入口地址无效")
		}
		for _, address := range addresses {
			host, port, e := net.SplitHostPort(address)
			if e != nil {
				continue
			}
			n, e := strconv.Atoi(port)
			if e != nil {
				continue
			}
			target.Account.Host, target.Account.Port = host, n
			db, _, e := s.openSQLTarget(ctx, target, database)
			if e == nil {
				return db, target, nil
			}
			if ctx.Err() != nil {
				break
			}
		}
		return nil, nil, errors.New("集群访问入口均不可用")
	}
	return s.openSQLTarget(ctx, target, database)
}
func (s *DatabaseOperationService) openSQLTarget(ctx context.Context, target *databaseTarget, database string) (*sql.DB, *databaseTarget, error) {
	if target.Asset.Type == 3 {
		return nil, nil, errors.New("Redis 不支持 SQL 操作")
	}
	tlsMode, err := normalizeTLS(target.Asset.Type, target.Asset.TLSMode)
	if err != nil {
		return nil, nil, err
	}
	database = strings.TrimSpace(database)
	if database == "" {
		database = strings.TrimSpace(target.Asset.DefaultDB)
	}
	var driver, dsn string
	if target.Asset.Type == 1 {
		config := mysqlDriver.NewConfig()
		config.User, config.Passwd, config.Net = target.Account.Name, target.Secret, "tcp"
		config.Addr, config.DBName, config.ParseTime = net.JoinHostPort(target.Account.Host, strconv.Itoa(target.Account.Port)), database, true
		config.Timeout, config.ReadTimeout, config.WriteTimeout = 8*time.Second, 30*time.Second, 30*time.Second
		if tlsMode == "verify-full" {
			config.TLSConfig = "true"
		}
		driver, dsn = "mysql", config.FormatDSN()
	} else {
		if database == "" {
			database = "postgres"
		}
		u := &url.URL{Scheme: "postgres", Host: net.JoinHostPort(target.Account.Host, strconv.Itoa(target.Account.Port)), Path: database}
		u.User = url.UserPassword(target.Account.Name, target.Secret)
		query := u.Query()
		query.Set("sslmode", tlsMode)
		query.Set("connect_timeout", "8")
		u.RawQuery = query.Encode()
		driver, dsn = "postgres", u.String()
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, nil, errors.New("数据源连接初始化失败")
	}
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(2 * time.Minute)
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, nil, errors.New("数据源连接或认证失败")
	}
	return db, target, nil
}

func (s *DatabaseOperationService) redisClient(id uint) (redis.UniversalClient, *databaseTarget, error) {
	target, err := s.target(id)
	if err != nil {
		return nil, nil, err
	}
	if target.Asset.Type != 3 {
		return nil, nil, errors.New("该数据源不是 Redis")
	}
	tlsMode, err := normalizeTLS(3, target.Asset.TLSMode)
	if err != nil {
		return nil, nil, err
	}
	dbNumber := 0
	if target.Asset.DefaultDB != "" {
		dbNumber, err = strconv.Atoi(target.Asset.DefaultDB)
		if err != nil || dbNumber < 0 || dbNumber > 1024 {
			return nil, nil, errors.New("Redis DB 编号无效")
		}
	}
	options := &redis.Options{Addr: net.JoinHostPort(target.Account.Host, strconv.Itoa(target.Account.Port)), Username: target.Account.Name, Password: target.Secret, DB: dbNumber, DialTimeout: 8 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, PoolSize: 3}
	if tlsMode == "verify-full" {
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12, ServerName: target.Account.Host}
	}
	options.MaxRetries = -1
	if target.Endpoint != nil {
		var addresses []string
		if json.Unmarshal(target.Endpoint.Addresses, &addresses) != nil || len(addresses) == 0 {
			return nil, nil, errors.New("Redis 集群种子无效")
		}
		switch target.Endpoint.Mode {
		case "cluster":
			if dbNumber != 0 {
				return nil, nil, errors.New("Redis Cluster 仅支持 DB 0")
			}
			return rediscompat.NewCluster(&redis.ClusterOptions{Addrs: addresses, Username: options.Username, Password: options.Password, TLSConfig: options.TLSConfig, DialTimeout: options.DialTimeout, ReadTimeout: options.ReadTimeout, WriteTimeout: options.WriteTimeout, PoolSize: 3, MaxRetries: -1, MaxRedirects: 8}), target, nil
		case "sentinel":
			var sentinel configmodel.AccountAuth
			if s.db.First(&sentinel, target.Endpoint.SentinelAccountID).Error != nil {
				return nil, nil, errors.New("Sentinel 凭据不存在")
			}
			password, e := sentinel.DecryptPassword()
			if e != nil {
				return nil, nil, errors.New("Sentinel 凭据无法解密")
			}
			return redis.NewFailoverClient(&redis.FailoverOptions{MasterName: target.Endpoint.MasterName, SentinelAddrs: addresses, SentinelUsername: sentinel.Name, SentinelPassword: password, Username: options.Username, Password: options.Password, DB: dbNumber, TLSConfig: options.TLSConfig, DialTimeout: options.DialTimeout, ReadTimeout: options.ReadTimeout, WriteTimeout: options.WriteTimeout, PoolSize: 3, MaxRetries: -1}), target, nil
		default:
			return nil, nil, errors.New("Redis 入口模式无效")
		}
	}
	return redis.NewClient(options), target, nil
}

func (s *DatabaseOperationService) Overview(ctx context.Context, id uint) (*DatabaseOverview, error) {
	start := time.Now()
	target, err := s.target(id)
	if err != nil {
		return nil, err
	}
	output := &DatabaseOverview{Type: databaseTypeName(target.Asset.Type), Address: net.JoinHostPort(target.Account.Host, strconv.Itoa(target.Account.Port)), Metrics: map[string]any{}, CheckedAt: time.Now().UTC()}
	if target.Asset.Type == 3 {
		client, _, err := s.redisClient(id)
		if err != nil {
			return nil, err
		}
		defer client.Close()
		checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := client.Ping(checkCtx).Err(); err != nil {
			return nil, errors.New("Redis 连接或认证失败")
		}
		info, err := client.Info(checkCtx, "server", "memory", "replication", "keyspace").Result()
		if err != nil {
			return nil, errors.New("Redis 状态读取失败")
		}
		for _, line := range strings.Split(info, "\n") {
			parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
			if len(parts) != 2 {
				continue
			}
			switch parts[0] {
			case "redis_version":
				output.Version = parts[1]
			case "role":
				output.Role = parts[1]
			case "used_memory_human", "connected_clients", "uptime_in_seconds":
				output.Metrics[parts[0]] = parts[1]
			}
		}
		output.Metrics["keys"], _ = client.DBSize(checkCtx).Result()
	} else {
		db, target, err := s.openSQL(ctx, id, "")
		if err != nil {
			return nil, err
		}
		defer db.Close()
		query := "SELECT VERSION()"
		if target.Asset.Type == 2 {
			query = "SHOW server_version"
		}
		if err := db.QueryRowContext(ctx, query).Scan(&output.Version); err != nil {
			return nil, errors.New("数据库版本读取失败")
		}
		if target.Asset.Type == 1 {
			var readOnly int
			var maxConnections int64
			if db.QueryRowContext(ctx, "SELECT @@read_only, @@max_connections").Scan(&readOnly, &maxConnections) == nil {
				output.Metrics["readOnly"], output.Metrics["maxConnections"] = readOnly == 1, maxConnections
			}
		} else {
			var inRecovery bool
			var maxConnections string
			if db.QueryRowContext(ctx, "SELECT pg_is_in_recovery(), current_setting('max_connections')").Scan(&inRecovery, &maxConnections) == nil {
				output.Metrics["inRecovery"], output.Metrics["maxConnections"] = inRecovery, maxConnections
			}
		}
	}
	output.LatencyMS = time.Since(start).Milliseconds()
	return output, nil
}

func (s *DatabaseOperationService) Catalog(ctx context.Context, id uint) ([]string, error) {
	target, err := s.target(id)
	if err != nil {
		return nil, err
	}
	if target.Asset.Type == 3 {
		return nil, errors.New("Redis 使用逻辑 DB 编号，不提供 SQL 数据库目录")
	}
	db, target, err := s.openSQL(ctx, id, "")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	query := "SELECT schema_name FROM information_schema.schemata ORDER BY schema_name"
	if target.Asset.Type == 2 {
		query = "SELECT datname FROM pg_database WHERE datallowconn AND NOT datistemplate ORDER BY datname"
	}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.New("数据库目录读取失败")
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			items = append(items, name)
		}
	}
	return items, rows.Err()
}

func (s *DatabaseOperationService) Tables(ctx context.Context, id uint, database string) ([]DatabaseTable, error) {
	db, target, err := s.openSQL(ctx, id, database)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	var rows *sql.Rows
	if target.Asset.Type == 1 {
		rows, err = db.QueryContext(ctx, "SELECT table_schema, table_name, table_type, COALESCE(table_rows,0) FROM information_schema.tables WHERE table_schema = DATABASE() ORDER BY table_name LIMIT 1000")
	} else {
		rows, err = db.QueryContext(ctx, "SELECT table_schema, table_name, table_type, 0 FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog','information_schema') ORDER BY table_schema, table_name LIMIT 1000")
	}
	if err != nil {
		return nil, errors.New("数据表目录读取失败")
	}
	defer rows.Close()
	items := []DatabaseTable{}
	for rows.Next() {
		var item DatabaseTable
		if rows.Scan(&item.Schema, &item.Name, &item.Type, &item.Rows) == nil {
			items = append(items, item)
		}
	}
	return items, rows.Err()
}

func sqlOperation(statement string) (string, error) {
	words, err := sqlWords(statement)
	if err != nil {
		return "", err
	}
	return words[0], nil
}

func DetectSQLOperation(statement string) (string, error) {
	return sqlOperation(statement)
}

func normalizeSQLValue(value any) any {
	switch v := value.(type) {
	case []byte:
		return string(v)
	case int64:
		if v > 9007199254740991 || v < -9007199254740991 {
			return strconv.FormatInt(v, 10)
		}
		return v
	case uint64:
		if v > 9007199254740991 {
			return strconv.FormatUint(v, 10)
		}
		return v
	case time.Time:
		return v.Format(time.RFC3339Nano)
	default:
		return v
	}
}

func (s *DatabaseOperationService) Query(ctx context.Context, id uint, database, statement string) (*DatabaseQueryResult, string, error) {
	operation, err := sqlOperation(statement)
	if err != nil {
		return nil, "", err
	}
	if !map[string]bool{"SELECT": true, "SHOW": true, "DESCRIBE": true, "DESC": true, "EXPLAIN": true, "WITH": true}[operation] {
		return nil, operation, errors.New("查询入口只允许只读 SQL")
	}
	words, _ := sqlWords(statement)
	for _, word := range words {
		if word == "OUTFILE" || word == "DUMPFILE" || word == "LOAD_FILE" {
			return nil, operation, errors.New("查询入口不允许访问服务器文件")
		}
	}
	db, _, err := s.openSQL(ctx, id, database)
	if err != nil {
		return nil, operation, err
	}
	defer db.Close()
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tx, err := db.BeginTx(queryCtx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, operation, errors.New("无法创建只读查询事务")
	}
	defer tx.Rollback()
	start := time.Now()
	rows, err := tx.QueryContext(queryCtx, statement)
	if err != nil {
		return nil, operation, errors.New("SQL 查询失败")
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, operation, errors.New("查询结果读取失败")
	}
	output := &DatabaseQueryResult{Columns: columns, Rows: []map[string]any{}}
	for rows.Next() {
		if len(output.Rows) >= databaseResultLimit {
			output.Truncated = true
			break
		}
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, operation, errors.New("查询结果解析失败")
		}
		item := make(map[string]any, len(columns))
		for i, column := range columns {
			item[column] = normalizeSQLValue(values[i])
		}
		output.Rows = append(output.Rows, item)
	}
	output.ReturnedRows, output.DurationMS = len(output.Rows), time.Since(start).Milliseconds()
	return output, operation, rows.Err()
}

func (s *DatabaseOperationService) Execute(ctx context.Context, id uint, database, statement, confirmName string) (*DatabaseQueryResult, string, error) {
	target, err := s.target(id)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(confirmName) != target.Asset.Name {
		return nil, "", errors.New("请输入完整数据源名称确认写操作")
	}
	operation, err := sqlOperation(statement)
	if err != nil {
		return nil, "", err
	}
	if !map[string]bool{"INSERT": true, "UPDATE": true, "DELETE": true}[operation] {
		return nil, operation, errors.New("写操作入口只允许 INSERT、UPDATE 或 DELETE")
	}
	words, _ := sqlWords(statement)
	hasWhere := false
	for _, word := range words {
		if word == "WHERE" {
			hasWhere = true
		}
	}
	if (operation == "UPDATE" || operation == "DELETE") && !hasWhere {
		return nil, operation, errors.New("UPDATE/DELETE 必须包含 WHERE 条件")
	}
	db, _, err := s.openSQL(ctx, id, database)
	if err != nil {
		return nil, operation, err
	}
	defer db.Close()
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	start := time.Now()
	result, err := db.ExecContext(execCtx, statement)
	if err != nil {
		return nil, operation, errors.New("SQL 写操作失败")
	}
	affected, _ := result.RowsAffected()
	return &DatabaseQueryResult{AffectedRows: affected, DurationMS: time.Since(start).Milliseconds()}, operation, nil
}

func (s *DatabaseOperationService) RedisKeys(ctx context.Context, id uint, cursor uint64, match string, count int64) ([]RedisKeySummary, uint64, error) {
	if count < 1 || count > 200 {
		count = 100
	}
	if len(match) > 256 {
		return nil, cursor, errors.New("匹配表达式过长")
	}
	client, _, err := s.redisClient(id)
	if err != nil {
		return nil, cursor, err
	}
	defer client.Close()
	keys, next, err := client.Scan(ctx, cursor, match, count).Result()
	if err != nil {
		return nil, cursor, errors.New("Redis SCAN 失败")
	}
	items := make([]RedisKeySummary, 0, len(keys))
	for _, key := range keys {
		kind, kindErr := client.Type(ctx, key).Result()
		ttl, ttlErr := client.TTL(ctx, key).Result()
		if kindErr == nil && ttlErr == nil {
			items = append(items, RedisKeySummary{Key: key, Type: kind, TTL: int64(ttl.Seconds())})
		}
	}
	return items, next, nil
}

func (s *DatabaseOperationService) RedisValue(ctx context.Context, id uint, key string) (map[string]any, error) {
	if key == "" || len(key) > 1024 {
		return nil, errors.New("Redis Key 无效")
	}
	client, _, err := s.redisClient(id)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	kind, err := client.Type(ctx, key).Result()
	if err != nil || kind == "none" {
		return nil, errors.New("Redis Key 不存在")
	}
	var value any
	truncated := false
	switch kind {
	case "string":
		length, lengthErr := client.StrLen(ctx, key).Result()
		if lengthErr != nil || length > redisValueLimit {
			return nil, errors.New("Redis String 超过 1 MiB 读取限制")
		}
		value, err = client.Get(ctx, key).Result()
	case "hash":
		var fields []string
		var next uint64
		fields, next, err = client.HScan(ctx, key, 0, "*", 200).Result()
		items := make(map[string]string, len(fields)/2)
		for i := 0; i+1 < len(fields); i += 2 {
			items[fields[i]] = fields[i+1]
		}
		value, truncated = items, next != 0
	case "list":
		value, err = client.LRange(ctx, key, 0, 199).Result()
	case "set":
		var members []string
		var next uint64
		members, next, err = client.SScan(ctx, key, 0, "*", 200).Result()
		value, truncated = members, next != 0
	case "zset":
		value, err = client.ZRangeWithScores(ctx, key, 0, 199).Result()
	default:
		return nil, errors.New("暂不支持读取该 Redis 数据类型")
	}
	if err != nil {
		return nil, errors.New("Redis 值读取失败")
	}
	serialized, _ := json.Marshal(value)
	if len(serialized) > redisValueLimit {
		return nil, errors.New("Redis 值超过 1 MiB 读取限制")
	}
	ttl, _ := client.TTL(ctx, key).Result()
	return map[string]any{"key": key, "type": kind, "ttl": int64(ttl.Seconds()), "value": value, "truncated": truncated}, nil
}

func (s *DatabaseOperationService) SetRedisString(ctx context.Context, id uint, key, value string, ttlSeconds int64, confirmName string) error {
	target, err := s.target(id)
	if err != nil {
		return err
	}
	if target.Asset.Type != 3 || strings.TrimSpace(confirmName) != target.Asset.Name {
		return errors.New("请输入完整数据源名称确认写操作")
	}
	if key == "" || len(key) > 1024 || len(value) > 1024*1024 || ttlSeconds < 0 || ttlSeconds > 31536000 {
		return errors.New("Redis Key、值或 TTL 无效")
	}
	client, _, err := s.redisClient(id)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Set(ctx, key, value, time.Duration(ttlSeconds)*time.Second).Err()
}

func (s *DatabaseOperationService) DeleteRedisKey(ctx context.Context, id uint, key, confirmKey string) (int64, error) {
	if key == "" || key != confirmKey {
		return 0, errors.New("请输入完整 Redis Key 确认删除")
	}
	client, _, err := s.redisClient(id)
	if err != nil {
		return 0, err
	}
	defer client.Close()
	return client.Del(ctx, key).Result()
}

func databaseAuditTarget(action, database, object string) string {
	value := strings.TrimSpace(database + "/" + object)
	if len(value) > 255 {
		value = value[:255]
	}
	if value == "/" {
		return action
	}
	return value
}

func MarshalDatabaseValue(value any) string {
	data, _ := json.Marshal(value)
	if len(data) > 500 {
		data = data[:500]
	}
	return string(data)
}
