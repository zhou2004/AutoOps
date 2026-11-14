package controller

import (
	"dodevops-api/api/database/service"
	"dodevops-api/common/result"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

func (ctrl *Controller) DatabaseLogs(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	rows, err := ctrl.s.DatabaseLogs(c.Request.Context(), u, id(c), queryID(c, "memberId"))
	if err != nil {
		fail(c, err)
		return
	}
	result.Success(c, rows)
}
func (ctrl *Controller) DatabaseLogStream(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	member, err := ctrl.s.LogMember(c.Request.Context(), u, id(c), queryID(c, "memberId"))
	if err != nil {
		fail(c, err)
		return
	}
	lines, _ := strconv.Atoi(c.DefaultQuery("lines", "500"))
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-store")
	c.Header("X-Accel-Buffering", "no")
	ended := false
	send := func(event service.DatabaseLogEvent) error {
		if c.Request.Context().Err() != nil {
			return c.Request.Context().Err()
		}
		_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(15 * time.Second))
		if event.Event == "end" || event.Event == "error" {
			ended = true
		}
		_, e := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event.Event, event.Data)
		if e == nil {
			c.Writer.Flush()
		}
		return e
	}
	err = ctrl.s.FollowDatabaseLogs(c.Request.Context(), u, member, c.Query("logId"), lines, c.DefaultQuery("follow", "true") != "false", send)
	if !c.Writer.Written() && err != nil {
		c.Header("Content-Type", "application/json")
		fail(c, err)
		return
	}
	if !ended && c.Request.Context().Err() == nil {
		event := service.DatabaseLogEvent{Event: "end", Data: service.JSON(map[string]string{"message": "日志连接已结束，可重新连接继续追踪"})}
		if err != nil {
			event.Event = "error"
			event.Data = service.JSON(map[string]string{"message": err.Error()})
		}
		_ = send(event)
	}
}
