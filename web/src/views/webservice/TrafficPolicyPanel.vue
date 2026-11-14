<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import { tr } from "@/locales";
import { canPlatform } from "@/utils/platformAccess";
import {
  webTrafficPolicy,
  webPreviewTrafficPolicy,
  webSaveTrafficPolicy,
  type TrafficPolicy,
  type TrafficPolicyInput,
  type TrafficPolicyPreview,
  type TrafficPolicySnapshot,
  type TrafficPolicyTarget,
  type TrafficUpstreamMember
} from "@/api/webTraffic";
const props = defineProps<{ clusterId?: number; instanceId?: number }>();
const router = useRouter();
const snapshot = ref<TrafficPolicySnapshot>(),
  loading = ref(false),
  error = ref(""),
  search = ref(""),
  page = ref(1),
  size = ref(10);
const editor = ref(false),
  details = ref(false),
  selected = ref<TrafficPolicyTarget>(),
  preview = ref<TrafficPolicyPreview>(),
  busy = ref(false),
  saving = ref(false),
  confirming = ref(false),
  formError = ref(""),
  allowText = ref(""),
  denyText = ref(""),
  useUpstream = ref(false),
  routeKeys = ref<string[]>([]),
  routeSchemes = ref<Record<string, "http" | "https">>({}),
  draftName = ref("");
const canWrite = computed(() => canPlatform("web:write"));
const defaultPolicy = (): TrafficPolicy => ({
  enabled: true,
  trafficLog: false,
  allow: [],
  deny: [],
  requestsPerSecond: 0,
  burst: 0,
  nodelay: false,
  maxConnections: 0,
  responseBytesPerSecond: 0,
  routes: []
});
const newMember = (): TrafficUpstreamMember => ({
  address: "",
  weight: 1,
  backup: false,
  down: false,
  maxFails: 1,
  failTimeout: 10
});
const form = ref<TrafficPolicy>(defaultPolicy());
const upstream = ref<{
  strategy: "round_robin" | "least_conn" | "ip_hash";
  members: TrafficUpstreamMember[];
}>({ strategy: "round_robin", members: [newMember()] });
let generation = 0,
  editGeneration = 0,
  abort: AbortController | undefined,
  editAbort: AbortController | undefined,
  validatedInput: TrafficPolicyInput | undefined;
const filteredTargets = computed(() =>
  (snapshot.value?.targets || []).filter(t =>
    `${t.path} ${t.serverNames.join(" ")} ${t.listens.join(" ")}`
      .toLowerCase()
      .includes(search.value.toLowerCase())
  )
);
const rows = computed(() =>
  filteredTargets.value.slice(
    (page.value - 1) * size.value,
    page.value * size.value
  )
);
const locationKey = (l: { path: string; modifier: string }) =>
  JSON.stringify([l.modifier, l.path]);
