# AutoOps 告警规则与数据源 (Alert Rule & Data Source) 参考文档

## 1. 业务概述与文件分布

此部分逻辑主要处理监控领域中**数据源(Data Source)**、**告警规则组(Group Rule)**、**监控告警规则(Alert Rule)**以及**规则分类(Rule Style)**的配置与变更。

### 相关文件结构
- **Controller 层**:
  - `api/monitor/controller/monitor_data_source.go`
  - `api/monitor/controller/monitor_alert_rule.go`
  - `api/monitor/controller/monitor_alert_rule_style.go`
- **Service 层**:
  - `api/monitor/service/monitor_data_source.go`
  - `api/monitor/service/monitor_alert_rule.go`
  - `api/monitor/service/monitor_alert_rule_style.go`
- **DAO 层**: 
  - `api/monitor/dao/monitor_data_source.go`
  - `api/monitor/dao/monitor_alert_rule.go` ...
- **Model 层**:
  - 表/结构体：`MonitorDataSource`, `MonitorAlertGroupRule`, `MonitorAlertRule`, `MonitorAlertRuleStyle`

---

## 2. 核心模块实现思路与方法解读

### 2.1 动态数据源管理 (MonitorDataSource)
用于对接不同的底层监控平台（如 Prometheus, Zabbix 等）。
- **DTO 设计模式 (`MonitorDataSourceDTO`)**:
  - **背景**: 不同的监控系统所需的 `config` 内容或格式大相径庭。前端对于配置信息有时以 String 传入，有时以 Object(JSON) 传入，极易引发 Gin `ShouldBindJSON` 数据类型反序列化失败。
  - **实现**: 在 Controller 获取请求时使用 `MonitorDataSourceDTO`，对 `Config` 字段标记为 `interface{}` 空接口；通过 `ToModel()` 内部的断言开关，归一化序列转换为标准的 String 格式落库，从而保障系统的强壮性。
- **级联删除 (`Delete` 方法)**:
  - **逻辑所在**: `api/monitor/service/monitor_data_source.go`。
  - **实现**: 数据源不仅关联自己，还关联在它之上定义的规则组(Group)及所有规则(Rule)。执行删除操作时，会首先找到对应数据源的所有 Group，清空该 Group 下的 Rules，然后再删除 Group 记录，最后清除数据源记录。没有使用 DB 级别的级联，而是在服务层显式业务级联。

### 2.2 告警组与规则同步 (YAML Sync 制)
Prometheus 告警配置一般以 Group 嵌套 Rules 的 YAML 形式存储加载。为了满足前后端单条规则精确管理以及原生文件级配置的需求，设计了双向同步机制。
- **自上而下解析 (`yamlSyncGroupToRules`)**:
  - 触发场景：用户通过导入/上传原生 YAML 文本创建/修改 Group。
  - 核心逻辑：利用 `gopkg.in/yaml.v3` 解析大块结构 (`promYamlRoot`)，删除旧有的 Group 属下所有规则，再逐条提取构建新的 `MonitorAlertRule` 对象保存进 DB。
- **自下而上拼装 (`yamlSyncRulesToGroup`)**:
  - 触发场景：用户在界面上单独新建、修改或删除了某个具体的 Rule。
  - 核心逻辑：根据变动后的单条规则信息，查出同一 Group 下的所有同伴记录，把它们拼装重组为规范的 Prometheus YAML (包含必要的 Labels 合并等)，写回 Group 的 `rule_content` 字段。

### 2.3 基于 AST 的 PromQL 解析与重写 (`CheckRuleExpr`)
在执行表达式前，常常需要挂载特定边界条件，如限制查询指定节点、指定时间窗口。
- **原理解读**:
  - 直接依赖官方 `github.com/prometheus/prometheus/promql/parser`。
  - 将用户表达式字符串转为 AST，再向下遍历如 `VectorSelector`，用构造好的 `labels.Matcher` 或自定义 Labels 注入。若查无命中，还支持在 AST 树的顶层操作符 (`BinaryExpr` 等) 逻辑反转 (< 与 >) 发起二次探索，以此应对特殊空值指标情况。
- **应用落地**:
  定义于 `monitor_alert_rule.go/modifyPromQL` 等隐式操作中。

### 2.4 规则分类风格 (Rule Style)
- 直接通过 `MonitorAlertRuleStyle` 基础 CRUD，用于给 UI 显示贴标签（如 Memory / CPU / Disk / Network）。属于扁平字典表。
