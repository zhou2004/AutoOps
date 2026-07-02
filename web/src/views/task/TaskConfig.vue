<template>
  <div class="ansible-config-management">
    <el-card shadow="hover" class="config-card">
      <div class="card-header">
        <div class="title">Ansible配置管理</div>
        <div class="header-actions">
          <el-button type="primary" :icon="Plus" @click="handleCreate">新增配置</el-button>
          <el-button :icon="Refresh" circle @click="handleRefresh"></el-button>
        </div>
      </div>

      <div class="search-section">
        <el-form :inline="true" class="search-form">
          <el-form-item>
            <el-input
              v-model="searchKeyword"
              placeholder="搜索配置名称"
              :prefix-icon="Search"
              clearable
              @clear="fetchConfigList"
              @keyup.enter="fetchConfigList"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="fetchConfigList">搜索</el-button>
            <el-button @click="resetSearch">重置</el-button>
          </el-form-item>
        </el-form>
      </div>

      <div class="config-tabs">
        <el-tabs v-model="activeTab" @tab-change="handleTabChange">
          <el-tab-pane label="主机清单" name="inventory">
            <template #label>
              <span class="custom-tab-label">
                <el-icon><List /></el-icon>
                <span>主机清单</span>
              </span>
            </template>
          </el-tab-pane>
          <el-tab-pane label="全局变量" name="globalVars">
            <template #label>
              <span class="custom-tab-label">
                <el-icon><Operation /></el-icon>
                <span>全局变量</span>
              </span>
            </template>
          </el-tab-pane>
          <el-tab-pane label="扩展变量" name="extraVars">
             <template #label>
              <span class="custom-tab-label">
                <el-icon><TopRight /></el-icon>
                <span>扩展变量</span>
              </span>
            </template>
          </el-tab-pane>
          <el-tab-pane label="命令行参数" name="cliArgs">
             <template #label>
              <span class="custom-tab-label">
                <el-icon><Terminal /></el-icon>
                <span>命令行参数</span>
              </span>
            </template>
          </el-tab-pane>
        </el-tabs>

        <div class="tab-content" v-loading="loading">
          <div class="content-header">
            <span class="resource-count">共 {{ total }} 个配置</span>
          </div>

          <div class="resource-table">
            <el-table :data="configList" style="width: 100%" class="config-table">
              <el-table-column prop="Name" label="名称" min-width="150" sortable>
                <template #default="{ row }">
                   <span class="name-text">{{ row.Name }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="Content" label="内容预览" min-width="250">
                 <template #default="{ row }">
                  <div class="content-preview">{{ formatContent(row.Content) }}</div>
                </template>
              </el-table-column>
               <el-table-column prop="Remark" label="备注" min-width="150" show-overflow-tooltip />
              <el-table-column prop="CreatedAt" label="创建时间" width="180">
                 <template #default="{ row }">
                  {{ formatDate(row.CreatedAt) }}
                </template>
              </el-table-column>
              <el-table-column label="操作" width="200" fixed="right">
                <template #default="{ row }">
                  <div class="operation-buttons">
                    <el-button type="primary" :icon="Edit" size="small" circle @click="handleEdit(row)" />
                    <el-button type="info" :icon="View" size="small" circle @click="handleView(row)" />
                    <el-button type="danger" :icon="Delete" size="small" circle @click="handleDelete(row)" />
                  </div>
                </template>
              </el-table-column>
            </el-table>
             <div class="pagination-container">
              <el-pagination
                v-model:current-page="currentPage"
                v-model:page-size="pageSize"
                :page-sizes="[10, 20, 50, 100]"
                layout="total, sizes, prev, pager, next, jumper"
                :total="total"
                @size-change="handleSizeChange"
                @current-change="handleCurrentChange"
              />
            </div>
          </div>
        </div>
      </div>
    </el-card>

    <!-- Create/Edit Dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogType === 'create' ? '新增配置' : '编辑配置'"
      width="60%"
      class="modern-dialog"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <el-form ref="configFormRef" :model="currentConfig" :rules="formRules" label-width="100px" label-position="top">
        <el-form-item label="配置名称" prop="name">
          <el-input v-model="currentConfig.name" placeholder="请输入配置名称" />
        </el-form-item>
        
        <el-form-item label="配置内容" prop="content">
            <template #label>
              <div class="label-with-tip">
                <span>配置内容</span>
                <span class="tip-text">{{ getContentTypeTip() }}</span>
              </div>
            </template>
          <el-input
            v-model="currentConfig.content"
            type="textarea"
            :rows="10"
            placeholder="请输入配置内容"
            class="code-editor"
            @keydown="handleEditorKeydown"
          />
        </el-form-item>

        <el-form-item label="备注描述" prop="remark">
          <el-input v-model="currentConfig.remark" type="textarea" rows="2" placeholder="请输入备注信息" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmit" :loading="submitting">
            确定
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- View Detail Dialog -->
    <el-dialog
      v-model="viewDialogVisible"
      title="配置详情"
      width="60%"
      class="detail-dialog"
    >
      <div class="detail-content" v-if="viewConfig">
        <div class="detail-item">
          <span class="label">名称:</span>
          <span class="value">{{ viewConfig.Name }}</span>
        </div>
         <div class="detail-item">
          <span class="label">类型:</span>
          <span class="value">{{ getTypeName(viewConfig.Type) }}</span>
        </div>
         <div class="detail-item">
          <span class="label">备注:</span>
          <span class="value">{{ viewConfig.Remark || '-' }}</span>
        </div>
        <div class="detail-item code-block">
            <div class="label">内容:</div>
             <pre class="code-content">{{ viewConfig.Content }}</pre>
        </div>
      </div>
       <template #footer>
        <span class="dialog-footer">
          <el-button @click="viewDialogVisible = false">关闭</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus,
  Refresh,
  Search,
  Edit,
  Delete,
  View,
  List,
  Operation,
  TopRight,
  Monitor as Terminal
} from '@element-plus/icons-vue'
import {
  GetAnsibleConfigList,
  CreateAnsibleConfig,
  UpdateAnsibleConfig,
  DeleteAnsibleConfig,
  GetAnsibleConfigById
} from '@/api/task'

