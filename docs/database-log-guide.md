# 数据库日志管理

数据库管理新增第五个子菜单“日志管理”，与基础管理中的数据库操作审计分开。前者读取数据库运行日志，后者记录平台运维任务与变更。

## 使用

1. 进入“数据库管理 → 日志管理”。默认 MySQL，可在右上角切换 Redis / PostgreSQL。
2. 按名称、业务分组（含子分组）、环境或状态筛选集群，在对应行点击“查看日志”。
3. 选择实例及日志源，选择最近 100 / 500 / 1000 / 2000 行。
4. “加载日志”读取一次；“实时追踪”先返回最近日志，再通过 SSE 持续接收新增内容。
5. 可停止追踪、清空当前显示、开启自动滚动或全屏查看。切换实例、日志源、关闭抽屉或离开页面会关闭连接。清空显示不删除远端日志。

所有数据库分页居中，包括数据源、集群、存储、备份恢复、操作审计嵌入视图、MySQL 表数据和日志列表。行内数据库图标与现有列表保持相同尺寸。

## 日志来源与前提

- 实例必须关联 CMDB 主机，CMDB SSH 连接及主机指纹应有效；远端需要 Python 3。
- “服务日志”通过 journalctl 读取实例登记的 systemd 服务。MySQL、Redis、PostgreSQL/Patroni 均可使用；SSH 用户必须具有相应 journal 读取权限。
- 文件日志来自该实例登记的数据目录，以及其 log、logs、data/log、data/pg_log 子目录。识别 .log、.err，以及 PostgreSQL 的 postgresql-*.csv / postgresql-*.json；不读取 binlog、WAL、RDB、AOF 等数据文件。
- 文件列表最多 100 项，每个目录最多检查 1000 项；不递归遍历磁盘，不允许客户端输入任意路径，不跟随目录或文件符号链接。轮转后继续读取原路径的新文件。
- 自定义日志写在其他位置时，需要先在实例管理侧核对实际目录，或使用服务日志。此版本不提供任意主机文件浏览。
- 未启用慢查询、日志只写入数据库表、日志仅保存在未登记的容器中时，不会凭空显示这些内容，也不会自动开启日志或修改数据库配置。
- 空日志不代表数据库健康；数据库健康仍由原有采集机制提供。

## 权限与连接

平台 database:read 加集群 db_cluster_view、db_cluster_audit 才能读取。只有数据源查询权或仅有集群查看权不能读取运行日志。使用 API Key 时同样验证平台范围和资产授权。

API：
- GET /api/v1/database/clusters/:id/logs?memberId=成员ID：发现该实例日志源。
- GET /api/v1/database/clusters/:id/logs/stream?memberId=成员ID&logId=日志ID&lines=500&follow=true：SSE 历史/实时读取。follow=false 读取一次。
- SSE 事件：ready、log、notice、heartbeat、error、end。log.data.text 是原始日志文本，不翻译业务日志内容。

浏览器使用携带 Authorization 请求头的 fetch 接收 SSE，不将令牌放入 URL。连接每 10 秒发送心跳并重新检查账号、API Key、集群权限及实例归属；变化后中断。不会自动重连并重放历史日志。单用户最多 4 路、每个 API 进程最多 32 路，每路最长 20 分钟。远端读取也有独立 20 分钟期限。

浏览器仅保留最近 10000 行或 2 MiB。文件历史读取最多回看 2 MiB，截断时提示；日志事件分块发送。SSH 断连、权限错误、过期连接或远端异常会显示错误，重新连接需点击“实时追踪”。

## 部署与验证

后端重启后执行幂等菜单迁移，新增日志菜单和管理员关联；自定义角色按需授予菜单及集群审计权限。前端需更新到对应版本。

本次没有新增数据库表、常驻进程或 Worker。Docker 的 docker/web/devops.conf 与 Kubernetes 的 k8s/api/devops-web-cm0-configmap.yaml 已配置 proxy_buffering off 和足够长的读取超时，SSE 响应同时设置 X-Accel-Buffering: no。若前方还有外部代理，需同样允许流式响应，读取超时大于 10 秒心跳间隔。

自动化检查：
- go -C api test ./api/database/... ./router/database/... ./pkg/db/...
- python3 -B -m unittest discover -s api/api/database/service/testdata -p test_log_reader.py -v
- node web/scripts/database-log-check.cjs
- node web/scripts/database-operation-check.cjs
- pnpm --dir web i18n:test
- pnpm --dir web build

这些测试验证本地真实文件读取、追加、轮转、截断、路径限制、中文/UTF-8 分片、缓冲上限、权限和菜单迁移。浏览器可使用隔离接口夹具检查交互；这不等同于远端生产数据库日志验收。不需要停止数据库、修改主从或执行故障演练。

### 本次验证结果（2026-11-14）

- 后端 go test ./... 通过；补充的日志授权测试覆盖非管理员授权、撤销集群审计权限、受限 API Key、停用账号、跨集群成员及连接额度释放。
- 远端读取脚本在本地真实临时文件上运行，4 个场景通过：日志源范围、历史/中文/容量上限、符号链接替换拒绝、实时追加/轮转/截断与停止。
- SSE 客户端 5 个场景通过，操作确认回归 7 项和国际化回归 2 项通过；前端生产构建通过。
- Playwright 使用隔离接口夹具验证英文、分页居中、日志渲染、深浅主题和 390px 窄屏，无页面运行错误。未使用真实数据库连接，也未更改数据库服务。
- 项目默认类型检查存在 TypeScript 5.9 与 ignoreDeprecations: 6.0 的既有不匹配；临时通过命令行指定 ignoreDeprecations 5.0 后，仍有 71 处既有类型诊断，未涉及本次新增日志文件。未修改项目全局 TypeScript 配置。
