<template>
  <el-drawer
    :model-value="modelValue"
    :title="data?.cluster?.name || $tr('集群详情')"
    size="min(1180px, 96vw)"
    append-to-body
    :lock-scroll="false"
    destroy-on-close
    @update:model-value="v => emit('update:modelValue', v)"
  >
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <el-skeleton v-if="loading && !data" :rows="8" animated />
    <template v-if="data">
      <div class="cluster-toolbar">
        <WebResourceIcon /><el-tag>Nginx</el-tag
        ><el-tag :type="tag(data.cluster.status)">{{
          $tr(statusNames[data.cluster.status] || data.cluster.status)
        }}</el-tag
        ><span class="muted">{{
          $tr(statusNames[data.cluster.management] || data.cluster.management)
        }}</span
        ><el-button :loading="loading" @click="load">{{
          $tr("刷新")
        }}</el-button
        ><el-button
          @click="emit('navigate', 'configurations', data.cluster.id)"
          >{{ $tr("配置管理") }}</el-button
        ><el-button @click="emit('navigate', 'logs', data.cluster.id)">{{
          $tr("日志管理")
        }}</el-button>
      </div>
      <el-button class="guide-button" @click="guideOpen = true">{{
        $tr("高可用配置教程")
      }}</el-button>
      <el-alert
        v-if="data.vip?.message"
        :title="$tr(data.vip.message)"
        type="info"
        :closable="false"
      />
      <el-tabs v-model="tab">
        <el-tab-pane :label="$tr('拓扑')" name="topology">
          <div class="topology" :aria-label="$tr('Nginx 集群拓扑')">
            <div class="entry-card">
              <el-icon :size="28"><Connection /></el-icon>
              <div>
                <strong>{{ data.vip?.cidr || $tr("业务访问入口") }}</strong
                ><small>{{
                  data.vip
                    ? `Keepalived · ${data.vip.interface} · VRID ${data.vip.routerId}`
                    : $tr("尚未配置 VIP，可在高可用标签中接入")
                }}</small>
              </div>
              <el-tag v-if="data.vip" :type="tag(data.vip.status)">{{
                $tr(statusNames[data.vip.status] || data.vip.status || "未知")
              }}</el-tag>
            </div>
            <div v-if="data.vip" class="vip-owner">
              {{ $tr("当前绑定：") }}{{ ownerName
              }}<small
                >{{ $tr("最近采集") }}
                {{ date(data.vip?.checkedAt || data.checkedAt) }}</small
              >
            </div>
            <div class="topology-nodes">
              <article
                v-for="node in data.members"
                :key="node.instance.id"
                class="node-branch"
                :class="{
                  owner:
                    data.vip?.ownerInstanceId === node.instance.id &&
                    data.vip?.status === 'healthy'
                }"
              >
                <div class="node-connector" />
                <div class="node-card">
                  <div class="node-heading">
                    <WebResourceIcon /><strong>{{ node.instance.name }}</strong>
                  </div>
                  <code>{{ node.address || "—" }}</code
                  ><el-tag :type="tag(node.status)">{{
                    $tr(statusNames[node.status] || node.status || "未知")
                  }}</el-tag
                  ><small class="node-version" :title="node.version">{{
                    versionLabel(node.version)
                  }}</small
                  ><el-tag
                    v-if="data.vip?.instanceIds?.includes(node.instance.id)"
                    :type="
                      data.vip?.ownerInstanceId === node.instance.id &&
                      data.vip?.status === 'healthy'
                        ? 'success'
                        : 'info'
                    "
                    size="small"
                    >{{
                      $tr(
                        data.vip?.ownerInstanceId === node.instance.id &&
                          data.vip?.status === "healthy"
                          ? "承载 VIP"
                          : "VIP 备选节点"
                      )
                    }}</el-tag
                  ><code class="digest" :title="node.configDigest"
                    >{{ $tr("配置摘要") }}
                    {{ node.configDigest?.slice(0, 12) || "—" }}</code
                  >
                </div>
              </article>
            </div>
            <el-empty
              v-if="!data.members?.length"
              :description="$tr('集群尚无成员，请先加入已发现的实例')"
            />
          </div>
          <el-alert
            :title="
              $tr(
                '拓扑展示配置与实时观测；只有完成唯一绑定和业务探测后才显示 VIP 正常。'
              )
            "
            type="info"
            :closable="false"
          />
        </el-tab-pane>
        <el-tab-pane :label="$tr('节点')" name="nodes">
          <el-table
            :data="data.members"
            class="asset-table"
            row-key="instance.id"
            ><el-table-column
              prop="instance.name"
              :label="$tr('实例')"
              min-width="160" /><el-table-column
              prop="address"
              :label="$tr('地址')"
              min-width="150" /><el-table-column
              prop="version"
              :label="$tr('版本')"
              min-width="160"
              ><template #default="{ row }"
                ><span class="node-version" :title="row.version">{{
                  versionLabel(row.version)
                }}</span></template
              ></el-table-column
            ><el-table-column :label="$tr('状态')" width="110"
              ><template #default="{ row }"
                ><el-tag :type="tag(row.status)">{{
                  $tr(statusNames[row.status] || row.status || "未知")
                }}</el-tag></template
              ></el-table-column
            ><el-table-column
              prop="configDigest"
              :label="$tr('配置摘要')"
              min-width="230"
              show-overflow-tooltip /><el-table-column
              prop="identity"
              :label="$tr('实例身份')"
              min-width="220"
              show-overflow-tooltip
          /></el-table>
        </el-tab-pane>
        <el-tab-pane :label="$tr('业务入口')" name="entries"
          ><el-table :data="data.entries || []" class="asset-table"
            ><el-table-column
              prop="name"
              :label="$tr('名称')" /><el-table-column
              prop="url"
              label="URL"
              min-width="260"
              show-overflow-tooltip /><el-table-column
              prop="host"
              label="Host" /><el-table-column
              prop="expectedStatus"
              :label="$tr('预期状态码')" /></el-table
        ></el-tab-pane>
        <el-tab-pane :label="$tr('高可用 / VIP')" name="vip">
          <el-alert
            :title="
              $tr(
                '为所选节点部署独立 Keepalived 服务，配置与切换均经过预检、确认和任务审计。'
              )
            "
            type="info"
            :closable="false"
          />
          <el-form label-position="top" class="vip-form" @submit.prevent
            ><div class="vip-grid">
              <el-form-item label="VIP / CIDR" required
                ><el-input
                  v-model="vip.cidr"
                  placeholder="192.168.x.x/24"
                  :disabled="!!data.vip"
              /></el-form-item>
              <el-form-item :label="$tr('网卡')" required
                ><el-input
                  v-model="vip.interface"
                  :placeholder="$tr('承载 VIP 的网卡名称')"
                  :disabled="!!data.vip"
              /></el-form-item>
              <el-form-item label="VRID" required
                ><el-input-number
                  v-model="vip.routerId"
                  :min="1"
                  :max="255"
                  :disabled="!!data.vip"
              /></el-form-item>
              <el-form-item :label="$tr('首选节点')"
                ><el-select
                  v-model="vip.preferredInstanceId"
                  :disabled="!!data.vip"
                  ><el-option
                    v-for="m in vipMembers"
                    :key="m.instance.id"
                    :value="m.instance.id"
                    :label="m.instance.name" /></el-select
              ></el-form-item>
              <el-form-item :label="$tr('VIP 成员')" required class="full"
                ><el-select
                  v-model="vip.instanceIds"
                  multiple
                  :disabled="!!data.vip"
                  ><el-option
                    v-for="m in data.members"
                    :key="m.instance.id"
                    :value="m.instance.id"
                    :label="`${m.instance.name} · ${m.address}`"
                    :disabled="
                      m.instance.spec?.runtime === 'docker' ||
                      m.instance.management !== 'operate'
                    " /></el-select
              ></el-form-item></div
          ></el-form>
          <div class="cluster-toolbar">
            <el-button
              v-for="action in vipActions"
              :key="action.action"
              :type="action.action === 'configure-vip' ? 'primary' : 'default'"
              :disabled="
                !action.enabled ||
                (action.action === 'configure-vip' && !validVIP)
              "
              :title="$tr(action.reason)"
              @click="
                emit(
                  'action',
                  data.cluster,
                  action.action,
                  action.action === 'configure-vip'
                    ? { vip: { ...vip, instanceIds: [...vip.instanceIds] } }
                    : {}
                )
              "
              >{{ $tr(action.label) }}</el-button
            >
          </div>
          <el-descriptions v-if="data.vip" :column="1" border
            ><el-descriptions-item :label="$tr('当前绑定：')">{{
              ownerName
            }}</el-descriptions-item
            ><el-descriptions-item :label="$tr('最近检查')">{{
              date(data.vip.checkedAt)
            }}</el-descriptions-item
            ><el-descriptions-item :label="$tr('状态')">{{
              $tr(statusNames[data.vip.status] || data.vip.status || "未知")
            }}</el-descriptions-item></el-descriptions
          >
          <el-alert
            v-if="data.vip?.message"
            :title="data.vip.message"
            type="warning"
            :closable="false"
          />
        </el-tab-pane>
      </el-tabs>
    </template>
    <SetupGuide v-model="guideOpen" kind="vip" />
  </el-drawer>
