package db

import (
	"dodevops-api/api/monitor/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"path/filepath"
	"testing"
)

func TestLegacyMonitorIncidentMigrationPreservesRecords(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := database.DB()
	t.Cleanup(func() { raw.Close() })
	if err := database.Exec(`CREATE TABLE monitor_incident (id INTEGER PRIMARY KEY, alert_time TEXT, alert_desc TEXT, alert_level TEXT, status INTEGER, handler TEXT, create_time DATETIME, update_time DATETIME)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`INSERT INTO monitor_incident VALUES (1,'2026-10-01 08:00:00','disk full','P1',1,'operator','2026-10-01','2026-10-02'),(2,'2026-10-01 08:00:00','recovered','P4',3,'operator','2026-10-01',NULL)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.MonitorIncident{}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := migrateLegacyMonitorIncidents(database); err != nil {
			t.Fatal(err)
		}
	}
	var list []model.MonitorIncident
	if err := database.Order("id").Find(&list).Error; err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Title != "disk full" || list[0].Status != "firing" || list[0].Source != "legacy" || list[0].Level != "critical" || list[1].Status != "resolved" || list[1].ResolvedAt == "" {
		t.Fatalf("legacy history not preserved: %+v", list)
	}
	var handler string
	database.Table("monitor_incident").Select("handler").Where("id = 1").Scan(&handler)
	if handler != "operator" {
		t.Fatal("migration removed legacy metadata")
	}
}
