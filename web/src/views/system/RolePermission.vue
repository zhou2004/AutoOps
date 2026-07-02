<template>
  <div class="permission-container">
    <el-row :gutter="20">
      <!-- 左侧：角色列表 -->
      <el-col :span="6">
        <el-card class="box-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span>选择角色 / 用户组</span>
            </div>
          </template>
          <el-menu :default-active="activeRoleId" @select="handleRoleSelect">
            <el-menu-item v-for="role in roleList" :key="role.id" :index="role.id.toString()">
              <el-icon><User /></el-icon>
              <span>{{ role.roleName }}</span>
            </el-menu-item>
          </el-menu>
        </el-card>
      </el-col>

      <!-- 右侧：权限配置区域 -->
      <el-col :span="18">
        <el-card class="box-card" shadow="hover" v-if="activeRoleId">
          <template #header>
            <div class="card-header">
              <span>【{{ activeRoleName }}】权限配置</span>
              <el-button type="primary" size="small" @click="savePermissions">保存所有配置</el-button>
            </div>
          </template>

          <!-- <el-tabs v-model="activeTab" class="permission-tabs"> -->
            
          <el-tabs v-model="activeTab" class="permission-tabs">
            
            <!-- 维度一：K8S 权限体系 (RBAC + 作用域) -->
            <el-tab-pane label="K8s 权限配置" name="k8s">
              <el-alert title="通过添加多条授权规则，实现在不同的集群、同群集下不同/多个 Namespace 分配不同的资源操作权限。" type="success" show-icon style="margin-bottom: 15px;" />
              
              <div style="margin-bottom: 15px;">
                <el-button type="primary" icon="Plus" @click="addK8sRule">新增 K8s 授权规则</el-button>
              </div>

              <!-- 规则列表 -->
              <el-collapse v-model="activeK8sRuleNames" v-if="k8sRules.length > 0">
                <el-collapse-item v-for="(rule, index) in k8sRules" :key="rule.ruleId" :name="rule.ruleId">
                  <template #title>
                    <div style="display: flex; align-items: center; justify-content: space-between; width: 100%; padding-right: 15px;">
                      <span style="font-weight: bold; font-size: 15px;">
                        🎯 规则 {{ index + 1 }} 
                        <el-tag size="small" style="margin-left: 10px;" v-if="rule.clusterId">已选集群: {{ getClusterName(rule.clusterId) }}</el-tag>
                      </span>
                      <el-button type="danger" size="small" plain icon="Delete" @click.stop="removeK8sRule(index)">删除规则</el-button>
                    </div>
                  </template>

                  <div class="rule-content" style="padding: 15px; background: #fafafa; border: 1px solid #EBEEF5; border-radius: 4px;">
                    <!-- 1. 选择作用域 (集群 + Namespace多选) -->
                    <el-form :inline="true" size="default" style="background: var(--bg-card); padding: 15px; border-radius: 4px; box-shadow: 0 2px 4px rgba(0,0,0,0.05); margin-bottom: 15px;">
                      <el-form-item label="目标集群" style="margin-bottom: 0;">
                        <el-select v-model="rule.clusterId" placeholder="请选择集群" @change="() => rule.namespaces = []" style="width: 220px;">
                          <el-option v-for="c in k8sClustersData" :key="c.id" :label="c.name" :value="c.id" />
                        </el-select>
                      </el-form-item>
                      <el-form-item label="目标命名空间" style="margin-bottom: 0;" v-if="rule.clusterId">
                        <el-select v-model="rule.namespaces" multiple collapse-tags placeholder="请选择 (不选默认为整个集群)" style="width: 300px;" clearable>
                          <el-option v-for="ns in getClusterNamespaces(rule.clusterId)" :key="ns.id" :label="ns.name" :value="ns.id" />
                        </el-select>
                      </el-form-item>
                    </el-form>

                    <!-- 2. 操作权限矩阵 -->
                    <div style="margin-bottom: 10px; display: flex; align-items: center; justify-content: space-between;">
                      <div>
                        <span style="font-size: 14px; color: var(--text-regular); font-weight: bold; margin-right: 15px;">配置权限矩阵：</span>
                        <el-radio-group v-model="rule.presetStrategy" size="small" @change="(val) => handlePresetChange(val, rule)">
                          <el-radio-button label="empty">清空权限</el-radio-button>
                          <el-radio-button label="readonly">全部只读 (Get/List/Watch)</el-radio-button>
                          <el-radio-button label="readwrite">研发读写 (无Delete)</el-radio-button>
                          <el-radio-button label="admin">系统管理员 (全量)</el-radio-button>
                        </el-radio-group>
                      </div>
                    </div>

                    <el-table
                      :data="rule.rbacResourceList"
                      border
                      row-key="id"
                      class="rbac-matrix-table"
                      size="small"
                      :span-method="(params) => objectSpanMethod(params, rule)"
                    >
                      <el-table-column prop="moduleName" label="系统模块" width="100" align="center" />
                      <el-table-column prop="resourceName" label="资源对象 (Resource)" width="160">
                        <template #default="{ row }">
                          <span class="resource-label">{{ row.resourceName }}</span>
                        </template>
                      </el-table-column>

                      <el-table-column label="全选" width="60" align="center">
                        <template #default="{ row }">
                          <el-checkbox
                            v-model="row.isAllChecked"
                            :indeterminate="row.isIndeterminate"
                            @change="(val) => handleRowCheckAll(val, row, rule)"
                          />
                        </template>
                      </el-table-column>

                      <el-table-column v-for="verb in rule.verbsList" :key="verb.key" :prop="verb.key" :label="verb.label" align="center">
                        <template #header>
                          <div class="verb-header">
                            <el-checkbox
                              v-model="verb.isAllChecked"
                              :indeterminate="verb.isIndeterminate"
                              @change="(val) => handleColumnCheckAll(val, verb.key, rule)"
                            />
                            <span>{{ verb.label }}</span>
                          </div>
                        </template>
                        <template #default="{ row }">
                          <el-checkbox
                            v-if="row.verbs.includes(verb.key)"
                            v-model="row.permissions[verb.key]"
                            @change="() => checkRowState(row, verb.key, rule)"
                          />
                          <span v-else style="color: #c0c4cc;">-</span>
                        </template>
                      </el-table-column>
                    </el-table>
                  </div>
                </el-collapse-item>
              </el-collapse>
              <el-empty v-else description="暂无 K8s 授权规则，请点击上方按钮新增" />
            </el-tab-pane>

            <!-- 维度二：系统菜单与API动作权限 -->
            <el-tab-pane label="基础菜单权限" name="menu">
              <el-alert title="勾选该角色可以访问的菜单及对应的操作按钮权限" type="info" show-icon style="margin-bottom: 15px;" />
              <el-tree
                ref="menuTree"
                :data="menuTreeData"
                show-checkbox
                node-key="id"
                default-expand-all
                :props="{ children: 'children', label: 'label' }"
              >
              </el-tree>
            </el-tab-pane>

            <!-- 维度三：CMDB 数据权限 -->
            <el-tab-pane label="CMDB 资源权限" name="cmdb">
              <el-alert title="配置该角色可管理的主机组 / 业务树" type="warning" show-icon style="margin-bottom: 15px;" />
              <el-tree
                  ref="cmdbTree"
                  :data="cmdbGroupData"
                  show-checkbox
                  node-key="id"
                  check-strictly
                  :props="{ children: 'children', label: 'groupName' }"
              />
            </el-tab-pane>

          </el-tabs>
        </el-card>
        
        <el-empty v-else description="请在左侧选择需要配置的角色"></el-empty>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'

