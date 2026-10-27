# 仪表盘业务数据

仪表盘 `/dashboard` 改为读取 `GET /api/v1/dashboard/overview?days=7`。支持 `days=7/30/90`，默认 7 天；页面每分钟刷新，可关闭自动刷新。页面隐藏或离开时停止自动查询。

## 数据口径

| 模块 | 业务来源 | 统计方式 |
| --- | --- | --- |
| 主机、数据库、物理机、网络设备 | `cmdb_host`、`cmdb_sql`、`cmdb_physical_machine`、`cmdb_network_device` | 当前台账记录，先按当前账号的 `list` 资产权限过滤 |
| 集群 | `k8s_cluster` | 使用集群列表现有权限中间件；展示保存的集群状态 |
| 业务应用 | `app_application`、`cmdb_group` | 全平台应用及业务组分布，状态 1 未激活、2 已激活；前 10 组以外合并，失效分组仍纳入统计 |
| 快速发布 | `quick_deployments` | 一条发布批次计一次，不将下属任务重复计数 |
| 任务作业 | `task_execution_history` | 一次完整执行计一次，不使用 `task_work` 的主机/模板明细代替执行次数 |
| Ansible | `task_ansible_history` | 一次历史执行计一次 |
| Agent | `monitor_agent` | 仅统计可见且仍存在的主机，每台主机选最新 Agent 记录；状态 3 且最近 5 分钟内有心跳才列为心跳正常 |
| API、证书 | `monitor_api_endpoint`、`monitor_domain_cert` | 最近一次保存的检查结果；没有 `check_time` 的对象列为“未检查” |
| 告警记录 | `monitor_alert_record` | 留存记录数，按触发/恢复状态分类；不与故障数相加 |
| 故障 | `monitor_incident` | 当前未完成记录及所选日期新增故障；支持新旧两套表结构 |

日期按北京时间的自然日计算，包含今天，截至本次查询时间。趋势中的无记录日期补零。执行指标按记录的 `created_at` 归属日期；运行、等待、取消、暂停、停止不会计为失败。完成成功率为 `成功 /（成功 + 失败）`，没有完成记录时返回 `null`，页面显示 `—`。

“累计留存”与“最近执行”读取全部留存历史，最近执行最多 8 条；这两项不受日期筛选限制。执行趋势和执行状态指标受日期筛选限制。任务历史有保留数量限制，因此累计留存不代表系统自上线以来全部执行次数。

主机认证状态、Agent 心跳、集群保存状态分别展示。台账条数不等同于去重后的硬件数量（例如同一物理主机可能有物理资产台账和主机管理记录）。仪表盘读取过程不进行 SSH、TCP 探测或远程任务。

## 旧版故障表

当前数据库可能沿用 SQL 初始化文件中的 `alert_desc / alert_level / status(1,2,3)` 结构。接口读取列元数据后选择对应字段：旧版状态 1、2 为待处理，3 为已完成；保留 P1/P2/P3/P4 级别。新版使用 `title / level / source`，`status=firing` 为待处理。仪表盘不迁移或改写历史记录，也不改变故障管理模块的其他接口。

## 返回与异常处理

返回沿用 `{code, message, data}`。`data` 包含 `generatedAt`、`days`、`timezone`、`startDate`、`endDate`、`assetScope`、各模块结果及 `errors`。

- 真实空数据返回数量 0、空数组；模块查询失败返回该模块 `null`，并在 `errors` 中给出提示。其他模块继续展示。
- 认证或权限读取失败不会退回全量资产。
- 模块汇总仅返回数量、状态和有限的执行/故障名称，不返回凭据、连接信息、命令、日志正文或配置。
- 整体刷新失败保留上次数据，并明确显示原更新时间；切换日期时清除旧范围，防止数据与筛选条件错配。
- 请求有 12 秒数据库查询期限，无跨用户缓存；图表随主题、容器尺寸变化更新，销毁时释放监听与实例。

本次仅新增读取接口和前端展示，不需要配置或部署清单调整。发布时需要同时更新 Go API 与前端构建产物。

## 验证

```bash
cd api
go test ./api/dashboard/... ./router/dashboard/...

# 可选：对已配置数据库执行只读事务校验，不运行迁移。
DASHBOARD_LIVE=1 DASHBOARD_CONFIG_PATH="$PWD/config.yaml" \
  go test ./api/dashboard/service -run TestDashboardConfiguredDatabaseReadOnly -v -count=1

cd ../web
pnpm build
```

自动测试覆盖认证、参数范围、权限过滤、状态计数、日期边界、每日补零、成功率、Agent 心跳、未检查监控对象、缺表降级、孤立业务组及新旧故障表。已对配置数据库完成只读核对，并在浏览器使用读取到的业务数据验证深浅色、窄屏、日期切换、历史记录、加载失败、刷新保留、部分失败、空数据和重试。

全仓库类型检查目前受既有 `ignoreDeprecations: "6.0"` 与当前 TypeScript 版本不匹配影响。使用命令行覆盖 `--ignoreDeprecations 5.0` 后仍有其他模块的既有类型错误，仪表盘修改文件未出现类型错误。
