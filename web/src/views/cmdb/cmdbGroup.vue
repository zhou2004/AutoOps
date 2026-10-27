<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox, type FormInstance } from "element-plus";
import {
  Search,
  Refresh,
  Plus,
  Edit,
  Delete,
  FolderOpened,
  Folder,
  Sort,
  Check,
  Close
} from "@element-plus/icons-vue";
import API from "@/api/cmdb";
import { tr } from "@/locales";

defineOptions({ name: "CmdbBusinessGroups" });
interface BusinessGroup {
  id: number;
  name: string;
  parentId: number;
  createTime?: string;
  hostCount: number;
  children?: BusinessGroup[];
  path?: string;
  depth?: number;
}
const DEFAULT_GROUP_ID = 1;
const groups = ref<BusinessGroup[]>([]);
const loading = ref(false),
  error = ref("");
const search = ref(""),
  expanded = ref(true),
  tableVersion = ref(0);
const dialog = ref(false),
  saving = ref(false);
const formRef = ref<FormInstance>();
const form = reactive({ id: 0, name: "", parentId: 0 });
const rules = {
  name: [
    {
      required: true,
      whitespace: true,
      message: tr("请输入分组名称"),
      trigger: "blur"
    }
  ]
};
function flatten(items: BusinessGroup[]): BusinessGroup[] {
  return items.flatMap(item => [item, ...flatten(item.children || [])]);
}
const flatGroups = computed(() => flatten(groups.value));
const hostCount = computed(() =>
  groups.value.reduce((total, group) => total + (group.hostCount || 0), 0)
);
const filtered = computed(() => {
  const keyword = search.value.trim().toLowerCase();
  function select(items: BusinessGroup[]): BusinessGroup[] {
    return items.flatMap(item => {
      if (item.name.toLowerCase().includes(keyword)) return [item];
      const children = select(item.children || []);
      return children.length ? [{ ...item, children }] : [];
    });
  }
  return keyword ? select(groups.value) : groups.value;
});
const shownCount = computed(() => flatten(filtered.value).length);
const parentOptions = computed(() => {
  const current = flatGroups.value.find(g => g.id === form.id);
  const excluded = new Set(current ? flatten([current]).map(g => g.id) : []);
  const mapOptions = (items: BusinessGroup[]): any[] =>
    items.map(g => ({
      id: g.id,
      name: g.name,
      disabled: g.id === DEFAULT_GROUP_ID || excluded.has(g.id),
      children: mapOptions(g.children || [])
    }));
  return [{ id: 0, name: tr("顶级分组"), children: mapOptions(groups.value) }];
});
function annotate(
  items: BusinessGroup[],
  parents: string[] = []
): BusinessGroup[] {
  return items.map(item => ({
    ...item,
    hostCount: item.hostCount || 0,
    depth: parents.length + 1,
    path: [...parents, item.name].join(" / "),
    children: annotate(item.children || [], [...parents, item.name])
  }));
}
async function load() {
  loading.value = true;
  error.value = "";
  try {
    const { data } = await API.getAllCmdbGroups();
    if (data.code !== 200)
      throw new Error(data.message || tr("读取业务分组失败"));
    groups.value = annotate(data.data || []);
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}
async function openForm(group?: BusinessGroup, parentId = 0) {
  Object.assign(
    form,
    group
      ? { id: group.id, name: group.name, parentId: group.parentId }
      : { id: 0, name: "", parentId }
  );
  dialog.value = true;
  await nextTick();
  formRef.value?.clearValidate();
}
async function save() {
  if (!(await formRef.value?.validate().catch(() => false))) return;
  saving.value = true;
  try {
    const payload = { name: form.name.trim(), parentId: form.parentId };
    const { data } = form.id
      ? await API.updateCmdbGroup({ ...payload, id: form.id })
      : await API.createCmdbGroup(payload);
    if (data.code !== 200) throw new Error(data.message || tr("保存分组失败"));
    dialog.value = false;
    await load();
    ElMessage.success(tr("业务分组已保存"));
  } catch (e) {
    ElMessage.error((e as Error).message);
  } finally {
    saving.value = false;
  }
}
async function remove(group: BusinessGroup) {
  try {
    await ElMessageBox.confirm(
      `${tr("确认删除业务分组")}「${group.name}」？`,
      tr("删除分组"),
      {
        type: "warning",
        confirmButtonText: tr("删除"),
        cancelButtonText: tr("取消")
      }
    );
  } catch {
    return;
  }
  try {
    const { data } = await API.deleteCmdbGroup(group.id);
    if (data.code !== 200) throw new Error(data.message || tr("删除分组失败"));
    await load();
    ElMessage.success(tr("删除分组成功"));
  } catch (e) {
    ElMessage.error((e as Error).message);
  }
}
function toggleExpand() {
  expanded.value = !expanded.value;
  tableVersion.value++;
}
onMounted(load);
</script>

<template>
  <div class="business-groups-page">
    <header class="business-header">
      <div>
        <span class="eyebrow">ASSET ORGANIZATION</span>
        <h1>{{ $tr("业务分组") }}</h1>
        <p>{{ $tr("按业务层级组织资产，为告警群组选择统一的业务线。") }}</p>
      </div>
      <el-button
        v-authority="['cmdb:group:add']"
        type="primary"
        :icon="Plus"
        @click="openForm()"
        >{{ $tr("新建分组") }}</el-button
      >
    </header>
    <div class="business-stats">
      <div>
        <span>{{ $tr("可见分组") }}</span
        ><strong>{{ flatGroups.length }}</strong
        ><small>{{ $tr("包含全部层级") }}</small>
      </div>
      <div>
        <span>{{ $tr("顶级分组") }}</span
        ><strong>{{ groups.length }}</strong
        ><small>{{ $tr("当前可见范围") }}</small>
      </div>
      <div>
        <span>{{ $tr("关联主机") }}</span
        ><strong>{{ hostCount }}</strong
        ><small>{{ $tr("按权限范围统计，包含子分组") }}</small>
      </div>
    </div>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      show-icon
      :closable="false"
    />
    <section class="business-surface">
      <div class="business-toolbar">
        <el-input
          v-model="search"
          :prefix-icon="Search"
          clearable
          :placeholder="$tr('搜索分组名称，保留上级层级')"
          :aria-label="$tr('搜索业务分组')"
        />
        <div class="toolbar-actions">
          <el-button :icon="Sort" @click="toggleExpand">{{
            $tr(expanded ? "全部折叠" : "全部展开")
          }}</el-button>
          <el-button :icon="Refresh" :loading="loading" @click="load">{{
            $tr("刷新")
          }}</el-button>
        </div>
      </div>
      <el-table
        :key="`${tableVersion}-${search.trim()}`"
        v-loading="loading"
        :data="filtered"
        row-key="id"
        :default-expand-all="!!search.trim() || expanded"
        :tree-props="{ children: 'children' }"
        class="business-table"
      >
        <el-table-column prop="name" :label="$tr('分组名称')" min-width="310">
          <template #default="{ row }"
            ><span class="group-identity"
              ><el-icon
                ><component
                  :is="row.children.length ? FolderOpened : Folder" /></el-icon
              ><strong>{{ row.name }}</strong
              ><el-tag
                v-if="row.id === DEFAULT_GROUP_ID"
                size="small"
                type="info"
                >{{ $tr("默认分组") }}</el-tag
              ></span
            ></template
          >
        </el-table-column>
        <el-table-column
          :label="$tr('分组路径')"
          min-width="230"
          show-overflow-tooltip
          ><template #default="{ row }"
            ><span class="group-path">{{ row.path }}</span></template
          ></el-table-column
        >
        <el-table-column :label="$tr('关联主机')" width="140" align="center"
          ><template #default="{ row }"
            ><el-tag effect="plain">{{ row.hostCount }}</el-tag></template
          ></el-table-column
        >
        <el-table-column
          prop="createTime"
          :label="$tr('创建时间')"
          min-width="180"
          ><template #default="{ row }">{{
            row.createTime || "—"
          }}</template></el-table-column
        >
        <el-table-column
          :label="$tr('操作')"
          width="148"
          fixed="right"
          align="center"
        >
          <template #default="{ row }"
            ><div v-if="row.id !== DEFAULT_GROUP_ID" class="group-row-actions">
              <el-tooltip :content="$tr('新建子分组')" placement="top"
                ><el-button
                  v-authority="['cmdb:group:add']"
                  class="action-icon"
                  :icon="Plus"
                  :aria-label="$tr('新建子分组')"
                  type="primary"
                  link
                  @click="openForm(undefined, row.id)"
              /></el-tooltip>
              <el-tooltip :content="$tr('编辑')" placement="top"
                ><el-button
                  v-authority="['cmdb:group:update']"
                  class="action-icon"
                  :icon="Edit"
                  :aria-label="$tr('编辑')"
                  type="primary"
                  link
                  @click="openForm(row)"
              /></el-tooltip>
              <el-tooltip :content="$tr('删除')" placement="top"
                ><el-button
                  v-authority="['cmdb:group:delete']"
                  class="action-icon"
                  :icon="Delete"
                  :aria-label="$tr('删除')"
                  type="danger"
                  link
                  @click="remove(row)"
              /></el-tooltip>
            </div>
            <span v-else class="group-path">{{
              $tr("系统保留")
            }}</span></template
          >
        </el-table-column>
        <template #empty
          ><el-empty
            :description="$tr(search ? '没有匹配的业务分组' : '暂无业务分组')"
            :image-size="90"
        /></template>
      </el-table>
      <div class="business-footer">
        {{ $tr("当前显示") }} {{ shownCount }}
        {{ $tr("个分组，主机数量包含子分组，不重复汇总。") }}
      </div>
    </section>
    <el-dialog
      v-model="dialog"
      :title="$tr(form.id ? '编辑业务分组' : '新增业务分组')"
      width="min(560px,94vw)"
      :close-on-click-modal="!saving"
      :close-on-press-escape="!saving"
      :show-close="!saving"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        :disabled="saving"
      >
        <el-form-item :label="$tr('分组名称')" prop="name"
          ><el-input
            v-model="form.name"
            maxlength="50"
            show-word-limit
            :placeholder="$tr('例如：支付业务、生产环境')"
        /></el-form-item>
        <el-form-item :label="$tr('父级分组')" prop="parentId"
          ><el-tree-select
            v-model="form.parentId"
            :data="parentOptions"
            :props="{ label: 'name', value: 'id' }"
            node-key="id"
            check-strictly
            filterable
            default-expand-all
            :render-after-expand="false"
            style="width: 100%"
        /></el-form-item>
        <p class="form-note">
          {{ $tr("选择顶级分组可移至根层级；不能选择自身或下级分组。") }}
        </p>
      </el-form>
      <template #footer
        ><el-button :icon="Close" :disabled="saving" @click="dialog = false">{{
          $tr("取消")
        }}</el-button
        ><el-button
          type="primary"
          :icon="Check"
          :loading="saving"
          @click="save"
          >{{ $tr("保存") }}</el-button
        ></template
      >
    </el-dialog>
  </div>
