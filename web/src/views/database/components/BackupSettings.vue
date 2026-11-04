<template>
  <el-dialog
    :model-value="modelValue"
    :title="
      kind === 'repository'
        ? $tr('登记备份仓库')
        : policy
          ? $tr('编辑备份策略')
          : $tr('创建备份策略')
    "
    width="min(720px, 94vw)"
    append-to-body
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <el-alert v-if="error" :title="$tr(error)" type="error" :closable="false" />
    <p v-if="kind === 'policy'">
      {{
        $tr(
          "策略保存后按计划运行，不会立即备份。首次使用可先通过“立即备份”验证方式和仓库。"
        )
      }}
    </p>
    <el-form label-width="130px">
      <el-form-item :label="$tr('名称')"
        ><el-input v-model="form.name" maxlength="64"
      /></el-form-item>
      <template v-if="kind === 'repository'">
        <el-form-item :label="$tr('业务分组')"
          ><el-tree-select
            v-model="form.groupId"
            :data="groups"
            :props="{ label: 'name', value: 'id', children: 'children' }"
            check-strictly
            filterable
        /></el-form-item>
        <el-form-item :label="$tr('存储类型')"
          ><el-select v-model="form.kind"
            ><el-option value="local" :label="$tr('受管目录')" /><el-option
              value="s3"
              :label="$tr('S3 兼容存储')" /></el-select
        ></el-form-item>
        <el-form-item
          :label="form.kind === 'local' ? $tr('仓库主机') : $tr('传输执行主机')"
          ><el-select v-model="form.hostId" filterable
            ><el-option
              v-for="h in hosts"
              :key="h.id"
              :value="h.id"
              :label="
                h.hostname || h.hostName || h.name || String(h.id)
              " /></el-select
        ></el-form-item>
        <el-form-item v-if="form.kind === 'local'" :label="$tr('受管目录')"
          ><el-input
            v-model="form.path"
            :placeholder="$tr('/var/lib/autoops-backup-repository/业务名称')"
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
          <el-form-item :label="$tr('配置中心凭据')"
            ><el-select v-model="form.keyId" filterable
              ><el-option
                v-for="k in keys"
                :key="k.id"
                :value="k.id"
                :label="
                  k.remark || k.name || k.alias || k.keyName || String(k.id)
                " /></el-select
          ></el-form-item>
        </template>
        <p>
          {{
            $tr(
              "登记后执行连接检查。生产备份应保存到数据库节点以外；暂存成功与独立副本成功分别记录。"
            )
          }}
        </p>
      </template>
      <template v-else>
        <el-form-item :label="$tr('集群')"
          ><el-select
            v-model="form.clusterId"
            :disabled="!!policy"
            filterable
            @change="selectCluster"
            ><el-option
              v-for="c in clusters"
              :key="c.id"
              :value="c.id"
              :label="`${c.name} · ${c.engine}`" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('备份方式')"
          ><el-select v-model="form.method"
            ><el-option
              v-for="m in methods"
              :key="m.value"
              :value="m.value"
              :label="m.label" /></el-select
        ></el-form-item>
        <p>{{ backupMethods[form.method]?.description }}</p>
        <el-form-item :label="$tr('备份仓库')"
          ><el-select
            v-model="form.repositoryId"
            clearable
            :placeholder="$tr('本地暂存')"
            ><el-option
              v-for="r in repositories.filter(
                r => r.status === 'healthy' && r.groupId === selected?.groupId
              )"
              :key="r.id"
              :value="r.id"
              :label="r.name" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('执行周期')"
          ><el-select v-model="schedule" @change="selectSchedule"
            ><el-option value="daily" :label="$tr('每天 02:00')" /><el-option
              value="weekly"
              :label="$tr('每周日 02:00')" /><el-option
              value="custom"
              :label="$tr('自定义 Cron')" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('调度（Cron）')"
          ><el-input
            v-model="form.cron"
            :disabled="schedule !== 'custom'"
            :placeholder="$tr('分 时 日 月 周，例如 0 2 * * *')"
        /></el-form-item>
        <el-form-item :label="$tr('保留天数')"
          ><el-input-number v-model="form.retentionDays" :min="1" :max="3650"
        /></el-form-item>
        <el-form-item :label="$tr('启用')"
          ><el-switch v-model="form.enabled"
        /></el-form-item>
        <p>
          {{
            $tr(
              "Cron 使用 Worker 的本地时区。备份与其他集群变更互斥。保留天数记录计划保存周期，手动清理不受该时间限制；备份完成后请到恢复演练进行验证。"
            )
          }}
        </p>
      </template>
    </el-form>
    <template #footer
      ><el-button @click="$emit('update:modelValue', false)">{{
        $tr("取消")
      }}</el-button
      ><el-button type="primary" :loading="busy" @click="save">{{
        $tr("保存")
      }}</el-button></template
    >
  </el-dialog>
</template>
<script setup lang="ts">
import { tr } from "@/locales";

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
  policy?: any;
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
    ? [
        {
          value: "rdb",
          get label() {
            return tr("RDB / 按分片备份");
          }
        }
      ]
    : [
        {
          value: "logical",
          get label() {
            return tr("逻辑备份");
          }
        },
        {
          value: "physical",
          get label() {
            return tr("物理全量（需配置匹配工具）");
          }
        },
        {
          value: "incremental",
          get label() {
            return tr("增量备份（要求已有基线）");
          }
        },
        ...(selected.value?.engine === "postgresql"
          ? [
              {
                value: "differential",
                get label() {
                  return tr("差异备份");
                }
              },
              {
                value: "wal-archive",
                get label() {
                  return tr("导出基线与 WAL 归档");
                }
              }
            ]
          : [
              {
                value: "binlog-archive",
                get label() {
                  return tr("导出基线与已关闭 binlog");
                }
              }
            ])
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
      props.kind === "repository"
        ? "repositories"
        : props.policy
          ? `backup-policies/${props.policy.id}`
          : "backup-policies",
      props.kind === "policy" && props.policy ? "patch" : "post",
      { ...form, repositoryId: form.repositoryId || 0 }
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
        if (props.policy) {
          Object.assign(form, props.policy);
          schedule.value =
            form.cron === "0 2 * * *"
              ? "daily"
              : form.cron === "0 2 * * 0"
                ? "weekly"
                : "custom";
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
