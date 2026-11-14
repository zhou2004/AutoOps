<template>
  <el-dialog
    class="web-certificate-dialog"
    modal-class="web-certificate-overlay"
    append-to-body
    destroy-on-close
    align-center
    :lock-scroll="false"
    :model-value="modelValue"
    :title="$tr(account ? '编辑 DNS 账户' : '新增 DNS 账户')"
    width="min(760px, calc(100vw - 32px))"
    :close-on-click-modal="!saving"
    :close-on-press-escape="!saving"
    :show-close="!saving"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <el-form label-position="top" :disabled="saving" @submit.prevent>
      <div class="account-grid">
        <el-form-item :label="$tr('账户名称')" required
          ><el-input v-model="name" maxlength="100"
        /></el-form-item>
        <el-form-item :label="$tr('DNS 服务商')" required
          ><el-select v-model="provider" @change="changeProvider"
            ><el-option value="cloudflare" label="Cloudflare" /><el-option
              value="alidns"
              :label="$tr('阿里云 DNS')" /><el-option
              value="tencentcloud"
              :label="$tr('腾讯云 DNSPod')" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('业务分组')" required
          ><el-select v-model="groupId" filterable
            ><el-option
              v-for="group in groups"
              :key="group.id"
              :value="group.id"
              :label="group.name" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('环境')"
          ><EnvironmentSelect v-model="environment"
        /></el-form-item>
      </div>
      <el-alert
        v-if="provider === 'cloudflare'"
        type="info"
        :closable="false"
        :title="
          $tr('使用 Cloudflare API Token，授权目标区域的 Zone:Read 和 DNS:Edit')
        "
      >
        <p>
          {{ $tr("建议仅授权需要申请证书的域名区域；Token 保存后不会回显。") }}
        </p>
      </el-alert>
      <el-form-item :label="$tr('凭据来源')" required>
        <el-radio-group
          v-model="credentialSource"
          @change="sourceTouched = true"
          ><el-radio-button value="configuration">{{
            $tr("配置中心凭据")
          }}</el-radio-button
          ><el-radio-button value="dedicated">{{
            $tr("专用凭据")
          }}</el-radio-button></el-radio-group
        >
      </el-form-item>
      <template v-if="credentialSource === 'configuration'">
        <el-form-item :label="$tr('选择已保存的凭据')" required
          ><el-select
            v-model="credentialId"
            :loading="loadingCredentials"
            filterable
            :placeholder="$tr('选择与 DNS 服务商匹配的凭据')"
            ><el-option
              v-for="item in credentials"
              :key="item.id"
              :value="item.id"
              :label="item.name" /></el-select
          ><el-button
            link
            :loading="loadingCredentials"
            @click="loadCredentials"
            >{{ $tr("刷新凭据") }}</el-button
          ></el-form-item
        >
        <el-alert
          v-if="!loadingCredentials && !credentials.length && !credentialError"
          type="info"
          :closable="false"
          :title="$tr('暂无匹配凭据，可先到配置中心添加，或改用专用凭据')"
        />
        <el-alert
          v-if="credentialError"
          type="error"
          :closable="false"
          :title="credentialError"
        />
      </template>
      <template v-else>
        <el-checkbox v-if="canKeepSecret" v-model="replaceSecret">{{
          $tr("更换已保存的凭据")
        }}</el-checkbox>
        <p v-if="canKeepSecret && !replaceSecret" class="muted">
          {{ $tr("继续使用已加密保存的凭据，不需要重新输入。") }}
        </p>
        <template v-if="!canKeepSecret || replaceSecret">
          <el-form-item
            :label="provider === 'cloudflare' ? 'API Token' : $tr('密钥 ID')"
            required
            ><el-input
              v-model="secretKey"
              type="password"
              show-password
              autocomplete="new-password"
              :placeholder="$tr('输入新的凭据，保存后不会回显')"
          /></el-form-item>
          <el-form-item
            v-if="provider !== 'cloudflare'"
            :label="$tr('密钥 Secret')"
            required
            ><el-input
              v-model="secretValue"
              type="password"
              show-password
              autocomplete="new-password"
          /></el-form-item>
        </template>
      </template>
    </el-form>
    <el-alert
      v-if="error"
      type="error"
      :closable="false"
      :title="error"
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
        :disabled="!canSave"
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
import {
  webDNSCredentials,
  webRequest,
  type WebDNSCredential,
  type WebResource
} from "@/api/webservice";
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
const name = ref(""),
  provider = ref("cloudflare"),
  groupId = ref<number>(),
  environment = ref("");
const credentialSource = ref<"configuration" | "dedicated">("configuration"),
  credentialId = ref<number>();
