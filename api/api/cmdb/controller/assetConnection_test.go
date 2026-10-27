package controller

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"dodevops-api/api/cmdb/model"
	"dodevops-api/api/cmdb/service"
	configmodel "dodevops-api/api/configcenter/model"
	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common/constant"
	"dodevops-api/common/util"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func assetControllerFixture(t *testing.T) (*AssetConnectionController, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "controller.db")+"?_busy_timeout=5000"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.CmdbNetworkDevice{}, &model.CmdbAssetConnection{}, &model.CmdbConnectionAudit{}, &model.CmdbConnectionTicket{}, &model.CmdbAssetPermission{}, &model.CmdbCredentialPermission{}, &model.CmdbUserGroup{}, &model.CmdbUserGroupMember{}, &configmodel.EcsAuth{}); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{"CREATE TABLE sys_admin (id integer PRIMARY KEY, status integer)", "INSERT INTO sys_admin (id, status) VALUES (7,1),(8,1)", "CREATE TABLE sys_role (id integer PRIMARY KEY, role_key text, status integer)", "CREATE TABLE sys_admin_role (admin_id integer, role_id integer)"} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, device := range []model.CmdbNetworkDevice{{ID: 1, SN: "ONE", Name: "one", DeviceType: 1}, {ID: 2, SN: "TWO", Name: "two", DeviceType: 1}} {
		if err := db.Create(&device).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, profile := range []model.CmdbAssetConnection{
		{ID: 31, AssetType: "network", AssetID: 1, Name: "primary", Protocol: "ssh", Address: "192.0.2.1", Port: 22, Enabled: true, CredentialID: 51},
		{ID: 32, AssetType: "network", AssetID: 2, Name: "other", Protocol: "ssh", Address: "192.0.2.2", Port: 22, Enabled: true, CredentialID: 51},
	} {
		if err := db.Create(&profile).Error; err != nil {
			t.Fatal(err)
		}
	}
	credential := configmodel.EcsAuth{ID: 51, Name: "operators", Type: 1, Username: "operator", Password: "test-only-password", CreateTime: util.HTime{Time: time.Now()}}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.CmdbAssetPermission{Name: "network-operators", UserIDs: "[7]", GroupIDs: "[]", AssetTypes: `["network"]`, NetworkIDs: "[1]", PermissionActions: `["get","update","connect_ssh"]`, IsActive: 1}).Error; err != nil {
		t.Fatal(err)
	}
	return NewAssetConnectionController(db), db
}

func assetControllerRouter(ctrl *AssetConnectionController, userID uint) *gin.Engine {
	router := gin.New()
	if userID > 0 {
		router.Use(func(c *gin.Context) { c.Set(constant.ContextKeyUserObj, &systemmodel.JwtAdmin{ID: userID}); c.Next() })
	}
	ctrl.Register(router.Group("/cmdb"))
	router.GET("/remote", ctrl.Connect)
	return router
}

func assetControllerRequest(t *testing.T, router http.Handler, method, path, body string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var response struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid response: %s", w.Body.String())
	}
	return response.Code, response.Data
}

func TestAssetConnectionControllerScopeAndAssetRevocation(t *testing.T) {
	ctrl, db := assetControllerFixture(t)
	router := assetControllerRouter(ctrl, 7)
	code, data := assetControllerRequest(t, router, "GET", "/cmdb/assets/network/1/connections", "")
	if code != 200 {
		t.Fatalf("allowed list returned %d", code)
	}
	connections := data["connections"].([]interface{})
	if len(connections) != 1 || !connections[0].(map[string]interface{})["canConnect"].(bool) {
		t.Fatal("list did not scope profiles correctly")
	}
	if code, _ := assetControllerRequest(t, router, "POST", "/cmdb/assets/network/1/connections/32/ticket", ""); code != 404 {
		t.Fatalf("cross-asset profile returned %d", code)
	}
	if code, _ := assetControllerRequest(t, router, "POST", "/cmdb/assets/network/2/connections/32/ticket", ""); code != 403 {
		t.Fatalf("asset outside grant returned %d", code)
	}
	if code, _ := assetControllerRequest(t, assetControllerRouter(ctrl, 0), "GET", "/cmdb/assets/network/1/connections", ""); code != 401 {
		t.Fatalf("anonymous list returned %d", code)
	}
	if err := db.Model(&model.CmdbAssetPermission{}).Where("name = ?", "network-operators").Update("permission_actions", `["get","update"]`).Error; err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"ticket", "test"} {
		if code, _ := assetControllerRequest(t, router, "POST", "/cmdb/assets/network/1/connections/31/"+suffix, ""); code != 403 {
			t.Fatalf("revoked asset permission %s returned %d", suffix, code)
		}
	}
	code, data = assetControllerRequest(t, router, "GET", "/cmdb/assets/network/1/connections", "")
	if code != 200 || data["connections"].([]interface{})[0].(map[string]interface{})["canConnect"].(bool) {
		t.Fatal("revoked asset permission remains connectable in list")
	}
}

