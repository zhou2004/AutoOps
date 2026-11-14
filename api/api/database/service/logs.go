package service

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	cmdbservice "dodevops-api/api/cmdb/service"
	"dodevops-api/api/database/model"
	"dodevops-api/pkg/apikey"
)

//go:embed log_reader.py
var databaseLogReader string

type DatabaseLogSource struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}
type DatabaseLogEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// Logs can contain statements and values; require explicit cluster audit access.
func (s *Service) LogMember(ctx context.Context, user, cluster, member uint) (model.Member, error) {
	var m model.Member
	identity, err := apikey.EffectiveIdentity(ctx, s.DB, user)
	if err != nil || !identity.Allows("database:read") {
		return m, ErrForbidden
	}
	for _, action := range []string{"db_cluster_view", "db_cluster_audit"} {
		if err = s.Authorize(user, cluster, action); err != nil {
			return m, err
		}
	}
	if member == 0 {
		return m, errors.New("请选择数据库实例")
	}
	if err = s.DB.Where("id = ? AND cluster_id = ?", member, cluster).First(&m).Error; err != nil {
		return m, errors.New("数据库实例不存在")
	}
	if m.HostID == 0 {
		return m, errors.New("实例尚未关联 CMDB 主机，无法读取日志")
	}
	if m.Service != "" && !unitPattern.MatchString(m.Service) {
		return m, errors.New("实例服务名称无效")
	}
	return m, nil
}
func logInput(m model.Member) map[string]any {
	return map[string]any{"service": m.Service, "dataDir": m.DataDir}
}
func (s *Service) DatabaseLogs(ctx context.Context, user, cluster, member uint) ([]DatabaseLogSource, error) {
	m, err := s.LogMember(ctx, user, cluster, member)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	input := logInput(m)
	input["action"] = "list"
	raw, err := s.remoteJSONCommand(ctx, m.HostID, "python3", databaseLogReader, input)
	if err != nil {
		return nil, err
	}
	var rows []DatabaseLogSource
	if err = json.Unmarshal(raw, &rows); err != nil {
		return nil, errors.New("无法发现日志源，请检查实例目录与 SSH 读取权限")
	}
	return rows, nil
}

var databaseLogStreams = struct {
	sync.Mutex
	users map[uint]int
	total int
}{users: map[uint]int{}}

func acquireDatabaseLogStream(user uint) (func(), error) {
	databaseLogStreams.Lock()
	defer databaseLogStreams.Unlock()
	if databaseLogStreams.total >= 32 || databaseLogStreams.users[user] >= 4 {
		return nil, errors.New("日志连接已达上限，请关闭其他追踪窗口")
	}
	databaseLogStreams.total++
	databaseLogStreams.users[user]++
	return func() {
		databaseLogStreams.Lock()
		defer databaseLogStreams.Unlock()
		databaseLogStreams.total--
		databaseLogStreams.users[user]--
		if databaseLogStreams.users[user] == 0 {
			delete(databaseLogStreams.users, user)
		}
	}, nil
}
func (s *Service) FollowDatabaseLogs(parent context.Context, user uint, m model.Member, logID string, lines int, follow bool, emit func(DatabaseLogEvent) error) error {
	current, err := s.LogMember(parent, user, m.ClusterID, m.ID)
	if err != nil {
		return err
	}
	if current.HostID != m.HostID || current.Service != m.Service || current.DataDir != m.DataDir {
		return errors.New("实例配置已变化，请重新加载日志")
	}
	release, err := acquireDatabaseLogStream(user)
	if err != nil {
		return err
	}
	defer release()
	ctx, cancel := context.WithTimeout(parent, 20*time.Minute)
	defer cancel()
	// Only server-issued identifiers are accepted; the reader rediscovers sources.
	if logID != "journal" {
		if len(logID) != 64 {
			return errors.New("日志标识无效")
		}
		for _, c := range logID {
			if !((c >= 'a' && c <= 'f') || (c >= '0' && c <= '9')) {
				return errors.New("日志标识无效")
			}
		}
	}
	if lines < 1 || lines > 2000 {
		return errors.New("日志行数必须为 1–2000")
	}
	client, err := cmdbservice.ResolveAssetSSH(s.DB, "host", m.HostID, 0)
	if err != nil {
		return errors.New("CMDB SSH 连接或指纹校验失败")
	}
	defer client.Close()
	stop := context.AfterFunc(ctx, func() { client.Close() })
	defer stop()
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	input := logInput(m)
	input["action"] = "read"
	input["logId"] = logID
	input["lines"] = lines
	input["follow"] = follow
	session.Stdin = bytes.NewReader(JSON(input))
	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	command := "python3 -u -c \"import base64;exec(base64.b64decode('" + base64.StdEncoding.EncodeToString([]byte(databaseLogReader)) + "'))\""
	if err = session.Start(command); err != nil {
		return errors.New("无法启动数据库日志读取")
	}
	events := make(chan DatabaseLogEvent, 8)
	done := make(chan error, 1)
	go func() {
		defer close(events)
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 65536), 16<<20)
		for scanner.Scan() {
			var event DatabaseLogEvent
			if json.Unmarshal(scanner.Bytes(), &event) != nil {
				done <- errors.New("日志流返回无效数据")
				return
			}
			select {
			case events <- event:
			case <-ctx.Done():
				done <- ctx.Err()
				return
			}
		}
		if e := scanner.Err(); e != nil {
			done <- errors.New("日志事件超过大小限制")
			return
		}
		done <- session.Wait()
	}()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-events:
			if !ok {
				if e := <-done; e != nil {
					return errors.New("数据库日志读取已中断")
				}
				return nil
			}
			if err = emit(event); err != nil {
				return err
			}
		case <-ticker.C:
			current, e := s.LogMember(ctx, user, m.ClusterID, m.ID)
			if e != nil {
				return e
			}
			if current.HostID != m.HostID || current.Service != m.Service || current.DataDir != m.DataDir {
				return errors.New("实例配置已变化，请重新加载日志")
			}
			if err = emit(DatabaseLogEvent{Event: "heartbeat", Data: JSON(map[string]any{"at": time.Now().Unix()})}); err != nil {
				return fmt.Errorf("日志连接中断: %w", err)
			}
		}
	}
}
