<template>
  <el-drawer
    v-model="visible"
    :title="`${asset?.hostName || asset?.name || '资产'} · 连接中心`"
    size="min(760px, 100vw)"
    destroy-on-close
  >
    <div class="connection-intro">
      <span class="asset-eyebrow">REMOTE ACCESS</span>
      <h3>{{ $tr("为每条访问路径独立配置连接") }}</h3>
      <p>
        {{ $tr("SSH 终端、VNC 控制台与 Windows RDP 可以使用不同的地址和凭据。SSH 不可用时，可切换到已配置且可达的桌面或管理口。") }}
      </p>
    </div>
    <el-alert
      v-if="loaded && !gatewayAvailable"
      type="info"
      :closable="false"
      show-icon
      :title='$tr("桌面网关尚未启用，VNC / RDP 档案可先保存，启用网关后即可连接。")'
    />
    <div class="asset-toolbar">
      <span class="asset-muted">{{ connections.length }} {{ $tr("个连接档案") }}</span>
      <el-button v-if="canManage" type="primary" @click="edit()"
        >{{ $tr("新增连接") }}</el-button
      >
      <el-button @click="load" :loading="loading">{{ $tr("刷新") }}</el-button>
    </div>
    <div v-loading="loading" class="connection-list">
      <article
        v-for="item in connections"
        :key="item.id"
        class="connection-card"
      >
        <div class="connection-card-heading">
          <span class="connection-protocol">{{
            item.protocol.toUpperCase()
          }}</span>
          <div class="asset-cell">
            <strong>{{ item.name }}</strong
            ><span class="asset-mono asset-muted"
              >{{ item.address }}:{{ item.port }}</span
            >
          </div>
          <el-tag v-if="!item.enabled" type="info" size="small">{{ $tr("已停用") }}</el-tag>
          <el-tag v-else-if="item.readOnly" type="warning" size="small"
            >{{ $tr("只读桌面") }}</el-tag
          >
        </div>
        <div class="connection-meta">
          <span>{{ $tr("账户") }} {{ item.username || "由凭据提供" }}</span
          ><span v-if="item.domain">{{ $tr("域") }} {{ item.domain }}</span>
        </div>
        <div class="asset-actions">
          <el-button
            type="primary"
            :disabled="
              !item.enabled ||
              !item.canConnect ||
              (item.protocol !== 'ssh' && !gatewayAvailable)
            "
            @click="connect(item)"
            >{{ $tr("连接") }}</el-button
          >
          <el-button
            v-if="item.canTest"
            :loading="testing === item.id"
            @click="test(item)"
            >{{ $tr("检测端口") }}</el-button
          >
          <el-button v-if="canManage || item.canManage" text @click="edit(item)"
            >{{ $tr("编辑") }}</el-button
          >
          <el-button
            v-if="canManage || item.canManage"
            text
            type="danger"
            @click="remove(item)"
            >{{ $tr("删除") }}</el-button
          >
        </div>
        <p
          v-if="results[item.id]"
          class="connection-result"
          :class="results[item.id].reachable ? 'is-success' : 'is-danger'"
        >
          {{ results[item.id].message
          }}<span v-if="results[item.id].latencyMs !== undefined">
            · {{ results[item.id].latencyMs }} ms</span
          >
        </p>
      </article>
      <el-empty
        v-if="loaded && !connections.length"
        :description='$tr("暂无连接档案，添加 SSH、VNC 或 RDP 访问路径")'
      />
    </div>
    <el-collapse v-if="canAudit" @change="loadAudits">
      <el-collapse-item :title='$tr("最近 100 条连接与会话审计")' name="audits">
        <el-table :data="audits" v-loading="auditsLoading" class="asset-table">
          <el-table-column :label='$tr("时间 / 协议")' min-width="170"
            ><template #default="{ row }"
              ><div class="asset-cell">
                <span>{{ formatTime(row.startedAt) }}</span
                ><span class="asset-muted"
                  >{{ row.protocol?.toUpperCase() }} {{ $tr("· 用户 #") }}{{
                    row.userId
                  }}</span
                >
              </div></template
            ></el-table-column
          >
          <el-table-column :label='$tr("操作 / 结果")' min-width="130"
            ><template #default="{ row }"
              ><div class="asset-cell">
                <span>{{ auditAction(row.action) }}</span
                ><el-tag
                  size="small"
                  :type="row.status === 'failed' ? 'danger' : 'info'"
                  >{{ auditStatus(row.status) }}</el-tag
                >
              </div></template
            ></el-table-column
          >
          <el-table-column :label='$tr("来源 / 详情")' min-width="150"
            ><template #default="{ row }"
              ><div class="asset-cell">
                <span class="asset-mono">{{ row.clientIp }}</span
                ><span class="asset-muted">{{
                  row.message ||
                  (row.endedAt ? `结束于 ${formatTime(row.endedAt)}` : "—")
                }}</span>
              </div></template
            ></el-table-column
          >
        </el-table>
      </el-collapse-item>
    </el-collapse>
    <el-dialog
      v-model="formVisible"
      :title="$tr(form.id ? '编辑连接档案' : '新增连接档案')"
      width="min(600px, 94vw)"
      append-to-body
      destroy-on-close
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        class="connection-form"
      >
        <el-form-item :label='$tr("连接名称")' prop="name"
          ><el-input
            v-model="form.name"
            :placeholder='$tr("例如：业务 SSH / 管理口 VNC / Windows 桌面")'
            maxlength="100"
        /></el-form-item>
        <div class="asset-form-grid">
          <el-form-item :label='$tr("协议")' prop="protocol"
            ><el-select v-model="form.protocol" @change="protocolChanged"
              ><el-option
                v-for="p in protocols"
                :key="p"
                :label="$tr(p.toUpperCase())"
                :value="p" /></el-select
          ></el-form-item>
          <el-form-item :label='$tr("端口")' prop="port"
            ><el-input-number
              v-model="form.port"
              :min="1"
              :max="65535"
              controls-position="right"
          /></el-form-item>
        </div>
        <el-form-item :label='$tr("连接地址")' prop="address"
          ><el-input
            v-model="form.address"
            :placeholder='$tr("IP 或主机名；管理口可与业务地址不同")'
        /></el-form-item>
        <div class="asset-form-grid">
          <el-form-item :label='$tr("认证凭据")' prop="credentialId"
            ><el-select
              v-model="form.credentialId"
              filterable
              clearable
              :placeholder='$tr("选择已有凭据")'
              ><el-option
                v-for="c in credentials"
                :key="c.id"
                :label="$tr(c.name)"
                :value="c.id" /></el-select
          ></el-form-item>
          <el-form-item :label='$tr("登录用户")'
            ><el-input
              v-model="form.username"
              :placeholder='$tr("留空使用凭据中的用户名")'
              autocomplete="off"
          /></el-form-item>
        </div>
        <template v-if="form.protocol === 'rdp'">
          <div class="asset-form-grid">
            <el-form-item :label='$tr("Windows 域")'
              ><el-input v-model="form.domain" :placeholder='$tr("可选")'
            /></el-form-item>
            <el-form-item :label='$tr("安全模式")'
              ><el-select v-model="form.security"
                ><el-option :label='$tr("NLA（推荐）")' value="nla" /><el-option
                  label="TLS"
                  value="tls" /><el-option label="RDP" value="rdp" /></el-select
            ></el-form-item>
          </div>
          <el-form-item :label='$tr("证书校验")'
            ><el-switch
              v-model="form.ignoreCert"
              :active-text='$tr("忽略服务端证书校验")'
            />
            <div class="asset-field-help">
              {{ $tr("仅在受信环境中为自签名证书开启。") }}
            </div></el-form-item
          >
        </template>
        <el-form-item
          v-if="form.protocol === 'ssh'"
          :label='$tr("SSH 主机密钥指纹")'
          prop="hostKey"
          ><el-input
            v-model="form.hostKey"
            :placeholder='$tr("SHA256:...，请通过可信渠道核对")'
            ><template #append
              ><el-button :loading="probing" @click="probeKey"
                >{{ $tr("获取指纹") }}</el-button
              ></template
            ></el-input
          >
          <p class="asset-field-help">
            {{ $tr("获取后请与目标主机指纹核对；保存将绑定此指纹，后续身份发生变化时会拒绝连接。") }}
          </p></el-form-item
        >
        <div class="asset-form-grid">
          <el-form-item :label='$tr("连接状态")'
            ><el-switch v-model="form.enabled" :active-text='$tr("启用")'
          /></el-form-item>
          <el-form-item v-if="form.protocol !== 'ssh'" :label='$tr("桌面权限")'
            ><el-switch v-model="form.readOnly" :active-text='$tr("仅查看")'
          /></el-form-item>
        </div>
      </el-form>
      <template #footer
        ><el-button @click="formVisible = false">{{ $tr("取消") }}</el-button
        ><el-button type="primary" :loading="saving" @click="save"
          >{{ $tr("保存连接") }}</el-button
        ></template
      >
    </el-dialog>
  </el-drawer>
