const Layout = () => import("@/layout/index.vue");
export default {
  path: "/aiops",
  name: "AIOps",
  component: Layout,
  redirect: "/aiops/diagnose",
  meta: { title: "AI运维", icon: "ri/robot-2-line", rank: 1 },
  children: [
    {
      path: "/aiops/workflow-run",
      name: "AIWorkflowRun",
      component: () => import("@/views/aiops/workflow-run.vue"),
      meta: { keepAlive: false, title: "工作流诊断", showLink: false }
    },
    {
      path: "/aiops/diagnose",
      name: "AIDiagnose",
      component: () => import("@/views/aiops/diagnose.vue"),
      meta: { keepAlive: false, title: "诊断工作台" }
    },
    {
      path: "/aiops/models",
      name: "AIModels",
      component: () => import("@/views/aiops/configs.vue"),
      meta: { keepAlive: false, title: "模型配置" }
    },
    {
      path: "/aiops/skills",
      name: "AISkills",
      component: () => import("@/views/aiops/configs.vue"),
      meta: { keepAlive: false, title: "Skills配置" }
    },
    {
      path: "/aiops/workflows",
      name: "AIWorkflows",
      component: () => import("@/views/aiops/configs.vue"),
      meta: { keepAlive: false, title: "工作流配置" }
    },
    {
      path: "/aiops/history",
      name: "AIHistory",
      component: () => import("@/views/aiops/history.vue"),
      meta: { keepAlive: false, title: "诊断记录" }
    }
  ]
} satisfies RouteConfigsTable;
