package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/webservice/model"
	"errors"
	"testing"
	"time"
)

type testLogRuntime struct {
	*fakeRuntime
	started chan struct{}
	stopped chan struct{}
}

func (f *testLogRuntime) Stream(ctx context.Context, r RemoteRequest, emit func(LogStreamEvent) error) error {
	defer close(f.stopped)
	close(f.started)
	if e := emit(LogStreamEvent{Event: "ready", Data: JSON(map[string]string{"path": "/var/log/nginx/access.log"})}); e != nil {
		return e
	}
	<-ctx.Done()
	return ctx.Err()
}
func logFixture(t *testing.T) (*Service, RemoteRequest) {
	s := fixture(t)
	s.DB.AutoMigrate(&cmdb.CmdbAssetConnection{})
	s.DB.Create(&cmdb.CmdbAssetConnection{ID: 7, AssetType: "host", AssetID: 9, Protocol: "ssh", Enabled: true})
	src, _ := s.Get("sources", 1)
	p, _ := decode[model.SourceSpec](src.Spec)
	p.HostID = 9
	p.ConnectionID = 7
	s.DB.Table(model.Tables["sources"]).Where("id=1").Update("spec", JSON(p))
	s.DB.Table(model.Tables["instances"]).Where("id=1").Update("observed", JSON(Observation{Identity: "nginx", Logs: []map[string]string{{"id": "a", "path": "/var/log/nginx/access.log", "realPath": "/var/log/nginx/access.log"}}}))
	return s, RemoteRequest{LogID: "a"}
}
func TestLogStreamTargetAuthorizationAndRegisteredScope(t *testing.T) {
	s, request := logFixture(t)
	target, e := s.LogTarget(context.Background(), 1, 1, request)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.LogTarget(context.Background(), 2, 1, request); !errors.Is(e, ErrForbidden) {
		t.Fatal("reader without log scope admitted", e)
	}
	bad := request
	bad.LogID = "/etc/passwd"
	if _, e = s.LogTarget(context.Background(), 1, 1, bad); e == nil {
		t.Fatal("path accepted as ID")
	}
	bad = request
	bad.Rotation = ".1"
	if _, e = s.LogTarget(context.Background(), 1, 1, bad); e == nil {
		t.Fatal("historical stream admitted")
	}
	changed := target
	changed.Source.Revision++
	if sameLogTarget(target, changed) {
		t.Fatal("changed source admitted")
	}
	changed = target
	changed.Profile.UpdatedAt = time.Now()
	if sameLogTarget(target, changed) {
		t.Fatal("changed connection admitted")
	}
}
func TestLogStreamCancellationStopsRemoteAndAudits(t *testing.T) {
	s, request := logFixture(t)
	target, e := s.LogTarget(context.Background(), 1, 1, request)
	if e != nil {
		t.Fatal(e)
	}
	remote := &testLogRuntime{fakeRuntime: s.Remote.(*fakeRuntime), started: make(chan struct{}), stopped: make(chan struct{})}
	s.Remote = remote
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := s.FollowLogs(ctx, 1, target, func(e LogStreamEvent) error { cancel(); return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	select {
	case <-remote.stopped:
	case <-time.After(time.Second):
		t.Fatal("remote stream leaked")
	}
	var count int64
	s.DB.Model(&model.Event{}).Where("kind IN ?", []string{"log-follow-start", "log-follow-end"}).Count(&count)
	if count != 2 {
		t.Fatal("missing connection audit", count)
	}
}
func TestLogStreamAccountRevocationClosesSession(t *testing.T) {
	s, request := logFixture(t)
	target, e := s.LogTarget(context.Background(), 1, 1, request)
	if e != nil {
		t.Fatal(e)
	}
	remote := &testLogRuntime{fakeRuntime: s.Remote.(*fakeRuntime), started: make(chan struct{}), stopped: make(chan struct{})}
	s.Remote = remote
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	err := s.FollowLogs(ctx, 1, target, func(e LogStreamEvent) error {
		if e.Event == "ready" {
			s.DB.Exec("UPDATE sys_admin SET status=2 WHERE id=1")
		}
		return nil
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatal("revoked user retained stream", err)
	}
	select {
	case <-remote.stopped:
	case <-time.After(time.Second):
		t.Fatal("revoked stream leaked")
	}
}
