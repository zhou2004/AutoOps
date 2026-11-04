<template>
  <div class="storage-details">
    <el-descriptions :column="3" border>
      <el-descriptions-item v-if="detail.path" :label="$tr('数据路径')">{{
        detail.path
      }}</el-descriptions-item>
      <el-descriptions-item
        v-if="detail.inodes"
        :label="$tr('inode 可用 / 总量')"
        >{{ detail.freeInodes }} / {{ detail.inodes }}</el-descriptions-item
      >
      <el-descriptions-item
        v-if="detail.estimatedDaysRemaining != null && !detail.stale"
        :label="$tr('预计剩余天数')"
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
import { tr } from "@/locales";

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
  get databases() {
    return tr("数据库容量（字节）");
  },
  get tablespaces() {
    return tr("表空间");
  },
  get binlogs() {
    return tr("二进制日志");
  },
  get replicationSlots() {
    return tr("复制槽");
  },
  get archive() {
    return tr("归档状态");
  }
};
const columns: Record<string, string> = {
  get name() {
    return tr("名称");
  },
  get data_bytes() {
    return tr("数据量");
  },
  get index_bytes() {
    return tr("索引量");
  },
  get bytes() {
    return tr("字节");
  },
  get path() {
    return tr("路径");
  },
  get Log_name() {
    return tr("日志文件");
  },
  get File_size() {
    return tr("大小（字节）");
  },
  get slot_name() {
    return tr("复制槽");
  },
  get active() {
    return tr("活跃");
  },
  get restart_lsn() {
    return tr("保留起点");
  }
};
const metrics: Record<string, string> = {
  get used_memory() {
    return tr("Redis 内存（字节）");
  },
  get used_memory_rss() {
    return tr("RSS（字节）");
  },
  get maxmemory() {
    return tr("内存上限（字节）");
  },
  get mem_fragmentation_ratio() {
    return tr("碎片率");
  },
  get rdb_last_bgsave_status() {
    return tr("RDB 状态");
  },
  get aof_enabled() {
    return tr("AOF 启用");
  },
  get aof_current_size() {
    return tr("AOF 大小（字节）");
  },
  get aof_last_write_status() {
    return tr("AOF 写入状态");
  },
  get warningPercent() {
    return tr("提醒阈值（%）");
  },
  get criticalPercent() {
    return tr("严重阈值（%）");
  }
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
