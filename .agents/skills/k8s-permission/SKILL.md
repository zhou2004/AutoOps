# K8s 权限管理模块 (k8s-permission)

## 概述

K8s 权限管理模块实现了对 Kubernetes 集群资源的细粒度访问控制。通过该系统，管理员可以为不同用户授予特定集群下特定命名空间的访问权限，实现多租户隔离和权限管控。

## 核心架构

### 数据模型

**k8s_permission 表** - 权限记录表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| user_id | uint | 用户ID (sys_admin.id) |
| cluster_id | uint | 集群ID (k8s_cluster.id) |
| namespace | string | 命名空间名称 |
| permission_type | string | 权限类型: readonly/write/admin |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |

唯一索引: `(user_id, cluster_id, namespace)` - 确保同一用户在同一集群的同一命名空间只有一条权限记录。

### 权限类型

- **readonly** (只读): 只能查看资源，不能创建、修改或删除
- **write** (读写): 可以查看和修改资源
- **admin** (管理员): 拥有该命名空间的所有操作权限

### 系统角色

- **管理员**: 通过 `sys_admin_role` 表关联 `role_key = 'admin'` 的角色判断。管理员拥有所有 K8s 资源的访问权限，不受命名空间权限限制。
- **普通用户**: 只能访问被明确授权的集群和命名空间。

## 后端实现

### 文件结构

```
api/api/k8s/
├── model/k8sPermission.go          # 数据模型和请求/响应结构体
├── dao/k8sPermission.go            # 数据访问层
├── service/k8sPermission.go        # 业务逻辑层
└── controller/k8sPermission.go     # HTTP控制器层

api/middleware/k8sPermissionMiddleware.go  # 权限中间件

api/router/k8s/k8s.go              # 路由注册
```

### API 接口

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/k8s/permission | 创建权限 | 管理员 |
| POST | /api/v1/k8s/permission/batch | 批量创建权限 | 管理员 |
| PUT | /api/v1/k8s/permission/:id | 更新权限类型 | 管理员 |
| DELETE | /api/v1/k8s/permission/:id | 删除权限 | 管理员 |
| GET | /api/v1/k8s/permission | 分页查询权限列表 | 管理员 |
| GET | /api/v1/k8s/permission/user/:userId | 获取用户的所有权限 | 管理员 |
| GET | /api/v1/k8s/permission/cluster/:clusterId | 获取集群的所有权限分配 | 管理员 |

### 权限中间件

`K8sPermissionMiddleware` 提供三个中间件函数：

1. **CheckNamespacePermission()**: 检查用户是否有权限访问指定集群的命名空间
   - 从 URL 路径中提取 `clusterId` 和 `namespace`
   - 管理员直接放行
   - 普通用户检查 `k8s_permission` 表

2. **CheckClusterPermission()**: 检查用户是否有权限访问指定集群
   - 从 URL 路径中提取 `clusterId`
   - 管理员直接放行
   - 普通用户检查是否有该集群的任意命名空间权限

3. **FilterAllowedNamespaces()**: 过滤命名空间列表
   - 对于非管理员用户，只返回其有权限的命名空间
   - 将允许的命名空间列表存入 Gin 上下文

### 路由注册示例

```go
// 权限管理路由 - 仅管理员可操作
router.POST("/k8s/permission", middleware.AuthMiddleware(), k8sPermCtrl.Create)
router.GET("/k8s/permission", middleware.AuthMiddleware(), k8sPermCtrl.GetList)

// 命名空间路由 - 需要命名空间权限
router.GET("/k8s/cluster/:id/namespaces", 
    middleware.AuthMiddleware(), 
    k8sPermMiddleware.FilterAllowedNamespaces(), 
    k8sNamespaceCtrl.GetNamespaces)

// 工作负载路由 - 需要命名空间权限
router.GET("/k8s/cluster/:id/namespaces/:namespaceName/workloads", 
    middleware.AuthMiddleware(), 
    k8sPermMiddleware.CheckNamespacePermission(), 
    k8sWorkloadCtrl.GetWorkloads)
```

## 前端实现

### 文件结构

```
web/src/
├── api/k8s.js                     # API 调用封装
├── router/k8s.js                  # 路由配置
└── views/K8s/k8s-permission.vue   # 权限管理页面
```

### 页面功能

- **权限列表**: 分页展示所有权限记录，支持按用户、集群、命名空间筛选
- **新增权限**: 选择用户、集群、命名空间和权限类型
- **批量授权**: 一次为某个用户在某个集群下授予多个命名空间的权限
- **编辑权限**: 修改权限类型
- **删除权限**: 删除指定权限记录

### API 调用

```javascript
import k8sApi from '@/api/k8s'

// 获取权限列表
k8sApi.getPermissionList({ userId, clusterId, namespace, page, size })

// 创建权限
k8sApi.createPermission({ userId, clusterId, namespace, permissionType })

// 批量创建权限
k8sApi.batchCreatePermission({ userId, clusterId, namespaces: [], permissionType })

// 更新权限
k8sApi.updatePermission(id, { permissionType })

// 删除权限
k8sApi.deletePermission(id)
```

## 可扩展性设计

### 模块化权限接口

权限系统设计为可扩展的，后续其他模块可以复用类似的权限控制模式：

1. **权限表设计**: 使用 `module` 字段可以扩展为通用权限表
2. **中间件模式**: 权限中间件可以独立应用于任何路由组
3. **接口抽象**: `IK8sPermissionService` 接口定义了完整的 CRUD 操作

### 扩展其他模块

如需为其他模块（如 CMDB、监控等）添加权限控制，可参考以下步骤：

1. 在 `k8s_permission` 表中添加 `module` 字段，或创建新的权限表
2. 实现对应的 DAO/Service/Controller
3. 创建对应的权限中间件
4. 在路由中注册中间件

## 数据库迁移

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

## 权限控制流程

```
用户请求 → AuthMiddleware(验证JWT) → K8sPermissionMiddleware(检查权限) → Controller → Service → DAO
```

1. **AuthMiddleware**: 验证用户身份，从 JWT 中提取用户信息
2. **K8sPermissionMiddleware**: 
   - 检查用户是否为管理员（拥有所有权限）
   - 非管理员用户检查 `k8s_permission` 表
   - 根据 URL 路径中的 `clusterId` 和 `namespace` 进行权限校验
3. **Controller/Service/DAO**: 执行业务逻辑

## 常见问题

### Q: 为什么管理员不需要权限配置？
A: 管理员通过 `sys_admin_role` 表关联 `role_key = 'admin'` 的角色，在中间件中会直接放行。

### Q: 如何给用户授予多个命名空间的权限？
A: 使用批量授权功能，一次选择多个命名空间，或多次调用单个授权接口。

### Q: 权限控制是否影响 API 调用？
A: 是的，权限中间件对所有 K8s 相关 API 进行拦截，未授权的 API 调用会返回 403 错误。
