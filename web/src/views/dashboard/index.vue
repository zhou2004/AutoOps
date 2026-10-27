<script setup lang="ts">
import { locale, tr } from "@/locales";
import {
  computed,
  onActivated,
  onBeforeUnmount,
  onDeactivated,
  onMounted,
  ref
} from "vue";
import { useRouter } from "vue-router";
import {
  getDashboardOverview,
  type DashboardOverview,
  type CountItem,
  type Inventory
} from "@/api/dashboard";
import BaseChart from "./components/BaseChart.vue";

defineOptions({ name: "Dashboard" });
const router = useRouter();
const overview = ref<DashboardOverview | null>(null);
const days = ref(7);
const loading = ref(false);
const error = ref("");
const autoRefresh = ref(true);
const executionTab = ref<"deployments" | "jobs" | "ansible">("deployments");
let timer: ReturnType<typeof setInterval> | undefined;
let requestID = 0;
let active = true;

async function refresh() {
  const id = ++requestID;
  loading.value = true;
  error.value = "";
  try {
    const data = await getDashboardOverview(days.value);
    if (id === requestID) overview.value = data;
  } catch (reason) {
    if (id === requestID)
      error.value =
        reason instanceof Error ? reason.message : "读取失败，请重试";
  } finally {
    if (id === requestID) loading.value = false;
  }
}
function changeRange() {
  // Do not label records from the previous range with a newly selected range.
  overview.value = null;
  refresh();
}
function startRefresh() {
  active = true;
  if (timer) return;
  timer = setInterval(() => {
    if (active && autoRefresh.value && !document.hidden && !loading.value)
      refresh();
  }, 60_000);
}
function stopRefresh() {
  active = false;
  clearInterval(timer);
  timer = undefined;
  requestID++;
  loading.value = false;
}
onMounted(() => {
  refresh();
  startRefresh();
});
onActivated(() => {
  if (!active) refresh();
  startRefresh();
});
onDeactivated(stopRefresh);
onBeforeUnmount(stopRefresh);

const count = (value?: number | null) =>
  value == null ? "—" : value.toLocaleString(locale.value);
const timeText = (value?: string) =>
  value
    ? new Intl.DateTimeFormat(locale.value, {
        timeZone: "Asia/Shanghai",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        hour12: false
      }).format(new Date(value))
    : tr("尚未更新");
const statusCount = (value: Inventory | null | undefined, name: string) =>
  value?.statuses.find(item => item.name === name)?.count ?? 0;
