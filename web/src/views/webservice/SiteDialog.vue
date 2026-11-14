<template>
  <el-dialog
    :model-value="modelValue"
    :title="$tr(record ? '编辑站点' : '新增站点')"
    class="web-site-dialog"
    width="min(960px, calc(100vw - 32px))"
    append-to-body
    destroy-on-close
    align-center
    :lock-scroll="false"
    :close-on-click-modal="!saving"
    :close-on-press-escape="!saving"
    :show-close="!saving"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <el-alert
      :title="$tr('保存仅记录站点设置；生成草稿后仍需校验和确认发布。')"
      type="info"
      :closable="false"
      show-icon
    />
    <el-form label-position="top" :disabled="saving" @submit.prevent>
      <div class="site-grid">
        <el-form-item :label="$tr('站点名称')" required
          ><el-input v-model="form.name" maxlength="100"
        /></el-form-item>
        <el-form-item :label="$tr('集群')" required>
          <el-select
            v-model="form.clusterId"
            filterable
            :disabled="!!record"
            @change="selectCluster"
          >
            <el-option
              v-for="cluster in clusters"
              :key="cluster.id"
              :value="cluster.id"
              :label="cluster.name"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="$tr('业务分组')" required
          ><el-select v-model="form.groupId" filterable
            ><el-option
              v-for="group in groups"
              :key="group.id"
              :value="group.id"
              :label="group.name" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('环境')"
          ><EnvironmentSelect v-model="form.environment"
        /></el-form-item>
      </div>
      <el-form-item :label="$tr('编辑方式')">
        <el-radio-group
          :model-value="form.spec.mode"
          :disabled="changingMode"
          @update:model-value="changeMode"
        >
          <el-radio-button value="structured">{{
            $tr("可视化配置")
          }}</el-radio-button>
          <el-radio-button value="raw">{{ $tr("原文配置") }}</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item :label="$tr('配置文件路径')" required>
        <el-input
          v-model="form.spec.path"
          @input="pathCustomized = true"
          :placeholder="$tr('配置文件绝对路径')"
        />
        <div class="field-help">
          <span>{{
            $tr(
              configRoots.length > 1
                ? "节点配置目录不同，请确认适用于全部目标节点的配置路径。"
                : "根据集群实例的配置目录建议 conf.d 路径，可按需修改。"
            )
          }}</span>
          <el-button
            v-if="suggestedPath && form.spec.path !== suggestedPath"
            link
            type="primary"
            @click="useSuggestedPath"
            >{{ $tr("使用建议路径") }}</el-button
          >
        </div>
      </el-form-item>
      <div class="site-grid">
        <el-form-item
          :label="$tr('域名')"
          :required="form.spec.mode === 'structured'"
        >
          <el-select
            v-model="form.spec.domains"
            multiple
            filterable
            allow-create
            default-first-option
            :reserve-keyword="false"
            :placeholder="$tr('输入域名后按 Enter 添加')"
          />
        </el-form-item>
        <el-form-item
          :label="$tr('监听端口')"
          :required="form.spec.mode === 'structured'"
        >
          <el-select
            v-model="form.spec.ports"
            multiple
            filterable
            allow-create
            default-first-option
            :reserve-keyword="false"
            :placeholder="$tr('输入端口后按 Enter 添加')"
          >
            <el-option
              v-for="port in [80, 443, 8080, 8443]"
              :key="port"
              :value="port"
              :label="String(port)"
            />
          </el-select>
        </el-form-item>
      </div>
      <template v-if="form.spec.mode === 'structured'">
        <div class="section-heading">
          <h3>{{ $tr("路径规则") }}</h3>
          <el-button @click="addLocation">{{ $tr("添加路径规则") }}</el-button>
        </div>
        <div
          v-for="(location, index) in form.spec.locations"
          :key="location.key"
          class="location-card"
        >
          <div class="location-title">
            <strong>{{ $tr("规则 {value0}", { value0: index + 1 }) }}</strong
            ><el-button
              link
              type="danger"
              :disabled="form.spec.locations.length <= 1"
              @click="form.spec.locations.splice(index, 1)"
              >{{ $tr("移除") }}</el-button
            >
          </div>
          <div class="site-grid">
            <el-form-item :label="$tr('匹配路径')" required
              ><el-input v-model="location.path" placeholder="/"
            /></el-form-item>
            <el-form-item :label="$tr('服务类型')" required
              ><el-select
                v-model="location.kind"
                @change="changeLocationKind(location)"
                ><el-option value="proxy" :label="$tr('反向代理')" /><el-option
                  value="static"
                  :label="$tr('静态目录')" /><el-option
                  value="redirect"
                  :label="$tr('重定向')" /></el-select
            ></el-form-item>
          </div>
          <el-form-item
            :label="
              $tr(
                location.kind === 'static'
                  ? '静态目录'
                  : location.kind === 'redirect'
                    ? '重定向地址'
                    : '反向代理目标'
              )
            "
            required
          >
            <el-input
              v-model="location.target"
              :placeholder="
                location.kind === 'static'
                  ? '/srv/www/example'
                  : location.kind === 'redirect'
                    ? 'https://example.com$request_uri'
                    : 'http://127.0.0.1:8080'
              "
            />
          </el-form-item>
          <template v-if="location.kind === 'proxy'">
            <div class="site-grid">
              <el-form-item :label="$tr('代理超时（秒）')"
                ><el-input-number
                  v-model="location.timeout"
                  :min="1"
                  :max="3600"
                  controls-position="right"
              /></el-form-item>
              <el-form-item label="WebSocket"
                ><el-switch v-model="location.websocket"
              /></el-form-item>
            </div>
            <el-form-item :label="$tr('请求头')"
              ><el-input
                v-model="location.headersText"
                type="textarea"
                :rows="2"
                :placeholder="$tr('请求头，每行 名称: 值')"
            /></el-form-item>
          </template>
        </div>
        <el-divider content-position="left">HTTPS</el-divider>
        <el-form-item :label="$tr('启用 HTTPS')"
          ><el-switch v-model="form.spec.tls"
        /></el-form-item>
        <template v-if="form.spec.tls">
          <el-alert
            :title="
              $tr(
                '启用 HTTPS 后，所有监听端口均使用 TLS；HTTP 跳转请创建独立站点。'
              )
            "
            type="info"
            :closable="false"
          />
          <el-form-item :label="$tr('证书')" required>
            <el-select
              v-model="form.spec.certificateId"
              filterable
              :placeholder="$tr('选择已绑定到当前集群的证书')"
            >
              <el-option
                v-for="certificate in certificateChoices"
                :key="certificate.id"
                :value="certificate.id"
                :label="certificate.name"
                :disabled="!hasBinding(certificate.id)"
              />
            </el-select>
          </el-form-item>
          <p class="field-help">
            {{
              $tr(
                "证书须先建立当前集群的部署绑定；站点沿用绑定中的证书与私钥路径。"
              )
            }}
          </p>
          <el-alert
            v-if="tlsError"
            :title="tlsError"
            type="warning"
            :closable="false"
          />
        </template>
      </template>
      <template v-else>
        <el-alert
          :title="
            $tr(
              '原文模式保留原生配置；请填写完整 server 配置，发布前仍会执行 Nginx 校验。'
            )
          "
          type="info"
          :closable="false"
        />
        <el-form-item :label="$tr('原生 Nginx 配置')" required
          ><el-input
            v-model="form.spec.raw"
            type="textarea"
            :rows="14"
            class="site-code"
            spellcheck="false"
        /></el-form-item>
      </template>
      <el-collapse v-model="expanded">
        <el-collapse-item name="advanced" :title="$tr('日志与健康检查')">
          <div class="site-grid">
            <el-form-item
              v-if="form.spec.mode === 'structured'"
              :label="$tr('访问日志路径')"
              ><el-input
                v-model="form.spec.accessLog"
                :placeholder="$tr('留空继承 Nginx 配置')"
            /></el-form-item>
            <el-form-item
              v-if="form.spec.mode === 'structured'"
              :label="$tr('错误日志路径')"
              ><el-input
                v-model="form.spec.errorLog"
                :placeholder="$tr('留空继承 Nginx 配置')"
            /></el-form-item>
            <el-form-item
              v-if="form.spec.mode === 'structured'"
              :label="$tr('请求体上限')"
              ><el-input v-model="form.spec.maxBodySize" placeholder="10m"
            /></el-form-item>
            <el-form-item :label="$tr('站点健康检查地址')"
              ><el-input
                v-model="form.spec.healthUrl"
                placeholder="http://127.0.0.1:80/health"
            /></el-form-item>
            <el-form-item :label="$tr('健康检查 Host')"
              ><el-input
                v-model="form.spec.healthHost"
                placeholder="example.com"
            /></el-form-item>
            <el-form-item :label="$tr('预期状态码')"
              ><el-input-number
                v-model="form.spec.expectedStatus"
                :min="100"
                :max="599"
                controls-position="right"
            /></el-form-item>
          </div>
          <p class="field-help">
            {{ $tr("健康检查从目标节点执行，可使用本机地址和站点 Host。") }}
          </p>
        </el-collapse-item>
      </el-collapse>
      <el-alert
        v-if="error"
        :title="error"
        type="error"
        show-icon
        :closable="false"
      />
    </el-form>
    <template #footer>
      <el-button :disabled="saving" @click="emit('update:modelValue', false)">{{
        $tr("取消")
      }}</el-button>
      <el-button
        :loading="saving && !generating"
        :disabled="!canSave"
        @click="save(false)"
        >{{ $tr("保存站点") }}</el-button
      >
      <el-button
        type="primary"
        :loading="saving && generating"
        :disabled="!canSave"
        @click="save(true)"
        >{{ $tr("保存并生成草稿") }}</el-button
      >
    </template>
  </el-dialog>
