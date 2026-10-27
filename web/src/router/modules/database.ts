const Layout = () => import("@/layout/index.vue");
export default {
  path: "/database",
  name: "DatabaseManagement",
  component: Layout,
  redirect: "/database/sources",
  meta: { icon: "ep/coin", title: "数据库管理", rank: 11 },
  children: [
    {
      path: "/database/sources",
      name: "DatabaseSources",
      component: () => import("@/views/cmdb/cmdbDB.vue"),
      meta: { title: "数据源管理", showParent: true }
    },
    {
      path: "/database/workbench",
      name: "DatabaseWorkbench",
      component: () => import("@/views/database/Workbench.vue"),
      meta: { title: "数据库工作台", showLink: false }
    }
  ]
};
