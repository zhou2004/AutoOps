<template>
  <div class="database-logs">
    <header class="heading">
      <div>
        <h2>{{ $tr("日志管理") }}</h2>
        <p>{{ $tr("按集群和实例查看数据库日志，通过 SSE 实时追踪。") }}</p>
      </div>
      <el-select
        v-model="filters.engine"
        :aria-label="$tr('数据库类型')"
        @change="search"
      >
        <el-option label="MySQL" value="mysql" /><el-option
          label="Redis"
          value="redis"
        /><el-option label="PostgreSQL" value="postgresql" />
      </el-select>
    </header>
    <el-card shadow="never">
      <div class="filters">
        <el-input
          v-model="filters.name"
          clearable
          :placeholder="$tr('集群名称')"
          @keyup.enter="search"
        />
        <el-tree-select
          v-model="filters.groupId"
          :data="groups"
          :props="{ label: 'name', value: 'id', children: 'children' }"
          check-strictly
          clearable
          filterable
          :placeholder="$tr('业务分组（含子分组）')"
          @change="search"
        />
        <el-input
          v-model="filters.environment"
          clearable
          :placeholder="$tr('环境')"
          @keyup.enter="search"
        />
        <el-select
          v-model="filters.status"
          clearable
          :placeholder="$tr('状态')"
          @change="search"
          ><el-option
            v-for="status in ['healthy', 'degraded', 'unknown', 'stale']"
            :key="status"
            :value="status"
            :label="label(status)"
        /></el-select>
        <el-button type="primary" :loading="loading" @click="search">{{
          $tr("查询")
        }}</el-button>
      </div>
      <el-alert
        v-if="listError"
        :title="listError"
        type="error"
        :closable="false"
      />
      <el-table v-loading="loading" :data="clusters" row-key="id">
        <el-table-column :label="$tr('集群名称')" min-width="240"
          ><template #default="{ row }"
            ><div class="identity">
              <DatabaseEngineIcon :engine="row.engine" /><span>{{
                row.name
              }}</span>
            </div></template
          ></el-table-column
        >
        <el-table-column
          prop="environment"
          :label="$tr('环境')"
          min-width="100"
        />
        <el-table-column :label="$tr('节点数')" width="100"
          ><template #default="{ row }">{{
            row.members.length
          }}</template></el-table-column
        >
        <el-table-column :label="$tr('状态')" width="140"
          ><template #default="{ row }"
            ><el-tag :type="tagType(row.status)">{{
              label(row.status)
            }}</el-tag></template
          ></el-table-column
        >
        <el-table-column :label="$tr('操作')" width="130" fixed="right"
          ><template #default="{ row }"
            ><el-button link type="primary" @click="openLogs(row)">{{
              $tr("查看日志")
            }}</el-button></template
          ></el-table-column
        >
      </el-table>
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="size"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @current-change="load"
        @size-change="search"
      />
    </el-card>
    <el-drawer
      v-model="opened"
      :title="(selected?.name || '') + ' · ' + $tr('数据库日志')"
      size="min(1100px, 96vw)"
      @close="closeLogs"
    >
      <div class="log-workspace">
        <el-alert
          :title="$tr('只读查看已有日志，不会开启慢查询或修改数据库日志配置。')"
          type="info"
          :closable="false"
        />
        <div class="log-options">
          <el-select
            v-model="memberId"
            :placeholder="$tr('数据库实例')"
            @change="loadSources"
            ><el-option
              v-for="member in selected?.members || []"
              :key="member.id"
              :value="member.id"
              :disabled="!member.hostId"
              :label="
                (member.name || member.address) +
                ':' +
                member.port +
                ' · ' +
                (member.service || member.kind)
              "
          /></el-select>
          <el-select
            v-model="logId"
            :loading="sourcesLoading"
            :placeholder="$tr('日志源')"
            @change="stop"
            ><el-option
              v-for="source in sources"
              :key="source.id"
              :value="source.id"
              :label="
                source.kind === 'journal'
                  ? $tr('服务日志') + ' · ' + source.path
                  : source.path
              "
          /></el-select>
          <el-select v-model="lines" :aria-label="$tr('最近行数')"
            ><el-option
              v-for="n in [100, 500, 1000, 2000]"
              :key="n"
              :value="n"
              :label="$tr('最近 {value0} 行', { value0: n })"
          /></el-select>
        </div>
        <div class="log-tools">
          <el-button
            :disabled="!logId || running || sourcesLoading"
            @click="start(false)"
            >{{ $tr("加载日志") }}</el-button
          >
          <el-button
            type="primary"
            :disabled="!logId || running || sourcesLoading"
            @click="start(true)"
            >{{ $tr("实时追踪") }}</el-button
          >
          <el-button :disabled="!running" @click="stop">{{
            $tr("停止追踪")
          }}</el-button>
          <el-button :disabled="sourcesLoading" @click="loadSources">{{
            $tr("刷新日志源")
          }}</el-button>
          <el-button @click="text = ''">{{ $tr("清空显示") }}</el-button>
          <el-checkbox v-model="autoScroll">{{ $tr("自动滚动") }}</el-checkbox>
          <el-tag :type="running ? 'success' : 'info'">{{
            $tr(running ? "连接中" : "已停止")
          }}</el-tag>
        </div>
        <el-alert v-if="error" :title="error" type="error" :closable="false" />
        <el-alert
          v-else-if="notice"
          :title="notice"
          type="info"
          :closable="false"
        />
        <el-empty
          v-if="!sourcesLoading && !sources.length && !error"
          :description="
            $tr('未发现日志源，请核对实例的服务名、数据目录和 CMDB SSH 权限。')
          "
        />
        <LogConsole
          v-else
          :text="text"
          :title="activePath || $tr('数据库日志')"
          :follow="running && autoScroll"
        />
        <small>{{
          $tr(
            "仅保留最近 10000 行 / 2 MiB；连接最多持续 20 分钟，结束后可重新连接。"
          )
        }}</small>
      </div>
    </el-drawer>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted, onBeforeUnmount, onDeactivated } from "vue";
