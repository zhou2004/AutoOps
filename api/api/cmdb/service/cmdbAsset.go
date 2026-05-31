package service

import (
	"encoding/json"
	"net/http"

	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/model"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ICmdbIDCService 机房服务接口
type ICmdbIDCService interface {
	Create(c *gin.Context, req *CmdbIDCReq)
	Update(c *gin.Context, id uint, req *CmdbIDCReq)
	Delete(c *gin.Context, id uint)
	GetByID(c *gin.Context, id uint)
	GetList(c *gin.Context, query model.CmdbIDCQuery)
	GetAll(c *gin.Context)
}

type CmdbIDCReq struct {
	Name        string `json:"name" binding:"required"`
	ShortName   string `json:"shortName"`
	Address     string `json:"address"`
	Contact     string `json:"contact"`
	Phone       string `json:"phone"`
	Level       string `json:"level"`
	Status      int    `json:"status"`
	Description string `json:"description"`
}

type CmdbIDCServiceImpl struct {
	dao *dao.CmdbIDCDao
}

func NewCmdbIDCService(db *gorm.DB) ICmdbIDCService {
	return &CmdbIDCServiceImpl{dao: dao.NewCmdbIDCDao(db)}
}

func (s *CmdbIDCServiceImpl) Create(c *gin.Context, req *CmdbIDCReq) {
	data := &model.CmdbIDC{
		Name: req.Name, ShortName: req.ShortName, Address: req.Address,
		Contact: req.Contact, Phone: req.Phone, Level: req.Level,
		Status: req.Status, Description: req.Description,
	}
	if data.Status == 0 {
		data.Status = 1
	}
	if err := s.dao.Create(data); err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建机房失败: "+err.Error())
		return
	}
	result.Success(c, data)
}

func (s *CmdbIDCServiceImpl) Update(c *gin.Context, id uint, req *CmdbIDCReq) {
	data, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "机房不存在")
		return
	}
	data.Name = req.Name
	data.ShortName = req.ShortName
	data.Address = req.Address
	data.Contact = req.Contact
	data.Phone = req.Phone
	data.Level = req.Level
	data.Status = req.Status
	data.Description = req.Description
	if data.Status == 0 {
		data.Status = 1
	}
	if err := s.dao.Update(data); err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新机房失败: "+err.Error())
		return
	}
	result.Success(c, data)
}

func (s *CmdbIDCServiceImpl) Delete(c *gin.Context, id uint) {
	if err := s.dao.Delete(id); err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除机房失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

func (s *CmdbIDCServiceImpl) GetByID(c *gin.Context, id uint) {
	data, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "机房不存在")
		return
	}
	result.Success(c, data)
}

func (s *CmdbIDCServiceImpl) GetList(c *gin.Context, query model.CmdbIDCQuery) {
	list, total, err := s.dao.GetList(query)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询机房失败: "+err.Error())
		return
	}
	result.Success(c, gin.H{"list": list, "total": total})
}

func (s *CmdbIDCServiceImpl) GetAll(c *gin.Context) {
	list, err := s.dao.GetAll()
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询机房失败: "+err.Error())
		return
	}
	result.Success(c, gin.H{"list": list, "total": len(list)})
}

// ======================= Cabinet =======================

type ICmdbCabinetService interface {
	Create(c *gin.Context, req *CmdbCabinetReq)
	Update(c *gin.Context, id uint, req *CmdbCabinetReq)
	Delete(c *gin.Context, id uint)
	GetByID(c *gin.Context, id uint)
	GetList(c *gin.Context, query model.CmdbCabinetQuery)
	GetByIDC(c *gin.Context, idcID uint)
}

type CmdbCabinetReq struct {
	Name     string  `json:"name" binding:"required"`
	IDCID    uint    `json:"idcId" binding:"required"`
	Position string  `json:"position"`
	UnitNum  int     `json:"unitNum"`
	UsedUnit int     `json:"usedUnit"`
	PowerKW  float64 `json:"powerKw"`
	Status   int     `json:"status"`
	Remark   string  `json:"remark"`
}

