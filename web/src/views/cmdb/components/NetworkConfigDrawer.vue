<template>
  <el-drawer
    v-model="visible"
    :title="`${asset?.name || '网络设备'} · 配置管理`"
    size="min(940px, 100vw)"
    destroy-on-close
  >
    <div class="connection-intro">
      <span class="asset-eyebrow">CONFIGURATION HISTORY</span>
      <h3>{{ $tr("备份、比较与恢复设备配置") }}</h3>
      <p>{{ $tr("配置快照用于追踪变更；每次恢复会先备份当前配置，并记录操作结果。") }}</p>
    </div>
    <el-alert
      v-if="!encryptionReady && !loading"
      :title='$tr("配置加密尚未启用，请联系管理员设置配置加密密钥后备份或恢复。")'
      type="warning"
      :closable="false"
      show-icon
    />
    <el-alert
      v-if="error"
      :title="$tr(error)"
      type="error"
      :closable="false"
      show-icon
    />
    <div class="asset-form-grid">
      <el-form-item :label='$tr("平台")'
        ><el-select v-model="platform"
          ><el-option
            v-for="p in capabilities"
            :key="p.platform"
            :label="$tr(p.label)"
            :value="p.platform" /></el-select
      ></el-form-item>
      <el-form-item :label='$tr("SSH 连接")'
        ><el-select v-model="connectionId" :placeholder='$tr("选择已配置的 SSH 档案")'
          ><el-option
            v-for="p in connections"
            :key="p.id"
            :label="`${p.name} · ${p.address}`"
            :value="p.id" /></el-select
      ></el-form-item>
    </div>
    <p class="asset-field-help">
      {{ $tr(capabilities.find(p => p.platform === platform)?.description) }}
    </p>
    <div class="asset-toolbar">
      <el-input
        v-model="remark"
        :placeholder='$tr("备份备注（可选）")'
        maxlength="200"
      /><el-button
        v-if="permissions.config_backup"
        type="primary"
        :disabled="!connectionId || !platform || !encryptionReady"
        :loading="busy"
        @click="backup"
        >{{ $tr("立即备份") }}</el-button
      ><el-button :loading="loading" @click="load">{{ $tr("刷新") }}</el-button>
    </div>
    <el-tabs v-model="tab">
      <el-tab-pane :label='$tr("配置快照")' name="snapshots">
        <div class="asset-toolbar">
          <span class="asset-muted">{{ $tr("选择两个版本比较差异") }}</span
          ><el-button
            :disabled="selected.length !== 2"
            :loading="comparing"
            @click="compare"
            >{{ $tr("比较版本 (") }}{{ selected.length }}/2)</el-button
          >
        </div>
        <el-table
          v-loading="loading"
          :data="snapshots"
          row-key="id"
          class="asset-table"
          @selection-change="selected = $event"
        >
          <el-table-column type="selection" width="44" />
          <el-table-column :label='$tr("快照")' min-width="190"
            ><template #default="{ row }"
              ><div class="asset-cell">
                <strong>#{{ row.id }} · {{ row.platform }}</strong
                ><span class="asset-muted">{{
                  formatTime(row.createdAt)
                }}</span>
              </div></template
            ></el-table-column
          >
          <el-table-column :label='$tr("来源 / 大小")' min-width="130"
            ><template #default="{ row }"
              ><div class="asset-cell">
                <span>{{
                  row.source === "pre_restore" ? "恢复前自动备份" : "手动备份"
                }}</span
                ><span class="asset-muted">{{ row.byteSize }} bytes</span>
              </div></template
            ></el-table-column
          >
          <el-table-column :label='$tr("校验摘要")' min-width="150"
            ><template #default="{ row }"
              ><span class="asset-mono" :title="$tr(row.checksum)">{{
                row.checksum?.slice(0, 12) || "—"
              }}</span></template
            ></el-table-column
          >
          <el-table-column :label='$tr("操作")' width="145"
            ><template #default="{ row }"
              ><el-button link type="primary" @click="view(row)">{{ $tr("查看") }}</el-button
              ><el-button
                v-if="permissions.config_restore && row.restorable"
                link
                type="warning"
                :disabled="!connectionId || busy || !encryptionReady"
                @click="restore(row)"
                >{{ $tr("恢复") }}</el-button
              ></template
            ></el-table-column
          >
          <template #empty
            ><el-empty :description='$tr("暂无备份快照")' :image-size="80"
          /></template>
        </el-table>
        <div class="asset-pagination">
          <el-pagination
            v-model:current-page="page"
            :page-size="20"
            :total="total"
            layout="total, prev, pager, next"
            @current-change="loadSnapshots"
          />
        </div>
      </el-tab-pane>
      <el-tab-pane :label='$tr("操作记录")' name="audits">
        <el-table :data="audits" class="asset-table"
          ><el-table-column :label='$tr("操作时间")' min-width="180"
            ><template #default="{ row }">{{
              formatTime(row.createdAt)
            }}</template></el-table-column
          ><el-table-column
            prop="action"
            :label='$tr("动作")'
            width="100" /><el-table-column
            prop="status"
            :label='$tr("结果")'
            width="100" /><el-table-column
            prop="message"
            :label='$tr("详情")'
            min-width="220"
            show-overflow-tooltip
        /></el-table>
        <div class="asset-pagination">
          <el-pagination
            v-model:current-page="auditPage"
            :page-size="20"
            :total="auditTotal"
            layout="total, prev, pager, next"
            @current-change="loadAudits"
          />
        </div>
      </el-tab-pane>
    </el-tabs>
    <el-dialog
      v-model="contentVisible"
      :title="$tr(contentTitle)"
      width="min(1000px, 94vw)"
      append-to-body
    >
      <pre class="asset-code">{{ content }}</pre>
    </el-dialog>
  </el-drawer>