// 激活的状态
const activeRoleId = ref('')
const activeTab = ref('k8s')
const k8sPermissionType = ref('custom')

// 模拟左侧角色数据
const roleList = ref([
  { id: 1, roleName: '超级管理员' },
  { id: 2, roleName: 'K8s运维研发' },
  { id: 3, roleName: '后端业务开发组' }
])

// 获取当前选中角色名
const activeRoleName = computed(() => {
  const role = roleList.value.find(r => r.id.toString() === activeRoleId.value)
  return role ? role.roleName : ''
})

// K8s 集群与命名空间数据源
const k8sClustersData = ref([
  {
    id: 'cluster-dev', name: '开发测试集群 (Dev)', namespaces: [
      { id: 'ns-dev-app', name: 'app-namespace' },
      { id: 'ns-dev-mid', name: 'middleware' }
    ]
  },
  {
    id: 'cluster-prod', name: '生产核心集群 (Prod)', namespaces: [
      { id: 'ns-prod-core', name: 'core-system' },
      { id: 'ns-prod-infra', name: 'infra-system' }
    ]
  }
])

const getClusterName = (clusterId) => {
  const c = k8sClustersData.value.find(c => c.id === clusterId);
  return c ? c.name : clusterId;
}

const getClusterNamespaces = (clusterId) => {
  const c = k8sClustersData.value.find(c => c.id === clusterId);
  return c ? c.namespaces : [];
}

