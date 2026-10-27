<template>
  <div class="category-tools">
    <el-button
      @click="
        visible = true;
        load();
      "
      >{{ $tr("分类管理") }}</el-button
    >
    <el-button
      :disabled="!ids.length || loading"
      @click="assignVisible = true"
      >{{ $tr("批量归类") }}</el-button
    >
  </div>
  <el-drawer
    v-model="visible"
    class="monitor-overlay"
    :title="$tr('分类管理')"
    size="min(620px, 100vw)"
  >
    <p class="category-hint">
      {{ $tr("按业务或环境整理监控目标，删除分类后目标转入未分类。") }}
    </p>
    <el-form
      ref="formRef"
      :model="form"
      label-position="top"
      @submit.prevent="save"
    >
      <el-form-item
        :label="$tr('分类名称')"
        prop="name"
        :rules="[
          {
            required: true,
            whitespace: true,
            message: $tr('请输入分类名称'),
            trigger: 'blur'
          }
        ]"
      >
        <el-input v-model="form.name" :maxlength="64" show-word-limit />
      </el-form-item>
      <el-form-item :label="$tr('说明')"
        ><el-input v-model="form.description" :maxlength="255"
      /></el-form-item>
      <el-button type="primary" :loading="saving" native-type="submit">{{
        form.id ? $tr("保存分类") : $tr("新增分类")
      }}</el-button>
      <el-button v-if="form.id" @click="resetForm">{{
        $tr("取消编辑")
      }}</el-button>
    </el-form>
    <el-input
      v-model="keyword"
      class="category-search"
      :placeholder="$tr('搜索分类名称')"
      clearable
    />
    <el-table v-loading="loading" :data="filteredCategories" row-key="id">
      <el-table-column :label="$tr('分类名称')" min-width="150">
        <template #default="{ row }"
          ><strong>{{ row.name }}</strong>
          <p class="category-hint">{{ row.description || "—" }}</p></template
        >
      </el-table-column>
      <el-table-column :label="$tr('操作')" width="130">
        <template #default="{ row }">
          <el-button
            link
            type="primary"
            :disabled="saving"
            @click="form = { ...row }"
            >{{ $tr("编辑") }}</el-button
          >
          <el-button
            link
            type="danger"
            :disabled="saving"
            @click="remove(row)"
            >{{ $tr("删除") }}</el-button
          >
        </template>
      </el-table-column>
      <template #empty
        ><el-empty :description="$tr('暂无匹配分类，可在上方新增分类')"
      /></template>
    </el-table>
  </el-drawer>
  <el-dialog
    v-model="assignVisible"
    class="monitor-overlay"
    :title="$tr('批量归类')"
    width="min(440px, 94vw)"
  >
    <p class="category-hint">
      {{ $tr("已选择") }} {{ ids.length }} {{ $tr("个监控目标") }}
    </p>
    <el-select
      v-model="categoryId"
      filterable
      :aria-label="$tr('目标分类')"
      style="width: 100%"
    >
      <el-option :value="0" :label="$tr('未分类')" />
      <el-option
        v-for="category in categories"
        :key="category.id"
        :value="category.id"
        :label="category.name"
      />
    </el-select>
    <template #footer>
      <el-button @click="assignVisible = false">{{ $tr("取消") }}</el-button>
      <el-button
        type="primary"
        :loading="saving"
        :disabled="!ids.length"
        @click="assign"
        >{{ $tr("确定") }}</el-button
      >
    </template>
  </el-dialog>
</template>
<script setup>
import { computed, onMounted, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { tr } from "@/locales";
import {
  getMonitorCategories,
  saveMonitorCategory,
  deleteMonitorCategory,
  assignMonitorCategory
} from "@/api/monitor";
const props = defineProps({
  scope: { type: String, required: true },
  ids: { type: Array, default: () => [] }
});
const emit = defineEmits(["loaded", "changed"]);
const visible = ref(false),
  assignVisible = ref(false),
  loading = ref(false),
  saving = ref(false);
const categories = ref([]),
  keyword = ref(""),
  categoryId = ref(0),
  formRef = ref();
const form = ref({ id: 0, name: "", description: "" });
const filteredCategories = computed(() =>
  categories.value.filter(c =>
    c.name.toLowerCase().includes(keyword.value.trim().toLowerCase())
  )
);
const resetForm = () => {
  form.value = { id: 0, name: "", description: "" };
  formRef.value?.clearValidate();
};
async function load() {
  loading.value = true;
  try {
    const { data: res } = await getMonitorCategories(props.scope);
    if (res.code !== 200) throw new Error(res.message);
    categories.value = res.data || [];
    emit("loaded", categories.value);
    if (!categories.value.some(c => c.id === categoryId.value))
      categoryId.value = 0;
  } catch (e) {
    ElMessage.error(e.message || tr("获取分类失败"));
  } finally {
    loading.value = false;
  }
}
async function save() {
  if (!(await formRef.value.validate().catch(() => false))) return;
  saving.value = true;
  try {
    const { data: res } = await saveMonitorCategory({
      ...form.value,
      name: form.value.name.trim(),
      scope: props.scope
    });
    if (res.code !== 200) throw new Error(res.message);
    ElMessage.success(tr("保存成功"));
    resetForm();
    await load();
    emit("changed");
  } catch (e) {
    ElMessage.error(e.message || tr("保存失败"));
  } finally {
    saving.value = false;
  }
}
async function remove(row) {
  if (
    !(await ElMessageBox.confirm(
      tr("删除分类后，分类下的监控目标将转入未分类。是否继续？"),
      tr("删除分类"),
      { type: "warning" }
    )
      .then(() => true)
      .catch(() => false))
  )
    return;
  saving.value = true;
  try {
    const { data: res } = await deleteMonitorCategory(row.id, props.scope);
    if (res.code !== 200) throw new Error(res.message);
    if (form.value.id === row.id) resetForm();
    await load();
    emit("changed");
    ElMessage.success(tr("删除成功"));
  } catch (e) {
    ElMessage.error(e.message || tr("删除失败"));
  } finally {
    saving.value = false;
  }
}
async function assign() {
  saving.value = true;
  try {
    const { data: res } = await assignMonitorCategory({
      scope: props.scope,
      ids: props.ids,
      categoryId: categoryId.value
    });
    if (res.code !== 200) throw new Error(res.message);
    assignVisible.value = false;
    emit("changed");
    ElMessage.success(tr("归类成功"));
  } catch (e) {
    ElMessage.error(e.message || tr("归类失败"));
  } finally {
    saving.value = false;
  }
}
onMounted(load);
</script>
<style scoped>
.category-tools {
  display: inline-flex;
  gap: 8px;
  flex-wrap: wrap;
}
.category-tools :deep(.el-button) {
  margin: 0;
}
.category-hint {
  color: var(--text-secondary);
  margin: 4px 0 16px;
  font-size: 13px;
  overflow-wrap: anywhere;
}
.category-search {
  margin: 24px 0 12px;
}
</style>