type CmdbCabinetServiceImpl struct {
	dao *dao.CmdbCabinetDao
}

func NewCmdbCabinetService(db *gorm.DB) ICmdbCabinetService {
	return &CmdbCabinetServiceImpl{dao: dao.NewCmdbCabinetDao(db)}
}

func (s *CmdbCabinetServiceImpl) Create(c *gin.Context, req *CmdbCabinetReq) {
	data := &model.CmdbCabinet{
		Name: req.Name, IDCID: req.IDCID, Position: req.Position,
		UnitNum: req.UnitNum, UsedUnit: req.UsedUnit, PowerKW: req.PowerKW,
		Status: req.Status, Remark: req.Remark,
	}
	if data.Status == 0 {
		data.Status = 1
	}
	if err := s.dao.Create(data); err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建机柜失败: "+err.Error())
		return
	}
	result.Success(c, data)
}

func (s *CmdbCabinetServiceImpl) Update(c *gin.Context, id uint, req *CmdbCabinetReq) {
	data, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "机柜不存在")
		return
	}
	data.Name = req.Name
	data.IDCID = req.IDCID
	data.Position = req.Position
	data.UnitNum = req.UnitNum
	data.UsedUnit = req.UsedUnit
	data.PowerKW = req.PowerKW
	data.Status = req.Status
	data.Remark = req.Remark
	if data.Status == 0 {
		data.Status = 1
	}
	if err := s.dao.Update(data); err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新机柜失败: "+err.Error())
		return
	}
	result.Success(c, data)
}

func (s *CmdbCabinetServiceImpl) Delete(c *gin.Context, id uint) {
	if err := s.dao.Delete(id); err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除机柜失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

func (s *CmdbCabinetServiceImpl) GetByID(c *gin.Context, id uint) {
	data, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "机柜不存在")
		return
	}
	result.Success(c, data)
}

func (s *CmdbCabinetServiceImpl) GetList(c *gin.Context, query model.CmdbCabinetQuery) {
	list, total, err := s.dao.GetList(query)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询机柜失败: "+err.Error())
		return
	}
	result.Success(c, gin.H{"list": list, "total": total})
}

func (s *CmdbCabinetServiceImpl) GetByIDC(c *gin.Context, idcID uint) {
	list, err := s.dao.GetByIDC(idcID)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询机柜失败: "+err.Error())
		return
	}
	result.Success(c, gin.H{"list": list})
}

// ======================= Physical Machine =======================

type ICmdbPhysicalMachineService interface {
	Create(c *gin.Context, req *CmdbPhysicalMachineReq)
	Update(c *gin.Context, id uint, req *CmdbPhysicalMachineReq)
	Delete(c *gin.Context, id uint)
	GetByID(c *gin.Context, id uint)
	GetList(c *gin.Context, query model.CmdbPhysicalMachineQuery)
	GetAll(c *gin.Context)
	GetStats(c *gin.Context)
}

type CmdbPhysicalMachineReq struct {
	SN           string `json:"sn" binding:"required"`
	HostName     string `json:"hostName"`
	ManageIP     string `json:"manageIp"`
	BusinessIP   string `json:"businessIp"`
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	CPU          string `json:"cpu"`
	Memory       string `json:"memory"`
	Disk         string `json:"disk"`
	Raid         string `json:"raid"`
	IDCID        uint   `json:"idcId"`
	CabinetID    uint   `json:"cabinetId"`
	UnitPosition int    `json:"unitPosition"`
	AssetStatus  int    `json:"assetStatus"`
	PurchaseDate string `json:"purchaseDate"`
	WarrantyDate string `json:"warrantyDate"`
	Vendor       string `json:"vendor"`
	Status       int    `json:"status"`
	Remark       string `json:"remark"`
}

