<template>
  <div class="incident-page">
    <!-- 统计大屏幕 -->
    <div class="stats-grid">
      <el-row :gutter="20">
        <el-col :span="6"><div class="stat-card stat-firing"><div class="stat-num">{{ stats.totalFiring }}</div><div class="stat-label">活跃故障</div></div></el-col>
        <el-col :span="6"><div class="stat-card stat-resolved"><div class="stat-num">{{ stats.totalResolved }}</div><div class="stat-label">已解决</div></div></el-col>
        <el-col :span="6"><div class="stat-card stat-24h"><div class="stat-num">{{ stats.last24hCount }}</div><div class="stat-label">24小时新增</div></div></el-col>
        <el-col :span="6"><div class="stat-card stat-today"><div class="stat-num">{{ stats.todayCount }}</div><div class="stat-label">今日新增</div></div></el-col>
      </el-row>
      <el-row :gutter="20" style="margin-top:20px">
        <el-col :span="12">
          <div class="stat-card-sub">
            <div class="sub-title">按告警等级</div>
            <div class="level-bars">
              <div class="level-item" v-for="(v,k) in stats.byLevel" :key="k">
                <span class="level-tag" :class="'level-'+k">{{ levelLabel(k) }}</span>
                <el-progress :percentage="levelPercent(v)" :color="levelColor(k)" :stroke-width="20" :text-inside="true" />
              </div>
              <div v-if="!Object.keys(stats.byLevel).length" class="empty-hint">暂无数据</div>
            </div>
          </div>
        </el-col>
        <el-col :span="12">
          <div class="stat-card-sub">
            <div class="sub-title">按来源</div>
            <div class="source-chart">
              <div class="source-item" v-for="(v,k) in stats.bySource" :key="k">
                <span class="source-tag">{{ sourceLabel(k) }}</span>
                <el-progress :percentage="sourcePercent(v)" :stroke-width="20" :text-inside="true" />
              </div>
              <div v-if="!Object.keys(stats.bySource).length" class="empty-hint">暂无数据</div>
            </div>
          </div>
        </el-col>
      </el-row>
    </div>

    <!-- 故障列表 -->
    <el-card shadow="hover" class="incident-card" style="margin-top:20px">
      <template #header><div class="card-header"><span class="title">故障列表</span></div></template>
      <div class="search-section">
        <el-form :model="q" :inline="true">
          <el-form-item label="状态"><el-select v-model="q.status" clearable size="small" style="width:130px" placeholder="状态"><el-option label="活跃" value="firing" /><el-option label="已解决" value="resolved" /></el-select></el-form-item>
          <el-form-item label="等级"><el-select v-model="q.level" clearable size="small" style="width:130px" placeholder="等级"><el-option label="严重" value="critical" /><el-option label="警告" value="warning" /><el-option label="信息" value="info" /></el-select></el-form-item>
          <el-form-item label="来源"><el-select v-model="q.source" clearable size="small" style="width:150px" placeholder="来源"><el-option label="域名证书" value="domain_cert" /><el-option label="API监控" value="api_endpoint" /><el-option label="Prometheus" value="prometheus" /></el-select></el-form-item>
          <el-form-item><el-button type="primary" icon="Search" size="small" @click="handleQuery">搜索</el-button><el-button icon="Refresh" size="small" @click="resetQuery">重置</el-button></el-form-item>
        </el-form>
      </div>
      <el-table v-loading="loading" :data="list" stripe>
        <el-table-column label="ID" prop="id" width="60" />
        <el-table-column label="标题" prop="title" min-width="200" show-overflow-tooltip />
        <el-table-column label="来源" width="120">
          <template #default="s"><el-tag size="small">{{ sourceLabel(s.row.source) }}</el-tag></template>
        </el-table-column>
        <el-table-column label="等级" width="90">
          <template #default="s"><el-tag :type="levelType(s.row.level)" size="small" effect="dark">{{ levelLabel(s.row.level) }}</el-tag></template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="s"><el-tag :type="s.row.status==='firing'?'danger':'success'" size="small" effect="dark">{{ s.row.status==='firing'?'活跃':'已解决' }}</el-tag></template>
        </el-table-column>
        <el-table-column label="告警时间" prop="alertTime" min-width="150" />
        <el-table-column label="解决时间" prop="resolvedAt" min-width="150" />
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="s">
            <el-button v-if="s.row.status==='firing'" type="success" icon="Check" size="small" circle @click="handleResolve(s.row)" />
            <el-button type="primary" icon="View" size="small" circle @click="handleDetail(s.row)" />
            <el-button type="danger" icon="Delete" size="small" circle @click="handleDelete(s.row)" />
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-section">
        <el-pagination @size-change="s=>{q.pageSize=s;getList()}" @current-change="p=>{q.page=p;getList()}" :current-page="q.page" :page-sizes="[10,20,50,100]" :page-size="q.pageSize" layout="total, sizes, prev, pager, next, jumper" :total="total" />
      </div>
    </el-card>

    <el-dialog v-model="detailVisible" title="故障详情" width="600px">
      <div v-if="detail" class="cert-detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="标题">{{ detail.title }}</el-descriptions-item>
          <el-descriptions-item label="来源">{{ sourceLabel(detail.source) }}</el-descriptions-item>
          <el-descriptions-item label="等级"><el-tag :type="levelType(detail.level)" size="small">{{ levelLabel(detail.level) }}</el-tag></el-descriptions-item>
          <el-descriptions-item label="状态"><el-tag :type="detail.status==='firing'?'danger':'success'" size="small">{{ detail.status==='firing'?'活跃':'已解决' }}</el-tag></el-descriptions-item>
          <el-descriptions-item label="描述">{{ detail.description || '-' }}</el-descriptions-item>
          <el-descriptions-item label="告警时间">{{ detail.alertTime }}</el-descriptions-item>
          <el-descriptions-item label="解决时间">{{ detail.resolvedAt || '-' }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ detail.createTime }}</el-descriptions-item>
        </el-descriptions>
      </div>
    </el-dialog>
  </div>
