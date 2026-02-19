<script setup>
import { ref, reactive, onMounted, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus,
  Edit,
  Delete,
  View,
  Document,
  Refresh,
  Setting,
  Key,
  Search,
  Files,
  Lock
} from '@element-plus/icons-vue'
import k8sApi from '@/api/k8s'
import PodYamlDialog from './pods/PodYamlDialog.vue'
import ClusterSelector from './pods/ClusterSelector.vue'
import NamespaceSelector from './pods/NamespaceSelector.vue'

// 基础状态
const loading = ref(false)
const activeTab = ref('configmap')
const searchKeyword = ref('')

// 集群和命名空间状态
const selectedClusterId = ref('')
const queryParams = reactive({
  namespace: 'default'
})

// 配置资源数据状态
const configMapList = ref([])
const secretList = ref([])

// 对话框状态
const configMapYamlDialogVisible = ref(false)
const secretYamlDialogVisible = ref(false)
const configMapDetailDialogVisible = ref(false)
const secretDetailDialogVisible = ref(false)

// 当前操作的资源
const currentConfigMapYaml = ref('')
const currentSecretYaml = ref('')
const currentResourceName = ref('')
const currentResourceType = ref('')

// 当前查看的资源详情
const currentConfigMapForDetail = ref({})
const currentSecretForDetail = ref({})

// 过滤后的列表
const filteredConfigMapList = computed(() => {
  const list = Array.isArray(configMapList.value) ? configMapList.value : []
  if (!searchKeyword.value) return list
  return list.filter(item =>
    item.name?.toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
    item.namespace?.toLowerCase().includes(searchKeyword.value.toLowerCase())
  )
})

const filteredSecretList = computed(() => {
  const list = Array.isArray(secretList.value) ? secretList.value : []
  if (!searchKeyword.value) return list
  return list.filter(item =>
    item.name?.toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
    item.type?.toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
    item.namespace?.toLowerCase().includes(searchKeyword.value.toLowerCase())
  )
})

// 处理集群选择变化
const handleClusterChange = (clusterId) => {
  selectedClusterId.value = clusterId
  console.log('集群选择变化:', clusterId)
  if (clusterId && queryParams.namespace) {
    loadAllConfigResources()
  }
}

// 处理命名空间选择变化
const handleNamespaceChange = (namespace) => {
  queryParams.namespace = namespace
  console.log('命名空间选择变化:', namespace)
  if (selectedClusterId.value && namespace) {
    loadAllConfigResources()
  }
}

// 加载所有配置资源
const loadAllConfigResources = async () => {
  if (!selectedClusterId.value) {
    console.warn('集群ID为空，无法加载配置资源')
    return
  }

  console.log('开始加载配置资源，集群ID:', selectedClusterId.value, '命名空间:', queryParams.namespace)

  loading.value = true
  try {
    // 并发加载所有配置资源
    await Promise.all([
      fetchConfigMapList(),
      fetchSecretList()
    ])
  } catch (error) {
    console.error('加载配置资源失败:', error)
  } finally {
    loading.value = false
  }
}

// 重置搜索
const resetSearch = () => {
  searchKeyword.value = ''
}

// 标签页切换处理
const handleTabChange = (tabName) => {
  console.log('标签页切换到:', tabName)
  activeTab.value = tabName
}

// 获取ConfigMap列表
const fetchConfigMapList = async () => {
  if (!selectedClusterId.value || !queryParams.namespace) {
    console.warn('集群ID或命名空间为空，无法获取 ConfigMap 列表')
    return
  }

  try {
    console.log('正在获取 ConfigMap 列表，集群ID:', selectedClusterId.value, '命名空间:', queryParams.namespace)

    const response = await k8sApi.getConfigMaps(selectedClusterId.value, queryParams.namespace)
    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      const rawData = responseData.data?.configMaps || responseData.data
      configMapList.value = Array.isArray(rawData) ? rawData : []
      console.log('获取到 ConfigMap 列表:', configMapList.value.length, '个')
    } else {
      ElMessage.error(responseData.message || '获取 ConfigMap 列表失败')
      configMapList.value = []
    }
  } catch (error) {
    console.error('获取 ConfigMap 列表失败:', error)
    ElMessage.error('获取 ConfigMap 列表失败，请检查网络连接')
    configMapList.value = []
  }
}

