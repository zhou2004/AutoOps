<script setup lang="ts">
import {
  computed,
  onActivated,
  onBeforeUnmount,
  onDeactivated,
  onMounted,
  reactive,
  ref,
  watch
} from "vue";
import { useRoute, useRouter } from "vue-router";
import { tr } from "@/locales";
import { webOptions, webRequest, type WebResource } from "@/api/webservice";
import {
  webTraffic,
  type TrafficFilter,
  type TrafficRank,
  type WebTraffic
} from "@/api/webTraffic";
import {
  trafficBytes,
  trafficChart,
  trafficNumber,
  trafficTime
} from "./trafficPresentation";
import BaseChart from "@/views/dashboard/components/BaseChart.vue";
import WebResourceIcon from "./WebResourceIcon.vue";
import TrafficPolicyPanel from "./TrafficPolicyPanel.vue";
import { trafficMessage } from "./trafficMessages";

defineOptions({ name: "WebTraffic" });
const route = useRoute(),
  router = useRouter();
const filters = reactive<TrafficFilter>({
  range: "1h",
  clusterId: Number(route.query.clusterId) || undefined,
  instanceId: Number(route.query.instanceId) || undefined
});
const data = ref<WebTraffic>(),
  clusters = ref<WebResource[]>([]),
  instances = ref<WebResource[]>([]),
  groups = ref<{ id: number; name: string }[]>([]);
const loading = ref(false),
  error = ref(""),
  optionsError = ref(""),
  automatic = ref(true),
  tab = ref("overview"),
  chartMetric = ref<"qps" | "activeConnections" | "averageLatencyMs">("qps"),
  rankTab = ref<"topIPs" | "topURIs" | "upstreams">("topIPs"),
  now = ref(Date.now());
const nodePage = ref(1),
  nodeSize = ref(10),
  rankPage = ref(1),
  rankSize = ref(10);
let generation = 0,
  optionsGeneration = 0,
  abort: AbortController | undefined,
  timer: ReturnType<typeof setInterval> | undefined,
  active = true;
