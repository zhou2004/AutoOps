# 监控分类与故障管理

域名证书和 API 监控各自拥有独立分类，可新增、编辑、搜索和删除分类。列表支持关键字、状态、分类组合筛选；API 关键字同时匹配名称和 URL。新增或编辑目标时可选择分类，勾选目标后可批量归类。删除分类仅把目标转入“未分类”，不会删除监控目标。

## 数据与升级

API 启动时通过 GORM 自动注册 `monitor_category`、`monitor_domain_cert`、`monitor_api_endpoint`、`monitor_incident`，为目标补充 `category_id`，为故障补充可为空的唯一 `event_key`。`category_id = 0` 表示未分类；列表查询省略 `categoryId` 表示所有分类，显式传 `0` 只查询未分类。

已有旧版故障表中的 `alert_desc`、`alert_level` 和数字状态会迁移到当前模型。状态 `1/2` 转为 `firing`，`3` 转为 `resolved`；来源标记为 `legacy`，处理人、原因、解决方案等原字段保留。转换可重复执行，不覆盖已经转换的状态。初始化 SQL 同步补齐新表和字段，并保留其中的历史故障记录。已有环境通过应用启动升级，不需要重新导入完整 SQL 初始化文件。

## 告警接入

故障在统一 Prometheus Webhook 处理入口写入，发生在通知路由之前，所以未配置通知渠道也会保存有效告警。内部域名/API 规则通过 `monitor_source`、`monitor_source_id`、`monitor_rule_id` 标识来源；普通 Prometheus 告警使用标签作为身份。

同一告警身份、同一 `startsAt` 属于同一事件，重复通知不会重复建故障；恢复事件会更新该记录。已自动恢复或人工解决的事件不会被迟到的 `firing` 重开，新一轮开始时间会创建新故障。告警需提供有效的 RFC3339 `startsAt`。内部规则以目标 ID 维持评估状态，响应耗时、状态或剩余天数的变化不会重置持续时间。

监控检查仍只保存检查结果；域名/API 故障需要在告警配置中启用相应规则并满足持续时间后产生。此次接入不会追溯导入历史通知表 `monitor_alert_record`，旧故障表中的记录会保留并转换。未接收到告警时页面显示真实空态。

## 接口

| 方法 | 路径（前缀 `/api/v1/monitor`） | 用途 |
| --- | --- | --- |
| GET | `/categories?scope=domain_cert` 或 `api_endpoint` | 查询该类型全部分类 |
| POST / PUT | `/category` | 新增 / 编辑分类，参数 `id`（编辑）、`scope`、`name`、`description` |
| DELETE | `/category/:id?scope=...` | 删除分类并解除目标归类 |
| POST | `/category/assign` | 批量归类，参数 `scope`、`ids`（最多 1000 个）、`categoryId` |
| GET | `/domain-cert/list`、`/api-endpoint/list` | 增加可选 `categoryId` 条件 |
| GET | `/incident/list` | 支持 `keyword`、`status`、`level`、`source`、`startDate`、`endDate`、分页 |
| GET | `/incident/stats` | 全量故障统计，附加 `criticalFiring` 与连续 7 天的 `trend` |

故障概览展示全量存储数据，列表筛选不影响概览。日期范围按告警时间筛选，结束日期包含当天；今日和趋势使用 API 服务进程时区。列表活跃故障优先，页大小最多 100。故障页支持 30 秒刷新、详情时间线、人工解决和删除，读取失败明确提示，不将错误伪装成零数据。

## 验证

- `cd api && go test ./api/monitor/... ./pkg/db/... ./router/monitor/...`
- `cd web && pnpm build`
- 后端回归覆盖分类隔离、未分类筛选、批量操作原子性、检查结果与分类变更并发、故障重复/乱序事件、人工解决、统计读取错误和旧故障迁移。

浏览器验证使用独立测试响应，未向业务数据库写入测试故障。覆盖分类增删改、分类筛选、批量归类、单项检查、API URL 搜索、故障详情与状态筛选，以及空态、失败、明暗主题和 390px 窄屏。

预览截图使用测试数据：[故障管理（浅色）](images/monitor-incident-light.png)、[故障管理（深色）](images/monitor-incident-dark.png)、[移动端](images/monitor-incident-mobile.png)、[域名分类](images/monitor-domain-categories.png)、[API 监控](images/monitor-api-dark.png)。
