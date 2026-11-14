# Web 流量监控、访问防护与负载均衡

Web 管理新增“流量监控”，仪表盘增加同一数据源的“Web 流量概览”。页面只汇总当前账号有权查看的实例；日志衍生的流量、状态码、来源、URI 与上游数据还要求实例的日志权限。

## 采集口径

Worker 每 30 秒采集一次，保存 7 天聚合数据。首次采集只建立计数器和文件游标基线，从后续请求开始形成趋势；不会导入整份历史日志。支持最近 1 小时、6 小时、24 小时和 7 天，实例超过 90 秒未产生可用采样时显示未知。

| 指标 | 来源与含义 |
| --- | --- |
| QPS、活跃连接 | Nginx `stub_status`；QPS 是连续有效采样的请求计数差除以时间差，重启/计数回退后重新建立基线 |
| 响应流量 | 已登记访问日志的响应字节；常规 combined 日志为 `body_bytes_sent`，不包含响应头或 TLS 开销，也不是主机网卡流量 |
| 请求流量 | JSON 日志具有 `request_length` 时采集，包含请求行、头和正文 |
| 状态码、5xx 错误率 | 可解析访问日志；错误率为 5xx 请求占比，403/429 单独进入状态码分布 |
| 平均请求耗时 | JSON 日志具有 `request_time` 时采集，按已记录请求加权，单位毫秒 |
| 来源 IP、URI | 日志中 Nginx 实际看到的来源；URI 去掉查询参数，原始请求记录不写入元数据库 |
| 上游流向 | JSON 日志具有 `upstream_addr` 时展示实际访问的上游，重试链保留原值；不将配置权重当成实际请求分布 |

缺少字段显示“—”，有界扫描、无法解析、共享日志、采集间断与权限不足均披露覆盖范围。日志轮转/截断会检查文件身份及游标锚点；多个同主机实例共用而无法归属的日志不重复计数。查询使用有界流式读取，达到查询上限时显示部分数据。

普通日志同时覆盖所有站点时可统计实例总量。启用专用流量日志后，统计优先使用专用日志，排除普通日志以避免一个请求被重复计数；此时字节和上游统计只覆盖已启用的站点。Nginx `stub_status` 仍是整个实例的请求计数，包括探测请求。官方指标定义见 [Nginx 状态模块](https://nginx.org/en/docs/http/ngx_http_stub_status_module.html)。

## 策略操作

在流量页选择集群和可选的单实例，打开“防御与流量分配”。读取实际 Nginx 配置中的 HTTP `server`，按配置文件、监听端口和域名选择目标。列表采用“详情、编辑、更多”，支持搜索、10/20/50/100 分页及跳页。

支持的策略：

- IP/CIDR 允许和拒绝列表，支持 IPv4/IPv6，拒绝优先；存在允许列表时构成白名单，拒绝返回 403。
- 按客户端 IP 的每秒请求限额、突发量及 `nodelay`；超限返回 429。
- 按客户端 IP 的并发请求连接上限，超限返回 429；HTTP/2/3 并发请求的语义沿用 Nginx。
- 每请求响应速率上限；不代表站点总带宽配额。
- 轮询、最少连接、IP 哈希，及上游成员权重、备用、停用、失败阈值与失败超时。IP 哈希不接受备用成员。
- 将选中的现有代理路径导向配置的上游组，保留原静态 `proxy_pass` URI 后缀；变量或无法安全解析的代理地址拒绝自动改写。
- 可选专用 JSON 流量日志，保存现有访问日志并加入请求长度、耗时与上游字段，供下一轮采集识别。日志存放在既有访问日志目录，需要纳入该主机的日志轮转和保留策略。

策略按 Nginx 的 `remote_addr` 生效；存在可信反向代理时，应先在原生配置中明确可信 `real_ip` 来源，不能直接信任外部请求发送的转发头。限速机制见 [请求限速](https://nginx.org/en/docs/http/ngx_http_limit_req_module.html)、[连接限制](https://nginx.org/en/docs/http/ngx_http_limit_conn_module.html)。上游语义见 [Nginx 上游模块](https://nginx.org/en/docs/http/ngx_http_upstream_module.html)。这些是 Nginx 应用层访问与流量策略，不提供上游网络清洗或完整 WAF 规则库。

## 校验、发布与恢复

1. 读取目标时冻结实际文件摘要；界面编辑不会修改服务。
2. 预览只修改明确选择的 `server` 和相关 `http` 作用域，保留原生注释、未知指令及文件结构。已有手工 ACL/限速继承冲突、动态 include、不一致节点或标记漂移会拒绝自动覆盖。有防御规则时，提前结束请求的 `return/rewrite` 以及可能改变拒绝结果的 `error_page` 也拒绝自动叠加，需先明确原生配置行为。
3. 完整候选配置在目标实际运行环境执行 `nginx -t`；错误不能保存草稿。
4. 保存后跳转配置管理并定位该草稿。用户通过现有预检、确认及 Worker 发布流程执行，发布权限与编辑权限分离。
5. “移除策略”同样先生成恢复草稿并校验，确认发布后才生效。仅恢复有原始元数据和一致受管标记的片段，不能盲目覆盖人工修改。

原生文件被策略修改后，相关站点沿用已有机制切换原文维护，避免结构化表单重新覆盖策略。Keepalived/VIP 负责入口主备；Nginx upstream 负责业务请求分流，两者使用不同配置并可组合。

## API 与存储

| API | 权限与用途 |
| --- | --- |
| `GET /api/v1/web/traffic` | `web:read` + 实例查看权限；支持 `range/groupId/clusterId/instanceId`，返回汇总、时序、节点、覆盖及日志聚合 |
| `GET /api/v1/web/clusters/:id/traffic-policy` | 读取授权配置目标，返回 `instanceIds/baselines/targets/warnings`；可选 `instanceId` |
| `POST .../traffic-policy/preview` | `web:write` + 配置编辑/查看权限，返回逐文件差异及节点语法校验结果 |
| `POST .../traffic-policy/draft` | 同样校验后保存普通配置草稿；不直接重载服务 |

草稿提交仍使用 `/operation-plans` 与 `/operations`，后者要求执行及发布权限。仪表盘复用流量 API，无额外绕过资产授权的统计接口。`web_traffic_state` 保存增量游标，`web_traffic_sample` 保存有界聚合；游标推进和采样写入同事务，过期数据由采集器清理。

监控中心现有 Web 风格规则可使用 `qps`、`response_bytes_per_second`、`request_bytes_per_second`、`error_rate`（百分数）、`average_latency_ms` 与 `connections_active`。仅启用的规则进入通知链路；缺失、过期或采集失败的指标不伪造为零，也不使用旧值触发流量告警。

本轮验证结果见 [Web 验收记录](web-management-validation.md)。
