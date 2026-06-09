---
name: k8s-permission
description: "K8s 权限管理模块实现了对 Kubernetes 集群资源的细粒度访问控制，支持传统三态(readonly/write/admin)和 RBAC verbs(get/list/watch/create/update/delete) 双模型。"
---

# K8s 权限管理模块 (k8s-permission)

## 概述

K8s 权限管理模块实现了对 Kubernetes 集群资源的细粒度访问控制，支持 **双权限模型**：

### 1. 传统权限模型（向后兼容）
采用 **用户 -> 用户组 -> 权限** 模型，分三档：**readonly** / **write** / **admin**。

### 2. RBAC 精细权限模型（推荐）
采用 K8s 风格 RBAC：**Role / ClusterRole → RoleBinding / ClusterRoleBinding → User / Group**  
权限以 **verbs** 为单位：`get`, `list`, `watch`, `create`, `update`, `delete`, `patch`, `*`

两者可共存，中间件按 `admin > RBAC > 直接授权 > 用户组继承` 优先级取最高级别判定。

## 核心架构

### 数据模型

#### 传统模型

##### 1. **k8s_permission 表** - 用户直接授权表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| user_id | uint | 用户ID (sys_admin.id) |
| cluster_id | uint | 集群ID (k8s_cluster.id) |
| namespace | string | 命名空间名称 |
| permission_type | string | 权限类型: readonly/write/admin |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |

唯一索引: `(user_id, cluster_id, namespace)`

#### 2. **k8s_user_group 表** - 用户组表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| name | string | 用户组名称(唯一) |
| code | string | 用户组编码 |
| description | string | 描述 |
| status | int | 状态:1-启用,0-禁用 |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |

#### 3. **k8s_user_group_member 表** - 用户组成员关系（多对多）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| group_id | uint | 用户组ID |
| user_id | uint | 用户ID |

唯一索引: `(group_id, user_id)` - 用户组与用户多对多

#### 4. **k8s_group_permission 表** - 用户组权限表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| group_id | uint | 用户组ID |
| cluster_id | uint | 集群ID |
| namespace | string | 命名空间名称 |
| permission_type | string | 权限类型: readonly/write/admin |

唯一索引: `(group_id, cluster_id, namespace)`

### 权限继承链路

```
管理员 → 所有权限

普通用户 → [RBAC 角色绑定(k8s_rbac_binding→k8s_rbac_role)] 
         + [直接权限(k8s_permission)] 
         + [用户组权限(k8s_group_permission)]
```

### 权限类型（传统）

- **readonly** (只读): 只能查看资源，不能创建、修改或删除
- **write** (读写): 可以查看和修改资源
- **admin** (管理员): 拥有该命名空间的所有操作权限

### RBAC 模型（精细权限）

#### k8s_rbac_role 表

存储角色定义，rules 为 JSON 数组，每个 rule 包含 apiGroups / resources / verbs。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| cluster_id | uint | 集群ID (k8s_cluster.id) |
| namespace | string | 作用命名空间(""表示集群级别) |
| name | string | 角色名称 |
| rules | json | 规则数组：[{ apiGroups, resources, verbs }] |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |

#### k8s_rbac_binding 表

将角色绑定到用户或用户组。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| cluster_id | uint | 集群ID |
| namespace | string | 命名空间 |
| role_id | uint | 角色ID |
| subject_type | string | User 或 Group |
| subject_id | uint | 用户ID或用户组ID |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |

#### 权限推导规则

verbs 与权限级别的映射（用于与传统模型兼容）：

| verbs | 推导级别 |
|-------|---------|
| `*` | admin |
| `create/update/delete/patch` 任一 | write |
| 仅 `get/list/watch` | readonly |
| 无任何 verbs | 无权限 |

### 系统角色

- **管理员**: 通过 `sys_admin_role` 表关联 `role_key = 'admin'` 的角色判断。管理员拥有所有 K8s 资源的访问权限，不受命名空间权限限制。
- **普通用户**: 通过直接授权和用户组继承获得权限。

## 后端实现

