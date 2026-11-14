---
name: web-management
description: "维护 AutoOps Web 管理模块：数据源、Nginx 主机与 Docker、集群、站点配置发布、证书、日志、流量监控与防护、上游负载均衡、Keepalived/VIP 和审计。用于该模块开发、仪表盘接入、权限或任务恢复排查；不替代通用网站开发、数据库管理或 Kubernetes Ingress 管理。"
---

# AutoOps Web 管理

本技能维护仓库内已有的 Web 管理功能。先识别请求涉及的页面、接口和执行路径，再修改相关代码；接入真实资产与创建测试数据是不同操作，不能把临时测试实例当作已登记的服务。

## 入口与按需阅读

- 集群高可用接入及持有者切换：阅读 [VIP 教程](../../../docs/web-vip-guide.md)，区分主备切换与尚未实现的地址迁移。
- 页面、筛选、下拉数据、日志或任务查询：阅读 [数据流与接口](references/data-integration.md)，对照实际请求参数和返回结构检查双方；数据源识别、拓扑和 Keepalived/VIP 契约也在该参考中。
- 原生 `nginx.conf` / `conf.d` 编辑、配置分组或证书文件导入导出：参阅 [专项操作指南](../../../docs/2026-11-12-web-native-configurations.md)，重点核对候选校验和发布范围。
- Cloudflare DNS 账户、域名区域与证书申请：参阅 [Cloudflare 接入指南](../../../docs/2026-11-12-web-cloudflare.md)，区分只读检查、签发和部署。
- 流量监控、仪表盘、IP/限速策略与上游分流：参阅 [流量与防护指南](../../../docs/2026-11-13-web-traffic.md)，区分实例计数器、日志覆盖与配置发布。
- 发布、回滚、证书或 Worker：阅读下文执行约束，再按需查阅仓库 [操作指南](../../../docs/2026-11-09-web-management.md)。
- 验证范围和已知环境限制：查阅 [验收记录](../../../docs/web-management-validation.md)，以当前实测为准，不将历史结果视为本次通过。

| 范围 | 代码入口（相对仓库根目录） |
| --- | --- |
| 前端资源页面与表单 | `web/src/views/webservice/Management.vue`、`definitions.ts` |
| 流量与仪表盘 | `web/src/views/webservice/Traffic.vue`、`TrafficPolicyPanel.vue`、`web/src/api/webTraffic.ts`、`web/src/views/dashboard/components/WebTrafficModule.vue` |
| 流量采集与防护 | `api/api/webservice/service/traffic.go`、`traffic_policy.go`、`traffic_policy_parser.go`、`api/api/webservice/model/traffic.go` |
| 数据源、集群拓扑与站点表单 | 同目录 `SourceDialog.vue`、`ClusterDialog.vue`、`ClusterDetail.vue`、`SiteDialog.vue`、`WebResourceIcon.vue` |
| 原生文件与证书导入 | 同目录 `NativeConfigurations.vue`、`CertificateImportDialog.vue` |
| DNS 与签发向导 | 同目录 `DnsAccountDialog.vue`、`DnsAccountInspection.vue`、`CertificateRequestDialog.vue`、`AcmeAccountDialog.vue` |
| 日志与历史事件 | 同目录 `Logs.vue`、`EventJournal.vue`、`web/src/components/LogConsole/index.vue`、`web/src/api/webLogStream.ts` |
| 环境与操作教程 | 同目录 `EnvironmentSelect.vue`、`SetupGuide.vue`、服务端 `environments.go` |
| 日志流与 DNS 传播 | 服务端 `log_stream.go`、`dns_propagation.go`、远端 `runner.py` |
| API 类型与 URL | `web/src/api/webservice.ts`、`platformPermissions.ts`、`platformUrl.ts` |
| 导航与页面权限 | `web/src/router/modules/webservice.ts`、`web/src/utils/platformAccess.ts` |
| REST / SSE | `api/router/webservice/webservice.go`、`api/api/webservice/controller/` |
| 资产、发现、查询 | `api/api/webservice/service/service.go`、`runtime.go`、`audit.go` |
| 接入识别、拓扑与 VIP | 同目录 `source_discovery.go`、`topology.go`、`vip.go` |
| 配置、证书、执行 | 同目录 `render.go`、`configfiles.go`、`certificates.go`、`certificate_import.go`、`operations.go` |
| DNS 检查与签发身份 | 同目录 `dns_accounts.go`、`issuance.go` |
| 采集和通知唤醒 | 同目录 `collector.go`、`events.go` |
| 模型与增量菜单 | `api/api/webservice/model/models.go`、`api/pkg/db/web_migrate.go`、`migrate.go` |
| 远端自动化 | `api/automation/web-platform/runner.py`，由 `assets.go` 嵌入 |
| 权限、密钥和监控 | `api/pkg/authorization/catalog.go`、`api/pkg/websecret/`、CMDB 资产授权、`api/api/monitor/service/web_rules.go` |
| 部署 | `api/main.go`、`docker/docker-compose.yml`、`k8s/api/web-worker-deployment.yaml` |

