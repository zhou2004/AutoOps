import { http } from "@/utils/http";

export type UserResult = {
  code: number;
  data: {
    accessToken?: string;
    token?: string;
    access_token?: string;
    username?: string;
    userName?: string;
    nickname?: string;
    roles?: Array<string>;
    permissions?: Array<string>;
    sysAdmin?: any;
    [key: string]: any;
  };
  message: string;
};

/** 登录 */
export const getLogin = (data?: object) => {
  return http.request<UserResult>("post", "/login", { data });
};

/** 获取验证码 */
export const getCaptcha = () => {
  return http.request<any>("get", "/captcha");
};

/** 获取当前用户菜单 */
export const getMenuList = () => {
  return http.request<any>("get", "/menu/vo/list");
};

