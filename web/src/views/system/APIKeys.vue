<script setup lang="ts">
import { useRoute } from "vue-router";
import { tr } from "@/locales";
import { computed, onMounted, reactive, ref } from "vue";
import {
  Key,
  Plus,
  Refresh,
  Delete,
  CopyDocument,
  Close,
  Check,
  Search,
  Lock
} from "@element-plus/icons-vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  listAPIKeys,
  createAPIKey,
  revokeAPIKey,
  keyUsers,
  keyUserPermissions,
  keyPermissionCatalog,
  type KeyUser,
  type PlatformKey
} from "@/api/apiKeys";
import { actionLabels, type PermissionModule } from "@/api/platformPermissions";
import KeyScopeSelector from "./components/KeyScopeSelector.vue";
const modules = ref<PermissionModule[]>([]);
const catalogLoading = ref(false),
  catalogError = ref("");
const ownerError = ref("");
async function loadCatalog() {
  catalogLoading.value = true;
  catalogError.value = "";
  try {
    modules.value = (await keyPermissionCatalog()).modules;
  } catch (e) {
    catalogError.value = (e as Error).message;
  } finally {
    catalogLoading.value = false;
  }
}
const rows = ref<PlatformKey[]>([]),
  loading = ref(false),
  saving = ref(false),
  open = ref(false),
  token = ref(""),
  search = ref("");
const error = ref("");
const canManage = ref(false),
  users = ref<KeyUser[]>([]),
  total = ref(0),
  page = ref(1);
const ownerAdmin = ref(false),
  ownerLoading = ref(false);
const ownerFilter = ref<number>();
const route = useRoute();
let ownerRequest = 0;
async function searchUsers(query = "") {
  try {
    users.value = await keyUsers(query);
  } catch (e) {
    ElMessage.error((e as Error).message);
  }
}
async function loadOwner(id: number) {
  const requestId = ++ownerRequest;
  ownerError.value = "";
  ownerAdmin.value = false;
  if (!id) return;
  ownerLoading.value = true;
  try {
    const data = await keyUserPermissions(id);
    if (requestId === ownerRequest) {
      ownerAdmin.value = data.isAdmin;
      if (!users.value.some(u => u.id === id))
        users.value.push({
          id,
          username: data.username,
          nickname: "",
          status: 1
        });
    }
  } catch (e) {
    if (requestId === ownerRequest) ownerError.value = (e as Error).message;
  } finally {
    if (requestId === ownerRequest) ownerLoading.value = false;
  }
}
const form = reactive({
  name: "",
  userId: undefined as number | undefined,
  days: 30,
  scopes: [] as string[]
});
const scopeLabels = computed<Record<string, string>>(() =>
  Object.fromEntries(
    modules.value.flatMap(m =>
      m.actions.map(a => [
        `${m.key}:${a}`,
        `${tr(m.name)} · ${tr(actionLabels[a] || a)}`
      ])
    )
  )
);
const keyScopes = (k: PlatformKey) => k.scopes.split(",").filter(Boolean);
const scopeLabel = (s: string) =>
  s === "user:inherit"
    ? tr("旧版：继承所属用户权限")
    : scopeLabels.value[s] || s;
const state = (k: PlatformKey) =>
  k.revokedAt
    ? "已撤销"
    : new Date(k.expiresAt).getTime() < Date.now()
      ? "已过期"
      : "有效";
const filtered = computed(() =>
  rows.value.filter(k =>
    k.name.toLowerCase().includes(search.value.toLowerCase())
  )
);
const date = (s: string | null) =>
  s ? new Date(s).toLocaleString() : tr("尚未使用");
