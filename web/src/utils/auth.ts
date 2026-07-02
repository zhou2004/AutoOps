import { storageLocal } from "@pureadmin/utils";
import Cookies from "js-cookie";

export interface DataInfo<T> {
  /** token */
  accessToken?: string;
  /** token (兼容旧格式) */
  token?: string;
  /** 用户名 */
  username?: string;
  /** 昵称 */
  nickname?: string;
  /** 当前登录用户的角色 */
  roles?: Array<string>;
  /** 按钮级别权限 */
  permissions?: Array<string>;
  /** 完整用户数据 */
  sysAdmin?: any;
}

export const userKey = "user-info";
export const TokenKey = "authorized-token";
export const multipleTabsKey = "multiple-tabs";

/** 获取`token` */
export function getToken(): string {
  return storageLocal().getItem<string>(TokenKey) || "";
}

/** 格式化token为Bearer格式 */
export function formatToken(token: string): string {
  return "Bearer " + token;
}

/** 判断是否已登录 */
export function isLoggedIn(): boolean {
  const token = getToken();
  const userInfo = getUserInfo();
  return !!(token && userInfo?.username);
}

/**
 * @description 设置`token`以及用户信息到localStorage，并设置登录状态Cookie
 */
export function setToken(data: any) {
  const token = data.accessToken || data.token || data.access_token;
  if (token) {
    storageLocal().setItem(TokenKey, token);
  }
  // 存储完整用户信息
  const userInfo = {
    username: data.username || data.userName || "",
    nickname: data.nickname || data.nickName || "",
    roles: data.roles || [],
    permissions: data.permissions || [],
    sysAdmin: data,
    avatar: data.avatar || ""
  };
  storageLocal().setItem(userKey, userInfo);
  // 设置登录状态标记（路由守卫依赖此判断）
  Cookies.set(multipleTabsKey, "true");
  return userInfo;
}

/** 删除`token`和用户信息 */
export function removeToken() {
  storageLocal().removeItem(TokenKey);
  storageLocal().removeItem(userKey);
  Cookies.remove(multipleTabsKey);
}

/** 获取用户信息 */
export function getUserInfo(): any {
  return storageLocal().getItem(userKey) || {};
}

/** 是否有按钮级别的权限 */
export const hasPerms = (value: string | Array<string>): boolean => {
  if (!value) return false;
  const allPerms = "*:*:*";
  const info = getUserInfo();
  const permissions = info?.permissions || [];
  if (!permissions || permissions.length === 0) return false;
  if (permissions.length === 1 && permissions[0] === allPerms) return true;
  if (typeof value === 'string') return permissions.includes(value);
  return value.every(v => permissions.includes(v));
};
