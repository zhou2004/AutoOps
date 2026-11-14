package service

import (
	"context"
	"dodevops-api/api/webservice/model"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math"
	"sort"
	"strings"
	"time"
)

type TrafficValues struct {
	QPS                    *float64 `json:"qps"`
	ActiveConnections      *float64 `json:"activeConnections"`
	ResponseBytesPerSecond *float64 `json:"responseBytesPerSecond"`
	RequestBytesPerSecond  *float64 `json:"requestBytesPerSecond"`
	ErrorRate              *float64 `json:"errorRate"`
	AverageLatencyMS       *float64 `json:"averageLatencyMs"`
	LoggedRequests         *float64 `json:"loggedRequests"`
}
type TrafficCount struct {
	Value         string `json:"value"`
	Requests      uint64 `json:"requests"`
	ResponseBytes uint64 `json:"responseBytes"`
}
type trafficCount struct {
	Requests      uint64 `json:"requests"`
	ResponseBytes uint64 `json:"responseBytes"`
}
type trafficCursor struct {
	Capabilities map[string]bool `json:"capabilities"`
}
type trafficLogs struct {
	RateValid        bool                    `json:"rateValid"`
	Cursors          json.RawMessage         `json:"cursors"`
	Requests         uint64                  `json:"requests"`
	ResponseBytes    uint64                  `json:"responseBytes"`
	RequestBytes     uint64                  `json:"requestBytes"`
	RequestByteCount uint64                  `json:"requestByteCount"`
	LatencyTotalMS   float64                 `json:"latencyTotalMs"`
	LatencyCount     uint64                  `json:"latencyCount"`
	Errors           uint64                  `json:"errors"`
	Unparsed         uint64                  `json:"unparsed"`
	ScannedBytes     uint64                  `json:"scannedBytes"`
	SupportedLogs    int                     `json:"supportedLogs"`
	Partial          bool                    `json:"partial"`
	Initialized      bool                    `json:"initialized"`
	StatusCodes      map[string]uint64       `json:"statusCodes"`
	IPs              map[string]trafficCount `json:"ips"`
	URIs             map[string]trafficCount `json:"uris"`
	Upstreams        map[string]trafficCount `json:"upstreams"`
	Messages         []string                `json:"messages"`
}
type TrafficCapabilities struct {
	StubStatus   bool `json:"stubStatus"`
	AccessLog    bool `json:"accessLog"`
	RequestBytes bool `json:"requestBytes"`
	Latency      bool `json:"latency"`
	Upstream     bool `json:"upstream"`
}
type TrafficNode struct {
	InstanceID   uint                `json:"instanceId"`
	SourceID     uint                `json:"sourceId"`
	ClusterID    uint                `json:"clusterId"`
	Name         string              `json:"name"`
	Address      string              `json:"address"`
	Status       string              `json:"status"`
	CollectedAt  *time.Time          `json:"collectedAt"`
	Metrics      TrafficValues       `json:"metrics"`
	Capabilities TrafficCapabilities `json:"capabilities"`
	Message      string              `json:"message"`
}
type TrafficPoint struct {
	At time.Time `json:"at"`
	TrafficValues
}
type TrafficCoverage struct {
	Instances              int      `json:"instances"`
	FreshInstances         int      `json:"freshInstances"`
	StubStatusInstances    int      `json:"stubStatusInstances"`
	LogInstances           int      `json:"logInstances"`
	LogAuthorizedInstances int      `json:"logAuthorizedInstances"`
	Partial                bool     `json:"partial"`
	Notes                  []string `json:"notes"`
}
type TrafficResult struct {
	Range           string          `json:"range"`
	From            time.Time       `json:"from"`
	To              time.Time       `json:"to"`
	IntervalSeconds int             `json:"intervalSeconds"`
	Summary         TrafficValues   `json:"summary"`
	Series          []TrafficPoint  `json:"series"`
	Nodes           []TrafficNode   `json:"nodes"`
	TopIPs          []TrafficCount  `json:"topIPs"`
	TopURIs         []TrafficCount  `json:"topURIs"`
	Upstreams       []TrafficCount  `json:"upstreams"`
	StatusCodes     []TrafficCount  `json:"statusCodes"`
	Coverage        TrafficCoverage `json:"coverage"`
}
type TrafficFilter struct {
	Range                          string
	GroupID, ClusterID, InstanceID uint
}

