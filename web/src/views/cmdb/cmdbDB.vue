<template>
  <div class="database-page">
    <section class="page-hero">
      <div>
        <p class="eyebrow">DATABASE SOURCES</p>
        <h1>{{ $tr("数据源管理") }}</h1>
        <p>
          {{
            $tr(
              "按业务分组管理 MySQL、PostgreSQL 与 Redis，从数据源进入数据库工作台。"
            )
          }}
        </p>
      </div>
      <div class="source-header-actions">
        <DatabaseEngineFilter v-model="query.type" numeric @change="search" />
        <el-button
          v-if="platformAccess.isAdmin"
          v-authority="['cmdb:db:add']"
          type="primary"
          size="large"
          @click="openCreate"
          >{{ $tr("新增数据源") }}</el-button
        >
      </div>
    </section>

    <section class="summary-grid">
      <article v-for="item in summary" :key="item.label">
        <span>{{ $tr(item.label) }}</span
        ><strong>{{ item.value }}</strong>
      </article>
    </section>

    <el-card shadow="never" class="asset-table-card">
      <div class="toolbar">
        <div class="filters">
          <el-input
            v-model="query.name"
            clearable
            :placeholder="$tr('按名称搜索')"
            @keyup.enter="search"
          />
          <el-tree-select
            class="source-group-filter"
            v-model="query.groupId"
            :data="groups"
            :props="{ label: 'name', value: 'id', children: 'children' }"
            check-strictly
            filterable
            clearable
            :placeholder="$tr('业务分组（含子分组）')"
          />
          <el-button type="primary" @click="search">{{ $tr("查询") }}</el-button
          ><el-button @click="resetSearch">{{ $tr("重置") }}</el-button>
        </div>
      </div>

      <el-table :data="rows" v-loading="loading" stripe>
        <el-table-column :label="$tr('数据源')" min-width="220"
          ><template #default="{ row }"
            ><div class="database-name">
              <DatabaseEngineIcon :engine="row.type" />
              <div>
                <strong :title="row.name">{{ row.name }}</strong
                ><small :title="row.description">{{
                  row.description || "暂无描述"
                }}</small>
              </div>
            </div></template
          ></el-table-column
        >
        <el-table-column :label="$tr('类型')" width="140"
          ><template #default="{ row }"
            ><el-tag :type="tagType(row.type)">{{
              typeName(row.type)
            }}</el-tag></template
          ></el-table-column
        >
        <el-table-column
          prop="groupName"
          :label="$tr('业务分组')"
          min-width="150"
        />
        <el-table-column :label="$tr('连接账号')" min-width="180"
          ><template #default="{ row }"
            ><span>{{ row.username || accountName(row.accountId) }}</span
            ><small class="cell-subtitle">{{
              row.address || accountAddress(row.accountId)
            }}</small></template
          ></el-table-column
        >
        <el-table-column :label="$tr('默认库 / DB')" min-width="150"
          ><template #default="{ row }">{{
            row.defaultDb || (row.type === 3 ? "0" : "未指定")
          }}</template></el-table-column
        >
        <el-table-column label="TLS" width="130"
          ><template #default="{ row }"
            ><el-tag
              effect="plain"
              :type="row.tlsMode === 'disable' ? 'info' : 'success'"
              >{{ row.tlsMode || "disable" }}</el-tag
            ></template
          ></el-table-column
        >
        <el-table-column
          :label="$tr('标签')"
          min-width="150"
          show-overflow-tooltip
          ><template #default="{ row }">{{
            row.tags || "-"
          }}</template></el-table-column
        >
        <el-table-column :label="$tr('连通性')" width="120"
          ><template #default="{ row }"
            ><el-tag :type="health[row.id]?.type || 'info'">{{
              health[row.id]?.text || "未检测"
            }}</el-tag></template
          ></el-table-column
        >
        <el-table-column :label="$tr('操作')" width="270" fixed="right"
          ><template #default="{ row }"
            ><el-button
              link
              type="primary"
              :disabled="row.retired"
              @click="openWorkbench(row)"
              >{{ row.retired ? $tr("已退役") : $tr("工作台") }}</el-button
            ><el-button
              link
              type="success"
              :loading="testingId === row.id"
              @click="testConnection(row)"
              >{{ $tr("测试") }}</el-button
            ><el-button
              v-authority="['cmdb:db:edit']"
              link
              type="warning"
              @click="openEdit(row)"
              >{{ $tr("编辑") }}</el-button
            ><el-button
              v-authority="['cmdb:db:delete']"
              link
              type="danger"
              @click="remove(row)"
              >{{ $tr("删除") }}</el-button
            ></template
          ></el-table-column
        >
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="query.page"
          v-model:page-size="query.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next"
          @size-change="
            query.page = 1;
            loadList();
          "
          @current-change="loadList"
        />
      </div>
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="form.id ? '编辑数据源' : $tr('新增数据源')"
      width="min(720px, 94vw)"
      destroy-on-close
    >
      <el-alert
        :title="
          $tr('连接账号由管理员配置；操作人员通过资产授权获得查询或执行权限。')
        "
        type="info"
        :closable="false"
        show-icon
      />
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="110px"
        class="source-form"
      >
        <el-row :gutter="18">
          <el-col :span="12"
            ><el-form-item :label="$tr('数据源名称')" prop="name"
              ><el-input v-model="form.name" maxlength="100" /></el-form-item
          ></el-col>
          <el-col :span="12"
            ><el-form-item :label="$tr('类型')" prop="type"
              ><el-select
                v-model="form.type"
                :disabled="!platformAccess.isAdmin"
                style="width: 100%"
                @change="handleTypeChange"
                ><el-option
                  v-for="item in dbTypes"
                  :key="item.value"
                  :label="$tr(item.label)"
                  :value="item.value" /></el-select></el-form-item
          ></el-col>
          <el-col :span="12"
            ><el-form-item :label="$tr('连接账号')" prop="accountId"
              ><el-select
                v-model="form.accountId"
                :disabled="!platformAccess.isAdmin"
                filterable
                style="width: 100%"
                :placeholder="$tr('选择同类型账号')"
                ><el-option
                  v-for="item in filteredAccounts"
                  :key="item.id"
                  :label="`${item.alias} · ${item.host}:${item.port}`"
                  :value="item.id" /></el-select></el-form-item
          ></el-col>
          <el-col :span="12"
            ><el-form-item :label="$tr('业务分组')" prop="groupId"
              ><el-cascader
                v-model="form.groupId"
                :options="groups"
                :props="groupProps"
                clearable
                filterable
                style="width: 100%" /></el-form-item
          ></el-col>
          <el-col :span="12"
            ><el-form-item :label="form.type === 3 ? 'Redis DB' : '默认数据库'"
              ><el-input
                v-model="form.defaultDb"
                :placeholder="
                  form.type === 3 ? '0 - 1024' : '可留空'
                " /></el-form-item
          ></el-col>
          <el-col :span="12"
            ><el-form-item :label="$tr('TLS 模式')" prop="tlsMode"
              ><el-select v-model="form.tlsMode" style="width: 100%"
                ><el-option
                  v-for="item in tlsOptions"
                  :key="item"
                  :label="$tr(tlsLabel(item))"
                  :value="item" /></el-select></el-form-item
          ></el-col>
          <el-col :span="24"
            ><el-form-item :label="$tr('标签')"
              ><el-input
                v-model="form.tags"
                :placeholder="$tr('例如 production,core')" /></el-form-item
          ></el-col>
          <el-col :span="24"
            ><el-form-item :label="$tr('描述')"
              ><el-input
                v-model="form.description"
                type="textarea"
                :rows="3"
                maxlength="500"
                show-word-limit /></el-form-item
          ></el-col>
        </el-row>
      </el-form>
      <template #footer
        ><el-button @click="dialogVisible = false">{{ $tr("取消") }}</el-button
        ><el-button type="primary" :loading="saving" @click="save">{{
          $tr("保存")
        }}</el-button></template
      >
    </el-dialog>
  </div>