const issues = computed(() => Object.values(overview.value?.errors ?? {}));
const assetCards = computed(() => [
  {
    key: "hosts",
    label: "管理主机",
    icon: "主",
    route: "/cmdb/ecs",
    data: overview.value?.hosts,
    note: "主机与云服务器"
  },
  {
    key: "databases",
    label: "数据库实例",
    icon: "库",
    route: "/database/sources",
    data: overview.value?.databases,
    note: "已登记数据库"
  },
  {
    key: "physical",
    label: "物理机",
    icon: "物",
    route: "/cmdb/physical",
    data: overview.value?.physical,
    note: "物理资产台账"
  },
  {
    key: "network",
    label: "网络设备",
    icon: "网",
    route: "/cmdb/network",
    data: overview.value?.network,
    note: "网络资产台账"
  },
  {
    key: "clusters",
    label: "Kubernetes 集群",
    icon: "集",
    route: "/k8s/list",
    data: overview.value?.clusters,
    note: "按已保存的集群状态统计"
  },
  {
    key: "applications",
    label: "业务应用",
    icon: "应",
    route: "/app/application",
    data: overview.value?.applications,
    note: "全平台已登记应用"
  }
]);
const assetDistribution = computed<CountItem[]>(() =>
  assetCards.value
    .filter(item => item.key !== "applications" && item.data)
    .map(item => ({ name: item.label, count: item.data!.total }))
);
const assetTotal = computed(() =>
  assetDistribution.value.reduce((sum, item) => sum + item.count, 0)
);
const assetUnavailable = computed(() =>
  assetCards.value.slice(0, 5).some(item => !item.data)
);
const assetDetails = computed(() => [
  {
    name: "主机来源",
    data: overview.value?.hosts,
    items: overview.value?.hosts?.distribution
  },
  {
    name: "数据库类型",
    data: overview.value?.databases,
    items: overview.value?.databases?.distribution
  },
  {
    name: "主机认证",
    data: overview.value?.hosts,
    items: overview.value?.hosts?.statuses
  },
  {
    name: "集群状态",
    data: overview.value?.clusters,
    items: overview.value?.clusters?.statuses
  }
]);
const execution = computed(() => overview.value?.[executionTab.value]);
const executionRoute = computed(
  () =>
    ({
      deployments: "/app/quick-release",
      jobs: "/task/job",
      ansible: "/task/ansible/history"
    })[executionTab.value]
);
const executionMetrics = computed(() => [
  { label: "执行总数", value: count(execution.value?.total) },
  { label: "成功", value: count(execution.value?.success) },
  { label: "失败", value: count(execution.value?.failed) },
  { label: "运行中", value: count(execution.value?.running) },
  { label: "等待中", value: count(execution.value?.pending) },
  {
    label: "完成成功率",
    value:
      execution.value?.successRate == null
        ? "—"
        : `${execution.value.successRate}%`
  }
]);
const monitors = computed(() => [
  {
    title: "Agent 心跳",
    unit: "个监控对象",
    data: overview.value?.agents,
    route: "/tools/agent",
    note: "授权主机的最新 Agent · 5 分钟心跳窗口"
  },
  {
    title: "API 端点",
    unit: "个监控对象",
    data: overview.value?.endpoints,
    route: "/monitor/api-endpoint",
    note: "全平台端点 · 最近一次检查结果"
  },
  {
    title: "域名证书",
    unit: "个监控对象",
    data: overview.value?.certificates,
    route: "/monitor/domain",
    note: "全平台证书 · 最近一次检查结果"
  },
  {
    title: "告警记录",
    unit: "条留存记录",
    data: overview.value?.alerts,
    route: "/monitor/alarm/history",
    note: "告警中心留存记录 · 与故障记录独立统计"
  }
]);
const palette = [
  "#1677ff",
  "#14b8a6",
  "#8b5cf6",
  "#f59e0b",
  "#06b6d4",
  "#ec4899",
  "#64748b"
];
function pieOption(items: CountItem[]) {
  return {
    color: palette,
    tooltip: {
      trigger: "item",
      renderMode: "richText",
      formatter: "{b}: {c} ({d}%)"
    },
    legend: { bottom: 0, type: "scroll" },
    series: [
      {
        type: "pie",
        radius: ["48%", "72%"],
        center: ["50%", "43%"],
        label: { show: false },
        emphasis: { label: { show: true } },
        data: items
          .filter(item => item.count > 0)
          .map(item => ({ name: tr(item.name), value: item.count }))
      }
    ]
  };
}
const businessOption = computed(() => ({
  color: [palette[0]],
  tooltip: {
    trigger: "axis",
    renderMode: "richText",
    axisPointer: { type: "shadow" }
  },
  grid: { left: 12, right: 32, top: 12, bottom: 16, containLabel: true },
  xAxis: { type: "value", minInterval: 1 },
  yAxis: {
    type: "category",
    inverse: true,
    data:
      overview.value?.applications?.distribution.map(item => item.name) ?? [],
    axisLabel: { width: 110, overflow: "truncate" }
  },
  series: [
    {
      type: "bar",
      barMaxWidth: 18,
      itemStyle: { borderRadius: [0, 4, 4, 0] },
      label: { show: true, position: "right" },
      data:
        overview.value?.applications?.distribution.map(item => item.count) ?? []
    }
  ]
}));
const executionOption = computed(() => ({
  color: [palette[0], "#22c55e", "#ef4444"],
  tooltip: { trigger: "axis", renderMode: "richText" },
  legend: { bottom: 0 },
  grid: { left: 12, right: 20, top: 20, bottom: 42, containLabel: true },
  xAxis: {
    type: "category",
    boundaryGap: false,
    data: execution.value?.trend.map(item => item.date.slice(5)) ?? []
  },
  yAxis: { type: "value", minInterval: 1 },
  series: [
    { name: "执行", key: "total" },
    { name: "成功", key: "success" },
    { name: "失败", key: "failed" }
  ].map(item => ({
    name: tr(item.name),
    type: "line",
    symbolSize: 6,
    data:
      execution.value?.trend.map(
        day => day[item.key as "total" | "success" | "failed"]
      ) ?? []
  }))
}));
const incidentOption = computed(() => ({
  color: ["#f59e0b"],
  tooltip: { trigger: "axis", renderMode: "richText" },
  grid: { left: 12, right: 16, top: 12, bottom: 20, containLabel: true },
  xAxis: {
    type: "category",
    data: overview.value?.incidents?.trend.map(item => item.name.slice(5)) ?? []
  },
  yAxis: { type: "value", minInterval: 1 },
  series: [
    {
      name: tr("新增故障"),
      type: "bar",
      barMaxWidth: 20,
      data: overview.value?.incidents?.trend.map(item => item.count) ?? []
    }
  ]
}));
function statusName(status: number) {
  const labels: Record<number, string> = {
    1: "等待中",
    2: "运行中",
    3: "成功",
    4: "失败",
    5: executionTab.value === "deployments" ? "已取消" : "已暂停",
    6: "已停止"
  };
  return tr(labels[status] ?? "未知状态");
}
function statusType(status: number): "success" | "danger" | "warning" | "info" {
  return status === 3
    ? "success"
    : status === 4
      ? "danger"
      : status === 2
        ? "warning"
        : "info";
}
function monitorType(name: string): "success" | "danger" | "warning" | "info" {
  if (["正常", "心跳正常", "运行中", "认证成功"].includes(name))
    return "success";
  if (
    [
      "异常",
      "已过期",
      "部署失败",
      "启动异常",
      "检查失败",
      "认证失败",
      "离线"
    ].includes(name)
  )
    return "danger";
  if (["即将过期", "超时", "心跳过期 / 未上报"].includes(name))
    return "warning";
  return "info";
}
const sourceName = (source: string) =>
  ({
    domain_cert: "证书监控",
    api_endpoint: "API 监控",
    prometheus: "Prometheus",
    legacy: "故障台账"
  })[source] ?? source;
