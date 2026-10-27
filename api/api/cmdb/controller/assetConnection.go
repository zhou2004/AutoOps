package controller

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/model"
	"dodevops-api/api/cmdb/service"
	configmodel "dodevops-api/api/configcenter/model"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

type AssetConnectionController struct {
	db          *gorm.DB
	permissions *dao.CmdbAssetPermissionDao
}

func NewAssetConnectionController(db *gorm.DB) *AssetConnectionController {
	return &AssetConnectionController{db: db, permissions: dao.NewCmdbAssetPermissionDao(db)}
}

func (ctrl *AssetConnectionController) Register(group *gin.RouterGroup) {
	group.GET("/connection-credentials", ctrl.Credentials)
	base := "/assets/:assetType/:assetId/connections"
	group.GET(base, ctrl.List)
	group.POST(base, ctrl.Save)
	group.POST(base+"/host-key", ctrl.HostKey)
	group.PUT(base+"/:connectionId", ctrl.Save)
	group.DELETE(base+"/:connectionId", ctrl.Delete)
	group.POST(base+"/:connectionId/test", ctrl.Test)
	group.POST(base+"/:connectionId/ticket", ctrl.Ticket)
	group.GET("/assets/:assetType/:assetId/connection-audits", ctrl.Audits)
}

func (ctrl *AssetConnectionController) Credentials(c *gin.Context) {
	uid, err := jwt.GetAdminId(c)
	if err != nil || uid == 0 {
		result.Failed(c, 401, "请先登录")
		return
	}
	if admin, err := ctrl.permissions.IsAdministrator(uid); err != nil || !admin {
		result.Failed(c, 403, "连接凭据由管理员配置")
		return
	}
	var credentials []configmodel.EcsAuth
	if err := ctrl.db.Select("id", "name", "type", "username").Order("id ASC").Find(&credentials).Error; err != nil {
		result.Failed(c, 500, "读取凭据失败")
		return
	}
	items := []gin.H{}
	for _, credential := range credentials {
		items = append(items, gin.H{"id": credential.ID, "name": credential.Name, "type": credential.Type, "username": credential.Username})
	}
	result.Success(c, gin.H{"list": items})
}

func (ctrl *AssetConnectionController) identity(c *gin.Context) (uint, string, uint, bool) {
	userID, err := jwt.GetAdminId(c)
	if err != nil || userID == 0 {
		result.Failed(c, 401, "请先登录")
		return 0, "", 0, false
	}
	if !ctrl.accountActive(userID) {
		result.Failed(c, 403, "账户不存在或已停用")
		return 0, "", 0, false
	}
	id, err := strconv.ParseUint(c.Param("assetId"), 10, 32)
	if err != nil || id == 0 {
		result.Failed(c, 400, "资产 ID 无效")
		return 0, "", 0, false
	}
	typ := c.Param("assetType")
	if err := service.AssetExists(ctrl.db, typ, uint(id)); err != nil {
		result.Failed(c, 404, "资产不存在")
		return 0, "", 0, false
	}
	return userID, typ, uint(id), true
}

func (ctrl *AssetConnectionController) allowed(userID uint, typ string, id uint, action string) bool {
	// Connection endpoints and credential bindings are shared infrastructure.
	// Asset operators use the existing profile; only active administrators configure it.
	if action == "update" {
		ok, err := ctrl.permissions.IsAdministrator(userID)
		return err == nil && ok
	}
	ok, err := ctrl.permissions.CanUserAccess(userID, typ, id, action)
	return err == nil && ok
}

func (ctrl *AssetConnectionController) accountActive(userID uint) bool {
	ok, err := ctrl.permissions.UserActive(userID)
	return err == nil && ok
}

func validRemoteOrigin(origin string) bool {
	u, err := url.Parse(origin)
	return len(origin) <= 512 && err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && u.Path == "" && u.Opaque == "" && !strings.ContainsAny(origin, " \t\r\n#")
}

func remoteOriginAllowed(r *http.Request, ticketOrigin string) bool {
	origin := r.Header.Get("Origin")
	// The ticket was issued after bearer authentication and asset
	// authorization. Bind it to that browser's exact origin, including its port.
	if ticketOrigin != "" {
		return validRemoteOrigin(origin) && origin == ticketOrigin
	}
	if origin == "" {
		return true
	}
	if !validRemoteOrigin(origin) {
		return false
	}
	u, _ := url.Parse(origin)
	if u.Host == r.Host {
		return true
	}
	for _, allowed := range strings.Split(os.Getenv("CMDB_REMOTE_ORIGINS"), ",") {
		if strings.TrimSpace(allowed) == origin {
			return true
		}
	}
	return false
}

