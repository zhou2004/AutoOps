<template>
  <div class="k8s-permission">
    <div class="page-header">
      <h2>K8s 权限管理</h2>
      <div class="header-actions">
        <el-button type="primary" @click="showCreateDialog">+ 新增权限</el-button>
        <el-button type="success" @click="showBatchCreateDialog">批量授权</el-button>
      </div>
    </div>

    <!-- 搜索区域 -->
    <div class="search-area">
      <el-form :inline="true" :model="queryParams" size="small">
        <el-form-item label="用户">
          <el-select v-model="queryParams.userId" placeholder="选择用户" clearable filterable style="width: 180px">
            <el-option v-for="user in userList" :key="user.id" :label="user.username + ' (' + user.nickname + ')'" :value="user.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="集群">
          <el-select v-model="queryParams.clusterId" placeholder="选择集群" clearable filterable style="width: 180px">
            <el-option v-for="cluster in clusterList" :key="cluster.id" :label="cluster.name" :value="cluster.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="命名空间">
          <el-input v-model="queryParams.namespace" placeholder="命名空间名称" clearable style="width: 180px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">查询</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 权限列表 -->
    <el-table :data="permissionList" v-loading="loading" border stripe style="width: 100%">
      <el-table-column prop="id" label="ID" width="60" align="center" />
      <el-table-column prop="username" label="用户名" min-width="120" />
      <el-table-column prop="nickname" label="昵称" min-width="120" />
      <el-table-column prop="clusterName" label="集群名称" min-width="150" />
      <el-table-column prop="namespace" label="命名空间" min-width="150" />
      <el-table-column prop="permissionType" label="权限类型" width="120" align="center">
        <template slot-scope="scope">
          <el-tag :type="scope.row.permissionType === 'admin' ? 'danger' : scope.row.permissionType === 'write' ? 'warning' : 'info'" size="small">
            {{ scope.row.permissionType === 'admin' ? '管理员' : scope.row.permissionType === 'write' ? '读写' : '只读' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="createdAt" label="创建时间" width="180" />
      <el-table-column label="操作" width="200" align="center" fixed="right">
        <template slot-scope="scope">
          <el-button type="text" size="small" @click="showEditDialog(scope.row)">编辑</el-button>
          <el-button type="text" size="small" style="color: #f56c6c" @click="handleDelete(scope.row)">删除</el-button>
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

    <!-- 创建/编辑权限对话框 -->
    <el-dialog :title="dialogTitle" v-model="dialogVisible" width="500px" @close="resetDialog">
      <el-form :model="formData" :rules="formRules" ref="formRef" label-width="100px">
        <el-form-item label="用户" prop="userId">
          <el-select v-model="formData.userId" placeholder="选择用户" filterable style="width: 100%">
            <el-option v-for="user in userList" :key="user.id" :label="user.username + ' (' + user.nickname + ')'" :value="user.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="集群" prop="clusterId">
          <el-select v-model="formData.clusterId" placeholder="选择集群" filterable style="width: 100%">
            <el-option v-for="cluster in clusterList" :key="cluster.id" :label="cluster.name" :value="cluster.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="命名空间" prop="namespace">
          <el-input v-model="formData.namespace" placeholder="输入命名空间名称" />
        </el-form-item>
        <el-form-item label="权限类型" prop="permissionType">
          <el-select v-model="formData.permissionType" placeholder="选择权限类型" style="width: 100%">
            <el-option label="只读 (readonly)" value="readonly" />
            <el-option label="读写 (write)" value="write" />
            <el-option label="管理员 (admin)" value="admin" />
          </el-select>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitLoading">确定</el-button>
      </span>
    </el-dialog>

    <!-- 批量授权对话框 -->
    <el-dialog title="批量授权" v-model="batchDialogVisible" width="550px" @close="resetBatchDialog">
      <el-form :model="batchFormData" :rules="batchFormRules" ref="batchFormRef" label-width="100px">
        <el-form-item label="用户" prop="userId">
          <el-select v-model="batchFormData.userId" placeholder="选择用户" filterable style="width: 100%">
            <el-option v-for="user in userList" :key="user.id" :label="user.username + ' (' + user.nickname + ')'" :value="user.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="集群" prop="clusterId">
          <el-select v-model="batchFormData.clusterId" placeholder="选择集群" filterable style="width: 100%">
            <el-option v-for="cluster in clusterList" :key="cluster.id" :label="cluster.name" :value="cluster.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="命名空间" prop="namespaces">
          <el-select v-model="batchFormData.namespaces" multiple filterable allow-create default-first-option placeholder="输入或选择命名空间" style="width: 100%">
            <el-option v-for="ns in availableNamespaces" :key="ns" :label="ns" :value="ns" />
          </el-select>
          <div style="color: #909399; font-size: 12px; margin-top: 4px;">支持手动输入多个命名空间，按回车确认</div>
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
  </div>
</template>

<script>
import k8sApi from '@/api/k8s'
import systemApi from '@/api/system'

export default {
  name: 'K8sPermission',
  data() {
    return {
      loading: false,
      submitLoading: false,
      permissionList: [],
      total: 0,
      userList: [],
      clusterList: [],
      availableNamespaces: [],
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
      }
    }
  },
  created() {
    this.fetchPermissionList()
    this.fetchUserList()
    this.fetchClusterList()
  },
  methods: {
    // 获取权限列表
    async fetchPermissionList() {
      this.loading = true
      try {
        const res = await k8sApi.getPermissionList(this.queryParams)
        const data = res.data
        if (data.code === 200) {
          // 根据后端返回结构适配：可能是 {list, total} 或 {data: {list, total}}
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
          // 用户列表在 data.data.list 或 data.list
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
          // 集群列表在 data.data.list 或 data.list
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
      this.availableNamespaces = []
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
    }
  }
}
</script>

<style scoped>
.k8s-permission {
  padding: 20px;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.page-header h2 {
  margin: 0;
  font-size: 20px;
  color: #303133;
}
.header-actions {
  display: flex;
  gap: 10px;
}
.search-area {
  background: #f5f7fa;
  padding: 16px;
  border-radius: 4px;
  margin-bottom: 16px;
}
.pagination-area {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