</template>

<script setup>
import DatabaseEngineFilter from "@/views/database/components/DatabaseEngineFilter.vue";
import DatabaseEngineIcon from "@/views/database/components/DatabaseEngineIcon.vue";
import { tr } from "@/locales";
import { platformAccess } from "@/utils/platformAccess";

import { computed, onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import cmdbAPI from "@/api/cmdb";
import configAPI from "@/api/config";

const router = useRouter();
const dbTypes = [
  { label: "MySQL", value: 1 },
  { label: "PostgreSQL", value: 2 },
  { label: "Redis", value: 3 }
];
const loading = ref(false),
  saving = ref(false),
  dialogVisible = ref(false),
  testingId = ref(0),
  total = ref(0),
  formRef = ref();
const rows = ref([]),
  accounts = ref([]),
  groups = ref([]),
  health = reactive({});
const query = reactive({
  name: "",
  type: undefined,
  groupId: undefined,
  page: 1,
  pageSize: 10
});
const emptyForm = () => ({
  id: undefined,
  name: "",
  type: 1,
  accountId: undefined,
  groupId: undefined,
  defaultDb: "",
  tlsMode: "disable",
  tags: "",
  description: ""
});
const form = reactive(emptyForm());
const rules = {
  name: [
    {
      required: true,
      get message() {
        return tr("请输入数据源名称");
      },
      trigger: "blur"
    }
  ],
  type: [
    {
      required: true,
      get message() {
        return tr("请选择类型");
      },
      trigger: "change"
    }
  ],
  accountId: [
    {
      required: true,
      get message() {
        return tr("请选择连接账号");
      },
      trigger: "change"
    }
  ],
  groupId: [
    {
      required: true,
      get message() {
        return tr("请选择业务分组");
      },
      trigger: "change"
    }
  ],
  tlsMode: [
    {
      required: true,
      get message() {
        return tr("请选择 TLS 模式");
      },
      trigger: "change"
    }
  ]
};
const groupProps = {
  checkStrictly: true,
  emitPath: false,
  value: "id",
  label: "name",
  children: "children"
};
const filteredAccounts = computed(() =>
  accounts.value.filter(item => Number(item.type) === Number(form.type))
);
const tlsOptions = computed(() =>
  form.type === 2
    ? ["disable", "require", "verify-ca", "verify-full"]
    : ["disable", "verify-full"]
);
const summary = computed(() => [
  { label: "匹配数据源", value: total.value },
  ...dbTypes.map(type => ({
    label: type.label + "（本页）",
    value: rows.value.filter(row => row.type === type.value).length
  }))
]);

function unpack(response) {
  if (response.data?.code !== 200)
    throw new Error(response.data?.message || tr("操作失败"));
  return response.data.data;
}
function typeName(type) {
  return dbTypes.find(item => item.value === Number(type))?.label || "未知";
}
function tagType(type) {
  return { 1: "success", 2: "primary", 3: "danger" }[type] || "info";
}
function tlsLabel(value) {
  return (
    {
      disable: "关闭",
      require: "加密（不校验证书）",
      "verify-ca": "校验 CA",
      "verify-full": "完整校验"
    }[value] || value
  );
}
function account(id) {
  return accounts.value.find(item => Number(item.id) === Number(id));
}
function accountName(id) {
  return account(id)?.alias || `账号 #${id}`;
}
function accountAddress(id) {
  const item = account(id);
  return item ? `${item.host}:${item.port}` : "账号不存在";
}

async function loadDependencies() {
  const [accountResponse, groupResponse] = await Promise.all([
    platformAccess.isAdmin
      ? configAPI.listAccountAuth({ page: 1, pageSize: 100 })
      : Promise.resolve({ data: { code: 200, data: { list: [] } } }),
    cmdbAPI.getDatabaseGroups()
  ]);
  accounts.value = unpack(accountResponse)?.list || [];
  groups.value = unpack(groupResponse) || [];
}
async function loadList() {
  loading.value = true;
  try {
    const data = unpack(
      await cmdbAPI.listDatabases({ ...query, name: query.name.trim() })
    );
    rows.value = data.list || [];
    total.value = Number(data.total || 0);
  } catch (error) {
    ElMessage.error(error.message);
  } finally {
    loading.value = false;
  }
}
function search() {
  query.page = 1;
  loadList();
}
function resetSearch() {
  query.name = "";
  query.type = undefined;
  query.groupId = undefined;
  query.page = 1;
  loadList();
}
function resetForm(value = {}) {
  Object.assign(form, emptyForm(), value);
}
function openCreate() {
  resetForm();
  dialogVisible.value = true;
}
function openEdit(row) {
  resetForm({
    ...row,
    id: Number(row.id),
    type: Number(row.type),
    accountId: Number(row.accountId),
    groupId: Number(row.groupId),
    tlsMode: row.tlsMode || "disable"
  });
  dialogVisible.value = true;
}
function handleTypeChange() {
  form.accountId = undefined;
  if (!tlsOptions.value.includes(form.tlsMode)) form.tlsMode = "disable";
  form.defaultDb = form.type === 3 ? "0" : "";
}
async function save() {
  try {
    await formRef.value.validate();
  } catch {
    return;
  }
  saving.value = true;
  try {
    const payload = {
      ...form,
      name: form.name.trim(),
      type: Number(form.type),
      accountId: Number(form.accountId),
      groupId: Number(form.groupId),
      defaultDb: form.defaultDb.trim(),
      tags: form.tags.trim(),
      description: form.description.trim()
    };
    unpack(
      await (payload.id
        ? cmdbAPI.updateDatabase(payload)
        : cmdbAPI.createDatabase(payload))
    );
    ElMessage.success(payload.id ? "数据源已更新" : "数据源已创建");
    dialogVisible.value = false;
    await loadList();
  } catch (error) {
    ElMessage.error(error.message);
  } finally {
    saving.value = false;
  }
}
async function remove(row) {
  try {
    await ElMessageBox.confirm(
      `确定删除数据源“${row.name}”？`,
      tr("删除确认"),
      {
        type: "warning"
      }
    );
    unpack(await cmdbAPI.deleteDatabase(row.id));
    ElMessage.success(tr("数据源已删除"));
    await loadList();
  } catch (error) {
    if (error !== "cancel" && error !== "close" && error?.message)
      ElMessage.error(error.message);
  }
}
async function testConnection(row) {
  testingId.value = row.id;
  try {
    const info = unpack(await cmdbAPI.getDatabaseOverview(row.id));
    health[row.id] = { type: "success", text: `${info.latencyMs} ms` };
    ElMessage.success(`${row.name} 连接正常`);
  } catch (error) {
    health[row.id] = { type: "danger", text: "失败" };
    ElMessage.error(error.message);
  } finally {
    testingId.value = 0;
  }
}
function openWorkbench(row) {
  router.push({ path: "/database/workbench", query: { id: row.id } });
}
onMounted(async () => {
  try {
    await loadDependencies();
    await loadList();
  } catch (error) {
    ElMessage.error(error.message);
  }
});
</script>

<style scoped>
.database-page :deep(.el-table__row > td) {
  height: 64px;
  padding: 7px 0;
  box-sizing: border-box;
}
.database-page :deep(.el-table__row > td .cell) {
  max-height: 48px;
  overflow: hidden;
  line-height: 20px;
}
:deep(.database-engine-icon) {
  width: 26px;
  height: 26px;
  flex: 0 0 26px;
}
.database-name {
  height: 44px;
  min-width: 0;
}
.database-name > div {
  min-width: 0;
}
.database-name strong,
.database-name small {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.filters .source-group-filter {
  width: 230px;
}
.database-page {
  padding: 22px;
}
.source-header-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  flex-wrap: wrap;
}
.page-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 26px 30px;
  border: 1px solid var(--border-light);
  border-radius: var(--radius-lg);
  background: linear-gradient(135deg, var(--bg-card), var(--bg-card-alt));
}
.page-hero h1 {
  margin: 2px 0 8px;
  color: var(--text-primary);
  font-size: 28px;
}
.page-hero p {
  margin: 0;
  color: var(--text-secondary);
}
.eyebrow {
  font-size: 11px !important;
  font-weight: 700;
  letter-spacing: 0.16em;
  color: var(--primary) !important;
}
.summary-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
  margin: 16px 0;
}
.summary-grid article {
  padding: 17px 20px;
  border: 1px solid var(--border-light);
  border-radius: var(--radius);
  background: var(--bg-card);
}
.summary-grid span,
.cell-subtitle {
  display: block;
  color: var(--text-secondary);
  font-size: 12px;
}
.summary-grid strong {
  display: block;
  margin-top: 6px;
  color: var(--text-primary);
  font-size: 23px;
}
.toolbar {
  display: flex;
  justify-content: space-between;
  margin-bottom: 16px;
}
.filters {
  display: flex;
  gap: 10px;
}
.filters .el-input {
  width: 230px;
}
.filters .el-select {
  width: 170px;
}
.database-name {
  display: flex;
  align-items: center;
  gap: 12px;
}
.database-name strong,
.database-name small {
  display: block;
}
.database-name small {
  max-width: 220px;
  margin-top: 3px;
  overflow: hidden;
  color: var(--text-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 18px;
}
.source-form {
  margin-top: 20px;
}
@media (max-width: 900px) {
  .summary-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .source-header-actions {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 12px;
    flex-wrap: wrap;
  }
  .page-hero {
    align-items: flex-start;
    gap: 18px;
  }
  .filters {
    flex-wrap: wrap;
  }
}
</style>
