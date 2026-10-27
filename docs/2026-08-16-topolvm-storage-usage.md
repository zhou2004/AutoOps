# TopoLVM 存储卷使用教程（界面操作 + 参数详解）

> 日期：2026-08-16　|　集群：K8s v1.34.3　|　存储类：`topolvm-provisioner`（TopoLVM CSI）

## 0. 先看结论：为什么你的 `test` PV 是 Available（不绑定）？

你手动在界面创建了一个 `test` PV（Local 类型，1Gi，绑定 worker-01），但一直
**Available（未绑定）**，界面显示"选不中"。根本原因有两点：

1. **`topolvm-provisioner` 是延迟绑定（WaitForFirstConsumer）存储类**
   这种模式下，PVC 必须先有一个 **Pod 消费它**，Pod 被调度到节点后，PVC 才会绑定 PV。
   你只创建了 PVC，没有创建使用它的 Pod，所以 PVC 一直 `Pending`，PV 自然无法绑定。

2. **手动创建 Local 静态 PV 不是 TopoLVM 的正确用法**
   TopoLVM 是 **CSI 动态供给**，正确用法是：**只创建 PVC**（选择 `topolvm-provisioner`），
   然后创建使用该 PVC 的 Pod/工作负载，TopoLVM 控制器会自动生成 PV（driver: `topolvm.io`）。
   不需要也不应该手动创建 PV。

> ✅ 实测验证：创建了一个挂载 `test` PVC 的 Pod 后（不指定 nodeName，让调度器调度），
> test PVC 立即从 `Pending` → `Bound`，test PV 从 `Available` → `Bound`。

---

## 1. TopoLVM 核心概念

### 1.1 动态供给（Dynamic Provisioning）— TopoLVM 的正确用法

```
用户创建 PVC ──> 创建 Pod 使用 PVC ──> 调度器选节点 ──> TopoLVM 控制器自动创建 PV ──> 绑定
```

- 你只需要关心 **PVC**（声明要多大、什么访问模式）
- TopoLVM 自动在节点的 LVM 卷组（`myvg`）上创建逻辑卷，并生成 PV 绑定
- PV 的所有参数（`csi.driver`、`volumeHandle`、`fsType` 等）都是**自动生成**的，无需填写

### 1.2 静态供给（Static Provisioning）— 手动创建 PV

适用于 HostPath / NFS / Local / CSI（已有卷句柄）等场景。**TopoLVM 场景不需要**。

---

## 2. 正确使用 TopoLVM（推荐，两种方式）

### 2.1 方式一：AutoOps 界面操作（推荐）

**第 1 步：创建 PVC**（存储管理 → PVC 标签页 → "创建 PVC"）

| 字段 | 填写 | 说明 |
|------|------|------|
| PVC 名称 | `my-app-data` | 自定义 |
| 存储类 | `topolvm-provisioner`（下拉选择） | 必须选 TopoLVM 的存储类 |
| 存储容量 | `5Gi` | 按需，建议 ≥ 1Gi |
| 访问模式 | `ReadWriteOnce` | TopoLVM 本地盘只支持 RWO |
| 卷模式 | `Filesystem` | 常规应用选文件系统 |

> ⚠️ 此时 PVC 会显示 `Pending`，**这是正常的**（等待消费者 Pod 绑定）。

**第 2 步：部署使用该 PVC 的应用**

在任意工作负载（Deployment/StatefulSet/普通 Pod）中挂载该 PVC：
- 界面：创建 Deployment 时选择"已有 PVC"并选中 `my-app-data`
- 或 YAML 方式（见下）

**第 3 步：等待绑定**

应用 Pod 调度到节点后，PVC 自动 `Bound`，Pod 正常挂载 `/data` 等路径使用存储。

### 2.2 方式二：kubectl YAML

```yaml
# 1) PVC（只声明，不创建 PV）
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-app-data
  namespace: default
spec:
  storageClassName: topolvm-provisioner
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 5Gi
```

```yaml
# 2) Deployment 使用 PVC（触发绑定）
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 1
  selector: { matchLabels: { app: my-app } }
  template:
    metadata: { labels: { app: my-app } }
    spec:
      containers:
        - name: app
          image: nginx:alpine
          volumeMounts:
            - name: data
              mountPath: /data
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: my-app-data
```

```bash
kubectl apply -f pvc.yaml -f deployment.yaml
kubectl get pvc,pv -A   # 观察 PVC 变 Bound，PV 自动生成
```

---

## 3. TopoLVM 自动生成的 PV 参数详解

以下是你看到的 TopoLVM PV spec 各字段含义（**全部自动生成，无需手动填**）：

