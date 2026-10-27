# VPA 安装与真实行为验证

## 版本与范围

本项目提供 [VPA 1.6.0 固定版本安装目录](../k8s/autoscaling/vpa/1.6.0/README.md)。选择依据是 Kubernetes Autoscaler 当前兼容矩阵：VPA 1.6 支持 Kubernetes 1.34–1.36；本次集群为 Kubernetes 1.34.3。当前较新的 VPA 1.8 面向 Kubernetes 1.36–1.38，因此本集群使用 1.6.0。[官方安装文档](https://github.com/kubernetes/autoscaler/blob/vertical-pod-autoscaler-1.8.0/vertical-pod-autoscaler/docs/installation.md)、[1.6.0 发布说明](https://github.com/kubernetes/autoscaler/releases/tag/vertical-pod-autoscaler-1.6.0)。

安装内容包含官方 CRD、recommender、updater、admission controller、TLS 和 RBAC。源码提交固定为 `9196162ba12e637ce4482107e1c17bcb80599050`，官方清单有 SHA256 校验，三个控制器镜像固定为 `registry.k8s.io/autoscaling/vpa-*:1.6.0`。脚本添加 worker 调度约束和 AutoOps 所有权标签，RBAC 使用 `autoops:vpa:` 前缀，避免占用官方通用角色名。

本目录负责部署与验证；AutoOps 的 K8s → 弹性伸缩页面负责日常 VPA 资源管理。VPA 的资源调整会创建新 Pod，应用应具备副本冗余、就绪探针和合适的 PodDisruptionBudget。[官方 VPA 工作方式](https://github.com/kubernetes/autoscaler/blob/vertical-pod-autoscaler-1.6.0/vertical-pod-autoscaler/docs/quickstart.md)。

## 安装前检查

管理机需要集群管理权限、`kubectl`、Python 3.9+、PyYAML、OpenSSL。指定的 worker 必须 Ready，且不能是控制平面节点。集群需要可用的 Metrics API。

```bash
cd k8s/autoscaling/vpa/1.6.0
kubectl version
kubectl get nodes -o wide
kubectl top nodes
./install.sh check --workers worker-01,worker-02 --compact
```

`check` 校验源码摘要、Kubernetes 版本、节点指标、worker 状态和既有 Deployment 所有权，不创建资源。它不提前拉镜像，也不替代容量规划和 API 权限检查。

worker 需能访问 `registry.k8s.io`。可分别在目标 worker 提前缓存固定版本镜像：

```bash
crictl pull registry.k8s.io/autoscaling/vpa-recommender:1.6.0
crictl pull registry.k8s.io/autoscaling/vpa-updater:1.6.0
crictl pull registry.k8s.io/autoscaling/vpa-admission-controller:1.6.0
```

## 执行安装

```bash
./install.sh install --workers worker-01,worker-02 --compact
```

脚本使用 server-side apply，等待 CRD Established、三个 Deployment 完成 rollout，以及 admission 异步注册 `vpa-webhook-config`。重复运行会保留现有 TLS Secret。已有 VPA Deployment 如果不带本安装器的所有权标签，脚本会停止，需先确认既有安装的维护方式。

`--compact` 适合本次三节点验证集群，资源配置如下：

| 控制器 | CPU request / limit | 内存 request / limit |
| --- | --- | --- |
| recommender | 50m / 300m | 128Mi / 384Mi |
| updater | 50m / 300m | 64Mi / 192Mi |
| admission controller | 50m / 300m | 64Mi / 192Mi |

不加该参数时保留上游资源配置。生产环境需根据 Pod 数量、历史样本及实际使用量调整。

本次 worker-02 拉取 admission 镜像遇到连接中断，worker-01 已成功缓存。使用下面的配置将 admission 约束到 worker-01，另外两个组件仍可使用两个 worker：

```bash
./install.sh install --workers worker-01,worker-02 --compact \
  --admission-worker worker-01
```

该选择缩小了 admission 的可调度节点范围；后续补齐所有 worker 的镜像缓存后，可以去掉该参数恢复两个 worker 候选。

```bash
kubectl -n kube-system get deployment vpa-recommender vpa-updater vpa-admission-controller
kubectl -n kube-system get pods -l app.kubernetes.io/managed-by=autoops-vpa -o wide
kubectl get mutatingwebhookconfiguration vpa-webhook-config
kubectl top pods -n kube-system -l app.kubernetes.io/managed-by=autoops-vpa
```

## 真实 Off → Recreate 验证

`run-e2e.py` 只接受一个新命名空间，目标命名空间已存在时立即退出。测试镜像必须固定摘要，且已经缓存在指定 worker；Pod 使用 `imagePullPolicy: Never`。

```bash
python3 -B run-e2e.py \
  --namespace autoops-vpa-e2e-20261002 \
  --worker worker-01 \
  --image docker.io/library/alpine@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 \
  --output /tmp/vpa-e2e-result.json \
  --timeout 900 --cleanup
```

上面的 Alpine 摘要是本次 worker-01 的缓存，其他环境应替换成实际可用摘要。两个测试 Pod 分别 request CPU 25m、内存 32Mi，limit CPU 250m、内存 128Mi，仅在指定 worker 产生负载。VPA 仅控制 CPU requests，范围 50m–250m；PDB 保留至少一个副本。

测试依次核对：

1. Off 模式创建的两个 Pod 保持 CPU request 25m。
2. 从首次建议开始继续观察至少 90 秒，跨过默认 updater 的 60 秒周期；每次核对两 Pod UID、就绪状态和 requests 不变。
3. 等待 `uncappedTarget.cpu > 25m`，证明已取得负载样本。首次 target 50m 可能只是 `minAllowed` 的作用，不能单独作为负载学习的证据。
4. 只将 VPA 模式改为 Recreate，不人工删除 Pod，也不触发 Deployment rollout。
5. 等待两 Pod 自动替换，UID 都变化且 Ready，CPU requests 大于 25m；保存 `EvictedByVPA` 事件和最终建议。
6. 确认 Deployment 模板仍 request 25m，资源调整由 admission 写入新 Pod。
7. 保存 JSON 证据并删除本次测试命名空间，VPA 控制器保留。

此测试覆盖有负载时的建议、Off 的持续观察及 Recreate 自动调整，不覆盖 InPlaceOrRecreate、内存压力、OOM 恢复、长期推荐稳定性或控制器高可用。

## TLS 维护

首次安装生成有效期 365 天的 CA 和 webhook 服务证书，证书 SAN 为 `vpa-webhook.kube-system.svc`，保存在 `kube-system/vpa-tls-certs`。临时私钥文件权限受限，脚本退出后清理。重复安装保留 Secret，不自动续期。

可只读取公开证书的有效期：

```bash
kubectl -n kube-system get secret vpa-tls-certs \
  -o 'jsonpath={.data.serverCert\.pem}' | base64 -d | openssl x509 -noout -dates
```

在到期前安排证书轮换，同时更新 Secret、重启 admission controller，并验证 Webhook 的 `caBundle`、TLS 连接及新 Pod 注入。不要直接替换证书后省略这些验证，也不要把私钥写入日志或 Git。此版本安装器不提供无人值守证书轮换。

## 本地检查

```bash
python3 -B test_render.py
bash -n install.sh
```

渲染测试验证 worker 约束、admission 独立调度、资源上限、RBAC 引用、清单分阶段输出及非法参数拒绝。真实集群行为以测试 JSON 和[联合验证报告](2026-10-02-autoscaling-live-validation.md)为准。
