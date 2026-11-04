import { menuIcon } from "@/utils/menuIcons";
const Layout = () => import("@/layout/index.vue");

export default {
  path: "/config",
  name: "ConfigCenter",
  component: Layout,
  redirect: "/config/ecskey",
  meta: {
    icon: menuIcon("/config"),
    title: "配置中心",
    rank: 50
  },
  children: [
    {
      path: "/config/ecskey",
      name: "ConfigEcsKey",
      component: () => import("@/views/configcenter/ecs-key.vue"),
      meta: { icon: menuIcon("/config/ecskey"), title: "主机凭据" }
    },
    {
      path: "/config/accountauth",
      name: "ConfigAccountAuth",
      component: () => import("@/views/configcenter/accountauth.vue"),
      meta: { icon: menuIcon("/config/accountauth"), title: "通用凭据" }
    },
    {
      path: "/config/keymanage",
      name: "ConfigKeyManage",
      component: () => import("@/views/configcenter/KeyManage.vue"),
      meta: { icon: menuIcon("/config/keymanage"), title: "密钥管理" }
    }
  ]
};
