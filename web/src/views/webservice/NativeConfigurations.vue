<template>
  <section class="native-configs">
    <div class="native-toolbar">
      <el-select
        :model-value="instanceId"
        :disabled="busy"
        filterable
        :placeholder="$tr('选择实例')"
        @update:model-value="selectInstance"
      >
        <el-option
          v-for="instance in instanceChoices"
          :key="instance.id"
          :value="instance.id"
          :label="instance.name"
        />
      </el-select>
      <el-button
        :loading="loading"
        :disabled="!instanceId || busy"
        @click="refresh"
        >{{ $tr("读取远端配置") }}</el-button
      >
      <el-button
        v-if="canWrite"
        :disabled="busy || !loaded || !selectedInstance?.clusterId"
        @click="openAdd"
        >{{ $tr("添加文件") }}</el-button
      >
      <span class="muted" v-if="loaded">{{ loaded.version }}</span>
    </div>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      show-icon
    />
    <el-alert
      v-if="loaded && !loaded.clusterId"
      :title="$tr('请先将实例加入集群并完成接管，再修改配置')"
      type="warning"
      :closable="false"
    />
    <template v-if="loaded">
      <p class="muted config-entry">
        {{ $tr("主配置入口") }}：{{ loaded.configPath }}
      </p>
      <div class="native-toolbar">
        <el-input
          v-model="search"
          clearable
          :placeholder="$tr('搜索文件路径或配置内容')"
        />
        <el-select
          v-model="groupFilter"
          clearable
          :placeholder="$tr('配置分组')"
        >
          <el-option :value="0" :label="$tr('未分组')" />
          <el-option
            v-for="group in instanceGroups"
            :key="group.id"
            :value="group.id"
            :label="group.name"
          />
        </el-select>
        <el-checkbox v-model="changedOnly">{{
          $tr("仅显示已修改")
        }}</el-checkbox>
        <span class="muted">{{
          $tr("共 {value0} 个文件", { value0: files.length })
        }}</span>
      </div>
      <div class="config-workspace" v-loading="loading">
        <aside class="config-tree" :aria-label="$tr('配置文件')">
          <div class="file-list-heading">
            <strong>{{ $tr("配置文件") }}</strong>
            <el-tag size="small" type="info">{{ visibleFiles.length }}</el-tag>
          </div>
          <div ref="fileList" class="file-list">
            <button
              v-for="file in pagedFiles"
              :key="file.path"
              type="button"
              class="file-row"
              :class="{
                active: activePath === file.path,
                deleted: file.deleted
              }"
              :aria-pressed="activePath === file.path"
              @click="activePath = file.path"
            >
              <span class="file-title"
                ><el-icon><Document /></el-icon
                ><span class="file-label" :title="file.path">{{
                  relativePath(file.path)
                }}</span></span
              >
              <small v-if="fileGroup(file.path)" class="muted">{{
                fileGroup(file.path)?.name
              }}</small>
              <el-tag v-if="file.deleted" size="small" type="danger">{{
                $tr("待删除")
              }}</el-tag>
              <el-tag v-else-if="changed(file)" size="small" type="warning">{{
                $tr("已修改")
              }}</el-tag>
              <el-tag
                v-else-if="file.included === false"
                size="small"
                type="info"
                >{{ $tr("未被引用") }}</el-tag
              >
            </button>
            <el-empty
              v-if="!visibleFiles.length"
              :description="$tr('没有匹配的配置文件')"
              :image-size="64"
            />
          </div>
          <div class="file-pagination">
            <el-pagination
              :current-page="filePage"
              :page-size="filePageSize"
              :page-sizes="[10, 20, 50, 100]"
              :total="visibleFiles.length"
              :pager-count="5"
              size="small"
              layout="sizes, prev, pager, next, jumper"
              @current-change="changeFilePage"
              @size-change="changeFilePageSize"
            />
          </div>
        </aside>
        <main class="config-editor" v-if="activeFile">
          <div class="editor-heading">
            <code>{{ activeFile.path }}</code>
            <el-button
              v-if="canWrite && editable(activeFile)"
              :type="activeFile.deleted ? 'primary' : 'danger'"
              link
              :disabled="busy || !!deletingPath"
              :loading="deletingPath === activeFile.path"
              @click="toggleDelete(activeFile)"
            >
              {{ $tr(activeFile.deleted ? "撤销删除" : "删除文件") }}
            </el-button>
          </div>
          <el-alert
            v-if="!editable(activeFile)"
            :title="$tr('文件位于受管范围之外，仅供查看')"
            type="info"
            :closable="false"
          />
          <el-alert
            v-if="activeFile.deleted"
            :title="$tr('该文件将在校验通过并确认发布后删除')"
            type="warning"
            :closable="false"
          />
          <el-input
            v-model="activeFile.content"
            type="textarea"
            :rows="23"
            class="native-code"
            spellcheck="false"
            :readonly="busy || !canWrite || !editable(activeFile)"
            :disabled="!!activeFile.deleted"
            :aria-label="$tr('原生 Nginx 配置')"
            @keydown="handleEditorKeydown"
          />
          <p class="editor-shortcuts muted">
            {{
              $tr(
                "Tab 缩进 · Shift+Tab 反缩进 · Ctrl/⌘+S 校验保存 · Esc 后按 Tab 离开编辑器"
              )
            }}
          </p>
        </main>
        <el-empty
          v-else
          class="config-editor"
          :description="$tr('选择配置文件开始查看或编辑')"
        />
      </div>
      <div class="native-toolbar" v-if="canWrite">
        <el-select
          v-model="draftGroupId"
          :disabled="busy"
          clearable
          :placeholder="$tr('保存到配置分组')"
        >
          <el-option
            v-for="group in instanceGroups"
            :key="group.id"
            :value="group.id"
            :label="group.name"
          />
        </el-select>
        <el-select
          v-model="targetScope"
          :disabled="busy || !selectedInstance?.clusterId"
          :aria-label="$tr('发布范围')"
        >
          <el-option value="instance" :label="$tr('仅当前实例')" />
          <el-option value="cluster" :label="$tr('整个集群')" />
        </el-select>
        <el-button
          :disabled="!changes.length || !loaded.clusterId || busy"
          :loading="validating"
          @click="validate"
          >{{ $tr("校验语法") }}</el-button
        >
        <el-button
          :disabled="!changes.length || !loaded.clusterId || busy"
          :loading="saving"
          @click="save(false)"
          >{{ $tr("校验并保存草稿") }}</el-button
        >
        <el-button
          v-if="canExecute"
          type="primary"
          :disabled="
            !changes.length ||
            !loaded.clusterId ||
            busy ||
            loaded.management !== 'operate'
          "
          :loading="saving"
          @click="save(true)"
          >{{ $tr("校验并发布") }}</el-button
        >
      </div>
      <p class="muted" v-if="changes.length">
        {{
          $tr(
            "变更 {value0} 个文件，发布范围：{value1}；确认发布前不会修改远端文件",
            {
              value0: changes.length,
              value1: $tr(targetScope === "cluster" ? "整个集群" : "仅当前实例")
            }
          )
        }}
      </p>
      <el-alert
        v-if="results.length"
        :type="results.every(v => v.valid) ? 'success' : 'error'"
        :title="
          $tr(
            results.every(v => v.valid)
              ? 'Nginx 语法校验通过'
              : 'Nginx 语法校验未通过，配置不会被更新'
          )
        "
        :closable="false"
      >
        <pre
          v-for="result in results"
          :key="result.instanceId"
          class="validation-output"
          >{{ result.name }}: {{ result.message || $tr("通过") }}</pre
        >
      </el-alert>
    </template>
    <el-empty
      v-else-if="!loading"
      :description="$tr('选择实例读取 nginx.conf 和 conf.d 原生配置')"
    />
    <el-dialog
      v-model="adding"
      :title="$tr('添加配置文件')"
      width="min(680px, 94vw)"
    >
      <el-input
        v-model="newPath"
        :placeholder="$tr('配置文件绝对路径')"
        @keyup.enter="addFile"
      />
      <p class="muted">
        {{ $tr("新增文件也会与完整配置一起校验，通过后才能发布") }}
      </p>
      <template #footer
        ><el-button @click="adding = false">{{ $tr("取消") }}</el-button
        ><el-button type="primary" @click="addFile">{{
          $tr("添加")
        }}</el-button></template
      >
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, watch } from "vue";
import { onBeforeRouteLeave } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import { tr } from "@/locales";
import { canPlatform } from "@/utils/platformAccess";
import Document from "~icons/ep/document";
import { indentConfiguration } from "./nativeEditorIndent";
import {
  webInstanceConfig,
  webRequest,
  webValidateClusterConfig,
  type WebConfigFile,
  type WebConfigValidation,
  type WebInstanceConfig,
  type WebResource
} from "@/api/webservice";

