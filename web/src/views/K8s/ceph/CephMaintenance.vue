<script setup lang="ts">
import { tr } from "@/locales";

import {
  computed,
  onActivated,
  onDeactivated,
  onUnmounted,
  reactive,
  ref,
  watch
} from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  cephApi,
  type CephOperation,
  type CephTask,
  type CephRepository,
  type CephBackup,
  type CephSchedule
} from "@/api/ceph";

const props = defineProps<{ clusterId: string | number; active: boolean }>();
const emit = defineEmits<{ changed: [] }>();
const operations = ref<CephOperation[]>([]);
const canWrite = ref(false);
const error = ref("");
const tab = ref("actions");
const busy = ref(false);
const submitting = ref(false);
const tasks = ref<CephTask[]>([]);
const repositories = ref<CephRepository[]>([]);
const backups = ref<CephBackup[]>([]);
const schedules = ref<CephSchedule[]>([]);
const repository = ref("");
const operationId = ref("backup-full");
const params = reactive<Record<string, string>>({});
const confirmation = ref("");
const timeoutHours = ref(6);
const scheduled = ref(false);
const scheduleForm = reactive({
  name: "",
  schedule: "0 2 * * *",
  timeZone: "Asia/Shanghai"
});
const repositoryDialog = ref(false);
const repositoryForm = reactive({ name: "", pvc: "" });
const logsDialog = ref(false);
const selectedTask = ref("");
const logs = ref<Awaited<ReturnType<typeof cephApi.logs>> | null>(null);
let generation = 0;
let taskQuery = 0;
let timer: ReturnType<typeof setTimeout> | undefined;
let disposed = false;
const selected = computed(() =>
  operations.value.find(item => item.id === operationId.value)
);
const groups = computed(() => [
  ...new Set(operations.value.map(item => item.group))
]);
const running = computed(
  () =>
    tasks.value.filter(item =>
      ["Pending", "Running", "Cancelling"].includes(item.state)
    ).length
);
const stateLabels: Record<string, string> = {
  Pending: "等待调度",
  Running: "执行中",
  Succeeded: "成功",
  Failed: "失败",
  Cancelling: "正在取消",
  Cancelled: "已取消"
};
const stateType = (state: string) =>
  state === "Succeeded"
    ? "success"
    : state === "Failed"
      ? "danger"
      : state === "Running"
        ? "primary"
        : "info";
const opLabel = (id: string) =>
  operations.value.find(item => item.id === id)?.label || id;
const date = (value: string | null) =>
  value
    ? new Date(value).toLocaleString("zh-CN", { timeZone: "Asia/Shanghai" })
    : "—";
const pretty = (value: unknown) => JSON.stringify(value, null, 2);
const bytes = (value: number) =>
  value ? `${(value / 1073741824).toFixed(2)} GiB` : "0 GiB";

function resetForm() {
  Object.keys(params).forEach(key => delete params[key]);
  selected.value?.fields.forEach(field => {
    params[field.key] = field.options?.[0] || "";
  });
  if (selected.value?.fields.some(field => field.key === "repository"))
    params.repository = repository.value;
  confirmation.value = "";
  scheduled.value = false;
}
watch(operationId, resetForm);

async function refresh() {
  if (!props.clusterId || !props.active || busy.value) return;
  const version = generation;
  const query = ++taskQuery;
  const cluster = props.clusterId;
  busy.value = true;
  try {
    const [cap, jobs, repos, plans] = await Promise.all([
      cephApi.capabilities(cluster),
      cephApi.tasks(cluster),
      cephApi.repositories(cluster),
      cephApi.schedules(cluster)
    ]);
    if (version !== generation) return;
    operations.value = cap.operations;
    canWrite.value = cap.canWrite;
    if (query === taskQuery) tasks.value = jobs;
    repositories.value = repos;
    schedules.value = plans;
    if (!repository.value) repository.value = repos[0]?.name || "";
    if (!Object.keys(params).length) resetForm();
    error.value = "";
  } catch (e) {
    if (version === generation) error.value = (e as Error).message;
  } finally {
    if (version === generation) busy.value = false;
  }
}

