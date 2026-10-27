# VPA 1.6.0 安装与真实行为验证

本目录把 Kubernetes Autoscaler 官方 VPA 1.6.0 固定在提交 `9196162ba12e637ce4482107e1c17bcb80599050`，用于 Kubernetes 1.34–1.36。官方 YAML 保存在 `upstream/`，渲染前校验 SHA256；来源与本地修改见 [SOURCE.md](SOURCE.md)。

## 安装

在有集群管理权限、`kubectl`、Python 3.9+ / PyYAML、OpenSSL 的管理机运行。集群应已有可用的 Metrics API。先替换下面的 worker 主机名：

```bash
./install.sh check --workers worker-01,worker-02 --compact
./install.sh install --workers worker-01,worker-02 --compact
```

`check` 只读。`install` 创建 CRD、独立命名的 RBAC、TLS Secret 和三个 VPA 控制器；控制器只调度到指定 worker，拒绝接管其他工具管理的既有 VPA Deployment。小集群使用 `--compact`；规模较大时先评估 VPA 的历史样本与内存需求。

如果某 worker 已完整缓存 admission 镜像，可以显式设置 `--admission-worker worker-01`。该参数仍要求节点包含在 `--workers` 中。

## 验证

```bash
python3 -B test_render.py
bash -n install.sh

python3 -B run-e2e.py \
  --namespace autoops-vpa-e2e-example \
  --worker worker-01 \
  --image docker.io/library/alpine@sha256:REPLACE_WITH_CACHED_IMAGE_DIGEST \
  --output /tmp/vpa-e2e-result.json \
  --timeout 900 --cleanup
```

测试需要指定 worker 已缓存的 Alpine 镜像；先在节点用 `crictl images --digests` 确认完整摘要。测试创建两个 CPU 上限 250m、内存上限 128Mi 的 Pod；Off 模式至少观察 90 秒，取得真实负载建议并确认 UID/requests 不变，再切换 Recreate，验证自动驱逐、重建及资源调整。测试不会人工重启 Pod。`--cleanup` 删除本次新建的测试命名空间，保留 VPA 安装。

安装说明、证书维护及真实验证结果见 [VPA 操作指南](../../../../docs/2026-10-02-vpa-installation-and-validation.md)。
