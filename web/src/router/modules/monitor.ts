import { menuIcon } from "@/utils/menuIcons";
const Layout = () => import("@/layout/index.vue");

export default {
  path: "/monitor",
  name: "Monitor",
  component: Layout,
  redirect: "/monitor/base",
  meta: {
    icon: menuIcon("/monitor"),
    title: "监控中心",
    rank: 40
  },
  children: [
    {
      path: "/monitor/domain",
      name: "MonitorDomain",
      component: () => import("@/views/monitor/DomainCert.vue"),
      meta: { icon: menuIcon("/monitor/domain"), title: "域名证书监控" }
    },
    {
      path: "/monitor/api-endpoint",
      name: "MonitorApiEndpoint",
      component: () => import("@/views/monitor/APIEndpoint.vue"),
      meta: { icon: menuIcon("/monitor/api-endpoint"), title: "API监控" }
    },
    {
      path: "/monitor/alarm/rules",
      name: "MonitorAlarmRules",
      component: () => import("@/views/monitor/Alarm-rules.vue"),
      meta: { icon: menuIcon("/monitor/alarm/rules"), title: "告警配置" }
    },
    {
      path: "/monitor/alarm/notify",
      name: "MonitorAlarmNotify",
      component: () => import("@/views/monitor/Alarm-notify.vue"),
      meta: { icon: menuIcon("/monitor/alarm/notify"), title: "告警通知" }
    },
    {
      path: "/monitor/alarm/history",
      name: "MonitorAlarmHistory",
      component: () => import("@/views/monitor/alarm-history.vue"),
      meta: { title: "告警历史", showLink: false }
    },
    {
      path: "/monitor/datasource",
      name: "MonitorDatasource",
      component: () => import("@/views/monitor/DataSource.vue"),
      meta: { icon: menuIcon("/monitor/datasource"), title: "数据源管理" }
    },
    {
      path: "/monitor/incident",
      name: "MonitorIncident",
      component: () => import("@/views/monitor/Incident.vue"),
      meta: { icon: menuIcon("/monitor/incident"), title: "故障管理" }
    }
  ]
};
