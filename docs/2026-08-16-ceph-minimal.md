# Ceph + Rook 最小实现方案（测试集群实战）

> 日期：2026-08-16　|　集群：K8s v1.34.3 三节点　|　Rook v1.14 + Ceph v18.2.0

## 1. 当前节点磁盘现状（每节点新增 50G 闲置盘, TopoLVM 保留）

| 节点 | 角色 | 系统盘 | 新增闲置盘（Ceph OSD） | TopoLVM 盘 |
|------|------|--------|------------------------|-----------|
| master-01 (192.168.80.51) | 控制面 | sda 50G | **sdb 50G** | - |
| worker-01 (192.168.80.52) | worker | sdc 50G | **sdb 50G** | sda 20G |
| worker-02 (192.168.80.53) | worker | sda 50G | **sdb 50G** | sdc 20G |

**每节点各一块 50G 闲置盘（/dev/sdb）用于 Ceph OSD，TopoLVM 不受影响。**

## 2. 最小方案需要多少存储盘？

| 方案 | OSD 盘数 | 副本 | 说明 |
|------|---------|------|------|
| **本方案（3 副本生产最小）** | **3 块**（每节点 1 块 50G，共 3 个 OSD） | 3 副本 | 3 个 host 各 1 OSD，任一节点故障数据不丢 |
| 最小测试（2 副本） | 2 块 | 2 副本 | 仅 2 个 OSD，允许 1 个 OSD 故障 |

> 当前每节点 1 块 50G 盘 → 3 个 OSD → 可配置 **3 副本（size 3）**。

## 3. 部署架构

```
┌─────────────┬─────────────┬─────────────┐
│  master-01  │  worker-01  │  worker-02  │
│  MON + OSD  │  MON + OSD  │  MON + OSD  │
│  sdb 50G    │  sdb 50G    │  sdb 50G    │
│             │  (TopoLVM   │  (TopoLVM   │
│             │   sda 20G)  │   sdc 20G)  │
└─────────────┴─────────────┴─────────────┘
   MON×3（3节点）  OSD×3（每节点1个）  副本 size 3
```

## 4. 部署步骤

### 4.0 前置：containerd 配置镜像源（quay.io / registry.k8s.io）

Ceph/Rook 镜像来自 quay.io 和 registry.k8s.io，需配置国内镜像源（3 节点都要）：

```bash
# quay.io（Ceph 主镜像）
mkdir -p /etc/containerd/certs.d/quay.io
cat > /etc/containerd/certs.d/quay.io/hosts.toml << 'EOF'
server = "https://quay.io"
[host."https://quay.m.daocloud.io"]
  capabilities = ["pull", "resolve"]
[host."https://quay.nju.edu.cn"]
  capabilities = ["pull", "resolve"]
EOF

# registry.k8s.io（CSI 插件镜像）
mkdir -p /etc/containerd/certs.d/registry.k8s.io
cat > /etc/containerd/certs.d/registry.k8s.io/hosts.toml << 'EOF'
server = "https://registry.k8s.io"
[host."https://k8s.m.daocloud.io"]
  capabilities = ["pull", "resolve"]
EOF
systemctl restart containerd
```

### 4.1 安装 Rook Operator

```bash
ROOK_VERSION="v1.14.0"
# github raw 大文件慢可用 jsdelivr: https://cdn.jsdelivr.net/gh/rook/rook@v1.14.0/...
kubectl apply -f "https://raw.githubusercontent.com/rook/rook/${ROOK_VERSION}/deploy/examples/crds.yaml"
kubectl apply -f "https://raw.githubusercontent.com/rook/rook/${ROOK_VERSION}/deploy/examples/common.yaml"
kubectl apply -f "https://raw.githubusercontent.com/rook/rook/${ROOK_VERSION}/deploy/examples/operator.yaml"
kubectl -n rook-ceph wait --for=condition=Ready pod -l app=rook-ceph-operator --timeout=300s
```

### 4.2 部署 Ceph 集群（适配本集群，3 OSD + 3 副本）

