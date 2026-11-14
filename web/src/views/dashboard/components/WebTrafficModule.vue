<script setup lang="ts">
import {
  computed,
  onActivated,
  onBeforeUnmount,
  onDeactivated,
  onMounted,
  ref,
  watch
} from "vue";
import { useRouter } from "vue-router";
import { canPlatform } from "@/utils/platformAccess";
import { tr } from "@/locales";
import { webTraffic, type WebTraffic } from "@/api/webTraffic";
import {
  trafficBytes,
  trafficChart,
  trafficNumber,
  trafficTime
} from "@/views/webservice/trafficPresentation";
import WebResourceIcon from "@/views/webservice/WebResourceIcon.vue";
import BaseChart from "./BaseChart.vue";
const router = useRouter();
const allowed = computed(() => canPlatform("web:read"));
const data = ref<WebTraffic>();
const loading = ref(false),
  error = ref(""),
  automatic = ref(true),
  now = ref(Date.now());
let abort: AbortController | undefined,
  timer: ReturnType<typeof setInterval> | undefined,
  generation = 0,
  active = true;
const expired = computed(
  () => !!data.value && now.value - Date.parse(data.value.to) > 90_000
);
const metrics = computed(() =>
  expired.value ? undefined : data.value?.summary
);
const option = computed(() =>
  trafficChart(
    data.value?.series || [],
    [{ key: "qps", name: "每秒请求数" }],
    "req/s"
  )
);
async function load() {
  if (!allowed.value) return;
  const current = ++generation;
  abort?.abort();
  abort = new AbortController();
  loading.value = true;
  error.value = "";
  try {
    const value = await webTraffic({ range: "1h" }, abort.signal);
    if (current === generation) data.value = value;
  } catch (e) {
    if (current === generation) error.value = (e as Error).message;
  } finally {
    if (current === generation) {
      loading.value = false;
      now.value = Date.now();
    }
  }
}
function start() {
  active = true;
  if (timer) return;
  timer = setInterval(() => {
    now.value = Date.now();
    if (active && automatic.value && !document.hidden && !loading.value) load();
  }, 30_000);
}
function stop() {
  active = false;
  clearInterval(timer);
  timer = undefined;
  generation++;
  abort?.abort();
  loading.value = false;
}
onMounted(() => {
  start();
});
watch(
  allowed,
  value => {
    generation++;
    abort?.abort();
    data.value = undefined;
    loading.value = false;
    error.value = "";
    if (value && active) load();
  },
  { immediate: true }
);
onActivated(() => {
  if (!active) load();
  start();
});
onDeactivated(stop);
onBeforeUnmount(stop);
</script>
<template>
  <section
    v-if="allowed"
    class="web-traffic-module"
    :aria-label="$tr('Web 流量概览')"
  >
    <div class="web-traffic-heading">
      <div>
        <h2><WebResourceIcon />{{ $tr("Web 流量概览") }}</h2>
        <p>{{ $tr("当前授权实例 · 最近 1 小时趋势") }}</p>
      </div>
      <div class="web-traffic-actions">
        <el-switch
          v-model="automatic"
          :active-text="$tr('每 30 秒刷新')"
        /><el-button :loading="loading" @click="load">{{
          $tr("刷新")
        }}</el-button
        ><el-button type="primary" plain @click="router.push('/web/traffic')"
          >{{ $tr("流量监控") }} ↗</el-button
        >
      </div>
    </div>
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
    <el-skeleton v-if="loading && !data" :rows="4" animated />
    <template v-else-if="data">
      <div class="web-traffic-cards">
        <div>
          <span>{{ $tr("每秒请求数") }}</span
          ><strong
            >{{ trafficNumber(metrics?.qps) }}<small>req/s</small></strong
          >
        </div>
        <div>
          <span>{{ $tr("活跃连接") }}</span
          ><strong>{{ trafficNumber(metrics?.activeConnections, 0) }}</strong>
        </div>
        <div>
          <span>{{ $tr("响应流量") }}</span
          ><strong>{{ trafficBytes(metrics?.responseBytesPerSecond) }}</strong>
        </div>
        <div>
          <span>{{ $tr("采集覆盖") }}</span
          ><strong
            >{{ expired ? "—" : data.coverage.freshInstances
            }}<small
              >/ {{ data.coverage.instances }} {{ $tr("实例") }}</small
            ></strong
          >
        </div>
      </div>
      <BaseChart
        v-if="data.series.some(p => p.qps != null)"
        :option="option"
        size="sm"
      />
      <el-empty
        v-else
        :description="$tr('尚无请求速率样本，等待连续采集或配置状态接口')"
        :image-size="60"
      />
      <div class="web-traffic-foot">
        <span
          >{{ $tr("最近更新：") }}{{ trafficTime(data.to) }} ·
          {{ $tr("北京时间") }}</span
        ><el-tag
          :type="data.coverage.partial || expired ? 'warning' : 'success'"
          >{{
            $tr(data.coverage.partial || expired ? "部分数据" : "采集正常")
          }}</el-tag
        ><small>{{
          $tr(
            "响应流量为日志记录的响应正文，不包含协议与 TLS 开销；缺失指标显示 —"
          )
        }}</small>
      </div>
    </template>
    <el-empty v-else :description="$tr('暂无流量数据')" :image-size="60" />
  </section>
</template>
<style scoped>
.web-traffic-module {
  padding: 24px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 12px;
  min-width: 0;
}
.web-traffic-heading,
.web-traffic-actions,
.web-traffic-foot {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.web-traffic-heading {
  justify-content: space-between;
  margin-bottom: 20px;
}
h2 {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 17px;
  margin: 0;
  color: var(--text-primary);
}
p,
.web-traffic-foot,
.web-traffic-cards span,
small {
  color: var(--text-secondary);
  font-size: 12px;
}
p {
  margin: 8px 0 0;
}
.web-traffic-actions .el-button {
  margin: 0;
}
.web-traffic-cards {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}
.web-traffic-cards > div {
  padding: 18px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  min-width: 0;
}
.web-traffic-cards strong {
  display: block;
  margin-top: 8px;
  font-size: 25px;
  font-weight: 650;
  color: var(--text-primary);
  overflow-wrap: anywhere;
}
.web-traffic-cards small {
  margin-left: 6px;
  font-weight: normal;
}
.web-traffic-foot {
  margin-top: 14px;
}
.web-traffic-module .el-alert {
  margin-bottom: 14px;
}
@media (max-width: 850px) {
  .web-traffic-cards {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (max-width: 520px) {
  .web-traffic-module {
    padding: 16px;
  }
  .web-traffic-cards {
    gap: 8px;
  }
  .web-traffic-cards > div {
    padding: 12px;
  }
  .web-traffic-cards strong {
    font-size: 21px;
  }
}
</style>
