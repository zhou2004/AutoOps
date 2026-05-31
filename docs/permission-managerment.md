# Kubernetes 细粒度权限管理 (permission-management)

## 1. 架构与实现原理
AutoOps 的 K8s 权限管理支持细粒度的 RBAC 模型。管理员现在不仅可以授予传统的 `readonly`、`write`、`admin` 层级，还可以为用户分配基于具体 `APIGroups`、`Resources` 和 `Verbs` 的角色。

前端获取到当前用户在 K8s 模块的权限时，其中包含 `Rules` 规则列表：
```json
{
  "clusterId": 1,
  "namespace": "default",
  "permissionType": "custom",
  "rules": [
    {
      "apiGroups": [""],
      "resources": ["deployments", "secrets"],
      "verbs": ["get", "list", "create"]
    }
  ]
}
```

## 2. 前端鉴权接入点
在前端应用中，已经提取了统一的组合式函数（Composable）来进行权限检查，参见 `web/src/composables/useK8sPermission.js`。

### 核心方法
* `hasResourcePermission(clusterId, namespace, resource, verb)`：用于检查特定资源和行为的权限。
  * **参数：**
    * `clusterId` (Number): 集群 ID
    * `namespace` (String): 命名空间。`"*"` 表示全命名空间。
    * `resource` (String): 资源名称，复数形式。如 `'deployments'`，`'pods'` 等。
    * `verb` (String): 操作。如 `'get'`，`'list'`，`'create'`，`'update'`，`'delete'`。
  * **返回：**
    * 返回 `Boolean` 说明是否拥有操作该资源的权限。

### 示例用法
在 `k8s-workloads.vue` 中可以这样控制 Tab 页签的渲染与操作按钮的禁用：
```javascript
import { useK8sPermission } from '@/composables/useK8sPermission'
const { loadPermissions, hasResourcePermission } = useK8sPermission()

// 确保在 onMounted 时加载权限信息
onMounted(async () => {
    await loadPermissions()
})

const canCreateDeployment = () => {
    return hasResourcePermission(clusterId, 'default', 'deployments', 'create')
}
```

```html
<!-- 如果没有权限，则隐藏部署选项卡 -->
<el-tab-pane label="Deployment" name="deployments" v-if="canCreateDeployment()">
</el-tab-pane>

<!-- 禁用不能更新的操作按钮 -->
<el-button :disabled="!hasResourcePermission(clusterId, namespace, 'deployments', 'update')">重启</el-button>
```

## 3. 测试步骤
如果要完整测试这一细粒度权限控制机制，请按照以下步骤操作：

1. **登录系统并进入 K8s 权限配置页：**
   作为系统管理员登录 AutoOps 后台，进入 “K8s 集群管理 -> 权限管理” 页面。
2. **创建一个自定义 RBAC 角色：**
   - 进入“角色管理” Tab 页，点击“新建角色”。
   - 填写角色名称为 `test-deploy-viewer`，选择要绑定的命名空间（如 `default`）。
   - 添加一条规则：
     * API Groups: 选择 `apps` （如果是 deployment 等）
     * 资源 (Resources): 选中 `deployments`
     * 操作动作 (Verbs): 选择 `get`, `list`, `watch`
3. **将该角色绑定给某个普通用户组 (或直接用户)：**
   - 进入“角色绑定 (RoleBindings)”页面，创建绑定。
   - 选中上述角色 `test-deploy-viewer`，绑定给用户A所在的组。
4. **切换账号测试界面响应：**
   - 使用用户A登录系统，打开“工作负载 (Workloads)” 页面。
   - 观察页面的 **[全部分类]** 及 **[Deployment]** 的 Tab 也将被显示出来，因为他们对其有 `list`/`get` 权限。
   - **其他选项卡隐藏**：用户A并未被授权访问 StatefulSet、DaemonSet 等资源，相应选项卡应当消失。
   - **创建/修改/删除被禁用**：尝试点击“创建工作负载”，或点击某个已有 Deployment 后的缩放、更新等按钮。因为该用户并未被分配 `create` 或 `update` 动作的权限，预期这些按钮均被禁用（置灰或不可见）。

通过此测试，即可印证前后端一体的 `APIGroups -> Resources -> Verbs` 细粒度资源沙盒访问能力已正常运作。
