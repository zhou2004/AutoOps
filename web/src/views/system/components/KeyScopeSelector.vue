<script setup lang="ts">
import { computed, ref } from "vue";
import { Check, Delete, Search } from "@element-plus/icons-vue";
import { actionLabels, type PermissionModule } from "@/api/platformPermissions";
const props = defineProps<{
  modules: PermissionModule[];
  disabled?: boolean;
}>();
const scopes = defineModel<string[]>({ required: true });
const search = ref("");
const visibleModules = computed(() => {
  const q = search.value.trim().toLowerCase();
  return props.modules.filter(m =>
    `${m.name} ${m.key}`.toLowerCase().includes(q)
  );
});
function readonlyPreset() {
  scopes.value = props.modules
    .filter(m => m.actions.includes("read"))
    .map(m => `${m.key}:read`);
}
</script>
<template>
  <div class="scope-selector">
    <div class="scope-toolbar">
      <el-input
        v-model="search"
        :prefix-icon="Search"
        clearable
        :placeholder="$tr('搜索模块')"
        :aria-label="$tr('搜索模块')"
      />
      <el-button :icon="Check" :disabled="disabled" @click="readonlyPreset">{{
        $tr("全部设为只读")
      }}</el-button>
      <el-button :icon="Delete" :disabled="disabled" @click="scopes = []">{{
        $tr("清空授权")
      }}</el-button>
    </div>
    <div class="scope-count" aria-live="polite">
      {{ $tr("已选择") }} {{ scopes.length }} {{ $tr("项权限") }}
    </div>
    <el-checkbox-group
      v-model="scopes"
      :disabled="disabled"
      :aria-label="$tr('权限范围')"
    >
      <div
        v-for="m in visibleModules"
        :key="m.key"
        class="module-row"
        :data-module="m.key"
      >
        <div class="module-name">
          <strong>{{ $tr(m.name) }}</strong
          ><small>{{ m.key }}</small>
        </div>
        <div class="module-actions">
          <el-checkbox
            v-for="a in m.actions"
            :key="a"
            :value="`${m.key}:${a}`"
            >{{ $tr(actionLabels[a] || a) }}</el-checkbox
          >
        </div>
      </div>
    </el-checkbox-group>
    <el-empty
      v-if="!visibleModules.length"
      :description="$tr('暂无匹配的模块')"
      :image-size="60"
    />
  </div>
</template>
<style scoped>
.scope-selector {
  width: 100%;
  min-width: 0;
}
.scope-toolbar {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.scope-toolbar .el-input {
  width: 210px;
  flex-grow: 1;
}
.scope-toolbar .el-button {
  margin-left: 0;
}
.scope-count {
  margin: 12px 0 4px;
  font-size: 13px;
  color: var(--text-secondary);
}
.module-row {
  display: grid;
  grid-template-columns: 150px minmax(0, 1fr);
  align-items: center;
  gap: 20px;
  padding: 12px;
  border-bottom: 1px solid var(--border);
}
.module-row:nth-child(odd) {
  background: var(--bg-card-alt);
}
.module-name {
  color: var(--text-primary);
  line-height: 1.6;
  font-size: 14px;
}
.module-name small {
  display: block;
  color: var(--text-secondary);
  font-weight: 400;
}
.module-actions {
  display: flex;
  flex-wrap: wrap;
  column-gap: 20px;
}
.module-actions .el-checkbox {
  margin-right: 0;
}
@media (max-width: 600px) {
  .module-row {
    grid-template-columns: 1fr;
    gap: 4px;
  }
  .module-name small {
    display: inline;
    margin-left: 8px;
  }
  .scope-toolbar .el-input {
    width: 100%;
  }
}
</style>
