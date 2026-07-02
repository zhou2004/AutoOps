const Layout = () => import("@/layout/index.vue");

export default {
    path: "/config",
    name: "ConfigCenter",
    component: Layout,
    redirect: "/config/ecskey",
    meta: {
        icon: "ep/set-up",
        title: "配置中心",
        rank: 50
    },
    children: [
        {
            path: "/config/ecskey",
            name: "ConfigEcsKey",
            component: () => import("@/views/configcenter/ecs-key.vue"),
            meta: { title: "主机凭据" }
        },
        {
            path: "/config/accountauth",
            name: "ConfigAccountAuth",
            component: () => import("@/views/configcenter/accountauth.vue"),
            meta: { title: "通用凭据" }
        },
        {
            path: "/config/keymanage",
            name: "ConfigKeyManage",
            component: () => import("@/views/configcenter/KeyManage.vue"),
            meta: { title: "密钥管理" }
        }
    ]
};
