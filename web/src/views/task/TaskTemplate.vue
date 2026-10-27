<template>
  <div class="tasktemplate-management">
    <el-card shadow="hover" class="tasktemplate-card">
      <template #header>
        <div class="card-header">
          <span class="title">{{ $tr("任务模板管理") }}</span>
        </div>
      </template>
      
      <!-- 搜索区域 -->
      <div class="search-section">
        <el-form :inline="true" :model="queryParams" class="search-form">
          <el-form-item :label='$tr("模板名称")'>
            <el-input 
              :placeholder='$tr("请输入模板名称")' 
              size="small" 
              clearable 
              v-model="queryParams.name"
              @keyup.enter="searchByName(queryParams.name)"
            />
          </el-form-item>
          <el-form-item :label='$tr("模板类型")' style="width: 200px">
            <el-select 
              size="small" 
              :placeholder='$tr("请选择模板类型")' 
              v-model="queryParams.type"
              @change="searchByType(queryParams.type)"
              style="width: 100%"
            >
              <el-option
                v-for="item in templateTypes"
                :key="item.value"
                :label="$tr(item.label)"
                :value="item.value"
              />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" size="small" @click="searchByName(queryParams.name)" class="modern-btn primary-btn">
              <el-icon><Search /></el-icon>
              {{ $tr("搜索") }}
            </el-button>
            <el-button size="small" @click="resetQuery" class="modern-btn reset-btn">
              <el-icon><Refresh /></el-icon>
              {{ $tr("重置") }}
            </el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- 操作按钮 -->
      <div class="action-section">
        <el-button 
          type="success" 
          size="small"
          @click="handleCreate"
          v-authority="['task:template:add']"
          class="modern-btn success-btn"
        >
          <el-icon><Plus /></el-icon>
          {{ $tr("新增模板") }}
        </el-button>
      </div>

      <!-- 列表区域 -->
      <div class="table-section">
        <el-table 
          v-loading="loading" 
          :data="templates" 
          border 
          stripe 
          style="width: 100%"

        >
          <el-table-column prop="name" :label='$tr("模板名称")' width="270">
            <template #default="{row}">
              <div style="display: flex; align-items: center; gap: 8px;">
                <img 
                  :src="$getAssetUrl('@/assets/image/moban.svg')" 
                  :alt='$tr("模板")'
                  style="width: 20px; height: 20px; object-fit: contain; flex-shrink: 0;"
                />
                <span>{{ row.name }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="type" :label='$tr("类型")' width="140">
            <template #default="{row}">
              <div style="display: flex; align-items: center; gap: 8px;">
                <img 
                  :src="getTypeIcon(row.type)" 
                  :alt="getTypeName(row.type)"
                  style="width: 18px; height: 18px; object-fit: contain; flex-shrink: 0;"
                />
                <span :class="getTypeClass(row.type)">
                  {{ getTypeName(row.type) }}
                </span>
              </div>
            </template>
          </el-table-column>
          <el-table-column :label='$tr("脚本内容")' width="100">
            <template #default="{row}">
              <el-tooltip :content='$tr("查看脚本")' placement="top">
                <el-button
                  type="primary"
                  :icon="View"
                  size="small"
                  circle
                  @click="handleViewContent(row.id)"
                />
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column prop="createdBy" :label='$tr("创建人")' width="140">
            <template #default="{row}">
              <div style="display: flex; align-items: center; gap: 8px;">
                <img 
                  :src="$getAssetUrl('@/assets/image/ren.svg')" 
                  :alt='$tr("创建人")'
                  style="width: 18px; height: 18px; object-fit: contain; flex-shrink: 0;"
                />
                <span>{{ row.createdBy }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="createdAt" :label='$tr("创建时间")' width="180" />
          <el-table-column prop="remark" :label='$tr("备注信息")' >
            <template #default="{row}">
              {{ row.remark }}
            </template>
          </el-table-column>

          <el-table-column :label='$tr("操作")' width="120">
            <template #default="{row}">
              <div class="operation-buttons">
                <el-tooltip :content='$tr("编辑")' placement="top">
                  <el-button
                    type="warning"
                    :icon="Edit"
                    size="small"
                    circle
                    v-authority="['task:template:edit']"
                    @click="handleEdit(row)"
                  />
                </el-tooltip>
                <el-tooltip :content='$tr("删除")' placement="top">
                  <el-button
                    type="danger"
                    :icon="Delete"
                    size="small"
                    circle
                    v-authority="['task:template:delete']"
                    @click="handleDelete(row.id)"
                  />
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- 分页区域 -->
      <div class="pagination-section">
        <el-pagination 
          @size-change="handleSizeChange" 
          @current-change="handleCurrentChange"
          :current-page="queryParams.pageNum" 
          :page-sizes="[10, 50, 100]" 
          :page-size="queryParams.pageSize"
          layout="total, sizes, prev, pager, next, jumper" 
          :total="total"
        />
      </div>
    </el-card>

    <!--模板表单对话框-->
    <el-dialog 
      :title="currentTemplate.id ? '编辑模板' : $tr('新增模板')" 
      v-model="formVisible" 
      width="40%"
      @close="formDialogClosed"
    >
      <el-form 
        ref="templateFormRef" 
        label-width="100px" 
        :model="currentTemplate" 
        :rules="templateRules"
      >
        <el-form-item :label='$tr("模板名称")' prop="name">
          <el-input v-model="currentTemplate.name" :placeholder='$tr("请输入模板名称")' />
        </el-form-item>
        <el-form-item :label='$tr("模板类型")' prop="type">
          <el-select v-model="currentTemplate.type" :placeholder='$tr("请选择模板类型")'>
            <el-option
              v-for="item in templateTypes"
              :key="item.value"
              :label="$tr(item.label)"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label='$tr("模板内容")' prop="content">
          <CodeEditor
            v-model="currentTemplate.content"
            :language="getLanguage(currentTemplate.type)"
            height="300px"
            :placeholder='$tr("请输入模板内容")'
          />
        </el-form-item>
        <el-form-item :label='$tr("备注信息")'>
          <el-input 
            v-model="currentTemplate.remark" 
            type="textarea" 
            :rows="2"
            :placeholder='$tr("请输入备注信息")'
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible = false">{{ $tr("取消") }}</el-button>
        <el-button type="primary" @click="handleSubmit">{{ $tr("确定") }}</el-button>
      </template>
    </el-dialog>

    <!--脚本内容查看对话框-->
    <el-dialog 
      :title='$tr("脚本内容")' 
      v-model="contentDialogVisible" 
      width="60%"
    >
      <pre 
        style="white-space: pre-wrap; background: #1e1e1e; padding: 10px; border-radius: 4px;"
        v-html="scriptContent"
      ></pre>
      <template #footer>
        <el-button @click="contentDialogVisible = false">{{ $tr("关闭") }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { tr } from "@/locales";

import { ref, reactive, onMounted } from 'vue'
import CodeEditor from '@/components/CodeEditor.vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { View, Edit, Delete, Search, Refresh, Plus } from '@element-plus/icons-vue'
import { highlight } from '@/utils/highlight'
import { 
  CreateTemplate, 
  GetAllTemplates,
  UpdateTemplate,
  DeleteTemplate,
  GetTemplateByID,
  GetTemplatesByName,
  GetTemplatesByType,
  GetTemplateContent
} from '@/api/task'

const templates = ref([])
const loading = ref(false)
const formVisible = ref(false)
const templateFormRef = ref(null)
const total = ref(0)

const queryParams = reactive({
  pageNum: 1,
  pageSize: 10,
  name: '',
  type: null
})

const currentTemplate = ref({
  name: '',
  type: 1,
  content: '',
  remark: ''
})

const contentDialogVisible = ref(false)
const scriptContent = ref('')

const templateTypes = [
  { value: 1, label: 'Shell', language: 'bash' },
  { value: 2, label: 'Python', language: 'python' },
  { value: 3, label: 'Ansible Playbook', language: 'yaml' },
  { value: 4, label: 'JavaScript (Node.js)', language: 'javascript' },
  { value: 5, label: 'PowerShell', language: 'powershell' },
  { value: 6, label: 'Go', language: 'go' },
  { value: 7, label: 'Ruby', language: 'ruby' },
  { value: 8, label: 'PHP', language: 'php' },
  { value: 9, label: 'Perl', language: 'perl' },
  { value: 10, label: '直接命令', language: 'bash' }
]

const templateRules = reactive({
  name: [{ required: true, get message() { return tr('请输入模板名称'); }, trigger: 'blur' }],
  type: [{ required: true, get message() { return tr('请选择模板类型'); }, trigger: 'change' }],
  content: [{ required: true, get message() { return tr('请输入模板内容'); }, trigger: 'blur' }]
})

const fetchTemplates = async () => {
  loading.value = true
  try {
    const response = await GetAllTemplates(queryParams)
    const responseData = response?.data || {}
    templates.value = Array.isArray(responseData.data) 
      ? responseData.data.map(item => ({
          id: item.id,
          name: item.name,
          type: item.type,
          content: item.content,
          remark: item.remark || '', // 确保remark字段存在
          createdBy: item.createdBy,
          updatedBy: item.updatedBy,
          createdAt: item.createdAt,
          updatedAt: item.updatedAt
        })) 
      : []
    console.log('模板数据:', JSON.stringify(templates.value, null, 2)) // 详细日志
    total.value = templates.value.length
  } catch (error) {
    console.error('获取模板列表异常:', error)
    templates.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const handleCreate = () => {
  currentTemplate.value = {
    name: '',
    type: 1,
    content: ''
  }
  formVisible.value = true
}

const handleEdit = (template) => {
  currentTemplate.value = { ...template }
  formVisible.value = true
}

const handleDelete = async (id) => {
  try {
    await ElMessageBox.confirm(tr('确定要删除该模板吗？'), tr('提示'), {
      confirmButtonText: tr('确定'),
      cancelButtonText: tr('取消'),
      type: 'warning'
    })
    await DeleteTemplate({ id })
    ElMessage.success(tr('删除成功'))
    fetchTemplates()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(tr('删除失败'))
    }
  }
}

const handleViewContent = async (id) => {
  try {
    const response = await GetTemplateContent({ id })
    const template = templates.value.find(t => t.id === id)
    const language = getLanguage(template?.type)
    scriptContent.value = highlight(response.data, language)
    contentDialogVisible.value = true
  } catch (error) {
    console.error('获取脚本内容失败:', error)
    scriptContent.value = '获取脚本内容失败'
  }
}

const handleSubmit = async () => {
  await templateFormRef.value.validate()
  if (currentTemplate.value.id) {
    const updateData = {
      id: currentTemplate.value.id,
      name: currentTemplate.value.name,
      type: currentTemplate.value.type,
      content: currentTemplate.value.content,
      remark: currentTemplate.value.remark || ''
    }
    await UpdateTemplate(updateData)
  } else {
    await CreateTemplate(currentTemplate.value)
  }
  formVisible.value = false
  fetchTemplates()
}

const searchByName = async (name) => {
  try {
    loading.value = true
    queryParams.name = name
    queryParams.pageNum = 1
    const params = {
      name: name || undefined,
      pageNum: queryParams.pageNum,
      pageSize: queryParams.pageSize
    }
    const res = await GetTemplatesByName(params)
    console.log('按名称搜索结果:', res)
    
    // 处理不同格式的响应
    if (res.data && Array.isArray(res.data)) {
      templates.value = res.data
      total.value = res.data.length
    } else if (res.data && res.data.data) {
      templates.value = res.data.data
      total.value = res.data.total || res.data.data.length
    } else {
      templates.value = []
      total.value = 0
      ElMessage.warning(tr('未获取到有效数据'))
    }
  } catch (error) {
    console.error('按名称搜索失败:', error)
    ElMessage.error(`按名称搜索失败: ${error.message || '未知错误'}`)
  } finally {
    loading.value = false
  }
}

const searchByType = async (type) => {
  if (!type) {
    queryParams.type = null
    queryParams.pageNum = 1
    await fetchTemplates()
    return
  }
  try {
    loading.value = true
    queryParams.type = type
    queryParams.pageNum = 1
    const params = {
      type: type || undefined,
      pageNum: queryParams.pageNum,
      pageSize: queryParams.pageSize
    }
    const res = await GetTemplatesByType(params)
    console.log('按类型搜索结果:', res)
    
    // 处理不同格式的响应
    if (res.data && Array.isArray(res.data)) {
      templates.value = res.data
      total.value = res.data.length
    } else if (res.data && res.data.data) {
      templates.value = res.data.data
      total.value = res.data.total || res.data.data.length
    } else {
      templates.value = []
      total.value = 0
      ElMessage.warning(tr('未获取到有效数据'))
    }
  } catch (error) {
    console.error('按类型搜索失败:', error)
    ElMessage.error(`按类型搜索失败: ${error.message || '未知错误'}`)
  } finally {
    loading.value = false
  }
}

const resetQuery = () => {
  queryParams.name = ''
  queryParams.type = null
  queryParams.pageNum = 1
  fetchTemplates()
}

const formDialogClosed = () => {
  templateFormRef.value?.resetFields()
}

const handleSizeChange = (val) => {
  queryParams.pageSize = val
  fetchTemplates()
}

const handleCurrentChange = (val) => {
  queryParams.pageNum = val
  fetchTemplates()
}

const getTypeClass = (type) => {
  return `type-language type-language-${type}`
}

const getLanguage = (type) => {
  return templateTypes.find(item => item.value === type)?.language || 'bash'
}

const getTypeIcon = (type) => {
  switch(type) {
    case 1: return new URL('@/assets/image/shell.svg', import.meta.url).href
    case 2: return new URL('@/assets/image/Python.svg', import.meta.url).href
    case 3: return new URL('@/assets/image/ansible.svg', import.meta.url).href
    default: return new URL('@/assets/image/shell.svg', import.meta.url).href
  }
}

const getTypeName = (type) => {
  return templateTypes.find(item => item.value === type)?.label || '未知类型'
}

onMounted(() => {
  fetchTemplates()
})
</script>

<style scoped>
.tasktemplate-management {
  padding: 20px;
  min-height: 100vh;
  background: var(--bg-page);
}

.tasktemplate-card {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--border);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.title {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
}

.search-section {
  margin-bottom: 16px;
  padding: 16px;
  background: var(--bg-card-alt);
  border-radius: var(--radius);
  border: 1px solid var(--border-light);
}

.search-form .el-form-item {
  margin-bottom: 0;
  margin-right: 16px;
}

.search-form .el-form-item__label {
  color: var(--text-regular);
  font-weight: 500;
}

.action-section {
  margin-bottom: 16px;
}

.table-section {
  margin-top: 0;
}

.pagination-section {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

.operation-buttons {
  display: flex;
  gap: 8px;
  justify-content: center;
}

.type-shell {
  color: var(--success);
  font-weight: 500;
}

.type-python {
  color: var(--primary);
  font-weight: 500;
}

.type-ansible {
  color: var(--warning);
  font-weight: 500;
}

.type-language {
  color: var(--primary);
  font-weight: 500;
}
</style>
