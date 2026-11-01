<template>
  <el-drawer
    :model-value="modelValue"
    :title="cluster?.name || '集群详情'"
    size="min(1200px, 94vw)"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
    /><el-skeleton v-if="loading && !cluster" :rows="6" animated />
    <template v-if="cluster"
      ><div class="cluster-toolbar">
        <DatabaseEngineIcon :engine="cluster.engine" /><el-tag>{{
          cluster.engine
        }}</el-tag
        ><el-tag effect="plain">{{ cluster.topology }}</el-tag
        ><el-tag :type="tagType(cluster.status)">{{
          label(cluster.status)
        }}</el-tag
        ><span
          >{{ label(cluster.management) }} · 版本 {{ cluster.revision }}</span
        ><el-button @click="load">刷新页面</el-button>
      </div>
      <el-alert
        v-if="cluster.compatibilityReason"
        type="warning"
        :closable="false"
        :title="cluster.compatibilityReason"
      />
      <el-alert
        v-if="cluster.status === 'stale'"
        type="warning"
        :closable="false"
        :title="`采集已过期，当前健康未知。最后采集：${date(cluster.collectedAt)}`"
      />
      <el-alert
        v-for="warning in cluster.placementWarnings || []"
        :key="warning"
        :title="warning"
        type="warning"
        :closable="false"
      />
      <el-tabs v-model="tab"
        ><el-tab-pane label="概览" name="overview"
          ><div class="health-grid">
            <article v-for="d in dimensions" :key="d.key">
              <span>{{ d.label }}</span
              ><el-tag :type="tagType(cluster.health?.[d.key])">{{
                label(cluster.health?.[d.key])
              }}</el-tag>
            </article>
          </div>
          <el-descriptions :column="2" border
            ><el-descriptions-item label="管理级别">{{
              label(cluster.management)
            }}</el-descriptions-item
            ><el-descriptions-item label="环境">{{
              cluster.environment || "未指定"
            }}</el-descriptions-item
            ><el-descriptions-item label="已接入">{{
              cluster.collectedAt ? "已采集" : "等待连接验证"
            }}</el-descriptions-item
            ><el-descriptions-item label="可运维">{{
              cluster.management === "observe" ? "观察模式" : "已托管"
            }}</el-descriptions-item
            ><el-descriptions-item label="高可用已验收">{{
              cluster.haValidatedAt ? date(cluster.haValidatedAt) : "未验收"
            }}</el-descriptions-item
            ><el-descriptions-item label="恢复已验证">{{
              cluster.restoreVerifiedAt
                ? date(cluster.restoreVerifiedAt)
                : "未验证"
            }}</el-descriptions-item></el-descriptions
          >
          <h3>访问入口</h3>
          <el-table :data="cluster.endpoints || []"
            ><el-table-column prop="mode" label="用途" /><el-table-column
              label="地址"
              min-width="300"
              ><template #default="{ row }">{{
                row.addresses?.join(", ")
              }}</template></el-table-column
            ><el-table-column
              prop="masterName"
              label="Sentinel masterName"
            /><el-table-column label="状态"
              ><template #default="{ row }">{{
                label(row.status)
              }}</template></el-table-column
            ></el-table
          ></el-tab-pane
        >
        <el-tab-pane label="拓扑" name="topology"
          ><DatabaseTopology :cluster="cluster" /><el-alert
            v-if="cluster.topology === 'redis-cluster'"
            :title="`槽位覆盖 ${slots?.covered || 0} / 16384，缺失 ${slots?.missing ?? 16384}，冲突 ${slots?.conflicts || 0}，迁移标记 ${slots?.migrating?.length || 0}`"
            :type="
              slots?.covered === 16384 && !slots?.conflicts
                ? 'success'
                : 'warning'
            "
            :closable="false"
        /></el-tab-pane>
        <el-tab-pane label="节点" name="nodes"
          ><el-table :data="cluster.members"
            ><el-table-column
              prop="name"
              label="实例"
              min-width="140"
            /><el-table-column
              prop="kind"
              label="组件"
              width="120"
            /><el-table-column label="地址" min-width="180"
              ><template #default="{ row }"
                >{{ row.address }}:{{ row.port }}</template
              ></el-table-column
            ><el-table-column label="角色" width="100"
              ><template #default="{ row }">{{
                label(row.role)
              }}</template></el-table-column
            ><el-table-column
              prop="version"
              label="版本"
              width="120"
            /><el-table-column
              prop="dataDir"
              label="数据目录"
              min-width="200"
              show-overflow-tooltip
            /><el-table-column
              prop="service"
              label="服务"
              min-width="200"
            /><el-table-column label="工作台" width="100"
              ><template #default="{ row }"
                ><el-button
                  v-if="row.sourceId"
                  link
                  type="primary"
                  @click="workbench(row)"
                  >打开</el-button
                ></template
              ></el-table-column
            ></el-table
          ></el-tab-pane
        >
        <el-tab-pane label="复制状态" name="ha"
          ><p class="explanation">
            数据保护优先。管理器选主成功与业务入口恢复分别核验；异步复制不承诺零数据丢失。
          </p>
          <el-table :data="dataNodes"
            ><el-table-column prop="name" label="节点" /><el-table-column
              label="角色"
              ><template #default="{ row }">{{
                label(row.role)
              }}</template></el-table-column
            ><el-table-column label="复制状态"
              ><template #default="{ row }"
                >{{
                  row.metrics?.replicationIO ||
                  row.metrics?.replicationState ||
                  row.metrics?.master_link_status ||
                  "—"
                }}
                {{ row.metrics?.replicationSQL || "" }}</template
              ></el-table-column
            ><el-table-column v-if="cluster.engine === 'mysql'" label="半同步"
              ><template #default="{ row }">{{
                semiSyncState(row)
              }}</template></el-table-column
            ><el-table-column label="延迟 / 偏移"
              ><template #default="{ row }">{{
                lag(row)
              }}</template></el-table-column
            ><el-table-column
              label="复制坐标"
              min-width="300"
              show-overflow-tooltip
              ><template #default="{ row }">{{
                row.metrics?.gtid_executed ||
                row.metrics?.lsn ||
                row.metrics?.replay_lsn ||
                row.metrics?.master_repl_offset ||
                "—"
              }}</template></el-table-column
            ></el-table
          ></el-tab-pane
        >
        <el-tab-pane label="参数配置" name="parameters" lazy>
          <ConfigurationDetail
            v-if="tab === 'parameters'"
            :cluster-id="clusterId"
            :model-value="modelValue && tab === 'parameters'"
            @task="id => $emit('task', id)"
          />
        </el-tab-pane>
      </el-tabs>
    </template>
  </el-drawer>
