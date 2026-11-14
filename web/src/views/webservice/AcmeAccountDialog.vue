<template>
  <el-dialog
    class="web-certificate-dialog"
    modal-class="web-certificate-overlay"
    append-to-body
    destroy-on-close
    align-center
    :lock-scroll="false"
    :model-value="modelValue"
    :title="$tr(account ? '编辑 ACME 账户' : '新增 ACME 账户')"
    width="min(680px, calc(100vw - 32px))"
    :close-on-click-modal="!saving"
    :close-on-press-escape="!saving"
    :show-close="!saving"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <el-form label-position="top" :disabled="saving" @submit.prevent>
      <el-form-item :label="$tr('账户名称')" required
        ><el-input v-model="name" maxlength="100"
      /></el-form-item>
      <div class="account-grid">
        <el-form-item :label="$tr('业务分组')" required
          ><el-select v-model="groupId" filterable
            ><el-option
              v-for="group in groups"
              :key="group.id"
              :label="group.name"
              :value="group.id" /></el-select></el-form-item
        ><el-form-item :label="$tr('环境')"
          ><EnvironmentSelect v-model="environment"
        /></el-form-item>
      </div>
      <el-form-item :label="$tr('联系邮箱')" required
        ><el-input v-model="email" type="email" autocomplete="email"
      /></el-form-item>
      <el-form-item :label="$tr('签发机构')" required
        ><el-select
          v-model="directoryUrl"
          :disabled="!!account"
          filterable
          allow-create
          default-first-option
          ><el-option
            :value="staging"
            :label="$tr('Let’s Encrypt 测试环境')" /><el-option
            :value="production"
            :label="$tr('Let’s Encrypt 正式环境')" /><el-option
            v-if="customDirectory"
            :value="customDirectory"
            :label="customDirectory" /></el-select
      ></el-form-item>
      <el-alert
        :title="
          $tr(
            directoryUrl === staging
              ? '测试证书仅用于联调，浏览器不会信任；对外服务请选择正式环境。'
              : '正式签发受 CA 的频率限制，请先确认域名和 DNS 账户。'
          )
        "
        :type="directoryUrl === staging ? 'info' : 'warning'"
        :closable="false"
      />
      <p class="muted">{{ directoryUrl }}</p>
      <p class="muted">
        {{ $tr("保存账户不会立即注册或签发证书；签发前仍需预检并确认执行。") }}
      </p>
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
        :disabled="!canSave"
        :loading="saving"
        @click="save"
        >{{ $tr("保存账户") }}</el-button
      ></template
    >
  </el-dialog>
</template>
<script setup lang="ts">
import EnvironmentSelect from "./EnvironmentSelect.vue";
import "./certificate-dialog.css";
import { computed, onUnmounted, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import { tr } from "@/locales";
import { webRequest, type WebResource } from "@/api/webservice";
const props = defineProps<{
  modelValue: boolean;
  account?: WebResource;
  groups: { id: number; name: string }[];
  initialGroupId?: number;
  initialEnvironment?: string;
}>();
const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  saved: [account: WebResource];
}>();
const staging = "https://acme-staging-v02.api.letsencrypt.org/directory",
  production = "https://acme-v02.api.letsencrypt.org/directory";
const name = ref(""),
  email = ref(""),
  environment = ref(""),
  groupId = ref<number>(),
  directoryUrl = ref(staging),
  error = ref(""),
  saving = ref(false);
let generation = 0;
const customDirectory = computed(() =>
  props.account?.spec.directoryUrl &&
  ![staging, production].includes(props.account.spec.directoryUrl)
    ? props.account.spec.directoryUrl
    : ""
);
const canSave = computed(
  () =>
    name.value.trim() &&
    groupId.value &&
    /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value.trim())
);
watch(
  () => props.modelValue,
  open => {
    generation++;
    saving.value = false;
    error.value = "";
    if (!open) return;
    name.value = props.account?.name || "";
    email.value = props.account?.spec.email || "";
    groupId.value = props.account?.groupId || props.initialGroupId;
    environment.value =
      props.account?.environment || props.initialEnvironment || "";
    directoryUrl.value = props.account?.spec.directoryUrl || staging;
  }
);
async function save() {
  if (!canSave.value || saving.value) return;
  const gen = generation;
  saving.value = true;
  error.value = "";
  try {
    const account = await webRequest<WebResource>(
      "/acme-accounts" + (props.account ? "/" + props.account.id : ""),
      props.account ? "PUT" : "POST",
      {
        name: name.value.trim(),
        groupId: groupId.value,
        environment: environment.value,
        engine: "nginx",
        revision: props.account?.revision,
        spec: { email: email.value.trim(), directoryUrl: directoryUrl.value }
      }
    );
    if (gen !== generation) return;
    ElMessage.success(tr("ACME 账户已保存"));
    emit("saved", account);
    emit("update:modelValue", false);
  } catch (e) {
    if (gen === generation) error.value = (e as Error).message;
  } finally {
    if (gen === generation) saving.value = false;
  }
}
onUnmounted(() => generation++);
</script>
<style scoped>
.account-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 0 20px;
}
.el-select {
  width: 100%;
}
.muted {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.65;
  overflow-wrap: anywhere;
}
@media (max-width: 640px) {
  .account-grid {
    grid-template-columns: 1fr;
  }
}
</style>
