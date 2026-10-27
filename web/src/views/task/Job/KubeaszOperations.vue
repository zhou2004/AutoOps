<template>
  <section v-loading="loading" class="kubeasz-panel">
    <el-alert
      v-if="error"
      :title="$tr(error)"
      type="error"
      :closable="false"
      show-icon
    />
    <template v-if="catalog">
      <div class="system-heading">
        <div>
          <strong>kubeasz {{ catalog.version }}</strong>
          <span class="muted">Kubernetes {{ catalog.kubernetes }}</span>
          <el-tag size="small">{{ $tr("系统 Ansible 模板") }}</el-tag>
        </div>
        <el-button :loading="loading" @click="refresh">{{ $tr("刷新环境") }}</el-button>
      </div>
      <el-collapse class="readiness">
        <el-collapse-item name="readiness">
          <template #title>
            <span>{{ $tr("执行环境检查") }}</span>
            <el-tag
              :type="
                readyCount === catalog.checks.length ? 'success' : 'warning'
              "
              class="check-count"
            >
              {{ readyCount }} / {{ catalog.checks.length }} {{ $tr("已就绪") }}
            </el-tag>
          </template>
          <div
            v-for="check in catalog.checks"
            :key="check.name"
            class="check-row"
          >
            <el-tag :type="check.ready ? 'success' : 'warning'" size="small">{{
              check.ready ? $tr("就绪") : "待准备"
            }}</el-tag>
            <span>{{ check.detail }}</span>
          </div>
          <p class="muted">
            {{ $tr("固定版本源码已集成到 API 镜像。二进制、镜像包、SSH 密钥和 known_hosts 需部署人员准备；详细步骤见仓库的 kubeasz 系统运维文档。") }}
          </p>
          <p class="muted">
            {{ $tr("系统目录：") }}{{ catalog.root }} {{ $tr("· 来源提交：") }}{{ catalog.commit }}
          </p>
        </el-collapse-item>
      </el-collapse>

      <el-tabs v-model="tab">
        <el-tab-pane :label='$tr("系统操作")' name="operate">
          <el-form label-width="120px" class="operation-form" @submit.prevent>
            <el-form-item :label='$tr("操作模板")'>
              <el-select v-model="form.action" class="full-width">
                <el-option
                  v-for="item in catalog.templates"
                  :key="item.id"
                  :label="$tr(item.name)"
                  :value="item.id"
                />
              </el-select>
              <p class="field-note">{{ $tr(selectedTemplate?.description) }}</p>
            </el-form-item>
            <el-form-item :label='$tr("目标集群")'>
              <el-input
                v-if="isNew"
                v-model="form.cluster"
                maxlength="63"
                :placeholder='$tr("例如 prod-east，以小写字母开头")'
              />
              <el-select
                v-else
                v-model="form.cluster"
                filterable
                :placeholder='$tr("选择 kubeasz 集群配置")'
                class="full-width"
              >
                <el-option
                  v-for="cluster in catalog.clusters"
                  :key="cluster.name"
                  :label="$tr(cluster.name)"
                  :value="cluster.name"
                />
              </el-select>
              <p v-if="!isNew && !catalog.clusters.length" class="field-note">
                {{ $tr("请先创建集群配置，或由部署人员导入完整的 kubeasz clusters 目录。") }}
              </p>
            </el-form-item>
            <div v-if="selectedCluster && !isNew" class="cluster-summary">
              <span
                >{{ $tr("控制面：") }}{{ selectedCluster.masters?.join(", ") || "无" }}</span
              >
              <span
                >{{ $tr("工作节点：") }}{{
                  selectedCluster.workers?.join(", ") || "无"
                }}</span
              >
              <span>etcd：{{ selectedCluster.etcd?.join(", ") || "无" }}</span>
            </div>
            <template v-if="isNew">
              <el-form-item :label='$tr("控制面 IPv4")'
                ><el-input
                  v-model="form.masters"
                  type="textarea"
                  :rows="2"
                  :placeholder='$tr("每行一个 IP，或用逗号分隔")'
              /></el-form-item>
              <el-form-item label="etcd IPv4"
                ><el-input
                  v-model="form.etcd"
                  type="textarea"
                  :rows="2"
                  :placeholder='$tr("1、3、5 或 7 个成员，可与控制面共用主机")'
              /></el-form-item>
              <el-form-item :label='$tr("工作节点 IPv4")'
                ><el-input
                  v-model="form.workers"
                  type="textarea"
                  :rows="2"
                  :placeholder='$tr("可选，每行一个 IP")'
              /></el-form-item>
              <div class="form-grid">
                <el-form-item :label='$tr("SSH 用户")'
                  ><el-input v-model="form.sshUser" placeholder="root"
                /></el-form-item>
                <el-form-item :label='$tr("SSH 端口")'
                  ><el-input-number
                    v-model="form.sshPort"
                    :min="1"
                    :max="65535"
                /></el-form-item>
                <el-form-item label="Pod CIDR"
                  ><el-input v-model="form.podCIDR"
                /></el-form-item>
                <el-form-item label="Service CIDR"
                  ><el-input v-model="form.serviceCIDR"
                /></el-form-item>
                <el-form-item :label='$tr("网络插件")'
                  ><el-select v-model="form.network"
                    ><el-option
                      v-for="item in ['calico', 'flannel', 'cilium']"
                      :key="item"
                      :label="$tr(item)"
                      :value="item" /></el-select
                ></el-form-item>
              </div>
              <p class="field-note">
                {{ $tr("节点需支持 systemd / Python 3。SSH 私钥由执行器挂载；非 root 用户需免密 sudo。新建配置不会连接目标主机。") }}
              </p>
            </template>
            <el-form-item v-if="selectedTemplate?.node" :label='$tr("目标节点 IPv4")'>
              <el-input
                v-model="form.node"
                :placeholder='$tr("例如 192.168.10.21；每次仅操作一个节点")'
              />
            </el-form-item>
            <el-form-item v-if="selectedTemplate?.step" :label='$tr("安装步骤")'>
              <el-select v-model="form.step" class="full-width"
                ><el-option
                  v-for="(label, key) in catalog.steps"
                  :key="key"
                  :label="`${key} · ${label}`"
                  :value="key"
              /></el-select>
            </el-form-item>
            <el-form-item v-if="selectedTemplate?.addon" :label='$tr("服务插件")'>
              <el-select v-model="form.addon" class="full-width"
                ><el-option
                  v-for="(label, key) in catalog.addons"
                  :key="key"
                  :label="$tr(label)"
                  :value="key"
              /></el-select>
            </el-form-item>
            <el-form-item v-if="form.action === 'restore'" :label='$tr("恢复快照")'>
              <el-select
                v-model="form.snapshot"
                class="full-width"
                :placeholder='$tr("选择集群 backup 目录中的快照")'
              >
                <el-option
                  v-for="snapshot in selectedCluster?.snapshots || []"
                  :key="snapshot"
                  :label="$tr(snapshot)"
                  :value="snapshot"
                />
              </el-select>
            </el-form-item>
            <el-form-item
              ><el-button type="primary" :loading="previewing" @click="preview"
                >{{ $tr("预览目标与命令") }}</el-button
              ></el-form-item
            >
          </el-form>

          <el-card v-if="plan" shadow="never" class="plan-card">
            <template #header
              ><strong
                >{{ $tr("操作预览 ·") }} {{ actionName(plan.request.action) }} ·
                {{ plan.request.cluster }}</strong
              ></template
            >
            <pre class="command-box">{{ plan.command }}</pre>
            <div
              v-if="Object.keys(plan.targets).length"
              class="cluster-summary"
            >
              <span v-for="(hosts, role) in plan.targets" :key="role"
                >{{ role }}：{{ hosts.join(", ") }}</span
              >
            </div>
            <el-alert
              v-for="blocker in plan.blockers"
              :key="blocker"
              :title="$tr(blocker)"
              type="error"
              :closable="false"
              show-icon
            />
            <el-alert
              v-for="warning in plan.warnings"
              :key="warning"
              :title="$tr(warning)"
              type="warning"
              :closable="false"
              show-icon
            />
            <el-collapse v-if="plan.inventory"
              ><el-collapse-item :title='$tr("查看将生成的 inventory")'>
                <pre class="command-box">{{ plan.inventory }}</pre>
              </el-collapse-item></el-collapse
            >
            <p>
              {{ $tr("请输入确认词：") }}<code>{{ plan.confirmation }}</code>
            </p>
            <el-input
              v-model="confirmation"
              autocomplete="off"
              :placeholder='$tr("完整输入上方操作、集群及目标")'
              :disabled="!plan.ready || submitting"
            />
            <el-button
              class="execute-button"
              :type="
                selectedTemplate?.risk === 'destructive' ? 'danger' : 'primary'
              "
              :disabled="!plan.ready || confirmation !== plan.confirmation"
              :loading="submitting"
              @click="execute"
              >{{ $tr("确认执行") }}</el-button
            >
          </el-card>
        </el-tab-pane>

        <el-tab-pane :label='$tr("操作历史与日志")' name="history">
          <div class="history-controls">
            <el-select
              v-model="historyCluster"
              clearable
              :placeholder='$tr("全部集群")'
              @change="changeHistoryFilter"
              ><el-option
                v-for="cluster in catalog.clusters"
                :key="cluster.name"
                :label="$tr(cluster.name)"
                :value="cluster.name"
            /></el-select>
            <el-button :loading="historyLoading" @click="fetchHistory"
              >{{ $tr("刷新历史") }}</el-button
            >
          </div>
          <el-table v-loading="historyLoading" :data="history" border stripe>
            <el-table-column prop="id" label="ID" width="75" />
            <el-table-column prop="cluster" :label='$tr("集群")' min-width="120" />
            <el-table-column :label='$tr("操作")' min-width="145"
              ><template #default="{ row }">{{
                actionName(row.action)
              }}</template></el-table-column
            >
            <el-table-column :label='$tr("状态")' width="115"
              ><template #default="{ row }"
                ><el-tag :type="statusType(row.status)">{{
                  statusName(row.status)
                }}</el-tag></template
              ></el-table-column
            >
            <el-table-column
              prop="operator_name"
              :label='$tr("执行人")'
              min-width="95"
            />
            <el-table-column :label='$tr("创建时间（上海）")' min-width="170"
              ><template #default="{ row }">{{
                formatTime(row.created_at)
              }}</template></el-table-column
            >
            <el-table-column :label='$tr("详情")' width="200"
              ><template #default="{ row }"
                ><el-button link type="primary" @click="showLog(row.id)"
                  >{{ $tr("日志") }}</el-button
                ><el-button
                  v-if="isRunning(row.status)"
                  link
                  type="danger"
                  @click="cancelOperation(row)"
                  >{{ $tr("取消") }}</el-button
                ><el-button
                  v-if="row.status === 'interrupted' && !row.reconciled_at"
                  link
                  type="warning"
                  @click="openReconcile(row)"
                  >{{ $tr("核查确认") }}</el-button
                ></template
              ></el-table-column
            >
          </el-table>
          <el-pagination
            v-model:current-page="historyPage"
            :page-size="20"
            :total="historyTotal"
            layout="total, prev, pager, next"
            class="history-pagination"
            @current-change="fetchHistory"
          />
        </el-tab-pane>
      </el-tabs>
    </template>
    <el-dialog
      v-model="reconcileVisible"
      :title='$tr("中断操作核查确认")'
      width="620px"
      append-to-body
    >
      <template v-if="reconcileTarget">
        <el-alert
          :title='$tr("先在执行器检查遗留 ansible / ssh 进程已结束，再核查实际节点、inventory、etcd 和业务状态。完成核查后填写记录，才能解除系统操作限制。")'
          type="warning"
          :closable="false"
        />
        <p>
          #{{ reconcileTarget.id }} · {{ reconcileTarget.cluster }} ·
          {{ actionName(reconcileTarget.action) }}
        </p>
        <el-input
          v-model="reconcileNote"
          type="textarea"
          :rows="4"
          maxlength="1000"
          :placeholder='$tr("记录遗留进程、节点状态、已采取的处理；不要填写凭据")'
        />
        <p>
          {{ $tr("请输入：") }}<code>{{ reconcilePhrase }}</code>
        </p>
        <el-input v-model="reconcileConfirmation" autocomplete="off" />
      </template>
      <template #footer
        ><el-button @click="reconcileVisible = false">{{ $tr("返回") }}</el-button
        ><el-button
          type="warning"
          :disabled="
            reconcileConfirmation !== reconcilePhrase ||
            reconcileNote.trim().length < 5
          "
          :loading="reconciling"
          @click="reconcile"
          >{{ $tr("保存核查记录") }}</el-button
        ></template
      >
    </el-dialog>
    <el-dialog
      v-model="logVisible"
      :title='$tr("kubeasz 操作详情")'
      width="85%"
      append-to-body
    >
      <template v-if="detail">
        <div class="system-heading">
          <strong
            >#{{ detail.operation.id }} · {{ detail.operation.cluster }} ·
            {{ actionName(detail.operation.action) }}</strong
          ><el-tag :type="statusType(detail.operation.status)">{{
            statusName(detail.operation.status)
          }}</el-tag>
        </div>
        <pre class="command-box">{{ detail.operation.command }}</pre>
        <el-alert
          v-if="detail.operation.error"
          :title="$tr(detail.operation.error)"
          type="warning"
          :closable="false"
        />
        <el-alert
          v-if="detail.operation.reconciled_at"
          :title="`已核查：${detail.operation.reconciled_by} · ${detail.operation.reconcile_note}`"
          type="info"
          :closable="false"
        />
        <p class="muted">
          {{ $tr("日志每 3 秒更新。含敏感参数的任务输出已隐藏，结束状态以进程退出结果为准。") }}
        </p>
        <pre class="log-box">{{ detail.log || "等待执行器写入日志…" }}</pre>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { tr } from "@/locales";

