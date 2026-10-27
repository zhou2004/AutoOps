/**
 * Dashboard API接口管理
 *
 * @author Generated
 */

import request from "@/utils/request";

export interface CountItem {
  name: string;
  count: number;
}
export interface Inventory {
  total: number;
  statuses: CountItem[];
  distribution: CountItem[];
}
export interface ExecutionOverview {
  allTimeTotal: number;
  total: number;
  success: number;
  failed: number;
  running: number;
  pending: number;
  other: number;
  successRate: number | null;
  trend: { date: string; total: number; success: number; failed: number }[];
  recent: { id: number; name: string; status: number; createdAt: string }[];
}
export interface IncidentOverview {
  levels: CountItem[];
  firing: number;
  critical: number;
  warning: number;
  info: number;
  periodTotal: number;
  trend: CountItem[];
  recent: {
    id: number;
    title: string;
    level: string;
    source: string;
    createdAt: string;
  }[];
}
export interface DashboardOverview {
  generatedAt: string;
  days: number;
  timezone: string;
  startDate: string;
  endDate: string;
  assetScope: "all" | "authorized";
  errors: Record<string, string>;
  hosts: Inventory | null;
  databases: Inventory | null;
  physical: Inventory | null;
  network: Inventory | null;
  clusters: Inventory | null;
  applications: Inventory | null;
  deployments: ExecutionOverview | null;
  jobs: ExecutionOverview | null;
  ansible: ExecutionOverview | null;
  agents: Inventory | null;
  endpoints: Inventory | null;
  certificates: Inventory | null;
  alerts: Inventory | null;
  incidents: IncidentOverview | null;
}
export async function getDashboardOverview(
  days: number
): Promise<DashboardOverview> {
  const { data } = await request({
    url: "dashboard/overview",
    method: "get",
    params: { days }
  });
  if (data.code !== 200 || !data.data?.generatedAt) {
    throw new Error(data.message || "仪表盘数据读取失败");
  }
  return data.data;
}

// Dashboard统计数据API
export function getDashboardStats() {
  return request({
    url: "dashboard/stats",
    method: "get"
  });
}

// 业务分布统计API
export function getBusinessDistribution() {
  return request({
    url: "dashboard/business-distribution",
    method: "get"
  });
}

// 资产数量统计API
export function getAssetsStatistics() {
  return request({
    url: "dashboard/assets",
    method: "get"
  });
}

export default {
  getDashboardOverview,
  // Dashboard统计数据
  getDashboardStats,
  // 业务分布统计
  getBusinessDistribution,
  // 资产数量统计
  getAssetsStatistics
};
