<template>
  <el-dialog
    :model-value="modelValue"
    :title="`确认执行 · ${label}`"
    width="min(780px, 94vw)"
    append-to-body
    @update:model-value="$emit('update:modelValue', $event)"
    ><el-alert v-if="error" :title="error" type="error" :closable="false" />
    <p class="operation-target">
      将在集群 <strong>{{ cluster.name }}</strong> 执行「{{ label }}」。
    </p>
    <el-alert
      v-if="action === 'backup'"
      type="info"
      :closable="false"
      show-icon
      title="选择备份方式与保存位置 → 预检并确认 → 在备份集查看文件与状态。完成备份后可继续执行恢复验证。"
    />
    <el-alert
      v-if="['restore', 'verify-backup'].includes(action)"
      type="info"
      :closable="false"
      show-icon
      title="恢复到新目录和独立端口，启动并校验后自动停止隔离实例；文件与报告保留，不自动替换业务数据库。"
    />
    <template v-if="action === 'configure-vip'">
      <el-form label-width="140px" :disabled="!!plan">
        <el-form-item label="虚拟 IP / 前缀"
          ><el-input
            v-model="parameters.vip.cidr"
            placeholder="已分配的 IPv4/CIDR"
        /></el-form-item>
        <el-form-item label="绑定网卡"
          ><el-input
            v-model="parameters.vip.interface"
            placeholder="例如 ens33"
        /></el-form-item>
        <el-form-item label="VRID"
          ><el-input-number
            v-model="parameters.vip.routerId"
            :min="1"
            :max="255"
        /></el-form-item>
        <el-form-item label="承载代理"
          ><el-select
            v-model="parameters.vip.memberIds"
            multiple
            :multiple-limit="2"
            ><el-option
              v-for="m in cluster.members.filter(m => m.kind === 'proxy')"
              :key="m.id"
              :value="m.id"
              :label="`${m.name} · ${m.address}`" /></el-select
        ></el-form-item>
      </el-form>
    </template>
    <el-alert
      v-if="action === 'verify-vip'"
      :closable="false"
      type="info"
      title="短暂停止当前承载节点的独立 VIP 服务，验证地址漂移与连续 SQL 连接，随后恢复原服务。已配置独立 60 秒自动恢复保护。"
    />
    <el-form label-width="140px" :disabled="!!plan"
      ><el-form-item v-if="action === 'reconcile'" label="结果待核验任务"
        ><el-select v-model="parameters.operationId"
          ><el-option
            v-for="op in pendingOperations"
            :key="op.id"
            :value="op.id"
            :label="`#${op.id} · ${op.action} · ${op.message || ''}`" /></el-select></el-form-item
      ><el-form-item v-if="action === 'adopt'" label="管理级别"
        ><el-select v-model="parameters.management"
          ><el-option value="observe" label="观察模式" /><el-option
            value="operate"
            label="运维托管" /><el-option
            value="ha"
            label="高可用托管（需验收）" /></el-select></el-form-item
      ><el-form-item
        v-if="
          [
            'switchover',
            'recover',
            'reshard',
            'add-node',
            'repair-replication',
            'assign-replica'
          ].includes(action) &&
          !(cluster.topology === 'redis-sentinel' && action === 'switchover')
        "
        label="目标候选"
        ><el-select v-model="parameters.candidateId"
          ><el-option
            v-for="m in cluster.members.filter(m => m.kind === 'data')"
            :key="m.id"
            :value="m.id"
            :label="`${m.name} · ${m.address}:${m.port} (${m.role})`" /></el-select></el-form-item
      ><el-alert
        v-if="cluster.topology === 'redis-sentinel' && action === 'switchover'"
        type="info"
        :closable="false"
        title="Sentinel 根据复制偏移、优先级与多数派选择新主，并自动重配副本。"
      />
      <template
        v-if="
          action === 'configure-endpoint' &&
          !cluster.members.some(m => m.kind === 'proxy')
        "
      >
        <el-form-item
          v-for="(proxy, index) in proxyMembers"
          :key="index"
          :label="`代理 ${index + 1}`"
        >
          <el-select v-model="proxy.hostId" filterable placeholder="CMDB 主机"
            ><el-option
              v-for="h in hosts"
              :key="h.id"
              :value="h.id"
              :label="`${h.hostname || h.hostName || h.name || h.id} · ${h.ipAddr || h.ip || ''}`"
          /></el-select>
          <el-input-number v-model="proxy.port" :min="1024" :max="65534" />
        </el-form-item>
        <el-alert
          type="info"
          :closable="false"
          title="写入口使用填写的端口，只读入口使用端口 +1。需要两个不同主机；工作台支持双代理地址，VIP 可后续绑定。"
        />
      </template>
      <template v-if="action === 'configure-semisync'">
        <el-form-item label="启用半同步"
          ><el-switch v-model="parameters.semiSyncEnabled"
        /></el-form-item>
        <el-form-item label="超时（毫秒）"
          ><el-input-number
            v-model="parameters.semiSyncTimeoutMs"
            :min="100"
            :max="60000"
        /></el-form-item>
        <el-alert
          type="warning"
          :closable="false"
          title="半同步超时可能退化为异步；这项设置不保证严格零数据丢失。"
        />
      </template>
      <template v-if="action === 'add-node'">
        <el-form-item label="新增实例主机"
          ><el-select v-model="newMember.hostId" filterable
            ><el-option
              v-for="h in hosts"
              :key="h.id"
              :value="h.id"
              :label="`${h.hostname || h.hostName || h.name || h.id} · ${h.ipAddr || h.ip || ''}`" /></el-select
        ></el-form-item>
        <el-form-item label="独立端口"
          ><el-input-number v-model="newMember.port" :min="1024" :max="55000"
        /></el-form-item>
        <el-form-item
          v-if="cluster.topology === 'redis-cluster'"
          label="节点用途"
          ><el-select v-model="newMember.role"
            ><el-option label="已有分片的副本" value="replica" /><el-option
              label="新增主分片（初始无槽位）"
              value="primary" /></el-select
        ></el-form-item>
        <el-alert
          type="info"
          :closable="false"
          title="系统会生成独立目录与服务名。新增主分片后，需配置副本并通过迁槽任务分配容量。"
        /> </template
      ><el-form-item
        v-if="
          [
            'restart',
            'rebuild-replica',
            'rejoin',
            'assign-replica',
            'remove-node',
            'reshard',
            'redis-bgsave',
            'redis-aof-rewrite'
          ].includes(action)
        "
        label="操作节点"
        ><el-select v-model="parameters.memberId"
          ><el-option
            v-for="m in cluster.members.filter(m => m.kind === 'data')"
            :key="m.id"
            :value="m.id"
            :label="`${m.name} · ${m.address}:${m.port}`" /></el-select></el-form-item
      ><el-form-item v-if="action === 'backup'" label="备份方式"
        ><el-select v-model="parameters.method"
          ><el-option
            v-if="cluster.engine !== 'redis'"
            value="logical"
            label="逻辑备份" /><el-option
            v-if="cluster.engine !== 'redis'"
            value="physical"
            label="物理备份（需匹配版本的工具）" /><template
            v-if="cluster.engine === 'postgresql'"
            ><el-option
              value="incremental"
              label="pgBackRest 增量备份" /><el-option
              value="differential"
              label="pgBackRest 差异备份" /><el-option
              value="wal-archive"
              label="导出物理基线与连续 WAL" /></template
          ><el-option
            v-if="cluster.engine === 'mysql'"
            value="incremental"
            label="XtraBackup 增量备份" /><el-option
            v-if="cluster.engine === 'mysql'"
            value="binlog-archive"
            label="导出物理基线与连续 binlog" /><el-option
            v-if="cluster.engine === 'redis'"
            value="rdb"
            label="RDB / 按分片备份" /></el-select
      ></el-form-item>
      <p v-if="action === 'backup'" class="field-help">
        {{ backupMethods[parameters.method]?.description }}
      </p>
      <el-form-item v-if="action === 'backup'" label="保存位置"
        ><el-select
          v-model="parameters.repositoryId"
          clearable
          placeholder="数据库节点本地暂存（无独立副本）"
          ><el-option
            v-for="r in repositories.filter(
              r => r.groupId === cluster.groupId && r.status === 'healthy'
            )"
            :key="r.id"
            :value="r.id"
            :label="r.name" /></el-select
      ></el-form-item>
      <p v-if="action === 'backup'" class="field-help">
        {{
          parameters.repositoryId
            ? "仅显示同业务分组、连接检查通过的仓库；上传并校验后才显示独立副本成功。"
            : "不选择仓库时保存在数据库节点。若需要独立副本，请先登记备份仓库并完成连接检查。"
        }}
      </p>
      <el-form-item v-if="action === 'purge-binlogs'" label="保留起始文件"
        ><el-input
          v-model="parameters.beforeFile"
          placeholder="mysql-bin.000123；此前日志须通过归档与依赖检查"
      /></el-form-item>
      <template
        v-if="['restore', 'verify-backup', 'expire-backup'].includes(action)"
        ><el-form-item label="备份集"
          ><el-select
            v-model="parameters.backupId"
            @change="parameters.targetTime = ''"
            ><el-option
              v-for="b in backups.filter(b => b.status === 'complete')"
              :key="b.id"
              :value="b.id"
              :label="`#${b.id} · ${backupMethodLabel(b.method)} · ${new Date(b.createdAt).toLocaleString()}`" /></el-select
        ></el-form-item>
        <p v-if="selectedBackup" class="field-help">
          {{ backupMethodLabel(selectedBackup.method) }} ·
          {{
            selectedBackup.manifest?.remoteComplete
              ? "从独立仓库读取并核验"
              : "从数据库节点本地文件读取"
          }}{{
            selectedBackup.parentId
              ? ` · 依赖备份 #${selectedBackup.parentId}`
              : ""
          }}
        </p>
        <el-form-item v-if="action !== 'expire-backup'" label="隔离目标主机"
          ><el-select v-model="parameters.hostId" filterable
            ><el-option
              v-for="h in hosts"
              :key="h.id"
              :value="h.id"
              :label="`${h.hostname || h.hostName || h.name || h.id}`" /></el-select></el-form-item
        ><el-form-item v-if="action !== 'expire-backup'" label="独立恢复端口"
          ><el-input-number v-model="parameters.port" :min="1024" :max="55000"
        /></el-form-item>
        <p v-if="action !== 'expire-backup'" class="field-help">
          目标主机需具备部署权限和可用资源。端口必须空闲；Redis Cluster
          按分片从起始端口依次分配。
        </p>
        <el-form-item
          v-if="action !== 'expire-backup' && restoreWindow"
          :label="
            selectedBackup?.method === 'binlog-archive'
              ? '目标时间（必填）'
              : '目标时间（可选）'
          "
        >
          <el-date-picker
            v-model="parameters.targetTime"
            type="datetime"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
            placeholder="选择归档覆盖范围内的时间"
          />
        </el-form-item>
        <p v-if="action !== 'expire-backup'" class="field-help">
          {{
            restoreWindow
              ? `可恢复范围：${new Date(restoreWindow.start).toLocaleString()} 至 ${new Date(restoreWindow.end).toLocaleString()}（当前浏览器时区）。MySQL 时间精确到秒，不包含目标秒及之后的事件。`
              : "此备份没有完整的归档时间窗口，只能恢复备份内容。"
          }}
        </p>
      </template>
      <el-form-item v-if="action === 'reshard'" label="迁移槽位"
        ><el-input
          v-model="slotsInput"
          placeholder="例如 100-199 或 100,101；每批最多 1024" /></el-form-item
      ><template v-if="action === 'configure'"
        ><el-form-item
          v-for="(change, i) in changes"
          :key="i"
          label="参数与期望值"
          ><el-select v-model="change.name" placeholder="参数"
            ><el-option
              v-for="d in definitions"
              :key="d.name"
              :value="d.name"
              :label="`${d.name} (${d.mode})`" /></el-select
          ><el-input
            v-model="change.value"
            style="width: 160px; margin-left: 10px"
            placeholder="期望值"
          /><el-button link type="danger" @click="changes.splice(i, 1)"
            >移除</el-button
          ></el-form-item
        ><el-button @click="changes.push({ name: '', value: '' })"
          >增加参数</el-button
        ></template
      ></el-form
    ><template v-if="plan"
      ><el-divider>预检结果 · 拓扑版本 {{ plan.revision }}</el-divider
      ><el-table :data="plan.checks" max-height="340"
        ><el-table-column
          prop="name"
          label="检查项"
          min-width="200"
        /><el-table-column label="结果" width="80"
          ><template #default="{ row }"
            ><el-tag :type="row.passed ? 'success' : 'danger'">{{
              row.passed ? "通过" : "阻断"
            }}</el-tag></template
          ></el-table-column
        ><el-table-column label="原因" min-width="240"
          ><template #default="{ row }">{{
            row.passed ? "已满足" : row.message
          }}</template></el-table-column
        ></el-table
      ></template
    ><template #footer
      ><el-button @click="$emit('update:modelValue', false)">取消</el-button
      ><el-button v-if="plan" :disabled="busy" @click="resetPlan"
        >调整参数</el-button
      ><el-button v-if="!plan" type="primary" :loading="busy" @click="preflight"
        >预检</el-button
      ><el-button
        v-else
        type="primary"
        :loading="busy"
        :disabled="!plan.executable || !key"
        @click="submit"
        >确认执行</el-button
      ></template
    ></el-dialog
  >