## 界面与高可用入口

- 数据源接入使用 `SourceDialog.vue`：基本连接 → `POST /source-discovery` 只读识别 → 明确选择候选。路径与服务信息收进高级设置；同机多实例不得合并，已有实例的定位保持不可变。
- 集群新增/编辑通过 `ClusterDialog.vue` 选择数据源下的实例，`SaveCluster` 原子保存元数据与显式成员差量；禁止用可见成员集合覆盖全部成员，权限、归属、版本、任务锁和 VIP 保护在后端复核。
- `ClusterDetail.vue` 使用 `GET /clusters/:id/topology` 显示节点与 VIP 观测；GET 只读后台快照，不进行同步 SSH，过期快照清除旧持有者并保留真实采集时间。Keepalived 使用独立 `autoops-web-vip-<clusterId>` 服务、配置和 PID，不修改数据库 VIP 或系统 Keepalived。
- VIP 配置、核验、切换分别为 `configure-vip` / `verify-vip` / `switch-vip`，均经过计划/Worker；普通集群保存不能修改 VIP。使用本次明确授权的地址和目标主机，预检确认无冲突，不能将历史验收地址当成当前授权。恢复按实际变更节点核对原任务 journal、文件摘要、实例与源版本，不盲目重放切换。
- 证书与管理弹窗挂载到 body，使用视口内最大高度和正文滚动，Web 域关闭 Element Plus body 宽度锁以避免快速重开时的延迟清理竞态；不能以隐藏溢出内容代替可访问的确认按钮。
- 原生文件列表滚动与分页，Tab/Shift+Tab 缩进、Ctrl/Cmd+S 校验保存；文件删除必须确认，只有确认发布后才改变远端。
- 站点专用表单 `SiteDialog.vue` 保留原文与结构化模式。日志按后端登记标识读取；站点日志筛选只匹配已登记路径，共用文件不能宣称完全隔离站点记录。

## 产品与数据约束

- `/web` 下保留数据源、集群、站点、配置、证书、日志、流量七个入口；流量使用独立 `Traffic.vue`。Web 终端入口与票据/连接路由已退役，保留历史审计和存量授权记录。Web 操作审计在“基础管理 → 数据日志”，路径 `/monitor/dblog?tab=web-audit`；不要另造第二套审计。
- 数据源描述连接位置，实例描述已发现的实际运行对象，集群描述运维边界。同机多个配置入口或容器分别定位，一个实例只能属于一个集群。
- 首个引擎是 Nginx；保留引擎和运行环境的分离。不要把未来 Caddy 能力显示为可用，不把 Web 集群当作 Kubernetes 集群；VIP 主备与 Nginx upstream 的请求负载均衡分别维护。
- 数据源引用现有 CMDB 主机和 SSH 连接，不复制密码到前端或配置表。主机与连接须匹配；地址、连接名称可由后端只读展示字段返回，不能返回连接密钥。
- 登记数据源不等于发现成功。SSH 可达但 Nginx 路径不存在时保留数据源和错误证据，不虚构实例、健康状态或已接管状态。读取权限不授权部署或接管服务。
- 尚未发现实例的数据源允许修正定位；已经关联实例的定位保持不可变。保存与发现并发时检查 revision 和源行锁，不直接更新数据库绕过身份约束。
- 新接入默认为观察模式。受管路径、健康检查和基线明确后才接管。移出管理范围保留远端服务及文件。
- 列表先按资产权限确定可见范围，再统计、分页。状态超过 90 秒未采集显示未知，并保留最后检查时间和证据。
- 不使用示例数组代替真实接口，不将请求失败显示为成功空列表。切换目标、路由、标签或关闭弹窗后，旧请求不能覆盖当前数据。
- 所有系统文案使用 `tr` / `$tr` 和 `en-US.json`；用户配置、域名、日志和远端错误保留原文。沿用项目 CSS 变量和主题。

