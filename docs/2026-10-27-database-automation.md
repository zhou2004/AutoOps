# 数据库 Ansible 部署与 MySQL Orchestrator 集成

## 使用入口

- **运维工具 → 工具列表**：MySQL、PostgreSQL、Redis 卡片进入统一的 Ansible 安装弹窗；数据库安装不再走旧 Docker Compose 接口。
- **运维工具 → 工具列表 → 数据库 Ansible 部署**：安装、执行历史、日志、备份、重试失败任务和高可用管理。
- 自动安装成功后创建配置中心账号和数据源，可从 **数据库管理 → 数据源管理** 进入对应工作台。数据源页不再展示部署及高可用区域；[MySQL 可视化工作台说明](2026-10-27-database-workbench.md)。编辑数据源不会丢失自动部署关联。
- **MySQL 复制拓扑**：发现/刷新、启动/停止/重启复制、计划切换、指定候选故障恢复、调整上游、只读/可写、维护、停机标记、自动恢复开关。
- **Orchestrator 原生界面与访问凭据**：提供完整上游 Web 界面入口；管理员显式查看独立 HTTP 凭据并记录审计。凭据不写入 URL、部署列表或操作日志。

集群操作会影响多个资产，因此当前部署、高可用、备份及管理端凭据入口仅管理员可用。已有数据源查询/执行仍使用原有数据库资产授权。

## 支持范围

| 数据库 | 安装方式 | 复制 / HA | 备份 |
| --- | --- | --- | --- |
| MySQL 8.0.24 | 基于用户 Ansible-Mysql 的固定版本二进制角色 | GTID 一主多从；Orchestrator 单节点或 3 节点 Raft | mysqldump 全库、事务一致性、存储过程/事件/触发器 |
| PostgreSQL | Debian / Ubuntu 系统软件源 | 当前为单机，不用 Orchestrator 管理 | pg_dumpall，包括角色与数据库 |
| Redis | Debian / Ubuntu 系统软件源，密码认证与 AOF | 当前为单机，不用 Orchestrator 管理 | redis-cli --rdb |

MySQL 版本沿用提供的仓库，**不是最新版本选择器**。安装包固定 SHA256，变更版本需同步验证角色和校验值。MySQL 的 RedHat 分支保留于角色，但本次实机目标均为 Debian 12；不能据此声称其他发行版已通过安装验收。PostgreSQL/Redis 的具体版本取决于目标机已配置的软件源。

Orchestrator 提供 MySQL 拓扑管理、故障检测与恢复，不提供 PostgreSQL/Redis 高可用，也不替应用自动改变连接地址。本实现没有配置业务 VIP、ProxySQL/HAProxy 或自定义网络隔离钩子；上线业务前需对接稳定访问地址与隔离策略。异步复制不能承诺任意故障下零数据丢失。

## 源码与部署

MySQL 安装角色来源：<https://github.com/zhou2004/Ansible-Mysql>，固定提交 `273d54dfb732b41afb13ecf154b0e381e91bcc62`。许可证及改动说明在 `api/automation/database/LICENSE.Ansible-Mysql` 和 `UPSTREAM`。

Orchestrator 使用 <https://github.com/openark/orchestrator> 的 `v3.2.6` 发布包。该仓库已归档；保持兼容并接入其真实 HTTP API，没有自行模拟选主或故障恢复。

控制端依赖：Ansible、OpenSSH client、sshpass（密码 SSH 时）、固定的 Ansible collections：

```bash
ansible-galaxy collection install -r api/automation/database/requirements.yml
```

容器镜像构建安装这些 collections，并携带 `api/automation/database`。运行目录通常为 `api/`；也可设置：

```text
AUTOOPS_DATABASE_AUTOMATION_DIR=/app/automation/database
```

`docker/api/Dockerfile` 与 Kubernetes API Deployment 已同步。无需在 API 镜像中安装受管数据库服务。

目标主机要求：Python 3、root 或免密 sudo、可用的软件源；CMDB 必须配置已启用且绑定 SSH SHA256 指纹的连接档案。私钥通过权限 0600 的临时文件传递，兼容缺少末尾换行的历史凭据；每次执行重新获取主机密钥并比对指纹。数据库密码和内部凭据加密入库，Ansible 使用私有 JSON inventory / extra-vars 文件，任务完成后删除。

MySQL 与 Orchestrator 发布包由控制端下载后分发，目标机不必直连 GitHub。PostgreSQL/Redis 使用主机软件源，不修改用户的 APT 配置。

## 隔离与执行行为

