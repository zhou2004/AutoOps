<template>
  <el-dialog
    :model-value="modelValue"
    title="业务分组容量阈值"
    width="540px"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <el-form label-width="110px">
      <el-form-item label="业务分组"
        ><el-tree-select
          v-model="form.groupId"
          :data="groups"
          :props="{ label: 'name', value: 'id', children: 'children' }"
          check-strictly
          filterable
          @change="read"
      /></el-form-item>
      <el-form-item label="提醒使用率"
        ><el-input-number v-model="form.warningPercent" :min="1" :max="99" />
        %</el-form-item
      >
      <el-form-item label="严重使用率"
        ><el-input-number v-model="form.criticalPercent" :min="2" :max="100" />
        %</el-form-item
      >
    </el-form>
    <p>
      默认提醒 80%、严重 90%。保存后应用到该分组及子分组；子分组的独立设置优先。
    </p>
    <p>
      告警中心使用数据库指标 disk_warning /
      disk_critical；仅已启用的告警规则会触发通知。
    </p>
    <template #footer
      ><el-button @click="$emit('update:modelValue', false)">取消</el-button
      ><el-button
        type="primary"
        :loading="busy"
        :disabled="!form.groupId"
        @click="save"
        >保存</el-button
      ></template
    >
  </el-dialog>
</template>
<script setup lang="ts">
import { reactive, ref } from "vue";
import { dbRequest } from "@/api/databasePlatform";
defineProps<{ modelValue: boolean; groups: any[] }>();
const emit = defineEmits(["update:modelValue", "saved"]);
const form = reactive({
  groupId: undefined as number | undefined,
  warningPercent: 80,
  criticalPercent: 90
});
const busy = ref(false),
  error = ref("");
async function read() {
  error.value = "";
  try {
    const row = await dbRequest("storage-thresholds", "get", undefined, {
      groupId: form.groupId
    });
    form.warningPercent = row.warningPercent;
    form.criticalPercent = row.criticalPercent;
  } catch (e) {
    error.value = (e as Error).message;
  }
}
async function save() {
  busy.value = true;
  error.value = "";
  try {
    await dbRequest("storage-thresholds", "post", form);
    emit("saved");
    emit("update:modelValue", false);
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    busy.value = false;
  }
}
</script>
