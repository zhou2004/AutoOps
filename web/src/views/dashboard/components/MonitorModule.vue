<template>
  <div class="glass-card module-section animate-fade-in-up" :style="{ animationDelay: '0.3s' }">
    <div class="section-title">
      <span class="title-bar"></span>
      <span>📊 监控中心</span>
    </div>

    <!-- Top 5 排行 -->
    <div class="dashboard-grid dashboard-grid-3" style="margin-bottom:16px">
      <div class="stat-card">
        <div style="font-size:14px;font-weight:600;color:var(--text-primary);margin-bottom:8px;display:flex;align-items:center;gap:6px">
          <span class="status-dot danger"></span> CPU 使用率 TOP5
        </div>
        <BaseChart :option="cpuOption" size="sm" />
      </div>
      <div class="stat-card">
        <div style="font-size:14px;font-weight:600;color:var(--text-primary);margin-bottom:8px;display:flex;align-items:center;gap:6px">
          <span class="status-dot warning"></span> 内存使用率 TOP5
        </div>
        <BaseChart :option="memOption" size="sm" />
      </div>
      <div class="stat-card">
        <div style="font-size:14px;font-weight:600;color:var(--text-primary);margin-bottom:8px;display:flex;align-items:center;gap:6px">
          <span class="status-dot info"></span> 磁盘使用率 TOP5
        </div>
        <BaseChart :option="diskOption" size="sm" />
      </div>
    </div>

    <!-- 网络 & 服务 -->
    <div class="dashboard-grid dashboard-grid-2">
      <div class="stat-card">
        <div style="font-size:14px;font-weight:600;color:var(--text-primary);margin-bottom:8px">🌐 网络延迟与吞吐</div>
        <BaseChart :option="networkOption" size="sm" />
      </div>
      <div class="stat-card">
        <div style="font-size:14px;font-weight:600;color:var(--text-primary);margin-bottom:8px">🔄 服务响应与错误率</div>
        <BaseChart :option="serviceOption" size="sm" />
      </div>
    </div>
  </div>
</template>

<script>
import BaseChart from './BaseChart.vue'

export default {
  name: 'MonitorModule',
  components: { BaseChart },
  computed: {
    cpuOption() {
      return {
        tooltip: { trigger: 'axis' },
        grid: { left: '3%', right: '3%', bottom: '5%', top: '3%', containLabel: true },
        xAxis: { type: 'value', max: 100, splitLine: { show: false }, axisLabel: { formatter: '{value}%', color: '#86909c', fontSize: 10 } },
        yAxis: { type: 'category', data: ['web-01','db-03','k8s-m01','redis-02','api-05'], axisLabel: { color: '#86909c', fontSize: 10 } },
        series: [{
          type: 'bar', barWidth: '50%', data: [
            { value: 92, itemStyle: { color: '#f5222d' } },
            { value: 87, itemStyle: { color: '#fa8c16' } },
            { value: 78, itemStyle: { color: '#fa8c16' } },
            { value: 72, itemStyle: { color: '#1677ff' } },
            { value: 65, itemStyle: { color: '#1677ff' } }
          ]
        }]
      }
    },
    memOption() {
      return {
        tooltip: { trigger: 'axis' },
        grid: { left: '3%', right: '3%', bottom: '5%', top: '3%', containLabel: true },
        xAxis: { type: 'value', max: 100, splitLine: { show: false }, axisLabel: { formatter: '{value}%', color: '#86909c', fontSize: 10 } },
        yAxis: { type: 'category', data: ['db-03','redis-02','web-01','k8s-m01','api-05'], axisLabel: { color: '#86909c', fontSize: 10 } },
        series: [{
          type: 'bar', barWidth: '50%', data: [
            { value: 95, itemStyle: { color: '#f5222d' } },
            { value: 82, itemStyle: { color: '#fa8c16' } },
            { value: 76, itemStyle: { color: '#fa8c16' } },
            { value: 68, itemStyle: { color: '#1677ff' } },
            { value: 55, itemStyle: { color: '#52c41a' } }
          ]
        }]
      }
    },
    diskOption() {
      return {
        tooltip: { trigger: 'axis' },
        grid: { left: '3%', right: '3%', bottom: '5%', top: '3%', containLabel: true },
        xAxis: { type: 'value', max: 100, splitLine: { show: false }, axisLabel: { formatter: '{value}%', color: '#86909c', fontSize: 10 } },
        yAxis: { type: 'category', data: ['minio-01','monitor-02','k8s-w01','gitlab-01','jenkins'], axisLabel: { color: '#86909c', fontSize: 10 } },
        series: [{
          type: 'bar', barWidth: '50%', data: [
            { value: 96, itemStyle: { color: '#f5222d' } },
            { value: 88, itemStyle: { color: '#fa8c16' } },
            { value: 74, itemStyle: { color: '#fa8c16' } },
            { value: 62, itemStyle: { color: '#1677ff' } },
            { value: 45, itemStyle: { color: '#52c41a' } }
          ]
        }]
      }
    },
    networkOption() {
      return {
        tooltip: { trigger: 'axis' },
        legend: { data: ['延迟(ms)', '带宽使用率'], bottom: 0, textStyle: { color: '#86909c', fontSize: 11 } },
        grid: { left: '3%', right: '4%', bottom: '18%', top: '3%', containLabel: true },
        xAxis: { type: 'category', data: ['00:00','04:00','08:00','12:00','16:00','20:00','23:00'], axisLabel: { color: '#86909c', fontSize: 10 } },
        yAxis: [
          { type: 'value', name: '延迟/ms', splitLine: { lineStyle: { color: 'var(--border)', type: 'dashed' } } },
          { type: 'value', name: '%', max: 100, splitLine: { show: false } }
        ],
        series: [
          { name: '延迟(ms)', type: 'line', smooth: true, data: [12, 8, 15, 22, 18, 25, 10],
            lineStyle: { color: '#1677ff', width: 2 }, symbol: 'circle', symbolSize: 6 },
          { name: '带宽使用率', type: 'line', smooth: true, yAxisIndex: 1, data: [45, 32, 68, 85, 72, 55, 38],
            lineStyle: { color: '#52c41a', width: 2 }, symbol: 'diamond', symbolSize: 6 }
        ]
      }
    },
    serviceOption() {
      return {
        tooltip: { trigger: 'axis' },
        legend: { data: ['响应时间(ms)', '错误率(5xx)'], bottom: 0, textStyle: { color: '#86909c', fontSize: 11 } },
        grid: { left: '3%', right: '4%', bottom: '18%', top: '3%', containLabel: true },
        xAxis: { type: 'category', data: ['S3终端','S4门禁','S5摄像头','S6租住','API网关','认证中心'], axisLabel: { color: '#86909c', fontSize: 10 } },
        yAxis: [
          { type: 'value', name: '响应/ms', splitLine: { lineStyle: { color: 'var(--border)', type: 'dashed' } } },
          { type: 'value', name: '%', max: 5, splitLine: { show: false } }
        ],
        series: [
          { name: '响应时间(ms)', type: 'bar', barWidth: '30%', data: [120, 85, 200, 95, 65, 150],
            itemStyle: { color: '#1677ff', borderRadius: [3,3,0,0] } },
          { name: '错误率(5xx)', type: 'line', yAxisIndex: 1, smooth: true, data: [0.2, 0.5, 1.2, 0.3, 0.1, 0.8],
            lineStyle: { color: '#f5222d', width: 2 }, symbol: 'circle', symbolSize: 6 }
        ]
      }
    }
  }
}
</script>