// == 管理多条 K8s RBAC 授权规则 ==
const activeK8sRuleNames = ref([]) // 展开的规则折叠面板
const k8sRules = ref([]) // 这里存放该角色的所有K8s规则

// 创建一个干净的权限矩阵实例
const createEmptyMatrix = () => {
  const verbsList = [
    { key: 'get', label: '获取(get)', isAllChecked: false, isIndeterminate: false },
    { key: 'list', label: '列表(list)', isAllChecked: false, isIndeterminate: false },
    { key: 'watch', label: '监听(watch)', isAllChecked: false, isIndeterminate: false },
    { key: 'create', label: '创建(create)', isAllChecked: false, isIndeterminate: false },
    { key: 'update', label: '更新(update)', isAllChecked: false, isIndeterminate: false },
    { key: 'patch', label: '修改(patch)', isAllChecked: false, isIndeterminate: false },
    { key: 'delete', label: '删除(delete)', isAllChecked: false, isIndeterminate: false }
  ];

  const rbacResourceList = [
    { id: 1, moduleName: '工作负载', resourceName: 'Deployments', verbs: ['get', 'list', 'watch', 'create', 'update', 'patch', 'delete'] },
    { id: 2, moduleName: '工作负载', resourceName: 'Pods', verbs: ['get', 'list', 'watch', 'create', 'update', 'patch', 'delete'] },
    { id: 3, moduleName: '工作负载', resourceName: 'StatefulSets', verbs: ['get', 'list', 'watch', 'create', 'update', 'patch', 'delete'] },
    { id: 4, moduleName: '工作负载', resourceName: 'DaemonSets', verbs: ['get', 'list', 'watch', 'create', 'update', 'patch', 'delete'] },
    { id: 5, moduleName: '网络服务', resourceName: 'Services', verbs: ['get', 'list', 'watch', 'create', 'update', 'patch', 'delete'] },
    { id: 6, moduleName: '网络服务', resourceName: 'Ingresses', verbs: ['get', 'list', 'watch', 'create', 'update', 'patch', 'delete'] },
    { id: 7, moduleName: '存储配置', resourceName: 'ConfigMaps', verbs: ['get', 'list', 'watch', 'create', 'update', 'patch', 'delete'] },
    { id: 8, moduleName: '存储配置', resourceName: 'Secrets', verbs: ['get', 'list', 'watch', 'create', 'update', 'patch', 'delete'] },
    { id: 9, moduleName: '集群视图', resourceName: 'Nodes', verbs: ['get', 'list', 'watch'] },
    { id: 10, moduleName: '集群视图', resourceName: 'Namespaces', verbs: ['get', 'list', 'watch', 'create', 'update', 'patch', 'delete'] },
    { id: 11, moduleName: '自定义资源', resourceName: 'CRDs', verbs: ['get', 'list', 'watch', 'create', 'update', 'patch', 'delete'] }
  ].map(res => ({
    ...res,
    isAllChecked: false,
    isIndeterminate: false,
    permissions: res.verbs.reduce((acc, v) => ({ ...acc, [v]: false }), {})
  }));

  return { verbsList, rbacResourceList };
}

// 新增规则
const addK8sRule = () => {
  const newRuleId = Date.now().toString();
  const matrix = createEmptyMatrix();
  k8sRules.value.push({
    ruleId: newRuleId,
    clusterId: '',
    namespaces: [], // 多选
    presetStrategy: '',
    verbsList: matrix.verbsList,
    rbacResourceList: matrix.rbacResourceList
  });
  activeK8sRuleNames.value.push(newRuleId); // 自动展开新加入的面板
}

// 删除规则
const removeK8sRule = (index) => {
  k8sRules.value.splice(index, 1);
}

// 表格合并控制，将 moduleName 相同的单元格合并
const objectSpanMethod = ({ row, column, rowIndex, columnIndex }, rule) => {
  if (columnIndex === 0) {
    if (rowIndex === 0 || rule.rbacResourceList[rowIndex].moduleName !== rule.rbacResourceList[rowIndex - 1].moduleName) {
      let rowspan = 1;
      for (let i = rowIndex + 1; i < rule.rbacResourceList.length; i++) {
        if (rule.rbacResourceList[i].moduleName === row.moduleName) rowspan++;
        else break;
      }
      return { rowspan, colspan: 1 };
    } else {
      return { rowspan: 0, colspan: 0 };
    }
  }
}