func TestRemoteOriginPolicy(t *testing.T) {
	t.Setenv("CMDB_REMOTE_ORIGINS", "https://ops.example.test, http://localhost:5173")
	for _, tc := range []struct {
		origin  string
		allowed bool
	}{
		{"", true}, {"https://api.example.test", true}, {"https://ops.example.test", true}, {"http://localhost:5173", true},
		{"https://attacker.test", false}, {"https://ops.example.test.evil", false}, {"null", false}, {"https://u:p@api.example.test", false},
		{"https://api.example.test/path", false}, {"https://api.example.test?query=1", false}, {"https://api.example.test#fragment", false}, {"javascript:alert(1)", false},
		{"https://api.example.test?", false}, {"https://api.example.test#", false}, {"https://", false},
	} {
		r := httptest.NewRequest("GET", "http://api.example.test/remote", nil)
		r.Header.Set("Origin", tc.origin)
		if got := remoteOriginAllowed(r, ""); got != tc.allowed {
			t.Errorf("origin %q: allowed=%v want %v", tc.origin, got, tc.allowed)
		}
	}
}

func TestAssetOperatorsConnectWithoutCredentialGrantsButCannotRebind(t *testing.T) {
	ctrl, db := assetControllerFixture(t)
	router := assetControllerRouter(ctrl, 7)
	// No independent credential grant exists in this fixture.
	if code, _ := assetControllerRequest(t, router, "POST", "/cmdb/assets/network/1/connections/31/ticket", ""); code != 200 {
		t.Fatalf("asset connection still depends on a credential grant: %d", code)
	}
	for _, path := range []string{"/cmdb/assets/network/1/connections", "/cmdb/assets/network/1/connections/31"} {
		method := "POST"
		if strings.HasSuffix(path, "/31") {
			method = "PUT"
		}
		if code, _ := assetControllerRequest(t, router, method, path, `{"name":"redirect","protocol":"ssh","address":"192.0.2.99","port":22,"credentialId":51}`); code != 403 {
			t.Fatalf("asset operator can rebind a shared credential: %d", code)
		}
	}
	if code, _ := assetControllerRequest(t, router, "GET", "/cmdb/connection-credentials", ""); code != 403 {
		t.Fatalf("operator can enumerate unrelated credentials: %d", code)
	}
	for _, sql := range []string{"INSERT INTO sys_role VALUES (1,'admin',1)", "INSERT INTO sys_admin_role VALUES (8,1)"} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	code, data := assetControllerRequest(t, assetControllerRouter(ctrl, 8), "GET", "/cmdb/connection-credentials", "")
	if code != 200 || len(data["list"].([]interface{})) != 1 {
		t.Fatal("administrator cannot configure credentials")
	}
	encoded, _ := json.Marshal(data)
	if strings.Contains(string(encoded), "password") {
		t.Fatal("credential secret leaked")
	}
	if err := db.Exec("UPDATE sys_admin SET status = 0 WHERE id = 8").Error; err != nil {
		t.Fatal(err)
	}
	if code, _ := assetControllerRequest(t, assetControllerRouter(ctrl, 8), "GET", "/cmdb/connection-credentials", ""); code != 403 {
		t.Fatal("inactive administrator accepted")
	}
}

