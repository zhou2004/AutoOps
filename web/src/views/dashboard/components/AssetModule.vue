<template>
  <div class="glass-card module-section animate-fade-in-up" :style="{ animationDelay: '0.1s' }">
    <div class="section-title">
      <span class="title-bar"></span>
      <span>📦 CMDB 资产管理</span>
    </div>

    <!-- 概览统计 -->
    <div class="dashboard-grid dashboard-grid-4" style="margin-bottom:16px">
      <StatCard icon="🖥️" iconBg="linear-gradient(135deg, #e6f4ff, #bae0ff)" :value="stats.servers" label="服务器总数" tag="自建+云主机" />
      <StatCard icon="🗄️" iconBg="linear-gradient(135deg, #f6ffed, #d9f7be)" :value="stats.databases" label="数据库总数" tag="MySQL/Redis/Mongo/ES/PG" />
      <StatCard icon="🌐" iconBg="linear-gradient(135deg, #fff7e6, #ffe7ba)" :value="stats.network" label="网络设备" tag="交换机+路由器" />
      <StatCard icon="☸️" iconBg="linear-gradient(135deg, #f9f0ff, #efdbff)" :value="stats.k8s" label="K8s集群" tag="健康: {{ stats.k8sHealthy }}/{{ stats.k8s }}" />
    </div>

    <!-- 硬件资产分布 -->
    <div class="dashboard-grid dashboard-grid-2">
      <div class="stat-card">
        <div style="font-size:14px;font-weight:600;color:var(--text-primary);margin-bottom:8px">硬件资产分布</div>
        <BaseChart :option="hardwareOption" size="sm" />
      </div>
      <div class="stat-card">
        <div style="font-size:14px;font-weight:600;color:var(--text-primary);margin-bottom:8px">软件系统健康状态</div>
        <BaseChart :option="softwareOption" size="sm" />
      </div>
    </div>
  </div>
</template>

<script>
import StatCard from './StatCard.vue'
import BaseChart from './BaseChart.vue'

export default {
  name: 'AssetModule',
  components: { StatCard, BaseChart },
  data() {
    return {
      stats: {
        servers: 1286,
        databases: 342,
        network: 89,
        k8s: 12,
        k8sHealthy: 11
      }
    }
  },
  computed: {
    hardwareOption() {
      return {
        tooltip: { trigger: 'item' },
        legend: { bottom: 0, textStyle: { color: '#86909c', fontSize: 11 } },
        series: [{
          type: 'pie',
          radius: ['40%', '65%'],
          center: ['50%', '45%'],
          avoidLabelOverlap: true,
          itemStyle: { borderRadius: 4, borderColor: '#fff', borderWidth: 2 },
          label: { show: false },
          emphasis: { label: { show: true, fontSize: 13, fontWeight: 'bold' } },
          data: [
            { value: 856, name: '自建服务器', itemStyle: { color: '#1677ff' } },
            { value: 430, name: '云主机', itemStyle: { color: '#36a3ff' } },
            { value: 89, name: '网络设备', itemStyle: { color: '#fa8c16' } }
          ]
        }]
      }
    },
    softwareOption() {
      return {
        tooltip: { trigger: 'axis' },
        grid: { left: '3%', right: '4%', bottom: '8%', top: '3%', containLabel: true },
        xAxis: {
          type: 'category', data: ['S3智能终端', 'S4智能门', 'S5摄像头', 'S6租住'],
          axisLabel: { color: '#86909c', fontSize: 10 }
        },
        yAxis: {
          type: 'value', max: 100,
          splitLine: { lineStyle: { color: 'var(--border)', type: 'dashed' } }
        },
        series: [
          {
            name: '健康度', type: 'bar', barWidth: '40%', barGap: '20%',
            itemStyle: { borderRadius: [4, 4, 0, 0] },
            data: [
              { value: 98, itemStyle: { color: '#52c41a' } },
              { value: 95, itemStyle: { color: '#52c41a' } },
              { value: 87, itemStyle: { color: '#fa8c16' } },
              { value: 92, itemStyle: { color: '#1677ff' } }
            ]
          }
        ]
      }
    }
  }
}
</script>
