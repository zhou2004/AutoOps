const Layout = () => import("@/layout/index.vue");

export default {
    path: "/tools",
    name: "Tools",
    component: Layout,
    redirect: "/tools/ops",
    meta: {
        icon: "ep/tools",
        title: "运维工具",
        rank: 60
    },
    children: [
        {
            path: "/tools/ops",
            name: "ToolsOps",
            component: () => import("@/views/Tools/Tools.vue"),
            meta: { title: "工具列表" }
        },
        {
            path: "/tools/agent",
            name: "ToolsAgent",
            component: () => import("@/views/Tools/Agent.vue"),
            meta: { title: "Agent列表" }
        }
    ]
};