const props = defineProps<{
  instances: WebResource[];
  allInstances: WebResource[];
  configGroups: WebResource[];
  initialInstanceId?: number;
  refreshKey: number;
}>();
const emit = defineEmits<{ saved: [draft: WebResource, publish: boolean] }>();
const instanceId = ref<number>();
const loaded = ref<WebInstanceConfig>();
const files = ref<WebConfigFile[]>([]);
const originals = ref<Record<string, WebConfigFile>>({});
const activePath = ref("");
const search = ref("");
const groupFilter = ref<number>();
const changedOnly = ref(false);
const filePage = ref(1),
  filePageSize = ref(20);
const fileList = ref<HTMLDivElement>();
const deletingPath = ref("");
let allowFocusTab = false;
const targetScope = ref<"instance" | "cluster">("instance");
const draftGroupId = ref<number>();
const loading = ref(false),
  validating = ref(false),
  saving = ref(false),
  error = ref("");
const adding = ref(false),
  newPath = ref("");
const results = ref<WebConfigValidation[]>([]);
let generation = 0,
  validationGeneration = 0;
let abort: AbortController | undefined;
let savedSignature = "";
let savedDraft: WebResource | undefined;
const selectedInstance = computed(() =>
  props.allInstances.find(i => i.id === instanceId.value)
);
const canWrite = computed(() => canPlatform("web:write"));
const canExecute = computed(() => canPlatform("web:execute"));
const busy = computed(() => loading.value || saving.value || validating.value);
const instanceGroups = computed(() =>
  props.configGroups.filter(
    g => g.clusterId === selectedInstance.value?.clusterId
  )
);
const targetIds = computed(() =>
  targetScope.value === "cluster"
    ? []
    : instanceId.value
      ? [instanceId.value]
      : []
);
const activeFile = computed(() =>
  files.value.find(f => f.path === activePath.value)
);
function changed(file: WebConfigFile) {
  const old = originals.value[file.path];
  return (
    !!file.deleted ||
    !old ||
    old.content !== file.content ||
    old.mode !== file.mode
  );
}
const changes = computed(() =>
  files.value
    .filter(f => changed(f) && editable(f))
    .map(f => ({
      path: f.path,
      content: f.content,
      deleted: f.deleted,
      mode: f.mode || 420,
      group: f.group
    }))
);
const instanceChoices = computed(() =>
  selectedInstance.value &&
  !props.instances.some(i => i.id === instanceId.value)
    ? [selectedInstance.value, ...props.instances]
    : props.instances
);
const signature = computed(() =>
  JSON.stringify({
    files: changes.value,
    targets: targetIds.value,
    group: draftGroupId.value,
    digest: loaded.value?.digest
  })
);
function fileGroup(path: string) {
  return instanceGroups.value.find(g => (g.spec.paths || []).includes(path));
}
const visibleFiles = computed(() =>
  files.value.filter(
    f =>
      (!changedOnly.value || changed(f)) &&
      (groupFilter.value == null ||
        (fileGroup(f.path)?.id || 0) === groupFilter.value) &&
      (!search.value ||
        f.path.includes(search.value) ||
        f.content.includes(search.value))
  )
);
const pagedFiles = computed(() =>
  visibleFiles.value.slice(
    (filePage.value - 1) * filePageSize.value,
    filePage.value * filePageSize.value
  )
);
function changeFilePage(page: number) {
  filePage.value = page;
  fileList.value?.scrollTo({ top: 0 });
}
function changeFilePageSize(size: number) {
  filePageSize.value = size;
  changeFilePage(1);
}
watch([search, groupFilter, changedOnly], () => changeFilePage(1));
watch(
  () => visibleFiles.value.length,
  total => {
    if (filePage.value > Math.max(1, Math.ceil(total / filePageSize.value)))
      changeFilePage(Math.max(1, Math.ceil(total / filePageSize.value)));
  }
);
watch(activePath, () => {
  allowFocusTab = false;
});

