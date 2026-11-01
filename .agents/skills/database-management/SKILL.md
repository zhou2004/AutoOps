---
name: database-management
description: "维护 AutoOps 数据库管理模块：数据源与工作台、集群与高可用、存储、备份恢复、参数配置、操作审计及 Ansible 自动化。用于该模块的功能开发、界面调整、权限和任务故障排查；不替代通用 SQL 咨询或 CMDB 主机管理。"
---

# AutoOps 数据库管理

数据库管理是独立主菜单，复用 CMDB 的业务分组、主机和凭据。先按改动范围阅读源码，再同步接口、前端和必要迁移；不把历史联调环境当成所有会话均可操作的环境。

## 产品约定

当前只有五个子菜单，顺序固定：数据源管理、集群管理、存储管理、备份恢复、操作审计。参数配置放在集群详情的标签页，通过集群行操作“参数配置详情”进入；不恢复独立参数菜单。没有独立“监控诊断”菜单或集群诊断标签；健康采集和监控中心的规则集成继续服务于集群状态。

- 数据源页只管理连接、业务分组和工作台，不放部署或高可用面板。MySQL 控制台命令提示使用行内虚影，Tab 补全，不增加右侧提示面板。
- 集群列表提供名称、分组（含子分组）、引擎、环境、状态筛选。每行有“详情”和“操作”；详情展示概览、拓扑、节点、复制状态及参数配置，变更从行操作菜单发起。列表上方不再显示匹配集群、当前页正常、高可用已验收、恢复已验证四张统计卡片。
- 数据库类型筛选统一在页头右侧，数据源默认全部，其余列表默认 MySQL；用户可切换类型。跨模块跳转携带 `engine`，避免 PostgreSQL/Redis 集群被 MySQL 默认条件隐藏。
- 存储、备份和审计使用各自子模块。集群行操作跳转时携带 `clusterId` 和 `engine`，目标页显示可清除的集群筛选。没有历史备份/容量快照时，也能在对应模块选择集群发起首次操作。
- 操作不填写原因，不要求手输集群名称。无参数操作自动预检；有参数操作只填写实际需要的节点、备份或配置。预检通过后点击“确认执行”，再弹出最终确认，展示集群、动作与目标节点；取消、关闭不发执行请求。保留预检、权限、幂等和锁，不能以简化交互为由绕过它们。
- 行操作以服务端 `capabilities` 为准，不为不支持或未授权的能力提供可点击按钮。菜单打开时获取最新能力。
- 三引擎图标复用运维工具资产：`mysql.svg`、`redis.svg`、`PostgreSQL.svg`（注意大小写）。统一通过 `DatabaseEngineIcon.vue` 使用，不另造文字徽标。
- 样式使用项目共享 CSS 变量；操作列轻量按钮、窄屏横向滚动、抽屉宽度受视口限制。未知/过期状态不能显示为正常。

## 代码入口

| 改动范围 | 主要文件 |
| --- | --- |
| 菜单与迁移 | `web/src/router/modules/database.ts`、`api/pkg/db/navigation_migrate.go` |
| 五模块列表与筛选 | `web/src/views/database/Management.vue` |
| 行操作及确认 | `components/ClusterActions.vue`、`components/OperationDialog.vue`（相对上一行目录） |
| 备份使用流程 | `components/BackupGuide.vue`、`BackupLauncher.vue`、`BackupDetail.vue`、`BackupSettings.vue` |
| 详情与参数 | `components/ClusterDetail.vue`、`components/ConfigurationDetail.vue` |
| 工作台、数据源 | `web/src/views/database/Workbench.vue`、`MySQLWorkbench.vue`、`components/MySQLConsole.vue`；`web/src/views/cmdb/cmdbDB.vue` |
| 前端请求与类型 | `web/src/api/databasePlatform.ts` |
| 领域模型 | `api/api/database/model/models.go` |
| API 与控制器 | `api/router/database/database.go`、`api/api/database/controller/database.go` |
| 服务、权限、发现 | `api/api/database/service/service.go`、`discovery.go`、`compatibility.go` |
| 任务与恢复核验 | `operations.go`、`execute.go`、`reconcile.go`、`remote_guard.go`（同上服务目录） |
| 引擎运维 | 同上目录的 `mysql_*`、`fencing.go`、`sentinel.go`、`membership.go`、`redis_placement.go`、`patroni_runtime.go` |
| 存储、备份、参数 | 同上目录的 `storage.go`、`backup.go`、`archive.go`、`repository*.go`、`retention.go`、`configuration.go` |
| 自动化 | `api/automation/database-platform/` 的 playbook、templates 和 files |
| 旧接口兼容 | `api/api/cmdb/service/databaseBridge.go`、`databaseClusterScan.go`、`databaseOperation.go`；`api/pkg/db/database_migrate.go` |
| 监控中心集成 | `api/api/monitor/service/database_rules.go` |

