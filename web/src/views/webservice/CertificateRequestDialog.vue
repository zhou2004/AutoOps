<template>
  <el-dialog
    class="web-certificate-dialog"
    modal-class="web-certificate-overlay"
    append-to-body
    destroy-on-close
    align-center
    :lock-scroll="false"
    :model-value="modelValue"
    :title="$tr('通过 Cloudflare 申请证书')"
    width="min(900px, calc(100vw - 32px))"
    :close-on-click-modal="!busy"
    :close-on-press-escape="!busy"
    :show-close="!busy"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <el-alert
      :title="$tr('Cloudflare 负责 DNS 域名验证，证书由所选 ACME 签发机构提供')"
      type="info"
      :closable="false"
    />
    <el-form label-position="top" :disabled="busy" @submit.prevent>
      <div class="request-grid">
        <el-form-item :label="$tr('证书名称')" required
          ><el-input v-model="name" maxlength="100"
        /></el-form-item>
        <el-form-item :label="$tr('业务分组')" required
          ><el-select v-model="groupId" filterable
            ><el-option
              v-for="group in groups"
              :key="group.id"
              :label="group.name"
              :value="group.id" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('环境')"
          ><EnvironmentSelect v-model="environment"
        /></el-form-item>
        <el-form-item :label="$tr('Cloudflare DNS 账户')" required
          ><el-select
            v-model="dnsAccountId"
            filterable
            :placeholder="$tr('选择已保存的 Cloudflare 账户')"
            @change="changeDNS"
            ><el-option
              v-for="account in cloudflareAccounts"
              :key="account.id"
              :label="account.name"
              :value="account.id" /></el-select
        ></el-form-item>
      </div>
      <el-alert
        v-if="!cloudflareAccounts.length"
        :title="$tr('请先添加 Cloudflare DNS 账户，再申请证书')"
        type="warning"
        :closable="false"
        ><el-button link type="primary" @click="emit('configure-dns')">{{
          $tr("配置 DNS 账户")
        }}</el-button></el-alert
      >
      <div class="zone-heading">
        <h3>{{ $tr("域名验证") }}</h3>
        <el-button
          link
          :loading="loadingZones"
          :disabled="!dnsAccountId"
          @click="loadZones"
          >{{ $tr("刷新域名区域") }}</el-button
        >
      </div>
      <el-alert
        v-if="zoneError"
        :title="zoneError"
        type="warning"
        :closable="false"
      />
      <el-form-item :label="$tr('域名区域（可选）')"
        ><el-select
          v-model="zoneId"
          filterable
          clearable
          :loading="loadingZones"
          :placeholder="$tr('选择区域快速添加根域名或泛域名')"
          ><el-option
            v-for="zone in zones"
            :key="zone.id"
            :label="zone.name"
            :value="zone.id"
            :disabled="zone.status !== 'active'" /></el-select
      ></el-form-item>
      <div class="domain-shortcuts">
        <el-button :disabled="!selectedZone" @click="addDomain(false)">{{
          $tr("添加根域名")
        }}</el-button
        ><el-button :disabled="!selectedZone" @click="addDomain(true)">{{
          $tr("添加泛域名")
        }}</el-button>
      </div>
      <el-alert
        v-if="zonesTruncated"
        type="info"
        :closable="false"
        :title="
          $tr('域名区域超过显示上限；目标未列出时请缩小 Token 授权范围后重试')
        "
      />
      <el-form-item :label="$tr('证书域名（SAN）')" required
        ><el-input
          v-model="domainText"
          type="textarea"
          :rows="4"
          :placeholder="$tr('每行一个域名；泛域名示例：*.example.com')"
          spellcheck="false"
        />
        <p class="hint">
          {{
            $tr(
              "泛域名不包含根域名，如需同时使用请分别添加。支持同账户下多个域名区域。"
            )
          }}
        </p></el-form-item
      >
      <el-button
        :loading="checking"
        :disabled="!dnsAccountId || !domains.length"
        @click="checkDomains"
        >{{ $tr("检查域名与 DNS 权限") }}</el-button
      >
      <div v-if="checks.length" class="domain-checks">
        <div v-for="check in checks" :key="check.domain" class="domain-check">
          <strong>{{ check.domain }}</strong
          ><el-tag
            :type="
              check.result.valid && check.result.zone ? 'success' : 'warning'
            "
            >{{
              $tr(
                check.result.valid && check.result.zone
                  ? "读取检查通过"
                  : "检查未通过"
              )
            }}</el-tag
          ><span v-if="check.result.zone" class="hint">{{
            check.result.zone.name
          }}</span>
          <p
            v-for="item in check.result.checks.filter(item => !item.passed)"
            :key="item.name"
            class="check-message"
          >
            {{ item.message }}
          </p>
        </div>
      </div>
      <p class="hint">
        {{
          $tr(
            "读取检查不会修改 DNS，也不能证明 DNS:Edit 权限；提交签发后才会创建并清理挑战 TXT。"
          )
        }}
      </p>
      <div class="zone-heading">
        <h3>{{ $tr("签发账户") }}</h3>
        <el-button link type="primary" @click="createAcme = true">{{
          $tr("新增 ACME 账户")
        }}</el-button>
      </div>
      <el-alert
        v-if="!availableAcme.length"
        type="warning"
        :closable="false"
        :title="$tr('尚无 ACME 账户，请填写联系邮箱并选择测试或正式签发环境')"
      />
      <el-form-item :label="$tr('ACME 账户')" required
        ><el-select
          v-model="acmeAccountId"
          filterable
          :placeholder="$tr('选择证书签发账户')"
          ><el-option
            v-for="account in availableAcme"
            :key="account.id"
            :value="account.id"
            :label="account.name + ' · ' + account.spec.email" /></el-select
      ></el-form-item>
      <p v-if="selectedAcme" class="hint">
        {{ selectedAcme.spec.directoryUrl }}
      </p>
      <el-alert
        v-if="selectedAcme?.spec.directoryUrl.includes('acme-staging')"
        type="warning"
        :closable="false"
        :title="$tr('当前选择测试 CA，签发的证书不受浏览器信任')"
      />
      <p class="hint">
        {{
          $tr("保存后进入签发预检，确认执行才会申请证书。签发与部署分别记录。")
        }}
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
      ><el-button :disabled="busy" @click="emit('update:modelValue', false)">{{
        $tr("取消")
      }}</el-button
      ><el-button
        :disabled="!readyToSave || busy"
        :loading="saving"
        @click="save(false)"
        >{{ $tr("保存申请") }}</el-button
      ><el-button
        v-if="canExecute"
        type="primary"
        :disabled="!readyToSave || busy"
        :loading="saving"
        @click="save(true)"
        >{{ $tr("保存并预检签发") }}</el-button
      ></template
    >
    <AcmeAccountDialog
      v-model="createAcme"
      :groups="groups"
      :initial-group-id="groupId"
      :initial-environment="environment"
      @saved="acmeCreated"
    />
  </el-dialog>
