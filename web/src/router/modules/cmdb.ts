const Layout = () => import("@/layout/index.vue");

export default {
    path: "/cmdb",
    name: "Cmdb",
    component: Layout,
    redirect: "/cmdb/ecs",
    meta: {
        icon: "ep/monitor",
        title: "资产管理",
        rank: 10
    },
    children: [
        {
            path: "/cmdb/ecs",
            name: "CmdbHost",
            component: () => import("@/views/cmdb/cmdbHost.vue"),
            meta: { title: "主机管理" }
        },
        {
            path: "/cmdb/group",
            name: "CmdbGroup",
            component: () => import("@/views/cmdb/cmdbGroup.vue"),
            meta: { title: "业务分组" }
        },
        {
            path: "/cmdb/db",
            name: "CmdbDb",
            component: () => import("@/views/cmdb/cmdbDB.vue"),
            meta: { title: "数据管理" }
        },
        {
            path: "/cmdb/ssh",
            name: "CmdbSsh",
            component: () => import("@/views/cmdb/Host/SSH.vue"),
            meta: { title: "终端登录", showLink: false }
        },
        {
            path: "/cmdb/dbdetails",
            name: "CmdbDbDetails",
            component: () => import("@/views/cmdb/DBdetails.vue"),
            meta: { title: "数据库操作", showLink: false }
        },
        {
            path: "/cmdb/physical",
            name: "CmdbPhysical",
            component: () => import("@/views/cmdb/physicalMachine.vue"),
            meta: { title: "物理机管理" }
        },
        {
            path: "/cmdb/network",
            name: "CmdbNetwork",
            component: () => import("@/views/cmdb/networkDevice.vue"),
            meta: { title: "网络设备管理" }
        },
        {
            path: "/cmdb/asset-permission",
            name: "CmdbAssetPermission",
            component: () => import("@/views/cmdb/assetPermission.vue"),
            meta: { title: "资产授权" }
        },
        {
            path: "/cmdb/user-group",
            name: "CmdbUserGroup",
            component: () => import("@/views/cmdb/cmdbUserGroup.vue"),
            meta: { title: "用户组管理" }
        },
        {
            path: "/cmdb/credential-permission",
            name: "CmdbCredentialPermission",
            component: () => import("@/views/cmdb/credentialPermission.vue"),
            meta: { title: "凭据授权" }
        },
        {
            path: "/cmdb/my-assets",
            name: "CmdbMyAssets",
            component: () => import("@/views/cmdb/myAssetPermissions.vue"),
            meta: { title: "我的授权资产" }
        }
    ]
};
