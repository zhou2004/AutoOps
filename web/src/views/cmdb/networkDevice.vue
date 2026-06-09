<template>
  <div class="network-device-management">
    <el-card shadow="hover" class="main-card">
      <template #header>
        <div class="card-header">
          <span class="title">网络设备管理</span>
        </div>
      </template>

      <!-- Search -->
      <div class="search-section">
        <el-form :inline="true" :model="query" size="small">
          <el-form-item label="关键字">
            <el-input v-model="query.keyword" placeholder="SN/名称/IP" clearable style="width:200px" @keyup.enter="search" />
          </el-form-item>
          <el-form-item label="设备类型">
            <el-select v-model="query.deviceType" placeholder="选择类型" clearable style="width:140px">
              <el-option label="路由器" :value="1" />
              <el-option label="交换机" :value="2" />
              <el-option label="防火墙" :value="3" />
              <el-option label="负载均衡" :value="4" />
              <el-option label="其他" :value="5" />
            </el-select>
          </el-form-item>
          <el-form-item label="机房">
            <el-select v-model="query.idcId" placeholder="选择机房" clearable filterable style="width:150px">
              <el-option v-for="d in idcList" :key="d.id" :label="d.name" :value="d.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="品牌">
            <el-input v-model="query.brand" placeholder="品牌" clearable style="width:120px" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" size="small" @click="search">查询</el-button>
            <el-button size="small" @click="reset">重置</el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- Actions -->
      <div class="action-section">
        <el-button type="primary" size="small" @click="showCreate">+ 新增网络设备</el-button>
      </div>

      <!-- Table -->
      <el-table :data="list" v-loading="loading" border stripe style="width:100%">
        <el-table-column prop="sn" label="SN" width="150" />
        <el-table-column prop="name" label="设备名称" width="140" />
        <el-table-column label="设备类型" width="90" align="center">
          <template #default="{row}">
            <el-tag :type="deviceTypeTag(row.deviceType)" size="small">{{ deviceTypeText(row.deviceType) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="brand" label="品牌" width="90" />
        <el-table-column prop="model" label="型号" width="130" />
        <el-table-column prop="manageIp" label="管理IP" width="140" />
        <el-table-column prop="version" label="版本" width="100" />
        <el-table-column prop="portNum" label="端口数" width="70" align="center" />
        <el-table-column label="位置" min-width="150">
          <template #default="{row}">
            <span v-if="row.idc">{{ row.idc.name }} / {{ row.cabinet ? row.cabinet.name : '-' }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="资产状态" width="90" align="center">
          <template #default="{row}">
            <el-tag :type="assetStatusType(row.assetStatus)" size="small">{{ assetStatusText(row.assetStatus) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="purchaseDate" label="采购日期" width="100" />
        <el-table-column prop="warrantyDate" label="维保到期" width="100" />
        <el-table-column label="操作" width="120" align="center" fixed="right">
          <template #default="{row}">
            <el-button type="primary" size="small" @click="showEdit(row)">编辑</el-button>
            <el-button type="danger" size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- Pagination -->
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
    <el-dialog :title="formTitle" v-model="formVisible" width="600px" @close="resetForm">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="SN" prop="sn"><el-input v-model="form.sn" placeholder="序列号" /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="设备名称" prop="name"><el-input v-model="form.name" placeholder="设备名称" /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="设备类型" prop="deviceType">
              <el-select v-model="form.deviceType" style="width:100%">
                <el-option label="路由器" :value="1" />
                <el-option label="交换机" :value="2" />
                <el-option label="防火墙" :value="3" />
                <el-option label="负载均衡" :value="4" />
                <el-option label="其他" :value="5" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="品牌"><el-input v-model="form.brand" placeholder="Cisco/Huawei/H3C" /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="型号"><el-input v-model="form.model" placeholder="如: S5700" /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="管理IP"><el-input v-model="form.manageIp" placeholder="管理IP" /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="固件版本"><el-input v-model="form.version" placeholder="版本号" /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="端口数量"><el-input-number v-model="form.portNum" :min="0" :max="500" style="width:100%" /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="机房">
              <el-select v-model="form.idcId" placeholder="选择机房" clearable filterable style="width:100%" @change="onIdcChange">
                <el-option v-for="d in idcList" :key="d.id" :label="d.name" :value="d.id" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="机柜">
              <el-select v-model="form.cabinetId" placeholder="选择机柜" clearable filterable style="width:100%">
                <el-option v-for="c in cabinetOptions" :key="c.id" :label="c.name" :value="c.id" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="资产状态">
              <el-select v-model="form.assetStatus" style="width:100%">
                <el-option label="在库" :value="1" />
                <el-option label="已上架" :value="2" />
                <el-option label="维修中" :value="3" />
                <el-option label="已下架" :value="4" />
                <el-option label="报废" :value="5" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="供应商"><el-input v-model="form.vendor" placeholder="供应商" /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="采购日期"><el-input v-model="form.purchaseDate" placeholder="YYYY-MM-DD" /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="维保到期"><el-input v-model="form.warrantyDate" placeholder="YYYY-MM-DD" /></el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" :rows="2" /></el-form-item>
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

export default {
  name: 'NetworkDevice',
  data() {
    return {
      loading: false, submitting: false,
      list: [], total: 0,
      idcList: [], cabinetOptions: [],
      query: { keyword: '', deviceType: undefined, idcId: undefined, brand: '', page: 1, size: 10 },
      formVisible: false, formTitle: '新增网络设备', isEdit: false, editId: null,
      form: { sn: '', name: '', deviceType: 2, brand: '', model: '', manageIp: '', version: '',
              portNum: 24, idcId: undefined, cabinetId: undefined, assetStatus: 2,
              purchaseDate: '', warrantyDate: '', vendor: '', remark: '' },
      rules: {
        sn: [{ required: true, message: '请输入SN', trigger: 'blur' }],
        name: [{ required: true, message: '请输入设备名称', trigger: 'blur' }],
        deviceType: [{ required: true, message: '请选择设备类型', trigger: 'change' }]
      }
    }
  },
  created() { this.fetch(); this.fetchIdcList() },
  methods: {
    async fetch() {
      this.loading = true
      try {
        const res = await cmdbAPI.getNetworkDeviceList(this.query)
        if (res.data.code === 200) {
          const d = res.data.data
          this.list = d.list || []
          this.total = d.total || 0
        }
      } catch (e) { this.$message.error('获取网络设备列表失败') } finally { this.loading = false }
    },
    async fetchIdcList() {
      try {
        const res = await cmdbAPI.getIdcAll()
        if (res.data.code === 200) this.idcList = res.data.data.list || []
      } catch (_) {}
    },
    async onIdcChange(idcId) {
      this.form.cabinetId = undefined
      this.cabinetOptions = []
      if (!idcId) return
      try {
        const res = await cmdbAPI.getCabinetByIdc(idcId)
        if (res.data.code === 200) this.cabinetOptions = res.data.data.list || []
      } catch (_) {}
    },
    search() { this.query.page = 1; this.fetch() },
    reset() { this.query = { keyword: '', deviceType: undefined, idcId: undefined, brand: '', page: 1, size: 10 }; this.fetch() },
    showCreate() {
      this.isEdit = false; this.editId = null; this.formTitle = '新增网络设备'
      this.form = { sn: '', name: '', deviceType: 2, brand: '', model: '', manageIp: '', version: '',
                    portNum: 24, idcId: undefined, cabinetId: undefined, assetStatus: 2,
                    purchaseDate: '', warrantyDate: '', vendor: '', remark: '' }
      this.cabinetOptions = []
      this.formVisible = true
    },
    async showEdit(row) {
      this.isEdit = true; this.editId = row.id; this.formTitle = '编辑网络设备'
      this.form = { ...row, idcId: row.idcId, cabinetId: row.cabinetId }
      if (row.idcId) await this.onIdcChange(row.idcId)
      this.formVisible = true
    },
    resetForm() { this.$refs.formRef?.resetFields() },
    handleSubmit() {
      this.$refs.formRef.validate(async v => {
        if (!v) return
        this.submitting = true
        try {
          let res
          if (this.isEdit) res = await cmdbAPI.updateNetworkDevice(this.editId, this.form)
          else res = await cmdbAPI.createNetworkDevice(this.form)
          if (res.data.code === 200) {
            this.$message.success(this.isEdit ? '更新成功' : '创建成功')
            this.formVisible = false; this.fetch()
          } else { this.$message.error(res.data.message) }
        } catch (e) { this.$message.error('操作失败: ' + (e.response?.data?.message || e.message))
        } finally { this.submitting = false }
      })
    },
    handleDelete(row) {
      this.$confirm(`确定删除网络设备 ${row.name} 吗？`, '确认', { type: 'warning' }).then(async () => {
        try {
          const res = await cmdbAPI.deleteNetworkDevice(row.id)
          if (res.data.code === 200) { this.$message.success('删除成功'); this.fetch() }
          else this.$message.error(res.data.message)
        } catch (e) { this.$message.error('删除失败') }
      }).catch(() => {})
    },
    deviceTypeTag(t) { return { 1:'primary', 2:'success', 3:'danger', 4:'warning', 5:'info' }[t] || 'info' },
    deviceTypeText(t) { return { 1:'路由器', 2:'交换机', 3:'防火墙', 4:'负载均衡', 5:'其他' }[t] || '未知' },
    assetStatusType(s) { return { 1:'info', 2:'success', 3:'warning', 4:'danger', 5:'info' }[s] || 'info' },
    assetStatusText(s) { return { 1:'在库', 2:'已上架', 3:'维修中', 4:'已下架', 5:'报废' }[s] || '未知' }
  }
}
</script>

<style scoped>
.network-device-management :deep(.el-card__body) { padding: 20px; }
</style>
