<template>
  <el-drawer
    :model-value="modelValue"
    :title="`备份集 #${backup?.id || ''}`"
    size="min(1000px, 94vw)"
    append-to-body
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <template v-if="backup">
      <div class="detail-actions">
        <DatabaseEngineIcon :engine="backup.engine" /><strong>{{
          backup.clusterName
        }}</strong>
        <el-button
          v-if="backup.operationId"
          link
          type="primary"
          @click="$emit('task', backup.operationId)"
          >备份任务日志</el-button
        >
        <ClusterActions
          :cluster-id="backup.clusterId"
          :backup-id="backup.id"
          :backup-status="backup.status"
          context="backups"
          button-label="恢复操作"
          @task="id => $emit('task', id)"
        />
      </div>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="备份方式">{{
          backupMethodLabel(backup.method)
        }}</el-descriptions-item>
        <el-descriptions-item label="完整性"
          ><el-tag :type="tagType(backup.status)">{{
            label(backup.status)
          }}</el-tag></el-descriptions-item
        >
        <el-descriptions-item label="创建时间">{{
          date(backup.createdAt)
        }}</el-descriptions-item>
        <el-descriptions-item label="依赖基线">{{
          backup.parentId ? `备份 #${backup.parentId}` : "无父备份"
        }}</el-descriptions-item>
        <el-descriptions-item label="副本位置"
          >{{
            backup.manifest?.remoteComplete
              ? "独立仓库已核验"
              : backup.manifest?.localComplete
                ? "数据库节点本地暂存"
                : "尚未完成"
          }}{{
            backup.repositoryId ? ` · 仓库 #${backup.repositoryId}` : ""
          }}</el-descriptions-item
        >
        <el-descriptions-item label="恢复验证">{{
          backup.verifiedAt ? date(backup.verifiedAt) : "尚未验证"
        }}</el-descriptions-item>
        <el-descriptions-item label="恢复范围" :span="2">{{
          window
            ? `${date(window.start)} 至 ${date(window.end)}（本地时区）`
            : "仅备份内容；没有可用于时间点恢复的完整归档窗口"
        }}</el-descriptions-item>
      </el-descriptions>
      <p class="note">{{ backupMethods[backup.method]?.description }}</p>
      <h3>备份文件与校验</h3>
      <el-table
        :data="artifacts"
        empty-text="没有可用的文件清单，请检查备份任务"
      >
        <el-table-column prop="member" label="实例 / 分片" min-width="130" />
        <el-table-column
          prop="file"
          label="本地文件"
          min-width="240"
          show-overflow-tooltip
        />
        <el-table-column label="大小" width="110"
          ><template #default="{ row }">{{
            bytes(row.size)
          }}</template></el-table-column
        >
        <el-table-column
          prop="sha256"
          label="SHA-256"
          min-width="180"
          show-overflow-tooltip
        />
      </el-table>
      <h3>恢复报告</h3>
      <el-alert
        v-if="reports.length"
        type="info"
        :closable="false"
        title="核验完成后隔离实例已停止，恢复目录与报告保留；未替换业务库。"
      />
      <el-empty
        v-else
        description="尚无恢复报告。通过“恢复操作 → 恢复验证”实际恢复并校验备份。"
      />
      <el-descriptions
        v-for="report in reports"
        :key="report.member"
        :column="2"
        border
        class="restore-report"
      >
        <el-descriptions-item label="实例 / 分片">{{
          report.member
        }}</el-descriptions-item>
        <el-descriptions-item label="数据核验">{{
          report.verified ? "通过" : "未通过"
        }}</el-descriptions-item>
        <el-descriptions-item label="主机 / 端口"
          >CMDB #{{ report.hostId }} : {{ report.port }}</el-descriptions-item
        >
        <el-descriptions-item label="恢复时间点">{{
          report.targetTime ? date(report.targetTime) : "备份内容"
        }}</el-descriptions-item>
        <el-descriptions-item label="保留目录" :span="2">{{
          report.path
        }}</el-descriptions-item>
        <el-descriptions-item label="校验内容" :span="2">{{
          report.engine === "redis"
            ? `恢复键 ${report.keys ?? "—"}，持久键 ${report.persistentKeys ?? "—"}`
            : (report.databases || [])
                .map(d => `${d.name}：${d.tables} 张表`)
                .join("；") || "查看任务日志中的数据校验结果"
        }}</el-descriptions-item>
      </el-descriptions>
    </template>
  </el-drawer>
</template>
<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { bytes, label, tagType, dbRequest } from "@/api/databasePlatform";
import {
  backupMethodLabel,
  backupMethods,
  recoveryWindow
} from "@/utils/databaseBackup";
import ClusterActions from "./ClusterActions.vue";
import DatabaseEngineIcon from "./DatabaseEngineIcon.vue";
const props = defineProps<{ modelValue: boolean; backup?: any }>();
defineEmits(["update:modelValue", "task"]);
const fresh = ref<any>(),
  error = ref(""),
  loading = ref(false);
const backup = computed(() => fresh.value || props.backup);
let generation = 0;
async function load() {
  const current = ++generation;
  if (!props.modelValue || !props.backup) return;
  loading.value = true;
  error.value = "";
  try {
    const data = await dbRequest(`backups/${props.backup.id}`);
    if (current === generation) fresh.value = data;
  } catch (e) {
    if (current === generation) error.value = (e as Error).message;
  } finally {
    if (current === generation) loading.value = false;
  }
}
watch(
  () => [props.modelValue, props.backup],
  () => {
    fresh.value = undefined;
    load();
  },
  { immediate: true }
);
const window = computed(() => recoveryWindow(backup.value));
const entries = (
  records: any
): Array<{ member: string } & Record<string, any>> =>
  Object.entries(records || {}).map(([member, detail]) => ({
    member,
    ...(detail as Record<string, any>)
  }));
const artifacts = computed(() => entries(backup.value?.manifest?.artifacts));
const reports = computed(() => entries(backup.value?.manifest?.restoreReport));
const date = (value?: string | number) =>
  value ? new Date(value).toLocaleString() : "—";
</script>
<style scoped>
.detail-actions {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
  margin-bottom: 20px;
}
.note {
  color: var(--el-text-color-secondary);
  line-height: 1.8;
  margin: 16px 0;
}
h3 {
  margin: 24px 0 14px;
}
.restore-report {
  margin-top: 16px;
  overflow-wrap: anywhere;
}
</style>
