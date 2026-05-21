# 监控中心增强 - 域名证书/API端点监控 + 告警规则引擎集成

> 日期: 2026-05-19  
> 概述: 在监控中心中补充域名证书监控和 API 端点监控，支持证书过期检测、API 健康检查，并通过告警规则引擎统一管理触发逻辑。

---

## 一、新增功能总览

| 功能模块 | 说明 | 路由 |
|---------|------|------|
| 域名证书监控 | 监控域名 TLS 证书过期时间，支持添加/编辑/删除/检查 | `/monitor/domain-cert` |
| API 端点监控 | 监控 HTTP API 可用性，支持请求方法/头/体/状态码/响应时间检查 | `/monitor/api-endpoint` |
| 故障管理大屏 | 告警故障统计仪表盘，按等级/来源/时间分布可视化 | `/monitor/incident` |
| 告警规则引擎 | 扩展支持 domain_cert / api_endpoint 风格规则的自动评估 | `monitor_alert_rule` |

---

## 二、后端架构

### 2.1 目录结构

```
api/api/monitor/
├── model/
│   └── domain_cert.go          # DomainCert, MonitorAPIEndpoint, MonitorIncident 模型
├── dao/
│   ├── domain_cert.go          # 域名证书 DAO
│   ├── api_endpoint.go         # API 端点 DAO
│   └── incident.go             # 故障管理 DAO
├── service/
│   ├── domain_cert.go          # 域名证书服务 (TLS 检查)
│   ├── api_endpoint.go         # API 端点服务 (HTTP 检查)
│   ├── incident.go             # 故障管理服务
│   └── monitor_alert_rule.go   # 规则引擎 (新增 evaluateInternalRules)
└── controller/
    ├── domain_cert.go          # 域名证书控制器
    ├── api_endpoint.go         # API 端点控制器
    └── incident.go             # 故障管理控制器
```

### 2.2 数据库表

```sql
-- 域名证书监控表
CREATE TABLE monitor_domain_cert (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    domain VARCHAR(255) NOT NULL UNIQUE COMMENT '域名',
    port INT DEFAULT 443 COMMENT '端口',
    issuer VARCHAR(512) COMMENT '颁发者',
    subject VARCHAR(512) COMMENT '主题',
    not_before VARCHAR(64) COMMENT '起始日期',
    not_after VARCHAR(64) COMMENT '到期日期',
    remaining_days INT DEFAULT -1 COMMENT '剩余天数(-1=未知)',
    status TINYINT DEFAULT 1 COMMENT '状态:1-正常,2-即将过期,3-已过期,4-检查失败',
    check_time VARCHAR(64) COMMENT '最近检查时间',
    error_msg TEXT COMMENT '错误信息',
    create_time DATETIME(3) NOT NULL,
    update_time DATETIME(3) NOT NULL
);

-- API端点监控表
CREATE TABLE monitor_api_endpoint (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL COMMENT '名称',
    url VARCHAR(1024) NOT NULL COMMENT '监控URL',
    method VARCHAR(16) DEFAULT 'GET' COMMENT '请求方法',
    headers JSON COMMENT '请求头',
    body TEXT COMMENT '请求体',
    check_interval INT DEFAULT 300 COMMENT '检查间隔(秒)',
    timeout INT DEFAULT 10 COMMENT '超时(秒)',
    expected_code INT DEFAULT 200 COMMENT '期望状态码',
    expected_body VARCHAR(512) COMMENT '期望响应体',
    last_status_code INT DEFAULT 0 COMMENT '最后状态码',
    last_response_time BIGINT DEFAULT 0 COMMENT '最后响应时间(ms)',
    status TINYINT DEFAULT 1 COMMENT '状态:1-正常,2-异常,3-超时,4-失败',
    check_time VARCHAR(64) COMMENT '最近检查时间',
    error_msg TEXT COMMENT '错误信息',
    create_time DATETIME(3) NOT NULL,
    update_time DATETIME(3) NOT NULL
);

-- 故障记录表
CREATE TABLE monitor_incident (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(512) NOT NULL COMMENT '故障标题',
    source VARCHAR(128) COMMENT '来源(domain_cert/api_endpoint/prometheus)',
    source_id INT DEFAULT 0 COMMENT '来源ID',
    level VARCHAR(32) DEFAULT 'warning' COMMENT '等级:critical/warning/info',
    status VARCHAR(32) DEFAULT 'firing' COMMENT '状态:firing/resolved',
    description TEXT COMMENT '描述',
    alert_time VARCHAR(64) COMMENT '告警时间',
    resolved_at VARCHAR(64) COMMENT '解决时间',
    create_time DATETIME(3) NOT NULL,
    update_time DATETIME(3) NOT NULL,
    KEY idx_source (source),
    KEY idx_status (status),
    KEY idx_level (level)
);
```

