# Web 管理模块：部署、操作与验收

本模块使用 `/web` 页面、`/api/v1/web` API 和独立 Web Worker。首个引擎为 Nginx，支持 Linux 主机和通过宿主机 SSH 管理的 Docker 容器。数据源、实例、集群分别表示连接位置、运行对象和发布边界；接入默认为观察模式。

原生文件编辑、配置分组和证书导入导出的最新操作见 [2026-11-12 专项指南](2026-11-12-web-native-configurations.md)。该次交付按用户要求暂缓自动续期，不新建或修改续期策略；下文保留已有 ACME/续期架构说明，不代表完成本次公网验收。

当前入口为数据源、集群、站点、配置、证书、日志、流量监控；Web 终端已退役。流量指标、仪表盘、防护与上游分流见 [2026-11-13 流量指南](2026-11-13-web-traffic.md)，真实 Keepalived/VIP 接入见 [高可用指南](web-vip-guide.md)。

## 代码入口

| 位置 | 职责 |
| --- | --- |
| `api/api/webservice/model` | `web_*` 元数据、不可变版本、计划、任务、步骤、锁、订单和审计事件 |
| `api/api/webservice/service` | 资产授权、发现、渲染、计划冻结、Worker、采集、证书和续期 |
| `api/api/webservice/controller` | REST、SSE、流量和策略接口；遗留终端接口已停止注册 |
| `api/automation/web-platform/runner.py` | 经 SSH 执行的 Nginx 主机/Docker 适配器 |
| `api/pkg/websecret` | 外部主密钥与 AES-256-GCM |
| `api/pkg/db/web_migrate.go` | 幂等菜单迁移，管理员菜单关联和 Web 告警类型 |
| `web/src/views/webservice` | 七个页面入口、操作计划、版本、流量、防护、日志和审计 |
| `api/api/monitor/service/web_rules.go` | 接入现有规则引擎，不在采集器直接发通知 |

`EngineProvider` 与 `RuntimeConnector` 分离。当前启用 Nginx 与独立 Keepalived/VIP 自动化；Caddy、Kubernetes Ingress 及集中日志平台尚未启用。

## 启动与密钥

API 和所有 Web Worker 必须使用**相同且稳定的 32 字节主密钥**。可使用 `WEB_ENCRYPTION_KEY`（32 字节随机值的 Base64 编码），或 `WEB_KEY_FILE` 指向包含原始 32 字节的文件。未配置环境变量时读取 `data/web.key`。正常启动不会自动生成或覆盖密钥；首次部署使用显式初始化命令。

首次安装前检查数据库是否已有加密的证书版本、ACME/DNS 凭据或操作计划。如果已有记录但密钥丢失，应恢复原密钥，不能重新生成。当前开发环境首次初始化已完成，文件为 `api/data/web.key`（32 字节、权限 `0600`，已被 Git 和镜像构建上下文排除）。

```bash
# 仓库根目录：按 api 工作目录初始化默认 data/web.key
# 命令保留既有密钥；损坏的文件会报错，不会替换。
go -C api run . -mode web-key-init

# 指定外部密钥文件；不需要数据库或 Redis 在线
WEB_KEY_FILE=/secure/autoops/web.key ./devops -mode web-key-init
```

`web-key-init` 以 `0600` 写入临时文件、同步后原子创建目标文件，重复执行不会轮换密钥，输出仅含路径和是否新建。现有有效 `WEB_ENCRYPTION_KEY` 优先；无效环境变量会报错，不会降级为另一把密钥。

主密钥应与数据库分别备份，恢复时必须成对恢复。更换密钥前需要解密重加密迁移；直接替换会使已有私钥、DNS Token、ACME 账户密钥和计划快照无法读取。不要把密钥提交到仓库或放入操作日志。

开发环境：

```bash
cd api
WEB_WORKER_MODE=embedded go run . -c config.yaml
```

生产环境：

```bash
WEB_WORKER_MODE=external ./devops -c config.yaml
./devops -mode web-worker -c config.yaml
# 可选：单独运行只读采集器
./devops -mode web-collector -c config.yaml
```

