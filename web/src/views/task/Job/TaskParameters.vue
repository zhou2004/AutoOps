<template>
  <div class="task-parameters">
    <p class="parameter-help">{{ help }}</p>
    <div v-for="(row, index) in rows" :key="row.id" class="parameter-row">
      <el-input v-model="row.key" placeholder="参数名，如 APP_NAME" @input="publish" />
      <el-input v-model="row.value" type="textarea" :autosize="{ minRows: 1, maxRows: 4 }" placeholder="参数值（可留空）" @input="publish" />
      <el-button :icon="Delete" text aria-label="删除参数" @click="remove(index)" />
    </div>
    <div class="parameter-actions">
      <el-button :icon="Plus" size="small" @click="add">添加参数</el-button>
      <el-button size="small" @click="toggleBatch">批量编辑 JSON / YAML</el-button>
    </div>
    <template v-if="batchVisible">
      <el-input v-model="batchText" type="textarea" :rows="6" placeholder="APP_NAME: orders&#10;VERSION: v1" />
      <el-button size="small" @click="applyBatch">应用批量参数</el-button>
    </template>
    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon />
    <details v-if="defaults.length" class="parameter-defaults">
      <summary>查看继承的默认参数（同名参数可在上方覆盖）</summary>
      <div v-for="item in defaults" :key="item.name">
        <strong>{{ item.name }}</strong>
        <pre>{{ JSON.stringify(item.parameters, null, 2) }}</pre>
      </div>
    </details>
    <p class="parameter-help" v-pre>脚本通过环境变量读取：Shell 使用 "$APP_NAME"；Python 使用 os.environ["APP_NAME"]；Ansible 使用 lookup('env', 'APP_NAME')。</p>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { Delete, Plus } from '@element-plus/icons-vue'
import { load } from 'js-yaml'

const props = defineProps({ modelValue: { type: Object, default: () => ({}) }, defaults: { type: Array, default: () => [] }, help: { type: String, default: '参数作为环境变量传入执行脚本，空值会覆盖默认值，删除参数则恢复继承。' } })
const emit = defineEmits(['update:modelValue'])
const rows = ref([]), error = ref(''), batchVisible = ref(false), batchText = ref('')
let nextId = 0, published = ''
watch(() => props.modelValue, value => {
  const serialized = JSON.stringify(value || {})
  if (serialized === published) return
  rows.value = Object.entries(value || {}).map(([key, value]) => ({ id: nextId++, key, value: String(value) }))
  error.value = ''; batchVisible.value = false; batchText.value = serialized
}, { immediate: true, deep: true })
function add() { rows.value.push({ id: nextId++, key: '', value: '' }) }
function remove(index) { rows.value.splice(index, 1); publish() }
function readRows() {
  const entries = [], keys = new Set()
  for (const row of rows.value) {
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(row.key)) throw new Error('参数名只能包含字母、数字和下划线，且不能以数字开头')
    if (keys.has(row.key)) throw new Error(`参数名重复：${row.key}`)
    keys.add(row.key); entries.push([row.key, row.value])
  }
  return Object.fromEntries(entries)
}
function publish() {
  try {
    const value = readRows(); error.value = ''; published = JSON.stringify(value)
    emit('update:modelValue', value)
  } catch (e) { error.value = e.message }
}
function toggleBatch() {
  if (!batchVisible.value) {
    try { batchText.value = JSON.stringify(readRows(), null, 2) }
    catch (e) { error.value = e.message; return }
  }
  batchVisible.value = !batchVisible.value
}
function applyBatch() {
  try {
    const value = load(batchText.value) ?? {}
    if (typeof value !== 'object' || Array.isArray(value)) throw new Error('请输入 JSON 对象或 YAML 键值配置')
    const next = Object.entries(value).map(([key, value]) => ({ id: nextId++, key, value: typeof value === 'object' && value !== null ? JSON.stringify(value) : String(value ?? '') }))
    rows.value = next
    const result = readRows()
    published = JSON.stringify(result); emit('update:modelValue', result)
    error.value = ''; batchVisible.value = false
  } catch (e) { error.value = e.message }
}
function getValue() {
  if (batchVisible.value) applyBatch()
  if (error.value) throw new Error(error.value)
  return readRows()
}
defineExpose({ getValue })
</script>

<style scoped>
.task-parameters { width: 100%; }
.parameter-row { display: grid; grid-template-columns: minmax(130px, 1fr) minmax(150px, 2fr) 32px; gap: 8px; margin-bottom: 8px; align-items: start; }
.parameter-actions { display: flex; gap: 8px; margin: 8px 0; flex-wrap: wrap; }
.parameter-help, .parameter-defaults { color: var(--text-secondary); font-size: 12px; line-height: 1.6; margin: 8px 0; }
.parameter-defaults pre { white-space: pre-wrap; overflow-wrap: anywhere; }
@media (max-width: 600px) { .parameter-row { grid-template-columns: 1fr 1fr 32px; } }
</style>
