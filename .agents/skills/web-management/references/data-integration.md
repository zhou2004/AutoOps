# Web 管理的数据流与接口

用于前端数据接入、空页面、筛选失效、过期结果或任务状态排查。路径均相对仓库根目录；先看实际源码，以下是维护时需要保持的契约。

## 请求与页面

`Management.vue` 的 `section` 来自路由 `meta.webSection`，证书/站点的标签再决定 `kind`。API 请求经 `webservice.ts` → `platformRequest`，返回项目 `{code,message,data}` 中的 `data`。

- 当前七个路由为 `/web/sources`、`/web/clusters`、`/web/sites`、`/web/configurations`、`/web/certificates`、`/web/logs`、`/web/traffic`；最后一项使用独立 `Traffic.vue`。Web 操作审计继续使用 `/monitor/dblog?tab=web-audit`。
- HTTP 前缀为 `/api/v1/web`。`apiV1Base` 接受源站、网关前缀或已经包含 `/api/v1` 的配置，只添加一次版本前缀。
- `platformRequest` 携带当前 token；HTTP 401 和响应体 401 都使会话失效。慢预检允许较长超时，用户切换日志目标时用 AbortController 取消旧请求。
- `/web/terminal`、`POST /terminal-tickets` 和 Web 终端连接 API 已退役。生产路由不调用遗留 `RegisterTerminal`；保留控制器/模型不代表可用，不为兼容旧页面恢复注册。已有终端事件继续按审计权限展示。
- `loadOptions` 按页面需要拉取业务分组、实例、集群、证书等真实选项，各响应独立落地；使用 `view=options` 排除观察快照和配置文件内容，不用整份配置树填下拉；管理员才加载 SSH 目标与 ACME/DNS 管理账户。选项失败应清除相应旧数据并展示错误。
- 列表刷新与选项刷新是独立动作；新建、发现或任务结束后更新两者，不能只刷新表格而保留旧下拉列表。
- `worker-status.encryptionReady` 反映运行中的 API 是否能读取主密钥。先定位运行环境和密文存量，再按 skill 的初始化流程处理；不要通过隐藏提示来绕过计划加密。
- 站点和上游状态由成员实际文件与渲染结果决定；“配置草稿保存成功”不能直接设为已发布。
- 首次数据为空应显示真实空态。不要为填充页面而自动部署 Nginx、生成证书或接管服务。

## 主要契约

