<template>
  <div class="api-monitor">
    <el-card shadow="hover" class="api-card">
      <template #header><div class="card-header"><span class="title">API端点监控</span></div></template>
      <div class="search-section" v-show="showSearch">
        <el-form :model="q" :inline="true">
          <el-form-item label="名称"><el-input v-model="q.name" placeholder="请输入名称" clearable size="small" style="width:200px" /></el-form-item>
          <el-form-item label="状态"><el-select v-model="q.status" placeholder="状态" clearable size="small" style="width:150px">
            <el-option label="正常" :value="1" /><el-option label="异常" :value="2" /><el-option label="超时" :value="3" /><el-option label="失败" :value="4" />
          </el-select></el-form-item>
          <el-form-item><el-button type="primary" icon="Search" size="small" @click="handleQuery">搜索</el-button><el-button icon="Refresh" size="small" @click="resetQuery">重置</el-button></el-form-item>
        </el-form>
      </div>
      <div class="action-section">
        <el-button type="primary" icon="Plus" size="small" @click="handleAdd">添加API</el-button>
        <el-button type="warning" icon="Refresh" size="small" :loading="checkingAll" @click="handleCheckAll">检查全部</el-button>
        <el-button type="danger" plain icon="Delete" size="small" :disabled="multiple" @click="handleBatchDelete">批量删除</el-button>
      </div>
      <div class="table-section">
        <el-table v-loading="loading" :data="list" stripe @selection-change="handleSelectionChange">
          <el-table-column type="selection" width="45" />
          <el-table-column label="名称" prop="name" min-width="140" />
          <el-table-column label="URL" prop="url" min-width="250" show-overflow-tooltip />
          <el-table-column label="方法" prop="method" width="80"><template #default="s"><el-tag size="small">{{ s.row.method }}</el-tag></template></el-table-column>
          <el-table-column label="状态码" prop="lastStatusCode" width="80" align="center" />
          <el-table-column label="响应时间" width="100" align="center">
            <template #default="s"><span>{{ s.row.lastResponseTime }}ms</span></template>
          </el-table-column>
          <el-table-column label="期望码" prop="expectedCode" width="80" align="center" />
          <el-table-column label="状态" width="80" align="center">
            <template #default="s"><el-tag :type="getStatusType(s.row.status)" size="small" effect="dark">{{ getStatusText(s.row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="检查时间" prop="checkTime" min-width="150" />
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="s">
              <el-button type="primary" icon="Edit" size="small" circle @click="handleEdit(s.row)" />
              <el-button type="warning" icon="Refresh" size="small" circle :loading="s.row._checking" @click="handleCheck(s.row)" />
              <el-button type="primary" icon="View" size="small" circle @click="handleDetail(s.row)" />
              <el-button type="danger" icon="Delete" size="small" circle @click="handleDelete(s.row)" />
            </template>
          </el-table-column>
        </el-table>
      </div>
      <div class="pagination-section">
        <el-pagination @size-change="s=>{q.pageSize=s;getList()}" @current-change="p=>{q.page=p;getList()}"
          :current-page="q.page" :page-sizes="[10,20,50,100]" :page-size="q.pageSize" layout="total, sizes, prev, pager, next, jumper" :total="total" />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEdit?'编辑API':'添加API'" width="650px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <el-form-item label="名称" prop="name"><el-input v-model="form.name" placeholder="如: 生产环境API" /></el-form-item>
        <el-form-item label="URL" prop="url"><el-input v-model="form.url" placeholder="https://api.example.com/health" /></el-form-item>
        <el-form-item label="请求方法"><el-select v-model="form.method" style="width:150px"><el-option label="GET" value="GET" /><el-option label="POST" value="POST" /><el-option label="PUT" value="PUT" /><el-option label="DELETE" value="DELETE" /></el-select></el-form-item>
        <el-form-item label="请求头(JSON)"><el-input v-model="form.headers" placeholder='{"Authorization":"Bearer xxx"}' /></el-form-item>
        <el-form-item label="请求体"><el-input v-model="form.body" type="textarea" :rows="2" placeholder='{"key":"value"}' /></el-form-item>
        <el-form-item label="检查间隔(秒)"><el-input-number v-model="form.checkInterval" :min="10" :max="86400" /></el-form-item>
        <el-form-item label="超时(秒)"><el-input-number v-model="form.timeout" :min="1" :max="120" /></el-form-item>
        <el-form-item label="期望状态码"><el-input-number v-model="form.expectedCode" :min="100" :max="599" /></el-form-item>
        <el-form-item label="期望响应体"><el-input v-model="form.expectedBody" placeholder="留空则不检查响应体" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="dialogVisible=false">取消</el-button><el-button type="primary" @click="handleSubmit" :loading="submitLoading">{{ isEdit?'保存':'确定' }}</el-button></template>
    </el-dialog>

    <el-dialog v-model="detailVisible" title="检查详情" width="550px">
      <div v-if="detail" class="cert-detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="名称">{{ detail.name }}</el-descriptions-item>
          <el-descriptions-item label="URL">{{ detail.url }}</el-descriptions-item>
          <el-descriptions-item label="方法">{{ detail.method }}</el-descriptions-item>
          <el-descriptions-item label="状态码">{{ detail.lastStatusCode }}</el-descriptions-item>
          <el-descriptions-item label="响应时间">{{ detail.lastResponseTime }}ms</el-descriptions-item>
          <el-descriptions-item label="状态"><el-tag :type="getStatusType(detail.status)" size="small">{{ getStatusText(detail.status) }}</el-tag></el-descriptions-item>
          <el-descriptions-item label="检查时间">{{ detail.checkTime }}</el-descriptions-item>
          <el-descriptions-item label="错误信息"><span v-if="detail.errorMsg" style="color:#f56c6c">{{ detail.errorMsg }}</span><span v-else>-</span></el-descriptions-item>
        </el-descriptions>
      </div>
    </el-dialog>
  </div>
