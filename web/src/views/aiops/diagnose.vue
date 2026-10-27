<script setup lang="ts">
import { tr } from "@/locales";
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref
} from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import {
  Plus,
  ChatDotRound,
  Promotion,
  VideoPause,
  Setting,
  Lock,
  Connection,
  Cpu,
  Timer,
  Document,
  Refresh,
  ArrowRight,
  Close,
  Check,
  Key,
  Operation,
  Monitor,
  Search,
  CopyDocument
} from "@element-plus/icons-vue";
import {
  getCatalog,
  getSessions,
  getSession,
  createSession,
  grantSession,
  getTargets,
  chat,
  type Catalog,
  type AISession,
  type AIMessage,
  type AIToolCall,
  type AITarget
} from "@/api/aiops";
import "./workbench.css";
const router = useRouter();
const catalog = ref<Catalog>({ admin: false, modules: [], configs: [] });
const sessions = ref<AISession[]>([]),
  active = ref<AISession | null>(null),
  messages = ref<AIMessage[]>([]),
  calls = ref<AIToolCall[]>([]),
  targets = ref<AITarget[]>([]);
const loading = ref(true),
  running = ref(false),
  error = ref(""),
  prompt = ref(""),
  phase = ref(""),
  output = ref(""),
  query = ref(""),
  panel = ref(false),
  granting = ref(false);
const form = reactive({ modelId: 0, skillId: 0, targets: [] as number[] });
const grants = reactive({ actions: [] as string[], minutes: 60 });
const feed = ref<HTMLElement>();
let abort: AbortController | undefined,
  polling: ReturnType<typeof setInterval> | undefined;
