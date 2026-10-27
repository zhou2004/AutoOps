# HPA / VPA / KPA 安装与真实集群验收

测试日期：2026-10-02，时区 Asia/Shanghai。证据文件中的时间使用 UTC。

## 目标环境与变更范围

| 节点 | 地址 | 系统 / Kubernetes / 运行时 |
| --- | --- | --- |
| master-01 | 192.168.80.51 | Debian 12 / v1.34.3 / containerd 2.1.4 |
| worker-01 | 192.168.80.52 | 同上 |
| worker-02 | 192.168.80.53 | 同上 |

三台节点通过 SSH 和 Kubernetes API 检查均为 Ready，使用 Calico。测试前 Metrics Server v0.8.0 和 `autoscaling/v2` 正常，`autoscaling.k8s.io/v1` 与 `serving.knative.dev/v1` 不存在。因此页面提示 API 不可用的原因是缺少组件。

安装和负载测试限制在两个 worker。master 测试前可用内存约 487Mi；两个 worker 可用内存约 1.7Gi / 2.2Gi。新增组件使用受限资源和单副本配置，测试应用分别放在新建命名空间，不接管现有业务资源。

测试前已有 `topolvm-controller-7b6755b974-wbxvd` 的一个容器处于 CrashLoopBackOff；这项存量故障记录为基线，不属于本次伸缩功能安装。

## HPA：实际负载扩缩容

测试脚本：[`hpa-live.py`](../k8s/autoscaling/tests/hpa-live.py)。
结果证据：[`2026-10-02-hpa.json`](../k8s/autoscaling/tests/results/2026-10-02-hpa.json)。

| 检查项 | 实际结果 |
| --- | --- |
| 资源指标 | Metrics Server 返回真实 Pod CPU 使用量；`ScalingActive=True` |
| 初始状态 | 1 个 Ready 副本；CPU 为 requests 的约 5% |
| 负载限制 | 单 Pod 请求 20m CPU / 16Mi，限制 100m CPU / 64Mi；最大 3 副本 |
| 扩容 | 真实 CPU 负载触发 1 → 3，CPU 利用率峰值为请求值的 505% |
| 最大副本保护 | `ScalingLimited=True`，原因为 `TooManyReplicas` |
| 缩容 | 停载后 3 → 2 → 1，保留 1 个 Ready 副本 |
| 清理 | 已确认 `autoops-hpa-e2e-final-20261002` 命名空间删除，脚本等待实际删除后才返回通过 |

使用 worker-02 已缓存的 Alpine 镜像，按 digest 固定并设置 `imagePullPolicy: Never`；镜像 ID、容器重启计数及 SuccessfulRescale 事件保存在证据文件。测试采用 30 秒缩容稳定窗口，正常业务应按流量模式选择窗口。

最终复测在北京时间 00:47–00:51 完成；00:48:40 扩至 3，00:50:55 缩至 2，00:51:10 缩至 1。脚本的 CPU 负载带固定 30 分钟绝对到期时间，重启不延长；清理带 Namespace UID 前提，失败或超时均返回非零退出码。

## VPA：安装、Off 持续观察与自动调整

安装 **VPA 1.6.0**，使用与 Kubernetes 1.34 相容的固定源码提交
`9196162ba12e637ce4482107e1c17bcb80599050`。安装脚本校验上游清单摘要，创建 CRD、独立命名的 RBAC、TLS Secret 和三个组件；不会覆盖其他管理器拥有的既存 Deployment。

安装及证书维护见 [`vpa/1.6.0/README.md`](../k8s/autoscaling/vpa/1.6.0/README.md)。本集群使用 compact 资源配置，admission controller 固定在已成功缓存镜像的 worker-01；recommender 和 updater 位于 worker-02。三个组件均 Ready。

测试脚本：[`run-e2e.py`](../k8s/autoscaling/vpa/1.6.0/run-e2e.py)。
结果证据：[`2026-10-02-vpa.json`](../k8s/autoscaling/tests/results/2026-10-02-vpa.json)。

