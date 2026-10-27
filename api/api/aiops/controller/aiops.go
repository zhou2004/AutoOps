package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	model "dodevops-api/api/aiops/model"
	"dodevops-api/api/aiops/service"
	"dodevops-api/common/result"
	"dodevops-api/middleware"
	"dodevops-api/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type Controller struct{ Service *service.Service }

func (c *Controller) identity(g *gin.Context) (uint, bool) {
	id, err := jwt.GetAdminId(g)
	if err != nil {
		result.Failed(g, 401, "Please sign in / 请先登录")
		return 0, false
	}
	if err = c.Service.Active(g.Request.Context(), id); err != nil {
		result.Failed(g, 403, err.Error())
		return 0, false
	}
	return id, true
}
func (c *Controller) admin(g *gin.Context) bool {
	id, ok := c.identity(g)
	if !ok {
		return false
	}
	admin, err := c.Service.IsAdmin(g.Request.Context(), id)
	if err != nil || !admin {
		result.Failed(g, 403, "Administrator permission required / 需要管理员权限")
		return false
	}
	return true
}
func parseID(g *gin.Context) (uint, bool) {
	n, err := strconv.ParseUint(g.Param("id"), 10, 32)
	if err != nil || n == 0 {
		result.Failed(g, 400, "Invalid ID / ID 不合法")
		return 0, false
	}
	return uint(n), true
}
func (c *Controller) Catalog(g *gin.Context) {
	id, ok := c.identity(g)
	if !ok {
		return
	}
	admin, err := c.Service.IsAdmin(g.Request.Context(), id)
	if err != nil {
		result.Failed(g, 500, "Cannot read permissions / 读取权限失败")
		return
	}
	configs, err := c.Service.List(g.Request.Context(), admin)
	if err != nil {
		result.Failed(g, 500, "Cannot load AI configuration / 读取 AI 配置失败")
		return
	}
	result.Success(g, gin.H{"admin": admin, "modules": service.Modules, "configs": configs, "harnessAvailable": service.HarnessAvailable()})
}
func (c *Controller) Save(g *gin.Context) {
	if !c.admin(g) {
		return
	}
	var id uint
	if g.Param("id") != "" {
		var ok bool
		id, ok = parseID(g)
		if !ok {
			return
		}
	}
	g.Request.Body = http.MaxBytesReader(g.Writer, g.Request.Body, 256<<10)
	var d model.ConfigDTO
	if err := g.ShouldBindJSON(&d); err != nil {
		result.Failed(g, 400, "Invalid configuration / 配置格式错误")
		return
	}
	out, err := c.Service.Save(g.Request.Context(), id, d)
	if err != nil {
		result.Failed(g, 400, err.Error())
		return
	}
	result.Success(g, out)
}
func (c *Controller) Delete(g *gin.Context) {
	if !c.admin(g) {
		return
	}
	id, ok := parseID(g)
	if !ok {
		return
	}
	if err := c.Service.Delete(g.Request.Context(), id); err != nil {
		result.Failed(g, 400, err.Error())
		return
	}
	result.Success(g, nil)
}
func (c *Controller) Test(g *gin.Context) {
	if !c.admin(g) {
		return
	}
	id, ok := parseID(g)
	if !ok {
		return
	}
	if err := c.Service.TestModel(g.Request.Context(), id); err != nil {
		result.Failed(g, 400, err.Error())
		return
	}
	result.Success(g, gin.H{"ok": true})
}
func (c *Controller) History(g *gin.Context) {
	id, ok := c.identity(g)
	if !ok {
		return
	}
	page, err := strconv.Atoi(g.DefaultQuery("page", "1"))
	if err != nil || page < 1 || page > 10000 {
		result.Failed(g, 400, "Invalid page / 页码不合法")
		return
	}
	rows, total, err := c.Service.History(g.Request.Context(), id, (page-1)*20)
	if err != nil {
		result.Failed(g, 500, "Cannot read history / 读取历史失败")
		return
	}
	result.Success(g, gin.H{"items": rows, "total": total})
}
func (c *Controller) Detail(g *gin.Context) {
	user, ok := c.identity(g)
	if !ok {
		return
	}
	id, ok := parseID(g)
	if !ok {
		return
	}
	run, err := c.Service.Run(g.Request.Context(), user, id)
	if err != nil {
		result.Failed(g, 404, "Record not found / 记录不存在")
		return
	}
	result.Success(g, run)
}
func (c *Controller) Stream(g *gin.Context) {
	user, ok := c.identity(g)
	if !ok {
		return
	}
	g.Request.Body = http.MaxBytesReader(g.Writer, g.Request.Body, 64<<10)
	var request model.DiagnoseRequest
	if err := g.ShouldBindJSON(&request); err != nil {
		result.Failed(g, 400, "Invalid request / 请求格式错误")
		return
	}
	clusters := middleware.GetUserAllowedClusterIDsFromContext(g)
	c.stream(g, func(ctx context.Context, emit func(model.Event) error) error {
		return c.Service.Diagnose(ctx, user, clusters, request, emit)
	})
}
func (c *Controller) stream(g *gin.Context, run func(context.Context, func(model.Event) error) error) {
	ctx, cancel := context.WithCancel(g.Request.Context())
	defer cancel()
	events := make(chan model.Event, 16)
	complete := make(chan error, 1)
	go func() {
		complete <- run(ctx, func(e model.Event) error {
			select {
			case events <- e:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	started := false
	write := func(event model.Event) error {
		if !started {
			g.Header("Content-Type", "text/event-stream; charset=utf-8")
			g.Header("Cache-Control", "no-cache, no-transform")
			g.Header("X-Accel-Buffering", "no")
			g.Status(200)
			started = true
		}
		_ = http.NewResponseController(g.Writer).SetWriteDeadline(time.Now().Add(15 * time.Second))
		payload, _ := json.Marshal(event)
		if _, err := fmt.Fprintf(g.Writer, "data: %s\n\n", payload); err != nil {
			return err
		}
		g.Writer.Flush()
		return nil
	}
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-events:
			if write(event) != nil {
				return
			}
		case err := <-complete:
			// Drain buffered events before closing, including the terminal done event.
			for len(events) > 0 {
				if write(<-events) != nil {
					return
				}
			}
			if err != nil {
				if !started {
					result.Failed(g, 400, err.Error())
				} else {
					_ = write(model.Event{Type: "error", Text: err.Error(), Status: "failed"})
				}
			}
			return
		case <-ticker.C:
			if started {
				if write(model.Event{Type: "heartbeat"}) != nil {
					return
				}
			}
		}
	}
}
