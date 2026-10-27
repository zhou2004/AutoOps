---
name: cmdb-permission
description: "AutoOps CMDB 资产授权模块实现了类JumpServer的细粒度资产授权系统，支持用户组管理、主机/物理机/网络设备精细授权和RBAC风格权限操作。"
---

# CMDB 资产授权模块 (cmdb-permission)

## 2026-10-24 授权整合

- 唯一管理入口：`/system/authorization`；`tab=assets` 为资产授权，`tab=containers` 为容器授权，`tab=groups` 为统一用户组，`groupType=assets|containers` 区分资产用户组和容器用户组。旧地址仅作隐藏重定向。
- Docker 页面位于 `/k8s/docker`，API 保持 `/cmdb/docker/...`；目标发现 `/cmdb/docker/targets` 和连接列表 `/cmdb/docker/assets/:assetType/:assetId/connections` 使用 Docker 模块及资产权限，不要求额外 CMDB 列表权限。
- `credentialPermission.vue` 及 `/cmdb/permission/credential*` 路由已移除。历史模型、DAO 和记录仅兼容保留，不参与运行时授权；不得再将 `CanUseCredential*` 接回连接、Docker、网络配置、数据库或任务执行。
- SSH/VNC/RDP 连接档案的增删改、凭据枚举、主机 SSH 新绑定和数据库账号新绑定仅活动管理员可配置；已有连接的使用依赖目标资产对应动作。资产有效期、用户状态和会话期间撤权检查继续生效。
- 数据库菜单由 `api/pkg/db/navigation_migrate.go` 在启动时幂等迁移，不删除历史规则或角色关系。

## 概述

CMDB 资产授权模块是 AutoOps 的类 JumpServer 授权控制系统，支持**用户组管理**、**细粒度资产授权**和**RBAC风格权限操作**。

### 核心功能

- **用户组管理**：CMDB 用户组（区别于 K8s 用户组），支持创建/编辑/删除用户组、管理组成员
- **凭据绑定**：管理员配置资产连接及凭据；使用连接按目标资产授权，不再单独授予凭据使用权。
- **细粒度资产授权**：支持按主机分组、业务线、具体主机、物理机、网络设备授权
- **RBAC 风格权限**：`get/list/connect/create/update/delete/admin` 等细粒度权限操作
- **强制权限校验**：中间件自动拦截非管理员用户，仅返回授权资产

## 数据模型

### 1. **cmdb_user_group 表** - CMDB 用户组

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| name | string | 用户组名称(唯一) |
| code | string | 用户组编码 |
| description | string | 描述 |
| status | int | 状态:1-启用,0-禁用 |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |

### 2. **cmdb_user_group_member 表** - CMDB 用户组成员 (多对多)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| group_id | uint | 用户组ID |
| user_id | uint | 用户ID |
| created_at | datetime | 创建时间 |

唯一索引: `(group_id, user_id)`

### 3. **cmdb_credential_permission 表** - 凭据授权

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| name | string | 授权规则名称 |
| credential_id | uint | 凭据ID (→config_ecsauth.id) |
| user_ids | text | 授权用户ID列表(JSON数组) |
| group_ids | text | 授权用户组ID列表(JSON数组) |
| is_active | int | 是否启用 |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |

### 4. **cmdb_asset_permission 表** (增强版) - 资产授权 (JumpServer风格)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| name | string | 授权规则名称(唯一) |
| description | text | 描述 |
| user_ids | text | 授权用户ID列表(JSON数组) |
| group_ids | text | 授权用户组ID列表(JSON数组) |
| asset_types | text | 授权资产类型(JSON数组) |
| host_group_ids | text | 授权主机分组ID列表 |
| host_ids | text | 授权具体主机ID列表 |
| physical_ids | text | 授权具体物理机ID列表 |
| network_ids | text | 授权具体网络设备ID列表 |
| database_ids | text | 授权数据库ID列表 |
| idc_ids | text | 授权机房ID列表 |
| permission_actions | text | 权限操作(JSON数组) |
| is_active | int | 是否启用 |
| date_start / date_expired | string | 有效期 |

### 5. **权限操作 (Permission Actions)**

RBAC 风格细粒度权限:

| 权限代码 | 说明 | 适用范围 |
|---------|------|---------|
| `get` | 查看详情 | 所有资产 |
| `list` | 查看列表 | 所有资产 |
| `connect` | SSH连接 | 主机 |
| `create` | 创建资产 | 所有资产 |
| `update` | 修改资产 | 所有资产 |
| `delete` | 删除资产 | 所有资产 |
| `admin` | 管理(所有操作) | 所有资产 |

## 后端实现

### 文件结构