type CmdbPhysicalMachineServiceImpl struct {
	dao *dao.CmdbPhysicalMachineDao
}

func NewCmdbPhysicalMachineService(db *gorm.DB) ICmdbPhysicalMachineService {
	return &CmdbPhysicalMachineServiceImpl{dao: dao.NewCmdbPhysicalMachineDao(db)}
}

func (s *CmdbPhysicalMachineServiceImpl) Create(c *gin.Context, req *CmdbPhysicalMachineReq) {
	if s.dao.CheckSNExists(req.SN, 0) {
		result.Failed(c, http.StatusBadRequest, "SN已存在")
		return
	}
	data := &model.CmdbPhysicalMachine{
		SN: req.SN, HostName: req.HostName, ManageIP: req.ManageIP,
		BusinessIP: req.BusinessIP, Brand: req.Brand, Model: req.Model,
		CPU: req.CPU, Memory: req.Memory, Disk: req.Disk, Raid: req.Raid,
		IDCID: req.IDCID, CabinetID: req.CabinetID, UnitPosition: req.UnitPosition,
		AssetStatus: req.AssetStatus, PurchaseDate: req.PurchaseDate,
		WarrantyDate: req.WarrantyDate, Vendor: req.Vendor,
		Status: req.Status, Remark: req.Remark,
	}
	if data.Status == 0 {
		data.Status = 1
	}
	if err := s.dao.Create(data); err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建物理机失败: "+err.Error())
		return
	}
	result.Success(c, data)
}

func (s *CmdbPhysicalMachineServiceImpl) Update(c *gin.Context, id uint, req *CmdbPhysicalMachineReq) {
	if s.dao.CheckSNExists(req.SN, id) {
		result.Failed(c, http.StatusBadRequest, "SN已存在")
		return
	}
	data, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "物理机不存在")
		return
	}
	data.SN = req.SN
	data.HostName = req.HostName
	data.ManageIP = req.ManageIP
	data.BusinessIP = req.BusinessIP
	data.Brand = req.Brand
	data.Model = req.Model
	data.CPU = req.CPU
	data.Memory = req.Memory
	data.Disk = req.Disk
	data.Raid = req.Raid
	data.IDCID = req.IDCID
	data.CabinetID = req.CabinetID
	data.UnitPosition = req.UnitPosition
	data.AssetStatus = req.AssetStatus
	data.PurchaseDate = req.PurchaseDate
	data.WarrantyDate = req.WarrantyDate
	data.Vendor = req.Vendor
	data.Status = req.Status
	data.Remark = req.Remark
	if err := s.dao.Update(data); err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新物理机失败: "+err.Error())
		return
	}
	result.Success(c, data)
}

func (s *CmdbPhysicalMachineServiceImpl) Delete(c *gin.Context, id uint) {
	if err := s.dao.Delete(id); err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除物理机失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

func (s *CmdbPhysicalMachineServiceImpl) GetByID(c *gin.Context, id uint) {
	data, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "物理机不存在")
		return
	}
	result.Success(c, data)
}

func (s *CmdbPhysicalMachineServiceImpl) GetList(c *gin.Context, query model.CmdbPhysicalMachineQuery) {
	list, total, err := s.dao.GetList(query)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询物理机失败: "+err.Error())
		return
	}
	// 按权限过滤
	allowedIDs, _ := c.Get("cmdb_allowed_asset_ids")
	if idSet, ok := allowedIDs.(map[uint]bool); ok && len(idSet) > 0 {
		var filtered []model.CmdbPhysicalMachine
		for _, item := range list {
			if idSet[item.ID] {
				filtered = append(filtered, item)
			}
		}
		list = filtered
		total = int64(len(filtered))
	}
	result.Success(c, gin.H{"list": list, "total": total})
}

