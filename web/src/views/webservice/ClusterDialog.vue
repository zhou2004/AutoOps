<template>
  <el-dialog
    v-model="visible"
    :title="$tr(record ? '编辑 Web 集群' : '新增 Web 集群')"
    width="min(860px, calc(100vw - 32px))"
    append-to-body
    destroy-on-close
    :lock-scroll="false"
    class="web-resource-dialog cluster-edit-dialog"
    :close-on-click-modal="!saving"
    :close-on-press-escape="!saving"
    :show-close="!saving"
  >
    <el-form
      v-loading="loading"
      label-position="top"
      :disabled="loading || saving"
      @submit.prevent
    >
      <div class="cluster-form-grid">
        <el-form-item :label="$tr('名称')" required
          ><el-input v-model="form.name" maxlength="100"
        /></el-form-item>
        <el-form-item :label="$tr('业务分组')" required>
          <el-select
            v-model="form.groupId"
            filterable
            :disabled="!!record"
            @change="selectedIds = []"
          >
            <el-option
              v-for="group in groups"
              :key="group.id"
              :value="group.id"
              :label="group.name"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="$tr('环境')"
          ><EnvironmentSelect v-model="form.environment"
        /></el-form-item>
      </div>
      <el-form-item :label="$tr('从数据源选择 Nginx 实例')">
        <el-select
          v-model="selectedIds"
          multiple
          filterable
          collapse-tags
          collapse-tags-tooltip
          :max-collapse-tags="2"
          :filter-method="query => (instanceSearch = query)"
          @visible-change="
            open => {
              if (!open) instanceSearch = '';
            }
          "
          :disabled="!form.groupId || !canExecute || !!loadError"
          :placeholder="$tr('搜索数据源、主机地址或实例名称')"
        >
          <el-option-group
            v-for="source in sourceGroups"
            :key="source.id"
            :label="source.name"
          >
            <el-option
              v-for="instance in source.instances"
              :key="instance.id"
              :value="instance.id"
              :label="instance.name"
              :disabled="!!disabledReason(instance)"
            >
              <div class="instance-option">
                <WebResourceIcon kind="instances" />
                <span class="instance-option-name">{{ instance.name }}</span>
                <small>{{
                  disabledReason(instance) ||
                  instance.connection?.address ||
                  $tr(instance.spec.runtime === "docker" ? "容器" : "主机")
                }}</small>
              </div>
            </el-option>
          </el-option-group>
        </el-select>
      </el-form-item>
      <p class="cluster-hint">
        {{
          $tr(
            "仅显示同业务分组中已发现的 Nginx 实例；未发现服务时，请先到数据源管理执行发现。"
          )
        }}
      </p>
      <p v-if="!canExecute" class="cluster-hint">
        {{ $tr("调整成员需要 Web 执行权限；仍可编辑集群基本信息。") }}
      </p>
      <p v-if="vipIds.length" class="cluster-hint">
        {{
          $tr(
            "高可用成员已锁定，不能通过编辑集群移出。新增成员不会自动加入 VIP 高可用组。"
          )
        }}
      </p>
      <div v-if="selectedInstances.length" class="selected-instances">
        <div
          v-for="instance in selectedInstances"
          :key="instance.id"
          class="selected-instance"
        >
          <WebResourceIcon kind="instances" />
          <div class="instance-summary">
            <strong>{{ instance.name }}</strong>
            <small
              >{{ sourceName(instance) }} ·
              {{ instance.connection?.address || "—" }} ·
              {{
                $tr(instance.spec.runtime === "docker" ? "容器" : "主机")
              }}</small
            >
            <small
              >{{
                instance.spec.container ? instance.spec.container + " · " : ""
              }}{{ instance.spec.configPath }}</small
            >
          </div>
          <el-tag v-if="vipIds.includes(instance.id)" size="small">{{
            $tr("高可用成员")
          }}</el-tag>
          <el-tag
            v-else
            size="small"
            :type="instance.management === 'operate' ? 'success' : 'info'"
            >{{
              $tr(instance.management === "operate" ? "已接管" : "观察模式")
            }}</el-tag
          >
        </div>
      </div>
      <p class="cluster-hint">
        {{
          $tr(
            "保存同时更新集群信息和所选成员，不会安装、重载或卸载 Nginx。取消勾选只移出管理关系，保留远端服务和文件。"
          )
        }}
      </p>
      <el-divider content-position="left">{{ $tr("业务入口探测") }}</el-divider>
      <div
        v-for="(entry, index) in form.spec.entries"
        :key="index"
        class="entry-row"
      >
        <el-input v-model="entry.name" :placeholder="$tr('名称')" />
        <el-input v-model="entry.url" placeholder="HTTP / HTTPS URL" />
        <el-input v-model="entry.host" placeholder="Host" />
        <el-input-number
          v-model="entry.expectedStatus"
          :min="100"
          :max="599"
          :aria-label="$tr('预期状态码')"
        />
        <el-button
          link
          type="danger"
          @click="form.spec.entries.splice(index, 1)"
          >{{ $tr("移除") }}</el-button
        >
      </div>
      <el-button
        @click="
          form.spec.entries.push({
            name: '',
            url: '',
            host: '',
            expectedStatus: 200
          })
        "
        >{{ $tr("添加入口") }}</el-button
      >
    </el-form>
    <el-alert
      v-if="loadError || error"
      :title="loadError || error"
      type="error"
      show-icon
      :closable="false"
    />
    <template #footer>
      <el-button v-if="loadError" :disabled="loading || saving" @click="load">{{
        $tr("重新加载")
      }}</el-button>
      <el-button :disabled="saving" @click="visible = false">{{
        $tr("取消")
      }}</el-button>
      <el-button
        type="primary"
        :loading="saving"
        :disabled="loading || !!loadError || !form.name.trim() || !form.groupId"
        @click="save"
        >{{ $tr("保存") }}</el-button
      >
    </template>
  </el-dialog>
