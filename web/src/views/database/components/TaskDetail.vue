<template>
  <el-drawer
    :model-value="modelValue"
    :title="`运维任务 #${operationId}`"
    size="70%"
    @update:model-value="$emit('update:modelValue', $event)"
    ><el-alert v-if="error" :title="error" type="error" /><template v-if="task"
      ><div class="task-summary">
        <el-tag :type="tagType(task.status)">{{
          operationStatusLabel(task.status)
        }}</el-tag
        ><span>{{ task.action }}</span
        ><span>{{ task.message }}</span
        ><el-button @click="load">刷新</el-button>
      </div>
      <el-alert
        v-if="task.status === 'unknown'"
        title="结果待核验：互斥锁已保留。请核对远端状态，勿重复提交切换、恢复或迁槽。"
        type="warning"
        :closable="false"
      /><el-empty
        v-if="!task.steps?.length"
        :description="
          task.status === 'queued'
            ? '任务已持久化，等待数据库 Worker 领取'
            : '尚无执行步骤'
        "
      /><el-timeline
        ><el-timeline-item
          v-for="step in task.steps || []"
          :key="step.id"
          :timestamp="new Date(step.startedAt).toLocaleString()"
          :type="
            step.status === 'succeeded'
              ? 'success'
              : step.status === 'failed'
                ? 'danger'
                : 'primary'
          "
          ><strong>{{ step.name }}</strong
          ><el-tag size="small" :type="tagType(step.status)">{{
            label(step.status)
          }}</el-tag>
          <pre v-if="step.log">{{ step.log }}</pre>
        </el-timeline-item></el-timeline
      ></template
    ></el-drawer
  >
</template>
<script setup lang="ts">
import { ref, watch, onUnmounted } from "vue";
import {
  dbPlatform,
  label,
  operationStatusLabel,
  tagType,
  type Operation
} from "@/api/databasePlatform";
const props = defineProps<{ modelValue: boolean; operationId: number }>();
const emit = defineEmits(["update:modelValue", "finished"]);
const task = ref<Operation>(),
  error = ref("");
let timer: ReturnType<typeof setTimeout> | undefined;
let generation = 0;
async function load() {
  const current = ++generation;
  clearTimeout(timer);
  try {
    const value = await dbPlatform.operation(props.operationId);
    if (current !== generation) return;
    const previous = task.value?.status;
    task.value = value;
    error.value = "";
    if (
      ["queued", "preflight", "running", "verifying"].includes(value.status) &&
      props.modelValue
    )
      timer = setTimeout(load, 2000);
    else if (previous !== value.status) emit("finished");
  } catch (e) {
    error.value = (e as Error).message;
  }
}
watch(
  () => [props.modelValue, props.operationId],
  () => {
    clearTimeout(timer);
    generation++;
    if (props.modelValue) {
      task.value = undefined;
      load();
    }
  },
  { immediate: true }
);
onUnmounted(() => {
  generation++;
  clearTimeout(timer);
});
</script>
<style scoped>
.task-summary {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}
.el-timeline {
  margin-top: 24px;
}
.el-tag {
  margin-left: 10px;
}
pre {
  padding: 18px;
  background: var(--el-fill-color-darker);
  color: var(--el-text-color-primary);
  border-radius: 8px;
  font: 12px/1.7 monospace;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  max-height: 440px;
  overflow: auto;
}
</style>
