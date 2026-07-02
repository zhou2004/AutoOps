<template>
  <el-dialog v-model="visible" title="YAML 查看器" width="80%">
    <pre>{{ yamlContent }}</pre>
  </el-dialog>
</template>
<script setup lang="ts">
import { ref } from 'vue';
defineOptions({ name: "YamlViewerDialog" });
const visible = ref(false);
const yamlContent = ref('');
const open = (content: string) => { yamlContent.value = content; visible.value = true; };
defineExpose({ open });
</script>