const models = computed(() =>
  catalog.value.configs.filter(c => c.kind === "model" && c.enabled)
);
const skills = computed(() =>
  catalog.value.configs.filter(c => c.kind === "skill" && c.enabled)
);
const model = computed(() =>
  models.value.find(c => c.id === (active.value?.modelId || form.modelId))
);
const harness = computed(() => model.value?.provider === "harness");
const filtered = computed(() =>
  sessions.value.filter(s =>
    s.title.toLowerCase().includes(query.value.toLowerCase())
  )
);
const selected = computed(() =>
  active.value ? JSON.parse(active.value.targets || "[]") || [] : form.targets
);
const canGrant = computed(
  () =>
    selected.value.length > 0 &&
    selected.value.every(
      (id: number) => targets.value.find(t => t.id === id)?.canOperate
    )
);
const authorized = computed(
  () =>
    !!active.value?.actions &&
    !!active.value.grantUntil &&
    new Date(active.value.grantUntil).getTime() > Date.now()
);
const toolLabels: Record<string, string> = {
  docker_info: "Docker 引擎状态",
  docker_list: "读取容器列表",
  docker_logs: "读取容器日志",
  docker_start: "启动容器",
  docker_stop: "停止容器",
  docker_restart: "重启容器"
};
const statusLabels: Record<string, string> = {
  running: "执行中",
  succeeded: "已完成",
  failed: "失败",
  interrupted: "已中断"
};
const starters = [
  {
    icon: Monitor,
    title: "检查容器状态",
    text: "检查所选节点上的容器状态，找出异常退出或反复重启的容器。"
  },
  {
    icon: Document,
    title: "分析故障日志",
    text: "先列出所选节点的容器，再结合异常容器最近的日志分析原因。"
  },
  {
    icon: Operation,
    title: "制定恢复方案",
    text: "分析当前故障，给出恢复步骤、影响范围和验证方法，执行前说明所需权限。"
  }
];
async function scroll() {
  await nextTick();
  if (feed.value) feed.value.scrollTop = feed.value.scrollHeight;
}
async function load() {
  loading.value = true;
  error.value = "";
  const results = await Promise.allSettled([
    getCatalog(),
    getSessions(),
    getTargets()
  ]);
  if (results[0].status === "fulfilled") {
    catalog.value = results[0].value;
    form.modelId =
      models.value.find(m => m.provider === "harness")?.id ||
      models.value[0]?.id ||
      0;
  } else error.value = results[0].reason.message;
  if (results[1].status === "fulfilled") sessions.value = results[1].value;
  else error.value = results[1].reason.message;
  if (results[2].status === "fulfilled") targets.value = results[2].value;
  else error.value = results[2].reason.message;
  loading.value = false;
}
function reset() {
  if (running.value) return;
  active.value = null;
  messages.value = [];
  calls.value = [];
  output.value = "";
  error.value = "";
  prompt.value = "";
  grants.actions = [];
}
async function openSession(row: AISession) {
  if (running.value) return;
  try {
    const data = await getSession(row.id);
    active.value = data.session;
    messages.value = data.messages.sort((a, b) => a.id - b.id);
    calls.value = data.calls;
    form.modelId = row.modelId;
    form.skillId = row.skillId;
    form.targets = JSON.parse(row.targets || "[]") || [];
    grants.actions = row.actions ? row.actions.split(",") : [];
    error.value = "";
    await scroll();
  } catch (e) {
    ElMessage.error((e as Error).message);
  }
}
async function refreshCalls() {
  if (!active.value) return;
  const id = active.value.id;
  try {
    const data = await getSession(id);
    if (active.value?.id === id) {
      calls.value = data.calls;
      active.value = data.session;
    }
  } catch {
    /* The final reload reports a persistent network failure. */
  }
}
async function send() {
  if (running.value || !prompt.value.trim()) return;
  if (!model.value) return ElMessage.warning(tr("请先配置并选择模型"));
  if (harness.value && !catalog.value.harnessAvailable)
    return ElMessage.warning(tr("请先完成 Harness 桥接服务配置"));
  running.value = true;
  error.value = "";
  output.value = "";
  phase.value = "正在建立会话";
  const text = prompt.value.trim();
  abort = new AbortController();
  try {
    if (!active.value) {
      active.value = await createSession({ ...form, title: text.slice(0, 60) });
      sessions.value.unshift(active.value);
    }
    const id = active.value.id;
    messages.value.push({
      id: -Date.now(),
      sessionId: id,
      role: "user",
      content: text,
      status: "succeeded",
      createdAt: new Date().toISOString()
    });
    prompt.value = "";
    await scroll();
    polling = setInterval(refreshCalls, 2500);
    await chat(id, text, abort.signal, event => {
      if (event.type === "delta") {
        output.value += event.text || "";
        void scroll();
      }
      if (event.type === "step" || event.type === "tool")
        phase.value = event.text || "正在分析";
      if (event.type === "start")
        phase.value = harness.value
          ? "Harness 正在推理与调用工具"
          : "正在生成回复";
    });
  } catch (e) {
    error.value =
      (e as Error).name === "AbortError"
        ? "已停止本轮对话，已执行的操作请在执行记录中核实。"
        : (e as Error).message;
  } finally {
    running.value = false;
    if (polling) clearInterval(polling);
    polling = undefined;
    if (active.value) {
      try {
        const data = await getSession(active.value.id);
        active.value = data.session;
        messages.value = data.messages.sort((a, b) => a.id - b.id);
        calls.value = data.calls;
        output.value = "";
        sessions.value = await getSessions();
      } catch (e) {
        error.value = (e as Error).message;
      }
    }
    await scroll();
  }
}
async function saveGrant(revoke = false) {
  if (!active.value) return;
  granting.value = true;
  try {
    active.value = await grantSession(
      active.value.id,
      revoke ? [] : grants.actions,
      grants.minutes
    );
    grants.actions = active.value.actions
      ? active.value.actions.split(",")
      : [];
    ElMessage.success(revoke ? "已撤销会话操作授权" : "会话授权已更新");
  } catch (e) {
    ElMessage.error((e as Error).message);
  } finally {
    granting.value = false;
  }
}
async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    ElMessage.success(tr("已复制"));
  } catch {
    ElMessage.warning(tr("请选中文本手动复制"));
  }
}
function enter(event: KeyboardEvent) {
  if ((event.ctrlKey || event.metaKey) && event.key === "Enter") {
    event.preventDefault();
    void send();
  }
}
onMounted(load);
onBeforeUnmount(() => {
  abort?.abort();
  if (polling) clearInterval(polling);
});
</script>
<template>
  <div v-loading="loading" class="ops-workbench">
    <header class="ops-topbar">
      <div class="ops-brand">
        <span class="ops-brand-icon"
          ><el-icon><Connection /></el-icon
        ></span>
        <div>
          <span class="ops-eyebrow">AI OPERATIONS</span>
          <h1>{{ $tr("诊断工作台") }}</h1>
        </div>
        <el-tag effect="plain" round>Harness</el-tag>
      </div>
      <div class="ops-top-actions">
        <el-button
          :icon="Operation"
          @click="router.push('/aiops/workflow-run')"
          >{{ $tr("工作流诊断") }}</el-button
        ><el-button :icon="Key" @click="router.push('/system/api-keys')"
          >API Key</el-button
        ><el-button :icon="Setting" @click="router.push('/aiops/models')">{{
          $tr("模型配置")
        }}</el-button
        ><el-button
          class="ops-panel-toggle"
          :icon="Lock"
          @click="panel = !panel"
          >{{ $tr("上下文与权限") }}</el-button
        >
      </div>
    </header>
    <div class="ops-layout">
      <aside class="ops-sessions">
        <el-button
          type="primary"
          :icon="Plus"
          :disabled="running"
          @click="reset"
          >{{ $tr("新建会话") }}</el-button
        ><el-input
          v-model="query"
          :prefix-icon="Search"
          :placeholder="$tr('搜索会话')"
          clearable
        />
        <div class="ops-section-label">
          {{ $tr("最近会话") }}<span>{{ sessions.length }}</span>
        </div>
        <nav :aria-label="$tr('诊断会话')">
          <button
            v-for="row in filtered"
            :key="row.id"
            class="ops-session"
            :class="{ active: active?.id === row.id }"
            :disabled="running"
            @click="openSession(row)"
          >
            <el-icon><ChatDotRound /></el-icon
            ><span
              ><strong>{{ row.title }}</strong
              ><small>{{
                new Date(row.updatedAt).toLocaleDateString()
              }}</small></span
            >
          </button>
          <div v-if="!filtered.length" class="ops-sidebar-empty">
            {{ $tr("开始一次对话，诊断过程将保存在这里。") }}
          </div>
        </nav>
        <div class="ops-session-footer">
          <el-icon><Lock /></el-icon>{{ $tr("会话与权限按用户隔离") }}
        </div>
      </aside>
      <main class="ops-conversation">
        <div class="ops-conversation-head">
          <div>
            <span class="ops-status-dot" :class="{ running }" /><strong>{{
              active?.title || $tr("从一个运维问题开始")
            }}</strong>
          </div>
          <el-tag
            size="small"
            :type="authorized ? 'warning' : 'info'"
            effect="plain"
            >{{ authorized ? $tr("已授权操作") : $tr("只读分析") }}</el-tag
          >
        </div>
        <el-alert
          v-if="error"
          class="ops-error"
          :title="error"
          type="error"
          show-icon
          closable
          @close="error = ''"
        />
        <div ref="feed" class="ops-feed" aria-live="polite">
          <div v-if="!messages.length && !running" class="ops-welcome">
            <div class="ops-orbit">
              <el-icon><Connection /></el-icon><span />
            </div>
            <div class="ops-eyebrow">OBSERVE · REASON · ACT</div>
            <h2>{{ $tr("让诊断接上真实操作") }}</h2>
            <p>
              {{ $tr("选择模型与资产，用对话定位问题。") }}<br />{{
                $tr("Harness 可读取状态、分析日志，并执行你授权的操作。")
              }}
            </p>
            <div class="ops-starters">
              <button
                v-for="item in starters"
                :key="item.title"
                @click="prompt = $tr(item.text)"
              >
                <el-icon><component :is="item.icon" /></el-icon
                ><strong>{{ $tr(item.title) }}</strong
                ><span>{{ $tr(item.text) }}</span
                ><el-icon class="ops-starter-arrow"><ArrowRight /></el-icon>
              </button>
            </div>
          </div>
          <article
            v-for="message in messages"
            :key="message.id"
            class="ops-message"
            :class="message.role"
          >
            <div class="ops-message-avatar">
              <el-icon
                ><ChatDotRound v-if="message.role === 'user'" /><Cpu v-else
              /></el-icon>
            </div>
            <div class="ops-message-main">
              <div class="ops-message-label">
                <strong>{{
                  message.role === "user" ? "你" : "运维助手"
                }}</strong
                ><span>{{
                  new Date(message.createdAt).toLocaleTimeString()
                }}</span
                ><el-tag
                  v-if="message.status !== 'succeeded'"
                  size="small"
                  type="info"
                  >{{
                    $tr(statusLabels[message.status] || message.status)
                  }}</el-tag
                >
              </div>
              <div class="ops-message-text">
                {{
                  message.content ||
                  (message.status === "interrupted"
                    ? $tr("本轮回复已中断，可继续对话。")
                    : $tr("未生成回复。"))
                }}
              </div>
              <el-button
                v-if="message.content"
                text
                size="small"
                :icon="CopyDocument"
                @click="copy(message.content)"
                >{{ $tr("复制") }}</el-button
              >
            </div>
          </article>
          <article v-if="running || output" class="ops-message assistant">
            <div class="ops-message-avatar">
              <el-icon><Cpu /></el-icon>
            </div>
            <div class="ops-message-main">
              <div class="ops-message-label">
                <strong>{{ $tr("运维助手") }}</strong
                ><span class="ops-live">{{ phase }}</span>
              </div>
              <div v-if="output" class="ops-message-text">{{ output }}</div>
              <div v-else class="ops-thinking"><span /><span /><span /></div>
            </div>
          </article>
        </div>
        <div class="ops-composer">
          <div class="ops-composer-meta">
            <span
              ><el-icon><Cpu /></el-icon
              >{{ model?.name || $tr("尚未选择模型") }}</span
            ><span
              ><el-icon><Monitor /></el-icon>{{ selected.length
              }}{{ $tr("个资产连接") }}</span
            ><span v-if="harness">DeepSeek Harness</span>
          </div>
          <el-input
            v-model="prompt"
            type="textarea"
            :autosize="{ minRows: 3, maxRows: 7 }"
            resize="none"
            maxlength="16000"
            :placeholder="
              $tr('描述故障现象、需要检查的容器，或继续追问诊断结果…')
            "
            :aria-label="$tr('诊断问题')"
            @keydown="enter"
          />
          <div class="ops-composer-bottom">
            <small>{{
              $tr("Ctrl / ⌘ + Enter 发送 · 操作受当前会话授权约束")
            }}</small
            ><el-button
              v-if="running"
              :icon="VideoPause"
              @click="abort?.abort()"
              >{{ $tr("停止") }}</el-button
            ><el-button
              v-else
              type="primary"
              :icon="Promotion"
              :disabled="!prompt.trim() || !model"
              @click="send"
              >{{ $tr("发送") }}</el-button
            >
          </div>
        </div>
      </main>
      <aside class="ops-context" :class="{ 'is-open': panel }">
        <div class="ops-context-title">
          <strong>{{ $tr("上下文与权限") }}</strong
          ><el-button
            class="ops-panel-toggle"
            text
            :icon="Close"
            :aria-label="$tr('关闭权限面板')"
            @click="panel = false"
          />
        </div>
        <section>
          <div class="ops-section-label">
            <el-icon><Cpu /></el-icon>{{ $tr("运行引擎") }}
          </div>
          <el-select
            v-model="form.modelId"
            :disabled="!!active || running"
            :placeholder="$tr('选择模型')"
            class="ops-full"
            ><el-option
              v-for="m in models"
              :key="m.id"
              :value="m.id"
              :label="m.name"
          /></el-select>
          <div class="ops-engine-status">
            <span
              class="ops-status-dot"
              :class="{ configured: harness && catalog.harnessAvailable }"
            />{{
              harness
                ? catalog.harnessAvailable
                  ? $tr("Harness 桥接已配置")
                  : $tr("Harness 桥接待配置")
                : $tr("模型对话 · 不调用操作工具")
            }}
          </div>
          <el-select
            v-model="form.skillId"
            :disabled="!!active || running"
            :placeholder="$tr('诊断 Skill（可选）')"
            clearable
            class="ops-full"
            ><el-option :value="0" :label="$tr('不使用 Skill')" />
            <el-option
              v-for="skill in skills"
              :key="skill.id"
              :value="skill.id"
              :label="skill.name"
          /></el-select>
          <p v-if="!models.length" class="ops-help">
            {{ $tr("请在模型配置中添加连接，再开始对话。") }}
          </p>
        </section>
        <section>
          <div class="ops-section-label">
            <el-icon><Monitor /></el-icon>{{ $tr("资产范围") }}
          </div>
          <el-select
            v-model="form.targets"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            :disabled="!!active || running"
            :placeholder="$tr('选择可访问的资产')"
            class="ops-full"
            ><el-option
              v-for="target in targets"
              :key="target.id"
              :value="target.id"
              :label="`${target.name} · ${target.assetType} #${target.assetId}`"
          /></el-select>
          <p class="ops-help">
            {{
              $tr(
                "仅列出你拥有 Docker 与凭据权限的连接。开始会话后固定资产范围，新建会话可重新选择。"
              )
            }}
          </p>
          <div v-for="id in selected" :key="id" class="ops-target">
            <el-icon><Connection /></el-icon
            ><span>{{
              targets.find(t => t.id === id)?.name || `连接 #${id}`
            }}</span
            ><el-tag size="small" effect="plain">#{{ id }}</el-tag>
          </div>
        </section>
        <section>
          <div class="ops-section-label">
            <el-icon><Lock /></el-icon>{{ $tr("操作授权") }}
          </div>
          <el-checkbox-group
            v-model="grants.actions"
            class="ops-grants"
            :disabled="!active || !canGrant || !harness"
            ><el-checkbox value="docker_start">{{
              $tr("启动容器")
            }}</el-checkbox
            ><el-checkbox value="docker_stop">{{ $tr("停止容器") }}</el-checkbox
            ><el-checkbox value="docker_restart">{{
              $tr("重启容器")
            }}</el-checkbox></el-checkbox-group
          >
          <div class="ops-grant-time">
            <span>{{ $tr("有效期") }}</span
            ><el-select v-model="grants.minutes" size="small"
              ><el-option :label="$tr('15 分钟')" :value="15" /><el-option
                :label="$tr('1 小时')"
                :value="60" /><el-option :label="$tr('4 小时')" :value="240"
            /></el-select>
          </div>
          <div class="ops-grant-actions">
            <el-button
              :icon="Check"
              size="small"
              :loading="granting"
              :disabled="!active || !canGrant || !harness"
              @click="saveGrant()"
              >{{ $tr("应用授权") }}</el-button
            ><el-button
              v-if="active?.actions"
              :icon="Lock"
              size="small"
              type="danger"
              text
              :loading="granting"
              @click="saveGrant(true)"
              >{{ $tr("撤销") }}</el-button
            >
          </div>
          <p class="ops-help">
            {{
              !active
                ? $tr("先发送问题创建会话，再按需授权操作。")
                : !canGrant
                  ? $tr("所选连接不具备完整操作权限，当前只能读取。")
                  : authorized
                    ? `授权至 ${new Date(active!.grantUntil!).toLocaleString()}`
                    : $tr("未授予写操作权限。读取日志仍检查资产权限。")
            }}
          </p>
        </section>
        <section class="ops-tools">
          <div class="ops-section-label">
            <el-icon><Timer /></el-icon>{{ $tr("执行记录")
            }}<span>{{ calls.length }}</span
            ><el-button
              text
              :icon="Refresh"
              size="small"
              :aria-label="$tr('刷新执行记录')"
              :disabled="!active"
              @click="refreshCalls"
            />
          </div>
          <p v-if="!calls.length" class="ops-help">
            {{ $tr("工具调用后，这里会展示实际执行状态与返回结果。") }}
          </p>
          <details v-for="call in calls" :key="call.id" class="ops-tool">
            <summary>
              <el-icon><Document /></el-icon
              ><span>{{ $tr(toolLabels[call.tool] || call.tool) }}</span
              ><el-tag
                size="small"
                :type="
                  call.status === 'succeeded'
                    ? 'success'
                    : call.status === 'failed'
                      ? 'danger'
                      : 'info'
                "
                >{{ $tr(statusLabels[call.status] || call.status) }}</el-tag
              >
            </summary>
            <pre>{{ call.arguments }}</pre>
            <pre>{{ call.result || $tr("尚未返回执行结果") }}</pre>
          </details>
        </section>
      </aside>
    </div>
  </div>
</template>
