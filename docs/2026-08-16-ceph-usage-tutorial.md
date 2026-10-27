# AutoOps Ceph 分布式存储使用教程

> 适用版本：Ceph 18.2.2 (Reef) / Rook v1.14 / AutoOps Ceph 管理模块
> 配套文档：`docs/2026-08-16-ceph-higher-level.md`（原理/架构/运维/排障）
> 2026-10-01 更新：[存储维护中心完整说明](2026-10-01-ceph-maintenance.md)，包含迁移、全量/增量备份与恢复、定时计划、CephFS 子卷和 RGW 维护。

本教程包含基础操作和业务接入；新版维护中心见配套说明：
1. **界面操作**：通过 AutoOps「K8s 存储管理 → Ceph 存储」完成日常管理
2. **业务使用**：让 K8s 应用通过 StorageClass/PVC 真正用上 Ceph 存储（块存储 RBD、文件存储 CephFS）
3. **维护中心**：图形化迁移、持久化备份恢复、定时计划和任务日志
4. **命令行**：在 rook-ceph-tools 中执行 `ceph`/`rbd` 命令供排障和理解操作语义

---

## 一、AutoOps 界面操作教程

入口：**K8s 集群 → 存储管理 → Ceph 存储** 标签页。

### 1.1 集群总览（顶部区域）

| 项目 | 说明 |
|---|---|
| 集群状态 | `HEALTH_OK` 正常 / `HEALTH_WARN` 有告警（看"监控诊断"定位）/ `HEALTH_ERR` 故障 |
| 运行阶段 | CephCluster CR 的 `phase`，`Ready` 表示就绪 |
| MON/OSD/PG 数量 | 基础规模指标 |
| 集群容量 | 总容量 / 已使用 / 可用 + 使用率进度条（>80% 变红，需扩容或清理） |

### 1.2 存储池（Pool）管理

存储池是 Ceph 的逻辑分区，每个 Pool 有独立的副本数、PG 数、配额、压缩策略。

- **创建 Pool**：点「创建 Pool」→ 填名称 / PG 数 / 副本数 / 应用类型。
  - PG 数估算：`(OSD 数量 × 100) ÷ 副本数`，例如 3 OSD 三副本 ≈ 100，测试环境 32 即可。
  - 应用类型：`rbd`（块存储）、`cephfs`（文件系统）、`rgw`（对象存储）。
- **调整副本数**（⚙ 按钮）：`size` 值。`size=3/min_size=2` 通常允许单副本故障时继续 I/O；故障影响还取决于 CRUSH 故障域和 PG 分布。**3 节点集群请保持 3**，降到 2 会降低可靠性。
- **调整 PG 数**（表盘按钮）：增加会触发 PG 分裂与数据重均衡，建议维护窗口操作。
- **设置配额**（文件按钮）：可限制容量上限（GB）和对象数上限，防止 Pool 写满拖垮集群，0 表示不限制。
- **设置压缩**（数据线按钮）：`lz4/snappy/zlib/zstd` 算法 + `none/passive/aggressive/force` 模式。适合日志、文本等可压缩数据，显著省空间。
- **删除 Pool**（垃圾桶按钮）：**危险操作**，会清空其中所有数据。系统保留池（`.` 开头）不可删。

### 1.3 RBD 块存储管理

RBD 是裸块设备（类似云盘的虚拟磁盘），可被 K8s PVC、虚拟机等使用。

- **创建镜像**：选 Pool → 镜像名 → 容量（如 `10G`/`5T`/`1024M`）。采用 thin provisioning，创建不占实际空间。
- **镜像列表**：显示所有 Pool 中的镜像，点击镜像名进入详情。
- **详情 + 快照管理**：
  - 详情含：容量、**实际使用量**、对象数、特性（layering/exclusive-lock/object-map/fast-diff/deep-flatten 等）。
  - **创建快照**：镜像某一时刻的只读副本，可用于备份与恢复。
  - **回滚**（回）：把镜像内容恢复到某快照（会覆盖当前数据，**镜像不能正被挂载使用**）。
  - **克隆**（克）：基于**已保护**快照做写时复制（COW）子镜像，初始不占额外空间。克隆前可在“维护中心 → RBD 快照 → 保护快照”操作。
  - **删除快照**（删）：有克隆子镜像的受保护快照需先取消保护。
- **扩容/缩容**（⇅ 按钮）：CSI 镜像请通过 PVC 扩容。独立 RBD 镜像缩容需先卸载、缩小内部文件系统并确认分区边界，不能仅根据实际使用量判断安全；后端会检查 watchers。
- **删除镜像**：**危险操作**，有快照时需先删快照。

> 界面创建的 RBD 镜像与 K8s 的 `rook-ceph-block` StorageClass 是同一套底层（RBD pool）。K8s 通过 PVC 自动创建的镜像也会出现在这里。

### 1.4 CephFS 文件系统

共享文件系统（POSIX 语义），多个 Pod 可同时读写同一份数据，适合日志聚合、共享存储。