新增模型要注册到领域 `Models()` 和平台迁移链。保留旧数据源、账号、业务分组和授权 ID；退役菜单禁用数据库菜单记录，不删除历史角色关联。

## 必须保留的领域关系与执行约束

数据源（连接）≠实例（进程）≠集群（复制/分片关系）≠部署任务。同机多 Redis 实例用独立成员 ID、端口、目录和服务名区分；不能仅靠主机 ID 找实例。

数据库 API 前缀为 `/api/v1/database`。执行顺序是 `operation-plans` → `operations` → `operations/:id`。新客户端提交 `{planId, idempotencyKey, confirmed: true}`；预检返回稳定的 `db-plan-<planId>` 幂等键，同一计划重试必须复用，不能依赖仅安全上下文可用的 `crypto.randomUUID()`。兼容旧后端时由计划 ID 推导同格式键；计划与有效键就绪后才允许确认。旧 `confirmName` 合约为兼容保留，不再展示输入框。计划绑定用户、集群、拓扑版本和有效期，任务持久化后由独立 Worker 执行。

- 数据源权限不赋予集群运维权。集群资产类型是 `database_cluster`，动作前缀 `db_cluster_`；加入新主机还检查主机部署权限，API Key 与调度执行再次校验范围。
- 一集群同时只执行一个运维变更。Worker 丢失心跳将任务置为待核验并保留锁；接手前验证远端状态、世代与排他锁。结果不明的切换、恢复、迁槽和业务写操作不能自动重放。
- 原生切换完成后单独记录业务入口核验步骤，最多等待 30 秒并要求连续两次角色/连接核验通过；只重试只读探测，不重放原生切换。核验超时保留任务锁，错误说明具体入口；任务的 `unknown` 显示“结果待核验”，与健康未知分开。
- 数据采集超过 90 秒显示状态未知及最后采集时间。高可用验收、恢复验证分别记录，安装成功不自动置位。
- MySQL 切换通过 Orchestrator 原生恢复链路与独立隔离钩子。Raft 多数派不等于数据多数派；隔离失败必须阻止危险提升。半同步退化需如实展示。
- PostgreSQL 采用 Patroni/etcd，参数通过原生管理器；严格同步、旧主隔离、回退/重建条件不能绕过。控制版本与安装包校验以当前自动化和兼容层为准。
- Redis 使用 Sentinel/Cluster 原生选主。Cluster 仅 DB 0，支持 MOVED/ASK；迁槽续行先核对真实槽位状态，缩容先迁空；检查每个复制组的故障域分布。
- 恢复使用新目录和隔离端口。远端仓库完成、备份链完整和实际恢复验证独立记录；清理不得破坏依赖链。PostgreSQL WAL 由数据库/备份工具管理，不直接删文件。
- 账号经配置中心引用，日志与文档不保存密码、私钥或会话令牌。SSH 使用已核对的主机指纹，Ansible 使用受控模板。

## 按任务选择验证

- 页面/交互改动：`pnpm --dir web build`；在浏览器检查真实菜单、行操作、确认与取消、三个图标、深浅主题和窄屏。用户只改 UI 时，不顺带触发数据库故障或重建。
- 前端提交回归：`node web/scripts/database-operation-check.cjs`，覆盖无 Crypto API、旧后端响应、响应丢失重试、失效/过期界面请求，以及最终确认的取消、关闭、重复点击和迟到确认。
- 确认或任务语义：`go -C api test ./api/database/... ./pkg/db/... ./router/database/...`。重点验证未确认拒绝、计划归属、幂等、互斥与过期计划，而非只匹配文案。
- 跨模块后端改动：`go -C api test ./...`。项目原有 TypeScript 全量检查问题应与新增错误区分，构建成功不能替代类型检查结论。
- 真实测试仅在当前任务已有授权的隔离环境执行。`live_*_test.go` 默认跳过；启用前阅读环境变量、实例归属、端口和清理逻辑，不把历史 `.51/.52/.53` 地址写成通用授权。