func number(v float64) *float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return nil
	}
	return &v
}

// Shared files cannot be attributed to a specific Nginx master. Exclude them
// rather than counting the same requests once for every source on the host.
func (s *Service) uniqueTrafficLogs(i model.Resource, spec model.SourceSpec, ob Observation) ([]map[string]string, []string) {
	logs := []map[string]string{}
	messages := []string{}
	var others []model.Resource
	if err := s.DB.Table(model.Tables["instances"]).Where("id <> ?", i.ID).Find(&others).Error; err != nil {
		return logs, []string{"Access-log ownership check failed"}
	}
	for _, log := range ob.Logs {
		if log["kind"] != "access" {
			continue
		}
		shared := false
		for _, other := range others {
			op, _ := decode[model.SourceSpec](other.Spec)
			oo, _ := decode[Observation](other.Observed)
			if op.HostID != spec.HostID || op.Runtime != spec.Runtime {
				continue
			}
			if spec.Runtime == "docker" && oo.Identity != ob.Identity {
				continue
			}
			for _, l := range oo.Logs {
				if l["kind"] == "access" && l["realPath"] != "" && l["realPath"] == log["realPath"] {
					shared = true
					break
				}
			}
			if shared {
				break
			}
		}
		if shared {
			messages = append(messages, "Shared access log excluded: requests cannot be attributed to one Nginx instance")
		} else {
			logs = append(logs, log)
		}
	}
	return logs, messages
}
func (s *Service) collectTraffic(ctx context.Context, i, source model.Resource, spec model.SourceSpec, ob *Observation, discoveryError error) {
	for _, key := range []string{"qps", "response_bytes_per_second", "request_bytes_per_second", "error_rate", "average_latency_ms"} {
		delete(ob.Metrics, key)
	}
	var old model.TrafficState
	err := s.DB.First(&old, "instance_id = ?", i.ID).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	now := time.Now().UTC()
	seconds := now.Sub(old.CollectedAt).Seconds()
	messages := []string{}
	cursors := old.Cursors
	gap := old.InstanceID != 0 && seconds > 90
	reset := old.InstanceID == 0 || old.SourceRevision != source.Revision || old.Identity != ob.Identity || gap
	if gap {
		messages = append(messages, "Collection gap exceeded 90 seconds; log cursors were reset instead of attributing old requests to the current interval")
	}
	if reset {
		cursors = JSON(map[string]any{})
		seconds = 0
	}
	logs := trafficLogs{Cursors: cursors}
	metrics := TrafficValues{}
	if discoveryError == nil {
		registered, notes := s.uniqueTrafficLogs(i, spec, *ob)
		messages = append(messages, notes...)
		child, cancel := context.WithTimeout(ctx, 25*time.Second)
		raw, e := s.Remote.Run(child, RemoteRequest{Action: "traffic", Spec: spec, Identity: ob.Identity, Logs: registered, TrafficCursors: cursors})
		cancel()
		if e == nil {
			e = json.Unmarshal(raw, &logs)
		}
		if e != nil {
			logs = trafficLogs{Cursors: JSON(map[string]any{}), Partial: true}
			messages = append(messages, "Access-log collection failed: "+e.Error(), "Failed log cursor cleared; recovery starts a new sampling baseline")
		}
		messages = append(messages, logs.Messages...)
		logs.Messages = nil
		if len(notes) > 0 {
			logs.Partial = true
		}
		if v, ok := ob.Metrics["connections_active"]; ok {
			metrics.ActiveConnections = number(v)
		}
		previous, _ := decode[map[string]float64](old.Metrics)
		count, valid := ob.Metrics["requests_total"]
		prior, priorValid := previous["requests_total"]
		if !reset && seconds > 0 && seconds <= 90 && valid && priorValid && count >= prior {
			metrics.QPS = number((count - prior) / seconds)
		} else if valid {
			messages = append(messages, "Request counter baseline initialized or reset; QPS resumes after the next sample")
		}
		if logs.RateValid && logs.SupportedLogs > 0 && !logs.Initialized && seconds > 0 && seconds <= 90 {
			metrics.ResponseBytesPerSecond = number(float64(logs.ResponseBytes) / seconds)
			metrics.LoggedRequests = number(float64(logs.Requests))
			if logs.RequestByteCount == logs.Requests && logs.RequestByteCount > 0 {
				metrics.RequestBytesPerSecond = number(float64(logs.RequestBytes) / seconds)
			}
			if logs.Requests > 0 {
				metrics.ErrorRate = number(100 * float64(logs.Errors) / float64(logs.Requests))
			}
			if logs.LatencyCount > 0 {
				metrics.AverageLatencyMS = number(logs.LatencyTotalMS / float64(logs.LatencyCount))
			}
		}
	} else {
		messages = append(messages, "Instance discovery failed; traffic metrics are unavailable")
	}
	if spec.StatusURL == "" {
		messages = append(messages, "stub_status endpoint is not configured")
	} else if metrics.ActiveConnections == nil {
		messages = append(messages, "stub_status collection failed")
	}
	if len(messages) > 12 {
		messages = messages[:12]
	}
	for n := range messages {
		if len(messages[n]) > 350 {
			messages[n] = messages[n][:350]
		}
	}
	state := model.TrafficState{InstanceID: i.ID, SourceRevision: source.Revision, Identity: ob.Identity, CollectedAt: now, Metrics: JSON(ob.Metrics), Cursors: logs.Cursors, Revision: old.Revision + 1}
	// The cursor is committed with the sample, and CAS rejects concurrent/late collectors.
	transactionErr := s.DB.Transaction(func(tx *gorm.DB) error {
		var current model.Resource
		if e := tx.Table(model.Tables["sources"]).Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, source.ID).Error; e != nil {
			return e
		}
		if current.Revision != source.Revision {
			return ErrConflict
		}
		if old.InstanceID == 0 {
			if e := tx.Create(&state).Error; e != nil {
				return e
			}
		} else {
			r := tx.Model(&model.TrafficState{}).Where("instance_id = ? AND revision = ?", i.ID, old.Revision).Updates(map[string]any{"source_revision": state.SourceRevision, "identity": state.Identity, "collected_at": now, "metrics": state.Metrics, "cursors": state.Cursors, "revision": state.Revision})
			if r.Error != nil {
				return r.Error
			}
			if r.RowsAffected != 1 {
				return ErrConflict
			}
		}
		logs.Cursors = nil
		return tx.Create(&model.TrafficSample{InstanceID: i.ID, At: now, Seconds: seconds, Metrics: JSON(metrics), Logs: JSON(logs), Messages: JSON(messages)}).Error
	})
	if transactionErr != nil {
		return
	}
	if ob.Metrics == nil {
		ob.Metrics = map[string]float64{}
	}
	for k, v := range trafficMetricMap(metrics) {
		ob.Metrics[k] = v
	}
}

