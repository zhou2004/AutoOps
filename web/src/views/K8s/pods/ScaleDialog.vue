<template>
  <el-dialog v-model="visible" title="扩缩容" width="400px">
    <div>Scale Dialog - Placeholder</div>
  </el-dialog>
</template>
<script setup lang="ts">
import { ref } from 'vue';
defineOptions({ name: "ScaleDialog" });
const visible = ref(false);
const open = () => { visible.value = true; };
defineExpose({ open });
</script>
