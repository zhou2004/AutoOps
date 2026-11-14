package dao

import (
	"dodevops-api/api/monitor/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"path/filepath"
	"testing"
)

func TestDomainCertCheckCannotOverwriteChangedEndpointOrNewerResult(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "cert.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.DomainCert{}); err != nil {
		t.Fatal(err)
	}
	row := model.DomainCert{Domain: "example.test", Port: 443, CheckTime: "2026-11-14 10:00:00", Status: 1}
	if err = db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	d := &DomainCertDao{db: db}
	old := row
	row.Status = 4
	row.CheckTime = "2026-11-14 10:05:00"
	row.RemainingDays = -1
	if err = d.UpdateCheck(&row); err != nil {
		t.Fatal(err)
	}
	old.Status = 2
	if err = d.UpdateCheck(&old); err != nil {
		t.Fatal(err)
	}
	got, _ := d.GetByID(row.ID)
	if got.Status != 4 || got.CheckTime != row.CheckTime {
		t.Fatal("older result overwrote latest check")
	}
	if err = db.Model(&row).Update("port", 8443).Error; err != nil {
		t.Fatal(err)
	}
	old.CheckTime = "2026-11-14 11:00:00"
	old.Status = 1
	if err = d.UpdateCheck(&old); err != nil {
		t.Fatal(err)
	}
	got, _ = d.GetByID(row.ID)
	if got.Status != 4 || got.Port != 8443 {
		t.Fatal("old endpoint probe overwrote changed endpoint")
	}
}
