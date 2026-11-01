import request from "@/utils/request";

export interface Member {
  id: number;
  clusterId: number;
  hostId: number;
  sourceId: number;
  accountId: number;
  name: string;
  kind: string;
  address: string;
  port: number;
  role: string;
  shard: string;
  nativeId: string;
  parentId: string;
  version: string;
  dataDir: string;
  service: string;
  managed: boolean;
  state: string;
  metrics: Record<string, any>;
  collectedAt?: string;
}
export interface Endpoint {
  id: number;
  name: string;
  mode: string;
  addresses: string[];
  status: string;
  masterName: string;
}
export interface Cluster {
  vip?: {
    clusterId: number;
    cidr: string;
    interface: string;
    routerId: number;
    memberIds: number[];
    status: string;
    ownerMemberId: number;
    collectedAt?: string;
    verifiedAt?: string;
    lastDrill?: Record<string, any>;
    observation?: Record<string, any>;
  };
  replication?: {
    sourceId: number;
    targetId: number;
    state: string;
    message: string;
    evidence: Record<string, any>;
    collectedAt?: string;
  }[];
  compatibilityReason?: string;
  placementWarnings?: string[];
  latestBackup?: {
    id: number;
    status: string;
    createdAt: string;
    verifiedAt?: string;
  };
  currentOperation?: { id: number; action: string; status: string };
  configurationStatus?: string;
  configurationCollectedAt?: string;
  id: number;
  name: string;
  engine: string;
  topology: string;
  groupId: number;
  environment: string;
  management: string;
  status: string;
  revision: number;
  members: Member[];
  endpoints: Endpoint[];
  health: Record<string, string>;
  policy: Record<string, any>;
  collectedAt?: string;
  haValidatedAt?: string;
  restoreVerifiedAt?: string;
}
export interface Capability {
  action: string;
  label: string;
  enabled: boolean;
  reason: string;
  permission: string;
}
export interface Plan {
  id: number;
  idempotencyKey?: string;
  revision: number;
  action: string;
  executable: boolean;
  expiresAt: string;
  checks: { name: string; passed: boolean; message: string }[];
}

// Keep retries bound to the same plan, including with older API versions and
// browsers accessing AutoOps over an HTTP LAN address.
export function operationPlanKey(plan: Plan): string {
  if (!Number.isSafeInteger(plan.id) || plan.id <= 0)
    throw new Error("执行计划无效，请重新预检");
  const key = plan.idempotencyKey ?? `db-plan-${plan.id}`;
  if (typeof key !== "string" || !/^[\x21-\x7e]{8,80}$/.test(key))
    throw new Error("执行计划缺少有效的提交标识，请重新预检");
  return key;
}
export interface Operation {
  id: number;
  clusterId: number;
  action: string;
  status: string;
  message: string;
  createdAt: string;
  heartbeatAt?: string;
  steps: {
    id: number;
    name: string;
    status: string;
    log: string;
    startedAt: string;
    finishedAt?: string;
  }[];
}
export interface Page<T> {
  list: T[];
  total: number;
}
export async function dbRequest<T = any>(
  path: string,
  method = "get",
  data?: unknown,
  params?: unknown,
  signal?: AbortSignal
): Promise<T> {
  const response = await request({
    url: `/api/v1/database/${path}`,
    method,
    data,
    params,
    signal,
    timeout: 180000
  });
  if (response.data?.code !== 200)
    throw new Error(response.data?.message || "数据库管理请求失败");
  return response.data.data;
}
export const dbPlatform = {
  clusters: (params?: unknown) =>
    dbRequest<Page<Cluster>>("clusters", "get", undefined, params),
  cluster: (id: number) =>
    dbRequest<{ cluster: Cluster; capabilities: Capability[]; slots: any }>(
      `clusters/${id}`
    ),
  plan: (id: number, action: string, parameters: unknown) =>
    dbRequest<Plan>(`clusters/${id}/operation-plans`, "post", {
      action,
      parameters
    }),
  submit: (
    id: number,
    planId: number,
    idempotencyKey: string,
    confirmed: boolean
  ) =>
    dbRequest<Operation>(`clusters/${id}/operations`, "post", {
      planId,
      idempotencyKey,
      confirmed
    }),
  operation: (id: number) => dbRequest<Operation>(`operations/${id}`)
};
export const statusLabels: Record<string, string> = {
  healthy: "正常",
  degraded: "降级",
  unavailable: "不可用",
  unreachable: "无法连接",
  conflict: "角色冲突",
  unknown: "未知",
  stale: "状态过期",
  planned: "待安装",
  missing: "未配置",
  "not-applicable": "不适用",
  observe: "观察模式",
  operate: "运维托管",
  ha: "高可用托管",
  queued: "排队",
  preflight: "预检",
  running: "执行中",
  verifying: "核验中",
  succeeded: "成功",
  failed: "失败",
  complete: "完整",
  primary: "主节点",
  replica: "副本",
  leader: "Leader",
  follower: "Follower",
  "pending-restart": "待重启",
  initializing: "等待复制基线",
  offline: "已隔离，等待重建",
  drifted: "配置漂移",
  superseded: "已被新版本替代",
  expired: "已过期清理",
  applied: "已应用"
};
export const label = (value?: string) =>
  value ? statusLabels[value] || value : "—";
export const operationStatusLabel = (value?: string) =>
  value === "unknown" ? "结果待核验" : label(value);
export const tagType = (
  value?: string
): "success" | "warning" | "danger" | "info" =>
  ["healthy", "succeeded", "complete", "applied"].includes(value || "")
    ? "success"
    : ["unavailable", "unreachable", "conflict", "failed"].includes(value || "")
      ? "danger"
      : ["degraded", "unknown", "stale", "pending-restart"].includes(
            value || ""
          )
        ? "warning"
        : "info";
export const bytes = (v: number) => {
  if (!Number.isFinite(v)) return "—";
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  let n = v,
    i = 0;
  while (n >= 1024 && i < 4) {
    n /= 1024;
    i++;
  }
  return `${n.toFixed(i ? 1 : 0)} ${units[i]}`;
};
