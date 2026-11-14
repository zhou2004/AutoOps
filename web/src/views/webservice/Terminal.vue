<template>
  <div>
    <div class="toolbar">
      <el-select v-model="chosen" filterable :placeholder="$tr('选择实例')"
        ><el-option
          v-for="i in instances"
          :key="i.id"
          :label="i.name"
          :value="i.id" /></el-select
      ><el-button
        type="primary"
        :disabled="!chosen || !canPlatform('web:execute')"
        @click="open"
        >{{ $tr("连接") }}</el-button
      >
    </div>
    <el-tabs v-model="active" type="card" closable @tab-remove="remove"
      ><el-tab-pane v-for="t in tabs" :key="t.id" :name="t.id" :label="t.name"
        ><TerminalSession
          v-show="active === t.id"
          :instance-id="t.instanceId" /></el-tab-pane></el-tabs
    ><el-empty
      v-if="!tabs.length"
      :description="$tr('选择实例打开主机或容器终端')"
    />
  </div>
</template>
<script setup lang="ts">
import { ref, watch } from "vue";
import { canPlatform } from "@/utils/platformAccess";
import type { WebResource } from "@/api/webservice";
import TerminalSession from "./TerminalSession.vue";
const props = defineProps<{
  instances: WebResource[];
  initialInstanceId?: number;
}>();
const chosen = ref<number>(),
  active = ref(""),
  tabs = ref<{ id: string; name: string; instanceId: number }[]>([]);
let sequence = 0;
watch(
  () => [props.instances, props.initialInstanceId] as const,
  ([instances, initial]) => {
    if (initial && instances.some(i => i.id === initial))
      chosen.value = initial;
    else if (!instances.some(i => i.id === chosen.value))
      chosen.value = instances.length === 1 ? instances[0].id : undefined;
  },
  { immediate: true }
);

function open() {
  const i = props.instances.find(i => i.id === chosen.value);
  if (!i) return;
  const id = String(++sequence);
  tabs.value.push({ id, name: i.name, instanceId: i.id });
  active.value = id;
}
function remove(id: string | number) {
  tabs.value = tabs.value.filter(t => t.id !== id);
  if (active.value === id) active.value = tabs.value.at(-1)?.id || "";
}
</script>
<style scoped>
.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}
</style>