| 检查项 | 实际结果 |
| --- | --- |
| 初始资源 | 两个 Ready Pod，CPU requests 均为 25m，limits 均为 250m |
| Off 模式 | 获得建议后继续观察至少 90 秒，7 次快照中的 Pod UID 与 requests 均保持不变 |
| 真实采样 | 负载下 uncappedTarget 达到 296m；target 被策略上限限制为 250m |
| Recreate 模式 | updater 自动驱逐并重建两个 Pod，两者均变为 250m 且 Ready |
| 调整来源 | Deployment 模板仍为 25m；测试未手工删除 Pod 或触发 rollout restart |
| 审计证据 | 两条 `EvictedByVPA` 事件与新旧 Pod UID 对应 |
| 清理 | 已确认 `autoops-vpa-e2e-20261002` 命名空间删除 |

严格复测在北京时间 00:33:51–00:37:56 完成；00:36:26 切换 Recreate，00:36:51 和 00:37:51 出现驱逐事件。第一次试跑验证了自动重建，但 Off 观察时间只有约 1 秒，因此延长观察并重新完成上述验收。短时负载用于确认控制链路，不代表长期资源容量建议。

## KPA：安装、并发 / RPS、缩零与唤醒

安装 **Knative Serving 1.23.0 + Kourier 1.23.0**，使用官方固定摘要清单和镜像；该版本要求 Kubernetes 至少为 1.34。本集群的六个组件各为单副本，合计 requests 600m CPU / 544Mi，全部运行于 worker。组件 HPA 的 min/max 固定为 1；这是有限内存测试集群配置，不具备高可用能力。

安装和镜像缓存步骤见 [`knative/README.md`](../k8s/autoscaling/knative/README.md)。节点直接访问 GCR 受限，因此先校验 OCI 镜像内容，再按完整 digest 导入两个 worker 的 containerd；未改动节点 registry 配置。Kourier 使用 ClusterIP，不改动既有 Ingress / DNS。默认 `*.svc.cluster.local` 路由通过 `kourier-internal` 和正确的 Host header 验证。

测试脚本：[`kpa-e2e.py`](../k8s/autoscaling/knative/kpa-e2e.py)。两种模式均独立运行并检查实际 Pod 数，而非仅检查 desiredReplicas。
完整事件、执行版本和清理证明见 [KPA 实测证据](../k8s/autoscaling/knative/results/2026-10-02/README.md)。

| 检查项 | concurrency | rps |
| --- | --- | --- |
| 稳定窗口 / min / max | 30 秒 / 0 / 3 | 30 秒 / 0 / 3 |
| 目标值 | 并发 1 | 每秒请求 5 |
| 负载 | 8 线程，75 秒 | 8 线程，45 秒 |
| 首次从零唤醒 | HTTP 200，4.753 秒 | HTTP 200，2.868 秒 |
| 负载期间 | 达到 3 个 Ready 副本，没有超过 3 | 达到 3 个 Ready 副本，没有超过 3 |
| 请求结果 | 1992 成功 / 0 失败 | 1352 成功 / 0 失败 |
| 停流后 | desired、ready、实际 Pod 数均归零 | desired、ready、实际 Pod 数均归零 |
| 再次从零唤醒 | HTTP 200，10.662 秒 | HTTP 200，2.737 秒 |

测试应用每次业务请求等待 250ms，health 检查立即响应；每副本含 queue-proxy 合计 requests 50m / 82Mi、limits 400m / 192Mi。该结果证明请求驱动伸缩链路可用，不代表生产吞吐量或延迟指标。

清理逻辑加固后，使用最终脚本又完整复测 RPS 30 秒：896 请求成功、0 失败，实际 Pod 再次完成 0 → 3 → 0，两次从零唤醒均为 2.626 秒。北京时间 00:57:49 出现 `cleanup-confirmed`，随后才输出最终 `passed`，进程退出码为 0；确认 UID 删除前提、Knative Service 前台删除及命名空间等待均实际执行。三轮原始证据分别保留。

## AutoOps 接口验收

测试使用生产 service、controller、DAO、权限中间件和真实 Kubernetes API。权限数据存放于临时 SQLite，身份只注入测试路由，不修改运行中 AutoOps 的数据库、登录状态或业务权限。

