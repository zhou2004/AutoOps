<script setup lang="ts">
import * as OperationIcons from "@element-plus/icons-vue";
import { tr } from "@/locales";

import {
  computed,
  reactive,
  ref,
  watch,
  onActivated,
  onDeactivated,
  onBeforeUnmount
} from "vue";
import { useRoute } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import ClusterSelector from "../pods/ClusterSelector.vue";
import NamespaceSelector from "../pods/NamespaceSelector.vue";
import {
  autoscalingApi,
  type ScalingKind,
  type ScalingObject,
  type ScalingCapabilities,
  type ScalingRequest
} from "@/api/autoscaling";

defineOptions({ name: "K8sAutoscaling" });
const props = defineProps<{
  embedded?: boolean;
  clusterId?: string;
  namespace?: string;
  targetKind?: string;
  targetName?: string;
}>();
const route = useRoute();
const cluster = ref(String(props.clusterId || route.query.clusterId || ""));
const namespace = ref(String(props.namespace || route.query.namespace || ""));
const kind = ref<ScalingKind>(props.targetKind === "DaemonSet" ? "vpa" : "hpa");
const caps = ref<ScalingCapabilities | null>(null);
const items = ref<ScalingObject[]>([]);
const busy = ref(false);
const saving = ref(false);
const error = ref("");
const search = ref("");
const editor = ref(false);
const detail = ref(false);
const selected = ref<ScalingObject | null>(null);
const editing = ref(false);
const preview = ref("");
const formError = ref("");
let generation = 0;
let active = true;
function invalidate() {
  active = false;
  generation++;
  busy.value = false;
  saving.value = false;
  editor.value = false;
  detail.value = false;
}
onDeactivated(invalidate);
onBeforeUnmount(invalidate);
onActivated(() => {
  if (!active) {
    active = true;
    void refresh();
  }
});
const p = "autoscaling.knative.dev/";
const form = reactive({
  name: "",
  version: "",
  targetKind: "Deployment",
  targetName: "",
  min: 1,
  max: 10,
  metrics: "",
  behavior: "",
  mode: "Off",
  vpaMinReplicas: 2 as number | undefined,
  policies: ""
});
const kpa = reactive<{
  metric: string;
  target?: number;
  minScale?: number;
  maxScale?: number;
  windowSeconds?: number;
  panicWindowPercentage?: number;
  panicThresholdPercentage?: number;
  targetUtilizationPercentage?: number;
  scaleDownDelaySeconds?: number;
}>({ metric: "" });
const allowed = (verb: string) => !!caps.value?.permissions[kind.value]?.[verb];
const available = computed(
  () => !!caps.value?.capabilities[kind.value]?.available
);
const filtered = computed(() =>
  items.value.filter(row => {
    const target = row.spec.scaleTargetRef || row.spec.targetRef;
    return (
      (!props.targetName ||
        (target?.name === props.targetName &&
          target?.kind === props.targetKind)) &&
      row.metadata.name.includes(search.value)
    );
  })
);
const managed = (row: ScalingObject) =>
  !!row.metadata.ownerReferences?.some(owner => owner.controller);
const pretty = (value: unknown) => JSON.stringify(value ?? {}, null, 2);
const target = (row: ScalingObject) => {
  const ref = row.spec.scaleTargetRef || row.spec.targetRef;
  return ref ? `${ref.kind}/${ref.name}` : `Service/${row.metadata.name}`;
};
const annotations = (row: ScalingObject) =>
  row.spec.template?.metadata?.annotations || {};
const summary = (row: ScalingObject) => {
  if (kind.value === "hpa")
    return `${row.status?.currentReplicas ?? "—"} → ${row.status?.desiredReplicas ?? "—"} 副本（${row.spec.minReplicas ?? 1}–${row.spec.maxReplicas}）`;
  if (kind.value === "vpa")
    return `${row.spec.updatePolicy?.updateMode || "控制器默认"} · ${row.status?.recommendation?.containerRecommendations?.length || 0} 个容器建议`;
  const a = annotations(row);
  const upper =
    a[p + "max-scale"] === "0" ? "∞" : a[p + "max-scale"] || "集群默认";
  return `${a[p + "class"]?.startsWith("hpa.") ? "Knative HPA" : a[p + "class"] ? "KPA" : "集群默认"} · ${a[p + "metric"] || "默认指标"} · ${a[p + "min-scale"] || "默认"}–${upper}`;
};
const condition = (row: ScalingObject) => {
  const conditions = row.status?.conditions || [];
  const failed = conditions.find((c: any) => c.status === "False");
  return failed
    ? `${failed.type}: ${failed.reason || failed.message || "False"}`
    : conditions.length
      ? conditions.map((c: any) => `${c.type}: ${c.status}`).join(" · ")
      : "等待控制器状态";
};

