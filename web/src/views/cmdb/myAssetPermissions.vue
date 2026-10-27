<template>
  <div class="asset-page">
    <header class="asset-page-header">
      <div>
        <span class="asset-eyebrow">MY ACCESS</span>
        <h1>{{ $tr("我的授权资产") }}</h1>
        <p>{{ $tr("这里展示你可以访问的资产及每台资产的实际操作权限。") }}</p>
      </div>
      <el-button :loading="loading" @click="fetch">{{ $tr("刷新资产") }}</el-button>
    </header>
    <el-card shadow="never" class="asset-surface">
      <el-alert
        v-if="error"
        :title="$tr(error)"
        type="error"
        :closable="false"
        show-icon
      />
      <el-tabs v-model="activeTab">
        <el-tab-pane
          v-for="type in assetTypes"
          :key="type.key"
          :name="type.key"
          :label="`${type.label} (${assets[type.key].length})`"
        >
          <el-table
            v-loading="loading"
            :data="assets[type.key]"
            row-key="id"
            class="asset-table"
          >
            <el-table-column :label='$tr("资产")' min-width="230"
              ><template #default="{ row }"
                ><div class="asset-identity">
                  <span class="asset-avatar">{{ type.short }}</span>
                  <div class="asset-cell">
                    <strong>{{ row.hostName || row.name || row.sn }}</strong
                    ><span class="asset-muted asset-mono">{{
                      row.sshIp || row.manageIp || row.privateIp || "未登记地址"
                    }}</span>
                  </div>
                </div></template
              ></el-table-column
            >
            <el-table-column :label='$tr("资产信息")' min-width="160"
              ><template #default="{ row }"
                ><div class="asset-cell">
                  <span>{{ row.os || row.osType || row.brand || "—" }}</span
                  ><span class="asset-muted">{{
                    row.model || row.sn || row.privateIp || "—"
                  }}</span>
                </div></template
              ></el-table-column
            >
            <el-table-column :label='$tr("当前权限")' min-width="320"
              ><template #default="{ row }"
                ><div class="asset-tags">
                  <el-tag
                    v-for="action in actions(type.key, row.id)"
                    :key="action"
                    size="small"
                    type="info"
                    effect="plain"
                    >{{ labels[action] || action }}</el-tag
                  ><span
                    v-if="!actions(type.key, row.id).length"
                    class="asset-muted"
                    >{{ $tr("无可用操作") }}</span
                  >
                </div></template
              ></el-table-column
            >
            <el-table-column :label='$tr("操作")' width="130" fixed="right"
              ><template #default="{ row }"
                ><el-button
                  v-if="canConnect(type.key, row.id)"
                  size="small"
                  plain
                  type="primary"
                  @click="open(type.key, row)"
                  >{{ $tr("连接") }}</el-button
                ><el-tag v-else type="info" size="small"
                  >{{ $tr("仅资产信息") }}</el-tag
                ></template
              ></el-table-column
            >
            <template #empty
              ><el-empty
                :description="`暂无授权${type.label}`"
                :image-size="90"
            /></template>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>
    <AssetConnections
      v-model="connectionsVisible"
      :asset-type="selectedType"
      :asset="selectedAsset"
    />
  </div>
</template>
<script setup>
import { ref, reactive, onMounted } from "vue";
import api from "@/api/cmdb";
import AssetConnections from "./components/AssetConnections.vue";
const isAdmin = ref(false);
const activeTab = ref("host"),
  loading = ref(false),
  error = ref(""),
  grants = ref([]),
  assets = reactive({ host: [], physical: [], network: [] });
const connectionsVisible = ref(false),
  selectedType = ref("host"),
  selectedAsset = ref(null);
const assetTypes = [
  { key: "host", label: "主机", short: "OS" },
  { key: "physical", label: "物理机", short: "PHY" },
  { key: "network", label: "网络设备", short: "NET" }
];
const labels = {
  get: "查看详情",
  list: "查看列表",
  connect: "SSH",
  connect_ssh: "SSH",
  connect_vnc: "VNC",
  connect_rdp: "RDP",
  command: "执行命令",
  audit: "会话审计",
  file_list: "浏览文件",
  file_upload: "上传",
  file_download: "下载",
  file_delete: "删除文件",
  config_view: "查看配置",
  config_backup: "备份配置",
  config_restore: "恢复配置",
  db_query: "数据库查询",
  db_execute: "数据库写操作",
  docker_view: "查看 Docker",
  docker_logs: "容器日志",
  docker_operate: "操作容器",
  docker_remove: "删除容器",
  update: "修改",
  create: "创建",
  delete: "删除资产",
  admin: "全部管理"
};
function unpack(res) {
  if (res.data.code !== 200) throw new Error(res.data.message || "加载失败");
  return res.data.data;
}
function actions(type, id) {
  return isAdmin.value
    ? ["admin"]
    : grants.value.find(
        g => g.assetType === type && Number(g.assetId) === Number(id)
      )?.actions || [];
}
function canConnect(type, id) {
  return actions(type, id).some(a =>
    ["admin", "connect", "connect_ssh", "connect_vnc", "connect_rdp"].includes(
      a
    )
  );
}
function open(type, asset) {
  selectedType.value = type;
  selectedAsset.value = asset;
  connectionsVisible.value = true;
}
async function loadHosts(ids) {
  if (!isAdmin.value && !ids.length) {
    assets.host = [];
    return;
  }
  const set = new Set(ids.map(Number)),
    found = [];
  let page = 1,
    total = Infinity;
  while ((page - 1) * 100 < total) {
    const d = unpack(await api.getCmdbHostList({ page, pageSize: 100 }));
    const rows = d.list || [];
    total = Number(d.total || rows.length);
    found.push(...rows.filter(row => isAdmin.value || set.has(Number(row.id))));
    if (!rows.length || (!isAdmin.value && found.length >= set.size)) break;
    page++;
  }
  assets.host = found;
}
async function fetch() {
  loading.value = true;
  error.value = "";
  try {
    const d = unpack(await api.getMyAssetPermissions());
    isAdmin.value = !!d.isAdmin;
    grants.value = d.assetPermissions || [];
    await Promise.all([
      loadHosts(d.allowedHostIds || []),
      (async () => {
        const ids = new Set((d.allowedPhysicalIds || []).map(Number));
        assets.physical =
          isAdmin.value || ids.size
            ? (unpack(await api.getPhysicalMachineAll()).list || []).filter(
                a => isAdmin.value || ids.has(Number(a.id))
              )
            : [];
      })(),
      (async () => {
        const ids = new Set((d.allowedNetworkIds || []).map(Number));
        assets.network =
          isAdmin.value || ids.size
            ? (unpack(await api.getNetworkDeviceAll()).list || []).filter(
                a => isAdmin.value || ids.has(Number(a.id))
              )
            : [];
      })()
    ]);
  } catch (e) {
    error.value = e.message || "获取授权资产失败";
  } finally {
    loading.value = false;
  }
}
onMounted(fetch);
</script>
