import { menuIcon } from "@/utils/menuIcons";
const Layout = () => import("@/layout/index.vue");

export default {
  path: "/system",
  name: "System",
  component: Layout,
  redirect: "/system/admin",
  meta: {
    icon: menuIcon("/system"),
    title: "基础管理",
    rank: 99
  },
  children: [
    {
      path: "/system/authorization",
      name: "SystemAuthorization",
      component: () => import("@/views/system/AuthorizationManagement.vue"),
      meta: { title: "授权管理", icon: menuIcon("/system/authorization") }
    },
    {
      path: "/system/role-permission",
      name: "SystemRolePermission",
      redirect: { path: "/system/api-keys", query: {} },
      meta: { title: "API Key", showLink: false }
    },
    {
      path: "/system/api-keys",
      name: "SystemAPIKeys",
      component: () => import("@/views/system/APIKeys.vue"),
      meta: { title: "API Key", icon: menuIcon("/system/api-keys") }
    },
    {
      path: "/system/admin",
      name: "SystemAdmin",
      component: () => import("@/views/system/Admin.vue"),
      meta: { icon: menuIcon("/system/admin"), title: "用户信息" }
    },
    {
      path: "/system/role",
      name: "SystemRole",
      component: () => import("@/views/system/Role.vue"),
      meta: { icon: menuIcon("/system/role"), title: "角色信息" }
    },
    {
      path: "/system/menu",
      name: "SystemMenu",
      component: () => import("@/views/system/Menu.vue"),
      meta: { icon: menuIcon("/system/menu"), title: "菜单信息" }
    },
    {
      path: "/system/dept",
      name: "SystemDept",
      component: () => import("@/views/system/Dept.vue"),
      meta: { icon: menuIcon("/system/dept"), title: "部门信息" }
    },
    {
      path: "/system/post",
      name: "SystemPost",
      component: () => import("@/views/system/Post.vue"),
      meta: { icon: menuIcon("/system/post"), title: "岗位信息" }
    },
    {
      path: "/system/machine",
      name: "SystemMachine",
      component: () => import("@/views/system/Machine.vue"),
      meta: { icon: menuIcon("/system/machine"), title: "机房信息" }
    },
    {
      path: "/system/personal",
      name: "SystemPersonal",
      component: () => import("@/views/system/Personal.vue"),
      meta: { title: "个人信息", showLink: false }
    },
    {
      path: "/system/asset-permission",
      name: "SystemAssetPermission",
      redirect: { path: "/system/authorization", query: { tab: "assets" } },
      meta: { title: "资产授权", showLink: false }
    },
    {
      path: "/system/k8s-permission",
      name: "SystemK8sPermission",
      redirect: { path: "/system/authorization", query: { tab: "containers" } },
      meta: { title: "容器授权", showLink: false }
    },
    {
      path: "/monitor/loginlog",
      name: "MonitorLoginLog",
      component: () => import("@/views/monitor/LoginLog.vue"),
      meta: { icon: menuIcon("/monitor/loginlog"), title: "登录日志" }
    },
    {
      path: "/monitor/operator",
      name: "MonitorOperator",
      component: () => import("@/views/monitor/Operator.vue"),
      meta: { icon: menuIcon("/monitor/operator"), title: "操作日志" }
    },
    {
      path: "/monitor/dblog",
      name: "MonitorDbLog",
      component: () => import("@/views/monitor/DBLog.vue"),
      meta: { icon: menuIcon("/monitor/dblog"), title: "数据日志" }
    }
  ]
};
