import { locale, tr } from "@/locales";
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
  labelEn?: string;
  enabled: boolean;
  reason: string;
  permission: string;
}
export const capabilityLabel = (capability: Capability) =>
  locale.value === "en-US" && capability.labelEn
    ? capability.labelEn
    : tr(capability.label);

// Preflight checks append member names to these system labels. Translate only
// the label, keeping the instance identity literal.
export const preflightCheckLabel = (name: string) => {
  const prefixes = [
    "保护性只读 ",
    "受管实例 ",
    "受管插件 ",
    "服务归属 ",
    "目标 "
  ];
  const prefix = prefixes.find(value => name.startsWith(value));
  return prefix
    ? tr(`${prefix}{value0}`, { value0: name.slice(prefix.length) })
    : tr(name);
};

export const operationMessage = (message?: string) => {
  const resolved = message?.match(
    /^实际结果由核验任务 #(\d+) 确认，未重放原切换或恢复命令$/
  );
  return resolved
    ? tr("实际结果由核验任务 #{value0} 确认，未重放原切换或恢复命令", {
        value0: resolved[1]
      })
    : tr(message);
};

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
    throw new Error(tr("执行计划无效，请重新预检"));
  const key = plan.idempotencyKey ?? `db-plan-${plan.id}`;
  if (typeof key !== "string" || !/^[\x21-\x7e]{8,80}$/.test(key))
    throw new Error(tr("执行计划缺少有效的提交标识，请重新预检"));
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
    throw new Error(tr(response.data?.message || "数据库管理请求失败"));
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
  get healthy() {
    return tr("正常");
  },
  get degraded() {
    return tr("降级");
  },
  get unavailable() {
    return tr("不可用");
  },
  get unreachable() {
    return tr("无法连接");
  },
  get conflict() {
    return tr("角色冲突");
  },
  get unknown() {
    return tr("未知");
  },
  get stale() {
    return tr("状态过期");
  },
  get planned() {
    return tr("待安装");
  },
  get missing() {
    return tr("未配置");
  },
  get "not-applicable"() {
    return tr("不适用");
  },
  get observe() {
    return tr("观察模式");
  },
  get operate() {
    return tr("运维托管");
  },
  get ha() {
    return tr("高可用托管");
  },
  get queued() {
    return tr("排队");
  },
  get preflight() {
    return tr("预检");
  },
  get running() {
    return tr("执行中");
  },
  get verifying() {
    return tr("核验中");
  },
  get succeeded() {
    return tr("成功");
  },
  get failed() {
    return tr("失败");
  },
  get complete() {
    return tr("完整");
  },
  get primary() {
    return tr("主节点");
  },
  get replica() {
    return tr("副本");
  },
  leader: "Leader",
  follower: "Follower",
  get "pending-restart"() {
    return tr("待重启");
  },
  get initializing() {
    return tr("等待复制基线");
  },
  get offline() {
    return tr("已隔离，等待重建");
  },
  get drifted() {
    return tr("配置漂移");
  },
  get superseded() {
    return tr("已被新版本替代");
  },
  get expired() {
    return tr("已清理");
  },
  get applied() {
    return tr("已应用");
  }
};
export const label = (value?: string) =>
  value ? statusLabels[value] || value : "—";
export const operationStatusLabel = (value?: string) =>
  value === "unknown" ? tr("结果待核验") : label(value);
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
