import request from "@/utils/request";

export interface CephField {
  key: string;
  label: string;
  required: boolean;
  hint?: string;
  options?: string[];
}
export interface CephOperation {
  id: string;
  group: string;
  label: string;
  description: string;
  dangerous: boolean;
  schedulable: boolean;
  fields: CephField[];
}
export interface CephTaskInput {
  operation: string;
  params: Record<string, string>;
  confirmation: string;
  timeoutSeconds: number;
}
export interface CephTask {
  name: string;
  operation: string;
  params: Record<string, string>;
  state: string;
  message: string;
  actor: string;
  createdAt: string;
  startedAt: string | null;
  finishedAt: string | null;
}
export interface CephRepository {
  name: string;
  pvc: string;
  createdAt: string;
}
export interface CephBackup {
  id: string;
  pool: string;
  image: string;
  snapshot: string;
  parent: string;
  size: number;
  bytes: number;
  createdAt: string;
  fileExists: boolean;
  sha256: string;
  error?: string;
}
export interface CephSchedule {
  name: string;
  schedule: string;
  timeZone: string;
  suspended: boolean;
  task: CephTaskInput;
  lastScheduleTime: string | null;
  lastSuccessfulTime: string | null;
  active: number;
}

async function call<T>(
  cluster: string | number,
  path: string,
  method = "get",
  data?: unknown,
  params?: unknown
): Promise<T> {
  const response = await request({
    url: `/api/v1/k8s/cluster/${encodeURIComponent(cluster)}/ceph/maintenance/${path}`,
    method,
    data,
    params
  });
  if (response.data.code !== 200)
    throw new Error(response.data.message || "Ceph 操作失败");
  return response.data.data as T;
}

export const cephApi = {
  capabilities: (id: string | number) =>
    call<{ operations: CephOperation[]; canWrite: boolean; namespace: string }>(
      id,
      "capabilities"
    ),
  tasks: (id: string | number) => call<CephTask[]>(id, "tasks"),
  createTask: (id: string | number, data: CephTaskInput) =>
    call<CephTask>(id, "tasks", "post", data),
  logs: (id: string | number, name: string) =>
    call<{
      task: CephTask;
      pods: Array<{
        pod: string;
        log?: string;
        logError?: string;
        conditions: unknown;
        containers: unknown;
        initContainers: unknown;
      }>;
    }>(id, `tasks/${encodeURIComponent(name)}/logs`),
  cancel: (id: string | number, name: string) =>
    call(id, `tasks/${encodeURIComponent(name)}/cancel`, "post"),
  repositories: (id: string | number) =>
    call<CephRepository[]>(id, "repositories"),
  createRepository: (
    id: string | number,
    data: { name: string; pvc: string }
  ) => call(id, "repositories", "post", data),
  backups: (id: string | number, repository: string) =>
    call<CephBackup[]>(id, "backups", "get", undefined, { repository }),
  schedules: (id: string | number) => call<CephSchedule[]>(id, "schedules"),
  createSchedule: (
    id: string | number,
    data: {
      name: string;
      schedule: string;
      timeZone: string;
      task: CephTaskInput;
    }
  ) => call(id, "schedules", "post", data),
  suspendSchedule: (id: string | number, name: string, suspended: boolean) =>
    call(id, `schedules/${encodeURIComponent(name)}`, "put", { suspended }),
  deleteSchedule: (id: string | number, name: string) =>
    call(id, `schedules/${encodeURIComponent(name)}`, "delete"),
  inspect: (
    id: string | number,
    kind: string,
    params: Record<string, string>
  ) => call<unknown>(id, "inspect", "get", undefined, { kind, ...params })
};
