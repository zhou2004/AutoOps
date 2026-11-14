<template>
  <section class="web-logs">
    <div class="log-filters">
      <el-form label-position="top" @submit.prevent="search">
        <div class="log-filter-grid">
          <el-form-item :label="$tr('实例')">
            <el-select
              v-model="instanceId"
              filterable
              :placeholder="$tr('选择实例')"
            >
              <el-option
                v-for="i in availableInstances"
                :key="i.id"
                :value="i.id"
                :label="i.name"
              />
            </el-select>
          </el-form-item>
          <el-form-item :label="$tr('日志文件')" class="log-file-filter">
            <el-select
              v-model="logId"
              filterable
              :loading="loadingLogs"
              :disabled="!instanceId || loadingLogs"
              :placeholder="$tr('选择日志')"
              @change="changeLog"
            >
              <el-option-group
                v-for="group in logGroups"
                :key="group.kind"
                :label="$tr(group.label)"
              >
                <el-option
                  v-for="log in group.items"
                  :key="log.id"
                  :value="log.id"
                  :label="log.path"
                />
              </el-option-group>
            </el-select>
          </el-form-item>
        </div>
        <div class="log-actions">
          <el-button
            type="primary"
            :disabled="!logId || loadingLogs"
            :loading="busy"
            @click="search"
            >{{ $tr("查询") }}</el-button
          >
          <el-button :aria-expanded="advanced" @click="advanced = !advanced"
            >{{ $tr(advanced ? "收起高级筛选" : "高级筛选")
            }}<el-tag
              v-if="activeFilterCount"
              size="small"
              class="filter-count"
              >{{ activeFilterCount }}</el-tag
            ></el-button
          >
          <el-switch
            v-model="follow"
            :disabled="!logId || loadingLogs || !!rotation || !!from || !!to"
            :active-text="$tr('实时追踪')"
          />
          <el-tag :type="streamState === 'live' ? 'success' : 'info'">{{
            $tr(
              streamState === "connecting"
                ? "正在连接"
                : streamState === "live"
                  ? "SSE 实时接收"
                  : "追踪已暂停"
            )
          }}</el-tag>
        </div>
        <div v-if="advanced" class="log-filter-grid advanced-filters">
          <el-form-item :label="$tr('站点')">
            <el-select
              v-model="siteId"
              clearable
              filterable
              :placeholder="$tr('全部站点')"
            >
              <el-option
                v-for="site in sites"
                :key="site.id"
                :value="site.id"
                :label="site.name"
              />
            </el-select>
          </el-form-item>
          <el-form-item :label="$tr('关键词')" class="log-keyword-filter">
            <el-input
              v-model="keyword"
              clearable
              :placeholder="$tr('关键词')"
              @keyup.enter="search"
            />
          </el-form-item>
          <el-form-item :label="$tr('每次扫描行数')">
            <el-select v-model="lines" :aria-label="$tr('每次扫描行数')">
              <el-option
                v-for="value in [100, 500, 1000, 5000, 10000]"
                :key="value"
                :value="value"
                :label="String(value)"
              />
            </el-select>
          </el-form-item>

          <el-form-item :label="$tr('轮转文件')">
            <el-input
              v-model="rotation"
              :disabled="logId === 'stdout'"
              clearable
              :placeholder="$tr('轮转后缀，例如 .1 或 .2.gz')"
            />
          </el-form-item>
          <el-form-item :label="$tr('HTTP 状态码')"
            ><el-input v-model="status" clearable placeholder="200 / 404 / 500"
          /></el-form-item>
          <el-form-item :label="$tr('来源 IP')"
            ><el-input v-model="ip" clearable
          /></el-form-item>
          <el-form-item label="URI"
            ><el-input v-model="uri" clearable placeholder="/api/"
          /></el-form-item>
          <el-form-item :label="$tr('时间范围')" class="log-time-range">
            <div class="date-range-inputs">
              <el-date-picker
                v-model="from"
                type="datetime"
                :placeholder="$tr('开始时间')"
              />
              <span>—</span>
              <el-date-picker
                v-model="to"
                type="datetime"
                :placeholder="$tr('结束时间')"
              />
            </div>
          </el-form-item>
        </div>
      </el-form>
    </div>
    <p v-if="streamNotice" class="log-stream-note" role="status">
      {{ streamNotice }}
    </p>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      show-icon
    />
    <el-alert
      v-if="siteNotice"
      :title="siteNotice"
      type="warning"
      :closable="false"
      show-icon
    />
    <el-alert
      v-else-if="selectedSite"
      :title="$tr('按站点登记的日志路径筛选；共享日志文件可能包含其他站点记录')"
      type="info"
      :closable="false"
    />
    <el-empty
      v-if="!instanceId"
      :description="$tr('选择实例后查看访问日志、错误日志或容器输出')"
      :image-size="90"
    />
    <el-empty
      v-else-if="!loadingLogs && !visibleLogs.length && !error"
      :description="
        $tr(
          siteId
            ? '没有匹配站点的日志文件'
            : '该实例尚未发现可读取的日志，请检查日志配置并重新发现实例'
        )
      "
      :image-size="90"
    />
    <template v-else>
      <div class="log-result-header">
        <div class="log-result-meta">
          <span v-if="at">{{ $tr("查询时间") }}：{{ at }}</span>
          <span v-if="lastQuery && !resultStreaming">{{
            $tr("匹配 {value0} 行 · 扫描偏移 {value1} · 扫描窗口 {value2} 行", {
              value0: matchedLines,
              value1: resultOffset,
              value2: resultWindow
            })
          }}</span>
        </div>
        <div class="log-actions result-actions">
          <el-tooltip
            :content="
              $tr(
                '查询节点实际保留的日志；每次仅扫描指定窗口，时间与状态筛选仅对可识别格式生效。'
              )
            "
            placement="top"
            :show-after="200"
          >
            <el-button text size="small">{{ $tr("查询说明") }}</el-button>
          </el-tooltip>
          <el-switch v-model="wrap" :active-text="$tr('自动换行')" />
          <el-button :disabled="!text" @click="copyResult">{{
            $tr("复制")
          }}</el-button>
          <el-button
            :disabled="!lastQuery || downloading"
            :loading="downloading"
            @click="download"
            >{{ $tr("下载当前结果") }}</el-button
          >
        </div>
      </div>
      <div
        ref="consoleHost"
        class="log-console"
        :style="{ height: consoleHeight + 'px' }"
        v-loading="busy || loadingLogs"
      >
        <LogConsole
          :text="text"
          :title="path || selectedLog?.path"
          :follow="follow"
          :wrap="wrap"
        />
      </div>
      <div v-if="!resultStreaming" class="log-navigation">
        <el-button
          :disabled="!resultOffset || busy || follow"
          @click="navigate(0)"
          >{{ $tr("最新日志") }}</el-button
        >
        <el-button
          :disabled="!resultOffset || busy || follow"
          @click="navigate(Math.max(0, resultOffset - resultWindow))"
          >{{ $tr("较新日志") }}</el-button
        >
        <el-tag v-if="lastQuery" type="info">{{
          $tr("第 {value0} 个扫描窗口", {
            value0: Math.floor(resultOffset / resultWindow) + 1
          })
        }}</el-tag>
        <el-button
          :disabled="!hasMore || busy || follow"
          @click="navigate(nextOffset)"
          >{{ $tr("更早日志") }}</el-button
        >
      </div>
      <p v-if="lastQuery && !resultStreaming" class="log-scan-note">
        {{
          $tr(
            "单次历史检索最多扫描最近 {value0} 行；翻页范围按扫描行数计算，不代表匹配记录总数。",
            { value0: scanLimit }
          )
        }}
      </p>
    </template>
  </section>