</template>
<script setup lang="ts">
import EnvironmentSelect from "./EnvironmentSelect.vue";
import "./certificate-dialog.css";
import { computed, onUnmounted, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import { tr } from "@/locales";
import {
  webCheckDNSAccount,
  webDNSZones,
  webRequest,
  type WebDNSCheck,
  type WebDNSZone,
  type WebResource
} from "@/api/webservice";
import AcmeAccountDialog from "./AcmeAccountDialog.vue";
const props = defineProps<{
  modelValue: boolean;
  groups: { id: number; name: string }[];
  dnsAccounts: WebResource[];
  acmeAccounts: WebResource[];
  initialGroupId?: number;
  initialEnvironment?: string;
  canExecute: boolean;
}>();
const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  saved: [certificate: WebResource, issue: boolean];
  "configure-dns": [];
  "account-created": [];
}>();
const name = ref(""),
  environment = ref(""),
  groupId = ref<number>(),
  dnsAccountId = ref<number>(),
  acmeAccountId = ref<number>(),
  domainText = ref(""),
  zoneId = ref<string>();
const zones = ref<WebDNSZone[]>([]),
  zoneError = ref(""),
  zonesTruncated = ref(false),
  loadingZones = ref(false),
  checking = ref(false),
  saving = ref(false),
  error = ref("");
