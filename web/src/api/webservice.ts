import { platformRequest } from "./platformPermissions";
export interface WebResource {
  id: number;
  name: string;
  groupId: number;
  clusterId: number;
  sourceId: number;
  engine: string;
  environment: string;
  status: string;
  management: string;
  revision: number;
  spec: Record<string, any>;
  observed?: Record<string, any>;
  collectedAt?: string;
  connection?: {
    hostId: number;
    hostName: string;
    address: string;
    port: number;
    username: string;
    alias: string;
  };
}
export interface WebOperation {
  id: number;
  targetType: string;
  targetId: number;
  clusterId: number;
  action: string;
  status: string;
  message: string;
  createdAt: string;
}
export interface WebPlan {
  id: number;
  executable: boolean;
  idempotencyKey: string;
  expiresAt: string;
  checks: { name: string; passed: boolean; message: string }[];
}
export interface WebCapability {
  action: string;
  label: string;
  enabled: boolean;
  reason: string;
}
export function webRequest<T = any>(
  path: string,
  method = "GET",
  body?: unknown,
  signal?: AbortSignal
) {
  return platformRequest<T>(`/web${path}`, method, body, 180000, signal);
}
export function webQuery(filters: Record<string, any> = {}) {
  const query = new URLSearchParams();
  Object.entries(filters).forEach(([k, v]) => {
    if (v !== "" && v != null && v !== 0) query.set(k, String(v));
  });
  return query.toString();
}
export function webLogQuery(filters: Record<string, any> & { range?: Date[] }) {
  const { range, ...values } = filters;
  return webQuery({
    ...values,
    from: range?.[0]?.toISOString(),
    to: range?.[1]?.toISOString()
  });
}
export function webList(
  kind: string,
  filters: Record<string, any> = {},
  signal?: AbortSignal
) {
  return webRequest<{ list: WebResource[]; total: number }>(
    `/${kind}?${webQuery(filters)}`,
    "GET",
    undefined,
    signal
  );
}
export async function webOptions(kind: string, signal?: AbortSignal) {
  const rows: WebResource[] = [];
  let page = 1;
  for (;;) {
    const result = await webList(
      kind,
      { page, size: 100, view: "options" },
      signal
    );
    rows.push(...result.list);
    if (rows.length >= result.total || result.list.length === 0) return rows;
    page++;
  }
}
export interface WebConfigFile {
  path: string;
  content: string;
  hash?: string;
  mode?: number;
  group?: string;
  deleted?: boolean;
  editable?: boolean;
  included?: boolean;
}
export interface WebInstanceConfig {
  instanceId: number;
  name: string;
  clusterId: number;
  management: string;
  configPath: string;
  prefix: string;
  managedPaths: string[];
  version: string;
  digest: string;
  identity: string;
  collectedAt?: string;
  files: WebConfigFile[];
}
export interface WebConfigValidation {
  valid: boolean;
  message: string;
  instanceId: number;
  name: string;
  digest: string;
  checked: number;
}
export interface WebCertificateVersion {
  id: number;
  name: string;
  domains: string[];
  issuer: string;
  fingerprint: string;
  notBefore: string;
  notAfter: string;
  renewAt: string;
  days: number;
  current: boolean;
  createdAt: string;
}

export function webInstanceConfig(id: number, signal?: AbortSignal) {
  return webRequest<WebInstanceConfig>(
    `/instances/${id}/config`,
    "GET",
    undefined,
    signal
  );
}
export function webValidateInstanceConfig(id: number, files: WebConfigFile[]) {
  return webRequest<WebConfigValidation>(
    `/instances/${id}/config/validate`,
    "POST",
    { files }
  );
}
export interface WebConfigScope {
  instanceIds?: number[];
  baselines?: Record<number, string>;
  variables?: Record<number, Record<string, string>>;
}
export function webValidateClusterConfig(
  id: number,
  files: WebConfigFile[],
  scope: WebConfigScope = {}
) {
  return webRequest<WebConfigValidation[]>(
    `/clusters/${id}/config/validate`,
    "POST",
    { files, ...scope }
  );
}
export function webCertificateVersions(id: number) {
  return webRequest<{
    list: WebCertificateVersion[];
    currentVersionId: number;
  }>(`/certificates/${id}/versions`);
}
export function webExportCertificate(
  id: number,
  versionId: number,
  includeKey = false
) {
  return webRequest<Record<string, string>>(
    `/certificates/${id}/versions/${versionId}/export?key=${includeKey}`
  );
}

export interface WebDNSCredential {
  id: number;
  name: string;
  provider: string;
}
export interface WebDNSZone {
  id: string;
  name: string;
  status: string;
}
export interface WebDNSCheck {
  domain?: string;
  valid: boolean;
  checkedAt: string;
  checks: { name: string; passed: boolean; message: string }[];
  zone?: WebDNSZone;
  zonesCount: number;
  writeAccess: "unverified";
}
export interface WebDNSZones {
  zones: WebDNSZone[];
  total: number;
  truncated: boolean;
  checkedAt: string;
}
export function webDNSCredentials(provider: string, signal?: AbortSignal) {
  return webRequest<WebDNSCredential[]>(
    `/dns-credentials?${webQuery({ provider })}`,
    "GET",
    undefined,
    signal
  );
}
export function webCheckDNSAccount(
  id: number,
  domain = "",
  signal?: AbortSignal
) {
  return webRequest<WebDNSCheck>(
    `/dns-accounts/${id}/check`,
    "POST",
    { domain },
    signal
  );
}
export function webDNSZones(id: number, signal?: AbortSignal) {
  return webRequest<WebDNSZones>(
    `/dns-accounts/${id}/zones`,
    "GET",
    undefined,
    signal
  );
}

export interface WebClusterMemberChanges {
  add: number[];
  remove: number[];
}
export function webSaveCluster(
  cluster: Partial<WebResource>,
  memberChanges: WebClusterMemberChanges
) {
  return webRequest<WebResource>(
    `/clusters${cluster.id ? "/" + cluster.id : ""}`,
    cluster.id ? "PUT" : "POST",
    { ...cluster, memberChanges }
  );
}
