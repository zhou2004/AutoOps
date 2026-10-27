<script setup>
import * as OperationIcons from "@element-plus/icons-vue";
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import { tr } from "@/locales";
import api from "@/api/cmdb";

const props = defineProps({
  source: { type: Object, required: true },
  assetGroups: { type: Array, required: true }
});
const emit = defineEmits(["close", "done", "busy"]);
const visible = ref(true);
const busy = ref(false);
const loadingConnections = ref(false);
const connections = ref([]);
const error = ref("");
const result = ref(null);
const form = ref({
  targetAsset: props.source.migrate ? "" : props.source.asset,
  targetConnectionId: undefined,
  name: props.source.migrate
    ? props.source.row.Names
    : `${props.source.row.Names}-copy`,
  start: props.source.row.State === "running",
  preservePorts: props.source.migrate,
  confirmName: ""
});
const sameNode = computed(() => form.value.targetAsset === props.source.asset);
const valid = computed(
  () =>
    !loadingConnections.value &&
    form.value.targetConnectionId &&
    /^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$/.test(form.value.name) &&
    form.value.confirmName === props.source.row.Names &&
    (!sameNode.value ||
      (!props.source.migrate && form.value.name !== props.source.row.Names))
);
watch([sameNode, () => form.value.start], ([same, start]) => {
  if (same && start && props.source.row.State === "running")
    form.value.preservePorts = false;
});
let generation = 0;
onBeforeUnmount(() => generation++);
function unpack(response) {
  if (response.data?.code !== 200)
    throw new Error(response.data?.message || tr("操作失败"));
  return response.data.data;
}
watch(
  () => form.value.targetAsset,
  async value => {
    const version = ++generation;
    connections.value = [];
    form.value.targetConnectionId = undefined;
    form.value.preservePorts = !!value && value !== props.source.asset;
    error.value = "";
    if (!value) return;
    loadingConnections.value = true;
    try {
      const [type, id] = value.split(":");
      const data = unpack(await api.getDockerConnections(type, Number(id)));
      if (version !== generation) return;
      if (!data.canDockerOperate)
        throw new Error(tr("没有目标节点的 Docker 操作权限"));
      connections.value = (data.connections || []).filter(
        x => x.protocol === "ssh" && x.enabled && x.canDocker && !x.readOnly
      );
      form.value.targetConnectionId =
        connections.value.find(
          x => sameNode.value && x.id === props.source.connectionId
        )?.id || connections.value[0]?.id;
      if (!connections.value.length)
        error.value = tr("目标节点没有可用的 SSH 连接档案");
    } catch (e) {
      if (version === generation) error.value = e.message;
    } finally {
      if (version === generation) loadingConnections.value = false;
    }
  },
  { immediate: true }
);
async function submit() {
  if (!valid.value || busy.value || result.value) return;
  const [targetAssetType, targetId] = form.value.targetAsset.split(":");
  busy.value = true;
  emit("busy", true);
  error.value = "";
  try {
    result.value = unpack(
      await api.transferDockerContainer(
        props.source.type,
        props.source.id,
        props.source.row.ID,
        {
          connectionId: props.source.connectionId,
          targetAssetType,
          targetAssetId: Number(targetId),
          targetConnectionId: form.value.targetConnectionId,
          name: form.value.name,
          migrate: props.source.migrate,
          start: form.value.start,
          preservePorts: form.value.preservePorts,
          confirmName: form.value.confirmName
        }
      )
    );
    ElMessage.success(
      tr(props.source.migrate ? "容器迁移完成" : "容器复制完成")
    );
    emit("done");
  } catch (e) {
    error.value = e.message;
    // Refresh even on failure: rollback/partial success may have changed container state.
    emit("done");
  } finally {
    busy.value = false;
    emit("busy", false);
  }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="$tr(source.migrate ? '迁移容器' : '复制容器')"
    width="min(720px, 94vw)"
    :close-on-click-modal="false"
    :close-on-press-escape="!busy"
    :show-close="!busy"
    @closed="emit('close')"
  >
    <div class="transfer-source">
      <span>{{ $tr("源容器") }}</span
      ><strong>{{ source.row.Names }}</strong
      ><code>{{ source.row.Image }}</code>
    </div>
    <el-alert
      type="warning"
      show-icon
      :closable="false"
      :title="
        $tr(
          source.migrate
            ? '目标验证成功后才删除源容器，源数据卷保留。'
            : '复制完成后保留源容器，目标使用独立数据卷。'
        )
      "
      :description="
        $tr(
          '运行中的源容器会临时停止，以复制文件系统和挂载目录。请安排停机窗口；共享数据目录需同时停止其他写入者。'
        )
      "
    />
    <p class="transfer-note">
      {{
        $tr(
          "保留启动配置和资源限制；命名卷及目录绑定挂载复制为目标独立数据卷。文件绑定挂载、设备绑定和共享容器命名空间暂不支持。目标需预先具备同名网络，静态 IP 将重新分配。"
        )
      }}
    </p>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      show-icon
      :closable="false"
    />
    <el-alert
      v-if="result"
      type="success"
      show-icon
      :closable="false"
      :title="`${$tr('目标容器已创建')}：${result.name}`"
      :description="result.containerId"
    />
    <el-form v-else label-position="top" :disabled="busy" class="transfer-form">
      <div class="transfer-grid">
        <el-form-item :label="$tr('目标节点')" required>
          <el-select
            v-model="form.targetAsset"
            filterable
            :placeholder="$tr('选择主机或物理机')"
          >
            <el-option-group
              v-for="group in assetGroups"
              :key="group.label"
              :label="$tr(group.label)"
            >
              <el-option
                v-for="asset in group.options"
                :key="asset.value"
                :label="asset.label"
                :value="asset.value"
                :disabled="source.migrate && asset.value === source.asset"
              />
            </el-option-group>
          </el-select>
        </el-form-item>
        <el-form-item :label="$tr('目标 SSH 连接档案')" required>
          <el-select
            v-model="form.targetConnectionId"
            :loading="loadingConnections"
            :placeholder="$tr('选择连接档案')"
          >
            <el-option
              v-for="connection in connections"
              :key="connection.id"
              :label="`${connection.name} · ${connection.address}`"
              :value="connection.id"
            />
          </el-select>
        </el-form-item>
      </div>
      <el-form-item :label="$tr('目标容器名称')" required
        ><el-input v-model="form.name" maxlength="128"
      /></el-form-item>
      <el-form-item :label="$tr('启动与端口')">
        <el-checkbox
          v-model="form.start"
          :disabled="source.migrate && source.row.State === 'running'"
          >{{ $tr("完成后启动目标容器") }}</el-checkbox
        >
        <el-checkbox
          v-model="form.preservePorts"
          :disabled="sameNode && form.start && source.row.State === 'running'"
          >{{ $tr("保留宿主机端口映射") }}</el-checkbox
        >
        <span class="transfer-note">{{
          $tr("取消保留端口时，由 Docker 分配可用端口。")
        }}</span>
      </el-form-item>
      <el-form-item
        :label="`${$tr('输入源容器名称确认')}：${source.row.Names}`"
        required
        ><el-input v-model="form.confirmName" autocomplete="off"
      /></el-form-item>
    </el-form>
    <p v-if="busy" class="transfer-note" role="status">
      {{ $tr("正在传输镜像与挂载数据并验证目标容器，请保持页面打开。") }}
    </p>
    <template #footer>
      <el-button :icon="OperationIcons.Close" :disabled="busy" @click="visible = false">{{
        $tr(result ? "完成" : "取消")
      }}</el-button>
      <el-button :icon="OperationIcons.Switch"
        v-if="!result"
        type="primary"
        :loading="busy"
        :disabled="!valid"
        @click="submit"
        >{{ $tr(source.migrate ? "开始迁移" : "开始复制") }}</el-button
      >
    </template>
  </el-dialog>
</template>

<style scoped>
.transfer-source {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  margin-bottom: 16px;
  color: var(--text-primary);
}
.transfer-source span,
.transfer-note {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.7;
}
.transfer-source code {
  overflow-wrap: anywhere;
}
.transfer-form {
  margin-top: 20px;
}
.transfer-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}
.transfer-note {
  width: 100%;
}
@media (max-width: 640px) {
  .transfer-grid {
    grid-template-columns: 1fr;
    gap: 0;
  }
}
</style>