func logCapabilities(log trafficLogs, cursor json.RawMessage) TrafficCapabilities {
	caps := TrafficCapabilities{AccessLog: log.SupportedLogs > 0}
	var cursors map[string]trafficCursor
	_ = json.Unmarshal(cursor, &cursors)
	for _, c := range cursors {
		caps.RequestBytes = caps.RequestBytes || c.Capabilities["requestBytes"]
		caps.Latency = caps.Latency || c.Capabilities["latency"]
		caps.Upstream = caps.Upstream || c.Capabilities["upstream"]
	}
	return caps
}
func redactTrafficLogs(v *TrafficValues) {
	v.ResponseBytesPerSecond = nil
	v.RequestBytesPerSecond = nil
	v.ErrorRate = nil
	v.AverageLatencyMS = nil
	v.LoggedRequests = nil
}
func addValue(target **float64, value *float64) {
	if value == nil {
		return
	}
	if *target == nil {
		*target = number(0)
	}
	**target += *value
}
func (s *Service) Traffic(ctx context.Context, u uint, f TrafficFilter) (TrafficResult, error) {
	now := time.Now().UTC()
	duration := time.Hour
	interval := 30
	switch f.Range {
	case "", "1h":
		f.Range = "1h"
	case "6h":
		duration = 6 * time.Hour
		interval = 60
	case "24h":
		duration = 24 * time.Hour
		interval = 300
	case "7d":
		duration = 7 * 24 * time.Hour
		interval = 1800
	default:
		return TrafficResult{}, errors.New("unsupported traffic range")
	}
	out := TrafficResult{Range: f.Range, From: now.Add(-duration), To: now, IntervalSeconds: interval, Series: []TrafficPoint{}, Nodes: []TrafficNode{}, TopIPs: []TrafficCount{}, TopURIs: []TrafficCount{}, Upstreams: []TrafficCount{}, StatusCodes: []TrafficCount{}, Coverage: TrafficCoverage{Notes: []string{}}}
	if e := s.Module(ctx, u, "read"); e != nil {
		return out, e
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var instances []model.Resource
	q := s.DB.WithContext(ctx).Table(model.Tables["instances"])
	if f.ClusterID > 0 {
		q = q.Where("cluster_id = ?", f.ClusterID)
	}
	if f.InstanceID > 0 {
		q = q.Where("id = ?", f.InstanceID)
	}
	if f.GroupID > 0 {
		ids, e := s.GroupIDs(f.GroupID)
		if e != nil {
			return out, e
		}
		q = q.Where("group_id IN ?", ids)
	}
	if e := q.Order("id").Find(&instances).Error; e != nil {
		return out, e
	}
	allowed := []uint{}
	byID := map[uint]model.Resource{}
	canLogs := map[uint]bool{}
	for _, i := range instances {
		if s.Authorize(u, "instances", i, "web_view") != nil {
			continue
		}
		allowed = append(allowed, i.ID)
		byID[i.ID] = i
		canLogs[i.ID] = s.Authorize(u, "instances", i, "web_logs") == nil
		if canLogs[i.ID] {
			out.Coverage.LogAuthorizedInstances++
		}
	}
	out.Coverage.Instances = len(allowed)
	if len(allowed) == 0 {
		return out, nil
	}
	var states []model.TrafficState
	if e := s.DB.WithContext(ctx).Where("instance_id IN ?", allowed).Find(&states).Error; e != nil {
		return out, e
	}
	stateMap := map[uint]model.TrafficState{}
	for _, st := range states {
		stateMap[st.InstanceID] = st
	}
	rows, err := s.DB.WithContext(ctx).Model(&model.TrafficSample{}).Where("instance_id IN ? AND at >= ? AND at <= ?", allowed, out.From, now).Order("at DESC").Limit(100001).Rows()
	if err != nil {
		return out, err
	}
	defer rows.Close()
	latest := map[uint]model.TrafficSample{}
	buckets := map[int64]map[uint]*trafficAccumulator{}
	totals := trafficAccumulator{}
	current := trafficAccumulator{}
	count, scannedBytes := 0, 0
	for rows.Next() {
		var sample model.TrafficSample
		if err := s.DB.ScanRows(rows, &sample); err != nil {
			return out, err
		}
		count++
		scannedBytes += len(sample.Metrics) + len(sample.Logs) + len(sample.Messages)
		if count > 100000 || scannedBytes > 32<<20 {
			out.Coverage.Partial = true
			out.Coverage.Notes = append(out.Coverage.Notes, "History reached the 100000-sample or 32 MiB query limit; older data in this range is omitted")
			break
		}
		values, _ := decode[TrafficValues](sample.Metrics)
		logs, _ := decode[trafficLogs](sample.Logs)
		if !canLogs[sample.InstanceID] {
			redactTrafficLogs(&values)
			logs = trafficLogs{}
		}
		bucket := sample.At.Unix() / int64(interval) * int64(interval)
		if buckets[bucket] == nil {
			buckets[bucket] = map[uint]*trafficAccumulator{}
		}
		if buckets[bucket][sample.InstanceID] == nil {
			buckets[bucket][sample.InstanceID] = &trafficAccumulator{}
		}
		buckets[bucket][sample.InstanceID].add(values, logs)
		totals.addCounts(logs)
		if logs.Partial {
			out.Coverage.Partial = true
		}
		if _, ok := latest[sample.InstanceID]; !ok {
			// Top lists are accumulated once globally, not retained per node/bucket.
			logs.IPs = nil
			logs.URIs = nil
			logs.Upstreams = nil
			logs.StatusCodes = nil
			sample.Logs = JSON(logs)
			sample.Metrics = JSON(values)
			latest[sample.InstanceID] = sample
		}
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	rows.Close()
	for _, id := range allowed {
		i := byID[id]
		node := TrafficNode{InstanceID: id, SourceID: i.SourceID, ClusterID: i.ClusterID, Name: i.Name, Status: "unknown"}
		s.Redact(u, "instances", &i)
		if i.Connection != nil {
			node.Address = i.Connection.Address
		}
		sample, ok := latest[id]
		if ok {
			node.CollectedAt = &sample.At
			if now.Sub(sample.At) <= 90*time.Second {
				node.Metrics, _ = decode[TrafficValues](sample.Metrics)
				logs, _ := decode[trafficLogs](sample.Logs)
				node.Capabilities = logCapabilities(logs, stateMap[id].Cursors)
				node.Capabilities.StubStatus = node.Metrics.ActiveConnections != nil
				messages, _ := decode[[]string](sample.Messages)
				node.Message = strings.Join(messages, "; ")
				if !canLogs[id] {
					redactTrafficLogs(&node.Metrics)
					logs = trafficLogs{}
					node.Capabilities = TrafficCapabilities{StubStatus: node.Capabilities.StubStatus}
					node.Message = "Access-log metrics require log permission"
				}
				if node.Capabilities.StubStatus || node.Capabilities.AccessLog {
					out.Coverage.FreshInstances++
					node.Status = "healthy"
					if !node.Capabilities.StubStatus || !node.Capabilities.AccessLog || logs.Partial {
						node.Status = "partial"
					}
				}
				if node.Capabilities.StubStatus {
					out.Coverage.StubStatusInstances++
				}
				if node.Capabilities.AccessLog {
					out.Coverage.LogInstances++
				}
				current.add(node.Metrics, logs)
			} else {
				node.Message = "No traffic sample in the last 90 seconds"
			}
		} else {
			node.Message = "Waiting for the first traffic sample"
		}
		out.Nodes = append(out.Nodes, node)
	}
	out.Summary = current.sumValues()
	if totals.requests > 0 || out.Coverage.LogInstances > 0 {
		out.Summary.LoggedRequests = number(float64(totals.requests))
	}
	for stamp := out.From.Unix() / int64(interval) * int64(interval); stamp <= now.Unix(); stamp += int64(interval) {
		sum := trafficAccumulator{}
		for _, acc := range buckets[stamp] {
			sum.add(acc.meanValues(), acc.logs())
		}
		out.Series = append(out.Series, TrafficPoint{At: time.Unix(stamp, 0).UTC(), TrafficValues: sum.sumValues()})
	}
	out.TopIPs = rankTraffic(totals.ips)
	out.TopURIs = rankTraffic(totals.uris)
	out.Upstreams = rankTraffic(totals.upstreams)
	out.StatusCodes = rankTraffic(totals.codes)
	if out.Coverage.FreshInstances < len(allowed) || out.Coverage.LogInstances < out.Coverage.LogAuthorizedInstances {
		out.Coverage.Partial = true
	}
	if totals.partial {
		out.Coverage.Partial = true
		out.Coverage.Notes = append(out.Coverage.Notes, "Top-list aggregation reached the 5000-key limit")
	}
	if out.Coverage.Partial {
		out.Coverage.Notes = append(out.Coverage.Notes, "Missing, unparsed, shared or bounded log data is excluded; figures represent observed traffic only")
	}
	if out.Coverage.LogAuthorizedInstances < len(allowed) {
		out.Coverage.Notes = append(out.Coverage.Notes, "Log-derived metrics are limited to instances with log permission")
	}
	out.Coverage.Notes = append(out.Coverage.Notes, "QPS is derived from Nginx counter deltas; byte rates represent logged HTTP bytes, not NIC bandwidth. 5xx responses define the error rate. Log statistics use collection intervals; bounded backlog is disclosed as partial.")
	return out, nil
}

type trafficAccumulator struct {
	sums                                                    [4]float64
	counts                                                  [4]int
	requests, bytes, inBytes, inCount, errors, latencyCount uint64
	latency                                                 float64
	ips, uris, upstreams, codes                             map[string]trafficCount
	partial                                                 bool
}

func (a *trafficAccumulator) add(v TrafficValues, logs trafficLogs) {
	for n, p := range []*float64{v.QPS, v.ActiveConnections, v.ResponseBytesPerSecond, v.RequestBytesPerSecond} {
		if p != nil {
			a.sums[n] += *p
			a.counts[n]++
		}
	}
	a.addNumbers(logs)
}
func (a *trafficAccumulator) addNumbers(l trafficLogs) {
	a.requests += l.Requests
	a.bytes += l.ResponseBytes
	a.inBytes += l.RequestBytes
	a.inCount += l.RequestByteCount
	a.errors += l.Errors
	a.latency += l.LatencyTotalMS
	a.latencyCount += l.LatencyCount
}
func (a *trafficAccumulator) addCounts(l trafficLogs) {
	a.addNumbers(l)
	if a.ips == nil {
		a.ips = map[string]trafficCount{}
		a.uris = map[string]trafficCount{}
		a.upstreams = map[string]trafficCount{}
		a.codes = map[string]trafficCount{}
	}
	merge := func(to map[string]trafficCount, from map[string]trafficCount) {
		for k, v := range from {
			if _, ok := to[k]; !ok && len(to) >= 5000 {
				a.partial = true
				continue
			}
			x := to[k]
			x.Requests += v.Requests
			x.ResponseBytes += v.ResponseBytes
			to[k] = x
		}
	}
	merge(a.ips, l.IPs)
	merge(a.uris, l.URIs)
	merge(a.upstreams, l.Upstreams)
	for k, v := range l.StatusCodes {
		x := a.codes[k]
		x.Requests += v
		a.codes[k] = x
	}
}
func (a *trafficAccumulator) logs() trafficLogs {
	return trafficLogs{Requests: a.requests, ResponseBytes: a.bytes, RequestBytes: a.inBytes, RequestByteCount: a.inCount, Errors: a.errors, LatencyTotalMS: a.latency, LatencyCount: a.latencyCount}
}
func (a *trafficAccumulator) sumValues() TrafficValues {
	v := TrafficValues{}
	for n, p := range []**float64{&v.QPS, &v.ActiveConnections, &v.ResponseBytesPerSecond, &v.RequestBytesPerSecond} {
		if a.counts[n] > 0 {
			*p = number(a.sums[n])
		}
	}
	if a.requests > 0 {
		v.ErrorRate = number(100 * float64(a.errors) / float64(a.requests))
		v.LoggedRequests = number(float64(a.requests))
	}
	if a.latencyCount > 0 {
		v.AverageLatencyMS = number(a.latency / float64(a.latencyCount))
	}
	return v
}
func (a *trafficAccumulator) meanValues() TrafficValues {
	v := a.sumValues()
	for n, p := range []*float64{v.QPS, v.ActiveConnections, v.ResponseBytesPerSecond, v.RequestBytesPerSecond} {
		if p != nil {
			*p /= float64(a.counts[n])
		}
	}
	return v
}
func rankTraffic(values map[string]trafficCount) []TrafficCount {
	out := []TrafficCount{}
	for k, v := range values {
		out = append(out, TrafficCount{Value: k, Requests: v.Requests, ResponseBytes: v.ResponseBytes})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Requests == out[j].Requests {
			return out[i].Value < out[j].Value
		}
		return out[i].Requests > out[j].Requests
	})
	if len(out) > 20 {
		out = out[:20]
	}
	return out
}

// TrafficRuleMetrics is a read-only, fresh-sample view for monitoring rules.
func (s *Service) TrafficRuleMetrics(instanceID uint) map[string]float64 {
	var sample model.TrafficSample
	if s.DB.Where("instance_id = ? AND at >= ?", instanceID, time.Now().UTC().Add(-90*time.Second)).Order("at DESC").First(&sample).Error != nil {
		return map[string]float64{}
	}
	values, _ := decode[TrafficValues](sample.Metrics)
	return trafficMetricMap(values)
}
func trafficMetricMap(v TrafficValues) map[string]float64 {
	out := map[string]float64{}
	for k, p := range map[string]*float64{"qps": v.QPS, "response_bytes_per_second": v.ResponseBytesPerSecond, "request_bytes_per_second": v.RequestBytesPerSecond, "error_rate": v.ErrorRate, "average_latency_ms": v.AverageLatencyMS} {
		if p != nil {
			out[k] = *p
		}
	}
	return out
}
