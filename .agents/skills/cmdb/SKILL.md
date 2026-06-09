---
name: cmdb
description: "AutoOps CMDB 模块实现了主机管理、分组管理、数据库管理、物理机/网络设备资产管理、机房机柜管理以及类JumpServer的资产授权系统。"
---

# CMDB 模块 (cmdb)

## 概述

CMDB 模块是 AutoOps 的配置管理数据库核心，涵盖主机生命周期管理、物理基础设施管理、数据库管理和资产授权控制。

### 核心功能

- **主机管理**：SSH 主机创建、编辑、删除、云主机同步(阿里云/腾讯云/百度云)、SSH 终端连接
- **分组管理**：树形资产分组，支持多级嵌套
- **数据库管理**：数据库连接管理、SQL 执行(查询/插入/更新/删除)
- **机房管理**：IDC 机房信息管理(名称/地址/联系人/等级)
- **机柜管理**：机柜管理(关联机房/U位/功率)
- **物理机管理**：物理服务器全生命周期管理(SN/品牌/型号/机房机柜/资产状态)
- **网络设备管理**：路由器/交换机/防火墙/负载均衡设备管理
- **资产授权**：类似 JumpServer 的细粒度资产授权控制

## 数据模型

### 1. **cmdb_group 表** - 资产分组（树形）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| parent_id | uint | 父级分组ID(0=根分组) |
| name | string | 分组名称 |

关联：`CmdbHost.GroupID` → `CmdbGroup.ID`

### 2. **cmdb_host 表** - SSH 主机

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| host_name | string | 主机名称(唯一) |
| group_id | uint | 分组ID |
| private_ip | string | 私网IP |
| public_ip | string | 公网IP |
| ssh_ip | string | SSH连接IP |
| ssh_name | string | SSH用户名 |
| ssh_key_id | uint | SSH凭据ID |
| ssh_port | int | SSH端口(默认22) |
| vendor | int | 厂商:1-自建,2-阿里云,3-腾讯云 |
| status | int | 状态:1-认证成功,2-未认证,3-认证失败 |
| os / cpu / memory / disk | string | 系统/CPU/内存/磁盘信息 |

### 3. **cmdb_idc 表** - 机房

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| name | string | 机房名称(唯一) |
| short_name | string | 简称 |
| address | string | 地址 |
| contact / phone | string | 联系人/电话 |
| level | string | 等级(T1-T4) |
| status | int | 状态:1-启用,2-停用 |

### 4. **cmdb_cabinet 表** - 机柜

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| name | string | 机柜名称(唯一) |
| idc_id | uint | 所属机房ID (FK→cmdb_idc) |
| position | string | 位置描述 |
| unit_num | int | 机柜U数 |
| used_unit | int | 已用U位 |
| power_kw | float | 额定功率(KW) |

### 5. **cmdb_physical_machine 表** - 物理机

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| sn | string | 序列号(唯一) |
| host_name | string | 主机名 |
| manage_ip | string | 管理IP(BMC/iLO/iDRAC) |
| business_ip | string | 业务IP |
| brand | string | 品牌(Dell/HP/Inspur等) |
| model | string | 型号 |
| cpu / memory / disk / raid | string | CPU/内存/磁盘/RAID信息 |
| idc_id | uint | 所属机房ID (FK→cmdb_idc) |
| cabinet_id | uint | 所属机柜ID (FK→cmdb_cabinet) |
| unit_position | int | 机柜U位(起始) |
| asset_status | int | 资产状态:1-在库,2-已上架,3-维修中,4-已下架,5-报废 |
| purchase_date / warranty_date | string | 采购日期/维保到期 |

### 6. **cmdb_network_device 表** - 网络设备

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| sn | string | 序列号(唯一) |
| name | string | 设备名称 |
| device_type | int | 类型:1-路由器,2-交换机,3-防火墙,4-负载均衡,5-其他 |
| brand / model | string | 品牌/型号 |
| manage_ip | string | 管理IP |
| version | string | 固件版本 |
| port_num | int | 端口数量 |
| idc_id / cabinet_id | uint | 所属机房/机柜 |
| asset_status | int | 资产状态(同物理机) |