```yaml
spec:
  accessModes:
    - ReadWriteOnce            # 访问模式（本地盘单节点读写）
  capacity:
    storage: 5Gi               # 容量（来自 PVC 请求，LVM 逻辑卷实际大小）
  claimRef:                    # 绑定的 PVC 引用（自动）
    apiVersion: v1
    kind: PersistentVolumeClaim
    name: data-test-mysql-1    # PVC 名称
    namespace: test-pv         # PVC 命名空间
    resourceVersion: "102924"  # 版本号（并发控制）
    uid: 5141356a-...          # PVC 的 UID
  csi:                         # CSI 存储源（TopoLVM 类型）
    driver: topolvm.io         # CSI 驱动名
    fsType: xfs                # 文件系统类型（可在 StorageClass 参数改，本集群用 ext4）
    volumeAttributes:
      storage.kubernetes.io/csiProvisionerIdentity: "1786788635122-16-topolvm.io"
                               # CSI provisioner 身份标识（自动）
    volumeHandle: 9a9ec130-759c-45e0-98b4-690c80e74edf
                               # 卷句柄 = LVM 逻辑卷的 UUID（lvmd 创建的真实卷）
```

**关键理解**：`volumeHandle` 对应节点上 `lvs myvg` 看到的逻辑卷 UUID。
例如 worker-02 上 `lvs myvg` 显示 `9a9ec130-759c-45e0-98b4-690c80e74edf 5.00g`，
正是这个 PV 的 `volumeHandle`。**数据实际存储在 LVM 逻辑卷上**。

---

## 4. 界面"创建 PV"各字段填写指南（静态供给场景）

> 提醒：**TopoLVM 不要用"创建 PV"**，用"创建 PVC"。以下供 HostPath/NFS/Local/CSI 静态卷参考。

### 4.1 HostPath（单机测试）
| 字段 | 说明 |
|------|------|
| 主机路径 | 节点上的目录，如 `/mnt/data` |
| 路径类型 | `DirectoryOrCreate` / `Directory` / `FileOrCreate` |

### 4.2 NFS（共享文件存储）
| 字段 | 说明 |
|------|------|
| NFS 服务器 | 如 `192.168.80.51` |
| NFS 路径 | 如 `/data/nfs` |
| 只读 | 按需开关 |

### 4.3 Local（本地持久卷，静态）
| 字段 | 说明 |
|------|------|
| 绑定节点 | **必选**，选择数据所在节点（生成 nodeAffinity） |
| 本地路径 | 节点上的已格式化目录，如 `/mnt/disks/vol1` |
| 文件系统 | `ext4` / `xfs` |
| 存储类 | 建议选 `kubernetes.io/no-provisioner` 的静态存储类（不要选 topolvm-provisioner） |

> ⚠️ Local 静态 PV 必须选"绑定节点"（否则 K8s 校验报 `Local volume requires node affinity`）。
> 且 Local 静态 PV 配 `WaitForFirstConsumer` 存储类时，仍需消费者 Pod 且调度到对应节点才绑定。

### 4.4 CSI（通用 CSI，已有卷句柄时）
| 字段 | 说明 |
|------|------|
| CSI 驱动 | 如 `topolvm.io` / `rook-ceph.rbd.csi.ceph.com` |
| 卷句柄 | 已存在的卷句柄（如 LVM 逻辑卷 UUID） |
| 文件系统 | `ext4` / `xfs` |

---

## 5. 常见问题（FAQ）

### Q1: 为什么 PVC 一直 Pending？
`topolvm-provisioner` 是 `WaitForFirstConsumer`。没有 Pod 使用该 PVC 时会一直 Pending。
创建并部署使用该 PVC 的应用后会自动绑定。

### Q2: 为什么我手动创建的 PV 是 Available（选不中）？
- 没有匹配的 PVC 请求它（容量/访问模式/存储类需匹配）
- 或 PVC 是 WaitForFirstConsumer 且没有消费者 Pod
- 或 Pod 手动指定了 `nodeName`（绕过调度器的 volume binding，导致 PVC 无法触发绑定）→ **不要给挂载 PVC 的 Pod 指定 nodeName**

### Q3: 手动创建 Local PV 配 topolvm-provisioner 可以吗？
**不建议**。`topolvm-provisioner` 是 CSI 动态供给存储类，Local 静态 PV 与它语义不匹配。
正确做法：Local 静态 PV 用 `kubernetes.io/no-provisioner` 存储类；TopoLVM 卷直接创建 PVC。

### Q4: 如何扩容？
界面：PVC 操作列 → 扩容，或 `kubectl patch pvc`（StorageClass 已开启 allowVolumeExpansion）。
LVM 逻辑卷会同步扩容，`df -h` 立即生效。

### Q5: 如何迁移/换卷？
存储管理 → 迁移按钮（rsync Job 复制数据到目标 PVC）→ 一键换卷（切换工作负载的 PVC）。

### Q6: 节点磁盘满了怎么办？
`vgs myvg` 查看剩余；给节点加盘后 `pvcreate` + `vgextend myvg`，TopoLVM 自动识别扩容。

---

## 6. 相关文件

| 文件 | 说明 |
|------|------|
| `k8s/storage/topolvm/topolvm-deploy.yaml` | StorageClass + MySQL StatefulSet 示例 |
| `k8s/storage/topolvm/bind-test-pvc.yaml` | WaitForFirstConsumer 绑定验证 Pod |
| `docs/2026-08-15-topolvm-guide.md` | 部署 + 维护完整文档 |
