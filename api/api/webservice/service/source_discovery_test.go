package service

import (
	"context"
	system "dodevops-api/api/system/model"
	"dodevops-api/api/webservice/model"
	"dodevops-api/pkg/apikey"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type discoveryRuntime struct {
	called int
	wrong  bool
}

func (f *discoveryRuntime) Run(_ context.Context, r RemoteRequest) (json.RawMessage, error) {
	f.called++
	if r.Action != "source-discovery" {
		return nil, errors.New("unexpected remote action")
	}
	spec := r.Spec
	if f.wrong {
		spec.HostID++
	}
	spec.Binary = "/usr/sbin/nginx"
	spec.ConfigPath = "/etc/nginx/nginx.conf"
	spec.Prefix = "/etc/nginx"
	return JSON(SourceDiscoveryResult{Candidates: []SourceCandidate{{ID: "candidate", Spec: spec}}, Warnings: []string{}}), nil
}
func TestSourceScanUsesBoundConnectionAndDoesNotRegister(t *testing.T) {
	s, _, _ := vipFixture(t)
	f := &discoveryRuntime{}
	s.Remote = f
	ctx := context.Background()
	input := SourceDiscoveryInput{HostID: 1, ConnectionID: 1, Runtime: "host"}
	result, err := s.DiscoverSources(ctx, 1, input)
	if err != nil || len(result.Candidates) != 1 {
		t.Fatal(err)
	}
	var count int64
	s.DB.Table(model.Tables["sources"]).Count(&count)
	if count != 3 {
		t.Fatal("scan modified registrations")
	}
	input.ConnectionID = 2
	if _, err = s.DiscoverSources(ctx, 1, input); err == nil {
		t.Fatal("cross-host connection accepted")
	}
	if f.called != 1 {
		t.Fatal("invalid target reached SSH")
	}
	input.ConnectionID = 1
	f.wrong = true
	if _, err = s.DiscoverSources(ctx, 1, input); err == nil {
		t.Fatal("wrong target observation accepted")
	}
}
func TestSourceScanRechecksAPIKeyHostExecutionScope(t *testing.T) {
	s, _, _ := vipFixture(t)
	f := &discoveryRuntime{}
	s.Remote = f
	if err := s.DB.AutoMigrate(&system.APIKey{}); err != nil {
		t.Fatal(err)
	}
	key := system.APIKey{UserID: 1, Hash: "source-scan-test", GrantMode: apikey.Scoped, Scopes: "web:write", ExpiresAt: time.Now().Add(time.Hour)}
	s.DB.Create(&key)
	ctx := apikey.WithKey(context.Background(), key.ID)
	input := SourceDiscoveryInput{HostID: 1, ConnectionID: 1, Runtime: "host"}
	if _, err := s.DiscoverSources(ctx, 1, input); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
	if f.called != 0 {
		t.Fatal("restricted key reached host")
	}
	s.DB.Model(&key).Update("scopes", "web:write,cmdb:execute")
	if _, err := s.DiscoverSources(ctx, 1, input); err != nil {
		t.Fatal(err)
	}
}