async function poll() {
  if (disposed || !props.active || !props.clusterId) return;
  const version = generation;
  const query = ++taskQuery;
  try {
    const jobs = await cephApi.tasks(props.clusterId);
    if (version !== generation || query !== taskQuery) return;
    const completed = jobs.some(
      job =>
        ["Succeeded", "Failed"].includes(job.state) &&
        tasks.value.some(
          old =>
            old.name === job.name && ["Running", "Pending"].includes(old.state)
        )
    );
    tasks.value = jobs;
    if (completed) {
      emit("changed");
      if (tab.value === "backups") await loadBackups();
      await refresh();
    }
    if (logsDialog.value && selectedTask.value)
      await loadLogs(selectedTask.value);
  } catch (e) {
    if (version === generation) error.value = (e as Error).message;
  } finally {
    if (version === generation && props.active && !disposed)
      timer = setTimeout(poll, 10000);
  }
}

watch(
  () => [props.clusterId, props.active],
  async () => {
    generation++;
    clearTimeout(timer);
    busy.value = false;
    submitting.value = false;
    tasks.value = [];
    backups.value = [];
    schedules.value = [];
    repositories.value = [];
    canWrite.value = false;
    operations.value = [];
    repository.value = "";
    repositoryDialog.value = false;
    error.value = "";
    logsDialog.value = false;
    logs.value = null;
    inspection.value = null;
    Object.keys(params).forEach(key => delete params[key]);
    const version = generation;
    await refresh();
    if (version === generation && props.active && !disposed)
      timer = setTimeout(poll, 10000);
  },
  { immediate: false }
);

async function submit() {
  if (!selected.value || !canWrite.value || submitting.value) return;
  for (const field of selected.value.fields) {
    if (field.required && !params[field.key]?.trim()) {
      ElMessage.warning(`请填写${field.label}`);
      return;
    }
  }
  if (selected.value.dangerous && confirmation.value !== operationId.value) {
    ElMessage.warning(tr("请输入完整操作标识确认风险"));
    return;
  }
  if (scheduled.value && !scheduleForm.name) {
    ElMessage.warning(tr("请填写计划名称"));
    return;
  }
  const version = generation;
  const cluster = props.clusterId;
  const task = {
    operation: operationId.value,
    params: Object.fromEntries(
      Object.entries(params).filter(([, value]) => value !== "")
    ),
    confirmation: confirmation.value,
    timeoutSeconds: timeoutHours.value * 3600
  };
  submitting.value = true;
  try {
    if (scheduled.value)
      await cephApi.createSchedule(cluster, { ...scheduleForm, task });
    else await cephApi.createTask(cluster, task);
    if (version !== generation) return;
    ElMessage.success(scheduled.value ? "定时计划已创建" : "维护任务已提交");
    tab.value = scheduled.value ? "schedules" : "tasks";
    confirmation.value = "";
    await refresh();
  } catch (e) {
    if (version === generation) error.value = (e as Error).message;
  } finally {
    if (version === generation) submitting.value = false;
  }
}

async function loadBackups() {
  backups.value = [];
  if (!repository.value) return;
  const version = generation;
  const repo = repository.value;
  try {
    const result = await cephApi.backups(props.clusterId, repo);
    if (version === generation && repo === repository.value)
      backups.value = result;
  } catch (e) {
    if (version === generation && repo === repository.value)
      error.value = (e as Error).message;
  }
}
watch(repository, loadBackups);
watch(tab, value => {
  if (value === "backups") loadBackups();
});