</template>
<script>
import { getAPIEndpointList, addAPIEndpoint, updateAPIEndpoint, deleteAPIEndpoint, batchDeleteAPIEndpoint, checkAPIEndpoint, checkAllAPIEndpoint } from '@/api/monitor'
export default {
  data() {
    return {
      loading: true, multiple: true, showSearch: true, total: 0, ids: [], list: [], q: { page: 1, pageSize: 20, name: '', status: undefined },
      checkingAll: false, dialogVisible: false, isEdit: false, submitLoading: false, detailVisible: false, detail: null,
      form: { id: 0, name: '', url: '', method: 'GET', headers: '', body: '', checkInterval: 300, timeout: 10, expectedCode: 200, expectedBody: '' },
      rules: { name: [{ required: true, message: '请输入名称', trigger: 'blur' }], url: [{ required: true, message: '请输入URL', trigger: 'blur' }] }
    }
  },
  methods: {
    async getList() { this.loading = true; try { const { data: r } = await getAPIEndpointList(this.q); if (r.code === 200) { this.list = r.data.list || []; this.total = r.data.total || 0 } else { this.$message.error(r.message) } } catch { this.$message.error('获取列表失败') } finally { this.loading = false } },
    handleQuery() { this.q.page = 1; this.getList() },
    resetQuery() { this.q = { page: 1, pageSize: 20, name: '', status: undefined }; this.getList() },
    handleSelectionChange(s) { this.ids = s.map(i => i.id); this.multiple = !s.length },
    handleAdd() { this.isEdit = false; this.form = { id: 0, name: '', url: '', method: 'GET', headers: '', body: '', checkInterval: 300, timeout: 10, expectedCode: 200, expectedBody: '' }; this.dialogVisible = true },
    handleEdit(r) { this.isEdit = true; this.form = { id: r.id, name: r.name, url: r.url, method: r.method, headers: r.headers, body: r.body, checkInterval: r.checkInterval, timeout: r.timeout, expectedCode: r.expectedCode, expectedBody: r.expectedBody }; this.dialogVisible = true },
    handleSubmit() { this.$refs.formRef.validate(async v => { if (!v) return; this.submitLoading = true; try { const fn = this.isEdit ? updateAPIEndpoint : addAPIEndpoint; const { data: r } = await fn(this.form); if (r.code !== 200) { this.$message.error(r.message) } else { this.$message.success(this.isEdit ? '更新成功' : '添加成功'); this.dialogVisible = false; this.getList() } } catch { this.$message.error('操作失败') } finally { this.submitLoading = false } }) },
    async handleDelete(r) { const c = await this.$confirm(`确认删除 "${r.name}"?`, '提示', { type: 'warning' }).catch(e => e); if (c !== 'confirm') return; try { const { data: r2 } = await deleteAPIEndpoint(r.id); if (r2.code !== 200) { this.$message.error(r2.message) } else { this.$message.success('删除成功'); this.getList() } } catch { this.$message.error('删除失败') } },
    async handleBatchDelete() { if (!this.ids.length) { this.$message.warning('请选择记录'); return }; const c = await this.$confirm(`确认删除 ${this.ids.length} 条?`, '提示', { type: 'warning' }).catch(e => e); if (c !== 'confirm') return; try { const { data: r } = await batchDeleteAPIEndpoint(this.ids); if (r.code !== 200) { this.$message.error(r.message) } else { this.$message.success('删除成功'); this.ids = []; this.getList() } } catch { this.$message.error('批量删除失败') } },
    async handleCheck(r) { this.$set(r, '_checking', true); try { const { data: r2 } = await checkAPIEndpoint(r.id); if (r2.code !== 200) { this.$message.error(r2.message) } else { this.$message.success('检查完成'); this.detail = r2.data; this.detailVisible = true; this.getList() } } catch { this.$message.error('检查失败') } finally { this.$set(r, '_checking', false) } },
    async handleCheckAll() { this.checkingAll = true; try { const { data: r } = await checkAllAPIEndpoint(); if (r.code !== 200) { this.$message.error(r.message) } else { this.$message.success(`检查完成,共 ${r.data.total} 个`); this.getList() } } catch { this.$message.error('批量检查失败') } finally { this.checkingAll = false } },
    handleDetail(r) { this.detail = r; this.detailVisible = true },
    getStatusType(s) { return { 1: 'success', 2: 'danger', 3: 'warning', 4: 'info' }[s] || 'info' },
    getStatusText(s) { return { 1: '正常', 2: '异常', 3: '超时', 4: '失败' }[s] || '未知' }
  },
  created() { this.getList() }
}
</script>
<style scoped>
.api-monitor { padding: 20px; min-height: 100vh; background: var(--bg-page); }
.api-card { background: rgba(255,255,255,0.95); backdrop-filter: blur(10px); border-radius: 16px; box-shadow: 0 8px 32px rgba(0,0,0,0.1); }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 20px; font-weight: 600; background: linear-gradient(45deg,#667eea,#764ba2); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
.search-section { margin-bottom: 24px; padding: 20px; background: rgba(103,126,234,0.05); border-radius: 12px; }
.action-section { margin-bottom: 24px; padding: 12px 0; }
.action-section .el-button { margin-right: 12px; }
.pagination-section { display: flex; justify-content: center; padding: 20px 0; }
.cert-detail { padding: 10px; }
.el-button { border-radius: 8px; transition: all .3s ease; }
.el-button:hover { transform: translateY(-2px); box-shadow: 0 4px 12px rgba(0,0,0,0.15); }
.el-tag { border-radius: 8px; border: none; }
</style>