### 2.3 API 接口列表

#### 域名证书监控 (`/api/v1/monitor/domain-cert/*`)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/domain-cert` | 添加域名 |
| PUT | `/domain-cert` | 编辑域名 |
| DELETE | `/domain-cert/:id` | 删除域名 |
| GET | `/domain-cert/:id` | 获取域名详情 |
| POST | `/domain-cert/batch-delete` | 批量删除 |
| GET | `/domain-cert/list` | 列表查询(分页+搜索) |
| GET | `/domain-cert/check/:id` | 检查单个证书 |
| POST | `/domain-cert/check-all` | 检查全部证书 |

#### API端点监控 (`/api/v1/monitor/api-endpoint/*`)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api-endpoint` | 添加 API |
| PUT | `/api-endpoint` | 编辑 API |
| DELETE | `/api-endpoint/:id` | 删除 API |
| GET | `/api-endpoint/:id` | 获取详情 |
| POST | `/api-endpoint/batch-delete` | 批量删除 |
| GET | `/api-endpoint/list` | 列表查询 |
| GET | `/api-endpoint/check/:id` | 检查单个 |
| POST | `/api-endpoint/check-all` | 检查全部 |

#### 故障管理 (`/api/v1/monitor/incident/*`)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/incident/list` | 故障列表 |
| GET | `/incident/stats` | 故障统计 |
| POST | `/incident/resolve/:id` | 解决故障 |
| DELETE | `/incident/:id` | 删除故障 |

---

## 三、告警规则引擎集成

### 3.1 设计原则

**域名证书和 API 端点监控不自动触发告警**。检查结果仅保存在数据库中，由告警规则引擎统一评估。只有用户在"告警配置"中创建并启用了对应规则，才会触发 Webhook 通知。

### 3.2 规则引擎扩展

在 `monitorAlertRuleService.evaluateRulesOnce()` 中：

```
evaluateRulesOnce()
  ├── evaluateInternalRules()    ← 新增: domain_cert / api_endpoint 规则评估
  └── 原有的 Prometheus 规则评估
```

### 3.3 支持的规则表达式

#### 域名证书 (Style = `domain_cert`)
| 表达式示例 | 说明 |
|-----------|------|
| `remaining_days <= 30` | 证书剩余天数 <= 30 天 |
| `remaining_days <= 7` | 证书剩余天数 <= 7 天 |
| `status != 1` | 证书状态异常 |
| `status > 1` | 证书状态不正常 |
| `status == 3` | 证书已过期 |

**支持字段**: `remaining_days`, `status`  
**支持运算符**: `<=`, `>=`, `<`, `>`, `==`, `!=`

#### API端点 (Style = `api_endpoint`)
| 表达式示例 | 说明 |
|-----------|------|
| `status != 1` | API 状态异常 |
| `last_status_code != 200` | 返回状态码非 200 |
| `last_response_ms >= 5000` | 响应时间 >= 5000ms |

**支持字段**: `status`, `last_status_code`, `last_response_ms`  
**支持运算符**: `<=`, `>=`, `<`, `>`, `==`, `!=`

### 3.4 约束条件 (Constraints)

通过规则中的 `Constraints` 字段（JSON 格式）精准限定目标范围。

**域名证书约束示例**:
```json
{"domain": "www.example.com", "port": "443"}
```

支持的约束字段: `domain` / `domain_name`, `port`, `status`

**API端点约束示例**:
```json
{"name": "生产环境API", "url": "https://api.example.com/health"}
```

支持的约束字段: `name`, `url`, `method`, `status`

> 约束条件必须**全部匹配**才会对目标进行评估。留空或 `{}` 表示作用于所有目标。

### 3.5 告警触发链路

```
规则评估(30s间隔)
  → 匹配条件满足 (pending 持续 for_duration)
  → 状态变为 firing
  → sendWebhook() 发送到内部 webhook
  → PrometheusAlertHandle() 匹配告警路由
  → 根据 labels 匹配对应的通知方式(钉钉/企微/飞书/Webhook等)
```

### 3.6 状态恢复机制

- 当规则条件不再满足时，状态从 `firing` → 立即发送 `resolved` webhook
- 状态从 `pending` → 直接清除，下次重新开始
- 目标记录从数据库删除 → Redis 状态清理 → 发送 `resolved`

