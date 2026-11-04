import { menuIcon } from "@/utils/menuIcons";
const Layout = () => import("@/layout/index.vue");

export default {
  path: "/task",
  name: "Task",
  component: Layout,
  redirect: "/task/template",
  meta: {
    icon: menuIcon("/task"),
    title: "任务中心",
    rank: 30
  },
  children: [
    {
      path: "/task/template",
      name: "TaskTemplate",
      component: () => import("@/views/task/TaskTemplate.vue"),
      meta: { icon: menuIcon("/task/template"), title: "任务模版" }
    },
    {
      path: "/task/job",
      name: "TaskJob",
      component: () => import("@/views/task/TaskJob.vue"),
      meta: { icon: menuIcon("/task/job"), title: "任务作业" }
    },
    {
      path: "/task/ansible",
      name: "TaskAnsible",
      component: () => import("@/views/task/TaskAnsible.vue"),
      meta: { icon: menuIcon("/task/ansible"), title: "Ansible任务" }
    },
    {
      path: "/task/ansible/history",
      name: "AnsibleTaskHistory",
      component: () => import("@/views/task/AnsibleTaskHistory.vue"),
      meta: {
        title: "Ansible执行记录",
        showLink: false,
        activePath: "/task/ansible"
      }
    },
    {
      path: "/task/config",
      name: "TaskConfig",
      component: () => import("@/views/task/TaskConfig.vue"),
      meta: { icon: menuIcon("/task/config"), title: "配置管理" }
    }
  ]
};
