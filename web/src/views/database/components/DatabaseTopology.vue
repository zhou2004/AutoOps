<template>
  <div class="database-topology">
    <div v-if="cluster.vip" class="vip-card" :class="cluster.vip.status">
      <el-icon :size="28"><Connection /></el-icon>
      <div>
        <strong>VIP · {{ cluster.vip.cidr }}</strong
        ><small
          >{{ cluster.vip.interface }} · VRID {{ cluster.vip.routerId }}</small
        >
      </div>
      <div>
        <el-tag :type="tagType(cluster.vip.status)">{{
          label(cluster.vip.status)
        }}</el-tag
        ><small>当前绑定：{{ vipOwner }}</small>
      </div>
      <div>
        <small>{{
          cluster.vip.verifiedAt
            ? `最近漂移验证：${date(cluster.vip.verifiedAt)}`
            : "尚未进行漂移验证"
        }}</small
        ><small v-if="cluster.vip.lastDrill?.elapsedMs"
          >最近连接核验耗时
          {{ (cluster.vip.lastDrill.elapsedMs / 1000).toFixed(1) }} 秒</small
        >
      </div>
    </div>
    <div v-else-if="cluster.engine !== 'redis'" class="vip-empty">
      <el-icon><Connection /></el-icon> 尚未绑定 VIP，可从集群行「操作 →
      配置虚拟 IP」部署。
    </div>
    <div class="control-plane">
      <article
        v-for="m in controls"
        :key="m.id"
        :class="['control-node', m.state]"
      >
        <DatabaseEngineIcon v-if="m.kind === 'sentinel'" engine="redis" />
        <el-icon v-else :size="26"
          ><component :is="m.kind === 'proxy' ? Switch : SetUp"
        /></el-icon>
        <div>
          <strong>{{ componentName(m.kind) }} · {{ label(m.role) }}</strong
          ><small>{{ m.address }}:{{ m.port }}</small
          ><el-tag size="small" :type="tagType(m.state)">{{
            label(m.state)
          }}</el-tag
          ><small
            v-if="
              cluster.vip?.ownerMemberId === m.id &&
              cluster.vip.status !== 'unknown'
            "
            >绑定 VIP {{ cluster.vip.cidr.split("/")[0] }}</small
          >
        </div>
      </article>
    </div>
    <div class="topology-legend">
      <span class="healthy">● 已追平采样位点</span
      ><span class="degraded">● 追赶中</span
      ><span class="unavailable">● 复制异常</span
      ><span class="unknown">● 未核验 / 采集过期</span>
    </div>
    <div class="replication-groups">
      <section v-for="p in primaries" :key="p.id" class="replication-group">
        <article class="data-node primary">
          <DatabaseEngineIcon :engine="cluster.engine" /><strong>{{
            p.name
          }}</strong
          ><span>{{ p.address }}:{{ p.port }}</span
          ><el-tag :type="tagType(p.state)"
            >主节点 · {{ label(p.state) }}</el-tag
          ><small v-if="p.shard">{{ p.shard }}</small>
        </article>
        <div class="replica-branches">
          <div
            v-for="link in linksOf(p.id)"
            :key="link.targetId"
            :class="['branch', link.state]"
          >
            <svg
              class="flow-link"
              viewBox="0 0 100 64"
              role="img"
              :aria-label="link.message"
            >
              <path class="track" d="M50 0 V64" />
              <path class="flow" d="M50 0 V64" />
              <path class="arrow" d="M44 55 L50 62 L56 55" />
            </svg>
            <article class="data-node replica">
              <DatabaseEngineIcon :engine="cluster.engine" /><strong>{{
                node(link.targetId)?.name
              }}</strong
              ><span
                >{{ node(link.targetId)?.address }}:{{
                  node(link.targetId)?.port
                }}</span
              >
              <el-tag :type="tagType(link.state)">{{ link.message }}</el-tag>
              <small v-if="cluster.engine === 'mysql'"
                >IO {{ link.evidence.replicationIO || "未知" }} / SQL
                {{ link.evidence.replicationSQL || "未知" }} · 延迟
                {{ link.evidence.lagSeconds ?? "未知" }} 秒</small
              >
              <small v-else-if="link.evidence.sampleLagBytes != null"
                >采样位点差 {{ link.evidence.sampleLagBytes }} 字节</small
              >
              <el-tooltip
                v-if="link.evidence.sampleWatermark"
                :content="String(link.evidence.sampleWatermark)"
                placement="top"
                ><small class="watermark"
                  >位点 {{ link.evidence.sampleWatermark }}</small
                ></el-tooltip
              >
              <small>采集 {{ date(link.collectedAt) }}</small>
            </article>
          </div>
        </div>
      </section>
      <article
        v-for="m in disconnected"
        :key="m.id"
        class="data-node disconnected"
      >
        <DatabaseEngineIcon :engine="cluster.engine" /><strong>{{
          m.name
        }}</strong
        ><span>{{ m.address }}:{{ m.port }}</span
        ><el-tag type="warning">未确认主从关系</el-tag>
      </article>
    </div>
    <el-empty v-if="!dataNodes.length" description="尚未发现数据节点" />
    <p class="topology-note">
      流动表示近期采集确认复制运行且副本已追平主库采样位点。位点核验不等于全表校验，也不保证后续故障零数据丢失；采集过期时停止动画。
    </p>
  </div>