async function refresh() {
  if (!active) return;
  const version = ++generation;
  items.value = [];
  caps.value = null;
  error.value = "";
  if (!cluster.value || !namespace.value) {
    busy.value = false;
    return;
  }
  busy.value = true;
  const id = String(cluster.value),
    ns = namespace.value,
    type = kind.value;
  try {
    const capability = await autoscalingApi.capabilities(id, ns);
    if (version !== generation) return;
    caps.value = capability;
    if (
      capability.capabilities[type].available &&
      capability.permissions[type].list
    ) {
      const list = await autoscalingApi.list(id, ns, type);
      if (version === generation) items.value = list;
    }
  } catch (e) {
    if (version === generation) error.value = (e as Error).message;
  } finally {
    if (version === generation) busy.value = false;
  }
}
watch(cluster, () => {
  namespace.value = "";
});
watch(
  [cluster, namespace, kind],
  () => {
    saving.value = false;
    editor.value = false;
    detail.value = false;
    selected.value = null;
    void refresh();
  },
  { immediate: true }
);

function openEditor(row?: ScalingObject) {
  editing.value = !!row;
  selected.value = row || null;
  const spec = row?.spec || {};
  const ref = spec.scaleTargetRef || spec.targetRef || {};
  Object.assign(form, {
    name:
      row?.metadata.name ||
      (props.targetName
        ? `${props.targetName}-${kind.value}`.slice(0, 253)
        : ""),
    version: row?.metadata.resourceVersion || "",
    targetKind: ref.kind || props.targetKind || "Deployment",
    targetName: ref.name || props.targetName || "",
    min: spec.minReplicas ?? 1,
    max: spec.maxReplicas ?? 10,
    metrics: pretty(
      spec.metrics || [
        {
          type: "Resource",
          resource: {
            name: "cpu",
            target: { type: "Utilization", averageUtilization: 70 }
          }
        }
      ]
    ),
    behavior: pretty(
      spec.behavior ||
        (row
          ? {}
          : {
              scaleDown: {
                stabilizationWindowSeconds: 300,
                policies: [{ type: "Percent", value: 25, periodSeconds: 60 }]
              }
            })
    ),
    mode: spec.updatePolicy?.updateMode || (row ? "" : "Off"),
    vpaMinReplicas: spec.updatePolicy?.minReplicas ?? (row ? undefined : 2),
    policies: pretty(
      spec.resourcePolicy?.containerPolicies ||
        (row
          ? []
          : [
              {
                containerName: "*",
                controlledResources: ["cpu", "memory"],
                controlledValues: "RequestsOnly",
                minAllowed: { cpu: "100m", memory: "128Mi" },
                maxAllowed: { cpu: "2", memory: "2Gi" }
              }
            ])
    )
  });
  const a = row ? annotations(row) : {};
  const numeric = (key: string) =>
    a[p + key] === undefined ? undefined : Number(a[p + key]);
  Object.assign(kpa, {
    metric: a[p + "metric"] || "",
    target: numeric("target"),
    minScale: numeric("min-scale"),
    maxScale: numeric("max-scale"),
    windowSeconds: durationSeconds(a[p + "window"]),
    panicWindowPercentage: numeric("panic-window-percentage"),
    panicThresholdPercentage: numeric("panic-threshold-percentage"),
    targetUtilizationPercentage: numeric("target-utilization-percentage"),
    scaleDownDelaySeconds: durationSeconds(a[p + "scale-down-delay"])
  });
  preview.value = "";
  formError.value = "";
  editor.value = true;
}
function durationSeconds(
  value: string | undefined,
  fallback: number | undefined = undefined
) {
  if (!value) return fallback;
  const parts = [...value.matchAll(/(\d+(?:\.\d+)?)(h|m|s)/g)];
  return parts.length
    ? parts.reduce(
        (total, part) =>
          total + Number(part[1]) * ({ h: 3600, m: 60, s: 1 }[part[2]] || 1),
        0
      )
    : fallback;
}
function parse(value: string, array: boolean) {
  const data = JSON.parse(value);
  if (!data || Array.isArray(data) !== array || typeof data !== "object")
    throw new Error(
      array ? "此字段必须是 JSON 数组" : "此字段必须是 JSON 对象"
    );
  return data;
}
function requestBody(): ScalingRequest {
  if (!form.name.trim()) throw new Error(tr("请填写资源名称"));
  const data: ScalingRequest = {
    name: form.name.trim(),
    resourceVersion: form.version
  };
  const ref = {
    apiVersion: "apps/v1",
    kind: form.targetKind,
    name: form.targetName.trim()
  };
  if (kind.value === "hpa")
    data.hpa = {
      scaleTargetRef: ref,
      minReplicas: form.min,
      maxReplicas: form.max,
      metrics: parse(form.metrics, true),
      behavior: parse(form.behavior, false)
    };
  if (kind.value === "vpa")
    data.vpa = {
      targetRef: ref,
      updatePolicy: {
        ...(form.mode ? { updateMode: form.mode } : {}),
        ...(form.vpaMinReplicas != null
          ? { minReplicas: form.vpaMinReplicas }
          : {})
      },
      resourcePolicy: { containerPolicies: parse(form.policies, true) }
    };
  if (kind.value === "kpa")
    data.kpa = Object.fromEntries(
      Object.entries(kpa).filter(
        ([, value]) => value !== undefined && value !== null && value !== ""
      )
    );
  return data;
}
async function save(onlyPreview: boolean) {
  saving.value = true;
  formError.value = "";
  const id = String(cluster.value),
    ns = namespace.value,
    type = kind.value,
    version = generation;
  try {
    const body = requestBody();
    const create = !editing.value;
    const checked = await autoscalingApi.save(id, ns, type, body, create, true);
    if (version !== generation) return;
    preview.value = pretty(checked);
    if (onlyPreview) {
      ElMessage.success(tr("服务端试运行通过，尚未保存"));
      return;
    }
    const impact =
      type === "kpa"
        ? "将更新 Knative Service 模板并产生新修订，可能调整流量承载副本。"
        : type === "vpa" && form.mode !== "Off"
          ? "VPA 可能调整 Pod 资源或驱逐重建 Pod，请确认业务可承受。"
          : "控制器将按此策略自动调整工作负载。";
    await ElMessageBox.confirm(
      impact,
      `保存 ${type.toUpperCase()} / ${body.name}`,
      {
        type: "warning",
        confirmButtonText: "保存策略",
        cancelButtonText: tr("取消")
      }
    );
    if (version !== generation) return;
    await autoscalingApi.save(id, ns, type, body, create, false);
    if (version !== generation) return;
    saving.value = false;
    ElMessage.success(tr("伸缩策略已保存"));
    editor.value = false;
    await refresh();
  } catch (e) {
    if (e !== "cancel" && e !== "close" && version === generation)
      formError.value = (e as Error).message;
  } finally {
    if (version === generation) saving.value = false;
  }
}
async function remove(row: ScalingObject) {
  const id = String(cluster.value),
    ns = namespace.value,
    type = kind.value,
    version = generation;
  try {
    await ElMessageBox.confirm(
      type === "kpa"
        ? `将清除 ${row.metadata.name} 的本页面伸缩注解并恢复集群默认策略，Service 会保留，新修订将由 Knative 创建。`
        : `删除 ${row.metadata.name} 后，此策略将停止控制目标工作负载。`,
      type === "kpa" ? "重置 KPA 策略" : "删除伸缩策略",
      {
        type: "warning",
        confirmButtonText: tr("确认"),
        cancelButtonText: tr("取消")
      }
    );
    if (version !== generation) return;
    saving.value = true;
    await autoscalingApi.remove(id, ns, type, row);
    if (version !== generation) return;
    saving.value = false;
    ElMessage.success(type === "kpa" ? "已恢复默认伸缩策略" : "策略已删除");
    await refresh();
  } catch (e) {
    if (e !== "cancel" && e !== "close" && version === generation)
      ElMessage.error((e as Error).message);
  } finally {
    if (version === generation) saving.value = false;
  }
}
function addMetric(name: "cpu" | "memory") {
  try {
    const metrics = parse(form.metrics, true);
    if (
      metrics.some(
        (m: any) => m.type === "Resource" && m.resource?.name === name
      )
    )
      return;
    metrics.push({
      type: "Resource",
      resource: {
        name,
        target: { type: "Utilization", averageUtilization: 70 }
      }
    });
    form.metrics = pretty(metrics);
  } catch (e) {
    formError.value = (e as Error).message;
  }
}
</script>

