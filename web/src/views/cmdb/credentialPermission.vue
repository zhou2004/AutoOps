<template>
  <div class="credential-permission">
    <el-card shadow="hover" class="main-card">
      <template #header>
        <div class="card-header">
          <span class="title">凭据授权管理</span>
          <el-tag type="info" effect="plain">控制用户对SSH凭据的使用权限</el-tag>
        </div>
      </template>

      <div class="action-section">
        <el-button type="primary" size="small" @click="showCreate">+ 新增凭据授权</el-button>
      </div>

      <el-table :data="list" v-loading="loading" border stripe style="width:100%">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" label="授权名称" min-width="160" />
        <el-table-column prop="credentialId" label="凭据ID" width="80" />
        <el-table-column label="授权用户" min-width="150">
          <template #default="{row}">
            <el-tag size="small" v-if="row.userIds && row.userIds!='[]'">用户</el-tag>
            <span v-else style="color:#999">-</span>
          </template>
        </el-table-column>
        <el-table-column label="授权用户组" min-width="150">
          <template #default="{row}">
            <el-tag size="small" type="success" v-if="row.groupIds && row.groupIds!='[]'">用户组</el-tag>
            <span v-else style="color:#999">-</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template #default="{row}">
            <el-tag :type="row.isActive===1?'success':'info'" size="small">{{ row.isActive===1?'启用':'禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="170" />
        <el-table-column label="操作" width="160" align="center" fixed="right">
          <template #default="{row}">
            <el-button type="warning" size="small" @click="showEdit(row)">编辑</el-button>
            <el-button type="danger" size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-section">
        <el-pagination
          @size-change="s=>{query.size=s;fetch()}"
          @current-change="p=>{query.page=p;fetch()}"
          :current-page="query.page"
          :page-sizes="[10,20,50,100]"
          :page-size="query.size"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total" />
      </div>
    </el-card>

    <el-dialog :title="formTitle" v-model="formVisible" width="600px" @close="resetForm">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="规则名称" prop="name">
          <el-input v-model="form.name" placeholder="如: 基础运维-凭据权限" />
        </el-form-item>
        <el-form-item label="选择凭据" prop="credentialId">
          <el-select v-model="form.credentialId" filterable placeholder="选择SSH凭据" style="width:100%">
            <el-option v-for="c in credentialList" :key="c.id" :label="`${c.name} (${c.username}@${c.port})`" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="授权用户">
              <el-select v-model="form.userIds" multiple filterable placeholder="选择用户" style="width:100%">
                <el-option v-for="u in userList" :key="u.id" :label="`${u.username||''} (${u.nickname||''})`" :value="u.id" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="授权用户组">
              <el-select v-model="form.groupIds" multiple filterable placeholder="选择用户组" style="width:100%">
                <el-option v-for="g in userGroupList" :key="g.id" :label="g.name" :value="g.id" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="启用状态">
          <el-switch v-model="form.isActive" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible=false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import cmdbAPI from '@/api/cmdb'
import systemApi from '@/api/system'

export default {
  name: 'CredentialPermission',
  data() {
    return {
      loading: false, submitting: false,
      list: [], total: 0,
      credentialList: [], userList: [], userGroupList: [],
      query: { name: '', page: 1, size: 10 },
      formVisible: false, formTitle: '新增凭据授权', isEdit: false, editId: null,
      form: { name: '', credentialId: undefined, userIds: [], groupIds: [], isActive: 1 },
      rules: { name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
               credentialId: [{ required: true, message: '请选择凭据', trigger: 'change' }] }
    }
  },
  created() { this.fetch(); this.loadOptions() },
  methods: {
    async fetch() {
      this.loading = true
      try {
        const res = await cmdbAPI.getCredentialPermissionList(this.query)
        if (res.data.code === 200) {
          this.list = res.data.data.list || []
          this.total = res.data.data.total || 0
        }
      } catch (e) { this.$message.error('获取凭据授权列表失败') } finally { this.loading = false }
    },
    async loadOptions() {
      try {
        const [res1, res2, res3] = await Promise.all([
          cmdbAPI.getCredentialList({ page: 1, pageSize: 1000 }),
          systemApi.queryAdminList({ page: 1, size: 1000 }),
          cmdbAPI.getCmdbUserGroupAll()
        ])
        if (res1.data.code === 200) this.credentialList = res1.data.data.list || []
        if (res2.data.code === 200) this.userList = res2.data.data.list || []
        if (res3.data && res3.data.data) this.userGroupList = res3.data.data.list || []
      } catch (_) {}
    },
    search() { this.query.page = 1; this.fetch() },
    reset() { this.query = { name: '', page: 1, size: 10 }; this.fetch() },
    showCreate() {
      this.isEdit = false; this.editId = null; this.formTitle = '新增凭据授权'
      this.form = { name: '', credentialId: undefined, userIds: [], groupIds: [], isActive: 1 }
      this.formVisible = true
    },
    showEdit(row) {
      this.isEdit = true; this.editId = row.id; this.formTitle = '编辑凭据授权'
      this.form = {
        name: row.name,
        credentialId: row.credentialId,
        userIds: this.safeJSON(row.userIds, []),
        groupIds: this.safeJSON(row.groupIds, []),
        isActive: row.isActive
      }
      this.formVisible = true
    },
    safeJSON(str, def) { try { return JSON.parse(str) || def } catch(e) { return def } },
    resetForm() { this.$refs.formRef?.resetFields() },
    async handleSubmit() {
      this.$refs.formRef.validate(async v => {
        if (!v) return
        this.submitting = true
        try {
          let res
          if (this.isEdit) res = await cmdbAPI.updateCredentialPermission(this.editId, this.form)
          else res = await cmdbAPI.createCredentialPermission(this.form)
          if (res.data.code === 200) {
            this.$message.success(this.isEdit ? '更新成功' : '创建成功')
            this.formVisible = false; this.fetch()
          } else { this.$message.error(res.data.message) }
        } catch (e) { this.$message.error('操作失败') } finally { this.submitting = false }
      })
    },
    handleDelete(row) {
      this.$confirm(`确定删除凭据授权 "${row.name}" 吗？`, '确认', { type: 'warning' }).then(async () => {
        try {
          const res = await cmdbAPI.deleteCredentialPermission(row.id)
          if (res.data.code === 200) { this.$message.success('删除成功'); this.fetch() }
          else this.$message.error(res.data.message)
        } catch (e) { this.$message.error('删除失败') }
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
.credential-permission { padding: 20px; min-height: 100vh; background: #f0f2f5; }
.main-card { border-radius: 12px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.card-header .title { font-size: 18px; font-weight: 600; color: #303133; }
.action-section { margin-bottom: 16px; }
.pagination-section { display: flex; justify-content: center; margin-top: 20px; }
</style>