---

## 四、前端页面

### 4.1 页面路由

| 路径 | 组件 | 菜单标题 |
|------|------|---------|
| `/monitor/domain-cert` | `DomainCert.vue` | 域名证书监控 |
| `/monitor/api-endpoint` | `APIEndpoint.vue` | API端点监控 |
| `/monitor/incident` | `Incident.vue` | 故障管理 |

### 4.2 域名证书监控页面功能
- 搜索筛选（域名、状态）
- 添加/编辑域名（域名、端口）
- 删除/批量删除
- 检查单个/全部证书
- 查看证书详情（颁发者、有效期、剩余天数等）
- 状态标签可视化（正常/即将过期/已过期/检查失败）

### 4.3 API端点监控页面功能
- 搜索筛选（名称、状态）
- 添加/编辑（名称、URL、方法、请求头、请求体、间隔、超时、期望状态码/响应体）
- 查看详情（状态码、响应时间、错误信息）
- 检查单个/全部端点
- 状态标签可视化（正常/异常/超时/失败）

### 4.4 故障管理大屏页面功能
- 四个统计卡片：活跃故障 / 已解决 / 24小时新增 / 今日新增
- 按告警等级分布（严重/警告/信息）
- 按来源分布（域名证书/API监控/Prometheus）
- 故障列表（支持搜索状态/等级/来源）
- 解决/删除操作

---

## 五、配置使用流程

### 添加监控 → 配置告警规则 → 设置通知路由

1. **配置域名监控**
   - 进入 `监控中心 → 域名证书监控`
   - 点击"添加域名"，输入域名和端口
   - 系统自动检查证书状态

2. **配置 API 监控**
   - 进入 `监控中心 → API端点监控`
   - 点击"添加API"，填写 URL、方法、期望状态码等
   - 系统自动发起 HTTP 检查

3. **创建告警规则**
   - 进入 `监控中心 → 告警配置`
   - 在"分类管理"中创建 `domain_cert` 或 `api_endpoint` 分类（如不存在）
   - 新建规则，选择对应分类
   - 填写表达式：如 `remaining_days <= 30`
   - 可选：填写约束条件：如 `{"domain": "example.com"}`
   - 设置标签（用于匹配告警路由）
   - 启用规则

4. **配置通知路由**
   - 进入 `监控中心 → 告警通知`
   - 创建告警模板（选择 Webhook/钉钉/企微/飞书等）
   - 创建路由，设置标签匹配规则
   - 关联模板

---

## 六、文件变更清单

### 后端新增文件

| 文件 | 行数 | 说明 |
|------|------|------|
| `api/api/monitor/model/domain_cert.go` | ~230 | DomainCert + MonitorAPIEndpoint + MonitorIncident 模型 |
| `api/api/monitor/dao/domain_cert.go` | ~80 | 域名证书 DAO |
| `api/api/monitor/dao/api_endpoint.go` | ~80 | API 端点 DAO |
| `api/api/monitor/dao/incident.go` | ~100 | 故障 DAO (含统计) |
| `api/api/monitor/service/domain_cert.go` | ~180 | 域名证书服务 (TLS 检查) |
| `api/api/monitor/service/api_endpoint.go` | ~200 | API 端点服务 (HTTP 检查) |
| `api/api/monitor/service/incident.go` | ~70 | 故障管理服务 |
| `api/api/monitor/controller/domain_cert.go` | ~100 | 域名证书控制器 |
| `api/api/monitor/controller/api_endpoint.go` | ~100 | API 端点控制器 |
| `api/api/monitor/controller/incident.go` | ~30 | 故障控制器 |

### 后端修改文件

| 文件 | 说明 |
|------|------|
| `api/router/monitor/monitor.go` | 新增域名证书/API端点/故障路由 |
| `api/api/monitor/service/monitor_alert_rule.go` | 扩展规则引擎(新增 evaluateInternalRules + match 引擎 + processInternalEval) |
| `api/sql/update.sql` | 新增 3 张建表 SQL |

### 前端新增文件

| 文件 | 说明 |
|------|------|
| `web/src/views/monitor/DomainCert.vue` | 域名证书监控页面(含编辑) |
| `web/src/views/monitor/APIEndpoint.vue` | API端点监控页面 |
| `web/src/views/monitor/Incident.vue` | 故障管理大屏页面 |

### 前端修改文件

| 文件 | 说明 |
|------|------|
| `web/src/api/monitor.js` | 新增域名证书/API端点/故障 API 函数 |
| `web/src/router/monitor.js` | 新增 3 个路由 |