func (s *CmdbPhysicalMachineServiceImpl) GetAll(c *gin.Context) {
	list, err := s.dao.GetAll()
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询物理机失败: "+err.Error())
		return
	}
	// 按权限过滤
	allowedIDs, _ := c.Get("cmdb_allowed_asset_ids")
	if idSet, ok := allowedIDs.(map[uint]bool); ok && len(idSet) > 0 {
		var filtered []model.CmdbPhysicalMachine
		for _, item := range list {
			if idSet[item.ID] {
				filtered = append(filtered, item)
			}
		}
		list = filtered
	}
	result.Success(c, gin.H{"list": list, "total": len(list)})
}

func (s *CmdbPhysicalMachineServiceImpl) GetStats(c *gin.Context) {
	stats := s.dao.GetStats()
	result.Success(c, stats)
}

// ======================= Network Device =======================

type ICmdbNetworkDeviceService interface {
	Create(c *gin.Context, req *CmdbNetworkDeviceReq)
	Update(c *gin.Context, id uint, req *CmdbNetworkDeviceReq)
	Delete(c *gin.Context, id uint)
	GetByID(c *gin.Context, id uint)
	GetList(c *gin.Context, query model.CmdbNetworkDeviceQuery)
	GetAll(c *gin.Context)
}

type CmdbNetworkDeviceReq struct {
	SN           string `json:"sn" binding:"required"`
	Name         string `json:"name" binding:"required"`
	DeviceType   int    `json:"deviceType" binding:"required"`
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	ManageIP     string `json:"manageIp"`
	Version      string `json:"version"`
	PortNum      int    `json:"portNum"`
	IDCID        uint   `json:"idcId"`
	CabinetID    uint   `json:"cabinetId"`
	AssetStatus  int    `json:"assetStatus"`
	PurchaseDate string `json:"purchaseDate"`
	WarrantyDate string `json:"warrantyDate"`
	Status       int    `json:"status"`
	Remark       string `json:"remark"`
}

type CmdbNetworkDeviceServiceImpl struct {
	dao *dao.CmdbNetworkDeviceDao
}

func NewCmdbNetworkDeviceService(db *gorm.DB) ICmdbNetworkDeviceService {
	return &CmdbNetworkDeviceServiceImpl{dao: dao.NewCmdbNetworkDeviceDao(db)}
}

func (s *CmdbNetworkDeviceServiceImpl) Create(c *gin.Context, req *CmdbNetworkDeviceReq) {
	data := &model.CmdbNetworkDevice{
		SN: req.SN, Name: req.Name, DeviceType: req.DeviceType,
		Brand: req.Brand, Model: req.Model, ManageIP: req.ManageIP,
		Version: req.Version, PortNum: req.PortNum,
		IDCID: req.IDCID, CabinetID: req.CabinetID,
		AssetStatus: req.AssetStatus, PurchaseDate: req.PurchaseDate,
		WarrantyDate: req.WarrantyDate, Status: req.Status, Remark: req.Remark,
	}
	if data.Status == 0 {
		data.Status = 1
	}
	if err := s.dao.Create(data); err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建网络设备失败: "+err.Error())
		return
	}
	result.Success(c, data)
}

func (s *CmdbNetworkDeviceServiceImpl) Update(c *gin.Context, id uint, req *CmdbNetworkDeviceReq) {
	data, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "网络设备不存在")
		return
	}
	data.SN = req.SN
	data.Name = req.Name
	data.DeviceType = req.DeviceType
	data.Brand = req.Brand
	data.Model = req.Model
	data.ManageIP = req.ManageIP
	data.Version = req.Version
	data.PortNum = req.PortNum
	data.IDCID = req.IDCID
	data.CabinetID = req.CabinetID
	data.AssetStatus = req.AssetStatus
	data.PurchaseDate = req.PurchaseDate
	data.WarrantyDate = req.WarrantyDate
	data.Status = req.Status
	data.Remark = req.Remark
	if err := s.dao.Update(data); err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新网络设备失败: "+err.Error())
		return
	}
	result.Success(c, data)
}

