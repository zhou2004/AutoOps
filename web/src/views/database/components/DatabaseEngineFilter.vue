<template>
  <el-select
    :model-value="value"
    class="database-engine-filter"
    aria-label="数据库类型"
    placeholder="全部类型"
    @change="change"
  >
    <el-option label="全部类型" value="" />
    <el-option
      v-for="item in engines"
      :key="item.value"
      :value="item.value"
      :label="item.label"
    >
      <span class="engine-option"
        ><DatabaseEngineIcon :engine="item.value" aria-hidden="true" />{{
          item.label
        }}</span
      >
    </el-option>
  </el-select>
</template>
<script setup lang="ts">
import { computed } from "vue";
import DatabaseEngineIcon from "./DatabaseEngineIcon.vue";
const props = defineProps<{
  modelValue?: string | number;
  numeric?: boolean;
}>();
const emit = defineEmits(["update:modelValue", "change"]);
const engines = [
  { value: "mysql", label: "MySQL", id: 1 },
  { value: "postgresql", label: "PostgreSQL", id: 2 },
  { value: "redis", label: "Redis", id: 3 }
];
const value = computed(() =>
  props.numeric
    ? engines.find(e => e.id === props.modelValue)?.value || ""
    : props.modelValue || ""
);
function change(engine: string) {
  const next = props.numeric
    ? engines.find(e => e.value === engine)?.id
    : engine;
  emit("update:modelValue", next);
  emit("change", next);
}
</script>
<style scoped>
.database-engine-filter {
  width: 166px;
  flex: 0 0 166px;
}
.engine-option {
  display: flex;
  align-items: center;
  gap: 10px;
}
.engine-option :deep(img) {
  width: 20px;
  height: 20px;
  flex-basis: 20px;
}
</style>