- 展示每个文件系统（如 `myfs`）的元数据 Pool、数据 Pool、MDS 状态（active）和 MDS 数量。
- “维护中心”支持子卷创建/配额/快照/克隆/删除及克隆进度查询，CSI 子卷请通过 PVC 管理。

### 1.5 OSD 管理

OSD 是真正的数据盘，分布在各节点（本例 master-01/worker-01/worker-02 各一块 50G）。

- **入 / 出**：`in`（加入集群参与数据分布）/ `out`（移出，数据会迁移到其他 OSD）。
  - 维护单块盘、更换磁盘前先 `out`，等数据迁完再操作。
- **停**：`down`（标记 OSD 进程故障，通常由系统自动触发，手动慎用）。
- **调整权重**（⚙）：0–1 之间，`0` 表示不接收新数据。可配合 `out` 做数据迁移。

### 1.6 监控诊断

- **集群健康**：`HEALTH_OK`/`WARN`/`ERR` + 具体检查项（如 `OSD_DOWN`、`PG_DEGRADED`）。
- **组件版本**：mon/mgr/osd/mds 各组件版本与数量。
- **PG 状态**：PG 总数与状态分布，`active+clean` 为健康，出现 `degraded`/`peering` 需关注。

---

## 二、让 K8s 应用使用 Ceph 存储

集群已提供两个 Ceph StorageClass（来自 Rook CSI 驱动）：

| StorageClass | 类型 | Provisioner | 特点 |
|---|---|---|---|
| `rook-ceph-block` | 块存储 RBD | `rook-ceph.rbd.csi.ceph.com` | 单读写、性能好，适合数据库/有状态应用 |
| `rook-cephfs` | 文件系统 CephFS | `rook-ceph.cephfs.csi.ceph.com` | 多读多写共享，适合共享文件场景 |

### 2.1 使用块存储（RBD）

**步骤 1：创建 PVC**

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: mysql-data
  namespace: default
spec:
  accessModes:
    - ReadWriteOnce          # RBD 通常单节点读写
  storageClassName: rook-ceph-block
  resources:
    requests:
      storage: 10Gi          # 后续可在线扩容（该 SC 支持扩容）
```

**步骤 2：Pod 挂载使用**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mysql
  namespace: default
spec:
  selector:
    matchLabels: { app: mysql }
  template:
    metadata:
      labels: { app: mysql }
    spec:
      containers:
        - name: mysql
          image: mysql:8.0
          volumeMounts:
            - name: data
              mountPath: /var/lib/mysql
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: mysql-data
```

### 2.2 使用文件存储（CephFS）

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: nfs-share
  namespace: default
spec:
  accessModes:
    - ReadWriteMany         # CephFS 支持多节点同时读写
  storageClassName: rook-cephfs
  resources:
    requests:
      storage: 20Gi
```

多个 Deployment 引用同一个 `nfs-share` PVC 即可共享同一份数据（日志聚合、Web 上传目录、共享配置等）。

### 2.3 在 AutoOps 界面创建 PVC

「K8s 存储管理 → PVC → 创建 PVC」：
1. 选择命名空间
2. 填名称
3. 存储类选 `rook-ceph-block` 或 `rook-cephfs`（会自动读取集群中的 StorageClass）
4. 填容量（GiB），访问模式按存储类型选择

创建后可在「Ceph 存储 → RBD 块存储」中看到由 CSI 自动创建的 RBD 镜像（命名 `csi-vol-xxx`）。

### 2.4 扩容在线卷

`rook-ceph-block` 支持在线扩容（`allowVolumeExpansion: true`）：
- AutoOps：PVC 列表操作栏「扩容」按钮修改容量；
- 或手动：`kubectl patch pvc mysql-data -p '{"spec":{"resources":{"requests":{"storage":"20Gi"}}}}'`。

扩容后文件系统层面需同步扩展（如 `resize2fs`，由 CSI 自动处理块设备层）。

---

## 三、命令行高级操作（rook-ceph-tools）

进入工具 Pod：

```bash
kubectl -n rook-ceph exec -it deploy/rook-ceph-tools -- bash
```

### 3.1 集群健康与容量

```bash
ceph status                 # 集群状态
ceph health detail          # 详细健康检查（定位 WARN/ERR 原因）
ceph df                     # 整体容量
ceph df detail              # 各 Pool 用量
ceph versions               # 组件版本分布
ceph quorum_status          # MON 选举状态
```

### 3.2 Pool 操作

```bash
# 创建
ceph osd pool create mypool 32 replicated
ceph osd pool application enable mypool rbd     # 关联应用
ceph osd pool set mypool size 3                 # 副本数
ceph osd pool set mypool pg_num 64              # PG 数

# 配额与压缩
ceph osd pool set-quota mypool max_bytes 10737418240   # 10GB
ceph osd pool set mypool compression_algorithm lz4
ceph osd pool set mypool compression_mode aggressive