完整架构、阶段和验收矩阵见 [计划方案](../../../docs/database-management-plan.md)。核对已实测能力及历史失败时阅读 [联调记录](../../../docs/2026-11-01-database-platform.md)。需要修改部署入口时同步 `docker/`、`k8s/` 与运行说明，区分代码修改和实际发布。

## 状态采集、列表与 VIP（2026-11-01 补充）

- 状态采集由 API 启动的只读 Collector 负责，Worker 也可承担采集；`database-collector` 模式可单独运行。每 15 秒扫描，集群间有界并发，数据库租约协调多进程。运行中任务的锁阻止新采集，任务执行前等待既有采集完成；仅 `unknown` 待核验任务允许只读采集继续，但保留锁与任务结果，禁止自动补写成功或重放命令。孤立锁继续阻止采集。Worker 停止不应导致全部健康过期；过期仍必须显示未知。
- 所有连接同一元数据库的进程必须使用相同 `TZ` 与 MySQL `loc=Local`。真实联调入口也要一致，避免时间戳与采集租约发生八小时时差。
- 列表行统一 64px，行内引擎图标 26px；长名称/描述省略并可悬停查看。存储和备份行使用服务端返回的 `engine`、`clusterName`，不能依赖当前页集群列表补名称。分页在权限、筛选、存储去重之后执行；分类、筛选和每页数量改变时回到第一页。
- `replication.go` 采集主库位点后核验副本回放。健康流线要求线程/链路正常、只读保护合格（MySQL）且已追平采样位点；未知、异常、过期不播放动画。位点追平不等于全表一致性校验或零 RPO。
- VIP 只用于受管 SQL 双代理，Redis 保留原生 Sentinel/Cluster 入口。`configure-vip`、`verify-vip` 从集群行操作进入，只需参数和确认。预检核对 CMDB 地址、同网段网卡、端口、ARP 应答、VRID、主机权限与服务归属。
- 每个集群使用独立 Keepalived 配置、PID 文件和 `autoops-db-vip-<clusterId>` 服务。演练只停止此服务，先设置独立的 60 秒 systemd 恢复定时器；核验两端唯一绑定、承载变化、连续 SQL 主库连接及原服务恢复。不能据此置位整个集群的 HA 验收。
- 操作审计默认保留 90 天，0 为永久。只有管理员且满足 API Key 动作限制才能修改周期/删除。删除清除终态任务步骤与日志，保留幂等回执、备份引用和原生事件去重标识；运行中、待核验或持锁任务不得删除。

VIP 操作与排查教程：[数据库 VIP 漂移](../../../docs/database-vip-guide.md)。

## 备份恢复的界面约定

- 页面按“选择集群并备份 → 查看备份集 → 隔离恢复 → 查看报告”引导，提供明显的“立即备份”按钮；首次备份不要求先创建定时策略。数据库类型和业务分组同时过滤操作集群。
- 备份集行展示方式中文名、完整性、副本位置和验证时间，提供“详情”和“恢复操作”。失败/过期备份不显示可执行恢复动作。定时策略显示名称、周期、下次运行和保留天数，保存策略不等于立即执行。
- `GET /api/v1/database/backups/:id` 按集群查看权鉴权，详情不能依赖列表前 100 条；凭据继续不序列化。详情展示备份文件/校验、依赖及恢复报告。
- 时间点恢复只有所有分片归档证据完整、时间窗口有交集时开放；逻辑/RDB 不显示时间选择，MySQL binlog 归档必须指定目标时间。前端说明不能替代后端/自动化预检。
- 当前隔离恢复及恢复验证均实际恢复、启动校验，然后停止实例，保留目录和报告。界面及文档必须明确这一行为，不能承诺自动覆盖业务库或完成业务切换。

操作教程见 [备份恢复使用指南](../../../docs/database-backup-guide.md)。
