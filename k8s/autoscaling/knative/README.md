# Knative Serving / Kourier 安装与 KPA 实测

这里固定使用 Serving **1.23.0** 和 Kourier **1.23.0**。Serving 该版本的[源码版本检查](https://github.com/knative/serving/blob/knative-v1.23.0/vendor/knative.dev/pkg/version/version.go)要求 Kubernetes **1.34.0 及以上**；不要修改 `KUBERNETES_MIN_VERSION` 绕过检查。

## 安装配置

这是为内存有限的开发集群准备的配置：六个组件各一个副本，总 requests 为 600m CPU / 544Mi 内存，内存 limits 合计 1664Mi；原有组件 HPA 的 min/max 均限制为 1。它没有高可用能力，不能直接当作生产容量配置。

- 所有组件使用 `nodeSelector: {kubernetes.io/role: node}`。该标签须仅存在于允许承载组件的工作节点。
- 开启 Knative 的 `kubernetes.podspec-nodeselector`，测试服务使用相同选择器。
- Kourier 使用独立 `kourier-system` 命名空间及 ClusterIP，不创建外部负载均衡器、不修改已有 Ingress 或 DNS。
- `config-network` 选择 Kourier；安装脚本拒绝覆盖已有、未标记为本配置的 Knative 命名空间。
- 六个组件 Deployment 的镜像和 queue-proxy 均固定 digest。Envoy 的上游可变标签被替换为下述已解析摘要。
- 官方 CRD、RBAC、webhook 规则保持原含义。webhook 处理 Knative API 组和 Knative 配置，不给普通业务 Pod 注入 sidecar。

上游依据：[Serving YAML 安装](https://knative.dev/docs/install/yaml-install/serving/install-serving-with-yaml/)、[Serving 1.23.0](https://github.com/knative/serving/releases/tag/knative-v1.23.0)、[Kourier 1.23.0](https://github.com/knative-extensions/net-kourier/releases/tag/knative-v1.23.0)。

## 下载、审查和安装

执行端需要 Python 3、PyYAML、kubectl 和已确认的 kubeconfig。`prepare.py` 只下载及生成本地文件，不访问集群。它校验三份上游 YAML 的 SHA256，写出修改后的 YAML、`images.txt` 和 `review.json`。

```bash
python3 k8s/autoscaling/knative/prepare.py \
  --output /tmp/autoops-knative-rendered \
  --envoy-image docker.io/envoyproxy/envoy@sha256:3e5f1b892f15d46a2674ffc81b9f86f4d0df7794e3ef49e92ee7062871b87be1

kubectl config current-context
kubectl version
kubectl get nodes -l kubernetes.io/role=node
cat /tmp/autoops-knative-rendered/review.json

# 确认镜像可拉取，或已按相同完整 image@sha256 引用导入每个工作节点。
bash k8s/autoscaling/knative/install.sh /tmp/autoops-knative-rendered
```

安装顺序为 CRD → Serving → 等待 Serving webhook 就绪 → Kourier。CRD 使用 server-side apply，避免大 CRD 超过 last-applied annotation 大小限制；不使用 `--force-conflicts` 覆盖其他管理器的字段。

若节点无法访问 GCR，可在可联网的受控机器使用官方 [crane](https://github.com/google/go-containerregistry/tree/main/cmd/crane) 和 `cache-images.py` 生成 OCI 包。脚本等待每个镜像下载完成，校验所有 blob，保留原始多架构 index 及完整 `image@sha256` 引用；仅有同名 tag 不能满足 digest 引用。此方式不需要改动节点现有 registry 配置。

```bash
cp /tmp/autoops-knative-rendered/images.txt /tmp/autoops-knative-images.txt
echo 'docker.io/library/python@sha256:4c47124a8391cb7a9f571164147d154777cf012a4ece5f86097130d7a4478111' >> /tmp/autoops-knative-images.txt
python3 k8s/autoscaling/knative/cache-images.py \
  --images-file /tmp/autoops-knative-images.txt --output /tmp/autoops-knative-archives
# 将 archive 传到每个工作节点，再逐个导入：
ctr -n k8s.io images import --platform linux/amd64 --digests <archive.tar>
```

在 kubeasz 节点上，`ctr` 常位于 `/opt/kube/bin/containerd-bin/ctr`。镜像导入完成后再安装控制器，避免 webhook 等待镜像期间影响新 Knative 资源创建。

## KPA 扩缩容测试

`kpa-e2e.py` 必须在可访问 Kubernetes API 和 Kourier ClusterIP 的执行端运行。默认 `.svc.cluster.local` 路由使用 `kourier-internal` Service 的内部监听端口；脚本用该 ClusterIP 加 Host header，不依赖外部 DNS。它创建全新的 `autoops-kpa-e2e-*` 命名空间，只操作其中的 `kpa-check` Service；命名空间已存在时拒绝复用。退出时检查测试标记和创建时记录的 UID，通过 API 删除前置条件保护同名重建资源；先等待 Service foreground 删除，再删除并等待命名空间消失。

```bash
python3 k8s/autoscaling/knative/kpa-e2e.py \
  --namespace autoops-kpa-e2e-20261002 --metric concurrency

# 前一命名空间删除完成后，可换用独立名称验证 RPS。
python3 k8s/autoscaling/knative/kpa-e2e.py \
  --namespace autoops-kpa-e2e-rps-20261002 --metric rps --duration 45
```

每次测试遵循以下断言：

1. Service Ready 后等待 Deployment desired/ready 和实际 Pod 数全部为 0。
2. 通过 Kourier ClusterIP 加正确 Host header 发请求，验证缩零后的 HTTP 200 和应用响应正文。
3. 使用 8 个请求线程施加 30–120 秒有界流量，要求至少 2 个就绪副本，且副本不超过 3。
4. 停流后等待真实 Pod 数回到 0，再请求一次，验证再次从零恢复。
5. 记录请求成功/失败数、每阶段副本和冷启动耗时；流量错误、未扩容或未归零均使脚本失败。

应用使用固定 Python 镜像，在每个业务请求中等待 250ms，提供稳定并发压力；health 请求不等待。应用显式处理 SIGTERM，避免容器 PID 1 忽略默认信号后等到 300 秒强制退出。应用与 queue-proxy 每副本合计 requests 50m / 82Mi、limits 400m / 192Mi，最高 3 副本。此测试验证 HTTP 请求驱动伸缩，不模拟生产长连接、故障切换或高吞吐容量。

KPA 的缩零也受[全局开关和保留期](https://knative.dev/docs/serving/autoscaling/scale-to-zero/)影响。脚本不修改全局缩零配置；若现有配置关闭缩零，测试会如实失败。

## 2026-10-02 实测

在 Kubernetes 1.34.3 / containerd 2.1.4 / Debian 12 / amd64 / Calico 集群上，按上述配置安装并验证。六个组件均在工作节点上运行，安装后保持 1/1 Ready。

| 指标 | 流量阶段 | 成功 / 失败请求 | 最大就绪副本 | 首次从零唤醒 | 再次从零唤醒 |
| --- | --- | --- | --- | --- | --- |
| concurrency，target=1 | 8 并发，75 秒 | 1992 / 0 | 3 | 4.753 秒 | 10.662 秒 |
| rps，target=5 | 8 并发，45 秒 | 1352 / 0 | 3 | 2.868 秒 | 2.737 秒 |
| rps，最终脚本复验 | 8 并发，30 秒 | 896 / 0 | 3 | 2.626 秒 | 2.626 秒 |

三轮都确认了初始真实 Pod 数为 0、负载扩到 3、停流后真实 Pod 数再次归零、HTTP 200 与应用响应正文正确。表中的请求数只统计负载阶段，每轮另有两次冷启动请求。耗时是本测试环境的观测值，不是时延承诺。测试范围不包含公网入口、TLS 证书配置、生产容量或组件故障切换。

最终脚本的额外复验也通过了 UID 删除保护、Service foreground 删除及命名空间删除等待，清理确认后才输出最终成功，退出码为 0。三个测试命名空间均已独立确认删除，Serving / Kourier 保留。完整日志、组件状态、脚本版本差异及清理确认见 [实测证据](results/2026-10-02/README.md)。