```bash
kubectl create namespace test-pv --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f k8s/storage/ceph-rook/rook-ceph-deploy.yaml
```

> ⚠️ 关键修正（已内置在 yaml）：
> - **tolerations 用 `spec.placement.all.tolerations`**（Rook v1.14 不支持 `spec.tolerations`），否则 master 上 MON/OSD 无法调度
> - CephFS StorageClass 的 pool 名称是 **`myfs-myfs-data0`**（Rook 实际创建的池，格式 `<fsname>-<poolname>`），不是 `myfs-data0`
>
> Ceph 集群初始化需 **10-20 分钟**（拉镜像、创建 OSD/MON）。若 PVC 报 secret not found，删除 PVC 重新创建即可（CSI secret 稍后才生成）。

### 4.3 验证

```bash
# 集群状态（HEALTH_OK）
kubectl get cephcluster -n rook-ceph -o jsonpath='{.status.ceph.health}'
# 完整状态（需 tools pod）
kubectl -n rook-ceph apply -f https://raw.githubusercontent.com/rook/rook/v1.14.0/deploy/examples/toolbox.yaml
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph -s
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph osd tree

# 存储类与 PVC
kubectl get sc
kubectl get pvc -n test-pv
```

## 4.4 实际验证结果（本集群）

```
cluster: health HEALTH_OK
  mon: 3 daemons, quorum a,b,c
  mgr: a(active)
  osd: 3 osds: 3 up, 3 in
  volumes: 1/1 healthy (CephFS)
  pools: 4 pools, 81 pgs

OSD tree（三副本，分布在 3 个 host）:
  host master-01 → osd.1 (sdb 50G)
  host worker-01 → osd.0 (sdb 50G)
  host worker-02 → osd.2 (sdb 50G)

PVC:
  test-pvc-ceph-rbd   Bound 5Gi RWO  rook-ceph-block  （RBD，数据写入 /dev/rbd0 成功+持久化）
  test-pvc-cephfs     Bound 2Gi RWX  rook-cephfs      （CephFS 共享读写成功）
```

## 5. 测试应用（RBD 块）

```bash
# 创建使用 rook-ceph-block 的 PVC（rook-ceph-deploy.yaml 已含 test-pvc-ceph-rbd）
kubectl get pvc test-pvc-ceph-rbd -n test-pv

# 部署 MySQL 测试（共享示例，与 TopoLVM 用法一致，只需换存储类）
kubectl apply -f - <<'EOF'
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: ceph-mysql
  namespace: test-pv
spec:
  serviceName: ceph-mysql
  replicas: 1
  selector: { matchLabels: { app: ceph-mysql } }
  template:
    metadata: { labels: { app: ceph-mysql } }
    spec:
      containers:
        - name: mysql
          image: swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/mysql:8.0.40
          env:
            - name: MYSQL_ROOT_PASSWORD
              value: "123456"
          volumeMounts:
            - name: data
              mountPath: /var/lib/mysql
  volumeClaimTemplates:
    - metadata: { name: data }
      spec:
        storageClassName: rook-ceph-block
        accessModes: [ReadWriteOnce]
        resources:
          requests: { storage: 5Gi }
EOF
```

## 6. 在 AutoOps 界面使用

Ceph 部署完成后，存储管理界面会自动出现新的 StorageClass：
- `rook-ceph-block`（RBD 块，单节点读写）
- `rook-cephfs`（CephFS，共享读写）

创建 PVC 时下拉选择 `rook-ceph-block` 即可（与 TopoLVM 操作一致）。

## 7. 注意事项

- **数据安全**：释放 TopoLVM 盘会删除其数据，请先备份（若有重要数据）
- **2 副本限制**：2 副本允许 1 个 OSD 故障；若需 3 副本，需 3+ 节点各一块盘
- **磁盘大小**：20G 仅测试够用；生产建议 OSD 每盘 ≥ 100G，且使用专用盘
- **镜像拉取**：Rook/Ceph 镜像来自 quay.io / registry.k8s.io，需配置镜像源或代理