### 文件结构

```
api/api/k8s/
├── model/
│   ├── k8sPermission.go          # 用户权限 + 用户组权限 数据模型
│   ├── k8sUserGroup.go           # 用户组数据模型
│   └── k8sRbac.go                # RBAC 角色/绑定数据模型
├── dao/
│   ├── k8sPermission.go          # 用户权限 + 用户组权限 DAO
│   ├── k8sUserGroup.go           # 用户组 DAO
│   └── k8sRbacDao.go             # RBAC DAO
├── service/
│   ├── k8sPermission.go          # 用户权限服务（含 GetMyPermissions 合并三源）
│   ├── k8sUserGroup.go           # 用户组服务
│   └── k8sRbacService.go         # RBAC 角色/绑定服务
└── controller/
    ├── k8sPermission.go          # 用户权限控制器
    ├── k8sUserGroup.go           # 用户组控制器
    └── k8sRbacController.go      # RBAC 控制器

api/middleware/k8sPermissionMiddleware.go  # 权限中间件（含 RBAC 整合）
api/router/k8s/k8s.go              # 路由注册
api/sql/rbac_update.sql            # RBAC 表 DDL
api/pkg/db/migrate.go              # AutoMigrate 注册点
```

### API 接口

#### 用户权限 API（传统模型）
| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/k8s/permission | 创建权限 | 管理员 |
| POST | /api/v1/k8s/permission/batch | 批量创建权限 | 管理员 |
| PUT | /api/v1/k8s/permission/:id | 更新权限类型 | 管理员 |
| DELETE | /api/v1/k8s/permission/:id | 删除权限 | 管理员 |
| GET | /api/v1/k8s/permission | 分页查询权限列表 | 管理员 |
| GET | /api/v1/k8s/permission/user/:userId | 获取用户的所有权限 | 管理员 |
| GET | /api/v1/k8s/permission/cluster/:clusterId | 获取集群的所有权限分配 | 管理员 |
| GET | /api/v1/k8s/permission/my | 获取当前用户权限(三源合并) | 登录用户 |

#### RBAC 角色管理 API
| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/k8s/rbac/role | 创建角色 | 管理员 |
| PUT | /api/v1/k8s/rbac/role/:id | 更新角色 | 管理员 |
| DELETE | /api/v1/k8s/rbac/role/:id | 删除角色 | 管理员 |
| GET | /api/v1/k8s/rbac/role | 获取角色列表 | 管理员 |

#### RBAC 绑定管理 API
| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/k8s/rbac/binding | 创建绑定时 | 管理员 |
| DELETE | /api/v1/k8s/rbac/binding/:id | 删除绑定时 | 管理员 |
| GET | /api/v1/k8s/rbac/binding | 获取绑定列表 | 管理员 |
| GET | /api/v1/k8s/rbac/my-permissions | 获取我的RBAC权限 | 登录用户 |

#### 用户组管理 API
| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/k8s/user-group | 创建用户组 | 管理员 |
| PUT | /api/v1/k8s/user-group/:id | 更新用户组 | 管理员 |
| DELETE | /api/v1/k8s/user-group/:id | 删除用户组 | 管理员 |
| GET | /api/v1/k8s/user-group | 分页查询用户组列表 | 管理员 |
| GET | /api/v1/k8s/user-group/:id/members | 获取组成员列表 | 管理员 |
| POST | /api/v1/k8s/user-group/members | 批量添加组成员 | 管理员 |
| DELETE | /api/v1/k8s/user-group/member | 移除组成员 | 管理员 |
| GET | /api/v1/k8s/user-group/user/:userId | 获取用户所属用户组 | 管理员 |

### 权限中间件

`K8sPermissionMiddleware` 提供核心权限控制函数：

1. **CheckNamespacePermission()**: 检查用户是否有权限访问指定集群的命名空间
   - 从 URL 路径中提取 `clusterId` 和 `namespace`
   - 管理员直接放行
   - 普通用户检查：`k8s_permission` 表 + 用户组继承权限 + RBAC规则

