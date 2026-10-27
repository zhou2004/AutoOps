import request from "@/utils/request";

export type ScalingKind = "hpa" | "vpa" | "kpa";
export interface ScalingObject {
  apiVersion: string;
  kind: string;
  metadata: {
    name: string;
    namespace: string;
    resourceVersion: string;
    creationTimestamp?: string;
    ownerReferences?: Array<{ controller?: boolean }>;
  };
  spec: Record<string, any>;
  status?: Record<string, any>;
}
export interface ScalingCapabilities {
  capabilities: Record<
    ScalingKind | "metrics",
    { available: boolean; apiVersion: string; message: string }
  >;
  permissions: Record<ScalingKind, Record<string, boolean>>;
}
export interface ScalingRequest {
  name: string;
  resourceVersion: string;
  hpa?: Record<string, unknown>;
  vpa?: Record<string, unknown>;
  kpa?: Record<string, unknown>;
}
async function call<T>(
  cluster: string,
  namespace: string,
  path: string,
  method = "get",
  data?: unknown,
  params?: unknown
): Promise<T> {
  const response = await request({
    url: `/api/v1/k8s/cluster/${encodeURIComponent(cluster)}/namespaces/${encodeURIComponent(namespace)}/autoscaling/${path}`,
    method,
    data,
    params
  });
  if (response.data.code !== 200)
    throw new Error(response.data.message || "弹性伸缩操作失败");
  return response.data.data as T;
}
export const autoscalingApi = {
  capabilities: (id: string, ns: string) =>
    call<ScalingCapabilities>(id, ns, "capabilities"),
  list: (id: string, ns: string, kind: ScalingKind) =>
    call<ScalingObject[]>(id, ns, kind),
  save: (
    id: string,
    ns: string,
    kind: ScalingKind,
    data: ScalingRequest,
    create: boolean,
    dryRun: boolean
  ) =>
    call<ScalingObject>(
      id,
      ns,
      create ? kind : `${kind}/${encodeURIComponent(data.name)}`,
      create ? "post" : "put",
      data,
      { dryRun }
    ),
  remove: (id: string, ns: string, kind: ScalingKind, row: ScalingObject) =>
    call(
      id,
      ns,
      `${kind}/${encodeURIComponent(row.metadata.name)}`,
      "delete",
      undefined,
      { resourceVersion: row.metadata.resourceVersion }
    )
};
