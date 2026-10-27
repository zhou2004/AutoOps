import {
  platformRequest,
  type PlatformIdentity,
  type PermissionModule
} from "./platformPermissions";
export interface PlatformKey {
  id: number;
  userId: number;
  issuedBy: number;
  username: string;
  nickname: string;
  name: string;
  prefix: string;
  scopes: string;
  grantMode: "" | "scoped";
  expiresAt: string;
  revokedAt: string | null;
  lastUsedAt: string | null;
  createdAt: string;
}
export interface KeyUser {
  id: number;
  username: string;
  nickname: string;
  status: number;
}
const base = "/system/api-keys";
export const listAPIKeys = (page = 1, userId?: number) =>
  platformRequest<{
    items: PlatformKey[];
    total: number;
    pageSize: number;
    canManage: boolean;
    permissions: string[];
    modules: PermissionModule[];
  }>(`${base}?page=${page}${userId ? `&userId=${userId}` : ""}`);
export const createAPIKey = (value: {
  userId: number;
  name: string;
  days: number;
  scopes: string[];
}) => platformRequest<{ key: PlatformKey; token: string }>(base, "POST", value);
export const revokeAPIKey = (id: number) =>
  platformRequest(`${base}/${id}`, "DELETE");
export const keyUsers = (q = "") =>
  platformRequest<KeyUser[]>(`${base}/users?q=${encodeURIComponent(q)}`);
export const keyUserPermissions = (id: number) =>
  platformRequest<PlatformIdentity>(`${base}/users/${id}/permissions`);

export const keyPermissionCatalog = () =>
  platformRequest<{ modules: PermissionModule[] }>(`${base}/catalog`);