async function handleEditorKeydown(event: KeyboardEvent) {
  const textarea = event.target;
  if (!(textarea instanceof HTMLTextAreaElement) || event.isComposing) return;
  if (event.key === "Escape") {
    allowFocusTab = true;
    return;
  }
  if (event.key === "Tab" && allowFocusTab) {
    allowFocusTab = false;
    return;
  }
  allowFocusTab = false;
  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "s") {
    event.preventDefault();
    if (
      canWrite.value &&
      !busy.value &&
      activeFile.value &&
      editable(activeFile.value)
    )
      await save(false);
    return;
  }
  const file = activeFile.value;
  if (
    event.key !== "Tab" ||
    event.ctrlKey ||
    event.altKey ||
    event.metaKey ||
    busy.value ||
    !canWrite.value ||
    !file ||
    !editable(file) ||
    file.deleted
  )
    return;
  event.preventDefault();
  const edit = indentConfiguration(
    textarea.value,
    textarea.selectionStart,
    textarea.selectionEnd,
    event.shiftKey
  );
  if (!edit) return;
  const value =
    textarea.value.slice(0, edit.from) +
    edit.text +
    textarea.value.slice(edit.to);
  const direction = textarea.selectionDirection;
  const top = textarea.scrollTop,
    left = textarea.scrollLeft;
  textarea.setSelectionRange(edit.from, edit.to);
  // insertText keeps the browser's Ctrl/Cmd+Z undo stack. setRangeText is the
  // fallback for browsers that do not provide the editing command.
  try {
    document.execCommand("insertText", false, edit.text);
  } catch {
    /* fallback below */
  }
  if (textarea.value !== value) {
    textarea.value = value;
    textarea.dispatchEvent(new Event("input", { bubbles: true }));
  }
  file.content = value;
  await nextTick();
  if (activeFile.value !== file) return;
  textarea.setSelectionRange(edit.selectionStart, edit.selectionEnd, direction);
  textarea.scrollTop = top;
  textarea.scrollLeft = left;
}
function relativePath(path: string) {
  const root = loaded.value?.configPath.replace(/\/[^/]+$/, "") || "";
  return path.startsWith(root + "/") ? path.slice(root.length + 1) : path;
}
function editable(file: WebConfigFile) {
  return (
    file.editable !== false &&
    !!loaded.value?.managedPaths.some(
      p => file.path === p || file.path.startsWith(p.replace(/\/$/, "") + "/")
    )
  );
}
async function discardChanges() {
  if (!changes.value.length || savedSignature === signature.value) return true;
  try {
    await ElMessageBox.confirm(
      tr("存在未发布的配置变更，重新读取将丢弃这些变更，是否继续？"),
      tr("配置变更"),
      { type: "warning" }
    );
    return true;
  } catch {
    return false;
  }
}
async function read() {
  const id = instanceId.value;
  if (!id) return;
  const gen = ++generation;
  abort?.abort();
  abort = new AbortController();
  loading.value = true;
  error.value = "";
  results.value = [];
  try {
    const data = await webInstanceConfig(id, abort.signal);
    if (gen !== generation) return;
    loaded.value = data;
    files.value = data.files.map(f => ({ ...f }));
    originals.value = Object.fromEntries(
      data.files.map(f => [f.path, { ...f }])
    );
    if (!files.value.some(f => f.path === activePath.value))
      activePath.value = data.configPath;
    changeFilePage(
      Math.floor(
        Math.max(
          0,
          visibleFiles.value.findIndex(f => f.path === activePath.value)
        ) / filePageSize.value
      ) + 1
    );
    savedSignature = "";
    savedDraft = undefined;
  } catch (e) {
    if (gen === generation) error.value = (e as Error).message;
  } finally {
    if (gen === generation) loading.value = false;
  }
}
async function selectInstance(id: number) {
  if (busy.value || id === instanceId.value || !(await discardChanges()))
    return;
  instanceId.value = id;
  loaded.value = undefined;
  files.value = [];
  originals.value = {};
  targetScope.value = "instance";
  draftGroupId.value = undefined;
  groupFilter.value = undefined;
  await read();
}
async function refresh() {
  if (await discardChanges()) await read();
}
function openAdd() {
  newPath.value = `${loaded.value!.configPath.replace(/\/[^/]+$/, "")}/conf.d/new.conf`;
  adding.value = true;
}
async function addFile() {
  const path = newPath.value.trim();
  if (
    !path.startsWith("/") ||
    path.split("/").some(p => p === ".." || p === ".") ||
    /[\r\n\0]/.test(path)
  ) {
    ElMessage.error(tr("配置路径无效"));
    return;
  }
  if (files.value.some(f => f.path === path)) {
    ElMessage.error(tr("配置路径重复或无效"));
    return;
  }
  const file = {
    path,
    content: "",
    mode: 420,
    editable: true,
    included: false
  };
  if (!editable(file)) {
    ElMessage.error(tr("配置路径不在受管目录内: {value0}", { value0: path }));
    return;
  }
  files.value.push(file);
  activePath.value = path;
  search.value = "";
  groupFilter.value = undefined;
  adding.value = false;
  await nextTick();
  changeFilePage(
    Math.floor(
      visibleFiles.value.findIndex(f => f.path === path) / filePageSize.value
    ) + 1
  );
}
async function toggleDelete(file: WebConfigFile) {
  if (busy.value || deletingPath.value || !canWrite.value || !editable(file))
    return;
  if (file.deleted) {
    file.deleted = false;
    return;
  }
  const gen = generation;
  deletingPath.value = file.path;
  try {
    await ElMessageBox.confirm(
      tr(
        originals.value[file.path]
          ? "确认将文件「{value0}」标记为删除？只有完整配置校验通过并确认发布后，远端文件才会被删除。"
          : "确认移除尚未保存的新文件「{value0}」？此文件的编辑内容将丢失。",
        { value0: file.path }
      ),
      tr("删除文件"),
      {
        type: "warning",
        confirmButtonText: tr("确认删除"),
        cancelButtonText: tr("取消"),
        closeOnClickModal: false
      }
    );
    if (
      gen !== generation ||
      busy.value ||
      !canWrite.value ||
      !editable(file) ||
      !files.value.includes(file)
    )
      return;
    if (originals.value[file.path]) file.deleted = true;
    else {
      const index = files.value.indexOf(file);
      files.value.splice(index, 1);
      activePath.value =
        files.value[Math.min(index, files.value.length - 1)]?.path || "";
    }
  } catch {
    /* cancellation leaves the file and its content unchanged */
  } finally {
    deletingPath.value = "";
  }
}
function scope() {
  return {
    instanceIds: targetIds.value,
    baselines: loaded.value
      ? { [loaded.value.instanceId]: loaded.value.digest }
      : {}
  };
}
async function validate() {
  const current = signature.value,
    gen = ++validationGeneration;
  validating.value = true;
  error.value = "";
  try {
    const value = await webValidateClusterConfig(
      loaded.value!.clusterId,
      changes.value,
      scope()
    );
    if (gen === validationGeneration && current === signature.value)
      results.value = value;
  } catch (e) {
    if (gen === validationGeneration) error.value = (e as Error).message;
  } finally {
    if (gen === validationGeneration) validating.value = false;
  }
}
async function save(publish: boolean) {
  if (
    !canWrite.value ||
    busy.value ||
    !changes.value.length ||
    !loaded.value?.clusterId
  )
    return;
  const current = signature.value;
  saving.value = true;
  error.value = "";
  try {
    if (!savedDraft || savedSignature !== current) {
      const resource = selectedInstance.value!;
      const draft = await webRequest<WebResource>("/configurations", "POST", {
        name: `${tr("原生配置")} · ${resource.name}`.slice(0, 100),
        groupId: resource.groupId,
        clusterId: loaded.value.clusterId,
        engine: resource.engine,
        environment: resource.environment,
        spec: {
          files: changes.value,
          ...scope(),
          configGroupId: draftGroupId.value || 0
        }
      });
      savedDraft = draft;
      savedSignature = current;
    }
    ElMessage.success(tr("配置校验通过，草稿已保存"));
    emit("saved", savedDraft, publish);
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    saving.value = false;
  }
}
watch(signature, () => {
  results.value = [];
  validationGeneration++;
  validating.value = false;
});
watch(
  () => [props.instances, props.initialInstanceId] as const,
  ([instances, initial]) => {
    const selected =
      initial && instances.some(i => i.id === initial)
        ? initial
        : instances[0]?.id;
    if (!instances.some(i => i.id === instanceId.value)) {
      if (changes.value.length && selectedInstance.value) return;
      generation++;
      abort?.abort();
      instanceId.value = undefined;
      loaded.value = undefined;
      files.value = [];
      originals.value = {};
      if (selected) {
        instanceId.value = selected;
        read();
      }
    }
  },
  { immediate: true }
);
watch(
  () => props.refreshKey,
  () => {
    if (!changes.value.length || savedSignature === signature.value) read();
  }
);
watch(
  () => props.initialInstanceId,
  initial => {
    if (initial && props.allInstances.some(i => i.id === initial))
      selectInstance(initial);
  }
);
defineExpose({ confirmLeave: discardChanges });
onBeforeRouteLeave(() => discardChanges());
onUnmounted(() => {
  generation++;
  validationGeneration++;
  abort?.abort();
});
</script>

