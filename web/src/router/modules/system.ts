const Layout = () => import("@/layout/index.vue");

export default {
    path: "/system",
    name: "System",
    component: Layout,
    redirect: "/system/admin",
    meta: {
        icon: "ep/setting",
        title: "基础管理",
        rank: 99
    },
    children: [
        {
            path: "/system/admin",
            name: "SystemAdmin",
            component: () => import("@/views/system/Admin.vue"),
            meta: { title: "用户信息" }
        },
        {
            path: "/system/role",
            name: "SystemRole",
            component: () => import("@/views/system/Role.vue"),
            meta: { title: "角色信息" }
        },
        {
            path: "/system/menu",
            name: "SystemMenu",
            component: () => import("@/views/system/Menu.vue"),
            meta: { title: "菜单信息" }
        },
        {
            path: "/system/dept",
            name: "SystemDept",
            component: () => import("@/views/system/Dept.vue"),
            meta: { title: "部门信息" }
        },
        {
            path: "/system/post",
            name: "SystemPost",
            component: () => import("@/views/system/Post.vue"),
            meta: { title: "岗位信息" }
        },
        {
            path: "/system/machine",
            name: "SystemMachine",
            component: () => import("@/views/system/Machine.vue"),
            meta: { title: "机房信息" }
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
            component: () => import("@/views/cmdb/assetPermission.vue"),
            meta: { title: "资产授权" }
        },
        {
            path: "/system/k8s-permission",
            name: "SystemK8sPermission",
            component: () => import("@/views/K8s/k8s-permission.vue"),
            meta: { title: "容器授权" }
        },
        {
            path: "/monitor/loginlog",
            name: "MonitorLoginLog",
            component: () => import("@/views/monitor/LoginLog.vue"),
            meta: { title: "登录日志" }
        },
        {
            path: "/monitor/operator",
            name: "MonitorOperator",
            component: () => import("@/views/monitor/Operator.vue"),
            meta: { title: "操作日志" }
        },
        {
            path: "/monitor/dblog",
            name: "MonitorDbLog",
            component: () => import("@/views/monitor/DBLog.vue"),
            meta: { title: "数据日志" }
        }
    ]
};