func (s *CmdbNetworkDeviceServiceImpl) Delete(c *gin.Context, id uint) {
	if err := s.dao.Delete(id); err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除网络设备失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

func (s *CmdbNetworkDeviceServiceImpl) GetByID(c *gin.Context, id uint) {
	data, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "网络设备不存在")
		return
	}
	result.Success(c, data)
}

func (s *CmdbNetworkDeviceServiceImpl) GetList(c *gin.Context, query model.CmdbNetworkDeviceQuery) {
	list, total, err := s.dao.GetList(query)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询网络设备失败: "+err.Error())
		return
	}
	// 按权限过滤
	allowedIDs, _ := c.Get("cmdb_allowed_asset_ids")
	if idSet, ok := allowedIDs.(map[uint]bool); ok && len(idSet) > 0 {
		var filtered []model.CmdbNetworkDevice
		for _, item := range list {
			if idSet[item.ID] {
				filtered = append(filtered, item)
			}
		}
		list = filtered
		total = int64(len(filtered))
	}
	result.Success(c, gin.H{"list": list, "total": total})
}

func (s *CmdbNetworkDeviceServiceImpl) GetAll(c *gin.Context) {
	list, err := s.dao.GetAll()
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询网络设备失败: "+err.Error())
		return
	}
	// 按权限过滤
	allowedIDs, _ := c.Get("cmdb_allowed_asset_ids")
	if idSet, ok := allowedIDs.(map[uint]bool); ok && len(idSet) > 0 {
		var filtered []model.CmdbNetworkDevice
		for _, item := range list {
			if idSet[item.ID] {
				filtered = append(filtered, item)
			}
		}
		list = filtered
	}
	result.Success(c, gin.H{"list": list, "total": len(list)})
}

// ======================= Asset Permission =======================

type ICmdbAssetPermissionService interface {
	Create(c *gin.Context, req *CmdbAssetPermissionReq)
	Update(c *gin.Context, id uint, req *CmdbAssetPermissionReq)
	Delete(c *gin.Context, id uint)
	GetByID(c *gin.Context, id uint)
	GetList(c *gin.Context, query model.CmdbAssetPermissionQuery)
	GetMyAssets(c *gin.Context)
	CheckUserAssetPermission(c *gin.Context, assetType string, assetID uint) bool
}

type CmdbAssetPermissionReq struct {
	Name              string   `json:"name" binding:"required"`
	Description       string   `json:"description"`
	UserIDs           []uint   `json:"userIds"`
	GroupIDs          []uint   `json:"groupIds"`
	AssetTypes        []string `json:"assetTypes"`
	HostGroupIDs      []uint   `json:"hostGroupIds"`
	HostIDs           []uint   `json:"hostIds"`
	PhysicalIDs       []uint   `json:"physicalIds"`
	NetworkIDs        []uint   `json:"networkIds"`
	DatabaseIDs       []uint   `json:"databaseIds"`
	IDCIDs            []uint   `json:"idcIds"`
	PermissionActions []string `json:"permissionActions"`
	IsActive          int      `json:"isActive"`
	DateStart         string   `json:"dateStart"`
	DateExpired       string   `json:"dateExpired"`
}

type CmdbAssetPermissionServiceImpl struct {
	dao              *dao.CmdbAssetPermissionDao
	groupDao         dao.CmdbGroupDao
	cmdbUserGroupDao *dao.CmdbUserGroupDao
	hostDao          dao.CmdbHostDao
	physicalDao      *dao.CmdbPhysicalMachineDao
	networkDao       *dao.CmdbNetworkDeviceDao
}

func NewCmdbAssetPermissionService(db *gorm.DB) ICmdbAssetPermissionService {
	return &CmdbAssetPermissionServiceImpl{
		dao:              dao.NewCmdbAssetPermissionDao(db),
		groupDao:         dao.NewCmdbGroupDao(),
		cmdbUserGroupDao: dao.NewCmdbUserGroupDao(),
		hostDao:          dao.NewCmdbHostDao(),
		physicalDao:      dao.NewCmdbPhysicalMachineDao(db),
		networkDao:       dao.NewCmdbNetworkDeviceDao(db),
	}
}