API、Worker 共用元数据库。Redis 只负责任务更新通知，任务事件 SSE 会从数据库恢复证据，Redis 不可用时后端按间隔读取任务记录；这与实时日志的持续 SSH 流分开。Worker 心跳超过 45 秒不可见时页面提示离线；已经提交的任务保留在队列中。执行中的 Worker 心跳超过两分钟未更新，任务进入“结果待核验”并保留锁。

Compose 的 API 与 `web-worker` 均设置 `WEB_KEY_FILE=/app/data/web.key`，挂载同一 `aiops-state` 状态卷（Worker 只读）。可以注入同一个 Base64 `WEB_ENCRYPTION_KEY`，或在首次部署时初始化卷内文件：

```bash
docker compose -f docker/docker-compose.yml build devops-api
docker compose -f docker/docker-compose.yml run --rm --no-deps devops-api /app/devops -mode web-key-init
docker compose -f docker/docker-compose.yml up -d devops-api web-worker
```

初始化前同样确认是否应恢复旧密钥；运行中的系统不能通过删除状态卷来消除提示。

Kubernetes 使用 `k8s/api/web-worker-deployment.yaml`，API 和 Worker 引用 `autoops-web` Secret 的 `encryption-key`。Secret 中应保存 Base64 **字符串值**，通过 `stringData` 或受控的 Secret 管理系统创建；不要混淆 Kubernetes 的传输编码与应用要求的编码。

SSH 主机须有 Python 3。Docker 容器无需 Python，使用容器已有的 POSIX 工具和 Nginx 二进制。选择 sudo 时，连接用户需要相应的免交互 sudo 授权。API/Worker 均通过 CMDB 已登记的 SSH 主机指纹验证身份。

## 权限配置

先在角色权限中配置 `web:read`、`web:write`、`web:execute` 和所需菜单，再在统一资产授权中选择 Web 数据源、Web 集群、Web 证书或业务分组范围。

| 资产动作 | 用途 |
| --- | --- |
| `web_view` | 查看资产、状态和成员 |
| `web_manage` | 接管、成员维护、服务控制与恢复核验 |
| `web_config_view` / `web_config_edit` | 配置查看 / 编辑 |
| `web_publish` | 配置、历史回滚和证书部署 |
| `web_certificate_use` / `web_certificate_manage` | 使用证书 / 导入及签发管理 |
| `web_logs` | 日志查询、下载 |
| `web_terminal` | 仅保留历史授权记录，入口已退役 |
| `web_audit` | 操作、终端连接及日志查询审计 |

集群执行还检查成员数据源的对应动作。数据源连接、签发账户、DNS 账户及签发域名范围由管理员配置。证书使用权限不会开放私钥读取。API Key 的模块范围、账号状态和资产权限在提交与执行时再次校验。

## 集群、环境与日志的页面操作

集群详情使用后台采集快照展示拓扑，GET 请求不等待 SSH；采集时间保持原值，超过 90 秒显示未知。要即时确认 VIP，使用独立核验任务。接入及切换步骤见 [高可用 / VIP 教程](web-vip-guide.md)，Cloudflare 证书申请、导出与部署步骤见 [证书教程](2026-11-12-web-cloudflare.md)，两份教程在对应页面也有入口。

环境字段提供 production、staging、test、development 以及当前授权资源使用过的值，支持搜索、清空和自定义输入；已有环境值保留原样。`GET /api/v1/web/environments` 按资产权限返回已有值。

日志实时追踪改用 `GET /api/v1/web/instances/:id/logs/stream` 的 SSE，携带 Authorization，不在 URL 传 Token。一个页面只保持一个追踪连接，暂停、切换目标或离开页面会取消；重连读取新的最新窗口，不承诺中断期间无损续传。历史查询仍为有界窗口与翻页，含轮转文件和时间范围时使用历史查询。

日志窗口与 Docker 容器管理共用 `LogConsole`，支持自适应、自动换行、滚动到底部与全屏。范围筛选、详细筛选折叠，当前启用条件有数量提示。下载历史结果复用已显示查询，下载实时结果导出当前接收缓冲。

## 接入、接管和新建服务

