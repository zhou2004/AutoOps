/**
 * Dashboard API接口管理
 *
 * @author Generated
 */

import request from "@/utils/request"

// Dashboard统计数据API
export function getDashboardStats() {
    return request({
        url: 'dashboard/stats',
        method: 'get'
    })
}

// 业务分布统计API
export function getBusinessDistribution() {
    return request({
        url: 'dashboard/business-distribution',
        method: 'get'
    })
}

// 资产数量统计API
export function getAssetsStatistics() {
    return request({
        url: 'dashboard/assets',
        method: 'get'
    })
}

export default {
    // Dashboard统计数据
    getDashboardStats,
    // 业务分布统计
    getBusinessDistribution,
    // 资产数量统计
    getAssetsStatistics
}