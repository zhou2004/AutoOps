<template>
  <el-select
    :model-value="modelValue || undefined"
    clearable
    filterable
    allow-create
    default-first-option
    :placeholder="$tr('选择环境（可留空）')"
    :loading="loading"
    @visible-change="open => open && load()"
    @update:model-value="value => emit('update:modelValue', value || '')"
    @change="value => emit('change', value || '')"
  >
    <el-option
      v-for="value in choices"
      :key="value"
      :value="value"
      :label="labels[value] ? `${$tr(labels[value])} · ${value}` : value"
    />
    <template #footer
      ><span v-if="error" class="environment-error">{{ error }}</span
      ><span v-else>{{
        $tr("可选择已有环境，也可输入自定义值")
      }}</span></template
    >
  </el-select>
</template>
<script setup lang="ts">
import { computed, ref, onUnmounted } from "vue";
import { webRequest } from "@/api/webservice";
const props = defineProps<{ modelValue?: string }>();
const emit = defineEmits<{
  (event: "update:modelValue", value: string): void;
  (event: "change", value: string): void;
}>();
const labels: Record<string, string> = {
  production: "生产",
  staging: "预发布",
  test: "测试",
  development: "开发"
};
const existing = ref<string[]>([]),
  loading = ref(false),
  error = ref("");
const choices = computed(() => [
  ...new Set([
    ...Object.keys(labels),
    ...existing.value,
    ...(props.modelValue ? [props.modelValue] : [])
  ])
]);
let abort: AbortController | undefined;
async function load() {
  if (loading.value) return;
  abort = new AbortController();
  loading.value = true;
  error.value = "";
  try {
    existing.value = await webRequest<string[]>(
      "/environments",
      "GET",
      undefined,
      abort.signal
    );
  } catch (e) {
    if (!abort.signal.aborted) error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}
onUnmounted(() => abort?.abort());
</script>
<style scoped>
.environment-error {
  color: var(--el-color-danger);
}
</style>
