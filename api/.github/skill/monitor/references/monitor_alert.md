# AutoOps 告警模块 (Alert) 实现思路与逻辑架构

## 1. 架构概述
Alert 模块的核心目标是接收来自不同监控系统（Prometheus, Zabbix, Gitlab, 阿里云等）的告警和事件，经过内部规则路由（Router）、模板渲染（Template）后，将规范化的消息分发到不同的通知渠道（钉钉、微信、飞书、邮件等），并对分发的记录进行留存，供审计和历史查询。

## 2. 逻辑架构与数据流

整个告警处理数据流可分为以下几个阶段：
`Webhook 接收 -> 数据格式化 -> 路由分发(Router) -> 模板渲染(Template) -> 渠道发送(Sender) -> 记录持久化(Record)`

### 2.1 Webhook 接收层 (Receiver)
- 暴露标准的 HTTP 接口供外部系统调用 (`/monitor/alert/webhook/prometheus` 等)。
- **多数据源兼容处理**: 支持 Prometheus 原生告警，同时在同一接口通过 Query 参数 `?from=aliyun` 兼容阿里云云监控告警结构 (`AliyunAlert`) 的映射转换。
- **数据提取**: 解析外部系统传入的 JSON Payload（`p_json` 和 `p_alertmanager_json`），并提取出 URL 参数中的动态配置信息。

### 2.2 路由分发层 (Router - `AlertRouter`)
- **定义**: 决定一条外部触发的告警应该发送给谁、通过什么媒介发送以及采取什么模版。
- **匹配逻辑**: 在收到告警时，通过 `GetAllAlertRouter` 获取全局定义的路由记录，匹配对应信息，然后组装出分发队列。支持 AtSomeone, 轮询(RR)、是否发送恢复通知(SendResolved) 等高级分发特性。

### 2.3 模板渲染引擎 (Template - `PrometheusAlertDB`)
- **定义**: 告警消息的展示内容样式，元数据存储在 `monitor_prometheus_alert` 表的中。
- **渲染技术**: 采用 Go 原生 `text/template` / `html/template`，具有高自由度。
- **内置增强函数**: `TransformAlertMessage` 阶段注册了诸多自定义函数（如 `GetCSTtime` 格式化时间、`TimeFormat`, `GetTimeDuration`, `toUpper`, `SplitString` 等），从而能够在模板中直接对 UTC 时间进行 CST 转换，或完成富文本拼接，避免 panic 和展示错乱。

### 2.4 渠道发送层 (Sender)
- 分析路由与模板最终决定的目标媒介进行 API 对接投递。
- 涵盖的核心发送通道：
  - **企业微信 (WeChat)**: `PostToWeiXin`，基于 Markdown 格式投递。
  - **钉钉 (DingDing)**: `PostToDingDing`，重点包含对 Webhook Secret 的安全 HmacSHA256 加签认证算法。
  - **飞书 (FeiShu)**: 囊括早期和新版本接口，支持 V1 与 V2 卡片消息 (`PostToFeiShuv2`)。
  - **邮件 (Email)**: `SendEmail` 借由 smtp / gomail 高效送达。
  - 其他如电话 (Phone Call), Bark, TG, Telegram 等。

### 2.5 数据持久化与生命周期 (Record)
- **留痕**: 成功触发规则的告警事件，会将告警的等级、触发时间、恢复时间、摘要等概况构建为 `AlertRecord` ，落库至表 `monitor_alert_record` (`SetRecord`)。
- **归档清理**: 提供 API `CleanRecords`，以便清理因大量触发告警而长久堆积的历史行以节省 DB 存储资源。

## 3. 设计亮点与最佳实践
1. **多重入口兼容**: 在单一长线收口接口中提取 `Query` 参数进行旁路反序列化（如阿里云的支持），以最小变动接收多元数据，减少了 API 路由的冗余繁杂。
2. **松散数据契约(Interface{} 传递)**: Controller 层获取上游 Webhook JSON 后采用 `p_json interface{}` 直接传递给模板引擎，放弃冗余的结构体绑定，保障上游系统字段动态调整扩列时，模板引擎皆能灵活获取而不会报错拦截。
