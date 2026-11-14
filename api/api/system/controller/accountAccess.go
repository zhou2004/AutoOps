package controller

import (
	"dodevops-api/api/system/service"
	"dodevops-api/common"
	"dodevops-api/common/result"
	"dodevops-api/common/util"
	"github.com/gin-gonic/gin"
)

func AccountOptions(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	result.Success(c, gin.H{"registrationEnabled": service.RegistrationEnabled(), "emailEnabled": service.AccountMailer(common.GetDB()).Ready(), "approvalRequired": true})
}
func SendAccountCode(c *gin.Context) {
	var req struct {
		Email   string `json:"email" binding:"required,max=254"`
		Purpose string `json:"purpose" binding:"required,oneof=register reset"`
		IdKey   string `json:"idKey" binding:"required,max=128"`
		Image   string `json:"image" binding:"required,len=4"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, 400, "请检查邮箱和验证码")
		return
	}
	if !(util.RedisStore{}).Verify(req.IdKey, req.Image, true) {
		result.Failed(c, 400, "图形验证码错误或已过期")
		return
	}
	svc := service.NewAccountAccessService(common.GetDB())
	if !service.AccountMailer(common.GetDB()).Ready() {
		result.Failed(c, 503, "邮箱验证尚未配置，请联系管理员")
		return
	}
	id, err := svc.SendCode(c.Request.Context(), req.Email, req.Purpose, c.ClientIP())
	if err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	c.Header("Cache-Control", "no-store")
	result.Success(c, gin.H{"verificationId": id, "expiresIn": 600, "retryAfter": 60, "message": "如邮箱符合条件，验证码已发送，请检查收件箱"})
}
func CompleteAccount(c *gin.Context) {
	var req struct {
		Email          string `json:"email" binding:"required,max=254"`
		VerificationID string `json:"verificationId" binding:"required,len=64"`
		Code           string `json:"code" binding:"required,len=6"`
		Username       string `json:"username" binding:"max=32"`
		Password       string `json:"password" binding:"required,min=10,max=72"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, 400, "请检查表单，密码长度需为 10–72 字节")
		return
	}
	purpose := "reset"
	if c.FullPath() == "/api/v1/auth/register" {
		purpose = "register"
	}
	if err := service.NewAccountAccessService(common.GetDB()).Complete(c.Request.Context(), req.VerificationID, req.Email, purpose, req.Code, req.Username, req.Password); err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	result.Success(c, gin.H{"approvalRequired": purpose == "register"})
}