// 获取Secret列表
const fetchSecretList = async () => {
  if (!selectedClusterId.value || !queryParams.namespace) {
    console.warn('集群ID或命名空间为空，无法获取 Secret 列表')
    return
  }

  try {
    console.log('正在获取 Secret 列表，集群ID:', selectedClusterId.value, '命名空间:', queryParams.namespace)

    const response = await k8sApi.getSecrets(selectedClusterId.value, queryParams.namespace)
    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      const rawData = responseData.data?.secrets || responseData.data
      secretList.value = Array.isArray(rawData) ? rawData : []
      console.log('获取到 Secret 列表:', secretList.value.length, '个')
    } else {
      ElMessage.error(responseData.message || '获取 Secret 列表失败')
      secretList.value = []
    }
  } catch (error) {
    console.error('获取 Secret 列表失败:', error)
    ElMessage.error('获取 Secret 列表失败，请检查网络连接')
    secretList.value = []
  }
}

// 刷新数据
const handleRefresh = () => {
  loadAllConfigResources()
}

// ConfigMap 操作
const handleCreateConfigMap = () => {
  if (!selectedClusterId.value) {
    ElMessage.warning('请先选择集群')
    return
  }
  if (!queryParams.namespace) {
    ElMessage.warning('请先选择命名空间')
    return
  }

  // 设置默认的ConfigMap YAML模板
  const defaultYaml = `apiVersion: v1
kind: ConfigMap
metadata:
  name: new-configmap
  namespace: ${queryParams.namespace}
data:
  # 配置键值对
  config.properties: |
    database.url=localhost:5432
    database.user=admin
  app.yaml: |
    app:
      name: my-app
      version: 1.0.0`

  currentConfigMapYaml.value = defaultYaml
  currentConfigMapForDetail.value = { name: 'new-configmap', namespace: queryParams.namespace }
  configMapYamlDialogVisible.value = true
}

// 查看 ConfigMap 详情
const handleViewConfigMap = async (row) => {
  try {
    const response = await k8sApi.getConfigMapDetail(selectedClusterId.value, queryParams.namespace, row?.name)
    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      currentConfigMapForDetail.value = responseData.data || row
      configMapDetailDialogVisible.value = true
    } else {
      ElMessage.error(responseData.message || '获取 ConfigMap 详情失败')
    }
  } catch (error) {
    console.error('获取 ConfigMap 详情失败:', error)
    ElMessage.error('获取 ConfigMap 详情失败')
  }
}

// 编辑 ConfigMap YAML
const handleEditConfigMapYaml = async (row) => {
  try {
    loading.value = true
    const response = await k8sApi.getConfigMapYaml(selectedClusterId.value, queryParams.namespace, row?.name)
    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      // 确保YAML内容是字符串格式
      let yamlContent = responseData.data

      // 如果后端返回的是包含yaml字段的对象，提取yaml字段
      if (typeof yamlContent === 'object' && yamlContent !== null && yamlContent.yaml) {
        yamlContent = yamlContent.yaml
      } else if (typeof yamlContent === 'object' && yamlContent !== null) {
        yamlContent = JSON.stringify(yamlContent, null, 2)
      } else if (yamlContent === null || yamlContent === undefined) {
        yamlContent = `# ConfigMap ${row?.name} YAML\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: ${row?.name}\n  namespace: ${queryParams.namespace}`
      }

      currentConfigMapYaml.value = String(yamlContent)
      currentConfigMapForDetail.value = row
      configMapYamlDialogVisible.value = true
    } else {
      ElMessage.error(responseData.message || '获取 ConfigMap YAML 失败')
    }
  } catch (error) {
    console.error('获取 ConfigMap YAML 失败:', error)
    ElMessage.error('获取 ConfigMap YAML 失败')
  } finally {
    loading.value = false
  }
}

