package controller

import (
	"net/http"
	"strconv"

	"dodevops-api/api/cmdb/model"
	"dodevops-api/api/cmdb/service"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CmdbIDCController struct {
	svc service.ICmdbIDCService
}

func NewCmdbIDCController(db *gorm.DB) *CmdbIDCController {
	return &CmdbIDCController{svc: service.NewCmdbIDCService(db)}
}

// Create 创建机房
func (ctrl *CmdbIDCController) Create(c *gin.Context) {
	var req service.CmdbIDCReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ctrl.svc.Create(c, &req)
}

// Update 更新机房
func (ctrl *CmdbIDCController) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	var req service.CmdbIDCReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ctrl.svc.Update(c, uint(id), &req)
}

// Delete 删除机房
func (ctrl *CmdbIDCController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	ctrl.svc.Delete(c, uint(id))
}

// GetByID 获取机房详情
func (ctrl *CmdbIDCController) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	ctrl.svc.GetByID(c, uint(id))
}

// GetList 分页查询机房
func (ctrl *CmdbIDCController) GetList(c *gin.Context) {
	var query model.CmdbIDCQuery
	query.Name = c.Query("name")
	query.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	query.Size, _ = strconv.Atoi(c.DefaultQuery("size", "10"))
	ctrl.svc.GetList(c, query)
}

// GetAll 获取所有机房
func (ctrl *CmdbIDCController) GetAll(c *gin.Context) {
	ctrl.svc.GetAll(c)
}

// ======================= Cabinet =======================

type CmdbCabinetController struct {
	svc service.ICmdbCabinetService
}

func NewCmdbCabinetController(db *gorm.DB) *CmdbCabinetController {
	return &CmdbCabinetController{svc: service.NewCmdbCabinetService(db)}
}

func (ctrl *CmdbCabinetController) Create(c *gin.Context) {
	var req service.CmdbCabinetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ctrl.svc.Create(c, &req)
}

func (ctrl *CmdbCabinetController) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	var req service.CmdbCabinetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ctrl.svc.Update(c, uint(id), &req)
}

func (ctrl *CmdbCabinetController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	ctrl.svc.Delete(c, uint(id))
}

func (ctrl *CmdbCabinetController) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	ctrl.svc.GetByID(c, uint(id))
}

func (ctrl *CmdbCabinetController) GetList(c *gin.Context) {
	var query model.CmdbCabinetQuery
	query.Name = c.Query("name")
	if v := c.Query("idcId"); v != "" {
		id, _ := strconv.ParseUint(v, 10, 64)
		query.IDCID = uint(id)
	}
	query.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	query.Size, _ = strconv.Atoi(c.DefaultQuery("size", "10"))
	ctrl.svc.GetList(c, query)
}

func (ctrl *CmdbCabinetController) GetByIDC(c *gin.Context) {
	idcIDStr := c.Param("idcId")
	idcID, err := strconv.ParseUint(idcIDStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的IDC ID")
		return
	}
	ctrl.svc.GetByIDC(c, uint(idcID))
}

// ======================= Physical Machine =======================

type CmdbPhysicalMachineController struct {
	svc service.ICmdbPhysicalMachineService
}

func NewCmdbPhysicalMachineController(db *gorm.DB) *CmdbPhysicalMachineController {
	return &CmdbPhysicalMachineController{svc: service.NewCmdbPhysicalMachineService(db)}
}

func (ctrl *CmdbPhysicalMachineController) Create(c *gin.Context) {
	var req service.CmdbPhysicalMachineReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ctrl.svc.Create(c, &req)
}

func (ctrl *CmdbPhysicalMachineController) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	var req service.CmdbPhysicalMachineReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ctrl.svc.Update(c, uint(id), &req)
}

func (ctrl *CmdbPhysicalMachineController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	ctrl.svc.Delete(c, uint(id))
}

func (ctrl *CmdbPhysicalMachineController) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	ctrl.svc.GetByID(c, uint(id))
}

func (ctrl *CmdbPhysicalMachineController) GetList(c *gin.Context) {
	var query model.CmdbPhysicalMachineQuery
	query.SN = c.Query("sn")
	query.HostName = c.Query("hostName")
	query.ManageIP = c.Query("manageIp")
	query.Keyword = c.Query("keyword")
	query.Brand = c.Query("brand")
	if v := c.Query("idcId"); v != "" {
		id, _ := strconv.ParseUint(v, 10, 64)
		query.IDCID = uint(id)
	}
	if v := c.Query("cabinetId"); v != "" {
		id, _ := strconv.ParseUint(v, 10, 64)
		query.CabinetID = uint(id)
	}
	if v := c.Query("assetStatus"); v != "" {
		query.AssetStatus, _ = strconv.Atoi(v)
	}
	query.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	query.Size, _ = strconv.Atoi(c.DefaultQuery("size", "10"))
	ctrl.svc.GetList(c, query)
}

