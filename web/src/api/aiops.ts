import { getToken } from "@/utils/auth";
import { localizeError, tr } from "@/locales";

export type ConfigKind = "model" | "skill" | "workflow";
export interface WorkflowStep {
  name: string;
  skillId: number;
  instructions: string;
}
export interface AIConfig {
  id: number;
  kind: ConfigKind;
  name: string;
  description: string;
  enabled: boolean;
  provider: string;
  baseUrl: string;
  model: string;
  hasKey: boolean;
  apiKey?: string;
  clearKey?: boolean;
  instructions: string;
  modules: string[];
  steps: WorkflowStep[];
  timeoutSeconds: number;
}
export interface Catalog {
  admin: boolean;
  harnessAvailable?: boolean;
  modules: string[];
  configs: AIConfig[];
}
export interface DiagnoseRequest {
  modelId: number;
  skillId: number;
  workflowId: number;
  prompt: string;
  modules: string[];
  days: number;
  locale: string;
}
export interface AIEvent {
  type: string;
  runId?: number;
  text?: string;
  step?: string;
  status?: string;
  data?: Record<string, unknown>;
}
export interface AIRun {
  id: number;
  modelId: number;
  skillId: number;
  workflowId: number;
  prompt: string;
  modules: string;
  locale: string;
  status: string;
  output: string;
  evidence: string;
  error: string;
  createdAt: string;
  finishedAt: string | null;
}
const base = `${(import.meta.env.VITE_API_BASE_URL || "").replace(/\/$/, "")}/api/v1/aiops`;
async function response(
  path: string,
  method = "GET",
  body?: unknown,
  signal?: AbortSignal
) {
  const res = await fetch(`${base}${path}`, {
    method,
    signal,
    headers: {
      Authorization: `Bearer ${getToken()}`,
      "Content-Type": "application/json"
    },
    ...(body === undefined ? {} : { body: JSON.stringify(body) })
  });
  if (!res.ok) throw new Error(`HTTP ${res.status}: ${tr("请求失败")}`);
  return res;
}
async function json<T>(
  path: string,
  method = "GET",
  body?: unknown,
  timeout = 20000
): Promise<T> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeout);
  try {
    const res = await response(path, method, body, controller.signal);
    const value = await res.json();
    if (value.code !== 200)
      throw new Error(
        localizeError(value.message || value.msg || tr("请求失败"))
      );
    return value.data;
  } finally {
    clearTimeout(timer);
  }
}
export const getCatalog = () => json<Catalog>("/catalog");
export const saveConfig = (value: AIConfig) =>
  json<AIConfig>(
    `/configs${value.id ? `/${value.id}` : ""}`,
    value.id ? "PUT" : "POST",
    value
  );
export const deleteConfig = (id: number) => json(`/configs/${id}`, "DELETE");
export const testModel = (id: number) =>
  json(`/models/${id}/test`, "POST", {}, 65000);
export const getRuns = (page: number) =>
  json<{ items: AIRun[]; total: number }>(`/runs?page=${page}`);
export const getRun = (id: number) => json<AIRun>(`/runs/${id}`);

// Incremental UTF-8 + SSE parsing; network chunks need not align with events or characters.
export async function diagnose(
  value: DiagnoseRequest,
  signal: AbortSignal,
  onEvent: (event: AIEvent) => void,
  path = "/diagnose"
) {
  const res = await response(path, "POST", value, signal);
  if (!res.headers.get("content-type")?.includes("text/event-stream")) {
    const body = await res.json();
    throw new Error(localizeError(body.message || body.msg || tr("诊断失败")));
  }
  const reader = res.body?.getReader();
  if (!reader) throw new Error(tr("响应意外中断，请重试"));
  const decoder = new TextDecoder();
  let buffer = "",
    complete = false;
  const dispatch = (block: string) => {
    const data = block
      .split("\n")
      .filter(line => line.startsWith("data:"))
      .map(line => line.slice(5).replace(/^ /, ""))
      .join("\n");
    if (!data) return;
    const event = JSON.parse(data) as AIEvent;
    if (event.type === "error")
      throw new Error(localizeError(event.text || tr("诊断失败")));
    if (event.type === "done") complete = true;
    onEvent(event);
  };
  try {
    while (!complete) {
      const { value: chunk, done } = await reader.read();
      buffer += done
        ? decoder.decode()
        : decoder.decode(chunk, { stream: true });
      // Normalize only complete lines, preserving a CR split across chunks.
      buffer = buffer.replace(/\r\n/g, "\n");
      let boundary: number;
      while ((boundary = buffer.indexOf("\n\n")) >= 0) {
        dispatch(buffer.slice(0, boundary));
        buffer = buffer.slice(boundary + 2);
      }
      if (buffer.length > 4 * 1024 * 1024) throw new Error(tr("诊断失败"));
      if (done) break;
    }
    if (!complete) throw new Error(tr("响应意外中断，请重试"));
  } finally {
    await reader.cancel().catch(() => {});
    reader.releaseLock();
  }
}
export const moduleLabels: Record<string, string> = {
  cmdb: "资产管理",
  k8s: "容器管理",
  monitor: "监控中心",
  task: "任务中心",
  application: "服务管理",
  configcenter: "配置中心概览",
  system: "系统概览",
  tools: "服务部署"
};
export function statusText(value: string) {
  return tr(
    (
      {
        running: "运行中",
        succeeded: "已成功",
        failed: "失败",
        cancelled: "已取消",
        interrupted: "已中断"
      } as Record<string, string>
    )[value] || value
  );
}

export interface AISession {
  id: number;
  title: string;
  modelId: number;
  skillId: number;
  targets: string;
  actions: string;
  grantUntil: string | null;
  busyUntil: string | null;
  updatedAt: string;
}
export interface AIMessage {
  id: number;
  sessionId: number;
  role: string;
  content: string;
  status: string;
  createdAt: string;
}
export interface AIToolCall {
  id: number;
  tool: string;
  arguments: string;
  result: string;
  status: string;
  createdAt: string;
}
export interface AITarget {
  id: number;
  name: string;
  assetType: string;
  assetId: number;
  canOperate: boolean;
}
export const getSessions = () => json<AISession[]>("/sessions");
export const getSession = (id: number) =>
  json<{ session: AISession; messages: AIMessage[]; calls: AIToolCall[] }>(
    `/sessions/${id}`
  );
export const createSession = (value: {
  title: string;
  modelId: number;
  skillId: number;
  targets: number[];
}) => json<AISession>("/sessions", "POST", value);
export const grantSession = (id: number, actions: string[], minutes: number) =>
  json<AISession>(`/sessions/${id}/grants`, "PUT", { actions, minutes });
export const getTargets = () => json<AITarget[]>("/targets");
export const chat = (
  id: number,
  prompt: string,
  signal: AbortSignal,
  onEvent: (event: AIEvent) => void
) =>
  diagnose(
    { prompt } as DiagnoseRequest,
    signal,
    onEvent,
    `/sessions/${id}/messages`
  );