// getUserAllGroupIDs 获取用户的所有分组ID（系统角色组 + CMDB用户组）
func (s *CmdbAssetPermissionServiceImpl) getUserAllGroupIDs(userID uint) []uint {
	// CMDB用户组
	cmdbGroupIDs, _ := s.cmdbUserGroupDao.GetUserGroupIDs(userID)
	// 系统角色组（兼容旧数据）
	sysGroupIDs, _ := s.groupDao.GetUserGroupIDs(userID)
	return append(cmdbGroupIDs, sysGroupIDs...)
}

func toJSONStr(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (s *CmdbAssetPermissionServiceImpl) Create(c *gin.Context, req *CmdbAssetPermissionReq) {
	data := &model.CmdbAssetPermission{
		Name:              req.Name,
		Description:       req.Description,
		UserIDs:           toJSONStr(req.UserIDs),
		GroupIDs:          toJSONStr(req.GroupIDs),
		AssetTypes:        toJSONStr(req.AssetTypes),
		HostGroupIDs:      toJSONStr(req.HostGroupIDs),
		HostIDs:           toJSONStr(req.HostIDs),
		PhysicalIDs:       toJSONStr(req.PhysicalIDs),
		NetworkIDs:        toJSONStr(req.NetworkIDs),
		DatabaseIDs:       toJSONStr(req.DatabaseIDs),
		IDCIDs:            toJSONStr(req.IDCIDs),
		PermissionActions: toJSONStr(req.PermissionActions),
		IsActive:          req.IsActive,
		DateStart:         req.DateStart,
		DateExpired:       req.DateExpired,
	}
	if data.IsActive == 0 {
		data.IsActive = 1
	}
	if len(data.PermissionActions) <= 3 { // default: ["connect"]
		data.PermissionActions = `["connect"]`
	}
	if err := s.dao.Create(data); err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建授权规则失败: "+err.Error())
		return
	}
	result.Success(c, data)
}

func (s *CmdbAssetPermissionServiceImpl) Update(c *gin.Context, id uint, req *CmdbAssetPermissionReq) {
	data, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "授权规则不存在")
		return
	}
	data.Name = req.Name
	data.Description = req.Description
	data.UserIDs = toJSONStr(req.UserIDs)
	data.GroupIDs = toJSONStr(req.GroupIDs)
	data.AssetTypes = toJSONStr(req.AssetTypes)
	data.HostGroupIDs = toJSONStr(req.HostGroupIDs)
	data.HostIDs = toJSONStr(req.HostIDs)
	data.PhysicalIDs = toJSONStr(req.PhysicalIDs)
	data.NetworkIDs = toJSONStr(req.NetworkIDs)
	data.DatabaseIDs = toJSONStr(req.DatabaseIDs)
	data.IDCIDs = toJSONStr(req.IDCIDs)
	data.PermissionActions = toJSONStr(req.PermissionActions)
	data.IsActive = req.IsActive
	data.DateStart = req.DateStart
	data.DateExpired = req.DateExpired
	if err := s.dao.Update(data); err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新授权规则失败: "+err.Error())
		return
	}
	result.Success(c, data)
}

func (s *CmdbAssetPermissionServiceImpl) Delete(c *gin.Context, id uint) {
	if err := s.dao.Delete(id); err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除授权规则失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

func (s *CmdbAssetPermissionServiceImpl) GetByID(c *gin.Context, id uint) {
	data, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "授权规则不存在")
		return
	}
	result.Success(c, data)
}

func (s *CmdbAssetPermissionServiceImpl) GetList(c *gin.Context, query model.CmdbAssetPermissionQuery) {
	list, total, err := s.dao.GetList(query)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询授权规则失败: "+err.Error())
		return
	}
	result.Success(c, gin.H{"list": list, "total": total})
}