</template>
<script setup lang="ts">
import { computed } from "vue";
import { Connection, Switch, SetUp } from "@element-plus/icons-vue";
import DatabaseEngineIcon from "./DatabaseEngineIcon.vue";
import { label, tagType, type Cluster } from "@/api/databasePlatform";
const props = defineProps<{ cluster: Cluster }>();
const dataNodes = computed(() =>
  props.cluster.members.filter(m => m.kind === "data")
);
const controls = computed(() =>
  props.cluster.members.filter(m => m.kind !== "data")
);
const primaries = computed(() =>
  dataNodes.value.filter(m => m.role === "primary")
);
const links = computed(() =>
  (props.cluster.replication || []).map(l => ({
    ...l,
    state:
      props.cluster.status === "stale" ||
      !l.collectedAt ||
      Date.now() - new Date(l.collectedAt).getTime() > 90000
        ? "unknown"
        : l.state
  }))
);
const linksOf = (id: number) => links.value.filter(l => l.sourceId === id);
const node = (id: number) => props.cluster.members.find(m => m.id === id);
const disconnected = computed(() =>
  dataNodes.value.filter(
    m =>
      m.role !== "primary" &&
      !links.value.some(l => l.targetId === m.id && l.sourceId)
  )
);
const vipOwner = computed(() =>
  props.cluster.vip?.status === "unknown"
    ? "状态未知"
    : node(props.cluster.vip?.ownerMemberId || 0)?.address || "未绑定 / 冲突"
);
const date = (v?: string) =>
  v ? new Date(v).toLocaleTimeString() : "尚未采集";
const componentName = (kind: string) =>
  ({
    proxy: "HAProxy",
    orchestrator: "Orchestrator",
    sentinel: "Sentinel",
    patroni: "Patroni",
    etcd: "etcd"
  })[kind] || kind;
</script>
<style scoped>
.database-topology {
  --link-color: var(--el-text-color-placeholder);
}
.vip-card {
  display: flex;
  gap: 18px;
  align-items: center;
  flex-wrap: wrap;
  padding: 18px;
  margin-bottom: 20px;
  border: 1px solid var(--el-color-primary-light-5);
  background: var(--el-color-primary-light-9);
  border-radius: 12px;
}
.vip-card > div {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.vip-card small,
.vip-empty,
small,
.topology-note {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.vip-empty {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}
.control-plane {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  padding: 16px;
  border: 1px dashed var(--el-border-color);
  border-radius: 12px;
}
.control-node {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1 0 170px;
  font-size: 12px;
}
.control-node > div {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
}
.topology-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  font-size: 12px;
  margin: 24px 0;
}
.healthy {
  --link-color: var(--el-color-success);
}
.degraded {
  --link-color: var(--el-color-warning);
}
.unavailable,
.conflict {
  --link-color: var(--el-color-danger);
}
.unknown {
  --link-color: var(--el-text-color-placeholder);
}
.topology-legend span {
  color: var(--link-color);
}
.replication-groups {
  display: flex;
  gap: 20px;
  flex-wrap: wrap;
  align-items: flex-start;
}
.replication-group {
  flex: 1 1 320px;
  min-width: 0;
}
.data-node {
  padding: 16px 10px;
  border: 1px solid var(--el-border-color);
  border-radius: 12px;
  display: flex;
  align-items: center;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  background: var(--el-bg-color);
  font-size: 12px;
}
.data-node strong {
  font-size: 14px;
}
.primary {
  border-top: 3px solid var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.replica-branches {
  display: flex;
  gap: 10px;
}
.branch {
  flex: 1;
  min-width: 0;
}
.replica {
  border-top: 3px solid var(--link-color);
}
.replica :deep(.el-tag) {
  height: auto;
  white-space: normal;
  text-align: center;
  line-height: 18px;
  padding: 4px;
}
.flow-link {
  width: 100%;
  height: 64px;
  display: block;
  fill: none;
  stroke: var(--link-color);
  stroke-width: 2;
}
.track {
  opacity: 0.2;
}
.flow {
  stroke-dasharray: 7 6;
}
.healthy .flow {
  animation: replication-flow 1s linear infinite;
}
.watermark {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: monospace;
}
.topology-note {
  line-height: 1.7;
  margin-top: 24px;
}
@keyframes replication-flow {
  to {
    stroke-dashoffset: -26;
  }
}
@media (prefers-reduced-motion: reduce) {
  .healthy .flow {
    animation: none;
  }
}
@media (max-width: 600px) {
  .replica-branches {
    flex-direction: column;
  }
  .replication-group {
    flex-basis: 100%;
  }
}
</style>
