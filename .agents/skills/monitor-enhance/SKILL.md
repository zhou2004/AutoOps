---
name: monitor-enhance
description: "理解 AutoOps 监控中心增强模块（域名证书监控 / API端点监控 / 故障管理 / 告警规则引擎集成）的完整架构、API 接口、数据库设计和告警触发链路。适用于新增监控类型、修改规则引擎、维护故障管理等场景。"
---

# AutoOps 监控中心增强模块 Skill

此技能记录监控中心增强模块（域名证书监控、API端点监控、故障管理、告警规则引擎集成）的完整架构设计。

---

## 1. 核心架构设计

### 1.1 设计原则

**检查与告警分离**：
- 域名证书和 API 端点监控**只负责检查并保存结果到数据库**，不自动触发告警
- 告警触发统一由**告警规则引擎**（`monitorAlertRuleService.evaluateRulesOnce()`）管理
- 用户必须在 `监控中心 → 告警配置` 中创建并启用对应风格的规则，才会触发 Webhook 通知

### 1.2 规则引擎扩展

在 `evaluateRulesOnce()` 中的执行顺序：

```
evaluateRulesOnce()
  ├── evaluateInternalRules()    ← domain_cert / api_endpoint 规则评估（新增）
  ├── 遍历数据源，查询 PromQL
  └── processRuleEvaluation()    ← Prometheus 规则评估（原有）
```

`evaluateInternalRules()` 会：
1. 查询所有启用的规则（`style = "domain_cert"` 或 `style = "api_endpoint"`）
2. 获取所有域名证书记录 / API端点记录
3. 对每条规则逐条检查记录是否匹配（先检查 constraints，再检查 expr）
4. 通过 `processInternalEval()` 使用 Redis 管理状态（pending → firing → resolved）
5. 状态变化时调用 `sendWebhook()` 发送到内部 webhook 端点
6. Prometheus 规则引擎的 `sendWebhook()` 同样走此路径

### 1.3 告警触发链路

```
每30秒评估一次
  → 规则条件满足 → pending 状态持续 for_duration 时间
  → 状态变为 firing → sendWebhook()
  → 内部 POST → /api/v1/monitor/alert/webhook/prometheus
  → PrometheusAlertHandle 匹配告警路由
  → 根据标签 (labels) 匹配 → 发送通知 (钉钉/企微/飞书/Webhook)
```

---

## 2. 数据库表

### 2.1 monitor_domain_cert
```sql
id              INT UNSIGNED AUTO_INCREMENT PRIMARY KEY
domain          VARCHAR(255) NOT NULL UNIQUE   -- 域名
port            INT DEFAULT 443                -- 端口
issuer          VARCHAR(512)                   -- 颁发者
subject         VARCHAR(512)                   -- 主题
not_before      VARCHAR(64)                    -- 起始日期
not_after       VARCHAR(64)                    -- 到期日期
remaining_days  INT DEFAULT -1                 -- 剩余天数(-1=未知)
status          TINYINT DEFAULT 1              -- 1-正常,2-即将过期,3-已过期,4-检查失败
check_time      VARCHAR(64)                    -- 最近检查时间
error_msg       TEXT                           -- 错误信息
create_time     DATETIME(3) NOT NULL
update_time     DATETIME(3) NOT NULL
```

### 2.2 monitor_api_endpoint
```sql
id                  INT UNSIGNED AUTO_INCREMENT PRIMARY KEY
name                VARCHAR(255) NOT NULL            -- 名称
url                 VARCHAR(1024) NOT NULL            -- 监控URL
method              VARCHAR(16) DEFAULT 'GET'         -- 请求方法
headers             JSON                              -- 请求头
body                TEXT                              -- 请求体
check_interval      INT DEFAULT 300                   -- 检查间隔(秒)
timeout             INT DEFAULT 10                    -- 超时(秒)
expected_code       INT DEFAULT 200                   -- 期望状态码
expected_body       VARCHAR(512)                      -- 期望响应体内容
last_status_code    INT DEFAULT 0                     -- 最后状态码
last_response_time  BIGINT DEFAULT 0                  -- 最后响应时间(ms)
status              TINYINT DEFAULT 1                 -- 1-正常,2-异常,3-超时,4-失败
check_time          VARCHAR(64)                       -- 最近检查
error_msg           TEXT                              -- 错误信息
create_time         DATETIME(3) NOT NULL
update_time         DATETIME(3) NOT NULL
```

### 2.3 monitor_incident
```sql
id              INT UNSIGNED AUTO_INCREMENT PRIMARY KEY
title           VARCHAR(512) NOT NULL           -- 故障标题
source          VARCHAR(128)                    -- 来源(domain_cert/api_endpoint/prometheus)
source_id       INT DEFAULT 0                   -- 来源ID
level           VARCHAR(32) DEFAULT 'warning'   -- critical/warning/info
status          VARCHAR(32) DEFAULT 'firing'    -- firing/resolved
description     TEXT                            -- 描述
alert_time      VARCHAR(64)                     -- 告警时间
resolved_at     VARCHAR(64)                     -- 解决时间
create_time     DATETIME(3) NOT NULL
update_time     DATETIME(3) NOT NULL
```

---

## 3. 后端文件结构