const credentials = ref<WebDNSCredential[]>([]),
  loadingCredentials = ref(false),
  credentialError = ref("");
const replaceSecret = ref(false),
  secretKey = ref(""),
  secretValue = ref(""),
  sourceTouched = ref(false),
  error = ref(""),
  saving = ref(false);
let generation = 0,
  credentialsRequest = 0;
let abort: AbortController | undefined;
const canKeepSecret = computed(
  () =>
    !!props.account &&
    !props.account.spec.credentialId &&
    props.account.spec.provider === provider.value
);
const canSave = computed(
  () =>
    !!name.value.trim() &&
    !!groupId.value &&
    (credentialSource.value === "configuration"
      ? !!credentialId.value &&
        credentials.value.some(c => c.id === credentialId.value)
      : (canKeepSecret.value && !replaceSecret.value) ||
        (!!secretKey.value.trim() &&
          (provider.value === "cloudflare" || !!secretValue.value.trim())))
);
async function loadCredentials() {
  abort?.abort();
  abort = new AbortController();
  const request = ++credentialsRequest,
    gen = generation;
  loadingCredentials.value = true;
  credentialError.value = "";
  try {
    const result = await webDNSCredentials(provider.value, abort.signal);
    if (gen !== generation || request !== credentialsRequest) return;
    credentials.value = result;
    if (!props.account && !sourceTouched.value)
      credentialSource.value = result.length ? "configuration" : "dedicated";
    if (!credentialId.value && result.length === 1)
      credentialId.value = result[0].id;
  } catch (e) {
    if (gen === generation && request === credentialsRequest) {
      credentials.value = [];
      credentialError.value = (e as Error).message;
    }
  } finally {
    if (gen === generation && request === credentialsRequest)
      loadingCredentials.value = false;
  }
}
function changeProvider() {
  credentialId.value = undefined;
  credentials.value = [];
  secretKey.value = "";
  secretValue.value = "";
  replaceSecret.value = false;
  loadCredentials();
}
async function save() {
  if (!canSave.value || saving.value) return;
  const gen = generation;
  saving.value = true;
  error.value = "";
  const secretInput =
    credentialSource.value === "dedicated" &&
    (!canKeepSecret.value || replaceSecret.value)
      ? JSON.stringify(
          provider.value === "cloudflare"
            ? { token: secretKey.value.trim() }
            : { key: secretKey.value.trim(), secret: secretValue.value.trim() }
        )
      : undefined;
  try {
    const result = await webRequest<WebResource>(
      "/dns-accounts" + (props.account ? "/" + props.account.id : ""),
      props.account ? "PUT" : "POST",
      {
        name: name.value.trim(),
        groupId: groupId.value,
        environment: environment.value,
        engine: "nginx",
        revision: props.account?.revision,
        spec: {
          provider: provider.value,
          credentialId:
            credentialSource.value === "configuration" ? credentialId.value : 0
        },
        secretInput
      }
    );
    if (gen !== generation) return;
    ElMessage.success(tr("DNS 账户已保存，可继续检查连接"));
    emit("saved", result);
    emit("update:modelValue", false);
  } catch (e) {
    if (gen === generation) error.value = (e as Error).message;
  } finally {
    if (gen === generation) saving.value = false;
  }
}
watch(
  () => props.modelValue,
  open => {
    generation++;
    credentialsRequest++;
    abort?.abort();
    loadingCredentials.value = false;
    saving.value = false;
    secretKey.value = "";
    secretValue.value = "";
    error.value = "";
    credentialError.value = "";
    credentials.value = [];
    replaceSecret.value = false;
    sourceTouched.value = false;
    if (!open) return;
    const account = props.account;
    name.value = account?.name || "";
    provider.value = account?.spec.provider || "cloudflare";
    groupId.value = account?.groupId || props.initialGroupId;
    environment.value = account?.environment || props.initialEnvironment || "";
    credentialId.value = account?.spec.credentialId || undefined;
    credentialSource.value =
      account && !credentialId.value ? "dedicated" : "configuration";
    loadCredentials();
  }
);
watch([credentialSource, replaceSecret], () => {
  secretKey.value = "";
  secretValue.value = "";
});
onUnmounted(() => {
  generation++;
  abort?.abort();
  secretKey.value = "";
  secretValue.value = "";
});
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
.el-alert {
  margin-bottom: 18px;
}
.el-checkbox {
  margin-bottom: 12px;
}
.muted {
  color: var(--el-text-color-secondary);
  margin-bottom: 16px;
}
@media (max-width: 640px) {
  .account-grid {
    grid-template-columns: 1fr;
  }
}
</style>
