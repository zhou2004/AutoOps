# containerd 国内镜像源配置（TopoLVM 部署前置）

TopoLVM 与 MySQL 镜像分别托管在 `ghcr.io` 与 `docker.io`。国内网络直连拉取会卡住/极慢，
需为 containerd 配置国内镜像源。本集群 3 个节点（master-01/worker-01/worker-02）均已配置。

## 配置方式（containerd v2, version=3, config_path 模式）

containerd 使用 `/etc/containerd/certs.d/<registry>/hosts.toml` 做 registry 镜像源映射。

### 1. ghcr.io（TopoLVM 镜像）

```bash
mkdir -p /etc/containerd/certs.d/ghcr.io
cat > /etc/containerd/certs.d/ghcr.io/hosts.toml << 'EOF'
server = "https://ghcr.io"

[host."https://ghcr.m.daocloud.io"]
  capabilities = ["pull", "resolve"]

[host."https://ghcr.nju.edu.cn"]
  capabilities = ["pull", "resolve"]
EOF
```

### 2. docker.io（MySQL / 通用应用镜像）

```bash
cat > /etc/containerd/certs.d/docker.io/hosts.toml << 'EOF'
server = "https://docker.io"

[host."https://docker.m.daocloud.io"]
  capabilities = ["pull", "resolve"]

[host."https://docker.1ms.run"]
  capabilities = ["pull", "resolve"]

[host."https://hub1.nat.tf"]
  capabilities = ["pull", "resolve"]

[host."https://hub.rat.dev"]
  capabilities = ["pull", "resolve"]
EOF
```

> 说明：以下源经验证**不可靠/不可用**，请勿加入配置：
> `docker.awsl9527.cn`(超时)、`docker.1panel.live`(超时)、`proxy.1panel.live`(DNS 失败)、
> `docker.anyhub.us.kg`(超时)、`docker.chenby.cn`(DNS 失败)、`dockerhub.jobcher.com`(DNS 失败)、
> `docker.ketches.cn`(502)、`hub.uuuadc.top`(manifest 校验失败)、`docker.hpcloud.cloud`(FailedPrecondition)、
> `atomhub.openatom.cn`(NotFound)

### 3. 应用配置

```bash
systemctl restart containerd && systemctl is-active containerd
```

## 验证

```bash
# 拉取 TopoLVM 镜像（~216MB）
crictl pull ghcr.io/topolvm/topolvm-with-sidecar:0.41.0
# 拉取 MySQL（推荐直接用华为云国内镜像，见下）
crictl pull swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/mysql:8.0.40
```

## 大镜像推荐：华为云 SWR（国内源，快）

`docker.io/mysql:8.0` 体积大（~550MB），即使走镜像源下载也易超时。
**推荐直接使用华为云 SWR 上的同步镜像**：

```
swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/mysql:8.0.40
```

实测 40 秒内拉取完成（网络直连华为云，无需走镜像源）。
