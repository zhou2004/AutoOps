<template>
  <div class="my-assets">
    <el-card shadow="hover" class="main-card">
      <template #header>
        <div class="card-header">
          <span class="title">我的授权资产</span>
          <el-tag type="success" effect="plain" v-if="permissionActions.length">权限: {{ permissionActions.join(', ') }}</el-tag>
        </div>
      </template>

      <el-tabs v-model="activeTab">
        <el-tab-pane label="主机" name="host">
          <el-table :data="hostList" v-loading="loading" border stripe size="small" style="width:100%">
            <el-table-column prop="hostName" label="主机名" min-width="140" />
            <el-table-column prop="sshIp" label="SSH-IP" width="130" />
            <el-table-column prop="privateIp" label="私网IP" width="130" />
            <el-table-column prop="os" label="操作系统" width="120" />
            <el-table-column label="状态" width="80" align="center">
              <template #default="{row}">{{ {1:'正常',2:'未认证',3:'失败'}[row.status]||'未知' }}</template>
            </el-table-column>
            <el-table-column label="操作" width="120" align="center">
              <template #default="{row}">
                <el-button type="primary" size="small" v-if="hasAction('connect')" @click="connectSSH(row)">连接</el-button>
                <el-tag v-else size="small" type="info">只读</el-tag>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!hostList.length && !loading" description="暂无授权主机" />
        </el-tab-pane>

        <el-tab-pane label="物理机" name="physical">
          <el-table :data="physicalList" v-loading="loading" border stripe size="small" style="width:100%">
            <el-table-column prop="sn" label="SN" width="140" />
            <el-table-column prop="hostName" label="主机名" width="140" />
            <el-table-column prop="manageIp" label="管理IP" width="130" />
            <el-table-column prop="brand" label="品牌" width="100" />
            <el-table-column prop="model" label="型号" width="120" />
          </el-table>
          <el-empty v-if="!physicalList.length && !loading" description="暂无授权物理机" />
        </el-tab-pane>

        <el-tab-pane label="网络设备" name="network">
          <el-table :data="networkList" v-loading="loading" border stripe size="small" style="width:100%">
            <el-table-column prop="name" label="设备名称" width="140" />
            <el-table-column label="类型" width="100">
              <template #default="{row}">{{ {1:'路由器',2:'交换机',3:'防火墙',4:'负载均衡',5:'其他'}[row.deviceType]||'未知' }}</template>
            </el-table-column>
            <el-table-column prop="manageIp" label="管理IP" width="130" />
            <el-table-column prop="brand" label="品牌" width="100" />
          </el-table>
          <el-empty v-if="!networkList.length && !loading" description="暂无授权网络设备" />
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script>
import cmdbAPI from '@/api/cmdb'

export default {
  name: 'MyAssets',
  data() {
    return {
      loading: false, activeTab: 'host',
      hostList: [], physicalList: [], networkList: [],
      permissionActions: []
    }
  },
  created() { this.fetch() },
  methods: {
    async fetch() {
      this.loading = true
      try {
        const res = await cmdbAPI.getMyAssetPermissions()
        if (res.data.code === 200) {
          const d = res.data.data
          // 加载完整数据
          this.loadHosts(d.allowedHostIds || [])
          this.loadPhysicals(d.allowedPhysicalIds || [])
          this.loadNetworks(d.allowedNetworkIds || [])
          this.permissionActions = d.permissionActions || []
        }
      } catch (e) { this.$message.error('获取授权资产失败') } finally { this.loading = false }
    },
    async loadHosts(ids) {
      if (!ids.length) { this.hostList = []; return }
      try {
        const res = await cmdbAPI.getCmdbHostList({ page: 1, pageSize: 1000 })
        if (res.data.code === 200) {
          const all = res.data.data.list || []
          const idSet = new Set(ids.map(Number))
          this.hostList = all.filter(h => idSet.has(h.id))
        }
      } catch (_) { this.hostList = [] }
    },
    async loadPhysicals(ids) {
      if (!ids.length) { this.physicalList = []; return }
      try {
        const res = await cmdbAPI.getPhysicalMachineAll()
        if (res.data.code === 200) {
          const all = res.data.data.list || []
          const idSet = new Set(ids.map(Number))
          this.physicalList = all.filter(p => idSet.has(p.id))
        }
      } catch (_) { this.physicalList = [] }
    },
    async loadNetworks(ids) {
      if (!ids.length) { this.networkList = []; return }
      try {
        const res = await cmdbAPI.getNetworkDeviceAll()
        if (res.data.code === 200) {
          const all = res.data.data.list || []
          const idSet = new Set(ids.map(Number))
          this.networkList = all.filter(n => idSet.has(n.id))
        }
      } catch (_) { this.networkList = [] }
    },
    hasAction(action) {
      return this.permissionActions.includes(action) || this.permissionActions.includes('admin')
    },
    connectSSH(row) {
      this.$router.push({ path: '/cmdb/ssh', query: { hostId: row.id, hostName: row.hostName } })
    }
  }
}
</script>

<style scoped>
.my-assets { padding: 20px; min-height: 100vh; background: #f0f2f5; }
.main-card { border-radius: 12px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.card-header .title { font-size: 18px; font-weight: 600; color: #303133; }
</style>