// 保存 ConfigMap YAML
const handleConfigMapYamlSave = async (data) => {
  try {
    // 检查是否是创建新ConfigMap (只有 new-configmap 表示新建)
    const isCreating = data.resourceName === 'new-configmap'

    let response
    if (isCreating) {
      // 创建新ConfigMap
      response = await k8sApi.createConfigMap(selectedClusterId.value, queryParams.namespace, { yamlContent: data.yamlContent })
    } else {
      // 更新现有ConfigMap
      response = await k8sApi.updateConfigMapYaml(selectedClusterId.value, queryParams.namespace, data.resourceName, data.yamlContent)
    }

    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(isCreating ? 'ConfigMap 创建成功' : 'ConfigMap YAML 更新成功')
      configMapYamlDialogVisible.value = false
      fetchConfigMapList() // 刷新列表
    } else {
      ElMessage.error(responseData.message || (isCreating ? 'ConfigMap 创建失败' : 'ConfigMap YAML 更新失败'))
    }
  } catch (error) {
    console.error('ConfigMap 操作失败:', error)
    ElMessage.error('ConfigMap 操作失败')
  }
}

// 删除 ConfigMap
const handleDeleteConfigMap = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确认删除 ConfigMap "${row?.name}"？删除后相关配置将无法恢复。`,
      '删除确认',
      {
        confirmButtonText: '确认删除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )

    const response = await k8sApi.deleteConfigMap(selectedClusterId.value, queryParams.namespace, row?.name)
    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      ElMessage.success('ConfigMap 删除成功')
      fetchConfigMapList()
    } else {
      ElMessage.error(responseData.message || 'ConfigMap 删除失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除 ConfigMap 失败:', error)
      ElMessage.error('删除 ConfigMap 失败')
    }
  }
}

// Secret 操作
const handleCreateSecret = () => {
  if (!selectedClusterId.value) {
    ElMessage.warning('请先选择集群')
    return
  }
  if (!queryParams.namespace) {
    ElMessage.warning('请先选择命名空间')
    return
  }

  // 设置默认的Secret YAML模板
  const defaultYaml = `apiVersion: v1
kind: Secret
metadata:
  name: new-secret
  namespace: ${queryParams.namespace}
type: Opaque
data:
  # Base64编码的数据
  username: YWRtaW4=  # admin
  password: MWYyZDFlMmU2N2Rm  # password
stringData:
  # 明文数据，k8s会自动编码
  config.json: |
    {
      "database": {
        "host": "localhost",
        "port": 5432
      }
    }`

  currentSecretYaml.value = defaultYaml
  currentSecretForDetail.value = { name: 'new-secret', namespace: queryParams.namespace }
  secretYamlDialogVisible.value = true
}

// 查看 Secret 详情
const handleViewSecret = async (row) => {
  try {
    const response = await k8sApi.getSecretDetail(selectedClusterId.value, queryParams.namespace, row?.name)
    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      currentSecretForDetail.value = responseData.data || row
      secretDetailDialogVisible.value = true
    } else {
      ElMessage.error(responseData.message || '获取 Secret 详情失败')
    }
  } catch (error) {
    console.error('获取 Secret 详情失败:', error)
    ElMessage.error('获取 Secret 详情失败')
  }
}

// 编辑 Secret YAML
const handleEditSecretYaml = async (row) => {
  try {
    loading.value = true
    const response = await k8sApi.getSecretYaml(selectedClusterId.value, queryParams.namespace, row?.name)
    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      // 确保YAML内容是字符串格式
      let yamlContent = responseData.data

      // 如果后端返回的是包含yaml字段的对象，提取yaml字段
      if (typeof yamlContent === 'object' && yamlContent !== null && yamlContent.yaml) {
        yamlContent = yamlContent.yaml
      } else if (typeof yamlContent === 'object' && yamlContent !== null) {
        yamlContent = JSON.stringify(yamlContent, null, 2)
      } else if (yamlContent === null || yamlContent === undefined) {
        yamlContent = `# Secret ${row?.name} YAML\napiVersion: v1\nkind: Secret\nmetadata:\n  name: ${row?.name}\n  namespace: ${queryParams.namespace}\ntype: Opaque`
      }

      currentSecretYaml.value = String(yamlContent)
      currentSecretForDetail.value = row
      secretYamlDialogVisible.value = true
    } else {
      ElMessage.error(responseData.message || '获取 Secret YAML 失败')
    }
  } catch (error) {
    console.error('获取 Secret YAML 失败:', error)
    ElMessage.error('获取 Secret YAML 失败')
  } finally {
    loading.value = false
  }
}