func (ctrl *AssetConnectionController) List(c *gin.Context) {
	uid, typ, id, ok := ctrl.identity(c)
	if !ok {
		return
	}
	manage := ctrl.allowed(uid, typ, id, "update")
	dockerView := ctrl.allowed(uid, typ, id, "docker_view")
	dockerOperate := ctrl.allowed(uid, typ, id, "docker_operate")
	dockerRemove := ctrl.allowed(uid, typ, id, "docker_remove")
	visible := dockerView || dockerOperate || manage || ctrl.allowed(uid, typ, id, "get") || ctrl.allowed(uid, typ, id, "list") || ctrl.allowed(uid, typ, id, "audit")
	if typ == "network" {
		for _, action := range []string{"config_view", "config_backup", "config_restore"} {
			visible = visible || ctrl.allowed(uid, typ, id, action)
		}
	}
	for _, protocol := range []string{"ssh", "vnc", "rdp"} {
		visible = visible || ctrl.allowed(uid, typ, id, "connect_"+protocol)
	}
	if !visible {
		result.Failed(c, 403, "没有该资产的连接查看权限")
		return
	}
	profiles := []model.CmdbAssetConnection{}
	if err := ctrl.db.Where("asset_type = ? AND asset_id = ?", typ, id).Order("id ASC").Find(&profiles).Error; err != nil {
		result.Failed(c, 500, "读取连接失败")
		return
	}
	type view struct {
		model.CmdbAssetConnection
		CanConnect bool `json:"canConnect"`
		CanTest    bool `json:"canTest"`
		CanManage  bool `json:"canManage"`
		CanDocker  bool `json:"canDocker"`
	}
	items := []view{}
	for _, p := range profiles {
		allowed := p.Enabled && ctrl.allowed(uid, typ, id, "connect_"+p.Protocol)
		items = append(items, view{p, allowed, allowed, manage, p.Enabled && p.Protocol == "ssh" && (dockerView || dockerOperate)})
	}
	result.Success(c, gin.H{"connections": items, "canManage": manage, "canDockerOperate": dockerOperate, "canDockerRemove": dockerRemove, "gatewayAvailable": service.GuacdAddress() != ""})
}

func (ctrl *AssetConnectionController) Save(c *gin.Context) {
	uid, typ, id, ok := ctrl.identity(c)
	if !ok {
		return
	}
	if !ctrl.allowed(uid, typ, id, "update") {
		result.Failed(c, 403, "没有该资产的连接管理权限")
		return
	}
	var body struct {
		model.CmdbAssetConnection
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		result.Failed(c, 400, "连接参数无效")
		return
	}
	p := body.CmdbAssetConnection
	p.ID = 0
	p.AssetType = typ
	p.AssetID = id
	p.CreatedAt = time.Time{}
	p.UpdatedAt = time.Time{}
	p.Enabled = body.Enabled == nil || *body.Enabled
	if err := service.ValidateAssetConnection(&p); err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	if err := service.ValidateConnectionCredential(ctrl.db, &p); err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	action := "create"
	if c.Request.Method == http.MethodPut {
		connectionID, e := strconv.ParseUint(c.Param("connectionId"), 10, 32)
		if e != nil || connectionID == 0 {
			result.Failed(c, 400, "连接 ID 无效")
			return
		}
		existing, e := service.GetAssetConnection(ctrl.db, typ, id, uint(connectionID))
		if e != nil {
			result.Failed(c, 404, "连接不存在")
			return
		}
		p.ID = existing.ID
		p.CreatedAt = existing.CreatedAt
		action = "update"
	}
	if err := ctrl.db.Transaction(func(tx *gorm.DB) error {
		if p.ID == 0 {
			if e := tx.Create(&p).Error; e != nil {
				return e
			}
		} else {
			if e := tx.Model(&model.CmdbAssetConnection{}).Where("id = ? AND asset_type = ? AND asset_id = ?", p.ID, typ, id).Select("name", "protocol", "address", "port", "username", "credential_id", "domain", "security", "ignore_cert", "read_only", "enabled", "host_key", "updated_at").Updates(&p).Error; e != nil {
				return e
			}
		}
		return tx.Create(&model.CmdbConnectionAudit{AssetType: typ, AssetID: id, ConnectionID: p.ID, UserID: uid, Protocol: p.Protocol, Action: action, Status: "success", ClientIP: c.ClientIP(), StartedAt: time.Now().UTC()}).Error
	}); err != nil {
		result.Failed(c, 500, "保存连接失败")
		return
	}
	result.Success(c, p)
}