const routeOptions = computed(() =>
  (selected.value?.locations || []).filter(
    l => !!l.proxyPass && ["", "=", "^~"].includes(l.modifier)
  )
);
function targetName(t?: TrafficPolicyTarget) {
  return t?.serverNames.join(" · ") || tr("未命名站点");
}
function policyState(t: TrafficPolicyTarget) {
  return t.policy?.enabled ? tr("已配置") : tr("未配置");
}
async function load() {
  const current = ++generation;
  abort?.abort();
  snapshot.value = undefined;
  error.value = "";
  page.value = 1;
  if (!props.clusterId) {
    loading.value = false;
    return;
  }
  loading.value = true;
  abort = new AbortController();
  try {
    const value = await webTrafficPolicy(
      props.clusterId,
      props.instanceId,
      abort.signal
    );
    if (current === generation) snapshot.value = value;
  } catch (e) {
    if (current === generation) error.value = (e as Error).message;
  } finally {
    if (current === generation) loading.value = false;
  }
}
function edit(target: TrafficPolicyTarget, disable = false) {
  selected.value = target;
  preview.value = undefined;
  validatedInput = undefined;
  formError.value = "";
  editGeneration++;
  const existing = JSON.parse(JSON.stringify(target.policy || {}));
  form.value = {
    ...defaultPolicy(),
    ...existing,
    allow: Array.isArray(existing.allow) ? existing.allow : [],
    deny: Array.isArray(existing.deny) ? existing.deny : [],
    routes: Array.isArray(existing.routes) ? existing.routes : []
  };
  if (disable) form.value.enabled = false;
  allowText.value = form.value.allow.join("\n");
  denyText.value = form.value.deny.join("\n");
  useUpstream.value = !!form.value.upstream;
  upstream.value = form.value.upstream
    ? {
        strategy: form.value.upstream.strategy,
        members: JSON.parse(JSON.stringify(form.value.upstream.members || []))
      }
    : { strategy: "round_robin", members: [newMember()] };
  routeKeys.value = form.value.routes.map(locationKey);
  routeSchemes.value = Object.fromEntries(
    form.value.routes.map(route => [locationKey(route), route.scheme])
  );
  draftName.value = `${tr("流量策略")} · ${targetName(target)}`;
  editor.value = true;
}
function closeEditor() {
  if (saving.value) return;
  editor.value = false;
  editGeneration++;
  editAbort?.abort();
  busy.value = false;
  preview.value = undefined;
  validatedInput = undefined;
}
function input(): TrafficPolicyInput {
  if (!snapshot.value || !selected.value)
    throw new Error(tr("请重新读取站点配置"));
  const list = (v: string) =>
    v
      .split(/[\s,，]+/)
      .map(x => x.trim())
      .filter(Boolean);
  const policy: TrafficPolicy = {
    ...form.value,
    allow: list(allowText.value),
    deny: list(denyText.value),
    upstream: useUpstream.value
      ? JSON.parse(JSON.stringify(upstream.value))
      : undefined,
    routes: useUpstream.value
      ? routeOptions.value
          .filter(l => routeKeys.value.includes(locationKey(l)))
          .map(l => ({
            path: l.path,
            modifier: l.modifier,
            scheme: routeSchemes.value[locationKey(l)] || "http",
            upstream: ""
          }))
      : []
  };
  if (!policy.requestsPerSecond) {
    policy.burst = 0;
    policy.nodelay = false;
  } else if (!policy.burst) policy.nodelay = false;
  if (policy.enabled && useUpstream.value && !policy.routes.length)
    throw new Error(tr("请选择需要分配流量的现有代理路径"));
  return {
    name: draftName.value.trim(),
    instanceIds: [...snapshot.value.instanceIds],
    path: selected.value.path,
    serverIndex: selected.value.serverIndex,
    baselines: { ...snapshot.value.baselines },
    policy
  };
}
async function validate() {
  if (!props.clusterId || busy.value || saving.value) return;
  const current = ++editGeneration;
  editAbort?.abort();
  editAbort = new AbortController();
  busy.value = true;
  formError.value = "";
  preview.value = undefined;
  validatedInput = undefined;
  try {
    const body = input();
    const value = await webPreviewTrafficPolicy(
      props.clusterId,
      body,
      editAbort.signal
    );
    if (current !== editGeneration || !editor.value) return;
    preview.value = value;
    if (value.valid) validatedInput = body;
  } catch (e) {
    if (current === editGeneration) formError.value = (e as Error).message;
  } finally {
    if (current === editGeneration) busy.value = false;
  }
}
async function save() {
  if (
    !props.clusterId ||
    !validatedInput ||
    !preview.value?.valid ||
    busy.value ||
    saving.value ||
    confirming.value
  )
    return;
  const confirmationGeneration = editGeneration;
  confirming.value = true;
  try {
    await ElMessageBox.confirm(
      tr(
        "保存经过校验的配置草稿，随后前往配置管理确认发布。线上配置将在发布成功后生效。"
      ),
      tr("保存流量策略草稿"),
      {
        type: "info",
        confirmButtonText: tr("保存草稿"),
        cancelButtonText: tr("取消"),
        closeOnClickModal: false,
        lockScroll: false
      }
    );
  } catch {
    return;
  } finally {
    confirming.value = false;
  }
  if (
    !editor.value ||
    !validatedInput ||
    confirmationGeneration !== editGeneration
  )
    return;
  const current = ++editGeneration;
  const body = validatedInput;
  const clusterId = props.clusterId;
  saving.value = true;
  formError.value = "";
  try {
    const value = await webSaveTrafficPolicy(clusterId, body);
    if (current !== editGeneration) return;
    editor.value = false;
    ElMessage.success(tr("策略草稿已保存，请确认发布"));
    router.push({
      path: "/web/configurations",
      query: { clusterId, configurationId: value.draft.id }
    });
  } catch (e) {
    if (current === editGeneration) formError.value = (e as Error).message;
  } finally {
    if (current === editGeneration) saving.value = false;
  }
}
async function removeMember(index: number) {
  if (upstream.value.members[index].address) {
    try {
      await ElMessageBox.confirm(
        tr("从当前草稿移除此上游成员？"),
        tr("移除上游成员"),
        {
          type: "warning",
          confirmButtonText: tr("移除"),
          cancelButtonText: tr("取消"),
          lockScroll: false
        }
      );
    } catch {
      return;
    }
  }
  upstream.value.members.splice(index, 1);
}
watch(
  [
    form,
    allowText,
    denyText,
    useUpstream,
    upstream,
    routeKeys,
    routeSchemes,
    draftName
  ],
  () => {
    preview.value = undefined;
    validatedInput = undefined;
  },
  { deep: true }
);
watch(
  routeKeys,
  keys => {
    for (const key of keys) {
      if (!routeSchemes.value[key]) {
        const location = routeOptions.value.find(
          item => locationKey(item) === key
        );
        routeSchemes.value[key] = location?.proxyPass.startsWith("https://")
          ? "https"
          : "http";
      }
    }
  },
  { deep: true }
);
watch(search, () => {
  page.value = 1;
});
watch(
  () => upstream.value.strategy,
  strategy => {
    if (strategy === "ip_hash")
      upstream.value.members.forEach(member => {
        member.backup = false;
      });
  }
);
watch(
  () => [props.clusterId, props.instanceId],
  () => {
    editGeneration++;
    editAbort?.abort();
    editor.value = false;
    busy.value = saving.value = false;
    preview.value = undefined;
    validatedInput = undefined;
    details.value = false;
    selected.value = undefined;
    load();
  },
  { immediate: true }
);
onBeforeUnmount(() => {
  generation++;
  editGeneration++;
  abort?.abort();
  editAbort?.abort();
});
</script>
<template>
  <section class="traffic-policy-panel">
    <el-alert
      :title="
        $tr(
          '策略按选定站点生效：先读取原生配置、预览差异并执行 nginx -t，再保存草稿并通过配置发布流程应用。'
        )
      "
      type="info"
      :closable="false"
    />
    <el-empty
      v-if="!clusterId"
      :description="$tr('请先在上方选择集群或实例，再管理防御与流量分配')"
    />
    <template v-else>
      <div class="policy-toolbar">
        <el-input
          v-model="search"
          clearable
          :placeholder="$tr('搜索域名、监听或配置路径')"
        /><span class="muted">{{
          $tr(instanceId ? "当前实例范围" : "整个集群范围")
        }}</span
        ><el-button :loading="loading" @click="load">{{
          $tr("重新读取配置")
        }}</el-button>
      </div>
      <el-alert v-if="error" :title="error" type="error" :closable="false" />
      <el-alert
        v-for="(warning, i) in snapshot?.warnings || []"
        :key="i"
        :title="$tr(warning)"
        type="warning"
        :closable="false"
      />
      <el-table
        :data="rows"
        v-loading="loading"
        class="asset-table"
        :row-key="r => `${r.path}:${r.serverIndex}`"
        ><el-table-column :label="$tr('站点 / 监听')" min-width="230"
          ><template #default="{ row }"
            ><strong>{{ targetName(row) }}</strong
            ><small class="block muted">{{
              row.listens.join(" · ")
            }}</small></template
          ></el-table-column
        ><el-table-column
          :label="$tr('配置文件')"
          min-width="260"
          show-overflow-tooltip
          ><template #default="{ row }"
            >{{ row.path }} · server {{ row.serverIndex + 1 }}</template
          ></el-table-column
        ><el-table-column :label="$tr('流量策略')" width="120"
          ><template #default="{ row }"
            ><el-tag :type="row.policy?.enabled ? 'success' : 'info'">{{
              policyState(row)
            }}</el-tag></template
          ></el-table-column
        ><el-table-column :label="$tr('操作')" width="220"
          ><template #default="{ row }"
            ><div class="policy-row-actions">
              <el-button
                link
                type="primary"
                @click="
                  selected = row;
                  details = true;
                "
                >{{ $tr("详情") }}</el-button
              ><el-button
                link
                type="primary"
                :disabled="!canWrite"
                @click="edit(row)"
                >{{ $tr("编辑") }}</el-button
              ><el-dropdown trigger="click" :persistent="false"
                ><el-button link type="primary">{{ $tr("更多") }} ▾</el-button
                ><template #dropdown
                  ><el-dropdown-menu
                    ><el-dropdown-item
                      :disabled="!canWrite || !row.policy?.enabled"
                      @click="edit(row, true)"
                      >{{ $tr("停用策略并生成草稿") }}</el-dropdown-item
                    ><el-dropdown-item
                      @click="
                        router.push({
                          path: '/web/configurations',
                          query: { clusterId, instanceId }
                        })
                      "
                      >{{ $tr("查看原生配置") }}</el-dropdown-item
                    ></el-dropdown-menu
                  ></template
                ></el-dropdown
              >
            </div></template
          ></el-table-column
        ><template #empty
          ><el-empty
            :description="
              $tr('未发现可管理的 HTTP 站点，请检查集群成员和配置权限')
            "
            :image-size="70" /></template
      ></el-table>
      <div class="asset-pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="size"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredTargets.length"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="page = 1"
        />
      </div>
    </template>
    <el-dialog
      :model-value="editor"
      :title="$tr('编辑防御与流量分配')"
      width="min(1080px, calc(100vw - 32px))"
      append-to-body
      destroy-on-close
      align-center
      :lock-scroll="false"
      :close-on-click-modal="false"
      :close-on-press-escape="!saving"
      :show-close="!saving"
      class="traffic-policy-dialog"
      @update:model-value="closeEditor"
    >
      <div v-if="selected">
        <div class="policy-target">
          <strong>{{ targetName(selected) }}</strong
          ><small
            >{{ selected.path }} · server {{ selected.serverIndex + 1 }} ·
            {{ $tr("目标实例") }} {{ snapshot?.instanceIds.join(", ") }}</small
          >
        </div>
        <el-alert
          v-if="formError"
          :title="formError"
          type="error"
          :closable="false"
        />
        <el-form
          label-position="top"
          :disabled="busy || saving"
          @submit.prevent="validate"
        >
          <div class="policy-grid">
            <el-form-item :label="$tr('草稿名称')"
              ><el-input v-model="draftName" maxlength="100" /></el-form-item
            ><el-form-item :label="$tr('启用此站点策略')"
              ><el-switch
                v-model="form.enabled"
                :active-text="$tr('启用')"
                :inactive-text="$tr('停用')"
            /></el-form-item>
          </div>
          <el-alert
            v-if="!form.enabled"
            :title="
              $tr(
                '停用将移除平台添加的策略，并恢复原代理目标；仍需校验和发布后生效。'
              )
            "
            type="warning"
            :closable="false"
          />
          <template v-else>
            <h3>{{ $tr("访问与限流") }}</h3>
            <div class="policy-grid">
              <el-form-item :label="$tr('允许的 IP / CIDR')"
                ><el-input
                  v-model="allowText"
                  type="textarea"
                  :rows="4"
                  :placeholder="$tr('每行一个 IP 或网段；留空表示不启用白名单')"
                /><small class="muted">{{
                  $tr("填写白名单后，其余来源将被拒绝；拒绝列表优先。")
                }}</small></el-form-item
              ><el-form-item :label="$tr('拒绝的 IP / CIDR')"
                ><el-input
                  v-model="denyText"
                  type="textarea"
                  :rows="4"
                  :placeholder="$tr('每行一个 IP 或网段')"
              /></el-form-item>
            </div>
            <div class="policy-grid policy-limits">
              <el-form-item :label="$tr('每个来源 IP 的请求速率（req/s）')"
                ><el-input-number
                  v-model="form.requestsPerSecond"
                  :min="0"
                  :max="100000"
                  :precision="0"
                /><small class="muted">{{
                  $tr("0 表示不限制")
                }}</small></el-form-item
              ><el-form-item :label="$tr('允许突发请求数')"
                ><el-input-number
                  v-model="form.burst"
                  :min="0"
                  :max="100000"
                  :precision="0"
                  :disabled="!form.requestsPerSecond" /></el-form-item
              ><el-form-item :label="$tr('每个来源 IP 的并发连接上限')"
                ><el-input-number
                  v-model="form.maxConnections"
                  :min="0"
                  :max="100000"
                  :precision="0"
                /><small class="muted">{{
                  $tr("0 表示不限制")
                }}</small></el-form-item
              ><el-form-item :label="$tr('单请求响应速率（B/s）')"
                ><el-input-number
                  v-model="form.responseBytesPerSecond"
                  :min="0"
                  :max="1073741824"
                  :precision="0"
                /><small class="muted">{{
                  $tr("0 表示不限制；不是整个站点的总带宽")
                }}</small></el-form-item
              >
            </div>
            <div class="policy-checks">
              <el-checkbox
                v-model="form.nodelay"
                :disabled="!form.requestsPerSecond || !form.burst"
                >{{ $tr("允许突发请求立即处理（nodelay）") }}</el-checkbox
              ><el-checkbox v-model="form.trafficLog">{{
                $tr("启用专用流量日志，采集请求字节、耗时、上游与限流结果")
              }}</el-checkbox>
            </div>
            <el-alert
              :title="
                $tr(
                  '按 Nginx 实际看到的客户端 IP 执行策略；经过代理或 CDN 时，请先正确配置可信代理与真实 IP。'
                )
              "
              type="info"
              :closable="false"
            />
            <h3>{{ $tr("负载均衡与上游流向") }}</h3>
            <el-switch
              v-model="useUpstream"
              :active-text="$tr('为现有代理路径配置上游池')"
            />
            <template v-if="useUpstream"
              ><div class="policy-grid upstream-settings">
                <el-form-item :label="$tr('负载均衡方式')"
                  ><el-select v-model="upstream.strategy"
                    ><el-option
                      value="round_robin"
                      :label="$tr('加权轮询')" /><el-option
                      value="least_conn"
                      :label="$tr('最少连接')" /><el-option
                      value="ip_hash"
                      :label="$tr('来源 IP 哈希')" /></el-select></el-form-item
                ><el-form-item :label="$tr('应用到现有代理路径')" class="full"
                  ><el-select
                    v-model="routeKeys"
                    multiple
                    :placeholder="$tr('选择已有 proxy_pass 的路径')"
                    ><el-option
                      v-for="l in routeOptions"
                      :key="locationKey(l)"
                      :value="locationKey(l)"
                      :label="`${l.modifier} ${l.path} → ${l.proxyPass}`" /></el-select
                ></el-form-item>
                <el-form-item
                  v-for="location in routeOptions.filter(item =>
                    routeKeys.includes(locationKey(item))
                  )"
                  :key="locationKey(location)"
                  :label="`${tr('上游协议')} · ${location.modifier} ${location.path}`"
                >
                  <el-select v-model="routeSchemes[locationKey(location)]"
                    ><el-option value="http" label="HTTP" /><el-option
                      value="https"
                      label="HTTPS"
                  /></el-select>
                </el-form-item>
              </div>
              <div class="member-scroll">
                <el-table
                  :data="upstream.members"
                  class="asset-table"
                  size="small"
                  ><el-table-column :label="$tr('后端地址')" min-width="210"
                    ><template #default="{ row }"
                      ><el-input
                        v-model="row.address"
                        placeholder="192.168.1.10:8080"
                        :aria-label="
                          $tr('后端地址')
                        " /></template></el-table-column
                  ><el-table-column :label="$tr('权重')" width="120"
                    ><template #default="{ row }"
                      ><el-input-number
                        v-model="row.weight"
                        :min="1"
                        :max="1000"
                        :precision="0"
                        controls-position="right"
                        :aria-label="$tr('权重')" /></template></el-table-column
                  ><el-table-column :label="$tr('失败次数')" width="120"
                    ><template #default="{ row }"
                      ><el-input-number
                        v-model="row.maxFails"
                        :min="0"
                        :max="100"
                        :precision="0"
                        controls-position="right"
                        :aria-label="
                          $tr('失败次数')
                        " /></template></el-table-column
                  ><el-table-column :label="$tr('失败超时（秒）')" width="140"
                    ><template #default="{ row }"
                      ><el-input-number
                        v-model="row.failTimeout"
                        :min="0"
                        :max="3600"
                        :precision="0"
                        controls-position="right"
                        :aria-label="
                          $tr('失败超时（秒）')
                        " /></template></el-table-column
                  ><el-table-column :label="$tr('备用')" width="85"
                    ><template #default="{ row }"
                      ><el-checkbox
                        v-model="row.backup"
                        :disabled="upstream.strategy === 'ip_hash'"
                        :aria-label="$tr('备用')" /></template></el-table-column
                  ><el-table-column :label="$tr('停用')" width="85"
                    ><template #default="{ row }"
                      ><el-checkbox
                        v-model="row.down"
                        :aria-label="$tr('停用')" /></template></el-table-column
                  ><el-table-column :label="$tr('操作')" width="80"
                    ><template #default="{ $index }"
                      ><el-button
                        link
                        type="danger"
                        @click="removeMember($index)"
                        >{{ $tr("移除") }}</el-button
                      ></template
                    ></el-table-column
                  ></el-table
                >
              </div>
              <el-button
                class="add-member"
                @click="upstream.members.push(newMember())"
                >{{ $tr("添加上游成员") }}</el-button
              ><small class="muted block">{{
                $tr(
                  "权重用于分配请求；备用成员仅在主成员不可用时接收流量，IP 哈希不支持备用成员。"
                )
              }}</small></template
            >
          </template>
        </el-form>
        <section v-if="preview" class="policy-preview">
          <h3>{{ $tr("校验与配置差异") }}</h3>
          <el-alert
            :type="preview.valid ? 'success' : 'error'"
            :title="
              $tr(
                preview.valid
                  ? '目标节点 nginx -t 校验通过'
                  : '配置校验未通过，无法保存'
              )
            "
            :closable="false"
          />
          <div
            v-for="check in preview.checks"
            :key="check.instanceId"
            class="policy-check-result"
          >
            <el-tag :type="check.valid ? 'success' : 'danger'">{{
              $tr(check.valid ? "通过" : "失败")
            }}</el-tag
            ><span>{{ check.name }} · {{ check.message }}</span>
          </div>
          <el-alert
            v-for="(warning, i) in preview.warnings"
            :key="i"
            :title="$tr(warning)"
            type="warning"
            :closable="false"
          /><el-collapse
            ><el-collapse-item
              v-for="file in preview.files"
              :key="file.path"
              :title="file.path"
              :name="file.path"
              ><div class="policy-diff">
                <div>
                  <strong>{{ $tr("当前配置") }}</strong>
                  <pre>{{ file.before || $tr("新文件") }}</pre>
                </div>
                <div>
                  <strong>{{ $tr("候选配置") }}</strong>
                  <pre>{{ file.after || $tr("删除文件") }}</pre>
                </div>
              </div></el-collapse-item
            ></el-collapse
          >
        </section>
      </div>
      <template #footer
        ><div class="policy-footer">
          <el-button :disabled="saving" @click="closeEditor">{{
            $tr("取消")
          }}</el-button
          ><el-button
            :loading="busy"
            :disabled="saving || !canWrite"
            @click="validate"
            >{{ $tr("预览并校验") }}</el-button
          ><el-button
            type="primary"
            :loading="saving || confirming"
            :disabled="!preview?.valid || busy || !canWrite"
            @click="save"
            >{{ $tr("保存草稿并前往发布") }}</el-button
          >
        </div></template
      >
    </el-dialog>
    <el-dialog
      v-model="details"
      :title="$tr('站点流量策略详情')"
      width="min(760px, calc(100vw - 32px))"
      append-to-body
      destroy-on-close
      align-center
      :lock-scroll="false"
      class="traffic-policy-dialog"
      ><template v-if="selected"
        ><el-descriptions :column="1" border
          ><el-descriptions-item :label="$tr('站点')">{{
            targetName(selected)
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('配置文件')"
            >{{ selected.path }} · server
            {{ selected.serverIndex + 1 }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('监听')">{{
            selected.listens.join(" · ")
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('流量策略')">{{
            policyState(selected)
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('请求速率限制')">{{
            selected.policy?.requestsPerSecond || $tr("未限制")
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('连接上限')">{{
            selected.policy?.maxConnections || $tr("未限制")
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('允许的 IP / CIDR')">{{
            selected.policy?.allow?.join(", ") || "—"
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('拒绝的 IP / CIDR')">{{
            selected.policy?.deny?.join(", ") || "—"
          }}</el-descriptions-item></el-descriptions
        ><el-table :data="selected.locations" class="asset-table"
          ><el-table-column
            prop="path"
            :label="$tr('请求路径')" /><el-table-column
            prop="proxyPass"
            :label="$tr('当前代理目标')"
            min-width="220"
            show-overflow-tooltip /></el-table></template
    ></el-dialog>
  </section>
</template>
<style scoped>
.el-alert {
  margin-bottom: 14px;
}
.muted {
  color: var(--text-secondary);
  font-size: 12px;
}
.block {
  display: block;
  margin-top: 6px;
}
.policy-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin: 18px 0;
}
.policy-toolbar .el-input {
  width: min(400px, 100%);
}
.policy-row-actions {
  display: flex;
  align-items: center;
  gap: 14px;
  white-space: nowrap;
}
.policy-row-actions .el-button {
  margin: 0;
}
.asset-pagination {
  display: flex;
  justify-content: center;
  margin-top: 18px;
}
.asset-pagination :deep(.el-pagination) {
  justify-content: center;
  flex-wrap: wrap;
  row-gap: 10px;
}
.policy-target {
  margin-bottom: 20px;
  overflow-wrap: anywhere;
}
.policy-target small {
  display: block;
  color: var(--text-secondary);
  margin-top: 6px;
}
.policy-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 18px;
}
.policy-grid .full {
  grid-column: 1 / -1;
}
.policy-grid .el-input-number,
.policy-grid .el-select {
  width: 100%;
}
.policy-grid small {
  line-height: 1.6;
  margin-top: 6px;
}
h3 {
  margin: 22px 0 16px;
  font-size: 15px;
}
.policy-checks {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  margin-bottom: 14px;
}
.policy-checks :deep(.el-checkbox) {
  height: auto;
  margin-bottom: 12px;
}
.policy-checks :deep(.el-checkbox__label) {
  white-space: normal;
  line-height: 1.6;
}
.upstream-settings {
  margin-top: 18px;
}
.member-scroll {
  min-width: 0;
  overflow: auto;
}
.member-scroll .el-input-number {
  width: 100%;
}
.add-member {
  margin-top: 14px;
}
.policy-check-result {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 10px;
}
.policy-diff {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}
.policy-diff > div {
  min-width: 0;
}
.policy-diff pre {
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-light);
  padding: 12px;
  max-height: 400px;
  overflow: auto;
  font-size: 12px;
  line-height: 1.6;
}
.policy-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  flex-wrap: wrap;
}
.policy-footer .el-button {
  margin: 0;
}
@media (max-width: 700px) {
  .policy-grid,
  .policy-diff {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
<style>
.traffic-policy-dialog.el-dialog {
  display: flex;
  flex-direction: column;
  max-height: calc(100dvh - 32px);
  margin: 16px auto;
}
.traffic-policy-dialog .el-dialog__body {
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
}
.traffic-policy-dialog .el-dialog__header,
.traffic-policy-dialog .el-dialog__footer {
  flex-shrink: 0;
}
</style>