// 保存 Secret YAML
const handleSecretYamlSave = async (data) => {
  try {
    // 检查是否是创建新Secret (只有 new-secret 表示新建)
    const isCreating = data.resourceName === 'new-secret'

    let response
    if (isCreating) {
      // 创建新Secret
      response = await k8sApi.createSecret(selectedClusterId.value, queryParams.namespace, { yamlContent: data.yamlContent })
    } else {
      // 更新现有Secret
      response = await k8sApi.updateSecretYaml(selectedClusterId.value, queryParams.namespace, data.resourceName, data.yamlContent)
    }

    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(isCreating ? 'Secret 创建成功' : 'Secret YAML 更新成功')
      secretYamlDialogVisible.value = false
      fetchSecretList() // 刷新列表
    } else {
      ElMessage.error(responseData.message || (isCreating ? 'Secret 创建失败' : 'Secret YAML 更新失败'))
    }
  } catch (error) {
    console.error('Secret 操作失败:', error)
    ElMessage.error('Secret 操作失败')
  }
}

// 删除 Secret
const handleDeleteSecret = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确认删除 Secret "${row?.name}"？删除后敏感数据将无法恢复。`,
      '删除确认',
      {
        confirmButtonText: '确认删除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )

    const response = await k8sApi.deleteSecret(selectedClusterId.value, queryParams.namespace, row?.name)
    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      ElMessage.success('Secret 删除成功')
      fetchSecretList()
    } else {
      ElMessage.error(responseData.message || 'Secret 删除失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除 Secret 失败:', error)
      ElMessage.error('删除 Secret 失败')
    }
  }
}

// 工具函数
const getSecretTypeTag = (type) => {
  const typeMap = {
    'Opaque': 'primary',
    'kubernetes.io/service-account-token': 'success',
    'kubernetes.io/dockercfg': 'warning',
    'kubernetes.io/dockerconfigjson': 'warning',
    'kubernetes.io/basic-auth': 'info',
    'kubernetes.io/ssh-auth': 'info',
    'kubernetes.io/tls': 'danger'
  }
  return typeMap[type] || 'info'
}

const formatSecretType = (type) => {
  if (type?.startsWith('kubernetes.io/')) {
    return type.replace('kubernetes.io/', 'k8s/')
  }
  return type || 'Opaque'
}

const formatDataKeys = (data) => {
  if (!data || typeof data !== 'object') return '无数据'
  const keys = Object.keys(data)
  if (keys.length === 0) return '无数据'
  if (keys.length <= 3) return keys.join(', ')
  return `${keys.slice(0, 3).join(', ')}... (${keys.length}项)`
}

// 监听集群和命名空间变化，自动加载数据
watch(
  [selectedClusterId, () => queryParams.namespace],
  ([clusterId, namespace]) => {
    console.log('监听到变化 - 集群ID:', clusterId, '命名空间:', namespace)
    if (clusterId && namespace) {
      console.log('集群和命名空间都已选择，开始加载配置资源')
      loadAllConfigResources()
    }
  },
  { immediate: true }
)

// 页面初始化
onMounted(async () => {
  console.log('🚀 开始加载k8s配置管理页面')
  const startTime = Date.now()

  try {
    console.log('🎉 页面初始化完成，总耗时:', Date.now() - startTime + 'ms')
  } catch (error) {
    console.error('页面初始化失败:', error)
  }
})
</script>

<template>
  <div class="k8s-config-management">
    <el-card shadow="hover" class="config-card">
      <template #header>
        <div class="card-header">
          <span class="title">K8s 配置管理</span>
          <div class="header-actions">
            <ClusterSelector
              v-model="selectedClusterId"
              @change="handleClusterChange"
            />
            <NamespaceSelector
              v-model="queryParams.namespace"
              :cluster-id="selectedClusterId"
              @change="handleNamespaceChange"
            />
          </div>
        </div>
      </template>

      <!-- 搜索表单 -->
      <div class="search-section">
        <el-form :inline="true" class="search-form">
          <el-form-item label="配置资源名称">
            <el-input
              v-model="searchKeyword"
              placeholder="请输入名称"
              clearable
              size="small"
              style="width: 200px"
              @keyup.enter="resetSearch"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :icon="Search" size="small" @click="resetSearch">
              搜索
            </el-button>
            <el-button :icon="Refresh" size="small" @click="handleRefresh">
              刷新
            </el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- 配置资源表格 -->
      <el-tabs v-model="activeTab" @tab-change="handleTabChange" class="config-tabs">
        <!-- ConfigMap 标签页 -->
        <el-tab-pane label="ConfigMap" name="configmap">
          <div class="tab-content">
            <div class="content-header">
              <span class="resource-count">共 {{ filteredConfigMapList.length }} 个 ConfigMap</span>
              <el-button type="primary" :icon="Plus" size="small" @click="handleCreateConfigMap">
                创建 ConfigMap
              </el-button>
            </div>

            <el-table
              :data="filteredConfigMapList"
              v-loading="loading"
              element-loading-text="加载中..."
              class="resource-table"
              empty-text="暂无 ConfigMap 资源"
            >
              <el-table-column prop="name" label="名称" min-width="150">
                <template #default="{ row }">
                  <div class="resource-name">
                    <el-icon class="resource-icon"><Setting /></el-icon>
                    <span class="resource-name-link" @click="handleViewConfigMap(row)">{{ row?.name || '-' }}</span>
                  </div>
                </template>
              </el-table-column>

              <el-table-column prop="namespace" label="命名空间" width="120">
                <template #default="{ row }">
                  <el-tag size="small" type="info">
                    {{ row?.namespace || '-' }}
                  </el-tag>
                </template>
              </el-table-column>

              <el-table-column label="数据键" min-width="200">
                <template #default="{ row }">
                  <div class="data-keys">
                    <span class="keys-text">{{ formatDataKeys(row?.data) }}</span>
                  </div>
                </template>
              </el-table-column>

              <el-table-column label="标签" min-width="150">
                <template #default="{ row }">
                  <div class="labels-info">
                    <el-tag
                      v-for="(value, key) in (row?.labels || {})"
                      :key="key"
                      size="small"
                      type="primary"
                      class="label-tag"
                    >
                      {{ key }}={{ value }}
                    </el-tag>
                    <span v-if="!row?.labels || Object.keys(row?.labels).length === 0" class="no-labels">无标签</span>
                  </div>
                </template>
              </el-table-column>

              <el-table-column prop="createdTime" label="创建时间" width="180">
                <template #default="{ row }">
                  <span>{{ row?.createdTime ? new Date(row?.createdTime).toLocaleString() : '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column label="操作" width="160" fixed="right">
                <template #default="{ row }">
                  <div class="operation-buttons">
                    <el-tooltip content="编辑 YAML" placement="top">
                      <el-button
                        :icon="Edit"
                        size="small"
                        type="primary"
                        circle
                        @click="handleEditConfigMapYaml(row)"
                      />
                    </el-tooltip>
                    <el-tooltip content="删除" placement="top">
                      <el-button
                        :icon="Delete"
                        size="small"
                        type="danger"
                        circle
                        @click="handleDeleteConfigMap(row)"
                      />
                    </el-tooltip>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <!-- Secret 标签页 -->
        <el-tab-pane label="Secret" name="secret">
          <div class="tab-content">
            <div class="content-header">
              <span class="resource-count">共 {{ filteredSecretList.length }} 个 Secret</span>
              <el-button type="primary" :icon="Plus" size="small" @click="handleCreateSecret">
                创建 Secret
              </el-button>
            </div>

            <el-table
              :data="filteredSecretList"
              v-loading="loading"
              element-loading-text="加载中..."
              class="resource-table"
              empty-text="暂无 Secret 资源"
            >
              <el-table-column prop="name" label="名称" min-width="150">
                <template #default="{ row }">
                  <div class="resource-name">
                    <el-icon class="resource-icon"><Key /></el-icon>
                    <span class="resource-name-link" @click="handleViewSecret(row)">{{ row?.name || '-' }}</span>
                  </div>
                </template>
              </el-table-column>

              <el-table-column prop="namespace" label="命名空间" width="120">
                <template #default="{ row }">
                  <el-tag size="small" type="info">
                    {{ row?.namespace || '-' }}
                  </el-tag>
                </template>
              </el-table-column>

              <el-table-column prop="type" label="类型" width="150">
                <template #default="{ row }">
                  <el-tag
                    :type="getSecretTypeTag(row?.type)"
                    size="small"
                  >
                    {{ formatSecretType(row?.type) }}
                  </el-tag>
                </template>
              </el-table-column>

              <el-table-column label="数据键" min-width="200">
                <template #default="{ row }">
                  <div class="data-keys">
                    <span class="keys-text">
                      <el-icon class="secret-icon"><Lock /></el-icon>
                      {{ formatDataKeys(row?.data) }}
                    </span>
                  </div>
                </template>
              </el-table-column>

              <el-table-column label="标签" min-width="150">
                <template #default="{ row }">
                  <div class="labels-info">
                    <el-tag
                      v-for="(value, key) in (row?.labels || {})"
                      :key="key"
                      size="small"
                      type="primary"
                      class="label-tag"
                    >
                      {{ key }}={{ value }}
                    </el-tag>
                    <span v-if="!row?.labels || Object.keys(row?.labels).length === 0" class="no-labels">无标签</span>
                  </div>
                </template>
              </el-table-column>

              <el-table-column prop="createdTime" label="创建时间" width="180">
                <template #default="{ row }">
                  <span>{{ row?.createdTime ? new Date(row?.createdTime).toLocaleString() : '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column label="操作" width="160" fixed="right">
                <template #default="{ row }">
                  <div class="operation-buttons">
                    <el-tooltip content="编辑 YAML" placement="top">
                      <el-button
                        :icon="Edit"
                        size="small"
                        type="primary"
                        circle
                        @click="handleEditSecretYaml(row)"
                      />
                    </el-tooltip>
                    <el-tooltip content="删除" placement="top">
                      <el-button
                        :icon="Delete"
                        size="small"
                        type="danger"
                        circle
                        @click="handleDeleteSecret(row)"
                      />
                    </el-tooltip>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- ConfigMap 详情对话框 -->
    <el-dialog
      v-model="configMapDetailDialogVisible"
      title="ConfigMap 详情"
      width="70%"
      class="detail-dialog"
    >
      <div class="detail-content">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="名称">{{ currentConfigMapForDetail.name }}</el-descriptions-item>
          <el-descriptions-item label="命名空间">{{ currentConfigMapForDetail.namespace }}</el-descriptions-item>
          <el-descriptions-item label="创建时间" :span="2">
            {{ currentConfigMapForDetail.createdTime ? new Date(currentConfigMapForDetail.createdTime).toLocaleString() : '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="标签" :span="2">
            <div class="labels-info">
              <el-tag
                v-for="(value, key) in (currentConfigMapForDetail.labels || {})"
                :key="key"
                size="small"
                type="primary"
                class="label-tag"
              >
                {{ key }}={{ value }}
              </el-tag>
              <span v-if="!currentConfigMapForDetail.labels || Object.keys(currentConfigMapForDetail.labels).length === 0">无标签</span>
            </div>
          </el-descriptions-item>
        </el-descriptions>

        <div v-if="currentConfigMapForDetail.data" class="data-section">
          <h4 class="section-title">配置数据</h4>
          <div class="data-list">
            <div
              v-for="(value, key) in currentConfigMapForDetail.data"
              :key="key"
              class="data-item"
            >
              <div class="data-key">{{ key }}</div>
              <pre class="data-value">{{ value }}</pre>
            </div>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- Secret 详情对话框 -->
    <el-dialog
      v-model="secretDetailDialogVisible"
      title="Secret 详情"
      width="70%"
      class="detail-dialog"
    >
      <div class="detail-content">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="名称">{{ currentSecretForDetail.name }}</el-descriptions-item>
          <el-descriptions-item label="命名空间">{{ currentSecretForDetail.namespace }}</el-descriptions-item>
          <el-descriptions-item label="类型">
            <el-tag
              :type="getSecretTypeTag(currentSecretForDetail.type)"
              size="small"
            >
              {{ formatSecretType(currentSecretForDetail.type) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">
            {{ currentSecretForDetail.createdTime ? new Date(currentSecretForDetail.createdTime).toLocaleString() : '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="标签" :span="2">
            <div class="labels-info">
              <el-tag
                v-for="(value, key) in (currentSecretForDetail.labels || {})"
                :key="key"
                size="small"
                type="primary"
                class="label-tag"
              >
                {{ key }}={{ value }}
              </el-tag>
              <span v-if="!currentSecretForDetail.labels || Object.keys(currentSecretForDetail.labels).length === 0">无标签</span>
            </div>
          </el-descriptions-item>
        </el-descriptions>

        <div v-if="currentSecretForDetail.data" class="data-section">
          <h4 class="section-title">
            <el-icon class="secret-icon"><Lock /></el-icon>
            敏感数据 (已脱敏)
          </h4>
          <div class="data-list">
            <div
              v-for="(value, key) in currentSecretForDetail.data"
              :key="key"
              class="data-item secret-item"
            >
              <div class="data-key">{{ key }}</div>
              <div class="data-value secret-value">
                <el-icon><Lock /></el-icon>
                *** ({{ String(value).length }} 字符)
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- ConfigMap YAML 编辑对话框 -->
    <PodYamlDialog
      :visible="configMapYamlDialogVisible"
      :yaml-content="currentConfigMapYaml"
      :resource-name="currentConfigMapForDetail.name || 'new-configmap'"
      :resource-type="'ConfigMap'"
      :editable="true"
      @update:visible="configMapYamlDialogVisible = $event"
      @close="configMapYamlDialogVisible = false"
      @save="handleConfigMapYamlSave"
    />

    <!-- Secret YAML 编辑对话框 -->
    <PodYamlDialog
      :visible="secretYamlDialogVisible"
      :yaml-content="currentSecretYaml"
      :resource-name="currentSecretForDetail.name || 'new-secret'"
      :resource-type="'Secret'"
      :editable="true"
      @update:visible="secretYamlDialogVisible = $event"
      @close="secretYamlDialogVisible = false"
      @save="handleSecretYamlSave"
    />
  </div>
</template>

<style scoped>
.k8s-config-management {
  padding: 20px;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.config-card {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.title {
  font-size: 20px;
  font-weight: 600;
  color: #2c3e50;
  background: linear-gradient(45deg, #667eea, #764ba2);
  background-clip: text;
  -webkit-text-fill-color: transparent;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.search-section {
  margin-bottom: 20px;
  padding: 16px;
  background: rgba(255, 255, 255, 0.5);
  border-radius: 12px;
  border: 1px solid rgba(103, 126, 234, 0.1);
}

.search-form {
  margin: 0;
}

/* 按钮样式 - 与k8s-clusters.vue保持一致 */
.el-button {
  border-radius: 8px;
  font-weight: 500;
  transition: all 0.3s ease;
}

.el-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

/* 输入框样式 - 与k8s-clusters.vue保持一致 */
.el-input :deep(.el-input__wrapper),
.el-select :deep(.el-input__wrapper) {
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(103, 126, 234, 0.2);
  border-radius: 8px;
  box-shadow: none;
  transition: all 0.3s ease;
}

.el-input :deep(.el-input__wrapper):hover,
.el-select :deep(.el-input__wrapper):hover {
  border-color: #c0c4cc;
}

.el-input :deep(.el-input__wrapper.is-focus),
.el-select :deep(.el-input__wrapper.is-focus) {
  border-color: #667eea;
  box-shadow: 0 0 0 2px rgba(103, 126, 234, 0.2);
  background: rgba(255, 255, 255, 1);
}

.el-input :deep(.el-input__inner),
.el-select :deep(.el-input__inner) {
  background: transparent;
  border: none;
  color: #2c3e50;
}

/* 标签样式 */
.el-tag {
  font-weight: 500;
  border-radius: 8px;
  border: none;
}

.config-tabs {
  margin-top: 20px;
}

.config-tabs :deep(.el-tabs__header) {
  margin-bottom: 20px;
}

.config-tabs :deep(.el-tabs__item) {
  font-weight: 500;
  color: #606266;
}

.config-tabs :deep(.el-tabs__item.is-active) {
  color: #409EFF;
  font-weight: 600;
}

.tab-content {
  padding: 0;
}

.content-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 12px 0;
}

.resource-count {
  font-size: 14px;
  color: #606266;
  font-weight: 500;
}

.resource-table {
  border-radius: 8px;
  overflow: hidden;
}

.resource-table :deep(.el-table__header) {
  background: #f8f9fa;
}

.resource-table :deep(.el-table__row:hover) {
  background-color: #f5f7fa;
}

.resource-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.resource-icon {
  color: #409EFF;
  font-size: 16px;
}

.resource-name-link {
  color: #409EFF;
  cursor: pointer;
  font-weight: 500;
  transition: color 0.3s;
}

.resource-name-link:hover {
  color: #66b1ff;
  text-decoration: underline;
}

.data-keys {
  display: flex;
  align-items: center;
  gap: 6px;
}

.keys-text {
  color: #606266;
  font-size: 13px;
}

.secret-icon {
  color: #E6A23C;
  font-size: 14px;
}

.labels-info {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
}

.label-tag {
  font-size: 11px;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.no-labels {
  color: #909399;
  font-size: 12px;
  font-style: italic;
}

.operation-buttons {
  display: flex;
  gap: 8px;
  justify-content: center;
}

/* 详情对话框样式 - 与k8s-clusters.vue保持一致 */
.detail-dialog :deep(.el-dialog) {
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
}

.detail-dialog :deep(.el-dialog__header) {
  background: linear-gradient(135deg, #667eea, #764ba2);
  color: white;
  border-top-left-radius: 16px;
  border-top-right-radius: 16px;
  padding: 20px 24px;
}

.detail-dialog :deep(.el-dialog__title) {
  color: white;
  font-weight: 600;
}

.detail-dialog :deep(.el-dialog__body) {
  padding: 24px;
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.data-section {
  margin-top: 20px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 16px 0;
  padding: 8px 0;
  border-bottom: 2px solid #f0f0f0;
  color: #2c3e50;
  font-size: 16px;
  font-weight: 600;
}

.data-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.data-item {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 12px;
  border: 1px solid #e9ecef;
}

.data-item.secret-item {
  background: #fff7e6;
  border-color: #ffd591;
}

.data-key {
  font-weight: 600;
  color: #409EFF;
  margin-bottom: 8px;
  font-size: 14px;
}

.data-value {
  margin: 0;
  color: #2c3e50;
  font-size: 13px;
  line-height: 1.5;
  background: transparent;
  white-space: pre-wrap;
  word-break: break-word;
}

.data-value.secret-value {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #E6A23C;
  font-weight: 500;
}

/* 加载动画样式 */
.el-loading-mask {
  background-color: rgba(103, 126, 234, 0.1);
  backdrop-filter: blur(4px);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .k8s-config-management {
    padding: 12px;
  }

  .card-header {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }

  .header-actions {
    justify-content: flex-end;
  }

  .detail-content {
    gap: 16px;
  }

  .resource-table :deep(.el-table) {
    font-size: 12px;
  }

  .operation-buttons {
    flex-direction: column;
    gap: 4px;
  }

  .search-section {
    padding: 12px;
  }

  .search-form {
    flex-direction: column;
    align-items: stretch;
  }

  .search-form :deep(.el-form-item) {
    margin-bottom: 8px;
  }

  .data-list {
    gap: 8px;
  }

  .data-item {
    padding: 8px;
  }

  .labels-info {
    gap: 2px;
  }

  .label-tag {
    max-width: 100px;
  }
}
</style>