2. **CheckNamespaceWritePermission()**: 检查命名空间写权限
   - 管理员直接放行
   - 检查用户通过 `getUserNamespacePermissionLevel()` 获取最高权限级别
   - 同时检查 RBAC verbs（如果有 `create/update/delete/patch` 任一，则判定为 write）

3. **CheckClusterPermission()**: 检查用户是否有权限访问指定集群
   - 从 URL 路径中提取 `clusterId`
   - 管理员直接放行
   - 普通用户检查是否有该集群的任意命名空间权限（含 RBAC binding）

4. **FilterAllowedNamespaces()**: 过滤命名空间列表
   - 对于非管理员用户，只返回其有权限的命名空间
   - 合并 `k8s_permission` + 用户组继承 + RBAC binding 中出现的所有命名空间
   - 将允许的命名空间列表存入 Gin 上下文 `k8s_allowed_namespaces`

5. **FilterAllowedClusters()**: 过滤集群列表
   - 对于非管理员用户，只返回其有权限的集群
   - 将允许的集群 ID 列表存入 Gin 上下文

### RBAC 中间件关键函数

```go
// 获取用户所有RBAC规则（合并 direct + group binding）
func (m *K8sPermissionMiddleware) getUserRbacRules(userID uint) map[string][]model.K8sRule

// verbs 级别检查
func (m *K8sPermissionMiddleware) hasRbacVerbPermission(userID, clusterID uint, namespace string, verbs []string) bool

// 从 RBAC rules 推导最大级别
func (m *K8sPermissionMiddleware) getRbacMaxLevel(userID, clusterID uint, namespace string) string
```

### 权限检查优先级

```
1. isAdmin → 直接放行
2. 传统模型：getUserNamespacePermissionLevel()
   - 直接授权 (k8s_permission)
   - 用户组继承 (k8s_group_permission)
3. RBAC verbs → 映射为级别 (getRbacMaxLevel)
4. 取所有结果中的最高级别
```

### 路由注册示例

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
├── api/k8s.js                     # API 调用封装（含 RBAC API）
├── router/k8s.js                  # 路由配置
├── composables/useK8sPermission.js # 权限组合式函数（单例全局共享）
└── views/K8s/
    ├── k8s-permission.vue         # 传统权限管理页面
    └── k8s-rbac/                  # RBAC 管理页面（待实现）
```

### 权限组合式函数 `useK8sPermission.js`

全局单例，只加载一次。提供：

```javascript
const {
  permissions,          // 响应式权限数组
  loaded,               // 是否已加载
  loading,              // 是否加载中
  loadPermissions(),    // 加载权限（自动去重）
  hasPermission(clusterId, namespace),       // 是否有任意权限
  hasWritePermission(clusterId, namespace),  // 是否有写权限
  hasAdminPermission(clusterId, namespace),  // 是否有管理员权限
  hasClusterPermission(clusterId),           // 是否有集群权限
  hasVerbPermission(clusterId, namespace, verb), // RBAC verbs 级别检查
  getPermissionLevelName(clusterId, namespace),  // 权限名称
  getPermissionTagType(clusterId, namespace),    // El-Tag 类型
  refreshPermissions()                          // 强制刷新
} = useK8sPermission()
```

### 页面使用模式

每个 K8s 页面在 `onMounted` 时调用 `loadPermissions()`：
```javascript
const { loadPermissions, hasWritePermission, hasPermission } = useK8sPermission()

onMounted(async () => {
  await loadPermissions()
})

// 模板中使用
const checkWritePermission = () => {
  if (!clusterId || !namespace) return false
  return hasWritePermission(Number(clusterId), namespace)
}

const checkReadPermission = () => {
  if (!clusterId || !namespace) return false
  return hasPermission(Number(clusterId), namespace)
}
```

### 按钮禁用模式

模板中所有写操作按钮使用 `:disabled` 绑定：
```html
<el-button :disabled="!checkWritePermission()" @click="handleCreate">
  创建
</el-button>
<el-button :disabled="!checkWritePermission()" @click="handleEdit(row)">
  编辑
