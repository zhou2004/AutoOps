<template>
  <span class="cluster-row-actions" @click.stop>
    <el-dropdown
      trigger="click"
      :max-height="440"
      @visible-change="load"
      @command="select"
    >
      <el-button
        link
        type="primary"
        :loading="loading"
        :aria-label="
          $tr('操作 {value0}', { value0: clusterName || $tr('集群') })
        "
      >
        {{ $tr(buttonLabel || "操作")
        }}<el-icon class="dropdown-arrow"><ArrowDown /></el-icon>
      </el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item v-if="loading" disabled>{{
            $tr("正在读取可用操作")
          }}</el-dropdown-item>
          <el-dropdown-item v-else-if="error" disabled>{{
            $tr(error)
          }}</el-dropdown-item>
          <template v-else>
            <template v-if="context === 'clusters' && cluster">
              <el-dropdown-item command="detail:overview">{{
                $tr("集群详情")
              }}</el-dropdown-item>
              <el-dropdown-item command="detail:parameters">{{
                $tr("参数配置详情")
              }}</el-dropdown-item>
            </template>
            <el-dropdown-item
              v-for="(item, index) in available"
              :key="item.action"
              :command="item.action"
              :divided="index > 0 && item.group !== available[index - 1].group"
            >
              {{ item.label }}
            </el-dropdown-item>
            <el-dropdown-item
              v-for="(item, index) in links"
              :key="item.section"
              :command="`page:${item.section}`"
              :divided="index === 0 && available.length > 0"
            >
              {{ item.label }}
            </el-dropdown-item>
            <el-dropdown-item
              v-if="!available.length && !links.length"
              disabled
              >{{ $tr("暂无可用操作") }}</el-dropdown-item
            >
          </template>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
    <OperationDialog
      v-if="cluster"
      v-model="operationOpen"
      :cluster="cluster"
      :action="selectedAction"
      :label="selectedLabel"
      :initial-parameters="initialParameters"
      @submitted="submitted"
    />
  </span>
</template>
<script setup lang="ts">
import { tr } from "@/locales";

import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { ArrowDown } from "@element-plus/icons-vue";
import {
  capabilityLabel,
  dbPlatform,
  type Capability,
  type Cluster
} from "@/api/databasePlatform";
import OperationDialog from "./OperationDialog.vue";
const props = withDefaults(
  defineProps<{
    clusterId: number;
    clusterName?: string;
    context?: string;
    backupId?: number;
    memberId?: number;
    backupStatus?: string;
    buttonLabel?: string;
  }>(),
  { context: "clusters" }
);
const emit = defineEmits(["task", "changed", "detail"]);
const router = useRouter();
const cluster = ref<Cluster>();
const capabilities = ref<Capability[]>([]);
const loading = ref(false),
  error = ref(""),
  operationOpen = ref(false),
  selectedAction = ref(""),
  selectedLabel = ref("");
const groups: Record<string, string[][]> = {
  clusters: [
    ["refresh", "provision", "adopt", "reconcile"],
    [
      "switchover",
      "recover",
      "resume-writes",
      "rejoin",
      "repair-replication",
      "enter-maintenance",
      "exit-maintenance",
      "acknowledge-recovery"
    ],
    [
      "add-node",
      "rebuild-replica",
      "remove-node",
      "restart",
      "reshard",
      "assign-replica",
      "balance-primaries"
    ],
    [
      "configure-endpoint",
      "configure-vip",
      "verify-vip",
      "configure-fencing",
      "configure-semisync",
      "update-patroni"
    ]
  ],
  storage: [
    ["collect-storage"],
    ["redis-bgsave", "redis-aof-rewrite", "purge-binlogs"]
  ],
  backups: [
    ["backup", "configure-archive"],
    ["restore", "verify-backup", "expire-backup"]
  ],
  configurations: [["configure", "restart"]],
  audit: [["reconcile"]]
};
const available = computed(() =>
  (groups[props.context] || []).flatMap((actions, group) =>
    actions.flatMap(action => {
      if (
        props.context === "backups" &&
        props.backupId &&
        (!["restore", "verify-backup", "expire-backup"].includes(action) ||
          (props.backupStatus && props.backupStatus !== "complete"))
      )
        return [];
      const capability = capabilities.value.find(
        c => c.action === action && c.enabled
      );
      return capability
        ? [{ ...capability, label: capabilityLabel(capability), group }]
        : [];
    })
  )
);
const links = computed(() =>
  props.context === "clusters" && cluster.value
    ? [
        {
          section: "storage",
          get label() {
            return tr("存储管理");
          }
        },
        {
          section: "backups",
          get label() {
            return tr("备份恢复");
          }
        },
        {
          section: "audit",
          get label() {
            return tr("操作审计");
          }
        }
      ]
    : []
);
const initialParameters = computed(() => ({
  ...(props.backupId &&
  ["restore", "verify-backup", "expire-backup"].includes(selectedAction.value)
    ? { backupId: props.backupId }
    : {}),
  ...(props.memberId &&
  ["redis-bgsave", "redis-aof-rewrite"].includes(selectedAction.value)
    ? { memberId: props.memberId }
    : {})
}));
async function load(visible: boolean) {
  if (!visible) return;
  loading.value = true;
  error.value = "";
  capabilities.value = [];
  try {
    const data = await dbPlatform.cluster(props.clusterId);
    cluster.value = data.cluster;
    capabilities.value = data.capabilities;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}
function select(command: string) {
  if (command.startsWith("detail:")) {
    emit("detail", command.slice(7));
    return;
  }
  if (command.startsWith("page:")) {
    router.push({
      path:
        command === "page:audit"
          ? "/monitor/dblog"
          : `/database/${command.slice(5)}`,
      query: {
        ...(command === "page:audit" ? { tab: "database-audit" } : {}),
        clusterId: String(props.clusterId),
        engine: cluster.value?.engine
      }
    });
    return;
  }
  const capability = available.value.find(c => c.action === command);
  if (!capability) return;
  selectedAction.value = command;
  selectedLabel.value =
    capabilities.value.find(c => c.action === command)?.label ||
    capability.label;
  operationOpen.value = true;
}
function submitted(id: number) {
  operationOpen.value = false;
  emit("task", id);
  emit("changed");
}
</script>
<style scoped>
.cluster-row-actions {
  display: inline-flex;
  align-items: center;
}
.dropdown-arrow {
  margin-left: 4px;
}
</style>
