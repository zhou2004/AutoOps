package controller

import (
	"crypto/rand"
	"crypto/sha256"
	cmdbdao "dodevops-api/api/cmdb/dao"
	cmdbmodel "dodevops-api/api/cmdb/model"
	cmdbservice "dodevops-api/api/cmdb/service"
	"dodevops-api/api/webservice/model"
	"dodevops-api/api/webservice/service"
	"dodevops-api/common/result"
	"dodevops-api/pkg/apikey"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func ticketHash(v string) string { h := sha256.Sum256([]byte(v)); return hex.EncodeToString(h[:]) }
func (c *Controller) terminalAllowed(ctx *gin.Context, u, iid uint) (model.Resource, model.SourceSpec, error) {
	i, e := c.S.Get("instances", iid)
	if e != nil {
		return i, model.SourceSpec{}, e
	}
	if e = c.S.Module(ctx.Request.Context(), u, "execute"); e != nil {
		return i, model.SourceSpec{}, e
	}
	if e = c.S.Authorize(u, "instances", i, "web_terminal"); e != nil {
		return i, model.SourceSpec{}, e
	}
	src, e := c.S.Get("sources", i.SourceID)
	if e != nil {
		return i, model.SourceSpec{}, e
	}
	var p model.SourceSpec
	json.Unmarshal(src.Spec, &p)
	action := "connect_ssh"
	if p.Runtime == "docker" {
		action = "docker_operate"
	}
	d := cmdbdao.NewCmdbAssetPermissionDao(c.S.DB)
	admin, _ := d.IsAdministrator(u)
	if !admin {
		grants, e := d.EffectivePermissions(u)
		if e != nil {
			return i, p, e
		}
		ids, e := d.AllowedAssetIDs(grants, "host", action)
		if e != nil || !ids[p.HostID] {
			return i, p, service.ErrForbidden
		}
	}
	return i, p, nil
}
func (c *Controller) TerminalTicket(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	var p struct {
		InstanceID uint `json:"instanceId"`
	}
	if !bind(ctx, &p) {
		return
	}
	if _, _, e := c.terminalAllowed(ctx, u, p.InstanceID); e != nil {
		fail(ctx, e)
		return
	}
	origin := ctx.GetHeader("Origin")
	parsed, e := url.Parse(origin)
	if e != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		result.Failed(ctx, 400, "终端需要有效页面来源")
		return
	}
	raw := make([]byte, 32)
	if _, e = rand.Read(raw); e != nil {
		fail(ctx, e)
		return
	}
	token := hex.EncodeToString(raw)
	t := model.Ticket{APIKeyID: apikey.ID(ctx.Request.Context()), Hash: ticketHash(token), UserID: u, InstanceID: p.InstanceID, Origin: origin, ExpiresAt: time.Now().UTC().Add(time.Minute)}
	if e = c.S.DB.Create(&t).Error; e != nil {
		fail(ctx, e)
		return
	}
	result.Success(ctx, gin.H{"ticket": token, "expiresIn": 60, "path": "/api/v1/web/terminal/connect"})
}

type socketWriter struct {
	mu *sync.Mutex
	ws *websocket.Conn
}

