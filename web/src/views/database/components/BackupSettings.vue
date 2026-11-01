<template>
  <el-dialog
    :model-value="modelValue"
    :title="kind === 'repository' ? '登记备份仓库' : '创建备份策略'"
    width="min(720px, 94vw)"
    append-to-body
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <p v-if="kind === 'policy'">
      策略保存后按计划运行，不会立即备份。首次使用可先通过“立即备份”验证方式和仓库。
    </p>
    <el-form label-width="130px">
      <el-form-item label="名称"
        ><el-input v-model="form.name" maxlength="64"
      /></el-form-item>
      <template v-if="kind === 'repository'">
        <el-form-item label="业务分组"
          ><el-tree-select
            v-model="form.groupId"
            :data="groups"
            :props="{ label: 'name', value: 'id', children: 'children' }"
            check-strictly
            filterable
        /></el-form-item>
        <el-form-item label="存储类型"
          ><el-select v-model="form.kind"
            ><el-option value="local" label="受管目录" /><el-option
              value="s3"
              label="S3 兼容存储" /></el-select
        ></el-form-item>
        <el-form-item
          :label="form.kind === 'local' ? '仓库主机' : '传输执行主机'"
          ><el-select v-model="form.hostId" filterable
            ><el-option
              v-for="h in hosts"
              :key="h.id"
              :value="h.id"
              :label="
                h.hostname || h.hostName || h.name || String(h.id)
              " /></el-select
        ></el-form-item>
        <el-form-item v-if="form.kind === 'local'" label="受管目录"
          ><el-input
            v-model="form.path"
            placeholder="/var/lib/autoops-backup-repository/业务名称"
        /></el-form-item>
        <template v-else>
          <el-form-item label="S3 Endpoint"
            ><el-input
              v-model="form.endpoint"
              placeholder="https://s3.example.com"
          /></el-form-item>
          <el-form-item label="Bucket"
            ><el-input v-model="form.bucket"
          /></el-form-item>
          <el-form-item label="Region"
            ><el-input v-model="form.region"
          /></el-form-item>
          <el-form-item label="配置中心凭据"
            ><el-select v-model="form.keyId" filterable
              ><el-option
                v-for="k in keys"
                :key="k.id"
                :value="k.id"
                :label="
                  k.name || k.alias || k.keyName || String(k.id)
                " /></el-select
          ></el-form-item>
        </template>
        <p>
          登记后执行连接检查。生产备份应保存到数据库节点以外；暂存成功与独立副本成功分别记录。
        </p>
      </template>
      <template v-else>
        <el-form-item label="集群"
          ><el-select
            v-model="form.clusterId"
            filterable
            @change="selectCluster"
            ><el-option
              v-for="c in clusters"
              :key="c.id"
              :value="c.id"
              :label="`${c.name} · ${c.engine}`" /></el-select
        ></el-form-item>
        <el-form-item label="备份方式"
          ><el-select v-model="form.method"
            ><el-option
              v-for="m in methods"
              :key="m.value"
              :value="m.value"
              :label="m.label" /></el-select
        ></el-form-item>
        <p>{{ backupMethods[form.method]?.description }}</p>
        <el-form-item label="备份仓库"
          ><el-select
            v-model="form.repositoryId"
            clearable
            placeholder="本地暂存"
            ><el-option
              v-for="r in repositories.filter(
                r => r.status === 'healthy' && r.groupId === selected?.groupId
              )"
              :key="r.id"
              :value="r.id"
              :label="r.name" /></el-select
        ></el-form-item>
        <el-form-item label="执行周期"
          ><el-select v-model="schedule" @change="selectSchedule"
            ><el-option value="daily" label="每天 02:00" /><el-option
              value="weekly"
              label="每周日 02:00" /><el-option
              value="custom"
              label="自定义 Cron" /></el-select
        ></el-form-item>
        <el-form-item label="调度（Cron）"
          ><el-input
            v-model="form.cron"
            :disabled="schedule !== 'custom'"
            placeholder="分 时 日 月 周，例如 0 2 * * *"
        /></el-form-item>
        <el-form-item label="保留天数"
          ><el-input-number v-model="form.retentionDays" :min="1" :max="3650"
        /></el-form-item>
        <el-form-item label="启用"
          ><el-switch v-model="form.enabled"
        /></el-form-item>
        <p>
          Cron 使用 Worker
          的本地时区。备份与其他集群变更互斥。保留策略始终保护恢复依赖链，备份完成后仍需执行隔离恢复验证。
        </p>
      </template>
    </el-form>
    <template #footer
      ><el-button @click="$emit('update:modelValue', false)">取消</el-button
      ><el-button type="primary" :loading="busy" @click="save"
        >保存</el-button
      ></template
    >
  </el-dialog>
