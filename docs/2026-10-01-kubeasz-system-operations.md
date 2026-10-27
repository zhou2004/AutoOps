# kubeasz 系统运维接入指南

## 1. 功能入口和范围

入口：**任务管理 → Ansible 任务 → kubeasz 系统运维**。需要启用状态的系统 `admin` 角色。该入口使用固定的系统模板、源码目录和独立 Ansible 运行时；页面可以查看就绪检查、目标、实际命令、操作历史和日志。

| 操作 | 实际执行方式 | 说明 |
| --- | --- | --- |
| 创建集群配置 | `ezctl new <cluster>`，随后写入结构化 inventory | 只创建本地文件，不连接节点 |
| 全部 / 分步安装 | `ezctl setup <cluster> all/01…07` | 证书、etcd、containerd、控制面、工作节点、网络、插件 |
| 安装服务 | 固定 wrapper 导入官方 `cluster-addon` 中指定任务 | Metrics Server、CoreDNS、Local Path、Prometheus、Ingress NGINX |
| 工作节点扩容 / 删除 | `ezctl add-node/del-node <cluster> <IPv4>` | 每次一个节点，确认词包含节点地址 |
| 控制面扩容 / 删除 | `ezctl add-master/del-master <cluster> <IPv4>` | 禁止删除最后一个控制面 |
| etcd 扩容 / 删除 | `ezctl add-etcd/del-etcd <cluster> <IPv4>` | 删除后至少保留 3 个成员 |
| etcd 备份 | `ezctl backup <cluster>` | 在集群 `backup` 目录生成快照 |
| etcd 恢复 | `ansible-playbook … -e '{"db_to_restore":"snapshot_….db"}' playbooks/95.restore.yml` | 只能选择本集群 backup 目录内的普通快照文件 |
| 服务启停 | `ezctl start/stop <cluster>` | 集群维护操作 |
| 组件升级 | `ezctl upgrade <cluster>` | 使用管理员预先审核、准备好的二进制和配置 |

系统接口不接受自定义仓库、shell、Playbook 路径、SSH 密码或私钥。节点、集群名、安装步骤和插件有服务端校验；不存在普通 Git 任务自动转换为系统任务的流程。

### 固定版本

- kubeasz：`3.6.8`。
- 提交：`0e9cee984c211e9e9f9e64910206baab9fbd3d8e`。
- 该版本默认 Kubernetes 资源包：`v1.34.1`，生产环境应另外审核所用补丁版本。
- 源码归档 SHA256：`4710380c801caba3e157a0589e62dd839452f8826b5fcae921f44699ccb72c18`。
- `ezctl` SHA256：`e5d7837102d2090f76775e277f629c3fa128f0295b0ce6c5963b2c0f1d96b76b`。
- 系统执行器：独立 `/opt/kubeasz-venv`，`ansible==11.13.0`、`ansible-core==2.18.18`，Python 3.11–3.13。普通 Ansible 任务保留原运行时。

