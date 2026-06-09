<template>
  <div ref="chartRef" class="chart-container" :class="sizeClass"></div>
</template>

<script>
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import * as echarts from 'echarts'

export default {
  name: 'BaseChart',
  props: {
    option: { type: Object, default: () => ({}) },
    size: { type: String, default: 'md' } // sm, md, lg
  },
  setup(props) {
    const chartRef = ref(null)
    let chart = null

    const sizeClass = {
      sm: 'chart-container-sm',
      md: '',
      lg: 'chart-container-lg'
    }[props.size] || ''

    const getTheme = () => document.documentElement.getAttribute('data-theme') || 'light'

    const initChart = () => {
      if (!chartRef.value) return
      if (chart) chart.dispose()
      chart = echarts.init(chartRef.value, getTheme() === 'dark' ? 'dark' : undefined)
      chart.setOption(props.option)
    }

    const resize = () => { chart?.resize() }

    onMounted(() => {
      nextTick(initChart)
      window.addEventListener('resize', resize)
    })

    onUnmounted(() => {
      window.removeEventListener('resize', resize)
      chart?.dispose()
    })

    watch(() => props.option, () => { nextTick(initChart) }, { deep: true })
    watch(() => props.theme, () => { nextTick(initChart) })

    return { chartRef, sizeClass }
  }
}
</script>
