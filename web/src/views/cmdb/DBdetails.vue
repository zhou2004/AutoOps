<template>
  <div class="workbench-page" v-loading="loading">
    <section class="workbench-header">
      <div class="header-main">
        <el-button circle @click="router.push('/database/sources')"
          >←</el-button
        >
        <div>
          <p class="eyebrow">DATABASE WORKBENCH</p>
          <h1>{{ source.name || $tr("数据源工作台") }}</h1>
          <p>
            {{ typeName }} · {{ overview?.address || $tr("正在读取连接信息") }}
          </p>
        </div>
      </div>
      <div class="header-actions">
        <el-tag :type="overview ? 'success' : 'info'">{{
          overview
            ? $tr("在线 · {value0} ms", { value0: overview.latencyMs })
            : $tr("未连接")
        }}</el-tag
        ><el-button type="primary" @click="refresh">{{
          $tr("刷新")
        }}</el-button>
      </div>
    </section>

    <el-alert
      v-if="error"
      :title="$tr(error)"
      type="error"
      show-icon
      :closable="false"
      class="status-alert"
    />
    <section v-if="overview" class="metric-grid">
      <article>
        <span>{{ $tr("版本") }}</span
        ><strong>{{ overview.version || "-" }}</strong>
      </article>
      <article>
        <span>{{ $tr("角色") }}</span
        ><strong>{{ overview.role || roleText }}</strong>
      </article>
      <article>
        <span>{{ $tr("默认库 / DB") }}</span
        ><strong>{{
          source.defaultDb || (source.type === 3 ? "0" : $tr("未指定"))
        }}</strong>
      </article>
      <article>
        <span>TLS</span><strong>{{ source.tlsMode || "disable" }}</strong>
      </article>
    </section>

    <el-card shadow="never" class="workbench-card">
      <el-tabs v-model="activeTab" @tab-change="loadTab">
        <template v-if="source.type !== 3">
          <el-tab-pane :label="$tr('对象浏览')" name="objects">
            <div class="object-toolbar">
              <el-select
                v-model="selectedDatabase"
                filterable
                :placeholder="$tr('选择数据库')"
                @change="loadTables"
                ><el-option
                  v-for="item in catalog"
                  :key="item"
                  :label="item"
                  :value="item" /></el-select
              ><el-input
                v-model="tableKeyword"
                clearable
                :placeholder="$tr('筛选表或视图')"
              />
            </div>
            <el-table :data="filteredTables" stripe
              ><el-table-column prop="schema" label="Schema" /><el-table-column
                prop="name"
                :label="$tr('表 / 视图')"
                min-width="260" /><el-table-column
                prop="type"
                :label="$tr('类型')" /><el-table-column
                prop="rows"
                :label="$tr('估算行数')"
            /></el-table>
          </el-tab-pane>
          <el-tab-pane :label="$tr('SQL 查询')" name="query">
            <el-alert
              :title="
                $tr(
                  '只读入口支持 SELECT、SHOW、DESCRIBE、EXPLAIN 和 WITH，最多返回 500 行，超时为 30 秒。'
                )
              "
              type="info"
              :closable="false"
              show-icon
            />
            <div class="sql-toolbar">
              <el-select
                v-model="selectedDatabase"
                filterable
                :placeholder="$tr('数据库')"
                ><el-option
                  v-for="item in catalog"
                  :key="item"
                  :label="item"
                  :value="item" /></el-select
              ><el-button
                type="primary"
                :loading="executing"
                @click="runQuery"
                >{{ $tr("执行查询") }}</el-button
              >
            </div>
            <el-input
              v-model="sqlText"
              type="textarea"
              :rows="9"
              class="sql-editor"
              spellcheck="false"
            />
            <div v-if="queryResult" class="result-meta">
              {{ $tr("返回") }} {{ queryResult.returnedRows }}
              {{ $tr("行 ·") }} {{ queryResult.durationMs }} ms
              <el-tag v-if="queryResult.truncated" type="warning">{{
                $tr("结果已截断")
              }}</el-tag>
            </div>
            <el-table
              v-if="queryResult"
              :data="queryResult.rows"
              stripe
              max-height="440"
              ><el-table-column
                v-for="column in queryResult.columns"
                :key="column"
                :prop="column"
                :label="column"
                min-width="140"
                show-overflow-tooltip
            /></el-table>
          </el-tab-pane>
          <el-tab-pane :label="$tr('写操作')" name="write">
            <el-alert
              :title="
                $tr(
                  '仅允许单条 INSERT、UPDATE 或 DELETE。UPDATE/DELETE 必须包含 WHERE，执行前需要输入完整数据源名称。'
                )
              "
              type="warning"
              :closable="false"
              show-icon
            />
            <div class="sql-toolbar">
              <el-select
                v-model="selectedDatabase"
                filterable
                :placeholder="$tr('数据库')"
                ><el-option
                  v-for="item in catalog"
                  :key="item"
                  :label="item"
                  :value="item" /></el-select
              ><el-button
                type="danger"
                :loading="executing"
                @click="runWrite"
                >{{ $tr("确认并执行") }}</el-button
              >
            </div>
            <el-input
              v-model="writeSQL"
              type="textarea"
              :rows="9"
              class="sql-editor"
              spellcheck="false"
            />
            <el-result
              v-if="writeResult"
              icon="success"
              :title="$tr('执行完成')"
              :sub-title="
                $tr('影响 {value0} 行，耗时 {value1} ms', {
                  value0: writeResult.affectedRows,
                  value1: writeResult.durationMs
                })
              "
            />
          </el-tab-pane>
        </template>

        <template v-else>
          <el-tab-pane :label="$tr('Key 浏览')" name="redis">
            <div class="redis-toolbar">
              <el-input
                v-model="redisMatch"
                :placeholder="$tr('SCAN 匹配式，例如 app:*')"
                @keyup.enter="scanKeys(true)"
              /><el-button type="primary" @click="scanKeys(true)">{{
                $tr("扫描")
              }}</el-button
              ><el-button
                :disabled="redisCursor === 0"
                @click="scanKeys(false)"
                >{{ $tr("下一批") }}</el-button
              ><el-button type="success" @click="openRedisEditor()">{{
                $tr("新增 String")
              }}</el-button>
            </div>
            <el-table :data="redisKeys" stripe
              ><el-table-column
                prop="key"
                label="Key"
                min-width="320"
                show-overflow-tooltip
              /><el-table-column
                prop="type"
                :label="$tr('类型')"
                width="120"
              /><el-table-column label="TTL" width="140"
                ><template #default="{ row }">{{
                  ttlText(row.ttl)
                }}</template></el-table-column
              ><el-table-column :label="$tr('操作')" width="220"
                ><template #default="{ row }"
                  ><el-button link type="primary" @click="readRedis(row)">{{
                    $tr("查看")
                  }}</el-button
                  ><el-button
                    v-if="row.type === 'string'"
                    link
                    type="warning"
                    @click="editRedis(row)"
                    >{{ $tr("编辑") }}</el-button
                  ><el-button link type="danger" @click="deleteRedis(row)">{{
                    $tr("删除")
                  }}</el-button></template
                ></el-table-column
              ></el-table
            >
          </el-tab-pane>
        </template>

        <el-tab-pane :label="$tr('操作审计')" name="audit">
          <el-table :data="audits" stripe
            ><el-table-column
              prop="createdAt"
              :label="$tr('时间')"
              min-width="190" /><el-table-column
              prop="userId"
              :label="$tr('用户')"
              width="90" /><el-table-column
              prop="action"
              :label="$tr('动作')"
              width="130" /><el-table-column
              prop="target"
              :label="$tr('目标')"
              min-width="220"
              show-overflow-tooltip /><el-table-column
              :label="$tr('结果')"
              width="100"
              ><template #default="{ row }"
                ><el-tag
                  :type="row.status === 'success' ? 'success' : 'danger'"
                  >{{ row.status }}</el-tag
                ></template
              ></el-table-column
            ><el-table-column
              prop="durationMs"
              :label="$tr('耗时(ms)')"
              width="110" /><el-table-column
              prop="clientIp"
              :label="$tr('客户端 IP')"
              width="150" /><el-table-column
              prop="message"
              :label="$tr('错误')"
              min-width="220"
              show-overflow-tooltip
          /></el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-drawer
      v-model="redisValueVisible"
      :title="$tr(redisValue.key || 'Redis 值')"
      size="50%"
      ><el-descriptions :column="2" border
        ><el-descriptions-item :label="$tr('类型')">{{
          redisValue.type
        }}</el-descriptions-item
        ><el-descriptions-item label="TTL">{{
          ttlText(redisValue.ttl)
        }}</el-descriptions-item></el-descriptions
      >
      <pre class="value-preview">{{ formattedRedisValue }}</pre>
    </el-drawer>
    <el-dialog
      v-model="redisEditorVisible"
      :title="$tr('写入 Redis String')"
      width="min(620px, 94vw)"
      ><el-form :model="redisForm" label-width="90px"
        ><el-form-item label="Key"
          ><el-input
            v-model="redisForm.key"
            :disabled="redisForm.editing" /></el-form-item
        ><el-form-item :label="$tr('TTL 秒')"
          ><el-input-number
            v-model="redisForm.ttl"
            :min="0"
            :max="31536000" /></el-form-item
        ><el-form-item :label="$tr('值')"
          ><el-input
            v-model="redisForm.value"
            type="textarea"
            :rows="7"
            maxlength="1048576"
            show-word-limit /></el-form-item></el-form
      ><template #footer
        ><el-button @click="redisEditorVisible = false">{{
          $tr("取消")
        }}</el-button
        ><el-button type="primary" @click="saveRedis">{{
          $tr("确认写入")
        }}</el-button></template
      ></el-dialog
    >
  </div>
