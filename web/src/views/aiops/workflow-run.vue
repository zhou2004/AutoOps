<script setup lang="ts">
import {
  computed,
  nextTick,
  onDeactivated,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref
} from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import {
  diagnose,
  getCatalog,
  moduleLabels,
  statusText,
  type Catalog,
  type DiagnoseRequest
} from "@/api/aiops";
import { locale, tr } from "@/locales";
import "./style.css";

const router = useRouter();
const catalog = ref<Catalog>({ admin: false, modules: [], configs: [] });
const loading = ref(true),
  running = ref(false),
  error = ref("");
const output = ref(""),
  runId = ref(0),
  status = ref("");
const evidence = ref<Record<string, unknown> | null>(null);
const steps = ref<{ name: string; status: string }[]>([]);
const resultPanel = ref<HTMLElement>();
const followOutput = ref(true);
let controller: AbortController | undefined;
const form = reactive<DiagnoseRequest>({
  modelId: 0,
  skillId: 0,
  workflowId: 0,
  prompt: "",
  modules: ["cmdb", "k8s", "monitor"],
  days: 7,
  locale: "zh-CN"
});
const models = computed(() =>
  catalog.value.configs.filter(c => c.kind === "model" && c.enabled)
);
const skills = computed(() =>
  catalog.value.configs.filter(c => c.kind === "skill" && c.enabled)
);
const workflows = computed(() =>
  catalog.value.configs.filter(c => c.kind === "workflow" && c.enabled)
);
const external = computed(() =>
  workflows.value.some(
    w => w.id === form.workflowId && w.provider !== "internal"
  )
);
const noProvider = computed(
  () =>
    models.value.length === 0 &&
    !workflows.value.some(w => w.provider !== "internal")
);
async function load() {
  loading.value = true;
  error.value = "";
  try {
    catalog.value = await getCatalog();
    form.modelId = models.value[0]?.id || 0;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}
function includeSkill(id: number) {
  const skill = skills.value.find(s => s.id === id);
  if (skill) form.modules = [...new Set([...form.modules, ...skill.modules])];
}
function includeWorkflow(id: number) {
  const workflow = workflows.value.find(w => w.id === id);
  workflow?.steps?.forEach(step => includeSkill(step.skillId));
}
async function start() {
  if (!form.prompt.trim() || !form.modules.length)
    return ElMessage.warning(tr("请输入问题并选择模块"));
  if (!external.value && !form.modelId)
    return ElMessage.warning(tr("请选择模型或外部工作流"));
  running.value = true;
  output.value = "";
  error.value = "";
  evidence.value = null;
  steps.value = [];
  runId.value = 0;
  status.value = "running";
  followOutput.value = true;
  controller = new AbortController();
  try {
    await diagnose(
      { ...form, locale: locale.value },
      controller.signal,
      event => {
        if (event.runId) runId.value = event.runId;
        if (event.type === "delta") {
          output.value += event.text || "";
          if (followOutput.value)
            nextTick(() => {
              if (resultPanel.value)
                resultPanel.value.scrollTop = resultPanel.value.scrollHeight;
            });
        }
        if (event.type === "evidence") evidence.value = event.data || {};
        if (event.type === "step" && event.step) {
          const existing = steps.value.find(
            s => s.name === event.step && s.status === "running"
          );
          if (existing) existing.status = event.status || "running";
          else
            steps.value.push({
              name: event.step,
              status: event.status || "running"
            });
        }
        if (event.type === "done") status.value = "succeeded";
      }
    );
  } catch (e) {
    if (controller.signal.aborted) {
      status.value = "cancelled";
    } else {
      status.value = "failed";
      error.value = (e as Error).message;
    }
    steps.value.forEach(s => {
      if (s.status === "running") s.status = status.value;
    });
  } finally {
    running.value = false;
    controller = undefined;
  }
}
function trackScroll() {
  const el = resultPanel.value;
  if (el)
    followOutput.value = el.scrollHeight - el.scrollTop - el.clientHeight < 70;
}
async function copy() {
  try {
    await navigator.clipboard.writeText(output.value);
    ElMessage.success(tr("复制成功"));
  } catch {
    ElMessage.error(tr("操作失败"));
  }
}
function stepName(name: string) {
  return name === "Collect evidence"
    ? tr("采集业务快照")
    : name === "Diagnosis"
      ? tr("生成诊断")
      : name;
}
onMounted(load);
onBeforeUnmount(() => controller?.abort());
onDeactivated(() => controller?.abort());
</script>
<template>
  <div v-loading="loading" class="ai-page">
    <header class="ai-header">
      <div>
        <div class="ai-eyebrow">AUTOOPS / AI OPERATIONS</div>
        <h1>{{ tr("诊断工作台") }}</h1>
        <p>{{ tr("跨模块分析，基于真实业务快照") }}</p>
      </div>
      <el-tag effect="plain" type="success">{{ tr("只读诊断") }}</el-tag>
    </header>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      show-icon
      :closable="false"
      class="ai-alert"
    />
    <el-alert
      v-if="!loading && noProvider"
      :title="tr('尚未配置模型')"
      :description="tr('请管理员先添加模型，或配置外部工作流。')"
      type="warning"
      :closable="false"
      show-icon
      class="ai-alert"
    >
      <template v-if="catalog.admin" #default
        ><p>{{ tr("请管理员先添加模型，或配置外部工作流。") }}</p>
        <el-button link type="primary" @click="router.push('/aiops/models')">{{
          tr("前往模型配置")
        }}</el-button></template
      >
    </el-alert>
    <div class="ai-workspace">
      <section class="ai-card ai-controls">
        <h2>{{ tr("跨模块分析，基于真实业务快照") }}</h2>
        <el-form label-position="top" :disabled="running">
          <el-form-item :label="tr('工作流')"
            ><el-select v-model="form.workflowId" @change="includeWorkflow"
              ><el-option :label="tr('直接诊断')" :value="0" /><el-option
                v-for="w in workflows"
                :key="w.id"
                :label="w.name"
                :value="w.id" /></el-select
          ></el-form-item>
          <el-form-item v-if="!external" :label="tr('模型')"
            ><el-select v-model="form.modelId" :placeholder="tr('选择模型')"
              ><el-option
                v-for="m in models"
                :key="m.id"
                :label="`${m.name} · ${m.model}`"
                :value="m.id" /></el-select
          ></el-form-item>
          <el-form-item label="Skill"
            ><el-select v-model="form.skillId" @change="includeSkill"
              ><el-option :label="tr('通用诊断')" :value="0" /><el-option
                v-for="s in skills"
                :key="s.id"
                :label="s.name"
                :value="s.id" /></el-select
          ></el-form-item>
          <el-form-item :label="tr('诊断模块')"
            ><el-checkbox-group v-model="form.modules" class="ai-modules"
              ><el-checkbox v-for="m in catalog.modules" :key="m" :value="m">{{
                tr(moduleLabels[m])
              }}</el-checkbox></el-checkbox-group
            ></el-form-item
          >
          <el-form-item :label="tr('统计范围')"
            ><el-radio-group v-model="form.days"
              ><el-radio-button :value="7">{{ tr("近 7 天") }}</el-radio-button
              ><el-radio-button :value="30">{{
                tr("近 30 天")
              }}</el-radio-button
              ><el-radio-button :value="90">{{
                tr("近 90 天")
              }}</el-radio-button></el-radio-group
            ></el-form-item
          >
          <el-form-item :label="tr('诊断问题')"
            ><el-input
              v-model="form.prompt"
              type="textarea"
              :rows="5"
              :maxlength="4000"
              show-word-limit
              :placeholder="tr('描述需要排查的问题、现象或关注点…')"
          /></el-form-item>
          <el-button
            link
            type="primary"
            @click="
              form.prompt = tr(
                '检查选定模块的运行状况，识别风险，按优先级给出处理建议和验证步骤。'
              )
            "
            >{{ tr("使用巡检示例") }}</el-button
          >
        </el-form>
        <p class="ai-hint">
          {{
            tr(
              "诊断使用授权范围内的聚合业务数据，并发送至所选模型或工作流服务。"
            )
          }}
          {{ tr("不执行命令或修改业务资源。") }}
        </p>
        <el-button
          v-if="!running"
          type="primary"
          size="large"
          class="ai-start"
          :disabled="loading || noProvider"
          @click="start"
          >{{ tr("开始诊断") }} →</el-button
        >
        <el-button
          v-else
          type="danger"
          plain
          size="large"
          class="ai-start"
          @click="controller?.abort()"
          >{{ tr("停止生成") }}</el-button
        >
      </section>
      <section class="ai-card ai-result">
        <div class="ai-section-head">
          <h2>
            {{ tr("诊断结果") }} <small v-if="runId">#{{ runId }}</small>
          </h2>
          <div>
            <el-tag
              v-if="status"
              :type="
                status === 'failed'
                  ? 'danger'
                  : status === 'succeeded'
                    ? 'success'
                    : 'info'
              "
              >{{ statusText(status) }}</el-tag
            >
            <el-button text :disabled="!output" @click="copy">{{
              tr("复制结果")
            }}</el-button>
          </div>
        </div>
        <div v-if="steps.length" class="ai-steps">
          <span
            v-for="(step, i) in steps"
            :key="i"
            :class="['ai-step', step.status]"
            >{{
              step.status === "succeeded"
                ? "✓"
                : step.status === "running"
                  ? "◌"
                  : "×"
            }}
            {{ stepName(step.name) }}</span
          >
        </div>
        <div
          ref="resultPanel"
          class="ai-output"
          role="log"
          :aria-label="tr('诊断结果')"
          :aria-busy="running"
          @scroll="trackScroll"
        >
          <pre
            v-if="
              output
            ">{{ output }}<span v-if="running" class="ai-cursor">▍</span></pre>
          <el-empty
            v-else
            :description="
              tr(
                running
                  ? '处理中'
                  : '输入问题后开始诊断，结果将逐步显示在这里。'
              )
            "
            :image-size="100"
          />
        </div>
        <el-collapse v-if="evidence" class="ai-evidence"
          ><el-collapse-item :title="tr('查看本次使用的数据')" name="evidence">
            <pre>{{ JSON.stringify(evidence, null, 2) }}</pre>
          </el-collapse-item></el-collapse
        >
      </section>
    </div>
  </div>
</template>