# 删除（需要集群允许删除池）
ceph osd pool delete mypool mypool --yes-i-really-really-mean-it
```

### 3.3 RBD 块设备

```bash
# 创建 / 查看 / 扩容
rbd create replicapool/mydisk --size 10G
rbd ls -l --pool replicapool
rbd info replicapool/mydisk
rbd resize replicapool/mydisk --size 20G
rbd resize replicapool/mydisk --size 8G --allow-shrink   # 缩容（慎用）

# 快照：备份 / 恢复 / 克隆
rbd snap create replicapool/mydisk@bak-20260816
rbd snap ls replicapool/mydisk
rbd snap rollback replicapool/mydisk@bak-20260816   # 恢复（镜像不能正被使用）
rbd snap protect replicapool/mydisk@bak-20260816    # 保护后可用于克隆
rbd clone replicapool/mydisk@bak-20260816 replicapool/clone-disk   # 写时复制克隆
rbd snap unprotect replicapool/mydisk@bak-20260816  # 取消保护
rbd snap rm replicapool/mydisk@bak-20260816

# 删除镜像（有快照先删快照）
rbd rm replicapool/mydisk
```

### 3.4 备份与恢复

推荐使用“维护中心 → 备份仓库”和“维护操作”，将数据保存到独立 RWX PVC，而非 toolbox 的 `/tmp`。操作方法见[全量、增量和恢复](2026-10-01-ceph-maintenance.md#4-全量增量和恢复)。

下面演示与系统一致的 diff 备份链；`/backup` 必须是持久化目录。数据库先由应用完成停写或一致性处理。

```bash
# 完整基线：不带 --from-snap，包含结束快照元数据
rbd snap create replicapool/mydisk@base-20261001
rbd export-diff replicapool/mydisk@base-20261001 /backup/base.diff

# 后续增量：保留父快照
rbd snap create replicapool/mydisk@inc-20261002
rbd export-diff --from-snap base-20261001 replicapool/mydisk@inc-20261002 /backup/inc.diff

# 恢复到新镜像，依次导入，容量以源镜像实际大小为准
rbd create replicapool/mydisk-restored --size 10G
rbd import-diff /backup/base.diff replicapool/mydisk-restored
rbd import-diff /backup/inc.diff replicapool/mydisk-restored
```

单独 `rbd export/import` 的原始镜像文件可用于全量数据拷贝；若使用这种基线，不能假设它自动恢复了后续 `import-diff` 所需的父快照。维护中心统一使用 diff 基线及清单校验来保持链连续性。

### 3.5 OSD 维护

```bash
ceph osd tree                 # OSD 树（节点归属）
ceph osd df                   # 各 OSD 容量
ceph osd out 0                # 移出 OSD.0（触发数据迁移）
ceph osd in 0                 # 重新加入
ceph osd reweight 0 0.5       # 调整权重
ceph osd perf                 # OSD 性能
# 彻底移除前必须检查 safe-to-destroy，并同步处理 Rook CR 和对应 OSD 资源
ceph osd safe-to-destroy 0
ceph osd purge 0 --yes-i-really-mean-it
```

---

## 四、最佳实践与常见问题

### 最佳实践
1. **三副本 + min_size 2**：3 节点集群默认配置，可容忍单点故障同时保障可用性。
2. **PG 数规划**：目标约 100–150 PG/OSD，总 PG 不要一次加太多（每次翻倍为佳）。
3. **定期快照**：重要业务（数据库）建议每日快照，保留最近 N 份。
4. **监控使用率**：集群容量 > 70% 开始规划扩容，> 85% 必须处理（nearfull/full 会阻塞写入）。
5. **OSD 维护流程**：`out` → 等 `ceph -s` 恢复 `HEALTH_OK` → 维护/换盘 → `in`。

### 常见问题
| 现象 | 处理 |
|---|---|
| `HEALTH_WARN: OSD_DOWN` | 到对应节点查 OSD 进程/磁盘，恢复后 `ceph osd in` |
| `HEALTH_WARN: PG_DEGRADED` | 等待数据重均衡；长时间不恢复查是否有 OSD 长期 out |
| `PG_DEGRADED` 卡住 | 检查 OSD、容量、网络及 peering 原因；只有定位到适合 repair 的不一致问题时才执行 PG repair |
| 集群 `nearfull` | 清理旧数据、删除无用镜像/快照、扩容 |
| PVC 一直 Pending | 检查 StorageClass、配额、Pool 是否写满 |
| 删除 Pool 失败 | 需在 CephCluster CR 开启 `mon_allow_pool_delete` |
| 快照无法删除 | 快照受保护，先 `rbd snap unprotect` |

---

## 五、Rook/Ceph 官方文档参考

- Rook Ceph 部署与 CRD：<https://rook.github.io/docs/rook/latest/>
- Ceph Pool 管理：<https://docs.ceph.com/en/reef/rados/operations/pools/>
- RBD 块设备命令：<https://docs.ceph.com/en/reef/rbd/rados-rbd-cmds/>
- Rook 块存储（StorageClass/PVC）：Rook 文档 Storage Configuration → Block Storage (RBD)
- Rook 共享文件系统（CephFS）：Rook 文档 Storage Configuration → Shared Filesystem (CephFS)
