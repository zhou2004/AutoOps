package service

import (
	"dodevops-api/api/k8s/model"
	"encoding/json"
	"testing"
)

func TestCephQuotaOnlyChangesExplicitFields(t *testing.T) {
	for _, tc := range []struct{ body, field, value string }{
		{`{"maxBytes":0}`, "max_bytes", "0"},
		{`{"maxBytes":1073741824}`, "max_bytes", "1073741824"},
		{`{"maxObjects":123}`, "max_objects", "123"},
	} {
		var req model.UpdateCephPoolQuotaRequest
		if err := json.Unmarshal([]byte(tc.body), &req); err != nil {
			t.Fatal(err)
		}
		commands, err := cephQuotaCommands("replicapool", &req)
		if err != nil || len(commands) != 1 || commands[0][4] != tc.field || commands[0][5] != tc.value {
			t.Fatalf("%s: got %#v, %v", tc.body, commands, err)
		}
	}
	if _, err := cephQuotaCommands("replicapool", &model.UpdateCephPoolQuotaRequest{}); err == nil {
		t.Fatal("empty update must not clear quotas")
	}
}
