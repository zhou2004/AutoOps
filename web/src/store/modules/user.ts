import { defineStore } from "pinia";
import {
  type userType,
  store,
  router,
  resetRouter,
  routerArrays,
  storageLocal
} from "../utils";
import {
  type UserResult,
  getLogin,
  getMenuList
} from "@/api/user";
import { useMultiTagsStoreHook } from "./multiTags";
import { type DataInfo, setToken, removeToken, userKey } from "@/utils/auth";

export const useUserStore = defineStore("pure-user", {
  state: (): userType => ({
    avatar: storageLocal().getItem<DataInfo<number>>(userKey)?.avatar ?? "",
    username: storageLocal().getItem<DataInfo<number>>(userKey)?.username ?? "",
    nickname: storageLocal().getItem<DataInfo<number>>(userKey)?.nickname ?? "",
    roles: storageLocal().getItem<DataInfo<number>>(userKey)?.roles ?? [],
    permissions: storageLocal().getItem<DataInfo<number>>(userKey)?.permissions ?? [],
    isRemembered: false,
    loginDay: 7,
    sysAdmin: storageLocal().getItem<DataInfo<number>>(userKey)?.sysAdmin ?? null,
    leftMenuList: storageLocal().getItem<DataInfo<number>>(userKey)?.leftMenuList ?? []
  }),
  actions: {
    SET_AVATAR(avatar: string) { this.avatar = avatar; },
    SET_USERNAME(username: string) { this.username = username; },
    SET_NICKNAME(nickname: string) { this.nickname = nickname; },
    SET_ROLES(roles: Array<string>) { this.roles = roles; },
    SET_PERMS(permissions: Array<string>) { this.permissions = permissions; },
    SET_ISREMEMBERED(bool: boolean) { this.isRemembered = bool; },
    SET_LOGINDAY(value: number) { this.loginDay = Number(value); },
    SET_SYSADMIN(data: any) { this.sysAdmin = data; },
    SET_LEFT_MENU_LIST(data: any[]) { this.leftMenuList = data; },

    /** 登录 - 适配 Go 后端
     *  后端 LoginDto: { username, password, image(验证码), idKey(验证码id) }
     *  后端返回: { code, data: { token, sysAdmin, leftMenuList, permissionList } }
     */
    async loginByUsername(data) {
      return new Promise<UserResult>((resolve, reject) => {
        getLogin(data)
          .then((res: any) => {
            if (res?.code === 200 && res?.data) {
              const resData = res.data;
              // Go 后端返回 token 字段（不是 accessToken）
              const token = resData.token || resData.accessToken || resData.access_token;
              if (token) {
                const tokenPayload = {
                  accessToken: token,
                  token: token,
                  ...resData.sysAdmin,
                  permissions: resData.permissionList || [],
                  roles: resData.sysAdmin?.roleIds || []
                };
                setToken(tokenPayload);
              }
              // 存储用户信息
              if (resData.sysAdmin) {
                this.SET_USERNAME(resData.sysAdmin.username || resData.sysAdmin.userName || "");
                this.SET_NICKNAME(resData.sysAdmin.nickname || resData.sysAdmin.nickName || "");
                this.SET_SYSADMIN(resData.sysAdmin);
              }
              if (resData.permissionList) {
                this.SET_PERMS(resData.permissionList);
              }
              // 存储菜单列表（后端登录接口直接返回）
              if (Array.isArray(resData.leftMenuList)) {
                this.SET_LEFT_MENU_LIST(resData.leftMenuList);
              }
            }
            resolve(res);
          })
          .catch(error => reject(error));
      });
    },

    /** 获取用户菜单 */
    async fetchMenuList() {
      try {
        const res: any = await getMenuList();
        if (res?.code === 200 && Array.isArray(res?.data)) {
          this.SET_LEFT_MENU_LIST(res.data);
          return res.data;
        }
        return [];
      } catch (err) {
        console.error("获取菜单失败:", err);
        return [];
      }
    },

    /** 登出 */
    logOut() {
      this.username = "";
      this.roles = [];
      this.permissions = [];
      this.sysAdmin = null;
      this.leftMenuList = [];
      removeToken();
      useMultiTagsStoreHook().handleTags("equal", [...routerArrays]);
      resetRouter();
      router.push("/login");
    }
  }
});

export function useUserStoreHook() {
  return useUserStore(store);
}