func (w socketWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ws.SetWriteDeadline(time.Now().Add(15 * time.Second))
	e := w.ws.WriteMessage(websocket.BinaryMessage, p)
	return len(p), e
}
func (c *Controller) Terminal(ctx *gin.Context) {
	var t model.Ticket
	now := time.Now().UTC()
	if c.S.DB.Where("hash = ? AND consumed_at IS NULL AND expires_at > ?", ticketHash(ctx.Query("ticket")), now).First(&t).Error != nil {
		ctx.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "终端票据无效或已过期"})
		return
	}
	if ctx.GetHeader("Origin") != t.Origin {
		ctx.AbortWithStatusJSON(403, gin.H{"code": 403, "message": "终端页面来源不匹配"})
		return
	}
	if t.APIKeyID > 0 {
		ctx.Request = ctx.Request.WithContext(apikey.WithKey(ctx.Request.Context(), t.APIKeyID))
	}
	i, p, e := c.terminalAllowed(ctx, t.UserID, t.InstanceID)
	if e != nil {
		fail(ctx, e)
		return
	}
	claim := c.S.DB.Model(&t).Where("consumed_at IS NULL AND expires_at > ?", now).Update("consumed_at", now)
	if claim.Error != nil || claim.RowsAffected != 1 {
		ctx.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "终端票据已使用"})
		return
	}
	var profile cmdbmodel.CmdbAssetConnection
	if c.S.DB.First(&profile, p.ConnectionID).Error != nil {
		fail(ctx, service.ErrForbidden)
		return
	}
	client, e := cmdbservice.ResolveAssetSSH(c.S.DB, "host", p.HostID, p.ConnectionID)
	if e != nil {
		fail(ctx, e)
		return
	}
	defer client.Close()
	session, e := client.NewSession()
	if e != nil {
		fail(ctx, e)
		return
	}
	defer session.Close()
	if e = session.RequestPty("xterm-256color", 24, 100, ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}); e != nil {
		fail(ctx, e)
		return
	}
	stdin, e := session.StdinPipe()
	if e != nil {
		fail(ctx, e)
		return
	}
	up := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return r.Header.Get("Origin") == t.Origin }}
	ws, e := up.Upgrade(ctx.Writer, ctx.Request, nil)
	if e != nil {
		return
	}
	defer ws.Close()
	ws.SetReadLimit(65536)
	ws.SetReadDeadline(time.Now().Add(75 * time.Second))
	ws.SetPongHandler(func(string) error { return ws.SetReadDeadline(time.Now().Add(75 * time.Second)) })
	mu := &sync.Mutex{}
	w := socketWriter{mu, ws}
	session.Stdout = w
	session.Stderr = w
	event := model.Event{UserID: t.UserID, ClusterID: i.ClusterID, SourceID: i.SourceID, Kind: "terminal-start", Message: fmt.Sprintf("instance=%d connection=%d runtime=%s client=%s", i.ID, p.ConnectionID, p.Runtime, ctx.ClientIP())}
	if c.S.DB.Create(&event).Error != nil {
		return
	}
	var exitStatus atomic.Value
	exitStatus.Store("connection-closed")
	defer func() {
		c.S.DB.Create(&model.Event{UserID: t.UserID, ClusterID: i.ClusterID, SourceID: i.SourceID, Kind: "terminal-end", Message: fmt.Sprintf("session=%d result=%s", event.ID, exitStatus.Load())})
	}()
	if p.Runtime == "docker" {
		var obs service.Observation
		json.Unmarshal(i.Observed, &obs)
		if len(obs.Identity) != 64 || strings.Trim(obs.Identity, "0123456789abcdef") != "" {
			return
		}
		command := "docker exec -it " + obs.Identity + " /bin/sh"
		if p.Sudo {
			command = "sudo -n " + command
		}
		e = session.Start(command)
	} else {
		e = session.Shell()
	}
	if e != nil {
		w.Write([]byte("Terminal failed\r\n"))
		return
	}
	done := make(chan struct{})
	defer close(done)
	go func() {
		err := session.Wait()
		status := "exited"
		if err != nil {
			status = err.Error()
		}
		exitStatus.Store(status)
		ws.Close()
	}()
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				_, currentSpec, e := c.terminalAllowed(ctx, t.UserID, t.InstanceID)
				var current cmdbmodel.CmdbAssetConnection
				changed := c.S.DB.First(&current, profile.ID).Error != nil || !current.UpdatedAt.Equal(profile.UpdatedAt)
				if e != nil || changed || string(service.JSON(currentSpec)) != string(service.JSON(p)) || time.Since(now) > 4*time.Hour {
					ws.Close()
					client.Close()
					return
				}
				mu.Lock()
				e = ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
				mu.Unlock()
				if e != nil {
					client.Close()
					return
				}
			}
		}
	}()
	for {
		_, b, e := ws.ReadMessage()
		if e != nil {
			return
		}
		var msg struct {
			Type string `json:"type"`
			Data string `json:"data"`
			Rows int    `json:"rows"`
			Cols int    `json:"cols"`
		}
		if json.Unmarshal(b, &msg) != nil {
			return
		}
		if msg.Type == "resize" && msg.Rows > 0 && msg.Rows < 500 && msg.Cols > 0 && msg.Cols < 1000 {
			session.WindowChange(msg.Rows, msg.Cols)
		} else if msg.Type == "input" {
			if _, e = io.WriteString(stdin, msg.Data); e != nil {
				return
			}
		}
	}
}