func (ctrl *AssetConnectionController) get(c *gin.Context, action string) (uint, *model.CmdbAssetConnection, bool) {
	uid, typ, id, ok := ctrl.identity(c)
	if !ok {
		return 0, nil, false
	}
	connectionID, err := strconv.ParseUint(c.Param("connectionId"), 10, 32)
	if err != nil || connectionID == 0 {
		result.Failed(c, 400, "连接 ID 无效")
		return 0, nil, false
	}
	p, err := service.GetAssetConnection(ctrl.db, typ, id, uint(connectionID))
	if err != nil {
		result.Failed(c, 404, "连接不存在")
		return 0, nil, false
	}
	if action == "connect" {
		action = "connect_" + p.Protocol
		if !p.Enabled {
			result.Failed(c, 400, "此连接已停用")
			return 0, nil, false
		}
	}
	if !ctrl.allowed(uid, typ, id, action) {
		result.Failed(c, 403, "没有该资产的操作权限")
		return 0, nil, false
	}
	return uid, p, true
}

func (ctrl *AssetConnectionController) Delete(c *gin.Context) {
	uid, p, ok := ctrl.get(c, "update")
	if !ok {
		return
	}
	if err := ctrl.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(p).Error; err != nil {
			return err
		}
		return tx.Create(&model.CmdbConnectionAudit{AssetType: p.AssetType, AssetID: p.AssetID, ConnectionID: p.ID, UserID: uid, Protocol: p.Protocol, Action: "delete", Status: "success", ClientIP: c.ClientIP(), StartedAt: time.Now().UTC()}).Error
	}); err != nil {
		result.Failed(c, 500, "删除连接失败")
		return
	}
	result.Success(c, true)
}

