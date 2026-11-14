package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	system "dodevops-api/api/system/model"
	"dodevops-api/api/webservice/model"
	"dodevops-api/pkg/apikey"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

type trafficTestRuntime struct {
	result  trafficLogs
	call    func()
	cursors json.RawMessage
	err     error
}

func (r *trafficTestRuntime) Run(_ context.Context, req RemoteRequest) (json.RawMessage, error) {
	if req.Action != "traffic" {
		return nil, errors.New("unexpected action")
	}
	r.cursors = req.TrafficCursors
	if r.call != nil {
		r.call()
	}
	return JSON(r.result), r.err
}
func trafficFixture(t *testing.T) (*Service, model.Resource, model.Resource, *trafficTestRuntime) {
	s := fixture(t)
	i, _ := s.Get("instances", 1)
	source, _ := s.Get("sources", 1)
	state := model.TrafficState{InstanceID: 1, SourceRevision: source.Revision, Identity: "identity", CollectedAt: time.Now().Add(-30 * time.Second), Metrics: JSON(map[string]float64{"requests_total": 100}), Cursors: JSON(map[string]any{"log": map[string]any{"offset": 50, "capabilities": map[string]bool{"accessLog": true, "latency": true}}}), Revision: 1}
	s.DB.Create(&state)
	remote := &trafficTestRuntime{result: trafficLogs{RateValid: true, Cursors: state.Cursors, Requests: 10, ResponseBytes: 6000, RequestBytes: 1200, RequestByteCount: 10, Errors: 2, LatencyCount: 10, LatencyTotalMS: 200, SupportedLogs: 1, IPs: map[string]trafficCount{"192.0.2.1": {Requests: 10, ResponseBytes: 6000}}, URIs: map[string]trafficCount{"/safe": {Requests: 10}}, StatusCodes: map[string]uint64{"200": 8, "500": 2}}}
	s.Remote = remote
	return s, i, source, remote
}
func TestTrafficCounterDeltaPermissionsAndStaleness(t *testing.T) {
	s, i, source, _ := trafficFixture(t)
	spec, _ := decode[model.SourceSpec](source.Spec)
	spec.StatusURL = "http://127.0.0.1/status"
	ob := Observation{Identity: "identity", Metrics: map[string]float64{"requests_total": 130, "connections_active": 4}}
	s.collectTraffic(context.Background(), i, source, spec, &ob, nil)
	result, e := s.Traffic(context.Background(), 1, TrafficFilter{})
	if e != nil {
		t.Fatal(e)
	}
	if result.Summary.QPS == nil || *result.Summary.QPS < .9 || *result.Summary.QPS > 1.1 || *result.Summary.ErrorRate != 20 || *result.Summary.AverageLatencyMS != 20 {
		t.Fatalf("bad real deltas: %#v", result.Summary)
	}
	if len(result.TopIPs) != 1 || *result.Summary.LoggedRequests != 10 {
		t.Fatal("log aggregate missing")
	}
	s.DB.Create(&cmdb.CmdbAssetPermission{Name: "read", UserIDs: "[2]", AssetTypes: `["web_source"]`, WebSourceIDs: "[1]", PermissionActions: `["web_view"]`, IsActive: 1})
	reader, e := s.Traffic(context.Background(), 2, TrafficFilter{})
	if e != nil {
		t.Fatal(e)
	}
	if reader.Summary.QPS == nil || reader.Summary.ResponseBytesPerSecond != nil || len(reader.TopIPs) > 0 || reader.Summary.ErrorRate != nil || reader.Summary.LoggedRequests != nil {
		t.Fatal("log data leaked to read-only asset permission")
	}
	s.DB.Model(&model.TrafficSample{}).Where("instance_id = ?", 1).Update("at", time.Now().Add(-91*time.Second))
	result, e = s.Traffic(context.Background(), 1, TrafficFilter{})
	if e != nil || result.Summary.QPS != nil || result.Nodes[0].Status != "unknown" || result.Coverage.FreshInstances != 0 {
		t.Fatal("stale traffic presented as current", e)
	}
}
func TestTrafficCounterResetAndSamplingGapHaveNoArtificialSpikes(t *testing.T) {
	s, i, source, remote := trafficFixture(t)
	spec, _ := decode[model.SourceSpec](source.Spec)
	ob := Observation{Identity: "identity", Metrics: map[string]float64{"requests_total": 2, "connections_active": 1}}
	s.collectTraffic(context.Background(), i, source, spec, &ob, nil)
	var sample model.TrafficSample
	s.DB.Order("id DESC").First(&sample)
	values, _ := decode[TrafficValues](sample.Metrics)
	if values.QPS != nil {
		t.Fatal("counter reset created rate")
	}
	s.DB.Model(&model.TrafficState{}).Where("instance_id = ?", 1).Update("collected_at", time.Now().Add(-2*time.Minute))
	ob.Metrics = map[string]float64{"requests_total": 100000, "connections_active": 1}
	s.collectTraffic(context.Background(), i, source, spec, &ob, nil)
	if string(remote.cursors) != "{}" {
		t.Fatal("gap reused old log cursor")
	}
	sample = model.TrafficSample{}
	s.DB.Order("id DESC").First(&sample)
	values, _ = decode[TrafficValues](sample.Metrics)
	if values.QPS != nil || values.ResponseBytesPerSecond != nil {
		t.Fatal("gap created current-window rate")
	}
}
func TestTrafficSampleAndCursorCommitTogetherAndRejectLateCollectors(t *testing.T) {
	s, i, source, remote := trafficFixture(t)
	spec, _ := decode[model.SourceSpec](source.Spec)
	ob := Observation{Identity: "identity", Metrics: map[string]float64{"requests_total": 130, "connections_active": 4}}
	if e := s.DB.Exec("CREATE TRIGGER traffic_fail BEFORE INSERT ON web_traffic_sample BEGIN SELECT RAISE(ABORT,'sample insertion failed'); END").Error; e != nil {
		t.Fatal(e)
	}
	s.collectTraffic(context.Background(), i, source, spec, &ob, nil)
	var state model.TrafficState
	s.DB.First(&state, 1)
	if state.Revision != 1 {
		t.Fatal("failed sample consumed cursor")
	}
	if _, ok := ob.Metrics["qps"]; ok {
		t.Fatal("uncommitted rate became live observation")
	}
	s.DB.Exec("DROP TRIGGER traffic_fail")
	remote.call = func() { s.DB.Model(&model.TrafficState{}).Where("instance_id = ?", 1).Update("revision", 2) }
	s.collectTraffic(context.Background(), i, source, spec, &ob, nil)
	var n int64
	s.DB.Model(&model.TrafficSample{}).Count(&n)
	if n != 0 {
		t.Fatal("late collector wrote duplicate sample")
	}
}
func TestTrafficFiltersBeforeAggregationAndRejectsScopedAPIKey(t *testing.T) {
	s, i, source, _ := trafficFixture(t)
	spec, _ := decode[model.SourceSpec](source.Spec)
	ob := Observation{Identity: "identity", Metrics: map[string]float64{"requests_total": 130, "connections_active": 4}}
	s.collectTraffic(context.Background(), i, source, spec, &ob, nil)
	s.DB.Table(model.Tables["instances"]).Create(&model.Resource{ID: 2, SourceID: 2, ClusterID: 2, GroupID: 999, Name: "hidden"})
	s.DB.Create(&model.TrafficSample{InstanceID: 2, At: time.Now(), Metrics: JSON(TrafficValues{QPS: number(9999)}), Logs: JSON(trafficLogs{SupportedLogs: 1, Requests: 1, IPs: map[string]trafficCount{"secret": {Requests: 1}}})})
	s.DB.Create(&cmdb.CmdbAssetPermission{Name: "logs", UserIDs: "[2]", AssetTypes: `["web_source"]`, WebSourceIDs: "[1]", PermissionActions: `["web_view","web_logs"]`, IsActive: 1})
	result, e := s.Traffic(context.Background(), 2, TrafficFilter{Range: "7d"})
	if e != nil || result.Coverage.Instances != 1 || strings.Contains(string(JSON(result)), "secret") || len(result.Series) > 338 {
		t.Fatal("unscoped or unbounded traffic", e)
	}
	result, e = s.Traffic(context.Background(), 2, TrafficFilter{InstanceID: 2})
	if e != nil || len(result.Nodes) != 0 {
		t.Fatal("hidden instance leaked")
	}
	s.DB.AutoMigrate(&system.APIKey{})
	key := system.APIKey{UserID: 1, Hash: "traffic-read", GrantMode: apikey.Scoped, Scopes: "cmdb:read", ExpiresAt: time.Now().Add(time.Hour)}
	s.DB.Create(&key)
	if _, e = s.Traffic(apikey.WithKey(context.Background(), key.ID), 1, TrafficFilter{}); !errors.Is(e, ErrForbidden) {
		t.Fatal("API key module scope bypass", e)
	}
}
func TestSharedAccessLogIsNotDoubleCountedAcrossInstances(t *testing.T) {
	s, i, source, _ := trafficFixture(t)
	spec, _ := decode[model.SourceSpec](source.Spec)
	log := map[string]string{"id": "shared", "kind": "access", "path": "/var/log/nginx/access.log", "realPath": "/var/log/nginx/access.log"}
	ob := Observation{Identity: "nginx1", Logs: []map[string]string{log}}
	s.DB.Table(model.Tables["instances"]).Create(&model.Resource{ID: 2, SourceID: 2, Spec: JSON(spec), Observed: JSON(Observation{Identity: "nginx2", Logs: []map[string]string{log}})})
	logs, notes := s.uniqueTrafficLogs(i, spec, ob)
	if len(logs) != 0 || len(notes) == 0 {
		t.Fatal("shared file attributed twice")
	}
}
func TestTrafficErrorAndLatencyAggregationAreWeighted(t *testing.T) {
	a := trafficAccumulator{}
	a.add(TrafficValues{QPS: number(2)}, trafficLogs{Requests: 1, Errors: 1, LatencyCount: 1, LatencyTotalMS: 100})
	a.add(TrafficValues{QPS: number(10)}, trafficLogs{Requests: 9, LatencyCount: 9, LatencyTotalMS: 90})
	v := a.sumValues()
	if *v.ErrorRate != 10 || *v.AverageLatencyMS != 19 || *v.QPS != 12 {
		t.Fatal("rates averaged with wrong weights")
	}
}

