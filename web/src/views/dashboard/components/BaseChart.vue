<template>
  <div ref="chartRef" class="business-chart" :class="`chart-${size}`" />
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import * as echarts from "echarts";

const props = withDefaults(
  defineProps<{ option: Record<string, unknown>; size?: string }>(),
  { size: "md" }
);
const chartRef = ref<HTMLDivElement>();
let chart: echarts.ECharts | undefined;
let resizeObserver: ResizeObserver | undefined;
let themeObserver: MutationObserver | undefined;

function render() {
  if (!chartRef.value) return;
  const styles = getComputedStyle(document.documentElement);
  const text = styles.getPropertyValue("--text-secondary").trim() || "#86909c";
  const border = styles.getPropertyValue("--border").trim() || "#e5e6eb";
  if (!chart) chart = echarts.init(chartRef.value);
  const axisStyle = {
    axisLabel: { color: text },
    axisLine: { lineStyle: { color: border } },
    splitLine: { lineStyle: { color: border, type: "dashed" as const } }
  };
  chart.setOption(
    {
      ...props.option,
      backgroundColor: "transparent",
      textStyle: { color: text },
      animation: !window.matchMedia("(prefers-reduced-motion: reduce)").matches,
      legend: {
        textStyle: { color: text },
        ...((props.option.legend as object) ?? {})
      },
      ...(props.option.xAxis
        ? {
            xAxis: echarts.util.merge(
              echarts.util.clone(axisStyle),
              props.option.xAxis,
              true
            )
          }
        : {}),
      ...(props.option.yAxis
        ? {
            yAxis: echarts.util.merge(
              echarts.util.clone(axisStyle),
              props.option.yAxis,
              true
            )
          }
        : {})
    },
    { notMerge: true }
  );
}
watch(() => props.option, render, { deep: true });
onMounted(() => {
  render();
  resizeObserver = new ResizeObserver(() => chart?.resize());
  if (chartRef.value) resizeObserver.observe(chartRef.value);
  themeObserver = new MutationObserver(render);
  themeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ["data-theme", "class"]
  });
});
onBeforeUnmount(() => {
  resizeObserver?.disconnect();
  themeObserver?.disconnect();
  chart?.dispose();
});
</script>

<style scoped>
.business-chart {
  width: 100%;
  height: 280px;
  min-width: 0;
}
.chart-sm {
  height: 220px;
}
.chart-lg {
  height: 350px;
}
</style>
