<template>
  <div class="process-node" :class="data?.status">
    <div class="node-label">{{ data?.label }}</div>
  </div>
</template>
<script setup lang="ts">
defineProps({ data: Object, sourcePosition: String, targetPosition: String });
</script>
<style scoped>
.process-node { padding: 10px; border: 1px solid #ccc; border-radius: 8px; background: var(--bg-card); }
.process-node.running { border-color: #2563eb; }
.process-node.success { border-color: #16a34a; }
.process-node.error { border-color: #dc2626; }
</style>