</template>
<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { dbRequest, type Cluster } from "@/api/databasePlatform";
import cmdb from "@/api/cmdb";
import { backupMethods } from "@/utils/databaseBackup";
import config from "@/api/config";
const props = defineProps<{
  modelValue: boolean;
  kind: "repository" | "policy";
  groups: any[];
  engine?: string;
  clusterId?: number;
}>();
const emit = defineEmits(["update:modelValue", "saved"]);
const schedule = ref("daily");
function selectSchedule() {
  if (schedule.value !== "custom")
    form.cron = schedule.value === "daily" ? "0 2 * * *" : "0 2 * * 0";
}
const form = reactive<Record<string, any>>({});
const error = ref(""),
  busy = ref(false),
  hosts = ref<any[]>([]),
  keys = ref<any[]>([]),
  repositories = ref<any[]>([]),
  clusters = ref<Cluster[]>([]);
const selected = computed(() =>
  clusters.value.find(c => c.id === form.clusterId)
);
const methods = computed(() =>
  selected.value?.engine === "redis"
    ? [{ value: "rdb", label: "RDB / 按分片备份" }]
    : [
        { value: "logical", label: "逻辑备份" },
        { value: "physical", label: "物理全量（需配置匹配工具）" },
        { value: "incremental", label: "增量备份（要求已有基线）" },
        ...(selected.value?.engine === "postgresql"
          ? [
              { value: "differential", label: "差异备份" },
              { value: "wal-archive", label: "导出基线与 WAL 归档" }
            ]
          : [{ value: "binlog-archive", label: "导出基线与已关闭 binlog" }])
      ]
);
function selectCluster() {
  form.method = selected.value?.engine === "redis" ? "rdb" : "logical";
  form.repositoryId = 0;
}
async function save() {
  busy.value = true;
  error.value = "";
  try {
    await dbRequest(
      props.kind === "repository" ? "repositories" : "backup-policies",
      "post",
      form
    );
    emit("saved");
    emit("update:modelValue", false);
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    busy.value = false;
  }
}
watch(
  () => props.modelValue,
  async value => {
    if (!value) return;
    Object.keys(form).forEach(k => delete form[k]);
    Object.assign(form, {
      name: "",
      kind: "local",
      path: "/var/lib/autoops-backup-repository/",
      region: "us-east-1",
      method: "logical",
      cron: "0 2 * * *",
      retentionDays: 14,
      enabled: true,
      repositoryId: 0
    });
    error.value = "";
    schedule.value = "daily";
    try {
      if (props.kind === "repository") {
        const [h, k] = await Promise.all([
          cmdb.getCmdbHostList({ page: 1, pageSize: 100 }),
          config.getKeyManageList({ page: 1, size: 100 })
        ]);
        hosts.value = h.data?.data?.list || h.data?.data?.records || [];
        keys.value = k.data?.data?.list || k.data?.data?.records || [];
      } else {
        const [c, r] = await Promise.all([
          dbRequest("clusters", "get", undefined, {
            page: 1,
            size: 100,
            engine: props.engine
          }),
          dbRequest("repositories", "get", undefined, { size: 100 })
        ]);
        clusters.value = c.list || [];
        repositories.value = r.list || [];
        if (
          props.clusterId &&
          clusters.value.some(c => c.id === props.clusterId)
        ) {
          form.clusterId = props.clusterId;
          selectCluster();
        }
      }
    } catch (e) {
      error.value = (e as Error).message;
    }
  }
);
</script>
<style scoped>
.el-alert {
  margin-bottom: 16px;
}
.el-select,
.el-tree-select {
  width: 100%;
}
p {
  color: var(--el-text-color-secondary);
  line-height: 1.7;
}
</style>
