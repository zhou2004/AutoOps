# K8s 权限管理模块实现文档

## 概述

本文档记录了 AutoOps 系统中 K8s 权限管理模块的完整实现。该模块实现了对 Kubernetes 集群资源的细粒度访问控制，支持管理员为不同用户授予特定集群下特定命名空间的访问权限。

## 需求分析

1. **K8s 权限控制**: 管理员能够进行 K8s 权限的操作，能够赋予普通用户（游客）对应权限
2. **权限操作界面**: 提供可视化的 K8s 权限管理界面
3. **API 调用权限**: 未授权的用户无法通过 API 访问对应资源
4. **系统用户权限统一结合**: 与系统用户权限体系结合，支持可扩展性

## 实现方案

### 1. 数据库设计

创建 `k8s_permission` 表，记录用户-集群-命名空间的权限映射关系：

```sql
CREATE TABLE IF NOT EXISTS `k8s_permission` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `user_id` int(10) unsigned NOT NULL COMMENT '用户ID(sys_admin.id)',
    `cluster_id` int(10) unsigned NOT NULL COMMENT '集群ID(k8s_cluster.id)',
    `namespace` varchar(255) NOT NULL COMMENT '命名空间名称',
    `permission_type` varchar(64) DEFAULT 'readonly' COMMENT '权限类型: readonly/write/admin',
    `created_at` datetime(3) NOT NULL COMMENT '创建时间',
    `updated_at` datetime(3) NOT NULL COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_user_cluster_ns` (`user_id`, `cluster_id`, `namespace`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_cluster_id` (`cluster_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='K8s权限管理表';
```

### 2. 后端实现

#### 2.1 数据模型 (model)

- `K8sPermission`: 权限记录实体
- `CreateK8sPermissionRequest`: 创建权限请求
- `UpdateK8sPermissionRequest`: 更新权限请求
- `K8sPermissionBatchCreateRequest`: 批量创建权限请求
- `K8sPermissionListResponse`: 权限列表响应
- `K8sPermissionVo`: 权限视图（包含用户名、集群名等关联信息）
- `K8sPermissionQuery`: 查询参数

#### 2.2 数据访问层 (dao)

- `Create`: 创建权限记录
- `BatchCreate`: 批量创建权限记录
- `Update`: 更新权限类型
- `Delete`: 删除权限记录
- `GetByID`: 根据ID获取权限
- `GetByUserAndCluster`: 获取用户在指定集群的所有权限
- `GetByUser`: 获取用户的所有权限
- `GetByCluster`: 获取集群的所有权限分配
- `CheckPermission`: 检查用户是否有指定集群命名空间的权限
- `IsAdmin`: 检查用户是否为管理员
- `GetUserAllowedNamespaces`: 获取用户允许访问的命名空间列表
- `GetList`: 分页查询权限列表（带关联信息）

#### 2.3 业务逻辑层 (service)

- 所有权限管理操作（创建、更新、删除、查询）均需管理员权限
- 通过 `checkAdmin` 方法验证当前用户是否为管理员

#### 2.4 权限中间件 (middleware)

`K8sPermissionMiddleware` 提供三个中间件函数：

1. **CheckNamespacePermission()**: 检查命名空间级别的权限
   - 管理员直接放行
   - 从 URL 路径提取 `clusterId` 和 `namespace`
   - 查询 `k8s_permission` 表验证权限

2. **CheckClusterPermission()**: 检查集群级别的权限
   - 管理员直接放行
   - 检查用户是否有该集群的任意命名空间权限

3. **FilterAllowedNamespaces()**: 过滤命名空间列表
   - 管理员不过滤
   - 非管理员只返回有权限的命名空间
   - 将允许的命名空间列表存入 Gin 上下文

#### 2.5 路由注册

所有 K8s 相关路由均已集成权限中间件：

- 权限管理路由（CRUD）: 仅管理员可访问
- 集群管理路由: 需要集群权限
- 命名空间管理路由: 需要命名空间权限
- 工作负载管理路由: 需要命名空间权限
- Pod 管理路由: 需要命名空间权限
- Service/Ingress 管理路由: 需要命名空间权限
- 存储管理路由: 需要命名空间权限
- 配置管理路由: 需要命名空间权限
- CRD 管理路由: 需要命名空间权限

### 3. 前端实现

#### 3.1 API 封装

在 `web/src/api/k8s.js` 中添加了权限管理相关的 API 调用：

```javascript
// 创建权限
createPermission(data)
// 批量创建权限
batchCreatePermission(data)
// 更新权限
updatePermission(id, data)
// 删除权限
deletePermission(id)
// 获取权限列表
getPermissionList(params)
// 获取用户的所有权限
getUserPermissions(userId)
// 获取集群的所有权限分配
getClusterPermissions(clusterId)
```

#### 3.2 权限管理页面

在 `web/src/views/K8s/k8s-permission.vue` 中实现了完整的权限管理界面：

- **权限列表**: 分页展示所有权限记录，支持按用户、集群、命名空间筛选
- **新增权限**: 选择用户、集群、命名空间和权限类型
- **批量授权**: 一次为某个用户在某个集群下授予多个命名空间的权限
- **编辑权限**: 修改权限类型
- **删除权限**: 删除指定权限记录

#### 3.3 路由配置

在 `web/src/router/k8s.js` 中添加了权限管理页面的路由：

```javascript
{
    path: '/k8s/permission',
    component: () => import('@/views/K8s/k8s-permission.vue'),
    meta: {sTitle: '容器管理', tTitle: '权限管理'}
}
```

### 4. 权限控制流程

```
用户请求 → AuthMiddleware(验证JWT) → K8sPermissionMiddleware(检查权限) → Controller → Service → DAO
```

1. **AuthMiddleware**: 验证用户身份，从 JWT 中提取用户信息
2. **K8sPermissionMiddleware**: 
   - 检查用户是否为管理员（拥有所有权限）
   - 非管理员用户检查 `k8s_permission` 表
   - 根据 URL 路径中的 `clusterId` 和 `namespace` 进行权限校验
3. **Controller/Service/DAO**: 执行业务逻辑

### 5. 可扩展性设计

权限系统设计为可扩展的，后续其他模块可以复用类似的权限控制模式：

1. **权限表设计**: 当前 `k8s_permission` 表专注于 K8s 权限，后续可添加 `module` 字段扩展为通用权限表
2. **中间件模式**: 权限中间件可以独立应用于任何路由组
3. **接口抽象**: `IK8sPermissionService` 接口定义了完整的 CRUD 操作，其他模块可实现类似接口

## 文件清单

### 后端文件

| 文件 | 说明 |
|------|------|
| api/api/k8s/model/k8sPermission.go | 数据模型 |
| api/api/k8s/dao/k8sPermission.go | 数据访问层 |
| api/api/k8s/service/k8sPermission.go | 业务逻辑层 |
| api/api/k8s/controller/k8sPermission.go | HTTP控制器 |
| api/middleware/k8sPermissionMiddleware.go | 权限中间件 |
| api/router/k8s/k8s.go | 路由注册 |
| api/sql/update.sql | 数据库迁移脚本 |

### 前端文件

| 文件 | 说明 |
|------|------|
| web/src/api/k8s.js | API 调用封装 |
| web/src/router/k8s.js | 路由配置 |
| web/src/views/K8s/k8s-permission.vue | 权限管理页面 |

### 文档文件

| 文件 | 说明 |
|------|------|
| docs/2026-05-21-k8s-permission-management.md | 本文档 |
| .agents/skills/k8s-permission/SKILL.md | Skill 文档 |

## 使用说明

### 管理员操作

1. 登录系统（需要管理员账号）
2. 进入「容器管理 → 权限管理」页面
3. 点击「新增权限」或「批量授权」按钮
4. 选择用户、集群、命名空间和权限类型
5. 提交保存

### 普通用户访问

1. 普通用户只能看到被授权的集群和命名空间
2. 未授权的 API 调用会返回 403 错误
3. 命名空间列表会自动过滤，只显示有权限的命名空间

## 迭代记录

### 2026-05-27：Bugfix — RBAC-only 用户无法访问集群级接口

**问题**: 仅有命名空间级别 RBAC 绑定（如 `namespace="fmusic"`）的用户在访问 `/pvs`、`/nodes`、`/storageclasses`、`/crds` 等集群级接口时返回 403。

**根因**: `hasAnyClusterPermission()` 第3步使用 `hasRbacVerbPermission(userID, clusterID, "", verbs)` 只匹配空 namespace 的集群级规则键 `"36:"`，但命名空间级绑定生成的是 `"36:fmusic"`，导致不匹配。

**修复**: 改为遍历 `getUserRbacBindings()` 直接检查 clusterID 匹配，不再要求 namespace 必须为空。

**文件**: `api/middleware/k8sPermissionMiddleware.go`

### 2026-05-27：Bugfix — 命名空间列表对 RBAC-only 用户返回空

**问题**: RBAC-only 用户访问 `GET /namespaces` 返回空数组。

**根因**: 命名空间过滤逻辑未处理 `allowedSet` 中的通配符 `*` 和 `""`。

**修复**: 增加 `hasAll := allowedSet["*"] || allowedSet[""]` 判断。

**文件**: `api/api/k8s/service/k8snamespace.go`

---

## 权限类型说明

| 类型 | 说明 | 操作范围 |
|------|------|----------|
| readonly | 只读 | 查看资源列表和详情 |
| write | 读写 | 查看、创建、更新、删除资源 |
| admin | 管理员 | 所有操作权限 |
