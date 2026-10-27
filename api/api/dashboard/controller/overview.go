package controller

import (
	"net/http"
	"strconv"

	"dodevops-api/common/result"
	"dodevops-api/middleware"
	"dodevops-api/pkg/jwt"
	"dodevops-api/pkg/log"
	"github.com/gin-gonic/gin"
)

// GetOverview returns current inventory plus business activity over 7, 30 or 90 days.
func (ctrl *DashboardController) GetOverview(c *gin.Context) {
	userID, err := jwt.GetAdminId(c)
	if err != nil {
		result.Failed(c, http.StatusUnauthorized, "请先登录")
		return
	}
	days, err := strconv.Atoi(c.DefaultQuery("days", "7"))
	if err != nil || (days != 7 && days != 30 && days != 90) {
		result.Failed(c, http.StatusBadRequest, "days 仅支持 7、30、90")
		return
	}
	data, err := ctrl.service.GetOverview(c.Request.Context(), userID,
		middleware.GetUserAllowedClusterIDsFromContext(c), days)
	if err != nil {
		log.Log().Errorf("读取仪表盘失败: %v", err)
		result.Failed(c, http.StatusInternalServerError, "仪表盘读取失败，请稍后重试")
		return
	}
	result.Success(c, data)
}