</el-button>
<el-button :disabled="!checkWritePermission()" @click="handleDelete(row)">
  删除
</el-button>
```

### API 调用

```javascript
import k8sApi from '@/api/k8s'

// 传统权限
k8sApi.getPermissionList({ userId, clusterId, namespace, page, size })
k8sApi.createPermission({ userId, clusterId, namespace, permissionType })
k8sApi.batchCreatePermission({ userId, clusterId, namespaces: [], permissionType })
k8sApi.updatePermission(id, { permissionType })
k8sApi.deletePermission(id)
k8sApi.getMyPermissions()                      // GET /k8s/permission/my

// RBAC 角色
k8sApi.createRbacRole(data)                    // POST /k8s/rbac/role
k8sApi.updateRbacRole(id, data)                // PUT /k8s/rbac/role/:id
k8sApi.deleteRbacRole(id)                      // DELETE /k8s/rbac/role/:id
k8sApi.getRbacRoleList(params)                 // GET /k8s/rbac/role

// RBAC 绑定
k8sApi.createRbacBinding(data)                 // POST /k8s/rbac/binding
k8sApi.deleteRbacBinding(id)                   // DELETE /k8s/rbac/binding/:id
k8sApi.getRbacBindingList(params)              // GET /k8s/rbac/binding
k8sApi.getMyRbacPermissions()                  // GET /k8s/rbac/my-permissions
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

### 传统权限表

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

### RBAC 表（新建表）

```sql
CREATE TABLE IF NOT EXISTS `k8s_rbac_role` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `cluster_id` int(10) unsigned NOT NULL,
    `namespace` varchar(255) DEFAULT '',
    `name` varchar(255) NOT NULL,
    `rules` json NOT NULL,
    `created_at` datetime(3) NOT NULL,
    `updated_at` datetime(3) NOT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `k8s_rbac_binding` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `cluster_id` int(10) unsigned NOT NULL,
    `namespace` varchar(255) DEFAULT '',
    `role_id` int(10) unsigned NOT NULL,
    `subject_type` varchar(32) NOT NULL,
    `subject_id` int(10) unsigned NOT NULL,
    `created_at` datetime(3) NOT NULL,
    `updated_at` datetime(3) NOT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

## 测试方法

### 准备工作

1. 确保后端 API 服务运行中：
   ```bash
   cd api && go run main.go -c config.yaml
   ```
2. 获取测试 JWT Token（通过登录或直接生成）:
   ```go
   // 用项目已有方法生成
   token, _ := jwtpkg.GenerateTokenByAdmin(model.SysAdmin{ID: 106, Username: "test", Nickname: "test"})
   ```
3. 将 token 存入 Redis（否则 auth 中间件会拒绝）：
   ```bash
   redis-cli SET "token:$TOKEN" "test" EX 86400
   ```

### 环境准备：插入测试数据

```sql
-- 1. 创建 RBAC 角色：fmusic 命名空间只读
INSERT INTO k8s_rbac_role (cluster_id, namespace, name, rules, created_at, updated_at) 
VALUES (36, 'fmusic', 'fmusic-viewer', 
        '[{"apiGroups":[""],"resources":["*"],"verbs":["get","list","watch"]}]', 
        NOW(), NOW());

-- 2. 绑定给 test 用户
INSERT INTO k8s_rbac_binding (cluster_id, namespace, role_id, subject_type, subject_id, created_at, updated_at)
SELECT 36, 'fmusic', id, 'User', 106, NOW(), NOW() 
FROM k8s_rbac_role WHERE name = 'fmusic-viewer' LIMIT 1;

-- 3. 清理旧权限（确保仅通过 RBAC 验证）
DELETE FROM k8s_permission WHERE user_id = 106;
DELETE FROM k8s_group_permission WHERE group_id IN (SELECT id FROM k8s_user_group_member WHERE user_id = 106);
```

### 测试场景

#### 场景 1：验证命名空间隔离

```bash
# test 用户只能看到 fmusic 命名空间
curl -s "http://localhost:8000/api/v1/k8s/cluster/36/namespaces" \
  -H "Authorization: Bearer $TEST_TOKEN" | jq '.data.namespaces[].name'