</template>
<script setup lang="ts">
import { computed, nextTick, ref, watch, onMounted, onUnmounted } from "vue";
import LogConsole from "@/components/LogConsole/index.vue";
import { webLogStream, appendLogText } from "@/api/webLogStream";
import { ElMessage } from "element-plus";
import { useClipboard } from "@vueuse/core";
import { tr } from "@/locales";
import { webRequest, webLogQuery, type WebResource } from "@/api/webservice";
const props = withDefaults(
  defineProps<{
    instances: WebResource[];
    sites?: WebResource[];
    initialInstanceId?: number;
    initialSiteId?: number;
    layoutKey?: boolean;
  }>(),
  { sites: () => [] }
);
type LogSource = { id: string; path: string; kind?: string };
type LogResult = {
  text: string;
  path: string;
  lines: number;
  at: number;
  offset: number;
  window: number;
  nextOffset: number;
  hasMore: boolean;
  scanLimit: number;
};
const siteId = ref<number>(),
  instanceId = ref<number>(),
  logId = ref(""),
  logs = ref<LogSource[]>([]);
const keyword = ref(""),
  rotation = ref(""),
  status = ref(""),
  ip = ref(""),
  uri = ref(""),
  from = ref<Date>(),
  to = ref<Date>();
const lines = ref(500),
  offset = ref(0),
  follow = ref(false),
  advanced = ref(false),
  wrap = ref(true);