func (ctrl *AssetConnectionController) HostKey(c *gin.Context) {
	uid, typ, id, ok := ctrl.identity(c)
	if !ok {
		return
	}
	if !ctrl.allowed(uid, typ, id, "update") {
		result.Failed(c, 403, "没有连接管理权限")
		return
	}
	var body struct {
		Address string `json:"address"`
		Port    int    `json:"port"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		result.Failed(c, 400, "地址参数无效")
		return
	}
	key, err := service.DiscoverAssetHostKey(body.Address, body.Port)
	if err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	result.Success(c, gin.H{"hostKey": key})
}

func (ctrl *AssetConnectionController) Test(c *gin.Context) {
	uid, p, ok := ctrl.get(c, "connect")
	if !ok {
		return
	}
	reachable, latency, message := service.ProbeAssetConnection(ctrl.db, p)
	status := "failed"
	if reachable {
		status = "success"
	}
	if err := ctrl.db.Create(&model.CmdbConnectionAudit{AssetType: p.AssetType, AssetID: p.AssetID, ConnectionID: p.ID, UserID: uid, Protocol: p.Protocol, Action: "test", Status: status, Message: message, ClientIP: c.ClientIP(), StartedAt: time.Now().UTC()}).Error; err != nil {
		result.Failed(c, 500, "连接测试审计写入失败")
		return
	}
	result.Success(c, gin.H{"reachable": reachable, "latencyMs": latency, "message": message})
}

func (ctrl *AssetConnectionController) Ticket(c *gin.Context) {
	uid, p, ok := ctrl.get(c, "connect")
	if !ok {
		return
	}
	if p.Protocol != "ssh" && service.GuacdAddress() == "" {
		result.Failed(c, 503, "尚未配置图形连接网关 CMDB_GUACD_ADDR")
		return
	}
	origin := c.GetHeader("Origin")
	if origin != "" && !validRemoteOrigin(origin) {
		result.Failed(c, 400, "页面来源无效，请通过 HTTP 或 HTTPS 访问系统")
		return
	}
	token, err := service.CreateConnectionTicket(ctrl.db, uid, p.ID, origin)
	if err != nil {
		result.Failed(c, 500, "创建连接票据失败")
		return
	}
	c.Header("Cache-Control", "no-store")
	result.Success(c, gin.H{"ticket": token, "protocol": p.Protocol, "expiresIn": 60, "websocketPath": "/api/v1/cmdb/remote/connect?ticket=" + url.QueryEscape(token)})
}

func (ctrl *AssetConnectionController) Audits(c *gin.Context) {
	uid, typ, id, ok := ctrl.identity(c)
	if !ok {
		return
	}
	if !ctrl.allowed(uid, typ, id, "audit") {
		result.Failed(c, 403, "没有会话审计权限")
		return
	}
	items := []model.CmdbConnectionAudit{}
	if err := ctrl.db.Where("asset_type = ? AND asset_id = ?", typ, id).Order("id DESC").Limit(100).Find(&items).Error; err != nil {
		result.Failed(c, 500, "读取会话审计失败")
		return
	}
	result.Success(c, gin.H{"list": items})
}

type remoteWriter struct {
	ws *websocket.Conn
	mu sync.Mutex
}

func (w *remoteWriter) send(kind int, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return w.ws.WriteMessage(kind, data)
}
func (w *remoteWriter) Write(data []byte) (int, error) {
	err := w.send(websocket.BinaryMessage, data)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

// Connect authenticates only through an atomically consumed, 60-second ticket.
// The long-lived login JWT is never sent in the WebSocket URL.
func (ctrl *AssetConnectionController) Connect(c *gin.Context) {
	if !websocket.IsWebSocketUpgrade(c.Request) {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	ticket, err := service.ConsumeConnectionTicket(ctrl.db, c.Query("ticket"), func(origin string) bool {
		return remoteOriginAllowed(c.Request, origin)
	})
	if err != nil {
		if errors.Is(err, service.ErrConnectionOrigin) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	var p model.CmdbAssetConnection
	if err := ctrl.db.First(&p, ticket.ConnectionID).Error; err != nil || !p.Enabled {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	if !ctrl.accountActive(ticket.UserID) || service.AssetExists(ctrl.db, p.AssetType, p.AssetID) != nil || !ctrl.allowed(ticket.UserID, p.AssetType, p.AssetID, "connect_"+p.Protocol) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	upgrader := websocket.Upgrader{ReadBufferSize: 4096, WriteBufferSize: 16384, Subprotocols: []string{"guacamole"}, CheckOrigin: func(r *http.Request) bool {
		return remoteOriginAllowed(r, ticket.Origin)
	}}
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()
	w := &remoteWriter{ws: ws}
	ws.SetReadLimit(1024 * 1024)
	_ = ws.SetReadDeadline(time.Now().Add(75 * time.Second))
	ws.SetPongHandler(func(string) error { return ws.SetReadDeadline(time.Now().Add(75 * time.Second)) })
	audit := model.CmdbConnectionAudit{AssetType: p.AssetType, AssetID: p.AssetID, ConnectionID: p.ID, UserID: ticket.UserID, Protocol: p.Protocol, Action: "connect", Status: "connecting", ClientIP: c.ClientIP(), StartedAt: time.Now().UTC()}
	if ctrl.db.Create(&audit).Error != nil {
		return
	}
	status := "failed"
	defer func() {
		now := time.Now().UTC()
		ctrl.db.Model(&audit).Updates(map[string]interface{}{"status": status, "ended_at": now})
	}()
	done := make(chan struct{})
	defer close(done)
	markConnected := func() error {
		return ctrl.db.Model(&model.CmdbConnectionAudit{}).Where("id = ?", audit.ID).Update("status", "connected").Error
	}
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				var current model.CmdbAssetConnection
				if time.Since(audit.StartedAt) > 4*time.Hour || ctrl.db.First(&current, p.ID).Error != nil || !current.Enabled || !current.UpdatedAt.Equal(p.UpdatedAt) || service.AssetExists(ctrl.db, p.AssetType, p.AssetID) != nil || !ctrl.allowed(ticket.UserID, p.AssetType, p.AssetID, "connect_"+p.Protocol) {
					ws.Close()
					return
				}
				if w.send(websocket.PingMessage, nil) != nil {
					ws.Close()
					return
				}
			}
		}
	}()
	if p.Protocol == "ssh" {
		if err := ctrl.serveSSH(&p, ws, w, markConnected); err != nil {
			message := "SSH 会话建立或通信失败，请先在连接档案中检测凭据、服务和主机指纹。"
			ctrl.db.Model(&model.CmdbConnectionAudit{}).Where("id = ?", audit.ID).Update("message", message)
			payload, _ := json.Marshal(gin.H{"type": "error", "message": message})
			_ = w.send(websocket.TextMessage, payload)
			return
		}
	} else {
		width, _ := strconv.Atoi(c.Query("width"))
		height, _ := strconv.Atoi(c.Query("height"))
		if err := ctrl.serveDesktop(&p, ws, w, width, height, markConnected); err != nil {
			_ = w.send(websocket.TextMessage, []byte(service.EncodeGuacamole("error", "远程桌面连接已结束或连接失败", "519")))
			return
		}
	}
	status = "closed"
}

func (ctrl *AssetConnectionController) serveSSH(p *model.CmdbAssetConnection, ws *websocket.Conn, w *remoteWriter, onConnected ...func() error) error {
	client, err := service.DialAssetSSH(ctrl.db, p)
	if err != nil {
		return err
	}
	defer client.Close()
	// Bound channel/PTY setup as well as the TCP/authentication handshake.
	setupTimeout := time.AfterFunc(15*time.Second, func() { client.Close() })
	defer setupTimeout.Stop()
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	session.Stdout = w
	session.Stderr = w
	if err := session.RequestPty("xterm-256color", 30, 120, ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}); err != nil {
		return err
	}
	if err := session.Shell(); err != nil {
		return err
	}
	setupTimeout.Stop()
	for _, ready := range onConnected {
		if err := ready(); err != nil {
			return err
		}
	}
	if err := w.send(websocket.TextMessage, []byte(`{"type":"ready"}`)); err != nil {
		return err
	}
	go func() { _ = session.Wait(); ws.Close() }()
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			return nil
		}
		var msg struct {
			Type string `json:"type"`
			Data string `json:"data"`
			Cols int    `json:"cols"`
			Rows int    `json:"rows"`
		}
		if json.Unmarshal(data, &msg) != nil {
			return fmt.Errorf("invalid terminal message")
		}
		switch msg.Type {
		case "input":
			if _, err := io.WriteString(stdin, msg.Data); err != nil {
				return err
			}
		case "resize":
			if msg.Cols > 0 && msg.Cols <= 500 && msg.Rows > 0 && msg.Rows <= 200 {
				if err := session.WindowChange(msg.Rows, msg.Cols); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("invalid terminal event")
		}
	}
}

func (ctrl *AssetConnectionController) serveDesktop(p *model.CmdbAssetConnection, ws *websocket.Conn, w *remoteWriter, width, height int, onConnected ...func() error) error {
	conn, reader, err := service.ConnectGuacamole(ctrl.db, p, width, height)
	if err != nil {
		return err
	}
	defer conn.Close()
	for _, ready := range onConnected {
		if err := ready(); err != nil {
			return err
		}
	}
	if err := w.send(websocket.TextMessage, []byte(service.EncodeGuacamole("", uuid.NewString()))); err != nil {
		return err
	}
	go func() {
		defer ws.Close()
		for {
			instruction, err := service.ReadGuacamole(reader)
			if err != nil {
				return
			}
			if err := w.send(websocket.TextMessage, []byte(service.EncodeGuacamole(instruction...))); err != nil {
				return
			}
		}
	}()
	for {
		kind, data, err := ws.ReadMessage()
		if err != nil {
			return nil
		}
		if kind != websocket.TextMessage {
			return fmt.Errorf("invalid desktop frame")
		}
		input := bufio.NewReader(strings.NewReader(string(data)))
		for {
			if _, err := input.Peek(1); err == io.EOF {
				break
			}
			instruction, err := service.ReadGuacamole(input)
			if err != nil {
				return err
			}
			if !service.AllowedGuacamoleInput(instruction[0], p.ReadOnly) {
				continue
			}
			encoded := service.EncodeGuacamole(instruction...)
			if instruction[0] == "" {
				if err := w.send(websocket.TextMessage, []byte(encoded)); err != nil {
					return err
				}
				continue
			}
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, err := io.WriteString(conn, encoded); err != nil {
				return err
			}
		}
	}
}