// 检查行的勾选状态更新 checkbox (全选/半选)
const checkRowState = (row, verbKey, rule) => {
  let checkedCount = 0;
  row.verbs.forEach(v => {
    if (row.permissions[v]) checkedCount++;
  })
  row.isAllChecked = checkedCount === row.verbs.length;
  row.isIndeterminate = checkedCount > 0 && checkedCount < row.verbs.length;
  
  if (verbKey) updateColumnHeaderState(verbKey, rule);
}

// 行级全选操作
const handleRowCheckAll = (val, row, rule) => {
  row.verbs.forEach(v => {
    row.permissions[v] = val;
  })
  row.isIndeterminate = false;
  row.verbs.forEach(v => updateColumnHeaderState(v, rule));
}

// 列级全选操作 (头部的 Checkbox)
const handleColumnCheckAll = (val, verbKey, rule) => {
  rule.rbacResourceList.forEach(row => {
    if (row.verbs.includes(verbKey)) {
      row.permissions[verbKey] = val;
      checkRowState(row, null, rule);
    }
  })
  updateColumnHeaderState(verbKey, rule);
}

// 更新列头的全选/半选状态
const updateColumnHeaderState = (verbKey, rule) => {
  const targetVerb = rule.verbsList.find(v => v.key === verbKey);
  if (!targetVerb) return;

  let total = 0;
  let checkedCount = 0;
  rule.rbacResourceList.forEach(row => {
    if (row.verbs.includes(verbKey)) {
      total++;
      if (row.permissions[verbKey]) checkedCount++;
    }
  });

  if (total === 0) return;
  targetVerb.isAllChecked = checkedCount === total;
  targetVerb.isIndeterminate = checkedCount > 0 && checkedCount < total;
}

// 快捷策略点击器
const handlePresetChange = (strategy, rule) => {
  rule.rbacResourceList.forEach(row => {
    row.verbs.forEach(verb => {
      let isAllow = false;
      if (strategy === 'readonly') {
        isAllow = ['get', 'list', 'watch'].includes(verb);
      } else if (strategy === 'readwrite') {
        isAllow = ['get', 'list', 'watch', 'create', 'update', 'patch'].includes(verb);
      } else if (strategy === 'admin') {
        isAllow = true; // all
      } else if (strategy === 'empty') {
        isAllow = false; // none
      }
      row.permissions[verb] = isAllow;
    })
    checkRowState(row, null, rule);
  });
  
  rule.verbsList.forEach(v => {
    updateColumnHeaderState(v.key, rule);
  });
}
// == 结束 Kuboard 风格 RBAC 设计 ==

const handleRoleSelect = (roleId) => {
  activeRoleId.value = roleId
  // TODO: 这里触发网络请求，拉取该 roleId 在各个维度的已有权限数据，并回显到树中
  // fetchRoleMenuPermissions(roleId)
  // fetchRoleK8sPermissions(roleId)
}

const savePermissions = () => {
  // TODO: 收集各个 Tree 中选中的 Key，组装成 JSON 提交给后端
  // menuTree.value.getCheckedKeys()
  
  // 收集所有的 K8s 规则
  const payloadK8sRules = k8sRules.value.map(rule => ({
    clusterId: rule.clusterId,
    namespaces: rule.namespaces.length === 0 ? ['*'] : rule.namespaces, // 如果空代表所有NS
    permissions: rule.rbacResourceList.map(row => ({
      resource: row.resourceName,
      verbs: row.permissions
    }))
  }));

  console.log("保存权限配置提交", {
    roleId: activeRoleId.value,
    k8sRoleBindings: payloadK8sRules
  });
}
</script>

<style scoped>
.permission-container {
  padding: 20px;
}
.custom-tree-node {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding-right: 20px;
}
.permission-tabs {
  margin-top: 20px;
}
.rbac-matrix-table :deep(th) {
  background-color: var(--bg-card-alt);
}
.verb-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  font-size: 12px;
  gap: 4px;
}
.resource-label {
  font-weight: bold;
  color: var(--text-primary);
}
.section-header {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 15px;
  padding-left: 10px;
  border-left: 4px solid #409EFF;
}
.sub-section {
  margin-bottom: 20px;
}
</style>