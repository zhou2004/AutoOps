import { menuIcon } from "@/utils/menuIcons";
const Layout = () => import("@/layout/index.vue");

export default {
  path: "/",
  name: "Home",
  component: Layout,
  redirect: "/dashboard",
  meta: {
    icon: menuIcon("/"),
    title: "首页",
    rank: 0
  },
  children: [
    {
      path: "/dashboard",
      name: "Dashboard",
      component: () => import("@/views/dashboard/index.vue"),
      meta: {
        title: "仪表盘",
        icon: menuIcon("/dashboard")
      }
    }
  ]
} satisfies RouteConfigsTable;
