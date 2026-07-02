const Layout = () => import("@/layout/index.vue");

export default {
    path: "/task",
    name: "Task",
    component: Layout,
    redirect: "/task/template",
    meta: {
        icon: "ep/operation",
        title: "任务中心",
        rank: 30
    },
    children: [
        {
            path: "/task/template",
            name: "TaskTemplate",
            component: () => import("@/views/task/TaskTemplate.vue"),
            meta: { title: "任务模版" }
        },
        {
            path: "/task/job",
            name: "TaskJob",
            component: () => import("@/views/task/TaskJob.vue"),
            meta: { title: "任务作业" }
        },
        {
            path: "/task/ansible",
            name: "TaskAnsible",
            component: () => import("@/views/task/TaskAnsible.vue"),
            meta: { title: "Ansible任务" }
        },
        {
            path: "/task/config",
            name: "TaskConfig",
            component: () => import("@/views/task/TaskConfig.vue"),
            meta: { title: "配置管理" }
        }
    ]
};
