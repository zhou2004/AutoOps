import { http } from "@/utils/http";

type Result = {
  code: number;
  data: Array<any>;
};

/** 获取后端动态路由（菜单） */
export const getAsyncRoutes = () => {
  return http.request<Result>("get", "/menu/vo/list");
};
