<template>
  <div class="physical-machine-management asset-page">
    <header class="asset-page-header">
      <div><span class="asset-eyebrow">ASSET INVENTORY / PHYSICAL</span><h1>{{ $tr("物理机") }}</h1><p>{{ $tr("从硬件资产到带外控制台，集中管理设备信息和独立访问路径。") }}</p></div>
      <div class="asset-header-count"><strong>{{ total }}</strong><span>{{ $tr("匹配资产") }}</span></div>
    </header>
    <el-card shadow="never" class="main-card asset-surface">


      <!-- Stat cards -->
      <el-row :gutter="20" class="stat-row">
        <el-col :xs="12" :sm="8" :lg="4" v-for="s in stats" :key="s.label">
          <el-card shadow="never" class="stat-card">
            <div class="stat-value">{{ s.value }}</div>
            <div class="stat-label">{{ $tr(s.label) }}</div>
          </el-card>
        </el-col>
      </el-row>

      <!-- Search -->
      <div class="search-section asset-filters">
        <el-form :inline="true" :model="query" size="small">
          <el-form-item :label='$tr("关键字")'>
            <el-input v-model="query.keyword" :placeholder='$tr("SN/主机名/IP/型号")' clearable style="width:200px" @keyup.enter="search" />
          </el-form-item>
          <el-form-item :label='$tr("机房")'>
            <el-select v-model="query.idcId" :placeholder='$tr("选择机房")' clearable filterable style="width:150px">
              <el-option v-for="d in idcList" :key="d.id" :label="$tr(d.name)" :value="d.id" />
            </el-select>
          </el-form-item>
          <el-form-item :label='$tr("品牌")'>
            <el-select v-model="query.brand" :placeholder='$tr("选择品牌")' clearable style="width:120px">
              <el-option label="Dell" value="Dell" />
              <el-option label="HP" value="HP" />
              <el-option label="Inspur" value="Inspur" />
              <el-option label="Huawei" value="Huawei" />
              <el-option label="Lenovo" value="Lenovo" />
              <el-option label="Supermicro" value="Supermicro" />
            </el-select>
          </el-form-item>
          <el-form-item :label='$tr("资产状态")'>
            <el-select v-model="query.assetStatus" :placeholder='$tr("资产状态")' clearable style="width:120px">
              <el-option :label='$tr("在库")' :value="1" />
              <el-option :label='$tr("已上架")' :value="2" />
              <el-option :label='$tr("维修中")' :value="3" />
              <el-option :label='$tr("已下架")' :value="4" />
              <el-option :label='$tr("报废")' :value="5" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" size="small" @click="search">{{ $tr("查询") }}</el-button>
            <el-button size="small" @click="reset">{{ $tr("重置") }}</el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- Actions -->
      <div class="action-section asset-toolbar">
        <el-button type="primary" size="small" @click="showCreate">{{ $tr("+ 新增物理机") }}</el-button>
      </div>

      <!-- Table -->
      <el-table :data="list" v-loading="loading" row-key="id" class="asset-table" style="width:100%">
        <el-table-column :label='$tr("设备")' min-width="230" fixed="left"><template #default="{row}"><div class="asset-identity"><span class="asset-avatar">PHY</span><div class="asset-cell"><strong>{{ row.hostName || row.sn }}</strong><span class="asset-muted asset-mono">{{ row.sn || '未登记序列号' }}</span></div></div></template></el-table-column>
        <el-table-column :label='$tr("访问地址")' min-width="190"><template #default="{row}"><div class="asset-cell"><span class="asset-mono">{{ row.manageIp || '—' }} <small class="asset-muted">{{ $tr("管理") }}</small></span><span class="asset-mono asset-muted">{{ row.businessIp || '—' }} <small>{{ $tr("业务") }}</small></span></div></template></el-table-column>
        <el-table-column :label='$tr("硬件")' min-width="170"><template #default="{row}"><div class="asset-cell"><span>{{ row.brand || '—' }} · {{ row.model || '—' }}</span><span class="asset-muted">{{ row.memory || '—' }} {{ $tr("内存 ·") }} {{ row.disk || '—' }}</span></div></template></el-table-column>
        <el-table-column :label='$tr("位置")' min-width="160">
          <template #default="{row}">
            <span v-if="row.idc">{{ row.idc.name }} / {{ row.cabinet ? row.cabinet.name : '-' }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column :label='$tr("资产状态")' width="90" align="center">
          <template #default="{row}">
            <el-tag :type="assetStatusType(row.assetStatus)" size="small">{{ $tr(assetStatusText(row.assetStatus)) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label='$tr("维保")' min-width="120"><template #default="{row}"><div class="asset-cell"><span>{{ row.warrantyDate || '未登记' }}</span><span class="asset-muted">{{ $tr("到期日期") }}</span></div></template></el-table-column>
        <el-table-column :label='$tr("操作")' width="205" fixed="right">
          <template #default="{row}"><div class="asset-actions">
            <el-button type="primary" size="small" plain @click="openConnections(row)">{{ $tr("连接") }}</el-button>
            <el-button link type="primary" @click="showEdit(row)">{{ $tr("编辑") }}</el-button>
            <el-dropdown trigger="click"><el-button text size="small" :aria-label='$tr("更多操作")'>{{ $tr("更多") }}</el-button><template #dropdown><el-dropdown-menu><el-dropdown-item @click="handleDelete(row)">{{ $tr("删除资产") }}</el-dropdown-item></el-dropdown-menu></template></el-dropdown>
          </div></template>
        </el-table-column>
        <template #empty><el-empty :description='$tr("没有匹配的资产，调整筛选或新增资产")' :image-size="90" /></template>
      </el-table>

      <!-- Pagination -->
      <div class="pagination-section asset-pagination">
        <el-pagination
          @size-change="s=>{query.size=s;query.page=1;fetch()}"
          @current-change="p=>{query.page=p;fetch()}"
          :current-page="query.page"
          :page-sizes="[10,20,50,100]"
          :page-size="query.size"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total" />
      </div>
    </el-card>

    <AssetConnections v-model="connectionsVisible" asset-type="physical" :asset="activeAsset" />
    <!-- Create/Edit Dialog -->
    <el-dialog :title="$tr(formTitle)" v-model="formVisible" width="650px" @close="resetForm">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="SN" prop="sn"><el-input v-model="form.sn" :placeholder='$tr("序列号")' /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label='$tr("主机名")'><el-input v-model="form.hostName" :placeholder='$tr("主机名")' /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label='$tr("管理IP")'><el-input v-model="form.manageIp" placeholder="BMC/iLO/iDRAC IP" /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label='$tr("业务IP")'><el-input v-model="form.businessIp" :placeholder='$tr("业务IP")' /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label='$tr("品牌")'><el-input v-model="form.brand" placeholder="Dell/HP/Inspur" /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label='$tr("型号")'><el-input v-model="form.model" placeholder="R750/DL380" /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="CPU"><el-input v-model="form.cpu" :placeholder='$tr("CPU信息")' /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label='$tr("内存")'><el-input v-model="form.memory" :placeholder='$tr("如: 256GB")' /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label='$tr("磁盘")'><el-input v-model="form.disk" :placeholder='$tr("如: 4*2TB SSD")' /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="RAID"><el-input v-model="form.raid" :placeholder='$tr("RAID类型")' /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label='$tr("机房")'>
              <el-select v-model="form.idcId" :placeholder='$tr("选择机房")' clearable filterable style="width:100%" @change="onIdcChange">
                <el-option v-for="d in idcList" :key="d.id" :label="$tr(d.name)" :value="d.id" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label='$tr("机柜")'>
              <el-select v-model="form.cabinetId" :placeholder='$tr("选择机柜")' clearable filterable style="width:100%">
                <el-option v-for="c in cabinetOptions" :key="c.id" :label="$tr(c.name)" :value="c.id" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="8">
            <el-form-item :label='$tr("U位")'><el-input-number v-model="form.unitPosition" :min="0" :max="100" /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item :label='$tr("资产状态")'>
              <el-select v-model="form.assetStatus" style="width:100%">
                <el-option :label='$tr("在库")' :value="1" />
                <el-option :label='$tr("已上架")' :value="2" />
                <el-option :label='$tr("维修中")' :value="3" />
                <el-option :label='$tr("已下架")' :value="4" />
                <el-option :label='$tr("报废")' :value="5" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item :label='$tr("供应商")'><el-input v-model="form.vendor" :placeholder='$tr("供应商")' /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label='$tr("采购日期")'><el-input v-model="form.purchaseDate" placeholder="YYYY-MM-DD" /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label='$tr("维保到期")'><el-input v-model="form.warrantyDate" placeholder="YYYY-MM-DD" /></el-form-item>
          </el-col>
        </el-row>
        <el-form-item :label='$tr("备注")'><el-input v-model="form.remark" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible=false">{{ $tr("取消") }}</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">{{ $tr("确定") }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { tr } from "@/locales";

import cmdbAPI from '@/api/cmdb'
import AssetConnections from './components/AssetConnections.vue'

export default {
  components: { AssetConnections },
  name: 'PhysicalMachine',
  data() {
    return {
      connectionsVisible: false, configVisible: false, activeAsset: null,
      loading: false, submitting: false,
      list: [], total: 0,
      idcList: [], cabinetOptions: [],
      stats: [
        { label: '总物理机', value: 0, key: 'total' },
        { label: '在库', value: 0, key: 'status_in_stock' },
        { label: '已上架', value: 0, key: 'status_online' },
        { label: '维修中', value: 0, key: 'status_repair' },
        { label: '已下架', value: 0, key: 'status_offline' },
        { label: '报废', value: 0, key: 'status_scrap' },
      ],
      query: { keyword: '', idcId: undefined, brand: '', assetStatus: undefined, page: 1, size: 10 },
      formVisible: false, formTitle: '新增物理机', isEdit: false, editId: null,
      form: { sn: '', hostName: '', manageIp: '', businessIp: '', brand: '', model: '',
              cpu: '', memory: '', disk: '', raid: '', idcId: undefined, cabinetId: undefined,
              unitPosition: 0, assetStatus: 2, purchaseDate: '', warrantyDate: '', vendor: '', remark: '' },
      rules: { sn: [{ required: true, get message() { return tr('请输入SN'); }, trigger: 'blur' }] }
    }
  },
  created() { this.fetch(); this.fetchIdcList() },
  methods: {
    openConnections(row) { this.activeAsset = row; this.connectionsVisible = true },
    async fetch() {
      this.loading = true
      try {
        const res = await cmdbAPI.getPhysicalMachineList(this.query)
        if (res.data.code === 200) {
          const d = res.data.data
          this.list = d.list || []
          this.total = d.total || 0
        }
      } catch (e) { this.$message.error(tr('获取物理机列表失败')) } finally { this.loading = false }
    },
    async fetchIdcList() {
      try {
        const res = await cmdbAPI.getIdcAll()
        if (res.data.code === 200) this.idcList = res.data.data.list || []
      } catch (_) {}
      try {
        const res = await cmdbAPI.getPhysicalMachineStats()
        if (res.data.code === 200) {
          const s = res.data.data
          this.stats.forEach(item => { item.value = s[item.key] || 0 })
        }
      } catch (_) {}
    },
    async onIdcChange(idcId) {
      this.form.cabinetId = undefined
      this.cabinetOptions = []
      if (!idcId) return
      try {
        console.log('物理机-选择机房ID:', idcId)
        const res = await cmdbAPI.getCabinetByIdc(idcId)
        console.log('物理机-机柜API响应:', res.data)
        if (res.data && res.data.code === 200) {
          const resultData = res.data.data
          // 兼容不同返回格式: {list:[...]} 或 直接返回数组
          if (Array.isArray(resultData)) {
            this.cabinetOptions = resultData
          } else if (resultData && Array.isArray(resultData.list)) {
            this.cabinetOptions = resultData.list
          } else {
            this.cabinetOptions = []
            console.warn('物理机-机柜数据格式异常:', resultData)
          }
        } else {
          console.warn('物理机-获取机柜失败:', res.data)
        }
      } catch (e) {
        console.error('物理机-获取机柜异常:', e)
      }
    },
    search() { this.query.page = 1; this.fetch() },
    reset() { this.query = { keyword: '', idcId: undefined, brand: '', assetStatus: undefined, page: 1, size: 10 }; this.fetch() },
    showCreate() {
      this.isEdit = false; this.editId = null; this.formTitle = '新增物理机'
      this.form = { sn: '', hostName: '', manageIp: '', businessIp: '', brand: '', model: '',
                    cpu: '', memory: '', disk: '', raid: '', idcId: undefined, cabinetId: undefined,
                    unitPosition: 0, assetStatus: 2, purchaseDate: '', warrantyDate: '', vendor: '', remark: '' }
      this.cabinetOptions = []
      this.formVisible = true
    },
    async showEdit(row) {
      this.isEdit = true; this.editId = row.id; this.formTitle = '编辑物理机'
      this.form = { ...row, idcId: row.idcId, cabinetId: row.cabinetId }
      if (row.idcId) await this.onIdcChange(row.idcId)
      this.form.cabinetId = row.cabinetId
      this.formVisible = true
    },
    resetForm() { this.$refs.formRef?.resetFields() },
    handleSubmit() {
      this.$refs.formRef.validate(async v => {
        if (!v) return
        this.submitting = true
        try {
          let res
          if (this.isEdit) res = await cmdbAPI.updatePhysicalMachine(this.editId, this.form)
          else res = await cmdbAPI.createPhysicalMachine(this.form)
          if (res.data.code === 200) {
            this.$message.success(this.isEdit ? '更新成功' : tr('创建成功'))
            this.formVisible = false; this.fetch()
          } else { this.$message.error(res.data.message) }
        } catch (e) { this.$message.error('操作失败: ' + (e.response?.data?.message || e.message))
        } finally { this.submitting = false }
      })
    },
    handleDelete(row) {
      this.$confirm(`确定删除物理机 ${row.sn} 吗？`, tr('确认'), { type: 'warning' }).then(async () => {
        try {
          const res = await cmdbAPI.deletePhysicalMachine(row.id)
          if (res.data.code === 200) { this.$message.success(tr('删除成功')); this.fetch() }
          else this.$message.error(res.data.message)
        } catch (e) { this.$message.error(tr('删除失败')) }
      }).catch(() => {})
    },
    assetStatusType(s) {
      return { 1: 'info', 2: 'success', 3: 'warning', 4: 'danger', 5: 'info' }[s] || 'info'
    },
    assetStatusText(s) {
      return { 1: '在库', 2: '已上架', 3: '维修中', 4: '已下架', 5: '报废' }[s] || '未知'
    }
  }
}
</script>

<style scoped>
.physical-machine-management { padding: 20px; min-height: 100vh; background: var(--bg-page); }
.main-card { border-radius: var(--radius-lg); }
.card-header .title { font-size: 18px; font-weight: 600; color: var(--text-primary); }
.stat-row { margin-bottom: 16px; }
.stat-card { text-align: center; border-radius: 8px; background: var(--bg-card); }
.stat-value { font-size: 28px; font-weight: 700; color: var(--primary); }
.stat-label { font-size: 13px; color: var(--text-secondary); margin-top: 4px; }
.search-section { background: var(--bg-card-alt); padding: 16px 20px; border-radius: var(--radius); margin-bottom: 16px; border: 1px solid var(--border-light); }
.action-section { margin-bottom: 16px; }
.pagination-section { display: flex; justify-content: center; margin-top: 20px; }
</style>