import { computed, onMounted, onUnmounted, reactive, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  cancelKubeaszOperation,
  getKubeaszCatalog,
  getKubeaszHistory,
  getKubeaszOperation,
  previewKubeasz,
  reconcileKubeaszOperation,
  startKubeasz
} from "@/api/kubeasz";
import type {
  KubeaszCatalog,
  KubeaszOperation,
  KubeaszPlan,
  KubeaszRequest
} from "@/api/kubeasz";

const catalog = ref<KubeaszCatalog | null>(null);
const loading = ref(false);
const error = ref("");
const tab = ref("operate");
const previewing = ref(false);
const submitting = ref(false);
const plan = ref<KubeaszPlan | null>(null);
const confirmation = ref("");
const form = reactive({
  action: "new",
  cluster: "",
  node: "",
  step: "all",
  addon: "metrics-server",
  snapshot: "",
  masters: "",
  workers: "",
  etcd: "",
  sshUser: "root",
  sshPort: 22,
  podCIDR: "10.244.0.0/16",
  serviceCIDR: "10.68.0.0/16",
  network: "calico"
});
const selectedTemplate = computed(() =>
  catalog.value?.templates.find(item => item.id === form.action)
);
const selectedCluster = computed(() =>
  catalog.value?.clusters.find(item => item.name === form.cluster)
);
const isNew = computed(() => form.action === "new");
const readyCount = computed(
  () => catalog.value?.checks.filter(item => item.ready).length || 0
);
const addresses = (value: string) =>
  value
    .split(/[\s,，]+/)
    .map(item => item.trim())
    .filter(Boolean);