```
api/api/monitor/
├── model/domain_cert.go          # DomainCert, MonitorAPIEndpoint, MonitorIncident 模型
├── dao/
│   ├── domain_cert.go            # 域名证书 DAO
│   ├── api_endpoint.go           # API 端点 DAO
│   └── incident.go               # 故障 DAO（含 GetStats 统计）
├── service/
│   ├── domain_cert.go            # 域名证书服务（TLS 连接检查证书）
│   ├── api_endpoint.go           # API 端点服务（HTTP 请求检查）
│   ├── incident.go               # 故障管理服务
│   └── monitor_alert_rule.go     # 规则引擎扩展（evaluateInternalRules + match + processInternalEval）
└── controller/
    ├── domain_cert.go            # 域名证书控制器
    ├── api_endpoint.go           # API 端点控制器
    └── incident.go               # 故障控制器
```

---

## 4. 规则表达式系统

### 4.1 域名证书规则 (style = "domain_cert")

表达式格式：`字段 运算符 数值`，如 `remaining_days <= 30`

| 字段 | 类型 | 说明 |
|------|------|------|
| `remaining_days` | int | 证书剩余天数 |
| `status` | int | 状态码(1=正常,2=即将过期,3=已过期,4=失败) |

约束条件(JSON)：`{"domain": "example.com", "port": "443"}`

### 4.2 API端点规则 (style = "api_endpoint")

表达式格式：`字段 运算符 数值`，如 `status != 1`

| 字段 | 类型 | 说明 |
|------|------|------|
| `status` | int | 状态码(1=正常,2=异常,3=超时,4=失败) |
| `last_status_code` | int | HTTP 返回状态码 |
| `last_response_ms` | int | 响应时间(ms) |

约束条件(JSON)：`{"name": "生产API", "url": "https://api.example.com/health"}`

### 4.3 运算符支持

`<=`, `>=`, `<`, `>`, `==`, `=`, `!=`

---

## 5. 状态管理机制

### 5.1 Redis 状态键

```
alert:eval:rule:{ruleID}
```

每个 key 是一个 Hash，field 为 metric 的 JSON fingerprint，value 为：
```json
{"active_at": 1234567890, "state": "pending|firing"}
```

### 5.2 状态转换

```
首次匹配 → pending (记录active_at)
pending + 持续 for_duration → firing (发送webhook)
不再匹配:
  - 当前为 firing → 发送 resolved webhook, 删除状态
  - 当前为 pending → 删除状态, 下次重新开始
```

---

## 6. API 接口路由

### 域名证书 (prefix: /api/v1/monitor/domain-cert)
| POST | `/domain-cert` | 添加 |
| PUT | `/domain-cert` | 编辑 |
| DELETE | `/domain-cert/:id` | 删除 |
| GET | `/domain-cert/:id` | 详情 |
| POST | `/domain-cert/batch-delete` | 批量删除 |
| GET | `/domain-cert/list` | 分页列表 |
| GET | `/domain-cert/check/:id` | 检查单个 |
| POST | `/domain-cert/check-all` | 检查全部 |

### API端点 (prefix: /api/v1/monitor/api-endpoint)
| POST | `/api-endpoint` | 添加 |
| PUT | `/api-endpoint` | 编辑 |
| DELETE | `/api-endpoint/:id` | 删除 |
| GET | `/api-endpoint/:id` | 详情 |
| POST | `/api-endpoint/batch-delete` | 批量删除 |
| GET | `/api-endpoint/list` | 分页列表 |
| GET | `/api-endpoint/check/:id` | 检查单个 |
| POST | `/api-endpoint/check-all` | 检查全部 |

### 故障管理 (prefix: /api/v1/monitor/incident)
| GET | `/incident/list` | 故障列表 |
| GET | `/incident/stats` | 故障统计 |
| POST | `/incident/resolve/:id` | 解决故障 |
| DELETE | `/incident/:id` | 删除故障 |

---

## 7. 前端页面

| 路由 | 组件 | 说明 |
|------|------|------|
| `/monitor/domain-cert` | `web/src/views/monitor/DomainCert.vue` | 域名证书监控 |
| `/monitor/api-endpoint` | `web/src/views/monitor/APIEndpoint.vue` | API端点监控 |
| `/monitor/incident` | `web/src/views/monitor/Incident.vue` | 故障管理大屏 |

API 函数统一在 `web/src/api/monitor.js` 中，路由注册在 `web/src/router/monitor.js`。

---

## 8. 故障管理统计

`GET /api/v1/monitor/incident/stats` 返回：
```json
{
  "totalFiring": 3,
  "totalResolved": 10,
  "byLevel": {"critical": 2, "warning": 8, "info": 3},
  "bySource": {"domain_cert": 5, "api_endpoint": 3, "prometheus": 5},
  "last24hCount": 2,
  "todayCount": 1
}
```

---

## 9. 使用流程

1. **添加监控目标** → 域名证书监控 / API端点监控页面
2. **创建告警规则** → 告警配置页面（选风格、填表达式、设约束、设标签、启用）
3. **配置通知路由** → 告警通知页面（建模板、建路由、关联模板）
4. **查看故障** → 故障管理大屏（监控统计、故障列表、解决/删除）