</template>
<script setup lang="ts">
import { computed, ref, reactive, watch } from "vue";
import { ElMessageBox } from "element-plus";
import {
  dbPlatform,
  dbRequest,
  operationPlanKey,
  type Cluster,
  type Plan
} from "@/api/databasePlatform";
import {
  backupMethodLabel,
  backupMethods,
  recoveryWindow
} from "@/utils/databaseBackup";
import cmdb from "@/api/cmdb";
const props = defineProps<{
  modelValue: boolean;
  cluster: Cluster;
  action: string;
  label: string;
  initialParameters?: Record<string, any>;
}>();
const emit = defineEmits(["update:modelValue", "submitted"]);
const busy = ref(false),
  error = ref(""),
  plan = ref<Plan>(),
  key = ref(""),
  backups = ref<any[]>([]),
  hosts = ref<any[]>([]),
  definitions = ref<any[]>([]),
  changes = ref([{ name: "", value: "" }]),
  slotsInput = ref("");
const parameters = reactive<Record<string, any>>({});
const selectedBackup = computed(() =>
  backups.value.find(b => b.id === parameters.backupId)
);
const restoreWindow = computed(() => recoveryWindow(selectedBackup.value));
const newMember = reactive({
  hostId: undefined as number | undefined,
  port: 18382,
  kind: "data",
  role: "replica"
});
const proxyMembers = reactive([
  { hostId: undefined as number | undefined, port: 23306, kind: "proxy" },
  { hostId: undefined as number | undefined, port: 23306, kind: "proxy" }
]);
const repositories = ref<any[]>([]);
const pendingOperations = ref<any[]>([]);
let preparationVersion = 0;
function resetPlan() {
  preparationVersion++;
  plan.value = undefined;
  key.value = "";
}
const requiresParameters = computed(() => {
  if (
    props.action === "switchover" &&
    props.cluster.topology === "redis-sentinel"
  )
    return false;
  if (props.action === "configure-endpoint")
    return !props.cluster.members.some(m => m.kind === "proxy");
  return [
    "adopt",
    "reconcile",
    "switchover",
    "recover",
    "reshard",
    "add-node",
    "repair-replication",
    "assign-replica",
    "configure-semisync",
    "configure-vip",
    "restart",
    "rebuild-replica",
    "rejoin",
    "remove-node",
    "redis-bgsave",
    "redis-aof-rewrite",
    "backup",
    "purge-binlogs",
    "restore",
    "verify-backup",
    "expire-backup",
    "configure"
  ].includes(props.action);
});
async function preflight() {
  if (busy.value) return;
  resetPlan();
  const version = preparationVersion;
  busy.value = true;
  error.value = "";
  try {
    if (["restore", "verify-backup"].includes(props.action)) {
      if (
        selectedBackup.value?.method === "binlog-archive" &&
        !parameters.targetTime
      )
        throw new Error("binlog 归档恢复必须选择归档范围内的目标时间");
      if (parameters.targetTime) {
        const target = Date.parse(parameters.targetTime);
        if (
          !restoreWindow.value ||
          !Number.isFinite(target) ||
          target < restoreWindow.value.start ||
          target > restoreWindow.value.end
        )
          throw new Error("目标时间不在已核验的连续归档范围内");
      }
    }
    if (props.action === "add-node") parameters.members = [{ ...newMember }];
    if (
      props.action === "configure-endpoint" &&
      !props.cluster.members.some(m => m.kind === "proxy")
    )
      parameters.members = proxyMembers.map(p => ({ ...p }));
    if (props.action === "configure")
      parameters.changes = Object.fromEntries(
        changes.value.map(c => [c.name, c.value])
      );
    if (props.action === "reshard") {
      const slots: number[] = [];
      for (const part of slotsInput.value.split(",")) {
        const [start, end] = part.trim().split("-").map(Number);
        if (
          !Number.isInteger(start) ||
          (end != null && (!Number.isInteger(end) || end < start)) ||
          slots.length + (end == null ? 1 : end - start + 1) > 1024
        )
          throw new Error("槽位范围无效或超过 1024");
        for (let n = start; n <= (end ?? start); n++) slots.push(n);
      }
      parameters.slots = slots;
    }
    const prepared = await dbPlatform.plan(
      props.cluster.id,
      props.action,
      parameters
    );
    const preparedKey = operationPlanKey(prepared);
    if (version !== preparationVersion || !props.modelValue) return;
    key.value = preparedKey;
    plan.value = prepared;
  } catch (e) {
    if (version === preparationVersion) error.value = (e as Error).message;
  } finally {
    if (version === preparationVersion) busy.value = false;
  }
}
async function submit() {
  if (busy.value || !plan.value?.executable || !key.value) return;
  const version = preparationVersion;
  busy.value = true;
  error.value = "";
  try {
    const target = props.cluster.members.find(
      m =>
        m.id === parameters.candidateId ||
        (!parameters.candidateId && m.id === parameters.memberId)
    );
    const targetText = target
      ? `目标节点：${target.name}（${target.address}:${target.port}）。`
      : "";
    const impact = ["switchover", "recover", "verify-vip", "restart"].includes(
      props.action
    )
      ? "执行期间连接可能短暂中断，客户端需要重连。"
      : "";
    await ElMessageBox.confirm(
      `确认在集群「${props.cluster.name}」执行「${props.label}」？${targetText}${impact}`,
      "执行确认",
      {
        confirmButtonText: "确认执行",
        cancelButtonText: "取消",
        type: "warning",
        customStyle: { width: "520px" },
        closeOnClickModal: false,
        distinguishCancelAndClose: true
      }
    );
    if (version !== preparationVersion || !props.modelValue) return;
    const operation = await dbPlatform.submit(
      props.cluster.id,
      plan.value.id,
      key.value,
      true
    );
    if (version === preparationVersion && props.modelValue)
      emit("submitted", operation.id);
  } catch (e) {
    if (version === preparationVersion && e !== "cancel" && e !== "close")
      error.value = (e as Error).message;
  } finally {
    if (version === preparationVersion) busy.value = false;
  }
}
watch(
  () => props.modelValue,
  async value => {
    resetPlan();
    const version = preparationVersion;
    busy.value = false;
    if (!value) return;
    error.value = "";
    Object.keys(parameters).forEach(k => delete parameters[k]);
    Object.assign(parameters, {
      vip: {
        cidr: props.cluster.vip?.cidr || "",
        interface: props.cluster.vip?.interface || "",
        routerId: props.cluster.vip?.routerId || undefined,
        memberIds:
          props.cluster.vip?.memberIds ||
          props.cluster.members.filter(m => m.kind === "proxy").map(m => m.id)
      },
      semiSyncEnabled: true,
      semiSyncTimeoutMs: 1000,
      management: "operate",
      method: props.cluster.engine === "redis" ? "rdb" : "logical",
      port:
        props.cluster.engine === "mysql"
          ? 14307
          : props.cluster.engine === "redis"
            ? 18379
            : 15434
    });
    Object.assign(parameters, props.initialParameters || {});
    changes.value = [{ name: "", value: "" }];
    slotsInput.value = "";
    Object.assign(newMember, {
      hostId: undefined,
      port: parameters.port,
      kind: "data",
      role: "replica"
    });
    proxyMembers.forEach(p => {
      p.hostId = undefined;
      p.port = props.cluster.engine === "mysql" ? 23306 : 25432;
    });
    try {
      if (props.action === "reconcile")
        pendingOperations.value =
          (
            await dbRequest("operations", "get", undefined, {
              clusterId: props.cluster.id,
              size: 100
            })
          ).list?.filter((op: any) => op.status === "unknown") || [];
      if (["add-node", "configure-endpoint"].includes(props.action)) {
        const h = await cmdb.getCmdbHostList({ page: 1, pageSize: 100 });
        hosts.value = h.data?.data?.list || h.data?.data?.records || [];
      }
      if (props.action === "backup")
        repositories.value =
          (await dbRequest("repositories", "get", undefined, { size: 100 }))
            .list || [];
      if (
        ["restore", "verify-backup", "expire-backup"].includes(props.action)
      ) {
        const [b, h] = await Promise.all([
          dbRequest("backups", "get", undefined, {
            clusterId: props.cluster.id,
            size: 100
          }),
          cmdb.getCmdbHostList({ page: 1, pageSize: 100 })
        ]);
        backups.value = b.list || [];
        if (
          props.initialParameters?.backupId &&
          !backups.value.some(b => b.id === props.initialParameters.backupId)
        ) {
          const selected = await dbRequest(
            `backups/${props.initialParameters.backupId}`
          );
          if (selected.clusterId !== props.cluster.id)
            throw new Error("备份不属于当前集群");
          backups.value.unshift(selected);
        }
        hosts.value = h.data?.data?.list || h.data?.data?.records || [];
      }
      if (props.action === "configure") {
        const c = await dbRequest(
          `clusters/${props.cluster.id}/configurations`
        );
        definitions.value = c.definitions || [];
      }
      if (version !== preparationVersion || !props.modelValue) return;
      if (version === preparationVersion && !requiresParameters.value)
        await preflight();
    } catch (e) {
      if (version === preparationVersion) error.value = (e as Error).message;
    }
  },
  { immediate: true }
);
</script>
<style scoped>
.field-help {
  color: var(--el-text-color-secondary);
  line-height: 1.8;
  font-size: 13px;
  margin: 8px 0 18px;
}
.el-alert {
  margin-bottom: 18px;
}
.el-select {
  min-width: 250px;
}
.operation-target {
  margin-top: 22px;
}
.el-form-item {
  margin-top: 16px;
}
</style>
