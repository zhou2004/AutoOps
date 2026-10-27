# Kubernetes 弹性伸缩维护：HPA、VPA、KPA

## 入口与能力

在 **容器管理 → 工作负载 → HPA / VPA / KPA** 打开；也可访问 `/k8s/autoscaling`。入口保留当前集群和命名空间。每次操作只作用于一个命名空间。

| 类型 | API | 本项目支持 |
| --- | --- | --- |
| HPA | `autoscaling/v2` | 创建、编辑、删除；副本上下限；CPU/内存、容器资源、自定义和外部指标；扩缩容稳定窗口与速率；当前/目标副本、指标、控制器条件 |
| VPA | `autoscaling.k8s.io/v1` | 创建、编辑、删除；Off/Initial/Recreate/Auto/InPlaceOrRecreate/InPlace；容器策略、资源边界、控制资源和 requests/limits 策略；容器建议值、上下界、未约束建议、条件 |
| KPA | `serving.knative.dev/v1` Service | 为已有 Knative Service 配置并发或 RPS 伸缩、目标值、副本边界、稳定/panic 窗口、panic 阈值、目标利用率、缩容延迟；查看就绪修订、路由、条件；恢复默认策略 |

这里 KPA 是 **Knative Pod Autoscaler**。页面修改 Knative Service 的 `spec.template.metadata.annotations`，由 Knative 生成 Revision 和 PodAutoscaler；不会直接写入控制器拥有的 PodAutoscaler。

HPA/VPA 目标支持同命名空间 `apps/v1` Deployment、StatefulSet；VPA 另支持 DaemonSet。其他目标种类暂未开放。受其他控制器管理的伸缩资源只读。

## 前提和安装边界

真实集群的安装与验收记录见 [2026-10-02 安装测试报告](2026-10-02-autoscaling-live-validation.md)，包含固定版本安装脚本及可重复执行的测试。

1. 集群需提供 `autoscaling/v2`。CPU/内存 HPA 需要 metrics-server，并为工作负载容器配置对应的 `resources.requests`。自定义/外部指标需要相应 Adapter。指标 API 存在不代表指标实时可用，需结合 HPA conditions 检查。
2. VPA 需事先安装兼容版本的 CRD、recommender、updater、admission-controller 及 webhook 证书。先以 Off 模式观察建议。`InPlace` 系列依赖 Kubernetes、VPA 版本和相关能力，安装版本的 CRD/Webhook 决定是否接受。
3. KPA 需安装 Knative Serving 和对应网络组件，并已有能正常服务的 Knative Service。`minScale=0` 允许缩至 0，但实际行为仍受集群配置和流量影响。
4. 本页面不会自动安装或升级以上组件。API 能力卡会提示缺失或访问失败。系统级组件安装可通过 Ansible 模块安排；需使用与目标 Kubernetes 版本匹配的组件版本。

