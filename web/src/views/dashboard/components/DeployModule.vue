<template>
  <div class="glass-card module-section animate-fade-in-up" :style="{ animationDelay: '0.2s' }">
    <div class="section-title">
      <span class="title-bar"></span>
      <span>🚀 服务上线</span>
      <div style="margin-left:auto;display:flex;gap:6px">
        <el-tag :type="deployRange==='week'?'primary':''" size="small" style="cursor:pointer" @click="deployRange='week'">本周</el-tag>
        <el-tag :type="deployRange==='month'?'primary':''" size="small" style="cursor:pointer" @click="deployRange='month'">本月</el-tag>
        <el-tag :type="deployRange==='year'?'primary':''" size="small" style="cursor:pointer" @click="deployRange='year'">本年</el-tag>
      </div>
    </div>

    <div class="dashboard-grid dashboard-grid-4" style="margin-bottom:16px">
      <StatCard icon="📦" iconBg="linear-gradient(135deg,#e6f4ff,#bae0ff)" :value="deployStats.total" label="上线总次数" :trend="12" />
      <StatCard icon="✅" iconBg="linear-gradient(135deg,#f6ffed,#d9f7be)" :value="deployStats.success" label="成功次数" />
      <StatCard icon="❌" iconBg="linear-gradient(135deg,#fff2f0,#ffd8d2)" :value="deployStats.fail" label="失败次数" />
      <StatCard icon="📊" iconBg="linear-gradient(135deg,#f9f0ff,#efdbff)" :value="deployStats.successRate" label="成功率" unit="%" />
    </div>

    <div class="dashboard-grid dashboard-grid-2">
      <div class="stat-card">
        <div style="font-size:14px;font-weight:600;color:var(--text-primary);margin-bottom:8px">各业务线上线统计</div>
        <BaseChart :option="businessOption" size="sm" />
      </div>
      <div class="stat-card">
        <div style="font-size:14px;font-weight:600;color:var(--text-primary);margin-bottom:8px">同环比趋势</div>
        <BaseChart :option="trendOption" size="sm" />
      </div>
    </div>
  </div>
</template>

<script>
import StatCard from './StatCard.vue'
import BaseChart from './BaseChart.vue'

export default {
  name: 'DeployModule',
  components: { StatCard, BaseChart },
  data() {
    return {
      deployRange: 'week',
      deployStats: { total: 247, success: 236, fail: 11, successRate: 95.5 }
    }
  },
  computed: {
    businessOption() {
      return {
        tooltip: { trigger: 'axis' },
        legend: { data: ['成功', '失败'], bottom: 0, textStyle: { color: '#86909c', fontSize: 11 } },
        grid: { left: '3%', right: '4%', bottom: '18%', top: '3%', containLabel: true },
        xAxis: { type: 'category', data: ['S3终端', 'S4门禁', 'S5摄像头', 'S6租住'], axisLabel: { color: '#86909c' } },
        yAxis: { type: 'value', splitLine: { lineStyle: { color: 'var(--border)', type: 'dashed' } } },
        series: [
          { name: '成功', type: 'bar', barWidth: '30%', barGap: '10%', itemStyle: { color: '#52c41a', borderRadius: [3,3,0,0] },
            data: [68, 72, 45, 51] },
          { name: '失败', type: 'bar', barWidth: '30%', itemStyle: { color: '#f5222d', borderRadius: [3,3,0,0] },
            data: [3, 4, 2, 2] }
        ]
      }
    },
    trendOption() {
      return {
        tooltip: { trigger: 'axis' },
        legend: { data: ['本周', '上周', '同比去年'], bottom: 0, textStyle: { color: '#86909c', fontSize: 11 } },
        grid: { left: '3%', right: '4%', bottom: '18%', top: '3%', containLabel: true },
        xAxis: { type: 'category', data: ['周一','周二','周三','周四','周五','周六','周日'], axisLabel: { color: '#86909c' } },
        yAxis: { type: 'value', splitLine: { lineStyle: { color: 'var(--border)', type: 'dashed' } } },
        series: [
          { name: '本周', type: 'line', smooth: true, data: [32, 28, 35, 40, 38, 20, 15],
            lineStyle: { color: '#1677ff', width: 2 }, areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(22,119,255,0.25)' }, { offset: 1, color: 'rgba(22,119,255,0.02)' }] } } },
          { name: '上周', type: 'line', smooth: true, data: [28, 30, 32, 35, 42, 25, 18],
            lineStyle: { color: '#86909c', width: 2, type: 'dashed' } },
          { name: '同比去年', type: 'line', smooth: true, data: [22, 25, 28, 30, 35, 18, 12],
            lineStyle: { color: '#52c41a', width: 2, type: 'dotted' } }
        ]
      }
    }
  }
}
</script>
