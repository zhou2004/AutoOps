<template>
  <div class="storage-details">
    <el-descriptions :column="3" border>
      <el-descriptions-item v-if="detail.path" label="数据路径">{{
        detail.path
      }}</el-descriptions-item>
      <el-descriptions-item v-if="detail.inodes" label="inode 可用 / 总量"
        >{{ detail.freeInodes }} / {{ detail.inodes }}</el-descriptions-item
      >
      <el-descriptions-item
        v-if="detail.estimatedDaysRemaining != null && !detail.stale"
        label="预计剩余天数"
        >{{ detail.estimatedDaysRemaining.toFixed(1) }}</el-descriptions-item
      >
      <el-descriptions-item
        v-for="key in metricKeys"
        :key="key"
        :label="metrics[key]"
        >{{ detail[key] }}</el-descriptions-item
      >
    </el-descriptions>
    <template v-for="key in sections" :key="key">
      <h4 v-if="detail[key]?.length">{{ labels[key] }}</h4>
      <el-table
        v-if="detail[key]?.length"
        :data="detail[key]"
        max-height="260"
        size="small"
      >
        <el-table-column
          v-for="column in Object.keys(detail[key][0] || {})"
          :key="column"
          :prop="column"
          :label="columns[column] || column"
          min-width="140"
          show-overflow-tooltip
        />
      </el-table>
    </template>
  </div>
</template>
<script setup lang="ts">
import { computed } from "vue";
const props = defineProps<{ detail: Record<string, any> }>();
const sections = [
  "databases",
  "tablespaces",
  "binlogs",
  "replicationSlots",
  "archive"
];
const labels: Record<string, string> = {
  databases: "数据库容量（字节）",
  tablespaces: "表空间",
  binlogs: "二进制日志",
  replicationSlots: "复制槽",
  archive: "归档状态"
};
const columns: Record<string, string> = {
  name: "名称",
  data_bytes: "数据量",
  index_bytes: "索引量",
  bytes: "字节",
  path: "路径",
  Log_name: "日志文件",
  File_size: "大小（字节）",
  slot_name: "复制槽",
  active: "活跃",
  restart_lsn: "保留起点"
};
const metrics: Record<string, string> = {
  used_memory: "Redis 内存（字节）",
  used_memory_rss: "RSS（字节）",
  maxmemory: "内存上限（字节）",
  mem_fragmentation_ratio: "碎片率",
  rdb_last_bgsave_status: "RDB 状态",
  aof_enabled: "AOF 启用",
  aof_current_size: "AOF 大小（字节）",
  aof_last_write_status: "AOF 写入状态",
  warningPercent: "提醒阈值（%）",
  criticalPercent: "严重阈值（%）"
};
const metricKeys = computed(() =>
  Object.keys(metrics).filter(k => props.detail[k] != null)
);
</script>
<style scoped>
.storage-details {
  padding: 12px 24px;
  max-width: 100%;
}
</style>
