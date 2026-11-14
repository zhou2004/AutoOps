package controller

import (
	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/webservice/model"
	"dodevops-api/api/webservice/service"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTerminalTicketOriginExpiryReplayAndRevocation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "tickets.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	defer raw.Close()
	if err = model.Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&cmdb.CmdbAssetConnection{}); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		"CREATE TABLE sys_admin (id INTEGER PRIMARY KEY, status INTEGER, username TEXT)",
		"CREATE TABLE sys_role (id INTEGER PRIMARY KEY, role_key TEXT, status INTEGER)",
		"CREATE TABLE sys_admin_role (admin_id INTEGER, role_id INTEGER)",
		"INSERT INTO sys_admin VALUES (1,1,'admin')", "INSERT INTO sys_role VALUES (1,'admin',1)", "INSERT INTO sys_admin_role VALUES (1,1)",
	} {
		if err = db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	spec := service.JSON(model.SourceSpec{Runtime: "host", HostID: 1, ConnectionID: 99})
	db.Table(model.Tables["sources"]).Create(&model.Resource{ID: 1, Name: "source", Spec: spec})
	db.Table(model.Tables["instances"]).Create(&model.Resource{ID: 1, Name: "instance", SourceID: 1})
	c := New(db)
	r := gin.New()
	r.GET("/terminal", c.Terminal)
	request := func(token, origin string) int {
		t.Helper()
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/terminal?ticket="+token, nil)
		req.Header.Set("Origin", origin)
		r.ServeHTTP(w, req)
		var body struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.Code
	}
	create := func(token string, expires time.Time) {
		t.Helper()
		if err := db.Create(&model.Ticket{Hash: ticketHash(token), UserID: 1, InstanceID: 1, Origin: "https://autoops.test", ExpiresAt: expires}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if request("missing", "https://autoops.test") != 401 {
		t.Fatal("unknown ticket accepted")
	}
	create("expired", time.Now().UTC().Add(-time.Second))
	if request("expired", "https://autoops.test") != 401 {
		t.Fatal("expired ticket accepted")
	}
	create("valid", time.Now().UTC().Add(time.Minute))
	if request("valid", "https://untrusted.test") != 403 {
		t.Fatal("foreign origin accepted")
	}
	var ticket model.Ticket
	db.First(&ticket, "hash = ?", ticketHash("valid"))
	if ticket.ConsumedAt != nil {
		t.Fatal("foreign origin consumed ticket")
	}
	// Missing SSH profile stops before network access, after the one-use claim.
	if request("valid", "https://autoops.test") != 403 {
		t.Fatal("missing profile was accepted")
	}
	if request("valid", "https://autoops.test") != 401 {
		t.Fatal("ticket replay accepted")
	}
	create("revoked", time.Now().UTC().Add(time.Minute))
	db.Exec("UPDATE sys_admin SET status = 2 WHERE id = 1")
	if request("revoked", "https://autoops.test") != 403 {
		t.Fatal("disabled account accepted")
	}
	var revoked model.Ticket
	db.First(&revoked, "hash = ?", ticketHash("revoked"))
	if revoked.ConsumedAt != nil {
		t.Fatal("revoked account reached ticket claim")
	}
}
