<template>
  <el-card shadow="never" class="database-deployments">
    <template #header>
      <div class="deployment-toolbar">
        <div>
          <strong>{{ $tr("数据库部署与高可用") }}</strong>
          <p>
            {{
              $tr(
                "Ansible 安装 · MySQL GTID 复制 · Orchestrator · 备份与执行记录"
              )
            }}
          </p>
        </div>
        <div>
          <el-button @click="load">{{ $tr("刷新") }}</el-button
          ><el-button
            v-if="!deploymentId"
            type="primary"
            @click="showCreate()"
            >{{ $tr("安装数据库") }}</el-button
          >
        </div>
      </div>
    </template>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      show-icon
    />
    <el-empty
      v-if="!rows.length && !loading"
      :description="$tr('尚未创建数据库部署，可从此处或工具列表安装')"
    />
    <el-table
      v-else
      v-loading="loading"
      :data="rows"
      @row-click="selectDeployment"
    >
      <el-table-column prop="name" :label="$tr('部署名称')" min-width="160" />
      <el-table-column prop="engine" :label="$tr('数据库')" width="130" />
      <el-table-column :label="$tr('模式')" width="140"
        ><template #default="{ row }">{{
          row.mode === "replication" ? $tr("GTID 主从") : $tr("单机")
        }}</template></el-table-column
      >
      <el-table-column prop="port" :label="$tr('端口')" width="90" />
      <el-table-column :label="$tr('状态')" min-width="140"
        ><template #default="{ row }"
          ><el-tag
            :type="
              row.status === 'ready'
                ? 'success'
                : row.status.includes('failed')
                  ? 'danger'
                  : 'warning'
            "
            >{{ statusLabel(row.status) }}</el-tag
          ></template
        ></el-table-column
      >
      <el-table-column :label="$tr('操作')" min-width="240"
        ><template #default="{ row }">
          <el-button link type="primary" @click.stop="selectDeployment(row)">{{
            $tr("管理 / 日志")
          }}</el-button>
          <el-button
            link
            :disabled="row.status !== 'ready'"
            @click.stop="runAction(row, 'backup')"
            >{{ $tr("备份") }}</el-button
          >
          <el-button
            v-if="row.status.includes('failed')"
            link
            type="warning"
            @click.stop="runAction(row, 'retry-install')"
            >{{ $tr("重试安装") }}</el-button
          >
        </template></el-table-column
      >
    </el-table>

    <section v-if="selected" class="deployment-detail">
      <div class="deployment-toolbar">
        <h3>{{ selected.name }}</h3>
        <el-button :loading="detailLoading" @click="refreshDetail">{{
          $tr("刷新状态")
        }}</el-button>
      </div>
      <el-tabs v-model="tab" @tab-change="refreshDetail">
        <el-tab-pane :label="$tr('执行历史与备份')" name="runs">
          <el-table :data="runs" @row-click="row => (selectedRun = row)">
            <el-table-column prop="id" label="ID" width="70" /><el-table-column
              prop="action"
              :label="$tr('动作')"
              min-width="150"
            />
            <el-table-column :label="$tr('结果')" width="130"
              ><template #default="{ row }">{{
                statusLabel(row.status)
              }}</template></el-table-column
            >
            <el-table-column
              prop="artifact"
              :label="$tr('目标主机备份目录')"
              min-width="220"
            /><el-table-column
              prop="createdAt"
              :label="$tr('开始时间')"
              min-width="200"
            />
          </el-table>
          <pre v-if="selectedRun" class="deployment-log">{{
            selectedRun.log || $tr("等待执行日志…")
          }}</pre>
        </el-tab-pane>
        <el-tab-pane
          v-if="hasOrchestrator"
          :label="$tr('MySQL 复制拓扑')"
          name="topology"
        >
          <el-alert
            :title="
              $tr(
                '切换会改变复制关系。业务连接地址需由代理或 VIP 配合调整；请在切换后核验应用连接。'
              )
            "
            type="info"
            :closable="false"
          />
          <div class="topology-grid">
            <article
              v-for="node in topology"
              :key="node.Key.Hostname + ':' + node.Key.Port"
              class="topology-node"
            >
              <el-tag :type="node.IsLastCheckValid ? 'success' : 'danger'">{{
                node.IsLastCheckValid ? $tr("在线") : $tr("检查失败")
              }}</el-tag>
              <h4>{{ node.Key.Hostname }}:{{ node.Key.Port }}</h4>
              <p>
                {{ $tr("上游") }}:
                {{
                  node.MasterKey?.Hostname && node.MasterKey.Port
                    ? `${node.MasterKey.Hostname}:${node.MasterKey.Port}`
                    : $tr("无（主库）")
                }}
              </p>
              <p>
                {{ node.ReadOnly ? $tr("只读") : $tr("可写") }} · GTID
                {{ node.SupportsOracleGTID ? "ON" : "OFF" }}
              </p>
              <p>
                IO
                {{
                  node.ReplicationIOThreadState === 1 || node.Slave_IO_Running
                    ? "✓"
                    : "—"
                }}
                / SQL
                {{
                  node.ReplicationSQLThreadState === 1 || node.Slave_SQL_Running
                    ? "✓"
                    : "—"
                }}
                · {{ $tr("延迟") }}
                {{
                  node.SecondsBehindMaster?.Valid
                    ? node.SecondsBehindMaster.Int64 + " s"
                    : "—"
                }}
              </p>
            </article>
          </div>
          <el-form label-width="110px" class="ha-form">
            <el-form-item :label="$tr('操作')"
              ><el-select v-model="ha.action"
                ><el-option
                  v-for="item in haActions"
                  :key="item.value"
                  :value="item.value"
                  :label="$tr(item.label)" /></el-select
            ></el-form-item>
            <el-form-item :label="$tr('源实例')"
              ><el-select v-model="ha.hostId"
                ><el-option
                  v-for="id in nodeIDs"
                  :key="id"
                  :value="id"
                  :label="hostLabel(id)" /></el-select
            ></el-form-item>
            <el-form-item v-if="needsCandidate" :label="$tr('候选实例')"
              ><el-select v-model="ha.candidateId"
                ><el-option
                  v-for="id in nodeIDs.filter(id => id !== ha.hostId)"
                  :key="id"
                  :value="id"
                  :label="hostLabel(id)" /></el-select
            ></el-form-item>
            <el-form-item
              v-if="ha.action.startsWith('begin-')"
              :label="$tr('原因')"
              ><el-input v-model="ha.reason" maxlength="100"
            /></el-form-item>
            <el-form-item
              ><el-button
                type="primary"
                :loading="submitting"
                @click="submitHA"
                >{{ $tr("执行操作") }}</el-button
              ></el-form-item
            >
          </el-form>
          <el-button @click="showOrchestratorAccess">{{
            $tr("Orchestrator 原生界面与访问凭据")
          }}</el-button>
        </el-tab-pane>
        <el-tab-pane
          v-if="hasOrchestrator"
          :label="$tr('故障分析')"
          name="analysis"
        >
          <pre class="deployment-log">{{ detailJSON }}</pre>
        </el-tab-pane>
        <el-tab-pane
          v-if="hasOrchestrator"
          :label="$tr('恢复审计')"
          name="recoveries"
        >
          <pre class="deployment-log">{{ detailJSON }}</pre>
        </el-tab-pane>
        <el-tab-pane
          v-if="hasOrchestrator"
          :label="$tr('管理端健康')"
          name="health"
        >
          <pre class="deployment-log">{{ detailJSON }}</pre>
        </el-tab-pane>
        <el-tab-pane
          v-if="hasOrchestrator"
          :label="$tr('自动恢复状态')"
          name="recovery-state"
        >
          <pre class="deployment-log">{{ detailJSON }}</pre>
        </el-tab-pane>
      </el-tabs>
    </section>

    <el-dialog
      v-model="accessVisible"
      :title="$tr('Orchestrator 访问')"
      width="min(560px, calc(100vw - 32px))"
      @closed="access = null"
    >
      <template v-if="access"
        ><p>{{ $tr("用户名") }}: {{ access.username }}</p>
        <el-input
          :model-value="access.password"
          readonly
          type="password"
          show-password
        />
        <p v-for="endpoint in access.endpoints" :key="endpoint">
          <a
            :href="endpoint + '/web/clusters'"
            target="_blank"
            rel="noopener noreferrer"
            >{{ endpoint }}</a
          >
        </p></template
      >
    </el-dialog>
    <el-dialog
      v-model="createVisible"
      :title="$tr('通过 Ansible 安装数据库')"
      width="min(760px, calc(100vw - 32px))"
      destroy-on-close
    >
      <el-alert
        :title="
          $tr(
            '目标主机需要 Python 3、root 或免密 sudo，并在 CMDB 配置已确认指纹的 SSH 连接。安装成功后自动登记数据源。'
          )
        "
        type="info"
        :closable="false"
      />
      <el-form label-width="130px" class="deployment-form">
        <el-form-item :label="$tr('部署名称')"
          ><el-input
            v-model="form.name"
            maxlength="64"
            placeholder="database-test"
        /></el-form-item>
        <el-form-item :label="$tr('数据库')"
          ><el-select v-model="form.engine" @change="changeEngine"
            ><el-option label="MySQL 8.0.24" value="mysql" /><el-option
              label="PostgreSQL（系统软件源）"
              value="postgresql" /><el-option
              label="Redis（系统软件源）"
              value="redis" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('部署模式')"
          ><el-radio-group v-model="form.mode"
            ><el-radio-button value="standalone">{{
              $tr("单机")
            }}</el-radio-button
            ><el-radio-button
              value="replication"
              :disabled="form.engine !== 'mysql'"
              >{{ $tr("GTID 主从") }}</el-radio-button
            ></el-radio-group
          ></el-form-item
        >
        <el-form-item :label="$tr('业务分组')"
          ><el-tree-select
            v-model="form.groupId"
            :data="groups"
            :props="{ label: 'name', value: 'id', children: 'children' }"
            check-strictly
        /></el-form-item>
        <el-form-item :label="$tr('数据库主机')"
          ><el-select
            v-model="form.hostIds"
            multiple
            filterable
            :placeholder="$tr('按顺序选择，第一台作为初始主库')"
            ><el-option
              v-for="host in hosts"
              :key="host.id"
              :value="host.id"
              :label="hostLabel(host.id)" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('数据库端口')"
          ><el-input-number v-model="form.port" :min="1024" :max="65535"
        /></el-form-item>
        <el-form-item :label="$tr('管理密码')"
          ><el-input
            v-model="form.password"
            type="password"
            show-password
            autocomplete="new-password"
            :placeholder="$tr('至少 12 位，使用字母、数字或 _!@%+=.,:-')"
        /></el-form-item>
        <template v-if="form.engine === 'mysql' && form.mode === 'replication'">
          <el-form-item label="Orchestrator"
            ><el-select
              v-model="form.managerIds"
              multiple
              filterable
              :placeholder="$tr('可选 1 台管理节点或 3 台 Raft 节点')"
              ><el-option
                v-for="host in hosts"
                :key="host.id"
                :value="host.id"
                :label="hostLabel(host.id)" /></el-select
          ></el-form-item>
          <el-form-item :label="$tr('自动故障恢复')"
            ><el-switch
              v-model="form.autoRecovery"
              :disabled="!form.managerIds.length"
          /></el-form-item>
        </template>
      </el-form>
      <template #footer
        ><el-button @click="createVisible = false">{{ $tr("取消") }}</el-button
        ><el-button type="primary" :loading="submitting" @click="create">{{
          $tr("开始安装")
        }}</el-button></template
      >
    </el-dialog>
  </el-card>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { databaseAutomation } from "@/api/databaseAutomation";
