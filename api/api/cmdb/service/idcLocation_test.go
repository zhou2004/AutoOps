package service

import (
	"encoding/json"
	"math"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"dodevops-api/api/cmdb/model"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestIDCLocationValidation(t *testing.T) {
	ptr := func(v float64) *float64 { return &v }
	for _, test := range []struct {
		name     string
		lon, lat *float64
		valid    bool
	}{
		{"unlocated", nil, nil, true},
		{"zero is a coordinate", ptr(0), ptr(0), true},
		{"positive boundaries", ptr(180), ptr(90), true},
		{"negative boundaries", ptr(-180), ptr(-90), true},
		{"longitude only", ptr(12), nil, false},
		{"latitude only", nil, ptr(12), false},
		{"longitude out of range", ptr(180.1), ptr(0), false},
		{"latitude out of range", ptr(0), ptr(-90.1), false},
		{"not finite", ptr(math.Inf(1)), ptr(0), false},
		{"not a number", ptr(0), ptr(math.NaN()), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateIDCLocation(&CmdbIDCReq{Longitude: test.lon, Latitude: test.lat})
			if (err == nil) != test.valid {
				t.Fatalf("unexpected location validation: %v", err)
			}
		})
	}
}

func TestIDCLocationPersistsClearsAndRejectsPartialUpdate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "location.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	t.Cleanup(func() { raw.Close() })
	if err = db.AutoMigrate(&model.CmdbIDC{}); err != nil {
		t.Fatal(err)
	}
	service := NewCmdbIDCService(db)
	call := func(run func(*gin.Context), code int) model.CmdbIDC {
		t.Helper()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		run(c)
		var response struct {
			Code int           `json:"code"`
			Data model.CmdbIDC `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if response.Code != code {
			t.Fatalf("unexpected API response: %s", w.Body.String())
		}
		return response.Data
	}
	lon, lat := 116.4074, 39.9042
	req := &CmdbIDCReq{Name: "test-dc", Country: "中国", City: "北京", Longitude: &lon, Latitude: &lat}
	created := call(func(c *gin.Context) { service.Create(c, req) }, 200)
	check := func() model.CmdbIDC {
		t.Helper()
		var stored model.CmdbIDC
		if err := db.First(&stored, created.ID).Error; err != nil {
			t.Fatal(err)
		}
		return stored
	}
	stored := check()
	if stored.Longitude == nil || *stored.Longitude != lon || stored.Latitude == nil || *stored.Latitude != lat || stored.City != "北京" || stored.Country != "中国" {
		t.Fatal("geography lost on create")
	}
	req.Latitude = nil
	call(func(c *gin.Context) { service.Update(c, created.ID, req) }, 400)
	if stored = check(); stored.Latitude == nil || *stored.Latitude != lat {
		t.Fatal("invalid update changed stored location")
	}
	zero := 0.0
	req.Longitude, req.Latitude = &zero, &zero
	call(func(c *gin.Context) { service.Update(c, created.ID, req) }, 200)
	if stored = check(); stored.Longitude == nil || *stored.Longitude != 0 || stored.Latitude == nil || *stored.Latitude != 0 {
		t.Fatal("zero location treated as empty")
	}
	req.Longitude, req.Latitude, req.Country, req.City = nil, nil, "", ""
	call(func(c *gin.Context) { service.Update(c, created.ID, req) }, 200)
	if stored = check(); stored.Longitude != nil || stored.Latitude != nil || stored.Country != "" || stored.City != "" {
		t.Fatal("location could not be cleared")
	}
}