可重复执行的命令和开关见[测试说明](../k8s/autoscaling/tests/README.md)。默认 `go test` 跳过真实集群测试；仅设置 kubeconfig 时执行读取，service 写入需另设 `AUTOOPS_AUTOSCALING_LIVE_MUTATE=1`，HTTP 写入需另设 `AUTOOPS_AUTOSCALING_HTTP_MUTATE=1`。

最终 **22 个子测试全部通过**，两套测试依次新建并删除隔离命名空间 `autoops-api-e2e-final-20261002`。

| 层级 | 数量 / 耗时 | 覆盖内容 |
| --- | --- | --- |
| Service | 6 / 57.15 秒 | 四类 API 能力；HPA CRUD、dry-run、真实旧 resourceVersion 冲突和同目标并发创建；VPA Off 共存、Auto / Recreate 与 HPA 冲突；KPA 稀疏更新、dry-run、重置后保留 Service / 非托管注解 / 镜像 |
| HTTP | 16 / 44.04 秒 | 未认证、只读写入、命名空间隔离、原生 Service 权限不能修改 Knative Service；非法参数、未知 JSON 字段、多个 JSON 对象；按钮权限；三类策略的真实操作和 dry-run |

完整日志：[service 最终结果](../k8s/autoscaling/tests/results/2026-10-02-service-final-result.log)、[HTTP 最终结果](../k8s/autoscaling/tests/results/2026-10-02-http-final-result.log)。

首次 service 测试功能断言已通过，但命名空间清理超过 90 秒，整套测试如实失败，保留[首轮失败日志](../k8s/autoscaling/tests/results/2026-10-02-service-first-cleanup-timeout.log)。原因是 Python 测试进程作为 PID 1 未处理 SIGTERM，加上 Knative 默认 300 秒终止宽限期。测试应用补充信号处理；清理改为带 UID 前提以前台传播删除本次 Knative Service、等待结束，再删除命名空间并检查实际消失。随后完整重跑通过，未强制移除 finalizer。

接口测试与持续负载测试分别保存结果。API 能发现资源、能保存策略，以及控制器能实际执行伸缩，是三项分别验证的条件。此次测试不覆盖浏览器自动化、JWT/Redis 登录流程、节点故障切换或生产容量压测。

## 最终状态与维护事项

北京时间 00:58:21 的[集群状态记录](../k8s/autoscaling/tests/results/2026-10-02-cluster-health.json)确认：

- 三台节点全部 Ready；MemoryPressure / DiskPressure / PIDPressure 均为 False。
- VPA 三个、Knative / Kourier 六个 Deployment 全部 1/1 Ready；九个组件 Pod 均位于 worker，容器重启计数均为 0。
- HPA、VPA、Knative Serving 和 Metrics 四类 API 均可发现。
- 所有本轮测试命名空间均已删除，组件保留供 AutoOps 使用。
- 基线中的业务 Pod 均仍存在。`demo-app` 的一次重启来自原命令 `sleep 3600` 正常退出（exitCode 0 / Completed）；TopoLVM 的既有 csi-resizer CrashLoopBackOff 仍在持续，未在本轮修改。

`api/` 下 `go test ./...` 全部通过，运行时移除了真实集群测试开关，见[回归日志](../k8s/autoscaling/tests/results/2026-10-02-go-regression.log)。此外通过 Python AST、安装脚本 `bash -n` 和 3 项 VPA 渲染测试。

在 AutoOps 选择本集群并刷新弹性伸缩页面即可重新发现 API。KPA 操作对象是 Knative Service；普通 Deployment 使用 HPA / VPA。两种管理器同时调整同一资源时需遵守页面的冲突检查。

当前安装针对有限内存的开发集群：组件为单副本；master 最终内存使用约 88%，worker 约 73%。VPA webhook 证书有效期 365 天，当前脚本不提供自动续期，续期方法见 VPA 安装说明。生产使用前需要另行配置高可用、容量和证书续期。

## 参考

- [Kubernetes HPA 实操文档](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale-walkthrough/)
- [AutoOps 弹性伸缩功能说明](2026-10-01-k8s-autoscaling.md)
