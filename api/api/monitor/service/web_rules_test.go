package service

import (
	web "dodevops-api/api/webservice/model"
	"encoding/json"
	"testing"
	"time"
)

func TestWebTrafficRuleRejectsStaleOrFailedObservations(t *testing.T) {
	now := time.Now()
	old := now.Add(-91 * time.Second)
	observed := json.RawMessage(`{"metrics":{"qps":200,"response_bytes_per_second":1234}}`)
	for _, tc := range []struct {
		name, status string
		at           *time.Time
		wantMetrics  bool
	}{
		{"fresh", "healthy", &now, true},
		{"fresh drift", "drifted", &now, true},
		{"stale", "healthy", &old, false},
		{"failed collection", "unavailable", &now, false},
		{"never collected", "unknown", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := webInstanceRuleValues(web.Resource{Observed: observed, CollectedAt: tc.at, Status: tc.status}, now)
			_, got := v["qps"]
			if got != tc.wantMetrics {
				t.Fatalf("unexpected availability: %#v", v)
			}
			if _, ok := v["average_latency_ms"]; ok {
				t.Fatal("missing latency was synthesized")
			}
		})
	}
}