</template>
<script setup lang="ts">
import ConfigurationDetail from "./ConfigurationDetail.vue";
import DatabaseTopology from "./DatabaseTopology.vue";
import DatabaseEngineIcon from "./DatabaseEngineIcon.vue";
import { ref, computed, watch, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import {
  dbPlatform,
  label,
  tagType,
  type Cluster,
  type Member
} from "@/api/databasePlatform";
const props = defineProps<{
  modelValue: boolean;
  clusterId: number;
  initialTab?: string;
}>();
defineEmits(["update:modelValue", "task"]);
const router = useRouter();
const cluster = ref<Cluster>(),
  slots = ref<any>(),
  loading = ref(false),
  error = ref(""),
  tab = ref("overview");
const dimensions = [
  { key: "database", label: "数据库服务" },
  { key: "replication", label: "复制" },
  { key: "control", label: "控制节点" },
  { key: "endpoint", label: "访问入口" },
  { key: "backup", label: "备份" }
];
const dataNodes = computed(
    () => cluster.value?.members.filter(m => m.kind === "data") || []
  ),
  controls = computed(
    () => cluster.value?.members.filter(m => m.kind !== "data") || []
  ),
  primaries = computed(() => dataNodes.value.filter(m => m.role === "primary"));
function replicasOf(p: Member) {
  return dataNodes.value.filter(
    m =>
      m.role === "replica" &&
      (m.parentId === p.nativeId ||
        m.parentId === `${p.address}:${p.port}` ||
        (m.metrics?.upstreamAddress === p.address &&
          Number(m.metrics?.upstreamPort) === p.port))
  );
}
const disconnected = computed(() =>
  dataNodes.value.filter(
    m =>
      m.role !== "primary" &&
      !primaries.value.some(p => replicasOf(p).some(r => r.id === m.id))
  )
);
const semiSyncState = (m: any) => {
  const metrics = m.metrics || {};
  if (m.role !== "primary")
    return metrics.Rpl_semi_sync_slave_status === "ON"
      ? "已应答"
      : "未启用 / 未应答";
  const enabled =
    metrics.rpl_semi_sync_master_enabled ||
    metrics.rpl_semi_sync_source_enabled;
  const active =
    metrics.Rpl_semi_sync_master_status || metrics.Rpl_semi_sync_source_status;
  return enabled === "ON"
    ? active === "ON"
      ? "半同步生效"
      : "已退化为异步"
    : "异步复制";
};
const date = (v?: string) => (v ? new Date(v).toLocaleString() : "尚未采集");
const lag = (m: Member) =>
  m.metrics?.lagSeconds != null
    ? `${m.metrics.lagSeconds} 秒`
    : m.metrics?.slave_repl_offset != null
      ? `offset ${m.metrics.slave_repl_offset}`
      : m.metrics?.replay_age_seconds != null
        ? `最后回放 ${Math.round(m.metrics.replay_age_seconds)} 秒前`
        : "—";
function workbench(m: Member) {
  router.push({
    path: "/database/workbench",
    query: { id: String(m.sourceId) }
  });
}
let timer: ReturnType<typeof setInterval>;
onUnmounted(() => clearInterval(timer));
async function load() {
  loading.value = true;
  error.value = "";
  try {
    const d = await dbPlatform.cluster(props.clusterId);
    cluster.value = d.cluster;
    slots.value = d.slots;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}
watch(
  () => [props.modelValue, props.clusterId, props.initialTab],
  () => {
    clearInterval(timer);
    if (props.modelValue) {
      timer = setInterval(() => {
        if (!loading.value && !document.hidden) load();
      }, 15000);
      tab.value = props.initialTab || "overview";
      load();
    }
  },
  { immediate: true }
);
</script>
<style scoped>
.cluster-toolbar {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
  margin-bottom: 22px;
}
.cluster-toolbar > span {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.health-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 12px;
  margin-bottom: 24px;
}
.health-grid article {
  border: 1px solid var(--el-border-color-light);
  border-radius: 10px;
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  align-items: flex-start;
}
.health-grid article > span {
  font-size: 13px;
}
h3 {
  font-size: 16px;
  margin: 24px 0 14px;
}
.explanation {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.topology-controls {
  padding: 18px;
  border: 1px dashed var(--el-border-color);
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  border-radius: 12px;
}
.control-node {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 12px;
}
.control-node small {
  color: var(--el-text-color-secondary);
}
.topology-groups {
  display: flex;
  justify-content: space-around;
  gap: 24px;
  flex-wrap: wrap;
  margin: 28px 0;
}
.replication-group {
  min-width: 220px;
  flex: 1;
  max-width: 460px;
}
.node {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 18px;
  border: 1px solid var(--el-border-color);
  border-radius: 12px;
  background: var(--el-bg-color);
}
.node.primary {
  border-top: 3px solid var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.node small {
  font: 11px monospace;
  overflow-wrap: anywhere;
}
.node > span {
  font-size: 13px;
}
.replica-links {
  display: flex;
  gap: 14px;
  justify-content: center;
  padding-top: 32px;
}
.replica {
  position: relative;
  min-width: 160px;
  flex: 1;
}
.replication-arrow {
  position: absolute;
  top: -29px;
  color: var(--el-color-primary);
}
.el-alert {
  margin-bottom: 16px;
}
.el-table {
  margin-top: 18px;
}
@media (max-width: 900px) {
  .health-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .replica-links {
    flex-direction: column;
    gap: 34px;
  }
}
</style>
