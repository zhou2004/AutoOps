<template>
  <div class="network-device-management asset-page">
    <header class="asset-page-header">
      <div><span class="asset-eyebrow">ASSET INVENTORY / NETWORK</span><h1>{{ $tr("网络设备") }}</h1><p>{{ $tr("统一管理路由器、交换机与防火墙，追踪配置变更与远程访问。") }}</p></div>
      <div class="asset-header-count"><strong>{{ total }}</strong><span>{{ $tr("匹配资产") }}</span></div>
    </header>
    <el-card shadow="never" class="main-card asset-surface">


      <!-- Search -->
      <div class="search-section asset-filters">
        <el-form :inline="true" :model="query" size="small">
          <el-form-item :label='$tr("关键字")'>
            <el-input v-model="query.keyword" :placeholder='$tr("SN/名称/IP")' clearable style="width:200px" @keyup.enter="search" />
          </el-form-item>
          <el-form-item :label='$tr("设备类型")'>
            <el-select v-model="query.deviceType" :placeholder='$tr("选择类型")' clearable style="width:140px">
              <el-option :label='$tr("路由器")' :value="1" />
              <el-option :label='$tr("交换机")' :value="2" />
              <el-option :label='$tr("防火墙")' :value="3" />
              <el-option :label='$tr("负载均衡")' :value="4" />
              <el-option :label='$tr("其他")' :value="5" />
            </el-select>
          </el-form-item>
          <el-form-item :label='$tr("机房")'>
            <el-select v-model="query.idcId" :placeholder='$tr("选择机房")' clearable filterable style="width:150px">
              <el-option v-for="d in idcList" :key="d.id" :label="$tr(d.name)" :value="d.id" />
            </el-select>
          </el-form-item>
          <el-form-item :label='$tr("品牌")'>
            <el-input v-model="query.brand" :placeholder='$tr("品牌")' clearable style="width:120px" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" size="small" @click="search">{{ $tr("查询") }}</el-button>
            <el-button size="small" @click="reset">{{ $tr("重置") }}</el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- Actions -->
      <div class="action-section asset-toolbar">
        <el-button type="primary" size="small" @click="showCreate">{{ $tr("+ 新增网络设备") }}</el-button>
      </div>

      <!-- Table -->
      <el-table :data="list" v-loading="loading" row-key="id" class="asset-table" style="width:100%">
        <el-table-column :label='$tr("设备")' min-width="230" fixed="left">
          <template #default="{row}"><div class="asset-identity"><span class="asset-avatar">NET</span><div class="asset-cell"><strong>{{ row.name }}</strong><span class="asset-muted asset-mono">{{ row.sn || '未登记序列号' }}</span></div></div></template>
        </el-table-column>
        <el-table-column :label='$tr("类型 / 型号")' min-width="165"><template #default="{row}"><div class="asset-cell"><span>{{ deviceTypeText(row.deviceType) }} · {{ row.brand || '—' }}</span><span class="asset-muted">{{ row.model || '未登记型号' }}</span></div></template></el-table-column>
        <el-table-column :label='$tr("管理地址")' min-width="165"><template #default="{row}"><div class="asset-cell"><span class="asset-mono">{{ row.manageIp || '—' }}</span><span class="asset-muted">{{ row.portNum || 0 }} {{ $tr("个端口 ·") }} {{ row.version || '版本未登记' }}</span></div></template></el-table-column>
        <el-table-column :label='$tr("位置")' min-width="150">
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
        <el-table-column :label='$tr("操作")' width="245" fixed="right">
          <template #default="{row}"><div class="asset-actions">
            <el-button type="primary" size="small" plain @click="openConnections(row)">{{ $tr("连接") }}</el-button>
            <el-button link type="primary" @click="openConfig(row)">{{ $tr("配置") }}</el-button>
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

    <AssetConnections v-model="connectionsVisible" asset-type="network" :asset="activeAsset" />
    <NetworkConfigDrawer v-model="configVisible" :asset="activeAsset" />
    <!-- Create/Edit Dialog -->
    <el-dialog :title="$tr(formTitle)" v-model="formVisible" width="min(720px, 94vw)" @close="resetForm">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="SN" prop="sn"><el-input v-model="form.sn" :placeholder='$tr("序列号")' /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label='$tr("设备名称")' prop="name"><el-input v-model="form.name" :placeholder='$tr("设备名称")' /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label='$tr("设备类型")' prop="deviceType">
              <el-select v-model="form.deviceType" style="width:100%">
                <el-option :label='$tr("路由器")' :value="1" />
                <el-option :label='$tr("交换机")' :value="2" />
                <el-option :label='$tr("防火墙")' :value="3" />
                <el-option :label='$tr("负载均衡")' :value="4" />
                <el-option :label='$tr("其他")' :value="5" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label='$tr("品牌")'><el-input v-model="form.brand" placeholder="Cisco/Huawei/H3C" /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label='$tr("型号")'><el-input v-model="form.model" :placeholder='$tr("如: S5700")' /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label='$tr("管理IP")'><el-input v-model="form.manageIp" :placeholder='$tr("管理IP")' /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label='$tr("固件版本")'><el-input v-model="form.version" :placeholder='$tr("版本号")' /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label='$tr("端口数量")'><el-input-number v-model="form.portNum" :min="0" :max="500" style="width:100%" /></el-form-item>
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
          <el-col :span="12">
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
          <el-col :span="12">
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
import NetworkConfigDrawer from './components/NetworkConfigDrawer.vue'