async function backupAction(op: string, backup: CephBackup) {
  operationId.value = op;
  // Wait for operation watcher before pre-filling fields.
  await Promise.resolve();
  resetForm();
  params.repository = repository.value;
  if (op === "backup-incremental") {
    params.baseBackup = backup.id;
    params.pool = backup.pool;
    params.image = backup.image;
  } else params.backupId = backup.id;
  tab.value = "actions";
}
async function registerRepository() {
  const version = generation;
  const request = { ...repositoryForm };
  submitting.value = true;
  try {
    await cephApi.createRepository(props.clusterId, request);
    if (version !== generation) return;
    repositoryDialog.value = false;
    await refresh();
    if (version !== generation) return;
    repository.value = request.name;
    ElMessage.success(tr("仓库已登记，目录读取组件就绪后可刷新备份列表"));
  } catch (e) {
    if (version === generation) error.value = (e as Error).message;
  } finally {
    if (version === generation) submitting.value = false;
  }
}
async function loadLogs(name: string) {
  const version = generation;
  try {
    const result = await cephApi.logs(props.clusterId, name);
    if (version === generation && selectedTask.value === name)
      logs.value = result;
  } catch (e) {
    if (version === generation) error.value = (e as Error).message;
  }
}
function showLogs(name: string) {
  selectedTask.value = name;
  logs.value = null;
  logsDialog.value = true;
  loadLogs(name);
}
async function cancelTask(task: CephTask) {
  const cluster = props.clusterId;
  const version = generation;
  try {
    await ElMessageBox.confirm(
      `停止任务 ${task.name}？已完成的数据修改不会回滚；CephFS 克隆等服务端操作可能仍在继续。`,
      tr("取消维护任务"),
      { type: "warning" }
    );
    if (version !== generation || disposed || !props.active) return;
    await cephApi.cancel(cluster, task.name);
    if (version !== generation || disposed || !props.active) return;
    await refresh();
  } catch (e) {
    if (version === generation && e !== "cancel" && e !== "close")
      error.value = (e as Error).message;
  }
}
async function changeSchedule(item: CephSchedule, remove = false) {
  const cluster = props.clusterId;
  const version = generation;
  try {
    if (remove) {
      await ElMessageBox.confirm(
        `删除计划 ${item.name}？已有任务、快照及备份会保留。`,
        tr("删除计划"),
        { type: "warning" }
      );
      if (version !== generation || disposed || !props.active) return;
      await cephApi.deleteSchedule(cluster, item.name);
    } else await cephApi.suspendSchedule(cluster, item.name, !item.suspended);
    if (version !== generation || disposed || !props.active) return;
    await refresh();
  } catch (e) {
    if (version === generation && e !== "cancel" && e !== "close")
      error.value = (e as Error).message;
  }
}

