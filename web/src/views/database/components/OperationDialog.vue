<template>
  <el-dialog
    :model-value="modelValue"
    :title="$tr('确认执行 · {value0}', { value0: $tr(label) })"
    width="min(780px, 94vw)"
    append-to-body
    @update:model-value="$emit('update:modelValue', $event)"
    ><el-alert
      v-if="error"
      :title="$tr(error)"
      type="error"
      :closable="false"
    />
    <p class="operation-target">
      {{
        $tr("将在集群 {value0} 执行 {value1}。", {
          value0: cluster.name,
          value1: $tr(label)
        })
      }}
    </p>
    <el-alert
      v-if="action === 'backup'"
      type="info"
      :closable="false"
      show-icon
      :title="
        $tr(
          '选择备份名称、方式与保存位置 → 预检并确认 → 在恢复演练验证，成功后进入备份集。'
        )
      "
    />
    <el-alert
      v-if="['restore', 'verify-backup'].includes(action)"
      type="info"
      :closable="false"
      show-icon
      :title="
        $tr(
          '恢复到新目录和独立端口，启动并校验后自动停止隔离实例；文件与报告保留，不自动替换业务数据库。'
        )
      "
    />
    <template v-if="action === 'configure-vip'">
      <el-form label-width="140px" :disabled="!!plan">
        <el-form-item :label="$tr('虚拟 IP / 前缀')"
          ><el-input
            v-model="parameters.vip.cidr"
            :placeholder="$tr('已分配的 IPv4/CIDR')"
        /></el-form-item>
        <el-form-item :label="$tr('绑定网卡')"
          ><el-input
            v-model="parameters.vip.interface"
            :placeholder="$tr('例如 ens33')"
        /></el-form-item>
        <el-form-item label="VRID"
          ><el-input-number
            v-model="parameters.vip.routerId"
            :min="1"
            :max="255"
        /></el-form-item>
        <el-form-item :label="$tr('承载代理')"
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
      :title="
        $tr(
          '短暂停止当前承载节点的独立 VIP 服务，验证地址漂移与连续 SQL 连接，随后恢复原服务。已配置独立 60 秒自动恢复保护。'
        )
      "
    />
    <el-form label-width="140px" :disabled="!!plan"
      ><el-form-item v-if="action === 'backup'" :label="$tr('备份名称')"
        ><el-input
          v-model="parameters.backupName"
          maxlength="100"
          show-word-limit
          clearable
          :placeholder="$tr('例如：发布前备份；留空自动生成')"
      /></el-form-item>
      <el-form-item v-if="action === 'reconcile'" :label="$tr('结果待核验任务')"
        ><el-select v-model="parameters.operationId"
          ><el-option
            v-for="op in pendingOperations"
            :key="op.id"
            :value="op.id"
            :label="`#${op.id} · ${op.action} · ${op.message || ''}`" /></el-select></el-form-item
      ><el-form-item v-if="action === 'adopt'" :label="$tr('管理级别')"
        ><el-select v-model="parameters.management"
          ><el-option value="observe" :label="$tr('观察模式')" /><el-option
            value="operate"
            :label="$tr('运维托管')" /><el-option
            value="ha"
            :label="$tr('高可用托管（需验收）')" /></el-select></el-form-item
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
        :label="$tr('目标候选')"
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
        :title="
          $tr('Sentinel 根据复制偏移、优先级与多数派选择新主，并自动重配副本。')
        "
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
          :label="$tr('代理 {value0}', { value0: index + 1 })"
        >
          <el-select
            v-model="proxy.hostId"
            filterable
            :placeholder="$tr('CMDB 主机')"
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
          :title="
            $tr(
              '写入口使用填写的端口，只读入口使用端口 +1。需要两个不同主机；工作台支持双代理地址，VIP 可后续绑定。'
            )
          "
        />
      </template>
      <template v-if="action === 'configure-semisync'">
        <el-form-item :label="$tr('启用半同步')"
          ><el-switch v-model="parameters.semiSyncEnabled"
        /></el-form-item>
        <el-form-item :label="$tr('超时（毫秒）')"
          ><el-input-number
            v-model="parameters.semiSyncTimeoutMs"
            :min="100"
            :max="60000"
        /></el-form-item>
        <el-alert
          type="warning"
          :closable="false"
          :title="
            $tr('半同步超时可能退化为异步；这项设置不保证严格零数据丢失。')
          "
        />
      </template>
      <template v-if="action === 'add-node'">
        <el-form-item :label="$tr('新增实例主机')"
          ><el-select v-model="newMember.hostId" filterable
            ><el-option
              v-for="h in hosts"
              :key="h.id"
              :value="h.id"
              :label="`${h.hostname || h.hostName || h.name || h.id} · ${h.ipAddr || h.ip || ''}`" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('独立端口')"
          ><el-input-number v-model="newMember.port" :min="1024" :max="55000"
        /></el-form-item>
        <el-form-item
          v-if="cluster.topology === 'redis-cluster'"
          :label="$tr('节点用途')"
          ><el-select v-model="newMember.role"
            ><el-option
              :label="$tr('已有分片的副本')"
              value="replica" /><el-option
              :label="$tr('新增主分片（初始无槽位）')"
              value="primary" /></el-select
        ></el-form-item>
        <el-alert
          type="info"
          :closable="false"
          :title="
            $tr(
              '系统会生成独立目录与服务名。新增主分片后，需配置副本并通过迁槽任务分配容量。'
            )
          "
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
        :label="$tr('操作节点')"
        ><el-select v-model="parameters.memberId"
          ><el-option
            v-for="m in cluster.members.filter(m => m.kind === 'data')"
            :key="m.id"
            :value="m.id"
            :label="`${m.name} · ${m.address}:${m.port}`" /></el-select></el-form-item
      ><el-form-item v-if="action === 'backup'" :label="$tr('备份方式')"
        ><el-select v-model="parameters.method"
          ><el-option
            v-if="cluster.engine !== 'redis'"
            value="logical"
            :label="$tr('逻辑备份')" /><el-option
            v-if="cluster.engine !== 'redis'"
            value="physical"
            :label="$tr('物理备份（需匹配版本的工具）')" /><template
            v-if="cluster.engine === 'postgresql'"
            ><el-option
              value="incremental"
              :label="$tr('pgBackRest 增量备份')" /><el-option
              value="differential"
              :label="$tr('pgBackRest 差异备份')" /><el-option
              value="wal-archive"
              :label="$tr('导出物理基线与连续 WAL')" /></template
          ><el-option
            v-if="cluster.engine === 'mysql'"
            value="incremental"
            :label="$tr('XtraBackup 增量备份')" /><el-option
            v-if="cluster.engine === 'mysql'"
            value="binlog-archive"
            :label="$tr('导出物理基线与连续 binlog')" /><el-option
            v-if="cluster.engine === 'redis'"
            value="rdb"
            :label="$tr('RDB / 按分片备份')" /></el-select
      ></el-form-item>
      <p v-if="action === 'backup'" class="field-help">
        {{ backupMethods[parameters.method]?.description }}
      </p>
      <el-form-item v-if="action === 'backup'" :label="$tr('保存位置')"
        ><el-select
          v-model="parameters.repositoryId"
          clearable
          :placeholder="$tr('数据库节点本地暂存（无独立副本）')"
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
            ? $tr(
                "仅显示同业务分组、连接检查通过的仓库；上传并校验后才显示独立副本成功。"
              )
            : $tr(
                "不选择仓库时保存在数据库节点。若需要独立副本，请先登记备份仓库并完成连接检查。"
              )
        }}
      </p>
      <el-form-item
        v-if="action === 'purge-binlogs'"
        :label="$tr('保留起始文件')"
        ><el-input
          v-model="parameters.beforeFile"
          :placeholder="$tr('mysql-bin.000123；此前日志须通过归档与依赖检查')"
      /></el-form-item>
      <template
        v-if="['restore', 'verify-backup', 'expire-backup'].includes(action)"
        ><el-form-item :label="$tr('备份集')"
          ><el-select
            v-model="parameters.backupId"
            @change="parameters.targetTime = ''"
            ><el-option
              v-for="b in backups.filter(b => b.status === 'complete')"
              :key="b.id"
              :value="b.id"
              :label="backupDisplayName(b)" /></el-select
        ></el-form-item>
        <p v-if="selectedBackup" class="field-help">
          {{ backupMethodLabel(selectedBackup.method) }} ·
          {{
            selectedBackup.manifest?.remoteComplete
              ? $tr("从独立仓库读取并核验")
              : $tr("从数据库节点本地文件读取")
          }}{{
            selectedBackup.parentId
              ? $tr(" · 依赖备份 #{value0}", {
                  value0: selectedBackup.parentId
                })
              : ""
          }}
        </p>
        <el-alert
          v-if="action === 'expire-backup'"
          type="warning"
          :closable="false"
          show-icon
          :title="
            $tr(
              '清理将删除此备份的本地文件和仓库副本，未过期备份同样可清理。其他备份包和业务数据保持不变；清理后无法使用此备份恢复。'
            )
          "
        />
        <p v-else class="field-help">
          {{
            action === "verify-backup"
              ? $tr(
                  "恢复验证用于定期演练，实际还原并校验成功后，备份才进入备份集。"
                )
              : $tr(
                  "隔离恢复用于误删找回、升级前回退评估或灾难恢复准备。先在隔离环境取回并核对数据，再安排回灌或业务切换。"
                )
          }}
          {{ $tr("两种操作均在校验后停止隔离实例，保留恢复文件与报告。") }}
        </p>
        <el-form-item
          v-if="action !== 'expire-backup'"
          :label="$tr('隔离目标主机')"
          ><el-select v-model="parameters.hostId" filterable
            ><el-option
              v-for="h in hosts"
              :key="h.id"
              :value="h.id"
              :label="`${h.hostname || h.hostName || h.name || h.id}`" /></el-select></el-form-item
        ><el-form-item
          v-if="action !== 'expire-backup'"
          :label="$tr('独立恢复端口')"
          ><el-input-number v-model="parameters.port" :min="1024" :max="55000"
        /></el-form-item>
        <p v-if="action !== 'expire-backup'" class="field-help">
          {{
            $tr(
              "目标主机需具备部署权限和可用资源。端口必须空闲；Redis Cluster 按分片从起始端口依次分配。"
            )
          }}
        </p>
        <el-form-item
          v-if="action !== 'expire-backup' && restoreWindow"
          :label="
            selectedBackup?.method === 'binlog-archive'
              ? $tr('目标时间（必填）')
              : $tr('目标时间（可选）')
          "
        >
          <el-date-picker
            v-model="parameters.targetTime"
            type="datetime"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
            :placeholder="$tr('选择归档覆盖范围内的时间')"
          />
        </el-form-item>
        <p v-if="action !== 'expire-backup'" class="field-help">
          {{
            restoreWindow
              ? $tr(
                  "可恢复范围：{value0} 至 {value1}（当前浏览器时区）。MySQL 时间精确到秒，不包含目标秒及之后的事件。",
                  {
                    value0: new Date(restoreWindow.start).toLocaleString(),
                    value1: new Date(restoreWindow.end).toLocaleString()
                  }
                )
              : $tr("此备份没有完整的归档时间窗口，只能恢复备份内容。")
          }}
        </p>
      </template>
      <el-form-item v-if="action === 'reshard'" :label="$tr('迁移槽位')"
        ><el-input
          v-model="slotsInput"
          :placeholder="
            $tr('例如 100-199 或 100,101；每批最多 1024')
          " /></el-form-item
      ><template v-if="action === 'configure'"
        ><el-form-item
          v-for="(change, i) in changes"
          :key="i"
          :label="$tr('参数与期望值')"
          ><el-select v-model="change.name" :placeholder="$tr('参数')"
            ><el-option
              v-for="d in definitions"
              :key="d.name"
              :value="d.name"
              :label="`${d.name} (${d.mode})`" /></el-select
          ><el-input
            v-model="change.value"
            style="width: 160px; margin-left: 10px"
            :placeholder="$tr('期望值')"
          /><el-button link type="danger" @click="changes.splice(i, 1)">{{
            $tr("移除")
          }}</el-button></el-form-item
        ><el-button @click="changes.push({ name: '', value: '' })">{{
          $tr("增加参数")
        }}</el-button></template
      ></el-form
    ><template v-if="plan"
      ><el-divider
        >{{ $tr("预检结果 · 拓扑版本") }} {{ plan.revision }}</el-divider
      ><el-table :data="plan.checks" max-height="340"
        ><el-table-column :label="$tr('检查项')" min-width="200"
          ><template #default="{ row }">{{
            preflightCheckLabel(row.name)
          }}</template></el-table-column
        ><el-table-column :label="$tr('结果')" width="80"
          ><template #default="{ row }"
            ><el-tag :type="row.passed ? 'success' : 'danger'">{{
              row.passed ? $tr("通过") : $tr("阻断")
            }}</el-tag></template
          ></el-table-column
        ><el-table-column :label="$tr('原因')" min-width="240"
          ><template #default="{ row }">{{
            row.passed ? $tr("已满足") : $tr(row.message)
          }}</template></el-table-column
        ></el-table
      ></template
    ><template #footer
      ><el-button @click="$emit('update:modelValue', false)">{{
        $tr("取消")
      }}</el-button
      ><el-button v-if="plan" :disabled="busy" @click="resetPlan">{{
        $tr("调整参数")
      }}</el-button
      ><el-button
        v-if="!plan"
        type="primary"
        :loading="busy"
        @click="preflight"
        >{{ $tr("预检") }}</el-button
      ><el-button
        v-else
        type="primary"
        :loading="busy"
        :disabled="!plan.executable || !key"
        @click="submit"
        >{{ $tr("确认执行") }}</el-button
      ></template
    ></el-dialog
  >
