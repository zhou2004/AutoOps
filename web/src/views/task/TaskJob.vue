<template>
  <div class="task-page">
    <header class="task-page-header">
      <div>
        <span class="task-eyebrow">JOB ORCHESTRATION</span>
        <h1>{{ $tr("任务作业") }}</h1>
        <p>{{ $tr("像 Crontab 一样配置命令或多语言脚本，并保留每次执行的版本与日志。") }}</p>
      </div>
      <div class="task-header-count"><strong>{{ total }}</strong><span>{{ $tr("匹配任务") }}</span></div>
    </header>

    <el-card shadow="never" class="task-surface">
      <div class="task-filters">
        <el-input v-model="query.name" clearable :placeholder='$tr("按任务名称搜索")' :prefix-icon="Search" @keyup.enter="loadTasks" @clear="loadTasks" />
        <el-select v-model="query.status" clearable :placeholder='$tr("全部状态")' @change="handleFilterChange">
          <el-option v-for="item in statusOptions" :key="item.value" :label="$tr(item.label)" :value="item.value" />
        </el-select>
        <el-button type="primary" :icon="Search" @click="handleFilterChange">{{ $tr("查询") }}</el-button>
        <el-button :icon="Refresh" @click="resetFilters">{{ $tr("重置") }}</el-button>
      </div>

      <div class="task-toolbar">
        <div class="toolbar-note">{{ $tr("定时任务可按 Cron 自动运行，也可随时手动触发一次。") }}</div>
        <el-button type="primary" :icon="Plus" v-authority="['task:job:add']" @click="openCreate">{{ $tr("新建任务") }}</el-button>
      </div>

      <el-table :data="tasks" row-key="id" class="task-table" v-loading="loading">
        <el-table-column :label='$tr("任务")' min-width="230" fixed="left">
          <template #default="{ row }">
            <div class="task-identity">
              <div class="task-avatar"><el-icon><Operation /></el-icon></div>
              <div class="task-cell">
                <button class="task-link" type="button" @click="openHistory(row)">{{ row.name }}</button>
                <span>{{ row.executionMode === 2 ? `${languageName(row.commandType)} · 直接命令` : `${row.templates.length} 个模板` }}</span>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label='$tr("触发方式")' width="125">
          <template #default="{ row }"><el-tag :type="row.type === 2 ? 'warning' : 'info'" effect="plain">{{ row.type === 2 ? 'Cron 定时' : '手动触发' }}</el-tag></template>
        </el-table-column>
        <el-table-column :label='$tr("状态")' width="120">
          <template #default="{ row }"><span class="status-line"><i :class="['status-dot', statusClass(taskDisplayStatus(row))]" />{{ taskStatusName(row) }}</span></template>
        </el-table-column>
        <el-table-column :label='$tr("Cron / 下次执行")' min-width="210">
          <template #default="{ row }">
            <div v-if="row.type === 2" class="task-cell task-mono"><strong>{{ row.cronExpr || '-' }}</strong><span>{{ row.nextRunTime ? formatTime(row.nextRunTime) : '尚未启用调度' }}</span></div>
            <span v-else class="task-muted">{{ $tr("按需执行") }}</span>
          </template>
        </el-table-column>
        <el-table-column :label='$tr("执行统计")' width="140">
          <template #default="{ row }"><div class="task-cell"><strong>{{ row.executeCount }} {{ $tr("次") }}</strong><span>{{ $tr("保留") }} {{ row.maxHistoryKeep }} {{ $tr("条日志") }}</span></div></template>
        </el-table-column>
        <el-table-column :label='$tr("定时开关")' width="110" align="center">
          <template #default="{ row }">
            <el-switch v-if="row.type === 2" :model-value="row.status === 2" :loading="row.scheduleLoading" @change="value => toggleSchedule(row, value)" />
            <span v-else class="task-muted">—</span>
          </template>
        </el-table-column>
        <el-table-column :label='$tr("备注")' min-width="160" show-overflow-tooltip prop="remark" />
        <el-table-column :label='$tr("操作")' width="285" fixed="right">
          <template #default="{ row }">
            <div class="row-actions">
              <el-button v-if="row.executionStatus === 2" type="warning" plain size="small" :icon="VideoPause" :loading="row.controlLoading" @click="pauseExecution(row)">{{ $tr("暂停") }}</el-button>
              <el-button v-else-if="row.executionStatus === 5" type="primary" plain size="small" :icon="VideoPlay" :loading="row.controlLoading" @click="resumeExecution(row)">{{ $tr("恢复") }}</el-button>
              <el-button v-else type="primary" plain size="small" :icon="VideoPlay" :loading="row.runLoading" @click="runNow(row)">{{ $tr("立即执行") }}</el-button>
              <el-button v-if="row.executionStatus === 2 || row.executionStatus === 5" link type="danger" size="small" :icon="CircleClose" :loading="row.controlLoading" @click="stopExecution(row)">{{ $tr("停止") }}</el-button>
              <el-button link type="primary" size="small" :icon="Document" @click="openHistory(row)">{{ $tr("历史") }}</el-button>
              <el-dropdown trigger="click">
                <el-button link size="small" :icon="MoreFilled" :aria-label='$tr("更多操作")' />
                <template #dropdown><el-dropdown-menu><el-dropdown-item :disabled="row.status === 2 || row.executionStatus > 0" @click="openEdit(row)">{{ $tr("编辑") }}</el-dropdown-item><el-dropdown-item divided :disabled="row.status === 2 || row.executionStatus > 0" @click="removeTask(row)">{{ $tr("删除") }}</el-dropdown-item></el-dropdown-menu></template>
              </el-dropdown>
            </div>
          </template>
        </el-table-column>
        <template #empty><el-empty :description='$tr("暂无匹配任务，可新建命令或脚本任务")' /></template>
      </el-table>

      <div class="task-pagination">
        <el-pagination v-model:current-page="query.page" v-model:page-size="query.pageSize" :page-sizes="[10, 20, 50, 100]" :total="total" layout="total, sizes, prev, pager, next, jumper" @size-change="handlePageSizeChange" @current-change="loadTasks" />
      </div>
    </el-card>

    <el-dialog v-model="formVisible" :title="form.id ? '编辑任务' : $tr('新建任务')" width="min(820px, 94vw)" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-position="top">
        <div class="form-grid">
          <el-form-item :label='$tr("任务名称")' prop="name"><el-input v-model="form.name" maxlength="255" :placeholder='$tr("例如：启动订单服务")' /></el-form-item>
          <el-form-item :label='$tr("触发方式")' prop="type"><el-radio-group v-model="form.type"><el-radio-button :value="1">{{ $tr("手动任务") }}</el-radio-button><el-radio-button :value="2">{{ $tr("定时任务") }}</el-radio-button></el-radio-group></el-form-item>
          <el-form-item v-if="form.type === 2" :label='$tr("Cron 表达式")' prop="cronExpr" class="form-span-2">
            <el-input v-model="form.cronExpr" :placeholder='$tr("分 时 日 月 周，例如 0 2 * * *")' @change="previewCron"><template #append>{{ nextExecution || '标准 5 位 Cron' }}</template></el-input>
          </el-form-item>
          <el-form-item :label='$tr("执行内容来源")' prop="executionMode" class="form-span-2"><el-radio-group v-model="form.executionMode"><el-radio :value="2">{{ $tr("直接命令 / 脚本") }}</el-radio><el-radio :value="1">{{ $tr("复用任务模板") }}</el-radio></el-radio-group></el-form-item>
          <template v-if="form.executionMode === 2">
            <el-form-item :label='$tr("语言 / 运行时")' prop="commandType" class="form-span-2">
              <el-select v-model="form.commandType" style="width: 100%"><el-option v-for="item in languageOptions" :key="item.value" :label="$tr(item.label)" :value="item.value" /></el-select>
              <div class="field-help">{{ $tr("目标服务器需已安装对应运行时；“直接命令”使用 bash -lc 执行。") }}</div>
            </el-form-item>
            <el-form-item :label='$tr("命令或脚本")' prop="command" class="form-span-2"><CodeEditor v-model="form.command" :language="languageMode(form.commandType)" height="260px" /></el-form-item>
          </template>
          <el-form-item v-else :label='$tr("任务模板")' prop="shell" class="form-span-2">
            <el-select v-model="selectedTemplateIds" multiple filterable collapse-tags collapse-tags-tooltip style="width: 100%" :placeholder='$tr("选择一个或多个模板")'><el-option v-for="item in templates" :key="item.id" :label="`${item.name} · ${languageName(item.type)}`" :value="item.id" /></el-select>
          </el-form-item>
          <el-form-item :label='$tr("执行主机")' prop="hostIDs" class="form-span-2">
            <div class="host-picker">
              <el-button :icon="Plus" @click="hostDialogVisible = true">{{ $tr("选择主机") }}</el-button>
              <div v-if="selectedHosts.length" class="host-tags"><el-tag v-for="host in selectedHosts" :key="host.id" closable @close="removeHost(host.id)">{{ host.name || `主机 #${host.id}` }}<span v-if="host.ip"> · {{ host.ip }}</span></el-tag></div>
              <span v-else class="task-muted">{{ $tr("尚未选择主机") }}</span>
            </div>
          </el-form-item>
          <el-form-item :label='$tr("最大日志记录")' prop="maxHistoryKeep"><el-input-number v-model="form.maxHistoryKeep" :min="1" :max="200" controls-position="right" /></el-form-item>
          <el-form-item :label='$tr("备注")'><el-input v-model="form.remark" maxlength="500" show-word-limit :placeholder='$tr("可选")' /></el-form-item>
        </div>
      </el-form>
      <template #footer><el-button @click="formVisible = false">{{ $tr("取消") }}</el-button><el-button type="primary" :loading="saving" @click="saveTask">{{ $tr("保存") }}</el-button></template>
    </el-dialog>

    <CreateTaskHost v-model="hostDialogVisible" :selected-hosts="selectedHosts" @hosts-selected="mergeHosts" />

    <el-drawer v-model="historyVisible" :title="`${activeTask?.name || ''} · 执行历史`" size="min(920px, 96vw)" destroy-on-close>
      <div class="history-toolbar"><span>{{ $tr("每条记录都包含当次实际执行的命令快照与逐主机日志。") }}</span><el-button :icon="Refresh" @click="loadHistory">{{ $tr("刷新") }}</el-button></div>
      <el-table :data="history" row-key="id" v-loading="historyLoading" class="task-table">
        <el-table-column :label='$tr("执行")' min-width="150"><template #default="{ row }"><div class="task-cell"><strong>#{{ row.id }}</strong><span>{{ row.trigger === 2 ? '定时触发' : '手动触发' }}</span></div></template></el-table-column>
        <el-table-column :label='$tr("状态")' width="110"><template #default="{ row }"><span class="status-line"><i :class="['status-dot', statusClass(row.status)]" />{{ statusName(row.status) }}</span></template></el-table-column>
        <el-table-column :label='$tr("执行人")' prop="operator" width="130" />
        <el-table-column :label='$tr("开始时间")' min-width="180"><template #default="{ row }">{{ formatTime(row.started_at) }}</template></el-table-column>
        <el-table-column :label='$tr("耗时")' width="90"><template #default="{ row }">{{ row.duration || 0 }}s</template></el-table-column>
        <el-table-column :label='$tr("操作")' width="150" fixed="right"><template #default="{ row }"><el-button link type="primary" @click="showHistoryDetail(row)">{{ $tr("日志") }}</el-button><el-button link type="warning" :disabled="row.status === 2 || row.status === 5" @click="rollback(row)">{{ $tr("回滚") }}</el-button></template></el-table-column>
        <template #empty><el-empty :description='$tr("该任务还没有执行记录")' /></template>
      </el-table>
      <div class="task-pagination"><el-pagination v-model:current-page="historyQuery.page" v-model:page-size="historyQuery.pageSize" :total="historyTotal" :page-sizes="[10, 20, 50]" layout="total, sizes, prev, pager, next" @size-change="loadHistory" @current-change="loadHistory" /></div>
    </el-drawer>

    <el-dialog v-model="detailVisible" :title='$tr("执行快照与日志")' width="min(960px, 96vw)" top="5vh" destroy-on-close>
      <div v-loading="detailLoading" class="history-detail">
        <template v-if="historyDetail">
          <div class="snapshot-summary"><el-tag effect="plain">{{ historyDetail.execution_mode === 2 ? $tr('直接命令') : '模板任务' }}</el-tag><span>{{ historyDetail.cron_expr || '手动触发' }}</span><span v-if="historyDetail.rolled_back_at">{{ $tr("已于") }} {{ formatTime(historyDetail.rolled_back_at) }} {{ $tr("回滚") }}</span></div>
          <el-collapse accordion>
            <el-collapse-item v-for="log in historyDetail.logs || []" :key="log.id" :name="log.id">
              <template #title><div class="log-title"><span class="status-line"><i :class="['status-dot', statusClass(log.status)]" />{{ statusName(log.status) }}</span><strong>{{ log.host_name || `主机 #${log.host_id}` }}</strong><span>{{ log.template_name }} · {{ languageName(log.command_type) }}</span></div></template>
              <div class="log-section"><label>{{ $tr("执行快照") }}</label><pre>{{ log.command_snapshot }}</pre></div>
              <div class="log-section"><label>{{ $tr("执行日志") }}</label><pre>{{ log.log || '暂无日志输出' }}</pre></div>
            </el-collapse-item>
          </el-collapse>
        </template>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { tr } from "@/locales";

import { nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CircleClose, Document, MoreFilled, Operation, Plus, Refresh, Search, VideoPause, VideoPlay } from '@element-plus/icons-vue'
import CodeEditor from '@/components/CodeEditor.vue'
import CreateTaskHost from './Job/CreateTaskHost.vue'
import { CreateTask, DeleteTask, GetAllTemplates, GetNextExecutionTime, GetTaskExecutionHistory, GetTaskExecutionHistoryDetail, ListTasks, PauseScheduledTask, PauseTaskExecution, ResumeTaskExecution, RollbackTaskExecution, RunTaskNow, ScheduleTask, StopTaskExecution, UpdateTask } from '@/api/task'

const languageOptions = [
  { value: 10, label: '直接命令 (bash -lc)', mode: 'bash' }, { value: 1, label: 'Shell', mode: 'bash' },
  { value: 2, label: 'Python', mode: 'python' }, { value: 3, label: 'Ansible Playbook', mode: 'yaml' },
  { value: 4, label: 'JavaScript (Node.js)', mode: 'javascript' }, { value: 5, label: 'PowerShell', mode: 'powershell' },
  { value: 6, label: 'Go', mode: 'go' }, { value: 7, label: 'Ruby', mode: 'ruby' },
  { value: 8, label: 'PHP', mode: 'php' }, { value: 9, label: 'Perl', mode: 'perl' }
]
const statusOptions = [{ value: 1, label: '等待中' }, { value: 2, label: '运行中' }, { value: 3, label: '成功' }, { value: 4, label: '异常' }, { value: 5, label: '已暂停' }, { value: 6, label: '已停止' }]