// Constants for Config Types
const CONFIG_TYPES = {
  INVENTORY: 1,
  GLOBAL_VARS: 2,
  EXTRA_VARS: 3,
  CLI_ARGS: 4
}

const TAB_TO_TYPE = {
  'inventory': CONFIG_TYPES.INVENTORY,
  'globalVars': CONFIG_TYPES.GLOBAL_VARS,
  'extraVars': CONFIG_TYPES.EXTRA_VARS,
  'cliArgs': CONFIG_TYPES.CLI_ARGS
}

// State
const loading = ref(false)
const submitting = ref(false)
const activeTab = ref('inventory')
const searchKeyword = ref('')
const configList = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(10)

// Dialog State
const dialogVisible = ref(false)
const viewDialogVisible = ref(false)
const dialogType = ref('create') // 'create' or 'edit'
const configFormRef = ref(null)

const currentConfig = reactive({
  id: null,
  name: '',
  content: '',
  remark: '',
  type: CONFIG_TYPES.INVENTORY
})

const viewConfig = ref(null)

const formRules = {
  name: [{ required: true, message: '请输入配置名称', trigger: 'blur' }],
  content: [{ required: true, message: '请输入配置内容', trigger: 'blur' }]
}

// Methods
const getTypeFromTab = (tab) => TAB_TO_TYPE[tab] || CONFIG_TYPES.INVENTORY

