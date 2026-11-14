package service

import (
	"bufio"
	"bytes"
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	cmdbservice "dodevops-api/api/cmdb/service"
	"dodevops-api/api/webservice/model"
	assets "dodevops-api/automation/web-platform"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

type LogStreamEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}
type LogStreamer interface {
	Stream(context.Context, RemoteRequest, func(LogStreamEvent) error) error
}

func (r *SSHRuntime) Stream(ctx context.Context, req RemoteRequest, emit func(LogStreamEvent) error) error {
	client, err := cmdbservice.ResolveAssetSSH(r.DB, "host", req.Spec.HostID, req.Spec.ConnectionID)
	if err != nil {
		return err
	}
	defer client.Close()
	stop := context.AfterFunc(ctx, func() { client.Close() })
	defer stop()
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	session.Stdin = bytes.NewReader(JSON(req))
	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stderr := &boundedBuffer{Max: 32 << 10}
	session.Stderr = stderr
	code := "import base64;exec(compile(base64.b64decode('" + base64.StdEncoding.EncodeToString([]byte(assets.Runner)) + "'),'<autoops-web>','exec'))"
	command := "python3 -u -c " + quote(code)
	if req.Spec.Sudo {
		command = "sudo -n " + command
	}
	if err = session.Start(command); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 65536), 1<<20)
	for scanner.Scan() {
		var event LogStreamEvent
		if json.Unmarshal(scanner.Bytes(), &event) != nil || len(event.Data) == 0 {
			return errors.New("日志流返回无效数据")
		}
		switch event.Event {
		case "ready", "log", "notice", "reset", "end", "error", "heartbeat":
		default:
			return errors.New("未知日志流事件")
		}
		if err = emit(event); err != nil {
			return err
		}
		if event.Event == "error" {
			return errors.New("远端日志流已中断")
		}
	}
	if err = scanner.Err(); err != nil {
		return err
	}
	return session.Wait()
}

type LogTarget struct {
	Instance model.Resource
	Source   model.Resource
	Request  RemoteRequest
	Profile  cmdb.CmdbAssetConnection
}

func (s *Service) LogTarget(ctx context.Context, u, id uint, request RemoteRequest) (LogTarget, error) {
	var target LogTarget
	if err := s.Module(ctx, u, "read"); err != nil {
		return target, err
	}
	i, err := s.Get("instances", id)
	if err != nil {
		return target, err
	}
	if err = s.Authorize(u, "instances", i, "web_logs"); err != nil {
		return target, err
	}
	src, err := s.Get("sources", i.SourceID)
	if err != nil {
		return target, err
	}
	spec, err := decode[model.SourceSpec](src.Spec)
	if err != nil {
		return target, err
	}
	ob, err := decode[Observation](i.Observed)
	if err != nil || ob.Identity == "" {
		return target, errors.New("实例尚未完成发现")
	}
	var registered map[string]string
	for _, log := range ob.Logs {
		if log["id"] == request.LogID {
			registered = log
			break
		}
	}
	if registered == nil {
		return target, errors.New("日志标识无效，请重新加载日志列表")
	}
	if request.Rotation != "" || request.From != "" || request.To != "" {
		return target, errors.New("实时追踪仅支持当前日志，请先清除轮转文件和时间范围")
	}
	if request.Lines < 1 {
		request.Lines = 500
	}
	if request.Lines > 10000 {
		request.Lines = 10000
	}
	if len(request.Keyword) > 1024 || len(request.URI) > 1024 || len(request.IP) > 64 || len(request.Status) > 3 {
		return target, errors.New("日志筛选条件过长")
	}
	if err = s.DB.Where("id = ? AND asset_type = ? AND asset_id = ? AND protocol = ? AND enabled = ?", spec.ConnectionID, "host", spec.HostID, "ssh", true).First(&target.Profile).Error; err != nil {
		return target, ErrForbidden
	}
	request.Action = "log-follow"
	request.Spec = spec
	request.Identity = ob.Identity
	request.Logs = []map[string]string{registered}
	target.Instance = i
	target.Source = src
	target.Request = request
	return target, nil
}

var logStreams = struct {
	sync.Mutex
	total int
	users map[uint]int
}{users: map[uint]int{}}

func acquireLogStream(user uint) (func(), error) {
	logStreams.Lock()
	defer logStreams.Unlock()
	if logStreams.total >= 32 || logStreams.users[user] >= 4 {
		return nil, errors.New("日志连接已达上限，请关闭其他追踪窗口")
	}
	logStreams.total++
	logStreams.users[user]++
	return func() {
		logStreams.Lock()
		defer logStreams.Unlock()
		logStreams.total--
		logStreams.users[user]--
		if logStreams.users[user] == 0 {
			delete(logStreams.users, user)
		}
	}, nil
}
func sameLogTarget(a, b LogTarget) bool {
	return a.Instance.SourceID == b.Instance.SourceID && a.Instance.ClusterID == b.Instance.ClusterID && a.Source.Revision == b.Source.Revision && a.Profile.UpdatedAt.Equal(b.Profile.UpdatedAt) && digest(JSON(a.Profile)) == digest(JSON(b.Profile)) && a.Request.Identity == b.Request.Identity && digest(JSON(a.Request.Spec)) == digest(JSON(b.Request.Spec)) && digest(JSON(a.Request.Logs)) == digest(JSON(b.Request.Logs))
}
func (s *Service) FollowLogs(parent context.Context, user uint, target LogTarget, emit func(LogStreamEvent) error) (result error) {
	release, err := acquireLogStream(user)
	if err != nil {
		return err
	}
	defer release()
	stream, ok := s.Remote.(LogStreamer)
	if !ok {
		return errors.New("当前运行环境不支持日志流")
	}
	ctx, cancel := context.WithTimeout(parent, 20*time.Minute)
	defer cancel()
	event := model.Event{UserID: user, ClusterID: target.Instance.ClusterID, SourceID: target.Instance.SourceID, Kind: "log-follow-start", Message: fmt.Sprintf("instance=%d log=%s", target.Instance.ID, target.Request.LogID)}
	if err = s.DB.Create(&event).Error; err != nil {
		return err
	}
	defer func() {
		status := "closed"
		if result != nil {
			status = "interrupted"
		}
		s.DB.Create(&model.Event{UserID: user, ClusterID: event.ClusterID, SourceID: event.SourceID, Kind: "log-follow-end", Message: fmt.Sprintf("session=%d result=%s", event.ID, status)})
	}()
	events := make(chan LogStreamEvent, 8)
	done := make(chan error, 1)
	go func() {
		done <- stream.Stream(ctx, target.Request, func(e LogStreamEvent) error {
			select {
			case events <- e:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-done:
			// A completed process can leave a final buffered log/end event.
			for {
				select {
				case e := <-events:
					if e2 := emit(e); e2 != nil {
						return e2
					}
				default:
					return err
				}
			}
		case e := <-events:
			if err = emit(e); err != nil {
				return err
			}
		case <-ticker.C:
			current, e := s.LogTarget(ctx, user, target.Instance.ID, target.Request)
			if e != nil {
				return e
			}
			if !sameLogTarget(target, current) {
				return errors.New("目标身份、日志登记或连接档案已变化，请重新加载")
			}
			if err = emit(LogStreamEvent{Event: "heartbeat", Data: JSON(map[string]any{"at": time.Now().UTC()})}); err != nil {
				return err
			}
		}
	}
}