const inspections = [
  { id: "rbd-status", label: "RBD 挂载与迁移状态", fields: ["pool", "image"] },
  {
    id: "rbd-children",
    label: "快照克隆依赖",
    fields: ["pool", "image", "snapshot"]
  },
  { id: "rbd-trash", label: "RBD 回收站", fields: ["pool"] },
  { id: "rbd-mirror", label: "RBD 镜像复制状态", fields: ["pool"] },
  { id: "cephfs-groups", label: "CephFS 子卷组", fields: ["filesystem"] },
  {
    id: "cephfs-subvolumes",
    label: "CephFS 子卷列表",
    fields: ["filesystem", "group"]
  },
  {
    id: "cephfs-info",
    label: "CephFS 子卷详情/配额",
    fields: ["filesystem", "group", "subvolume"]
  },
  {
    id: "cephfs-snapshots",
    label: "CephFS 快照列表",
    fields: ["filesystem", "group", "subvolume"]
  },
  {
    id: "cephfs-clone-status",
    label: "CephFS 克隆进度",
    fields: ["filesystem", "group", "subvolume"]
  },
  { id: "rgw-users", label: "RGW 用户列表", fields: [] },
  { id: "rgw-user", label: "RGW 用户及配额", fields: ["uid"] },
  { id: "rgw-buckets", label: "RGW 桶列表", fields: [] },
  { id: "rgw-bucket-stats", label: "RGW 桶用量", fields: ["bucket"] },
  { id: "osd-tree", label: "OSD 拓扑", fields: [] },
  { id: "osd-perf", label: "OSD 延迟", fields: [] },
  { id: "osd-flags", label: "OSD 集群标志", fields: [] },
  { id: "pg-stuck", label: "卡住的 PG", fields: [] },
  { id: "balancer", label: "自动均衡状态", fields: [] },
  { id: "pool-autoscale", label: "PG 自动伸缩状态", fields: [] }
];
const inspectKind = ref("rbd-status");
const inspectParams = reactive<Record<string, string>>({});
const inspecting = ref(false);
const inspection = ref<unknown>(null);
const inspectFields = computed(
  () => inspections.find(item => item.id === inspectKind.value)?.fields || []
);
const fieldLabels: Record<string, string> = {
  pool: "存储池",
  image: "镜像",
  snapshot: "快照",
  filesystem: "文件系统",
  group: "子卷组（可选）",
  subvolume: "子卷",
  uid: "RGW 用户",
  bucket: "桶名称"
};
const inspectRows = computed(() =>
  Array.isArray(inspection.value)
    ? inspection.value.map(item =>
        typeof item === "object" && item !== null ? item : { name: item }
      )
    : []
);
const inspectColumns = computed(() =>
  [...new Set(inspectRows.value.flatMap(row => Object.keys(row)))].slice(0, 8)
);
watch(inspectKind, () => {
  inspection.value = null;
  Object.keys(inspectParams).forEach(key => delete inspectParams[key]);
});
async function inspect() {
  const version = generation;
  const kind = inspectKind.value;
  inspecting.value = true;
  inspection.value = null;
  try {
    const result = await cephApi.inspect(props.clusterId, kind, {
      ...inspectParams
    });
    if (version === generation && kind === inspectKind.value)
      inspection.value = result;
  } catch (e) {
    if (version === generation) error.value = (e as Error).message;
  } finally {
    inspecting.value = false;
  }
}

onActivated(() => {
  disposed = false;
  clearTimeout(timer);
  refresh();
  timer = setTimeout(poll, 10000);
});
onDeactivated(() => {
  disposed = true;
  generation++;
  clearTimeout(timer);
  busy.value = false;
});
onUnmounted(() => {
  disposed = true;
  generation++;
  clearTimeout(timer);
});
// First mount also works when the parent is not wrapped in KeepAlive.
refresh().then(() => {
  clearTimeout(timer);
  if (!disposed) timer = setTimeout(poll, 10000);
});
</script>