const expired = computed(
  () => !!data.value && now.value - Date.parse(data.value.to) > 90_000
);
const summary = computed(() =>
  expired.value ? undefined : data.value?.summary
);
const availableClusters = computed(() =>
  clusters.value.filter(c => !filters.groupId || c.groupId === filters.groupId)
);
const availableInstances = computed(() =>
  instances.value.filter(
    i =>
      (!filters.groupId || i.groupId === filters.groupId) &&
      (!filters.clusterId || i.clusterId === filters.clusterId)
  )
);
const policyClusterId = computed(
  () =>
    filters.clusterId ||
    instances.value.find(i => i.id === filters.instanceId)?.clusterId ||
    undefined
);
const nodeRows = computed(() =>
  (data.value?.nodes || []).slice(
    (nodePage.value - 1) * nodeSize.value,
    nodePage.value * nodeSize.value
  )
);
const ranking = computed<TrafficRank[]>(
  () => data.value?.[rankTab.value] || []
);
const rankRows = computed(() =>
  ranking.value.slice(
    (rankPage.value - 1) * rankSize.value,
    rankPage.value * rankSize.value
  )
);
const trend = computed(() =>
  trafficChart(
    data.value?.series || [],
    [
      {
        key: chartMetric.value,
        name: {
          qps: "每秒请求数",
          activeConnections: "活跃连接",
          averageLatencyMs: "平均请求耗时"
        }[chartMetric.value]
      }
    ],
    chartMetric.value === "qps"
      ? "req/s"
      : chartMetric.value === "averageLatencyMs"
        ? "ms"
        : ""
  )
);
const bytesTrend = computed(() =>
  trafficChart(
    data.value?.series || [],
    [
      { key: "responseBytesPerSecond", name: "响应流量" },
      { key: "requestBytesPerSecond", name: "请求流量" }
    ],
    "B/s"
  )
);
const statusOption = computed(() => ({
  color: ["#14b8a6", "#1677ff", "#f59e0b", "#ef4444", "#8b5cf6"],
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
      data: (data.value?.statusCodes || []).map(s => ({
        name: s.value,
        value: s.requests
      }))
    }
  ]
}));
const hasTrend = computed(() =>
  data.value?.series.some(point => point[chartMetric.value] != null)
);
const hasBytes = computed(() =>
  data.value?.series.some(
    point =>
      point.responseBytesPerSecond != null ||
      point.requestBytesPerSecond != null
  )
);
async function load() {
  const current = ++generation;
  abort?.abort();
  abort = new AbortController();
  loading.value = true;
  error.value = "";
  try {
    const value = await webTraffic({ ...filters }, abort.signal);
    if (current !== generation) return;
    data.value = value;
    nodePage.value = Math.min(
      nodePage.value,
      Math.max(1, Math.ceil(value.nodes.length / nodeSize.value))
    );
    rankPage.value = Math.min(
      rankPage.value,
      Math.max(
        1,
        Math.ceil((value[rankTab.value] || []).length / rankSize.value)
      )
    );
  } catch (e) {
    if (current === generation) error.value = (e as Error).message;
  } finally {
    if (current === generation) {
      loading.value = false;
      now.value = Date.now();
    }
  }
}
async function loadOptions() {
  const current = ++optionsGeneration;
  optionsError.value = "";
  const results = await Promise.allSettled([
    webOptions("clusters"),
    webOptions("instances"),
    webRequest<{ id: number; name: string }[]>("/groups")
  ]);
  if (current !== optionsGeneration) return;
  if (results[0].status === "fulfilled") clusters.value = results[0].value;
  if (results[1].status === "fulfilled") instances.value = results[1].value;
  if (results[2].status === "fulfilled") groups.value = results[2].value;
  if (results.some(r => r.status === "rejected"))
    optionsError.value = tr("部分筛选选项读取失败，请刷新重试");
}
function changeFilters() {
  data.value = undefined;
  nodePage.value = rankPage.value = 1;
  load();
}
function changeGroup() {
  filters.clusterId = undefined;
  filters.instanceId = undefined;
  changeFilters();
}
function changeCluster() {
  filters.instanceId = undefined;
  changeFilters();
}
function clearFilters() {
  filters.groupId = filters.clusterId = filters.instanceId = undefined;
  filters.range = "1h";
  router.replace({ path: "/web/traffic" });
  changeFilters();
}
function start() {
  active = true;
  if (timer) return;
  timer = setInterval(() => {
    now.value = Date.now();
    if (
      automatic.value &&
      active &&
      !document.hidden &&
      !loading.value &&
      tab.value === "overview"
    )
      load();
  }, 30_000);
}
function stop() {
  active = false;
  clearInterval(timer);
  timer = undefined;
  generation++;
  optionsGeneration++;
  abort?.abort();
  loading.value = false;
}
watch(rankTab, () => {
  rankPage.value = 1;
});
watch(
  () => [route.query.clusterId, route.query.instanceId],
  ([clusterId, instanceId]) => {
    filters.clusterId = Number(clusterId) || undefined;
    filters.instanceId = Number(instanceId) || undefined;
    changeFilters();
  }
);
onMounted(() => {
  loadOptions();
  load();
  start();
});
onActivated(() => {
  if (!active) {
    loadOptions();
    load();
  }
  start();
});
onDeactivated(stop);
onBeforeUnmount(stop);
</script>
<template>
  <div class="asset-page web-traffic-page">
    <header class="asset-page-header">
      <div>
        <span class="asset-eyebrow">{{ $tr("WEB / 流量治理") }}</span>
        <h1><WebResourceIcon />{{ $tr("流量监控") }}</h1>
        <p>{{ $tr("查看真实请求与连接，按站点校验防御和流量分配策略。") }}</p>
      </div>
      <div class="asset-header-count">
        <strong>{{ data ? data.coverage.instances : "—" }}</strong
        ><span>{{ $tr("授权范围内实例") }}</span>
      </div>
    </header>
    <el-card class="asset-surface" shadow="never">
      <div class="asset-filters traffic-filters">
        <el-select
          v-model="filters.groupId"
          clearable
          filterable
          :placeholder="$tr('全部业务分组')"
          :aria-label="$tr('业务分组')"
          @change="changeGroup"
          ><el-option
            v-for="g in groups"
            :key="g.id"
            :value="g.id"
            :label="g.name"
        /></el-select>
        <el-select
          v-model="filters.clusterId"
          clearable
          filterable
          :placeholder="$tr('全部集群')"
          :aria-label="$tr('集群')"
          @change="changeCluster"
          ><el-option
            v-for="c in availableClusters"
            :key="c.id"
            :value="c.id"
            :label="c.name"
        /></el-select>
        <el-select
          v-model="filters.instanceId"
          clearable
          filterable
          :placeholder="$tr('全部实例')"
          :aria-label="$tr('实例')"
          @change="changeFilters"
          ><el-option
            v-for="i in availableInstances"
            :key="i.id"
            :value="i.id"
            :label="i.name"
        /></el-select>
        <el-select
          v-model="filters.range"
          :aria-label="$tr('时间范围')"
          @change="changeFilters"
          ><el-option
            v-for="r in [
              { value: '1h', label: '最近 1 小时' },
              { value: '6h', label: '最近 6 小时' },
              { value: '24h', label: '最近 24 小时' },
              { value: '7d', label: '最近 7 天' }
            ]"
            :key="r.value"
            :value="r.value"
            :label="$tr(r.label)"
        /></el-select>
        <el-button @click="clearFilters">{{ $tr("重置筛选") }}</el-button>
      </div>
      <div class="asset-toolbar">
        <div class="traffic-toolbar-copy">
          <el-tag
            v-if="data"
            :type="data.coverage.partial || expired ? 'warning' : 'success'"
            >{{
              $tr(
                expired
                  ? "数据已过期"
                  : data.coverage.partial
                    ? "部分数据"
                    : "采集正常"
              )
            }}</el-tag
          ><span v-if="data"
            >{{ $tr("最近更新：") }}{{ trafficTime(data.to) }} ·
            {{ $tr("北京时间") }}</span
          >
        </div>
        <div class="traffic-toolbar-actions">
          <el-switch
            v-model="automatic"
            :active-text="$tr('每 30 秒刷新')"
          /><el-button
            :loading="loading"
            @click="
              load();
              loadOptions();
            "
            >{{ $tr("刷新") }}</el-button
          >
        </div>
      </div>
      <el-alert
        v-if="optionsError"
        :title="optionsError"
        type="warning"
        :closable="false"
      />
      <el-alert
        v-if="error"
        :title="$tr('流量数据读取失败')"
        :description="error"
        type="error"
        :closable="false"
      />
      <el-alert
        v-if="expired"
        :title="$tr('当前数据已过期，请刷新后查看实时流量')"
        type="warning"
        :closable="false"
      />
      <el-tabs v-model="tab">
        <el-tab-pane :label="$tr('流量概览')" name="overview">
          <el-skeleton v-if="loading && !data" :rows="12" animated />
          <template v-else-if="data">
            <div class="traffic-metrics">
              <article>
                <span>{{ $tr("每秒请求数") }}</span
                ><strong
                  >{{ trafficNumber(summary?.qps) }}<small>req/s</small></strong
                ><small>{{ $tr("由连续请求计数计算") }}</small>
              </article>
              <article>
                <span>{{ $tr("活跃连接") }}</span
                ><strong>{{
                  trafficNumber(summary?.activeConnections, 0)
                }}</strong
                ><small>{{ $tr("Nginx 状态接口") }}</small>
              </article>
              <article>
                <span>{{ $tr("响应流量") }}</span
                ><strong>{{
                  trafficBytes(summary?.responseBytesPerSecond)
                }}</strong
                ><small
                  :title="
                    $tr('响应正文大小，不包含协议与 TLS 开销，也不是网卡总流量')
                  "
                  >{{ $tr("访问日志响应正文") }}</small
                >
              </article>
              <article>
                <span>{{ $tr("请求流量") }}</span
                ><strong>{{
                  trafficBytes(summary?.requestBytesPerSecond)
                }}</strong
                ><small>{{ $tr("日志需包含请求字节") }}</small>
              </article>
              <article>
                <span>{{ $tr("5xx 错误率") }}</span
                ><strong
                  >{{ trafficNumber(summary?.errorRate)
                  }}<small v-if="summary?.errorRate != null">%</small></strong
                ><small>{{ $tr("基于可识别访问日志") }}</small>
              </article>
              <article>
                <span>{{ $tr("平均请求耗时") }}</span
                ><strong
                  >{{ trafficNumber(summary?.averageLatencyMs)
                  }}<small v-if="summary?.averageLatencyMs != null"
                    >ms</small
                  ></strong
                ><small>{{ $tr("日志需包含请求耗时") }}</small>
              </article>
            </div>
            <div class="traffic-coverage">
              <span
                >{{ $tr("采集覆盖") }}
                {{ expired ? "—" : data.coverage.freshInstances }} /
                {{ data.coverage.instances }}</span
              ><span>stub_status {{ data.coverage.stubStatusInstances }}</span
              ><span
                >{{ $tr("日志采集") }} {{ data.coverage.logInstances }}</span
              ><span
                >{{ $tr("范围内日志请求") }}
                {{ trafficNumber(data.summary.loggedRequests, 0) }}</span
              >
            </div>
            <el-alert
              :title="
                $tr(
                  '连接与请求计数来自状态接口；字节、状态码和来源排行来自授权的可识别日志。缺失指标显示 —，图表缺口表示无有效样本。'
                )
              "
              type="info"
              :closable="false"
            />
            <el-alert
              v-for="(note, i) in data.coverage.notes"
              :key="i"
              :title="trafficMessage(note)"
              type="warning"
              :closable="false"
            />
            <div class="traffic-charts">
              <section class="traffic-panel">
                <div class="traffic-panel-heading">
                  <h2>{{ $tr("请求与连接趋势") }}</h2>
                  <el-select v-model="chartMetric" :aria-label="$tr('趋势指标')"
                    ><el-option
                      value="qps"
                      :label="$tr('每秒请求数')" /><el-option
                      value="activeConnections"
                      :label="$tr('活跃连接')" /><el-option
                      value="averageLatencyMs"
                      :label="$tr('平均请求耗时')"
                  /></el-select>
                </div>
                <BaseChart v-if="hasTrend" :option="trend" /><el-empty
                  v-else
                  :description="$tr('暂无该指标的有效采样')"
                  :image-size="70"
                />
              </section>
              <section class="traffic-panel">
                <div class="traffic-panel-heading">
                  <h2>{{ $tr("流量速率趋势") }}</h2>
                  <span>B/s</span>
                </div>
                <BaseChart v-if="hasBytes" :option="bytesTrend" /><el-empty
                  v-else
                  :description="
                    $tr('暂无字节数据，请检查访问日志格式与采集权限')
                  "
                  :image-size="70"
                />
              </section>
            </div>
            <section class="traffic-panel">
              <div class="traffic-panel-heading">
                <h2>{{ $tr("节点流量") }}</h2>
                <small>{{ $tr("超过 90 秒未更新显示未知") }}</small>
              </div>
              <el-table
                :data="nodeRows"
                row-key="instanceId"
                class="asset-table"
                ><el-table-column :label="$tr('实例')" min-width="240"
                  ><template #default="{ row }"
                    ><div class="traffic-node">
                      <WebResourceIcon />
                      <div>
                        <strong>{{ row.name }}</strong
                        ><small>{{ row.address }}</small>
                      </div>
                    </div></template
                  ></el-table-column
                ><el-table-column :label="$tr('采集状态')" width="120"
                  ><template #default="{ row }"
                    ><el-tag
                      :type="
                        expired || row.status === 'unknown'
                          ? 'info'
                          : row.status === 'healthy'
                            ? 'success'
                            : 'warning'
                      "
                      >{{
                        $tr(
                          expired || row.status === "unknown"
                            ? "未知"
                            : row.status === "healthy"
                              ? "采集正常"
                              : "部分数据"
                        )
                      }}</el-tag
                    ></template
                  ></el-table-column
                ><el-table-column label="QPS" width="100"
                  ><template #default="{ row }">{{
                    trafficNumber(expired ? null : row.metrics.qps)
                  }}</template></el-table-column
                ><el-table-column :label="$tr('活跃连接')" width="110"
                  ><template #default="{ row }">{{
                    trafficNumber(
                      expired ? null : row.metrics.activeConnections,
                      0
                    )
                  }}</template></el-table-column
                ><el-table-column :label="$tr('响应流量')" width="145"
                  ><template #default="{ row }">{{
                    trafficBytes(
                      expired ? null : row.metrics.responseBytesPerSecond
                    )
                  }}</template></el-table-column
                ><el-table-column :label="$tr('最近采集')" min-width="165"
                  ><template #default="{ row }">{{
                    trafficTime(row.collectedAt)
                  }}</template></el-table-column
                ><el-table-column
                  :label="$tr('采集说明')"
                  min-width="220"
                  show-overflow-tooltip
                  ><template #default="{ row }">{{
                    row.message
                      ? trafficMessage(row.message)
                      : $tr("可用指标已更新")
                  }}</template></el-table-column
                ><el-table-column :label="$tr('操作')" width="100"
                  ><template #default="{ row }"
                    ><el-button
                      link
                      type="primary"
                      @click="
                        router.push({
                          path: '/web/logs',
                          query: {
                            clusterId: row.clusterId,
                            instanceId: row.instanceId
                          }
                        })
                      "
                      >{{ $tr("查看日志") }}</el-button
                    ></template
                  ></el-table-column
                ></el-table
              >
              <div class="asset-pagination">
                <el-pagination
                  v-model:current-page="nodePage"
                  v-model:page-size="nodeSize"
                  :page-sizes="[10, 20, 50, 100]"
                  :total="data.nodes.length"
                  layout="total, sizes, prev, pager, next, jumper"
                  @size-change="nodePage = 1"
                />
              </div>
            </section>
            <div class="traffic-charts traffic-breakdown">
              <section class="traffic-panel">
                <div class="traffic-panel-heading">
                  <h2>{{ $tr("HTTP 状态码") }}</h2>
                  <small>{{ $tr("所选时间范围") }}</small>
                </div>
                <BaseChart
                  v-if="data.statusCodes.length"
                  :option="statusOption"
                /><el-empty
                  v-else
                  :description="$tr('暂无可识别的访问日志统计')"
                  :image-size="70"
                />
              </section>
              <section class="traffic-panel">
                <el-tabs v-model="rankTab"
                  ><el-tab-pane
                    :label="$tr('来源 IP')"
                    name="topIPs" /><el-tab-pane
                    :label="$tr('请求路径')"
                    name="topURIs" /><el-tab-pane
                    :label="$tr('上游流向')"
                    name="upstreams" /></el-tabs
                ><el-table :data="rankRows" row-key="value" class="asset-table"
                  ><el-table-column
                    prop="value"
                    :label="
                      $tr(
                        rankTab === 'topIPs'
                          ? '来源 IP'
                          : rankTab === 'topURIs'
                            ? '请求路径'
                            : '上游地址'
                      )
                    "
                    min-width="180"
                    show-overflow-tooltip /><el-table-column
                    :label="$tr('请求数')"
                    width="100"
                    ><template #default="{ row }">{{
                      trafficNumber(row.requests, 0)
                    }}</template></el-table-column
                  ><el-table-column :label="$tr('响应字节')" width="140"
                    ><template #default="{ row }">{{
                      trafficBytes(row.responseBytes, false)
                    }}</template></el-table-column
                  ><template #empty
                    ><el-empty
                      :description="
                        $tr(
                          rankTab === 'upstreams'
                            ? '日志未提供上游信息或尚无代理请求'
                            : '暂无可识别的访问日志统计'
                        )
                      "
                      :image-size="60" /></template
                ></el-table>
                <div class="asset-pagination">
                  <el-pagination
                    v-model:current-page="rankPage"
                    v-model:page-size="rankSize"
                    :page-sizes="[10, 20, 50, 100]"
                    :total="ranking.length"
                    layout="sizes, prev, pager, next, jumper"
                    @size-change="rankPage = 1"
                  />
                </div>
              </section>
            </div>
          </template>
          <el-empty v-else :description="$tr('暂无流量数据')" />
        </el-tab-pane>
        <el-tab-pane :label="$tr('防御与流量分配')" name="policies" lazy
          ><TrafficPolicyPanel
            :cluster-id="policyClusterId"
            :instance-id="filters.instanceId"
        /></el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>