# 应只返回: "fmusic"

# 访问 fmusic 应成功
curl -s "http://localhost:8000/api/v1/k8s/cluster/36/namespaces/fmusic" \
  -H "Authorization: Bearer $TEST_TOKEN" | jq '.code'

# 访问 default 应 403
curl -s "http://localhost:8000/api/v1/k8s/cluster/36/namespaces/default" \
  -H "Authorization: Bearer $TEST_TOKEN" | jq '.code'
# → 403
```

#### 场景 2：验证读写隔离

```bash
# 列 Pod（有 list verb → 成功 200）
curl -s "http://localhost:8000/api/v1/k8s/cluster/36/namespaces/fmusic/pods" \
  -H "Authorization: Bearer $TEST_TOKEN" | jq '.code'
# → 200

# 删除 Pod（无 delete verb → 403）
curl -s -X DELETE "http://localhost:8000/api/v1/k8s/cluster/36/namespaces/fmusic/pods/test" \
  -H "Authorization: Bearer $TEST_TOKEN" | jq '.code'
# → 403

# 创建 Deployment（无 create verb → 403）
curl -s -X POST "http://localhost:8000/api/v1/k8s/cluster/36/namespaces/fmusic/deployments" \
  -H "Authorization: Bearer $TEST_TOKEN" -H "Content-Type: application/json" -d '{}' | jq '.code'
# → 403
```

#### 场景 3：验证 GetMyPermissions

```bash
# 传统模型 + RBAC 合并
curl -s "http://localhost:8000/api/v1/k8s/permission/my" \
  -H "Authorization: Bearer $TEST_TOKEN" | jq '.'
# 应包含 source=rbac 的条目，permissionType=readonly

# 纯 RBAC
curl -s "http://localhost:8000/api/v1/k8s/rbac/my-permissions" \
  -H "Authorization: Bearer $TEST_TOKEN" | jq '.'
# 应返回 rules 数组含 verbs
```

#### 场景 4：管理员权限

```bash
# admin token 应有所有权限
curl -s "http://localhost:8000/api/v1/k8s/permission/my" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.'
# → clusterId=0, namespace="*", permissionType="admin", source="system"
```

### 前端验证

```javascript
// 打开浏览器 F12 → Console
const { loadPermissions, hasWritePermission, hasPermission } = await import('@/composables/useK8sPermission')
const perm = useK8sPermission()
await perm.loadPermissions()

// 检查 fmusic 命名空间权限
perm.hasPermission(36, 'fmusic')      // → true
perm.hasWritePermission(36, 'fmusic') // → false（只有 get/list/watch）