func TestAssetConnectionTicketRejectsInvalidOrigin(t *testing.T) {
	ctrl, db := assetControllerFixture(t)
	router := assetControllerRouter(ctrl, 7)
	for _, origin := range []string{"null", "https://", "https://user:password@example.test", "https://example.test/path", "https://example.test?", "https://example.test#", "https://" + strings.Repeat("a", 512)} {
		req := httptest.NewRequest(http.MethodPost, "/cmdb/assets/network/1/connections/31/ticket", nil)
		req.Header.Set("Origin", origin)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		var body struct {
			Code int `json:"code"`
		}
		if json.Unmarshal(response.Body.Bytes(), &body) != nil || body.Code != 400 {
			t.Fatalf("invalid ticket origin accepted: %q", origin)
		}
	}
	var count int64
	if err := db.Model(&model.CmdbConnectionTicket{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("invalid origins created connection tickets")
	}
}

// Vite and the API use different ports in development. The authenticated
// ticket request must authorize its own browser origin for the later socket.
func TestAssetConnectionCrossOriginTicket(t *testing.T) {
	ctrl, db := assetControllerFixture(t)
	t.Setenv("CMDB_REMOTE_ORIGINS", "https://another-allowed.test")
	if err := db.Model(&model.CmdbAssetConnection{}).Where("id = ?", 31).Updates(map[string]interface{}{"address": "127.0.0.1", "port": 1}).Error; err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(assetControllerRouter(ctrl, 7))
	defer server.Close()
	origin := "http://localhost:8080"
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/cmdb/assets/network/1/connections/31/ticket", nil)
	req.Header.Set("Origin", origin)
	response, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var issued struct {
		Code int `json:"code"`
		Data struct {
			Ticket string `json:"ticket"`
		} `json:"data"`
	}
	err = json.NewDecoder(response.Body).Decode(&issued)
	response.Body.Close()
	if err != nil || issued.Code != 200 || issued.Data.Ticket == "" {
		t.Fatal("cross-origin authenticated ticket issuance failed")
	}
	endpoint := "ws" + strings.TrimPrefix(server.URL, "http") + "/remote?ticket=" + url.QueryEscape(issued.Data.Ticket)
	for _, wrongOrigin := range []string{"", "null", "https://attacker.invalid", "https://another-allowed.test", server.URL} {
		ws, res, err := websocket.DefaultDialer.Dial(endpoint, http.Header{"Origin": []string{wrongOrigin}})
		if ws != nil {
			ws.Close()
		}
		if err == nil || res == nil || res.StatusCode != http.StatusForbidden {
			t.Fatalf("ticket accepted a different origin %q", wrongOrigin)
		}
	}
	var stored model.CmdbConnectionTicket
	if err := db.First(&stored).Error; err != nil || stored.ConsumedAt != nil {
		t.Fatal("rejected origins consumed the ticket")
	}
	ws, res, err := websocket.DefaultDialer.Dial(endpoint, http.Header{"Origin": []string{origin}})
	if err != nil {
		if res != nil {
			t.Fatalf("authenticated browser origin rejected: HTTP %d", res.StatusCode)
		}
		t.Fatal("authenticated browser origin rejected")
	}
	defer ws.Close()
	_ = ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	// No SSH server is listening, so the application must report failure.
	kind, data, err := ws.ReadMessage()
	var event struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}
	if err != nil || kind != websocket.TextMessage || json.Unmarshal(data, &event) != nil || event.Type != "error" || event.Message == "" {
		t.Fatal("failed SSH setup did not report an application error")
	}
	if err := db.First(&stored).Error; err != nil || stored.ConsumedAt == nil {
		t.Fatal("accepted origin did not consume ticket")
	}
}

func TestAssetConnectionWebSocketRejectsOriginAndRevokedAccess(t *testing.T) {
	ctrl, db := assetControllerFixture(t)
	t.Setenv("CMDB_REMOTE_ORIGINS", "")
	server := httptest.NewServer(assetControllerRouter(ctrl, 0))
	defer server.Close()
	token, err := service.CreateConnectionTicket(db, 7, 31, "")
	if err != nil {
		t.Fatal(err)
	}
	endpoint := "ws" + strings.TrimPrefix(server.URL, "http") + "/remote?ticket=" + url.QueryEscape(token)
	conn, response, err := websocket.DefaultDialer.Dial(endpoint, http.Header{"Origin": []string{"https://attacker.invalid"}})
	if conn != nil {
		conn.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatal("cross-origin WebSocket accepted")
	}
	var stored model.CmdbConnectionTicket
	if err := db.First(&stored).Error; err != nil || stored.ConsumedAt != nil {
		t.Fatal("rejected origin consumed a valid ticket")
	}
	if err := db.Model(&model.CmdbAssetPermission{}).Where("name = ?", "network-operators").Update("permission_actions", `["get","update"]`).Error; err != nil {
		t.Fatal(err)
	}
	conn, response, err = websocket.DefaultDialer.Dial(endpoint, http.Header{"Origin": []string{server.URL}})
	if conn != nil {
		conn.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatal("asset permission revoked after ticket issuance still accepted")
	}
	_, response, err = websocket.DefaultDialer.Dial(endpoint, http.Header{"Origin": []string{server.URL}})
	if err == nil || response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatal("ticket was replayable")
	}
}

