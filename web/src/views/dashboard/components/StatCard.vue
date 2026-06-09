<template>
  <div class="stat-card" :style="{ animationDelay: delay + 's' }" @click="$emit('click')">
    <div class="card-header">
      <div class="card-icon" :style="{ background: iconBg }">
        <span v-html="icon"></span>
      </div>
      <el-tag v-if="tag" :type="tagType" size="small" effect="plain">{{ tag }}</el-tag>
    </div>
    <div class="card-value">
      <CountUp :value="value" :duration="1.5" />
      <span v-if="unit" class="card-unit">{{ unit }}</span>
    </div>
    <div class="card-label">{{ label }}</div>
    <div class="card-trend" v-if="trend !== undefined">
      <span :style="{ color: trend >= 0 ? 'var(--success)' : 'var(--danger)' }">
        {{ trend >= 0 ? '↑' : '↓' }} {{ Math.abs(trend) }}%
      </span>
      <span class="text-secondary">同比昨日</span>
    </div>
  </div>
</template>

<script>
import CountUp from './CountUp.vue'

export default {
  name: 'StatCard',
  components: { CountUp },
  props: {
    icon: { type: String, default: '' },
    iconBg: { type: String, default: 'var(--primary-light)' },
    value: { type: [Number, String], default: 0 },
    label: { type: String, default: '' },
    unit: { type: String, default: '' },
    trend: { type: Number, default: undefined },
    tag: { type: String, default: '' },
    tagType: { type: String, default: 'info' },
    delay: { type: Number, default: 0 }
  },
  emits: ['click']
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}
.card-icon :deep(svg) {
  width: 24px; height: 24px;
}
.card-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.2;
}
.card-unit {
  font-size: 14px;
  font-weight: 400;
  color: var(--text-secondary);
  margin-left: 2px;
}
.card-label {
  font-size: 13px;
  color: var(--text-secondary);
  margin-top: 4px;
}
.card-trend span:last-child {
  margin-left: 4px;
}
.text-secondary { color: var(--text-secondary); font-size: 12px; }
</style>