const busy = ref(false),
  loadingLogs = ref(false),
  downloading = ref(false),
  text = ref(""),
  error = ref(""),
  path = ref(""),
  at = ref("");
const hasMore = ref(false),
  resultOffset = ref(0),
  resultWindow = ref(500),
  nextOffset = ref(0),
  scanLimit = ref(100000),
  matchedLines = ref(0);
const lastQuery = ref<{ instanceId: number; query: string }>();
const streamState = ref("paused"),
  streamNotice = ref(""),
  resultStreaming = ref(false);
const consoleHost = ref<HTMLElement>(),
  consoleHeight = ref(380);
let resize: ResizeObserver | undefined;
function fitConsole() {
  const top = consoleHost.value?.getBoundingClientRect().top || 0;
  consoleHeight.value = Math.max(
    280,
    window.innerHeight - Math.max(0, top) - (resultStreaming.value ? 36 : 110)
  );
}
onMounted(() => {
  window.addEventListener("resize", fitConsole);
  resize = new ResizeObserver(fitConsole);
  if (consoleHost.value?.parentElement)
    resize.observe(consoleHost.value.parentElement);
  fitConsole();
});
watch(consoleHost, (host, previous) => {
  if (previous?.parentElement) resize?.unobserve(previous.parentElement);
  if (host?.parentElement) resize?.observe(host.parentElement);
  nextTick(fitConsole);
});
watch(
  [
    advanced,
    error,
    siteId,
    instanceId,
    logId,
    resultStreaming,
    streamNotice,
    () => props.layoutKey
  ],
  () => nextTick(fitConsole)
);
const activeFilterCount = computed(
  () =>
    [
      siteId.value,
      keyword.value,
      rotation.value,
      status.value,
      ip.value,
      uri.value,
      from.value,
      to.value,
      lines.value !== 500
    ].filter(Boolean).length
);
const { copy } = useClipboard({ legacy: true });
const selectedSite = computed(() =>
  props.sites.find(site => site.id === siteId.value)
);
const availableInstances = computed(() =>
  props.instances.filter(
    instance =>
      !siteId.value ||
      (!!selectedSite.value &&
        instance.clusterId === selectedSite.value.clusterId)
  )
);
const siteLogPaths = computed(
  () =>
    new Set<string>(
      [
        selectedSite.value?.spec.accessLog,
        selectedSite.value?.spec.errorLog
      ].filter(
        (path): path is string => typeof path === "string" && path.length > 0
      )
    )
);
const visibleLogs = computed(() =>
  siteId.value
    ? logs.value.filter(log => siteLogPaths.value.has(log.path))
    : logs.value
);
const selectedLog = computed(() =>
  visibleLogs.value.find(log => log.id === logId.value)
);
const siteSignature = computed(() =>
  JSON.stringify({
    id: siteId.value,
    clusterId: selectedSite.value?.clusterId,
    paths: [...siteLogPaths.value]
  })
);
const siteNotice = computed(() => {
  if (!siteId.value) return "";
  if (!selectedSite.value) return tr("所选站点不可用，请清除筛选或重新选择");
  if (!availableInstances.value.length)
    return tr("站点所属集群没有符合筛选条件的实例");
  if (!siteLogPaths.value.size)
    return tr("该站点未设置独立日志路径，可清除站点筛选后查看实例日志");
  if (instanceId.value && !loadingLogs.value && !visibleLogs.value.length)
    return tr("该实例未发现该站点的日志文件，请检查站点发布与实例发现结果");
  return "";
});
const logGroups = computed(() =>
  [
    { kind: "access", label: "访问日志" },
    { kind: "error", label: "错误日志" },
    { kind: "container", label: "容器输出" },
    { kind: "file", label: "文件日志" }
  ]
    .map(group => ({
      ...group,
      items: visibleLogs.value.filter(
        log =>
          (log.id === "stdout" ? "container" : log.kind || "file") ===
          group.kind
      )
    }))
    .filter(group => group.items.length)
);
let gen = 0,
  abort: AbortController | undefined,
  downloadAbort: AbortController | undefined;
