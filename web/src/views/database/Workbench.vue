<template>
  <div v-loading="loading">
    <el-result v-if="error" icon="error" :title="error"
      ><template #extra
        ><el-button @click="load">{{ $tr("重新加载") }}</el-button></template
      ></el-result
    >
    <MySQLWorkbench
      v-else-if="source?.type === 1"
      :key="source.id"
      :source="source"
    />
    <LegacyWorkbench v-else-if="source" :key="source.id" />
  </div>
</template>
<script setup>
import { ref, watch } from "vue";
import { useRoute } from "vue-router";
import api from "@/api/cmdb";
import { databaseResult } from "@/api/mysqlWorkbench";
import MySQLWorkbench from "./MySQLWorkbench.vue";
import LegacyWorkbench from "@/views/cmdb/DBdetails.vue";
const route = useRoute(),
  source = ref(null),
  loading = ref(false),
  error = ref("");
let revision = 0;
async function load() {
  const current = ++revision;
  loading.value = true;
  error.value = "";
  source.value = null;
  try {
    const value = await databaseResult(api.getDatabase(Number(route.query.id)));
    if (current === revision) source.value = value;
  } catch (e) {
    if (current === revision) error.value = e.message;
  } finally {
    if (current === revision) loading.value = false;
  }
}
watch(() => route.query.id, load, { immediate: true });
</script>
