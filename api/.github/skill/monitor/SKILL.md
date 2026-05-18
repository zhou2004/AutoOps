---
name: monitor
description: AutoOps系统的Monitor监控与告警模块上下文、实体架构和开发最佳实践。涉及Prometheus告警、Webhooks、规则、数据源等。
applyTo: 
  - "api/monitor/**"
  - "**/*monitor*/**"
  - "**/*alert*/**"
---

# AutoOps 监控与告警开发指南 (Monitor & Alert Skill)

如果你正在开发或调试监控模块(`monitor`)、告警路由(`alert router`)、告警规则(`alert rule`)或者数据源(`data source`)相关功能，请参考以下架构设计和最佳实践。

💡 **延伸阅读 (Reference Documents)**:
- 详细告警流转与路由分发逻辑参见: `references/monitor_alert.md`
- 详细数据源DTO设计与规则AST解析逻辑参见: `references/monitor_alert_rule.md`
- 基础的监控 API 定义参见: `references/monitor.md`

## 1. 核心业务与代码分布

监控模块主要位于 `api/api/monitor/` 目录下：

- **Controller (`api/monitor/controller/`)**: 
  - `monitor_data_source.go`: 监控数据源API（注意使用 DTO 模式处理动态 JSON config）。
  - `monitor_alert_rule.go`: 告警规则与群组 API（包含表达式检查、级联查询）。
  - `alert.go`: Webhooks 接收端（Prometheus, Zabbix, Gitlab）、告警模板与告警路由管理。
- **Service (`api/monitor/service/`)**: 
  - `monitor_alert_rule.go`: 包含 PromQL AST 重写（动态注入 Constraints 和反转运算符）、Yaml 同步与拆解。
  - `monitor_data_source.go`: 处理实体级的关联逻辑（如删除时级联删除群组和规则）。
  - `alert.go`: 包含告警的流转逻辑（Alert Evaluator -> Rule Matcher -> Webhook -> DB Template 模版渲染 -> 外部通知如微信/钉钉）。
- **Model (`api/monitor/model/`)**: 
  - `monitor_data_source.go`, `monitor_alert_group_rule.go`, `monitor_alert_rule.go`, `alert.go`。

## 2. 关键设计模式与踩坑记录

### 2.1 动态 JSON 字段的反序列化 (DTO 模式)
- **问题**: Web 前端传递的数据源 `config` 参数时而为序列化的 JSON 字符串 (`"{}"`)，时而为 JSON 对象 (`{}`)，导致 Gin 的 `ShouldBindJSON` 在严格类型（如 `string` 或 `map`）下频繁报错 `cannot unmarshal object into Go struct field`。
- **方案**: 
  - 必须在 Controller 层使用 DTO（如 `MonitorDataSourceDTO`），将动态字段声明为 `interface{}`。
  - 在 `ToModel()` 转换方法中，通过类型断言 (type assertion) 将 `interface{}` 安全转换为 Gorm 可接受的字符串形式存入数据库。

### 2.2 实体级联删除
- **问题**: 虽然 Gorm 提供 ForeignKey 的级联删除，但有些遗留外键或者跨服务的删除操作更容易出错，导致出现悬空记录（Dangling records）。
- **方案**: 在 Service 层手动保障级联逻辑。例如，删除 `MonitorDataSource` 时，必须显式查出属于它的 `MonitorAlertGroupRule`，并调用 DAO 删除组下的 `MonitorAlertRule`，最后再删除 Group 和 DataSource。

### 2.3 PromQL AST 操作与表达式动态改写
- **问题**: 需要对前端传来的 PromQL 动态插入约束条件 (Constraints, 如 `instance="192.168.1.1"`)，或者对空结果执行反转查询（例如将 `>` 改为 `<` 看是否有数据）。
- **方案**: 完全抛弃字符串正则替换，统一使用 `github.com/prometheus/prometheus/promql/parser` 的 AST 解析器（`parser.ParseExpr`）。在 AST 树中递归遍历 `*parser.VectorSelector` 注入 LabelMatchers，或者反转 `*parser.BinaryExpr` 的操作符，然后调用 `.String()` 重新还原为 PromQL。

### 2.4 Alert Webhook 接收与模板渲染
- **问题**: 触发告警恢复时，Prometheus 发送的 Webhook 渲染由于缺失变量（如 `EndsAt` 或特定自定义函数报错）导致 `template: panic`。
- **方案**: 
  - 保证入库并进入渲染引擎前，赋予默认的时间戳占位（如 `EndsAt = time.Now()`），或检查相关的 `Tpl` 逻辑边界。
  - 使用注册到 `template.New()` 的自定义函数（如 `GetCSTtime`）对时间进行标准时区的格式化。

## 3. 常见开发 Checklist

1. **接口修改**: 如果调整了某个 `api/monitor/controller` 接口的返回或参数，注意检查前端 JSON 的宽松类型。
2. **Prometheus 集成**: 添加任何新的监控指标时，优先考虑在 `service/monitorService.go` 或报警验证阶段增加 AST 检测，以防止注入非法的 PromQL。
3. **数据一致性**: 在 `model` 定义的外键操作时，始终在 `service` 中确保相关的关联表（规则分类 - 规则组 - 具体规则）进行了相应的增删同步。
