<template>
  <div class="incident-page">
    <header class="page-header">
      <div>
        <span class="eyebrow">INCIDENT CENTER</span>
        <h1>{{ $tr("故障管理") }}</h1>
        <p>{{ $tr("集中跟踪监控告警，定位异常并记录恢复状态。") }}</p>
      </div>
      <div class="refresh-actions">
        <el-switch v-model="autoRefresh" :active-text="$tr('30 秒刷新')" />
        <el-button
          icon="Refresh"
          :loading="loading || statsLoading"
          @click="refresh"
          >{{ $tr("刷新数据") }}</el-button
        >
      </div>
    </header>
    <div class="scope-note">
      <span>{{ $tr("概览统计全部故障，列表支持独立筛选") }}</span
      ><span v-if="updatedAt">{{ $tr("最近更新") }} {{ updatedAt }}</span>
    </div>
    <el-alert
      v-if="statsError"
      :title="statsError"
      type="error"
      show-icon
      :closable="false"
      class="error-banner"
    />
    <el-skeleton v-if="!stats && statsLoading" :rows="4" animated />
    <div v-if="stats" v-loading="statsLoading" class="stats-grid">
      <button class="metric metric-danger" @click="filterStatus('firing')">
        <span>{{ $tr("活跃故障") }}</span
        ><strong>{{ stats.totalFiring }}</strong
        ><small>{{ $tr("其中严重故障") }} {{ stats.criticalFiring }}</small>
      </button>
      <button class="metric metric-success" @click="filterStatus('resolved')">
        <span>{{ $tr("已解决") }}</span
        ><strong>{{ stats.totalResolved }}</strong
        ><small>{{ $tr("累计恢复与人工解决") }}</small>
      </button>
      <div class="metric">
        <span>{{ $tr("24 小时新增") }}</span
        ><strong>{{ stats.last24hCount }}</strong
        ><small>{{ $tr("按告警发生时间统计") }}</small>
      </div>
      <div class="metric">
        <span>{{ $tr("今日新增") }}</span
        ><strong>{{ stats.todayCount }}</strong
        ><small>{{ $tr("今日零点至今") }}</small>
      </div>
    </div>
    <div v-if="stats" class="insights-grid">
      <section class="surface trend-panel">
        <div class="section-heading">
          <h2>{{ $tr("近 7 天故障趋势") }}</h2>
          <span>{{ $tr("新增故障数") }}</span>
        </div>
        <div class="trend-chart" role="img" :aria-label="trendDescription">
          <div v-for="point in stats.trend" :key="point.date" class="trend-day">
            <div class="bar-track">
              <div
                class="trend-bar"
                :style="{
                  height: `${point.count ? Math.max(3, (point.count / trendMax) * 100) : 0}%`
                }"
              >
                <span>{{ point.count }}</span>
              </div>
            </div>
            <time>{{ point.date.slice(5) }}</time>
          </div>
        </div>
        <p v-if="!stats.trend.some(p => p.count)" class="trend-empty">
          {{ $tr("近 7 天暂无新增故障") }}
        </p>
      </section>
      <section class="surface distribution-panel">
        <div class="section-heading">
          <h2>{{ $tr("故障分布") }}</h2>
          <span>{{ $tr("全部历史记录") }}</span>
        </div>
        <div class="level-summary">
          <button
            v-for="level in levels"
            :key="level.value"
            @click="
              q.level = level.value;
              handleQuery();
            "
          >
            <span class="level-dot" :class="level.value" /><span>{{
              $tr(level.label)
            }}</span
            ><strong>{{ stats.byLevel[level.value] || 0 }}</strong>
          </button>
        </div>
        <div class="source-list">
          <button
            v-for="source in sourceDistribution"
            :key="source.value"
            class="source-row"
            @click="
              q.source = source.value;
              handleQuery();
            "
          >
            <span>{{ sourceLabel(source.value) }}</span>
            <div class="source-track">
              <i :style="{ width: `${source.percent}%` }" />
            </div>
            <strong>{{ source.count }}</strong>
          </button>
        </div>
        <p v-if="!sourceDistribution.length" class="muted">
          {{ $tr("收到告警后将展示来源分布") }}
        </p>
      </section>
    </div>
    <section class="surface incident-list">
      <div class="section-heading">
        <h2>
          {{ $tr("故障记录") }} <span class="count-badge">{{ total }}</span>
        </h2>
        <span>{{ $tr("活跃故障优先展示") }}</span>
      </div>
      <el-form
        :inline="true"
        :model="q"
        class="filters"
        @submit.prevent="handleQuery"
      >
        <el-form-item :label="$tr('搜索')"
          ><el-input
            v-model="q.keyword"
            :placeholder="$tr('搜索标题或描述')"
            clearable
            @keyup.enter="handleQuery"
        /></el-form-item>
        <el-form-item :label="$tr('状态')"
          ><el-select
            v-model="q.status"
            clearable
            :placeholder="$tr('全部状态')"
            ><el-option :label="$tr('活跃')" value="firing" /><el-option
              :label="$tr('已解决')"
              value="resolved" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('等级')"
          ><el-select v-model="q.level" clearable :placeholder="$tr('全部等级')"
            ><el-option
              v-for="level in levels"
              :key="level.value"
              :label="$tr(level.label)"
              :value="level.value" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('来源')"
          ><el-select
            v-model="q.source"
            clearable
            :placeholder="$tr('全部来源')"
            ><el-option
              v-for="source in sources"
              :key="source.value"
              :label="$tr(source.label)"
              :value="source.value" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('告警日期')"
          ><el-date-picker
            v-model="dateRange"
            type="daterange"
            value-format="YYYY-MM-DD"
            :start-placeholder="$tr('开始日期')"
            :end-placeholder="$tr('结束日期')"
        /></el-form-item>
        <el-form-item
          ><el-button type="primary" native-type="submit" icon="Search">{{
            $tr("搜索")
          }}</el-button
          ><el-button @click="resetQuery">{{
            $tr("重置")
          }}</el-button></el-form-item
        >
      </el-form>
      <el-alert
        v-if="listError"
        :title="listError"
        type="error"
        :closable="false"
        show-icon
        class="error-banner"
      />
      <el-table
        v-loading="loading"
        :data="list"
        row-key="id"
        class="incident-table"
      >
        <el-table-column :label="$tr('故障')" min-width="290">
          <template #default="{ row }"
            ><el-button class="incident-title" link @click="showDetail(row)">{{
              row.title
            }}</el-button>
            <div class="incident-description">
              #{{ row.id }} · {{ row.description || $tr("暂无描述") }}
            </div></template
          >
        </el-table-column>
        <el-table-column :label="$tr('等级')" width="90"
          ><template #default="{ row }"
            ><el-tag :type="levelType(row.level)">{{
              levelLabel(row.level)
            }}</el-tag></template
          ></el-table-column
        >
        <el-table-column :label="$tr('来源')" min-width="120"
          ><template #default="{ row }">{{
            sourceLabel(row.source)
          }}</template></el-table-column
        >
        <el-table-column :label="$tr('状态')" width="100"
          ><template #default="{ row }"
            ><el-tag
              :type="row.status === 'firing' ? 'danger' : 'success'"
              effect="plain"
              >{{
                row.status === "firing" ? $tr("活跃") : $tr("已解决")
              }}</el-tag
            ></template
          ></el-table-column
        >
        <el-table-column :label="$tr('告警 / 恢复时间')" min-width="190"
          ><template #default="{ row }"
            ><div>{{ row.alertTime || "—" }}</div>
            <div class="muted">
              {{ row.resolvedAt || $tr("尚未恢复") }}
            </div></template
          ></el-table-column
        >
        <el-table-column :label="$tr('操作')" width="205" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="showDetail(row)">{{
              $tr("详情")
            }}</el-button>
            <el-button
              v-if="row.status === 'firing'"
              link
              type="success"
              :loading="busyId === row.id"
              @click="handleResolve(row)"
              >{{ $tr("解决") }}</el-button
            >
            <el-dropdown trigger="click" @command="handleDelete(row)"
              ><el-button link>{{ $tr("更多") }}</el-button
              ><template #dropdown
                ><el-dropdown-menu
                  ><el-dropdown-item command="delete">{{
                    $tr("删除")
                  }}</el-dropdown-item></el-dropdown-menu
                ></template
              ></el-dropdown
            >
          </template>
        </el-table-column>
        <template #empty
          ><el-empty
            :description="
              listError
                ? $tr('数据加载失败，请重试')
                : $tr('暂无匹配故障，请调整筛选条件或等待告警触发')
            "
        /></template>
      </el-table>
      <div class="pagination">
        <el-pagination
          v-model:current-page="q.page"
          v-model:page-size="q.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next"
          @size-change="handleQuery"
          @current-change="getList"
        />
      </div>
    </section>
    <el-drawer
      v-model="detailVisible"
      class="monitor-overlay"
      :title="$tr('故障详情')"
      size="min(660px, 100vw)"
    >
      <template v-if="detail">
        <div class="detail-heading">
          <el-tag :type="levelType(detail.level)">{{
            levelLabel(detail.level)
          }}</el-tag>
          <h2>{{ detail.title }}</h2>
        </div>
        <el-descriptions :column="1" border>
          <el-descriptions-item :label="$tr('故障编号')"
            >#{{ detail.id }}</el-descriptions-item
          >
          <el-descriptions-item :label="$tr('来源')"
            >{{ sourceLabel(detail.source)
            }}<span v-if="detail.sourceId">
              · #{{ detail.sourceId }}</span
            ></el-descriptions-item
          >
          <el-descriptions-item :label="$tr('状态')">{{
            detail.status === "firing" ? $tr("活跃") : $tr("已解决")
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('描述')"
            ><div class="detail-description">
              {{ detail.description || "—" }}
            </div></el-descriptions-item
          >
        </el-descriptions>
        <h3 class="timeline-title">{{ $tr("处理时间线") }}</h3>
        <el-timeline>
          <el-timeline-item :timestamp="detail.alertTime" type="danger">{{
            $tr("告警触发")
          }}</el-timeline-item>
          <el-timeline-item
            v-if="detail.status === 'resolved'"
            :timestamp="detail.resolvedAt || '—'"
            type="success"
            >{{ $tr("故障已解决") }}</el-timeline-item
          >
          <el-timeline-item v-else type="warning">{{
            $tr("等待恢复或人工解决")
          }}</el-timeline-item>
        </el-timeline>
        <el-button
          v-if="detail.status === 'firing'"
          type="success"
          :loading="busyId === detail.id"
          @click="handleResolve(detail)"
          >{{ $tr("标记为已解决") }}</el-button
        >
      </template>
    </el-drawer>
  </div>
</template>
<script setup>
import "./monitor-theme.css";
import {
  computed,
  onActivated,
  onDeactivated,
  onMounted,
  onUnmounted,
  reactive,
  ref
} from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { tr } from "@/locales";
import {
  getIncidentList,
  getIncidentStats,
  resolveIncident,
  deleteIncident
} from "@/api/monitor";
const stats = ref(null),
  statsLoading = ref(false),
  statsError = ref(""),
  listError = ref("");
const list = ref([]),
  total = ref(0),
  loading = ref(false),
  updatedAt = ref(""),
  autoRefresh = ref(true);
const detail = ref(null),
  detailVisible = ref(false),
  busyId = ref(null),
  dateRange = ref([]);
const q = reactive({
  page: 1,
  pageSize: 20,
  keyword: "",
  status: "",
  level: "",
  source: ""
});
const levels = [
  { value: "critical", label: "严重" },
  { value: "warning", label: "警告" },
  { value: "info", label: "信息" }
];
const sources = [
  { value: "domain_cert", label: "域名证书" },
  { value: "api_endpoint", label: "API 监控" },
  { value: "prometheus", label: "Prometheus" },
  { value: "legacy", label: "历史故障" }
];
const levelLabel = value =>
  tr(levels.find(l => l.value === value)?.label || value);
const sourceLabel = value =>
  tr(sources.find(s => s.value === value)?.label || value || "未知来源");
const levelType = value =>
  ({ critical: "danger", warning: "warning", info: "info" })[value] || "info";
const trendMax = computed(() =>
  Math.max(1, ...(stats.value?.trend || []).map(p => p.count))
);
const trendDescription = computed(() =>
  (stats.value?.trend || []).map(p => `${p.date}: ${p.count}`).join(", ")
);
const sourceDistribution = computed(() => {
  const entries = Object.entries(stats.value?.bySource || {});
  const count = entries.reduce((sum, [, n]) => sum + n, 0);
  return entries.map(([value, n]) => ({
    value,
    count: n,
    percent: count ? (n / count) * 100 : 0
  }));
});
let requestId = 0,
  timer;
async function getList() {
  const current = ++requestId;
  loading.value = true;
  listError.value = "";
  try {
    const { data: res } = await getIncidentList({
      ...q,
      startDate: dateRange.value?.[0] || "",
      endDate: dateRange.value?.[1] || ""
    });
    if (current !== requestId) return;
    if (res.code !== 200) throw new Error(res.message);
    list.value = res.data.list || [];
    total.value = res.data.total || 0;
    if (!list.value.length && total.value && q.page > 1) {
      q.page = Math.ceil(total.value / q.pageSize);
      return getList();
    }
    const latest = list.value.find(row => row.id === detail.value?.id);
    if (latest) detail.value = latest;
  } catch (e) {
    if (current === requestId)
      listError.value = e.message || tr("获取故障列表失败");
  } finally {
    if (current === requestId) loading.value = false;
  }
}
async function loadStats() {
  if (statsLoading.value) return;
  statsLoading.value = true;
  statsError.value = "";
  try {
    const { data: res } = await getIncidentStats();
    if (res.code !== 200) throw new Error(res.message);
    stats.value = res.data;
    updatedAt.value = new Date().toLocaleTimeString();
  } catch (e) {
    statsError.value = e.message || tr("获取统计失败，请刷新重试");
  } finally {
    statsLoading.value = false;
  }
}
async function refresh() {
  await Promise.all([getList(), loadStats()]);
}
function handleQuery() {
  q.page = 1;
  getList();
}
function resetQuery() {
  Object.assign(q, { page: 1, keyword: "", status: "", level: "", source: "" });
  dateRange.value = [];
  getList();
}
function filterStatus(status) {
  q.status = status;
  handleQuery();
}
function showDetail(row) {
  detail.value = row;
  detailVisible.value = true;
}
async function handleResolve(row) {
  if (
    !(await ElMessageBox.confirm(
      tr("确认将该故障标记为已解决？"),
      tr("解决故障"),
      { type: "warning" }
    )
      .then(() => true)
      .catch(() => false))
  )
    return;
  busyId.value = row.id;
  try {
    const { data: res } = await resolveIncident(row.id);
    if (res.code !== 200) throw new Error(res.message);
    detailVisible.value = false;
    ElMessage.success(tr("已解决"));
    await refresh();
  } catch (e) {
    ElMessage.error(e.message || tr("操作失败"));
  } finally {
    busyId.value = null;
  }
}
async function handleDelete(row) {
  if (
    !(await ElMessageBox.confirm(tr("确认删除该故障记录？"), tr("删除故障"), {
      type: "warning"
    })
      .then(() => true)
      .catch(() => false))
  )
    return;
  try {
    const { data: res } = await deleteIncident(row.id);
    if (res.code !== 200) throw new Error(res.message);
    if (detail.value?.id === row.id) detailVisible.value = false;
    ElMessage.success(tr("删除成功"));
    await refresh();
  } catch (e) {
    ElMessage.error(e.message || tr("删除失败"));
  }
}
function startTimer() {
  clearInterval(timer);
  timer = setInterval(() => {
    if (
      autoRefresh.value &&
      !document.hidden &&
      !loading.value &&
      !statsLoading.value
    )
      refresh();
  }, 30000);
}
onMounted(() => {
  refresh();
  startTimer();
});
onActivated(startTimer);
onDeactivated(() => clearInterval(timer));
onUnmounted(() => {
  clearInterval(timer);
  requestId++;
});
</script>
<style scoped>
.incident-page {
  padding: 24px;
  color: var(--text-primary);
  background: var(--bg-page);
  min-height: 100%;
}
.page-header,
.section-heading,
.scope-note {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.page-header {
  margin-bottom: 20px;
}
.eyebrow {
  color: var(--primary);
  font-weight: 700;
  font-size: 11px;
  letter-spacing: 1.6px;
}
h1 {
  font-size: 26px;
  font-weight: 650;
  margin: 6px 0 8px;
}
.page-header p,
.scope-note,
.muted,
.section-heading > span {
  color: var(--text-secondary);
  font-size: 12px;
}
.page-header p {
  font-size: 13px;
  margin: 0;
}
.refresh-actions {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}
.scope-note {
  margin-bottom: 12px;
}
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 18px;
}
.metric {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 14px;
  padding: 20px;
  text-align: left;
  display: flex;
  flex-direction: column;
  gap: 10px;
  color: var(--text-primary);
}
.metric span {
  font-size: 13px;
  color: var(--text-regular);
}
.metric strong {
  font-size: 34px;
  font-weight: 650;
  line-height: 1;
}
.metric small {
  color: var(--text-secondary);
  font-size: 12px;
}
button.metric,
.source-row,
.level-summary button {
  cursor: pointer;
}
.metric-danger strong {
  color: var(--el-color-danger);
}
.metric-success strong {
  color: var(--el-color-success);
}
button.metric:hover {
  border-color: var(--primary);
}
.insights-grid {
  display: grid;
  grid-template-columns: 1.3fr 1fr;
  gap: 18px;
  margin-bottom: 18px;
}
.surface {
  border-radius: 16px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  padding: 22px;
  min-width: 0;
}
.section-heading {
  margin-bottom: 20px;
}
h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
.trend-panel {
  position: relative;
}
.trend-chart {
  display: flex;
  gap: 18px;
  height: 172px;
  padding: 24px 10px 0;
}
.trend-day {
  flex: 1;
  min-width: 0;
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.bar-track {
  height: 120px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: flex-end;
  justify-content: center;
}
.trend-bar {
  position: relative;
  width: 60%;
  max-width: 36px;
  background: var(--primary);
  border-radius: 5px 5px 0 0;
}
.trend-bar span {
  position: absolute;
  top: -22px;
  left: 50%;
  transform: translateX(-50%);
  font-size: 12px;
  color: var(--text-regular);
}
.trend-day time {
  font-size: 11px;
  color: var(--text-secondary);
}
.trend-empty {
  font-size: 12px;
  color: var(--text-secondary);
  margin: 8px 0 0;
  text-align: center;
}
.level-summary {
  display: flex;
  gap: 8px;
  border-bottom: 1px solid var(--border);
  padding-bottom: 18px;
  margin-bottom: 16px;
}
.level-summary button {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 7px;
  font-size: 12px;
  padding: 7px;
  background: var(--bg-card-alt);
  border: 0;
  border-radius: 6px;
  color: var(--text-regular);
}
.level-summary strong {
  margin-left: auto;
  color: var(--text-primary);
}
.level-dot {
  height: 7px;
  width: 7px;
  border-radius: 50%;
  background: var(--text-secondary);
}
.level-dot.critical {
  background: var(--el-color-danger);
}
.level-dot.warning {
  background: var(--el-color-warning);
}
.source-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.source-row {
  display: flex;
  gap: 14px;
  align-items: center;
  background: none;
  border: 0;
  color: var(--text-regular);
  font-size: 12px;
  text-align: left;
  width: 100%;
}
.source-row > span {
  width: 86px;
  flex-shrink: 0;
}
.source-row strong {
  min-width: 22px;
  text-align: right;
}
.source-track {
  flex: 1;
  height: 6px;
  background: var(--bg-card-alt);
  border-radius: 4px;
  overflow: hidden;
}
.source-track i {
  display: block;
  height: 100%;
  background: var(--primary);
  border-radius: 4px;
}
.count-badge {
  margin-left: 8px;
  font-size: 12px;
  color: var(--primary);
  background: var(--bg-card-alt);
  padding: 3px 9px;
  border-radius: 6px;
}
.filters {
  padding: 16px 16px 0;
  background: var(--bg-card-alt);
  border-radius: 10px;
  margin-bottom: 16px;
}
.filters :deep(.el-form-item) {
  max-width: 100%;
}
.filters :deep(.el-form-item__content) {
  min-width: 0;
}
.filters :deep(.el-select) {
  width: 135px;
}
.filters :deep(.el-input) {
  width: 190px;
}
.filters :deep(.el-date-editor) {
  max-width: 290px;
}
:deep(.el-table th.el-table__cell) {
  background: var(--bg-card-alt);
  font-size: 12px;
}
:deep(.el-table td.el-table__cell) {
  padding: 15px 0;
}
.incident-title {
  color: var(--text-primary);
  font-weight: 600;
  max-width: 100%;
  white-space: normal;
  text-align: left;
  height: auto;
}
.incident-description {
  color: var(--text-secondary);
  font-size: 12px;
  margin-top: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
:deep(.el-table .el-dropdown) {
  margin-left: 12px;
  vertical-align: middle;
}
.pagination {
  display: flex;
  justify-content: flex-end;
  overflow-x: auto;
  padding-top: 20px;
}
.error-banner {
  margin-bottom: 16px;
}
.detail-heading h2 {
  margin: 14px 0 24px;
  overflow-wrap: anywhere;
}
.detail-description {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.timeline-title {
  margin: 28px 0 20px;
  font-size: 14px;
}
button:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 3px;
}
@media (max-width: 1100px) {
  .insights-grid {
    grid-template-columns: 1fr;
  }
  .page-header {
    align-items: flex-start;
  }
}
@media (max-width: 768px) {
  .incident-page {
    padding: 14px;
  }
  .filters :deep(.el-form-item) {
    display: flex;
    margin-right: 0;
  }
  .filters :deep(.el-form-item__label) {
    width: 68px;
    justify-content: flex-start;
  }
  .filters :deep(.el-date-editor) {
    width: 100%;
    max-width: 100%;
    min-width: 0;
  }
  .page-header {
    flex-direction: column;
  }
  .stats-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }
  .metric {
    padding: 16px;
  }
  .surface {
    padding: 16px;
  }
  .scope-note {
    flex-wrap: wrap;
  }
  .pagination {
    justify-content: flex-start;
  }
  .trend-chart {
    gap: 10px;
  }
  .section-heading {
    align-items: flex-start;
  }
}
</style>