// 检查 default 命名空间
perm.hasPermission(36, 'default')     // → false
```

## 迭代记录 / Changelog

### 2026-05-27：Bugfix — RBAC-only 用户无法访问 CRD / Storage / Nodes 等集群级别接口

#### 问题描述

用户在 K8s RBAC 系统中为 `test` 用户（ID=106）绑定了集群 `36` 的 `fmusic` 命名空间中包含 `pvcs`, `pvs`, `crds`, `nodes` 资源的 `get/list/watch` 权限。但访问以下接口时返回 `403 您没有该集群的访问权限`：

- `GET /api/v1/k8s/cluster/:id/pvs`（PV 列表）
- `GET /api/v1/k8s/cluster/:id/storageclasses`（StorageClass 列表）
- `GET /api/v1/k8s/cluster/:id/nodes`（节点列表）
- `GET /api/v1/k8s/cluster/:id/crds`（CRD 列表）
- `GET /api/v1/k8s/cluster/:id/crds/groups`（CRD Groups）

这些路由均使用 `CheckClusterPermission()` 中间件，该中间件内部调用 `hasAnyClusterPermission()` 来判定用户是否拥有该集群的访问权限。

#### 根因分析

`hasAnyClusterPermission()` 的第3步原本使用 `hasRbacVerbPermission(userID, clusterID, "", verbs)` 来检查 RBAC 绑定。该方法在 `ruleMap` 中查找键 `"36:"`（集群级别规则，namespace 为空字符串）。但用户的 RBAC 绑定是**命名空间级别**的，`binding.Namespace = "fmusic"`，因此规则映射中的键是 `"36:fmusic"` 而非 `"36:"`，导致 `hasRbacVerbPermission` 永远返回 `false`。

```go
// ❌ 旧代码：只检查集群级别规则 "36:"，不匹配命名空间级 "36:fmusic"
if m.hasRbacVerbPermission(userID, uint(clusterID), "", []string{...}) {
```

#### 修复方案

将 `hasAnyClusterPermission()` 的第3步改为直接遍历用户的 RBAC 绑定列表，检查是否有任何绑定指向该集群，而不要求必须匹配空 namespace 的集群级规则：

```go
// ✅ 新代码：遍历 user 的所有 bindings，匹配 clusterID 即可
bindings := m.getUserRbacBindings(userID)
for _, b := range bindings {
    if b.ClusterID == uint(clusterID) {
        return true, nil
    }
}
```

**文件**: `api/middleware/k8sPermissionMiddleware.go` — `hasAnyClusterPermission()` 方法

#### 影响范围

修复后，所有使用 `CheckClusterPermission()` 中间件的集群级接口均能正确识别命名空间级别 RBAC 绑定用户的权限，包括：

| 路由组 | 中间件 | 示例端点 |
|--------|--------|----------|
| PV 管理 | CheckClusterPermission | `/pvs`, `/pvs/:name` |
| StorageClass 管理 | CheckClusterPermission | `/storageclasses` |
| 节点管理 | CheckClusterPermission | `/nodes` |
| CRD 管理 | CheckClusterPermission | `/crds`, `/crds/groups` |

---

### 2026-05-27：Bugfix — 命名空间列表接口对 RBAC-only 用户返回空数据

#### 问题描述

`test` 用户通过 RBAC 绑定了 `fmusic` 命名空间的权限，但访问 `GET /api/v1/k8s/cluster/:id/namespaces` 时返回空数组。

#### 根因分析

`GetNamespaces()` 在 `k8snamespace.go` 中从数据库中获取用户的 `allowedNS`（传统模型）和 `rbacNS`（RBAC 模型）后合并为 `allowedSet`。在后续过滤集群真实 namespace 列表时，原有的匹配逻辑未正确处理通配符 `*` 和空字符串 `""` 的情况，导致 RBAC-only 用户（其 allowedSet 中包含 `*` 或 `""` 表示全量）被错误过滤。

#### 修复方案

在 `GetNamespaces()` 的两个过滤逻辑分支（数据库查询和 Redis 缓存）中，增加 `hasAll` 判断：

```go
hasAll := allowedSet["*"] || allowedSet[""]
if hasAll || allowedSet[ns.Name] {
    // 允许通过
}
```

**文件**: `api/api/k8s/service/k8snamespace.go` — `GetNamespaces()` 方法

---

### 2026-05-25：功能增强 — RBAC 细粒度权限模型上线

- 新增 `k8s_rbac_role` 和 `k8s_rbac_binding` 表
- 实现 RBAC 角色/绑定 CRUD 的 DAO / Service / Controller
- 权限中间件新增 `hasRbacResourceVerb()`、`getUserRbacRules()` 等函数，支持在 `CheckNamespacePermission()` 中 fallback 检查 RBAC verbs
- 新增 `getResourceAndVerbFromCtx()` 从路由路径中提取资源名和 HTTP 动词
- `getUserNamespacePermissionLevel()` 现集成了 RBAC 规则推导（verbs → 权限级别映射）
- `GetMyPermissions` 接口合并返回传统权限 + RBAC 权限
- 前端 `useK8sPermission` 新增 `hasResourcePermission(clusterId, namespace, resource, verb)` 细粒度检查

---

### 2026-05-29：Bugfix — RBAC 中 namespaces 资源权限不生效

#### 问题描述

用户创建 RBAC 角色并绑定 `namespaces` 资源的 `get/list` 权限，但无论是否有此权限，命名空间列表和详情页始终可以访问，权限控制无效。

#### 根因分析

1. **`getResourceAndVerbFromCtx()` 缺少 `namespaces` 资源名提取** — 该函数通过解析路由路径 `parts` 来获取资源名。对于路径 `/k8s/cluster/:id/namespaces`，`namespaces` 在 `:id` 后面被显式跳过（因为 parts[i+1] == "namespaces"），且 `:namespaceName` 后面没有资源段，最终 `resource` 始终为空字符串 `""`。RBAC 规则检查时 `"namespaces" == ""` 永远不匹配。

2. **`getResourceAndVerbFromCtx()` 缺少 `/:namespaceName` 后缀判断** — GET 请求的 verb 判断列表中缺少 `/:namespaceName`，导致命名空间详情页的 verb 被误判为 `list` 而非 `get`。

#### 修复方案

1. 在 resources 检测逻辑末尾添加兜底检测：当所有常规方式都无法提取资源时，检查路径中是否包含 `namespaces` 段。
2. 在 GET verb 判定列表中添加 `strings.HasSuffix(path, "/:namespaceName")`。

```go
// 修复前后对比
// 修复前：resource="" — 无法匹配 RBAC 规则中的 "namespaces"
// 修复后：resource="namespaces" — 正确匹配 RBAC 规则

// 缺的 verb 判定
strings.HasSuffix(path, "/:namespaceName") // → verb = "get"
```

**文件**: `api/middleware/k8sPermissionMiddleware.go` — `getResourceAndVerbFromCtx()` 方法

#### 影响范围

| 路由 | 修复前 | 修复后 |
|------|--------|--------|
| `GET /k8s/cluster/:id/namespaces` | resource="" → 不匹配任何规则 | resource="namespaces" → 匹配 rules.resources 含 "namespaces" |
| `GET /k8s/cluster/:id/namespaces/:ns` | verb="list" → 误判 | verb="get" → 正确 |
| `POST /k8s/cluster/:id/namespaces` | resource="" → 不匹配 | resource="namespaces" → 匹配 |
| `DELETE /k8s/cluster/:id/namespaces/:ns` | resource="" → 不匹配 | resource="namespaces" → 匹配 |

---

### 2026-05-29：Bugfix — 我的权限中集群名称显示为 cluster-:id

#### 问题描述

`GET /k8s/permission/my` 接口返回的 `clusterName` 字段对于**直接授权**和 **RBAC 授权**的任务显示为 `"cluster-1"`、`"cluster-2"` 而非集群实际名称。

#### 根因分析

`GetMyPermissions()` 中构建的 `clusterNameMap` 仅从**用户组权限**（`k8s_group_permission` 表 JOIN 查询）中获取集群名称，直接授权和 RBAC 权限的 `clusterName` 为空。RBAC 的 `getMyRbacPermissions()` 中使用了降级逻辑 `fmt.Sprintf("cluster-%d", b.ClusterID)`。

#### 修复方案

1. 在 `K8sPermissionServiceImpl` 中新增 `clusterDao *dao.KubeClusterDao`
2. `GetMyPermissions()` 开始时从数据库查询所有集群构建完整 `clusterNameMap`
3. 直接授权项和 RBAC 项都从 `clusterNameMap` 中获取集群名称
4. 去掉 `"cluster-%d"` 的降级逻辑

**文件**: `api/api/k8s/service/k8sPermission.go`

---

### 2026-05-21：初始实现 — 传统三态权限模型

- 创建 `k8s_permission` 表，实现用户-集群-命名空间三级授权
- 实现 `K8sPermissionMiddleware` 的三个中间件：`CheckClusterPermission`、`CheckNamespacePermission`、`FilterAllowedNamespaces`
- 实现完整的 CRUD API 和管理页面
- 支持用户组继承权限
- `FilterAllowedNamespaces` 合并直接授权 + 用户组继承 + RBAC binding 的所有命名空间
