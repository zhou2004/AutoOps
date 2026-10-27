// pkg/db/migrate.go
package db

import (
	aiopsmodel "dodevops-api/api/aiops/model"
	appmodel "dodevops-api/api/app/model"
	cmdbmodel "dodevops-api/api/cmdb/model"
	ccmodel "dodevops-api/api/configcenter/model"
	k8smodel "dodevops-api/api/k8s/model"
	monitormodel "dodevops-api/api/monitor/model"
	systemmodel "dodevops-api/api/system/model"
	taskmodel "dodevops-api/api/task/model"
	toolmodel "dodevops-api/api/tool/model"

	"gorm.io/gorm"
)

// 注册所有需要自动建表的 model
var models = []interface{}{
	&aiopsmodel.Config{},
	&aiopsmodel.Run{},
	&aiopsmodel.Session{},
	&aiopsmodel.Message{},
	&aiopsmodel.ToolCall{},
	&aiopsmodel.HarnessTicket{},
	&systemmodel.APIKey{},
	&systemmodel.RolePermission{},
	&cmdbmodel.CmdbGroup{},
	&ccmodel.EcsAuth{},
	&ccmodel.KeyManage{},
	&ccmodel.SyncSchedule{},
	&cmdbmodel.CmdbHost{},
	&cmdbmodel.CmdbAssetConnection{},
	&cmdbmodel.CmdbConnectionTicket{},
	&cmdbmodel.CmdbConnectionAudit{},
	&cmdbmodel.CmdbNetworkConfigSnapshot{},
	&cmdbmodel.CmdbNetworkConfigAudit{},
	&cmdbmodel.CmdbSQLRecord{},
	&cmdbmodel.CmdbSQL{},
	&cmdbmodel.CmdbDatabaseAudit{},
	&cmdbmodel.DatabaseDeployment{},
	&cmdbmodel.DatabaseAutomationRun{},
	&cmdbmodel.DatabaseAutomationLock{},
	&cmdbmodel.CmdbDockerAudit{},
	&cmdbmodel.CmdbIDC{},
	&cmdbmodel.CmdbCabinet{},
	&cmdbmodel.CmdbPhysicalMachine{},
	&cmdbmodel.CmdbNetworkDevice{},
	&cmdbmodel.CmdbAssetPermission{},
	&cmdbmodel.CmdbUserGroup{},
	&cmdbmodel.CmdbUserGroupMember{},
	&cmdbmodel.CmdbCredentialPermission{},
	&ccmodel.AccountAuth{},
	&taskmodel.TaskTemplate{},
	&taskmodel.Task{},
	&taskmodel.TaskWork{},
	&taskmodel.TaskExecutionHistory{},
	&taskmodel.TaskExecutionLog{},
	&taskmodel.TaskAnsible{},
	&taskmodel.TaskAnsibleWork{},
	&taskmodel.KubeaszOperation{},
	&monitormodel.Agent{},
	&monitormodel.MonitorCategory{},
	&monitormodel.DomainCert{},
	&monitormodel.MonitorAPIEndpoint{},
	&monitormodel.MonitorIncident{},
	&monitormodel.MonitorDataSource{},
	&monitormodel.MonitorAlertGroupRule{},
	&monitormodel.MonitorAlertRule{},
	&monitormodel.MonitorAlertRuleStyle{},
	&monitormodel.PrometheusAlertDB{},
	&monitormodel.AlertRecord{},
	&monitormodel.AlertRouter{},
	&monitormodel.AlertConfig{},
	&k8smodel.KubeCluster{},
	&k8smodel.K8sRbacRole{},
	&k8smodel.K8sRbacBinding{},
	&appmodel.Application{},
	&appmodel.JenkinsEnv{},
	&appmodel.QuickDeployment{},
	&appmodel.QuickDeploymentTask{},
	&systemmodel.SysOperationLog{},
	&toolmodel.Tool{},
	&toolmodel.ServiceDeploy{},
	// 可以继续添加其他模型...
}

// 自动迁移所有模型
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(models...); err != nil {
		return err
	}
	if err := migrateLegacyMonitorIncidents(db); err != nil {
		return err
	}
	return migrateManagementNavigation(db)
}