import cmdbAPI from "@/api/cmdb";
import { tr } from "@/locales";
const accessVisible = ref(false),
  access = ref(null);
async function showOrchestratorAccess() {
  try {
    access.value = unpack(await databaseAutomation.access(selected.value.id));
    accessVisible.value = true;
  } catch (e) {
    ElMessage.error(e.message);
  }
}
const props = defineProps({ deploymentId: { type: Number, default: 0 } });
const rows = ref([]),
  hosts = ref([]),
  groups = ref([]),
  runs = ref([]),
  topology = ref([]);
const selected = ref(null),
  selectedRun = ref(null),
  tab = ref("runs"),
  detailJSON = ref("");
const loading = ref(false),
  detailLoading = ref(false),
  submitting = ref(false),
  error = ref(""),
  createVisible = ref(false);
const form = reactive({
  name: "",
  engine: "mysql",
  mode: "standalone",
  groupId: null,
  hostIds: [],
  managerIds: [],
  port: 3306,
  password: "",
  autoRecovery: false
});
const ha = reactive({
  action: "graceful-master-takeover",
  hostId: null,
  candidateId: null,
  reason: ""
});
const ids = value => {
  try {
    return JSON.parse(value || "[]") || [];
  } catch {
    return [];
  }
};
const nodeIDs = computed(() => ids(selected.value?.nodes));
const managerIDs = computed(() => ids(selected.value?.managers));
const hasOrchestrator = computed(
  () => selected.value?.engine === "mysql" && managerIDs.value.length > 0
);
const needsCandidate = computed(() =>
  ["graceful-master-takeover", "relocate", "recover"].includes(ha.action)
);
const haActions = [
  ["graceful-master-takeover", "计划主从切换"],
  ["recover", "故障恢复（指定候选）"],
  ["relocate", "调整复制上游"],
  ["discover", "发现实例"],
  ["refresh", "刷新实例"],
  ["start-replica", "启动复制"],
  ["stop-replica", "停止复制"],
  ["restart-replica", "重启复制"],
  ["set-read-only", "设置只读"],
  ["set-writeable", "设置可写"],
  ["begin-maintenance", "进入维护"],
  ["end-maintenance", "结束维护"],
  ["begin-downtime", "停机标记（1 小时）"],
  ["end-downtime", "结束停机标记"],
  ["enable-global-recoveries", "启用自动恢复"],
  ["disable-global-recoveries", "停用自动恢复"]
].map(([value, label]) => ({ value, label }));
function unpack(res) {
  if (res.data?.code !== 200)
    throw new Error(res.data?.message || tr("操作失败"));
  return res.data.data;
}
function statusLabel(s) {
  return tr(
    {
      pending: "等待安装",
      installing: "安装中",
      ready: "已就绪",
      failed: "失败",
      registration_failed: "资产登记失败",
      queued: "排队中",
      running: "运行中",
      success: "成功",
      unknown: "结果待核验"
    }[s] || s
  );
}
function hostLabel(id) {
  const h = hosts.value.find(h => h.id === id);
  return h ? `${h.hostName} · ${h.sshIp}` : `主机 #${id}`;
}