| 接口 | 返回 / 注意事项 |
| --- | --- |
| `GET /:kind` | `{list,total}`；`name,groupId,clusterId,engine,environment,status,page,size`；size 上限 100 |
| `GET /:kind/:id`、`POST /:kind`、`PUT /:kind/:id` | Resource；保存已有记录携带 `revision`，冲突不得静默覆盖 |
| `GET /environments` | 授权资源已使用的环境字符串数组；前端与内置 production/staging/test/development 合并，可清空或自定义，不迁移旧值 |
| `GET /groups` | 已授权业务分组数组，包含 `id,parentId,name` |
| `GET /targets` | 管理员可用的 `{hosts,connections}`；连接选项按 `assetId === hostId` 过滤 |
| `POST /source-discovery` | `{hostId,connectionId,runtime,container?,sudo?}` → `{candidates,warnings}`；候选为 `{id,name,spec,identity,version,warnings}`，只读扫描不创建数据源 |
| `POST /sources/:id/check`、`/discovery` | 发现成功返回数据源；失败返回错误并记录最近结果，不创建虚假实例 |
| `POST /clusters`、`PUT /clusters/:id` | Resource 加可选 `memberChanges: {add: uint[], remove: uint[]}`；省略/空变更保留原成员，集群与成员在同一事务保存；PUT 必须带 revision |
| `GET/POST /clusters/:id/members` | GET 为成员数组；POST 为 `{instanceId,remove}` |
| `GET /clusters/:id/capabilities` | 动作、标签、是否启用及原因；不是客户端硬编码的可执行清单 |
| `GET /clusters/:id/topology` | `{cluster,members,entries,vip,checkedAt}`；成员带 `instance,address,status,identity,version,configDigest,vip?`；VIP 观测独立于实例健康状态 |
| `GET /traffic` | `range=1h/6h/24h/7d,groupId,clusterId,instanceId` → `{summary,series,nodes,topIPs,topURIs,upstreams,statusCodes,coverage}`；缺失值 null，字节/日志排行叠加日志权限 |
| `GET /clusters/:id/traffic-policy` | 可选 `instanceId` → `{clusterId,instanceIds,baselines,targets,warnings}`，读取真实已加载 server 和当前平台策略 |
| `POST /clusters/:id/traffic-policy/preview` | `{instanceIds,path,serverIndex,baselines,policy}` → `{files,checks,valid,policy,upstreamName,warnings}`；编辑权限与实际 nginx 校验 |
| `POST /clusters/:id/traffic-policy/draft` | 同上加可选 `name` → `{draft,preview}`；生成普通草稿，通过 `configurationId` 定位并单独发布 |
| `GET /sites/:id/render`、`/upstreams/:id/render` | 配置草稿 `ConfigSpec`，生成后仍要保存并预检发布 |
| `GET /instances/:id/config` | 实时文件、`editable` / `included`、身份和摘要；受管范围外只读 |
| `POST /clusters/:id/config/validate` | `{files,instanceIds?,baselines?,variables?}` → 节点校验结果；不写线上配置 |
| `POST/PUT /configurations[/id]` | 在持久化前强制校验完整候选配置，错误不保存；已有草稿携带 `revision` |
| `/config-groups` | 集群内 `spec.paths` 唯一归属；草稿列表支持 `configGroupId` 筛选 |
| `POST /certificates/import` | `{name,groupId,environment,certificate,privateKey}` → `{certificate,version}`；事务创建 |
| `POST /certificates/:id/import` | 已有资产导入新版本，不自动部署 |
| `GET /certificates/:id/versions` | `{list,currentVersionId}`，版本元数据不含私钥 |
| `GET /certificates/:id/versions/:versionId/export?key=false` | 版本 `0` 表示当前；私钥须显式 `key=true` 且额外授权，审计失败拒绝返回 |
| `GET /dns-credentials?provider=cloudflare` | 管理员可见的脱敏凭据选项 `{id,name,provider}[]`，不返回 Token 或密文 |
| `GET /dns-accounts/:id/zones` | `{zones,total,truncated,checkedAt}`；只读发现最多 500 个区域 |
| `POST /dns-accounts/:id/check` | `{domain?}` → 检查项、实际检查域名、匹配区域和 `writeAccess: unverified`；管理员写权限，保存最近检查和审计 |
| `GET /config-revisions` | 配置用 `clusterId`，证书用 `certificateId`；两者权限不同 |
| `POST /operation-plans` | 冻结计划、checks、executable、expiresAt、idempotencyKey |
| `GET /operation-plans/:id/preview` | 按节点的渲染文件，私钥内容已移除，仍需配置查看权限 |
| `POST /operations`、`GET /operations/:id` | 前者返回任务，后者 `{operation,steps}`；关闭页面不影响执行 |
| `GET /operations/:id/events` | SSE，仍经过用户和资产权限校验；DB 是可恢复的证据源 |
| `GET /audit` | `{list,total}`；接受资源过滤及任务 status，目标名通过 `name` 返回 |
| `GET /audit/events` | 连接/查询事件，`clusterId,before` 游标；total 是授权后的剩余数量 |
| `GET /worker-status` | `online,count,checkedAt,encryptionReady`；接口失败与 Worker 离线应区别显示 |
| `GET /instances/:id/logs/stream` | 带 Authorization 的 SSE，支持 `logId,keyword,status,ip,uri,lines`；当前文件追踪，历史参数不允许混用 |
| `GET /instances/:id/logs` | 无 logId 返回登记的 `{id,path,kind?}[]`；查询返回 `text,path,lines,at,offset,window,nextOffset,hasMore,scanLimit`，下载复用同一查询参数 |

`Resource.spec` 是对应资源的结构化配置，`observed` 是已采集事实。`Resource.connection` 是 CMDB 只读展示信息，包含地址、端口、用户名、主机名、连接名称；不能作为更新 SSH 凭据的入口。

## 列表与联动

