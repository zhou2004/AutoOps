<template>
  <el-select
    :model-value="modelValue"
    placeholder="请选择集群"
    size="small"
    style="width: 200px; margin-right: 12px"
    @change="handleChange"
    :loading="!clusterList.length"
  >
    <el-option
      v-for="cluster in clusterList"
      :key="cluster.id"
      :label="`${cluster.name} ${getClusterStatusText(cluster.status)}`"
      :value="cluster.id"
      :disabled="cluster.status !== 2"
    >
      <div class="cluster-option">
        <span class="cluster-name">{{ cluster.name }}</span>
        <el-tag
          :type="getClusterStatusTag(cluster.status)"
          size="small"
          class="cluster-status-tag"
        >
          {{ getClusterStatusText(cluster.status) }}
        </el-tag>
      </div>
    </el-option>
  </el-select>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import k8sApi from '@/api/k8s'

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:modelValue', 'change'])

const clusterList = ref([])

// 获取集群列表
const fetchClusterList = async () => {
  try {
    const response = await k8sApi.getClusterList()
    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      const clusters = responseData.data?.list || responseData.data || []
      clusterList.value = clusters.map(cluster => ({
        id: cluster.id,
        name: cluster.name,
        status: cluster.status
      }))

      if (clusterList.value.length > 0 && !props.modelValue) {
        const onlineCluster = clusterList.value.find(cluster => cluster.status === 2)
        const selectedId = onlineCluster ? onlineCluster.id : clusterList.value[0].id
        emit('update:modelValue', selectedId)
        emit('change', selectedId)
      }

      console.log('集群列表加载成功:', clusterList.value)
    } else {
      ElMessage.error(responseData.message || '获取集群列表失败')
    }
  } catch (error) {
    console.error('获取集群列表失败:', error)
    ElMessage.warning('无法获取集群列表，请检查后端服务')
  }
}

const handleChange = (value) => {
  emit('update:modelValue', value)
  emit('change', value)
}

// 集群状态相关方法
const getClusterStatusTag = (status) => {
  const tagMap = {
    1: 'info',
    2: 'success',
    3: 'warning',
    0: 'danger'
  }
  return tagMap[status] || 'info'
}

const getClusterStatusText = (status) => {
  const textMap = {
    0: '离线',
    1: '连接中',
    2: '在线',
    3: '异常'
  }
  return textMap[status] || '未知'
}

onMounted(() => {
  fetchClusterList()
})
</script>

<style scoped>
/* 集群选择样式 */
.cluster-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}
.cluster-name {
  font-weight: 500;
  color: #2c3e50;
}
.cluster-status-tag {
  margin-left: 8px;
}
</style>