</template>
<script setup lang="ts">
import EnvironmentSelect from "./EnvironmentSelect.vue";
import { computed, onUnmounted, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { tr } from "@/locales";
import { canPlatform } from "@/utils/platformAccess";
import { webRequest, type WebResource } from "@/api/webservice";

const props = defineProps<{
  modelValue: boolean;
  record?: WebResource;
  groups: { id: number; name: string }[];
  clusters: WebResource[];
  instances: WebResource[];
  certificates: WebResource[];
  bindings: WebResource[];
}>();
const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  saved: [site: WebResource];
  render: [site: WebResource];
}>();
interface Location {
  key: number;
  path: string;
  kind: "static" | "proxy" | "redirect";
  target: string;
  headersText: string;
  timeout: number;
  websocket: boolean;
}
let locationKey = 0;
function newLocation(value: Record<string, any> = {}): Location {
  return {
    key: ++locationKey,
    path: value.path || "/",
    kind: value.kind || "proxy",
    target: value.target || "",
    headersText: Object.entries(value.headers || {})
      .map(([name, value]) => `${name}: ${value}`)
      .join("\n"),
    timeout: value.timeout || 60,
    websocket: !!value.websocket
  };
}
function emptyForm() {
  return {
    name: "",
    groupId: undefined as number | undefined,
    clusterId: undefined as number | undefined,
    environment: "",
    engine: "nginx",
    spec: {
      mode: "structured",
      path: "",
      domains: [] as string[],
      ports: [80] as (string | number)[],
      locations: [newLocation()],
      tls: false,
      certificateId: undefined as number | undefined,
      raw: "",
      accessLog: "",
      errorLog: "",
      maxBodySize: "",
      healthUrl: "",
      healthHost: "",
      expectedStatus: 200
    }
  };
}
const form = ref(emptyForm());
const saving = ref(false),
  generating = ref(false),
  changingMode = ref(false),
  error = ref("");