const checks = ref<{ domain: string; result: WebDNSCheck }[]>([]),
  checkedSignature = ref(""),
  createAcme = ref(false),
  createdAccounts = ref<WebResource[]>([]);
let generation = 0,
  zoneGeneration = 0,
  checkGeneration = 0;
let zoneAbort: AbortController | undefined,
  checkAbort: AbortController | undefined;
const cloudflareAccounts = computed(() =>
  props.dnsAccounts.filter(a => a.spec.provider === "cloudflare")
);
const selectedDNS = computed(() =>
  cloudflareAccounts.value.find(a => a.id === dnsAccountId.value)
);
const availableAcme = computed(() => [
  ...props.acmeAccounts,
  ...createdAccounts.value.filter(
    a => !props.acmeAccounts.some(p => p.id === a.id)
  )
]);
const selectedAcme = computed(() =>
  availableAcme.value.find(a => a.id === acmeAccountId.value)
);
const selectedZone = computed(() =>
  zones.value.find(z => z.id === zoneId.value)
);
const domains = computed(() => [
  ...new Set(
    domainText.value
      .split(/[\s,，]+/)
      .map(d => d.trim().toLowerCase())
      .filter(Boolean)
  )
]);
const domainSignature = computed(() =>
  JSON.stringify({
    id: dnsAccountId.value,
    revision: selectedDNS.value?.revision,
    domains: domains.value
  })
);
const busy = computed(() => checking.value || saving.value);
const readyToSave = computed(
  () =>
    !!name.value.trim() &&
    !!groupId.value &&
    !!selectedDNS.value &&
    !!selectedAcme.value &&
    checkedSignature.value === domainSignature.value &&
    checks.value.length === domains.value.length &&
    checks.value.every(c => c.result.valid && c.result.zone)
);
async function loadZones() {
  zoneAbort?.abort();
  zoneAbort = new AbortController();
  const gen = generation,
    request = ++zoneGeneration;
  zones.value = [];
  zoneId.value = undefined;
  zoneError.value = "";
  zonesTruncated.value = false;
  if (!dnsAccountId.value) {
    loadingZones.value = false;
    return;
  }
  loadingZones.value = true;
  try {
    const result = await webDNSZones(dnsAccountId.value, zoneAbort.signal);
    if (gen !== generation || request !== zoneGeneration) return;
    zones.value = result.zones;
    zonesTruncated.value = result.truncated;
    if (result.zones.length === 1 && result.zones[0].status === "active")
      zoneId.value = result.zones[0].id;
  } catch (e) {
    if (gen === generation && request === zoneGeneration)
      zoneError.value = (e as Error).message;
  } finally {
    if (gen === generation && request === zoneGeneration)
      loadingZones.value = false;
  }
}
function changeDNS() {
  if (!groupId.value && selectedDNS.value)
    groupId.value = selectedDNS.value.groupId;
  loadZones();
}
function addDomain(wildcard: boolean) {
  if (!selectedZone.value) return;
  const domain = (wildcard ? "*." : "") + selectedZone.value.name;
  if (!domains.value.includes(domain))
    domainText.value = [...domains.value, domain].join("\n");
  if (!name.value) name.value = selectedZone.value.name;
}
async function checkDomains() {
  if (!dnsAccountId.value || !domains.value.length || busy.value) return;
  if (domains.value.length > 100) {
    error.value = tr("单张证书最多申请 100 个域名");
    return;
  }
  const gen = generation,
    request = ++checkGeneration,
    signature = domainSignature.value,
    id = dnsAccountId.value,
    targets = [...domains.value];
  checkAbort?.abort();
  checkAbort = new AbortController();
  checking.value = true;
  error.value = "";
  checks.value = [];
  checkedSignature.value = "";
  try {
    for (const domain of targets) {
      const result = await webCheckDNSAccount(id, domain, checkAbort.signal);
      if (
        gen !== generation ||
        request !== checkGeneration ||
        signature !== domainSignature.value
      )
        return;
      checks.value.push({ domain, result });
    }
    checkedSignature.value = signature;
  } catch (e) {
    if (gen === generation && request === checkGeneration)
      error.value = (e as Error).message;
  } finally {
    if (gen === generation && request === checkGeneration)
      checking.value = false;
  }
}
async function save(issue: boolean) {
  if (!readyToSave.value || busy.value) return;
  const gen = generation;
  saving.value = true;
  error.value = "";
  try {
    const result = await webRequest<WebResource>("/certificates", "POST", {
      name: name.value.trim(),
      groupId: groupId.value,
      environment: environment.value,
      engine: "nginx",
      spec: {
        owner: "platform",
        challenge: "dns-01",
        domains: domains.value,
        acmeAccountId: acmeAccountId.value,
        dnsAccountId: dnsAccountId.value,
        instanceIds: []
      }
    });
    if (gen !== generation) return;
    ElMessage.success(tr("证书申请已保存"));
    emit("saved", result, issue);
    emit("update:modelValue", false);
  } catch (e) {
    if (gen === generation) error.value = (e as Error).message;
  } finally {
    if (gen === generation) saving.value = false;
  }
}
function acmeCreated(account: WebResource) {
  createdAccounts.value.push(account);
  acmeAccountId.value = account.id;
  emit("account-created");
}
watch(
  domainSignature,
  () => {
    checkGeneration++;
    checkAbort?.abort();
    checking.value = false;
    checks.value = [];
    checkedSignature.value = "";
  },
  { flush: "sync" }
);
watch(
  () => props.modelValue,
  open => {
    generation++;
    zoneGeneration++;
    checkGeneration++;
    zoneAbort?.abort();
    checkAbort?.abort();
    loadingZones.value = false;
    checking.value = false;
    saving.value = false;
    error.value = "";
    zoneError.value = "";
    checks.value = [];
    checkedSignature.value = "";
    createAcme.value = false;
    if (!open) return;
    name.value = "";
    environment.value = props.initialEnvironment || "";
    groupId.value = props.initialGroupId;
    domainText.value = "";
    zoneId.value = undefined;
    zones.value = [];
    dnsAccountId.value =
      cloudflareAccounts.value.length === 1
        ? cloudflareAccounts.value[0].id
        : undefined;
    acmeAccountId.value =
      availableAcme.value.length === 1 ? availableAcme.value[0].id : undefined;
    changeDNS();
  }
);
onUnmounted(() => {
  generation++;
  zoneAbort?.abort();
  checkAbort?.abort();
});
</script>
<style scoped>
.request-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 0 20px;
}
.el-select {
  width: 100%;
}
.el-alert {
  margin-bottom: 16px;
}
.zone-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 16px;
}
.zone-heading h3 {
  font-size: 16px;
  margin: 0 0 12px;
}
.domain-shortcuts {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 18px;
}
.domain-shortcuts .el-button {
  margin-left: 0;
}
.hint {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.65;
  overflow-wrap: anywhere;
  margin: 8px 0;
}
.domain-checks {
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  padding: 10px 14px;
  margin-top: 14px;
}
.domain-check {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding: 6px 0;
}
.domain-check strong {
  overflow-wrap: anywhere;
}
.check-message {
  width: 100%;
  margin: 0;
  font-size: 13px;
  color: var(--el-color-warning);
}
@media (max-width: 640px) {
  .request-grid {
    grid-template-columns: 1fr;
  }
}
</style>
