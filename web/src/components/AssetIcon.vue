<template>
  <svg
    viewBox="0 0 32 32"
    fill="none"
    stroke="currentColor"
    stroke-width="1.6"
    stroke-linecap="round"
    stroke-linejoin="round"
    :width="size"
    :height="size"
    role="img"
    :aria-label="tr(label)"
  >
    <template v-if="kind === 'physical' || kind === 'host'">
      <rect x="4" y="4" width="24" height="10" rx="2" />
      <rect x="4" y="18" width="24" height="10" rx="2" />
      <path d="M8 9h10M8 23h10" />
      <circle cx="24" cy="9" r="1" fill="currentColor" />
      <circle cx="24" cy="23" r="1" fill="currentColor" />
    </template>
    <template v-else-if="kind === 'idc'">
      <path d="M3 28V8l13-5 13 5v20H3Z" />
      <path d="M8 28V13h6v15m4 0V13h6v15M8 18h6m4 0h6M8 23h6m4 0h6" />
    </template>
    <template v-else-if="kind === 'cabinet'">
      <rect x="7" y="2" width="18" height="28" rx="2" />
      <path d="M11 7h10M11 12h10M11 17h10M11 22h10" />
      <circle cx="21" cy="27" r=".7" fill="currentColor" />
    </template>
    <template v-else-if="deviceType === 3">
      <path d="M16 3 27 7v9c0 7-11 13-11 13S5 23 5 16V7L16 3Z" />
      <path d="M10 13h12m-12 5h12m-8-10v5m4 0v5m-4 0v6" />
    </template>
    <template v-else-if="deviceType === 1">
      <rect x="4" y="17" width="24" height="10" rx="2" />
      <path
        d="M8 17V7m16 10V7M12 6a7 7 0 0 1 8 0M14 10a3 3 0 0 1 4 0M8 22h2m4 0h2m4 0h4"
      />
    </template>
    <template v-else-if="deviceType === 4">
      <rect x="11" y="3" width="10" height="7" rx="1" />
      <path d="M16 10v6M5 21v-5h22v5" />
      <rect x="2" y="21" width="7" height="7" rx="1" />
      <rect x="12.5" y="21" width="7" height="7" rx="1" />
      <rect x="23" y="21" width="7" height="7" rx="1" />
    </template>
    <template v-else>
      <rect x="2" y="10" width="28" height="15" rx="2" />
      <path
        d="M6 15h4v5H6zm8 0h4v5h-4zm8 0h4v5h-4zM8 6h16m-3-3 3 3-3 3M11 3 8 6l3 3"
      />
    </template>
  </svg>
</template>
<script setup>
import { computed } from "vue";
import { tr } from "@/locales";
const props = defineProps({
  kind: { type: String, default: "physical" },
  deviceType: { type: Number, default: 2 },
  size: { type: [Number, String], default: 28 }
});
const label = computed(
  () =>
    ({ physical: "物理机", host: "主机", idc: "机房", cabinet: "机柜" })[
      props.kind
    ] ||
    { 1: "路由器", 2: "交换机", 3: "防火墙", 4: "负载均衡" }[
      props.deviceType
    ] ||
    "网络设备"
);
</script>