### 7. **cmdb_asset_permission 表** - 资产授权（JumpServer风格）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| name | string | 授权规则名称(唯一) |
| description | text | 描述 |
| user_ids | text | 授权用户ID列表(JSON数组) |
| group_ids | text | 授权用户组ID列表(JSON数组) |
| asset_types | text | 授权资产类型(JSON数组: host/physical/network/database) |
| host_group_ids | text | 授权主机分组ID列表(JSON数组) |
| physical_ids / network_ids / database_ids | text | 具体资产ID列表 |
| idc_ids | text | 授权机房ID列表(含其下所有资产) |
| permission_actions | text | 权限操作(JSON数组: connect/upload/download/delete/admin) |
| is_active | int | 是否启用 |
| date_start / date_expired | string | 有效期 |

### 8. **cmdb_sql 表** - 数据库连接管理

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| name | string | 数据库名称 |
| host / port | string / int | 连接地址 |
| db_type | string | 数据库类型(MySQL/PostgreSQL等) |
| username / password | string | 登录凭据 |

### 9. **cmdb_sql_record 表** - SQL 执行记录

记录所有 SQL 执行操作的审计日志。

## 后端实现

### 文件结构

```
api/api/cmdb/
├── model/
│   ├── cmdbGroup.go               # 资产分组模型 + 树构建
│   ├── cmdbGroupHost.go           # 分组-主机关联DTO
│   ├── cmdbHost.go                # 主机模型(DTO/Create/Update)
│   ├── cmdbIDC.go                 # 机房模型 + 查询参数
│   ├── cmdbCabinet.go             # 机柜模型 + 查询参数
│   ├── cmdbPhysicalMachine.go     # 物理机模型 + 查询参数
│   ├── cmdbNetworkDevice.go       # 网络设备模型 + 查询参数
│   ├── cmdbAssetPermission.go     # 资产授权模型 + 查询参数
│   ├── cmdbSQL.go                 # 数据库连接模型
│   └── cmdbSQLRecord.go           # SQL执行记录模型
├── dao/
│   ├── cmdbGroup.go               # 分组CRUD + 树查询
│   ├── cmdbHost.go                # 主机CRUD + 按多种条件查询 + 缓存
│   ├── cmdbHostSSH.go             # SSH连接/命令执行/文件管理
│   ├── CmdbHostcloud.go           # 云主机(阿里云/腾讯云/百度云)同步
│   ├── cmdbIDC.go                 # 机房CRUD + 分页
│   ├── cmdbCabinet.go             # 机柜CRUD + 按机房查询
│   ├── cmdbPhysicalMachine.go     # 物理机CRUD + 多条件查询 + 统计
│   ├── cmdbNetworkDevice.go       # 网络设备CRUD + 多条件查询
│   ├── cmdbAssetPermission.go     # 授权规则CRUD + 用户权限查询(含有效期)
│   ├── cmdbSQL.go                 # 数据库连接CRUD
│   └── cmdbSQLRecord.go           # SQL执行记录
├── service/
│   ├── cmdbHost.go                # 主机服务
│   ├── cmdbHostSSH.go             # SSH服务(WebSocket终端/命令/文件)
│   ├── CmdbHostcloud.go           # 云主机服务
│   ├── cmdbGroup.go               # 分组服务(含树形构建)
│   ├── cmdbAsset.go               # 机房/机柜/物理机/网络设备/资产授权服务
│   ├── cmdbSQL.go                 # 数据库服务
│   └── cmdbSQLRecord.go           # SQL执行记录服务
├── controller/
│   ├── cmdbHost.go                # 主机控制器
│   ├── cmdbHostSSH.go             # SSH控制器
│   ├── CmdbHostcloud.go           # 云主机控制器
│   ├── cmdbGroup.go               # 分组控制器
│   ├── cmdbAsset.go               # 机房/机柜/物理机/网络设备/资产授权控制器
│   ├── cmdbSQL.go                 # 数据库控制器
│   ├── cmdbSqlLog.go              # SQL操作日志控制器
│   └── cmdbSQLRecord.go           # SQL执行记录控制器

api/middleware/cmdbAssetPermissionMiddleware.go  # 资产授权中间件
api/router/cmdb/cmdb.go                          # 路由注册
api/pkg/db/migrate.go                            # AutoMigrate 注册点
```