<template>
  <section class="ceph-maintenance">
    <div class="maintenance-header">
      <div>
        <h3>{{ $tr("存储维护中心") }}</h3>
        <p>{{ $tr("迁移、备份恢复和日常维护。") }}{{ running }} {{ $tr("个任务等待或正在执行。") }}</p>
      </div>
      <el-button :loading="busy" @click="refresh">{{ $tr("刷新") }}</el-button>
    </div>
    <el-alert
      v-if="error"
      :title="$tr(error)"
      type="error"
      show-icon
      closable
      @close="error = ''"
    />
    <el-alert
      v-if="!canWrite && operations.length"
      :title='$tr("当前为只读模式；提交任务需要 Ceph 维护写权限。")'
      type="info"
      :closable="false"
    />
    <el-tabs v-model="tab">
      <el-tab-pane :label='$tr("维护操作")' name="actions">
        <div class="action-layout">
          <el-form
            label-position="top"
            class="action-form"
            @submit.prevent="submit"
          >
            <el-form-item :label='$tr("操作类型")'>
              <el-select v-model="operationId" filterable>
                <el-option-group
                  v-for="group in groups"
                  :key="group"
                  :label="$tr(group)"
                >
                  <el-option
                    v-for="item in operations.filter(
                      item => item.group === group
                    )"
                    :key="item.id"
                    :label="$tr(item.label)"
                    :value="item.id"
                  />
                </el-option-group>
              </el-select>
            </el-form-item>
            <el-alert
              v-if="selected?.description"
              :title="$tr(selected.description)"
              :type="selected.dangerous ? 'warning' : 'info'"
              :closable="false"
              show-icon
            />
            <el-form-item
              v-for="field in selected?.fields || []"
              :key="field.key"
              :label="$tr(field.label)"
              :required="field.required"
            >
              <el-select
                v-if="field.key === 'repository'"
                v-model="params[field.key]"
                :placeholder='$tr("请先在备份仓库登记 PVC")'
              >
                <el-option
                  v-for="repo in repositories"
                  :key="repo.name"
                  :label="`${repo.name} (${repo.pvc})`"
                  :value="repo.name"
                />
              </el-select>
              <el-select
                v-else-if="field.options?.length"
                v-model="params[field.key]"
              >
                <el-option
                  v-for="option in field.options"
                  :key="option"
                  :label="$tr(option)"
                  :value="option"
                />
              </el-select>
              <el-input
                v-else
                v-model="params[field.key]"
                :maxlength="128"
                :placeholder="$tr(field.hint)"
              />
              <div v-if="field.hint" class="hint">{{ field.hint }}</div>
            </el-form-item>
            <el-form-item :label='$tr("最长执行时间（小时）")'
              ><el-input-number v-model="timeoutHours" :min="1" :max="168"
            /></el-form-item>
            <el-form-item v-if="selected?.schedulable" :label='$tr("执行方式")'
              ><el-switch
                v-model="scheduled"
                :active-text='$tr("定时执行")'
                :inactive-text='$tr("立即执行")'
            /></el-form-item>
            <template v-if="scheduled">
              <el-form-item :label='$tr("计划名称")' required
                ><el-input
                  v-model="scheduleForm.name"
                  :placeholder='$tr("例如 daily-backup，小写字母、数字和连字符")'
                  :maxlength="32"
              /></el-form-item>
              <el-form-item :label='$tr("Cron（分 时 日 月 周）")' required
                ><el-input
                  v-model="scheduleForm.schedule"
                  placeholder="0 2 * * *"
              /></el-form-item>
              <el-form-item :label='$tr("时区")'
                ><el-input v-model="scheduleForm.timeZone"
              /></el-form-item>
              <el-alert
                :title='$tr("同一计划不会并发运行。备份及快照不会自动清理，请定期检查容量和备份链。")'
                type="info"
                :closable="false"
              />
            </template>
            <el-form-item
              v-if="selected?.dangerous"
              :label="`确认风险：输入 ${selected.id}`"
              required
              ><el-input
                v-model="confirmation"
                :placeholder="$tr(selected.id)"
                autocomplete="off"
            /></el-form-item>
            <el-button
              type="primary"
              :disabled="!canWrite || !selected"
              :loading="submitting"
              native-type="submit"
              >{{ scheduled ? "创建定时计划" : "提交维护任务" }}</el-button
            >
          </el-form>
          <aside class="maintenance-guide">
            <h4>{{ $tr("迁移流程") }}</h4>
            <ol>
              <li>{{ $tr("停写并检查挂载、快照及可用容量。") }}</li>
              <li>{{ $tr("复制到新镜像，或依次执行迁移准备和数据复制。") }}</li>
              <li>{{ $tr("校验数据与应用，再提交迁移或切换业务。") }}</li>
            </ol>
            <h4>{{ $tr("备份流程") }}</h4>
            <ol>
              <li>{{ $tr("登记独立存储上的 RWX PVC 仓库。") }}</li>
              <li>{{ $tr("创建全量备份，再基于备份编号创建增量。") }}</li>
              <li>{{ $tr("定期校验，并恢复到新镜像进行演练。") }}</li>
            </ol>
            <p>
              {{ $tr("快照保留在源集群，无法替代异地备份。数据库备份前请由应用完成停写或一致性快照。") }}
            </p>
            <p>
              {{ $tr("CSI 管理的卷请使用 PVC 扩容、迁移和换卷流程。直接修改底层镜像不会更新 PVC 元数据。") }}
            </p>
          </aside>
        </div>
      </el-tab-pane>
      <el-tab-pane :label="`任务记录 (${tasks.length})`" name="tasks">
        <el-alert
          :title='$tr("任务每 10 秒刷新。长任务由集群持续执行；取消不会撤销已完成的修改。巡检、修复命令成功后仍需观察 Ceph 状态。")'
          type="info"
          :closable="false"
        />
        <el-table :data="tasks" stripe :empty-text='$tr("暂无维护任务")'>
          <el-table-column type="expand"
            ><template #default="{ row }">
              <pre>{{ pretty(row.params) }}</pre>
              <p>{{ row.message }}</p></template
            ></el-table-column
          >
          <el-table-column
            prop="name"
            :label='$tr("任务编号")'
            min-width="210"
            show-overflow-tooltip
          />
          <el-table-column :label='$tr("操作")' min-width="160"
            ><template #default="{ row }">{{
              opLabel(row.operation)
            }}</template></el-table-column
          >
          <el-table-column :label='$tr("状态")' width="110"
            ><template #default="{ row }"
              ><el-tag :type="stateType(row.state)">{{
                stateLabels[row.state] || row.state
              }}</el-tag></template
            ></el-table-column
          >
          <el-table-column prop="actor" :label='$tr("发起人")' width="110" />
          <el-table-column :label='$tr("创建时间（上海）")' min-width="185"
            ><template #default="{ row }">{{
              date(row.createdAt)
            }}</template></el-table-column
          >
          <el-table-column :label='$tr("操作")' width="155" fixed="right"
            ><template #default="{ row }"
              ><el-button link type="primary" @click="showLogs(row.name)"
                >{{ $tr("日志") }}</el-button
              ><el-button
                v-if="['Pending', 'Running'].includes(row.state)"
                link
                type="danger"
                :disabled="!canWrite"
                @click="cancelTask(row)"
                >{{ $tr("取消") }}</el-button
              ></template
            ></el-table-column
          >
        </el-table>
      </el-tab-pane>
      <el-tab-pane :label='$tr("备份仓库")' name="backups">
        <div class="toolbar">
          <el-select v-model="repository" :placeholder='$tr("选择仓库")'
            ><el-option
              v-for="repo in repositories"
              :key="repo.name"
              :label="`${repo.name} (${repo.pvc})`"
              :value="repo.name" /></el-select
          ><el-button @click="loadBackups">{{ $tr("刷新目录") }}</el-button
          ><el-button
            type="primary"
            :disabled="!canWrite"
            @click="repositoryDialog = true"
            >{{ $tr("登记仓库") }}</el-button
          >
        </div>
        <el-alert
          :title='$tr("目录读取持久化 PVC 中的备份清单，任务记录清理后仍可恢复。文件存在不代表校验通过，请执行校验并定期恢复演练。")'
          type="info"
          :closable="false"
        />
        <el-table
          :data="backups"
          stripe
          :empty-text='$tr("暂无备份，请选择已就绪的仓库并创建全量备份")'
        >
          <el-table-column type="expand"
            ><template #default="{ row }">
              <pre>{{ pretty(row) }}</pre>
            </template></el-table-column
          >
          <el-table-column
            prop="id"
            :label='$tr("备份编号")'
            min-width="210"
            show-overflow-tooltip
          />
          <el-table-column :label='$tr("源镜像")' min-width="160"
            ><template #default="{ row }"
              >{{ row.pool }}/{{ row.image }}</template
            ></el-table-column
          >
          <el-table-column :label='$tr("类型")' width="90"
            ><template #default="{ row }">{{
              row.parent ? $tr("增量") : "全量"
            }}</template></el-table-column
          >
          <el-table-column :label='$tr("文件大小")' width="115"
            ><template #default="{ row }">{{
              bytes(row.bytes)
            }}</template></el-table-column
          >
          <el-table-column :label='$tr("文件")' width="85"
            ><template #default="{ row }"
              ><el-tag :type="row.fileExists ? 'success' : 'danger'">{{
                row.fileExists ? "存在" : $tr("异常")
              }}</el-tag></template
            ></el-table-column
          >
          <el-table-column :label='$tr("创建时间（上海）")' min-width="185"
            ><template #default="{ row }">{{
              date(row.createdAt)
            }}</template></el-table-column
          >
          <el-table-column :label='$tr("操作")' min-width="230" fixed="right"
            ><template #default="{ row }"
              ><el-button
                link
                type="primary"
                :disabled="!canWrite || !row.fileExists"
                @click="backupAction('backup-restore', row)"
                >{{ $tr("恢复") }}</el-button
              ><el-button
                link
                :disabled="!canWrite"
                @click="backupAction('backup-verify', row)"
                >{{ $tr("校验") }}</el-button
              ><el-button
                link
                :disabled="!canWrite || !row.fileExists"
                @click="backupAction('backup-incremental', row)"
                >{{ $tr("增量") }}</el-button
              ><el-button
                link
                type="danger"
                :disabled="!canWrite"
                @click="backupAction('backup-delete', row)"
                >{{ $tr("删除") }}</el-button
              ></template
            ></el-table-column
          >
        </el-table>
      </el-tab-pane>
      <el-tab-pane :label='$tr("定时计划")' name="schedules">
        <el-table
          :data="schedules"
          :empty-text='$tr("在维护操作中选择创建快照或全量备份，开启定时执行")'
        >
          <el-table-column prop="name" :label='$tr("计划")' min-width="180" />
          <el-table-column :label='$tr("操作")' min-width="125"
            ><template #default="{ row }">{{
              opLabel(row.task.operation)
            }}</template></el-table-column
          >
          <el-table-column prop="schedule" label="Cron" min-width="125" />
          <el-table-column prop="timeZone" :label='$tr("时区")' min-width="135" />
          <el-table-column :label='$tr("状态")' width="100"
            ><template #default="{ row }"
              ><el-tag :type="row.suspended ? 'info' : 'success'">{{
                row.suspended ? $tr("暂停") : $tr("启用")
              }}</el-tag></template
            ></el-table-column
          >
          <el-table-column :label='$tr("上次成功（上海）")' min-width="185"
            ><template #default="{ row }">{{
              date(row.lastSuccessfulTime)
            }}</template></el-table-column
          >
          <el-table-column :label='$tr("操作")' width="145"
            ><template #default="{ row }"
              ><el-button
                link
                type="primary"
                :disabled="!canWrite"
                @click="changeSchedule(row)"
                >{{ row.suspended ? $tr("恢复") : $tr("暂停") }}</el-button
              ><el-button
                link
                type="danger"
                :disabled="!canWrite"
                @click="changeSchedule(row, true)"
                >{{ $tr("删除") }}</el-button
              ></template
            ></el-table-column
          >
        </el-table>
      </el-tab-pane>
      <el-tab-pane :label='$tr("高级诊断")' name="inspect">
        <el-form inline @submit.prevent="inspect"
          ><el-form-item :label='$tr("诊断项")'
            ><el-select v-model="inspectKind" class="inspect-select" filterable
              ><el-option
                v-for="item in inspections"
                :key="item.id"
                :value="item.id"
                :label="$tr(item.label)" /></el-select></el-form-item
          ><el-form-item
            v-for="field in inspectFields"
            :key="field"
            :label="$tr(fieldLabels[field])"
            ><el-input v-model="inspectParams[field]" /></el-form-item
          ><el-form-item
            ><el-button
              type="primary"
              native-type="submit"
              :loading="inspecting"
              >{{ $tr("查询") }}</el-button
            ></el-form-item
          ></el-form
        >
        <el-table v-if="inspectRows.length" :data="inspectRows"
          ><el-table-column
            v-for="column in inspectColumns"
            :key="column"
            :label="$tr(column)"
            min-width="150"
            ><template #default="{ row }">{{
              typeof row[column] === "object"
                ? pretty(row[column])
                : row[column]
            }}</template></el-table-column
          ></el-table
        >
        <pre v-if="inspection !== null">{{ pretty(inspection) }}</pre>
        <el-empty
          v-else
          :description='$tr("选择诊断项并查询。RGW 功能需集群已部署对象网关。")'
        />
      </el-tab-pane>
    </el-tabs>
    <el-dialog
      v-model="repositoryDialog"
      :title='$tr("登记备份仓库")'
      width="560px"
      :close-on-click-modal="false"
    >
      <el-alert
        :title='$tr("准备 rook-ceph 命名空间中的专用 RWX 文件系统 PVC，添加 autoops.io/ceph-backup=true 标签。优先使用独立 NFS 或异地存储。")'
        type="info"
        :closable="false"
      />
      <el-form label-position="top"
        ><el-form-item :label='$tr("仓库名")' required
          ><el-input
            v-model="repositoryForm.name"
            :placeholder='$tr("如 nfs-backup")'
            :maxlength="32" /></el-form-item
        ><el-form-item :label='$tr("PVC 名称")' required
          ><el-input
            v-model="repositoryForm.pvc"
            :placeholder='$tr("如 autoops-ceph-backups")' /></el-form-item
      ></el-form>
      <template #footer
        ><el-button @click="repositoryDialog = false">{{ $tr("取消") }}</el-button
        ><el-button
          type="primary"
          :disabled="!canWrite || !repositoryForm.name || !repositoryForm.pvc"
          :loading="submitting"
          @click="registerRepository"
          >{{ $tr("登记") }}</el-button
        ></template
      >
    </el-dialog>
    <el-dialog
      v-model="logsDialog"
      :title="`任务日志 · ${selectedTask}`"
      width="85%"
    >
      <el-button @click="loadLogs(selectedTask)">{{ $tr("刷新日志") }}</el-button>
      <el-alert
        v-if="logs"
        :title="`${stateLabels[logs.task.state]} ${logs.task.message || ''}`"
        type="info"
        :closable="false"
      />
      <el-empty
        v-if="!logs?.pods.length"
        :description='$tr("尚无任务 Pod，或日志已被集群清理")'
      />
      <section v-for="pod in logs?.pods || []" :key="pod.pod">
        <h4>{{ pod.pod }}</h4>
        <pre>{{ pod.log || pod.logError || "暂无日志" }}</pre>
        <el-collapse
          ><el-collapse-item :title='$tr("调度及容器状态")'>
            <pre>{{
              pretty({
                conditions: pod.conditions,
                containers: pod.containers,
                initContainers: pod.initContainers
              })
            }}</pre>
          </el-collapse-item></el-collapse
        >
      </section>
    </el-dialog>
  </section>