版本选择按 [kubeasz 官方版本对照](https://github.com/easzlab/kubeasz#readme) 对应 Kubernetes 1.34；动作映射以 [3.6.8 的 ezctl](https://github.com/easzlab/kubeasz/blob/3.6.8/ezctl) 和 [恢复 Playbook](https://github.com/easzlab/kubeasz/blob/3.6.8/playbooks/95.restore.yml) 为准。升级系统模板需要同时更新版本、提交、两项摘要并重新验证，页面不自动拉取主分支。

## 2. 部署准备

### 2.1 构建包含系统源码的 API 镜像

在仓库根目录：

```bash
docker build -f docker/api/Dockerfile -t devops-api:latest .
```

Dockerfile 调用 `docker/kubeasz/install-system.sh`，按固定提交下载源码、核对归档摘要、解压到 `/etc/kubeasz`，并生成五个只引用固定官方 role 的插件 Playbook。此步骤不会运行集群安装或连接节点。

构建需要访问 GitHub codeload 和 Python 包索引。受限网络应把同一归档及 Python 包同步到可信制品源，保留摘要校验；不要改成未经审核的 `master`。

源码安装脚本拒绝覆盖已有 `/etc/kubeasz/ezctl`。容器应由镜像提供源码，仅挂载下文列出的资源和状态子目录；整目录挂载会覆盖镜像自带的版本标记和插件模板。

### 2.2 准备二进制和离线镜像

源码内置不等于集群可安装。需要与目标 CPU 架构一致的 Kubernetes / etcd / cfssl / containerd / crictl 工具，以及网络、DNS、Metrics Server 等镜像。页面会列出缺失文件；插件 Prometheus、Ingress NGINX 还需要 `bin/helm`。

在**专用资源准备机**上按官方 [离线部署指南](https://github.com/easzlab/kubeasz/blob/3.6.8/docs/setup/offline_install.md) 准备。以下命令是部署步骤，开发和测试本功能时没有执行：

```bash
# 使用上述固定源码内的 ezdown。-D 会安装/使用 Docker、准备资源并启动本地镜像仓库。
cd /etc/kubeasz
./ezdown -D
# 如使用其他网络或高级插件，再准备对应镜像（以该版本 ezdown 帮助为准）：
./ezdown -X cilium
./ezdown -X prometheus
./ezdown -X ingress-nginx
```

`ezdown -D` 会下载上游的源码及工具，不要在正在执行任务的 API 容器中运行它。将准备机 `/etc/kubeasz/bin` 和 `/etc/kubeasz/down` 的内容复制到执行器资源卷，保留可执行权限，并核对架构、组件版本和镜像清单。不要覆盖 API 镜像中的 `ezctl`、`roles`、`playbooks` 和版本标记。

界面的镜像就绪项只验证 `down` 中存在 tar 资源，不验证每个节点能拉到镜像，也不能替代离线包完整性检查。默认新配置关闭 `ENABLE_MIRROR_REGISTRY`：联网环境使用上游镜像源；离线环境必须由部署人员在 `clusters/<cluster>/config.yml` 配置可达的镜像仓库，或按官方流程在目标节点预加载对应镜像。需要离线系统包时，准备 `down/packages` 并按目标系统配置。

### 2.3 SSH、持久化及网络

推荐单实例专用执行器。API 进程本身需要能访问目标 SSH、Kubernetes API 和 etcd，并以 kubeasz 官方要求的权限管理本地工具目录。不要把执行器放在它将要停止或恢复的同一个集群中。

| 路径 | 挂载内容 | 权限 / 生命周期 |
| --- | --- | --- |
| `/etc/kubeasz/bin` | 已审核的二进制 | 可写；上游 prepare 会设置工具权限 |
| `/etc/kubeasz/down` | 离线镜像和系统包 | 可只读 |
| `/etc/kubeasz/clusters` | inventory、config、证书、kubeconfig、etcd 快照 | 持久化，目录 0700，限制管理人员访问 |
| `/etc/kubeasz/.autoops` | 操作锁和脱敏日志 | 持久化，目录 0700，文件 0600 |
| `/root/.ssh` | 私钥和已核验 `known_hosts` | 只读挂载；私钥 0600 或 0400 |
| MySQL `task_kubeasz_operation` | 参数、状态、执行人、命令、核查记录 | 随数据库备份 |

SSH 使用执行器默认密钥 `id_ed25519` / `id_rsa`，也支持部署侧 SSH agent；页面不上传密钥。核验目标指纹后准备 `known_hosts`。执行器强制 `StrictHostKeyChecking=yes` 和 `BatchMode=yes`；未登记主机或密钥不匹配会失败。非 root SSH 用户必须配置免密 sudo。

不要使用仓库内示例私钥作为生产凭据。Docker Compose 已有 `/root/.ssh:ro` 挂载，部署时替换为你自己的专用运维密钥和 `known_hosts`。

创建资源与状态目录、放入资源后，使用附带覆盖文件：

```bash
mkdir -p /srv/autoops-kubeasz/bin /srv/autoops-kubeasz/down
install -d -m 700 /srv/autoops-kubeasz/clusters /srv/autoops-kubeasz/state
docker compose -f docker/docker-compose.yml \
  -f docker/kubeasz/docker-compose.override.yml up -d devops-api
```

Kubernetes 部署参考 `k8s/kubeasz/api-patch.example.yaml`。先创建并填充 `autoops-kubeasz-artifacts` PVC 的 `bin/down` 子目录、创建 `autoops-kubeasz-state` PVC，以及含私钥和 known_hosts 的 `autoops-kubeasz-ssh` Secret。补丁保留 `replicas: 1` / `Recreate`。这里不使用本地 Docker socket，也不自动申请管理目标集群的 ServiceAccount 权限。

新建集群时选择不与节点、执行器容器网络、VPN 或现有集群重叠的 Pod / Service CIDR。服务端检查两个 CIDR 彼此及与节点地址的冲突；执行器其他网络仍需人工规划。

## 3. 页面操作流程

1. 打开 **Ansible 任务 → kubeasz 系统运维**，展开环境检查，按提示准备缺失资源。
2. 选择“创建集群配置”，填写控制面、工作节点、etcd IPv4、SSH 用户 / 端口和网络。etcd 新建数量必须是 1、3、5、7，生产环境通常使用 3 或 5。
3. 点击“预览目标与命令”，检查所有目标和生成的 inventory，完整输入确认词后执行。配置名称仅允许小写字母开头，后续使用字母、数字和连字符。
4. 选择“安装集群 / 分步安装”，选择刚创建的集群，按需执行 `01` 到 `07`，或 `all`。每次执行重新预览并确认。
5. 在历史页查看状态和日志。成功后，将该集群 kubeconfig 通过项目原有 K8s 集群管理功能接入，随后使用工作负载、HPA / KPA / VPA、存储等维护页面。

现有 kubeasz 集群可以由管理员导入完整 `clusters/<name>` 目录，包括 `hosts`、`config.yml`、`ssl` 和 `kubectl.kubeconfig`。系统只枚举合法名称且路径不含符号链接的目录；浏览器只看到节点角色和快照文件名，不获取 inventory 中的凭据。执行时仅支持以 IPv4 开头的官方 INI 主机行；别名、children、IP 范围、动态 inventory、ansible_host 地址重映射和连接类型覆盖会被拒绝，避免预览目标与 SSH 目标不一致。

### 安装服务插件

选择“安装服务插件”，选择目标和插件后预览。系统只导入对应官方任务文件，命名空间、存储类、版本等配置读取该集群的 `config.yml`。

- Metrics Server 为 HPA CPU / 内存指标提供依赖；安装后仍需确认聚合 API 和节点证书访问正常。
- Local Path 使用节点本地目录，不能代替跨节点高可用存储。
- Prometheus 依赖可用 StorageClass 和相关镜像；组件证书来自该 kubeasz 集群目录。
- Ingress NGINX 的官方任务只部署到带 `ingress-controller/provider=ingress-nginx` 标签的节点，先配置该标签并检查网络入口。
- KPA / VPA 控制器的安装与版本由对应自动伸缩文档说明；这五个服务模板没有隐式安装 Knative 或 VPA。

### 扩缩容和升级

添加节点前，准备目标系统、SSH 密钥、known_hosts、免密 sudo、路由及镜像访问；输入的是新节点 IP，不能追加 Ansible 参数。增加主节点会重新配置负载均衡，建议在维护窗口执行。

删除节点前迁移业务、检查 PDB、DaemonSet、静态 Pod、emptyDir 和本地 PV。官方删除脚本使用 drain 并清理节点服务；其部分清理子步骤会忽略错误。因此“命令完成”表示外层进程退出成功，**不证明节点磁盘和服务已经全部清理**。删除后检查 Kubernetes Node、inventory、目标机服务及磁盘，不能仅凭绿色状态回收主机。

etcd 更换节点建议先增加新成员、验证同步健康后再删除旧成员；平台要求删除后至少保留 3 个成员。不能删除最后一个控制面。

升级操作不下载组件，也不自行决定目标 Kubernetes 版本。管理员先备份 etcd、核对 Kubernetes 版本偏差规则和 kubeasz 官方 [升级步骤](https://github.com/easzlab/kubeasz/blob/3.6.8/docs/op/upgrade.md)，准备对应 `bin` 资源并更新 `config.yml`，再执行升级模板。升级后核查所有节点与控制面健康。

### 备份和恢复

“备份 etcd”保存在 `clusters/<name>/backup`，包含 Kubernetes API 对象；它不备份业务 PV、Ceph 对象或数据库事务数据。业务数据使用存储模块 / 应用备份方案，并把 etcd 快照和集群证书复制到独立备份介质。

恢复时从列表明确选择快照；平台只接受本集群 backup 目录中的 `snapshot.db` 或带时间戳的 `snapshot_YYYYMMDDHHMM.db`。恢复先使用固定 `bin/etcdctl snapshot restore` 在执行器临时目录做完整恢复校验，保留默认快照哈希检查，不启动 etcd 服务、不连接集群。校验失败不会调用远端恢复 Playbook；临时目录结束后清理，需要预留足够本地磁盘。通过校验后才会停控制面 / kubelet / kube-proxy 并重建 etcd 数据，必须有维护窗口，先保护当前状态。完成后检查 etcd 健康、控制面、Node、工作负载和存储一致性。

## 4. 并发、取消和故障恢复

- 同一个系统目录一次只允许一个系统操作，因为上游流程会共享本地工具和配置。使用文件 `flock`，子进程继承锁文件描述符；单实例模式下取消能够终止整个本地进程组。
- 任务最长 4 小时；关闭浏览器不会取消后台操作。取消或超时不会回滚远端已执行的修改。
- API 重启后遗留记录会标记为 `interrupted`；不会自动重试。只要还有未核查的中断记录，新操作就被阻止。
- 管理员先检查执行器上的遗留 Ansible / SSH 进程、inventory、远端节点和业务，确认没有仍在修改基础设施的旧进程，再在历史行点击“核查确认”。输入包含记录 ID 和集群的确认词，并记录检查结果；系统保存核查时间和操作人。
- 部署保持一个 API 执行器副本。文件锁能阻止共享同一目录的并发执行，但取消依赖执行进程所在实例；多 API 副本需要额外的执行器队列 / 归属路由，本版本没有自动跨实例转发取消。
- 日志启用 Ansible `no_log`，并对凭据字段 / 私钥块再次脱敏。每次日志最多 20 MiB，页面显示末尾 256 KiB。详细参数被隐藏时，可查看任务名称、失败阶段及状态，在受控维护环境检查目标；不要通过关闭 no_log 向浏览器暴露凭据。
- 暂未自动清理操作审计和备份。按组织保留策略归档历史、日志和快照，保留故障恢复所需的证书与 inventory。

## 5. API 和实现位置

前缀：`/api/v1/task/kubeasz`，所有接口均需登录及启用的系统 `admin` 角色。

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/catalog` | 固定模板、环境检查、集群和快照名 |
| POST | `/preview` | 验证参数，返回实际命令、目标和计划摘要 |
| POST | `/operations` | 携带 `plan_hash` / `confirmation` 提交后台操作 |
| GET | `/operations?cluster=&page=1&size=20` | 分页历史 |
| GET | `/operations/:id` | 状态和日志 |
| POST | `/operations/:id/cancel` | 请求取消运行中的操作 |
| POST | `/operations/:id/reconcile` | 保存中断操作的核查记录 |

预览摘要绑定结构化参数、当前 inventory / config 内容和恢复快照身份。提交前配置或目标发生变化将被拒绝，需要重新预览。请求 JSON 大小限制 64 KiB，未知字段会被拒绝。

- 后端：`api/api/task/service/kubeasz/`、`controller/kubeasz.go`、`model/kubeasz.go`。
- 路由：`api/router/task/task.go`。
- 前端：`web/src/views/task/Job/KubeaszOperations.vue`、`web/src/api/kubeasz.ts`。
- 数据库：启动自动迁移创建 `task_kubeasz_operation`。

## 6. 验证范围

本地测试覆盖固定命令映射、输入注入 / 路径穿越、符号链接、节点角色与删除保护、计划绑定、跨进程锁、日志脱敏、真实本地子进程成功 / 失败 / 取消后的持久化状态及管理员权限。源码归档和 ezctl 摘要已校验，安装脚本通过 `sh -n`，Compose 覆盖与 Kubernetes 补丁通过 YAML 解析。使用隔离的 Python 3.12 / Ansible core 2.18.18 对五个插件 wrapper、全量安装及十一项节点 / 维护 Playbook 共 17 项执行 `--syntax-check`，全部通过；语法检查没有运行任何任务。

```bash
go -C api test ./api/task/service/kubeasz ./api/task/controller ./router/task
sh -n docker/kubeasz/install-system.sh
```

本次实现没有运行 `ezdown -D`、创建真实集群、操作生产节点或构建并部署 API 镜像。上线前需要在与生产相同系统 / CPU 架构的隔离集群，验证安装、插件、节点扩缩容、etcd 备份恢复及中断核查流程。