func TestTrafficFailedLogCollectionResetsCursorBeforeRecovery(t *testing.T) {
	s, i, source, remote := trafficFixture(t)
	spec, _ := decode[model.SourceSpec](source.Spec)
	ob := Observation{Identity: "identity", Metrics: map[string]float64{"requests_total": 130, "connections_active": 1}}
	remote.err = errors.New("temporary SSH failure")
	s.collectTraffic(context.Background(), i, source, spec, &ob, nil)
	var state model.TrafficState
	s.DB.First(&state, 1)
	if string(state.Cursors) != "{}" {
		t.Fatal("failed sampling retained a multi-window cursor")
	}
	s.DB.Model(&model.TrafficState{}).Where("instance_id = ?", 1).Update("collected_at", time.Now().Add(-30*time.Second))
	remote.err = nil
	remote.result = trafficLogs{RateValid: true, Initialized: true, Cursors: JSON(map[string]any{"a": map[string]any{"offset": 100}})}
	ob.Metrics = map[string]float64{"requests_total": 160, "connections_active": 1}
	s.collectTraffic(context.Background(), i, source, spec, &ob, nil)
	if string(remote.cursors) != "{}" {
		t.Fatal("recovery replayed missed traffic")
	}
	var sample model.TrafficSample
	s.DB.Order("id DESC").First(&sample)
	values, _ := decode[TrafficValues](sample.Metrics)
	if values.ResponseBytesPerSecond != nil {
		t.Fatal("recovery initialization reported accumulated bytes as current rate")
	}
}
func TestTrafficInvalidLogWindowDoesNotReportZeroOrInflatedRate(t *testing.T) {
	s, i, source, remote := trafficFixture(t)
	spec, _ := decode[model.SourceSpec](source.Spec)
	remote.result.RateValid = false
	remote.result.Partial = true
	remote.result.Requests = 0
	remote.result.ResponseBytes = 0
	remote.result.Unparsed = 10
	ob := Observation{Identity: "identity", Metrics: map[string]float64{"requests_total": 130, "connections_active": 1}}
	s.collectTraffic(context.Background(), i, source, spec, &ob, nil)
	var sample model.TrafficSample
	s.DB.Order("id DESC").First(&sample)
	values, _ := decode[TrafficValues](sample.Metrics)
	if values.ResponseBytesPerSecond != nil {
		t.Fatal("unsupported or incomplete window returned false zero")
	}
}