1. 在数据源管理选择业务分组、已登记的 CMDB 主机、SSH 连接、主机或 Docker 运行环境。
2. 填写实际 Nginx 二进制、主配置入口、运行前缀。Docker 必须指定目标容器；同机多实例使用不同的配置入口及独立 PID。
3. 发现实例，检查版本、配置文件、日志和证书元数据。发现不会更改服务配置，也不会导入现有私钥。
4. 在集群管理点击“新增”，选择业务分组后，直接在“从数据源选择 Nginx 实例”中多选成员并保存。编辑集群时会回显当前可见成员，可在同一弹窗增减实例；每个实例只能属于一个集群。
5. 明确受管目录和可从节点访问的健康检查地址。预检并确认“接管服务”，保存节点基线后开放发布。

集群编辑器按数据源分组展示已发现的实例，可搜索数据源名称、实例名称、地址、容器或配置入口；同机多实例仍分别选择。其他集群已占用的实例禁选，已部署 VIP 的保护成员不可取消勾选。调整成员需要 Web 执行权限以及集群和数据源的管理授权；仅修改名称、环境和入口探测仍只需原有编辑权限。移出成员保存前有确认弹窗，远端服务和文件保留。

集群信息与成员变化通过同一请求、同一数据库事务保存，任一成员不合法、无权操作、存在任务锁或版本冲突时整次回滚。请求使用 `memberChanges: {add: [实例ID], remove: [实例ID]}`，只调整用户明确选择的成员，保留当前用户不可见或未操作的其他成员。成员变化使旧操作计划失效，拓扑等待新一轮采集；新增实例不会自动部署、接管或加入 VIP 高可用组。

管理连接与业务 URL 分开保存。实际控制前验证 Nginx master PID 对应的配置入口；Docker 使用发现时的不可变容器 ID，容器重建需要重新发现及预检。

集群列表的“新建部署”会预填业务分组、环境和独立目录，保存后进入部署预检与确认。成功发现实例后，在“成员管理”将实例加入目标集群。

新建部署使用 `/etc/autoops-web/<name>/current/nginx.conf`，受管文件放在 `current` 之下。主机 systemd 服务名以 `autoops-web-` 开头，程序为 `/usr/sbin/nginx`。主机包版本必须明确固定，当前自动化支持 Debian 12、Ubuntu 22.04/24.04；已有不同版本的系统 Nginx 包不会被自动升级。Docker 要求固定镜像摘要，可使用仓库的 `@sha256:...`，或目标 Docker Engine 已有的完整 `sha256:...` 镜像 ID。预检检查目标目录、端口、磁盘、运行环境及版本可用性。

已发现实例的数据源定位不可修改；尚未发现任何实例的数据源允许修正错误的配置入口，修正与发现通过行锁和 revision 检查互斥。已有自编译 Nginx 可以填写实际 `/usr/bin/nginx` 或 `/usr/local/nginx/sbin/nginx` 等路径；发现时不会把同机 `kube-lb` 当作待部署实例。新建部署表单的运行前缀为规范路径 `.../current`，不带末尾斜杠；systemd 和 Docker 启动命令使用相同 `-p` 前缀。首次安装包时阻止默认服务启动并禁用新安装的默认 `nginx.service`，只启用独立 `autoops-web-*` 服务。

移出集群只移出管理关系，保留远端文件及服务。数据源存在关联实例时，删除会明确拒绝，以保留实例历史。

## 配置和发布

站点表单支持静态目录、反向代理、多路径、重定向、请求头、WebSocket、超时、请求体限制和 HTTPS；上游支持权重、备用成员、轮询、最少连接和 IP hash。生成配置草稿后，仍需预检并确认发布。站点和上游的“已发布 / 部分部署成功 / 草稿”根据实际成员文件与当前渲染内容比较，发布期间保留上次确定状态；回滚或外部修改会在采集后更新。已有复杂配置使用原文模式，编辑平台生成文件会把对应站点切换为原文维护。