const tasks = ref([]), templates = ref([]), loading = ref(false), total = ref(0)
const query = reactive({ page: 1, pageSize: 10, name: '', status: null })
const formVisible = ref(false), formRef = ref(), saving = ref(false), hostDialogVisible = ref(false)
const selectedTemplateIds = ref([]), selectedHosts = ref([]), nextExecution = ref('')
const form = reactive(emptyForm())
const activeTask = ref(null), historyVisible = ref(false), historyLoading = ref(false), history = ref([]), historyTotal = ref(0)
const historyQuery = reactive({ page: 1, pageSize: 10 })
const detailVisible = ref(false), detailLoading = ref(false), historyDetail = ref(null)

function emptyForm() { return { id: null, name: '', type: 1, cronExpr: '', executionMode: 2, command: '', commandType: 10, hostIDs: '', maxHistoryKeep: 20, remark: '' } }
const rules = {
  name: [{ required: true, get message() { return tr('请输入任务名称'); }, trigger: 'blur' }],
  type: [{ required: true, get message() { return tr('请选择触发方式'); }, trigger: 'change' }],
  cronExpr: [{ validator: (_r, value, cb) => form.type === 2 && !value?.trim() ? cb(new Error(tr('请输入 Cron 表达式'))) : cb(), trigger: 'blur' }],
  command: [{ validator: (_r, value, cb) => form.executionMode === 2 && !value?.trim() ? cb(new Error(tr('请输入命令或脚本'))) : cb(), trigger: 'blur' }],
  shell: [{ validator: (_r, _v, cb) => form.executionMode === 1 && selectedTemplateIds.value.length === 0 ? cb(new Error(tr('请选择任务模板'))) : cb(), trigger: 'change' }],
  hostIDs: [{ validator: (_r, _v, cb) => selectedHosts.value.length === 0 ? cb(new Error(tr('请选择执行主机'))) : cb(), trigger: 'change' }],
  maxHistoryKeep: [{ required: true, get message() { return tr('请设置最大日志记录数'); }, trigger: 'change' }]
}

