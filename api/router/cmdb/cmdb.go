package cmdb

import (
	"dodevops-api/api/cmdb/controller"
	"dodevops-api/api/cmdb/service"
	"dodevops-api/common"
	"dodevops-api/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterCmdbRoutes 注册系统相关路由
func RegisterCmdbRoutes(router *gin.RouterGroup) {
	db := common.GetDB()
	permMiddleware := middleware.NewCmdbAssetPermissionMiddleware()

	// ======================= 需要CMDB权限的数据路由 =======================
	// 资产分组 (assetType: group)
	router.POST("/cmdb/groupadd", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("group"), controller.CreateCmdbGroup)
	router.GET("/cmdb/grouplist", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("group"), controller.GetAllCmdbGroups)
	router.GET("/cmdb/grouplistwithhosts", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("group"), controller.GetAllCmdbGroupsWithHosts)
	router.PUT("/cmdb/groupupdate", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("group"), controller.UpdateCmdbGroup)
	router.DELETE("/cmdb/groupdelete", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("group"), controller.DeleteCmdbGroup)
	router.GET("/cmdb/groupbyname", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("group"), controller.GetCmdbGroupByName)
	// 主机管理 (assetType: host)
	router.POST("/cmdb/hostcreate", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostController().CreateCmdbHost)
	router.PUT("/cmdb/hostupdate", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostController().UpdateCmdbHost)
	router.DELETE("/cmdb/hostdelete", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostController().DeleteCmdbHost)
	router.GET("/cmdb/hostlist", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostController().GetCmdbHostListWithPage)
	router.GET("/cmdb/hostinfo", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostController().GetCmdbHostById)
	router.GET("/cmdb/hostgroup", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostController().GetCmdbHostsByGroupId)
	router.GET("/cmdb/hostbyname", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostController().GetCmdbHostsByHostNameLike)
	router.GET("/cmdb/hostbyip", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostController().GetCmdbHostsByIP)
	router.GET("/cmdb/hostbystatus", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostController().GetCmdbHostsByStatus)
	router.POST("/cmdb/hostimport", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostController().ImportHostsFromExcel)
	router.GET("/cmdb/hosttemplate", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostController().DownloadHostTemplate)
	router.POST("/cmdb/hostsync", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostController().SyncHostInfo)
	// 云主机管理 (assetType: host)
	router.POST("/cmdb/hostcloudcreatealiyun", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostCloudController().CreateAliyunHost)
	router.POST("/cmdb/hostcloudcreatetencent", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostCloudController().CreateTencentHost)
	router.POST("/cmdb/hostcloudcreatebaidu", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostCloudController().CreateBaiduHost)
	// SSH操作需要主机级别的细粒度权限
	router.GET("/cmdb/hostssh/connect/:id", middleware.AuthMiddleware(), permMiddleware.CheckHostPermission(), controller.NewCmdbHostSSHController(service.GetCmdbHostSSHService()).ConnectTerminal)
	router.GET("/cmdb/hostssh/command/:id", middleware.AuthMiddleware(), permMiddleware.CheckHostPermission(), controller.NewCmdbHostSSHController(service.GetCmdbHostSSHService()).ExecuteCommand)
	router.POST("/cmdb/hostssh/upload/:id", middleware.AuthMiddleware(), permMiddleware.CheckHostPermission(), controller.NewCmdbHostSSHController(service.GetCmdbHostSSHService()).UploadFile)
	router.GET("/cmdb/hostssh/files", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostSSHController(service.GetCmdbHostSSHService()).FileList)
	router.DELETE("/cmdb/hostssh/file", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostSSHController(service.GetCmdbHostSSHService()).DeleteFile)
	router.GET("/cmdb/hostssh/download", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("host"), controller.NewCmdbHostSSHController(service.GetCmdbHostSSHService()).DownloadFile)
	// SQL执行 (assetType: database)
	router.POST("/cmdb/sql/select", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("database"), controller.GetCmdbSQLRecordController().ExecuteSelect)
	router.POST("/cmdb/sql", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("database"), controller.GetCmdbSQLRecordController().ExecuteInsert)
	router.PUT("/cmdb/sql", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("database"), controller.GetCmdbSQLRecordController().ExecuteUpdate)
	router.DELETE("/cmdb/sql", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("database"), controller.GetCmdbSQLRecordController().ExecuteDelete)
	router.POST("/cmdb/sql/execute", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("database"), controller.GetCmdbSQLRecordController().ExecuteSQL)
	router.POST("/cmdb/sql/databaselist", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("database"), controller.GetCmdbSQLRecordController().ListDatabases)
	// SQL日志管理 (assetType: database)
	router.GET("/cmdb/sqlLog/list", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("database"), controller.GetCmdbSqlLogList)
	router.DELETE("/cmdb/sqlLog/delete", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("database"), controller.DeleteCmdbSqlLogById)
	router.DELETE("/cmdb/sqlLog/clean", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("database"), controller.CleanCmdbSqlLog)
	// 数据库管理 (assetType: database)
	router.POST("/cmdb/database", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("database"), controller.NewCmdbSQLController().CreateDatabase)
	router.PUT("/cmdb/database", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("database"), controller.NewCmdbSQLController().UpdateDatabase)
	router.DELETE("/cmdb/database", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("database"), controller.NewCmdbSQLController().DeleteDatabase)
	router.GET("/cmdb/database/info", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("database"), controller.NewCmdbSQLController().GetDatabase)
	router.GET("/cmdb/databaselist", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("database"), controller.NewCmdbSQLController().ListDatabases)
	router.GET("/cmdb/database/byname", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("database"), controller.NewCmdbSQLController().GetDatabasesByName)
	router.GET("/cmdb/database/bytype", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("database"), controller.NewCmdbSQLController().GetDatabasesByType)

	// ======================= 机房管理 (assetType: idc) =======================
	cmdbIDCCtrl := controller.NewCmdbIDCController(db)
	router.POST("/cmdb/idc", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("idc"), cmdbIDCCtrl.Create)
	router.PUT("/cmdb/idc/:id", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("idc"), cmdbIDCCtrl.Update)
	router.DELETE("/cmdb/idc/:id", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("idc"), cmdbIDCCtrl.Delete)
	router.GET("/cmdb/idc/:id", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("idc"), cmdbIDCCtrl.GetByID)
	router.GET("/cmdb/idc", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("idc"), cmdbIDCCtrl.GetList)
	router.GET("/cmdb/idc/all", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("idc"), cmdbIDCCtrl.GetAll)

	// ======================= 机柜管理 (assetType: cabinet) =======================
	cmdbCabinetCtrl := controller.NewCmdbCabinetController(db)
	router.POST("/cmdb/cabinet", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("cabinet"), cmdbCabinetCtrl.Create)
	router.PUT("/cmdb/cabinet/:id", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("cabinet"), cmdbCabinetCtrl.Update)
	router.DELETE("/cmdb/cabinet/:id", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("cabinet"), cmdbCabinetCtrl.Delete)
	router.GET("/cmdb/cabinet/:id", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("cabinet"), cmdbCabinetCtrl.GetByID)
	router.GET("/cmdb/cabinet", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("cabinet"), cmdbCabinetCtrl.GetList)
	router.GET("/cmdb/cabinet/idc/:idcId", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("cabinet"), cmdbCabinetCtrl.GetByIDC)

	// ======================= 物理机管理 (assetType: physical) =======================
	cmdbPhysicalCtrl := controller.NewCmdbPhysicalMachineController(db)
	router.POST("/cmdb/physical", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("physical"), cmdbPhysicalCtrl.Create)
	router.PUT("/cmdb/physical/:id", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("physical"), cmdbPhysicalCtrl.Update)
	router.DELETE("/cmdb/physical/:id", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("physical"), cmdbPhysicalCtrl.Delete)
	router.GET("/cmdb/physical/:id", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("physical"), cmdbPhysicalCtrl.GetByID)
	router.GET("/cmdb/physical", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("physical"), cmdbPhysicalCtrl.GetList)
	router.GET("/cmdb/physical/all", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("physical"), cmdbPhysicalCtrl.GetAll)
	router.GET("/cmdb/physical/stats", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("physical"), cmdbPhysicalCtrl.GetStats)

	// ======================= 网络设备管理 (assetType: network) =======================
	cmdbNetworkCtrl := controller.NewCmdbNetworkDeviceController(db)
	router.POST("/cmdb/network", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("network"), cmdbNetworkCtrl.Create)
	router.PUT("/cmdb/network/:id", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("network"), cmdbNetworkCtrl.Update)
	router.DELETE("/cmdb/network/:id", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("network"), cmdbNetworkCtrl.Delete)
	router.GET("/cmdb/network/:id", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("network"), cmdbNetworkCtrl.GetByID)
	router.GET("/cmdb/network", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("network"), cmdbNetworkCtrl.GetList)
	router.GET("/cmdb/network/all", middleware.AuthMiddleware(), permMiddleware.RequireCmdbPermission("network"), cmdbNetworkCtrl.GetAll)

	// ======================= 资产授权管理 (仅管理员) =======================
	cmdbPermCtrl := controller.NewCmdbAssetPermissionController(db)
	router.POST("/cmdb/permission", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbPermCtrl.Create)
	router.PUT("/cmdb/permission/:id", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbPermCtrl.Update)
	router.DELETE("/cmdb/permission/:id", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbPermCtrl.Delete)
	router.GET("/cmdb/permission/:id", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbPermCtrl.GetByID)
	router.GET("/cmdb/permission", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbPermCtrl.GetList)
	router.GET("/cmdb/permission/my", middleware.AuthMiddleware(), cmdbPermCtrl.GetMyAssets)
	router.GET("/cmdb/permission/check/:assetType/:assetId", middleware.AuthMiddleware(), cmdbPermCtrl.CheckPermission)

	// ======================= CMDB 用户组管理 (仅管理员) =======================
	cmdbUserGroupCtrl := controller.NewCmdbUserGroupController()
	router.POST("/cmdb/permission/user-group", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbUserGroupCtrl.Create)
	router.PUT("/cmdb/permission/user-group/:id", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbUserGroupCtrl.Update)
	router.DELETE("/cmdb/permission/user-group/:id", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbUserGroupCtrl.Delete)
	router.GET("/cmdb/permission/user-group", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbUserGroupCtrl.GetList)
	router.GET("/cmdb/permission/user-group/all", middleware.AuthMiddleware(), cmdbUserGroupCtrl.GetAll)
	router.GET("/cmdb/permission/user-group/:id/members", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbUserGroupCtrl.GetMembers)
	router.POST("/cmdb/permission/user-group/members", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbUserGroupCtrl.AddMembers)
	router.DELETE("/cmdb/permission/user-group/member", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbUserGroupCtrl.RemoveMember)

	// ======================= 凭据授权管理 (仅管理员) =======================
	cmdbCredPermCtrl := controller.NewCmdbCredentialPermissionController()
	router.POST("/cmdb/permission/credential", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbCredPermCtrl.Create)
	router.PUT("/cmdb/permission/credential/:id", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbCredPermCtrl.Update)
	router.DELETE("/cmdb/permission/credential/:id", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbCredPermCtrl.Delete)
	router.GET("/cmdb/permission/credential/:id", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbCredPermCtrl.GetByID)
	router.GET("/cmdb/permission/credential", middleware.AuthMiddleware(), permMiddleware.AdminOnly(), cmdbCredPermCtrl.GetList)
	router.GET("/cmdb/permission/credential/my", middleware.AuthMiddleware(), cmdbCredPermCtrl.GetMyCredentials)
}
