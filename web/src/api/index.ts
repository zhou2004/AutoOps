/**
 * AutoOps API Plugin
 * 将各模块 API 注入为全局属性 $api
 */
import type { App } from "vue";
import systemApi from "@/api/system";
import cmdbApi from "@/api/cmdb";
import dashboardApi from "@/api/dashboard";
import * as taskApi from "@/api/task";
import * as k8sApi from "@/api/k8s";
import * as monitorApi from "@/api/monitor";
import * as appApi from "@/api/app";
import * as toolApi from "@/api/tool";
import * as configApi from "@/api/config";
import { getCaptcha, getLogin, getMenuList } from "@/api/user";

const api = {
    // 基础 API
    captcha: getCaptcha,
    login: getLogin,
    // 系统管理
    ...systemApi,
    // CMDB
    ...cmdbApi,
    // Dashboard
    ...dashboardApi,
    // 任务管理
    ...taskApi,
    // K8s 管理
    ...k8sApi,
    // 监控管理
    ...monitorApi,
    // 应用管理
    ...appApi,
    // 运维工具
    ...toolApi,
    // 配置中心
    ...configApi,
    // 菜单
    querySysMenuVoList: getMenuList,
};

export default {
    install(app: App) {
        app.config.globalProperties.$api = api;
    }
};

export { api };
