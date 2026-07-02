const Layout = () => import("@/layout/index.vue");

export default {
    path: "/k8s",
    name: "K8s",
    component: Layout,
    redirect: "/k8s/list",
    meta: {
        icon: "ep/cpu",
        title: "容器管理",
        rank: 20
    },
    children: [
        {
            path: "/k8s/list",
            name: "K8sClusters",
            component: () => import("@/views/K8s/k8s-clusters.vue"),
            meta: { title: "集群管理" }
        },
        {
            path: "/k8s/node",
            name: "K8sNodes",
            component: () => import("@/views/K8s/k8s-nodes.vue"),
            meta: { title: "节点管理" }
        },
        {
            path: "/k8s/namespace",
            name: "K8sNamespace",
            component: () => import("@/views/K8s/k8s-namespace.vue"),
            meta: { title: "命名空间" }
        },
        {
            path: "/k8s/workload",
            name: "K8sWorkload",
            component: () => import("@/views/K8s/k8s-workloads.vue"),
            meta: { title: "工作负载" }
        },
        {
            path: "/k8s/network",
            name: "K8sNetwork",
            component: () => import("@/views/K8s/k8s-network.vue"),
            meta: { title: "网络管理" }
        },
        {
            path: "/k8s/config",
            name: "K8sConfig",
            component: () => import("@/views/K8s/k8s-config.vue"),
            meta: { title: "配置管理" }
        },
        {
            path: "/k8s/storage",
            name: "K8sStorage",
            component: () => import("@/views/K8s/k8s-storage.vue"),
            meta: { title: "存储管理" }
        },
        {
            path: "/k8s/crd",
            name: "K8sCrd",
            component: () => import("@/views/K8s/k8s-crd.vue"),
            meta: { title: "自定义资源" }
        },
        {
            path: "/k8s/monitoring",
            name: "K8sMonitoring",
            component: () => import("@/views/K8s/nodes/k8s-monitoring.vue"),
            meta: { title: "监控仪表板" }
        },
        // 以下为隐藏路由（不在菜单中显示）
        {
            path: "/k8s/cluster/:clusterId",
            name: "K8sClusterDetail",
            component: () => import("@/views/K8s/clusters/K8sDetails.vue"),
            meta: { title: "集群详情", showLink: false }
        },
        {
            path: "/k8s/cluster/:clusterId/node/:nodeName",
            name: "K8sNodeDetail",
            component: () => import("@/views/K8s/nodes/NodeDetails.vue"),
            meta: { title: "节点详情", showLink: false }
        },
        {
            path: "/k8s/pod/:clusterId/:namespace/:podName",
            name: "K8sPodDetail",
            component: () => import("@/views/K8s/pods/k8s-pod.vue"),
            meta: { title: "容器详情", showLink: false }
        },
        {
            path: "/k8s/terminal/:clusterId/:namespace/:podName",
            name: "K8sTerminal",
            component: () => import("@/views/K8s/pods/K8S-sterminal.vue"),
            meta: { title: "容器终端", showLink: false }
        }
    ]
};
