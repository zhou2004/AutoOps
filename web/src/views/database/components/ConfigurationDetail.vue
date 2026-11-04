<template>
  <section class="configuration-detail">
    <div class="configuration-toolbar">
      <p>{{ $tr("比较期望配置与各实例实际值，查看变更版本及待重启项。") }}</p>
      <el-button @click="load">{{ $tr("刷新参数") }}</el-button>
      <ClusterActions
        :cluster-id="clusterId"
        context="configurations"
        @task="id => $emit('task', id)"
        @changed="load"
      />
    </div>
    <el-alert v-if="error" :title="$tr(error)" type="error" :closable="false" />
    <el-skeleton v-if="loading" :rows="6" animated />
    <template v-else>
      <el-table :data="configurationRows"
        ><el-table-column prop="name" :label="$tr('参数')" /><el-table-column
          :label="$tr('期望值')"
          ><template #default="{ row }">{{
            configuration.desired?.[row.name] ?? $tr("未设置")
          }}</template></el-table-column
        ><el-table-column :label="$tr('说明')" min-width="240"
          ><template #default="{ row }">{{
            $tr(row.description)
          }}</template></el-table-column
        ><el-table-column
          prop="mode"
          :label="$tr('生效方式')"
        /><el-table-column
          v-for="instance in configuration.instances || []"
          :key="instance.memberId"
          :label="instance.name"
          ><template #default="{ row }"
            ><span>{{ instance.values?.[row.name] ?? $tr("未知") }}</span
            ><el-tag
              v-if="instance.differences?.[row.name]"
              type="warning"
              size="small"
              >{{
                instance.pendingRestart?.[row.name]
                  ? $tr("待重启")
                  : $tr("配置差异")
              }}</el-tag
            ></template
          ></el-table-column
        ></el-table
      >
      <h3>{{ $tr("配置版本") }}</h3>
      <el-table :data="configuration.revisions || []"
        ><el-table-column prop="id" :label="$tr('版本')" /><el-table-column
          :label="$tr('范围')"
          ><template #default="{ row }"
            >{{
              row.memberId
                ? $tr("实例 #{value0}", { value0: row.memberId })
                : $tr("集群")
            }}
            ·
            {{ row.runtimeOnly ? $tr("仅运行时") : $tr("持久配置") }}</template
          ></el-table-column
        ><el-table-column :label="$tr('状态')"
          ><template #default="{ row }">{{
            label(row.status)
          }}</template></el-table-column
        ><el-table-column :label="$tr('变更')"
          ><template #default="{ row }"
            ><span v-for="(value, key) in row.desired" :key="key"
              >{{ key }} = {{ value }}　</span
            ></template
          ></el-table-column
        ><el-table-column :label="$tr('时间')"
          ><template #default="{ row }">{{
            date(row.createdAt)
          }}</template></el-table-column
        ></el-table
      >
    </template>
  </section>
</template>
<script setup lang="ts">
import { tr } from "@/locales";

import { computed, ref, watch } from "vue";
import ClusterActions from "./ClusterActions.vue";
import { dbRequest, label } from "@/api/databasePlatform";
const props = defineProps<{ modelValue: boolean; clusterId: number }>();
defineEmits(["task"]);
const configuration = ref<any>({}),
  error = ref(""),
  loading = ref(false);
const configurationRows = computed(() => configuration.value.definitions || []);
const date = (v?: string) =>
  v ? new Date(v).toLocaleString() : tr("尚未采集");
async function load() {
  if (!props.modelValue) return;
  const id = props.clusterId;
  loading.value = true;
  error.value = "";
  try {
    const data = await dbRequest(`clusters/${id}/configurations`);
    if (id === props.clusterId) configuration.value = data;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}
watch(
  () => [props.modelValue, props.clusterId],
  () => {
    configuration.value = {};
    load();
  },
  { immediate: true }
);
</script>
<style scoped>
.configuration-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 16px;
}
.configuration-toolbar p {
  flex: 1;
  color: var(--el-text-color-secondary);
}
h3 {
  margin: 24px 0 16px;
}
</style>
