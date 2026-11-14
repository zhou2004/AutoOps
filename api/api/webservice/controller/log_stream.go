package controller

import (
	"dodevops-api/api/webservice/service"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

func (c *Controller) LogStream(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	lines, _ := strconv.Atoi(ctx.DefaultQuery("lines", "500"))
	target, err := c.S.LogTarget(ctx.Request.Context(), u, id(ctx), service.RemoteRequest{LogID: ctx.Query("logId"), Lines: lines, Keyword: ctx.Query("keyword"), Status: ctx.Query("status"), IP: ctx.Query("ip"), URI: ctx.Query("uri"), Rotation: ctx.Query("rotation"), From: ctx.Query("from"), To: ctx.Query("to")})
	if err != nil {
		fail(ctx, err)
		return
	}
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache, no-store")
	ctx.Header("X-Accel-Buffering", "no")
	var seq uint64
	ended := false
	send := func(event service.LogStreamEvent) error {
		if ctx.Request.Context().Err() != nil {
			return ctx.Request.Context().Err()
		}
		// Bound a slow consumer; closing the browser cancels the remote process.
		_ = http.NewResponseController(ctx.Writer).SetWriteDeadline(time.Now().Add(15 * time.Second))
		seq++
		if event.Event == "end" || event.Event == "error" {
			ended = true
		}
		_, e := fmt.Fprintf(ctx.Writer, "id: %d\nevent: %s\ndata: %s\n\n", seq, event.Event, event.Data)
		if e == nil {
			ctx.Writer.Flush()
		}
		return e
	}
	err = c.S.FollowLogs(ctx.Request.Context(), u, target, send)
	if !ctx.Writer.Written() && err != nil {
		ctx.Header("Content-Type", "application/json")
		fail(ctx, err)
		return
	}
	if !ended && ctx.Request.Context().Err() == nil {
		event := service.LogStreamEvent{Event: "end", Data: service.JSON(map[string]string{"message": "日志连接已结束，可重新连接继续追踪"})}
		if err != nil {
			event.Event = "error"
			event.Data = service.JSON(map[string]string{"message": err.Error()})
		}
		_ = send(event)
	}
}
