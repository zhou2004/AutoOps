
import request from '@/utils/request'

// ---- 数据源 ----
export function getDataSources(params) {
    return request({ url: '/monitor/datasources', method: 'get', params })
}
export function getDataSourceById(id) {
    return request({ url: `/monitor/datasource/${id}`, method: 'get' })
}
export function createDataSource(data) {
    return request({ url: '/monitor/datasource', method: 'post', data })
}
export function updateDataSource(data) {
    return request({ url: '/monitor/datasource', method: 'put', data })
}
export function deleteDataSource(id) {
    return request({ url: `/monitor/datasource/${id}`, method: 'delete' })
}

// ---- 告警规则分组 (Group) ----
export function getAlertGroupList(params) {
    return request({ url: '/monitor/alert/groups', method: 'get', params })
}
export function getAlertGroupById(id) {
    return request({ url: `/monitor/alert/group/${id}`, method: 'get' })
}
export function createAlertGroup(data) {
    return request({ url: '/monitor/alert/group', method: 'post', data })
}
export function updateAlertGroup(data) {
    return request({ url: '/monitor/alert/group', method: 'put', data })
}
export function deleteAlertGroup(id) {
    return request({ url: `/monitor/alert/group/${id}`, method: 'delete' })
}

// ---- 告警规则 (Rule) ----
export function getAlertRulesList(params) {
    return request({ url: '/monitor/alert/rules_list', method: 'get', params })
}
export function getRuleListByGroup(id, params) {
    return request({ url: `/monitor/alert/rules/${id}`, method: 'get', params })
}
export function createAlertRule(data) {
    return request({ url: '/monitor/alert/rule', method: 'post', data })
}
export function updateAlertRule(data) {
    return request({ url: '/monitor/alert/rule', method: 'put', data })
}
export function deleteAlertRule(id) {
    return request({ url: `/monitor/alert/rule/${id}`, method: 'delete' })
}
export function checkAlertRule(data) {
    return request({ url: '/monitor/alert/rule/check', method: 'post', data })
}

// ---- 告警分类 (Style) ----
export function getAlertStyles(params) {
    return request({ url: '/monitor/alert/styles', method: 'get', params })
}
export function createAlertStyle(data) {
    return request({ url: '/monitor/alert/style', method: 'post', data })
}
export function updateAlertStyle(data) {
    return request({ url: '/monitor/alert/style', method: 'put', data })
}
export function deleteAlertStyle(id) {
    return request({ url: `/monitor/alert/style/${id}`, method: 'delete' })
}

// ==== 告警通知模块 ====
export function getAlertTemplates(params) {
    return request({ url: '/monitor/alert/templates', method: 'get', params })
}
export function createAlertTemplate(data) {
    return request({ url: '/monitor/alert/template', method: 'post', data })
}
export function updateAlertTemplate(id, data) {
    return request({ url: `/monitor/alert/template/${id}`, method: 'put', data })
}
export function deleteAlertTemplate(id) {
    return request({ url: `/monitor/alert/template/${id}`, method: 'delete' })
}

export function getAlertRouters(params) {
    return request({ url: '/monitor/alert/routers', method: 'get', params })
}
export function createAlertRouter(data) {
    return request({ url: '/monitor/alert/router', method: 'post', data })
}
export function updateAlertRouter(id, data) {
    return request({ url: `/monitor/alert/router/${id}`, method: 'put', data })
}
export function deleteAlertRouter(id) {
    return request({ url: `/monitor/alert/router/${id}`, method: 'delete' })
}
export function reloadAlertRouters() {
    return request({ url: '/monitor/alert/router/reload', method: 'post' })
}

export function getAlertRecords(params) {
    return request({ url: '/monitor/alert/records', method: 'get', params })
}
export function cleanAlertRecords() {
    return request({ url: '/monitor/alert/records/clean', method: 'delete' })
}

// ---- Agent管理 ----
export function scanNodeExporter(data) {
    return request({ url: '/monitor/agent/scan-node-exporter', method: 'post', data })
}

// ---- 域名证书监控 ----
export function getDomainCertList(params) {
    return request({ url: '/monitor/domain-cert/list', method: 'get', params })
}
export function getDomainCertById(id) {
    return request({ url: `/monitor/domain-cert/${id}`, method: 'get' })
}
export function addDomainCert(data) {
    return request({ url: '/monitor/domain-cert', method: 'post', data })
}
export function updateDomainCert(data) {
    return request({ url: '/monitor/domain-cert', method: 'put', data })
}
export function deleteDomainCert(id) {
    return request({ url: `/monitor/domain-cert/${id}`, method: 'delete' })
}
export function batchDeleteDomainCert(data) {
    return request({ url: '/monitor/domain-cert/batch-delete', method: 'post', data })
}
export function checkDomainCert(id) {
    return request({ url: `/monitor/domain-cert/check/${id}`, method: 'get' })
}
export function checkAllDomainCert() {
    return request({ url: '/monitor/domain-cert/check-all', method: 'post' })
}

// ---- API端点监控 ----
export function getAPIEndpointList(params) {
    return request({ url: '/monitor/api-endpoint/list', method: 'get', params })
}
export function getAPIEndpointById(id) {
    return request({ url: `/monitor/api-endpoint/${id}`, method: 'get' })
}
export function addAPIEndpoint(data) {
    return request({ url: '/monitor/api-endpoint', method: 'post', data })
}
export function updateAPIEndpoint(data) {
    return request({ url: '/monitor/api-endpoint', method: 'put', data })
}
export function deleteAPIEndpoint(id) {
    return request({ url: `/monitor/api-endpoint/${id}`, method: 'delete' })
}
export function batchDeleteAPIEndpoint(data) {
    return request({ url: '/monitor/api-endpoint/batch-delete', method: 'post', data })
}
export function checkAPIEndpoint(id) {
    return request({ url: `/monitor/api-endpoint/check/${id}`, method: 'get' })
}
export function checkAllAPIEndpoint() {
    return request({ url: '/monitor/api-endpoint/check-all', method: 'post' })
}

// ---- 故障管理 ----
export function getIncidentList(params) {
    return request({ url: '/monitor/incident/list', method: 'get', params })
}
export function getIncidentStats() {
    return request({ url: '/monitor/incident/stats', method: 'get' })
}
export function resolveIncident(id) {
    return request({ url: `/monitor/incident/resolve/${id}`, method: 'post' })
}
export function deleteIncident(id) {
    return request({ url: `/monitor/incident/${id}`, method: 'delete' })
}