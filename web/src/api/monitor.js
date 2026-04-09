import request from '@/utils/request'

// ---- 数据源 ----
export function getDataSources(params) {
    return request({ url: '/monitor/datasources', method: 'get', params })
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