function invalidate() {
  gen++;
  abort?.abort();
  downloadAbort?.abort();
  streamState.value = "paused";
  busy.value = false;
  downloading.value = false;
  return gen;
}
function clearResult() {
  resultStreaming.value = false;
  streamNotice.value = "";
  text.value = "";
  path.value = "";
  at.value = "";
  hasMore.value = false;
  resultOffset.value = 0;
  matchedLines.value = 0;
  lastQuery.value = undefined;
}
async function loadLogs() {
  follow.value = false;
  const request = invalidate();
  offset.value = 0;
  logId.value = "";
  logs.value = [];
  error.value = "";
  clearResult();
  if (!instanceId.value) {
    loadingLogs.value = false;
    return;
  }
  abort = new AbortController();
  loadingLogs.value = true;
  try {
    const result = await webRequest<LogSource[]>(
      "/instances/" + instanceId.value + "/logs",
      "GET",
      undefined,
      abort.signal
    );
    if (request !== gen) return;
    logs.value = [
      ...new Map((result || []).map(log => [log.id, log])).values()
    ];
    logId.value = visibleLogs.value[0]?.id || "";
  } catch (e) {
    if (request === gen) error.value = (e as Error).message;
  } finally {
    if (request === gen) loadingLogs.value = false;
  }
  if (request === gen && logId.value) await refresh();
}
async function changeLog() {
  follow.value = false;
  offset.value = 0;
  clearResult();
  await refresh();
}
async function search() {
  follow.value = false;
  offset.value = 0;
  await refresh();
}
async function navigate(value: number) {
  offset.value = value;
  await refresh();
}
async function refresh() {
  if (
    !instanceId.value ||
    !logId.value ||
    !availableInstances.value.some(
      instance => instance.id === instanceId.value
    ) ||
    !visibleLogs.value.some(log => log.id === logId.value)
  )
    return;
  const request = invalidate(),
    selected = instanceId.value;
  resultStreaming.value = false;
  streamNotice.value = "";
  error.value = "";
  if (!!from.value !== !!to.value) {
    error.value = tr("请同时选择开始和结束时间");
    follow.value = false;
    clearResult();
    return;
  }
  if (from.value && to.value && from.value > to.value) {
    error.value = tr("开始时间不能晚于结束时间");
    follow.value = false;
    clearResult();
    return;
  }
  abort = new AbortController();
  busy.value = true;
  const query = webLogQuery({
    logId: logId.value,
    keyword: keyword.value,
    rotation: logId.value === "stdout" ? "" : rotation.value,
    status: status.value,
    ip: ip.value,
    uri: uri.value,
    lines: lines.value,
    offset: offset.value,
    range: from.value && to.value ? [from.value, to.value] : undefined
  });
  try {
    const result = await webRequest<LogResult>(
      "/instances/" + selected + "/logs?" + query,
      "GET",
      undefined,
      abort.signal
    );
    if (request !== gen) return;
    text.value = result.text;
    hasMore.value = result.hasMore && result.nextOffset < result.scanLimit;
    path.value = result.path;
    at.value = new Date(result.at * 1000).toLocaleString();
    resultOffset.value = result.offset;
    resultWindow.value = result.window;
    nextOffset.value = result.nextOffset;
    scanLimit.value = result.scanLimit;
    matchedLines.value = result.lines;
    lastQuery.value = { instanceId: selected, query };
    await nextTick();
    fitConsole();
  } catch (e) {
    if (request === gen) {
      error.value = (e as Error).message;
      follow.value = false;
      clearResult();
    }
  } finally {
    if (request === gen) {
      busy.value = false;
    }
  }
}
async function startStream() {
  if (
    !instanceId.value ||
    !logId.value ||
    !visibleLogs.value.some(log => log.id === logId.value)
  )
    return;
  const request = invalidate(),
    selected = instanceId.value;
  clearResult();
  error.value = "";
  if (rotation.value || from.value || to.value) {
    follow.value = false;
    error.value = tr("实时追踪仅支持当前日志，请先清除轮转文件和时间范围");
    return;
  }
  abort = new AbortController();
  streamState.value = "connecting";
  const query = webLogQuery({
    logId: logId.value,
    keyword: keyword.value,
    status: status.value,
    ip: ip.value,
    uri: uri.value,
    lines: lines.value
  });
  try {
    await webLogStream(selected, query, abort.signal, event => {
      if (request !== gen) return;
      if (event.event === "ready") {
        streamState.value = "live";
        path.value = event.data.path || "";
        resultStreaming.value = true;
        lastQuery.value = { instanceId: selected, query };
        streamNotice.value = tr(
          "SSE 已连接；重连显示当前最新日志，窗口最多保留 10000 行 / 2 MiB"
        );
        if (event.data.bounded)
          streamNotice.value += " · " + tr("首次读取已按大小截断");
        nextTick(fitConsole);
      } else if (event.event === "log") {
        text.value = appendLogText(text.value, event.data.text || "");
        matchedLines.value = text.value.split("\n").filter(Boolean).length;
        at.value = new Date(
          (event.data.at || Date.now() / 1000) * 1000
        ).toLocaleString();
      } else if (event.event === "reset") {
        streamNotice.value = tr(
          event.data.reason === "rotated"
            ? "日志已轮转，继续追踪新文件"
            : "日志已截断，继续追踪新内容"
        );
      } else if (event.event === "notice") {
        streamNotice.value = tr(event.data.message || "日志流提示");
      } else if (event.event === "error") {
        throw new Error(event.data.message || tr("日志连接失败"));
      } else if (event.event === "end") {
        streamNotice.value = tr(
          event.data.message || "日志连接已结束，可重新连接继续追踪"
        );
      }
    });
    if (request === gen) {
      follow.value = false;
      streamState.value = "paused";
    }
  } catch (e) {
    if (request === gen && !abort?.signal.aborted) {
      error.value = (e as Error).message;
      follow.value = false;
    }
  }
}
watch(
  follow,
  value => {
    invalidate();
    if (value) {
      offset.value = 0;
      startStream();
    }
  },
  { flush: "sync" }
);
watch(
  [keyword, rotation, status, ip, uri, from, to, lines],
  () => {
    if (loadingLogs.value) return;
    follow.value = false;
    invalidate();
    offset.value = 0;
    clearResult();
    error.value = "";
  },
  { flush: "sync" }
);
watch(
  () => props.initialSiteId,
  id => {
    siteId.value = id;
  },
  { immediate: true }
);
let initialInstanceSelected = false;
watch(
  () => [availableInstances.value, props.initialInstanceId] as const,
  ([instances, initial], previous) => {
    if (previous && previous[1] !== initial) initialInstanceSelected = false;
    if (
      !initialInstanceSelected &&
      initial &&
      instances.some(i => i.id === initial)
    ) {
      instanceId.value = initial;
      initialInstanceSelected = true;
    } else if (!instances.some(i => i.id === instanceId.value)) {
      instanceId.value =
        (siteId.value && instances.length) || instances.length === 1
          ? instances[0].id
          : undefined;
    }
  },
  { immediate: true }
);
watch([instanceId, siteSignature], loadLogs, {
  immediate: true,
  flush: "post"
});
async function copyResult() {
  try {
    await copy(text.value);
    ElMessage.success(tr("复制成功"));
  } catch {
    ElMessage.error(tr("复制失败"));
  }
}
async function download() {
  const selection = lastQuery.value;
  if (!selection || downloading.value) return;
  follow.value = false;
  if (resultStreaming.value) {
    const url = URL.createObjectURL(
      new Blob([text.value], { type: "text/plain;charset=utf-8" })
    );
    const a = document.createElement("a");
    a.href = url;
    a.download = "web-log-live.txt";
    a.click();
    URL.revokeObjectURL(url);
    return;
  }
  const request = gen;
  downloading.value = true;
  downloadAbort = new AbortController();
  try {
    const result = await webRequest<LogResult>(
      `/instances/${selection.instanceId}/logs?${selection.query}&download=json`,
      "GET",
      undefined,
      downloadAbort.signal
    );
    if (request !== gen) return;
    const url = URL.createObjectURL(
      new Blob([result.text], { type: "text/plain;charset=utf-8" })
    );
    const a = document.createElement("a");
    a.href = url;
    a.download = "web-log.txt";
    a.click();
    URL.revokeObjectURL(url);
  } catch (e) {
    if (request === gen) error.value = (e as Error).message;
  } finally {
    if (request === gen) downloading.value = false;
  }
}
onUnmounted(() => {
  invalidate();
  resize?.disconnect();
  window.removeEventListener("resize", fitConsole);
});
</script>
<style scoped>
.web-logs {
  min-width: 0;
  width: 100%;
}
.log-console {
  min-height: 280px;
  min-width: 0;
  width: 100%;
  max-width: 100%;
}
.log-filters {
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  background: var(--el-fill-color-light);
  margin-bottom: 8px;
}
.log-filter-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 2fr);
  gap: 0 12px;
}
.advanced-filters {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
.log-filter-grid .el-form-item {
  margin-bottom: 12px;
}
.log-filter-grid :deep(.el-form-item__label) {
  margin-bottom: 4px;
}
.filter-count {
  margin-left: 6px;
}
.log-stream-note {
  margin: 6px 0;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.log-filter-grid .el-form-item,
.log-result-meta {
  min-width: 0;
}
.log-filter-grid .el-select {
  width: 100%;
}
.log-actions,
.log-navigation {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}
.log-actions .el-button,
.log-navigation .el-button {
  margin-left: 0;
}
.advanced-filters {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.log-time-range {
  grid-column: 1 / -1;
}
.date-range-inputs {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  width: 100%;
}
.date-range-inputs :deep(.el-date-editor) {
  flex: 1;
  width: auto;
  min-width: 180px;
  max-width: 260px;
}
.el-alert {
  margin-bottom: 12px;
}
.log-result-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 6px 12px;
  margin: 6px 0;
}
.log-result-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 12px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.result-actions {
  gap: 8px;
}
.log-navigation {
  justify-content: center;
  margin: 18px 0 8px;
}
.log-scan-note {
  text-align: center;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.6;
}
@media (max-width: 1100px) {
  .advanced-filters {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (max-width: 640px) {
  .log-filters {
    padding: 10px;
  }
  .result-actions {
    gap: 6px;
  }
  .result-actions :deep(.el-button) {
    padding: 8px;
  }
  .log-filter-grid {
    grid-template-columns: minmax(0, 1fr);
  }
  .date-range-inputs :deep(.el-date-editor) {
    min-width: 0;
    max-width: none;
    flex-basis: 100%;
  }
  .date-range-inputs > span {
    display: none;
  }
}
</style>