- MySQL 默认业务端口 3306，可修改；实例配置 `/etc/autoops-mysql.cnf`，数据 `/var/lib/autoops-mysql`，服务 `autoops-mysql`，二进制 `/opt/autoops/mysql/`。
- Orchestrator HTTP `13000`，Raft `11008`，服务 `autoops-orchestrator`；配置 `/etc/autoops-orchestrator.json`、状态 `/var/lib/autoops-orchestrator`、二进制 `/opt/autoops/orchestrator`。每节点使用独立 SQLite 后端。
- PostgreSQL/Redis 为原生软件包服务。预检拒绝占用端口、未知数据目录、其他部署的所有权标记；不会把已有数据库默认为新装成功。
- 备份保留在每个数据库节点的 `/var/backups/autoops/<部署名>/<执行 ID>/`，权限 0700/0600，附文件大小和 SHA256 manifest。操作成功只表示本次备份命令及文件校验成功，恢复验证另行执行。
- MySQL 主从部署的节点重启默认只读，避免旧主库重启后直接作为可写节点返回；Orchestrator 使用 `UseSuperReadOnly` 管理运行时角色。单机部署保持可写。整个主从集群同时停机重启后，需要核对最新 GTID 并明确主库角色，不能把这一行为视为完整的网络隔离机制。
- 计划切换使用上游 `graceful-master-takeover-auto`，自动启动降级后旧主的复制线程。拓扑逐个读取当前部署的所有实例，避免切换期间因为集群归属变化漏掉节点。
- 任务写入持久化执行记录与跨 API 实例互斥锁。同一部署的 Ansible/HA 写操作互斥；高可用写请求发生超时后不自动重试，状态为“结果待核验”。
- API 进程异常退出后，未完成任务的锁保留；应先核对远端任务/实例状态再由管理员处理，当前不提供自动抢占锁或自动重放安装。
- 不提供卸载、删除数据目录或恢复覆盖入口；旧工具卸载接口也不管理这些新部署。

## API

统一前缀 `/api/v1/cmdb/database-deployments`，均需管理员身份：

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET / POST | `/` | 列出 / 创建部署 |
| POST | `/:id/actions` | `backup` / `retry-install`，需完整部署名称确认 |
| GET | `/:id/runs` | 最近 30 次执行及日志 |
| GET | `/:id/ha/:view` | topology / analysis / recoveries / health / recovery-state / maintenance / problems |
| POST | `/:id/ha` | 白名单内的 Orchestrator 操作，源实例和候选均须属于当前部署 |
| GET | `/:id/orchestrator-access` | 显式查看管理端访问凭据，响应禁止缓存 |

创建示例（密码使用实际输入，不要沿用占位符）：

```json
{
  "name": "mysql-lab",
  "engine": "mysql",
  "mode": "replication",
  "groupId": 18,
  "hostIds": [528, 531, 530],
  "managerIds": [528, 531, 530],
  "port": 13306,
  "password": "<至少12位的独立密码>",
  "autoRecovery": true
}
```

第一台数据库主机为初始主库。安装请求不允许传入任意 shell、Ansible 参数、Git 地址或 API 路径。

## 验证命令

```bash
cd api
GOMAXPROCS=2 go test -p 2 ./api/cmdb/... ./api/tool/... ./router/cmdb ./pkg/db
```

实机测试是显式开启的，创建并保留独立的测试服务及数据源：

```bash
AUTOOPS_DB_LIVE=1 \
AUTOOPS_DB_TEST_HOST_IDS=528,531,530 \
CMDB_CONFIG_PATH=/absolute/path/to/api/config.yaml \
AUTOOPS_DATABASE_AUTOMATION_DIR=/absolute/path/to/api/automation/database \
AUTOOPS_DB_EVIDENCE_DIR=/tmp/autoops-db-live-evidence \
go test ./api/cmdb/service -run '^TestDatabaseAutomationLive$' -v -timeout=100m
```

故障测试仅停止 `autoops-mysql` / `autoops-orchestrator` 测试服务，验证数据复制、计划切换、Raft Leader 丢失重新选举、数据库主库故障自动恢复、旧主只读重启及重新加入：

```bash
AUTOOPS_DB_HA_LIVE=1 \
CMDB_CONFIG_PATH=/absolute/path/to/api/config.yaml \
AUTOOPS_DB_EVIDENCE_DIR=/tmp/autoops-db-live-evidence \
go test ./api/cmdb/service -run '^TestDatabaseHAFailoverLive$' -v -timeout=15m
```

备份恢复测试会先生成新的备份，再在目标主机创建只监听私有 Unix socket 的临时实例，恢复备份并核对预写入的数据；最后停止并删除临时实例，不覆盖当前数据库：

```bash
AUTOOPS_DB_RESTORE_LIVE=1 \
CMDB_CONFIG_PATH=/absolute/path/to/api/config.yaml \
AUTOOPS_DATABASE_AUTOMATION_DIR=/absolute/path/to/api/automation/database \
AUTOOPS_DB_EVIDENCE_DIR=/tmp/autoops-db-live-evidence \
go test ./api/cmdb/service -run '^TestDatabaseBackupRestoreLive$' -v -timeout=35m
```