// GetMyAssets 获取当前用户拥有的资产授权
func (s *CmdbAssetPermissionServiceImpl) GetMyAssets(c *gin.Context) {
	admin, err := jwt.GetAdmin(c)
	if err != nil {
		result.Failed(c, http.StatusUnauthorized, "未授权")
		return
	}

	// 获取用户所属用户组
	userGroupIDs, _ := s.groupDao.GetUserGroupIDs(admin.ID)

	// 获取有效授权
	perms, err := s.dao.GetUserPermissions(admin.ID, userGroupIDs)
	if err != nil || len(perms) == 0 {
		result.Success(c, gin.H{
			"allowedHostIds":     []uint{},
			"allowedPhysicalIds": []uint{},
			"allowedNetworkIds":  []uint{},
			"permissionActions":  []string{},
		})
		return
	}

	// 合并所有授权中的资产ID和操作权限
	hostIDSet := make(map[uint]bool)
	physicalIDSet := make(map[uint]bool)
	networkIDSet := make(map[uint]bool)
	dbIDSet := make(map[uint]bool)
	hostGroupIDSet := make(map[uint]bool)
	actionSet := make(map[string]bool)

	for _, p := range perms {
		var actions []string
		json.Unmarshal([]byte(p.PermissionActions), &actions)
		for _, a := range actions {
			actionSet[a] = true
		}
		// 具体主机ID
		var hostIDs []uint
		json.Unmarshal([]byte(p.HostIDs), &hostIDs)
		for _, id := range hostIDs {
			hostIDSet[id] = true
		}
		// 具体物理机ID
		var phIDs []uint
		json.Unmarshal([]byte(p.PhysicalIDs), &phIDs)
		for _, id := range phIDs {
			physicalIDSet[id] = true
		}
		// 具体网络设备ID
		var netIDs []uint
		json.Unmarshal([]byte(p.NetworkIDs), &netIDs)
		for _, id := range netIDs {
			networkIDSet[id] = true
		}
		// 主机分组 (需要解析为ID集)
		var hgIDs []uint
		json.Unmarshal([]byte(p.HostGroupIDs), &hgIDs)
		for _, id := range hgIDs {
			hostGroupIDSet[id] = true
		}
		// 从主机分组加载所有主机
		for _, gid := range hgIDs {
			hosts := s.hostDao.GetCmdbHostsByGroupId(gid)
			for _, h := range hosts {
				hostIDSet[h.ID] = true
			}
		}
		// 数据库
		var dIDs []uint
		json.Unmarshal([]byte(p.DatabaseIDs), &dIDs)
		for _, id := range dIDs {
			dbIDSet[id] = true
		}
	}

	var actions []string
	for a := range actionSet {
		actions = append(actions, a)
	}
	var hostIDs, physicalIDs, networkIDs, dbIDs []uint
	for id := range hostIDSet {
		hostIDs = append(hostIDs, id)
	}
	for id := range physicalIDSet {
		physicalIDs = append(physicalIDs, id)
	}
	for id := range networkIDSet {
		networkIDs = append(networkIDs, id)
	}
	for id := range dbIDSet {
		dbIDs = append(dbIDs, id)
	}

	result.Success(c, gin.H{
		"allowedHostGroupIds": hostGroupIDSet,
		"allowedHostIds":      hostIDs,
		"allowedPhysicalIds":  physicalIDs,
		"allowedNetworkIds":   networkIDs,
		"allowedDatabaseIds":  dbIDs,
		"permissionActions":   actions,
	})
}