const pathCustomized = ref(false),
  expanded = ref<string[]>([]);
let generation = 0;
const cluster = computed(() =>
  props.clusters.find(c => c.id === form.value.clusterId)
);
const configRoots = computed(() => [
  ...new Set(
    props.instances
      .filter(i => i.clusterId === form.value.clusterId)
      .map(i => {
        const path = String(i.spec?.configPath || "");
        return path.startsWith("/")
          ? path.slice(0, path.lastIndexOf("/"))
          : String(i.spec?.prefix || "").replace(/\/$/, "");
      })
      .filter(p => p.startsWith("/"))
  )
]);
const suggestedPath = computed(() => {
  if (configRoots.value.length !== 1) return "";
  const name =
    form.value.name
      .toLowerCase()
      .replace(/[^a-z0-9_-]+/g, "-")
      .replace(/^-+|-+$/g, "") ||
    form.value.spec.domains[0]
      ?.replace(/[^a-zA-Z0-9_-]+/g, "-")
      .replace(/^-+|-+$/g, "") ||
    "site";
  return `${configRoots.value[0]}/conf.d/${name}.conf`;
});
function useSuggestedPath() {
  pathCustomized.value = false;
  form.value.spec.path = suggestedPath.value;
}
watch(suggestedPath, value => {
  if (!pathCustomized.value) form.value.spec.path = value;
});
function selectCluster() {
  if (!props.record && cluster.value) {
    form.value.groupId = cluster.value.groupId;
    form.value.environment = cluster.value.environment || "";
    form.value.engine = cluster.value.engine || "nginx";
  }
  if (
    form.value.spec.certificateId &&
    !hasBinding(form.value.spec.certificateId)
  )
    form.value.spec.certificateId = undefined;
}
function hasBinding(id: number) {
  return props.bindings.some(
    b => b.clusterId === form.value.clusterId && b.spec.certificateId === id
  );
}
const certificateChoices = computed(() =>
  props.certificates.filter(
    c => hasBinding(c.id) || c.id === form.value.spec.certificateId
  )
);
const certificate = computed(() =>
  props.certificates.find(c => c.id === form.value.spec.certificateId)
);
function covered(domain: string, san: string) {
  if (domain === san) return true;
  if (!san.startsWith("*.") || domain.startsWith("*.")) return false;
  const suffix = san.slice(1);
  return (
    domain.endsWith(suffix) && !domain.slice(0, -suffix.length).includes(".")
  );
}
const tlsError = computed(() => {
  if (!form.value.spec.tls || form.value.spec.mode === "raw") return "";
  if (
    !form.value.spec.certificateId ||
    !hasBinding(form.value.spec.certificateId)
  )
    return tr("请先为集群创建证书部署绑定");
  if (!certificate.value?.spec.currentVersionId)
    return tr("请先导入或签发证书");
  const sans = (certificate.value.spec.domains || []).map((domain: string) =>
    domain.toLowerCase()
  );
  if (
    form.value.spec.domains.some(
      domain =>
        !sans.some((san: string) => covered(domain.trim().toLowerCase(), san))
    )
  )
    return tr("所选证书未覆盖全部站点域名");
  return "";
});
const canSave = computed(
  () =>
    canPlatform("web:write") &&
    !saving.value &&
    !!form.value.name.trim() &&
    !!form.value.groupId &&
    !!form.value.clusterId &&
    !!form.value.spec.path &&
    (form.value.spec.mode === "raw"
      ? !!form.value.spec.raw.trim()
      : !!form.value.spec.domains.length &&
        !!form.value.spec.ports.length &&
        !!form.value.spec.locations.length &&
        !tlsError.value)
);
function addLocation() {
  form.value.spec.locations.push(newLocation());
}
function changeLocationKind(location: Location) {
  location.target = "";
  location.headersText = "";
  location.websocket = false;
}
async function changeMode(value: string | number | boolean | undefined) {
  if (changingMode.value || saving.value) return;
  const gen = generation;
  changingMode.value = true;
  try {
    if (
      value === "structured" &&
      form.value.spec.mode === "raw" &&
      form.value.spec.raw.trim()
    ) {
      try {
        await ElMessageBox.confirm(
          tr(
            "原文不会自动转换为可视化规则；保存可视化配置后将使用下方规则生成文件，是否继续？"
          ),
          tr("编辑方式"),
          {
            type: "warning",
            confirmButtonText: tr("确定"),
            cancelButtonText: tr("取消"),
            lockScroll: false
          }
        );
      } catch {
        return;
      }
    }
    if (gen === generation && props.modelValue)
      form.value.spec.mode = String(value);
  } finally {
    if (gen === generation) changingMode.value = false;
  }
}
watch(
  () => props.modelValue,
  open => {
    generation++;
    saving.value = false;
    generating.value = false;
    changingMode.value = false;
    error.value = "";
    if (!open) return;
    const record = props.record;
    const value = emptyForm();
    if (record) {
      Object.assign(value, {
        name: record.name,
        groupId: record.groupId,
        clusterId: record.clusterId,
        environment: record.environment,
        engine: record.engine
      });
      Object.assign(value.spec, JSON.parse(JSON.stringify(record.spec)));
      value.spec.locations = (record.spec.locations || []).map(newLocation);
      if (!value.spec.locations.length)
        value.spec.locations.push(newLocation());
      value.spec.domains ||= [];
      value.spec.ports ||= [];
    } else if (props.clusters.length === 1) {
      value.clusterId = props.clusters[0].id;
      value.groupId = props.clusters[0].groupId;
      value.environment = props.clusters[0].environment;
      value.engine = props.clusters[0].engine;
    }
    pathCustomized.value = !!record?.spec.path;
    form.value = value;
    expanded.value =
      record &&
      (record.spec.accessLog || record.spec.errorLog || record.spec.healthUrl)
        ? ["advanced"]
        : [];
    if (!pathCustomized.value) form.value.spec.path = suggestedPath.value;
  },
  { immediate: true }
);
function headers(text: string) {
  const value: Record<string, string> = Object.create(null);
  for (const line of text.split("\n").filter(v => v.trim())) {
    const colon = line.indexOf(":");
    const name = line.slice(0, colon).trim();
    if (colon < 1 || !/^[A-Za-z0-9_-]+$/.test(name))
      throw new Error(tr("请求头格式无效，请使用每行 名称: 值"));
    value[name] = line.slice(colon + 1).trim();
  }
  return value;
}
async function save(render: boolean) {
  if (!canSave.value) return;
  const gen = generation;
  saving.value = true;
  generating.value = render;
  error.value = "";
  try {
    const spec = {
      ...form.value.spec,
      domains: [
        ...new Set(
          form.value.spec.domains
            .map(d => d.trim().toLowerCase())
            .filter(Boolean)
        )
      ],
      ports: [...new Set(form.value.spec.ports.map(Number))],
      locations:
        form.value.spec.mode === "raw"
          ? []
          : form.value.spec.locations.map(
              ({ key, headersText, ...location }) => ({
                ...location,
                headers: headers(headersText)
              })
            )
    };
    if (
      spec.mode !== "raw" &&
      spec.ports.some(p => !Number.isInteger(p) || p < 1 || p > 65535)
    )
      throw new Error(tr("监听端口无效"));
    if (!spec.tls && spec.mode !== "raw") spec.certificateId = undefined;
    const saved = await webRequest<WebResource>(
      "/sites" + (props.record ? `/${props.record.id}` : ""),
      props.record ? "PUT" : "POST",
      {
        ...form.value,
        name: form.value.name.trim(),
        revision: props.record?.revision,
        spec
      }
    );
    if (gen !== generation || !props.modelValue) return;
    ElMessage.success(tr("站点设置已保存，尚未发布"));
    emit("saved", saved);
    emit("update:modelValue", false);
    if (render) emit("render", saved);
  } catch (e) {
    if (gen === generation) error.value = (e as Error).message;
  } finally {
    if (gen === generation) saving.value = false;
  }
}
onUnmounted(() => {
  generation++;
});
</script>
<style scoped>
.site-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 20px;
}
.el-form {
  margin-top: 18px;
}
.el-select,
.el-input-number {
  width: 100%;
}
.field-help {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 12px;
  align-items: center;
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.6;
}
.section-heading,
.location-title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.section-heading h3 {
  margin: 0;
  font-size: 15px;
}
.location-card {
  padding: 16px;
  margin-bottom: 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  background: var(--el-fill-color-light);
}
.location-card .el-form-item:last-child {
  margin-bottom: 0;
}
.site-code :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
}
.el-alert {
  margin-bottom: 14px;
}
@media (max-width: 700px) {
  .site-grid {
    grid-template-columns: 1fr;
  }
}
</style>
<style>
.web-site-dialog {
  max-height: calc(100dvh - 32px);
  display: flex;
  flex-direction: column;
  margin: 0 auto;
}
.web-site-dialog > .el-dialog__body {
  min-height: 0;
  overflow: auto;
}
.web-site-dialog > .el-dialog__header,
.web-site-dialog > .el-dialog__footer {
  flex-shrink: 0;
}
.web-site-dialog > .el-dialog__footer {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
}
.web-site-dialog > .el-dialog__footer .el-button {
  margin-left: 0;
}
</style>
