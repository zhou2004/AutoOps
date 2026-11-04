<template>
  <div class="backup-launcher">
    <el-tooltip
      :content="$tr(reason) || $tr('为选定集群创建一次备份')"
      placement="top"
    >
      <span
        ><el-button
          type="primary"
          :loading="loading"
          :disabled="!canBackup"
          @click="open('backup')"
          >{{ $tr("立即备份") }}</el-button
        ></span
      >
    </el-tooltip>
    <el-button v-if="canArchive" @click="open('configure-archive')">{{
      $tr("配置 WAL 归档")
    }}</el-button>
    <span v-if="reason && clusterId" class="hint">{{ $tr(reason) }}</span>
    <OperationDialog
      v-if="cluster"
      v-model="operationOpen"
      :cluster="cluster"
      :action="action"
      :label="action === 'backup' ? $tr('立即备份') : $tr('配置 WAL 归档')"
      @submitted="submitted"
    />
  </div>
</template>
<script setup lang="ts">
import { tr } from "@/locales";

import { computed, ref, watch } from "vue";
import {
  dbPlatform,
  type Cluster,
  type Capability
} from "@/api/databasePlatform";
import OperationDialog from "./OperationDialog.vue";
const props = defineProps<{ clusterId?: number }>();
const emit = defineEmits(["task"]);
const cluster = ref<Cluster>(),
  capabilities = ref<Capability[]>([]),
  loading = ref(false),
  error = ref(""),
  action = ref("backup"),
  operationOpen = ref(false);
const canBackup = computed(() =>
  capabilities.value.some(c => c.action === "backup" && c.enabled)
);
const canArchive = computed(() =>
  capabilities.value.some(c => c.action === "configure-archive" && c.enabled)
);
const reason = computed(
  () =>
    error.value ||
    (!props.clusterId
      ? tr("请先选择操作集群")
      : capabilities.value.find(c => c.action === "backup" && !c.enabled)
          ?.reason || "")
);
function open(value: string) {
  action.value = value;
  operationOpen.value = true;
}
function submitted(id: number) {
  operationOpen.value = false;
  emit("task", id);
}
let generation = 0;
watch(
  () => props.clusterId,
  async id => {
    const current = ++generation;
    cluster.value = undefined;
    capabilities.value = [];
    error.value = "";
    operationOpen.value = false;
    loading.value = !!id;
    if (!id) return;
    try {
      const data = await dbPlatform.cluster(id);
      if (current !== generation) return;
      cluster.value = data.cluster;
      capabilities.value = data.capabilities;
    } catch (e) {
      if (current === generation) error.value = (e as Error).message;
    } finally {
      if (current === generation) loading.value = false;
    }
  },
  { immediate: true }
);
</script>
<style scoped>
.backup-launcher {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}
.hint {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
</style>
