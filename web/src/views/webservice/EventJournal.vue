<template>
  <div>
    <div class="toolbar">
      <el-button @click="load()">{{ $tr("刷新") }}</el-button
      ><el-button :disabled="!hasMore || busy" @click="load(rows.at(-1)?.id)">{{
        $tr("更早记录")
      }}</el-button>
    </div>
    <el-alert v-if="error" :title="error" type="error" />
    <el-table v-loading="busy" :data="rows"
      ><el-table-column
        prop="createdAt"
        :label="$tr('时间')"
        width="220" /><el-table-column
        prop="userId"
        :label="$tr('用户')"
        width="90" /><el-table-column
        prop="kind"
        :label="$tr('事件')"
        width="160" /><el-table-column
        prop="sourceId"
        :label="$tr('数据源')"
        width="90" /><el-table-column
        prop="clusterId"
        :label="$tr('集群')"
        width="90" /><el-table-column
        prop="message"
        :label="$tr('说明')"
        min-width="250"
    /></el-table>
  </div>
</template>
<script setup lang="ts">
import { ref, watch, onUnmounted } from "vue";
import { webRequest } from "@/api/webservice";
const props = defineProps<{ clusterId?: number }>();
let generation = 0;
const hasMore = ref(false);
const rows = ref<any[]>([]),
  busy = ref(false),
  error = ref("");
async function load(before?: number) {
  const gen = ++generation;
  busy.value = true;
  error.value = "";
  try {
    const result = await webRequest(
      "/audit/events?clusterId=" +
        (props.clusterId || 0) +
        "&before=" +
        (before || 0)
    );
    if (gen !== generation) return;
    rows.value = result.list;
    hasMore.value = result.total > result.list.length;
  } catch (e) {
    if (gen === generation) {
      rows.value = [];
      hasMore.value = false;
      error.value = (e as Error).message;
    }
  } finally {
    if (gen === generation) busy.value = false;
  }
}
watch(
  () => props.clusterId,
  () => load(),
  { immediate: true }
);
onUnmounted(() => {
  generation++;
});
</script>
<style scoped>
.toolbar {
  display: flex;
  gap: 12px;
  margin: 16px 0;
}
</style>