<style scoped>
.web-traffic-page {
  min-width: 0;
}
h1 {
  display: flex;
  align-items: center;
  gap: 12px;
}
.traffic-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}
.traffic-filters .el-select {
  width: 210px;
}
.traffic-toolbar-copy,
.traffic-toolbar-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  font-size: 12px;
  color: var(--text-secondary);
}
.traffic-toolbar-actions .el-button {
  margin: 0;
}
.el-alert {
  margin-bottom: 14px;
}
.traffic-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 16px;
}
.traffic-metrics article {
  min-width: 0;
  padding: 18px;
  border-radius: 10px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
}
.traffic-metrics span,
.traffic-metrics small {
  color: var(--text-secondary);
  font-size: 12px;
}
.traffic-metrics strong {
  display: block;
  color: var(--text-primary);
  font-size: 27px;
  font-weight: 650;
  margin: 10px 0 8px;
  overflow-wrap: anywhere;
}
.traffic-metrics strong small {
  margin-left: 8px;
  font-weight: normal;
}
.traffic-coverage {
  display: flex;
  gap: 18px;
  flex-wrap: wrap;
  color: var(--text-secondary);
  font-size: 12px;
  margin-bottom: 18px;
}
.traffic-charts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
  margin-bottom: 18px;
}
.traffic-panel {
  min-width: 0;
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  padding: 18px;
  margin-bottom: 18px;
}
.traffic-charts .traffic-panel {
  margin-bottom: 0;
}
.traffic-panel-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}
.traffic-panel-heading h2 {
  font-size: 15px;
  margin: 0;
  color: var(--text-primary);
}
.traffic-panel-heading small,
.traffic-panel-heading span {
  color: var(--text-secondary);
  font-size: 12px;
}
.traffic-panel-heading .el-select {
  width: 160px;
}
.traffic-node {
  display: flex;
  align-items: center;
  gap: 12px;
}
.traffic-node small {
  display: block;
  margin-top: 5px;
  color: var(--text-secondary);
}
.asset-pagination {
  display: flex;
  justify-content: center;
  overflow: auto;
}
.asset-pagination :deep(.el-pagination) {
  justify-content: center;
  flex-wrap: wrap;
  row-gap: 10px;
}
@media (max-width: 1100px) {
  .traffic-charts {
    grid-template-columns: minmax(0, 1fr);
  }
}
@media (max-width: 650px) {
  .traffic-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }
  .traffic-metrics article {
    padding: 12px;
  }
  .traffic-metrics strong {
    font-size: 22px;
  }
  .traffic-filters .el-select {
    width: 100%;
  }
  .traffic-panel {
    padding: 12px;
  }
}
</style>