- 普通资源列表的行操作保留“详情”“编辑”，其他操作进入“更多”菜单，按原有权限和能力显示；审计行仅提供详情。底部分页居中，使用服务端 `page/size/total`，支持每页 10/20/50/100 条和跳页；修改每页数量或筛选条件回到第一页，末页数据减少后回退到有效页。原生文件列表在已读取文件上单独分页，游标事件查询不套用页码分页。
- 数据源用连接位置定位，不能把业务站点 URL 作为 SSH 管理地址。
- 集群过滤数据源时，通过实例成员关系获取 source ID；证书通过部署绑定获取 certificate ID，不直接查资源自身的 `cluster_id`。
- 业务分组包含子组。日志和原生配置的实例选择也应应用页面筛选，不能只让普通表格筛选生效。
- 审计状态是 `queued/preflight/running/verifying/succeeded/failed/rolled-back/unknown`，与实例健康状态不同。
- `unknown` 在操作页显示“结果待核验”，在采集状态显示“未知”。不要因为映射相同就混淆两者。
- URL 的 `clusterId` 保留并可清除；`instanceId` 可预选原生配置或日志目标。站点跳转日志携带 `siteId`；失效站点不得静默回落到其他实例日志。
- 保存、实例发现及任务轮询使用请求代次，旧结果和旧错误都不能覆盖新的目标。组件卸载后不再追加轮询定时器。

## 数据源识别、拓扑与 VIP

`ClusterDialog.vue` 在集群新增和编辑中直接选择数据源下已发现的 Nginx 实例，使用授权后的轻量 sources/instances 选项。按业务分组和引擎筛选，其他集群占用实例禁选，已有 VIP 成员禁止取消；不能依靠前端筛选代替后端校验。保存发送显式 add/remove 差量，不发送完整覆盖集合，以免删除不可见成员。`SaveCluster` 在事务中复用保存与成员校验；失败时包括元数据、成员和审计全部回滚。成员改动还要求 execute 和双方资产 manage，保持只改元数据的权限兼容。成员改动递增实例/集群版本并清除旧拓扑快照；任务提交与成员编辑共享集群行锁，旧成员集合的计划不得执行。


`SourceDialog.vue` 默认展示基本连接字段，高级定位参数折叠。`POST /source-discovery` 要求管理员、`web:write` 和目标主机 SSH 权限，连接须属于该主机且已启用；扫描上下文限时 90 秒。识别返回运行中 Nginx 的独立候选及未识别项告警；选择候选后仍通过数据源保存与已有 `/sources/:id/discovery` 登记实例。不要把一次扫描成功等同于完成接入，也不要用第一个进程覆盖同机其他实例。已关联实例的数据源定位不可更换。

`ClusterDetail.vue` 从拓扑接口展示授权成员、业务入口和 VIP 持有者。GET 只读取后台每约 30 秒更新的快照，剥离实例完整观察文件和集群冗余观察值，不等待 SSH。快照超过 90 秒、时间异常或 VIP 配置摘要不匹配时返回未知且清空旧持有者，`checkedAt` 不伪装为请求时间；能力接口与拓扑并行独立显示。`vip` 可能为空；存在时结构为 `{config,status,ownerInstanceId,nodes,checkedAt,message?}`，`config` 是下述 VIP 参数。`nodes` 按实例 ID 返回 `bound,active,owned,healthy,address,mac,configDigest` 等观测。界面应分别展示服务存活、业务健康、唯一绑定及采集时间；成员不可见、探测失败或状态过期时不能把缓存状态画成正常流量。

VIP 操作仍使用 `POST /operation-plans`，`targetType="clusters"`、`targetId=clusterId`：

| action | 参数与效果 |
| --- | --- |
| `configure-vip` | `parameters.vip={cidr,interface,routerId,instanceIds,preferredInstanceId?}`；新建独立 Keepalived 服务，已有 VIP 配置不能直接覆盖 |
| `verify-vip` | 可省略 `parameters.vip` 复用集群已存配置；核验服务、唯一持有者和经 VIP 的业务响应 |
| `switch-vip` | 可省略 VIP 参数；短暂停止当前持有者的独立 Web VIP 服务，核验其他节点接管后恢复原服务，`nopreempt` 避免立即抢回 |
| `reconcile` | `parameters.operationId` 指向结果待核验的原任务；检查原冻结目标和远端执行证据，不重新发布或重放切换 |

提交仍为 `{planId,idempotencyKey,confirmed:true}`。VIP 计划要求管理员、模块执行权限和目标主机命令权限；主机需要可写 SSH 连接、不同宿主机上的 2–10 个已接管主机 Nginx 实例。预检检查 IPv4/CIDR、网卡、VRID、已有 Web/数据库 VIP、ARP/VRRP、真实 Nginx 监听归属和健康地址。已有数据库 VIP 不应被误当主机主地址，也不能为通过预检而移除它们。集群锁与 VIP 地址/VRID 锁约束并发变更。