const payload = computed<KubeaszRequest>(() => {
  const request: KubeaszRequest = {
    action: form.action,
    cluster: form.cluster.trim()
  };
  if (selectedTemplate.value?.node) request.node = form.node.trim();
  if (selectedTemplate.value?.step) request.step = form.step;
  if (selectedTemplate.value?.addon) request.addon = form.addon;
  if (form.action === "restore") request.snapshot = form.snapshot;
  if (isNew.value)
    request.config = {
      masters: addresses(form.masters),
      workers: addresses(form.workers),
      etcd: addresses(form.etcd),
      ssh_user: form.sshUser.trim(),
      ssh_port: form.sshPort,
      pod_cidr: form.podCIDR.trim(),
      service_cidr: form.serviceCIDR.trim(),
      network: form.network
    };
  return request;
});
watch(
  payload,
  () => {
    plan.value = null;
    confirmation.value = "";
  },
  { deep: true }
);
watch(
  () => form.action,
  () => {
    form.snapshot = "";
  }
);
watch(
  () => form.cluster,
  () => {
    form.snapshot = "";
  }
);
const message = (e: unknown) => (e instanceof Error ? e.message : "操作失败");
const actionName = (action: string) =>
  catalog.value?.templates.find(item => item.id === action)?.name || action;
