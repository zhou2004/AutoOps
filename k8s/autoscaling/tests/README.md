# 弹性伸缩真实集群测试

这里的测试用于已授权的测试集群。安装入口分别是
[`../vpa/1.6.0/install.sh`](../vpa/1.6.0/install.sh) 和
[`../knative/install.sh`](../knative/install.sh)。默认 `go test` 不会连接集群。

## HPA：真实负载扩缩容

在能够执行 `kubectl` 的机器上运行，先确认其当前 context。指定新的测试命名空间、允许调度的 worker 和可信镜像；镜像需包含 `sh`、`cat`、`date`、`sleep`、`timeout`。

```bash
python3 hpa-live.py \
  --namespace autoops-hpa-e2e-unique-run \
  --node worker-02 \
  --image 'registry.example/alpine@sha256:REPLACE_WITH_VERIFIED_DIGEST' \
  --image-pull-policy IfNotPresent \
  --output /tmp/autoops-hpa-result.json
```

测试通过实际 CPU 指标验证：空闲时 1 个副本，启用负载后达到 3 个副本，停载后回到 1 个副本。单 Pod CPU 上限为 100m、内存上限为 64Mi；命名空间有独立 ResourceQuota。负载在启动测试 30 分钟后自动停止，容器重启不会延长该期限。测试不会接管已有同名命名空间，正常结束或 Python 异常时带 namespace UID 前提请求删除，并等待最多 180 秒；清理超时也会导致测试失败。进程被强制终止时，依据 `app.kubernetes.io/managed-by=autoops-live-test` 标签、UID 和运行记录核查清理。

结果 JSON 包含时间、指标、条件、镜像、伸缩事件及命名空间实际删除结果。此次运行证据保存于 [`results/2026-10-02-hpa.json`](results/2026-10-02-hpa.json)。

## AutoOps 服务与 HTTP 接口

在项目 `api/` 目录执行。kubeconfig 必须放在仓库之外并限制读取权限。

```bash
# 只读：真实 Kubernetes API 能力、生产 controller/DAO/RBAC 的 HTTP 联调。
AUTOOPS_AUTOSCALING_LIVE_KUBECONFIG=/secure/kubeconfig \
  go test ./api/k8s/service ./api/k8s/controller \
  -run '^TestAutoscaling(Live|HTTPLive)$' -v -count=1

# 创建新的隔离命名空间执行 CRUD / dry-run / 冲突与 KPA 配置测试。
AUTOOPS_AUTOSCALING_LIVE_KUBECONFIG=/secure/kubeconfig \
AUTOOPS_AUTOSCALING_LIVE_MUTATE=1 \
AUTOOPS_AUTOSCALING_LIVE_IMAGE='registry.example/http-test@sha256:REPLACE_WITH_VERIFIED_DIGEST' \
  go test ./api/k8s/service -run '^TestAutoscalingLive$' -v -count=1 -timeout=10m

# 通过生产 HTTP controller、DAO、权限中间件执行三类接口 CRUD。
AUTOOPS_AUTOSCALING_LIVE_KUBECONFIG=/secure/kubeconfig \
AUTOOPS_AUTOSCALING_HTTP_MUTATE=1 \
AUTOOPS_AUTOSCALING_LIVE_IMAGE='registry.example/http-test@sha256:REPLACE_WITH_VERIFIED_DIGEST' \
  go test ./api/k8s/controller -run '^TestAutoscalingHTTPLive$' -v -count=1 -timeout=10m
```

HTTP 联调使用临时 SQLite 和测试身份，读取真实 Kubernetes API；不会更改运行中的 AutoOps 数据库，也不覆盖 JWT/Redis 登录和浏览器交互。服务测试的命名空间清理带 UID 前提并等待删除完成。

镜像须监听 8080 并能处理 SIGTERM；可用 `AUTOOPS_AUTOSCALING_LIVE_HTTP_COMMAND_JSON` 提供启动命令。当前 fixture 使用 `kubernetes.io/role=node` 限制 worker，Knative 安装需启用 `kubernetes.podspec-nodeselector`。可用 `AUTOOPS_AUTOSCALING_LIVE_KINDS=hpa` 单独验证原生 HPA；完整验收必须安装 VPA/Knative 后再运行全部类型。服务与 HTTP 写入测试应顺序运行，各自使用新建命名空间；可用 `AUTOOPS_AUTOSCALING_LIVE_NAMESPACE` 指定唯一名称。

VPA 的持续采样和资源调整、KPA 的真实流量与零副本唤醒，分别使用对应组件目录中的测试脚本；成功创建 CRD 对象不等于这些行为已验证。
