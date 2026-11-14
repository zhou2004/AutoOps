package service

import (
	cmdb "dodevops-api/api/cmdb/model"
	"testing"
)

func TestTrafficLogMetricsCannotBypassLogPermissionViaResourceDetail(t *testing.T) {
	s := fixture(t)
	s.DB.Create(&cmdb.CmdbAssetPermission{Name: "source view", UserIDs: "[2]", AssetTypes: `["web_source"]`, WebSourceIDs: "[1]", PermissionActions: `["web_view"]`, IsActive: 1})
	for _, kind := range []string{"sources", "instances"} {
		t.Run(kind, func(t *testing.T) {
			r, _ := s.Get(kind, 1)
			r.Observed = JSON(Observation{Metrics: map[string]float64{"qps": 12, "connections_active": 3, "response_bytes_per_second": 50, "request_bytes_per_second": 30, "error_rate": .2, "average_latency_ms": 40}})
			s.Redact(2, kind, &r)
			ob, _ := decode[Observation](r.Observed)
			if len(ob.Metrics) != 2 || ob.Metrics["qps"] != 12 || ob.Metrics["connections_active"] != 3 {
				t.Fatalf("unexpected scoped metrics: %#v", ob.Metrics)
			}
		})
	}
}