async function loadTasks(silent = false) {
  const background = silent === true
  if (!background) loading.value = true
  try {
    const response = await ListTasks({ page: query.page, pageSize: query.pageSize, name: query.name || undefined, status: query.status || undefined })
    const payload = response?.data?.data || {}
    tasks.value = (payload.list || []).map(item => ({ id: item.id, name: item.name, type: item.type, status: item.status, executionStatus: item.execution_status || 0, currentHistoryID: item.current_history_id || 0, shell: item.shell || '', executionMode: item.execution_mode || 1, command: item.command || '', commandType: item.command_type || 10, hostIDs: item.host_ids || '', cronExpr: item.cron_expr || '', maxHistoryKeep: item.max_history_keep || 20, executeCount: item.execute_count || 0, nextRunTime: item.next_run_time, remark: item.remark || '', templates: item.templates || [], hosts: item.hosts || [], runLoading: false, scheduleLoading: false, controlLoading: false }))
    total.value = payload.total || 0
  } catch (error) {
    if (!background) { tasks.value = []; total.value = 0; ElMessage.error(error.response?.data?.message || '获取任务列表失败') }
  } finally { if (!background) loading.value = false }
}
async function loadTemplates() { try { const response = await GetAllTemplates(); templates.value = response?.data?.data || [] } catch (_) { ElMessage.error(tr('获取任务模板失败')) } }
function handleFilterChange() { query.page = 1; loadTasks() }
function resetFilters() { Object.assign(query, { page: 1, name: '', status: null }); loadTasks() }
function handlePageSizeChange() { query.page = 1; loadTasks() }
function openCreate() { Object.assign(form, emptyForm()); selectedTemplateIds.value = []; selectedHosts.value = []; nextExecution.value = ''; formVisible.value = true }
function openEdit(row) {
  if (row.status === 2 || row.executionStatus > 0) return ElMessage.warning(tr('请先停止当前执行，并暂停定时调度'))
  Object.assign(form, { id: row.id, name: row.name, type: row.type, cronExpr: row.cronExpr, executionMode: row.executionMode, command: row.command, commandType: row.commandType, hostIDs: row.hostIDs, maxHistoryKeep: row.maxHistoryKeep, remark: row.remark })
  selectedTemplateIds.value = row.templates.map(item => item.id)
  selectedHosts.value = row.hosts.map(host => ({ id: host.id, name: host.hostName, ip: host.sshIp || host.privateIp || host.publicIp }))
  previewCron(); formVisible.value = true
}
function mergeHosts(hosts) {
  const map = new Map(selectedHosts.value.map(host => [host.id, host])); for (const host of hosts || []) map.set(host.id, { id: host.id, name: host.name, ip: host.privateIp || host.publicIp })
  selectedHosts.value = [...map.values()]; form.hostIDs = selectedHosts.value.map(host => host.id).join(','); hostDialogVisible.value = false
  nextTick(() => formRef.value?.validateField('hostIDs'))
}
function removeHost(id) { selectedHosts.value = selectedHosts.value.filter(host => host.id !== id); form.hostIDs = selectedHosts.value.map(host => host.id).join(',') }
async function previewCron() {
  nextExecution.value = ''; if (form.type !== 2 || !form.cronExpr.trim()) return
  try { const response = await GetNextExecutionTime({ cron: form.cronExpr.trim() }); nextExecution.value = response?.data?.data?.next_execution_time || '' } catch (_) { nextExecution.value = 'Cron 格式无效' }
}
async function saveTask() {
  try {
    await formRef.value?.validate(); saving.value = true
    const data = { id: form.id, name: form.name.trim(), type: form.type, execution_mode: form.executionMode, command: form.executionMode === 2 ? form.command : '', command_type: form.executionMode === 2 ? form.commandType : 0, shell: form.executionMode === 1 ? selectedTemplateIds.value.join(',') : '', host_ids: selectedHosts.value.map(host => host.id).join(','), cron_expr: form.type === 2 ? form.cronExpr.trim() : '', max_history_keep: form.maxHistoryKeep, remark: form.remark }
    const response = form.id ? await UpdateTask(data) : await CreateTask(data); if (response?.data?.code !== 200) throw new Error(response?.data?.message || tr('保存失败'))
    ElMessage.success(form.id ? '任务已更新' : '任务已创建'); formVisible.value = false; await loadTasks()
  } catch (error) { if (!error?.fields) ElMessage.error(error.response?.data?.message || error.message || '保存任务失败') }
  finally { saving.value = false }
}
async function runNow(row) {
  try { row.runLoading = true; const response = await RunTaskNow(row.id); if (response?.data?.code !== 200) throw new Error(response?.data?.message || '提交失败'); ElMessage.success(`任务已提交，执行记录 #${response.data.data?.history_id}`); await loadTasks() }
  catch (error) { ElMessage.error(error.response?.data?.message || error.message || '手动触发失败') } finally { row.runLoading = false }
}
async function pauseExecution(row) {
  row.controlLoading = true
  try { await PauseTaskExecution(row.id); ElMessage.success(tr('任务执行已暂停')) }
  catch (error) { ElMessage.error(error.response?.data?.message || '暂停执行失败') }
  finally { row.controlLoading = false; await loadTasks() }
}
async function resumeExecution(row) {
  row.controlLoading = true
  try { await ResumeTaskExecution(row.id); ElMessage.success(tr('任务执行已恢复')) }
  catch (error) { ElMessage.error(error.response?.data?.message || '恢复执行失败') }
  finally { row.controlLoading = false; await loadTasks() }
}
async function stopExecution(row) {
  try {
    await ElMessageBox.confirm(`确定停止任务“${row.name}”当前这次执行吗？已产生的日志会保留。`, tr('停止任务执行'), { type: 'warning', confirmButtonText: '停止执行' })
    row.controlLoading = true; await StopTaskExecution(row.id); ElMessage.success(tr('任务执行已停止'))
  } catch (error) { if (error !== 'cancel') ElMessage.error(error.response?.data?.message || '停止执行失败') }
  finally { row.controlLoading = false; await loadTasks() }
}
async function toggleSchedule(row, enabled) {
  row.scheduleLoading = true
  try { if (enabled) await ScheduleTask(row.id); else await PauseScheduledTask(row.id); ElMessage.success(enabled ? '定时调度已启用' : '定时调度已暂停'); await loadTasks() }
  catch (error) { ElMessage.error(error.response?.data?.message || (enabled ? '启用调度失败' : '暂停调度失败')) } finally { row.scheduleLoading = false }
}
async function removeTask(row) {
  try { await ElMessageBox.confirm(`确定删除任务“${row.name}”及其执行历史吗？`, tr('删除确认'), { type: 'warning' }); await DeleteTask({ id: row.id }); ElMessage.success(tr('任务已删除')); loadTasks() }
  catch (error) { if (error !== 'cancel') ElMessage.error(error.response?.data?.message || tr('删除失败')) }
}
function openHistory(row) { activeTask.value = row; historyQuery.page = 1; historyVisible.value = true; loadHistory() }
async function loadHistory() {
  if (!activeTask.value) return; historyLoading.value = true
  try { const response = await GetTaskExecutionHistory(activeTask.value.id, historyQuery); const payload = response?.data?.data || {}; history.value = payload.list || []; historyTotal.value = payload.total || 0 }
  catch (_) { ElMessage.error(tr('获取执行历史失败')) } finally { historyLoading.value = false }
}
async function showHistoryDetail(row) {
  detailVisible.value = true; detailLoading.value = true; historyDetail.value = null
  try { const response = await GetTaskExecutionHistoryDetail(activeTask.value.id, row.id); historyDetail.value = response?.data?.data || null }
  catch (_) { ElMessage.error(tr('获取日志详情失败')) } finally { detailLoading.value = false }
}
async function rollback(row) {
  try { await ElMessageBox.confirm(`恢复执行记录 #${row.id} 当时的命令、模板、主机和 Cron 配置？恢复后不会自动执行。`, tr('回滚任务配置'), { type: 'warning', confirmButtonText: tr('确认回滚') }); await RollbackTaskExecution(activeTask.value.id, row.id); ElMessage.success(tr('历史配置已恢复，未自动执行')); await Promise.all([loadHistory(), loadTasks(), loadTemplates()]) }
  catch (error) { if (error !== 'cancel') ElMessage.error(error.response?.data?.message || '回滚失败') }
}
function statusName(status) { return statusOptions.find(item => item.value === status)?.label || '未知' }
function taskDisplayStatus(row) { return row.executionStatus || row.status }
function taskStatusName(row) {
  if (row.executionStatus) return row.executionStatus === 2 ? '执行中' : statusName(row.executionStatus)
  if (row.type === 2 && row.status === 2) return '调度中'
  if (row.type === 2 && row.status === 5) return '调度暂停'
  return statusName(row.status)
}
function statusClass(status) { return ({ 1: 'info', 2: 'warning', 3: 'success', 4: 'danger', 5: 'info', 6: 'danger' })[status] || 'info' }
function languageName(type) { return languageOptions.find(item => item.value === type)?.label || 'Shell' }
function languageMode(type) { return languageOptions.find(item => item.value === type)?.mode || 'bash' }
function formatTime(value) { return value ? new Date(value).toLocaleString('zh-CN', { hour12: false }).replaceAll('/', '-') : '-' }
let taskRefreshTimer
onMounted(async () => {
  await Promise.all([loadTasks(), loadTemplates()])
  taskRefreshTimer = window.setInterval(() => {
    const isOperating = tasks.value.some(row => row.runLoading || row.controlLoading || row.scheduleLoading)
    if (!loading.value && !isOperating && tasks.value.some(row => row.executionStatus === 2 || row.executionStatus === 5)) loadTasks(true)
  }, 4000)
})
onBeforeUnmount(() => window.clearInterval(taskRefreshTimer))
</script>