const fetchConfigList = async () => {
  loading.value = true
  try {
    const type = getTypeFromTab(activeTab.value)
    const params = {
      page: currentPage.value,
      size: pageSize.value,
      type: type,
      name: searchKeyword.value
    }
    
    console.log('Fetching configs with params:', params)
    const res = await GetAnsibleConfigList(params)
    
    if (res && res.data && res.data.code === 200) {
      const data = res.data.data.list || []
      if (Array.isArray(data)) {
          // Sometimes backend returns array directly
          configList.value = data
          total.value = res.data.data.total // Simplified, ideally backend returns total
      } else {
          configList.value = []
          total.value = 0
      }

    } else {
      configList.value = []
      total.value = 0
    }
  } catch (error) {
    console.error('Failed to fetch configs:', error)
    ElMessage.error('获取配置列表失败')
  } finally {
    loading.value = false
  }
}

const handleTabChange = (tab) => {
  activeTab.value = tab
  currentPage.value = 1
  searchKeyword.value = ''
  fetchConfigList()
}

const handleRefresh = () => {
  fetchConfigList()
}

const resetSearch = () => {
  searchKeyword.value = ''
  currentPage.value = 1
  fetchConfigList()
}

const handleSizeChange = (val) => {
  pageSize.value = val
  fetchConfigList()
}

const handleCurrentChange = (val) => {
  currentPage.value = val
  fetchConfigList()
}

const handleCreate = () => {
  dialogType.value = 'create'
  currentConfig.id = null
  currentConfig.name = ''
  currentConfig.content = ''
  currentConfig.remark = ''
  // Ensure type is set based on current tab
  currentConfig.type = getTypeFromTab(activeTab.value)
  console.log('Preparing to create config of type:', currentConfig.type)
  dialogVisible.value = true
}

const handleEdit = async (row) => {
  dialogType.value = 'edit'
  // Pre-fill from row (handle both cases just to be safe)
  currentConfig.id = row.id || row.ID
  currentConfig.name = row.name || row.Name
  currentConfig.content = row.content || row.Content || '' 
  currentConfig.remark = row.remark || row.Remark
  currentConfig.type = row.type || row.Type
  
  // Fetch fresh detail
  try {
      const res = await GetAnsibleConfigById(currentConfig.id)
      if (res && res.data && res.data.code === 200) {
          const detail = res.data.data
          // Update form with detail data, handling PascalCase/camelCase
          currentConfig.content = detail.content || detail.Content || ''
          currentConfig.remark = detail.remark || detail.Remark || ''
          currentConfig.name = detail.name || detail.Name || ''
          currentConfig.type = detail.type || detail.Type || currentConfig.type
      }
  } catch(e) { console.error('获取配置详情失败:', e) }

  dialogVisible.value = true
}

const handleView = async (row) => {
    // Initialize with row data, handling casing
    viewConfig.value = {
      Name: row.name || row.Name,
      Type: row.type || row.Type,
      Remark: row.remark || row.Remark,
      Content: row.content || row.Content
    }
     try {
      const res = await GetAnsibleConfigById(row.id || row.ID)
      if (res && res.data && res.data.code === 200) {
          const detail = res.data.data
          // Update with fresh data, ensuring PascalCase for template binding
          viewConfig.value = {
            Name: detail.name || detail.Name,
            Type: detail.type || detail.Type,
            Remark: detail.remark || detail.Remark,
            Content: detail.content || detail.Content
          }
      }
  } catch(e) { console.error('获取配置详情失败:', e) }
    viewDialogVisible.value = true
}

const handleDelete = (row) => {
    ElMessageBox.confirm(
    `确定要删除配置 "${row.Name}" 吗?`,
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  )
    .then(async () => {
      try {
        const res = await DeleteAnsibleConfig(row.id || row.ID)
        if (res && res.data && res.data.code === 200) {
             ElMessage.success('删除成功')
             fetchConfigList()
        } else {
             ElMessage.error(res.data.message || '删除失败')
        }
      } catch (error) {
         ElMessage.error('删除配置失败')
      }
    })
    .catch(() => {})
}