独立服务为 `autoops-web-vip-<clusterId>`，文件在 `/etc/autoops-web-vip/<clusterId>/`，PID 独立保存；不能操作系统默认或数据库的 Keepalived 服务。切换在停止前设置 120 秒的独立恢复定时器；成功核验后恢复服务并撤销定时器。连接中断或结果不明保留原锁。

恢复先复查当前操作者权限、源 revision、实例身份和 Nginx 基线。全新部署节点按原操作 journal 和文件清单摘要核验，未完成部署可仅回退本次创建的独立服务/文件，已安装依赖包保留；检测漂移时拒绝覆盖。切换仅原持有者有该次变更 journal，备选节点及 `verify-vip` 不应被要求提供不存在的写日志，而应与冻结的 VIP 配置摘要比较。恢复授权使用当前操作者，审计与远端 journal 仍关联原任务 ID。

## 原生配置与证书页面

`NativeConfigurations.vue` 是配置页默认标签；高级草稿、配置分组保留独立标签。读取远端后保留基线，新增/编辑/删除提交候选文件；服务端保存成功只代表草稿通过校验。发布需冻结计划并再次确认，不能以“已保存”替代节点生效状态。

- `ConfigSpec.instanceIds` 非空时只操作指定成员，空数组/缺省时表示整个集群。全群操作由后端解析所有成员和逐节点授权，不能从可见列表或页面筛选推导完整成员。
- 保存选择 `configGroupId` 时，把本次非删除文件加入该组 `paths`；同集群相同路径不能属于两个组。分组不决定部署目标，也不修改文件内容。
- 左侧文件列表垂直滚动并支持 10/20/50/100 条分页；过滤重置页码，分页不能改变发布范围。Tab/Shift+Tab 保留多行选择和原文换行，Ctrl/Cmd+S 调用强制校验保存，不能绕过后端校验；删除已有文件或未保存新文件均先确认，取消保留内容。
- 切换实例、内部标签和路由均保护未保存编辑；读取使用请求代次和取消信号，URL 初始实例只负责预选，不能在选项刷新时覆盖用户当前选择。
- 草稿保存前的校验与发布计划都使用节点变量、基线和绑定证书材料；私钥不能进入草稿响应或日志。发布完成只有修订号与内容摘要仍匹配的草稿才标记为已发布。
- `CertificateImportDialog.vue` 支持文件选择或粘贴 PEM；SAN、有效期、指纹由后端解析。关闭后清除输入，较早的文件读取不能覆盖新的输入或已关闭弹窗。公开导出与私钥下载分开，历史版本按钮传实际版本 ID。导入成功不表示已部署，不自动启用续期。

详细流程见 [原生配置与证书导入导出](../../../../docs/2026-11-12-web-native-configurations.md)。

`SiteDialog.vue` 将站点设置保存与生成草稿分开；生成后仍需候选校验和发布计划。配置路径按集群实例的真实配置目录建议，成员目录不一致时不得套用 `/etc/nginx` 默认值，用户手工覆盖的路径保持不变。HTTPS 使用当前集群已建立的证书绑定并检查域名覆盖；访问/错误日志路径分别为 `accessLog/errorLog`，原文模式保留原生配置。

## Cloudflare 账户与证书申请

`DnsAccountDialog.vue` 单独编辑凭据来源，配置中心下拉由 `/dns-credentials` 提供。已有专用 Token 不回显；不输入时保留旧值，切换服务商或来源后必须重新选择有效凭据。成功保存不等于连接已验证，凭据变化会清除旧检查结果。

`DnsAccountInspection.vue` 对已保存账户执行只读检查并显示可访问区域。无指定域名时抽样一个可用区域；指定域名时检查对应区域。读取成功不证明 DNS 写权限，`_acme-challenge` CNAME 委派返回当前流程不支持。检查保存时复核账户版本和实际凭据摘要，不能把较早请求的结果挂到新 Token 上。

`CertificateRequestDialog.vue` 使用现有 Cloudflare DNS 账户、区域及 ACME 账户。根域名和泛域名分别进入 SAN；更改域名或账户使旧检查失效。保存申请仅写平台资产，预检和确认签发仍走现有计划/操作流程。`AcmeAccountDialog.vue` 默认测试 CA，保存本身不注册账户或签发。