### API 接口

#### 主机管理
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /cmdb/hostcreate | 创建主机 |
| PUT | /cmdb/hostupdate | 更新主机 |
| DELETE | /cmdb/hostdelete | 删除主机 |
| GET | /cmdb/hostlist | 主机列表(分页) |
| GET | /cmdb/hostinfo | 根据ID获取主机 |
| GET | /cmdb/hostgroup | 根据分组ID获取主机 |
| GET | /cmdb/hostbyname | 按名称模糊查询 |
| GET | /cmdb/hostbyip | 按IP查询 |
| GET | /cmdb/hostbystatus | 按状态查询 |
| POST | /cmdb/hostimport | Excel导入主机 |
| GET | /cmdb/hosttemplate | 下载导入模板 |
| POST | /cmdb/hostsync | 同步主机信息(SSH采集) |

#### 云主机
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /cmdb/hostcloudcreatealiyun | 创建阿里云主机 |
| POST | /cmdb/hostcloudcreatetencent | 创建腾讯云主机 |
| POST | /cmdb/hostcloudcreatebaidu | 创建百度云主机 |

#### SSH 终端
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /cmdb/hostssh/connect/:id | WebSocket SSH终端连接 |
| GET | /cmdb/hostssh/command/:id | 执行命令 |
| POST | /cmdb/hostssh/upload/:id | 上传文件 |
| GET | /cmdb/hostssh/files | 文件列表 |
| DELETE | /cmdb/hostssh/file | 删除文件 |
| GET | /cmdb/hostssh/download | 下载文件 |

#### 分组管理
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /cmdb/groupadd | 添加分组 |
| GET | /cmdb/grouplist | 分组树 |
| GET | /cmdb/grouplistwithhosts | 分组树+主机 |
| PUT | /cmdb/groupupdate | 更新分组 |
| DELETE | /cmdb/groupdelete | 删除分组 |
| GET | /cmdb/groupbyname | 按名称查询 |

#### 数据库/SQL
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /cmdb/sql/select | 查询SQL |
| POST | /cmdb/sql | 插入SQL |
| PUT | /cmdb/sql | 更新SQL |
| DELETE | /cmdb/sql | 删除SQL |
| POST | /cmdb/sql/execute | 执行原生SQL |
| POST | /cmdb/sql/databaselist | 获取数据库列表 |
| POST/GET | /cmdb/database | 数据库CRUD |

#### 机房管理
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /cmdb/idc | 创建机房 |
| PUT | /cmdb/idc/:id | 更新机房 |
| DELETE | /cmdb/idc/:id | 删除机房 |
| GET | /cmdb/idc/:id | 获取机房详情 |
| GET | /cmdb/idc | 分页查询机房 |
| GET | /cmdb/idc/all | 获取所有机房 |

#### 机柜管理
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /cmdb/cabinet | 创建机柜 |
| PUT | /cmdb/cabinet/:id | 更新机柜 |
| DELETE | /cmdb/cabinet/:id | 删除机柜 |
| GET | /cmdb/cabinet/:id | 获取机柜详情 |
| GET | /cmdb/cabinet | 分页查询 |
| GET | /cmdb/cabinet/idc/:idcId | 按机房查询 |

