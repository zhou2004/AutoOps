<template>
  <div class="asset-permission-management">
    <el-card shadow="hover" class="main-card">
      <template #header>
        <div class="card-header">
          <span class="title">资产授权管理</span>
          <el-tag type="info" effect="plain">类JumpServer · RBAC细粒度权限</el-tag>
        </div>
      </template>

      <!-- Search -->
      <div class="search-section">
        <el-form :inline="true" :model="query" size="small">
          <el-form-item label="规则名称">
            <el-input v-model="query.name" placeholder="规则名称" clearable style="width:180px" @keyup.enter="search" />
          </el-form-item>
          <el-form-item label="授权对象">
            <el-select v-model="query.subjectType" placeholder="全部" clearable style="width:120px" @change="search">
              <el-option label="用户" value="user" />
              <el-option label="用户组" value="group" />
            </el-select>
          </el-form-item>
          <el-form-item label="授权主体">
            <el-select v-model="query.subjectId" filterable clearable placeholder="选择用户/用户组" style="width:200px" @change="search"
              v-if="query.subjectType === 'user'">
              <el-option v-for="u in userList" :key="u.id" :label="`${u.username||''} (${u.nickname||''})`" :value="u.id" />
            </el-select>
            <el-select v-model="query.subjectId" filterable clearable placeholder="选择用户组" style="width:200px" @change="search"
              v-if="query.subjectType === 'group'">
              <el-option v-for="g in userGroupList" :key="g.id" :label="g.name" :value="g.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="query.isActive" placeholder="全部" clearable style="width:120px">
              <el-option label="启用" :value="1" />
              <el-option label="禁用" :value="0" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" size="small" @click="search">查询</el-button>
            <el-button size="small" @click="reset">重置</el-button>
          </el-form-item>
        </el-form>
      </div>

      <div class="action-section">
        <el-button type="primary" size="small" @click="showCreate">+ 新增授权规则</el-button>
      </div>

      <el-table :data="list" v-loading="loading" border stripe style="width:100%">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" label="授权名称" min-width="160" />
        <el-table-column label="授权主体" min-width="220">
          <template #default="{row}">
            <div>
              <el-tag size="small" type="primary" v-if="row.userIds && row.userIds!='[]'">
                用户: {{ resolveUserNames(row.userIds) }}
              </el-tag>
            </div>
            <div style="margin-top:2px">
              <el-tag size="small" type="success" v-if="row.groupIds && row.groupIds!='[]'">
                用户组: {{ resolveGroupNames(row.groupIds) }}
              </el-tag>
            </div>
            <span v-if="(!row.userIds||row.userIds=='[]')&&(!row.groupIds||row.groupIds=='[]')" style="color:#999">-</span>
          </template>
        </el-table-column>
        <el-table-column label="资产类型" min-width="140">
          <template #default="{row}">
            <el-tag size="small" v-for="t in safeJSON(row.assetTypes,[])" :key="t" style="margin:2px">{{ assetTypeLabel(t) }}</el-tag>
            <span v-if="!row.assetTypes||row.assetTypes=='[]'" style="color:#999">-</span>
          </template>
        </el-table-column>
        <el-table-column label="权限操作" min-width="200">
          <template #default="{row}">
            <el-tag size="small" v-for="a in safeJSON(row.permissionActions,[])" :key="a" :type="actionTag(a)" style="margin:2px">{{ actionText(a) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template #default="{row}">
            <el-tag :type="row.isActive===1?'success':'info'" size="small">{{ row.isActive===1?'启用':'禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="有效期" width="200">
          <template #default="{row}">
            <span v-if="row.dateStart || row.dateExpired">{{ row.dateStart||'永久' }} ~ {{ row.dateExpired||'永久' }}</span>
            <span v-else style="color:#999">永久</span>
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

    <!-- Create/Edit Dialog -->
    <el-dialog :title="formTitle" v-model="formVisible" width="900px" @close="resetForm" top="3vh">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="规则名称" prop="name">
              <el-input v-model="form.name" placeholder="如: 基础运维-主机管理权限" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="启用状态">
              <el-switch v-model="form.isActive" :active-value="1" :inactive-value="0" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="规则描述" />
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

        <el-form-item label="资产类型">
          <el-checkbox-group v-model="form.assetTypes">
            <el-checkbox label="host">主机</el-checkbox>
            <el-checkbox label="physical">物理机</el-checkbox>
            <el-checkbox label="network">网络设备</el-checkbox>
            <el-checkbox label="database">数据库</el-checkbox>
          </el-checkbox-group>
        </el-form-item>

        <el-form-item label="资产范围">
          <el-tabs type="border-card" size="small" style="width:100%">
            <el-tab-pane label="主机授权" v-if="form.assetTypes.includes('host')">
              <div style="margin-bottom:10px">
                <el-button size="mini" type="primary" @click="showHostSelector">选择主机</el-button>
                <el-tag size="mini" type="info" style="margin-left:8px">已选 {{ form.hostIds.length }} 台主机</el-tag>
                <el-tag size="mini" type="warning" style="margin-left:4px">已选 {{ form.hostGroupIds.length }} 个分组</el-tag>
              </div>
              <el-select v-model="form.hostGroupIds" multiple filterable placeholder="选择主机分组(业务线)" style="width:100%">
                <el-option v-for="g in groupList" :key="g.id" :label="g.name" :value="g.id" />
              </el-select>
              <div style="margin-top:8px;max-height:150px;overflow-y:auto" v-if="form.hostIds.length">
                <el-tag closable size="mini" v-for="id in form.hostIds" :key="id" style="margin:2px" @close="removeHost(id)">
                  {{ hostMap[id] || '主机#'+id }}
                </el-tag>
              </div>
            </el-tab-pane>
            <el-tab-pane label="物理机授权" v-if="form.assetTypes.includes('physical')">
              <div style="margin-bottom:10px">
                <el-button size="mini" type="primary" @click="showPhysicalSelector">选择物理机</el-button>
                <el-tag size="mini" type="info" style="margin-left:8px">已选 {{ form.physicalIds.length }} 台</el-tag>
              </div>
              <el-row :gutter="10">
                <el-col :span="8">
                  <el-select v-model="physicalFilter.idcId" filterable clearable placeholder="选择机房" style="width:100%" size="mini" @change="loadPhysicalList">
                    <el-option v-for="d in idcList" :key="d.id" :label="d.name" :value="d.id" />
                  </el-select>
                </el-col>
                <el-col :span="8">
                  <el-select v-model="physicalFilter.cabinetId" filterable clearable placeholder="选择机柜" style="width:100%" size="mini" :disabled="!physicalFilter.idcId" @change="loadPhysicalList">
                    <el-option v-for="c in cabinetList" :key="c.id" :label="c.name" :value="c.id" />
                  </el-select>
                </el-col>
                <el-col :span="8">
                  <el-input v-model="physicalFilter.keyword" placeholder="搜索名称/SN" size="mini" clearable @input="loadPhysicalList" />
                </el-col>
              </el-row>
              <el-table :data="physicalList" border stripe size="mini" style="width:100%;margin-top:8px" max-height="200" @selection-change="onPhysicalSelect">
                <el-table-column type="selection" width="40" />
                <el-table-column prop="sn" label="SN" width="120" />
                <el-table-column prop="hostName" label="名称" width="120" />
                <el-table-column prop="brand" label="品牌" width="80" />
                <el-table-column prop="manageIp" label="管理IP" width="120" />
              </el-table>
              <div style="margin-top:4px">
                <el-tag closable size="mini" v-for="id in form.physicalIds" :key="id" style="margin:2px" @close="removePhysical(id)">
                  {{ physicalMap[id] || '物理机#'+id }}
                </el-tag>
              </div>
            </el-tab-pane>
            <el-tab-pane label="网络设备授权" v-if="form.assetTypes.includes('network')">
              <div style="margin-bottom:10px">
                <el-button size="mini" type="primary" @click="showNetworkSelector">选择网络设备</el-button>
                <el-tag size="mini" type="info" style="margin-left:8px">已选 {{ form.networkIds.length }} 台</el-tag>
              </div>
              <el-row :gutter="10">
                <el-col :span="6">
                  <el-select v-model="networkFilter.idcId" filterable clearable placeholder="选择机房" style="width:100%" size="mini" @change="loadNetworkList">
                    <el-option v-for="d in idcList" :key="d.id" :label="d.name" :value="d.id" />
                  </el-select>
                </el-col>
                <el-col :span="6">
                  <el-select v-model="networkFilter.deviceType" clearable placeholder="设备类型" style="width:100%" size="mini" @change="loadNetworkList">
                    <el-option label="路由器" :value="1" />
                    <el-option label="交换机" :value="2" />
                    <el-option label="防火墙" :value="3" />
                    <el-option label="负载均衡" :value="4" />
                    <el-option label="其他" :value="5" />
                  </el-select>
                </el-col>
                <el-col :span="6">
                  <el-input v-model="networkFilter.keyword" placeholder="搜索名称" size="mini" clearable @input="loadNetworkList" />
                </el-col>
              </el-row>
              <el-table :data="networkList" border stripe size="mini" style="width:100%;margin-top:8px" max-height="200" @selection-change="onNetworkSelect">
                <el-table-column type="selection" width="40" />
                <el-table-column prop="name" label="名称" width="120" />
                <el-table-column label="类型" width="80">
                  <template #default="{row}">{{ deviceTypeLabel(row.deviceType) }}</template>
                </el-table-column>
                <el-table-column prop="manageIp" label="管理IP" width="120" />
                <el-table-column prop="brand" label="品牌" width="80" />
              </el-table>
              <div style="margin-top:4px">
                <el-tag closable size="mini" v-for="id in form.networkIds" :key="id" style="margin:2px" @close="removeNetwork(id)">
                  {{ networkMap[id] || '设备#'+id }}
                </el-tag>
              </div>
            </el-tab-pane>
            <el-tab-pane label="机房授权">
              <el-select v-model="form.idcIds" multiple filterable placeholder="选择机房（含其下所有资产）" style="width:100%">
                <el-option v-for="d in idcList" :key="d.id" :label="d.name" :value="d.id" />
              </el-select>
            </el-tab-pane>
          </el-tabs>
        </el-form-item>

        <el-form-item label="权限操作">
          <el-checkbox-group v-model="form.permissionActions">
            <el-checkbox label="get">查看详情</el-checkbox>
            <el-checkbox label="list">查看列表</el-checkbox>
            <el-checkbox label="connect">SSH连接</el-checkbox>
            <el-checkbox label="create">创建</el-checkbox>
            <el-checkbox label="update">修改</el-checkbox>
            <el-checkbox label="delete">删除</el-checkbox>
            <el-checkbox label="admin">管理</el-checkbox>
          </el-checkbox-group>
        </el-form-item>

        <el-row :gutter="20">
          <el-col :span="8">
            <el-form-item label="生效日期"><el-input v-model="form.dateStart" placeholder="YYYY-MM-DD" /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="失效日期"><el-input v-model="form.dateExpired" placeholder="YYYY-MM-DD" /></el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="formVisible=false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>

    <!-- Host Selector Dialog -->
    <el-dialog title="选择主机" v-model="hostSelectorVisible" width="800px" top="5vh">
      <el-row :gutter="10" style="margin-bottom:10px">
        <el-col :span="8">
          <el-select v-model="hostFilter.groupId" filterable clearable placeholder="选择业务线/分组" style="width:100%" size="small" @change="loadHostList">
            <el-option v-for="g in flatGroupList" :key="g.id" :label="g.name" :value="g.id" />
          </el-select>
        </el-col>
        <el-col :span="8">
          <el-input v-model="hostFilter.keyword" placeholder="搜索主机名/IP" size="small" clearable @input="loadHostList" />
        </el-col>
        <el-col :span="4">
          <el-select v-model="hostFilter.status" clearable placeholder="状态" size="small" style="width:100%" @change="loadHostList">
            <el-option label="认证成功" :value="1" />
            <el-option label="未认证" :value="2" />
            <el-option label="认证失败" :value="3" />
          </el-select>
        </el-col>
        <el-col :span="4">
          <el-button size="small" @click="selectAllHosts">全选本页</el-button>
        </el-col>
      </el-row>
      <el-table :data="hostList" border stripe size="mini" style="width:100%" max-height="350" ref="hostTable"
        @selection-change="onHostSelect">
        <el-table-column type="selection" width="40" />
        <el-table-column prop="hostName" label="主机名" width="140" />
        <el-table-column prop="sshIp" label="SSH-IP" width="130" />
        <el-table-column prop="privateIp" label="私网IP" width="130" />
        <el-table-column label="状态" width="80">
          <template #default="{row}">{{ {1:'正常',2:'未认证',3:'失败'}[row.status]||'未知' }}</template>
        </el-table-column>
        <el-table-column label="分组" min-width="120">
          <template #default="{row}">{{ row.group?.name || '-' }}</template>
        </el-table-column>
      </el-table>
      <div style="text-align:center;margin-top:10px">
        <el-pagination small @current-change="p=>{hostPage=p;loadHostList()}" :current-page="hostPage" :page-size="10" layout="prev, pager, next" :total="hostTotal" />
      </div>
      <template #footer>
        <el-button @click="hostSelectorVisible=false">关闭</el-button>
        <el-button type="primary" @click="confirmHostSelect">确定选择 ({{ selectedHostIds.length }})</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import cmdbAPI from '@/api/cmdb'
import systemApi from '@/api/system'

const DEVICE_TYPE_MAP = {1:'路由器',2:'交换机',3:'防火墙',4:'负载均衡',5:'其他'}
const ASSET_TYPE_MAP = {host:'主机',physical:'物理机',network:'网络设备',database:'数据库'}
const ACTION_MAP = {get:'查看',list:'列表',connect:'连接',create:'创建',update:'修改',delete:'删除',admin:'管理'}
const ACTION_TAG = {get:'',list:'info',connect:'primary',create:'success',update:'warning',delete:'danger',admin:'danger'}

export default {
  name: 'AssetPermission',
  data() {
    return {
      loading: false, submitting: false,
      list: [], total: 0,
      userList: [], userGroupList: [], groupList: [], idcList: [],
      cabinetList: [], flatGroupList: [],
      query: { name: '', subjectType: undefined, subjectId: undefined, isActive: undefined, page: 1, size: 10 },
      formVisible: false, formTitle: '新增授权规则', isEdit: false, editId: null,
      hostSelectorVisible: false, hostList: [], hostTotal: 0, hostPage: 1,
      selectedHostIds: [], hostFilter: { groupId: undefined, keyword: '', status: undefined },
      hostMap: {},
      physicalFilter: { idcId: undefined, cabinetId: undefined, keyword: '' },
      physicalList: [], physicalMap: {},
      networkFilter: { idcId: undefined, deviceType: undefined, keyword: '' },
      networkList: [], networkMap: {},
      form: {
        name: '', description: '', userIds: [], groupIds: [],
        assetTypes: ['host'], hostGroupIds: [], hostIds: [],
        physicalIds: [], networkIds: [], databaseIds: [], idcIds: [],
        permissionActions: ['get','list'], isActive: 1, dateStart: '', dateExpired: ''
      },
      rules: { name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }] }
    }
  },
  created() { this.fetch(); this.loadOptions() },
  methods: {
    async fetch() {
      this.loading = true
      try {
        const res = await cmdbAPI.getAssetPermissionList(this.query)
        if (res.data.code === 200) {
          this.list = res.data.data.list || []
          this.total = res.data.data.total || 0
        }
      } catch (e) { this.$message.error('获取授权列表失败') } finally { this.loading = false }
    },
    safeJSON(str, def) { try { return JSON.parse(str) || def } catch(e) { return def } },
    assetTypeLabel(t) { return ASSET_TYPE_MAP[t] || t },
    actionText(a) { return ACTION_MAP[a] || a },
    actionTag(a) { return ACTION_TAG[a] || 'info' },
    deviceTypeLabel(t) { return DEVICE_TYPE_MAP[t] || '未知' },
    async loadOptions() {
      try {
        const [res1, res2, res3, res4, res5] = await Promise.all([
          systemApi.queryAdminList({ page: 1, size: 1000 }),
          cmdbAPI.getAllCmdbGroups(),
          cmdbAPI.getIdcAll(),
          cmdbAPI.getCmdbUserGroupAll(),
          cmdbAPI.getHostList({ page: 1, pageSize: 1000 })
        ])
        if (res1.data.code === 200) {
            this.userList = res1.data.data.list || []
            console.log('用户列表', this.userList)
        }
        if (res2.data && res2.data.data) {
          const groups = Array.isArray(res2.data.data) ? res2.data.data : res2.data.data.list || []
          this.groupList = groups
          this.flatGroupList = this.flattenGroups(groups)
        }
        if (res3.data.code === 200) this.idcList = res3.data.data.list || []
        if (res4.data && res4.data.data) this.userGroupList = res4.data.data.list || []
        if (res5.data && res5.data.data) {
          const hosts = res5.data.data.list || res5.data.data || []
          hosts.forEach(h => { this.hostMap[h.id] = h.hostName || h.name })
        }
      } catch (_) {}
    },
    flattenGroups(groups, prefix='') {
      let result = []
      for (const g of groups) {
        result.push({ ...g, name: prefix + g.name })
        if (g.children) result = result.concat(this.flattenGroups(g.children, prefix + g.name + '/'))
      }
      return result
    },
    async loadHostList() {
      try {
        const params = { page: this.hostPage, pageSize: 10 }
        if (this.hostFilter.groupId) params.groupId = this.hostFilter.groupId
        if (this.hostFilter.keyword) params.name = this.hostFilter.keyword
        if (this.hostFilter.status) params.status = this.hostFilter.status
        const res = await cmdbAPI.getCmdbHostList(params)
        if (res.data.code === 200) {
          this.hostList = res.data.data.list || []
          this.hostTotal = res.data.data.total || 0
        }
      } catch (_) { this.hostList = []; this.hostTotal = 0 }
    },
    async loadPhysicalList() {
      try {
        const params = { page: 1, size: 200 }
        if (this.physicalFilter.idcId) params.idcId = this.physicalFilter.idcId
        if (this.physicalFilter.cabinetId) params.cabinetId = this.physicalFilter.cabinetId
        if (this.physicalFilter.keyword) params.keyword = this.physicalFilter.keyword
        const res = await cmdbAPI.getPhysicalMachineList(params)
        if (res.data.code === 200) {
          this.physicalList = res.data.data.list || []
          this.physicalList.forEach(p => { this.physicalMap[p.id] = p.hostName || p.sn })
        }
      } catch (_) { this.physicalList = [] }
    },
    async loadNetworkList() {
      try {
        const params = { page: 1, size: 200 }
        if (this.networkFilter.idcId) params.idcId = this.networkFilter.idcId
        if (this.networkFilter.deviceType) params.deviceType = this.networkFilter.deviceType
        if (this.networkFilter.keyword) params.keyword = this.networkFilter.keyword
        const res = await cmdbAPI.getNetworkDeviceList(params)
        if (res.data.code === 200) {
          this.networkList = res.data.data.list || []
          this.networkList.forEach(n => { this.networkMap[n.id] = n.name })
        }
      } catch (_) { this.networkList = [] }
    },
    search() { this.query.page = 1; this.fetch() },
    reset() { this.query = { name: '', subjectType: undefined, subjectId: undefined, isActive: undefined, page: 1, size: 10 }; this.fetch() },
    showCreate() {
      this.isEdit = false; this.editId = null; this.formTitle = '新增授权规则'
      this.form = { name: '', description: '', userIds: [], groupIds: [], assetTypes: ['host'],
        hostGroupIds: [], hostIds: [], physicalIds: [], networkIds: [], databaseIds: [], idcIds: [],
        permissionActions: ['get','list'], isActive: 1, dateStart: '', dateExpired: '' }
      this.formVisible = true
    },
    showEdit(row) {
      this.isEdit = true; this.editId = row.id; this.formTitle = '编辑授权规则'
      this.form = {
        name: row.name, description: row.description || '',
        userIds: this.safeJSON(row.userIds, []),
        groupIds: this.safeJSON(row.groupIds, []),
        assetTypes: this.safeJSON(row.assetTypes, ['host']),
        hostGroupIds: this.safeJSON(row.hostGroupIds, []),
        hostIds: this.safeJSON(row.hostIds, []),
        physicalIds: this.safeJSON(row.physicalIds, []),
        networkIds: this.safeJSON(row.networkIds, []),
        databaseIds: this.safeJSON(row.databaseIds, []),
        idcIds: this.safeJSON(row.idcIds, []),
        permissionActions: this.safeJSON(row.permissionActions, ['get','list']),
        isActive: row.isActive, dateStart: row.dateStart || '', dateExpired: row.dateExpired || ''
      }
      this.formVisible = true
    },
    resetForm() { this.$refs.formRef?.resetFields() },
    async handleSubmit() {
      this.$refs.formRef.validate(async v => {
        if (!v) return
        this.submitting = true
        try {
          let res
          if (this.isEdit) res = await cmdbAPI.updateAssetPermission(this.editId, this.form)
          else res = await cmdbAPI.createAssetPermission(this.form)
          if (res.data.code === 200) {
            this.$message.success(this.isEdit ? '更新成功' : '创建成功')
            this.formVisible = false; this.fetch()
          } else { this.$message.error(res.data.message) }
        } catch (e) { this.$message.error('操作失败: ' + (e.response?.data?.message || e.message))
        } finally { this.submitting = false }
      })
    },
    handleDelete(row) {
      this.$confirm(`确定删除授权规则 "${row.name}" 吗？`, '确认', { type: 'warning' }).then(async () => {
        try {
          const res = await cmdbAPI.deleteAssetPermission(row.id)
          if (res.data.code === 200) { this.$message.success('删除成功'); this.fetch() }
          else this.$message.error(res.data.message)
        } catch (e) { this.$message.error('删除失败') }
      }).catch(() => {})
    },
    // Host Selection
    showHostSelector() {
      this.hostSelectorVisible = true
      this.hostFilter = { groupId: undefined, keyword: '', status: undefined }
      this.hostPage = 1; this.selectedHostIds = []
      this.loadHostList()
    },
    onHostSelect(rows) { this.selectedHostIds = rows.map(r => r.id) },
    selectAllHosts() {
      this.$nextTick(() => {
        if (this.$refs.hostTable) {
          this.hostList.forEach(row => { this.$refs.hostTable.toggleRowSelection(row, true) })
        }
      })
    },
    confirmHostSelect() {
      const existing = new Set(this.form.hostIds)
      this.selectedHostIds.forEach(id => existing.add(id))
      this.form.hostIds = Array.from(existing)
      this.hostSelectorVisible = false
      this.$message.success(`已选择 ${this.selectedHostIds.length} 台主机`)
    },
    removeHost(id) { this.form.hostIds = this.form.hostIds.filter(i => i !== id) },
    showPhysicalSelector() {
      this.physicalFilter = { idcId: undefined, cabinetId: undefined, keyword: '' }
      this.loadPhysicalList()
    },
    onPhysicalSelect(rows) {
      rows.forEach(r => {
        if (!this.form.physicalIds.includes(r.id)) this.form.physicalIds.push(r.id)
      })
    },
    removePhysical(id) { this.form.physicalIds = this.form.physicalIds.filter(i => i !== id) },
    showNetworkSelector() {
      this.networkFilter = { idcId: undefined, deviceType: undefined, keyword: '' }
      this.loadNetworkList()
    },
    onNetworkSelect(rows) {
      rows.forEach(r => {
        if (!this.form.networkIds.includes(r.id)) this.form.networkIds.push(r.id)
      })
    },
    removeNetwork(id) { this.form.networkIds = this.form.networkIds.filter(i => i !== id) },
    async loadCabinetByIdc(idcId) {
      if (!idcId) { this.cabinetList = []; return }
      try {
        const res = await cmdbAPI.getCabinetByIdc(idcId)
        if (res.data.code === 200) this.cabinetList = res.data.data.list || []
      } catch (_) { this.cabinetList = [] }
    },
    // 将JSON数组格式的userIds解析为用户名称字符串
    resolveUserNames(userIdsJSON) {
      const ids = this.safeJSON(userIdsJSON, [])
      return ids.map(id => {
        const u = this.userList.find(u => u.id === id)
        return u ? (u.nickname || u.username) : '用户#'+id
      }).join(', ') || '-'
    },
    // 将JSON数组格式的groupIds解析为用户组名称字符串
    resolveGroupNames(groupIdsJSON) {
      const ids = this.safeJSON(groupIdsJSON, [])
      return ids.map(id => {
        const g = this.userGroupList.find(g => g.id === id)
        return g ? g.name : '组#'+id
      }).join(', ') || '-'
    }
  },
  watch: {
    'physicalFilter.idcId'(val) {
      if (val) this.loadCabinetByIdc(val)
    },
    'networkFilter.idcId'(val) {
      if (val) this.loadCabinetByIdc(val)
    }
  }
}
</script>

<style scoped>
.asset-permission-management { padding: 20px; min-height: 100vh; background: var(--bg-page); }
.main-card { border-radius: var(--radius-lg); }
.card-header .title { font-size: 18px; font-weight: 600; color: var(--text-primary); }
.search-section { background: var(--bg-card-alt); padding: 16px 20px; border-radius: var(--radius); margin-bottom: 16px; border: 1px solid var(--border-light); }
.action-section { margin-bottom: 16px; }
.pagination-section { display: flex; justify-content: center; margin-top: 20px; }
</style>
