# 2026-10-02 实测结果

集群：Kubernetes v1.34.3，master-01 / worker-01 / worker-02。日志中的 UTC 时间对应北京时间 2026-10-02。

| 文件 | 验证内容 |
| --- | --- |
| [HPA](2026-10-02-hpa.json) | 实际 CPU 负载 1 → 3 → 1，副本上限、镜像、事件、清理完成和执行脚本 SHA256 |
| [VPA](2026-10-02-vpa.json) | Off 持续 90 秒不变，Recreate 自动驱逐重建并修改 requests，命名空间清理 |
| [Service 最终日志](2026-10-02-service-final-result.log) | 6 个真实 API 子测试通过，含完整清理 |
| [HTTP 最终日志](2026-10-02-http-final-result.log) | 16 个权限 / 参数 / 三类策略操作子测试通过，含完整清理 |
| [Service 首次失败](2026-10-02-service-first-cleanup-timeout.log) | 保留清理超时失败记录；修复测试进程 SIGTERM 后重跑通过 |
| [Go 回归](2026-10-02-go-regression.log) | 关闭真实集群开关后 `go test ./...` 通过 |
| [集群检查](2026-10-02-cluster-health.json) | 节点状态、9 个已安装组件、API 发现、测试命名空间清理和既有 Pod 基线对比 |

KPA 并发与 RPS 的原始事件和执行版本单独保存在 [Knative 结果目录](../../knative/results/2026-10-02/README.md)。完整说明见[安装与真实集群验收](../../../../docs/2026-10-02-autoscaling-live-validation.md)。

HTTP 测试通过隔离 SQLite 和注入身份调用生产权限中间件及 controller，未覆盖 JWT / Redis 登录或浏览器交互。结果文件只保留测试输出与资源状态，不保存 SSH 密码、kubeconfig 或证书私钥。