async function load() {
  loading.value = true;
  error.value = "";
  try {
    const data = await listAPIKeys(page.value, ownerFilter.value);
    rows.value = data.items;
    total.value = data.total;
    canManage.value = data.canManage;
    modules.value = data.modules;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}
async function add() {
  if (!canManage.value) return;
  await searchUsers();
  ownerAdmin.value = false;
  Object.assign(form, {
    name: "",
    userId: undefined,
    days: 30,
    scopes: []
  });
  ownerError.value = "";
  await loadCatalog();
  const userId = Number(route.query.userId);
  if (Number.isSafeInteger(userId) && userId > 0) {
    form.userId = userId;
    await loadOwner(userId);
  }
  open.value = true;
}
async function save() {
  if (!form.name.trim() || !form.userId)
    return ElMessage.warning(tr("请输入名称并选择所属用户"));
  if (!form.scopes.length) return ElMessage.warning(tr("请至少选择一项权限"));
  if (
    catalogLoading.value ||
    catalogError.value ||
    ownerLoading.value ||
    ownerError.value
  )
    return;
  saving.value = true;
  try {
    const result = await createAPIKey({
      ...form,
      userId: form.userId,
      scopes: [...form.scopes]
    });
    token.value = result.token;
    open.value = false;
    await load();
  } catch (e) {
    ElMessage.error((e as Error).message);
  } finally {
    saving.value = false;
  }
}
async function revoke(row: PlatformKey) {
  try {
    await ElMessageBox.confirm(
      `撤销「${row.name}」后，使用该密钥的连接将立即失效。`,
      "撤销 API Key",
      { type: "warning", confirmButtonText: "撤销", cancelButtonText: "取消" }
    );
  } catch {
    return;
  }
  try {
    await revokeAPIKey(row.id);
    await load();
    ElMessage.success(tr("密钥已撤销"));
  } catch (e) {
    ElMessage.error((e as Error).message);
  }
}
async function copy() {
  try {
    await navigator.clipboard.writeText(token.value);
    ElMessage.success(tr("已复制"));
  } catch {
    ElMessage.warning(tr("自动复制失败，请选中密钥手动复制"));
  }
}
onMounted(async () => {
  await load();
  if (canManage.value) {
    await searchUsers();
    if (route.query.userId) await add();
  }
});
</script>
<template>
  <div class="key-page">
    <header class="key-header">
      <div>
        <span class="key-eyebrow">PLATFORM ACCESS</span>
        <h1>
          <el-icon><Key /></el-icon>{{ $tr("API Key") }}
        </h1>
        <p>
          {{
            $tr(
              "API Key 即用户访问 Token，可供 Harness 和其他外部系统调用平台接口。"
            )
          }}
        </p>
      </div>
      <el-button v-if="canManage" type="primary" :icon="Plus" @click="add">{{
        $tr("分配 Key")
      }}</el-button>
    </header>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      show-icon
    />
    <div class="key-note">
      <el-icon><Lock /></el-icon
      ><span>{{
        $tr(
          "每个 Key 独立配置模块权限，资产与容器范围受所属用户授权限制。仅管理员可分配和撤销 Key，用户停用后 Key 立即失效。"
        )
      }}</span>
    </div>
    <section class="key-surface">
      <div class="key-toolbar">
        <el-select
          v-if="canManage"
          v-model="ownerFilter"
          clearable
          filterable
          remote
          :remote-method="searchUsers"
          :placeholder="$tr('按所属用户筛选')"
          @change="
            page = 1;
            load();
          "
          ><el-option
            v-for="u in users"
            :key="u.id"
            :value="u.id"
            :label="`${u.username} · ${u.nickname || u.id}`" /></el-select
        ><el-input
          v-model="search"
          :prefix-icon="Search"
          :placeholder="$tr('搜索密钥名称')"
          clearable
        /><el-button :icon="Refresh" :loading="loading" @click="load">{{
          $tr("刷新")
        }}</el-button>
      </div>
      <el-table v-loading="loading" :data="filtered" row-key="id">
        <el-table-column :label="$tr('密钥')" min-width="220"
          ><template #default="{ row }"
            ><strong>{{ row.name }}</strong>
            <div class="key-muted key-mono">
              {{ row.prefix }}••••••••
            </div></template
          ></el-table-column
        >
        <el-table-column :label="$tr('所属用户')" min-width="150"
          ><template #default="{ row }"
            >{{ row.username || `#${row.userId}` }}
            <div class="key-muted">{{ row.nickname }}</div></template
          ></el-table-column
        >
        <el-table-column :label="$tr('权限范围')" min-width="260"
          ><template #default="{ row }">
            <el-tag
              v-for="scope in keyScopes(row).slice(0, 2)"
              :key="scope"
              size="small"
              effect="plain"
              >{{ scopeLabel(scope) }}</el-tag
            >
            <el-popover
              v-if="keyScopes(row).length > 2"
              trigger="click"
              :width="320"
            >
              <template #reference
                ><el-button link type="primary"
                  >+{{ keyScopes(row).length - 2 }}
                  {{ $tr("项权限") }}</el-button
                ></template
              >
              <div class="scope-popover">
                <el-tag
                  v-for="scope in keyScopes(row)"
                  :key="scope"
                  size="small"
                  effect="plain"
                  >{{ scopeLabel(scope) }}</el-tag
                >
              </div>
            </el-popover>
            <div v-if="row.grantMode !== 'scoped'" class="key-muted">
              {{ $tr("旧版 Key，保留原授权规则") }}
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="$tr('状态')" width="100"
          ><template #default="{ row }"
            ><el-tag :type="state(row) === '有效' ? 'success' : 'info'">{{
              $tr(state(row))
            }}</el-tag></template
          ></el-table-column
        >
        <el-table-column :label="$tr('到期时间')" min-width="170"
          ><template #default="{ row }">{{
            date(row.expiresAt)
          }}</template></el-table-column
        >
        <el-table-column :label="$tr('最近使用')" min-width="170"
          ><template #default="{ row }">{{
            date(row.lastUsedAt)
          }}</template></el-table-column
        >
        <el-table-column
          v-if="canManage"
          :label="$tr('操作')"
          width="110"
          fixed="right"
          ><template #default="{ row }"
            ><el-button
              link
              type="danger"
              :icon="Delete"
              :disabled="!!row.revokedAt"
              @click="revoke(row)"
              >{{ $tr("撤销") }}</el-button
            ></template
          ></el-table-column
        >
        <template #empty
          ><el-empty :description="$tr('暂无 Key，请联系管理员为该用户分配')"
        /></template>
      </el-table>
      <div class="key-count">
        <el-pagination
          v-model:current-page="page"
          :total="total"
          :page-size="50"
          layout="total, prev, pager, next"
          @current-change="load"
        />
      </div>
    </section>
    <el-dialog
      v-model="open"
      :title="$tr('分配 Key')"
      width="min(900px,94vw)"
      class="key-create-dialog"
      :close-on-press-escape="!saving"
      :show-close="!saving"
      :close-on-click-modal="!saving"
    >
      <el-form label-position="top" :disabled="saving">
        <div class="key-fields">
          <el-form-item :label="$tr('名称')" required
            ><el-input
              v-model="form.name"
              maxlength="100"
              :placeholder="$tr('例如：生产环境 Harness')"
          /></el-form-item>
          <el-form-item :label="$tr('所属用户')" required
            ><el-select
              v-model="form.userId"
              filterable
              remote
              :remote-method="searchUsers"
              :placeholder="$tr('搜索并选择用户')"
              @change="loadOwner"
              ><el-option
                v-for="u in users"
                :key="u.id"
                :value="u.id"
                :disabled="u.status !== 1"
                :label="`${u.username} · ${u.nickname || u.id}${u.status !== 1 ? ' (' + $tr('已停用') + ')' : ''}`" /></el-select
          ></el-form-item>
          <el-form-item :label="$tr('有效期（天）')"
            ><el-input-number v-model="form.days" :min="1" :max="365"
          /></el-form-item>
        </div>
        <el-alert
          v-if="ownerError"
          :title="ownerError"
          type="error"
          :closable="false"
        />
        <p v-if="form.userId && !ownerLoading && !ownerError" class="key-muted">
          {{
            $tr(
              ownerAdmin
                ? "该用户是管理员；此 Key 仍只允许已勾选的操作。"
                : "管理员专属操作仍要求所属用户具有管理员身份。"
            )
          }}
        </p>
        <el-form-item :label="$tr('权限范围')" required>
          <div v-loading="catalogLoading" class="scope-field">
            <p class="key-muted">
              {{
                $tr(
                  "勾选此 Key 可访问的模块和操作，未勾选的操作不予授权。资产、集群与命名空间沿用所属用户的资源授权。"
                )
              }}
            </p>
            <el-alert
              v-if="catalogError"
              :title="catalogError"
              type="error"
              :closable="false"
            />
            <el-button
              v-if="catalogError"
              :icon="Refresh"
              @click="loadCatalog"
              >{{ $tr("重新加载权限") }}</el-button
            >
            <KeyScopeSelector
              v-else
              v-model="form.scopes"
              :modules="modules"
              :disabled="saving || catalogLoading"
            />
          </div>
        </el-form-item>
      </el-form>
      <template #footer
        ><el-button :icon="Close" :disabled="saving" @click="open = false">{{
          $tr("取消")
        }}</el-button
        ><el-button
          type="primary"
          :icon="Key"
          :loading="saving"
          :disabled="
            !form.userId ||
            !form.name.trim() ||
            !form.scopes.length ||
            catalogLoading ||
            !!catalogError ||
            ownerLoading ||
            !!ownerError
          "
          @click="save"
          >{{ $tr("分配 Key") }}</el-button
        ></template
      >
    </el-dialog>
    <el-dialog
      :model-value="!!token"
      :title="$tr('保存已生成的 Key')"
      width="min(640px,94vw)"
      :close-on-click-modal="false"
      @close="token = ''"
      ><el-alert
        :title="$tr('Key 仅显示一次，请交付给所属用户或用于其授权的外部系统。')"
        type="success"
        :closable="false"
        show-icon
      /><el-input
        class="key-secret"
        :model-value="token"
        readonly
        type="textarea"
        :rows="3"
        :aria-label="$tr('新建 API Key')"
      />
      <p class="key-muted">
        {{ $tr("请求头格式") }}：<code
          >Authorization: Bearer &lt;API_KEY&gt;</code
        >
      </p>
      <template #footer
        ><el-button :icon="CopyDocument" @click="copy">{{
          $tr("复制密钥")
        }}</el-button
        ><el-button type="primary" :icon="Check" @click="token = ''">{{
          $tr("已保存")
        }}</el-button></template
      ></el-dialog
    >
  </div>