</template>
<script>
import { getIncidentList, getIncidentStats, resolveIncident, deleteIncident } from '@/api/monitor'
export default {
  data() {
    return {
      loading: false, list: [], total: 0, stats: { totalFiring: 0, totalResolved: 0, byLevel: {}, bySource: {}, last24hCount: 0, todayCount: 0 },
      q: { page: 1, pageSize: 20, status: '', level: '', source: '' },
      detailVisible: false, detail: null
    }
  },
  methods: {
    async getList() { this.loading = true; try { const { data: r } = await getIncidentList(this.q); if (r.code === 200) { this.list = r.data.list || []; this.total = r.data.total || 0 } } catch { this.$message.error('获取列表失败') } finally { this.loading = false } },
    async getStats() { try { const { data: r } = await getIncidentStats(); if (r.code === 200) { this.stats = r.data } } catch { console.error('获取统计失败') } },
    handleQuery() { this.q.page = 1; this.getList() },
    resetQuery() { this.q = { page: 1, pageSize: 20, status: '', level: '', source: '' }; this.getList() },
    async handleResolve(r) { const c = await this.$confirm(`确认解决该故障?`, '提示', { type: 'warning' }).catch(e => e); if (c !== 'confirm') return; try { const { data: r2 } = await resolveIncident(r.id); if (r2.code !== 200) { this.$message.error(r2.message) } else { this.$message.success('已解决'); this.getList(); this.getStats() } } catch { this.$message.error('操作失败') } },
    async handleDelete(r) { const c = await this.$confirm(`确认删除?`, '提示', { type: 'warning' }).catch(e => e); if (c !== 'confirm') return; try { const { data: r2 } = await deleteIncident(r.id); if (r2.code !== 200) { this.$message.error(r2.message) } else { this.$message.success('删除成功'); this.getList(); this.getStats() } } catch { this.$message.error('删除失败') } },
    handleDetail(r) { this.detail = r; this.detailVisible = true },
    levelLabel(l) { return { critical: '严重', warning: '警告', info: '信息' }[l] || l },
    levelType(l) { return { critical: 'danger', warning: 'warning', info: 'info' }[l] || 'info' },
    levelColor(l) { return { critical: '#f56c6c', warning: '#e6a23c', info: 'var(--text-secondary)' }[l] || '#409eff' },
    levelPercent(v) { const t = Object.values(this.stats.byLevel).reduce((a, b) => a + b, 0); return t ? Math.round(v / t * 100) : 0 },
    sourceLabel(s) { return { domain_cert: '域名证书', api_endpoint: 'API监控', prometheus: 'Prometheus' }[s] || s },
    sourcePercent(v) { const t = Object.values(this.stats.bySource).reduce((a, b) => a + b, 0); return t ? Math.round(v / t * 100) : 0 }
  },
  created() { this.getList(); this.getStats() }
}
</script>
<style scoped>
.incident-page { padding: 20px; min-height: 100vh; background: var(--bg-page); }
.stats-grid { margin-bottom: 0; }
.stat-card { background: rgba(255,255,255,0.95); backdrop-filter: blur(10px); border-radius: 16px; padding: 30px 20px; text-align: center; box-shadow: 0 8px 32px rgba(0,0,0,0.1); }
.stat-num { font-size: 48px; font-weight: 700; color: var(--text-primary); }
.stat-label { font-size: 14px; color: var(--text-secondary); margin-top: 8px; }
.stat-firing { border-left: 5px solid #f56c6c; }
.stat-resolved { border-left: 5px solid #67c23a; }
.stat-24h { border-left: 5px solid #e6a23c; }
.stat-today { border-left: 5px solid #409eff; }
.stat-card-sub { background: rgba(255,255,255,0.95); backdrop-filter: blur(10px); border-radius: 16px; padding: 20px; box-shadow: 0 8px 32px rgba(0,0,0,0.1); min-height: 150px; }
.sub-title { font-size: 16px; font-weight: 600; color: var(--text-primary); margin-bottom: 16px; }
.level-item, .source-item { margin-bottom: 12px; display: flex; align-items: center; gap: 12px; }
.level-tag, .source-tag { display: inline-block; padding: 2px 10px; border-radius: 4px; font-size: 12px; font-weight: 600; min-width: 50px; text-align: center; }
.level-critical { background: #fef0f0; color: #f56c6c; }
.level-warning { background: #fdf6ec; color: #e6a23c; }
.level-info { background: #f4f4f5; color: var(--text-secondary); }
.source-tag { background: #ecf5ff; color: #409eff; }
.level-item .el-progress, .source-item .el-progress { flex: 1; }
.empty-hint { color: #c0c4cc; text-align: center; padding: 30px 0; }
.incident-card { background: rgba(255,255,255,0.95); backdrop-filter: blur(10px); border-radius: 16px; box-shadow: 0 8px 32px rgba(0,0,0,0.1); }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 20px; font-weight: 600; background: linear-gradient(45deg,#667eea,#764ba2); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
.search-section { margin-bottom: 20px; }
.pagination-section { display: flex; justify-content: center; padding: 20px 0; }
.cert-detail { padding: 10px; }
.el-button { border-radius: 8px; }
.el-tag { border-radius: 8px; border: none; }
</style>