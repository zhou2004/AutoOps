<script setup lang="ts">
import * as OperationIcons from "@element-plus/icons-vue";
import { useRoute } from "vue-router";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  deleteConfig,
  getCatalog,
  moduleLabels,
  saveConfig,
  testModel,
  type AIConfig,
  type Catalog,
  type ConfigKind
} from "@/api/aiops";
import { tr } from "@/locales";
import "./style.css";
const route = useRoute();
const kind = computed<ConfigKind>(() =>
  route.path.endsWith("/skills")
    ? "skill"
    : route.path.endsWith("/workflows")
      ? "workflow"
      : "model"
);
const catalog = ref<Catalog>({ admin: false, modules: [], configs: [] });
const loading = ref(false),
  saving = ref(false),
  open = ref(false),
  error = ref("");
const testing = ref(0);
const title = computed(() =>
  tr(
    { model: "平台模型连接", skill: "诊断 Skills", workflow: "诊断工作流" }[
      kind.value
    ]
  )
);
const subtitle = computed(() =>
  tr(
    {
      model: "连接 DeepSeek Harness、OpenAI 兼容服务与本地 Ollama。",
      skill: "将运维经验整理为可复用的诊断指令。",
      workflow: "编排多步诊断，或连接 Dify 与 Webhook。"
    }[kind.value]
  )
);
const addLabel = computed(() =>
  tr(
    { model: "新增模型", skill: "新增 Skill", workflow: "新增工作流" }[
      kind.value
    ]
  )
);
const rows = computed(() =>
  catalog.value.configs.filter(c => c.kind === kind.value)
);
const skills = computed(() =>
  catalog.value.configs.filter(c => c.kind === "skill")
);
function empty(): AIConfig {
  return {
    id: 0,
    kind: kind.value,
    name: "",
    description: "",
    enabled: true,
    provider: kind.value === "workflow" ? "internal" : "openai",
    baseUrl: "",
    model: "",
    hasKey: false,
    apiKey: "",
    clearKey: false,
    instructions: "",
    modules: ["cmdb", "monitor"],
    steps: [{ name: "", skillId: 0, instructions: "" }],
    timeoutSeconds: 180
  };
}
const form = reactive<AIConfig>(empty());
async function load() {
  loading.value = true;
  error.value = "";
  try {
    catalog.value = await getCatalog();
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}
function edit(row?: AIConfig) {
  Object.assign(form, row ? JSON.parse(JSON.stringify(row)) : empty());
  form.apiKey = "";
  form.clearKey = false;
  if (!form.steps) form.steps = [];
  if (!form.modules) form.modules = [];
  open.value = true;
}
async function save() {
  if (!form.name.trim()) return ElMessage.warning(tr("请输入名称"));
  if (kind.value === "model" && !form.model.trim())
    return ElMessage.warning(tr("请输入模型 ID"));
  if (
    (kind.value === "model" ||
      (kind.value === "workflow" && form.provider !== "internal")) &&
    !form.baseUrl.trim()
  )
    return ElMessage.warning(tr("请输入接口地址"));
  saving.value = true;
  try {
    await saveConfig({ ...form });
    open.value = false;
    form.apiKey = "";
    ElMessage.success(tr("配置已保存"));
    await load();
  } catch (e) {
    ElMessage.error((e as Error).message);
  } finally {
    saving.value = false;
  }
}
async function remove(row: AIConfig) {
  try {
    await ElMessageBox.confirm(tr("确认删除该配置？"), row.name, {
      type: "warning",
      confirmButtonText: tr("删除"),
      cancelButtonText: tr("取消")
    });
  } catch {
    return;
  }
  try {
    await deleteConfig(row.id);
    ElMessage.success(tr("配置已删除"));
    await load();
  } catch (e) {
    ElMessage.error((e as Error).message);
  }
}
async function test(row: AIConfig) {
  testing.value = row.id;
  try {
    await testModel(row.id);
    ElMessage.success(tr("连接测试成功"));
  } catch (e) {
    ElMessage.error((e as Error).message);
  } finally {
    testing.value = 0;
  }
}
function move(index: number, delta: number) {
  const steps = form.steps;
  [steps[index], steps[index + delta]] = [steps[index + delta], steps[index]];
}
function protocol(value: string) {
  return (
    {
      openai: "OpenAI Compatible",
      ollama: "Ollama",
      internal: tr("内部多步骤"),
      dify: "Dify",
      webhook: "Webhook"
    }[value] || value
  );
}
watch(
  () => kind.value,
  () => {
    open.value = false;
    load();
  }
);
onMounted(load);
</script>
<template>
  <div v-loading="loading" class="ai-page">
    <header class="ai-header">
      <div>
        <div class="ai-eyebrow">AUTOOPS / AI OPERATIONS</div>
        <h1>{{ title }}</h1>
        <p>{{ subtitle }}</p>
      </div>
      <div>
        <el-button :icon="OperationIcons.Refresh" @click="load">{{
          tr("刷新")
        }}</el-button
        ><el-button
          :icon="OperationIcons.Operation"
          v-if="catalog.admin"
          type="primary"
          @click="edit()"
          >+ {{ addLabel }}</el-button
        >
      </div>
    </header>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      class="ai-alert"
    />
    <el-alert
      v-if="!loading && !catalog.admin"
      :title="tr('仅管理员可以管理配置')"
      type="info"
      :closable="false"
      class="ai-alert"
    />
    <section class="ai-card">
      <el-table :data="rows" row-key="id">
        <el-table-column prop="name" :label="tr('名称')" min-width="170"
          ><template #default="{ row }"
            ><strong>{{ row.name }}</strong>
            <p class="ai-hint">{{ row.description }}</p></template
          ></el-table-column
        >
        <el-table-column
          v-if="kind !== 'skill'"
          :label="tr('协议')"
          min-width="150"
          ><template #default="{ row }"
            >{{ protocol(row.provider) }}
            <p v-if="row.model" class="ai-hint">{{ row.model }}</p></template
          ></el-table-column
        >
        <el-table-column
          v-if="kind === 'skill'"
          :label="tr('需要的模块')"
          min-width="220"
          ><template #default="{ row }"
            ><el-tag
              v-for="m in row.modules"
              :key="m"
              class="ai-inline-tag"
              size="small"
              >{{ tr(moduleLabels[m]) }}</el-tag
            ></template
          ></el-table-column
        >
        <el-table-column :label="tr('状态')" width="120"
          ><template #default="{ row }"
            ><el-tag :type="row.enabled ? 'success' : 'info'">{{
              tr(row.enabled ? "启用" : "停用")
            }}</el-tag></template
          ></el-table-column
        >
        <el-table-column
          v-if="catalog.admin"
          :label="tr('操作')"
          :min-width="kind === 'model' ? 250 : 160"
          fixed="right"
          ><template #default="{ row }"
            ><el-button
              :icon="OperationIcons.Edit"
              link
              type="primary"
              @click="edit(row)"
              >{{ tr("编辑") }}</el-button
            ><el-button
              :icon="OperationIcons.Connection"
              v-if="kind === 'model'"
              link
              type="success"
              :loading="testing === row.id"
              :disabled="testing !== 0 || !row.enabled"
              @click="test(row)"
              >{{ tr("连接测试") }}</el-button
            ><el-button
              :icon="OperationIcons.Delete"
              link
              type="danger"
              @click="remove(row)"
              >{{ tr("删除") }}</el-button
            ></template
          ></el-table-column
        >
        <template #empty><el-empty :description="tr('暂无数据')" /></template>
      </el-table>
      <p v-if="kind === 'model'" class="ai-hint">
        {{ tr("测试会发送简短请求，可能产生模型费用。") }}
      </p>
    </section>
    <el-dialog
      v-model="open"
      :title="tr(form.id ? '编辑配置' : '新增配置')"
      width="760px"
      class="ai-dialog"
      :close-on-click-modal="false"
      :close-on-press-escape="!saving"
      :show-close="!saving"
      destroy-on-close
    >
      <el-form label-position="top" :disabled="saving" @submit.prevent="save">
        <div class="ai-form-grid">
          <el-form-item :label="tr('配置名称')" required
            ><el-input
              v-model="form.name"
              :maxlength="100"
              :placeholder="tr('请输入名称')" /></el-form-item
          ><el-form-item :label="tr('状态')"
            ><el-switch
              v-model="form.enabled"
              :active-text="tr('启用')"
              :inactive-text="tr('停用')"
          /></el-form-item>
        </div>
        <el-form-item :label="tr('描述')"
          ><el-input v-model="form.description" :maxlength="300"
        /></el-form-item>
        <template v-if="kind !== 'skill'">
          <el-alert
            v-if="form.provider === 'harness'"
            type="info"
            :closable="false"
            show-icon
            :title="
              $tr(
                'Harness 使用官方 SDK 桥接服务运行，接口地址填写 DeepSeek Messages 根地址（https://api.deepseek.com/anthropic）。资产与操作权限在诊断会话中授予。'
              )
            "
          />
          <el-form-item :label="tr('协议')"
            ><el-select v-model="form.provider"
              ><template v-if="kind === 'model'"
                ><el-option
                  label="DeepSeek Harness"
                  value="harness" /><el-option
                  label="OpenAI Compatible"
                  value="openai" /><el-option
                  label="Ollama"
                  value="ollama" /></template
              ><template v-else
                ><el-option
                  :label="tr('内部多步骤')"
                  value="internal" /><el-option
                  label="Dify"
                  value="dify" /><el-option
                  label="Webhook"
                  value="webhook" /></template></el-select
          ></el-form-item>
          <template v-if="form.provider !== 'internal'">
            <el-form-item :label="tr('接口地址')" required
              ><el-input
                v-model="form.baseUrl"
                :placeholder="
                  form.provider === 'harness'
                    ? 'https://api.deepseek.com/anthropic'
                    : form.provider === 'ollama'
                      ? 'http://127.0.0.1:11434'
                      : form.provider === 'webhook'
                        ? 'https://example.com/workflow'
                        : 'https://example.com/v1'
                "
            /></el-form-item>
            <p class="ai-hint">
              {{
                tr(
                  kind === "model"
                    ? "OpenAI 兼容地址通常以 /v1 结尾；Ollama 填写服务根地址。"
                    : "Dify 填写 /v1 地址，工作流需定义 prompt、context、locale 三个文本输入。Webhook 填写完整 URL。"
                )
              }}
            </p>
            <el-form-item
              v-if="kind === 'model'"
              :label="tr('模型 ID')"
              required
              ><el-input
                v-model="form.model"
                :maxlength="200"
                :placeholder="tr('请输入模型 ID')"
            /></el-form-item>
            <el-form-item :label="tr('API 密钥')"
              ><el-input
                v-model="form.apiKey"
                type="password"
                show-password
                autocomplete="new-password"
                :placeholder="
                  tr(
                    form.hasKey
                      ? '已保存，留空保持不变'
                      : '可选，本地模型可留空'
                  )
                "
              /><el-checkbox v-if="form.hasKey" v-model="form.clearKey">{{
                tr("清除已保存密钥")
              }}</el-checkbox></el-form-item
            >
          </template>
          <el-form-item :label="tr('超时（秒）')"
            ><el-input-number
              v-model="form.timeoutSeconds"
              :min="10"
              :max="600"
          /></el-form-item>
        </template>
        <template v-if="kind === 'skill'">
          <el-form-item :label="tr('需要的模块')" required
            ><el-checkbox-group v-model="form.modules" class="ai-modules"
              ><el-checkbox v-for="m in catalog.modules" :key="m" :value="m">{{
                tr(moduleLabels[m])
              }}</el-checkbox></el-checkbox-group
            ></el-form-item
          >
          <el-form-item :label="tr('诊断指令')" required
            ><el-input
              v-model="form.instructions"
              type="textarea"
              :rows="8"
              :maxlength="4000"
              show-word-limit
              :placeholder="tr('例如：分析异常数量、给出可能原因和验证步骤。')"
          /></el-form-item>
          <el-button
            :icon="OperationIcons.Operation"
            link
            type="primary"
            @click="
              form.instructions = tr(
                '请基于业务快照区分事实与推测，列出异常、缺失数据和下一步检查建议。'
              )
            "
            >{{ tr("使用巡检示例") }}</el-button
          >
        </template>
        <template v-if="kind === 'workflow' && form.provider === 'internal'">
          <p class="ai-hint">
            {{
              tr(
                "内部工作流按顺序运行，每一步可引用 Skill，后续步骤会收到上一步结果。"
              )
            }}
          </p>
          <div
            v-for="(step, i) in form.steps"
            :key="i"
            class="ai-workflow-step"
          >
            <div class="ai-section-head">
              <strong>{{ tr("步骤") }} {{ i + 1 }}</strong>
              <div>
                <el-button
                  :icon="OperationIcons.Operation"
                  text
                  :disabled="i === 0"
                  @click="move(i, -1)"
                  >{{ tr("移上") }}</el-button
                ><el-button
                  :icon="OperationIcons.Operation"
                  text
                  :disabled="i === form.steps.length - 1"
                  @click="move(i, 1)"
                  >{{ tr("移下") }}</el-button
                ><el-button
                  :icon="OperationIcons.Delete"
                  text
                  type="danger"
                  @click="form.steps.splice(i, 1)"
                  >{{ tr("删除") }}</el-button
                >
              </div>
            </div>
            <el-form-item :label="tr('步骤名称')" required
              ><el-input v-model="step.name" :maxlength="100"
            /></el-form-item>
            <el-form-item label="Skill"
              ><el-select v-model="step.skillId"
                ><el-option :label="tr('无')" :value="0" /><el-option
                  v-for="s in skills"
                  :key="s.id"
                  :label="s.name"
                  :value="s.id"
                  :disabled="!s.enabled" /></el-select
            ></el-form-item>
            <el-form-item :label="tr('步骤指令')" :required="!step.skillId"
              ><el-input
                v-model="step.instructions"
                type="textarea"
                :rows="3"
                :maxlength="2000"
            /></el-form-item>
          </div>
          <el-button
            :icon="OperationIcons.Plus"
            :disabled="form.steps.length >= 8"
            @click="form.steps.push({ name: '', skillId: 0, instructions: '' })"
            >+ {{ tr("添加步骤") }}</el-button
          >
        </template>
      </el-form>
      <template #footer
        ><el-button
          :icon="OperationIcons.Close"
          :disabled="saving"
          @click="open = false"
          >{{ tr("取消") }}</el-button
        ><el-button
          :icon="OperationIcons.DocumentChecked"
          type="primary"
          :loading="saving"
          @click="save"
          >{{ tr("保存") }}</el-button
        ></template
      >
    </el-dialog>
  </div>
</template>