// Opt-in end-to-end check of the HTTP ticket and WebSocket terminal protocol.
// It runs only printf and exit; no device files or configuration are changed.
func TestAssetTerminalLive(t *testing.T) {
	host, user, password := os.Getenv("CMDB_TEST_HOST"), os.Getenv("CMDB_TEST_USER"), os.Getenv("CMDB_TEST_PASSWORD")
	if host == "" || user == "" || password == "" {
		t.Skip("set CMDB_TEST_HOST, CMDB_TEST_USER and CMDB_TEST_PASSWORD for a read-only terminal check")
	}
	port := 22
	if raw := os.Getenv("CMDB_TEST_PORT"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 65535 {
			t.Fatal("invalid CMDB_TEST_PORT")
		}
		port = parsed
	}
	ctrl, db := assetControllerFixture(t)
	fingerprint, err := service.DiscoverAssetHostKey(host, port)
	if err != nil {
		t.Fatal("SSH host-key discovery failed")
	}
	if err := db.Model(&configmodel.EcsAuth{}).Where("id = ?", 51).Updates(map[string]interface{}{"username": user, "password": password}).Error; err != nil {
		t.Fatal("test credential initialization failed")
	}
	if err := db.Model(&model.CmdbAssetConnection{}).Where("id = ?", 31).Updates(map[string]interface{}{"address": host, "port": port, "username": user, "host_key": fingerprint, "enabled": true}).Error; err != nil {
		t.Fatal("test connection initialization failed")
	}
	server := httptest.NewServer(assetControllerRouter(ctrl, 7))
	defer server.Close()
	origin := "http://localhost:8080"
	request, _ := http.NewRequest(http.MethodPost, server.URL+"/cmdb/assets/network/1/connections/31/ticket", nil)
	request.Header.Set("Origin", origin)
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal("HTTP ticket request failed")
	}
	var ticketResponse struct {
		Code int `json:"code"`
		Data struct {
			Ticket string `json:"ticket"`
		} `json:"data"`
	}
	err = json.NewDecoder(response.Body).Decode(&ticketResponse)
	response.Body.Close()
	if err != nil || ticketResponse.Code != 200 || ticketResponse.Data.Ticket == "" {
		t.Fatal("HTTP ticket issuance failed")
	}
	endpoint := "ws" + strings.TrimPrefix(server.URL, "http") + "/remote?ticket=" + url.QueryEscape(ticketResponse.Data.Ticket)
	ws, _, err := websocket.DefaultDialer.Dial(endpoint, http.Header{"Origin": []string{origin}})
	if err != nil {
		t.Fatal("terminal WebSocket upgrade failed")
	}
	defer ws.Close()
	_ = ws.SetReadDeadline(time.Now().Add(45 * time.Second))
	if err := ws.WriteJSON(map[string]interface{}{"type": "resize", "cols": 120, "rows": 35}); err != nil {
		t.Fatal("terminal resize event failed")
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal("test marker initialization failed")
	}
	marker := "AUTOOPS_WS_" + hex.EncodeToString(nonce) + "_OK"
	var octal strings.Builder
	for _, b := range []byte(marker) {
		fmt.Fprintf(&octal, "\\%03o", b)
	}
	// Only the octal representation is sent to the shell. Seeing the literal
	// marker therefore proves printf executed, rather than matching PTY echo.
	command := "printf '" + octal.String() + "\\n'\nexit\n"
	if strings.Contains(command, marker) {
		t.Fatal("test command permits an echo false-positive")
	}
	if err := ws.WriteJSON(map[string]string{"type": "input", "data": command}); err != nil {
		t.Fatal("terminal input event failed")
	}
	var received strings.Builder
	found := false
	ready := false
	for {
		kind, data, readErr := ws.ReadMessage()
		if readErr != nil {
			break
		}
		if kind == websocket.TextMessage {
			var event struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(data, &event) != nil || event.Type != "ready" {
				t.Fatal("SSH returned an error or unexpected control event")
			}
			ready = true
			continue
		}
		if kind != websocket.BinaryMessage {
			t.Fatal("SSH terminal returned an unexpected frame type")
		}
		if received.Len()+len(data) > 1024*1024 {
			t.Fatal("unexpectedly large terminal output")
		}
		received.Write(data)
		found = found || strings.Contains(received.String(), marker)
	}
	_ = ws.Close()
	if !found || !ready {
		t.Fatal("terminal did not execute the read-only marker command")
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		var audit model.CmdbConnectionAudit
		err := db.Where("connection_id = ? AND action = ?", 31, "connect").Order("id DESC").First(&audit).Error
		if err == nil && audit.EndedAt != nil {
			if audit.Status != "closed" || audit.UserID != 7 || audit.Protocol != "ssh" {
				t.Fatal("terminal session did not finish with the expected audit")
			}
			t.Log("HTTP ticket, WebSocket resize/input, remote command output and final session audit verified")
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("terminal completion audit was not recorded")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestDesktopWebSocketFiltersReadOnlyAndFileEvents(t *testing.T) {
	for _, readOnly := range []bool{true, false} {
		t.Run(fmt.Sprintf("readOnly=%t", readOnly), func(t *testing.T) {
			ctrl, _ := assetControllerFixture(t)
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			t.Setenv("CMDB_GUACD_ADDR", listener.Addr().String())
			forwarded := make(chan []string, 1)
			gatewayErrors := make(chan error, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					gatewayErrors <- err
					return
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(4 * time.Second))
				reader := bufio.NewReader(conn)
				if _, err := service.ReadGuacamole(reader); err != nil {
					gatewayErrors <- err
					return
				}
				if _, err := io.WriteString(conn, service.EncodeGuacamole("args", "hostname", "password", "read-only")); err != nil {
					gatewayErrors <- err
					return
				}
				for i := 0; i < 5; i++ {
					if _, err := service.ReadGuacamole(reader); err != nil {
						gatewayErrors <- err
						return
					}
				}
				if _, err := io.WriteString(conn, service.EncodeGuacamole("ready", "mock-session")); err != nil {
					gatewayErrors <- err
					return
				}
				ops := []string{}
				for {
					instruction, err := service.ReadGuacamole(reader)
					if err != nil {
						gatewayErrors <- err
						return
					}
					ops = append(ops, instruction[0])
					if instruction[0] == "nop" {
						forwarded <- ops
						return
					}
				}
			}()
			profile := model.CmdbAssetConnection{Protocol: "rdp", Address: "192.0.2.30", Port: 3389, CredentialID: 51, ReadOnly: readOnly, Enabled: true}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upgrader := websocket.Upgrader{}
				ws, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer ws.Close()
				_ = ctrl.serveDesktop(&profile, ws, &remoteWriter{ws: ws}, 1280, 800)
			}))
			defer server.Close()
			ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer ws.Close()
			_ = ws.SetReadDeadline(time.Now().Add(3 * time.Second))
			if _, _, err := ws.ReadMessage(); err != nil {
				t.Fatal("desktop handshake failed")
			}
			frames := service.EncodeGuacamole("key", "65", "1") + service.EncodeGuacamole("mouse", "10", "20", "1") + service.EncodeGuacamole("clipboard", "0", "text/plain") + service.EncodeGuacamole("file", "1", "text/plain", "danger.txt") + service.EncodeGuacamole("blob", "1", "c2VjcmV0") + service.EncodeGuacamole("sync", "123") + service.EncodeGuacamole("nop")
			if err := ws.WriteMessage(websocket.TextMessage, []byte(frames)); err != nil {
				t.Fatal(err)
			}
			select {
			case ops := <-forwarded:
				want := []string{"sync", "nop"}
				if !readOnly {
					want = []string{"key", "mouse", "sync", "nop"}
				}
				if !reflect.DeepEqual(ops, want) {
					t.Fatalf("unsafe events forwarded: %v", ops)
				}
			case err := <-gatewayErrors:
				t.Fatal(err)
			case <-time.After(5 * time.Second):
				t.Fatal("desktop event forwarding timed out")
			}
		})
	}
}