签发计划的依赖摘要与证书修订号独立，包含 ACME、DNS 账户配置和加密凭据材料的哈希；提交和执行前再次比较。未完成订单的身份摘要绑定 CA、ACME 账户密钥和域名集合，允许修复 DNS Token 后用新计划重试。签发后的版本保存与订单 `stored` 同事务，避免状态落库失败导致重复版本。

`dns_propagation.go` 在 lego 传播检查失败后补充 TCP 权威核验（SOA → NS → 所有 NS TXT），要求 AA、正确名称和本次完整挑战值，CNAME/缺失/部分失败均拒绝。不能关闭校验来适配 UDP DNS 劫持或陈旧缓存。

接入 Cloudflare、保存申请或导入证书不会隐式创建续期策略；需要自动续期时，按当前用户需求单独配置授权身份及部署目标。更多说明见 [Cloudflare 接入指南](../../../../docs/2026-11-12-web-cloudflare.md)。

## 日志与历史事件

`webLogQuery` 统一生成 `logId,keyword,rotation,status,ip,uri,lines,offset,from,to`。下载使用最后一次显示结果对应的实例和筛选快照，不能漏掉日期范围或使用用户尚未提交的新输入。查询失败清空旧结果，切换实例先清空日志选项；实时追踪由 `webLogStream.ts` 使用 fetch 读取 SSE，携带 Authorization 且绑定 AbortSignal，不使用轮询或自动重试。

SSE 事件为 `ready/log/notice/reset/heartbeat/error/end`，文本来自一个持久 SSH 进程，host 文件使用已登记路径与 nofollow 文件描述符。处理重命名轮转、截断及部分行；Docker 输出使用 logs --follow，容器文件通过宿主机 `/proc/<PID>/root` 读取，不要求容器安装 Python。运行中检查容器 ID/进程与文件范围，不追踪重建后的陌生目标。

后端每 10 秒复核账号/API Key、资产权限、实例与连接档案，最多 20 分钟自动结束；进程级总计 32 条、每用户 4 条连接。浏览器暂停或切换目标时取消并释放额度；下游写入超时限制 15 秒。开始/结束写审计，错误不可变成空成功。单次初始读取限制 1 MiB、单行 64 KiB、积压 2 MiB，前端只保留最近 10000 行/2 MiB；断线后明确由用户重连最新窗口，不承诺精确续传。

`LogConsole` 与 Docker 共用只读 xterm，日志中的 ANSI/OSC 控制字符按字面显示，不能执行终端控制。高级筛选和范围筛选折叠后保留启用数量；调整窗口、布局、全屏、退出全屏均重新 fit。历史下载复用查询，实时下载当前缓冲；隐藏历史分页不能导致读取新窗口时沿用旧实时状态。

按服务端 `nextOffset` 推进历史窗口，达到 `scanLimit` 后停止；当前行数选择与正在展示窗口信息分别记录。站点筛选是前端将 `accessLog/errorLog` 与实例登记日志路径匹配，并限制所属集群的实例，不向后端传任意路径，也不把 `siteId` 当成新的日志权限边界。同文件中的多站点记录仍需关键词等查询条件区分；站点失效、无独立路径或未发现文件时明确提示，不静默显示无关日志。

`EventJournal.vue` 保留过去的终端连接事件及当前日志查询事件，继续按来源资产授权和游标读取。终端退役不删除历史审计、角色关联或表；当前验收检查入口/API 未重新开放，而不要求启动旧终端会话。

## 接入真实资产时的检查

1. 查当前配置指向的数据库和 API，确认 Web 表、菜单和已登记记录；不要从测试报告猜测资产 ID。
2. 复用目标 CMDB 主机、同主机启用的 SSH 档案和业务分组。按实际定位查重，重复操作复用已有数据源。
3. 经领域服务或已认证 API 保存，保留资产授权、定位校验、审计和观察模式。不要直接写出“healthy”记录。
4. 运行只读发现；程序或配置路径不匹配时显示错误，等待修正或已授权的部署动作。注册来源数据与安装远端服务是不同范围。
5. 从页面使用的列表/详情接口检查名称、ID、分组、连接和真实观测。持久接入的数据应在正式配置的元数据库；临时 SQLite 只用于测试。

排查时可参考仓库 [部署与验收指南](../../../../docs/2026-11-09-web-management.md)。如果修改中的后端尚未重启，区分磁盘代码、当前运行二进制与数据库中的数据，不将编译成功描述为已更新在线服务。