const handleSubmit = async () => {
  if (!configFormRef.value) return
  
  await configFormRef.value.validate(async (valid) => {
    if (valid) {
      submitting.value = true
      try {
        const payload = {
            name: currentConfig.name,
            content: currentConfig.content,
            remark: currentConfig.remark,
            type: currentConfig.type
        }
        
        let res
        if (dialogType.value === 'create') {
             res = await CreateAnsibleConfig(payload)
        } else {
             payload.id = currentConfig.id
             res = await UpdateAnsibleConfig(payload)
        }

        if (res && res.data && res.data.code === 200) {
          ElMessage.success(dialogType.value === 'create' ? '创建成功' : '更新成功')
          dialogVisible.value = false
          fetchConfigList()
        } else {
           ElMessage.error(res.data.message || '操作失败')
        }
      } catch (error) {
        console.error(error)
        ElMessage.error(dialogType.value === 'create' ? '创建失败' : '更新失败')
      } finally {
        submitting.value = false
      }
    }
  })
}

// Helpers
const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString()
}

const formatContent = (content) => {
    if (!content) return ''
    if (content.length > 100) return content.substring(0, 100) + '...'
    return content
}

const getTypeName = (type) => {
    switch(type) {
        case CONFIG_TYPES.INVENTORY: return '主机清单'
        case CONFIG_TYPES.GLOBAL_VARS: return '全局变量'
        case CONFIG_TYPES.EXTRA_VARS: return '扩展变量'
        case CONFIG_TYPES.CLI_ARGS: return '命令行参数'
        default: return '未知'
    }
}

const getContentTypeTip = () => {
  const type = getTypeFromTab(activeTab.value)
  switch(type) {
    case CONFIG_TYPES.INVENTORY: return '支持INI格式格式的主机清单配置'
    case CONFIG_TYPES.GLOBAL_VARS: return '支持JSON格式'
    case CONFIG_TYPES.EXTRA_VARS: return '支持YAML格式'
    case CONFIG_TYPES.CLI_ARGS: return '标准ansible-playbook命令行参数'
    default: return ''
  }
}

// Tab 键缩进支持
const handleEditorKeydown = (e) => {
  if (e.key === 'Tab') {
    e.preventDefault()
    const textarea = e.target
    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const indent = '    ' // 4 spaces
    const before = currentConfig.content.substring(0, start)
    const after = currentConfig.content.substring(end)
    currentConfig.content = before + indent + after
    // 恢复光标位置
    textarea.setSelectionRange(start + indent.length, start + indent.length)
  }
}

onMounted(() => {
  fetchConfigList()
})
</script>

<style scoped>
.ansible-config-management {
  padding: 20px;
  min-height: 100vh;
  background: var(--bg-page);
}

.config-card {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
  border: 1px solid var(--border);
  min-height: 80vh;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.title {
  font-size: 22px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: 0.5px;
}

.header-actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

.header-actions .el-button {
  transition: all 0.25s ease;
}

.header-actions .el-button:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
}

.search-section {
  margin-bottom: 20px;
  padding: 16px;
  background: var(--bg-card-alt);
  border-radius: var(--radius);
  border: 1px solid var(--border-light);
}

.search-form {
  margin: 0;
}

.custom-tab-label {
    display: flex;
    align-items: center;
    gap: 6px;
}

.config-tabs :deep(.el-tabs__nav-wrap::after) {
  height: 1px;
  background: var(--border-light);
}

