import Coin from "~icons/ep/coin";
import Connection from "~icons/ep/connection";
import Files from "~icons/ep/files";
import RefreshLeft from "~icons/ep/refresh-left";
const Layout = () => import("@/layout/index.vue");
export default {
  path: "/database",
  name: "DatabaseManagement",
  component: Layout,
  redirect: "/database/sources",
  meta: { icon: Coin, title: "数据库管理", rank: 11 },
  children: [
    {
      path: "/database/sources",
      name: "DatabaseSources",
      component: () => import("@/views/cmdb/cmdbDB.vue"),
      meta: { icon: Coin, title: "数据源管理", rank: 1, showParent: true }
    },
    {
      path: "/database/clusters",
      name: "DatabaseClusters",
      component: () => import("@/views/database/Management.vue"),
      meta: {
        icon: Connection,
        title: "集群管理",
        rank: 2,
        showParent: true,
        databaseSection: "clusters"
      }
    },
    {
      path: "/database/storage",
      name: "DatabaseStorage",
      component: () => import("@/views/database/Management.vue"),
      meta: {
        icon: Files,
        rank: 3,
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
        icon: RefreshLeft,
        rank: 4,
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
      redirect: to => ({
        path: "/monitor/dblog",
        query: { ...to.query, tab: "database-audit" }
      }),
      meta: { title: "数据库操作审计", showLink: false }
    },

    {
      path: "/database/workbench",
      name: "DatabaseWorkbench",
      component: () => import("@/views/database/Workbench.vue"),
      meta: { title: "数据库工作台", showLink: false }
    }
  ]
};