#### 物理机管理
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /cmdb/physical | 创建物理机 |
| PUT | /cmdb/physical/:id | 更新物理机 |
| DELETE | /cmdb/physical/:id | 删除物理机 |
| GET | /cmdb/physical/:id | 获取详情 |
| GET | /cmdb/physical | 分页查询(支持keyword/idcId/brand/assetStatus) |
| GET | /cmdb/physical/all | 获取所有 |
| GET | /cmdb/physical/stats | 获取统计(总量及各资产状态) |

#### 网络设备管理
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /cmdb/network | 创建网络设备 |
| PUT | /cmdb/network/:id | 更新 |
| DELETE | /cmdb/network/:id | 删除 |
| GET | /cmdb/network/:id | 获取详情 |
| GET | /cmdb/network | 分页查询 |
| GET | /cmdb/network/all | 获取所有 |

#### 资产授权管理
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /cmdb/permission | 创建授权规则 |
| PUT | /cmdb/permission/:id | 更新 |
| DELETE | /cmdb/permission/:id | 删除 |
| GET | /cmdb/permission/:id | 获取详情 |
| GET | /cmdb/permission | 分页查询 |
| GET | /cmdb/permission/my | 获取当前用户的授权资产 |

### 资产授权中间件

**文件**: `api/middleware/cmdbAssetPermissionMiddleware.go`

类似 JumpServer 的资产授权控制，提供核心权限控制函数：

1. **CheckHostPermission()**：检查用户是否有权访问指定主机
   - 管理员直接放行
   - 普通用户检查授权规则中的主机分组

2. **FilterAllowedHosts()**：过滤主机列表，只返回用户有权限的主机
   - 将允许的主机ID集合存入 Gin 上下文 `cmdb_allowed_host_ids`

3. **CheckActionPermission()**：检查是否有指定操作权限(connect/upload/download/delete/admin)
   - 检查授权规则的 `permission_actions` 字段

权限检测流程：
```
管理员 → 直接放行
普通用户 → 获取有效授权(含有效期检查) → 匹配资产类型 → 匹配资产ID/分组ID → 匹配操作权限
```

授权有效期检查在 DAO 层的 `GetUserPermissions()` 中：
```go
WHERE is_active = 1
  AND (date_start IS NULL OR date_start <= NOW())
  AND (date_expired IS NULL OR date_expired >= NOW())
```

## 前端实现

### 文件结构

```
web/src/
├── api/cmdb.js                     # 所有CMDB API封装
├── router/cmdb.js                  # CMDB模块路由
├── router/router.js                # 主路由(引入cmdbRoutes → systemRoutes)
└── views/cmdb/
    ├── cmdbHost.vue                # 主机管理(左侧分组树+右侧主机表格)
    ├── cmdbGroup.vue               # 分组管理(树形组件)
    ├── cmdbDB.vue                  # 数据库管理
    ├── DBdetails.vue               # 数据库详情操作
    ├── Host/
    │   ├── SSH.vue                 # SSH终端
    │   └── FullscreenTerminal.vue  # 全屏终端
    ├── physicalMachine.vue         # 物理机管理(统计卡片+搜索+CRUD)
    ├── networkDevice.vue           # 网络设备管理(分类筛选+CRUD)
    └── assetPermission.vue         # 资产授权管理(JumpServer风格)

web/src/views/system/
    └── Machine.vue                 # 机房/机柜管理(标签页整合)
```

### 路由配置

```javascript
// CMDB 路由 — 侧边栏「资产管理」
{ path: '/cmdb/ecs',         meta: {sTitle: '资产管理', tTitle: '主机管理'} }
{ path: '/cmdb/group',       meta: {sTitle: '资产管理', tTitle: '业务分组'} }
{ path: '/cmdb/db',          meta: {sTitle: '资产管理', tTitle: '数据管理'} }
{ path: '/cmdb/ssh',         meta: {sTitle: '资产管理', tTitle: '终端登录'} }
{ path: '/cmdb/dbdetails',   meta: {sTitle: '数据管理', tTitle: '数据库操作'} }
{ path: '/cmdb/physical',    meta: {sTitle: '资产管理', tTitle: '物理机管理'} }
{ path: '/cmdb/network',     meta: {sTitle: '资产管理', tTitle: '网络设备管理'} }
{ path: '/cmdb/permission',  meta: {sTitle: '资产管理', tTitle: '资产授权'} }

// 系统路由 — 侧边栏「基础管理」
{ path: '/system/machine',   meta: {sTitle: '基础管理', tTitle: '机房信息'} }
```