</template>
<script setup lang="ts">
import { tr } from "@/locales";

import { computed, ref, reactive, watch } from "vue";
import { ElMessageBox } from "element-plus";
import {
  dbPlatform,
  dbRequest,
  operationPlanKey,
  preflightCheckLabel,
  type Cluster,
  type Plan
} from "@/api/databasePlatform";
import {
  backupDisplayName,
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
        throw new Error(tr("binlog 归档恢复必须选择归档范围内的目标时间"));
      if (parameters.targetTime) {
        const target = Date.parse(parameters.targetTime);
        if (
          !restoreWindow.value ||
          !Number.isFinite(target) ||
          target < restoreWindow.value.start ||
          target > restoreWindow.value.end
        )
          throw new Error(tr("目标时间不在已核验的连续归档范围内"));
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
          throw new Error(tr("槽位范围无效或超过 1024"));
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
      ? tr("目标节点：{value0}（{value1}:{value2}）。", {
          value0: target.name,
          value1: target.address,
          value2: target.port
        })
      : "";
    const impact = ["switchover", "recover", "verify-vip", "restart"].includes(
      props.action
    )
      ? tr("执行期间连接可能短暂中断，客户端需要重连。")
      : props.action === "expire-backup"
        ? tr(
            "将永久删除{value0}（{value1}）的本地文件和仓库副本，清理后无法再从此备份恢复。",
            {
              value0: backupMethodLabel(selectedBackup.value?.method || ""),
              value1: selectedBackup.value?.createdAt
                ? new Date(selectedBackup.value.createdAt).toLocaleString()
                : tr("所选备份")
            }
          )
        : "";
    await ElMessageBox.confirm(
      tr("确认在集群「{value0}」执行「{value1}」？{value2}{value3}{value4}", {
        value0: props.cluster.name,
        value1: tr(props.label),
        value2: targetText,
        value3:
          props.action === "backup"
            ? tr("备份名称：{value0}。", {
                value0: parameters.backupName?.trim() || tr("自动生成")
              })
            : "",
        value4: impact
      }),
      tr("执行确认"),
      {
        get confirmButtonText() {
          return tr("确认执行");
        },
        get cancelButtonText() {
          return tr("取消");
        },
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
            throw new Error(tr("备份不属于当前集群"));
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