// CheckUserAssetPermission 检查用户是否有指定资产的权限
func (s *CmdbAssetPermissionServiceImpl) CheckUserAssetPermission(c *gin.Context, assetType string, assetID uint) bool {
	admin, err := jwt.GetAdmin(c)
	if err != nil {
		return false
	}

	// 管理员拥有所有权限
	isAdmin, _ := s.groupDao.IsAdmin(admin.ID)
	if isAdmin {
		return true
	}

	userGroupIDs, _ := s.groupDao.GetUserGroupIDs(admin.ID)
	perms, err := s.dao.GetUserPermissions(admin.ID, userGroupIDs)
	if err != nil || len(perms) == 0 {
		return false
	}

	for _, p := range perms {
		switch assetType {
		case "host":
			// 检查具体主机
			var hostIDs []uint
			json.Unmarshal([]byte(p.HostIDs), &hostIDs)
			for _, id := range hostIDs {
				if id == assetID {
					return true
				}
			}
			// 检查主机分组
			var hgIDs []uint
			json.Unmarshal([]byte(p.HostGroupIDs), &hgIDs)
			for _, gid := range hgIDs {
				hosts := s.hostDao.GetCmdbHostsByGroupId(gid)
				for _, h := range hosts {
					if h.ID == assetID {
						return true
					}
				}
			}
		case "physical":
			var ids []uint
			json.Unmarshal([]byte(p.PhysicalIDs), &ids)
			for _, id := range ids {
				if id == assetID {
					return true
				}
			}
		case "network":
			var ids []uint
			json.Unmarshal([]byte(p.NetworkIDs), &ids)
			for _, id := range ids {
				if id == assetID {
					return true
				}
			}
		}
	}
	return false
}

// GetUserGroupIDs 辅助函数，返回用户组ID列表
func (s *CmdbAssetPermissionServiceImpl) GetUserGroupIDs(userID uint) ([]uint, error) {
	return s.groupDao.GetUserGroupIDs(userID)
}

func (s *CmdbAssetPermissionServiceImpl) GetMyAllowedAssets(c *gin.Context) {
	admin, err := jwt.GetAdmin(c)
	if err != nil {
		result.Failed(c, http.StatusUnauthorized, "未授权")
		return
	}
	userGroupIDs, _ := s.groupDao.GetUserGroupIDs(admin.ID)
	perms, err := s.dao.GetUserPermissions(admin.ID, userGroupIDs)
	if err != nil || len(perms) == 0 {
		result.Success(c, gin.H{
			"hosts":    []model.CmdbHost{},
			"physical": []model.CmdbPhysicalMachine{},
			"networks": []model.CmdbNetworkDevice{},
		})
		return
	}

	hostIDSet := make(map[uint]bool)
	physicalIDSet := make(map[uint]bool)
	networkIDSet := make(map[uint]bool)

	for _, p := range perms {
		// Hosts from host groups
		var hgIDs []uint
		json.Unmarshal([]byte(p.HostGroupIDs), &hgIDs)
		for _, gid := range hgIDs {
			hosts := s.hostDao.GetCmdbHostsByGroupId(gid)
			for _, h := range hosts {
				hostIDSet[h.ID] = true
			}
		}
		var phIDs []uint
		json.Unmarshal([]byte(p.PhysicalIDs), &phIDs)
		for _, id := range phIDs {
			physicalIDSet[id] = true
		}
		var nIDs []uint
		json.Unmarshal([]byte(p.NetworkIDs), &nIDs)
		for _, id := range nIDs {
			networkIDSet[id] = true
		}
	}

	var hostIDs, physicalIDs, networkIDs []uint
	for id := range hostIDSet {
		hostIDs = append(hostIDs, id)
	}
	for id := range physicalIDSet {
		physicalIDs = append(physicalIDs, id)
	}
	for id := range networkIDSet {
		networkIDs = append(networkIDs, id)
	}

	hosts, _ := s.hostDao.GetListByIDs(hostIDs)
	physicals, _ := s.physicalDao.GetListByIDs(physicalIDs)
	networks, _ := s.networkDao.GetListByIDs(networkIDs)

	result.Success(c, gin.H{
		"hosts":    hosts,
		"physical": physicals,
		"networks": networks,
	})
}
