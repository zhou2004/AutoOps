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
      path: "/database/clusters",
      name: "DatabaseClusters",
      component: () => import("@/views/database/Management.vue"),
      meta: { title: "集群管理", showParent: true, databaseSection: "clusters" }
    },
    {
      path: "/database/storage",
      name: "DatabaseStorage",
      component: () => import("@/views/database/Management.vue"),
      meta: {
        icon: "ep/files",
        title: "存储管理",
        showParent: true,
        databaseSection: "storage"
      }
    },
    {
      path: "/database/backups",
      name: "DatabaseBackups",
      component: () => import("@/views/database/Management.vue"),
      meta: {
        icon: "ep/refresh-left",
        title: "备份恢复",
        showParent: true,
        databaseSection: "backups"
      }
    },
    {
      path: "/database/configurations",
      name: "DatabaseConfigurations",
      redirect: to => ({
        path: "/database/clusters",
        query: { ...to.query, detail: "parameters" }
      }),
      meta: { title: "集群参数", showLink: false }
    },
    {
      path: "/database/audit",
      name: "DatabaseAudit",
      component: () => import("@/views/database/Management.vue"),
      meta: { title: "操作审计", showParent: true, databaseSection: "audit" }
    },

    {
      path: "/database/workbench",
      name: "DatabaseWorkbench",
      component: () => import("@/views/database/Workbench.vue"),
      meta: { title: "数据库工作台", showLink: false }
    }
  ]
};