<style scoped>
.native-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin: 16px 0;
}
.native-toolbar .el-select {
  width: 290px;
  max-width: 100%;
}
.native-toolbar .el-input {
  width: 320px;
  max-width: 100%;
}
.muted {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.config-entry {
  overflow-wrap: anywhere;
}
.config-workspace {
  display: grid;
  grid-template-columns: minmax(260px, 28%) minmax(0, 1fr);
  height: clamp(520px, 68vh, 760px);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  overflow: hidden;
}
.config-tree {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  background: var(--el-fill-color-light);
  border-right: 1px solid var(--el-border-color);
}
.file-list-heading {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px;
  border-bottom: 1px solid var(--el-border-color);
}
.file-list {
  flex: 1;
  min-height: 0;
  padding: 8px;
  overflow-y: auto;
  scrollbar-gutter: stable;
}
.file-pagination {
  padding: 12px 8px;
  border-top: 1px solid var(--el-border-color);
}
.file-pagination :deep(.el-pagination) {
  justify-content: center;
  flex-wrap: wrap;
  gap: 8px;
}
.file-pagination :deep(.el-pagination > *) {
  margin: 0;
}
.file-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 12px 10px;
  margin-bottom: 4px;
  text-align: left;
  color: var(--el-text-color-primary);
  background: transparent;
  border: 1px solid transparent;
  border-radius: 6px;
  cursor: pointer;
}
.file-row.active {
  border-color: var(--el-color-primary-light-5);
  background: var(--el-color-primary-light-9);
}
.file-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: monospace;
}
.file-title {
  display: flex;
  align-items: center;
  width: 100%;
  gap: 8px;
}
.file-title .el-icon {
  flex-shrink: 0;
}
.file-row.deleted .file-label {
  text-decoration: line-through;
}
.config-editor {
  display: flex;
  flex-direction: column;
  padding: 16px;
  min-width: 0;
  min-height: 0;
}
.editor-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  overflow-wrap: anywhere;
}
.editor-heading code {
  min-width: 0;
}
.editor-heading .el-button {
  flex-shrink: 0;
}
.native-code {
  flex: 1;
  min-height: 0;
}
.native-code :deep(textarea) {
  height: 100%;
  resize: none;
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 13px;
  line-height: 1.65;
  tab-size: 4;
  white-space: pre;
}
.editor-shortcuts {
  margin: 10px 0 0;
  line-height: 1.6;
}
.validation-output {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  margin: 6px 0;
}
@media (max-width: 760px) {
  .config-workspace {
    grid-template-columns: 1fr;
    grid-template-rows: 340px minmax(480px, auto);
    height: auto;
  }
  .config-tree {
    border-right: 0;
    border-bottom: 1px solid var(--el-border-color);
  }
  .native-toolbar .el-select,
  .native-toolbar .el-input {
    width: 100%;
  }
}
</style>
