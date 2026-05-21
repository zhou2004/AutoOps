---
name: monitor-agent
description: "理解 AutoOps 监控中心 Agent 管理模块的 node_exporter 扫描集成方案。涵盖模型扩展、DAO/Service/Controller 修改、监控数据获取改造等完整技术细节。适用于维护 Agent 管理、扩展新的监控数据采集方式等场景。"
---

# AutoOps Agent 管理 - Node Exporter 集成 Skill

此技能记录在 Agent 管理模块中增加 node_exporter 扫描功能的完整架构设计。

---

## 1. 设计目标

在保留现有自定义 Agent 部署功能的同时，增加 **node_exporter 扫描** 功能：
- 用户可选择目标主机，通过 SSH 扫描主机上的 node_exporter
- 验证进程存在、端口监听、HTTP 端点可访问
- 将扫描到的端点 URL 存入 `monitor_agent` 表
- **主机监控数据获取**（`GetHostMetrics`）优先直接从 node_exporter 拉取 `/metrics` 数据
- node_exporter 不可用时降级到原有的 Prometheus 查询

---

## 2. 数据库模型扩展

### 2.1 Agent 模型新增字段

文件: `api/api/monitor/model/agent.go`

| 字段 | Go 类型 | 数据库 | 说明 |
|------|---------|--------|------|
| `NodeExporterURL` | string | `node_exporter_url varchar(512)` | node_exporter 端点 URL |
| `NodeExporterStatus` | int | `node_exporter_status tinyint(1)` | 0-未扫描,1-在线,2-离线,3-未安装 |
| `NodeExporterPort` | int | `node_exporter_port int(11)` | node_exporter 端口，默认 9100 |
| `NodeExporterScanTime` | string | `node_exporter_scan_time varchar(64)` | 最近扫描时间 |

### 2.2 新增 DTO

```go
// 扫描请求
type NodeExporterScanDto struct {
    HostIDs []uint `json:"hostIds" binding:"required"`
    Port    int    `json:"port"` // 默认 9100
}

// 扫描结果
type NodeExporterResult struct {
    HostID   uint   `json:"hostId"`
    HostName string `json:"hostName"`
    SSHIP    string `json:"sshIp"`
    Port     int    `json:"port"`
    URL      string `json:"url"`     // http://{ip}:{port}/metrics
    Status   int    `json:"status"`  // 1-在线,2-离线,3-未安装
    ErrorMsg string `json:"errorMsg"`
}
```

---

## 3. 扫描流程

```
ScanNodeExporter(c, dto)
  ├── 遍历 HostIDs
  │   ├── 获取主机信息（hostDao.GetCmdbHostById）
  │   ├── 获取 SSH 认证（getSSHKeyByID）
  │   ├── 步骤1: 检查进程
  │   │   └── ps aux | grep node_exporter
  │   │       └── 未找到 → Status=3(未安装)
  │   ├── 步骤2: 检查端口
  │   │   └── ss -tunlp | grep :{port}
  │   │       └── 未监听 → Status=2(离线)
  │   ├── 步骤3: HTTP 验证
  │   │   └── curl http://127.0.0.1:{port}/metrics
  │   │       └── 失败 → Status=2(离线)
  │   ├── 成功 → Status=1(在线), URL = http://{ssh_ip}:{port}/metrics
  │   └── UpdateNodeExporterInfo(hostID, result) ← 更新数据库
  └── 返回扫描结果列表
```

---

## 4. 文件变更

### 后端修改

| 文件 | 变更内容 |
|------|---------|
| `api/api/monitor/model/agent.go` | Agent 模型增加 NodeExporterURL/Status/Port/ScanTime 字段 |
| `api/api/monitor/model/agent.go` | 新增 NodeExporterScanDto / NodeExporterResult 结构体 |
| `api/api/monitor/dao/agentDao.go` | 接口新增 UpdateNodeExporterInfo 方法 |
| `api/api/monitor/dao/agentDao.go` | 实现 UpdateNodeExporterInfo（更新 4 个 node_exporter 字段） |
| `api/api/monitor/service/agent.go` | ServiceInterface 新增 ScanNodeExporter 方法 |
| `api/api/monitor/service/agent.go` | 实现 ScanNodeExporter（SSH 扫描三步骤） |
| `api/api/monitor/controller/agent.go` | 新增 ScanNodeExporter 控制器 |
| `api/router/monitor/monitor.go` | 新增路由 `POST /agent/scan-node-exporter` |

### 前端修改

| 文件 | 变更内容 |
|------|---------|
| `web/src/api/monitor.js` | 新增 `scanNodeExporter` API 函数 |

---

## 5. API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/monitor/agent/scan-node-exporter` | 扫描主机 node_exporter |

请求体：
```json
{"hostIds": [1, 2, 3], "port": 9100}
```

响应：
```json
{
  "code": 200,
  "data": {
    "results": [
      {"hostId": 1, "hostName": "web-01", "sshIp": "192.168.1.1", "port": 9100, "url": "http://192.168.1.1:9100/metrics", "status": 1, "errorMsg": ""},
      {"hostId": 2, "hostName": "web-02", "sshIp": "192.168.1.2", "port": 9100, "url": "", "status": 3, "errorMsg": "未检测到 node_exporter 进程"}
    ],
    "total": 2
  }
}
```

状态码说明：
- `1` = 在线（进程运行 + 端口监听 + HTTP 可达）
- `2` = 离线（端口未监听或 HTTP 不可达）
- `3` = 未安装（未检测到 node_exporter 进程）

---

## 6. 与现有系统的关系

- **自定义 Agent 部署**（DeployAgent）保持不变
- **主机监控**（GetHostMetrics）后续会改造为优先读取 `node_exporter_url` 直接从 `/metrics` 拉取数据
- **扫描结果**存储在 `monitor_agent` 表的 `node_exporter_*` 字段中，Agent 列表可展示