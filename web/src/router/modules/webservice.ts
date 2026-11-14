import Globe from "~icons/ep/monitor";
import Connection from "~icons/ep/connection";
import Cluster from "~icons/ep/set-up";
import Document from "~icons/ep/document";
import Lock from "~icons/ep/lock";
import Tickets from "~icons/ep/tickets";
import TrendCharts from "~icons/ep/trend-charts";
const Layout = () => import("@/layout/index.vue");
const sections = [
  ["sources", "数据源管理", Connection],
  ["clusters", "集群管理", Cluster],
  ["sites", "站点管理", Globe],
  ["configurations", "配置管理", Document],
  ["certificates", "证书管理", Lock],
  ["logs", "日志管理", Tickets],
  ["traffic", "流量监控", TrendCharts]
] as const;
export default {
  path: "/web",
  name: "WebServiceManagement",
  component: Layout,
  redirect: "/web/sources",
  meta: { title: "Web 管理", icon: Globe, rank: 12 },
  children: sections.map(([section, title, icon], i) => ({
    path: `/web/${section}`,
    name: `WebService${section}`,
    component:
      section === "traffic"
        ? () => import("@/views/webservice/Traffic.vue")
        : () => import("@/views/webservice/Management.vue"),
    meta: { title, icon, rank: i + 1, showParent: true, webSection: section }
  }))
};
