package service

import (
	"testing"
	"dodevops-api/api/monitor/model"
)

func TestDatabaseRuleDoesNotUseStaleMetricsOrCrossGroupLabels(t *testing.T) {
	rule:=&model.MonitorAlertRule{Expr:"disk_used_percent >= 80",Constraints:`{"engine":"mysql","business_group_id":"18"}`}
	labels:=map[string]string{"engine":"mysql","business_group_id":"18","db_cluster_id":"4","instance_id":"27","metric_kind":"filesystem"}
	if !matchDatabaseRule(rule,labels,map[string]float64{"disk_used_percent":85}) {t.Fatal("fresh capacity must match")}
	if matchDatabaseRule(rule,labels,map[string]float64{"state_unknown":1}) {t.Fatal("missing samples cannot be treated as capacity readings")}
	labels["business_group_id"]="19"
	if matchDatabaseRule(rule,labels,map[string]float64{"disk_used_percent":95}) {t.Fatal("cross group sample matched")}
	rule.Constraints="invalid-json"
	if matchDatabaseRule(rule,labels,map[string]float64{"disk_used_percent":95}) {t.Fatal("invalid constraints must fail closed")}
}

func TestDatabaseAlertFingerprintTracksEachInstanceAcrossRoleChanges(t *testing.T) {
	a:=map[string]string{"monitor_source_id":"4","db_cluster_id":"4","instance_id":"27","metric_kind":"instance","role":"primary"}
	b:=map[string]string{"monitor_source_id":"4","db_cluster_id":"4","instance_id":"28","metric_kind":"instance","role":"replica"}
	if internalMetricFingerprint(a)==internalMetricFingerprint(b){t.Fatal("two members collided")}
	before:=internalMetricFingerprint(a);a["role"]="replica"
	if internalMetricFingerprint(a)!=before{t.Fatal("role change must resolve the same pending/firing alert identity")}
}