## 测试方法

### 环境准备

```bash
cd api && go run main.go -c config.yaml
```

### 测试场景

#### 场景 1：机房和机柜

```bash
# 创建机房
curl -X POST "http://localhost:8000/api/v1/cmdb/idc" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"北京亦庄数据中心","shortName":"BJ-YZ","level":"T3"}'

# 创建机柜
curl -X POST "http://localhost:8000/api/v1/cmdb/cabinet" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"A01","idcId":1,"unitNum":42}'

# 获取所有机房
curl "http://localhost:8000/api/v1/cmdb/idc/all" -H "Authorization: Bearer $TOKEN"
```

#### 场景 2：物理机

```bash
# 创建物理机
curl -X POST "http://localhost:8000/api/v1/cmdb/physical" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"sn":"SN20260530-001","hostName":"prod-db-01","manageIp":"10.0.0.1","brand":"Dell","model":"R750","idcId":1,"cabinetId":1,"assetStatus":2}'

# 统计
curl "http://localhost:8000/api/v1/cmdb/physical/stats" -H "Authorization: Bearer $TOKEN"
```

#### 场景 3：网络设备

```bash
# 创建交换机
curl -X POST "http://localhost:8000/api/v1/cmdb/network" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"sn":"SN-SW-001","name":"核心交换机-01","deviceType":2,"brand":"Cisco","model":"C9500","idcId":1}'
```

#### 场景 4：资产授权

```bash
# 创建授权规则 — 允许用户106连接物理机1
curl -X POST "http://localhost:8000/api/v1/cmdb/permission" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"基础运维-物理机","userIds":[106],"assetTypes":["physical"],"physicalIds":[1],"permissionActions":["connect"]}'

# 查看我的授权资产
curl "http://localhost:8000/api/v1/cmdb/permission/my" -H "Authorization: Bearer $TOKEN"
```

## 数据库迁移

所有 CMDB 模型通过 GORM AutoMigrate 自动建表，注册在 `api/pkg/db/migrate.go`：

```go
var models = []interface{}{
    &cmdbmodel.CmdbGroup{},
    &cmdbmodel.CmdbHost{},
    &cmdbmodel.CmdbSQLRecord{},
    &cmdbmodel.CmdbSQL{},
    &cmdbmodel.CmdbIDC{},
    &cmdbmodel.CmdbCabinet{},
    &cmdbmodel.CmdbPhysicalMachine{},
    &cmdbmodel.CmdbNetworkDevice{},
    &cmdbmodel.CmdbAssetPermission{},
    // ...
}
```

## 已知问题和注意事项

### 物理机 SN 唯一性

物理机表的 `sn` 字段有唯一索引，创建时 Service 层会检查 `CheckSNExists()`。编辑时排除自身ID避免误判。

### 资产授权 PermissionActions 默认值

`cmdb_asset_permission` 表的 `permission_actions` 是 `text` 类型，**不能**在 GORM tag 中设置 `default`（MySQL限制）。默认值在 Service 层 `Create()` 方法中设置：

```go
if len(data.PermissionActions) <= 3 {
    data.PermissionActions = `["connect"]`
}
```

### IDCIDs 列名

`CmdbAssetPermission` 模型中的 `IDCIDs` 字段需要使用 `column:idc_ids` 显式指定列名，否则 GORM 会将驼峰的 `IDC` 拆分为 `id_c`。