## 2026-10-27 实机验收

使用用户指定的 CMDB 主机，均为 Debian 12。预检发现主机已有数据库及其他服务，因此新服务使用独立端口、目录及服务名，保留现有 MySQL 3306、Orchestrator 3000 和 Kubernetes。下表记录本轮安装及故障验收完成时的角色；后续工作台验收时主库已为 .51，实时角色以拓扑为准。

| 主机 | CMDB ID | 本次保留的服务 | 最终状态 |
| --- | --- | --- | --- |
| 192.168.80.52 | 528 | MySQL 8.0.24 :13306；PostgreSQL 15.19 :15432；Orchestrator :13000 | MySQL 可写主库 |
| 192.168.80.53 | 531 | MySQL 8.0.24 :13306；Orchestrator :13000 | 从库，IO/SQL 线程正常 |
| 192.168.80.51 | 530 | MySQL 8.0.24 :13306；Redis 7.0.15 :16379；Orchestrator :13000 | 从库，IO/SQL 线程正常 |

部署名为 `autoops-mysql-validation`、`autoops-postgresql-validation`、`autoops-redis-validation`，均为 `ready`。对应 CMDB 数据源 ID：MySQL 为 8/9/10，PostgreSQL 为 12，Redis 为 11。三节点 Orchestrator 使用 Raft 11008，管理入口为上述三台主机的 HTTP 13000。管理员可在部署的“Orchestrator 原生界面与访问凭据”中查看登录凭据。

| 验证项目 | 结果与证据 |
| --- | --- |
| 三种数据库 Ansible 安装、凭据设置、资产自动登记 | 实机完成；最终统一回归 `TestDatabaseAutomationLive` 全部通过（24.61 秒），SQL 查询及 Redis 写入/读取通过 |
| MySQL 计划切换 | 通过；在新主写入新的记录，两从库复制线程正常且收到该记录 |
| Orchestrator Leader 故障 | 停止实际 Leader 的 `autoops-orchestrator`，剩余节点重新选主，恢复管理访问 |
| MySQL 主库故障自动恢复 | 停止 `autoops-mysql` 后自动提升从库；新主的新写入继续同步到存活从库 |
| 旧主重启与恢复 | 只读启动；本次无分叉 GTID 场景重新加入后同步完整；最终切回 .52，并再次验证新写入在两从库可见 |
| 故障测试总结果 | `TestDatabaseHAFailoverLive` 全部通过（65.27 秒）；测试只停止本次创建的独立服务 |
| 三种备份恢复 | `TestDatabaseBackupRestoreLive` 全部通过（51.10 秒）；MySQL/PG/Redis 的备份执行 ID 为 25/26/27，校验 SHA256、实际恢复和测试数据一致性，临时进程与目录清理完成 |
| 最终备份 | 执行 ID 28/29/30，分别对应 MySQL/PostgreSQL/Redis；MySQL 在三节点分别备份 |
| 后端回归与构建 | CMDB、工具模块、CMDB 路由、数据库迁移包测试通过；API 完整编译通过 |
| Ansible 静态检查 | install、backup、verify-backup 三份 playbook 语法检查通过 |
| 前端 | Vite 生产构建及新增 API/组件 ESLint 通过；浏览器验证 CMDB 主机可选、非法安装参数拒绝、输入名称确认后刷新实例、三节点拓扑、两种 SQL 工作台、Redis 浏览与 Orchestrator 原生集群页面，无页面脚本异常 |

恢复验证确实导入了备份，而不只是检查文件存在。MySQL 全库备份增加 `--flush-privileges`，恢复后使用备份中的 root 凭据验证数据。备份与清理输出保存在当前开发机 `/tmp/autoops-db-live-evidence/restore-*.log`；故障后的拓扑、恢复审计及健康记录保存在该目录的 `after-failover-*.json`。

测试边界：实机安装覆盖 MySQL 三节点 GTID 主从、三节点 Orchestrator、PostgreSQL 单机及 Redis 单机；没有测试跨机房网络分区、磁盘损坏、业务 VIP/代理切换、其他 Linux 发行版或 Kubernetes 镜像发布。旧主自动重新加入未作为生产功能启用，本次在确认没有分叉的测试数据后显式执行 GTID 重新加入。

收尾：本地开发 API 已加载新代码；临时浏览器测试 API Key 已撤销、临时令牌文件已删除；三个部署均就绪，无仍在执行的数据库任务或部署操作锁。测试部署、测试数据和主机上的备份保留，便于直接从页面检查。

仓库全量 TypeScript 检查存在原有阻塞：当前 TypeScript 5.9 不接受 tsconfig 中的 `ignoreDeprecations: "6.0"`；临时覆盖为 5.0 后仍有现存的 SSE、主题、权限等模块类型错误。新增数据库 API/组件未出现对应类型错误，但不能据此声明全仓库 typecheck 通过。