</template>

<style scoped>
.ceph-maintenance {
  color: var(--el-text-color-primary);
}
.maintenance-header,
.toolbar {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.maintenance-header h3 {
  margin: 0 0 8px;
}
.maintenance-header p,
.hint {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.toolbar {
  justify-content: flex-start;
  flex-wrap: wrap;
}
.toolbar .el-select {
  width: 300px;
}
.action-layout {
  display: grid;
  grid-template-columns: minmax(300px, 620px) minmax(240px, 1fr);
  gap: 32px;
}
.action-form .el-select {
  width: 100%;
}
.el-alert {
  margin-bottom: 18px;
}
.maintenance-guide {
  padding: 20px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-fill-color-light);
  line-height: 1.9;
  align-self: start;
}
.maintenance-guide h4 {
  margin: 0 0 10px;
}
.maintenance-guide ol {
  padding-left: 22px;
  margin-bottom: 24px;
}
.maintenance-guide p {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.hint {
  width: 100%;
  margin-top: 4px;
}
.inspect-select {
  width: 240px;
}
pre {
  max-height: 520px;
  padding: 16px;
  overflow: auto;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  background: var(--el-fill-color-light);
  border-radius: 6px;
  font-size: 12px;
}
@media (max-width: 1000px) {
  .action-layout {
    grid-template-columns: 1fr;
  }
}
</style>