const isRunning = (status: string) =>
  status === "queued" || status === "running";
const statusName = (status: string) =>
  ({
    queued: "等待中",
    running: "执行中",
    succeeded: "命令完成",
    failed: "失败",
    canceled: "已取消",
    interrupted: "中断待核查"
  })[status] || status;
const statusType = (
  status: string
): "success" | "danger" | "warning" | "info" | "primary" =>
  status === "succeeded"
    ? "success"
    : status === "failed"
      ? "danger"
      : isRunning(status)
        ? "primary"
        : "warning";
const formatTime = (value: string) =>
  new Date(value).toLocaleString("zh-CN", {
    timeZone: "Asia/Shanghai",
    hour12: false
  });
async function refresh() {
  loading.value = true;
  error.value = "";
  try {
    catalog.value = await getKubeaszCatalog();
    await fetchHistory();
  } catch (e) {
    error.value = message(e);
  } finally {
    loading.value = false;
  }
}
async function preview() {
  previewing.value = true;
  plan.value = null;
  confirmation.value = "";
  const request = structuredClone(payload.value);
  const signature = JSON.stringify(request);
  try {
    const result = await previewKubeasz(request);
    if (JSON.stringify(payload.value) === signature) plan.value = result;
  } catch (e) {
    ElMessage.error(message(e));
  } finally {
    previewing.value = false;
  }
}
async function execute() {
  if (
    !plan.value ||
    !plan.value.ready ||
    confirmation.value !== plan.value.confirmation
  )
    return;
  submitting.value = true;
  try {
    const op = await startKubeasz({
      ...plan.value.request,
      plan_hash: plan.value.hash,
      confirmation: confirmation.value
    });
    confirmation.value = "";
    plan.value = null;
    ElMessage.success(`操作 #${op.id} 已提交`);
    await fetchHistory();
    await showLog(op.id);
  } catch (e) {
    ElMessage.error(message(e));
    plan.value = null;
    confirmation.value = "";
  } finally {
    submitting.value = false;
  }
}
const history = ref<KubeaszOperation[]>([]);
const historyCluster = ref("");
const historyPage = ref(1);
const historyTotal = ref(0);
const historyLoading = ref(false);
let historyRequest = 0;
async function fetchHistory() {
  const request = ++historyRequest;
  historyLoading.value = true;
  try {
    const result = await getKubeaszHistory({
      page: historyPage.value,
      size: 20,
      cluster: historyCluster.value || undefined
    });
    if (!disposed && request === historyRequest) {
      history.value = result.list;
      historyTotal.value = result.total;
    }
  } catch (e) {
    if (!disposed && request === historyRequest) error.value = message(e);
  } finally {
    if (!disposed && request === historyRequest) historyLoading.value = false;
  }
}
function changeHistoryFilter() {
  historyPage.value = 1;
  void fetchHistory();
}
const logVisible = ref(false);
const detail = ref<{ operation: KubeaszOperation; log: string } | null>(null);
let detailRequest = 0;
let detailID = 0;
async function showLog(id: number) {
  if (disposed) return;
  const request = ++detailRequest;
  detailID = id;
  detail.value = null;
  logVisible.value = true;
  try {
    const result = await getKubeaszOperation(id);
    if (!disposed && request === detailRequest && detailID === id)
      detail.value = result;
  } catch (e) {
    if (!disposed && request === detailRequest) ElMessage.error(message(e));
  }
}
watch(logVisible, visible => {
  if (!visible) {
    detailRequest++;
    detailID = 0;
  }
});
async function cancelOperation(operation: KubeaszOperation) {
  try {
    await ElMessageBox.confirm(
      `终止 ${operation.cluster} 的 ${actionName(operation.action)}（#${operation.id}）？已完成的远端修改不会回滚，取消后需核查节点状态。`,
      tr("取消系统操作"),
      {
        type: "warning",
        confirmButtonText: "终止执行",
        cancelButtonText: "继续执行"
      }
    );
    await cancelKubeaszOperation(operation.id);
    ElMessage.success(tr("已请求终止执行"));
    await fetchHistory();
  } catch (e) {
    if (e !== "cancel" && e !== "close") ElMessage.error(message(e));
  }
}
const reconcileVisible = ref(false);
const reconcileTarget = ref<KubeaszOperation | null>(null);
const reconcileNote = ref("");
const reconcileConfirmation = ref("");
const reconciling = ref(false);
const reconcilePhrase = computed(() =>
  reconcileTarget.value
    ? `reconcile ${reconcileTarget.value.id} ${reconcileTarget.value.cluster}`
    : ""
);
function openReconcile(operation: KubeaszOperation) {
  reconcileTarget.value = operation;
  reconcileNote.value = "";
  reconcileConfirmation.value = "";
  reconcileVisible.value = true;
}
async function reconcile() {
  if (
    !reconcileTarget.value ||
    reconcileConfirmation.value !== reconcilePhrase.value
  )
    return;
  reconciling.value = true;
  try {
    await reconcileKubeaszOperation(reconcileTarget.value.id, {
      confirmation: reconcileConfirmation.value,
      note: reconcileNote.value.trim()
    });
    ElMessage.success(tr("核查记录已保存"));
    reconcileVisible.value = false;
    await fetchHistory();
  } catch (e) {
    ElMessage.error(message(e));
  } finally {
    reconciling.value = false;
  }
}
let pollTimer: ReturnType<typeof setTimeout> | undefined;
let disposed = false;
async function poll() {
  if (disposed) return;
  try {
    if (
      logVisible.value &&
      detail.value &&
      isRunning(detail.value.operation.status)
    ) {
      const id = detail.value.operation.id;
      const request = ++detailRequest;
      const result = await getKubeaszOperation(id);
      if (
        !disposed &&
        request === detailRequest &&
        detailID === id &&
        logVisible.value
      ) {
        detail.value = result;
        if (!isRunning(result.operation.status)) {
          const refreshedCatalog = await getKubeaszCatalog();
          if (!disposed) {
            catalog.value = refreshedCatalog;
            await fetchHistory();
          }
        }
      }
    }
    if (history.value.some(item => isRunning(item.status)))
      await fetchHistory();
  } catch (e) {
    error.value = message(e);
  } finally {
    if (!disposed) pollTimer = setTimeout(poll, 3000);
  }
}
onMounted(async () => {
  await refresh();
  if (!disposed) pollTimer = setTimeout(poll, 3000);
});
onUnmounted(() => {
  disposed = true;
  if (pollTimer) clearTimeout(pollTimer);
});
</script>