```
api/api/cmdb/
├── model/
│   ├── cmdbUserGroup.go           # CMDB 用户组模型(新增)
│   ├── cmdbCredentialPermission.go # 凭据授权模型(新增)
│   ├── cmdbAssetPermission.go     # 资产授权模型(增强: 增加 host_ids 等字段)
│   └── ...
├── dao/
│   ├── cmdbUserGroup.go           # 用户组 DAO(新增)
│   ├── cmdbCredentialPermission.go # 凭据授权 DAO(新增)
│   └── ...
├── service/
│   ├── cmdbUserGroup.go           # 用户组 Service(新增)
│   ├── cmdbCredentialPermission.go # 凭据授权 Service(新增)
│   ├── cmdbAsset.go               # 资产授权 Service(增强: 支持新权限字段)
│   └── ...
├── controller/
│   ├── cmdbUserGroup.go           # 用户组 Controller(新增)
│   ├── cmdbCredentialPermission.go # 凭据授权 Controller(新增)
│   └── cmdbAsset.go               # 资产授权 Controller(增强)

api/middleware/cmdbAssetPermissionMiddleware.go  # 资产授权中间件(增强强制校验)
api/router/cmdb/cmdb.go                          # 路由注册(新增路由)
api/pkg/db/migrate.go                            # AutoMigrate 注册点
```

### API 接口

#### 用户组管理
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /cmdb/permission/user-group | 创建用户组 |
| PUT | /cmdb/permission/user-group/:id | 更新用户组 |
| DELETE | /cmdb/permission/user-group/:id | 删除用户组 |
| GET | /cmdb/permission/user-group | 获取用户组列表(分页) |
| GET | /cmdb/permission/user-group/:id/members | 获取组成员 |
| POST | /cmdb/permission/user-group/members | 添加组成员 |
| DELETE | /cmdb/permission/user-group/member | 移除组成员 |

#### 独立凭据授权（已移除）
原 `/cmdb/permission/credential*` 接口不再注册；历史表保留，不影响资产使用权限。

#### 资产授权管理 (增强)
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /cmdb/permission | 创建授权规则 |
| PUT | /cmdb/permission/:id | 更新授权规则 |
| DELETE | /cmdb/permission/:id | 删除授权规则 |
| GET | /cmdb/permission/:id | 获取授权规则 |
| GET | /cmdb/permission | 授权规则列表 |
| GET | /cmdb/permission/my | 我的授权资产 |
| GET | /cmdb/permission/check/:assetType/:assetId | 检查资产权限 |

### 权限校验链路

```
请求 → AuthMiddleware (JWT认证)
  ↓
请求 → CmdbAssetPermissionMiddleware
  ├─ 数据路由 (host/physical/network/idc/cabinet/group/database)
  │   ├─ RequireCmdbPermission(assetType)
  │   │   ├─ 管理员? → 通过
  │   │   ├─ 有该资产类型的授权?
  │   │   │   ├─ 有 → 继续检查操作权限
  │   │   │   └─ 无 → 403 "您没有XX的访问权限"
  │   │   ├─ 有对应操作权限? (GET→list, POST→create, PUT→update, DELETE→delete)
  │   │   │   ├─ 有 (或admin) → 通过
  │   │   │   └─ 无 → 403 "您没有XX操作权限"
  │
  ├─ SSH连接路由 (hostssh/connect/:id等)
  │   ├─ CheckHostPermission()
  │   │   ├─ 管理员? → 通过
  │   │   └─ 普通用户: 有该主机的connect权限? → 通过 / 无权限 → 403
  │
  └─ 管理路由 (permission/user-group/credential CRUD)
      ├─ AdminOnly()
      │   ├─ 管理员? → 通过
      │   └─ 非管理员 → 403
```

### HTTP Method → Permission Action 映射

| HTTP Method | 所需权限 | 说明 |
|-------------|---------|------|
| GET | `list` | 查看列表/详情（两者都允许） |
| POST | `create` | 创建资产 |
| PUT | `update` | 修改资产 |
| DELETE | `delete` | 删除资产 |

## 前端实现

### 文件结构

```
web/src/
├── api/cmdb.js                   # CMDB API(新增用户组/凭据/我的资产等接口)
├── views/cmdb/
│   ├── assetPermission.vue       # 资产授权页面(管理员, 勾选/筛选)
│   ├── cmdbUserGroup.vue         # 用户组管理页面(管理员)
│   └── myAssetPermissions.vue    # 我的授权资产页面(所有用户)
├── router/cmdb.js                # 路由(含所有页面路由)
```

### 前端页面路由

| 路径 | 页面 | 访问权限 |
|------|------|---------|
| /system/authorization?tab=assets | 资产授权管理 | 管理员 |
| /system/authorization?tab=groups | 资产用户组管理 | 管理员 |
| /system/authorization?tab=containers | 容器授权管理 | 管理员 |
| /cmdb/my-assets | 我的授权资产 | 所有用户 |

### 资产授权页面功能

1. **资产类型选择**: 主机/物理机/网络设备/数据库
2. **主机授权**: 
   - 按业务线(分组树)勾选全部主机
   - 搜索主机名/IP进行模糊搜索
   - 过滤条件: 分组、IP、状态
   - 表格勾选具体主机
3. **网络设备授权**:
   - 按机房/机柜筛选
   - 搜索设备名称
   - 过滤条件: 设备类型(路由器/交换机/防火墙等)
   - 表格勾选
4. **物理机授权**:
   - 按机房/机柜筛选
   - 搜索主机名/SN
   - 过滤条件: 品牌、资产状态
   - 表格勾选
5. **权限操作**: RBAC风格(get/list/connect/create/update/delete/admin)
