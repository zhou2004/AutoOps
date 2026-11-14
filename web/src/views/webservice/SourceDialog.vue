<template>
  <el-dialog
    :model-value="modelValue"
    :title="$tr(record ? '编辑数据源' : '接入 Nginx 服务')"
    class="web-resource-dialog"
    width="min(820px, calc(100vw - 32px))"
    append-to-body
    :lock-scroll="false"
    destroy-on-close
    :close-on-click-modal="false"
    :close-on-press-escape="!saving"
    :show-close="!saving"
    @update:model-value="close"
  >
    <el-form label-position="top" :disabled="saving" @submit.prevent>
      <div class="source-grid">
        <el-form-item :label="$tr('主机')" required
          ><el-select
            v-model="form.spec.hostId"
            filterable
            :disabled="bound || saving"
            @change="changeHost"
            ><el-option
              v-for="h in hosts"
              :key="h.id"
              :label="hostLabel(h)"
              :value="h.id" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('SSH 连接')" required
          ><el-select
            v-model="form.spec.connectionId"
            filterable
            :disabled="bound || saving"
            @change="invalidateDiscovery"
            ><el-option
              v-for="c in hostConnections"
              :key="c.id"
              :label="
                c.name ||
                c.alias ||
                `${c.username || 'SSH'} · ${c.address || ''}:${c.port || 22}`
              "
              :value="c.id" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('名称')" required
          ><el-input v-model="form.name" maxlength="100"
        /></el-form-item>
        <el-form-item :label="$tr('业务分组')" required
          ><el-select v-model="form.groupId" :disabled="!!form.id" filterable
            ><el-option
              v-for="g in groups"
              :key="g.id"
              :label="g.name"
              :value="g.id" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('运行环境')"
          ><el-radio-group
            v-model="form.spec.runtime"
            :disabled="bound || saving"
            @change="invalidateDiscovery"
            ><el-radio-button value="host">{{ $tr("主机") }}</el-radio-button
            ><el-radio-button value="docker"
              >Docker</el-radio-button
            ></el-radio-group
          ></el-form-item
        >
        <el-form-item :label="$tr('环境')"
          ><EnvironmentSelect v-model="form.environment"
        /></el-form-item>
        <el-form-item
          v-if="form.spec.runtime === 'docker'"
          :label="$tr('容器名称')"
          ><el-input
            v-model="form.spec.container"
            :disabled="bound || saving"
            :placeholder="$tr('留空识别运行中的 Nginx 容器')"
            @change="invalidateDiscovery"
        /></el-form-item>
      </div>
      <el-alert
        v-if="bound"
        :title="$tr('已有实例的数据源保留服务定位；可调整名称和健康检查。')"
        type="info"
        :closable="false"
      />
      <template v-else>
        <div class="source-discovery">
          <el-button
            type="primary"
            :loading="discovering"
            :disabled="!form.spec.hostId || !form.spec.connectionId || saving"
            @click="discover"
            >{{ $tr("自动识别 Nginx") }}</el-button
          ><el-checkbox
            v-model="form.spec.sudo"
            :disabled="saving"
            @change="invalidateDiscovery"
            >{{ $tr("使用 sudo") }}</el-checkbox
          >
        </div>
        <p class="muted">
          {{
            $tr(
              "选择主机和连接后识别运行中的服务，再选择要接入的实例。识别不会修改远端配置。"
            )
          }}
        </p>
        <el-radio-group
          v-if="candidates.length"
          v-model="candidateId"
          class="candidate-list"
          @change="selectCandidate"
        >
          <el-radio
            v-for="candidate in candidates"
            :key="candidate.id"
            :value="candidate.id"
            border
          >
            <span class="candidate-name"
              ><WebResourceIcon />{{ candidate.name }}
              <small>{{ candidate.version?.split("\n")[0] }}</small></span
            >
            <code>{{ candidate.spec.configPath }}</code>
            <small>{{
              candidate.spec.runtime === "docker"
                ? candidate.spec.container
                : candidate.spec.service
            }}</small>
          </el-radio>
        </el-radio-group>
        <el-alert
          v-for="(warning, i) in warnings"
          :key="i"
          :title="warning"
          type="warning"
          :closable="false"
        />
      </template>
      <el-collapse v-model="expanded" class="source-advanced"
        ><el-collapse-item
          name="advanced"
          :title="$tr('高级设置（自动识别后填充）')"
        >
          <div class="source-grid">
            <el-form-item :label="$tr('Nginx 程序路径')" required
              ><el-input v-model="form.spec.binary" :disabled="bound"
            /></el-form-item>
            <el-form-item :label="$tr('主配置入口')" required
              ><el-input v-model="form.spec.configPath" :disabled="bound"
            /></el-form-item>
            <el-form-item :label="$tr('运行前缀')" required
              ><el-input v-model="form.spec.prefix" :disabled="bound"
            /></el-form-item>
            <el-form-item
              v-if="form.spec.runtime === 'host'"
              :label="$tr('服务名称')"
              ><el-input v-model="form.spec.service" :disabled="bound"
            /></el-form-item>
            <el-form-item :label="$tr('受管目录')"
              ><el-input
                :model-value="(form.spec.managedPaths || []).join('\n')"
                type="textarea"
                :disabled="bound"
                @update:model-value="
                  v =>
                    (form.spec.managedPaths = v
                      .split('\n')
                      .map(x => x.trim())
                      .filter(Boolean))
                "
            /></el-form-item>
            <el-form-item :label="$tr('健康检查地址')"
              ><el-input
                v-model="form.spec.healthUrl"
                placeholder="http://127.0.0.1:80/"
            /></el-form-item>
            <el-form-item :label="$tr('健康检查 Host')"
              ><el-input v-model="form.spec.healthHost"
            /></el-form-item>
            <el-form-item :label="$tr('预期状态码')"
              ><el-input-number
                v-model="form.spec.expectedStatus"
                :min="100"
                :max="599"
            /></el-form-item>
            <el-form-item :label="$tr('stub_status 地址')"
              ><el-input v-model="form.spec.statusUrl"
            /></el-form-item>
          </div> </el-collapse-item
      ></el-collapse>
      <el-checkbox v-if="!bound" v-model="discoverOnSave">{{
        $tr("保存后发现实例（观察模式）")
      }}</el-checkbox>
    </el-form>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      show-icon
    />
    <template #footer
      ><el-button :disabled="saving" @click="close(false)">{{
        $tr("取消")
      }}</el-button
      ><el-button
        type="primary"
        :loading="saving"
        :disabled="!canSave || discovering"
        @click="save"
        >{{ $tr("保存") }}</el-button
      ></template
    >
  </el-dialog>