</template>
<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { tr } from "@/locales";
import {
  webOptions,
  webRequest,
  webSaveCluster,
  type WebResource
} from "@/api/webservice";
import EnvironmentSelect from "./EnvironmentSelect.vue";
import WebResourceIcon from "./WebResourceIcon.vue";
const props = defineProps<{
  modelValue: boolean;
  record?: WebResource;
  groups: { id: number; name: string }[];
  initialGroupId?: number;
  initialEnvironment?: string;
  canExecute: boolean;
}>();
const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  saved: [value: WebResource];
}>();
const visible = computed({
  get: () => props.modelValue,
  set: value => emit("update:modelValue", value)
});
function fresh() {
  return {
    name: "",
    groupId: undefined as number | undefined,
    environment: "",
    engine: "nginx",
    revision: 0,
    spec: { entries: [] }
  } as Partial<WebResource> & {
    name: string;
    environment: string;
    spec: Record<string, any>;
  };
}
const form = ref(fresh()),
  instances = ref<WebResource[]>([]),
  sources = ref<WebResource[]>([]),
  selectedIds = ref<number[]>([]);
const instanceSearch = ref("");
const initialIds = ref<number[]>([]),
  loading = ref(false),
  saving = ref(false),
  loadError = ref(""),
  error = ref("");
let generation = 0,
  abort: AbortController | undefined;
