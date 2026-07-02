const Layout = () => import("@/layout/index.vue");

export default {
    path: "/app",
    name: "App",
    component: Layout,
    redirect: "/app/application",
    meta: {
        icon: "ep/collection",
        title: "服务管理",
        rank: 70
    },
    children: [
        {
            path: "/app/application",
            name: "AppApplication",
            component: () => import("@/views/app/application.vue"),
            meta: { title: "应用列表" }
        },
        {
            path: "/app/quick-release",
            name: "AppQuickRelease",
            component: () => import("@/views/app/app_quick_release.vue"),
            meta: { title: "快速发布" }
        },
        {
            path: "/app/quick-temp/:id",
            name: "AppQuickTemp",
            component: () => import("@/views/app/app_quick_temp.vue"),
            meta: { title: "发布模板", showLink: false }
        }
    ]
};
