<template>
  <el-drawer
    class="web-certificate-drawer"
    modal-class="web-certificate-overlay"
    append-to-body
    destroy-on-close
    :lock-scroll="false"
    :model-value="modelValue"
    :title="account?.name || $tr('DNS 账户')"
    size="min(800px, calc(100vw - 24px))"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <template v-if="account">
      <el-alert
        :title="$tr('连接检查只读取 Cloudflare 数据，不创建或删除 DNS 记录')"
        type="info"
        :closable="false"
      />
      <div class="inspection-toolbar">
        <el-input
          v-model="domain"
          clearable
          :placeholder="$tr('可选：输入要申请证书的域名')"
          @keyup.enter="check"
        /><el-button type="primary" :loading="checking" @click="check">{{
          $tr("检查连接")
        }}</el-button>
      </div>
      <el-alert
        v-if="error"
        :title="error"
        type="error"
        :closable="false"
        show-icon
      />
      <template v-if="result">
        <el-alert
          :title="
            $tr(
              result.valid ? 'Cloudflare 读取检查通过' : 'Cloudflare 检查未通过'
            )
          "
          :type="result.valid ? 'success' : 'warning'"
          :closable="false"
        />
        <p class="muted">
          {{ $tr("检查时间") }}：{{
            new Date(result.checkedAt).toLocaleString()
          }}
        </p>
        <el-table :data="result.checks"
          ><el-table-column
            prop="name"
            :label="$tr('检查项')"
            min-width="130" /><el-table-column :label="$tr('结果')" width="90"
            ><template #default="{ row }"
              ><el-tag :type="row.passed ? 'success' : 'warning'">{{
                $tr(row.passed ? "通过" : "未通过")
              }}</el-tag></template
            ></el-table-column
          ><el-table-column prop="message" :label="$tr('说明')" min-width="200"
        /></el-table>
        <p v-if="result.zone" class="matched-zone">
          {{ $tr(result.domain ? "匹配域名区域" : "抽样检查区域") }}：<strong>{{
            result.zone.name
          }}</strong>
        </p>
      </template>
      <p class="muted">
        {{ $tr("账户检查只抽样一个可用区域；申请时会逐个检查证书域名。") }}
      </p>
      <p class="muted">
        {{
          $tr(
            "读取成功不代表已验证 DNS 写权限；签发需要目标区域的 DNS:Edit 权限。"
          )
        }}
      </p>
      <div class="inspection-toolbar">
        <h3>{{ $tr("可访问的域名区域") }}</h3>
        <el-button :loading="loadingZones" @click="loadZones">{{
          $tr("读取域名区域")
        }}</el-button>
      </div>
      <el-alert
        v-if="zoneError"
        :title="zoneError"
        type="error"
        :closable="false"
      />
      <el-input
        v-if="zoneData"
        v-model="search"
        :placeholder="$tr('搜索域名区域')"
        clearable
      />
      <el-table v-if="zoneData" :data="filteredZones" class="zone-table"
        ><el-table-column
          prop="name"
          :label="$tr('域名区域')"
          min-width="200"
        /><el-table-column :label="$tr('状态')" width="130"
          ><template #default="{ row }"
            ><el-tag :type="row.status === 'active' ? 'success' : 'warning'">{{
              row.status
            }}</el-tag></template
          ></el-table-column
        ></el-table
      >
      <p v-if="zoneData" class="muted">
        {{ $tr("已读取 {value0} 个区域", { value0: zoneData.zones.length }) }}
      </p>
      <el-alert
        v-if="zoneData?.truncated"
        :title="
          $tr('域名区域超过显示上限；目标未列出时请缩小 Token 授权范围后重试')
        "
        type="warning"
        :closable="false"
      />
    </template>
  </el-drawer>
</template>
<script setup lang="ts">
import "./certificate-dialog.css";
import { computed, onUnmounted, ref, watch } from "vue";
import {
  webCheckDNSAccount,
  webDNSZones,
  type WebDNSCheck,
  type WebDNSZones,
  type WebResource
} from "@/api/webservice";
const props = defineProps<{
  modelValue: boolean;
  account?: WebResource;
  initialAction?: "check" | "zones";
}>();
const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  checked: [];
}>();
const domain = ref(""),
  search = ref(""),
  error = ref(""),
  zoneError = ref(""),
  checking = ref(false),
  loadingZones = ref(false);
const result = ref<WebDNSCheck>(),
  zoneData = ref<WebDNSZones>();
let generation = 0,
  checkGeneration = 0,
  zoneGeneration = 0;
let checkAbort: AbortController | undefined,
  zoneAbort: AbortController | undefined;
const filteredZones = computed(
  () =>
    zoneData.value?.zones.filter(z =>
      z.name.includes(search.value.trim().toLowerCase())
    ) || []
);
async function check() {
  if (!props.account || checking.value) return;
  checkAbort?.abort();
  checkAbort = new AbortController();
  const gen = generation,
    request = ++checkGeneration,
    id = props.account.id,
    target = domain.value.trim();
  checking.value = true;
  error.value = "";
  result.value = undefined;
  try {
    const value = await webCheckDNSAccount(id, target, checkAbort.signal);
    if (gen !== generation || request !== checkGeneration) return;
    result.value = value;
    emit("checked");
  } catch (e) {
    if (gen === generation && request === checkGeneration)
      error.value = (e as Error).message;
  } finally {
    if (gen === generation && request === checkGeneration)
      checking.value = false;
  }
}
async function loadZones() {
  if (!props.account) return;
  zoneAbort?.abort();
  zoneAbort = new AbortController();
  const gen = generation,
    request = ++zoneGeneration;
  loadingZones.value = true;
  zoneError.value = "";
  zoneData.value = undefined;
  try {
    const value = await webDNSZones(props.account.id, zoneAbort.signal);
    if (gen === generation && request === zoneGeneration)
      zoneData.value = value;
  } catch (e) {
    if (gen === generation && request === zoneGeneration)
      zoneError.value = (e as Error).message;
  } finally {
    if (gen === generation && request === zoneGeneration)
      loadingZones.value = false;
  }
}
watch(
  () => [props.modelValue, props.account?.id] as const,
  ([open]) => {
    generation++;
    checkGeneration++;
    zoneGeneration++;
    checkAbort?.abort();
    zoneAbort?.abort();
    checking.value = false;
    loadingZones.value = false;
    domain.value = "";
    search.value = "";
    error.value = "";
    zoneError.value = "";
    result.value = undefined;
    zoneData.value = undefined;
    if (open && props.account) {
      if (props.initialAction === "zones") loadZones();
      else check();
    }
  }
);
watch(
  domain,
  () => {
    checkGeneration++;
    checkAbort?.abort();
    checking.value = false;
    result.value = undefined;
    error.value = "";
  },
  { flush: "sync" }
);
onUnmounted(() => {
  generation++;
  checkAbort?.abort();
  zoneAbort?.abort();
});
</script>
<style scoped>
.inspection-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin: 20px 0;
}
.inspection-toolbar .el-input {
  flex: 1;
  min-width: min(220px, 100%);
}
.inspection-toolbar h3 {
  flex: 1;
  margin: 0;
}
.el-alert {
  margin-bottom: 14px;
}
.muted {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.65;
}
.zone-table {
  margin-top: 16px;
}
.matched-zone {
  overflow-wrap: anywhere;
}
</style>