配置草稿可读取实例发现的完整文件树，逐文件编辑、标记删除、设置节点变量。`${变量名}` 由显式节点变量替换。草稿使用版本号比较，并发修改返回冲突。版本历史保存各节点的独立文件集合，支持统一差异和生成新的回滚任务。

发布计划有效期五分钟，冻结目标、源连接版本、配置摘要、节点变量渲染结果及证书版本。提交格式：

```json
{"planId":123,"idempotencyKey":"web-plan-123","confirmed":true}
```

重复提交相同计划返回同一任务。配置、证书部署和服务操作共享集群锁。执行时逐节点处理：

1. 再次检查授权、实例身份、连接版本及配置基线。
2. 在运行环境创建候选树，重写嵌套/绝对 include 和候选证书引用，运行实际 Nginx 的 `-t`。
3. 保存远端执行记录及文件备份，再切换文件。接管实例保留布局及符号链接；新建实例复制到独立版本目录后切换 `current`。
4. 对实际入口再次 `-t`、平滑 reload，并检查 Nginx master、源健康 URL、相关站点 URL、TLS 指纹。
5. 成功后继续下一个节点。任一节点健康失败时恢复该节点并倒序回退此前已发布节点。

接管实例的多个文件逐个原子替换，不构成多文件整体原子提交。恢复只能使用任务保存的旧内容，不能把临时读取的新配置当成可覆盖的基线。
### 原生配置（conf.d）维护

进入「配置管理 → 实例配置」可直接维护 Nginx 原文；「发布草稿」保留高级编辑和历史发布入口，具体步骤见 [专项指南](2026-11-12-web-native-configurations.md)。

- `GET /api/v1/web/instances/:id/config` 实时读取主配置、include 依赖及受管 `conf.d` 文件；`editable` 标识可编辑范围，`included` 区分是否被入口引用。
- 新增、修改、删除形成完整候选文件树，显式缺失引用和未引用 `.conf` 均须检查。保存接口强制执行实际 Nginx `-t`，任何目标失败均拒绝保存；手动「校验语法」供提前查看错误，不能替代保存或发布阶段校验。
- 校验与保存不会修改线上配置。默认草稿只包含当前实例的 `spec.instanceIds`；选择整个集群时为空，由服务端解析并授权全部成员。确认计划后 Worker 才切换文件并重载。
- 配置分组通过 `spec.paths` 关联绝对路径，同集群内唯一归属；保存到分组会加入非删除文件。原生页面支持组、未分组、路径/内容及已修改筛选，草稿列表支持 `configGroupId`。

配置草稿限制为 1–200 个文件、单次内容合计 4 MiB；私钥请通过证书管理导入。计划冻结草稿修订号与摘要，执行旧计划不能把后来的修改误标成已发布。

### 证书导入与导出

顶部「导入证书」可选择文件或粘贴 PEM，一次创建资产和首版本。新建接口将资产、版本和审计置于同一事务，错误不会留下空资产；已有资产导入会创建新版本。导入成功不自动部署。

列表展示当前版本剩余天数、建议续期时间和到期日。「证书版本」查看历史元数据，并可导出指定版本：证书链和私钥分别下载，版本 `0` 表示当前版本。公开证书材料不得混入私钥，历史过期版本可供备份导出。

私钥导出要求管理员身份、`web:write` 和 `web_certificate_manage`，API Key 同样经过权限检查。每次导出记录 `certificate-export` 审计；审计写入失败则拒绝返回，响应设置禁止缓存。本次按用户要求暂缓自动续期，未将证书导入操作绑定为续期或部署任务。

### 中断恢复

“结果待核验”不会自动重放发布，也不会自动释放锁。在任务详情选择“核验中断任务”：

- 已完成的远端记录：核验当前内容、服务和证书后更新任务结果。
- 已准备/部分写入：只有每个变更文件仍等于任务记录的旧值或候选值、未修改的文件也没有漂移时，才恢复旧文件并验证服务。
- 已有人工修改、实例身份变化、证据缺失或恢复失败：保留原锁和待核验状态，展示冲突。

远端恢复记录位于 `/var/tmp/autoops-web/<实例摘要>/`，其中备份可能含私钥，目录/文件限制为 `0700/0600`。活动任务与待核验任务的执行记录不能删除。元数据库默认清理 90 天前已终态任务的步骤与独立审计事件，保留操作回执、幂等信息和历史版本。

