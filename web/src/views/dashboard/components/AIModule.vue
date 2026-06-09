<template>
  <div class="glass-card module-section animate-fade-in-up" :style="{ animationDelay: '0.7s' }">
    <div class="section-title">
      <span class="title-bar"></span>
      <span>🤖 AI 智能分析</span>
      <el-tag type="danger" size="small" effect="dark" style="margin-left:8px">实时</el-tag>
    </div>

    <div class="dashboard-grid dashboard-grid-3" style="margin-bottom:16px">
      <div class="stat-card" v-for="(item, i) in aiCards" :key="i" :style="{ borderLeft: '3px solid ' + item.color }">
        <div style="font-size:14px;font-weight:600;color:var(--text-primary);display:flex;align-items:center;gap:6px">
          <span v-html="item.icon"></span> {{ item.title }}
        </div>
        <div style="margin-top:8px;font-size:13px;color:var(--text-regular);line-height:1.6">
          {{ item.desc }}
        </div>
        <div style="margin-top:8px;display:flex;gap:8px;flex-wrap:wrap">
          <el-tag v-for="(tag, ti) in item.tags" :key="ti" :type="tag.type" size="small" effect="plain">{{ tag.text }}</el-tag>
        </div>
      </div>
    </div>

    <!-- 预测图表 -->
    <div class="dashboard-grid dashboard-grid-2">
      <div class="stat-card">
        <div style="font-size:14px;font-weight:600;color:var(--text-primary);margin-bottom:8px">📈 资源使用率预测（未来7天）</div>
        <BaseChart :option="predictOption" size="sm" />
      </div>
      <div class="stat-card">
        <div style="font-size:14px;font-weight:600;color:var(--text-primary);margin-bottom:8px">💰 成本优化建议</div>
        <div style="display:flex;flex-direction:column;gap:8px">
          <div class="cost-item" v-for="(item, i) in costItems" :key="i">
            <div style="display:flex;justify-content:space-between;align-items:center">
              <span style="font-size:13px;color:var(--text-regular)">{{ item.name }}</span>
              <span style="font-size:13px;font-weight:600;color:var(--success)">-¥{{ item.saving }}/月</span>
            </div>
            <el-progress :percentage="item.percent" :stroke-width="6" :color="item.color" style="margin-top:4px" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import BaseChart from './BaseChart.vue'

export default {
  name: 'AIModule',
  components: { BaseChart },
  data() {
    return {
      aiCards: [
        { icon: '🛡️', title: '自动化故障自愈', color: '#f5222d',
          desc: '检测到 k8s-m01 节点 CPU 持续 5 分钟超过 90%，建议自动触发Pod驱逐和节点扩容流程。',
          tags: [{ text: '自动处理', type: 'success' }, { text: '高优先级', type: 'danger' }] },
        { icon: '📋', title: '智能巡检报告', color: '#1677ff',
          desc: '昨夜巡检完成：检查 1286 台服务器、12 个 K8s 集群、342 个数据库实例，发现 3 个潜在风险。',
          tags: [{ text: '已生成报告', type: 'primary' }, { text: '3个风险项', type: 'warning' }] },
        { icon: '💡', title: '系统预测建议', color: '#722ed1',
          desc: '根据近30天趋势预测：6月中旬将迎来资源使用高峰，建议提前扩容 web 集群和数据库连接池。',
          tags: [{ text: '提前规划', type: 'primary' }, { text: '6月15日', type: 'info' }] }
      ],
      costItems: [
        { name: '闲置ECS实例（3台）', saving: 2850, percent: 72, color: '#1677ff' },
        { name: '预留实例建议', saving: 1800, percent: 55, color: '#52c41a' },
        { name: '存储降配（冷热分离）', saving: 960, percent: 38, color: '#fa8c16' }
      ]
    }
  },
  computed: {
    predictOption() {
      return {
        tooltip: { trigger: 'axis' },
        legend: { data: ['CPU预测', '内存预测', '磁盘预测'], bottom: 0, textStyle: { color: '#86909c', fontSize: 11 } },
        grid: { left: '3%', right: '4%', bottom: '18%', top: '3%', containLabel: true },
        xAxis: { type: 'category', data: ['6/1','6/2','6/3','6/4','6/5','6/6','6/7'], axisLabel: { color: '#86909c' } },
        yAxis: { type: 'value', max: 100, splitLine: { lineStyle: { color: 'var(--border)', type: 'dashed' } } },
        series: [
          { name: 'CPU预测', type: 'line', smooth: true, data: [65, 68, 72, 78, 82, 85, 88],
            lineStyle: { color: '#1677ff', width: 2 }, symbol: 'emptyCircle' },
          { name: '内存预测', type: 'line', smooth: true, data: [70, 72, 75, 76, 78, 80, 82],
            lineStyle: { color: '#52c41a', width: 2 }, symbol: 'emptyCircle' },
          { name: '磁盘预测', type: 'line', smooth: true, data: [55, 57, 60, 62, 65, 68, 72],
            lineStyle: { color: '#fa8c16', width: 2, type: 'dashed' }, symbol: 'emptyCircle' }
        ]
      }
    }
  }
}
</script>

<style scoped>
.cost-item {
  padding: 8px 0;
  border-bottom: 1px solid var(--border-light);
}
.cost-item:last-child { border-bottom: none; }
</style>