</template>

<style scoped>
.business-groups-page {
  color: var(--text-primary);
}
.business-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  margin-bottom: 22px;
}
.eyebrow {
  color: var(--text-secondary);
  font-size: 11px;
  letter-spacing: 2px;
}
h1 {
  margin: 8px 0;
}
.business-header p,
.form-note {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.7;
  margin: 0;
}
.business-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}
.business-stats > div {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--ui-radius-panel);
  padding: 18px 22px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.business-stats span {
  color: var(--text-regular);
  font-size: 13px;
}
.business-stats strong {
  font-size: 28px;
  font-weight: 600;
}
.business-stats small {
  color: var(--text-secondary);
  font-size: 12px;
}
.business-surface {
  margin-top: 16px;
  padding: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--ui-radius-panel);
}
.business-toolbar {
  display: flex;
  gap: 16px;
  justify-content: space-between;
  flex-wrap: wrap;
  margin-bottom: 20px;
}
.business-toolbar > .el-input {
  width: min(360px, 100%);
}
.toolbar-actions,
.group-row-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.group-row-actions {
  gap: 4px;
  justify-content: center;
}
.group-row-actions .el-button {
  margin: 0;
}
.group-identity {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  vertical-align: middle;
}
.group-identity > .el-icon {
  color: var(--primary);
  font-size: 18px;
}
.group-path,
.business-footer {
  color: var(--text-secondary);
  font-size: 13px;
}
.business-footer {
  padding-top: 18px;
}
@media (max-width: 700px) {
  .business-header {
    align-items: flex-start;
    flex-direction: column;
  }
  .business-stats {
    gap: 8px;
  }
  .business-stats > div {
    padding: 12px;
  }
  .business-stats strong {
    font-size: 24px;
  }
  .business-surface {
    padding: 12px;
  }
}
</style>