const levelName = (level: string) =>
  ({ critical: "严重", warning: "警告", info: "提示" })[level] ??
  (level || "未分级");
const levelType = (level: string): "danger" | "warning" | "info" =>
  ["critical", "P1", "P2"].includes(level)
    ? "danger"
    : ["warning", "P3"].includes(level)
      ? "warning"
      : "info";
</script>

<template>
  <div class="business-dashboard">
    <header class="dashboard-header">
      <div>
        <div class="eyebrow">
          <span class="brand-dot" /> {{ $tr("AUTOOPS / 运营总览") }}
        </div>
        <h1>{{ $tr("业务运维仪表盘") }}</h1>
        <p>
          {{
            $tr("掌握资产规模、交付进展与运行风险，让每一项数据都有业务依据。")
          }}
        </p>
      </div>
      <div class="header-actions">
        <el-switch
          v-model="autoRefresh"
          :active-text="$tr('每分钟刷新')"
        /><el-button :loading="loading" type="primary" @click="refresh">{{
          $tr("刷新数据")
        }}</el-button>
      </div>
    </header>
    <div class="scope-bar">
      <span
        >{{ $tr("资产范围：")
        }}{{
          overview
            ? overview.assetScope === "all"
              ? $tr("全部资产")
              : $tr("当前账号授权资产")
            : $tr("读取中")
        }}</span
      >
      <span>{{ $tr("业务与监控：全平台汇总") }}</span
      ><span class="updated-time"
        >{{ $tr("最近更新：") }}{{ timeText(overview?.generatedAt) }}
        {{ $tr("· 北京时间") }}</span
      >
    </div>
    <el-alert
      v-if="error"
      type="error"
      :closable="false"
      show-icon
      :title="
        overview
          ? `${tr('刷新失败，当前保留')} ${timeText(overview.generatedAt)}`
          : $tr('仪表盘加载失败')
      "
      :description="$tr(error)"
    />
    <el-alert
      v-if="issues.length"
      type="warning"
      :closable="false"
      show-icon
      :title="$tr('部分模块暂不可用')"
      :description="$tr(issues.join('；'))"
    />
    <el-skeleton
      v-if="loading && !overview"
      :rows="12"
      animated
      class="surface initial-loading"
    />
    <el-empty v-else-if="!overview" :description="$tr('暂时无法读取业务数据')"
      ><el-button type="primary" @click="refresh">{{
        $tr("重新加载")
      }}</el-button></el-empty
    >
    <template v-else>
      <section class="asset-cards" :aria-label="$tr('资产概览')">
        <button
          v-for="item in assetCards"
          :key="item.key"
          class="asset-card"
          @click="router.push(item.route)"
        >
          <div class="card-top">
            <span>{{ $tr(item.label) }}</span
            ><span class="asset-icon">{{ tr(item.icon) }}</span>
          </div>
          <strong>{{ count(item.data?.total) }}</strong>
          <div class="card-note">
            {{ item.data ? tr(item.note) : $tr("数据暂不可用") }}
            <span>↗</span>
          </div>
        </button>
      </section>
      <div class="two-columns">
        <section class="surface">
          <div class="panel-heading">
            <div>
              <h2>{{ $tr("资产构成") }}</h2>
              <p>{{ $tr("CMDB 台账与 Kubernetes 集群") }}</p>
            </div>
            <el-tag effect="plain">{{
              assetUnavailable
                ? $tr("部分数据")
                : `${count(assetTotal)} ${tr("条资产记录")}`
            }}</el-tag>
          </div>
          <BaseChart v-if="assetTotal" :option="pieOption(assetDistribution)" />
          <el-empty
            v-else
            :description="
              assetUnavailable
                ? $tr('部分资产数据暂不可用')
                : $tr('暂无可见资产，可前往资产管理登记或申请授权')
            "
            :image-size="80"
          />
          <div class="asset-breakdown">
            <div
              v-for="detail in assetDetails"
              :key="detail.name"
              class="breakdown-row"
            >
              <span>{{ tr(detail.name) }}</span>
              <div v-if="detail.items?.length">
                <el-tag
                  v-for="item in detail.items"
                  :key="item.name"
                  size="small"
                  type="info"
                  effect="plain"
                  >{{ tr(item.name) }} {{ count(item.count) }}</el-tag
                >
              </div>
              <small v-else>{{
                detail.data ? $tr("暂无记录") : $tr("数据暂不可用")
              }}</small>
            </div>
          </div>
        </section>
        <section class="surface">
          <div class="panel-heading">
            <div>
              <h2>{{ $tr("业务应用分布") }}</h2>
              <p>{{ $tr("按所属业务组统计 · 前 10 组，其余合并") }}</p>
            </div>
            <el-button
              text
              type="primary"
              @click="router.push('/app/application')"
              >{{ $tr("应用管理 ↗") }}</el-button
            >
          </div>
          <div class="application-summary">
            <span
              >{{ $tr("已激活") }}
              <b>{{
                overview.applications
                  ? count(statusCount(overview.applications, "已激活"))
                  : "—"
              }}</b></span
            ><span
              >{{ $tr("未激活") }}
              <b>{{
                overview.applications
                  ? count(statusCount(overview.applications, "未激活"))
                  : "—"
              }}</b></span
            >
          </div>
          <BaseChart
            v-if="overview.applications?.total"
            :option="businessOption"
            size="lg"
          />
          <el-empty
            v-else
            :description="
              overview.applications
                ? $tr('暂无业务应用，请在应用管理中登记')
                : $tr('应用数据暂不可用')
            "
            :image-size="80"
          />
        </section>
      </div>
      <div class="activity-heading">
        <div>
          <h2>{{ $tr("交付与运行动态") }}</h2>
          <p>
            {{
              $tr(
                "执行记录和新增故障按所选日期统计；资产与待处理故障为当前快照。"
              )
            }}
          </p>
        </div>
        <div class="range-controls">
          <el-radio-group
            v-model="days"
            size="small"
            :disabled="loading"
            @change="changeRange"
            ><el-radio-button :value="7">{{ $tr("近 7 天") }}</el-radio-button
            ><el-radio-button :value="30">{{ $tr("近 30 天") }}</el-radio-button
            ><el-radio-button :value="90">{{
              $tr("近 90 天")
            }}</el-radio-button></el-radio-group
          ><small
            >{{ overview.startDate }} {{ $tr("至") }}
            {{ overview.endDate }}</small
          >
        </div>
      </div>
      <section class="surface">
        <div class="panel-heading">
          <div>
            <h2>{{ $tr("发布与自动化执行") }}</h2>
            <p>
              {{
                $tr("按执行批次统计，成功率 = 成功 /（成功 + 失败） · 累计留存")
              }}
              {{ count(execution?.allTimeTotal) }} {{ $tr("次") }}
            </p>
          </div>
          <el-button text type="primary" @click="router.push(executionRoute)">{{
            $tr("查看记录 ↗")
          }}</el-button>
        </div>
        <el-tabs v-model="executionTab"
          ><el-tab-pane
            :label="$tr('快速发布')"
            name="deployments" /><el-tab-pane
            :label="$tr('任务作业')"
            name="jobs" /><el-tab-pane label="Ansible" name="ansible"
        /></el-tabs>
        <div class="execution-metrics">
          <div v-for="item in executionMetrics" :key="item.label">
            <span>{{ $tr(item.label) }}</span
            ><strong>{{ item.value }}</strong>
          </div>
        </div>
        <p v-if="execution?.other" class="muted">
          {{ $tr("另有") }}
          {{ count(execution.other) }}
          {{ $tr("次取消、暂停、停止或其他状态，未计入成功率。") }}
        </p>
        <div v-if="execution" class="two-columns execution-content">
          <div>
            <h3>{{ $tr("每日执行趋势") }}</h3>
            <BaseChart
              v-if="execution.total"
              :option="executionOption"
            /><el-empty
              v-else
              :description="$tr('所选日期没有执行记录，执行任务后将自动汇总')"
              :image-size="72"
            />
          </div>
          <div>
            <h3>
              {{ $tr("最近执行") }}
              <small>{{ $tr("全部留存记录 · 最近 8 条") }}</small>
            </h3>
            <el-table
              :data="execution.recent"
              row-key="id"
              class="records-table"
            >
              <el-table-column
                prop="name"
                :label="$tr('名称')"
                min-width="140"
                show-overflow-tooltip
              />
              <el-table-column :label="$tr('状态')" width="90"
                ><template #default="{ row }"
                  ><el-tag :type="statusType(row.status)" size="small">{{
                    statusName(row.status)
                  }}</el-tag></template
                ></el-table-column
              >
              <el-table-column :label="$tr('创建时间')" min-width="150"
                ><template #default="{ row }">{{
                  timeText(row.createdAt)
                }}</template></el-table-column
              >
              <template #empty
                ><span class="muted">{{
                  $tr("暂无执行历史记录")
                }}</span></template
              >
            </el-table>
          </div>
        </div>
        <el-empty
          v-else
          :description="$tr('执行数据暂不可用，请稍后刷新')"
          :image-size="72"
        />
      </section>
      <section class="monitor-grid" :aria-label="$tr('监控概览')">
        <div
          v-for="item in monitors"
          :key="item.title"
          class="surface monitor-card"
        >
          <div class="panel-heading">
            <h2>{{ $tr(item.title) }}</h2>
            <el-button text type="primary" @click="router.push(item.route)">{{
              $tr("查看 ↗")
            }}</el-button>
          </div>
          <div class="monitor-total">
            {{ count(item.data?.total) }} <small>{{ tr(item.unit) }}</small>
          </div>
          <p class="muted">{{ tr(item.note) }}</p>
          <div v-if="item.data?.statuses.length" class="status-list">
            <div v-for="status in item.data.statuses" :key="status.name">
              <el-tag
                :type="monitorType(status.name)"
                size="small"
                effect="plain"
                >{{ tr(status.name) }}</el-tag
              ><b>{{ count(status.count) }}</b>
            </div>
          </div>
          <el-empty
            v-else
            :description="item.data ? $tr('暂无记录') : $tr('数据暂不可用')"
            :image-size="48"
          />
        </div>
      </section>
      <section class="surface">
        <div class="panel-heading">
          <div>
            <h2>{{ $tr("故障与待处理事项") }}</h2>
            <p>{{ $tr("当前未完成或未恢复记录 · 优先展示严重故障") }}</p>
          </div>
          <el-button
            text
            type="primary"
            @click="router.push('/monitor/incident')"
            >{{ $tr("故障中心 ↗") }}</el-button
          >
        </div>
        <template v-if="overview.incidents">
          <div class="incident-summary">
            <span
              >{{ $tr("待处理") }}
              <b>{{ count(overview.incidents.firing) }}</b></span
            >
            <el-tag
              v-for="level in overview.incidents.levels"
              :key="level.name"
              :type="levelType(level.name)"
              effect="plain"
            >
              {{ tr(levelName(level.name)) }} {{ count(level.count) }}
            </el-tag>
            <span class="muted"
              >{{ $tr("所选日期新增") }}
              {{ count(overview.incidents.periodTotal) }} {{ $tr("起") }}</span
            >
          </div>
          <div class="two-columns execution-content">
            <div>
              <h3>{{ $tr("新增故障趋势") }}</h3>
              <BaseChart
                v-if="overview.incidents.periodTotal"
                :option="incidentOption"
              /><el-empty
                v-else
                :description="$tr('所选日期没有新增故障记录')"
                :image-size="72"
              />
            </div>
            <div>
              <h3>
                {{ $tr("当前待处理") }} <small>{{ $tr("最多 8 条") }}</small>
              </h3>
              <el-table
                :data="overview.incidents.recent"
                row-key="id"
                class="records-table"
              >
                <el-table-column
                  :label="$tr('故障')"
                  min-width="180"
                  show-overflow-tooltip
                  ><template #default="{ row }"
                    ><div>{{ row.title }}</div>
                    <small class="muted"
                      >{{ tr(sourceName(row.source)) }} ·
                      {{ timeText(row.createdAt) }}</small
                    ></template
                  ></el-table-column
                >
                <el-table-column :label="$tr('级别')" width="90"
                  ><template #default="{ row }"
                    ><el-tag :type="levelType(row.level)" size="small">{{
                      tr(levelName(row.level))
                    }}</el-tag></template
                  ></el-table-column
                >
                <template #empty
                  ><span class="muted">{{
                    $tr("当前没有未恢复的故障记录")
                  }}</span></template
                >
              </el-table>
            </div>
          </div>
        </template>
        <el-empty
          v-else
          :description="$tr('故障数据暂不可用，请稍后刷新')"
          :image-size="72"
        />
      </section>
      <footer>
        {{
          $tr(
            "数据来自业务台账、执行历史与最近监控结果。主机认证、Agent 心跳和集群保存状态分别统计。"
          )
        }}
      </footer>
    </template>
  </div>