</template>
<script setup lang="ts">
import EnvironmentSelect from "./EnvironmentSelect.vue";
import { computed, ref, watch, onUnmounted } from "vue";
import { ElMessage } from "element-plus";
import { webRequest, type WebResource } from "@/api/webservice";
import { tr } from "@/locales";
import WebResourceIcon from "./WebResourceIcon.vue";
const props = defineProps<{
  modelValue: boolean;
  record?: WebResource;
  groups: any[];
  hosts: any[];
  connections: any[];
  instances: WebResource[];
  initialGroupId?: number;
  environment?: string;
}>();
const emit = defineEmits<{
  "update:modelValue": [boolean];
  saved: [WebResource];
}>();
interface Candidate {
  id: string;
  name: string;
  spec: Record<string, any>;
  version: string;
  warnings?: string[];
}
const form = ref<any>({ spec: {} }),
  candidates = ref<Candidate[]>([]),
  candidateId = ref(""),
  warnings = ref<string[]>([]),
  error = ref(""),
  discovering = ref(false),
  saving = ref(false),
  discoverOnSave = ref(true),
  expanded = ref<string[]>([]);
let generation = 0;
let abort: AbortController | undefined;
const bound = computed(
  () =>
    !!form.value.id && props.instances.some(i => i.sourceId === form.value.id)
);
const hostConnections = computed(() =>
  props.connections.filter(c => c.assetId === form.value.spec.hostId)
);
const canSave = computed(
  () =>
    !!form.value.name?.trim() &&
    !!form.value.groupId &&
    !!form.value.spec.hostId &&
    !!form.value.spec.connectionId &&
    ["binary", "configPath", "prefix"].every(
      k => !!form.value.spec[k]?.trim()
    ) &&
    (form.value.spec.runtime !== "docker" ||
      !!form.value.spec.container?.trim())
);
function hostLabel(h: any) {
  return `${h.name || h.hostName || h.id} · ${h.privateIp || ""}`;
}
function invalidateDiscovery() {
  generation++;
  abort?.abort();
  candidates.value = [];
  candidateId.value = "";
  warnings.value = [];
  error.value = "";
  discovering.value = false;
  if (!bound.value) {
    for (const k of ["binary", "configPath", "prefix", "service"])
      form.value.spec[k] = "";
    form.value.spec.managedPaths = [];
  }
}
function changeHost() {
  invalidateDiscovery();
  form.value.spec.connectionId =
    hostConnections.value.length === 1
      ? hostConnections.value[0].id
      : undefined;
  const h = props.hosts.find(h => h.id === form.value.spec.hostId);
  if (h) {
    if (!form.value.groupId) form.value.groupId = h.groupId;
    if (!form.value.name)
      form.value.name = `Nginx · ${h.privateIp || h.name || h.hostName}`;
  }
}
function selectCandidate() {
  const candidate = candidates.value.find(c => c.id === candidateId.value);
  if (!candidate) return;
  form.value.spec = { ...form.value.spec, ...candidate.spec };
  warnings.value = candidate.warnings || [];
}
async function discover() {
  if (saving.value || discovering.value) return;
  invalidateDiscovery();
  const gen = generation;
  abort = new AbortController();
  discovering.value = true;
  error.value = "";
  candidates.value = [];
  candidateId.value = "";
  try {
    const value = await webRequest<{
      candidates: Candidate[];
      warnings: string[];
    }>(
      "/source-discovery",
      "POST",
      {
        hostId: form.value.spec.hostId,
        connectionId: form.value.spec.connectionId,
        runtime: form.value.spec.runtime,
        sudo: !!form.value.spec.sudo,
        container: form.value.spec.container || ""
      },
      abort.signal
    );
    if (gen !== generation || !props.modelValue) return;
    candidates.value = value.candidates || [];
    warnings.value = value.warnings || [];
    if (!candidates.value.length)
      error.value = tr("未识别到运行中的 Nginx，可检查连接或使用高级设置。");
    if (candidates.value.length === 1) {
      candidateId.value = candidates.value[0].id;
      selectCandidate();
    }
  } catch (e) {
    if (gen === generation) error.value = (e as Error).message;
  } finally {
    if (gen === generation) discovering.value = false;
  }
}
async function save() {
  if (saving.value || discovering.value || !canSave.value || !props.modelValue)
    return;
  const gen = ++generation;
  saving.value = true;
  error.value = "";
  try {
    const payload = JSON.parse(JSON.stringify(form.value));
    const saved = await webRequest<WebResource>(
      "/sources" + (payload.id ? "/" + payload.id : ""),
      payload.id ? "PUT" : "POST",
      payload
    );
    if (gen !== generation || !props.modelValue) return;
    form.value = JSON.parse(JSON.stringify(saved));
    if (discoverOnSave.value && !bound.value) {
      try {
        await webRequest("/sources/" + saved.id + "/discovery", "POST");
      } catch (e) {
        if (gen !== generation || !props.modelValue) return;
        error.value = tr("数据源已保存，实例发现失败：") + (e as Error).message;
        emit("saved", saved);
        return;
      }
    }
    if (gen !== generation || !props.modelValue) return;
    emit("saved", saved);
    emit("update:modelValue", false);
    ElMessage.success(tr("保存成功"));
  } catch (e) {
    if (gen === generation && props.modelValue)
      error.value = (e as Error).message;
  } finally {
    if (gen === generation) saving.value = false;
  }
}
function close(value: boolean) {
  if (!saving.value) emit("update:modelValue", value);
}
watch(
  () => [props.modelValue, props.record?.id] as const,
  ([open]) => {
    generation++;
    abort?.abort();
    error.value = "";
    discovering.value = false;
    saving.value = false;
    if (!open) return;
    form.value = props.record
      ? JSON.parse(JSON.stringify(props.record))
      : {
          name: "",
          groupId: props.initialGroupId,
          environment: props.environment || "",
          engine: "nginx",
          spec: { runtime: "host", expectedStatus: 200, managedPaths: [] }
        };
    candidates.value = [];
    candidateId.value = "";
    warnings.value = [];
    expanded.value = [];
    discoverOnSave.value = !props.record;
  },
  { immediate: true }
);
onUnmounted(() => {
  generation++;
  abort?.abort();
});
</script>
<style scoped>
.source-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 18px;
}
.source-grid .el-select,
.source-grid .el-input-number {
  width: 100%;
}
.source-discovery {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}
.muted {
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}
.candidate-list {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 10px;
  max-height: 280px;
  overflow: auto;
}
.candidate-list :deep(.el-radio) {
  width: 100%;
  height: auto;
  min-height: 82px;
  margin: 0;
  padding: 12px;
}
.candidate-list :deep(.el-radio__label) {
  min-width: 0;
  white-space: normal;
}
.candidate-list code,
.candidate-list small {
  display: block;
  overflow-wrap: anywhere;
  line-height: 1.6;
}
.candidate-name {
  display: flex;
  align-items: center;
  gap: 8px;
}
.source-advanced {
  margin: 20px 0;
}
:deep(.el-dialog__body) {
  max-height: calc(100dvh - 220px);
  overflow: auto;
  overscroll-behavior: contain;
}
@media (max-width: 640px) {
  .source-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