</template>

<script setup>
import { tr } from "@/locales";

import { computed, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import api from "@/api/cmdb";

const route = useRoute(),
  router = useRouter(),
  id = Number(route.query.id);
const loading = ref(false),
  executing = ref(false),
  error = ref(""),
  activeTab = ref("objects");
const source = ref({}),
  overview = ref(null),
  catalog = ref([]),
  selectedDatabase = ref(""),
  tables = ref([]),
  tableKeyword = ref("");
const sqlText = ref("SELECT 1 AS connection_ok;"),
  writeSQL = ref(""),
  queryResult = ref(null),
  writeResult = ref(null),
  audits = ref([]);
const redisMatch = ref("*"),
  redisCursor = ref(0),
  redisKeys = ref([]),
  redisValueVisible = ref(false),
  redisValue = ref({}),
  redisEditorVisible = ref(false),
  redisForm = reactive({ key: "", value: "", ttl: 0, editing: false });
const typeName = computed(
  () =>
    ({ 1: "MySQL", 2: "PostgreSQL", 3: "Redis" })[source.value.type] ||
    tr("未知类型")
);
const roleText = computed(() =>
  overview.value?.metrics?.inRecovery
    ? "standby"
    : overview.value?.metrics?.readOnly
      ? "read only"
      : "primary"
);
const filteredTables = computed(() => {
  const word = tableKeyword.value.trim().toLowerCase();
  return word
    ? tables.value.filter(item =>
        `${item.schema}.${item.name}`.toLowerCase().includes(word)
      )
    : tables.value;
});
const formattedRedisValue = computed(() =>
  typeof redisValue.value.value === "string"
    ? redisValue.value.value
    : JSON.stringify(redisValue.value.value, null, 2)
);
function unpack(response) {
  if (response.data?.code !== 200)
    throw new Error(response.data?.message || tr("操作失败"));
  return response.data.data;
}
function ttlText(ttl) {
  return ttl === -1
    ? tr("永久")
    : ttl === -2
      ? tr("不存在")
      : tr("{value0} 秒", { value0: ttl });
}

async function refresh() {
  if (!id) {
    error.value = tr("数据源 ID 无效");
    return;
  }
  loading.value = true;
  error.value = "";
  try {
    const [assetResponse, overviewResponse] = await Promise.all([
      api.getDatabase(id),
      api.getDatabaseOverview(id)
    ]);
    source.value = unpack(assetResponse);
    overview.value = unpack(overviewResponse);
    if (source.value.type === 3) {
      activeTab.value = "redis";
      await scanKeys(true);
    } else {
      await loadCatalog();
    }
  } catch (e) {
    error.value = e.message;
  } finally {
    loading.value = false;
  }
}
async function loadCatalog() {
  const data = unpack(await api.getDatabaseCatalog(id));
  catalog.value = data?.list || [];
  if (!selectedDatabase.value)
    selectedDatabase.value = source.value.defaultDb || catalog.value[0] || "";
  if (selectedDatabase.value) await loadTables();
}
async function loadTables() {
  if (!selectedDatabase.value) return;
  tables.value =
    unpack(await api.getDatabaseTables(id, selectedDatabase.value))?.list || [];
}
async function runQuery() {
  executing.value = true;
  try {
    queryResult.value = unpack(
      await api.queryDatabase(id, {
        database: selectedDatabase.value,
        sql: sqlText.value
      })
    );
    ElMessage.success(tr("查询完成"));
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    executing.value = false;
  }
}
async function runWrite() {
  try {
    const prompt = await ElMessageBox.prompt(
      tr("输入数据源名称 {value0} 确认写操作", { value0: source.value.name }),
      tr("高风险操作确认"),
      {
        type: "warning",
        inputValidator: value => value === source.value.name || tr("名称不匹配")
      }
    );
    executing.value = true;
    writeResult.value = unpack(
      await api.executeDatabaseWrite(id, {
        database: selectedDatabase.value,
        sql: writeSQL.value,
        confirmName: prompt.value
      })
    );
    ElMessage.success(tr("写操作执行完成"));
  } catch (e) {
    if (e !== "cancel" && e !== "close" && e?.message)
      ElMessage.error(e.message);
  } finally {
    executing.value = false;
  }
}
async function scanKeys(reset) {
  try {
    if (reset) redisCursor.value = 0;
    const data = unpack(
      await api.scanRedisKeys(id, {
        cursor: redisCursor.value,
        match: redisMatch.value || "*",
        count: 100
      })
    );
    redisKeys.value = data.list || [];
    redisCursor.value = data.nextCursor || 0;
  } catch (e) {
    ElMessage.error(e.message);
  }
}
async function readRedis(row) {
  try {
    redisValue.value = unpack(await api.getRedisValue(id, row.key));
    redisValueVisible.value = true;
  } catch (e) {
    ElMessage.error(e.message);
  }
}
function openRedisEditor(value = {}) {
  Object.assign(
    redisForm,
    { key: "", value: "", ttl: 0, editing: false },
    value
  );
  redisEditorVisible.value = true;
}
async function editRedis(row) {
  try {
    const data = unpack(await api.getRedisValue(id, row.key));
    openRedisEditor({
      key: data.key,
      value: String(data.value ?? ""),
      ttl: Math.max(0, Number(data.ttl || 0)),
      editing: true
    });
  } catch (e) {
    ElMessage.error(e.message);
  }
}
async function saveRedis() {
  try {
    const prompt = await ElMessageBox.prompt(
      tr("输入数据源名称 {value0} 确认写入", { value0: source.value.name }),
      tr("写入确认"),
      {
        type: "warning",
        inputValidator: value => value === source.value.name || tr("名称不匹配")
      }
    );
    unpack(
      await api.setRedisValue(id, {
        key: redisForm.key,
        value: redisForm.value,
        ttl: Number(redisForm.ttl),
        confirmName: prompt.value
      })
    );
    redisEditorVisible.value = false;
    ElMessage.success(tr("Redis String 已保存"));
    await scanKeys(true);
  } catch (e) {
    if (e !== "cancel" && e !== "close" && e?.message)
      ElMessage.error(e.message);
  }
}
async function deleteRedis(row) {
  try {
    const prompt = await ElMessageBox.prompt(
      tr("输入完整 Key {value0} 确认删除", { value0: row.key }),
      tr("删除 Redis Key"),
      {
        type: "error",
        inputValidator: value => value === row.key || tr("Key 不匹配")
      }
    );
    unpack(
      await api.deleteRedisValue(id, { key: row.key, confirmKey: prompt.value })
    );
    ElMessage.success(tr("Key 已删除"));
    await scanKeys(true);
  } catch (e) {
    if (e !== "cancel" && e !== "close" && e?.message)
      ElMessage.error(e.message);
  }
}
async function loadAudits() {
  try {
    audits.value =
      unpack(await api.getDatabaseAudits(id, { page: 1, size: 100 }))?.list ||
      [];
  } catch (e) {
    ElMessage.error(e.message);
  }
}
async function loadTab(name) {
  if (name === "audit") await loadAudits();
  if (name === "objects" && source.value.type !== 3 && !tables.value.length)
    await loadTables();
}
onMounted(refresh);
</script>

<style scoped>
.workbench-page {
  padding: 22px;
}
.workbench-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 23px 28px;
  border: 1px solid var(--border-light);
  border-radius: var(--radius-lg);
  background: linear-gradient(135deg, var(--bg-card), var(--bg-card-alt));
}
.header-main,
.header-actions {
  display: flex;
  align-items: center;
  gap: 15px;
}
.workbench-header h1 {
  margin: 2px 0 5px;
  color: var(--text-primary);
  font-size: 26px;
}
.workbench-header p {
  margin: 0;
  color: var(--text-secondary);
}
.eyebrow {
  color: var(--primary) !important;
  font-size: 11px !important;
  font-weight: 700;
  letter-spacing: 0.14em;
}
.status-alert {
  margin-top: 14px;
}
.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
  margin: 16px 0;
}
.metric-grid article {
  min-width: 0;
  padding: 17px 20px;
  border: 1px solid var(--border-light);
  border-radius: var(--radius);
  background: var(--bg-card);
}
.metric-grid span {
  display: block;
  color: var(--text-secondary);
  font-size: 12px;
}
.metric-grid strong {
  display: block;
  margin-top: 7px;
  overflow: hidden;
  color: var(--text-primary);
  font-size: 17px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.workbench-card {
  margin-top: 16px;
}
.object-toolbar,
.sql-toolbar,
.redis-toolbar {
  display: flex;
  gap: 10px;
  margin: 5px 0 16px;
}
.object-toolbar .el-select,
.sql-toolbar .el-select {
  width: 240px;
}
.object-toolbar .el-input {
  width: 280px;
}
.redis-toolbar .el-input {
  width: 340px;
}
.sql-editor {
  margin-bottom: 16px;
}
.sql-editor :deep(textarea),
.value-preview {
  font:
    13px/1.65 ui-monospace,
    SFMono-Regular,
    Consolas,
    monospace;
}
.result-meta {
  margin: 12px 0;
  color: var(--text-secondary);
}
.value-preview {
  min-height: 320px;
  margin-top: 18px;
  padding: 16px;
  border-radius: 10px;
  background: #08111e;
  color: #d8e4f3;
  white-space: pre-wrap;
  word-break: break-all;
}
@media (max-width: 900px) {
  .metric-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .workbench-header {
    align-items: flex-start;
  }
  .header-actions {
    flex-direction: column;
  }
}
</style>