<style scoped>
.kubeasz-panel {
  color: var(--text-primary, var(--el-text-color-primary));
}
.system-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}
.system-heading > div {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}
.muted,
.field-note {
  color: var(--text-secondary, var(--el-text-color-secondary));
  font-size: 13px;
  line-height: 1.7;
}
.field-note {
  width: 100%;
  margin: 6px 0 0;
}
.readiness {
  margin-bottom: 18px;
}
.check-count {
  margin-left: 12px;
}
.check-row {
  display: flex;
  gap: 12px;
  align-items: baseline;
  margin: 10px 0;
}
.check-row .el-tag {
  flex-shrink: 0;
}
.full-width {
  width: 100%;
}
.operation-form {
  max-width: 980px;
  margin: 16px auto;
}
.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 14px;
}
.cluster-summary {
  display: flex;
  flex-direction: column;
  gap: 6px;
  background: var(--bg-page, var(--el-fill-color-light));
  border-radius: 6px;
  padding: 14px;
  margin: 10px 0 20px;
  overflow-wrap: anywhere;
}
.plan-card .el-alert {
  margin: 12px 0;
}
.command-box,
.log-box {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  background: var(--bg-page, var(--el-fill-color-light));
  color: var(--text-primary, var(--el-text-color-primary));
  border: 1px solid var(--border, var(--el-border-color));
  border-radius: 6px;
  padding: 16px;
  line-height: 1.6;
  font-family: monospace;
}
.log-box {
  max-height: 52vh;
  overflow: auto;
}
.execute-button {
  margin-top: 14px;
}
.history-controls {
  display: flex;
  gap: 12px;
  margin: 12px 0 20px;
}
.history-controls .el-select {
  max-width: 240px;
}
.history-pagination {
  margin-top: 18px;
  justify-content: flex-end;
}
@media (max-width: 760px) {
  .form-grid {
    grid-template-columns: 1fr;
  }
  .system-heading {
    align-items: flex-start;
  }
}
</style>
