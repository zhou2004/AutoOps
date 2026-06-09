<template>
  <div class="dashboard-grid dashboard-grid-2" style="margin-bottom:16px">
    <!-- 用户模块 -->
    <div class="glass-card module-section animate-fade-in-up" :style="{ animationDelay: '0.5s' }">
      <div class="section-title">
        <span class="title-bar"></span>
        <span>👥 用户分析</span>
      </div>
      <div class="dashboard-grid dashboard-grid-4" style="margin-bottom:12px">
        <StatCard icon="👤" iconBg="linear-gradient(135deg,#e6f4ff,#bae0ff)" :value="userStats.total" label="用户总数" />
        <StatCard icon="📅" iconBg="linear-gradient(135deg,#f6ffed,#d9f7be)" :value="userStats.weekly" label="本周登录" />
        <StatCard icon="📆" iconBg="linear-gradient(135deg,#fff7e6,#ffe7ba)" :value="userStats.monthly" label="本月登录" />
        <StatCard icon="⭐" iconBg="linear-gradient(135deg,#f9f0ff,#efdbff)" :value="userStats.active" label="活跃用户" />
      </div>
    </div>

    <!-- 工单模块 -->
    <div class="glass-card module-section animate-fade-in-up" :style="{ animationDelay: '0.6s' }">
      <div class="section-title">
        <span class="title-bar"></span>
        <span>📋 工单统计</span>
      </div>
      <div class="dashboard-grid dashboard-grid-2" style="margin-bottom:12px">
        <div class="stat-card">
          <div style="font-size:14px;font-weight:600;color:var(--text-primary);margin-bottom:8px">工单分类统计</div>
          <BaseChart :option="ticketOption" size="sm" />
        </div>
        <div class="stat-card">
          <div style="font-size:14px;font-weight:600;color:var(--text-primary);margin-bottom:8px">审批类型分布</div>
          <BaseChart :option="approveOption" size="sm" />
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import StatCard from './StatCard.vue'
import BaseChart from './BaseChart.vue'

export default {
  name: 'UserTicketModule',
  components: { StatCard, BaseChart },
  data() {
    return {
      userStats: { total: 286, weekly: 168, monthly: 1258, active: 42 }
    }
  },
  computed: {
    ticketOption() {
      return {
        tooltip: { trigger: 'item' },
        legend: { bottom: 0, textStyle: { color: '#86909c', fontSize: 11 } },
        series: [{
          type: 'pie', radius: ['35%', '60%'], center: ['50%', '45%'],
          itemStyle: { borderRadius: 4, borderColor: '#fff', borderWidth: 2 },
          label: { show: false },
          data: [
            { value: 45, name: '服务器权限', itemStyle: { color: '#1677ff' } },
            { value: 32, name: '数据库权限', itemStyle: { color: '#52c41a' } },
            { value: 28, name: '应用权限', itemStyle: { color: '#fa8c16' } },
            { value: 18, name: '云平台权限', itemStyle: { color: '#722ed1' } },
            { value: 12, name: '日志权限', itemStyle: { color: '#13c2c2' } }
          ]
        }]
      }
    },
    approveOption() {
      return {
        tooltip: { trigger: 'axis' },
        grid: { left: '3%', right: '3%', bottom: '5%', top: '3%', containLabel: true },
        xAxis: { type: 'category', data: ['服务上线', '服务迭代', '服务下线', '脚本执行', 'SQL审批'], axisLabel: { color: '#86909c', fontSize: 10 } },
        yAxis: { type: 'value', splitLine: { lineStyle: { color: 'var(--border)', type: 'dashed' } } },
        series: [{
          type: 'bar', barWidth: '45%',
          itemStyle: { borderRadius: [4,4,0,0], color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: '#1677ff' }, { offset: 1, color: '#36a3ff' }] } },
          data: [38, 25, 12, 42, 30]
        }]
      }
    }
  }
}
</script>
