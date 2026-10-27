import request from "@/utils/request";

export interface KubeaszConfig {
  masters: string[];
  workers: string[];
  etcd: string[];
  ssh_user: string;
  ssh_port: number;
  pod_cidr: string;
  service_cidr: string;
  network: string;
}
export interface KubeaszRequest {
  cluster: string;
  action: string;
  node?: string;
  step?: string;
  addon?: string;
  snapshot?: string;
  config?: KubeaszConfig;
  plan_hash?: string;
  confirmation?: string;
}
export interface KubeaszTemplate {
  id: string;
  name: string;
  description: string;
  risk: string;
  node: boolean;
  step: boolean;
  addon: boolean;
}
export interface KubeaszCluster {
  name: string;
  masters: string[] | null;
  workers: string[] | null;
  etcd: string[] | null;
  snapshots: string[];
}
export interface KubeaszCatalog {
  version: string;
  commit: string;
  kubernetes: string;
  root: string;
  checks: { name: string; ready: boolean; detail: string }[];
  templates: KubeaszTemplate[];
  steps: Record<string, string>;
  addons: Record<string, string>;
  clusters: KubeaszCluster[];
}
export interface KubeaszPlan {
  request: KubeaszRequest;
  command: string;
  confirmation: string;
  hash: string;
  ready: boolean;
  blockers: string[];
  warnings: string[];
  inventory?: string;
  targets: Record<string, string[]>;
}
export interface KubeaszOperation {
  id: number;
  cluster: string;
  action: string;
  parameters: string;
  command: string;
  version: string;
  status: string;
  operator_id: number;
  operator_name: string;
  error: string;
  created_at: string;
  started_at: string | null;
  finished_at: string | null;
  reconciled_at: string | null;
  reconciled_by: string;
  reconcile_note: string;
}
const base = "/task/kubeasz";
async function unwrap<T>(response: ReturnType<typeof request>): Promise<T> {
  const { data } = await response;
  if (data.code !== 200) throw new Error(data.message || "kubeasz 请求失败");
  return data.data as T;
}
export const getKubeaszCatalog = () =>
  unwrap<KubeaszCatalog>(request({ url: `${base}/catalog`, method: "get" }));
export const previewKubeasz = (data: KubeaszRequest) =>
  unwrap<KubeaszPlan>(
    request({ url: `${base}/preview`, method: "post", data })
  );
export const startKubeasz = (data: KubeaszRequest) =>
  unwrap<KubeaszOperation>(
    request({ url: `${base}/operations`, method: "post", data })
  );
export const getKubeaszHistory = (params: {
  cluster?: string;
  page: number;
  size: number;
}) =>
  unwrap<{ list: KubeaszOperation[]; total: number }>(
    request({ url: `${base}/operations`, method: "get", params })
  );
export const getKubeaszOperation = (id: number) =>
  unwrap<{ operation: KubeaszOperation; log: string }>(
    request({ url: `${base}/operations/${id}`, method: "get" })
  );
export const cancelKubeaszOperation = (id: number) =>
  unwrap<{ message: string }>(
    request({ url: `${base}/operations/${id}/cancel`, method: "post" })
  );

export const reconcileKubeaszOperation = (
  id: number,
  data: { confirmation: string; note: string }
) =>
  unwrap<{ message: string }>(
    request({ url: `${base}/operations/${id}/reconcile`, method: "post", data })
  );
