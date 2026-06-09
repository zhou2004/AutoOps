<template>
  <div class="dashboard-content">
    <div class="welcome-banner" style="background:var(--gradient-primary);border-radius:var(--radius-lg);padding:24px 32px;color:#fff;display:flex;align-items:center;justify-content:space-between;margin-bottom:16px">
      <div>
        <div style="font-size:20px;font-weight:700;margin-bottom:4px">👋 早上好</div>
        <div style="font-size:13px;opacity:.9">今日待处理：<span style="font-weight:700;background:rgba(255,255,255,.2);padding:0 6px;border-radius:3px">3</span> 个告警 · <span style="font-weight:700;background:rgba(255,255,255,.2);padding:0 6px;border-radius:3px">12</span> 个待审批工单</div>
      </div>
      <div style="display:flex;gap:24px">
        <div style="text-align:center"><div style="font-size:24px;font-weight:700">99.96<small style="font-size:12px;opacity:.8">%</small></div><div style="font-size:12px;opacity:.8">系统可用率</div></div>
        <div style="text-align:center"><div style="font-size:24px;font-weight:700">1,286</div><div style="font-size:12px;opacity:.8">管理主机</div></div>
        <div style="text-align:center"><div style="font-size:24px;font-weight:700">1.2<small style="font-size:12px;opacity:.8">ms</small></div><div style="font-size:12px;opacity:.8">平均响应</div></div>
      </div>
    </div>
    <AssetModule />
    <DeployModule />
    <MonitorModule />
    <AlertModule />
    <UserTicketModule />
    <AIModule />
    <div style="text-align:center;padding:20px;font-size:12px;color:var(--text-secondary)">AutoOps v2.0 · 智能运维管理平台 © 2026</div>
  </div>
</template>

<script>
import AssetModule from './components/AssetModule.vue'
import DeployModule from './components/DeployModule.vue'
import MonitorModule from './components/MonitorModule.vue'
import AlertModule from './components/AlertModule.vue'
import UserTicketModule from './components/UserTicketModule.vue'
import AIModule from './components/AIModule.vue'

export default {
  name: 'DashboardNew',
  components: { AssetModule, DeployModule, MonitorModule, AlertModule, UserTicketModule, AIModule }
}
</script>

<style scoped>
.dashboard-content { display:flex;flex-direction:column;gap:16px; }
</style>
