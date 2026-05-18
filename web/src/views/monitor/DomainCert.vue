<template>
  <div class="domain-cert-monitor">
    <el-card shadow="hover" class="domain-cert-card">
      <template #header>
        <div class="card-header">
          <span class="title">域名证书监控</span>
        </div>
      </template>
      <div class="search-section" v-show="showSearch">
        <el-form :model="queryParams" :inline="true" class="search-form">
          <el-form-item prop="domain" label="域名">
            <el-input v-model="queryParams.domain" placeholder="请输入域名" clearable size="small"
                      style="width: 200px" @keyup.enter="handleQuery" />
          </el-form-item>
          <el-form-item prop="status" label="状态">
            <el-select v-model="queryParams.status" placeholder="请选择状态" clearable size="small" style="width: 150px">
              <el-option label="正常" :value="1" /><el-option label="即将过期" :value="2" />
              <el-option label="已过期" :value="3" /><el-option label="检查失败" :value="4" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" icon="Search" size="small" @click="handleQuery">搜索</el-button>
            <el-button icon="Refresh" size="small" @click="resetQuery">重置</el-button>
          </el-form-item>
        </el-form>
      </div>
      <div class="action-section">
        <el-button type="primary" icon="Plus" size="small" @click="handleAdd">添加域名</el-button>
        <el-button type="warning" icon="Refresh" size="small" @click="handleCheckAll" :loading="checkingAll">检查全部</el-button>
        <el-button type="danger" plain icon="Delete" size="small" :disabled="multiple" @click="handleBatchDelete">批量删除</el-button>
      </div>
      <div class="table-section">
        <el-table v-loading="Loading" :data="domainCertList" stripe style="width: 100%"
                  class="domain-cert-table" @selection-change="handleSelectionChange">
          <el-table-column type="selection" width="45" />
          <el-table-column label="域名" prop="domain" min-width="180">
            <template #default="scope">
              <el-link type="primary" :underline="false" @click="handleViewDetail(scope.row)">{{ scope.row.domain }}</el-link>
            </template>
          </el-table-column>
          <el-table-column label="端口" prop="port" width="70" />
          <el-table-column label="颁发者" prop="issuer" min-width="150" show-overflow-tooltip />
          <el-table-column label="到期日期" prop="notAfter" min-width="150" />
          <el-table-column label="剩余天数" width="100" align="center">
            <template #default="scope">
              <el-tag v-if="scope.row.remainingDays >= 0" :type="getRemainingDaysType(scope.row.remainingDays)" size="small" effect="dark">{{ scope.row.remainingDays }} 天</el-tag>
              <el-tag v-else type="info" size="small" effect="dark">未知</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100" align="center">
            <template #default="scope">
              <el-tag :type="getStatusType(scope.row.status)" size="small" effect="dark">{{ getStatusText(scope.row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="最近检查" prop="checkTime" min-width="150" />
          <el-table-column label="操作" width="180" fixed="right">
            <template #default="scope">
              <el-button type="primary" icon="Edit" size="small" circle @click="handleEdit(scope.row)" />
              <el-button type="warning" icon="Refresh" size="small" circle @click="handleCheckSingle(scope.row)" :loading="scope.row._checking" />
              <el-button type="danger" icon="Delete" size="small" circle @click="handleDelete(scope.row)" />
            </template>
          </el-table-column>
        </el-table>
      </div>
      <div class="pagination-section">
        <el-pagination @size-change="handleSizeChange" @current-change="handleCurrentChange"
          :current-page="queryParams.page" :page-sizes="[10,20,50,100]" :page-size="queryParams.pageSize"
          layout="total, sizes, prev, pager, next, jumper" :total="total" />
      </div>
    </el-card>

    <!-- 添加/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑域名' : '添加域名'" width="500px" :close-on-click-modal="false">
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="80px">
        <el-form-item label="域名" prop="domain">
          <el-input v-model="form.domain" placeholder="请输入域名，如: example.com" />
        </el-form-item>
        <el-form-item label="端口" prop="port">
          <el-input-number v-model="form.port" :min="1" :max="65535" placeholder="默认443" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitLoading">{{ isEdit ? '保存' : '确定' }}</el-button>
      </template>
    </el-dialog>

    <!-- 证书详情对话框 -->
    <el-dialog v-model="detailVisible" title="证书详情" width="600px">
      <div v-if="detailData" class="cert-detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="域名">{{ detailData.domain }}</el-descriptions-item>
          <el-descriptions-item label="端口">{{ detailData.port }}</el-descriptions-item>
          <el-descriptions-item label="颁发者">{{ detailData.issuer || '-' }}</el-descriptions-item>
          <el-descriptions-item label="主题">{{ detailData.subject || '-' }}</el-descriptions-item>
          <el-descriptions-item label="起始日期">{{ detailData.notBefore || '-' }}</el-descriptions-item>
          <el-descriptions-item label="到期日期">{{ detailData.notAfter || '-' }}</el-descriptions-item>
          <el-descriptions-item label="剩余天数">
            <el-tag v-if="detailData.remainingDays >= 0" :type="getRemainingDaysType(detailData.remainingDays)" size="small">{{ detailData.remainingDays }} 天</el-tag><span v-else>-</span>
          </el-descriptions-item>
          <el-descriptions-item label="状态"><el-tag :type="getStatusType(detailData.status)" size="small">{{ getStatusText(detailData.status) }}</el-tag></el-descriptions-item>
          <el-descriptions-item label="检查时间">{{ detailData.checkTime || '-' }}</el-descriptions-item>
          <el-descriptions-item label="错误信息"><span v-if="detailData.errorMsg" style="color:#f56c6c">{{ detailData.errorMsg }}</span><span v-else>-</span></el-descriptions-item>
        </el-descriptions>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { getDomainCertList, addDomainCert, updateDomainCert, deleteDomainCert, batchDeleteDomainCert, checkDomainCert, checkAllDomainCert, getDomainCertById } from '@/api/monitor'
export default {
  data() {
    return {
      Loading: true, multiple: true, showSearch: true, total: 0, ids: [], domainCertList: [],
      queryParams: { page: 1, pageSize: 20, domain: '', status: undefined },
      dialogVisible: false, isEdit: false, submitLoading: false,
      form: { id: 0, domain: '', port: 443 },
      formRules: { domain: [{ required: true, message: '请输入域名', trigger: 'blur' }] },
      detailVisible: false, detailData: null, checkingAll: false
    }
  },
  methods: {
    async getList() {
      this.Loading = true
      try {
        const { data: res } = await getDomainCertList(this.queryParams)
        if (res.code === 200) { this.domainCertList = res.data.list || []; this.total = res.data.total || 0 }
        else { this.$message.error(res.message) }
      } catch { this.$message.error('获取列表失败') } finally { this.Loading = false }
    },
    handleQuery() { this.queryParams.page = 1; this.getList() },
    resetQuery() { this.queryParams = { page: 1, pageSize: 20, domain: '', status: undefined }; this.getList(); this.$message.success('重置成功') },
    handleSelectionChange(selection) { this.ids = selection.map(i => i.id); this.multiple = !selection.length },
    handleSizeChange(s) { this.queryParams.pageSize = s; this.getList() },
    handleCurrentChange(p) { this.queryParams.page = p; this.getList() },
    handleAdd() { this.isEdit = false; this.form = { id: 0, domain: '', port: 443 }; this.dialogVisible = true },
    async handleEdit(row) {
      this.isEdit = true; this.form = { id: row.id, domain: row.domain, port: row.port }; this.dialogVisible = true
    },
    handleSubmit() {
      this.$refs.formRef.validate(async valid => {
        if (!valid) return; this.submitLoading = true
        try {
          const fn = this.isEdit ? updateDomainCert : addDomainCert
          const { data: res } = await fn(this.form)
          if (res.code !== 200) { this.$message.error(res.message) } else {
            this.$message.success(this.isEdit ? '更新成功' : '添加成功'); this.dialogVisible = false; this.getList()
          }
        } catch { this.$message.error('操作失败') } finally { this.submitLoading = false }
      })
    },
    async handleDelete(row) {
      const c = await this.$confirm(`是否确认删除 "${row.domain}"？`, '提示', { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' }).catch(e => e)
      if (c !== 'confirm') return
      try {
        const { data: res } = await deleteDomainCert(row.id)
        if (res.code !== 200) { this.$message.error(res.message) } else { this.$message.success('删除成功'); this.getList() }
      } catch { this.$message.error('删除失败') }
    },
    async handleBatchDelete() {
      if (!this.ids.length) { this.$message.warning('请选择要删除的记录'); return }
      const c = await this.$confirm(`确认删除 ${this.ids.length} 条记录？`, '提示', { type: 'warning' }).catch(e => e)
      if (c !== 'confirm') return
      try {
        const { data: res } = await batchDeleteDomainCert(this.ids)
        if (res.code !== 200) { this.$message.error(res.message) } else { this.$message.success('删除成功'); this.ids = []; this.getList() }
      } catch { this.$message.error('批量删除失败') }
    },
    async handleCheckSingle(row) {
      this.$set(row, '_checking', true)
      try {
        const { data: res } = await checkDomainCert(row.id)
        if (res.code !== 200) { this.$message.error(res.message || '检查失败') } else {
          this.$message.success(`检查完成: ${res.data.domain}`); this.detailData = res.data; this.detailVisible = true; this.getList()
        }
      } catch { this.$message.error('检查失败') } finally { this.$set(row, '_checking', false) }
    },
    async handleCheckAll() {
      this.checkingAll = true
      try {
        const { data: res } = await checkAllDomainCert()
        if (res.code !== 200) { this.$message.error(res.message) } else { this.$message.success(`检查完成，共 ${res.data.total} 个`); this.getList() }
      } catch { this.$message.error('批量检查失败') } finally { this.checkingAll = false }
    },
    async handleViewDetail(row) {
      try {
        const { data: res } = await getDomainCertById(row.id)
        if (res.code === 200) { this.detailData = res.data; this.detailVisible = true }
      } catch { this.$message.error('获取详情失败') }
    },
    getStatusType(s) { return { 1: 'success', 2: 'warning', 3: 'danger', 4: 'info' }[s] || 'info' },
    getStatusText(s) { return { 1: '正常', 2: '即将过期', 3: '已过期', 4: '检查失败' }[s] || '未知' },
    getRemainingDaysType(d) { if (d <= 7) return 'danger'; if (d <= 30) return 'warning'; return 'success' }
  },
  created() { this.getList() }
}
</script>
<style scoped>
.domain-cert-monitor { padding: 20px; min-height: 100vh; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
.domain-cert-card { background: rgba(255,255,255,0.95); backdrop-filter: blur(10px); border-radius: 16px; box-shadow: 0 8px 32px rgba(0,0,0,0.1); }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 20px; font-weight: 600; color: #2c3e50; background: linear-gradient(45deg,#667eea,#764ba2); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
.search-section { margin-bottom: 24px; padding: 20px; background: rgba(103,126,234,0.05); border-radius: 12px; border: 1px solid rgba(103,126,234,0.1); }
.action-section { margin-bottom: 24px; padding: 12px 0; }
.action-section .el-button { margin-right: 12px; }
.table-section { margin-bottom: 24px; }
.domain-cert-table { border-radius: 12px; overflow: hidden; box-shadow: 0 4px 16px rgba(0,0,0,0.08); }
.domain-cert-table :deep(.el-table__header) { background: linear-gradient(135deg,#667eea,#764ba2); }
.domain-cert-table :deep(.el-table__header th) { background: transparent !important; color: #2c3e50 !important; font-weight: 700 !important; }
.pagination-section { display: flex; justify-content: center; padding: 20px 0; }
.cert-detail { padding: 10px; }
.el-button { border-radius: 8px; transition: all .3s ease; }
.el-button:hover { transform: translateY(-2px); box-shadow: 0 4px 12px rgba(0,0,0,0.15); }
.el-tag { border-radius: 8px; border: none; }
</style>