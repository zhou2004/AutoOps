import { menuIcon } from "@/utils/menuIcons";
const Layout = () => import("@/layout/index.vue");

export default {
  path: "/tools",
  name: "Tools",
  component: Layout,
  redirect: "/tools/ops",
  meta: {
    icon: menuIcon("/tools"),
    title: "运维工具",
    rank: 60
  },
  children: [
    {
      path: "/tools/ops",
      name: "ToolsOps",
      component: () => import("@/views/Tools/Tools.vue"),
      meta: { icon: menuIcon("/tools/ops"), title: "工具列表" }
    },
    {
      path: "/tools/agent",
      name: "ToolsAgent",
      component: () => import("@/views/Tools/Agent.vue"),
      meta: { icon: menuIcon("/tools/agent"), title: "Agent列表" }
    }
  ]
};