</template>
<style scoped>
.key-page {
  padding: 24px;
  color: var(--text-primary);
}
.key-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 20px;
  margin-bottom: 24px;
}
.key-eyebrow {
  font-size: 11px;
  letter-spacing: 2px;
  color: var(--text-secondary);
}
h1 {
  display: flex;
  gap: 12px;
  align-items: center;
  font-size: 26px;
  margin: 10px 0;
}
p {
  color: var(--text-secondary);
  line-height: 1.7;
}
.key-note {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 0;
  color: var(--text-regular);
  font-size: 13px;
}
.key-surface {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 20px;
}
.key-toolbar {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}
.key-toolbar .el-input,
.key-toolbar .el-select {
  max-width: 360px;
}
.key-muted {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 6px;
}
.key-mono,
.key-secret {
  font-family: monospace;
}
.key-secret {
  margin-top: 16px;
}
.el-tag {
  margin: 3px;
}
.key-count {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 24px;
}
@media (max-width: 700px) {
  .key-page {
    padding: 14px;
  }
  .key-header,
  .key-toolbar {
    align-items: stretch;
    flex-direction: column;
  }
  .key-surface {
    padding: 12px;
  }
}
</style>

<style>
.key-create-dialog .el-dialog__body {
  max-height: 65vh;
  overflow-y: auto;
}
.key-create-dialog .scope-field {
  width: 100%;
  min-width: 0;
}
.scope-popover {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  max-height: 360px;
  overflow-y: auto;
}
</style>

<style scoped>
.key-fields {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(0, 1fr) 150px;
  gap: 16px;
}
.key-fields .el-form-item {
  min-width: 0;
}
.key-fields .el-input-number {
  width: 100%;
}
@media (max-width: 700px) {
  .key-fields {
    grid-template-columns: 1fr;
    gap: 0;
  }
}
</style>
