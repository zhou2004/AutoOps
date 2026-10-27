import { getToken, removeToken } from "@/utils/auth";
export interface PlatformIdentity {
  userId: number;
  username: string;
  isAdmin: boolean;
  permissions: string[];
}
export interface PermissionModule {
  key: string;
  name: string;
  actions: string[];
}
export interface PlatformRole {
  id: number;
  roleName: string;
  roleKey: string;
  status: number;
}
const base = `${(import.meta.env.VITE_API_BASE_URL || "").replace(/\/$/, "")}/api/v1`;
export async function platformRequest<T>(
  path: string,
  method = "GET",
  body?: unknown
): Promise<T> {
  const res = await fetch(`${base}${path}`, {
    method,
    signal: AbortSignal.timeout(20000),
    headers: {
      Authorization: `Bearer ${getToken()}`,
      "Content-Type": "application/json"
    },
    ...(body === undefined ? {} : { body: JSON.stringify(body) })
  });
  if (res.status === 401) removeToken();
  const data = await res.json();
  if (!res.ok || data.code !== 200)
    throw new Error(data.message || data.msg || "请求失败");
  return data.data;
}
export const myPermissions = () =>
  platformRequest<PlatformIdentity>("/system/permissions/me");
export const permissionCatalog = () =>
  platformRequest<{ modules: PermissionModule[]; roles: PlatformRole[] }>(
    "/system/permissions/catalog"
  );
export const rolePermissions = (id: number) =>
  platformRequest<{ permissions: string[]; legacy: boolean; isAdmin: boolean }>(
    `/system/permissions/roles/${id}`
  );
export const saveRolePermissions = (id: number, permissions: string[]) =>
  platformRequest(`/system/permissions/roles/${id}`, "PUT", { permissions });
export const actionLabels: Record<string, string> = {
  read: "查看",
  write: "新增 / 修改 / 删除",
  execute: "执行操作",
  chat: "诊断 / 对话",
  operate: "授权 / 执行工具"
};
