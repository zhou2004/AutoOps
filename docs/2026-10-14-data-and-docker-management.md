# 数据与 Docker 管理

## 交付范围

- 修复主机编辑时 `vendor` 从列表字符串回传到整数 DTO，导致 JSON 绑定报“参数错误”的问题。
- 数据源支持 MySQL、PostgreSQL、Redis 的连通性、目录、对象、查询、受控写操作和审计。
- Docker 管理通过主机或物理机已有的 SSH 连接档案执行，复用固定主机指纹和凭据授权。
- 连接账号列表不再向浏览器返回加密后的密码；修改账号时密码留空会保留原密码。

## 数据源配置

数据源记录引用“配置中心 / 账号认证”的连接账号。账号类型必须与数据源类型一致：

| 类型 | 账号类型 | 默认端口 | 可用 TLS 模式 |
| --- | --- | ---: | --- |
| MySQL | 1 | 3306 | `disable`, `verify-full` |
| PostgreSQL | 2 | 5432 | `disable`, `require`, `verify-ca`, `verify-full` |
| Redis | 3 | 6379 | `disable`, `verify-full` |

MySQL 和 PostgreSQL 的默认数据库可以留空。Redis 默认库是 `0` 到 `1024` 的数字。部署后由 GORM 自动增加 `default_db`、`tls_mode`、`cmdb_database_audit` 和 `cmdb_docker_audit`。

### 操作约束

- SQL 查询在只读事务中执行，单次最多返回 500 行，超时 30 秒。
- SQL 写入口只允许单条 `INSERT`、`UPDATE` 或 `DELETE`；`UPDATE` 和 `DELETE` 必须包含 `WHERE`。
- SQL 写入前必须输入完整数据源名称。
- Redis 使用 `SCAN` 浏览 Key，不执行阻塞式 `KEYS`。
- Redis String 写入需要输入完整数据源名称；删除需要输入完整 Key。
- 审计只保存动作、目标、状态、客户端 IP 和耗时，不保存 SQL 正文、Redis 值或连接密码。

## Docker 管理

Docker 页面从 CMDB 主机和物理机中选择资产，并读取已启用且当前用户可用的 SSH 连接档案。后端通过固定的 SSH 主机指纹连接资产，不使用跳过指纹校验的回退逻辑。

支持以下功能：

- Docker daemon 状态、容器、镜像、网络和数据卷清单。
- 最多 2000 行、1 MiB 的容器日志。
- 容器启动、停止、重启、暂停、恢复和删除。
- 使用目标机器已有镜像创建容器，可配置重启策略、环境变量和端口映射。
- Docker 操作审计。

创建容器不开放 `privileged`、宿主机网络或任意宿主机目录挂载。删除不使用 `--force`，运行中的容器必须先停止。除启动外，生命周期操作要求输入完整容器名称。

## 细分权限

数据库资产新增：

- `db_query`：状态、目录、表、SQL 查询、Redis 读取。
- `db_execute`：SQL 写操作、Redis 写入与删除。
- `audit`：读取审计。

Docker 使用主机或物理机资产授权：

- `docker_view`
- `docker_logs`
- `docker_operate`
- `docker_remove`
- `audit`

数据库操作同时校验关联连接账号的凭据授权。Docker 操作同时校验 SSH 连接档案引用的凭据授权。凭据规则带有 `credentialType` 来源，值为 `ssh` 或 `database`；相同数字 ID 在两个来源中互不授权。

## 验证

```bash
cd api
go test ./api/cmdb/... ./api/configcenter/... ./middleware ./router/cmdb ./pkg/db

cd ../web
pnpm build
```

只读实机检查通过环境变量显式启用：

```bash
CMDB_DOCKER_LIVE=1 CMDB_CONFIG_PATH=/absolute/path/config.yaml \
  go test ./api/cmdb/service -run '^TestDockerInventoryLive$' -v

CMDB_DATABASE_LIVE=1 CMDB_CONFIG_PATH=/absolute/path/config.yaml \
  go test ./api/cmdb/service -run '^TestConfiguredDatabaseServicesLive$' -v
```

Docker 实机测试只读取 daemon、容器、镜像、网络和数据卷。数据库实机测试只执行 MySQL `SELECT 1` 以及 Redis 的 PING/INFO。
