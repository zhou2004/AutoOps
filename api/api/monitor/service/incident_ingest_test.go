package service

import (
	"dodevops-api/api/monitor/dao"
	"dodevops-api/api/monitor/model"
	dbpkg "dodevops-api/pkg/db"
	"encoding/json"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"path/filepath"
	"testing"
	"time"
)

func incidentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "incident.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	t.Cleanup(func() { raw.Close() })
	if err := db.AutoMigrate(&model.MonitorIncident{}); err != nil {
		t.Fatal(err)
	}
	return db
}
func testAlert(status, source, id, rule string, start time.Time) map[string]interface{} {
	return map[string]interface{}{"status": status, "alerts": []interface{}{map[string]interface{}{
		"status": status, "startsAt": start.Format(time.RFC3339), "endsAt": start.Add(time.Minute).Format(time.RFC3339),
		"labels":      map[string]string{"alertname": "availability", "severity": "critical", "monitor_source": source, "monitor_source_id": id, "monitor_rule_id": rule, "last_response_ms": "123"},
		"annotations": map[string]string{"description": "endpoint unavailable"},
	}}}
}
func TestIncidentLifecycleDedupRecoveryAndNewEpisode(t *testing.T) {
	db := incidentTestDB(t)
	d := dao.NewIncidentDaoWithDB(db)
	start := time.Now().Add(-time.Hour)
	send := func(p map[string]interface{}) {
		t.Helper()
		if err := ingestIncidentAlerts(d, p); err != nil {
			t.Fatal(err)
		}
	}
	send(testAlert("firing", "api_endpoint", "7", "1", start))
	send(testAlert("firing", "api_endpoint", "7", "1", start))
	changed := testAlert("firing", "api_endpoint", "7", "1", start)
	changed["alerts"].([]interface{})[0].(map[string]interface{})["labels"].(map[string]string)["last_response_ms"] = "900"
	send(changed)
	list, total, err := d.GetList(&model.IncidentListReq{})
	if err != nil || total != 1 || list[0].SourceID != 7 {
		t.Fatalf("duplicate event not deduped: %d %v", total, err)
	}
	send(testAlert("resolved", "api_endpoint", "7", "1", start))
	send(changed)
	m, _ := d.GetByID(list[0].ID)
	if m.Status != "resolved" || m.ResolvedAt == "" {
		t.Fatal("late firing reopened recovered incident")
	}
	// New episode, separate rule, and separate source must remain distinct.
	send(testAlert("firing", "api_endpoint", "7", "1", start.Add(time.Minute)))
	send(testAlert("firing", "api_endpoint", "7", "2", start))
	send(testAlert("firing", "domain_cert", "7", "1", start))
	_, total, _ = d.GetList(&model.IncidentListReq{})
	if total != 4 {
		t.Fatalf("different episodes merged: %d", total)
	}
	stats, err := d.GetStats()
	if err != nil || stats.TotalFiring != 3 || stats.TotalResolved != 1 || stats.CriticalFiring != 3 || len(stats.Trend) != 7 {
		t.Fatalf("incorrect stats: %+v %v", stats, err)
	}
	// Resolved arriving before firing stays resolved.
	send(testAlert("resolved", "prometheus", "", "", start))
	send(testAlert("firing", "prometheus", "", "", start))
	_, total, _ = d.GetList(&model.IncidentListReq{Source: "prometheus", Status: "resolved"})
	if total != 1 {
		t.Fatal("out-of-order recovery lost")
	}
}
func TestIncidentManualResolveAndStatsErrors(t *testing.T) {
	db := incidentTestDB(t)
	d := dao.NewIncidentDaoWithDB(db)
	start := time.Now().Add(-time.Hour)
	p := testAlert("firing", "domain_cert", "3", "5", start)
	if err := ingestIncidentAlerts(d, p); err != nil {
		t.Fatal(err)
	}
	list, _, _ := d.GetList(&model.IncidentListReq{})
	if err := d.Resolve(list[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := ingestIncidentAlerts(d, p); err != nil {
		t.Fatal(err)
	}
	m, _ := d.GetByID(list[0].ID)
	if m.Status != "resolved" {
		t.Fatal("manual resolution overwritten")
	}
	_, total, err := d.GetList(&model.IncidentListReq{Keyword: "unavailable", StartDate: start.Format("2006-01-02"), EndDate: start.Format("2006-01-02")})
	if err != nil || total != 1 {
		t.Fatal("description/date query failed")
	}
	if err := d.Resolve(999); err == nil {
		t.Fatal("missing incident reported success")
	}
	if err := db.Migrator().DropTable(&model.MonitorIncident{}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.GetStats(); err == nil {
		t.Fatal("missing data source silently converted to zeros")
	}
}
func TestInternalMetricIdentitySurvivesMeasurementChanges(t *testing.T) {
	a := map[string]string{"monitor_source_id": "3", "status": "2", "last_response_ms": "10"}
	b := map[string]string{"monitor_source_id": "3", "status": "1", "last_response_ms": "300"}
	if internalMetricFingerprint(a) != internalMetricFingerprint(b) {
		t.Fatal("measurements changed incident identity")
	}
	var recovered map[string]string
	if err := json.Unmarshal([]byte(internalMetricFingerprint(a)), &recovered); err != nil || recovered["monitor_source_id"] != "3" {
		t.Fatal("removed target cannot be recovered from identity")
	}
	b["monitor_source_id"] = "4"
	if internalMetricFingerprint(a) == internalMetricFingerprint(b) {
		t.Fatal("different targets share identity")
	}
}

func TestWebhookRecordsIncidentWithoutNotificationRoutes(t *testing.T) {
	database := incidentTestDB(t)
	if err := database.AutoMigrate(&model.PrometheusAlertDB{}); err != nil {
		t.Fatal(err)
	}
	previous := dbpkg.Db
	dbpkg.Db = database
	t.Cleanup(func() { dbpkg.Db = previous })
	service := NewAlertService()
	payload := testAlert("firing", "api_endpoint", "12", "4", time.Now().Add(-time.Minute))
	if _, err := service.PrometheusAlertHandle(model.PrometheusAlertMsg{}, payload, payload, nil); err != nil {
		t.Fatal(err)
	}
	records, total, err := dao.NewIncidentDaoWithDB(database).GetList(&model.IncidentListReq{})
	if err != nil || total != 1 || records[0].Source != "api_endpoint" {
		t.Fatalf("no-route webhook lost incident: %d %v", total, err)
	}
}