<template>
  <div class="autoscaling-page" :class="{ embedded }">
    <el-card shadow="never">
      <template #header
        ><div class="heading">
          <div>
            <h2>{{ $tr("弹性伸缩") }}</h2>
            <p>
              {{
                $tr("HPA 水平伸缩 · VPA 资源建议与调整 · Knative KPA 请求伸缩")
              }}
            </p>
          </div>
          <el-button :icon="OperationIcons.Refresh"
            :loading="busy"
            :disabled="!cluster || !namespace || saving"
            @click="refresh"
            >{{ $tr("刷新状态") }}</el-button
          >
        </div></template
      >
      <div v-if="embedded" class="workload-context">
        <el-tag>{{ namespace }}</el-tag
        ><strong>{{ targetKind }} / {{ targetName }}</strong>
      </div>
      <div class="toolbar">
        <ClusterSelector v-if="!embedded" v-model="cluster" />
        <NamespaceSelector
          v-if="!embedded"
          :key="String(cluster)"
          v-model="namespace"
          :cluster-id="String(cluster)"
        />
        <el-input
          v-model="search"
          :placeholder="$tr('筛选策略 / Service 名称')"
          clearable
          class="search"
        />
      </div>
      <el-alert
        v-if="error"
        :title="$tr(error)"
        type="error"
        :closable="false"
        show-icon
      />
      <el-tabs v-model="kind">
        <el-tab-pane
          v-if="targetKind !== 'DaemonSet'"
          :label="$tr('HPA · 水平伸缩')"
          name="hpa"
        />
        <el-tab-pane :label="$tr('VPA · 垂直伸缩')" name="vpa" />
        <el-tab-pane
          v-if="!targetName"
          :label="$tr('KPA · Knative 伸缩')"
          name="kpa"
        />
      </el-tabs>
      <el-empty
        v-if="!cluster || !namespace"
        :description="$tr('请选择集群与命名空间')"
      />
      <template v-else>
        <el-alert
          v-if="caps && !available"
          :title="`${kind.toUpperCase()} API 不可用`"
          :description="$tr(caps.capabilities[kind].message)"
          type="warning"
          :closable="false"
          show-icon
        />
        <el-alert
          v-else-if="caps && !allowed('list')"
          :title="$tr('当前账号没有此类型的 list 权限')"
          type="warning"
          :closable="false"
        />
        <el-alert
          v-if="kind === 'hpa' && caps && !caps.capabilities.metrics.available"
          :title="
            $tr(
              '未检测到 metrics.k8s.io/v1beta1：CPU/内存伸缩需要可用的 metrics-server；业务指标还需对应 Adapter。'
            )
          "
          type="warning"
          :closable="false"
        />
        <el-alert
          v-if="kind === 'vpa'"
          :title="
            $tr(
              '建议先使用 Off 模式观察推荐值。资源指标 HPA 与自动调整 requests 的 VPA 会互相影响；本页面会阻止同目标冲突。'
            )
          "
          type="info"
          :closable="false"
        />
        <el-alert
          v-if="kind === 'kpa'"
          :title="
            $tr(
              '选择已有 Knative Service 配置请求伸缩。修改会产生新 Revision；缩至 0 还取决于 Knative 集群配置。'
            )
          "
          type="info"
          :closable="false"
        />
        <div class="actions">
          <el-tag v-if="caps">{{ caps.capabilities[kind].apiVersion }}</el-tag
          ><el-button :icon="OperationIcons.Plus"
            v-if="kind !== 'kpa'"
            type="primary"
            :disabled="!available || !allowed('create') || saving"
            @click="openEditor()"
            >{{ $tr("创建") }} {{ kind.toUpperCase() }}</el-button
          >
        </div>
        <el-table
          v-loading="busy"
          :data="filtered"
          :empty-text="$tr('此命名空间暂无资源')"
          stripe
        >
          <el-table-column :label="$tr('名称')" min-width="160"
            ><template #default="{ row }"
              ><strong>{{ row.metadata.name }}</strong
              ><el-tag v-if="managed(row)" size="small" type="info">{{
                $tr("控制器管理")
              }}</el-tag></template
            ></el-table-column
          >
          <el-table-column :label="$tr('目标')" min-width="200"
            ><template #default="{ row }">{{
              target(row)
            }}</template></el-table-column
          >
          <el-table-column :label="$tr('伸缩状态')" min-width="250"
            ><template #default="{ row }">{{
              summary(row)
            }}</template></el-table-column
          >
          <el-table-column
            :label="$tr('条件')"
            min-width="220"
            show-overflow-tooltip
            ><template #default="{ row }">{{
              condition(row)
            }}</template></el-table-column
          >
          <el-table-column :label="$tr('操作')" width="225" fixed="right"
            ><template #default="{ row }">
              <el-button :icon="OperationIcons.View"
                link
                type="primary"
                @click="
                  selected = row;
                  detail = true;
                "
                >{{ $tr("详情") }}</el-button
              >
              <el-button :icon="OperationIcons.Edit"
                link
                type="primary"
                :disabled="!allowed('update') || managed(row) || saving"
                @click="openEditor(row)"
                >{{ kind === "kpa" ? $tr("配置") : $tr("编辑") }}</el-button
              >
              <el-button :icon="OperationIcons.Delete"
                link
                type="danger"
                :disabled="!allowed('delete') || managed(row) || saving"
                @click="remove(row)"
                >{{ kind === "kpa" ? $tr("恢复默认") : $tr("删除") }}</el-button
              >
            </template></el-table-column
          >
        </el-table>
      </template>
    </el-card>

    <el-dialog
      v-model="editor"
      :title="`${editing ? $tr('编辑') : $tr('创建')} ${kind.toUpperCase()} 策略`"
      width="min(880px, 94vw)"
      append-to-body
      :close-on-click-modal="false"
      :close-on-press-escape="!saving"
      :show-close="!saving"
      destroy-on-close
    >
      <el-alert
        v-if="formError"
        :title="$tr(formError)"
        type="error"
        :closable="false"
        show-icon
      />
      <el-form label-width="160px" :disabled="saving">
        <el-form-item
          :label="kind === 'kpa' ? 'Knative Service' : '策略名称'"
          required
          ><el-input v-model="form.name" :disabled="editing"
        /></el-form-item>
        <template v-if="kind !== 'kpa'">
          <el-form-item :label="$tr('工作负载类型')" required
            ><el-select v-model="form.targetKind" :disabled="!!targetName"
              ><el-option label="Deployment" value="Deployment" /><el-option
                label="StatefulSet"
                value="StatefulSet" /><el-option
                v-if="kind === 'vpa'"
                label="DaemonSet"
                value="DaemonSet" /></el-select
          ></el-form-item>
          <el-form-item :label="$tr('工作负载名称')" required
            ><el-input
              v-model="form.targetName"
              :disabled="!!targetName"
              :placeholder="$tr('此命名空间内的已有工作负载')"
          /></el-form-item>
        </template>
        <template v-if="kind === 'hpa'">
          <el-form-item :label="$tr('副本边界')"
            ><el-input-number v-model="form.min" :min="1" :precision="0" /><span
              class="between"
              >{{ $tr("至") }}</span
            ><el-input-number v-model="form.max" :min="form.min" :precision="0"
          /></el-form-item>
          <el-form-item :label="$tr('指标 metrics')" required
            ><div class="field">
              <div>
                <el-button :icon="OperationIcons.Plus" size="small" @click="addMetric('cpu')">{{
                  $tr("添加 CPU")
                }}</el-button
                ><el-button :icon="OperationIcons.Plus" size="small" @click="addMetric('memory')">{{
                  $tr("添加内存")
                }}</el-button>
              </div>
              <el-input
                v-model="form.metrics"
                type="textarea"
                :rows="12"
                class="code"
              /><small>{{
                $tr(
                  "支持 Resource、ContainerResource、Pods、Object、External 的 autoscaling/v2 指标数组。利用率以 requests 为基准。"
                )
              }}</small>
            </div></el-form-item
          >
          <el-form-item :label="$tr('伸缩 behavior')"
            ><div class="field">
              <el-input
                v-model="form.behavior"
                type="textarea"
                :rows="9"
                class="code"
              /><small>{{
                $tr(
                  "设置 scaleUp / scaleDown、stabilizationWindowSeconds、selectPolicy 和 policies。{} 使用集群默认行为。"
                )
              }}</small>
            </div></el-form-item
          >
        </template>
        <template v-if="kind === 'vpa'">
          <el-form-item :label="$tr('更新模式')"
            ><el-select v-model="form.mode"
              ><el-option
                :label="$tr('控制器默认（保留未指定模式）')"
                value="" /><el-option
                :label="$tr('Off · 仅提供建议')"
                value="Off" /><el-option
                :label="$tr('Initial · 创建 Pod 时设置')"
                value="Initial" /><el-option
                :label="$tr('Recreate · 驱逐并重建')"
                value="Recreate" /><el-option
                :label="$tr('Auto · 控制器决定')"
                value="Auto" /><el-option
                :label="$tr('InPlaceOrRecreate · 原地或重建')"
                value="InPlaceOrRecreate" /><el-option
                :label="$tr('InPlace · 仅原地调整')"
                value="InPlace" /></el-select
          ></el-form-item>
          <el-form-item :label="$tr('驱逐最少副本')"
            ><el-input-number
              v-model="form.vpaMinReplicas"
              :placeholder="$tr('控制器默认')"
              :min="1"
              :precision="0"
          /></el-form-item>
          <el-form-item :label="$tr('容器资源策略')"
            ><div class="field">
              <el-input
                v-model="form.policies"
                type="textarea"
                :rows="16"
                class="code"
              /><small>{{
                $tr(
                  "containerPolicies 数组：支持每容器 minAllowed/maxAllowed、controlledResources、RequestsOnly/RequestsAndLimits 和 mode: Off。* 为默认容器规则。"
                )
              }}</small>
            </div></el-form-item
          >
          <el-alert
            :title="
              $tr(
                'InPlace 系列需要已安装的 VPA 与 Kubernetes 版本支持，服务端试运行会校验 CRD。Auto 的行为取决于控制器版本。'
              )
            "
            type="info"
            :closable="false"
          />
        </template>
        <template v-if="kind === 'kpa'">
          <el-alert
            :title="
              $tr(
                '未设置的字段留空，保存时保留现有注解与集群默认值。需要移除已设置的策略时使用列表中的恢复默认。'
              )
            "
            type="info"
            :closable="false"
          />
          <el-form-item :label="$tr('指标')"
            ><el-select v-model="kpa.metric"
              ><el-option
                :label="$tr('保留现有 / 集群默认')"
                value="" /><el-option
                :label="$tr('并发请求 concurrency')"
                value="concurrency" /><el-option
                :label="$tr('每秒请求 rps')"
                value="rps" /></el-select
          ></el-form-item>
          <el-form-item :label="$tr('每副本目标')"
            ><el-input-number v-model="kpa.target" :min="0.01"
          /></el-form-item>
          <el-form-item :label="$tr('最少 / 最多副本')"
            ><el-input-number
              v-model="kpa.minScale"
              :min="0"
              :precision="0"
            /><span class="between">/</span
            ><el-input-number
              v-model="kpa.maxScale"
              :min="0"
              :precision="0"
            /><small>{{ $tr("最大值 0 表示不限制") }}</small></el-form-item
          >
          <el-form-item :label="$tr('稳定窗口（秒）')"
            ><el-input-number
              v-model="kpa.windowSeconds"
              :min="6"
              :max="3600"
              :precision="0"
          /></el-form-item>
          <el-form-item :label="$tr('panic 窗口百分比')"
            ><el-input-number
              v-model="kpa.panicWindowPercentage"
              :min="1"
              :max="100"
          /></el-form-item>
          <el-form-item :label="$tr('panic 阈值百分比')"
            ><el-input-number
              v-model="kpa.panicThresholdPercentage"
              :min="110"
              :max="1000"
          /></el-form-item>
          <el-form-item :label="$tr('目标利用率百分比')"
            ><el-input-number
              v-model="kpa.targetUtilizationPercentage"
              :min="0.01"
              :max="100"
          /></el-form-item>
          <el-form-item :label="$tr('缩容延迟（秒）')"
            ><el-input-number
              v-model="kpa.scaleDownDelaySeconds"
              :min="0"
              :max="3600"
              :precision="0"
          /></el-form-item>
        </template>
      </el-form>
      <el-collapse v-if="preview"
        ><el-collapse-item :title="$tr('服务端试运行结果（尚未保存）')">
          <pre>{{ preview }}</pre>
        </el-collapse-item></el-collapse
      >
      <template #footer
        ><el-button :icon="OperationIcons.Close" :disabled="saving" @click="editor = false">{{
          $tr("取消")
        }}</el-button
        ><el-button :icon="OperationIcons.Operation" :loading="saving" @click="save(true)">{{
          $tr("服务端试运行")
        }}</el-button
        ><el-button :icon="OperationIcons.DocumentChecked" type="primary" :loading="saving" @click="save(false)">{{
          $tr("校验并保存")
        }}</el-button></template
      >
    </el-dialog>

    <el-drawer
      v-model="detail"
      :title="$tr('伸缩状态与策略详情')"
      size="65%"
      destroy-on-close
    >
      <template v-if="selected">
        <el-descriptions :column="1" border
          ><el-descriptions-item :label="$tr('名称')">{{
            selected.metadata.name
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('目标')">{{
            target(selected)
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('版本')">{{
            selected.metadata.resourceVersion
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('状态')">{{
            summary(selected)
          }}</el-descriptions-item></el-descriptions
        >
        <h3>{{ $tr("控制器条件") }}</h3>
        <el-table :data="selected.status?.conditions || []"
          ><el-table-column prop="type" :label="$tr('条件')" /><el-table-column
            prop="status"
            :label="$tr('状态')"
            width="90" /><el-table-column
            prop="reason"
            :label="$tr('原因')" /><el-table-column
            prop="message"
            :label="$tr('说明')"
            min-width="250"
        /></el-table>
        <template v-if="kind === 'vpa'"
          ><h3>{{ $tr("容器资源建议") }}</h3>
          <el-table
            :data="
              selected.status?.recommendation?.containerRecommendations || []
            "
            ><el-table-column
              prop="containerName"
              :label="$tr('容器')"
            /><el-table-column :label="$tr('建议值')"
              ><template #default="{ row }">
                <pre>{{ pretty(row.target) }}</pre>
              </template></el-table-column
            ><el-table-column :label="$tr('下界 / 上界')"
              ><template #default="{ row }">
                <pre>{{
                  pretty({ lower: row.lowerBound, upper: row.upperBound })
                }}</pre>
              </template></el-table-column
            ><el-table-column :label="$tr('未约束建议')"
              ><template #default="{ row }">
                <pre>{{ pretty(row.uncappedTarget) }}</pre>
              </template></el-table-column
            ></el-table
          ></template
        >
        <template v-if="kind === 'hpa'"
          ><h3>{{ $tr("当前指标 / 目标指标") }}</h3>
          <pre>{{
            pretty({
              current: selected.status?.currentMetrics,
              desired: selected.spec.metrics
            })
          }}</pre>
        </template>
        <template v-if="kind === 'kpa'"
          ><h3>{{ $tr("Knative 修订与路由") }}</h3>
          <pre>{{
            pretty({
              latestReadyRevision: selected.status?.latestReadyRevisionName,
              latestCreatedRevision: selected.status?.latestCreatedRevisionName,
              url: selected.status?.url,
              traffic: selected.status?.traffic
            })
          }}</pre>
        </template>
        <h3>{{ $tr("完整资源") }}</h3>
        <pre>{{ pretty(selected) }}</pre>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
.autoscaling-page.embedded {
  padding: 0;
}
.workload-context {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  color: var(--text-primary);
}

.autoscaling-page {
  padding: 16px;
}
.heading,
.toolbar,
.actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.heading {
  justify-content: space-between;
}
h2 {
  margin: 0 0 8px;
}
h3 {
  margin: 24px 0 12px;
}
p,
small {
  color: var(--el-text-color-secondary);
}
p {
  margin: 0;
}
.toolbar,
.actions {
  margin-bottom: 16px;
}
.actions {
  justify-content: space-between;
  margin-top: 16px;
}
.search {
  max-width: 280px;
}
.el-alert {
  margin-bottom: 14px;
}
.field {
  width: 100%;
  display: grid;
  gap: 8px;
}
.between {
  margin: 0 12px;
}
small {
  margin-left: 8px;
  line-height: 1.6;
}
pre {
  white-space: pre-wrap;
  word-break: break-word;
  background: var(--el-fill-color-light);
  padding: 12px;
  border-radius: 6px;
  font-size: 12px;
}
.code :deep(textarea) {
  font-family: monospace;
}
</style>