<style scoped>
.task-page { min-height: 100%; padding: 24px; background: var(--bg-page); color: var(--text-primary); }
.task-page-header { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; margin-bottom: 18px; }
.task-page-header h1 { margin: 4px 0 6px; font-size: 28px; line-height: 1.2; }.task-page-header p { margin: 0; color: var(--text-secondary); }
.task-eyebrow { color: var(--primary); font-size: 12px; font-weight: 700; letter-spacing: .12em; }.task-header-count { display: flex; min-width: 112px; flex-direction: column; align-items: flex-end; }.task-header-count strong { font-size: 28px; }
.task-header-count span, .task-muted, .task-cell span, .toolbar-note, .field-help, .history-toolbar { color: var(--text-secondary); font-size: 12px; }
.task-surface { border: 1px solid var(--border); border-radius: 16px; background: var(--bg-card); }.task-filters { display: grid; grid-template-columns: minmax(220px, 1fr) 180px auto auto; gap: 10px; padding: 14px; border: 1px solid var(--border); border-radius: 12px; background: var(--bg-card-alt); }
.task-toolbar, .history-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin: 16px 0; }.task-table { width: 100%; }.task-identity { display: flex; align-items: center; gap: 12px; }
.task-avatar { display: grid; width: 38px; height: 38px; flex: 0 0 auto; place-items: center; border-radius: 10px; color: var(--primary); background: color-mix(in srgb, var(--primary) 12%, transparent); }.task-cell { display: flex; min-width: 0; flex-direction: column; gap: 3px; }.task-cell strong { color: var(--text-primary); font-size: 13px; }
.task-link { padding: 0; border: 0; color: var(--primary); background: transparent; font: inherit; font-weight: 650; text-align: left; cursor: pointer; }.task-link:hover { text-decoration: underline; }.task-mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.status-line { display: inline-flex; align-items: center; gap: 7px; white-space: nowrap; }.status-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--text-secondary); }.status-dot.success { background: var(--success, #22c55e); }.status-dot.warning { background: var(--warning, #f59e0b); }.status-dot.danger { background: var(--danger, #ef4444); }.status-dot.info { background: var(--text-secondary); }
.row-actions { display: flex; align-items: center; gap: 2px; }.task-pagination { display: flex; justify-content: flex-end; margin-top: 18px; }.form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 18px; }.form-span-2 { grid-column: 1 / -1; }.field-help { margin-top: 6px; line-height: 1.5; }
.host-picker { display: flex; width: 100%; align-items: center; gap: 10px; flex-wrap: wrap; }.host-tags { display: flex; gap: 7px; flex-wrap: wrap; }.snapshot-summary { display: flex; align-items: center; gap: 12px; margin-bottom: 14px; color: var(--text-secondary); }.log-title { display: flex; min-width: 0; align-items: center; gap: 14px; }.log-title > span:last-child { overflow: hidden; color: var(--text-secondary); text-overflow: ellipsis; white-space: nowrap; }
.log-section { margin: 12px 0; }.log-section label { display: block; margin-bottom: 7px; color: var(--text-secondary); font-size: 12px; font-weight: 600; }.log-section pre { max-height: 340px; overflow: auto; margin: 0; padding: 14px; border: 1px solid var(--border); border-radius: 10px; color: var(--text-regular); background: var(--bg-card-alt); font: 12px/1.65 ui-monospace, SFMono-Regular, Menlo, monospace; white-space: pre-wrap; word-break: break-word; }
:deep(.el-table) { --el-table-bg-color: var(--bg-card); --el-table-tr-bg-color: var(--bg-card); --el-table-header-bg-color: var(--bg-card-alt); --el-table-row-hover-bg-color: var(--bg-card-alt); color: var(--text-regular); }:deep(.el-table th.el-table__cell) { color: var(--text-secondary); font-size: 12px; }:deep(.el-table td.el-table__cell) { padding: 13px 0; border-bottom-color: var(--border); }
@media (max-width: 900px) { .task-page { padding: 14px; }.task-page-header { align-items: flex-start; }.task-header-count { display: none; }.task-filters { grid-template-columns: 1fr 1fr; }.form-grid { grid-template-columns: 1fr; }.form-span-2 { grid-column: auto; } }
@media (max-width: 600px) { .task-page-header h1 { font-size: 23px; }.task-filters { grid-template-columns: 1fr; }.task-toolbar, .history-toolbar { align-items: flex-start; flex-direction: column; } }
</style>