用户提供的[背景文章](https://www.cnblogs.com/yangmeichong/p/16638895.html)用于理解三类伸缩方案；实现使用当前稳定 API，没有使用文中的旧 `v2beta1/v2beta2` HPA 示例。

## HPA：多指标与稳定缩容

1. 创建 HPA，选择已有 Deployment 或 StatefulSet，填写名称和副本范围，例如 2–10。
2. 默认提供 CPU 70% 指标，按钮可追加内存 70%。编辑 `metrics` JSON 支持完整 `autoscaling/v2` 指标结构，例如按外部队列长度伸缩：

```json
[
  {
    "type": "External",
    "external": {
      "metric": { "name": "queue_messages", "selector": { "matchLabels": { "queue": "orders" } } },
      "target": { "type": "AverageValue", "averageValue": "30" }
    }
  }
]
```

3. `behavior` 设置缩容稳定窗口和每周期最大变化。默认表单为 300 秒窗口、每 60 秒最多缩容 25%；这些是本页面的新建示例值。传 `{}` 使用 Kubernetes 默认行为。
4. 点击“服务端试运行”查看 Kubernetes 接受后的对象；点击“校验并保存”也会先执行试运行，通过后确认保存。
5. 在“详情”查看 currentMetrics、目标指标和 conditions。`ScalingActive=False` 常与指标不可用、目标不存在或缺少 requests 有关；`ScalingLimited=True` 表示期望值受到边界限制。

HPA 行为与多指标结构参考 [Kubernetes 官方 HPA 文档](https://kubernetes.io/docs/concepts/workloads/autoscaling/horizontal-pod-autoscale/)。

## VPA：先建议，再调整

1. 创建策略，默认 `Off`：收集并显示建议，不修改 Pod 资源。
2. `containerPolicies` 是数组，`containerName: "*"` 定义默认策略；可添加特定容器规则，`mode: "Off"` 排除某个容器。
3. `minAllowed/maxAllowed` 使用 Kubernetes quantity，例如 CPU `100m`、`2`，内存 `128Mi`、`2Gi`。`controlledResources` 选择 CPU/内存；`RequestsOnly` 只调整 requests。
4. 观察详情中的 `target`、`lowerBound`、`upperBound`、`uncappedTarget`，再切换自动更新模式。Recreate 可能驱逐并重建 Pod，应结合 PDB、可用副本和业务容忍度；`minReplicas` 影响 updater 的驱逐条件。
5. 编辑保留本编辑器未管理的 VPA spec 扩展字段，例如自定义 recommender 和其他 updatePolicy 字段。已有策略未指定 mode、minReplicas 或容器策略时，编辑保留该缺省状态；新建示例值不会注入已有策略。

字段和模式以 [VPA 官方 CRD](https://github.com/kubernetes/autoscaler/blob/master/vertical-pod-autoscaler/deploy/vpa-v1-crd-gen.yaml) 为准，集群已安装的版本可能提供较少模式。

## KPA：请求负载与突发流量

1. KPA 标签展示该命名空间已有 Knative Service，包括使用 Knative HPA class 的服务。点击“配置”会显式设置 `kpa.autoscaling.knative.dev` class。
2. `concurrency` 以并发请求为指标，`rps` 以请求速率为指标。配置 target、最小/最大副本；最大副本 0 表示不设上限。
3. 未单独设置的字段显示为空并继续继承，保存只合并显式值，避免固定集群默认值。稳定窗口接受 6–3600 秒；panic 窗口为稳定窗口的 1–100%；panic 阈值接受 110–1000%；目标利用率为 (0,100]%；缩容延迟为 0–3600 秒。
4. 修改保留容器镜像、环境、流量配置和其他注解，移除旧的模板 Revision 名，让 Knative 分配新名称。若流量固定指向旧 Revision，需在 Knative 流量配置中完成切换；本页面不会改写该流量意图。
5. “恢复默认”只清除本模块管理的十项伸缩注解，包括 class、metric、target、min/max-scale、window、panic-window-percentage、panic-threshold-percentage、target-utilization-percentage、scale-down-delay。Service 保留，之后按其余配置和集群默认策略运行。它不是暂停伸缩。

请求指标、边界和窗口参考 [Knative 指标文档](https://knative.dev/docs/serving/autoscaling/autoscaling-metrics/)、[副本边界](https://knative.dev/docs/serving/autoscaling/scale-bounds/)、[KPA 窗口与阈值](https://knative.dev/docs/serving/autoscaling/kpa-specific/)。

## 权限、冲突与并发保护

AutoOps 用户权限与导入 kubeconfig 的 Kubernetes 权限分别生效：

| AutoOps RBAC apiGroups | resources | verbs |
| --- | --- | --- |
| `autoscaling` | `horizontalpodautoscalers` | list/get/create/update/delete |
| `autoscaling.k8s.io` | `verticalpodautoscalers` | list/get/create/update/delete |
| `serving.knative.dev` | `services` | list/get/update；KPA 恢复默认也要求 update |

后端同时匹配 group、resource、verb 和命名空间。核心 API 的 `services` 权限不等同于 Knative Service 权限；仅 HPA list 不授予 delete。兼容传统命名空间 readonly/write/admin 授权，停用管理员角色和停用用户组不授予权限，角色不得跨集群或超出声明的命名空间绑定。前端从服务端能力接口取得按钮权限。

后端 kubeconfig 还需读取目标工作负载，并 list HPA/VPA 才能完成冲突检查。最小命名空间授权示例：[namespace-role.example.yaml](../k8s/autoscaling/namespace-role.example.yaml)。示例没有自动应用，需替换 namespace 和已有 ServiceAccount。

- 阻止同一目标多个同类型策略。
- 阻止 CPU/内存 Resource 或 ContainerResource HPA 与非 Off VPA 同目标运行；这是保守保护，包括 VPA 只控制部分资源的情况。可以使用 Off VPA，或基于业务指标的 HPA。
- 阻止为 Knative 控制器拥有的 Deployment/StatefulSet 添加独立 HPA/VPA。
- API 已安装但没有读取权限时，冲突检查失败即停止写入。VPA API 不存在时允许独立 HPA。
- 编辑和删除必须携带最新 `resourceVersion`；版本冲突需刷新后重试。试运行使用 Kubernetes `dryRun=All`，不写入对象，但无法预判未来指标和调度结果。
- 单个 API 服务实例将策略检查与写入串行化；冲突检查无法消除其他客户端或多个 API 实例同时创建不同名称策略的竞争。集群级强制唯一性可进一步通过 admission policy/webhook 保证。

## API

基础路径：`/api/v1/k8s/cluster/:id/namespaces/:namespaceName/autoscaling`。

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/capabilities` | HPA/VPA/KPA/metrics API 能力与当前用户权限 |
| GET | `/:kind` | 列表，kind 为 hpa/vpa/kpa |
| GET | `/:kind/:name` | 对象详情 |
| POST | `/:kind?dryRun=true` | 创建 HPA/VPA；dryRun 可选 |
| PUT | `/:kind/:name?dryRun=true` | 编辑策略或配置已有 Knative Service |
| DELETE | `/:kind/:name?resourceVersion=...` | 删除 HPA/VPA 或恢复 KPA 默认配置 |

保存对象为 `{ "name": "...", "resourceVersion": "...", "hpa": { ... } }`，或 `vpa` / `kpa` 对应配置。创建不需要 resourceVersion。`dryRun` 仅接受 `true` 或 `false`；API 请求体限制为 1 MiB，超时为 30 秒。KPA POST 明确拒绝，防止误建不完整 Service。

## 本地验证与上线验收

无需实际集群的测试：

```bash
cd api
go test ./api/k8s/service ./middleware -run '^TestAutoscaling' -count=1
```

测试覆盖输入边界、HPA/VPA 冲突、Knative 工作负载保护、KPA 更新/重置保留 Service 配置、旧版本拒绝、dry-run 转发、控制器资源保护，以及权限的 group/verb/namespace/角色绑定隔离。

上线后用专用测试命名空间验收：准备配置 requests 的 Deployment；先试运行并创建 HPA；创建 Off VPA 确认建议；测试切换自动模式时冲突提示；在已有 Knative Service 上配置/重置并确认新 Revision；验证只读账号不能保存和删除。组件缺失、指标缺失、目标不存在、旧 resourceVersion 都应有可操作的错误提示。