</template>

<script setup>
import { tr } from "@/locales";

import { computed, ref, watch, reactive } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import api from "@/api/cmdb";
const props = defineProps({ modelValue: Boolean, asset: Object }),
  emit = defineEmits(["update:modelValue"]);
const visible = computed({
  get: () => props.modelValue,
  set: v => emit("update:modelValue", v)
});
const encryptionReady = ref(false);
const loading = ref(false),
  busy = ref(false),
  comparing = ref(false),
  error = ref(""),
  tab = ref("snapshots"),
  capabilities = ref([]),
  connections = ref([]),
  platform = ref("openwrt"),
  connectionId = ref(),
  remark = ref("");
const snapshots = ref([]),
  audits = ref([]),
  selected = ref([]),
  page = ref(1),
  total = ref(0),
  auditPage = ref(1),
  auditTotal = ref(0),
  contentVisible = ref(false),
  contentTitle = ref(""),
  content = ref("");
const permissions = reactive({
  config_view: false,
  config_backup: false,
  config_restore: false
});
function unpack(res) {
  if (res.data.code !== 200) throw new Error(res.data.message || tr("操作失败"));
  return res.data.data;
}
function formatTime(value) {
  if (!value) return "—";
  const d = new Date(value);
  return Number.isNaN(d.getTime())
    ? value
    : d.toLocaleString("zh-CN", { hour12: false });
}
async function loadSnapshots() {
  try {
    const d = unpack(
      await api.getNetworkConfigSnapshots(props.asset.id, {
        page: page.value,
        size: 20
      })
    );
    snapshots.value = d.list || [];
    total.value = d.total || 0;
    selected.value = [];
  } catch (e) {
    error.value = e.message;
  }
}
async function loadAudits() {
  try {
    const d = unpack(
      await api.getNetworkConfigAudits(props.asset.id, {
        page: auditPage.value,
        size: 20
      })
    );
    audits.value = d.list || [];
    auditTotal.value = d.total || 0;
  } catch (e) {
    error.value = e.message;
  }
}
async function load() {
  if (!props.asset?.id) return;
  loading.value = true;
  error.value = "";
  try {
    const results = await Promise.all([
      api.getNetworkConfigCapabilities(props.asset.id),
      api.getAssetConnections("network", props.asset.id).catch(() => null),
      ...Object.keys(permissions).map(action =>
        api.checkAssetAction("network", props.asset.id, action)
      )
    ]);
    const caps = unpack(results[0]);
    capabilities.value = caps.list || [];
    encryptionReady.value = !!caps.encryptionReady;
    connections.value = (
      results[1] ? unpack(results[1]).connections || [] : []
    ).filter(p => p.protocol === "ssh" && p.enabled);
    if (!connections.value.some(c => c.id === connectionId.value))
      connectionId.value = connections.value[0]?.id;
    Object.keys(permissions).forEach((action, i) => {
      permissions[action] = !!unpack(results[i + 2]).hasPermission;
    });
    if (permissions.config_view)
      await Promise.all([loadSnapshots(), loadAudits()]);
    else {
      snapshots.value = [];
      audits.value = [];
      error.value = "当前账号没有查看此设备配置的权限。";
    }
  } catch (e) {
    error.value = e.message || "加载配置管理失败";
  } finally {
    loading.value = false;
  }
}
watch(
  () => [props.modelValue, props.asset?.id],
  ([open]) => {
    if (open) {
      page.value = 1;
      auditPage.value = 1;
      connectionId.value = undefined;
      encryptionReady.value = false;
      snapshots.value = [];
      audits.value = [];
      selected.value = [];
      load();
    }
  }
);
async function backup() {
  busy.value = true;
  try {
    unpack(
      await api.backupNetworkConfig(props.asset.id, {
        connectionId: connectionId.value,
        platform: platform.value,
        remark: remark.value
      })
    );
    ElMessage.success(tr("配置备份成功"));
    page.value = 1;
    await load();
  } catch (e) {
    ElMessage.error(e.message || "备份失败");
  } finally {
    busy.value = false;
  }
}
async function view(row) {
  try {
    const d = unpack(
      await api.getNetworkConfigSnapshot(props.asset.id, row.id)
    );
    content.value = d.content || "";
    contentTitle.value = `快照 #${row.id} · ${row.platform}`;
    contentVisible.value = true;
  } catch (e) {
    ElMessage.error(e.message);
  }
}
async function compare() {
  comparing.value = true;
  try {
    const [a, b] = [...selected.value].sort((a, b) => a.id - b.id);
    const d = unpack(
      await api.getNetworkConfigDiff(props.asset.id, a.id, b.id)
    );
    content.value = d.changed
      ? d.diff || "配置有变化，变化内容已脱敏"
      : "两个版本内容一致";
    contentTitle.value = `配置差异 #${a.id} → #${b.id}`;
    contentVisible.value = true;
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    comparing.value = false;
  }
}
async function restore(row) {
  try {
    const { value } = await ElMessageBox.prompt(
      `即将恢复快照 #${row.id}。系统会先备份当前配置，再写入 UCI 持久配置；不会自动重载服务，请在维护窗口检查并重载。输入设备名称「${props.asset.name}」确认。`,
      tr("恢复 OpenWrt 配置"),
      {
        type: "warning",
        confirmButtonText: "备份当前配置并恢复",
        inputValidator: value => value === props.asset.name || "设备名称不匹配"
      }
    );
    busy.value = true;
    unpack(
      await api.restoreNetworkConfig(props.asset.id, {
        snapshotId: row.id,
        connectionId: connectionId.value,
        confirmName: value
      })
    );
    ElMessage.success(tr("配置已恢复，请在维护窗口检查并重载服务"));
    await load();
  } catch (e) {
    if (e !== "cancel" && e !== "close")
      ElMessage.error(e.message || "恢复失败");
  } finally {
    busy.value = false;
  }
}
</script>