## 证书与 ACME

证书导入校验证书链顺序、签名、SAN、用途、有效期及私钥匹配。证书版本、绑定期望版本、节点实际 TLS 指纹分别记录。首次发布 HTTPS 站点时，先创建证书和部署绑定，再生成站点草稿；发布计划会带上绑定的证书文件。

部署绑定需要配置证书/私钥路径、节点可访问的 TLS 地址、端口和 SNI 域名。测试 CA 或内部 CA 的证书通过已导入叶证书指纹验证，不依赖操作系统对该 CA 的信任。证书本身仍须覆盖 SNI 域名。

ACME 使用 `github.com/go-acme/lego/v4 v4.25.2`，与项目 Go 1.25 兼容。支持 HTTP-01，以及 AliDNS、Tencent Cloud DNSPod、Cloudflare DNS-01。推荐先使用 Let's Encrypt staging directory。DNS 账户可引用配置中心已有云凭据，或录入专用 DNS 凭据；Cloudflare 在配置中心对应类型 `6`，Token 不支持读取或导出。

HTTP-01 要求业务入口的公网 80 端口可达，预先配置 `/.well-known/acme-challenge/` 对应根目录，并登记所有可能承接请求的实例。泛域名必须使用 DNS-01。DNS TXT 记录的创建、传播等待和按挑战值清理由 lego provider 处理。

续期策略保存明确的授权身份和部署绑定列表。执行前重新检查账号、API Key 和资产范围。优先使用 CA 返回的 ARI 建议窗口，否则使用证书有效期的三分之二位置。订单 URL 与私钥持久保存，重试优先恢复已有订单；新订单创建结果丢失且没有 URL 时保留不确定订单，不重复创建。

签发成功后，各集群独立生成部署计划和任务。部分失败保留新证书及子任务结果，可从部署绑定手动重新预检、部署。签发失败保留旧证书；部署成功以节点实际 TLS 核验为准。

## 日志与监控

日志使用后端发现的日志标识，不接受任意文件路径。支持 access/error、数字轮转后缀、gzip 文件和 Docker stdout/stderr。提供原文、关键词、常见日志格式的状态码/IP/URI、时间范围、历史窗口和结果下载。单次最多 10,000 行，历史扫描最多 100,000 行；远端命令输出限制 8 MiB，gzip 文件还限制压缩大小。超限会明确失败。实时视图通过 SSE 持续推送，支持日志轮转、截断和容器输出，暂停/离页取消 SSH；历史查询与实时追踪分别处理，不保存全量访问日志。

Web 终端入口与票据/连接路由已退役，遗留实现和连接审计保留用于历史追溯。

基础采集每 30 秒运行，超过 90 秒未更新显示未知。可选配置 `stub_status` 地址采集连接数和累计请求数；不从这些计数推断延迟或状态码分布。

监控中心选择 `web` 规则类型，可使用：

```text
instance_unavailable == 1
configuration_drift == 1
state_unknown == 1
remaining_days < 30
certificate_deployment_lag == 1
renewal_failed == 1
publish_failed == 1
connections_active > 1000
```

支持业务分组、引擎、环境、实例/集群等标签约束；仅启用的规则通过现有通知及故障流程触发通知。

## 主要 API

前端使用 `web/src/api/webservice.ts` 及 `webTraffic.ts` 对接真实接口。API 基址支持源站、网关前缀和已包含 `/api/v1` 的地址，版本前缀只拼接一次。列表刷新会同时更新实例、集群、连接等选项，接口失败显示错误并清除对应旧选项。

数据源列表展示 CMDB 管理连接与最近发现时间。发现失败保留原因，不创建运行实例；源连接在发现期间被编辑时拒绝保存旧结果。日志页面复用业务分组、集群、环境、名称及状态过滤，URL 中的 `instanceId` 可预选实例。日志下载使用最后一次已显示查询的完整筛选快照，包括时间范围。