export default {
  components: { AssetConnections, NetworkConfigDrawer },
  name: 'NetworkDevice',
  data() {
    return {
      connectionsVisible: false, configVisible: false, activeAsset: null,
      loading: false, submitting: false,
      list: [], total: 0,
      idcList: [], cabinetOptions: [],
      query: { keyword: '', deviceType: undefined, idcId: undefined, brand: '', page: 1, size: 10 },
      formVisible: false, formTitle: '新增网络设备', isEdit: false, editId: null,
      form: { sn: '', name: '', deviceType: 2, brand: '', model: '', manageIp: '', version: '',
              portNum: 24, idcId: undefined, cabinetId: undefined, assetStatus: 2,
              purchaseDate: '', warrantyDate: '', vendor: '', remark: '' },
      rules: {
        sn: [{ required: true, get message() { return tr('请输入SN'); }, trigger: 'blur' }],
        name: [{ required: true, get message() { return tr('请输入设备名称'); }, trigger: 'blur' }],
        deviceType: [{ required: true, get message() { return tr('请选择设备类型'); }, trigger: 'change' }]
      }
    }
  },
  created() { this.fetch(); this.fetchIdcList() },
  methods: {
    openConnections(row) { this.activeAsset = row; this.connectionsVisible = true },
    openConfig(row) { this.activeAsset = row; this.configVisible = true },
    async fetch() {
      this.loading = true
      try {
        const res = await cmdbAPI.getNetworkDeviceList(this.query)
        if (res.data.code === 200) {
          const d = res.data.data
          this.list = d.list || []
          this.total = d.total || 0
        }
      } catch (e) { this.$message.error(tr('获取网络设备列表失败')) } finally { this.loading = false }
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
          if (this.isEdit) res = await cmdbAPI.updateNetworkDevice(this.editId, this.form)
          else res = await cmdbAPI.createNetworkDevice(this.form)
          if (res.data.code === 200) {
            this.$message.success(this.isEdit ? '更新成功' : tr('创建成功'))
            this.formVisible = false; this.fetch()
          } else { this.$message.error(res.data.message) }
        } catch (e) { this.$message.error('操作失败: ' + (e.response?.data?.message || e.message))
        } finally { this.submitting = false }
      })
    },
    handleDelete(row) {
      this.$confirm(`确定删除网络设备 ${row.name} 吗？`, tr('确认'), { type: 'warning' }).then(async () => {
        try {
          const res = await cmdbAPI.deleteNetworkDevice(row.id)
          if (res.data.code === 200) { this.$message.success(tr('删除成功')); this.fetch() }
          else this.$message.error(res.data.message)
        } catch (e) { this.$message.error(tr('删除失败')) }
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
.network-device-management { padding: 20px; min-height: 100vh; background: var(--bg-page); }
.main-card { border-radius: var(--radius-lg); }
.card-header .title { font-size: 18px; font-weight: 600; color: var(--text-primary); }
.search-section { background: var(--bg-card-alt); padding: 16px 20px; border-radius: var(--radius); margin-bottom: 16px; border: 1px solid var(--border-light); }
.action-section { margin-bottom: 16px; }
.pagination-section { display: flex; justify-content: center; margin-top: 20px; }
</style>