</template>

<style scoped>
.business-dashboard {
  padding: 8px 6px 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  color: var(--text-primary);
}
.dashboard-header,
.panel-heading,
.activity-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.dashboard-header {
  padding: 12px 0 4px;
}
.eyebrow {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 11px;
  letter-spacing: 1.6px;
  font-weight: 600;
  color: var(--text-secondary);
}
.brand-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--primary);
}
h1 {
  font-size: 26px;
  line-height: 1.4;
  margin: 10px 0 6px;
  font-weight: 650;
  letter-spacing: -0.5px;
}
h2 {
  font-size: 16px;
  font-weight: 600;
  margin: 0;
}
h3 {
  font-size: 13px;
  font-weight: 600;
  margin: 6px 0 16px;
}
h3 small {
  margin-left: 8px;
  font-size: 11px;
  color: var(--text-secondary);
  font-weight: 400;
}
p {
  margin: 5px 0 0;
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.7;
}
.header-actions,
.scope-bar,
.incident-summary {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
}
.scope-bar {
  padding: 12px 16px;
  border: 1px solid var(--border);
  border-radius: 10px;
  font-size: 11px;
  color: var(--text-secondary);
  background: var(--bg-card);
  gap: 8px 20px;
}
.updated-time {
  margin-left: auto;
}
.surface,
.asset-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 14px;
  box-shadow: 0 2px 4px rgb(0 0 0 / 2%);
}
.surface {
  padding: 22px;
  min-width: 0;
}
.initial-loading {
  padding: 24px;
}
.asset-cards {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 14px;
}
.asset-card {
  text-align: left;
  padding: 18px;
  cursor: pointer;
  color: var(--text-primary);
  transition:
    border-color 0.2s,
    transform 0.2s;
  font: inherit;
}
.asset-card:hover {
  border-color: var(--primary);
  transform: translateY(-2px);
}
.asset-card:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 3px;
}
.card-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-regular);
}
.asset-icon {
  flex: 0 0 auto;
  border-radius: 8px;
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  color: var(--primary);
  background: var(--bg-card-alt, var(--bg-page));
  font-size: 12px;
}
.asset-card strong {
  display: block;
  font-size: 32px;
  font-weight: 650;
  font-variant-numeric: tabular-nums;
  line-height: 1.6;
  margin: 8px 0;
}
.card-note {
  display: flex;
  justify-content: space-between;
  gap: 5px;
  font-size: 10px;
  line-height: 1.5;
  color: var(--text-secondary);
}
.two-columns {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}
.two-columns > * {
  min-width: 0;
}
.panel-heading {
  margin-bottom: 18px;
  align-items: flex-start;
}
.panel-heading :deep(.el-button) {
  margin-top: -4px;
  padding-right: 0;
}
.asset-breakdown {
  border-top: 1px solid var(--border);
  padding-top: 12px;
}
.breakdown-row {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 6px 0;
  font-size: 11px;
  color: var(--text-secondary);
}
.breakdown-row > span {
  min-width: 60px;
  line-height: 22px;
}
.breakdown-row > div {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}
.application-summary {
  display: flex;
  gap: 24px;
  padding-bottom: 12px;
  font-size: 12px;
  color: var(--text-secondary);
}
.application-summary b {
  font-size: 18px;
  color: var(--text-primary);
  margin-left: 8px;
}
.activity-heading {
  margin-top: 8px;
}
.range-controls {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: flex-end;
  flex-shrink: 0;
}
.range-controls small {
  font-size: 10px;
  color: var(--text-secondary);
}
.execution-metrics {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 14px;
  padding: 16px;
  background: var(--bg-card-alt, var(--bg-page));
  border-radius: 10px;
}
.execution-metrics span {
  display: block;
  font-size: 11px;
  color: var(--text-secondary);
  margin-bottom: 8px;
}
.execution-metrics strong {
  font-size: 24px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.execution-content {
  margin-top: 22px;
  gap: 28px;
}
.muted {
  font-size: 11px;
  color: var(--text-secondary);
  line-height: 1.7;
}
.records-table {
  font-size: 12px;
}
.records-table :deep(.el-table__cell) {
  padding: 12px 0;
}
.monitor-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 20px;
}
.monitor-total {
  font-size: 30px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.monitor-total small {
  font-size: 11px;
  color: var(--text-secondary);
  font-weight: 400;
}
.status-list {
  display: grid;
  gap: 12px;
  margin-top: 20px;
}
.status-list > div {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.status-list b {
  font-size: 13px;
}
.incident-summary {
  border-radius: 10px;
  padding: 14px 16px;
  background: var(--bg-card-alt, var(--bg-page));
  font-size: 12px;
  gap: 12px;
}
.incident-summary b {
  font-size: 22px;
  padding: 0 8px;
}
footer {
  text-align: center;
  font-size: 11px;
  color: var(--text-secondary);
  line-height: 1.8;
}
@media (max-width: 1500px) {
  .asset-cards {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
@media (max-width: 1050px) {
  .two-columns {
    grid-template-columns: 1fr;
  }
  .monitor-grid {
    grid-template-columns: 1fr;
  }
  .dashboard-header {
    align-items: flex-start;
  }
}
@media (max-width: 650px) {
  .business-dashboard {
    padding: 0 0 16px;
    gap: 14px;
  }
  .dashboard-header,
  .activity-heading {
    flex-direction: column;
    align-items: flex-start;
  }
  h1 {
    font-size: 23px;
  }
  .asset-cards {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }
  .asset-card {
    padding: 14px;
  }
  .surface {
    padding: 16px;
  }
  .execution-metrics {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
  .range-controls {
    align-items: flex-start;
  }
  .updated-time {
    margin-left: 0;
  }
  .panel-heading {
    gap: 8px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .asset-card {
    transition: none;
  }
  .asset-card:hover {
    transform: none;
  }
}
</style>