const vipIds = computed<number[]>(() => form.value.spec.vip?.instanceIds || []);
const selectedInstances = computed(() =>
  instances.value.filter(instance => selectedIds.value.includes(instance.id))
);
function sourceName(instance: WebResource) {
  return (
    sources.value.find(source => source.id === instance.sourceId)?.name ||
    tr("数据源 #{value0}", { value0: instance.sourceId })
  );
}
function optionLabel(instance: WebResource) {
  return [
    sourceName(instance),
    instance.name,
    instance.connection?.address,
    instance.spec.container,
    instance.spec.configPath
  ]
    .filter(Boolean)
    .join(" · ");
}
function disabledReason(instance: WebResource) {
  if (instance.clusterId && instance.clusterId !== form.value.id)
    return tr("已加入其他集群");
  if (vipIds.value.includes(instance.id)) return tr("高可用成员");
  return "";
}
const sourceGroups = computed(() => {
  const groups = new Map<
    number,
    { id: number; name: string; instances: WebResource[] }
  >();
  for (const instance of instances.value) {
    if (
      instance.groupId !== form.value.groupId ||
      instance.engine !== "nginx" ||
      !optionLabel(instance)
        .toLowerCase()
        .includes(instanceSearch.value.trim().toLowerCase())
    )
      continue;
    if (!groups.has(instance.sourceId))
      groups.set(instance.sourceId, {
        id: instance.sourceId,
        name: sourceName(instance),
        instances: []
      });
    groups.get(instance.sourceId)!.instances.push(instance);
  }
  return [...groups.values()];
});
async function load() {
  const gen = ++generation;
  abort?.abort();
  abort = new AbortController();
  saving.value = false;
  loading.value = true;
  loadError.value = "";
  error.value = "";
  instanceSearch.value = "";
  instances.value = [];
  sources.value = [];
  initialIds.value = [];
  selectedIds.value = [];
  form.value = fresh();
  form.value.groupId = props.record?.groupId || props.initialGroupId;
  form.value.environment =
    props.record?.environment || props.initialEnvironment || "";
  try {
    const [record, nodes, connected] = await Promise.all([
      props.record
        ? webRequest<WebResource>(
            "/clusters/" + props.record.id,
            "GET",
            undefined,
            abort.signal
          )
        : Promise.resolve(undefined),
      webOptions("instances", abort.signal),
      webOptions("sources", abort.signal)
    ]);
    if (gen !== generation || !props.modelValue) return;
    if (record)
      form.value = {
        ...record,
        spec: { ...record.spec, entries: record.spec.entries || [] }
      };
    instances.value = nodes;
    sources.value = connected;
    initialIds.value = nodes
      .filter(instance => record && instance.clusterId === record.id)
      .map(instance => instance.id);
    selectedIds.value = [...initialIds.value];
  } catch (e) {
    if (gen === generation && props.modelValue)
      loadError.value = (e as Error).message;
  } finally {
    if (gen === generation) loading.value = false;
  }
}
async function save() {
  if (
    loading.value ||
    saving.value ||
    loadError.value ||
    !form.value.groupId ||
    !form.value.name.trim()
  )
    return;
  error.value = "";
  const add = selectedIds.value.filter(id => !initialIds.value.includes(id));
  const remove = initialIds.value.filter(id => !selectedIds.value.includes(id));
  saving.value = true;
  const gen = generation;
  try {
    if (remove.length)
      await ElMessageBox.confirm(
        tr(
          "将从集群移出 {value0} 个实例，远端 Nginx 和文件会保留。确认保存？",
          { value0: remove.length }
        ),
        tr("确认成员变更"),
        { type: "warning" }
      );
    if (gen !== generation || !props.modelValue) return;
    const result = await webSaveCluster(
      { ...form.value, name: form.value.name.trim() },
      { add, remove }
    );
    if (gen !== generation || !props.modelValue) return;
    ElMessage.success(tr("集群信息和成员已保存"));
    emit("saved", result);
    visible.value = false;
  } catch (e) {
    if (gen === generation && e !== "cancel" && e !== "close")
      error.value = (e as Error).message;
  } finally {
    if (gen === generation) saving.value = false;
  }
}
watch(
  () => props.modelValue,
  open => {
    if (open) load();
    else {
      generation++;
      abort?.abort();
    }
  }
);
onUnmounted(() => {
  generation++;
  abort?.abort();
});
</script>
<style scoped>
.cluster-form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 18px;
}
.el-select {
  width: 100%;
}
.cluster-hint {
  margin: 8px 0 14px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
.instance-option {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.instance-option-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
}
.instance-option small {
  color: var(--el-text-color-secondary);
}
.selected-instances {
  max-height: 240px;
  overflow: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}
.selected-instance {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
}
.selected-instance + .selected-instance {
  border-top: 1px solid var(--el-border-color-lighter);
}
.instance-summary {
  flex: 1;
  min-width: 0;
}
.instance-summary strong,
.instance-summary small {
  display: block;
  overflow-wrap: anywhere;
}
.instance-summary small {
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}
.entry-row {
  display: grid;
  grid-template-columns: 1fr 2fr 1fr 130px 40px;
  gap: 8px;
  margin-bottom: 10px;
}
.el-alert {
  margin-top: 14px;
}
@media (max-width: 640px) {
  .cluster-form-grid {
    grid-template-columns: 1fr;
  }
  .entry-row {
    grid-template-columns: 1fr;
  }
  .instance-option {
    gap: 4px;
  }
  .selected-instance {
    flex-wrap: wrap;
  }
}
</style>
