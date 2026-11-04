<template>
  <el-drawer
    :model-value="modelValue"
    :title="$tr('备份仓库详情')"
    size="min(960px, 94vw)"
    append-to-body
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <el-alert v-if="error" :title="$tr(error)" type="error" :closable="false" />
    <div v-loading="loading">
      <template v-if="detail">
        <div class="detail-heading">
          <strong>{{ repository.name }}</strong
          ><el-button
            v-if="platform.isAdmin"
            :loading="checking"
            @click="check"
            >{{ $tr("连接检查") }}</el-button
          >
        </div>
        <el-descriptions :column="2" border>
          <el-descriptions-item :label="$tr('存储类型')">{{
            repository.kind === "s3" ? $tr("S3 兼容存储") : $tr("受管目录")
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('业务分组')">{{
            detail.groupName ||
            $tr("分组 #{value0}", { value0: repository.groupId })
          }}</el-descriptions-item>
          <el-descriptions-item
            :label="
              repository.kind === 's3' ? $tr('传输执行主机') : $tr('仓库主机')
            "
            :span="2"
            >{{ detail.host?.name || `CMDB #${repository.hostId}`
            }}<span
              v-if="
                detail.host?.address && detail.host.address !== detail.host.name
              "
            >
              · {{ detail.host.address }}</span
            ></el-descriptions-item
          >
          <template v-if="repository.kind === 's3'">
            <el-descriptions-item label="Endpoint" :span="2">{{
              repository.endpoint
            }}</el-descriptions-item>
            <el-descriptions-item label="Bucket">{{
              repository.bucket
            }}</el-descriptions-item>
            <el-descriptions-item label="Region">{{
              repository.region || "us-east-1"
            }}</el-descriptions-item>
            <el-descriptions-item :label="$tr('配置中心凭据')" :span="2">{{
              detail.credentialName ||
              $tr("凭据引用 #{value0}", { value0: repository.keyId })
            }}</el-descriptions-item>
          </template>
          <el-descriptions-item v-else :label="$tr('受管目录')" :span="2">{{
            repository.path
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('最近检查结果')"
            ><el-tag :type="tagType(repository.status)">{{
              label(repository.status)
            }}</el-tag></el-descriptions-item
          >
          <el-descriptions-item :label="$tr('最近检查时间')">{{
            date(repository.checkedAt)
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('检查说明')" :span="2">{{
            $tr(repository.checkMessage) ||
            $tr("尚无详细检查结果，可执行连接检查获取")
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('检查内容')" :span="2">{{
            repository.checkReport?.probe === "HeadBucket"
              ? $tr("S3 桶存在性与访问授权（HeadBucket）")
              : repository.checkReport?.probe
                ? $tr("临时文件写入、同步及删除")
                : $tr("尚未记录")
          }}</el-descriptions-item>
          <el-descriptions-item
            v-if="repository.checkReport?.totalBytes"
            :label="$tr('文件系统容量')"
            >{{
              bytes(repository.checkReport.totalBytes)
            }}</el-descriptions-item
          >
          <el-descriptions-item
            v-if="repository.checkReport?.availableBytes != null"
            :label="$tr('文件系统可用空间')"
            >{{
              bytes(repository.checkReport.availableBytes)
            }}</el-descriptions-item
          >
          <el-descriptions-item :label="$tr('登记时间')" :span="2">{{
            date(repository.createdAt)
          }}</el-descriptions-item>
        </el-descriptions>
        <h3>{{ $tr("备份使用情况") }}</h3>
        <el-descriptions :column="2" border>
          <el-descriptions-item :label="$tr('完整备份')"
            >{{ detail.statistics.backupCount }}
            {{ $tr("份") }}</el-descriptions-item
          >
          <el-descriptions-item :label="$tr('恢复已验证')"
            >{{ detail.statistics.verifiedBackupCount }}
            {{ $tr("份") }}</el-descriptions-item
          >
          <el-descriptions-item :label="$tr('受管备份占用')">{{
            bytes(detail.statistics.managedBytes)
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('最近备份时间')">{{
            date(detail.statistics.lastBackupAt)
          }}</el-descriptions-item>
        </el-descriptions>
        <p class="note">
          {{ $tr("占用根据平台已核验、尚未清理的备份清单统计，")
          }}{{ detail.statistics.scoped ? $tr("仅包含当前授权集群；") : ""
          }}{{ $tr("不代表整个桶或磁盘的实时总占用。") }}
        </p>
        <h3>{{ $tr("关联备份策略") }}</h3>
        <el-table
          :data="detail.policies || []"
          :empty-text="$tr('暂无关联策略')"
          ><el-table-column
            prop="name"
            :label="$tr('策略名称')"
            min-width="170"
            show-overflow-tooltip
          /><el-table-column
            prop="cron"
            :label="$tr('周期（Cron）')"
            min-width="130"
          /><el-table-column :label="$tr('状态')" width="100"
            ><template #default="{ row }"
              ><el-tag :type="row.enabled ? 'success' : 'info'">{{
                row.enabled ? $tr("已启用") : $tr("已暂停")
              }}</el-tag></template
            ></el-table-column
          ></el-table
        >
        <h3>{{ $tr("最近完整备份") }}</h3>
        <el-table
          :data="detail.recentBackups || []"
          :empty-text="$tr('暂无完整备份')"
          ><el-table-column
            :label="$tr('备份名称')"
            min-width="220"
            show-overflow-tooltip
            ><template #default="{ row }">{{
              backupDisplayName(row)
            }}</template></el-table-column
          ><el-table-column
            prop="clusterName"
            :label="$tr('集群')"
            min-width="170"
            show-overflow-tooltip
          /><el-table-column :label="$tr('恢复验证')" min-width="170"
            ><template #default="{ row }">{{
              row.verifiedAt ? date(row.verifiedAt) : $tr("等待恢复演练")
            }}</template></el-table-column
          ></el-table
        >
      </template>
    </div>
  </el-drawer>