import { useRoute } from "vue-router";
import { tr } from "@/locales";
import {
  dbPlatform,
  dbRequest,
  label,
  tagType,
  type Cluster
} from "@/api/databasePlatform";
import { databaseLogStream, appendLogText } from "@/api/databaseLogStream";
import DatabaseEngineIcon from "./components/DatabaseEngineIcon.vue";
import LogConsole from "@/components/LogConsole/index.vue";
interface LogSource {
  id: string;
  kind: string;
  path: string;
  name: string;
  size: number;
}
const route = useRoute();
const filters = reactive({
  engine: ["mysql", "redis", "postgresql"].includes(String(route.query.engine))
    ? String(route.query.engine)
    : "mysql",
  name: "",
  groupId: undefined as number | undefined,
  environment: "",
  status: ""
});
const clusters = ref<Cluster[]>([]),
  groups = ref<any[]>([]),
  total = ref(0),
  page = ref(1),
  size = ref(20),
  loading = ref(false),
  listError = ref("");
const opened = ref(false),
  selected = ref<Cluster>(),
  memberId = ref<number>(),
  sources = ref<LogSource[]>([]),
  logId = ref(""),
  lines = ref(500),
  sourcesLoading = ref(false);
const running = ref(false),
  text = ref(""),
  error = ref(""),
  notice = ref(""),
  activePath = ref(""),
  autoScroll = ref(true);
let stream: AbortController | undefined,
  sourceRequest: AbortController | undefined,
  listVersion = 0,
  sourceVersion = 0;
