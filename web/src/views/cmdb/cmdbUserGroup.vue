<template>
  <div class="user-group-management">
    <el-card shadow="hover" class="main-card">
      <template v-if="!embedded" #header>
        <div class="card-header">
          <span class="title">{{ $tr("CMDB 用户组管理") }}</span>
          <el-tag type="info" effect="plain">{{ $tr("用于资产授权") }}</el-tag>
        </div>
      </template>

      <div class="action-section">
        <el-button type="primary" size="small" @click="showCreate">{{ $tr("+ 新增用户组") }}</el-button>
      </div>

      <el-table :data="list" v-loading="loading" border stripe style="width:100%">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" :label='$tr("用户组名称")' min-width="160" />
        <el-table-column prop="code" :label='$tr("编码")' width="120" />
        <el-table-column prop="description" :label='$tr("描述")' min-width="200" show-overflow-tooltip />
        <el-table-column prop="memberCount" :label='$tr("成员数")' width="80" align="center" />
        <el-table-column :label='$tr("状态")' width="80" align="center">
          <template #default="{row}">
            <el-tag :type="row.status===1?'success':'info'" size="small">{{ row.status===1?$tr('启用'):$tr('禁用') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" :label='$tr("创建时间")' width="170" />
        <el-table-column :label='$tr("操作")' width="250" align="center" fixed="right">
          <template #default="{row}">
            <el-button type="primary" size="small" @click="showMembers(row)">{{ $tr("成员") }}</el-button>
            <el-button type="warning" size="small" @click="showEdit(row)">{{ $tr("编辑") }}</el-button>
            <el-button type="danger" size="small" @click="handleDelete(row)">{{ $tr("删除") }}</el-button>
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

    <!-- Create/Edit Dialog -->
    <el-dialog :title="$tr(formTitle)" v-model="formVisible" width="500px" @close="resetForm">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item :label='$tr("用户组名称")' prop="name">
          <el-input v-model="form.name" :placeholder='$tr("如: 基础运维组")' />
        </el-form-item>
        <el-form-item :label='$tr("编码")'>
          <el-input v-model="form.code" :placeholder='$tr("如: ops-basic")' />
        </el-form-item>
        <el-form-item :label='$tr("描述")'>
          <el-input v-model="form.description" type="textarea" :rows="2" :placeholder='$tr("描述")' />
        </el-form-item>
        <el-form-item :label='$tr("状态")' v-if="isEdit">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible=false">{{ $tr("取消") }}</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">{{ $tr("确定") }}</el-button>
      </template>
    </el-dialog>

    <!-- Member Dialog -->
    <el-dialog :title='$tr("组成员管理")' v-model="memberVisible" width="600px">
      <div style="margin-bottom:16px">
        <el-button type="primary" size="small" @click="showAddMember">{{ $tr("+ 添加成员") }}</el-button>
      </div>
      <el-table :data="members" border stripe size="small" style="width:100%">
        <el-table-column prop="userId" :label='$tr("用户ID")' width="80" />
        <el-table-column prop="username" :label='$tr("用户名")' width="120" />
        <el-table-column prop="nickname" :label='$tr("昵称")' min-width="150" />
        <el-table-column prop="createdAt" :label='$tr("添加时间")' width="170" />
        <el-table-column :label='$tr("操作")' width="80" align="center">
          <template #default="{row}">
            <el-button type="danger" size="small" @click="handleRemoveMember(row)">{{ $tr("移除") }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- Add Member Dialog -->
    <el-dialog :title='$tr("添加成员")' v-model="addMemberVisible" width="500px">
      <el-form label-width="80px">
        <el-form-item :label='$tr("选择用户")'>
          <el-select v-model="addMemberUserIds" multiple filterable :placeholder='$tr("搜索用户")' style="width:100%">
            <el-option v-for="u in userList" :key="u.id" :label="`${u.username||''} (${u.nickname||''})`" :value="u.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addMemberVisible=false">{{ $tr("取消") }}</el-button>
        <el-button type="primary" @click="handleAddMembers" :loading="submitting">{{ $tr("确定") }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { tr } from "@/locales";

import cmdbAPI from '@/api/cmdb'
import systemApi from '@/api/system'

export default {
  name: 'CmdbUserGroup',
  props: { embedded: Boolean },
  data() {
    return {
      loading: false, submitting: false,
      list: [], total: 0,
      members: [],
      userList: [],
      query: { name: '', page: 1, size: 10 },
      formVisible: false, formTitle: '新增用户组', isEdit: false, editId: null,
      memberVisible: false, currentGroupId: null,
      addMemberVisible: false, addMemberUserIds: [],
      form: { name: '', code: '', description: '', status: 1 },
      rules: { name: [{ required: true, get message() { return tr('请输入名称'); }, trigger: 'blur' }] }
    }
  },
  created() { this.fetch(); this.loadUsers() },
  methods: {
    async fetch() {
      this.loading = true
      try {
        const res = await cmdbAPI.getCmdbUserGroupList(this.query)
        if (res.data.code === 200) {
          this.list = res.data.data.list || []
          this.total = res.data.data.total || 0
        }
      } catch (e) { this.$message.error(tr('获取用户组列表失败')) } finally { this.loading = false }
    },
    async loadUsers() {
      try {
        const res = await systemApi.queryAdminList({ page: 1, size: 1000 })
        if (res.data.code === 200) this.userList = res.data.data.list || []
      } catch (_) {}
    },
    showCreate() {
      this.isEdit = false; this.editId = null; this.formTitle = '新增用户组'
      this.form = { name: '', code: '', description: '', status: 1 }
      this.formVisible = true
    },
    showEdit(row) {
      this.isEdit = true; this.editId = row.id; this.formTitle = '编辑用户组'
      this.form = { name: row.name, code: row.code || '', description: row.description || '', status: row.status }
      this.formVisible = true
    },
    resetForm() { this.$refs.formRef?.resetFields() },
    handleSubmit() {
      this.$refs.formRef.validate(async v => {
        if (!v) return
        this.submitting = true
        try {
          let res
          if (this.isEdit) res = await cmdbAPI.updateCmdbUserGroup(this.editId, this.form)
          else res = await cmdbAPI.createCmdbUserGroup(this.form)
          if (res.data.code === 200) {
            this.$message.success(this.isEdit ? '更新成功' : tr('创建成功'))
            this.formVisible = false; this.fetch()
          } else { this.$message.error(res.data.message) }
        } catch (e) { this.$message.error(tr('操作失败')) } finally { this.submitting = false }
      })
    },
    handleDelete(row) {
      this.$confirm(`确定删除用户组 "${row.name}" 吗？`, tr('确认'), { type: 'warning' }).then(async () => {
        try {
          const res = await cmdbAPI.deleteCmdbUserGroup(row.id)
          if (res.data.code === 200) { this.$message.success(tr('删除成功')); this.fetch() }
          else this.$message.error(res.data.message)
        } catch (e) { this.$message.error(tr('删除失败')) }
      }).catch(() => {})
    },
    async showMembers(row) {
      this.currentGroupId = row.id
      this.memberVisible = true
      try {
        const res = await cmdbAPI.getCmdbUserGroupMembers(row.id)
        if (res.data.code === 200) this.members = res.data.data.list || []
        else this.members = []
      } catch (e) { this.$message.error(tr('获取成员失败')) }
    },
    showAddMember() {
      this.addMemberUserIds = []
      this.addMemberVisible = true
    },
    async handleAddMembers() {
      if (!this.addMemberUserIds.length) { this.$message.warning(tr('请选择用户')); return }
      this.submitting = true
      try {
        const res = await cmdbAPI.addCmdbUserGroupMembers({ groupId: this.currentGroupId, userIds: this.addMemberUserIds })
        if (res.data.code === 200) {
          this.$message.success(tr('添加成功'))
          this.addMemberVisible = false
          this.showMembers({ id: this.currentGroupId })
        } else { this.$message.error(res.data.message) }
      } catch (e) { this.$message.error(tr('添加失败')) } finally { this.submitting = false }
    },
    async handleRemoveMember(row) {
      this.$confirm(`确定移除成员 "${row.username}" 吗？`, tr('确认'), { type: 'warning' }).then(async () => {
        try {
          const res = await cmdbAPI.removeCmdbUserGroupMember({ groupId: this.currentGroupId, userId: row.userId })
          if (res.data.code === 200) {
            this.$message.success(tr('移除成功'))
            this.showMembers({ id: this.currentGroupId })
          } else this.$message.error(res.data.message)
        } catch (e) { this.$message.error(tr('移除失败')) }
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
.user-group-management { padding: 20px; min-height: 100vh; background: var(--bg-page); }
.main-card { border-radius: var(--radius-lg); }
.card-header .title { font-size: 18px; font-weight: 600; color: var(--text-primary); }
.action-section { margin-bottom: 16px; }
.pagination-section { display: flex; justify-content: center; margin-top: 20px; }
</style>
