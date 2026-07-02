<template>
  <div class="k8s-permission">
    <div class="page-header">
      <h2>K8s 权限管理</h2>
    </div>

    <!-- 标签页切换 -->
    <el-tabs v-model="activeTab" @tab-click="handleTabClick">
      <!-- ==================== 用户权限管理 ==================== -->
      <el-tab-pane label="用户权限" name="userPermission">
        <div class="header-actions">
          <el-button type="primary" @click="showCreateDialog">+ 新增权限</el-button>
          <el-button type="success" @click="showBatchCreateDialog" >批量授权</el-button>
        </div>

        <!-- 搜索区域 -->
        <div class="search-area">
          <el-form :inline="true" :model="queryParams" size="small">
            <el-form-item label="用户">
              <el-select v-model="queryParams.userId" placeholder="选择用户" clearable filterable style="width: 180px">
                <el-option v-for="user in userList" :key="user.id" :label="(user.username||'') + ' (' + (user.nickname||'') + ')'" :value="user.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="集群">
              <el-select v-model="queryParams.clusterId" placeholder="选择集群" clearable filterable style="width: 180px" @change="onClusterChangeForFilter">
                <el-option v-for="cluster in clusterList" :key="cluster.id" :label="cluster.name" :value="cluster.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="命名空间">
              <el-select v-model="queryParams.namespace" placeholder="选择命名空间" clearable filterable style="width: 180px">
                <el-option v-for="ns in filterNamespaceOptions" :key="ns" :label="ns" :value="ns" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleSearch">查询</el-button>
              <el-button @click="handleReset">重置</el-button>
            </el-form-item>
          </el-form>
        </div>

        <!-- 权限列表 -->
        <el-table :data="permissionList" v-loading="loading" class="modern-table">
          <el-table-column label="ID" prop="id" v-if="false" />
          <el-table-column prop="username" label="用户名" min-width="120">
            <template #default="scope">
              <span>{{ scope.row.username || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="nickname" label="昵称" min-width="120">
            <template #default="scope">
              <span>{{ scope.row.nickname || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="clusterName" label="集群名称" min-width="150">
            <template #default="scope">
              <span>{{ scope.row.clusterName || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="命名空间" min-width="150">
            <template #default="scope">
              <span>{{ scope.row.namespace || '(集群级别)' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="permissionType" label="权限类型" width="120" align="center">
            <template #default="scope">
              <el-tag :type="scope.row.permissionType === 'admin' ? 'danger' : scope.row.permissionType === 'write' ? 'warning' : 'info'" size="small">
                {{ scope.row.permissionType === 'admin' ? '管理员' : scope.row.permissionType === 'write' ? '读写' : '只读' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="createdAt" label="创建时间" width="180" />
          <el-table-column label="操作" width="200" align="center" fixed="right">
            <template #default="scope">
              <div class="operation-buttons">
                <el-tooltip content="编辑" placement="top">
                  <el-button type="warning" size="small"  @click="showEditDialog(scope.row)">
                    <el-icon><Edit /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="删除" placement="top">
                  <el-button type="danger" size="small"  @click="handleDelete(scope.row)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
        </el-table>

        <!-- 分页 -->
        <div class="pagination-area">
          <el-pagination
            @size-change="handleSizeChange"
            @current-change="handleCurrentChange"
            :current-page="queryParams.page"
            :page-sizes="[10, 20, 50, 100]"
            :page-size="queryParams.size"
            layout="total, sizes, prev, pager, next, jumper"
            :total="total">
          </el-pagination>
        </div>
      </el-tab-pane>

      <!-- ==================== 用户组管理 ==================== -->
      <el-tab-pane label="用户组管理" name="userGroup">
        <div class="header-actions">
          <el-button type="primary" @click="showGroupCreateDialog" >+ 新增用户组</el-button>
        </div>
        <div class="search-area">
          <el-form :inline="true" size="small">
            <el-form-item label="用户组名称">
              <el-input v-model="groupQuery.name" placeholder="用户组名称" clearable style="width: 200px" @keyup.enter="handleGroupSearch" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleGroupSearch">查询</el-button>
              <el-button @click="groupQuery={page:1,size:10,name:''};fetchGroupList()">重置</el-button>
            </el-form-item>
          </el-form>
        </div>

        <!-- 用户组列表 -->
        <el-table :data="groupList" v-loading="groupLoading" class="modern-table">
          <el-table-column label="ID" prop="id" v-if="false" />
          <el-table-column prop="name" label="用户组名称" min-width="150" />
          <el-table-column prop="code" label="编码" min-width="120" />
          <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
          <el-table-column prop="status" label="状态" width="80" align="center">
            <template #default="scope">
              <el-tag :type="scope.row.status === 1 ? 'success' : 'info'" size="small">
                {{ scope.row.status === 1 ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="createdAt" label="创建时间" width="180" />
          <el-table-column label="操作" width="300" align="center" fixed="right">
            <template #default="scope">
              <div class="operation-buttons">
                <el-tooltip content="编辑" placement="top">
                  <el-button type="warning" size="small"  @click="showGroupEditDialog(scope.row)">
                    <el-icon><Edit /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="成员" placement="top">
                  <el-button type="primary" size="small"  @click="showGroupMembersDialog(scope.row)">
                    <el-icon><User /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="权限" placement="top">
                  <el-button type="success" size="small"  @click="showGroupPermissionsDialog(scope.row)">
                    <el-icon><Key /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="删除" placement="top">
                  <el-button type="danger" size="small"  @click="handleDeleteGroup(scope.row)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
        </el-table>

        <!-- 用户组分页 -->
        <div class="pagination-area">
          <el-pagination
            @size-change="handleGroupSizeChange"
            @current-change="handleGroupCurrentChange"
            :current-page="groupQuery.page"
            :page-sizes="[10, 20, 50, 100]"
            :page-size="groupQuery.size"
            layout="total, sizes, prev, pager, next, jumper"
            :total="groupTotal">
          </el-pagination>
        </div>
      </el-tab-pane>

      <!-- ==================== RBAC 角色管理 ==================== -->
      <el-tab-pane label="RBAC 角色" name="rbacRole">
        <div class="header-actions">
          <el-button type="primary" @click="showRbacRoleCreateDialog">+ 新增角色</el-button>
        </div>
        <div class="search-area">
          <el-form :inline="true" size="small">
            <el-form-item label="集群">
              <el-select v-model="rbacRoleQuery.clusterId" placeholder="选择集群" clearable filterable style="width: 180px" @change="onRbacRoleFilterClusterChange">
                <el-option v-for="c in clusterList" :key="c.id" :label="c.name" :value="c.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="命名空间">
              <el-select v-model="rbacRoleQuery.namespace" placeholder="选择命名空间" clearable filterable multiple style="width: 200px">
                <el-option v-for="ns in rbacRoleFilterNsOptions" :key="ns" :label="ns" :value="ns" />
              </el-select>
            </el-form-item>
            <el-form-item label="角色名称">
              <el-input v-model="rbacRoleQuery.name" placeholder="角色名称" clearable style="width: 150px" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="rbacRoleQuery.page=1;fetchRbacRoleList()">查询</el-button>
              <el-button @click="rbacRoleQuery={clusterId:undefined,namespace:'',name:'',page:1,size:10};rbacRoleFilterNsOptions=[];fetchRbacRoleList()">重置</el-button>
            </el-form-item>
          </el-form>
        </div>
        <el-table :data="rbacRoleList" v-loading="rbacRoleLoading" class="modern-table">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="name" label="角色名称" min-width="140" />
          <el-table-column label="集群" min-width="120">
            <template #default="scope">
              <span>{{ scope.row.clusterName || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="命名空间" width="120">
            <template #default="scope">
              <span>{{ scope.row.namespace || '(集群级别)' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="规则 (Rules)" min-width="300">
            <template #default="scope">
              <div v-if="scope.row.rules && scope.row.rules.length" v-for="(rule, i) in scope.row.rules" :key="i" class="rbac-rule-item">
                <el-tag size="small" type="primary">API: {{ (rule.apiGroups||['']).join(',') }}</el-tag>
                <el-tag size="small" type="success">资源: {{ (rule.resources||['']).join(',') }}</el-tag>
                <el-tag size="small" :type="getVerbsTagType(rule.verbs)">操作: {{ (rule.verbs||['']).join(',') }}</el-tag>
              </div>
              <span v-else class="no-verbs">无规则</span>
            </template>
          </el-table-column>
          <el-table-column prop="createdAt" label="创建时间" width="170" />
          <el-table-column label="操作" width="160" align="center" fixed="right">
            <template #default="scope">
              <el-button type="warning" size="small" @click="showRbacRoleEditDialog(scope.row)">
                <el-icon><Edit /></el-icon>
              </el-button>
              <el-button type="danger" size="small" @click="handleDeleteRbacRole(scope.row)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="pagination-area">
          <el-pagination
            @size-change="s=>{rbacRoleQuery.size=s;fetchRbacRoleList()}"
            @current-change="p=>{rbacRoleQuery.page=p;fetchRbacRoleList()}"
            :current-page="rbacRoleQuery.page"
            :page-sizes="[10, 20, 50, 100]"
            :page-size="rbacRoleQuery.size"
            layout="total, sizes, prev, pager, next, jumper"
            :total="rbacRoleTotal">
          </el-pagination>
        </div>
      </el-tab-pane>

      <!-- ==================== RBAC 绑定管理 ==================== -->
      <el-tab-pane label="RBAC 绑定" name="rbacBinding">
        <div class="header-actions">
          <el-button type="primary" @click="showRbacBindingCreateDialog">+ 新增绑定</el-button>
        </div>
        <div class="search-area">
          <el-form :inline="true" size="small">
            <el-form-item label="集群">
              <el-select v-model="rbacBindingQuery.clusterId" placeholder="选择集群" clearable filterable style="width: 180px" @change="onRbacBindingFilterClusterChange">
                <el-option v-for="c in clusterList" :key="c.id" :label="c.name" :value="c.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="命名空间">
              <el-select v-model="rbacBindingQuery.namespace" placeholder="选择命名空间" clearable filterable multiple style="width: 200px">
                <el-option v-for="ns in rbacBindingFilterNsOptions" :key="ns" :label="ns" :value="ns" />
              </el-select>
            </el-form-item>
            <el-form-item label="主体类型">
              <el-select v-model="rbacBindingQuery.subjectType" placeholder="主体类型" clearable style="width: 120px">
                <el-option label="用户" value="User" />
                <el-option label="用户组" value="Group" />
              </el-select>
            </el-form-item>
            <el-form-item label="主体名称">
              <el-input v-model="rbacBindingQuery.subjectName" placeholder="主体名称" clearable style="width: 150px" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="rbacBindingQuery.page=1;fetchRbacBindingList()">查询</el-button>
              <el-button @click="rbacBindingQuery={clusterId:undefined,namespace:'',subjectType:'',subjectName:'',page:1,size:10};rbacBindingFilterNsOptions=[];fetchRbacBindingList()">重置</el-button>
            </el-form-item>
          </el-form>
        </div>
        <el-table :data="rbacBindingList" v-loading="rbacBindingLoading" class="modern-table">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column label="集群" min-width="120">
            <template #default="scope">
              <span>{{ scope.row.clusterName || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="命名空间" width="120">
            <template #default="scope">
              <span>{{ scope.row.namespace || '(集群级别)' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="角色" min-width="140">
            <template #default="scope">
              <span>{{ scope.row.roleName || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="subjectType" label="主体类型" width="100" align="center">
            <template #default="scope">
              <el-tag :type="scope.row.subjectType==='User'?'primary':'success'" size="small">
                {{ scope.row.subjectType==='User'?'用户':'用户组' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="主体名称" min-width="140">
            <template #default="scope">
              <span>{{ scope.row.subjectName || scope.row.subjectId || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="createdAt" label="创建时间" width="170" />
          <el-table-column label="操作" width="150" align="center" fixed="right">
            <template #default="scope">
              <el-button type="warning" size="small" @click="showRbacBindingEditDialog(scope.row)">
                <el-icon><Edit /></el-icon>
              </el-button>
              <el-button type="danger" size="small" @click="handleDeleteRbacBinding(scope.row)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="pagination-area">
          <el-pagination
            @size-change="s=>{rbacBindingQuery.size=s;fetchRbacBindingList()}"
            @current-change="p=>{rbacBindingQuery.page=p;fetchRbacBindingList()}"
            :current-page="rbacBindingQuery.page"
            :page-sizes="[10, 20, 50, 100]"
            :page-size="rbacBindingQuery.size"
            layout="total, sizes, prev, pager, next, jumper"
            :total="rbacBindingTotal">
          </el-pagination>
        </div>
      </el-tab-pane>

      <!-- ==================== 我的权限 ==================== -->
      <el-tab-pane label="我的权限" name="myPermissions">
        <el-alert title="以下显示您在当前 K8s 模块中的有效权限，合并自直接授权、用户组继承和 RBAC 角色绑定" type="info" show-icon :closable="false" style="margin-bottom:16px" />
        <div class="header-actions">
          <el-button type="primary" @click="fetchMyPermissions">刷新权限</el-button>
        </div>
        <el-table :data="myPermList" v-loading="myPermLoading" class="modern-table">
          <el-table-column label="集群" min-width="140">
            <template #default="scope">
              <span>{{ scope.row.clusterName || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="namespace" label="命名空间" width="140">
            <template #default="scope">
              <span v-if="scope.row.clusterId===0 && scope.row.namespace==='*'" style="font-weight:bold">所有命名空间</span>
              <span v-else>{{ scope.row.namespace || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="permissionType" label="权限级别" width="120" align="center">
            <template #default="scope">
              <el-tag v-if="scope.row.permissionType" :type="scope.row.permissionType==='admin'?'danger':scope.row.permissionType==='write'?'warning':'info'" size="small">
                {{ scope.row.permissionType==='admin'?'管理员':scope.row.permissionType==='write'?'读写':'只读' }}
              </el-tag>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column prop="source" label="来源" width="100" align="center">
            <template #default="scope">
              <el-tag v-if="scope.row.source==='system'" type="danger" size="small">系统</el-tag>
              <el-tag v-else-if="scope.row.source==='direct'" type="primary" size="small">直接授权</el-tag>
              <el-tag v-else-if="scope.row.source==='group'" type="success" size="small">用户组</el-tag>
              <el-tag v-else-if="scope.row.source==='rbac'" type="warning" size="small">RBAC</el-tag>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column label="RBAC Verbs" min-width="280">
            <template #default="scope">
              <div v-if="scope.row.rules && scope.row.rules.length>0" style="display:flex;flex-wrap:wrap;gap:4px">
                <template v-for="(rule,ri) in scope.row.rules" :key="ri">
                  <el-tag v-for="v in rule.verbs||[]" :key="v" :type="getVerbTagType(v)" size="small" style="margin:2px">{{ v }}</el-tag>
                </template>
              </div>
              <span v-else class="no-verbs">-</span>
            </template>
          </el-table-column>
        </el-table>
        <div class="pagination-area">
          <el-pagination
            @size-change="s=>{myPermPageSize=s;myPermPage=1}"
            @current-change="p=>{myPermPage=p}"
            :current-page="myPermPage"
            :page-sizes="[10, 20, 50, 100]"
            :page-size="myPermPageSize"
            layout="total, sizes, prev, pager, next, jumper"
            :total="myPermList.length">
          </el-pagination>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- ==================== 创建/编辑权限对话框 ==================== -->
    <el-dialog :title="dialogTitle" v-model="dialogVisible" width="500px" @close="resetDialog">
      <el-form :model="formData" :rules="formRules" ref="formRef" label-width="100px">
        <el-form-item label="用户" prop="userId">
          <el-select v-model="formData.userId" placeholder="选择用户" filterable style="width: 100%">
            <el-option v-for="user in userList" :key="user.id" :label="user.username + ' (' + user.nickname + ')'" :value="user.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="集群" prop="clusterId">
          <el-select v-model="formData.clusterId" placeholder="选择集群" filterable style="width: 100%" @change="onDialogClusterChange">
            <el-option v-for="cluster in clusterList" :key="cluster.id" :label="cluster.name" :value="cluster.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="命名空间" prop="namespace">
          <el-select v-model="formData.namespace" placeholder="选择命名空间" filterable allow-create default-first-option style="width: 100%">
            <el-option v-for="ns in dialogNamespaceOptions" :key="ns" :label="ns" :value="ns" />
          </el-select>
          <div style="color: var(--text-secondary); font-size: 12px; margin-top: 4px;">选择命名空间，也可手动输入</div>
        </el-form-item>
        <el-form-item label="权限类型" prop="permissionType">
          <el-select v-model="formData.permissionType" placeholder="选择权限类型" style="width: 100%">
            <el-option label="只读 (readonly)" value="readonly" />
            <el-option label="读写 (write)" value="write" />
            <el-option label="管理员 (admin)" value="admin" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitLoading">确定</el-button>
      </template>
    </el-dialog>

    <!-- ==================== 批量授权对话框 ==================== -->
    <el-dialog title="批量授权" v-model="batchDialogVisible" width="550px" @close="resetBatchDialog">
      <el-form :model="batchFormData" :rules="batchFormRules" ref="batchFormRef" label-width="100px">
        <el-form-item label="用户" prop="userId">
          <el-select v-model="batchFormData.userId" placeholder="选择用户" filterable style="width: 100%">
            <el-option v-for="user in userList" :key="user.id" :label="user.username + ' (' + user.nickname + ')'" :value="user.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="集群" prop="clusterId">
          <el-select v-model="batchFormData.clusterId" placeholder="选择集群" filterable style="width: 100%" @change="onBatchClusterChange">
            <el-option v-for="cluster in clusterList" :key="cluster.id" :label="cluster.name" :value="cluster.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="命名空间" prop="namespaces">
          <el-select v-model="batchFormData.namespaces" multiple filterable allow-create default-first-option placeholder="选择或输入命名空间" style="width: 100%">
            <el-option v-for="ns in dialogNamespaceOptions" :key="ns" :label="ns" :value="ns" />
          </el-select>
          <div style="color: var(--text-secondary); font-size: 12px; margin-top: 4px;">支持手动输入多个命名空间，按回车确认</div>
        </el-form-item>
        <el-form-item label="权限类型" prop="permissionType">
          <el-select v-model="batchFormData.permissionType" placeholder="选择权限类型" style="width: 100%">
            <el-option label="只读 (readonly)" value="readonly" />
            <el-option label="读写 (write)" value="write" />
            <el-option label="管理员 (admin)" value="admin" />
          </el-select>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="batchDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleBatchSubmit" :loading="submitLoading">确定</el-button>
      </span>
    </el-dialog>

    <!-- ==================== 创建/编辑用户组对话框 ==================== -->
    <el-dialog :title="groupDialogTitle" v-model="groupDialogVisible" width="500px" @close="resetGroupDialog">
      <el-form :model="groupFormData" :rules="groupFormRules" ref="groupFormRef" label-width="100px">
        <el-form-item label="用户组名称" prop="name">
          <el-input v-model="groupFormData.name" placeholder="输入用户组名称" />
        </el-form-item>
        <el-form-item label="编码" prop="code">
          <el-input v-model="groupFormData.code" placeholder="输入用户组编码（可选）" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="groupFormData.description" type="textarea" :rows="3" placeholder="输入描述（可选）" />
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-radio-group v-model="groupFormData.status">
            <el-radio :label="1">启用</el-radio>
            <el-radio :label="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="groupDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleGroupSubmit" :loading="submitLoading">确定</el-button>
      </span>
    </el-dialog>

    <!-- ==================== 用户组成员管理对话框 ==================== -->
    <el-dialog title="用户组成员管理" v-model="memberDialogVisible" width="700px" @close="resetMemberDialog">
      <div v-if="currentGroup">
        <div style="margin-bottom: 16px;">
          <strong>用户组：</strong>{{ currentGroup.name }}
          <el-button type="primary" size="small" style="float: right;"  @click="showAddMemberDialog">+ 添加成员</el-button>
        </div>
        <el-table :data="memberList" v-loading="memberLoading" class="modern-table">
          <el-table-column label="ID" prop="id" v-if="false" />
          <el-table-column label="用户名" min-width="120">
            <template #default="scope">
              <span>{{ scope.row.username || scope.row.name || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="昵称" min-width="120">
            <template #default="scope">
              <span>{{ scope.row.nickname || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" align="center">
            <template #default="scope">
              <div class="operation-buttons">
                <el-tooltip content="移除" placement="top">
                  <el-button type="danger" size="small"  @click="handleRemoveMember(scope.row)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>

    <!-- ==================== 添加成员对话框 ==================== -->
    <el-dialog title="添加成员" v-model="addMemberDialogVisible" width="500px">
      <el-form :model="addMemberForm" ref="addMemberFormRef" label-width="80px">
        <el-form-item label="选择用户" prop="userIds" :rules="[{ required: true, message: '请选择用户', trigger: 'change' }]">
          <el-select v-model="addMemberForm.userIds" multiple filterable placeholder="选择用户" style="width: 100%">
            <el-option v-for="user in userList" :key="user.id" :label="user.username + ' (' + user.nickname + ')'" :value="user.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="addMemberDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleAddMembers" :loading="submitLoading">确定</el-button>
      </span>
    </el-dialog>

    <!-- ==================== 用户组权限管理对话框 ==================== -->
    <el-dialog title="用户组权限管理" v-model="groupPermDialogVisible" width="800px" @close="resetGroupPermDialog">
      <div v-if="currentGroup">
        <div style="margin-bottom: 16px;">
          <strong>用户组：</strong>{{ currentGroup.name }}
            <el-button type="primary" size="small" style="float: right;"  @click="showGroupPermCreateDialog">+ 新增权限</el-button>
            <el-button type="success" size="small" style="float: right; margin-right: 8px;"  @click="showGroupPermBatchDialog">批量授权</el-button>
        </div>
        <el-table :data="groupPermList" v-loading="groupPermLoading" class="modern-table">
          <el-table-column label="ID" prop="id" v-if="false" />
          <el-table-column prop="clusterName" label="集群名称" min-width="150" />
          <el-table-column prop="namespace" label="命名空间" min-width="150" />
          <el-table-column prop="permissionType" label="权限类型" width="120" align="center">
            <template #default="scope">
              <el-tag :type="scope.row.permissionType === 'admin' ? 'danger' : scope.row.permissionType === 'write' ? 'warning' : 'info'" size="small">
                {{ scope.row.permissionType === 'admin' ? '管理员' : scope.row.permissionType === 'write' ? '读写' : '只读' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="createdAt" label="创建时间" width="180" />
          <el-table-column label="操作" width="150" align="center">
            <template #default="scope">
              <div class="operation-buttons">
                <el-tooltip content="编辑" placement="top">
                  <el-button type="warning" size="small"  @click="showGroupPermEditDialog(scope.row)">
                    <el-icon><Edit /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="删除" placement="top">
                  <el-button type="danger" size="small"  @click="handleDeleteGroupPerm(scope.row)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>

    <!-- ==================== 用户组权限创建/编辑对话框 ==================== -->
    <el-dialog :title="groupPermDialogTitle" v-model="groupPermFormVisible" width="500px" @close="resetGroupPermForm">
      <el-form :model="groupPermForm" :rules="groupPermFormRules" ref="groupPermFormRef" label-width="100px">
        <el-form-item label="集群" prop="clusterId">
          <el-select v-model="groupPermForm.clusterId" placeholder="选择集群" filterable style="width: 100%" @change="onGroupPermClusterChange">
            <el-option v-for="cluster in clusterList" :key="cluster.id" :label="cluster.name" :value="cluster.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="命名空间" prop="namespace">
          <el-select v-model="groupPermForm.namespace" placeholder="选择命名空间" filterable allow-create default-first-option style="width: 100%">
            <el-option v-for="ns in dialogNamespaceOptions" :key="ns" :label="ns" :value="ns" />
          </el-select>
          <div style="color: var(--text-secondary); font-size: 12px; margin-top: 4px;">选择命名空间，也可手动输入</div>
        </el-form-item>
        <el-form-item label="权限类型" prop="permissionType">
          <el-select v-model="groupPermForm.permissionType" placeholder="选择权限类型" style="width: 100%">
            <el-option label="只读 (readonly)" value="readonly" />
            <el-option label="读写 (write)" value="write" />
            <el-option label="管理员 (admin)" value="admin" />
          </el-select>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="groupPermFormVisible = false">取消</el-button>
        <el-button type="primary" @click="handleGroupPermSubmit" :loading="submitLoading">确定</el-button>
      </span>
    </el-dialog>

    <!-- ==================== 用户组批量授权对话框 ==================== -->
    <el-dialog title="用户组批量授权" v-model="groupPermBatchVisible" width="550px" @close="resetGroupPermBatchForm">
      <el-form :model="groupPermBatchForm" :rules="groupPermBatchRules" ref="groupPermBatchRef" label-width="100px">
        <el-form-item label="集群" prop="clusterId">
          <el-select v-model="groupPermBatchForm.clusterId" placeholder="选择集群" filterable style="width: 100%" @change="onGroupPermBatchClusterChange">
            <el-option v-for="cluster in clusterList" :key="cluster.id" :label="cluster.name" :value="cluster.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="命名空间" prop="namespaces">
          <el-select v-model="groupPermBatchForm.namespaces" multiple filterable allow-create default-first-option placeholder="选择或输入命名空间" style="width: 100%">
            <el-option v-for="ns in dialogNamespaceOptions" :key="ns" :label="ns" :value="ns" />
          </el-select>
          <div style="color: var(--text-secondary); font-size: 12px; margin-top: 4px;">支持手动输入多个命名空间，按回车确认</div>
        </el-form-item>
        <el-form-item label="权限类型" prop="permissionType">
          <el-select v-model="groupPermBatchForm.permissionType" placeholder="选择权限类型" style="width: 100%">
            <el-option label="只读 (readonly)" value="readonly" />
            <el-option label="读写 (write)" value="write" />
            <el-option label="管理员 (admin)" value="admin" />
          </el-select>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="groupPermBatchVisible = false">取消</el-button>
        <el-button type="primary" @click="handleGroupPermBatchSubmit" :loading="submitLoading">确定</el-button>
      </span>
    </el-dialog>

    <!-- ==================== RBAC 角色创建/编辑对话框 ==================== -->
    <el-dialog :title="rbacRoleDialogTitle" v-model="rbacRoleDialogVisible" width="750px" @close="resetRbacRoleDialog">
      <el-form :model="rbacRoleForm" :rules="rbacRoleFormRules" ref="rbacRoleFormRef" label-width="100px">
        <el-form-item label="集群" prop="clusterId">
          <el-select v-model="rbacRoleForm.clusterId" placeholder="选择集群" filterable style="width: 100%" @change="onRbacRoleClusterChange">
            <el-option v-for="c in clusterList" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="命名空间" prop="namespace">
          <el-select v-model="rbacRoleForm.namespace" placeholder="留空=集群级别" filterable allow-create clearable default-first-option style="width:100%">
            <el-option v-for="ns in dialogNamespaceOptions" :key="ns" :label="ns" :value="ns" />
          </el-select>
          <div style="color:var(--text-secondary);font-size:12px;margin-top:2px">留空 = ClusterRole（集群级别），选择或输入 = Role（命名空间级别）</div>
        </el-form-item>
        <el-form-item label="角色名称" prop="name">
          <el-input v-model="rbacRoleForm.name" placeholder="例如: pod-reader, deployment-admin" />
        </el-form-item>
        <el-form-item label="权限规则">
          <div style="width:100%">
            <div v-for="(rule, idx) in rbacRoleForm.rules" :key="idx" class="rbac-rule-card">
              <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:8px">
                <strong>规则 #{{ idx+1 }}</strong>
                <el-button type="danger" size="small" @click="removeRbacRule(idx)" v-if="rbacRoleForm.rules.length>1">移除</el-button>
              </div>
              <el-form :model="rule" label-width="60px" size="small">
                <el-form-item label="API组">
                  <el-select v-model="rule.apiGroups" multiple filterable allow-create default-first-option placeholder="例如: apps, batch, '' (核心)" style="width:100%">
                    <el-option label="空(核心API)" value="" />
                    <el-option label="apps" value="apps" />
                    <el-option label="batch" value="batch" />
                    <el-option label="extensions" value="extensions" />
                    <el-option label="networking.k8s.io" value="networking.k8s.io" />
                    <el-option label="storage.k8s.io" value="storage.k8s.io" />
                    <el-option label="rbac.authorization.k8s.io" value="rbac.authorization.k8s.io" />
                    <el-option label="* (所有)" value="*" />
                  </el-select>
                </el-form-item>
                <el-form-item label="资源">
                  <el-select v-model="rule.resources" multiple filterable allow-create default-first-option placeholder="例如: pods, deployments, services" style="width:100%">
                    <el-option label="* (所有资源)" value="*" />
                    <el-option label="pods" value="pods" />
                    <el-option label="deployments" value="deployments" />
                    <el-option label="services" value="services" />
                    <el-option label="ingresses" value="ingresses" />
                    <el-option label="configmaps" value="configmaps" />
                    <el-option label="secrets" value="secrets" />
                    <el-option label="pvc" value="persistentvolumeclaims" />
                    <el-option label="pv" value="persistentvolumes" />
                    <el-option label="nodes" value="nodes" />
                    <el-option label="namespaces" value="namespaces" />
                    <el-option label="events" value="events" />
                    <el-option label="endpoints" value="endpoints" />
                  </el-select>
                </el-form-item>
                <el-form-item label="操作(verbs)">
                  <el-select v-model="rule.verbs" multiple filterable allow-create default-first-option placeholder="例如: get, list, watch" style="width:100%">
                    <el-option label="* (所有操作)" value="*" />
                    <el-option label="get" value="get" />
                    <el-option label="list" value="list" />
                    <el-option label="watch" value="watch" />
                    <el-option label="create" value="create" />
                    <el-option label="update" value="update" />
                    <el-option label="patch" value="patch" />
                    <el-option label="delete" value="delete" />
                    <el-option label="deletecollection" value="deletecollection" />
                  </el-select>
                </el-form-item>
              </el-form>
            </div>
            <el-button type="primary" size="small" @click="addRbacRule" style="margin-top:8px">+ 添加规则</el-button>
          </div>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="rbacRoleDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleRbacRoleSubmit" :loading="submitLoading">确定</el-button>
      </span>
    </el-dialog>

    <!-- ==================== RBAC 绑定创建/编辑对话框 ==================== -->
    <el-dialog :title="rbacBindingDialogTitle" v-model="rbacBindingDialogVisible" width="550px" @close="resetRbacBindingDialog">
      <el-form :model="rbacBindingForm" :rules="rbacBindingFormRules" ref="rbacBindingFormRef" label-width="100px">
        <el-form-item label="集群" prop="clusterId">
          <el-select v-model="rbacBindingForm.clusterId" placeholder="选择集群" filterable style="width:100%" @change="onRbacBindingClusterChange">
            <el-option v-for="c in clusterList" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="命名空间" prop="namespace">
          <el-select v-model="rbacBindingForm.namespace" placeholder="留空=集群级别绑定" filterable allow-create clearable default-first-option style="width: 100%" @change="updateRbacRoleOptions">
            <el-option v-for="ns in dialogNamespaceOptions" :key="ns" :label="ns" :value="ns" />
          </el-select>
          <div style="color: var(--text-secondary); font-size: 12px; margin-top: 2px;">选择命名空间(选填)，也可手动输入</div>
        </el-form-item>
        <el-form-item label="角色" prop="roleId">
          <el-select v-model="rbacBindingForm.roleId" placeholder="选择RBAC角色" filterable style="width:100%">
            <el-option v-for="r in rbacRoleOptions" :key="r.id" :label="`${r.name} (${r.namespace||'集群级'})`" :value="r.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="主体类型" prop="subjectType">
          <el-radio-group v-model="rbacBindingForm.subjectType">
            <el-radio label="User">用户</el-radio>
            <el-radio label="Group">用户组</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="主体" prop="subjectId">
          <el-select v-model="rbacBindingForm.subjectId" placeholder="选择用户或用户组" filterable style="width:100%">
            <el-option v-if="rbacBindingForm.subjectType==='User'" v-for="u in userList" :key="u.id" :label="`${u.username} (${u.nickname})`" :value="u.id" />
            <el-option v-if="rbacBindingForm.subjectType==='Group'" v-for="g in groupListForBinding" :key="g.id" :label="g.name" :value="g.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="rbacBindingDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleRbacBindingSubmit" :loading="submitLoading">确定</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import k8sApi from '@/api/k8s'
import systemApi from '@/api/system'

export default {
  name: 'K8sPermission',
  data() {
    return {
      // 标签页
      activeTab: 'userPermission',

      // ===== 命名空间选项（从集群动态获取）=====
      filterNamespaceOptions: [],
      dialogNamespaceOptions: [],
      rbacRoleFilterNsOptions: [],
      rbacBindingFilterNsOptions: [],

      // ===== 用户权限相关 =====
      loading: false,
      permissionList: [],
      total: 0,
      userList: [],
      clusterList: [],
      queryParams: {
        userId: undefined,
        clusterId: undefined,
        namespace: '',
        page: 1,
        size: 10
      },
      dialogVisible: false,
      dialogTitle: '新增权限',
      isEdit: false,
      editId: null,
      formData: {
        userId: undefined,
        clusterId: undefined,
        namespace: '',
        permissionType: 'readonly'
      },
      formRules: {
        userId: [{ required: true, message: '请选择用户', trigger: 'change' }],
        clusterId: [{ required: true, message: '请选择集群', trigger: 'change' }],
        namespace: [{ required: true, message: '请输入命名空间', trigger: 'blur' }],
        permissionType: [{ required: true, message: '请选择权限类型', trigger: 'change' }]
      },
      batchDialogVisible: false,
      batchFormData: {
        userId: undefined,
        clusterId: undefined,
        namespaces: [],
        permissionType: 'readonly'
      },
      batchFormRules: {
        userId: [{ required: true, message: '请选择用户', trigger: 'change' }],
        clusterId: [{ required: true, message: '请选择集群', trigger: 'change' }],
        namespaces: [{ required: true, message: '请选择或输入命名空间', trigger: 'change' }],
        permissionType: [{ required: true, message: '请选择权限类型', trigger: 'change' }]
      },

      // ===== 用户组相关 =====
      groupLoading: false,
      groupList: [],
      groupTotal: 0,
      groupQuery: {
        page: 1,
        size: 10,
        name: ''
      },
      groupDialogVisible: false,
      groupDialogTitle: '新增用户组',
      isGroupEdit: false,
      groupEditId: null,
      groupFormData: {
        name: '',
        code: '',
        description: '',
        status: 1
      },
      groupFormRules: {
        name: [{ required: true, message: '请输入用户组名称', trigger: 'blur' }]
      },

      // ===== 用户组成员相关 =====
      memberDialogVisible: false,
      memberLoading: false,
      memberList: [],
      currentGroup: null,
      addMemberDialogVisible: false,
      addMemberForm: {
        userIds: []
      },

      // ===== 用户组权限相关 =====
      groupPermDialogVisible: false,
      groupPermLoading: false,
      groupPermList: [],
      groupPermFormVisible: false,
      groupPermDialogTitle: '新增权限',
      isGroupPermEdit: false,
      groupPermEditId: null,
      groupPermForm: {
        clusterId: undefined,
        namespace: '',
        permissionType: 'readonly'
      },
      groupPermFormRules: {
        clusterId: [{ required: true, message: '请选择集群', trigger: 'change' }],
        namespace: [{ required: true, message: '请输入命名空间', trigger: 'blur' }],
        permissionType: [{ required: true, message: '请选择权限类型', trigger: 'change' }]
      },
      groupPermBatchVisible: false,
      groupPermBatchForm: {
        clusterId: undefined,
        namespaces: [],
        permissionType: 'readonly'
      },
      groupPermBatchRules: {
        clusterId: [{ required: true, message: '请选择集群', trigger: 'change' }],
        namespaces: [{ required: true, message: '请选择或输入命名空间', trigger: 'change' }],
        permissionType: [{ required: true, message: '请选择权限类型', trigger: 'change' }]
      },

      // 通用
      submitLoading: false,

      // ===== RBAC 角色管理 =====
      rbacRoleLoading: false,
      rbacRoleList: [],
      rbacRoleTotal: 0,
      rbacRoleQuery: {
        clusterId: undefined,
        namespace: '',
        name: '',
        page: 1,
        size: 10
      },
      rbacRoleDialogVisible: false,
      rbacRoleDialogTitle: '新增 RBAC 角色',
      isRbacRoleEdit: false,
      rbacRoleEditId: null,
      rbacRoleForm: {
        clusterId: undefined,
        namespace: '',
        name: '',
        rules: [{ apiGroups: [''], resources: ['*'], verbs: ['get', 'list', 'watch'] }]
      },
      rbacRoleFormRules: {
        clusterId: [{ required: true, message: '请选择集群', trigger: 'change' }],
        name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }]
      },

      // ===== RBAC 绑定管理 =====
      rbacBindingLoading: false,
      rbacBindingList: [],
      rbacBindingTotal: 0,
      rbacBindingQuery: {
        clusterId: undefined,
        namespace: '',
        subjectType: '',
        subjectName: '',
        page: 1,
        size: 10
      },
      rbacBindingDialogVisible: false,
      rbacBindingDialogTitle: '新增 RBAC 绑定',
      isRbacBindingEdit: false,
      rbacBindingEditId: null,
      rbacBindingForm: {
        clusterId: undefined,
        namespace: '',
        roleId: undefined,
        subjectType: 'User',
        subjectId: undefined
      },
      rbacBindingFormRules: {
        clusterId: [{ required: true, message: '请选择集群', trigger: 'change' }],
        roleId: [{ required: true, message: '请选择角色', trigger: 'change' }],
        subjectType: [{ required: true, message: '请选择主体类型', trigger: 'change' }],
        subjectId: [{ required: true, message: '请选择主体', trigger: 'change' }]
      },
      rbacRoleOptions: [],
      groupListForBinding: [],

      // ===== 我的权限 =====
      myPermLoading: false,
      myPermList: [],
      myPermPage: 1,
      myPermPageSize: 10
    }
  },
  created() {
    this.fetchUserList()
    this.fetchClusterList()
    this.fetchPermissionList()
  },
  watch: {
    // 标签页切换时响应式触发数据加载
    activeTab(newTab) {
      if (newTab === 'userGroup') {
        this.fetchGroupList()
      } else if (newTab === 'rbacRole') {
        this.fetchRbacRoleList()
      } else if (newTab === 'rbacBinding') {
        this.fetchRbacBindingList()
      } else if (newTab === 'myPermissions') {
        this.fetchMyPermissions()
      }
    }
  },
  methods: {
    // ==================== 标签页切换 ====================
    handleTabClick(tab) {
      // 主动触发watch activeTab
      // 重置对应标签的页码到第一页
      if (tab.name === 'userGroup') {
        this.groupQuery.page = 1
      } else if (tab.name === 'rbacRole') {
        this.rbacRoleQuery.page = 1
      } else if (tab.name === 'rbacBinding') {
        this.rbacBindingQuery.page = 1
      }
    },

    // ==================== 命名空间选项 ====================

    // 获取指定集群的命名空间列表
    async fetchNamespaceOptions(clusterId) {
      if (!clusterId) return []
      try {
        const res = await k8sApi.getNamespaces(clusterId)
        const data = res.data
        if (data.code === 200) {
          const nsData = data.data || data
          const namespaces = nsData.namespaces || []
          return namespaces.map(ns => ns.name || ns).filter(Boolean)
        }
      } catch (err) {
        console.error('获取命名空间列表失败:', err)
      }
      return []
    },

    // 搜索区域集群变更时更新命名空间下拉
    async onClusterChangeForFilter(clusterId) {
      this.queryParams.namespace = ''
      this.filterNamespaceOptions = await this.fetchNamespaceOptions(clusterId)
    },

    // 对话框集群变更时更新命名空间下拉
    async onDialogClusterChange(clusterId) {
      this.formData.namespace = ''
      this.dialogNamespaceOptions = await this.fetchNamespaceOptions(clusterId)
    },

    // 批量授权对话框集群变更
    async onBatchClusterChange(clusterId) {
      this.batchFormData.namespaces = []
      this.dialogNamespaceOptions = await this.fetchNamespaceOptions(clusterId)
    },

    // 用户组权限对话框集群变更
    async onGroupPermClusterChange(clusterId) {
      this.groupPermForm.namespace = ''
      this.dialogNamespaceOptions = await this.fetchNamespaceOptions(clusterId)
    },

    // 用户组批量授权集群变更
    async onGroupPermBatchClusterChange(clusterId) {
      this.groupPermBatchForm.namespaces = []
      this.dialogNamespaceOptions = await this.fetchNamespaceOptions(clusterId)
    },

    // ==================== 用户权限相关 ====================

    // 获取权限列表
    async fetchPermissionList() {
      this.loading = true
      try {
        const res = await k8sApi.getPermissionList(this.queryParams)
        const data = res.data
        if (data.code === 200) {
          const result = data.data || data
          this.permissionList = result.list || []
          this.total = result.total || 0
        } else {
          this.$message.error(data.message || '获取权限列表失败')
        }
      } catch (err) {
        this.$message.error('获取权限列表失败: ' + (err.msg || err.message))
      } finally {
        this.loading = false
      }
    },

    // 获取用户列表
    async fetchUserList() {
      try {
        const res = await systemApi.queryAdminList({ page: 1, size: 1000 })
        const data = res.data
        if (data.code === 200) {
          const result = data.data || data
          this.userList = result.list || []
        }
      } catch (err) {
        console.error('获取用户列表失败:', err)
      }
    },

    // 获取集群列表
    async fetchClusterList() {
      try {
        const res = await k8sApi.getClusterList({ page: 1, size: 1000 })
        const data = res.data
        if (data.code === 200) {
          const result = data.data || data
          this.clusterList = result.list || []
        }
      } catch (err) {
        console.error('获取集群列表失败:', err)
      }
    },

    // 搜索
    handleSearch() {
      this.queryParams.page = 1
      this.fetchPermissionList()
    },

    // 重置搜索
    handleReset() {
      this.queryParams = {
        userId: undefined,
        clusterId: undefined,
        namespace: '',
        page: 1,
        size: 10
      }
      this.fetchPermissionList()
    },

    // 分页
    handleSizeChange(size) {
      this.queryParams.size = size
      this.fetchPermissionList()
    },
    handleCurrentChange(page) {
      this.queryParams.page = page
      this.fetchPermissionList()
    },

    // 显示创建对话框
    showCreateDialog() {
      this.isEdit = false
      this.editId = null
      this.dialogTitle = '新增权限'
      this.formData = {
        userId: undefined,
        clusterId: undefined,
        namespace: '',
        permissionType: 'readonly'
      }
      this.dialogNamespaceOptions = []
      this.dialogVisible = true
    },

    // 显示编辑对话框
    showEditDialog(row) {
      this.isEdit = true
      this.editId = row.id
      this.dialogTitle = '编辑权限'
      this.formData = {
        userId: row.userId,
        clusterId: row.clusterId,
        namespace: row.namespace,
        permissionType: row.permissionType
      }
      this.dialogVisible = true
    },

    // 重置对话框
    resetDialog() {
      this.$refs.formRef && this.$refs.formRef.resetFields()
    },

    // 提交表单
    handleSubmit() {
      this.$refs.formRef.validate(async (valid) => {
        if (!valid) return
        this.submitLoading = true
        try {
          let res
          if (this.isEdit) {
            res = await k8sApi.updatePermission(this.editId, {
              permissionType: this.formData.permissionType
            })
          } else {
            res = await k8sApi.createPermission(this.formData)
          }
          const data = res.data
          if (data.code === 200) {
            this.$message.success(this.isEdit ? '更新成功' : '创建成功')
            this.dialogVisible = false
            this.fetchPermissionList()
          } else {
            this.$message.error(data.message || '操作失败')
          }
        } catch (err) {
          const errMsg = err.response?.data?.message || err.message || '操作失败'
          this.$message.error('操作失败: ' + errMsg)
        } finally {
          this.submitLoading = false
        }
      })
    },

    // 删除权限
    handleDelete(row) {
      this.$confirm(`确定删除用户 "${row.username}" 在集群 "${row.clusterName}" 的命名空间 "${row.namespace}" 权限吗？`, '确认删除', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(async () => {
        try {
          const res = await k8sApi.deletePermission(row.id)
          const data = res.data
          if (data.code === 200) {
            this.$message.success('删除成功')
            this.fetchPermissionList()
          } else {
            this.$message.error(data.message || '删除失败')
          }
        } catch (err) {
          this.$message.error('删除失败: ' + (err.response?.data?.message || err.message))
        }
      }).catch(() => {})
    },

    // 显示批量授权对话框
    showBatchCreateDialog() {
      this.batchFormData = {
        userId: undefined,
        clusterId: undefined,
        namespaces: [],
        permissionType: 'readonly'
      }
      this.dialogNamespaceOptions = []
      this.batchDialogVisible = true
    },

    // 重置批量对话框
    resetBatchDialog() {
      this.$refs.batchFormRef && this.$refs.batchFormRef.resetFields()
    },

    // 批量提交
    handleBatchSubmit() {
      this.$refs.batchFormRef.validate(async (valid) => {
        if (!valid) return
        this.submitLoading = true
        try {
          const res = await k8sApi.batchCreatePermission(this.batchFormData)
          const data = res.data
          if (data.code === 200) {
            this.$message.success('批量授权成功')
            this.batchDialogVisible = false
            this.fetchPermissionList()
          } else {
            this.$message.error(data.message || '批量授权失败')
          }
        } catch (err) {
          this.$message.error('批量授权失败: ' + (err.response?.data?.message || err.message))
        } finally {
          this.submitLoading = false
        }
      })
    },

    // ==================== 用户组管理相关 ====================

    // 获取用户组列表
    async fetchGroupList() {
      this.groupLoading = true
      try {
        const res = await k8sApi.getUserGroupList(this.groupQuery)
        const data = res.data
        if (data.code === 200) {
          const result = data.data || data
          this.groupList = result.list || []
          this.groupTotal = result.total || 0
        } else {
          this.$message.error(data.message || '获取用户组列表失败')
        }
      } catch (err) {
        this.$message.error('获取用户组列表失败: ' + (err.msg || err.message))
      } finally {
        this.groupLoading = false
      }
    },

    // 用户组名称搜索
    handleGroupSearch() {
      this.groupQuery.page = 1
      this.fetchGroupList()
    },

    // 用户组分页
    handleGroupSizeChange(size) {
      this.groupQuery.size = size
      this.fetchGroupList()
    },
    handleGroupCurrentChange(page) {
      this.groupQuery.page = page
      this.fetchGroupList()
    },

    // 显示创建用户组对话框
    showGroupCreateDialog() {
      this.isGroupEdit = false
      this.groupEditId = null
      this.groupDialogTitle = '新增用户组'
      this.groupFormData = {
        name: '',
        code: '',
        description: '',
        status: 1
      }
      this.groupDialogVisible = true
    },

    // 显示编辑用户组对话框
    showGroupEditDialog(row) {
      this.isGroupEdit = true
      this.groupEditId = row.id
      this.groupDialogTitle = '编辑用户组'
      this.groupFormData = {
        name: row.name,
        code: row.code || '',
        description: row.description || '',
        status: row.status
      }
      this.groupDialogVisible = true
    },

    // 重置用户组对话框
    resetGroupDialog() {
      this.$refs.groupFormRef && this.$refs.groupFormRef.resetFields()
    },

    // 提交用户组表单
    handleGroupSubmit() {
      this.$refs.groupFormRef.validate(async (valid) => {
        if (!valid) return
        this.submitLoading = true
        try {
          let res
          if (this.isGroupEdit) {
            res = await k8sApi.updateUserGroup(this.groupEditId, this.groupFormData)
          } else {
            res = await k8sApi.createUserGroup(this.groupFormData)
          }
          const data = res.data
          if (data.code === 200) {
            this.$message.success(this.isGroupEdit ? '更新成功' : '创建成功')
            this.groupDialogVisible = false
            this.fetchGroupList()
          } else {
            this.$message.error(data.message || '操作失败')
          }
        } catch (err) {
          this.$message.error('操作失败: ' + (err.response?.data?.message || err.message))
        } finally {
          this.submitLoading = false
        }
      })
    },

    // 删除用户组
    handleDeleteGroup(row) {
      this.$confirm(`确定删除用户组 "${row.name}" 吗？`, '确认删除', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(async () => {
        try {
          const res = await k8sApi.deleteUserGroup(row.id)
          const data = res.data
          if (data.code === 200) {
            this.$message.success('删除成功')
            this.fetchGroupList()
          } else {
            this.$message.error(data.message || '删除失败')
          }
        } catch (err) {
          this.$message.error('删除失败: ' + (err.response?.data?.message || err.message))
        }
      }).catch(() => {})
    },

    // ==================== 用户组成员管理 ====================
    showGroupMembersDialog(row) {
      this.currentGroup = row
      this.memberDialogVisible = true
      this.fetchGroupMembers()
    },
    resetMemberDialog() {
      this.currentGroup = null
      this.memberList = []
    },
    async fetchGroupMembers() {
      if (!this.currentGroup) return
      this.memberLoading = true
      try {
        const res = await k8sApi.getGroupMembers(this.currentGroup.id)
        if (res.data && res.data.code === 200) {
          const result = res.data.data || res.data
          this.memberList = result.list || Object.values(result) || []
          if (!Array.isArray(this.memberList) && result.users) {
            this.memberList = result.users
          }
        } else {
          this.$message.error(res.data?.message || '获取成员失败')
        }
      } catch (err) {
        this.$message.error('获取成员失败: ' + (err.response?.data?.message || err.message))
      } finally {
        this.memberLoading = false
      }
    },
    showAddMemberDialog() {
      this.addMemberForm.userIds = []
      this.addMemberDialogVisible = true
    },
    async handleAddMembers() {
      if (!this.addMemberForm.userIds || this.addMemberForm.userIds.length === 0) {
        this.$message.warning('请选择用户')
        return
      }
      this.submitLoading = true
      try {
        const res = await k8sApi.addGroupMembers({
          groupId: this.currentGroup.id,
          userIds: this.addMemberForm.userIds
        })
        if (res.data && res.data.code === 200) {
          this.$message.success('添加成功')
          this.addMemberDialogVisible = false
          this.fetchGroupMembers()
        } else {
          this.$message.error(res.data?.message || '添加失败')
        }
      } catch (err) {
        this.$message.error('添加失败: ' + (err.response?.data?.message || err.message))
      } finally {
        this.submitLoading = false
      }
    },
    handleRemoveMember(row) {
      this.$confirm(`确定移除成员 "${row.username || row.nickname}" 吗？`, '确认移除', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(async () => {
        try {
          const res = await k8sApi.removeGroupMember({
            groupId: this.currentGroup.id,
            userId: row.userId
          })
          if (res.data && res.data.code === 200) {
            this.$message.success('移除成功')
            this.fetchGroupMembers()
          } else {
            this.$message.error(res.data?.message || '移除失败')
          }
        } catch (err) {
          this.$message.error('移除失败: ' + (err.response?.data?.message || err.message))
        }
      }).catch(() => {})
    },

    // ==================== 用户组权限管理 ====================
    showGroupPermissionsDialog(row) {
      this.currentGroup = row
      this.groupPermDialogVisible = true
      this.fetchGroupPermList()
    },
    resetGroupPermDialog() {
      this.currentGroup = null
      this.groupPermList = []
    },
    async fetchGroupPermList() {
      if (!this.currentGroup) return
      this.groupPermLoading = true
      try {
        const res = await k8sApi.getGroupPermissionList({ groupId: this.currentGroup.id, page: 1, size: 100 })
        if (res.data && res.data.code === 200) {
          const result = res.data.data || res.data
          this.groupPermList = result.list || Object.values(result) || []
          if (!Array.isArray(this.groupPermList) && result.permissions) {
            this.groupPermList = result.permissions
          }
        } else {
          this.$message.error(res.data?.message || '获取组权限失败')
        }
      } catch (err) {
        this.$message.error('获取组权限失败: ' + (err.response?.data?.message || err.message))
      } finally {
        this.groupPermLoading = false
      }
    },
    showGroupPermCreateDialog() {
      this.isGroupPermEdit = false
      this.groupPermEditId = null
      this.groupPermDialogTitle = '新增组权限'
      this.groupPermForm = {
        clusterId: undefined,
        namespace: '',
        permissionType: 'readonly'
      }
      this.dialogNamespaceOptions = []
      this.groupPermFormVisible = true
    },
    showGroupPermEditDialog(row) {
      this.isGroupPermEdit = true
      this.groupPermEditId = row.id
      this.groupPermDialogTitle = '编辑组权限'
      this.groupPermForm = {
        clusterId: row.clusterId,
        namespace: row.namespace,
        permissionType: row.permissionType
      }
      this.groupPermFormVisible = true
    },
    resetGroupPermForm() {
      this.$refs.groupPermFormRef && this.$refs.groupPermFormRef.resetFields()
    },
    handleGroupPermSubmit() {
      this.$refs.groupPermFormRef.validate(async (valid) => {
        if (!valid) return
        this.submitLoading = true
        try {
          let res
          const payload = { ...this.groupPermForm, groupId: this.currentGroup.id }
          if (this.isGroupPermEdit) {
            res = await k8sApi.updateGroupPermission(this.groupPermEditId, payload)
          } else {
            res = await k8sApi.createGroupPermission(payload)
          }
          if (res.data && res.data.code === 200) {
            this.$message.success(this.isGroupPermEdit ? '更新成功' : '创建成功')
            this.groupPermFormVisible = false
            this.fetchGroupPermList()
          } else {
            this.$message.error(res.data?.message || '操作失败')
          }
        } catch (err) {
          this.$message.error('操作失败: ' + (err.response?.data?.message || err.message))
        } finally {
          this.submitLoading = false
        }
      })
    },
    showGroupPermBatchDialog() {
      this.groupPermBatchForm = {
        clusterId: undefined,
        namespaces: [],
        permissionType: 'readonly'
      }
      this.dialogNamespaceOptions = []
      this.groupPermBatchVisible = true
    },
    resetGroupPermBatchForm() {
      this.$refs.groupPermBatchRef && this.$refs.groupPermBatchRef.resetFields()
    },
    handleGroupPermBatchSubmit() {
      this.$refs.groupPermBatchRef.validate(async (valid) => {
        if (!valid) return
        this.submitLoading = true
        try {
          const payload = { ...this.groupPermBatchForm, groupId: this.currentGroup.id }
          const res = await k8sApi.batchCreateGroupPermission(payload)
          if (res.data && res.data.code === 200) {
            this.$message.success('批量授权成功')
            this.groupPermBatchVisible = false
            this.fetchGroupPermList()
          } else {
            this.$message.error(res.data?.message || '批量授权失败')
          }
        } catch (err) {
          this.$message.error('批量授权失败: ' + (err.response?.data?.message || err.message))
        } finally {
          this.submitLoading = false
        }
      })
    },
    handleDeleteGroupPerm(row) {
      this.$confirm(`确定删除命名空间 "${row.namespace}" 的组权限吗？`, '确认删除', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(async () => {
        try {
          const res = await k8sApi.deleteGroupPermission(row.id)
          if (res.data && res.data.code === 200) {
            this.$message.success('删除成功')
            this.fetchGroupPermList()
          } else {
            this.$message.error(res.data?.message || '删除失败')
          }
        } catch (err) {
          this.$message.error('删除失败: ' + (err.response?.data?.message || err.message))
        }
      }).catch(() => {})
    },

    // ==================== RBAC 角色管理 ====================

    // 获取RBAC角色列表
    async fetchRbacRoleList() {
      this.rbacRoleLoading = true
      try {
        const params = { ...this.rbacRoleQuery }
        if (Array.isArray(params.namespace)) params.namespace = params.namespace.join(',')
        const res = await k8sApi.getRbacRoleList(params)
        const data = res.data
        if (data.code === 200) {
          const result = data.data || data
          this.rbacRoleList = result.list || result || []
          this.rbacRoleTotal = result.total || this.rbacRoleList.length || 0
        } else {
          this.$message.error(data.message || '获取RBAC角色列表失败')
        }
      } catch (err) {
        this.$message.error('获取RBAC角色列表失败: ' + (err.msg || err.message))
      } finally {
        this.rbacRoleLoading = false
      }
    },

    // 显示创建RBAC角色对话框
    showRbacRoleCreateDialog() {
      this.isRbacRoleEdit = false
      this.rbacRoleEditId = null
      this.rbacRoleDialogTitle = '新增 RBAC 角色'
      this.rbacRoleForm = {
        clusterId: undefined,
        namespace: '',
        name: '',
        rules: [{ apiGroups: [''], resources: ['*'], verbs: ['get', 'list', 'watch'] }]
      }
      this.dialogNamespaceOptions = []
      this.rbacRoleDialogVisible = true
    },

    // 显示编辑RBAC角色对话框
    
    showRbacRoleEditDialog(row) {
      this.isRbacRoleEdit = true
      this.rbacRoleEditId = row.id
      this.rbacRoleDialogTitle = '编辑 RBAC 角色'
      this.rbacRoleForm = {
        clusterId: row.clusterId,
        namespace: row.namespace || '',
        name: row.name,
        rules: row.rules ? JSON.parse(JSON.stringify(row.rules)) : []
      }
      this.fetchNamespaceOptions(row.clusterId).then(nsList => {
        this.dialogNamespaceOptions = nsList
      })
      this.rbacRoleDialogVisible = true
    },

    // 重置RBAC角色对话框
    resetRbacRoleDialog() {
      this.$refs.rbacRoleFormRef && this.$refs.rbacRoleFormRef.resetFields()
    },

    // 添加RBAC规则
    addRbacRule() {
      this.rbacRoleForm.rules.push({ apiGroups: [''], resources: ['*'], verbs: ['get', 'list', 'watch'] })
    },

    // 移除RBAC规则
    removeRbacRule(idx) {
      this.rbacRoleForm.rules.splice(idx, 1)
    },

    // RBAC角色对话框集群变更
    async onRbacRoleClusterChange(clusterId) {
      this.rbacRoleForm.namespace = ''
      this.dialogNamespaceOptions = await this.fetchNamespaceOptions(clusterId)
    },

    // 获取verb标签类型
    getVerbTagType(verb) {
      const readVerbs = ['get', 'list', 'watch']
      const writeVerbs = ['create', 'update', 'patch', 'delete', 'deletecollection']
      if (verb === '*') return 'danger'
      if (writeVerbs.includes(verb)) return 'warning'
      if (readVerbs.includes(verb)) return 'primary'
      return 'info'
    },

    // 获取verbs的标签类型（取最高级）
    getVerbsTagType(verbs) {
      if (!verbs || verbs.length === 0) return 'info'
      if (verbs.includes('*')) return 'danger'
      if (verbs.some(v => ['create','update','delete','patch','deletecollection'].includes(v))) return 'warning'
      return 'primary'
    },

    // 提交RBAC角色表单
    handleRbacRoleSubmit() {
      this.$refs.rbacRoleFormRef.validate(async (valid) => {
        if (!valid) return
        this.submitLoading = true
        try {
          let res
          const payload = {
            clusterId: Number(this.rbacRoleForm.clusterId),
            namespace: this.rbacRoleForm.namespace || '',
            name: this.rbacRoleForm.name,
            rules: this.rbacRoleForm.rules
          }
          if (this.isRbacRoleEdit) {
            res = await k8sApi.updateRbacRole(this.rbacRoleEditId, payload)
          } else {
            res = await k8sApi.createRbacRole(payload)
          }
          const data = res.data
          if (data.code === 200) {
            this.$message.success(this.isRbacRoleEdit ? '更新成功' : '创建成功')
            this.rbacRoleDialogVisible = false
            this.fetchRbacRoleList()
          } else {
            this.$message.error(data.message || '操作失败')
          }
        } catch (err) {
          this.$message.error('操作失败: ' + (err.response?.data?.message || err.message))
        } finally {
          this.submitLoading = false
        }
      })
    },

    // 删除RBAC角色
    handleDeleteRbacRole(row) {
      this.$confirm(`确定删除 RBAC 角色 "${row.name}" 吗？`, '确认删除', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(async () => {
        try {
          const res = await k8sApi.deleteRbacRole(row.id)
          const data = res.data
          if (data.code === 200) {
            this.$message.success('删除成功')
            this.fetchRbacRoleList()
          } else {
            this.$message.error(data.message || '删除失败')
          }
        } catch (err) {
          this.$message.error('删除失败: ' + (err.response?.data?.message || err.message))
        }
      }).catch(() => {})
    },

    // RBAC角色筛选集群变更
    async onRbacRoleFilterClusterChange(clusterId) {
      this.rbacRoleQuery.namespace = ''
      this.rbacRoleFilterNsOptions = await this.fetchNamespaceOptions(clusterId)
    },

    // RBAC绑定筛选集群变更
    async onRbacBindingFilterClusterChange(clusterId) {
      this.rbacBindingQuery.namespace = ''
      this.rbacBindingFilterNsOptions = await this.fetchNamespaceOptions(clusterId)
    },

    // ==================== RBAC 绑定管理 ====================

    // 获取RBAC绑定列表
    async fetchRbacBindingList() {
      this.rbacBindingLoading = true
      try {
        const params = { ...this.rbacBindingQuery }
        if (Array.isArray(params.namespace)) params.namespace = params.namespace.join(',')
        const res = await k8sApi.getRbacBindingList(params)
        const data = res.data
        if (data.code === 200) {
          const result = data.data || data
          this.rbacBindingList = result.list || result || []
          this.rbacBindingTotal = result.total || this.rbacBindingList.length || 0
        } else {
          this.$message.error(data.message || '获取RBAC绑定列表失败')
        }
      } catch (err) {
        this.$message.error('获取RBAC绑定列表失败: ' + (err.msg || err.message))
      } finally {
        this.rbacBindingLoading = false
      }
    },

    // 显示创建RBAC绑定对话框
    async showRbacBindingCreateDialog() {
      this.isRbacBindingEdit = false
      this.rbacBindingEditId = null
      this.rbacBindingDialogTitle = '新增 RBAC 绑定'
      this.rbacBindingForm = {
        clusterId: undefined,
        namespace: '',
        roleId: undefined,
        subjectType: 'User',
        subjectId: undefined
      }
      this.dialogNamespaceOptions = []
      this.rbacRoleOptions = []
      // 加载用户组选项
      try {
        const groupRes = await k8sApi.getUserGroupList({ page: 1, size: 1000 })
        if (groupRes.data.code === 200) {
          const result = groupRes.data.data || groupRes.data
          this.groupListForBinding = result.list || []
        }
      } catch (_) {}
      this.rbacBindingDialogVisible = true
    },

    // 显示编辑RBAC绑定对话框
    async showRbacBindingEditDialog(row) {
      this.isRbacBindingEdit = true
      this.rbacBindingEditId = row.id
      this.rbacBindingDialogTitle = '编辑 RBAC 绑定'
      this.rbacBindingForm = {
        clusterId: row.clusterId,
        namespace: row.namespace || '',
        roleId: row.roleId,
        subjectType: row.subjectType,
        subjectId: row.subjectId
      }
      this.dialogNamespaceOptions = await this.fetchNamespaceOptions(row.clusterId)
      this.rbacRoleOptions = []
      try {
        const groupRes = await k8sApi.getUserGroupList({ page: 1, size: 1000 })
        if (groupRes.data.code === 200) {
          const result = groupRes.data.data || groupRes.data
          this.groupListForBinding = result.list || []
        }
      } catch (_) {}
      await this.updateRbacRoleOptions()
      this.rbacBindingDialogVisible = true
    },

    // 集群变更时过滤角色和命名空间
    async onRbacBindingClusterChange(clusterId) {
      this.rbacBindingForm.namespace = ''
      this.rbacBindingForm.roleId = undefined
      this.dialogNamespaceOptions = await this.fetchNamespaceOptions(clusterId)
      await this.updateRbacRoleOptions()
    },

    // 更新RBAC角色选项
    async updateRbacRoleOptions() {
      try {
        const params = { clusterId: this.rbacBindingForm.clusterId || undefined, page: 1, size: 1000 }
        const res = await k8sApi.getRbacRoleList(params)
        if (res.data.code === 200) {
          const result = res.data.data || res.data
          this.rbacRoleOptions = result.list || result || []
        }
      } catch (_) {}
    },

    // 重置RBAC绑定对话框
    resetRbacBindingDialog() {
      this.$refs.rbacBindingFormRef && this.$refs.rbacBindingFormRef.resetFields()
    },

    // 提交RBAC绑定
    handleRbacBindingSubmit() {
      this.$refs.rbacBindingFormRef.validate(async (valid) => {
        if (!valid) return
        this.submitLoading = true
        try {
          let res
          if (this.isRbacBindingEdit) {
            res = await k8sApi.updateRbacBinding(this.rbacBindingEditId, this.rbacBindingForm)
          } else {
            res = await k8sApi.createRbacBinding(this.rbacBindingForm)
          }
          const data = res.data
          if (data.code === 200) {
            this.$message.success(this.isRbacBindingEdit ? '绑定更新成功' : '绑定创建成功')
            this.rbacBindingDialogVisible = false
            this.fetchRbacBindingList()
          } else {
            this.$message.error(data.message || '操作失败')
          }
        } catch (err) {
          this.$message.error('操作失败: ' + (err.response?.data?.message || err.message))
        } finally {
          this.submitLoading = false
        }
      })
    },

    // 删除RBAC绑定
    handleDeleteRbacBinding(row) {
      this.$confirm('确定删除此 RBAC 绑定吗？', '确认删除', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(async () => {
        try {
          const res = await k8sApi.deleteRbacBinding(row.id)
          const data = res.data
          if (data.code === 200) {
            this.$message.success('删除成功')
            this.fetchRbacBindingList()
          } else {
            this.$message.error(data.message || '删除失败')
          }
        } catch (err) {
          this.$message.error('删除失败: ' + (err.response?.data?.message || err.message))
        }
      }).catch(() => {})
    },

    // ==================== 我的权限 ====================

    // 获取我的权限
    async fetchMyPermissions() {
      this.myPermLoading = true
      try {
        const res = await k8sApi.getMyPermissions()
        const data = res.data
        if (data.code === 200) {
          this.myPermList = data.data || []
        } else {
          this.$message.error(data.message || '获取权限失败')
        }
      } catch (err) {
        this.$message.error('获取权限失败: ' + (err.msg || err.message))
      } finally {
        this.myPermLoading = false
      }
    }
  }
}
</script>

<style scoped>
.k8s-permission {
  padding: 20px;
  background: var(--bg-card);
  border-radius: 8px;
  min-height: calc(100vh - 120px);
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border);
}
.page-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
}
.header-actions {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
}
.search-area {
  background: var(--bg-card-alt);
  padding: 16px 20px;
  border-radius: 8px;
  margin-bottom: 20px;
  border: 1px solid #e8eaee;
}
.search-area .el-form-item {
  margin-bottom: 0;
}
.pagination-area {
  margin-top: 20px;
  display: flex;
  justify-content: center;
  padding: 12px 0;
}
.el-table {
  border-radius: 6px;
  overflow: hidden;
}
.el-table th.el-table__cell {
  background-color: var(--bg-card-alt);
  font-weight: 600;
  color: var(--text-regular);
}
.operation-buttons {
  display: flex;
  gap: 6px;
  justify-content: center;
}
.rbac-rule-item {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin: 3px 0;
}
.rbac-rule-item .el-tag {
  margin: 1px;
}
.rbac-rule-card {
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 14px;
  margin-bottom: 12px;
  background: #fafafa;
  transition: box-shadow 0.2s;
}
.rbac-rule-card:hover {
  box-shadow: 0 1px 4px rgba(0,0,0,0.08);
}
.no-verbs {
  color: #c0c4cc;
  font-size: 13px;
}
.el-dialog .el-form-item:last-child {
  margin-bottom: 0;
}
.el-tabs__item {
  font-size: 14px;
  font-weight: 500;
}
.el-tabs__item.is-active {
  font-weight: 600;
}
.el-alert {
  border-radius: 6px;
}
</style>