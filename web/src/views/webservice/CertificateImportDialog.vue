<template>
  <el-dialog
    class="web-certificate-dialog"
    modal-class="web-certificate-overlay"
    append-to-body
    destroy-on-close
    align-center
    :lock-scroll="false"
    :model-value="modelValue"
    :title="$tr('导入证书')"
    width="min(800px, calc(100vw - 32px))"
    :close-on-click-modal="!saving"
    :close-on-press-escape="!saving"
    :show-close="!saving"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <el-form label-position="top" @submit.prevent>
      <div class="import-grid">
        <el-form-item :label="$tr('名称')" required
          ><el-input v-model="name" maxlength="100" :disabled="saving"
        /></el-form-item>
        <el-form-item :label="$tr('业务分组')" required
          ><el-select v-model="groupId" filterable :disabled="saving"
            ><el-option
              v-for="group in groups"
              :key="group.id"
              :value="group.id"
              :label="group.name" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('环境')"
          ><EnvironmentSelect v-model="environment" :disabled="saving"
        /></el-form-item>
      </div>
      <el-alert
        :title="
          $tr('自动识别证书域名、有效期和指纹；证书与私钥不匹配时不会创建资产')
        "
        type="info"
        :closable="false"
      />
      <el-form-item :label="$tr('PEM 证书链')" required>
        <label class="file-picker"
          >{{ $tr("选择证书文件")
          }}<input
            type="file"
            accept=".pem,.crt,.cer"
            :disabled="saving"
            @change="readFile($event, 'certificate')"
        /></label>
        <el-input
          v-model="certificate"
          type="textarea"
          :rows="7"
          spellcheck="false"
          :disabled="saving"
          :placeholder="$tr('粘贴 PEM 证书链或选择文件')"
        />
      </el-form-item>
      <el-form-item :label="$tr('PEM 私钥')" required>
        <label class="file-picker"
          >{{ $tr("选择私钥文件")
          }}<input
            type="file"
            accept=".pem,.key"
            :disabled="saving"
            @change="readFile($event, 'privateKey')"
        /></label>
        <el-input
          v-model="privateKey"
          type="textarea"
          :rows="7"
          spellcheck="false"
          autocomplete="off"
          :disabled="saving"
          :placeholder="$tr('私钥只在服务端加密保存，不会回显')"
        />
      </el-form-item>
    </el-form>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      show-icon
    />
    <template #footer
      ><el-button
        :disabled="saving"
        @click="emit('update:modelValue', false)"
        >{{ $tr("取消") }}</el-button
      ><el-button
        type="primary"
        :loading="saving"
        :disabled="!name || !groupId || !certificate || !privateKey"
        @click="save"
        >{{ $tr("校验并导入") }}</el-button
      ></template
    >
  </el-dialog>
</template>
<script setup lang="ts">
import EnvironmentSelect from "./EnvironmentSelect.vue";
import "./certificate-dialog.css";
import { onUnmounted, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import { tr } from "@/locales";
import { webRequest, type WebResource } from "@/api/webservice";
const props = defineProps<{
  modelValue: boolean;
  groups: { id: number; name: string }[];
  initialGroupId?: number;
  initialEnvironment?: string;
}>();
const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  saved: [certificate: WebResource];
}>();
const name = ref(""),
  groupId = ref<number>(),
  environment = ref(""),
  certificate = ref(""),
  privateKey = ref(""),
  error = ref(""),
  saving = ref(false);
let generation = 0;
const fileReads = { certificate: 0, privateKey: 0 };
watch(
  () => props.modelValue,
  open => {
    generation++;
    fileReads.certificate++;
    fileReads.privateKey++;
    name.value = "";
    certificate.value = "";
    privateKey.value = "";
    error.value = "";
    saving.value = false;
    if (open) {
      groupId.value =
        props.initialGroupId ||
        (props.groups.length === 1 ? props.groups[0].id : undefined);
      environment.value = props.initialEnvironment || "";
    }
  }
);
async function readFile(event: Event, target: "certificate" | "privateKey") {
  const input = event.target as HTMLInputElement,
    file = input.files?.[0];
  input.value = "";
  if (!file) return;
  if (file.size > 1024 * 1024) {
    error.value = tr("单个证书或私钥文件不能超过 1 MiB");
    return;
  }
  const gen = generation,
    request = ++fileReads[target];
  try {
    const text = await file.text();
    if (gen !== generation || request !== fileReads[target]) return;
    if (target === "certificate") {
      certificate.value = text;
      if (!name.value)
        name.value = file.name.replace(/\.[^.]+$/, "").slice(0, 100);
    } else privateKey.value = text;
  } catch (e) {
    if (gen === generation) error.value = (e as Error).message;
  }
}
async function save() {
  if (saving.value) return;
  const gen = generation;
  saving.value = true;
  error.value = "";
  try {
    const result = await webRequest<{ certificate: WebResource }>(
      "/certificates/import",
      "POST",
      {
        name: name.value,
        groupId: groupId.value,
        environment: environment.value,
        certificate: certificate.value,
        privateKey: privateKey.value
      }
    );
    if (gen !== generation) return;
    ElMessage.success(tr("证书校验通过并已导入"));
    emit("saved", result.certificate);
    emit("update:modelValue", false);
  } catch (e) {
    if (gen === generation) error.value = (e as Error).message;
  } finally {
    if (gen === generation) saving.value = false;
  }
}
onUnmounted(() => {
  generation++;
  certificate.value = "";
  privateKey.value = "";
});
</script>
<style scoped>
.import-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 0 20px;
}
.import-grid .el-select {
  width: 100%;
}
.file-picker {
  display: inline-flex;
  align-items: center;
  padding: 4px 12px;
  margin-bottom: 10px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  color: var(--el-color-primary);
  cursor: pointer;
}
.file-picker input {
  width: 0;
  height: 0;
  opacity: 0;
}
:deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, monospace;
}
.el-alert {
  margin-bottom: 16px;
}
@media (max-width: 640px) {
  .import-grid {
    grid-template-columns: 1fr;
  }
}
</style>
