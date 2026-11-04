import { menuIcon } from "@/utils/menuIcons";
const Layout = () => import("@/layout/index.vue");

export default {
  path: "/cmdb",
  name: "Cmdb",
  component: Layout,
  redirect: "/cmdb/ecs",
  meta: {
    icon: menuIcon("/cmdb"),
    title: "资产管理",
    rank: 10
  },
  children: [
    {
      path: "/cmdb/remote",
      name: "CmdbRemote",
      component: () => import("@/views/cmdb/RemoteSession.vue"),
      meta: {
        title: "远程会话",
        showLink: false,
        keepAlive: false,
        uiLayout: "canvas"
      }
    },
    {
      path: "/cmdb/ecs",
      name: "CmdbHost",
      component: () => import("@/views/cmdb/cmdbHost.vue"),
      meta: { icon: menuIcon("/cmdb/ecs"), title: "主机管理" }
    },
    {
      path: "/cmdb/group",
      name: "CmdbGroup",
      component: () => import("@/views/cmdb/cmdbGroup.vue"),
      meta: { icon: menuIcon("/cmdb/group"), title: "业务分组" }
    },
    {
      path: "/cmdb/db",
      name: "CmdbDb",
      redirect: to => ({ path: "/database/sources", query: to.query }),
      meta: { title: "数据源管理", showLink: false }
    },
    {
      path: "/cmdb/docker",
      name: "CmdbDocker",
      redirect: to => ({ path: "/k8s/docker", query: to.query }),
      meta: { title: "Docker管理", showLink: false }
    },
    {
      path: "/cmdb/docker/terminal/:assetType/:assetId/:container",
      name: "CmdbDockerTerminal",
      redirect: to => ({
        name: "K8sDockerTerminal",
        params: to.params,
        query: to.query
      }),
      meta: { title: "Docker容器终端", showLink: false }
    },
    {
      path: "/cmdb/ssh",
      name: "CmdbSsh",
      component: () => import("@/views/cmdb/Host/SSH.vue"),
      meta: { title: "终端登录", showLink: false, uiLayout: "canvas" }
    },
    {
      path: "/cmdb/dbdetails",
      name: "CmdbDbDetails",
      redirect: to => ({ path: "/database/workbench", query: to.query }),
      meta: { title: "数据库工作台", showLink: false }
    },
    {
      path: "/cmdb/physical",
      name: "CmdbPhysical",
      component: () => import("@/views/cmdb/physicalMachine.vue"),
      meta: { icon: menuIcon("/cmdb/physical"), title: "物理机管理" }
    },
    {
      path: "/cmdb/network",
      name: "CmdbNetwork",
      component: () => import("@/views/cmdb/networkDevice.vue"),
      meta: { icon: menuIcon("/cmdb/network"), title: "网络设备管理" }
    },
    {
      path: "/cmdb/asset-permission",
      name: "CmdbAssetPermission",
      redirect: { path: "/system/authorization", query: { tab: "assets" } },
      meta: { title: "资产授权", showLink: false }
    },
    {
      path: "/cmdb/user-group",
      name: "CmdbUserGroup",
      redirect: { path: "/system/authorization", query: { tab: "groups" } },
      meta: { title: "用户组管理", showLink: false }
    },
    {
      path: "/cmdb/credential-permission",
      name: "CmdbCredentialPermission",
      redirect: { path: "/system/authorization", query: { tab: "assets" } },
      meta: { title: "授权管理", showLink: false }
    },
    {
      path: "/cmdb/my-assets",
      name: "CmdbMyAssets",
      component: () => import("@/views/cmdb/myAssetPermissions.vue"),
      meta: { icon: menuIcon("/cmdb/my-assets"), title: "我的授权资产" }
    }
  ]
};
