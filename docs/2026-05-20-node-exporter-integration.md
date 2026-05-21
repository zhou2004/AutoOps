# Node Exporter 集成方案 - 替代自定义 Agent 监控

> 日期: 2026-05-20  
> 目标: 在现有 Agent 管理模块中增加 node_exporter 扫描功能，直接通过 HTTP 拉取 node_exporter 的 `/metrics` 端点获取主机监控数据，摆脱自定义 Agent 部署限制。

---

## 一、设计思路

### 1.1 现状问题
- 当前使用自定义自部署 Agent 采集主机监控数据，需要编译、SSH 传输、Systemd 部署等一系列复杂操作
- Agent 采集数据后推送到 Pushgateway，再由 Prometheus 查询，链路较长
- 对于已部署 node_exporter 的主机，无法复用现有基础设施

### 1.2 解决方案
在 Agent 管理模块中增加 **Agent 扫描** 功能：
1. 用户选择目标主机，发起 node_exporter 扫描
2. 系统通过 SSH 连接到主机，检查 node_exporter 进程和端口（默认 9100）
3. 验证 HTTP 端点 `http://{host}:{port}/metrics` 是否可访问
4. 将扫描结果（node_exporter 端点 URL）存储到 Agent 记录中
5. 主机监控接口 `GetHostMetrics` 优先直接拉取 node_exporter 的 `/metrics` 数据并解析
6. 如果 node_exporter 不可用，降级到原有的 Prometheus 查询模式

---

## 二、数据库模型扩展

### 2.1 Agent 表新增字段

在 `monitor_agent` 表中增加以下字段：

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `node_exporter_url` | varchar(512) | '' | node_exporter 端点 URL，如 `http://192.168.0.51:9100/metrics` |
| `node_exporter_status` | tinyint(1) | 0 | node_exporter 状态: 0-未扫描, 1-在线, 2-离线, 3-未安装 |
| `node_exporter_port` | int(11) | 9100 | node_exporter 监听端口 |
| `node_exporter_scan_time` | varchar(64) | '' | 最近扫描时间 |

### 2.2 节点扫描 DTO

```go
type NodeExporterScanDto struct {
    HostIDs []uint `json:"hostIds" binding:"required"` // 要扫描的主机ID列表
    Port    int    `json:"port"`                        // 扫描端口，默认9100
}

type NodeExporterResult struct {
    HostID   uint   `json:"hostId"`
    HostName string `json:"hostName"`
    SSHIP    string `json:"sshIp"`
    Port     int    `json:"port"`
    URL      string `json:"url"`
    Status   int    `json:"status"`   // 1-在线, 2-离线, 3-未安装
    ErrorMsg string `json:"errorMsg"`
}
```

---

## 三、node_exporter 扫描流程

```
选择主机 → 发起扫描
  → 通过 SSH 连接主机
  → 检查是否存在 node_exporter 进程 (ps aux | grep node_exporter)
  → 检查端口是否监听 (ss -tunlp | grep :{port})
  → HTTP 请求 http://{ip}:{port}/metrics 验证
    └─ 成功: 记录 URL, 标记状态为在线
    └─ 失败: 标记状态
  → 更新数据库记录
```

---

## 四、监控数据获取改造

### 4.1 直接从 node_exporter 拉取指标

新增 `queryNodeExporterMetrics(hostID uint)` 方法：
1. 从 Agent 记录获取 `node_exporter_url`
2. HTTP GET 请求该 URL
3. 解析 Prometheus text format，提取关键指标
4. 填充 `HostMetrics` 结构体

### 4.2 支持的 node_exporter 指标

| 原始指标名 | 计算公式 | 对应字段 |
|-----------|---------|---------|
| `node_cpu_seconds_total{mode="idle"}` | 100 - (idle/total)*100 | cpuUsage |
| `node_memory_MemTotal_bytes` / `node_memory_MemFree_bytes` | (total - free - buffers - cached)/total*100 | memoryUsage |
| `node_filesystem_size_bytes{mountpoint="/"}` / `node_filesystem_free_bytes{mountpoint="/"}` | (size - free)/size*100 | diskUsage |
| `node_load1` / `node_load5` / `node_load15` | 直接使用 | 系统负载 |
| `node_network_receive_bytes_total` / `node_network_transmit_bytes_total` | 速率计算 | 网络流量 |

### 4.3 获取流程

```
GetHostMetrics(hostID)
  → 获取 Agent 记录
  → 如果 node_exporter_url 不为空且 node_exporter_status=在线
    → 直接 HTTP GET node_exporter_url
    → 解析 Prometheus text/metrics 格式
    → 提取 CPU/内存/磁盘指标
    → 返回 HostMetrics
  → 否则（降级到原有 Prometheus 查询）
    → 原有 queryPrometheus 逻辑
```

---

## 五、API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/monitor/agent/scan-node-exporter` | 扫描主机 node_exporter |
| GET | `/api/v1/monitor/hosts/{id}/metrics-direct` | 直接从 node_exporter 获取指标 |

---

## 六、前端改动

### 6.1 Agent 列表页增加操作按钮
- 在原有"部署Agent"旁边增加"扫描NodeExporter"按钮
- 弹出选择主机对话框（同部署功能）
- 扫描完成后显示结果列表（成功/失败）

### 6.2 Agent 列表项增加状态列
- node_exporter 状态标签（未扫描/在线/离线/未安装）
- node_exporter 端点 URL 显示

---

## 七、文件变更清单

### 后端修改
| 文件 | 变更 |
|------|------|
| `api/api/monitor/model/agent.go` | 新增 NodeExporter 字段、ScanDto |
| `api/api/monitor/dao/agentDao.go` | 新增 updateNodeExporterInfo 方法 |
| `api/api/monitor/service/agent.go` | 新增 ScanNodeExporter 方法 (SSH扫描逻辑) |
| `api/api/monitor/controller/agent.go` | 新增 ScanNodeExporter 控制器 |
| `api/api/monitor/service/monitorService.go` | 新增 queryNodeExporterMetrics 直接抓取/解析逻辑 |
| `api/router/monitor/monitor.go` | 新增路由 |

### 前端修改
| 文件 | 变更 |
|------|------|
| `web/src/views/monitor/Agent.vue` | 新增扫描按钮和结果展示 |
| `web/src/api/monitor.js` | 新增 scanNodeExporter API |