async function load() {
  loading.value = true;
  try {
    rows.value = (unpack(await databaseAutomation.list()).list || []).filter(
      r => !props.deploymentId || r.id === props.deploymentId
    );
    if (selected.value)
      selected.value = rows.value.find(r => r.id === selected.value.id) || null;
    if (props.deploymentId && !selected.value && rows.value.length)
      await selectDeployment(rows.value[0]);
  } catch (e) {
    error.value = e.message;
  } finally {
    loading.value = false;
  }
}
async function selectDeployment(row) {
  selected.value = row;
  selectedRun.value = null;
  ha.hostId = ids(row.nodes)[0];
  ha.candidateId = ids(row.nodes)[1];
  tab.value = "runs";
  await refreshDetail();
}
async function refreshDetail() {
  if (!selected.value || detailLoading.value) return;
  detailLoading.value = true;
  error.value = "";
  const id = selected.value.id,
    view = tab.value;
  try {
    if (view === "runs") {
      const data = unpack(await databaseAutomation.runs(id));
      if (id !== selected.value?.id) return;
      runs.value = data.list || [];
      selectedRun.value =
        runs.value.find(r => r.id === selectedRun.value?.id) || runs.value[0];
    } else {
      const data = unpack(await databaseAutomation.view(id, view));
      if (id !== selected.value?.id || view !== tab.value) return;
      if (view === "topology")
        topology.value = Array.isArray(data) ? data : data?.Details || [];
      else detailJSON.value = JSON.stringify(data, null, 2);
    }
  } catch (e) {
    if (id === selected.value?.id && view === tab.value)
      error.value = e.message;
  } finally {
    detailLoading.value = false;
    // A tab/selection change may arrive while the previous request is loading.
    // Fetch that latest view after the in-flight request finishes.
    if (selected.value && (id !== selected.value.id || view !== tab.value))
      void refreshDetail();
  }
}
function changeEngine() {
  form.port = { mysql: 3306, postgresql: 5432, redis: 6379 }[form.engine];
  form.mode = "standalone";
  form.managerIds = [];
  form.autoRecovery = false;
}
function showCreate(engine = "mysql") {
  form.engine = engine;
  changeEngine();
  createVisible.value = true;
}
async function create() {
  submitting.value = true;
  try {
    const data = unpack(
      await databaseAutomation.create({
        ...form,
        managerIds: form.mode === "replication" ? form.managerIds : [],
        autoRecovery:
          form.mode === "replication" &&
          form.managerIds.length > 0 &&
          form.autoRecovery
      })
    );
    form.password = "";
    createVisible.value = false;
    await load();
    await selectDeployment(data.deployment);
    ElMessage.success(tr("Ansible 安装任务已提交"));
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    submitting.value = false;
  }
}
async function confirmation(row) {
  const { value } = await ElMessageBox.prompt(
    `${tr("请输入完整部署名称确认操作")}：${row.name}`,
    tr("确认数据库操作"),
    { inputValidator: value => value === row.name || tr("名称不匹配") }
  );
  return value;
}
async function runAction(row, action) {
  try {
    const confirmName = await confirmation(row);
    unpack(await databaseAutomation.action(row.id, { action, confirmName }));
    await selectDeployment(row);
  } catch (e) {
    if (e instanceof Error) ElMessage.error(e.message);
  }
}
async function submitHA() {
  try {
    const confirmName = await confirmation(selected.value);
    submitting.value = true;
    unpack(
      await databaseAutomation.ha(selected.value.id, {
        ...ha,
        candidateId: needsCandidate.value ? ha.candidateId : 0,
        confirmName
      })
    );
    ElMessage.success(tr("操作完成，请核验拓扑"));
    await refreshDetail();
  } catch (e) {
    if (e instanceof Error) ElMessage.error(e.message);
  } finally {
    submitting.value = false;
  }
}
let timer;
let disposed = false;
async function loadHosts() {
  const result = [];
  for (let page = 1; !disposed; page++) {
    const data = unpack(await cmdbAPI.getCmdbHostList({ page, pageSize: 100 }));
    const batch = data.list || data.records || [];
    result.push(...batch);
    if (batch.length < 100 || result.length >= Number(data.total || Infinity))
      break;
  }
  return result;
}
onMounted(async () => {
  try {
    const [h, g] = await Promise.all([loadHosts(), cmdbAPI.getAllCmdbGroups()]);
    hosts.value = h;
    groups.value = unpack(g) || [];
  } catch (e) {
    error.value = e.message;
  }
  await load();
  if (disposed) return;
  timer = setInterval(async () => {
    if (document.hidden) return;
    await load();
    if (tab.value === "runs") await refreshDetail();
  }, 5000);
});
onUnmounted(() => {
  disposed = true;
  clearInterval(timer);
});
defineExpose({ showCreate });
</script>
<style scoped>
.database-deployments {
  margin-bottom: 20px;
}
.deployment-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.deployment-toolbar p {
  margin: 6px 0 0;
  color: var(--text-secondary);
  font-size: 13px;
}
.deployment-detail {
  margin-top: 20px;
}
.deployment-log {
  max-height: 460px;
  overflow: auto;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  padding: 16px;
  background: var(--bg-page);
  color: var(--text-primary);
  border: 1px solid var(--border);
  border-radius: 8px;
}
.deployment-form,
.ha-form {
  margin-top: 20px;
}
.deployment-form :deep(.el-select),
.ha-form :deep(.el-select) {
  width: 100%;
}
.topology-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
  margin: 20px 0;
}
.topology-node {
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--bg-card);
}
.topology-node p {
  font-size: 13px;
  color: var(--text-secondary);
}
a {
  color: var(--primary);
}
</style>
