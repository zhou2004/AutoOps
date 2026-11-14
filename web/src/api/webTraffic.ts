import { webQuery, webRequest } from "./webservice";

export type TrafficRange = "1h" | "6h" | "24h" | "7d";
export interface TrafficMetrics {
  qps: number | null;
  activeConnections: number | null;
  responseBytesPerSecond: number | null;
  requestBytesPerSecond: number | null;
  errorRate: number | null;
  averageLatencyMs: number | null;
  loggedRequests: number | null;
}
export interface TrafficPoint extends Omit<TrafficMetrics, "loggedRequests"> {
  at: string;
}
export interface TrafficNode {
  instanceId: number;
  sourceId: number;
  clusterId: number;
  name: string;
  address: string;
  status: "healthy" | "partial" | "unknown";
  collectedAt: string | null;
  metrics: TrafficMetrics;
  capabilities: {
    stubStatus: boolean;
    accessLog: boolean;
    requestBytes: boolean;
    latency: boolean;
    upstream: boolean;
  };
  message: string;
}
export interface TrafficRank {
  value: string;
  requests: number;
  responseBytes: number;
}
export interface WebTraffic {
  range: TrafficRange;
  from: string;
  to: string;
  intervalSeconds: number;
  summary: TrafficMetrics;
  series: TrafficPoint[];
  nodes: TrafficNode[];
  topIPs: TrafficRank[];
  topURIs: TrafficRank[];
  upstreams: TrafficRank[];
  statusCodes: { value: string; requests: number }[];
  coverage: {
    instances: number;
    freshInstances: number;
    stubStatusInstances: number;
    logInstances: number;
    logAuthorizedInstances: number;
    partial: boolean;
    notes: string[];
  };
}
export interface TrafficFilter {
  range: TrafficRange;
  clusterId?: number;
  instanceId?: number;
  groupId?: number;
}
export function webTraffic(filters: TrafficFilter, signal?: AbortSignal) {
  return webRequest<WebTraffic>(
    `/traffic?${webQuery(filters)}`,
    "GET",
    undefined,
    signal
  );
}

export interface TrafficUpstreamMember {
  address: string;
  weight: number;
  backup: boolean;
  down: boolean;
  maxFails: number;
  failTimeout: number;
}
export interface TrafficPolicy {
  enabled: boolean;
  trafficLog?: boolean;
  allow: string[];
  deny: string[];
  requestsPerSecond: number;
  burst: number;
  nodelay: boolean;
  maxConnections: number;
  responseBytesPerSecond: number;
  upstream?: {
    name?: string;
    strategy: "round_robin" | "least_conn" | "ip_hash";
    members: TrafficUpstreamMember[];
  };
  routes: {
    path: string;
    modifier: string;
    scheme: "http" | "https";
    upstream: string;
  }[];
}
export interface TrafficPolicyTarget {
  path: string;
  serverIndex: number;
  serverNames: string[];
  listens: string[];
  locations: { path: string; modifier: string; proxyPass: string }[];
  policy?: TrafficPolicy;
}
export interface TrafficPolicySnapshot {
  clusterId: number;
  instanceIds: number[];
  baselines: Record<number, string>;
  targets: TrafficPolicyTarget[];
  warnings: string[];
}
export interface TrafficPolicyPreview {
  files: { path: string; before: string; after: string }[];
  checks: {
    instanceId: number;
    name: string;
    valid: boolean;
    message: string;
  }[];
  valid: boolean;
  baselines: Record<number, string>;
  policy: TrafficPolicy;
  upstreamName: string;
  warnings: string[];
}
export interface TrafficPolicyInput {
  name?: string;
  instanceIds: number[];
  path: string;
  serverIndex: number;
  baselines: Record<number, string>;
  policy: TrafficPolicy;
}
export function webTrafficPolicy(
  clusterId: number,
  instanceId?: number,
  signal?: AbortSignal
) {
  return webRequest<TrafficPolicySnapshot>(
    `/clusters/${clusterId}/traffic-policy?${webQuery({ instanceId })}`,
    "GET",
    undefined,
    signal
  );
}
export function webPreviewTrafficPolicy(
  clusterId: number,
  body: TrafficPolicyInput,
  signal?: AbortSignal
) {
  return webRequest<TrafficPolicyPreview>(
    `/clusters/${clusterId}/traffic-policy/preview`,
    "POST",
    body,
    signal
  );
}
export function webSaveTrafficPolicy(
  clusterId: number,
  body: TrafficPolicyInput,
  signal?: AbortSignal
) {
  return webRequest<{
    draft: import("./webservice").WebResource;
    preview: TrafficPolicyPreview;
  }>(`/clusters/${clusterId}/traffic-policy/draft`, "POST", body, signal);
}