</template>

<script setup>
import { tr } from "@/locales";

import { computed, reactive, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import api from "@/api/cmdb";
const props = defineProps({
  modelValue: Boolean,
  assetType: { type: String, required: true },
  asset: Object
});
const emit = defineEmits(["update:modelValue"]);
const visible = computed({
  get: () => props.modelValue,
  set: v => emit("update:modelValue", v)
});
const router = useRouter();
const canAudit = ref(false),
  audits = ref([]),
  auditsLoading = ref(false);
const connections = ref([]),
  credentials = ref([]),
  loading = ref(false),
  loaded = ref(false),
  gatewayAvailable = ref(false),
  canManage = ref(false);
const formVisible = ref(false),
  saving = ref(false),
  probing = ref(false),
  testing = ref(null),
  formRef = ref(),
  results = reactive({});
const form = reactive({});
const protocols = computed(() =>
  props.assetType === "network" ? ["ssh"] : ["ssh", "vnc", "rdp"]
);
const rules = {
  credentialId: [
    { required: true, get message() { return tr("请选择认证凭据"); }, trigger: "change" }
  ],
  hostKey: [
    {
      validator: (_, value, callback) =>
        callback(
          form.protocol === "ssh" && !value
            ? new Error(tr("请获取并核对 SSH 主机指纹"))
            : undefined
        ),
      trigger: "blur"
    }
  ],
  name: [{ required: true, get message() { return tr("请输入连接名称"); }, trigger: "blur" }],
  address: [{ required: true, get message() { return tr("请输入连接地址"); }, trigger: "blur" }],
  port: [{ required: true, get message() { return tr("请输入端口"); }, trigger: "change" }]
};
function unpack(res) {
  if (res.data.code !== 200) throw new Error(res.data.message || tr("操作失败"));
  return res.data.data;
}
async function load() {
  if (!props.asset?.id) return;
  loading.value = true;
  try {
    const d = unpack(
      await api.getAssetConnections(props.assetType, props.asset.id)
    );
    connections.value = d.connections || [];
    gatewayAvailable.value = !!d.gatewayAvailable;
    canManage.value = !!d.canManage;
    loaded.value = true;
    try {
      canAudit.value = !!unpack(
        await api.checkAssetAction(props.assetType, props.asset.id, "audit")
      ).hasPermission;
    } catch {
      canAudit.value = false;
    }
  } catch (e) {
    ElMessage.error(e.message || "加载连接失败");
  } finally {
    loading.value = false;
  }
}
watch(
  () => [props.modelValue, props.asset?.id],
  ([open]) => {
    if (open) {
      loaded.value = false;
      connections.value = [];
      canManage.value = false;
      canAudit.value = false;
      audits.value = [];
      Object.keys(results).forEach(k => delete results[k]);
      load();
    }
  }
);
async function edit(item) {
  Object.keys(form).forEach(k => delete form[k]);
  Object.assign(
    form,
    item
      ? { ...item }
      : {
          name: "",
          protocol: props.asset?.osType === "windows" ? "rdp" : "ssh",
          address:
            props.asset?.sshIp ||
            props.asset?.manageIp ||
            props.asset?.privateIp ||
            "",
          port: props.asset?.osType === "windows" ? 3389 : 22,
          username: "",
          credentialId: undefined,
          domain: "",
          security: "nla",
          ignoreCert: false,
          readOnly: false,
          enabled: true,
          hostKey: ""
        }
  );
  formVisible.value = true;
  try {
    const d = unpack(await api.getConnectionCredentials());
    credentials.value = d.list || [];
  } catch {
    ElMessage.warning(tr("凭据列表加载失败，请使用管理员账号配置连接"));
  }
}
async function probeKey() {
  if (!form.address || !form.port)
    return ElMessage.warning(tr("请先填写连接地址和端口"));
  probing.value = true;
  try {
    form.hostKey = unpack(
      await api.getConnectionHostKey(props.assetType, props.asset.id, {
        address: form.address,
        port: form.port
      })
    ).hostKey;
    ElMessage.success(tr("已获取指纹，请核对后保存"));
  } catch (e) {
    ElMessage.error(e.message || "获取指纹失败");
  } finally {
    probing.value = false;
  }
}
function protocolChanged(protocol) {
  form.port = { ssh: 22, vnc: 5900, rdp: 3389 }[protocol];
  if (protocol === "ssh") form.readOnly = false;
}
async function save() {
  if (!(await formRef.value.validate().catch(() => false))) return;
  saving.value = true;
  try {
    const data = {
      name: form.name,
      protocol: form.protocol,
      address: form.address,
      port: form.port,
      username: form.username || "",
      credentialId: form.credentialId || 0,
      domain: form.domain || "",
      security: form.security || "nla",
      ignoreCert: !!form.ignoreCert,
      readOnly: !!form.readOnly,
      enabled: !!form.enabled,
      hostKey: form.hostKey || ""
    };
    unpack(
      await (form.id
        ? api.updateAssetConnection(
            props.assetType,
            props.asset.id,
            form.id,
            data
          )
        : api.createAssetConnection(props.assetType, props.asset.id, data))
    );
    ElMessage.success(tr("连接已保存"));
    formVisible.value = false;
    await load();
  } catch (e) {
    ElMessage.error(e.message || tr("保存失败"));
  } finally {
    saving.value = false;
  }
}
async function remove(item) {
  try {
    await ElMessageBox.confirm(`删除连接“${item.name}”？`, tr("删除连接"), {
      type: "warning"
    });
    unpack(
      await api.deleteAssetConnection(props.assetType, props.asset.id, item.id)
    );
    await load();
  } catch (e) {
    if (e !== "cancel" && e !== "close")
      ElMessage.error(e.message || tr("删除失败"));
  }
}
async function test(item) {
  testing.value = item.id;
  try {
    results[item.id] = unpack(
      await api.testAssetConnection(props.assetType, props.asset.id, item.id)
    );
  } catch (e) {
    ElMessage.error(e.message || "检测失败");
  } finally {
    testing.value = null;
  }
}
function formatTime(value) {
  return value
    ? new Date(value).toLocaleString("zh-CN", { hour12: false })
    : "—";
}
function auditAction(action) {
  return (
    {
      connect: "建立会话",
      create: "新增连接",
      update: "修改连接",
      delete: "删除连接",
      test: "检测连接"
    }[action] || action
  );
}
function auditStatus(status) {
  return (
    {
      success: "成功",
      failed: "失败",
      connecting: "连接中",
      connected: "已连接",
      closed: "已结束",
      active: "进行中"
    }[status] || status
  );
}
async function loadAudits(open) {
  if (!open.includes("audits")) return;
  auditsLoading.value = true;
  try {
    audits.value =
      unpack(
        await api.getAssetConnectionAudits(props.assetType, props.asset.id)
      ).list || [];
  } catch (e) {
    ElMessage.error(e.message || "读取审计失败");
  } finally {
    auditsLoading.value = false;
  }
}
function connect(item) {
  const url = router.resolve({
    path: "/cmdb/remote",
    query: {
      assetType: props.assetType,
      assetId: props.asset.id,
      connectionId: item.id,
      name: item.name
    }
  });
  window.open(url.href, "_blank", "noopener,noreferrer");
}
</script>