.config-tabs :deep(.el-tabs__active-bar) {
  height: 3px;
  border-radius: 3px 3px 0 0;
  background: linear-gradient(90deg, var(--primary), var(--primary-dark, #667eea));
}

.config-tabs :deep(.el-tabs__item) {
  font-size: 15px;
  font-weight: 500;
  color: var(--text-regular);
  transition: color 0.25s ease;
  padding: 0 20px;
}

.config-tabs :deep(.el-tabs__item:hover) {
  color: var(--primary);
}

.config-tabs :deep(.el-tabs__item.is-active) {
  color: var(--primary);
  font-weight: 600;
}

.content-header {
  margin: 10px 0;
  color: var(--text-secondary);
  font-size: 14px;
}

.resource-table {
    margin-top: 10px;
}

/* 表格 */
.config-table :deep(.el-table__header th) {
  background: var(--bg-card-alt);
  color: var(--text-primary);
  font-weight: 600;
  border-bottom: 2px solid var(--border);
}

.config-table :deep(.el-table__body tr:hover > td) {
  background-color: var(--bg-card-alt) !important;
}

.name-text {
    font-weight: 600;
    color: var(--primary);
    transition: color 0.2s ease;
    cursor: default;
}

.name-text:hover {
    color: var(--primary-dark);
}

/* 操作按钮 */
.operation-buttons {
  display: flex;
  gap: 8px;
  justify-content: center;
}

.operation-buttons .el-button {
  transition: all 0.25s ease;
}

.operation-buttons .el-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.content-preview {
    font-family: 'Fira Code', 'Cascadia Code', 'Menlo', 'Monaco', monospace;
    font-size: 13px;
    color: var(--text-regular);
    background: var(--bg-card-alt);
    padding: 6px 12px;
    border-radius: 6px;
    border-left: 3px solid var(--primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    line-height: 1.6;
    transition: all 0.2s ease;
}

.content-preview:hover {
    border-left-color: var(--primary-dark);
    background: linear-gradient(135deg, rgba(var(--primary-rgb, 102, 126, 234), 0.05) 0%, var(--bg-card-alt) 100%);
}

.pagination-container {
    margin-top: 20px;
    display: flex;
    justify-content: center;
}

/* Dialog Styles */
:deep(.modern-dialog) {
  border-radius: var(--radius);
  overflow: hidden;
}

:deep(.modern-dialog .el-dialog__header) {
  margin: 0;
  padding: 20px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-card-alt);
}

:deep(.modern-dialog .el-dialog__body) {
    padding: 24px;
    background: var(--bg-card);
}

.label-with-tip {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
}

.tip-text {
    font-size: 12px;
    color: var(--text-secondary);
    font-weight: normal;
}

.code-editor :deep(.el-textarea__inner) {
    font-family: 'Fira Code', 'Cascadia Code', 'Menlo', 'Monaco', 'Courier New', monospace;
    font-size: 14px;
    background-color: var(--bg-card-alt);
    color: var(--text-primary);
    line-height: 1.8;
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 14px 16px;
    transition: all 0.3s ease;
}

.code-editor :deep(.el-textarea__inner):focus {
    border-color: var(--primary);
    box-shadow: 0 0 0 3px rgba(var(--primary-rgb, 102, 126, 234), 0.12);
}

.code-editor :deep(.el-textarea__inner)::placeholder {
    color: var(--text-secondary);
    opacity: 0.6;
}

/* View Dialog */
:deep(.detail-dialog .el-dialog__header) {
  background: var(--bg-card-alt);
  border-bottom: 1px solid var(--border);
}

.detail-content {
  padding: 4px 0;
}

.detail-item {
    margin-bottom: 20px;
    display: flex;
    align-items: flex-start;
}

.detail-item .label {
    width: 72px;
    font-weight: 600;
    color: var(--text-secondary);
    flex-shrink: 0;
    font-size: 14px;
    padding-top: 2px;
}

.detail-item .value {
    color: var(--text-primary);
    font-size: 14px;
    font-weight: 500;
}

.code-block {
    flex-direction: column;
}

.code-block .label {
    margin-bottom: 8px;
    color: var(--text-primary);
}

.code-content {
    background: var(--bg-card-alt);
    color: var(--text-primary);
    padding: 16px;
    border-radius: 8px;
    border: 1px solid var(--border);
    border-left: 4px solid var(--primary);
    overflow-x: auto;
    font-family: 'Fira Code', 'Cascadia Code', 'Menlo', 'Monaco', 'Courier New', monospace;
    font-size: 14px;
    line-height: 1.8;
    margin: 0;
    white-space: pre-wrap;
    word-break: break-word;
}
</style>
