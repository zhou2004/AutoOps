<template>
  <div class="glass-card module-section animate-fade-in-up" :style="{ animationDelay: '0.4s' }">
    <div class="section-title">
      <span class="title-bar"></span>
      <span>🔔 告警中心</span>
    </div>

    <div class="dashboard-grid dashboard-grid-4" style="margin-bottom:16px">
      <StatCard icon="🔴" iconBg="linear-gradient(135deg,#fff2f0,#ffd8d2)" :value="alertStats.critical" label="严重告警" tag="立即处理" tagType="danger" />
      <StatCard icon="🟡" iconBg="linear-gradient(135deg,#fff7e6,#ffe7ba)" :value="alertStats.warning" label="警告" tag="关注" tagType="warning" />
      <StatCard icon="🔵" iconBg="linear-gradient(135deg,#e6f4ff,#bae0ff)" :value="alertStats.info" label="提示" />
      <StatCard icon="✅" iconBg="linear-gradient(135deg,#f6ffed,#d9f7be)" :value="alertStats.resolved" label="已处理" tag="今日" />
    </div>

    <div class="dashboard-grid dashboard-grid-2">
      <div class="stat-card">
        <div style="font-size:14px;font-weight:600;color:var(--text-primary);margin-bottom:8px">告警趋势（近7天）</div>
        <BaseChart :option="trendOption" size="sm" />
      </div>
      <div class="stat-card">
        <div style="font-size:14px;font-weight:600;color:var(--text-primary);margin-bottom:8px">告警对比分析</div>
        <BaseChart :option="compareOption" size="sm" />
      </div>
    </div>
  </div>
</template>

<script>
import StatCard from './StatCard.vue'
import BaseChart from './BaseChart.vue'

export default {
  name: 'AlertModule',
  components: { StatCard, BaseChart },
  data() {
    return {
      alertStats: { critical: 3, warning: 12, info: 28, resolved: 156 }
    }
  },
  computed: {
    trendOption() {
      return {
        tooltip: { trigger: 'axis' },
        legend: { data: ['严重', '警告', '提示'], bottom: 0, textStyle: { color: '#86909c', fontSize: 11 } },
        grid: { left: '3%', right: '4%', bottom: '18%', top: '3%', containLabel: true },
        xAxis: { type: 'category', data: ['5/25','5/26','5/27','5/28','5/29','5/30','今天'], axisLabel: { color: '#86909c' } },
        yAxis: { type: 'value', splitLine: { lineStyle: { color: 'var(--border)', type: 'dashed' } } },
        series: [
          { name: '严重', type: 'bar', stack: 'total', barWidth: '50%', data: [2,0,1,3,1,0,2], itemStyle: { color: '#f5222d' } },
          { name: '警告', type: 'bar', stack: 'total', barWidth: '50%', data: [5,8,6,10,7,9,12], itemStyle: { color: '#fa8c16' } },
          { name: '提示', type: 'bar', stack: 'total', barWidth: '50%', data: [15,20,18,22,25,19,28], itemStyle: { color: '#1677ff' } }
        ]
      }
    },
    compareOption() {
      return {
        tooltip: { trigger: 'axis' },
        radar: {
          indicator: [
            { name: '同比昨日', max: 100 },
            { name: '同比上周', max: 100 },
            { name: '同比上月', max: 100 },
            { name: '同比去年', max: 100 }
          ],
          shape: 'circle',
          splitArea: { areaStyle: { color: ['rgba(22,119,255,0.02)','rgba(22,119,255,0.05)','rgba(22,119,255,0.08)','rgba(22,119,255,0.12)'] } }
        },
        series: [{
          type: 'radar',
          data: [{ value: [65, 82, 45, 128], name: '告警对比', areaStyle: { color: 'rgba(22,119,255,0.2)' }, lineStyle: { color: '#1677ff', width: 2 }, itemStyle: { color: '#1677ff' } }]
        }]
      }
    }
  }
}
</script>