资源列表统一返回 `{list,total}`，先进行资产过滤，再统计分页。详情/保存/错误使用项目标准响应格式。

| API | 用途 |
| --- | --- |
| `/sources`、`/sources/:id/check`、`/sources/:id/discovery` | 连接管理、测试、观察模式发现 |
| `/instances`、`/clusters`、`/clusters/:id/members` | 实例和集群成员 |
| `/clusters/:id/capabilities` | 动作及禁用原因 |
| `/sites`、`/upstreams`、`/:kind/:id/render` | 结构化配置及草稿生成 |
| `/configurations`、`/config-revisions`、`/config-revisions/:id/diff` | 草稿、历史、差异 |
| `/instances/:id/config`、`/instances/:id/config/validate`、`/clusters/:id/config/validate` | 实时原生配置、单节点或 `instanceIds` 指定范围的候选 `nginx -t` 校验 |
| `/config-groups` | 集群配置分组，`spec.paths` 记录文件路径，供筛选 |
| `/certificates`、`/certificates/import`、`/certificates/:id/import`、`/certificate-bindings` | 证书资产、事务导入首版本、已有资产导入新版本及部署 |
| `/certificates/:id/versions`、`/certificates/:id/versions/:vid/export` | 证书版本与导入导出（私钥仅管理员并审计） |
| `/acme-accounts`、`/dns-accounts`、`/renewal-policies` | 签发与续期配置 |
| `/operation-plans`、`/operation-plans/:id/preview`、`/operations` | 冻结计划、脱敏预览、确认执行 |
| `/operations/:id`、`/operations/:id/events`、`/operations/:id/cancel` | 查询、SSE、边界取消 |
| `/traffic`、`/clusters/:id/traffic-policy[/preview\|/draft]` | 流量统计、原生策略读取、校验与草稿 |
| `/instances/:id/logs` | 日志目录、查询、窗口和下载 |
| `/audit`、`/audit/events`、`/worker-status` | 操作、连接/查询审计、Worker 在线状态 |

## 验证方式与边界

```bash
cd api
go test ./...
cd ../web
pnpm build
pnpm i18n:test
pnpm i18n:audit
pnpm typecheck
```

本机 Nginx 和已有 Docker 镜像的隔离测试：

```bash
python3 -m unittest discover -s api/automation/web-platform/tests -v
AUTOOPS_WEB_DOCKER_TEST=1 python3 -m unittest discover -s api/automation/web-platform/tests -v
```

测试只使用自行创建的目录、容器和端口；Docker 测试不拉取新镜像。显式 SSH 联调读取指定配置中的 CMDB 连接资料，仅把 Web 元数据写入临时 SQLite，远端通过解压固定软件包创建独立测试进程，结束后清理：

```bash
AUTOOPS_WEB_SSH_TEST=1 \
AUTOOPS_WEB_SSH_HOSTS=192.168.80.51,192.168.80.52,192.168.80.53 \
AUTOOPS_WEB_CONFIG=/absolute/path/to/api/config.yaml \
AUTOOPS_WEB_NGINX_PACKAGE=1.22.1-9+deb12u10 \
go -C api test ./api/webservice/service -run TestLiveSSHClusterLifecycle -v -count=1
```

测试包版本必须在目标软件源中可用。该命令不会安装系统软件包、启动默认 nginx.service 或更改现有 CMDB 连接。

实际验收记录见同目录 `web-management-validation.md`。公网 ACME 签发与三个 DNS provider 的真实账号联调，需要测试域名及相应凭据；不能用本机自签名证书测试替代此项验收。

前端数据契约测试：`pnpm --dir web webservice:test`，覆盖 API 前缀、查询分页、日志时间窗口、会话失效、请求取消、原生文件与节点范围、历史证书导出和证书导入请求。

## 项目 skill

模块开发技能位于 [`.agents/skills/web-management/SKILL.md`](../.agents/skills/web-management/SKILL.md)，可通过 `$web-management` 调用。主文件维护模块入口、执行与权限约束，`references/data-integration.md` 记录前后端数据契约及真实数据源接入流程。修改相关契约时同步维护技能，避免将历史测试主机或凭据当作默认授权环境。
