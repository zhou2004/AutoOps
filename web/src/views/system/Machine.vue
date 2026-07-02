<template>
  <div class="machine-management">
    <el-card shadow="hover" class="main-card">
      <template #header>
        <div class="card-header">
          <span class="title">机房信息</span>
        </div>
      </template>

      <el-tabs v-model="activeTab">
        <!-- ==================== 机房管理 ==================== -->
        <el-tab-pane label="机房管理" name="idc">
          <div class="search-section">
            <el-form :inline="true" :model="idcQuery" size="small">
              <el-form-item label="机房名称">
                <el-input v-model="idcQuery.name" placeholder="机房名称" clearable style="width:180px" @keyup.enter="searchIdc" />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" size="small" @click="searchIdc">查询</el-button>
                <el-button size="small" @click="resetIdc">重置</el-button>
              </el-form-item>
            </el-form>
          </div>
          <div class="action-section">
            <el-button type="primary" size="small" @click="showCreateIdc">+ 新增机房</el-button>
          </div>
          <el-table :data="idcList" v-loading="idcLoading" border stripe>
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="name" label="机房名称" min-width="140" />
            <el-table-column prop="shortName" label="简称" width="100" />
            <el-table-column prop="address" label="地址" min-width="200" show-overflow-tooltip />
            <el-table-column prop="contact" label="联系人" width="100" />
            <el-table-column prop="phone" label="联系电话" width="130" />
            <el-table-column prop="level" label="等级" width="80" align="center">
              <template #default="{row}"><el-tag size="small">{{ row.level || '-' }}</el-tag></template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="80" align="center">
              <template #default="{row}">
                <el-tag :type="row.status===1?'success':'info'" size="small">{{ row.status===1?'启用':'停用' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="140" align="center" fixed="right">
              <template #default="{row}">
                <el-button type="warning" size="small" @click="showEditIdc(row)">编辑</el-button>
                <el-button type="danger" size="small" @click="handleDeleteIdc(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="pagination-section">
            <el-pagination
              @size-change="s=>{idcQuery.size=s;fetchIdc()}"
              @current-change="p=>{idcQuery.page=p;fetchIdc()}"
              :current-page="idcQuery.page" :page-sizes="[10,20,50,100]" :page-size="idcQuery.size"
              layout="total, sizes, prev, pager, next, jumper" :total="idcTotal" />
          </div>
        </el-tab-pane>

        <!-- ==================== 机柜管理 ==================== -->
        <el-tab-pane label="机柜管理" name="cabinet">
          <div class="search-section">
            <el-form :inline="true" :model="cabinetQuery" size="small">
              <el-form-item label="机柜名称">
                <el-input v-model="cabinetQuery.name" placeholder="机柜名称" clearable style="width:160px" @keyup.enter="searchCabinet" />
              </el-form-item>
              <el-form-item label="机房">
                <el-select v-model="cabinetQuery.idcId" placeholder="选择机房" clearable filterable style="width:160px">
                  <el-option v-for="d in idcOptions" :key="d.id" :label="d.name" :value="d.id" />
                </el-select>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" size="small" @click="searchCabinet">查询</el-button>
                <el-button size="small" @click="resetCabinet">重置</el-button>
              </el-form-item>
            </el-form>
          </div>
          <div class="action-section">
            <el-button type="primary" size="small" @click="showCreateCabinet">+ 新增机柜</el-button>
          </div>
          <el-table :data="cabinetList" v-loading="cabinetLoading" border stripe>
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="name" label="机柜名称" min-width="140" />
            <el-table-column label="所属机房" width="140">
              <template #default="{row}">{{ row.idc.id ? row.idc.name : '-' }}</template>
            </el-table-column>
            <el-table-column prop="position" label="位置" width="120" />
            <el-table-column prop="unitNum" label="U数" width="70" align="center" />
            <el-table-column prop="usedUnit" label="已用U" width="70" align="center" />
            <el-table-column prop="powerKw" label="功率(KW)" width="100" align="center">
              <template #default="{row}">{{ row.powerKw || 0 }}</template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="80" align="center">
              <template #default="{row}">
                <el-tag :type="row.status===1?'success':'info'" size="small">{{ row.status===1?'启用':'停用' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="140" align="center" fixed="right">
              <template #default="{row}">
                <el-button type="warning" size="small" @click="showEditCabinet(row)">编辑</el-button>
                <el-button type="danger" size="small" @click="handleDeleteCabinet(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="pagination-section">
            <el-pagination
              @size-change="s=>{cabinetQuery.size=s;fetchCabinet()}"
              @current-change="p=>{cabinetQuery.page=p;fetchCabinet()}"
              :current-page="cabinetQuery.page" :page-sizes="[10,20,50,100]" :page-size="cabinetQuery.size"
              layout="total, sizes, prev, pager, next, jumper" :total="cabinetTotal" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- ==================== 机房对话框 ==================== -->
    <el-dialog :title="idcFormTitle" v-model="idcFormVisible" width="550px" @close="resetIdcForm">
      <el-form :model="idcForm" :rules="idcRules" ref="idcFormRef" label-width="100px">
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="机房名称" prop="name"><el-input v-model="idcForm.name" placeholder="机房名称" /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="简称"><el-input v-model="idcForm.shortName" placeholder="机房简称" /></el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="地址"><el-input v-model="idcForm.address" placeholder="机房地址" /></el-form-item>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="联系人"><el-input v-model="idcForm.contact" placeholder="联系人" /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="联系电话"><el-input v-model="idcForm.phone" placeholder="联系电话" /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="机房等级">
              <el-select v-model="idcForm.level" placeholder="T1-T4" style="width:100%">
                <el-option label="T1" value="T1" /><el-option label="T2" value="T2" />
                <el-option label="T3" value="T3" /><el-option label="T4" value="T4" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="状态">
              <el-switch v-model="idcForm.status" :active-value="1" :inactive-value="0" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="描述"><el-input v-model="idcForm.description" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="idcFormVisible=false">取消</el-button>
        <el-button type="primary" @click="handleIdcSubmit" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>

    <!-- ==================== 机柜对话框 ==================== -->
    <el-dialog :title="cabinetFormTitle" v-model="cabinetFormVisible" width="500px" @close="resetCabinetForm">
      <el-form :model="cabinetForm" :rules="cabinetRules" ref="cabinetFormRef" label-width="100px">
        <el-form-item label="机柜名称" prop="name"><el-input v-model="cabinetForm.name" placeholder="如: A01" /></el-form-item>
        <el-form-item label="所属机房" prop="idcId">
          <el-select v-model="cabinetForm.idcId" placeholder="选择机房" filterable style="width:100%">
            <el-option v-for="d in idcOptions" :key="d.id" :label="d.name" :value="d.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="位置"><el-input v-model="cabinetForm.position" placeholder="如: A列3排" /></el-form-item>
        <el-row :gutter="20">
          <el-col :span="8">
            <el-form-item label="U数"><el-input-number v-model="cabinetForm.unitNum" :min="0" :max="100" style="width:100%" /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="已用U"><el-input-number v-model="cabinetForm.usedUnit" :min="0" :max="100" style="width:100%" /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="功率KW"><el-input-number v-model="cabinetForm.powerKw" :min="0" :step="0.5" style="width:100%" /></el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="状态">
          <el-switch v-model="cabinetForm.status" :active-value="1" :inactive-value="0" />
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="cabinetForm.remark" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="cabinetFormVisible=false">取消</el-button>
        <el-button type="primary" @click="handleCabinetSubmit" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import cmdbAPI from '@/api/cmdb'

export default {
  name: 'MachineManagement',
  data() {
    return {
      activeTab: 'idc',
      submitting: false,

      // === IDC ===
      idcLoading: false, idcList: [], idcTotal: 0,
      idcQuery: { name: '', page: 1, size: 10 },
      idcFormVisible: false, idcFormTitle: '新增机房', isEditIdc: false, idcEditId: null,
      idcForm: { name: '', shortName: '', address: '', contact: '', phone: '', level: 'T3', status: 1, description: '' },
      idcRules: { name: [{ required: true, message: '请输入机房名称', trigger: 'blur' }] },

      // === Cabinet ===
      cabinetLoading: false, cabinetList: [], cabinetTotal: 0,
      idcOptions: [],
      cabinetQuery: { name: '', idcId: undefined, page: 1, size: 10 },
      cabinetFormVisible: false, cabinetFormTitle: '新增机柜', isEditCabinet: false, cabinetEditId: null,
      cabinetForm: { name: '', idcId: undefined, position: '', unitNum: 42, usedUnit: 0, powerKw: 0, status: 1, remark: '' },
      cabinetRules: {
        name: [{ required: true, message: '请输入机柜名称', trigger: 'blur' }],
        idcId: [{ required: true, message: '请选择机房', trigger: 'change' }]
      }
    }
  },
  created() { this.fetchIdc(); this.fetchIdcOptions() },
  watch: {
    activeTab(tab) {
      if (tab === 'cabinet') { this.fetchCabinet(); this.fetchIdcOptions() }
    }
  },
  methods: {
    // =========== IDC ===========
    async fetchIdc() {
      this.idcLoading = true
      try {
        const res = await cmdbAPI.getIdcList(this.idcQuery)
        if (res.data.code === 200) { const d = res.data.data; this.idcList = d.list || []; this.idcTotal = d.total || 0 }
      } catch (_) { this.$message.error('获取机房列表失败') } finally { this.idcLoading = false }
    },
    async fetchIdcOptions() {
      try { const res = await cmdbAPI.getIdcAll(); if (res.data.code === 200) this.idcOptions = res.data.data.list || [] } catch (_) {}
    },
    searchIdc() { this.idcQuery.page = 1; this.fetchIdc() },
    resetIdc() { this.idcQuery = { name: '', page: 1, size: 10 }; this.fetchIdc() },
    showCreateIdc() {
      this.isEditIdc = false; this.idcEditId = null; this.idcFormTitle = '新增机房'
      this.idcForm = { name: '', shortName: '', address: '', contact: '', phone: '', level: 'T3', status: 1, description: '' }
      this.idcFormVisible = true
    },
    showEditIdc(row) {
      this.isEditIdc = true; this.idcEditId = row.id; this.idcFormTitle = '编辑机房'
      this.idcForm = { name: row.name, shortName: row.shortName || '', address: row.address || '',
                       contact: row.contact || '', phone: row.phone || '', level: row.level || 'T3',
                       status: row.status, description: row.description || '' }
      this.idcFormVisible = true
    },
    resetIdcForm() { this.$refs.idcFormRef?.resetFields() },
    handleIdcSubmit() {
      this.$refs.idcFormRef.validate(async v => {
        if (!v) return; this.submitting = true
        try {
          let res
          if (this.isEditIdc) res = await cmdbAPI.updateIdc(this.idcEditId, this.idcForm)
          else res = await cmdbAPI.createIdc(this.idcForm)
          if (res.data.code === 200) { this.$message.success(this.isEditIdc ? '更新成功' : '创建成功'); this.idcFormVisible = false; this.fetchIdc(); this.fetchIdcOptions() }
          else this.$message.error(res.data.message)
        } catch (e) { this.$message.error('操作失败: ' + (e.response?.data?.message || e.message))
        } finally { this.submitting = false }
      })
    },
    handleDeleteIdc(row) {
      this.$confirm(`确定删除机房 "${row.name}" 吗？`, '确认', { type: 'warning' }).then(async () => {
        try { const res = await cmdbAPI.deleteIdc(row.id); if (res.data.code === 200) { this.$message.success('删除成功'); this.fetchIdc(); this.fetchIdcOptions() } else this.$message.error(res.data.message) } catch (_) { this.$message.error('删除失败') }
      }).catch(() => {})
    },

    // =========== Cabinet ===========
    async fetchCabinet() {
      this.cabinetLoading = true
      try {
        const res = await cmdbAPI.getCabinetList(this.cabinetQuery)
        if (res.data.code === 200) { const d = res.data.data; this.cabinetList = d.list || []; this.cabinetTotal = d.total || 0 }
      } catch (_) { this.$message.error('获取机柜列表失败') } finally { this.cabinetLoading = false }
    },
    searchCabinet() { this.cabinetQuery.page = 1; this.fetchCabinet() },
    resetCabinet() { this.cabinetQuery = { name: '', idcId: undefined, page: 1, size: 10 }; this.fetchCabinet() },
    showCreateCabinet() {
      this.isEditCabinet = false; this.cabinetEditId = null; this.cabinetFormTitle = '新增机柜'
      this.cabinetForm = { name: '', idcId: this.idcOptions[0]?.id || undefined, position: '', unitNum: 42, usedUnit: 0, powerKw: 0, status: 1, remark: '' }
      this.cabinetFormVisible = true
    },
    showEditCabinet(row) {
      this.isEditCabinet = true; this.cabinetEditId = row.id; this.cabinetFormTitle = '编辑机柜'
      this.cabinetForm = { name: row.name, idcId: row.idcId, position: row.position || '',
                           unitNum: row.unitNum || 42, usedUnit: row.usedUnit || 0, powerKw: row.powerKw || 0,
                           status: row.status, remark: row.remark || '' }
      this.cabinetFormVisible = true
    },
    resetCabinetForm() { this.$refs.cabinetFormRef?.resetFields() },
    handleCabinetSubmit() {
      this.$refs.cabinetFormRef.validate(async v => {
        if (!v) return; this.submitting = true
        try {
          let res
          if (this.isEditCabinet) res = await cmdbAPI.updateCabinet(this.cabinetEditId, this.cabinetForm)
          else res = await cmdbAPI.createCabinet(this.cabinetForm)
          if (res.data.code === 200) { this.$message.success(this.isEditCabinet ? '更新成功' : '创建成功'); this.cabinetFormVisible = false; this.fetchCabinet() }
          else this.$message.error(res.data.message)
        } catch (e) { this.$message.error('操作失败: ' + (e.response?.data?.message || e.message))
        } finally { this.submitting = false }
      })
    },
    handleDeleteCabinet(row) {
      this.$confirm(`确定删除机柜 "${row.name}" 吗？`, '确认', { type: 'warning' }).then(async () => {
        try { const res = await cmdbAPI.deleteCabinet(row.id); if (res.data.code === 200) { this.$message.success('删除成功'); this.fetchCabinet() } else this.$message.error(res.data.message) } catch (_) { this.$message.error('删除失败') }
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
.machine-management { padding: 20px; min-height: 100vh; background: #f0f2f5; }
.main-card { border-radius: 12px; }
.card-header .title { font-size: 18px; font-weight: 600; color: var(--text-primary); }
.search-section { background: var(--bg-card-alt); padding: 16px 20px; border-radius: 8px; margin-bottom: 16px; }
.action-section { margin-bottom: 16px; }
.pagination-section { display: flex; justify-content: center; margin-top: 20px; }
</style>