func (ctrl *CmdbPhysicalMachineController) GetAll(c *gin.Context) {
	ctrl.svc.GetAll(c)
}

func (ctrl *CmdbPhysicalMachineController) GetStats(c *gin.Context) {
	ctrl.svc.GetStats(c)
}

// ======================= Network Device =======================

type CmdbNetworkDeviceController struct {
	svc service.ICmdbNetworkDeviceService
}

func NewCmdbNetworkDeviceController(db *gorm.DB) *CmdbNetworkDeviceController {
	return &CmdbNetworkDeviceController{svc: service.NewCmdbNetworkDeviceService(db)}
}

func (ctrl *CmdbNetworkDeviceController) Create(c *gin.Context) {
	var req service.CmdbNetworkDeviceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ctrl.svc.Create(c, &req)
}

func (ctrl *CmdbNetworkDeviceController) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	var req service.CmdbNetworkDeviceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ctrl.svc.Update(c, uint(id), &req)
}

func (ctrl *CmdbNetworkDeviceController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	ctrl.svc.Delete(c, uint(id))
}

func (ctrl *CmdbNetworkDeviceController) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	ctrl.svc.GetByID(c, uint(id))
}

func (ctrl *CmdbNetworkDeviceController) GetList(c *gin.Context) {
	var query model.CmdbNetworkDeviceQuery
	query.Name = c.Query("name")
	query.ManageIP = c.Query("manageIp")
	query.Keyword = c.Query("keyword")
	query.Brand = c.Query("brand")
	if v := c.Query("deviceType"); v != "" {
		query.DeviceType, _ = strconv.Atoi(v)
	}
	if v := c.Query("idcId"); v != "" {
		id, _ := strconv.ParseUint(v, 10, 64)
		query.IDCID = uint(id)
	}
	query.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	query.Size, _ = strconv.Atoi(c.DefaultQuery("size", "10"))
	ctrl.svc.GetList(c, query)
}

func (ctrl *CmdbNetworkDeviceController) GetAll(c *gin.Context) {
	ctrl.svc.GetAll(c)
}

// ======================= Asset Permission =======================

type CmdbAssetPermissionController struct {
	svc service.ICmdbAssetPermissionService
}

func NewCmdbAssetPermissionController(db *gorm.DB) *CmdbAssetPermissionController {
	return &CmdbAssetPermissionController{svc: service.NewCmdbAssetPermissionService(db)}
}

func (ctrl *CmdbAssetPermissionController) Create(c *gin.Context) {
	var req service.CmdbAssetPermissionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ctrl.svc.Create(c, &req)
}

func (ctrl *CmdbAssetPermissionController) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	var req service.CmdbAssetPermissionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	ctrl.svc.Update(c, uint(id), &req)
}

func (ctrl *CmdbAssetPermissionController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	ctrl.svc.Delete(c, uint(id))
}

func (ctrl *CmdbAssetPermissionController) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	ctrl.svc.GetByID(c, uint(id))
}

func (ctrl *CmdbAssetPermissionController) GetList(c *gin.Context) {
	var query model.CmdbAssetPermissionQuery
	query.Name = c.Query("name")
	query.AssetTypes = c.Query("assetTypes")
	query.SubjectType = c.Query("subjectType")
	if v := c.Query("subjectId"); v != "" {
		id, _ := strconv.ParseUint(v, 10, 64)
		query.SubjectID = uint(id)
	}
	if v := c.Query("isActive"); v != "" {
		query.IsActive, _ = strconv.Atoi(v)
	}
	query.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	query.Size, _ = strconv.Atoi(c.DefaultQuery("size", "10"))
	ctrl.svc.GetList(c, query)
}

// GetMyAssets 获取当前用户的授权资产
func (ctrl *CmdbAssetPermissionController) GetMyAssets(c *gin.Context) {
	ctrl.svc.GetMyAssets(c)
}

// CheckPermission 检查当前用户是否有指定资产的权限
func (ctrl *CmdbAssetPermissionController) CheckPermission(c *gin.Context) {
	assetType := c.Param("assetType")
	assetIDStr := c.Param("assetId")
	assetID, err := strconv.ParseUint(assetIDStr, 10, 64)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的资产ID")
		return
	}
	hasPerm := ctrl.svc.CheckUserAssetPermission(c, assetType, uint(assetID))
	result.Success(c, gin.H{"hasPermission": hasPerm})
}