</template>
<script setup lang="ts">
import SetupGuide from "./SetupGuide.vue";
import { computed, ref, watch, onUnmounted } from "vue";
import Connection from "~icons/ep/connection";
import WebResourceIcon from "./WebResourceIcon.vue";
import {
  webRequest,
  type WebCapability,
  type WebResource
} from "@/api/webservice";
import { canPlatform } from "@/utils/platformAccess";
import { tr } from "@/locales";
import { statusNames } from "./definitions";
const guideOpen = ref(false);
const props = defineProps<{
  modelValue: boolean;
  clusterId?: number;
  initialTab?: string;
}>();
const emit = defineEmits<{
  "update:modelValue": [boolean];
  action: [WebResource, string, Record<string, any>];
  navigate: [string, number];
}>();
const data = ref<any>(),
  capabilities = ref<WebCapability[]>([]),
  loading = ref(false),
  error = ref(""),
  tab = ref("topology");
const vip = ref({
  cidr: "",
  interface: "",
  routerId: 51,
  instanceIds: [] as number[],
  preferredInstanceId: undefined as number | undefined
});
let generation = 0,
  abort: AbortController | undefined;
const vipMembers = computed(() =>
  (data.value?.members || []).filter(m =>
    vip.value.instanceIds.includes(m.instance.id)
  )
);
const validVIP = computed(
  () =>
    !!vip.value.cidr.trim() &&
    !!vip.value.interface.trim() &&
    vip.value.instanceIds.length >= 2
);
const vipActions = computed(() =>
  canPlatform("web:execute")
    ? capabilities.value.filter(a =>
        ["configure-vip", "verify-vip", "switch-vip"].includes(a.action)
      )
    : []
);
const ownerName = computed(() => {
  const observed = data.value?.vip;
  if (!observed || observed.status !== "healthy" || !observed.ownerInstanceId)
    return tr("未知");
  const m = data.value.members.find(
    m => m.instance.id === observed.ownerInstanceId
  );
  return m
    ? `${m.instance.name} · ${m.address}`
    : "#" + observed.ownerInstanceId;
});
function tag(s: string): "success" | "danger" | "info" | "warning" {
  return s === "healthy"
    ? "success"
    : ["failed", "unavailable"].includes(s)
      ? "danger"
      : ["drifted", "degraded"].includes(s)
        ? "warning"
        : "info";
}
function date(v?: string) {
  return v ? new Date(v).toLocaleString() : "—";
}
function versionLabel(value?: string) {
  return value?.split(/\r?\n/, 1)[0]?.trim() || "Nginx";
}
async function load() {
  if (!props.clusterId || !props.modelValue) return;
  const gen = ++generation;
  abort?.abort();
  abort = new AbortController();
  loading.value = true;
  error.value = "";
  try {
    const actionsPromise = webRequest<WebCapability[]>(
      "/clusters/" + props.clusterId + "/capabilities",
      "GET",
      undefined,
      abort.signal
    )
      .then(actions => {
        if (gen === generation && props.modelValue)
          capabilities.value = actions;
      })
      .catch(e => {
        if (gen === generation) error.value = (e as Error).message;
      });
    const snapshot = await webRequest<any>(
      "/clusters/" + props.clusterId + "/topology",
      "GET",
      undefined,
      abort.signal
    );
    void actionsPromise;
    if (gen !== generation || !props.modelValue) return;
    if (snapshot.vip?.config)
      snapshot.vip = { ...snapshot.vip.config, ...snapshot.vip };
    data.value = snapshot;
    const current = snapshot.vip || snapshot.cluster.spec?.vip;
    vip.value = current
      ? {
          cidr: current.cidr,
          interface: current.interface,
          routerId: current.routerId,
          instanceIds: [...current.instanceIds],
          preferredInstanceId: current.preferredInstanceId
        }
      : {
          cidr: "",
          interface: "",
          routerId: (props.clusterId % 200) + 50,
          instanceIds: snapshot.members
            .filter(
              m =>
                m.instance.management === "operate" &&
                m.instance.spec?.runtime !== "docker"
            )
            .map(m => m.instance.id),
          preferredInstanceId: snapshot.members[0]?.instance.id
        };
  } catch (e) {
    if (gen === generation) error.value = (e as Error).message;
  } finally {
    if (gen === generation) loading.value = false;
  }
}
watch(
  () => [props.modelValue, props.clusterId] as const,
  ([open]) => {
    generation++;
    abort?.abort();
    data.value = undefined;
    capabilities.value = [];
    error.value = "";
    if (open) {
      tab.value = props.initialTab || "topology";
      load();
    }
  },
  { immediate: true }
);
watch(
  () => vip.value.instanceIds,
  ids => {
    if (!ids.includes(vip.value.preferredInstanceId!))
      vip.value.preferredInstanceId = ids[0];
  }
);
onUnmounted(() => {
  generation++;
  abort?.abort();
});
</script>
<style scoped>
.guide-button {
  margin-bottom: 12px;
}
.cluster-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin: 0 0 20px;
}
.cluster-toolbar .el-button {
  margin: 0;
}
.muted,
small {
  color: var(--el-text-color-secondary);
}
.topology {
  padding: 24px 12px;
  min-width: 0;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  margin-bottom: 18px;
}
.entry-card {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin: auto;
  padding: 16px;
  max-width: 620px;
  border: 1px solid var(--el-border-color);
  background: var(--el-bg-color);
  border-radius: 10px;
}
.entry-card small,
.vip-owner small {
  display: block;
  margin-top: 6px;
}
.vip-owner {
  text-align: center;
  margin: 12px;
}
.topology-nodes {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 20px;
}
.node-connector {
  height: 44px;
  width: 2px;
  background: var(--el-border-color);
  margin: auto;
}
.owner .node-connector {
  background: var(--el-color-success);
}
.node-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 10px;
  padding: 18px;
  min-width: 0;
}
.owner .node-card {
  border-color: var(--el-color-success);
}
.node-heading {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.node-heading strong {
  overflow-wrap: anywhere;
}
.digest {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}
.node-version {
  display: block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.vip-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 20px;
}
.vip-grid .full {
  grid-column: 1/-1;
}
.vip-form {
  margin-top: 20px;
}
.vip-grid .el-select,
.vip-grid .el-input-number {
  width: 100%;
}
@media (max-width: 650px) {
  .vip-grid {
    grid-template-columns: minmax(0, 1fr);
  }
  .entry-card {
    flex-wrap: wrap;
  }
  .topology {
    padding: 12px;
  }
}
</style>