async function load() {
  const version = ++listVersion;
  loading.value = true;
  listError.value = "";
  try {
    const data = await dbPlatform.clusters({
      ...filters,
      page: page.value,
      size: size.value
    });
    if (version === listVersion) {
      clusters.value = data.list;
      total.value = data.total;
    }
  } catch (e: any) {
    if (version === listVersion) listError.value = tr(e.message);
  } finally {
    if (version === listVersion) loading.value = false;
  }
}
function search() {
  page.value = 1;
  void load();
}
function stop() {
  stream?.abort();
  stream = undefined;
  running.value = false;
}
function closeLogs() {
  stop();
  sourceRequest?.abort();
  sourceVersion++;
  sourcesLoading.value = false;
}
function openLogs(cluster: Cluster) {
  closeLogs();
  selected.value = cluster;
  memberId.value = cluster.members.find(m => m.hostId)?.id;
  opened.value = true;
  void loadSources();
}
async function loadSources() {
  stop();
  sourceRequest?.abort();
  const version = ++sourceVersion;
  sources.value = [];
  logId.value = "";
  text.value = "";
  error.value = "";
  notice.value = "";
  activePath.value = "";
  if (!selected.value || !memberId.value) {
    error.value = tr("实例尚未关联 CMDB 主机，无法读取日志");
    sourcesLoading.value = false;
    return;
  }
  const request = new AbortController();
  sourceRequest = request;
  sourcesLoading.value = true;
  try {
    const data = await dbRequest<LogSource[]>(
      "clusters/" + selected.value.id + "/logs",
      "get",
      undefined,
      { memberId: memberId.value },
      request.signal
    );
    if (version !== sourceVersion) return;
    sources.value = data;
    logId.value = data[0]?.id || "";
  } catch (e: any) {
    if (version === sourceVersion && !request.signal.aborted)
      error.value = tr(e.message);
  } finally {
    if (version === sourceVersion) sourcesLoading.value = false;
  }
}
async function start(follow: boolean) {
  if (!selected.value || !memberId.value || !logId.value || running.value)
    return;
  stop();
  const request = new AbortController();
  stream = request;
  running.value = true;
  text.value = "";
  error.value = "";
  notice.value = "";
  activePath.value = "";
  const query = new URLSearchParams({
    memberId: String(memberId.value),
    logId: logId.value,
    lines: String(lines.value),
    follow: String(follow)
  });
  try {
    await databaseLogStream(
      selected.value.id,
      query.toString(),
      request.signal,
      event => {
        if (stream !== request) return;
        if (event.event === "log")
          text.value = appendLogText(text.value, event.data.text || "");
        if (event.event === "ready") activePath.value = event.data.path || "";
        if (event.event === "error")
          error.value = tr(event.data.message || "日志连接失败");
        if (event.event === "notice" || event.event === "end")
          notice.value = tr(event.data.message || "");
      }
    );
  } catch (e: any) {
    if (stream === request && !request.signal.aborted)
      error.value = tr(e.message);
  } finally {
    if (stream === request) {
      running.value = false;
      stream = undefined;
    }
  }
}
onMounted(async () => {
  void load();
  try {
    groups.value = await dbRequest("groups");
  } catch (e: any) {
    listError.value = tr(e.message);
  }
});
onBeforeUnmount(() => {
  listVersion++;
  closeLogs();
});
onDeactivated(closeLogs);
</script>
<style scoped>
.database-logs {
  padding: 20px;
}
.heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}
h2 {
  margin: 0 0 8px;
}
p,
small {
  color: var(--el-text-color-secondary);
}
.heading > .el-select {
  width: 180px;
  flex-shrink: 0;
}
.filters,
.log-tools,
.log-options {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  align-items: center;
}
.filters {
  margin-bottom: 18px;
}
.filters > .el-input,
.filters > .el-select,
.filters > .el-tree-select {
  width: 180px;
}
.identity {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.identity span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.identity :deep(img) {
  width: 26px;
  height: 26px;
  flex-basis: 26px;
}
:deep(.el-table__row) {
  height: 64px;
}
.el-pagination {
  margin-top: 20px;
  justify-content: center;
  flex-wrap: wrap;
  gap: 8px;
}
.log-workspace {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
  min-height: 500px;
}
.log-options .el-select {
  flex: 1;
  min-width: 180px;
}
.log-options .el-select:last-child {
  flex: 0 0 160px;
}
.log-workspace > :deep(.log-console-panel) {
  flex: 1;
  min-height: 280px;
}
.log-tools .el-button {
  margin-left: 0;
}
@media (max-width: 600px) {
  .database-logs {
    padding: 12px;
  }
  .heading {
    flex-wrap: wrap;
  }
}
</style>