</template>
<script setup lang="ts">
import { tr } from "@/locales";

import { computed, ref, watch } from "vue";
import { dbRequest, bytes, label, tagType } from "@/api/databasePlatform";
import { backupDisplayName } from "@/utils/databaseBackup";
import { platformAccess as platform } from "@/utils/platformAccess";
const props = defineProps<{ modelValue: boolean; repositoryId?: number }>();
const emit = defineEmits(["update:modelValue", "changed"]);
const detail = ref<any>(),
  loading = ref(false),
  checking = ref(false),
  error = ref("");
const repository = computed(() => detail.value?.repository || {});
const date = (v?: string) =>
  v ? new Date(v).toLocaleString() : tr("尚无记录");
let generation = 0;
async function load() {
  const version = ++generation;
  loading.value = true;
  error.value = "";
  try {
    const value = await dbRequest(`repositories/${props.repositoryId}`);
    if (version === generation && props.modelValue) detail.value = value;
  } catch (e) {
    if (version === generation) error.value = (e as Error).message;
  } finally {
    if (version === generation) loading.value = false;
  }
}
async function check() {
  checking.value = true;
  try {
    await dbRequest(`repositories/${props.repositoryId}/check`, "post");
    emit("changed");
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    checking.value = false;
    if (props.modelValue) await load();
  }
}
watch(
  () => [props.modelValue, props.repositoryId],
  () => {
    generation++;
    detail.value = undefined;
    if (props.modelValue && props.repositoryId) load();
  },
  { immediate: true }
);
</script>
<style scoped>
.detail-heading {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
}
h3 {
  margin: 24px 0 14px;
  font-size: 16px;
}
.note {
  color: var(--el-text-color-secondary);
  line-height: 1.8;
  font-size: 13px;
}
:deep(.el-descriptions__content) {
  overflow-wrap: anywhere;
}
</style>
