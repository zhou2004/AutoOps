/**
 * 导航工具API接口管理
 *
 * @author xiaoRui
 */

import request from '@/utils/request'

/**
 * 创建导航工具
 * @param {Object} data - 导航工具信息
 * @param {string} data.title - 导航标题，1-100字符
 * @param {string} data.icon - 图标URL
 * @param {string} data.link - 链接地址，必须是有效URL
 * @param {number} data.sort - 排序，默认0
 */
export function CreateTool(data) {
  return request({
    url: '/api/v1/tool',
    method: 'post',
    data,
    headers: {
      'Content-Type': 'application/json'
    }
  })
}

/**
 * 获取导航工具详情
 * @param {number} id - 工具ID
 */
export function GetToolById(id) {
  return request({
    url: `/api/v1/tool/${id}`,
    method: 'get',
    headers: {
      'Accept': 'application/json'
    }
  })
}

/**
 * 更新导航工具
 * @param {Object} data - 导航工具信息
 * @param {number} data.id - 工具ID
 * @param {string} data.title - 导航标题，1-100字符
 * @param {string} data.icon - 图标URL
 * @param {string} data.link - 链接地址，必须是有效URL
 * @param {number} data.sort - 排序
 */
export function UpdateTool(data) {
  return request({
    url: '/api/v1/tool',
    method: 'put',
    data,
    headers: {
      'Content-Type': 'application/json'
    }
  })
}

/**
 * 删除导航工具
 * @param {number} id - 工具ID
 */
export function DeleteTool(id) {
  return request({
    url: `/api/v1/tool/${id}`,
    method: 'delete',
    headers: {
      'Accept': 'application/json'
    }
  })
}

/**
 * 获取导航工具列表（分页）
 * @param {Object} params - 查询参数
 * @param {string} params.title - 标题模糊搜索
 * @param {number} params.pageNum - 页码，默认1
 * @param {number} params.pageSize - 每页数量，默认10
 */
export function GetToolList(params) {
  return request({
    url: '/api/v1/tool/list',
    method: 'get',
    params,
    headers: {
      'Accept': 'application/json'
    }
  })
}

/**
 * 获取所有导航工具（不分页）
 */
export function GetAllTools() {
  return request({
    url: '/api/v1/tool/all',
    method: 'get',
    headers: {
      'Accept': 'application/json'
    }
  })
}

/**
 * 上传图标
 * @param {FormData} formData - 包含file字段的FormData对象
 */
export function UploadIcon(formData) {
  return request({
    url: '/api/v1/upload',
    method: 'post',
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}

// ==================== 运维工具箱API ====================

/**
 * 获取可部署服务列表
 */
export function getServicesList() {
  return request({
    url: '/api/v1/tool/services',
    method: 'get'
  })
}

/**
 * 获取服务详情
 * @param {string} serviceId - 服务ID
 */
export function getServiceDetail(serviceId) {
  return request({
    url: `/api/v1/tool/services/${serviceId}`,
    method: 'get'
  })
}

/**
 * 创建部署任务
 * @param {Object} data - 部署参数
 * @param {string} data.serviceId - 服务ID
 * @param {string} data.version - 版本ID
 * @param {number} data.hostId - 主机ID
 * @param {string} data.installDir - 安装目录
 * @param {Object} data.envVars - 环境变量
 * @param {boolean} data.autoStart - 是否自动启动
 */
export function createDeploy(data) {
  return request({
    url: '/api/v1/tool/deploy',
    method: 'post',
    data
  })
}

/**
 * 获取部署历史列表
 * @param {Object} params - 查询参数
 * @param {string} params.serviceName - 服务名称
 * @param {number} params.status - 状态
 * @param {number} params.pageNum - 页码
 * @param {number} params.pageSize - 每页数量
 */
export function getDeployList(params) {
  return request({
    url: '/api/v1/tool/deploy/list',
    method: 'get',
    params
  })
}

/**
 * 获取部署状态
 * @param {number} id - 部署ID
 */
export function getDeployStatus(id) {
  return request({
    url: `/api/v1/tool/deploy/${id}/status`,
    method: 'get'
  })
}

/**
 * 卸载服务
 * @param {number} id - 部署ID
 */
export function deleteDeploy(id) {
  return request({
    url: `/api/v1/tool/deploy/${id}`,
    method: 'delete'
  })
}