## 必须保持的执行边界

- 所有远端变更经过 `/operation-plans` → `/operations`，提交 `{planId, idempotencyKey, confirmed: true}`。同计划重试复用 `web-plan-<id>`；取消确认不得提交。
- 计划冻结操作者、目标、连接版本、配置基线、渲染文件和证书版本，默认五分钟过期。Worker 执行前重新检查账号、API Key、资产权限、源连接、目标身份及基线。
- 配置、证书部署和服务动作共享集群锁。逐节点发布，候选配置必须在实际运行环境校验。保留绝对/嵌套 include、注释、符号链接和未知指令；原文维护不能被结构化表单覆盖。
- 原生配置实时读取，受管范围外的依赖只可查看。新增、修改、删除都先对完整候选树执行实际 `nginx -t`；保存接口也强制校验，失败不能落草稿。暂存校验不修改线上文件，确认发布仍走计划与 Worker。
- `spec.instanceIds` 明确节点范围，空集合表示整个集群，不能用页面筛选结果静默缩小全群操作。保存与计划使用同一范围；回滚与核验保留原目标。配置分组的 `spec.paths` 在同集群内唯一归属，只组织展示，不改变 include 或发布范围。
- 新建证书导入将资产、版本及审计放在同一事务，校验失败不能留下空资产。历史导出必须校验版本归属；公开证书 PEM 不得夹带私钥。私钥导出叠加管理员、模块写权限及 `web_certificate_manage`，API Key 同样检查；审计失败不返回材料，导出响应禁止缓存。导入导出不自动创建续期策略。
- DNS 账户管理仅管理员可用；已经由管理员绑定到证书的签发身份，可由具有证书管理权限的操作者使用，不因此授予其 Token 或账户管理权限。配置中心引用与专用凭据互斥，切换来源或服务商不能沿用不匹配的旧凭据。
- Cloudflare 连接和域名检查只调用固定 HTTPS API 的读取操作，返回 `writeAccess=unverified`；不能声称已验证 DNS Edit、TXT 写入或 DNS 传播。当前入口发现 ACME 验证记录 CNAME 委派即明确拒绝；区域选择不是隐式扩大 Token 授权。
- 签发计划另存 ACME、DNS 及引用凭据的依赖摘要，提交及 Worker 执行前复核。账户或 Token 变化后重新预检；未完成订单只可在 ACME 身份和域名集合一致时恢复，DNS 凭据修复不改变订单身份。历史缺少身份摘要的订单必须先核验，不自动重放；签发版本和订单完成状态同事务保存。
- DNS 传播失败可使用 TCP 权威查询回退；要求所有 NS 的 AA 响应与本次 TXT 完全匹配，不能通过关闭传播检查来解决网络问题。真实签发、记录清理和导出验证与只读 Token 检查分别报告。
- 成功后核验实际配置、服务、HTTP/TLS 结果。明确失败使用远端原始备份回退；不能读取最新现场后把它当作允许覆盖的基线。
- Worker 或连接失联时可能已经发生变更，进入 `unknown`（结果待核验）并保留锁。恢复先检查远端 journal 和漂移，不直接重放或解锁。
- 新建主机服务的 `-p` 必须与数据源运行前缀一致，表单使用规范 `.../current` 路径。首次安装包时阻止默认 nginx.service 启动，只启用平台独立单元；已有系统包或其他 Nginx（包括 kube-lb）不能被顺带升级或重启。
- 新建实例使用版本目录和原子指针；接管布局只能逐文件原子替换，不声称整体原子性。
- 证书的签发版本、期望部署版本、节点实际版本分开记录。证书使用不等于私钥导出；私钥、ACME 密钥、DNS Token 和计划快照使用外部主密钥加密，API/Worker 共用相同密钥。
- 续期保留授权身份、订单与子任务。提交任务和持久化策略引用在同一事务完成；不自动重放结果不明的部署，人工重试后按实际节点版本恢复调度状态。
- Web 终端菜单、前端路由和生产 API 注册已退役；遗留组件、控制器或票据模型的存在不代表入口仍开放。保留历史连接审计与授权关联，不因维护其他 Web 功能重新注册票据/连接接口。
- 日志通过后端登记的标识访问，保留路径范围、扫描上限、超时和下载审计。实时流使用单连接 SSE → 单 SSH 进程，取消/撤权释放远端与连接配额；不恢复 HTTP 轮询。Web 和 Docker 共用只读 LogConsole；历史下载复用查询快照，实时下载导出已接收缓冲。细节见数据流参考。
- 采集器只收集状态，告警通过监控中心已启用的规则触发。Redis 用于更新通知，SQL 是任务与证据的持久来源。
- 流量计数每 30 秒采集、7 天有界保留；首轮建立游标，重启/计数回退/采集间断不能产生伪造 QPS。日志字节不是网卡吞吐，缺字段返回 null；共享或专用日志覆盖不完整时必须说明，不重复计算同一请求。
- 流量 API 和仪表盘先过滤实例查看权限，日志派生数据叠加 `web_logs`；普通资源详情也必须脱敏日志指标，不能从旁路泄漏。SQL 查询与文件扫描均有上限，游标与聚合采样同事务。
- 防护策略只精确编辑已加载的 HTTP server 和明确代理路径，保留原始字节与静态 URI；手工策略继承、标记漂移或不一致节点必须拒绝覆盖。预览和草稿均需配置编辑权限及实际 nginx 校验；保存后通过 `configurationId` 跳转草稿，不自动发布或自选生产限流阈值。

## 密钥缺失与部署排查

页面显示“Web 加密密钥尚未配置”时，先检查正在运行的 API/Worker 的工作目录、`WEB_ENCRYPTION_KEY`、`WEB_KEY_FILE` 是否一致，不能只检查 Compose 源文件。API 默认读取工作目录下 `data/web.key`，缺失、错误长度或无效环境变量均会导致 `encryptionReady=false`。

首次初始化前确认数据库是否已有加密证书版本、ACME/DNS 密钥或操作计划。已有密文但密钥丢失时恢复原密钥；没有既有密文且当前任务授权部署时，可执行 `go -C api run . -mode web-key-init`。命令只在目标不存在时原子创建 `0600` 的 32 字节文件，禁止在每次 API 启动时无条件生成。主密钥不得写入 skill、文档或工具日志；备份应独立于数据库。

Compose 的 API 与 Worker 使用同一个 `aiops-state` 卷和 `/app/data/web.key`，Worker 只读。文件生成后 API 动态读取即生效；更改代码或容器环境仍需重启相应进程。

## 开发与验证

修改 API 数据契约时同步页面、类型、权限和相关测试；菜单、模型、Worker 参数变化还需同步迁移、部署和文档。保留工作区已有的数据库、任务等模块修改。

按影响执行检查，避免仅为验证文案或机械镜像实现而增加测试：

```bash
go -C api test ./api/webservice/... ./pkg/websecret ./pkg/db ./pkg/authorization ./router
pnpm --dir web webservice:test
pnpm --dir web i18n:test
pnpm --dir web build
```

仅在改动涉及远端执行且存在适当隔离环境时，运行 `api/automation/web-platform/tests`；Docker 测试由 `AUTOOPS_WEB_DOCKER_TEST=1` 显式启用。真实 SSH 测试需要当前任务授权的主机和配置，`TestLiveSSHClusterLifecycle` 使用临时元数据库，不能作为持久接入的证明。不得从历史验收文档推导任意主机的操作授权。

持久部署验收须在当前数据库检查数据源、实例、集群和操作回执，并实际访问节点 HTTP/TLS；浏览器检查数据、配置快捷键与删除确认、日志下载、证书弹窗反复开关、主题和窄屏。涉及 VIP 时核验唯一持有者、业务入口及切换后的独立服务恢复，并确认既有数据库 VIP 与 Nginx 未受影响。Worker 故障注入只能针对本次授权的隔离任务：保留锁，等心跳过期进入待核验，再由恢复计划检查远端记录；测试后恢复原健康地址、终止临时探测服务并撤销临时凭据。

公网 ACME/DNS 验收需要可用测试域名和凭据引用，优先测试 CA。本机自签名证书只能证明导入、部署及 TLS 指纹链路。全项目类型检查或国际化审计存在错误时，分别报告本次新增问题与已有问题；构建通过不等于所有浏览器、生产部署和外部服务场景均已通过。
