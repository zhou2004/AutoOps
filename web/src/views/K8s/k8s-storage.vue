<script setup>
import { tr } from "@/locales";

import { ref, reactive, onMounted, onActivated, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus,
  Edit,
  Delete,
  View,
  Document,
  Refresh,
  Connection,
  DataLine,
  Cpu,
  Search,
  Monitor,
  Files,
  Rank,
  CopyDocument,
  Box,
  Setting,
  Warning,
  Odometer,
  InfoFilled,
  FolderOpened
} from '@element-plus/icons-vue'
import k8sApi from '@/api/k8s'
import { cephApi } from '@/api/ceph'
import CephMaintenance from './ceph/CephMaintenance.vue'
import PodYamlDialog from './pods/PodYamlDialog.vue'
import ClusterSelector from './pods/ClusterSelector.vue'
import NamespaceSelector from './pods/NamespaceSelector.vue'

// 基础状态
const loading = ref(false)
const activeTab = ref('pvc')
const searchKeyword = ref('')

// 集群和命名空间状态
const selectedClusterId = ref('')
const queryParams = reactive({
  namespace: 'default'
})

// 存储资源数据状态
const pvcList = ref([])
const pvList = ref([])
const storageClassList = ref([])

// ========== Ceph 分布式存储状态 ==========
const cephLoading = ref(false)
const cephStatus = ref(null)
const cephPools = ref([])
const cephOSDs = ref([])
const cephSubTab = ref('pool')
const cephView = ref('overview')
const cephCanWrite = ref(false)

// 新增数据
const rbdImages = ref([])
const cephFSList = ref([])
const cephHealth = ref(null)
const cephVersions = ref(null)
const cephPGStat = ref(null)

// 副本调整对话框状态
const sizeDialogVisible = ref(false)
const sizeForm = reactive({
  poolName: '',
  poolApp: '',
  size: 3
})

// Ceph Pool 操作对话框状态
const createPoolDialogVisible = ref(false)
const createPoolForm = reactive({
  name: '',
  pgNum: 32,
  size: 3,
  application: 'rbd'
})
const pgNumDialogVisible = ref(false)
const pgNumForm = reactive({ poolName: '', pgNum: 32 })
const quotaDialogVisible = ref(false)
const quotaForm = reactive({ poolName: '', maxBytes: 0, maxObjects: 0, applyBytes: false, applyObjects: false })
const compressDialogVisible = ref(false)
const compressForm = reactive({ poolName: '', algorithm: 'lz4', mode: 'passive' })

// Ceph RBD 操作对话框状态
const rbdCreateDialogVisible = ref(false)
const rbdCreateForm = reactive({ pool: 'replicapool', name: '', size: '10G' })
const rbdResizeDialogVisible = ref(false)
const rbdResizeForm = reactive({ pool: '', image: '', size: '', allowShrink: false })
const rbdDetailDialogVisible = ref(false)
const rbdDetail = ref({})
const rbdSnapshots = ref([])
const rbdSnapCreateDialogVisible = ref(false)
const rbdSnapCreateForm = reactive({ pool: '', image: '', snapName: '' })
const rbdCloneDialogVisible = ref(false)
const rbdCloneForm = reactive({ pool: '', image: '', snapName: '', targetPool: 'replicapool', targetName: '' })

// OSD 操作
const osdReweightDialogVisible = ref(false)
const osdReweightForm = reactive({ id: 0, weight: 1 })

// Ceph 计算属性：集群总容量/已用/可用（MB）
const cephTotalBytes = computed(() => {
  return (cephStatus.value?.totalBytes || 0)
})
const cephUsedBytes = computed(() => {
  return (cephStatus.value?.usedBytes || 0)
})
const cephUsedPercent = computed(() => {
  const total = cephTotalBytes.value
  if (!total) return 0
  return Math.round((cephUsedBytes.value / total) * 1000) / 10
})

// 对话框状态
const pvcYamlDialogVisible = ref(false)
const pvYamlDialogVisible = ref(false)
const storageClassYamlDialogVisible = ref(false)

// 详情对话框状态
const pvcDetailDialogVisible = ref(false)
const pvDetailDialogVisible = ref(false)
const storageClassDetailDialogVisible = ref(false)

// 当前操作的资源
const currentPVCYaml = ref('')
const currentPVYaml = ref('')
const currentStorageClassYaml = ref('')
const currentResourceName = ref('')
const currentResourceType = ref('')

// 扩容对话框状态
const expandPVCDialogVisible = ref(false)
const expandPVCForm = reactive({
  newSize: '',
  currentSize: ''
})
const expandTargetPVC = ref({})

// 克隆/备份对话框状态
const clonePVCDialogVisible = ref(false)
const clonePVCForm = reactive({
  targetName: '',
  targetNamespace: '',
  newSize: ''
})
const cloneSourcePVC = ref({})

// 当前查看的资源详情
const currentPVCForDetail = ref({})
const currentPVForDetail = ref({})
const currentStorageClassForDetail = ref({})

// 过滤后的列表
const filteredPVCList = computed(() => {
  const list = Array.isArray(pvcList.value) ? pvcList.value : []
  if (!searchKeyword.value) return list
  return list.filter(item =>
    item.name?.toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
    item.status?.toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
    item.storageClass?.toLowerCase().includes(searchKeyword.value.toLowerCase())
  )
})

const filteredPVList = computed(() => {
  const list = Array.isArray(pvList.value) ? pvList.value : []
  if (!searchKeyword.value) return list
  return list.filter(item =>
    item.name?.toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
    item.status?.toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
    item.storageClass?.toLowerCase().includes(searchKeyword.value.toLowerCase())
  )
})

const filteredStorageClassList = computed(() => {
  const list = Array.isArray(storageClassList.value) ? storageClassList.value : []
  if (!searchKeyword.value) return list
  return list.filter(item =>
    item.name?.toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
    item.provisioner?.toLowerCase().includes(searchKeyword.value.toLowerCase())
  )
})

// 分页
const pvcCurrentPage = ref(1)
const pvcPageSize = ref(20)
const pvcTotal = ref(0)
const pvCurrentPage = ref(1)
const pvPageSize = ref(20)
const pvTotal = ref(0)
const scCurrentPage = ref(1)
const scPageSize = ref(20)
const scTotal = ref(0)

// ===================== 界面化创建状态 =====================

// 界面创建 PVC 对话框
const createPVCDialogVisible = ref(false)
const createPVCForm = reactive({
  name: '',
  storageClass: '',
  storageSize: '10Gi',
  accessMode: 'ReadWriteOnce',
  volumeMode: 'Filesystem'
})

// 界面创建 PV 对话框
const createPVDialogVisible = ref(false)
const createPVForm = reactive({
  name: '',
  volumeType: 'hostPath', // hostPath/nfs/local/csi/iscsi
  capacity: '10Gi',
  accessMode: 'ReadWriteOnce',
  reclaimPolicy: 'Retain',
  storageClass: '',
  volumeMode: 'Filesystem',
  // hostPath
  hostPath: '/mnt/data',
  hostPathType: 'DirectoryOrCreate',
  // NFS
  nfsServer: '',
  nfsPath: '',
  nfsReadOnly: false,
  // Local
  localPath: '/mnt/local',
  localFsType: 'ext4',
  localNode: '',
  // CSI
  csiDriver: '',
  csiVolumeHandle: '',
  csiFsType: 'ext4'
})

// 节点列表（用于 Local 卷类型选择绑定节点）
const nodeList = ref([])

// 判断所选存储类是否为动态供给（CSI），用于提示用户应使用"创建 PVC"
const isDynamicStorageClass = computed(() => {
  if (!createPVForm.storageClass) return false
  const sc = storageClassList.value.find(s => s.name === createPVForm.storageClass)
  // no-provisioner 为静态供给，其余（topolvm.io 等）均为动态供给
  return !!sc && sc.provisioner !== 'kubernetes.io/no-provisioner'
})

// 界面创建 StorageClass 对话框
const createSCDialogVisible = ref(false)
const createSCForm = reactive({
  name: '',
  provisioner: 'kubernetes.io/no-provisioner',
  reclaimPolicy: 'Delete',
  volumeBindingMode: 'Immediate',
  allowVolumeExpansion: false,
  parameters: ''
})

// ===================== 存储迁移状态 =====================

// 卷到卷迁移对话框
const migrateDialogVisible = ref(false)
const migrateForm = reactive({
  sourcePvc: '',
  sourceNs: '',
  targetPvc: '',
  targetNs: '',
  targetSize: '',
  targetSc: '',
  createIfAbsent: false
})

// 一键换卷对话框
const switchDialogVisible = ref(false)
const switchForm = reactive({
  workloadType: 'deployment',
  workloadName: '',
  oldPvcName: '',
  newPvcName: ''
})

const paginatedPVCList = computed(() => {
  const start = (pvcCurrentPage.value - 1) * pvcPageSize.value
  const end = start + pvcPageSize.value
  return filteredPVCList.value.slice(start, end)
})
const paginatedPVList = computed(() => {
  const start = (pvCurrentPage.value - 1) * pvPageSize.value
  const end = start + pvPageSize.value
  return filteredPVList.value.slice(start, end)
})
const paginatedStorageClassList = computed(() => {
  const start = (scCurrentPage.value - 1) * scPageSize.value
  const end = start + scPageSize.value
  return filteredStorageClassList.value.slice(start, end)
})

watch(filteredPVCList, () => { pvcTotal.value = filteredPVCList.value.length })
watch(filteredPVList, () => { pvTotal.value = filteredPVList.value.length })
watch(filteredStorageClassList, () => { scTotal.value = filteredStorageClassList.value.length })
watch(searchKeyword, () => { pvcCurrentPage.value = 1; pvCurrentPage.value = 1; scCurrentPage.value = 1 })

const handlePvcSizeChange = (val) => { pvcPageSize.value = val; pvcCurrentPage.value = 1 }
const handlePvcCurrentChange = (val) => { pvcCurrentPage.value = val }
const handlePvSizeChange = (val) => { pvPageSize.value = val; pvCurrentPage.value = 1 }
const handlePvCurrentChange = (val) => { pvCurrentPage.value = val }
const handleScSizeChange = (val) => { scPageSize.value = val; scCurrentPage.value = 1 }
const handleScCurrentChange = (val) => { scCurrentPage.value = val }

// 处理集群选择变化
const handleClusterChange = (clusterId) => {
  selectedClusterId.value = clusterId
  console.log('集群选择变化:', clusterId)
  if (clusterId && queryParams.namespace) {
    loadAllStorageResources()
  }
}

// 处理命名空间选择变化
const handleNamespaceChange = (namespace) => {
  queryParams.namespace = namespace
  console.log('命名空间选择变化:', namespace)
  if (selectedClusterId.value && namespace) {
    loadAllStorageResources()
  }
}

// 获取命名空间列表
const fetchNamespaceList = async () => {
  if (!selectedClusterId.value) return

  try {
    const response = await k8sApi.getNamespaces(selectedClusterId.value)
    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      namespaceList.value = responseData.data || []
      // 如果当前选中的命名空间不在新列表中，重置为default
      if (!namespaceList.value.some(ns => ns.name === queryParams.namespace)) {
        queryParams.namespace = namespaceList.value.length > 0 ? namespaceList.value[0].name : 'default'
      }
    } else {
      ElMessage.error(responseData.message || tr('获取命名空间列表失败'))
    }
  } catch (error) {
    console.error('获取命名空间列表失败:', error)
    ElMessage.error(tr('获取命名空间列表失败'))
  }
}

// 加载所有存储资源
const loadAllStorageResources = async () => {
  if (!selectedClusterId.value) {
    console.warn('集群ID为空，无法加载存储资源')
    return
  }

  console.log('开始加载存储资源，集群ID:', selectedClusterId.value, '命名空间:', queryParams.namespace)

  loading.value = true
  try {
    // 并发加载所有存储资源
    await Promise.all([
      fetchPVCList(),
      fetchPVList(),
      fetchStorageClassList(),
      loadCephData()
    ])
  } catch (error) {
    console.error('加载存储资源失败:', error)
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
  // 切换/进入 Ceph 标签页时刷新数据
  if (tabName === 'ceph' && selectedClusterId.value) {
    loadCephData()
  }
}

// 获取PVC列表
const fetchPVCList = async () => {
  if (!selectedClusterId.value || !queryParams.namespace) {
    console.warn('集群ID或命名空间为空，无法获取 PVC 列表')
    return
  }

  try {
    console.log('正在获取 PVC 列表，集群ID:', selectedClusterId.value, '命名空间:', queryParams.namespace)

    const response = await k8sApi.getPVCList(selectedClusterId.value, queryParams.namespace)
    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      const rawData = responseData.data?.pvcs || responseData.data
      pvcList.value = Array.isArray(rawData) ? rawData : []
      console.log('获取到 PVC 列表:', pvcList.value.length, '个')
    } else {
      ElMessage.error(responseData.message || '获取 PVC 列表失败')
      pvcList.value = []
    }
  } catch (error) {
    console.error('获取 PVC 列表失败:', error)
    ElMessage.error(tr('获取 PVC 列表失败，请检查网络连接'))
    pvcList.value = []
  }
}

// 获取PV列表
const fetchPVList = async () => {
  if (!selectedClusterId.value) {
    console.warn('集群ID为空，无法获取 PV 列表')
    return
  }

  try {
    console.log('正在获取 PV 列表，集群ID:', selectedClusterId.value)

    const response = await k8sApi.getPVList(selectedClusterId.value)
    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      const rawData = responseData.data?.pvs || responseData.data
      pvList.value = Array.isArray(rawData) ? rawData : []
      console.log('获取到 PV 列表:', pvList.value.length, '个')
    } else {
      ElMessage.error(responseData.message || '获取 PV 列表失败')
      pvList.value = []
    }
  } catch (error) {
    console.error('获取 PV 列表失败:', error)
    ElMessage.error(tr('获取 PV 列表失败，请检查网络连接'))
    pvList.value = []
  }
}

// 获取StorageClass列表
const fetchStorageClassList = async () => {
  if (!selectedClusterId.value) {
    console.warn('集群ID为空，无法获取 StorageClass 列表')
    return
  }

  try {
    console.log('正在获取 StorageClass 列表，集群ID:', selectedClusterId.value)

    const response = await k8sApi.getStorageClassList(selectedClusterId.value)
    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      const rawData = responseData.data?.storageClasses || responseData.data
      storageClassList.value = Array.isArray(rawData) ? rawData : []
      console.log('获取到 StorageClass 列表:', storageClassList.value.length, '个')
    } else {
      ElMessage.error(responseData.message || '获取 StorageClass 列表失败')
      storageClassList.value = []
    }
  } catch (error) {
    console.error('获取 StorageClass 列表失败:', error)
    ElMessage.error(tr('获取 StorageClass 列表失败，请检查网络连接'))
    storageClassList.value = []
  }
}

// 刷新数据
const handleRefresh = () => {
  loadAllStorageResources()
}

let cephEpoch = 0
const cephQuerySequence = {}
watch(selectedClusterId, () => {
  cephEpoch++
  cephCanWrite.value = false
  cephStatus.value = null
  cephPools.value = []
  cephOSDs.value = []
  rbdImages.value = []
  cephFSList.value = []
  cephHealth.value = null
  cephVersions.value = null
  cephPGStat.value = null
  rbdDetail.value = {}
  rbdSnapshots.value = []
  ;[sizeDialogVisible, createPoolDialogVisible, pgNumDialogVisible, quotaDialogVisible,
    compressDialogVisible, rbdCreateDialogVisible, rbdResizeDialogVisible,
    rbdDetailDialogVisible, rbdSnapCreateDialogVisible, rbdCloneDialogVisible,
    osdReweightDialogVisible].forEach(dialog => { dialog.value = false })
}, { flush: 'sync' })

// ========== Ceph 数据加载 ==========
const fetchCephStatus = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  const sequence = (cephQuerySequence.fetchCephStatus || 0) + 1
  cephQuerySequence.fetchCephStatus = sequence
  if (!selectedClusterId.value) return
  try {
    const response = await k8sApi.getCephStatus(requestCluster)
    if (sequence !== cephQuerySequence.fetchCephStatus || requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      cephStatus.value = responseData.data || null
    } else {
      cephStatus.value = null
      if (responseData.message?.includes('rook-ceph-tools')) {
        console.warn('Ceph toolbox 未部署:', responseData.message)
      }
    }
  } catch (error) {
    if (sequence !== cephQuerySequence.fetchCephStatus || requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    console.error('获取 Ceph 状态失败:', error)
    cephStatus.value = null
  }
}

const fetchCephPools = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  const sequence = (cephQuerySequence.fetchCephPools || 0) + 1
  cephQuerySequence.fetchCephPools = sequence
  if (!selectedClusterId.value) return
  try {
    const response = await k8sApi.getCephPools(requestCluster)
    if (sequence !== cephQuerySequence.fetchCephPools || requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      cephPools.value = Array.isArray(responseData.data) ? responseData.data : []
    } else {
      cephPools.value = []
    }
  } catch (error) {
    if (sequence !== cephQuerySequence.fetchCephPools || requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    console.error('获取 Ceph Pool 列表失败:', error)
    cephPools.value = []
  }
}

const fetchCephOSDs = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  const sequence = (cephQuerySequence.fetchCephOSDs || 0) + 1
  cephQuerySequence.fetchCephOSDs = sequence
  if (!selectedClusterId.value) return
  try {
    const response = await k8sApi.getCephOSDs(requestCluster)
    if (sequence !== cephQuerySequence.fetchCephOSDs || requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      cephOSDs.value = Array.isArray(responseData.data) ? responseData.data : []
    } else {
      cephOSDs.value = []
    }
  } catch (error) {
    if (sequence !== cephQuerySequence.fetchCephOSDs || requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    console.error('获取 Ceph OSD 列表失败:', error)
    cephOSDs.value = []
  }
}

const loadCephData = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  if (!selectedClusterId.value) return
  cephLoading.value = true
  cephCanWrite.value = false
  const requestedCluster = requestCluster
  cephApi.capabilities(requestedCluster).then(data => {
    if (requestEpoch === cephEpoch && selectedClusterId.value === requestedCluster) cephCanWrite.value = data.canWrite
  }).catch(() => {})
  try {
    await Promise.all([
      fetchCephStatus(),
      fetchCephPools(),
      fetchCephOSDs(),
      fetchCephRBDImages(),
      fetchCephFS(),
      fetchCephHealth(),
      fetchCephVersions(),
      fetchCephPGStat()
    ])
  } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    console.error('加载 Ceph 数据失败:', error)
  } finally {
    if (requestEpoch === cephEpoch) cephLoading.value = false
  }
}

// 新增数据加载
const fetchCephRBDImages = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  const sequence = (cephQuerySequence.fetchCephRBDImages || 0) + 1
  cephQuerySequence.fetchCephRBDImages = sequence
  if (!selectedClusterId.value) return
  try {
    const response = await k8sApi.getCephRBDImages(requestCluster)
    if (sequence !== cephQuerySequence.fetchCephRBDImages || requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    rbdImages.value = (responseData.code === 200 || responseData.success) && Array.isArray(responseData.data)
      ? responseData.data : []
  } catch (error) {
    if (sequence !== cephQuerySequence.fetchCephRBDImages || requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    console.error('获取 RBD 镜像失败:', error)
    rbdImages.value = []
  }
}

const fetchCephFS = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  const sequence = (cephQuerySequence.fetchCephFS || 0) + 1
  cephQuerySequence.fetchCephFS = sequence
  if (!selectedClusterId.value) return
  try {
    const response = await k8sApi.getCephFS(requestCluster)
    if (sequence !== cephQuerySequence.fetchCephFS || requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    cephFSList.value = (responseData.code === 200 || responseData.success) && Array.isArray(responseData.data)
      ? responseData.data : []
  } catch (error) {
    if (sequence !== cephQuerySequence.fetchCephFS || requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    console.error('获取 CephFS 失败:', error)
    cephFSList.value = []
  }
}

const fetchCephHealth = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  const sequence = (cephQuerySequence.fetchCephHealth || 0) + 1
  cephQuerySequence.fetchCephHealth = sequence
  if (!selectedClusterId.value) return
  try {
    const response = await k8sApi.getCephHealth(requestCluster)
    if (sequence !== cephQuerySequence.fetchCephHealth || requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    cephHealth.value = (responseData.code === 200 || responseData.success) ? responseData.data : null
  } catch (error) {
    if (sequence !== cephQuerySequence.fetchCephHealth || requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    console.error('获取 Ceph 健康失败:', error)
    cephHealth.value = null
  }
}

const fetchCephVersions = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  const sequence = (cephQuerySequence.fetchCephVersions || 0) + 1
  cephQuerySequence.fetchCephVersions = sequence
  if (!selectedClusterId.value) return
  try {
    const response = await k8sApi.getCephVersions(requestCluster)
    if (sequence !== cephQuerySequence.fetchCephVersions || requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    cephVersions.value = (responseData.code === 200 || responseData.success) ? responseData.data : null
  } catch (error) {
    if (sequence !== cephQuerySequence.fetchCephVersions || requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    console.error('获取 Ceph 版本失败:', error)
    cephVersions.value = null
  }
}

const fetchCephPGStat = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  const sequence = (cephQuerySequence.fetchCephPGStat || 0) + 1
  cephQuerySequence.fetchCephPGStat = sequence
  if (!selectedClusterId.value) return
  try {
    const response = await k8sApi.getCephPGStat(requestCluster)
    if (sequence !== cephQuerySequence.fetchCephPGStat || requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    cephPGStat.value = (responseData.code === 200 || responseData.success) ? responseData.data : null
  } catch (error) {
    if (sequence !== cephQuerySequence.fetchCephPGStat || requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    console.error('获取 Ceph PG 状态失败:', error)
    cephPGStat.value = null
  }
}

// ========== Ceph Pool 操作 ==========

// 打开副本调整对话框
const openSizeDialog = (row) => {
  sizeForm.poolName = row.name
  sizeForm.poolApp = row.application || '-'
  sizeForm.size = row.size || 3
  sizeDialogVisible.value = true
}

// 提交副本数调整
const submitPoolSize = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  if (!sizeForm.poolName || !sizeForm.size) {
    ElMessage.warning(tr('请选择副本数'))
    return
  }
  try {
    const response = await k8sApi.updateCephPoolSize(requestCluster, sizeForm.poolName, sizeForm.size)
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(`Pool ${sizeForm.poolName} 副本数已调整为 ${sizeForm.size}`)
      sizeDialogVisible.value = false
      fetchCephPools()
      fetchCephStatus()
    } else {
      ElMessage.error(responseData.message || '调整副本数失败')
    }
  } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    console.error('调整副本数失败:', error)
    ElMessage.error('调整副本数失败: ' + (error.response?.data?.message || error.message))
  }
}

// 创建 Pool
const submitCreatePool = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  if (!createPoolForm.name) {
    ElMessage.warning(tr('请输入 Pool 名称'))
    return
  }
  try {
    const response = await k8sApi.createCephPool(requestCluster, createPoolForm)
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(responseData.data?.message || 'Pool 创建成功')
      createPoolDialogVisible.value = false
      createPoolForm.name = ''
      fetchCephPools()
    } else {
      ElMessage.error(responseData.message || '创建 Pool 失败')
    }
  } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    ElMessage.error('创建 Pool 失败: ' + (error.response?.data?.message || error.message))
  }
}

// 删除 Pool
const handleDeletePool = (row) => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  ElMessageBox.confirm(
    `确定要删除 Pool "${row.name}" 吗？该操作会删除其中所有数据且不可恢复！`,
    tr('危险操作确认'),
    { type: 'warning', confirmButtonText: tr('确认删除'), cancelButtonText: tr('取消') }
  ).then(async () => {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    try {
      const response = await k8sApi.deleteCephPool(requestCluster, row.name)
      if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
      if (responseData.code === 200 || responseData.success) {
        ElMessage.success(`Pool ${row.name} 已删除`)
        fetchCephPools()
      } else {
        ElMessage.error(responseData.message || '删除 Pool 失败')
      }
    } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
      ElMessage.error('删除 Pool 失败: ' + (error.response?.data?.message || error.message))
    }
  }).catch(() => {})
}

// 调整 PG 数
const openPGNumDialog = (row) => {
  pgNumForm.poolName = row.name
  pgNumForm.pgNum = row.pgNum || 32
  pgNumDialogVisible.value = true
}
const submitPGNum = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  try {
    const response = await k8sApi.updateCephPoolPGNum(requestCluster, pgNumForm.poolName, pgNumForm.pgNum)
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(responseData.data?.message || 'PG 数已调整')
      pgNumDialogVisible.value = false
      fetchCephPools()
    } else {
      ElMessage.error(responseData.message || '调整 PG 数失败')
    }
  } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    ElMessage.error('调整 PG 数失败: ' + (error.response?.data?.message || error.message))
  }
}

// 设置配额
const openQuotaDialog = (row) => {
  quotaForm.poolName = row.name
  quotaForm.applyBytes = false
  quotaForm.applyObjects = false
  quotaForm.maxBytes = 0
  quotaForm.maxObjects = 0
  quotaDialogVisible.value = true
}
const submitQuota = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  if (!quotaForm.applyBytes && !quotaForm.applyObjects) { ElMessage.warning(tr('请勾选要修改的配额')); return }
  const maxBytes = quotaForm.maxBytes ? Number(quotaForm.maxBytes) * 1024 * 1024 * 1024 : 0 // GB -> bytes
  try {
    const response = await k8sApi.updateCephPoolQuota(requestCluster, quotaForm.poolName, {
      ...(quotaForm.applyBytes ? { maxBytes } : {}),
      ...(quotaForm.applyObjects ? { maxObjects: Number(quotaForm.maxObjects || 0) } : {})
    })
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(responseData.data?.message || '配额已设置')
      quotaDialogVisible.value = false
    } else {
      ElMessage.error(responseData.message || '设置配额失败')
    }
  } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    ElMessage.error('设置配额失败: ' + (error.response?.data?.message || error.message))
  }
}

// 设置压缩
const openCompressDialog = (row) => {
  compressForm.poolName = row.name
  compressDialogVisible.value = true
}
const submitCompress = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  try {
    const response = await k8sApi.updateCephPoolCompression(requestCluster, compressForm.poolName, {
      algorithm: compressForm.algorithm,
      mode: compressForm.mode
    })
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(responseData.data?.message || '压缩配置已更新')
      compressDialogVisible.value = false
    } else {
      ElMessage.error(responseData.message || '设置压缩失败')
    }
  } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    ElMessage.error('设置压缩失败: ' + (error.response?.data?.message || error.message))
  }
}

// ========== Ceph RBD 操作 ==========

// 创建 RBD 镜像
const submitCreateRBD = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  if (!rbdCreateForm.name || !rbdCreateForm.size) {
    ElMessage.warning(tr('请输入镜像名称和容量'))
    return
  }
  try {
    const response = await k8sApi.createCephRBDImage(requestCluster, rbdCreateForm)
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(responseData.data?.message || 'RBD 镜像创建成功')
      rbdCreateDialogVisible.value = false
      rbdCreateForm.name = ''
      fetchCephRBDImages()
    } else {
      ElMessage.error(responseData.message || '创建 RBD 镜像失败')
    }
  } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    ElMessage.error('创建 RBD 镜像失败: ' + (error.response?.data?.message || error.message))
  }
}

// 查看 RBD 镜像详情 + 快照
const handleViewRBD = async (row) => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  try {
    const [detailResp, snapResp] = await Promise.all([
      k8sApi.getCephRBDImageDetail(requestCluster, row.pool, row.name),
      k8sApi.getCephRBDSnapshots(requestCluster, row.pool, row.name)
    ])
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const d = (detailResp.data || detailResp)
    const s = (snapResp.data || snapResp)
    if (d.code === 200 || d.success) {
      rbdDetail.value = d.data || row
    } else {
      rbdDetail.value = row
    }
    rbdSnapshots.value = (s.code === 200 || s.success) && Array.isArray(s.data) ? s.data : []
    rbdDetailDialogVisible.value = true
  } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    ElMessage.error('获取 RBD 详情失败: ' + (error.message || error))
  }
}

// 扩容 RBD
const openRBDResize = (row) => {
  rbdResizeForm.pool = row.pool
  rbdResizeForm.image = row.name
  rbdResizeForm.size = ''
  rbdResizeForm.allowShrink = false
  rbdResizeDialogVisible.value = true
}
const submitRBDResize = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  if (!rbdResizeForm.size) {
    ElMessage.warning(tr('请输入目标容量'))
    return
  }
  try {
    const response = await k8sApi.resizeCephRBDImage(requestCluster, rbdResizeForm.pool, rbdResizeForm.image, {
      size: rbdResizeForm.size,
      allowShrink: rbdResizeForm.allowShrink
    })
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(responseData.data?.message || '容量已调整')
      rbdResizeDialogVisible.value = false
      fetchCephRBDImages()
    } else {
      ElMessage.error(responseData.message || '调整容量失败')
    }
  } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    ElMessage.error('调整容量失败: ' + (error.response?.data?.message || error.message))
  }
}

// 删除 RBD 镜像
const handleDeleteRBD = (row) => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  ElMessageBox.confirm(
    `确定要删除 RBD 镜像 "${row.pool}/${row.name}" 吗？该操作会删除所有数据且不可恢复！`,
    tr('危险操作确认'),
    { type: 'warning', confirmButtonText: tr('确认删除'), cancelButtonText: tr('取消') }
  ).then(async () => {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    try {
      const response = await k8sApi.deleteCephRBDImage(requestCluster, row.pool, row.name)
      if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
      if (responseData.code === 200 || responseData.success) {
        ElMessage.success(responseData.data?.message || tr('镜像已删除'))
        fetchCephRBDImages()
      } else {
        ElMessage.error(responseData.message || '删除镜像失败')
      }
    } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
      ElMessage.error('删除镜像失败: ' + (error.response?.data?.message || error.message))
    }
  }).catch(() => {})
}

// 创建快照
const openSnapCreate = () => {
  rbdSnapCreateForm.pool = rbdDetail.value.pool
  rbdSnapCreateForm.image = rbdDetail.value.name
  rbdSnapCreateForm.snapName = ''
  rbdSnapCreateDialogVisible.value = true
}
const submitSnapCreate = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  if (!rbdSnapCreateForm.snapName) {
    ElMessage.warning(tr('请输入快照名称'))
    return
  }
  try {
    const response = await k8sApi.createCephRBDSnapshot(requestCluster, rbdSnapCreateForm.pool, rbdSnapCreateForm.image, {
      snapName: rbdSnapCreateForm.snapName
    })
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(responseData.data?.message || '快照已创建')
      rbdSnapCreateDialogVisible.value = false
      handleViewRBD(rbdDetail.value)
    } else {
      ElMessage.error(responseData.message || '创建快照失败')
    }
  } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    ElMessage.error('创建快照失败: ' + (error.response?.data?.message || error.message))
  }
}

// 删除快照
const handleDeleteSnap = (snap) => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  const targetImage = { ...rbdDetail.value }
  ElMessageBox.confirm(
    `确定要删除快照 "${targetImage.name}@${snap.name}" 吗？`,
    tr('删除快照'),
    { type: 'warning', confirmButtonText: tr('确认删除'), cancelButtonText: tr('取消') }
  ).then(async () => {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    try {
      const response = await k8sApi.deleteCephRBDSnapshot(requestCluster, targetImage.pool, targetImage.name, snap.name)
      if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
      if (responseData.code === 200 || responseData.success) {
        ElMessage.success(tr('快照已删除'))
        handleViewRBD(targetImage)
      } else {
        ElMessage.error(responseData.message || '删除快照失败')
      }
    } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
      ElMessage.error('删除快照失败: ' + (error.response?.data?.message || error.message))
    }
  }).catch(() => {})
}

// 回滚快照
const handleRollbackSnap = (snap) => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  const targetImage = { ...rbdDetail.value }
  ElMessageBox.confirm(
    `确定要将镜像 "${targetImage.name}" 回滚到快照 "${snap.name}" 吗？当前数据将被覆盖！`,
    tr('回滚确认'),
    { type: 'warning', confirmButtonText: tr('确认回滚'), cancelButtonText: tr('取消') }
  ).then(async () => {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    try {
      const response = await k8sApi.rollbackCephRBDImage(requestCluster, targetImage.pool, targetImage.name, snap.name)
      if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
      if (responseData.code === 200 || responseData.success) {
        ElMessage.success(responseData.data?.message || '回滚成功')
      } else {
        ElMessage.error(responseData.message || '回滚失败')
      }
    } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
      ElMessage.error('回滚失败: ' + (error.response?.data?.message || error.message))
    }
  }).catch(() => {})
}

// 克隆镜像
const openCloneDialog = (snap) => {
  rbdCloneForm.pool = rbdDetail.value.pool
  rbdCloneForm.image = rbdDetail.value.name
  rbdCloneForm.snapName = snap.name
  rbdCloneForm.targetPool = rbdDetail.value.pool
  rbdCloneForm.targetName = ''
  rbdCloneDialogVisible.value = true
}
const submitClone = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  if (!rbdCloneForm.targetName) {
    ElMessage.warning(tr('请输入目标镜像名称'))
    return
  }
  try {
    const response = await k8sApi.cloneCephRBDImage(requestCluster, rbdCloneForm.pool, rbdCloneForm.image, rbdCloneForm)
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(responseData.data?.message || '克隆成功')
      rbdCloneDialogVisible.value = false
      fetchCephRBDImages()
    } else {
      ElMessage.error(responseData.message || '克隆失败（源快照需先保护）')
    }
  } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    ElMessage.error('克隆失败: ' + (error.response?.data?.message || error.message))
  }
}

// ========== Ceph OSD 操作 ==========
const handleOSDAction = async (row, action) => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  try {
    const response = await k8sApi.setCephOSDStatus(requestCluster, row.id, action)
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(responseData.data?.message || `OSD.${row.id} ${action} 成功`)
      fetchCephOSDs()
      fetchCephStatus()
    } else {
      ElMessage.error(responseData.message || tr('操作失败'))
    }
  } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    ElMessage.error('操作失败: ' + (error.response?.data?.message || error.message))
  }
}

const openOSDReweight = (row) => {
  osdReweightForm.id = row.id
  osdReweightForm.weight = 1
  osdReweightDialogVisible.value = true
}
const submitOSDReweight = async () => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  try {
    const response = await k8sApi.reweightCephOSD(requestCluster, osdReweightForm.id, Number(osdReweightForm.weight))
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(responseData.data?.message || '权重已调整')
      osdReweightDialogVisible.value = false
      fetchCephOSDs()
    } else {
      ElMessage.error(responseData.message || '调整权重失败')
    }
  } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    ElMessage.error('调整权重失败: ' + (error.response?.data?.message || error.message))
  }
}

const handleDeleteOSD = (row) => {
  const requestCluster = selectedClusterId.value
  const requestEpoch = cephEpoch
  ElMessageBox.confirm(
    `确定要删除 OSD.${row.id} 吗？这会将 OSD 从集群彻底移除并触发数据重均衡，请确保其数据已安全迁移！`,
    tr('危险操作确认'),
    { type: 'warning', confirmButtonText: tr('确认删除'), cancelButtonText: tr('取消') }
  ).then(async () => {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    try {
      const response = await k8sApi.deleteCephOSD(requestCluster, row.id)
      if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
    const responseData = response.data || response
      if (responseData.code === 200 || responseData.success) {
        ElMessage.success(responseData.data?.message || `OSD.${row.id} 已删除`)
        fetchCephOSDs()
        fetchCephStatus()
      } else {
        ElMessage.error(responseData.message || '删除 OSD 失败')
      }
    } catch (error) {
    if (requestEpoch !== cephEpoch || requestCluster !== selectedClusterId.value) return
      ElMessage.error('删除 OSD 失败: ' + (error.response?.data?.message || error.message))
    }
  }).catch(() => {})
}

// 工具函数：格式化字节
const formatBytes = (bytes, digits = 2) => {
  if (!bytes && bytes !== 0) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  let i = 0
  let val = bytes
  while (val >= 1024 && i < units.length - 1) {
    val /= 1024
    i++
  }
  return val.toFixed(digits) + ' ' + units[i]
}

// PVC 操作
const handleCreatePVC = () => {
  if (!selectedClusterId.value) {
    ElMessage.warning(tr('请先选择集群'))
    return
  }
  if (!queryParams.namespace) {
    ElMessage.warning(tr('请先选择命名空间'))
    return
  }

  // 打开界面化创建对话框
  createPVCForm.name = ''
  createPVCForm.storageClass = ''
  createPVCForm.storageSize = '10Gi'
  createPVCForm.accessMode = 'ReadWriteOnce'
  createPVCForm.volumeMode = 'Filesystem'
  createPVCDialogVisible.value = true
}

// 提交界面化创建 PVC
const submitCreatePVC = async () => {
  if (!createPVCForm.name) {
    ElMessage.warning(tr('请输入 PVC 名称'))
    return
  }
  if (!createPVCForm.storageSize) {
    ElMessage.warning(tr('请输入存储容量'))
    return
  }
  try {
    const response = await k8sApi.createPVC(selectedClusterId.value, queryParams.namespace, {
      name: createPVCForm.name,
      storageClass: createPVCForm.storageClass || undefined,
      accessModes: [createPVCForm.accessMode],
      volumeMode: createPVCForm.volumeMode || undefined,
      resources: {
        requests: { storage: createPVCForm.storageSize }
      }
    })
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(tr('PVC 创建成功'))
      createPVCDialogVisible.value = false
      loadAllStorageResources()
    } else {
      ElMessage.error(responseData.message || 'PVC 创建失败')
    }
  } catch (error) {
    console.error('创建 PVC 失败:', error)
    ElMessage.error('创建 PVC 失败: ' + (error.response?.data?.message || error.message))
  }
}

// 通过 YAML 创建 PVC
const handleCreatePVCYaml = () => {
  if (!selectedClusterId.value) {
    ElMessage.warning(tr('请先选择集群'))
    return
  }
  if (!queryParams.namespace) {
    ElMessage.warning(tr('请先选择命名空间'))
    return
  }

  const defaultYaml = `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: new-pvc
  namespace: ${queryParams.namespace}
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: standard`

  currentPVCYaml.value = defaultYaml
  currentPVCForDetail.value = { name: 'new-pvc', namespace: queryParams.namespace }
  pvcYamlDialogVisible.value = true
}
const handleViewPVC = async (row) => {
  try {
    console.log('🔍 获取PVC详情:', {
      clusterId: selectedClusterId.value,
      namespace: queryParams.namespace,
      pvcName: row?.name,
      row: row
    })

    const response = await k8sApi.getPVCDetail(selectedClusterId.value, queryParams.namespace, row?.name)
    const responseData = response.data || response

    console.log('📥 PVC详情API响应:', responseData)

    if (responseData.code === 200 || responseData.success) {
      currentPVCForDetail.value = responseData.data || row
      console.log('✅ PVC详情数据已设置，打开对话框')
      pvcDetailDialogVisible.value = true
    } else {
      console.error('❌ API返回失败:', responseData)
      ElMessage.error(responseData.message || '获取 PVC 详情失败')
    }
  } catch (error) {
    console.error('❌ 获取 PVC 详情失败:', error)
    ElMessage.error('获取 PVC 详情失败: ' + (error.message || '未知错误'))
  }
}

// 编辑 PVC YAML
const handleEditPVCYaml = async (row) => {
  try {
    loading.value = true
    const response = await k8sApi.getPVCYaml(selectedClusterId.value, queryParams.namespace, row?.name)
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
        yamlContent = `# PVC ${row?.name} YAML\napiVersion: v1\nkind: PersistentVolumeClaim\nmetadata:\n  name: ${row?.name}`
      }

      currentPVCYaml.value = String(yamlContent)
      currentPVCForDetail.value = row
      pvcYamlDialogVisible.value = true
    } else {
      ElMessage.error(responseData.message || tr('获取 PVC YAML 失败'))
    }
  } catch (error) {
    console.error('获取 PVC YAML 失败:', error)
    ElMessage.error(tr('获取 PVC YAML 失败'))
  } finally {
    loading.value = false
  }
}

// 保存 PVC YAML
const handlePVCYamlSave = async (data) => {
  try {
    // 检查是否是创建新PVC (new-pvc表示新建)
    const isCreating = data.resourceName === 'new-pvc' || !currentPVCForDetail.value?.createdAt

    let response
    if (isCreating) {
      // 创建新PVC，使用createPodFromYaml API（支持多种资源类型）
      response = await k8sApi.createPodFromYaml(selectedClusterId.value, queryParams.namespace, { yamlContent: data.yamlContent })
    } else {
      // 更新现有PVC
      response = await k8sApi.updatePVCYaml(selectedClusterId.value, queryParams.namespace, data.resourceName, data.yamlContent)
    }

    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(isCreating ? tr('PVC 创建成功') : 'PVC YAML 更新成功')
      pvcYamlDialogVisible.value = false
      loadAllStorageResources() // 完整刷新列表
    } else {
      ElMessage.error(responseData.message || (isCreating ? 'PVC 创建失败' : 'PVC YAML 更新失败'))
    }
  } catch (error) {
    console.error('PVC 操作失败:', error)
    ElMessage.error(tr('PVC 操作失败'))
  }
}

// 删除 PVC
const handleDeletePVC = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确认删除 PVC "${row?.name}"？删除后数据将无法恢复。`,
      tr('删除确认'),
      {
        confirmButtonText: tr('确认删除'),
        cancelButtonText: tr('取消'),
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )

    const response = await k8sApi.deletePVC(selectedClusterId.value, queryParams.namespace, row?.name)
    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(tr('PVC 删除成功'))
      loadAllStorageResources()
    } else {
      ElMessage.error(responseData.message || 'PVC 删除失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除 PVC 失败:', error)
      ElMessage.error(tr('删除 PVC 失败'))
    }
  }
}

// 获取集群节点列表（Local 卷选择绑定节点用）
const getNodeName = (node) => {
  if (!node) return ''
  if (typeof node === 'string') return node
  return node.name || node.nodeName || node.metadata?.name || node.hostname || ''
}

const fetchNodeList = async () => {
  if (!selectedClusterId.value) return
  try {
    const response = await k8sApi.getClusterNodes(selectedClusterId.value)
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      const rawData = responseData.data?.nodes || responseData.data?.items || responseData.data
      nodeList.value = Array.isArray(rawData) ? rawData : []
      // 过滤出可调度的 worker 节点优先显示
      if (Array.isArray(rawData)) {
        nodeList.value = rawData.filter(n => n && typeof n === 'object')
      }
    }
  } catch (error) {
    console.error('获取节点列表失败:', error)
    nodeList.value = []
  }
}

// 打开界面化创建 PV 对话框
const handleCreatePV = () => {
  if (!selectedClusterId.value) {
    ElMessage.warning(tr('请先选择集群'))
    return
  }

  // 打开界面化创建对话框
  createPVForm.name = ''
  createPVForm.volumeType = 'hostPath'
  createPVForm.capacity = '10Gi'
  createPVForm.accessMode = 'ReadWriteOnce'
  createPVForm.reclaimPolicy = 'Retain'
  createPVForm.storageClass = ''
  createPVForm.volumeMode = 'Filesystem'
  createPVForm.hostPath = '/mnt/data'
  createPVForm.hostPathType = 'DirectoryOrCreate'
  createPVForm.nfsServer = ''
  createPVForm.nfsPath = ''
  createPVForm.nfsReadOnly = false
  createPVForm.localPath = '/mnt/local'
  createPVForm.localFsType = 'ext4'
  createPVForm.localNode = ''
  createPVForm.csiDriver = ''
  createPVForm.csiVolumeHandle = ''
  createPVForm.csiFsType = 'ext4'
  fetchNodeList()
  createPVDialogVisible.value = true
}

// 提交界面化创建 PV
const submitCreatePV = async () => {
  if (!createPVForm.name) {
    ElMessage.warning(tr('请输入 PV 名称'))
    return
  }
  // 阻止用动态供给（CSI）存储类创建静态 PV（如 topolvm-provisioner）
  if (isDynamicStorageClass.value) {
    ElMessage.warning(tr('所选存储类为动态供给（CSI）存储类，请改用「创建 PVC」自动生成 PV，无需手动创建 PV'))
    return
  }
  // Local 卷必须选择节点（用于生成 nodeAffinity）
  if (createPVForm.volumeType === 'local' && !createPVForm.localNode) {
    ElMessage.warning(tr('Local 卷必须选择绑定的节点'))
    return
  }
  // Local 静态卷：若未选择存储类且集群默认存储类是动态供给，提示风险（但不强制，避免无静态存储类时无法创建）
  if (createPVForm.volumeType === 'local' && !createPVForm.storageClass) {
    ElMessage.warning(tr('Local 静态盘建议选择 kubernetes.io/no-provisioner 的存储类；若使用默认存储类（动态供给）可能导致无法正常绑定'))
  }
  try {
    // 构建存储源
    const source = {}
    if (createPVForm.volumeType === 'hostPath') {
      source.hostPath = { path: createPVForm.hostPath, type: createPVForm.hostPathType }
    } else if (createPVForm.volumeType === 'nfs') {
      source.nfs = { server: createPVForm.nfsServer, path: createPVForm.nfsPath, readOnly: createPVForm.nfsReadOnly }
    } else if (createPVForm.volumeType === 'local') {
      source.local = { path: createPVForm.localPath, fsType: createPVForm.localFsType }
    } else if (createPVForm.volumeType === 'csi') {
      source.csi = { driver: createPVForm.csiDriver, volumeHandle: createPVForm.csiVolumeHandle, fsType: createPVForm.csiFsType, volumeAttributes: {} }
    } else if (createPVForm.volumeType === 'iscsi') {
      source.iscsi = { targetPortal: '', iqn: '', lun: 0 }
    }

    const response = await k8sApi.createPV(selectedClusterId.value, {
      name: createPVForm.name,
      capacity: { storage: createPVForm.capacity },
      accessModes: [createPVForm.accessMode],
      reclaimPolicy: createPVForm.reclaimPolicy,
      storageClassName: createPVForm.storageClass || undefined,
      volumeMode: createPVForm.volumeMode || undefined,
      nodeName: createPVForm.volumeType === 'local' ? createPVForm.localNode : undefined,
      persistentVolumeSource: source
    })
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(tr('PV 创建成功'))
      createPVDialogVisible.value = false
      loadAllStorageResources()
    } else {
      ElMessage.error(responseData.message || 'PV 创建失败')
    }
  } catch (error) {
    console.error('创建 PV 失败:', error)
    ElMessage.error('创建 PV 失败: ' + (error.response?.data?.message || error.message))
  }
}

// 通过 YAML 创建 PV
const handleCreatePVYaml = () => {
  if (!selectedClusterId.value) {
    ElMessage.warning(tr('请先选择集群'))
    return
  }

  const defaultYaml = `apiVersion: v1
kind: PersistentVolume
metadata:
  name: new-pv
spec:
  capacity:
    storage: 20Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  storageClassName: standard
  hostPath:
    path: /data/storage`

  currentPVYaml.value = defaultYaml
  currentPVForDetail.value = { name: 'new-pv' }
  pvYamlDialogVisible.value = true
}
const handleViewPV = async (row) => {
  try {
    console.log('🔍 获取PV详情:', {
      clusterId: selectedClusterId.value,
      pvName: row?.name,
      row: row
    })

    const response = await k8sApi.getPVDetail(selectedClusterId.value, row?.name)
    const responseData = response.data || response

    console.log('📥 PV详情API响应:', responseData)

    if (responseData.code === 200 || responseData.success) {
      currentPVForDetail.value = responseData.data || row
      console.log('✅ PV详情数据已设置，打开对话框')
      pvDetailDialogVisible.value = true
    } else {
      console.error('❌ API返回失败:', responseData)
      ElMessage.error(responseData.message || '获取 PV 详情失败')
    }
  } catch (error) {
    console.error('❌ 获取 PV 详情失败:', error)
    ElMessage.error('获取 PV 详情失败: ' + (error.message || '未知错误'))
  }
}

// 编辑 PV YAML
const handleEditPVYaml = async (row) => {
  try {
    loading.value = true
    const response = await k8sApi.getPVYaml(selectedClusterId.value, row?.name)
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
        yamlContent = `# PV ${row?.name} YAML\napiVersion: v1\nkind: PersistentVolume\nmetadata:\n  name: ${row?.name}`
      }

      currentPVYaml.value = String(yamlContent)
      currentPVForDetail.value = row
      pvYamlDialogVisible.value = true
    } else {
      ElMessage.error(responseData.message || tr('获取 PV YAML 失败'))
    }
  } catch (error) {
    console.error('获取 PV YAML 失败:', error)
    ElMessage.error(tr('获取 PV YAML 失败'))
  } finally {
    loading.value = false
  }
}

// 保存 PV YAML
const handlePVYamlSave = async (data) => {
  try {
    // 检查是否是创建新PV (new-pv表示新建)
    const isCreating = data.resourceName === 'new-pv' || !currentPVForDetail.value?.createdAt

    let response
    if (isCreating) {
      // 创建新PV，使用createPodFromYaml API（支持多种资源类型）
      response = await k8sApi.createPodFromYaml(selectedClusterId.value, queryParams.namespace, { yamlContent: data.yamlContent })
    } else {
      // 更新现有PV
      response = await k8sApi.updatePVYaml(selectedClusterId.value, data.resourceName, data.yamlContent)
    }

    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(isCreating ? tr('PV 创建成功') : 'PV YAML 更新成功')
      pvYamlDialogVisible.value = false
      loadAllStorageResources() // 完整刷新列表
    } else {
      ElMessage.error(responseData.message || (isCreating ? 'PV 创建失败' : 'PV YAML 更新失败'))
    }
  } catch (error) {
    console.error('PV 操作失败:', error)
    ElMessage.error(tr('PV 操作失败'))
  }
}

// 删除 PV
const handleDeletePV = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确认删除 PV "${row?.name}"？删除后数据将无法恢复。`,
      tr('删除确认'),
      {
        confirmButtonText: tr('确认删除'),
        cancelButtonText: tr('取消'),
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )

    const response = await k8sApi.deletePV(selectedClusterId.value, row?.name)
    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(tr('PV 删除成功'))
      loadAllStorageResources()
    } else {
      ElMessage.error(responseData.message || 'PV 删除失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除 PV 失败:', error)
      ElMessage.error(tr('删除 PV 失败'))
    }
  }
}

// StorageClass 操作
const handleCreateStorageClass = () => {
  if (!selectedClusterId.value) {
    ElMessage.warning(tr('请先选择集群'))
    return
  }

  // 打开界面化创建对话框
  createSCForm.name = ''
  createSCForm.provisioner = 'kubernetes.io/no-provisioner'
  createSCForm.reclaimPolicy = 'Delete'
  createSCForm.volumeBindingMode = 'Immediate'
  createSCForm.allowVolumeExpansion = false
  createSCForm.parameters = ''
  createSCDialogVisible.value = true
}

// 提交界面化创建 StorageClass
const submitCreateSC = async () => {
  if (!createSCForm.name) {
    ElMessage.warning(tr('请输入 StorageClass 名称'))
    return
  }
  try {
    // 解析参数（key=value 格式，每行一个）
    const parameters = {}
    if (createSCForm.parameters.trim()) {
      createSCForm.parameters.trim().split('\n').forEach(line => {
        const idx = line.indexOf('=')
        if (idx > 0) {
          parameters[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
        }
      })
    }

    const response = await k8sApi.createStorageClass(selectedClusterId.value, {
      name: createSCForm.name,
      provisioner: createSCForm.provisioner,
      reclaimPolicy: createSCForm.reclaimPolicy,
      volumeBindingMode: createSCForm.volumeBindingMode,
      allowVolumeExpansion: createSCForm.allowVolumeExpansion,
      parameters: Object.keys(parameters).length > 0 ? parameters : undefined
    })
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(tr('StorageClass 创建成功'))
      createSCDialogVisible.value = false
      loadAllStorageResources()
    } else {
      ElMessage.error(responseData.message || 'StorageClass 创建失败')
    }
  } catch (error) {
    console.error('创建 StorageClass 失败:', error)
    ElMessage.error('创建 StorageClass 失败: ' + (error.response?.data?.message || error.message))
  }
}

// 通过 YAML 创建 StorageClass
const handleCreateSCYaml = () => {
  if (!selectedClusterId.value) {
    ElMessage.warning(tr('请先选择集群'))
    return
  }

  const defaultYaml = `apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: new-storageclass
provisioner: kubernetes.io/no-provisioner
reclaimPolicy: Delete
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer`

  currentStorageClassYaml.value = defaultYaml
  currentStorageClassForDetail.value = { name: 'new-storageclass' }
  storageClassYamlDialogVisible.value = true
}
const handleViewStorageClass = async (row) => {
  try {
    console.log('🔍 获取StorageClass详情:', {
      clusterId: selectedClusterId.value,
      storageClassName: row?.name,
      row: row
    })

    const response = await k8sApi.getStorageClassDetail(selectedClusterId.value, row?.name)
    const responseData = response.data || response

    console.log('📥 StorageClass详情API响应:', responseData)

    if (responseData.code === 200 || responseData.success) {
      currentStorageClassForDetail.value = responseData.data || row
      console.log('✅ StorageClass详情数据已设置，打开对话框')
      storageClassDetailDialogVisible.value = true
    } else {
      console.error('❌ API返回失败:', responseData)
      ElMessage.error(responseData.message || '获取 StorageClass 详情失败')
    }
  } catch (error) {
    console.error('❌ 获取 StorageClass 详情失败:', error)
    ElMessage.error('获取 StorageClass 详情失败: ' + (error.message || '未知错误'))
  }
}

// 编辑 StorageClass YAML
const handleEditStorageClassYaml = async (row) => {
  try {
    loading.value = true
    const response = await k8sApi.getStorageClassYaml(selectedClusterId.value, row?.name)
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
        yamlContent = `# StorageClass ${row?.name} YAML\napiVersion: storage.k8s.io/v1\nkind: StorageClass\nmetadata:\n  name: ${row?.name}`
      }

      currentStorageClassYaml.value = String(yamlContent)
      currentStorageClassForDetail.value = row
      storageClassYamlDialogVisible.value = true
    } else {
      ElMessage.error(responseData.message || tr('获取 StorageClass YAML 失败'))
    }
  } catch (error) {
    console.error('获取 StorageClass YAML 失败:', error)
    ElMessage.error(tr('获取 StorageClass YAML 失败'))
  } finally {
    loading.value = false
  }
}

// 保存 StorageClass YAML
const handleStorageClassYamlSave = async (data) => {
  try {
    // 检查是否是创建新StorageClass (new-storageclass表示新建)
    const isCreating = data.resourceName === 'new-storageclass' || !currentStorageClassForDetail.value?.createdAt

    let response
    if (isCreating) {
      // 创建新StorageClass，使用createPodFromYaml API（支持多种资源类型）
      response = await k8sApi.createPodFromYaml(selectedClusterId.value, queryParams.namespace, { yamlContent: data.yamlContent })
    } else {
      // 更新现有StorageClass
      response = await k8sApi.updateStorageClassYaml(selectedClusterId.value, data.resourceName, data.yamlContent)
    }

    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(isCreating ? tr('StorageClass 创建成功') : 'StorageClass YAML 更新成功')
      storageClassYamlDialogVisible.value = false
      loadAllStorageResources() // 完整刷新列表
    } else {
      ElMessage.error(responseData.message || (isCreating ? 'StorageClass 创建失败' : 'StorageClass YAML 更新失败'))
    }
  } catch (error) {
    console.error('StorageClass 操作失败:', error)
    ElMessage.error(tr('StorageClass 操作失败'))
  }
}

// 删除 StorageClass
const handleDeleteStorageClass = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确认删除 StorageClass "${row?.name}"？删除后可能影响使用该存储类的PVC。`,
      tr('删除确认'),
      {
        confirmButtonText: tr('确认删除'),
        cancelButtonText: tr('取消'),
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )

    const response = await k8sApi.deleteStorageClass(selectedClusterId.value, row?.name)
    const responseData = response.data || response

    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(tr('StorageClass 删除成功'))
      loadAllStorageResources()
    } else {
      ElMessage.error(responseData.message || 'StorageClass 删除失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除 StorageClass 失败:', error)
      ElMessage.error(tr('删除 StorageClass 失败'))
    }
  }
}

// ===================== PVC 高阶操作 =====================

// 打开扩容对话框
const handleExpandPVC = (row) => {
  expandTargetPVC.value = row
  expandPVCForm.currentSize = row?.capacity || '未知'
  expandPVCForm.newSize = ''
  expandPVCDialogVisible.value = true
}

// 确认 PVC 扩容
const confirmExpandPVC = async () => {
  if (!expandPVCForm.newSize) {
    ElMessage.warning(tr('请输入新的存储容量'))
    return
  }
  try {
    const response = await k8sApi.expandPVC(
      selectedClusterId.value,
      queryParams.namespace,
      expandTargetPVC.value.name,
      expandPVCForm.newSize
    )
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(responseData.data?.message || 'PVC 扩容请求已提交')
      expandPVCDialogVisible.value = false
      loadAllStorageResources()
    } else {
      ElMessage.error(responseData.message || 'PVC 扩容失败')
    }
  } catch (error) {
    console.error('PVC 扩容失败:', error)
    ElMessage.error('PVC 扩容失败: ' + (error.response?.data?.message || error.message))
  }
}

// 打开克隆/备份对话框
const handleClonePVC = (row) => {
  cloneSourcePVC.value = row
  clonePVCForm.targetName = row?.name + '-backup'
  clonePVCForm.targetNamespace = queryParams.namespace
  clonePVCForm.newSize = ''
  clonePVCDialogVisible.value = true
}

// 确认 PVC 克隆
const confirmClonePVC = async () => {
  if (!clonePVCForm.targetName) {
    ElMessage.warning(tr('请输入目标 PVC 名称'))
    return
  }
  try {
    const response = await k8sApi.clonePVC(
      selectedClusterId.value,
      queryParams.namespace,
      cloneSourcePVC.value.name,
      {
        targetName: clonePVCForm.targetName,
        targetNamespace: clonePVCForm.targetNamespace || queryParams.namespace,
        newSize: clonePVCForm.newSize || undefined
      }
    )
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(responseData.data?.message || 'PVC 克隆请求已提交')
      clonePVCDialogVisible.value = false
      loadAllStorageResources()
    } else {
      ElMessage.error(responseData.message || 'PVC 克隆失败')
    }
  } catch (error) {
    console.error('PVC 克隆失败:', error)
    ElMessage.error('PVC 克隆失败: ' + (error.response?.data?.message || error.message))
  }
}

// ===================== StorageClass 高阶操作 =====================

// 设置默认存储类
const handleSetDefaultSC = async (row) => {
  const isCurrentlyDefault = row?.isDefault || false
  const action = isCurrentlyDefault ? '取消默认' : '设为默认'
  try {
    await ElMessageBox.confirm(
      `确定要将存储类 "${row?.name}" ${action}吗？`,
      tr('确认操作'),
      {
        confirmButtonText: tr('确定'),
        cancelButtonText: tr('取消'),
        type: 'info'
      }
    )
    const response = await k8sApi.setDefaultStorageClass(
      selectedClusterId.value,
      row?.name,
      !isCurrentlyDefault
    )
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(responseData.data?.message || `存储类已${action}`)
      loadAllStorageResources()
    } else {
      ElMessage.error(responseData.message || tr('操作失败'))
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('设置默认存储类失败:', error)
      ElMessage.error('设置默认存储类失败: ' + (error.response?.data?.message || error.message))
    }
  }
}

// ===================== 存储迁移操作 =====================

// 打开卷到卷迁移对话框
const openMigrateDialog = (row) => {
  migrateForm.sourcePvc = row?.name || ''
  migrateForm.sourceNs = queryParams.namespace || ''
  migrateForm.targetPvc = row?.name ? row.name + '-migrated' : ''
  migrateForm.targetNs = queryParams.namespace || ''
  migrateForm.targetSize = ''
  migrateForm.targetSc = ''
  migrateForm.createIfAbsent = false
  migrateDialogVisible.value = true
}

// 提交卷到卷迁移
const submitMigrate = async () => {
  if (!migrateForm.sourcePvc || !migrateForm.targetPvc) {
    ElMessage.warning(tr('请填写源 PVC 和目标 PVC 名称'))
    return
  }
  try {
    const response = await k8sApi.migratePVC(selectedClusterId.value, {
      sourcePvc: migrateForm.sourcePvc,
      sourceNs: migrateForm.sourceNs,
      targetPvc: migrateForm.targetPvc,
      targetNs: migrateForm.targetNs || migrateForm.sourceNs,
      targetSize: migrateForm.targetSize || undefined,
      targetSc: migrateForm.targetSc || undefined,
      createIfAbsent: migrateForm.createIfAbsent
    })
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(responseData.data?.message || '卷迁移任务已启动')
      migrateDialogVisible.value = false
      loadAllStorageResources()
    } else {
      ElMessage.error(responseData.message || '卷迁移失败')
    }
  } catch (error) {
    console.error('卷迁移失败:', error)
    ElMessage.error('卷迁移失败: ' + (error.response?.data?.message || error.message))
  }
}

// 打开一键换卷对话框
const openSwitchDialog = (row) => {
  switchForm.workloadType = 'deployment'
  switchForm.workloadName = ''
  switchForm.oldPvcName = row?.name || ''
  switchForm.newPvcName = ''
  switchDialogVisible.value = true
}

// 提交一键换卷
const submitSwitch = async () => {
  if (!switchForm.workloadName || !switchForm.newPvcName) {
    ElMessage.warning(tr('请填写工作负载名称和新 PVC 名称'))
    return
  }
  try {
    const response = await k8sApi.switchWorkloadPVC(
      selectedClusterId.value,
      queryParams.namespace,
      {
        workloadType: switchForm.workloadType,
        workloadName: switchForm.workloadName,
        oldPvcName: switchForm.oldPvcName || undefined,
        newPvcName: switchForm.newPvcName
      }
    )
    const responseData = response.data || response
    if (responseData.code === 200 || responseData.success) {
      ElMessage.success(responseData.data?.message || '一键换卷成功')
      switchDialogVisible.value = false
      loadAllStorageResources()
    } else {
      ElMessage.error(responseData.message || '换卷失败')
    }
  } catch (error) {
    console.error('换卷失败:', error)
    ElMessage.error('换卷失败: ' + (error.response?.data?.message || error.message))
  }
}

// 工具函数
const getClusterStatusText = (status) => {
  const statusMap = {
    1: '(运行中)',
    2: '(正常)',
    3: '(异常)',
    4: '(连接失败)'
  }
  return statusMap[status] || '(未知状态)'
}

const getClusterStatusTag = (status) => {
  const tagMap = {
    1: 'success',
    2: 'success',
    3: 'warning',
    4: 'danger'
  }
  return tagMap[status] || 'info'
}

const getStatusTagType = (status) => {
  const statusMap = {
    'Available': 'success',
    'Bound': 'success',
    'Released': 'warning',
    'Failed': 'danger',
    'Pending': 'info'
  }
  return statusMap[status] || 'info'
}

// 监听集群和命名空间变化，自动加载数据
watch(
  [selectedClusterId, () => queryParams.namespace],
  ([clusterId, namespace]) => {
    console.log('监听到变化 - 集群ID:', clusterId, '命名空间:', namespace)
    if (clusterId && namespace) {
      console.log('集群和命名空间都已选择，开始加载存储资源')
      loadAllStorageResources()
    }
  },
  { immediate: true }
)

// 页面初始化
onMounted(async () => {
  console.log('🚀 开始加载k8s存储管理页面')
  const startTime = Date.now()

  try {
    console.log('🎉 页面初始化完成，总耗时:', Date.now() - startTime + 'ms')
  } catch (error) {
    console.error('页面初始化失败:', error)
  }
})

// KeepAlive 缓存激活时刷新数据
// 解决：从其他页面切回存储管理页时组件被 KeepAlive 缓存，onMounted 不会重新触发导致列表不更新
let activatedRefreshed = false
onActivated(() => {
  if (activatedRefreshed) {
    console.log('🔄 存储管理页面被激活，刷新数据')
    loadAllStorageResources()
  } else {
    // 首次挂载后 onActivated 也会触发，避免与 onMounted/watch 重复加载
    activatedRefreshed = true
  }
})
</script>

<template>
  <div class="k8s-storage-management">
    <el-card shadow="hover" class="storage-card">
      <template #header>
        <div class="card-header">
          <span class="title">{{ $tr("K8s 存储管理") }}</span>
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
          <el-form-item :label='$tr("存储资源名称")'>
            <el-input
              v-model="searchKeyword"
              :placeholder='$tr("请输入名称")'
              clearable
              size="small"
              style="width: 200px"
              @keyup.enter="resetSearch"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :icon="Search" size="small" @click="resetSearch">
              {{ $tr("搜索") }}
            </el-button>
            <el-button :icon="Refresh" size="small" @click="handleRefresh">
              {{ $tr("刷新") }}
            </el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- 存储资源表格 -->
      <el-tabs v-model="activeTab" @tab-change="handleTabChange" class="storage-tabs">
        <!-- PVC 标签页 -->
        <el-tab-pane label="PVC" name="pvc">
          <div class="tab-content">
            <div class="content-header">
              <span class="resource-count">{{ $tr("共") }} {{ pvcTotal }} {{ $tr("个 PVC") }}</span>
              <div>
                <el-button type="primary" :icon="Plus" size="small" @click="handleCreatePVC">
                  {{ $tr("创建 PVC") }}
                </el-button>
                <el-button type="warning" :icon="Document" size="small" @click="handleCreatePVCYaml" style="margin-left: 8px;">
                  {{ $tr("YAML 创建") }}
                </el-button>
              </div>
            </div>

            <el-table
              :data="paginatedPVCList"
              v-loading="loading"
              element-loading-text="加载中..."
              class="resource-table"
              :empty-text='$tr("暂无 PVC 资源")'
            >
              <el-table-column prop="name" :label='$tr("名称")' min-width="150">
                <template #default="{ row }">
                  <div class="resource-name">
                    <el-icon class="resource-icon"><DataLine /></el-icon>
                    <span class="resource-name-link" @click="handleViewPVC(row)">{{ row?.name || '-' }}</span>
                  </div>
                </template>
              </el-table-column>

              <el-table-column prop="status" :label='$tr("状态")' width="120">
                <template #default="{ row }">
                  <el-tag
                    :type="getStatusTagType(row?.status)"
                    size="small"
                  >
                    {{ row?.status || '未知' }}
                  </el-tag>
                </template>
              </el-table-column>

              <el-table-column prop="capacity" :label='$tr("容量")' width="100">
                <template #default="{ row }">
                  <span>{{ row?.capacity || '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column prop="accessModes" :label='$tr("访问模式")' width="140">
                <template #default="{ row }">
                  <div class="access-modes">
                    <el-tag
                      v-for="mode in (row?.accessModes || [])"
                      :key="mode"
                      size="small"
                      type="info"
                      class="mode-tag"
                    >
                      {{ mode }}
                    </el-tag>
                  </div>
                </template>
              </el-table-column>

              <el-table-column prop="storageClass" :label='$tr("存储类")' width="150">
                <template #default="{ row }">
                  <span>{{ row?.storageClass || '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column prop="createdAt" :label='$tr("创建时间")' width="180">
                <template #default="{ row }">
                  <span>{{ row?.createdAt || '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column :label='$tr("操作")' width="320" fixed="right">
                <template #default="{ row }">
                  <div class="operation-buttons">
                    <el-tooltip :content='$tr("扩容")' placement="top">
                      <el-button
                        :icon="Rank"
                        size="small"
                        type="warning"
                        circle
                        @click="handleExpandPVC(row)"
                      />
                    </el-tooltip>
                    <el-tooltip :content='$tr("备份/克隆")' placement="top">
                      <el-button
                        :icon="CopyDocument"
                        size="small"
                        type="info"
                        circle
                        @click="handleClonePVC(row)"
                      />
                    </el-tooltip>
                    <el-tooltip :content='$tr("卷数据迁移")' placement="top">
                      <el-button
                        :icon="Files"
                        size="small"
                        type="primary"
                        circle
                        @click="openMigrateDialog(row)"
                      />
                    </el-tooltip>
                    <el-tooltip :content='$tr("一键换卷")' placement="top">
                      <el-button
                        :icon="Connection"
                        size="small"
                        type="success"
                        circle
                        @click="openSwitchDialog(row)"
                      />
                    </el-tooltip>
                    <el-tooltip :content='$tr("编辑 YAML")' placement="top">
                      <el-button
                        :icon="Edit"
                        size="small"
                        type="primary"
                        circle
                        @click="handleEditPVCYaml(row)"
                      />
                    </el-tooltip>
                    <el-tooltip :content='$tr("删除")' placement="top">
                      <el-button
                        :icon="Delete"
                        size="small"
                        type="danger"
                        circle
                        @click="handleDeletePVC(row)"
                      />
                    </el-tooltip>
                  </div>
                </template>
              </el-table-column>
            </el-table>
            <div class="pagination-container">
              <el-pagination
                v-model:current-page="pvcCurrentPage"
                v-model:page-size="pvcPageSize"
                :page-sizes="[10, 20, 50, 100]"
                :background="true"
                layout="total, sizes, prev, pager, next, jumper"
                :total="pvcTotal"
                @size-change="handlePvcSizeChange"
                @current-change="handlePvcCurrentChange"
              />
            </div>
          </div>
        </el-tab-pane>

        <!-- PV 标签页 -->
        <el-tab-pane label="PV" name="pv">
          <div class="tab-content">
            <div class="content-header">
              <span class="resource-count">{{ $tr("共") }} {{ pvTotal }} {{ $tr("个 PV") }}</span>
              <div>
                <el-button type="primary" :icon="Plus" size="small" @click="handleCreatePV">
                  {{ $tr("创建 PV") }}
                </el-button>
                <el-button type="warning" :icon="Document" size="small" @click="handleCreatePVYaml" style="margin-left: 8px;">
                  {{ $tr("YAML 创建") }}
                </el-button>
              </div>
            </div>

            <el-table
              :data="paginatedPVList"
              v-loading="loading"
              element-loading-text="加载中..."
              class="resource-table"
              :empty-text='$tr("暂无 PV 资源")'
            >
              <el-table-column prop="name" :label='$tr("名称")' min-width="150">
                <template #default="{ row }">
                  <div class="resource-name">
                    <el-icon class="resource-icon"><Cpu /></el-icon>
                    <span class="resource-name-link" @click="handleViewPV(row)">{{ row?.name || '-' }}</span>
                  </div>
                </template>
              </el-table-column>

              <el-table-column prop="status" :label='$tr("状态")' width="120">
                <template #default="{ row }">
                  <el-tag
                    :type="getStatusTagType(row?.status)"
                    size="small"
                  >
                    {{ row?.status || '未知' }}
                  </el-tag>
                </template>
              </el-table-column>

              <el-table-column prop="capacity" :label='$tr("容量")' width="100">
                <template #default="{ row }">
                  <span>{{ row?.capacity || '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column prop="accessModes" :label='$tr("访问模式")' width="140">
                <template #default="{ row }">
                  <div class="access-modes">
                    <el-tag
                      v-for="mode in (row?.accessModes || [])"
                      :key="mode"
                      size="small"
                      type="info"
                      class="mode-tag"
                    >
                      {{ mode }}
                    </el-tag>
                  </div>
                </template>
              </el-table-column>

              <el-table-column prop="reclaimPolicy" :label='$tr("回收策略")' width="120">
                <template #default="{ row }">
                  <span>{{ row?.reclaimPolicy || '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column prop="storageClass" :label='$tr("存储类")' width="150">
                <template #default="{ row }">
                  <span>{{ row?.storageClass || '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column prop="createdAt" :label='$tr("创建时间")' width="180">
                <template #default="{ row }">
                  <span>{{ row?.createdAt || '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column :label='$tr("操作")' width="160" fixed="right">
                <template #default="{ row }">
                  <div class="operation-buttons">
                    <el-tooltip :content='$tr("编辑 YAML")' placement="top">
                      <el-button
                        :icon="Edit"
                        size="small"
                        type="primary"
                        circle
                        @click="handleEditPVYaml(row)"
                      />
                    </el-tooltip>
                    <el-tooltip :content='$tr("删除")' placement="top">
                      <el-button
                        :icon="Delete"
                        size="small"
                        type="danger"
                        circle
                        @click="handleDeletePV(row)"
                      />
                    </el-tooltip>
                  </div>
                </template>
              </el-table-column>
            </el-table>
            <div class="pagination-container">
              <el-pagination
                v-model:current-page="pvCurrentPage"
                v-model:page-size="pvPageSize"
                :page-sizes="[10, 20, 50, 100]"
                :background="true"
                layout="total, sizes, prev, pager, next, jumper"
                :total="pvTotal"
                @size-change="handlePvSizeChange"
                @current-change="handlePvCurrentChange"
              />
            </div>
          </div>
        </el-tab-pane>

        <!-- StorageClass 标签页 -->
        <el-tab-pane label="StorageClass" name="storageclass">
          <div class="tab-content">
            <div class="content-header">
              <span class="resource-count">{{ $tr("共") }} {{ scTotal }} {{ $tr("个 StorageClass") }}</span>
              <div>
                <el-button type="primary" :icon="Plus" size="small" @click="handleCreateStorageClass">
                  {{ $tr("创建 StorageClass") }}
                </el-button>
                <el-button type="warning" :icon="Document" size="small" @click="handleCreateSCYaml" style="margin-left: 8px;">
                  {{ $tr("YAML 创建") }}
                </el-button>
              </div>
            </div>

            <el-table
              :data="paginatedStorageClassList"
              v-loading="loading"
              element-loading-text="加载中..."
              class="resource-table"
              :empty-text='$tr("暂无 StorageClass 资源")'
            >
              <el-table-column prop="name" :label='$tr("名称")' min-width="150">
                <template #default="{ row }">
                  <div class="resource-name">
                    <el-icon class="resource-icon"><Connection /></el-icon>
                    <span class="resource-name-link" @click="handleViewStorageClass(row)">{{ row?.name || '-' }}</span>
                  </div>
                </template>
              </el-table-column>

              <el-table-column prop="provisioner" :label='$tr("供应商")' min-width="200">
                <template #default="{ row }">
                  <span>{{ row?.provisioner || '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column prop="reclaimPolicy" :label='$tr("回收策略")' width="120">
                <template #default="{ row }">
                  <span>{{ row?.reclaimPolicy || '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column prop="allowVolumeExpansion" :label='$tr("支持扩容")' width="100">
                <template #default="{ row }">
                  <el-tag
                    :type="row?.allowVolumeExpansion ? 'success' : 'info'"
                    size="small"
                  >
                    {{ row?.allowVolumeExpansion ? $tr('是') : $tr('否') }}
                  </el-tag>
                </template>
              </el-table-column>

              <el-table-column prop="volumeBindingMode" :label='$tr("绑定模式")' width="150">
                <template #default="{ row }">
                  <span>{{ row?.volumeBindingMode || '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column prop="createdAt" :label='$tr("创建时间")' width="180">
                <template #default="{ row }">
                  <span>{{ row?.createdAt || '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column :label='$tr("操作")' width="240" fixed="right">
                <template #default="{ row }">
                  <div class="operation-buttons">
                    <el-tooltip :content="$tr(row?.isDefault ? '取消默认' : '设为默认')" placement="top">
                      <el-button
                        :icon="Rank"
                        size="small"
                        :type="row?.isDefault ? 'warning' : 'success'"
                        circle
                        @click="handleSetDefaultSC(row)"
                      />
                    </el-tooltip>
                    <el-tooltip :content='$tr("编辑 YAML")' placement="top">
                      <el-button
                        :icon="Edit"
                        size="small"
                        type="primary"
                        circle
                        @click="handleEditStorageClassYaml(row)"
                      />
                    </el-tooltip>
                    <el-tooltip :content='$tr("删除")' placement="top">
                      <el-button
                        :icon="Delete"
                        size="small"
                        type="danger"
                        circle
                        @click="handleDeleteStorageClass(row)"
                      />
                    </el-tooltip>
                  </div>
                </template>
              </el-table-column>
            </el-table>
            <div class="pagination-container">
              <el-pagination
                v-model:current-page="scCurrentPage"
                v-model:page-size="scPageSize"
                :page-sizes="[10, 20, 50, 100]"
                :background="true"
                layout="total, sizes, prev, pager, next, jumper"
                :total="scTotal"
                @size-change="handleScSizeChange"
                @current-change="handleScCurrentChange"
              />
            </div>
          </div>
        </el-tab-pane>

        <!-- Ceph 分布式存储标签页 -->
        <el-tab-pane :label='$tr("Ceph 存储")' name="ceph">
          <el-tabs v-model="cephView">
            <el-tab-pane :label='$tr("概览与资源")' name="overview">
          <div v-loading="cephLoading" class="ceph-pane">
            <!-- 未部署提示 -->
            <el-empty
              v-if="!cephStatus && !cephLoading"
              :description='$tr("未检测到 Ceph 集群（rook-ceph）。请先部署 Rook Ceph 集群并启动 toolbox。")'
            >
              <el-button type="primary" :icon="Refresh" @click="loadCephData">
                {{ $tr("重新检测") }}
              </el-button>
            </el-empty>

            <template v-if="cephStatus">
              <!-- 集群健康状态卡片 -->
              <div class="ceph-status-grid">
                <div class="ceph-stat-card">
                  <div class="stat-label">{{ $tr("集群状态") }}</div>
                  <div class="stat-value">
                    <el-tag
                      :type="cephStatus.health === 'HEALTH_OK' ? 'success' : (cephStatus.health === 'HEALTH_WARN' ? 'warning' : 'danger')"
                      size="large"
                    >
                      {{ cephStatus.health || '-' }}
                    </el-tag>
                  </div>
                </div>
                <div class="ceph-stat-card">
                  <div class="stat-label">{{ $tr("运行阶段") }}</div>
                  <div class="stat-value">
                    <el-tag :type="cephStatus.phase === 'Ready' ? 'success' : 'info'" size="large">
                      {{ cephStatus.phase || '-' }}
                    </el-tag>
                  </div>
                </div>
                <div class="ceph-stat-card">
                  <div class="stat-label">{{ $tr("MON 数量") }}</div>
                  <div class="stat-value">
                    <el-icon class="stat-icon"><Monitor /></el-icon>
                    <span>{{ cephStatus.monCount ?? '-' }}</span>
                  </div>
                </div>
                <div class="ceph-stat-card">
                  <div class="stat-label">{{ $tr("OSD 数量") }}</div>
                  <div class="stat-value">
                    <el-icon class="stat-icon"><Box /></el-icon>
                    <span>{{ cephStatus.osdCount ?? '-' }}</span>
                  </div>
                </div>
                <div class="ceph-stat-card">
                  <div class="stat-label">{{ $tr("PG 数量") }}</div>
                  <div class="stat-value">
                    <el-icon class="stat-icon"><Odometer /></el-icon>
                    <span>{{ cephStatus.pgCount ?? '-' }}</span>
                  </div>
                </div>
                <div class="ceph-stat-card">
                  <div class="stat-label">{{ $tr("版本") }}</div>
                  <div class="stat-value stat-version">{{ cephStatus.version || '-' }}</div>
                </div>
              </div>

              <!-- 集群容量 -->
              <el-card shadow="never" class="ceph-sub-card">
                <template #header>
                  <span class="sub-card-title">{{ $tr("集群容量") }}</span>
                </template>
                <div class="capacity-row">
                  <div class="capacity-item">
                    <span class="cap-label">{{ $tr("总容量") }}</span>
                    <span class="cap-value">{{ formatBytes(cephTotalBytes) }}</span>
                  </div>
                  <div class="capacity-item">
                    <span class="cap-label">{{ $tr("已使用") }}</span>
                    <span class="cap-value" style="color: #e6a23c">{{ formatBytes(cephUsedBytes) }}</span>
                  </div>
                  <div class="capacity-item">
                    <span class="cap-label">{{ $tr("可用") }}</span>
                    <span class="cap-value" style="color: #67c23a">{{ formatBytes(cephTotalBytes - cephUsedBytes) }}</span>
                  </div>
                  <div class="capacity-bar">
                    <el-progress
                      :percentage="cephUsedPercent"
                      :stroke-width="18"
                      :color="cephUsedPercent > 80 ? '#f56c6c' : (cephUsedPercent > 60 ? '#e6a23c' : '#67c23a')"
                      striped
                      striped-flow
                    >
                      <span>{{ $tr("已用") }} {{ cephUsedPercent }}%</span>
                    </el-progress>
                  </div>
                </div>
              </el-card>

              <!-- Ceph 子功能标签页 -->
              <el-tabs v-model="cephSubTab" class="ceph-sub-tabs">
                <!-- 存储池 -->
                <el-tab-pane :label='$tr("存储池")' name="pool">
                  <div class="sub-card-header">
                    <span class="sub-card-title">{{ $tr("存储池 (Pool)") }}</span>
                    <div>
                      <el-button :disabled="!cephCanWrite" size="small" type="primary" :icon="Plus" @click="createPoolDialogVisible = true">
                        {{ $tr("创建 Pool") }}
                      </el-button>
                      <el-button size="small" :icon="Refresh" @click="fetchCephPools">{{ $tr("刷新") }}</el-button>
                    </div>
                  </div>
                  <el-table :data="cephPools" stripe style="width: 100%" size="small" class="ceph-table">
                    <el-table-column :label='$tr("名称")' min-width="180">
                      <template #default="{ row }">
                        <div class="resource-name">
                          <el-icon class="resource-icon"><DataLine /></el-icon>
                          <span>{{ row.name || '-' }}</span>
                        </div>
                      </template>
                    </el-table-column>
                    <el-table-column prop="poolId" label="ID" width="60" align="center">
                      <template #default="{ row }">{{ row.poolId ?? '-' }}</template>
                    </el-table-column>
                    <el-table-column prop="size" :label='$tr("副本")' width="80" align="center">
                      <template #default="{ row }">
                        <el-tag :type="row.size >= 3 ? 'success' : 'warning'" size="small">
                          {{ row.size ?? '-' }}
                        </el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column prop="pgNum" label="PG" width="80" align="center">
                      <template #default="{ row }">{{ row.pgNum ?? '-' }}</template>
                    </el-table-column>
                    <el-table-column prop="application" :label='$tr("应用")' width="110" align="center">
                      <template #default="{ row }">
                        <el-tag size="small" type="info">{{ row.application || '-' }}</el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column prop="usedBytes" :label='$tr("已使用")' min-width="110">
                      <template #default="{ row }">{{ formatBytes(row.usedBytes) }}</template>
                    </el-table-column>
                    <el-table-column prop="rawUsedPercent" :label='$tr("使用率")' min-width="130">
                      <template #default="{ row }">
                        <el-progress
                          :percentage="Number(row.rawUsedPercent || 0)"
                          :stroke-width="14"
                          :color="Number(row.rawUsedPercent) > 80 ? '#f56c6c' : (Number(row.rawUsedPercent) > 60 ? '#e6a23c' : '#67c23a')"
                        />
                      </template>
                    </el-table-column>
                    <el-table-column :label='$tr("操作")' width="280" fixed="right" align="center">
                      <template #default="{ row }">
                        <el-tooltip :content='$tr("调整副本数")' placement="top">
                          <el-button :disabled="!cephCanWrite" :icon="Setting" size="small" type="primary" circle @click="openSizeDialog(row)" />
                        </el-tooltip>
                        <el-tooltip :content='$tr("调整PG数")' placement="top">
                          <el-button :disabled="!cephCanWrite" :icon="Odometer" size="small" type="warning" circle @click="openPGNumDialog(row)" />
                        </el-tooltip>
                        <el-tooltip :content='$tr("设置配额")' placement="top">
                          <el-button :disabled="!cephCanWrite" :icon="Files" size="small" type="info" circle @click="openQuotaDialog(row)" />
                        </el-tooltip>
                        <el-tooltip :content='$tr("设置压缩")' placement="top">
                          <el-button :disabled="!cephCanWrite" :icon="DataLine" size="small" type="success" circle @click="openCompressDialog(row)" />
                        </el-tooltip>
                        <el-tooltip :content='$tr("删除")' placement="top">
                          <el-button :disabled="!cephCanWrite" :icon="Delete" size="small" type="danger" circle @click="handleDeletePool(row)" />
                        </el-tooltip>
                      </template>
                    </el-table-column>
                  </el-table>
                </el-tab-pane>

                <!-- RBD 块存储 -->
                <el-tab-pane :label='$tr("RBD 块存储")' name="rbd">
                  <div class="sub-card-header">
                    <span class="sub-card-title">{{ $tr("RBD 块设备镜像") }}</span>
                    <div>
                      <el-button :disabled="!cephCanWrite" size="small" type="primary" :icon="Plus" @click="rbdCreateDialogVisible = true">
                        {{ $tr("创建镜像") }}
                      </el-button>
                      <el-button size="small" :icon="Refresh" @click="fetchCephRBDImages">{{ $tr("刷新") }}</el-button>
                    </div>
                  </div>
                  <el-table :data="rbdImages" stripe style="width: 100%" size="small" class="ceph-table">
                    <el-table-column :label='$tr("Pool / 镜像")' min-width="260">
                      <template #default="{ row }">
                        <div class="resource-name">
                          <el-icon class="resource-icon"><Box /></el-icon>
                          <span class="resource-name-link" @click="handleViewRBD(row)">{{ row.pool }}/{{ row.name }}</span>
                        </div>
                      </template>
                    </el-table-column>
                    <el-table-column prop="size" :label='$tr("容量")' width="120">
                      <template #default="{ row }">{{ formatBytes(row.size) }}</template>
                    </el-table-column>
                    <el-table-column prop="format" :label='$tr("格式")' width="80" align="center">
                      <template #default="{ row }">v{{ row.format ?? '-' }}</template>
                    </el-table-column>
                    <el-table-column prop="features" :label='$tr("特性")' min-width="220">
                      <template #default="{ row }">
                        <el-tag v-for="f in (row.features || '').split(',').filter(Boolean)" :key="f" size="small" type="info" class="mode-tag">
                          {{ f }}
                        </el-tag>
                        <span v-if="!row.features">-</span>
                      </template>
                    </el-table-column>
                    <el-table-column :label='$tr("操作")' width="180" fixed="right" align="center">
                      <template #default="{ row }">
                        <el-tooltip :content='$tr("详情/快照")' placement="top">
                          <el-button :icon="View" size="small" type="primary" circle @click="handleViewRBD(row)" />
                        </el-tooltip>
                        <el-tooltip :content='$tr("扩容/缩容")' placement="top">
                          <el-button :disabled="!cephCanWrite" :icon="Rank" size="small" type="warning" circle @click="openRBDResize(row)" />
                        </el-tooltip>
                        <el-tooltip :content='$tr("删除")' placement="top">
                          <el-button :disabled="!cephCanWrite" :icon="Delete" size="small" type="danger" circle @click="handleDeleteRBD(row)" />
                        </el-tooltip>
                      </template>
                    </el-table-column>
                  </el-table>
                </el-tab-pane>

                <!-- CephFS 文件系统 -->
                <el-tab-pane :label='$tr("CephFS 文件系统")' name="fs">
                  <div class="sub-card-header">
                    <span class="sub-card-title">{{ $tr("共享文件系统 (CephFS)") }}</span>
                    <el-button size="small" :icon="Refresh" @click="fetchCephFS">{{ $tr("刷新") }}</el-button>
                  </div>
                  <el-table :data="cephFSList" stripe style="width: 100%" size="small" class="ceph-table">
                    <el-table-column :label='$tr("文件系统")' min-width="160">
                      <template #default="{ row }">
                        <div class="resource-name">
                          <el-icon class="resource-icon"><FolderOpened /></el-icon>
                          <span>{{ row.name || '-' }}</span>
                        </div>
                      </template>
                    </el-table-column>
                    <el-table-column prop="metadataPool" :label='$tr("元数据 Pool")' min-width="160">
                      <template #default="{ row }">{{ row.metadataPool || '-' }}</template>
                    </el-table-column>
                    <el-table-column :label='$tr("数据 Pool")' min-width="200">
                      <template #default="{ row }">
                        <el-tag v-for="p in (row.dataPools || [])" :key="p" size="small" type="info" class="mode-tag">
                          {{ p }}
                        </el-tag>
                        <span v-if="!row.dataPools || !row.dataPools.length">-</span>
                      </template>
                    </el-table-column>
                    <el-table-column prop="state" :label='$tr("MDS 状态")' width="140" align="center">
                      <template #default="{ row }">
                        <el-tag :type="row.state === 'active' ? 'success' : 'info'" size="small">
                          {{ row.state || '-' }}
                        </el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column prop="mdsCount" :label='$tr("MDS 数")' width="100" align="center">
                      <template #default="{ row }">{{ row.mdsCount ?? '-' }}</template>
                    </el-table-column>
                  </el-table>
                </el-tab-pane>

                <!-- OSD 管理 -->
                <el-tab-pane :label='$tr("OSD 管理")' name="osd">
                  <div class="sub-card-header">
                    <span class="sub-card-title">{{ $tr("OSD 状态管理") }}</span>
                    <el-button size="small" :icon="Refresh" @click="fetchCephOSDs">{{ $tr("刷新") }}</el-button>
                  </div>
                  <el-table :data="cephOSDs" stripe style="width: 100%" size="small" class="ceph-table">
                    <el-table-column label="OSD" width="100">
                      <template #default="{ row }">
                        <span class="osd-name">osd.{{ row.id }}</span>
                      </template>
                    </el-table-column>
                    <el-table-column prop="host" :label='$tr("所属节点")' min-width="140">
                      <template #default="{ row }">{{ row.host || '-' }}</template>
                    </el-table-column>
                    <el-table-column :label='$tr("状态")' width="150" align="center">
                      <template #default="{ row }">
                        <el-tag :type="row.up ? 'success' : 'danger'" size="small">
                          {{ row.up ? 'up' : 'down' }}
                        </el-tag>
                        <el-tag :type="row.in ? 'success' : 'warning'" size="small" class="osd-state-tag">
                          {{ row.in ? 'in' : 'out' }}
                        </el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column prop="sizeBytes" :label='$tr("容量")' width="120">
                      <template #default="{ row }">{{ formatBytes(row.sizeBytes) }}</template>
                    </el-table-column>
                    <el-table-column prop="usedBytes" :label='$tr("已使用")' width="120">
                      <template #default="{ row }">{{ formatBytes(row.usedBytes) }}</template>
                    </el-table-column>
                    <el-table-column prop="usedPercent" :label='$tr("使用率")' min-width="150">
                      <template #default="{ row }">
                        <el-progress
                          :percentage="Math.round(Number(row.usedPercent || 0) * 10) / 10"
                          :stroke-width="14"
                          :color="Number(row.usedPercent) > 80 ? '#f56c6c' : (Number(row.usedPercent) > 60 ? '#e6a23c' : '#67c23a')"
                        />
                      </template>
                    </el-table-column>
                    <el-table-column prop="weight" :label='$tr("权重")' width="90" align="center">
                      <template #default="{ row }">{{ row.weight ?? '-' }}</template>
                    </el-table-column>
                    <el-table-column :label='$tr("操作")' width="200" fixed="right" align="center">
                      <template #default="{ row }">
                        <el-tooltip :content='$tr("标记 in（加入集群）")' placement="top">
                          <el-button size="small" type="success" circle :disabled="!cephCanWrite || (row.in)" @click="handleOSDAction(row, 'in')">{{ $tr("入") }}</el-button>
                        </el-tooltip>
                        <el-tooltip :content='$tr("标记 out（移出集群）")' placement="top">
                          <el-button size="small" type="warning" circle :disabled="!cephCanWrite || (!row.in)" @click="handleOSDAction(row, 'out')">{{ $tr("出") }}</el-button>
                        </el-tooltip>
                        <el-tooltip :content='$tr("标记 down")' placement="top">
                          <el-button size="small" type="danger" circle :disabled="!cephCanWrite || (!row.up)" @click="handleOSDAction(row, 'down')">{{ $tr("停") }}</el-button>
                        </el-tooltip>
                        <el-tooltip :content='$tr("调整权重")' placement="top">
                          <el-button :disabled="!cephCanWrite" :icon="Setting" size="small" type="primary" circle @click="openOSDReweight(row)" />
                        </el-tooltip>
                      </template>
                    </el-table-column>
                  </el-table>
                </el-tab-pane>

                <!-- 监控 -->
                <el-tab-pane :label='$tr("监控诊断")' name="monitor">
                  <div class="monitor-grid">
                    <el-card shadow="never" class="monitor-card">
                      <template #header><span class="sub-card-title">{{ $tr("集群健康") }}</span></template>
                      <div class="monitor-item">
                        <span class="monitor-label">{{ $tr("状态") }}</span>
                        <el-tag
                          :type="(cephHealth?.status || cephStatus.health) === 'HEALTH_OK' ? 'success' : ((cephHealth?.status || cephStatus.health) === 'HEALTH_WARN' ? 'warning' : 'danger')"
                          size="large"
                        >
                          {{ cephHealth?.status || cephStatus.health || '-' }}
                        </el-tag>
                      </div>
                      <div class="monitor-item">
                        <span class="monitor-label">{{ $tr("检查项") }}</span>
                        <span>{{ cephHealth && Object.keys(cephHealth.checks || {}).length ? Object.keys(cephHealth.checks).join(', ') : '无异常' }}</span>
                      </div>
                    </el-card>
                    <el-card shadow="never" class="monitor-card">
                      <template #header><span class="sub-card-title">{{ $tr("组件版本") }}</span></template>
                      <div v-for="(counts, comp) in (cephVersions || {})" :key="comp" class="monitor-item">
                        <span class="monitor-label">{{ comp }}</span>
                        <span>{{ JSON.stringify(counts) }}</span>
                      </div>
                    </el-card>
                    <el-card shadow="never" class="monitor-card">
                      <template #header><span class="sub-card-title">{{ $tr("PG 状态") }}</span></template>
                      <div class="monitor-item">
                        <span class="monitor-label">{{ $tr("PG 总数") }}</span>
                        <span>{{ cephPGStat?.pg_summary?.num_pgs ?? cephStatus.pgCount ?? '-' }}</span>
                      </div>
                      <div v-for="st in (cephPGStat?.pg_summary?.num_pg_by_state || [])" :key="st.name" class="monitor-item">
                        <span class="monitor-label">{{ st.name }}</span>
                        <el-tag size="small" type="success">{{ st.num }}</el-tag>
                      </div>
                      <div class="monitor-item">
                        <span class="monitor-label">{{ $tr("已用字节") }}</span>
                        <span>{{ formatBytes(cephPGStat?.pg_summary?.total_used_bytes) }}</span>
                      </div>
                    </el-card>
                  </div>
                </el-tab-pane>
              </el-tabs>
            </template>
          </div>
            </el-tab-pane>
            <el-tab-pane :label='$tr("维护中心")' name="maintenance" lazy>
              <CephMaintenance :key="selectedClusterId" :cluster-id="selectedClusterId" :active="activeTab === 'ceph' && cephView === 'maintenance'" @changed="loadCephData" />
            </el-tab-pane>
          </el-tabs>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- Ceph Pool 副本数调整对话框 -->
    <el-dialog
      v-model="sizeDialogVisible"
      :title='$tr("调整 Pool 副本数")'
      width="420px"
      class="detail-dialog"
    >
      <el-form label-width="90px">
        <el-form-item :label='$tr("Pool 名称")'>
          <el-input :model-value="sizeForm.poolName" disabled />
        </el-form-item>
        <el-form-item :label='$tr("应用类型")'>
          <el-input :model-value="sizeForm.poolApp" disabled />
        </el-form-item>
        <el-form-item :label='$tr("副本数")'>
          <el-select v-model="sizeForm.size" style="width: 100%">
            <el-option v-for="n in [1, 2, 3, 4, 5]" :key="n" :label="$tr(n + ' 副本')" :value="n" />
          </el-select>
        </el-form-item>
      </el-form>
      <div class="size-tip">
        <el-icon><Warning /></el-icon>
        <span>
          {{ $tr("调整副本数会触发数据重新均衡（rebalance）。在 3 节点集群中，建议副本数为 3；过低的副本数会降低数据可靠性。") }}
        </span>
      </div>
      <template #footer>
        <el-button @click="sizeDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button :disabled="!cephCanWrite" type="primary" @click="submitPoolSize">{{ $tr("确定") }}</el-button>
      </template>
    </el-dialog>

    <!-- 创建 Pool 对话框 -->
    <el-dialog v-model="createPoolDialogVisible" :title='$tr("创建存储池 (Pool)")' width="460px" class="detail-dialog">
      <el-form label-width="90px">
        <el-form-item :label='$tr("Pool 名称")'>
          <el-input v-model="createPoolForm.name" :placeholder='$tr("如 data-pool")' />
        </el-form-item>
        <el-form-item :label='$tr("PG 数")'>
          <el-input-number v-model="createPoolForm.pgNum" :min="1" :max="1024" style="width: 100%" />
        </el-form-item>
        <el-form-item :label='$tr("副本数")'>
          <el-select v-model="createPoolForm.size" style="width: 100%">
            <el-option v-for="n in [1, 2, 3, 4, 5]" :key="n" :label="$tr(n + ' 副本')" :value="n" />
          </el-select>
        </el-form-item>
        <el-form-item :label='$tr("应用类型")'>
          <el-select v-model="createPoolForm.application" style="width: 100%">
            <el-option :label='$tr("RBD 块存储")' value="rbd" />
            <el-option :label='$tr("CephFS 文件系统")' value="cephfs" />
            <el-option :label='$tr("RGW 对象存储")' value="rgw" />
          </el-select>
        </el-form-item>
      </el-form>
      <div class="size-tip">
        <el-icon><InfoFilled /></el-icon>
        <span>{{ $tr("PG 数建议按 (OSD数量 × 100) / 副本数 估算，如 3 OSD 三副本约 100。创建后可通过\"调整PG数\"动态修改。") }}</span>
      </div>
      <template #footer>
        <el-button @click="createPoolDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button :disabled="!cephCanWrite" type="primary" @click="submitCreatePool">{{ $tr("创建") }}</el-button>
      </template>
    </el-dialog>

    <!-- 调整 PG 数对话框 -->
    <el-dialog v-model="pgNumDialogVisible" :title='$tr("调整 Pool PG 数")' width="420px" class="detail-dialog">
      <el-form label-width="90px">
        <el-form-item :label='$tr("Pool 名称")'>
          <el-input :model-value="pgNumForm.poolName" disabled />
        </el-form-item>
        <el-form-item :label='$tr("PG 数")'>
          <el-input-number v-model="pgNumForm.pgNum" :min="1" :max="4096" style="width: 100%" />
        </el-form-item>
      </el-form>
      <div class="size-tip">
        <el-icon><Warning /></el-icon>
        <span>{{ $tr("增加 PG 数会触发 PG 分裂与数据重均衡；减小 PG 数可能触发合并。建议在维护窗口操作。") }}</span>
      </div>
      <template #footer>
        <el-button @click="pgNumDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button :disabled="!cephCanWrite" type="primary" @click="submitPGNum">{{ $tr("确定") }}</el-button>
      </template>
    </el-dialog>

    <!-- 设置配额对话框 -->
    <el-dialog v-model="quotaDialogVisible" :title='$tr("设置 Pool 配额")' width="460px" class="detail-dialog">
      <el-form label-width="110px">
        <el-form-item :label='$tr("Pool 名称")'>
          <el-input :model-value="quotaForm.poolName" disabled />
        </el-form-item>
        <el-form-item :label='$tr("容量上限 (GiB)")'>
          <el-checkbox v-model="quotaForm.applyBytes">{{ $tr("修改容量上限") }}</el-checkbox>
          <el-input-number :disabled="!quotaForm.applyBytes" v-model="quotaForm.maxBytes" :min="0" :max="10240" style="width: 100%" />
          <div class="form-tip">{{ $tr("未勾选时保持当前配额；勾选并填 0 解除容量限制") }}</div>
        </el-form-item>
        <el-form-item :label='$tr("对象数上限")'>
          <el-checkbox v-model="quotaForm.applyObjects">{{ $tr("修改对象数上限") }}</el-checkbox>
          <el-input-number :disabled="!quotaForm.applyObjects" v-model="quotaForm.maxObjects" :min="0" :max="100000000" style="width: 100%" />
          <div class="form-tip">{{ $tr("未勾选时保持当前配额；勾选并填 0 解除对象数限制") }}</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="quotaDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button :disabled="!cephCanWrite" type="primary" @click="submitQuota">{{ $tr("确定") }}</el-button>
      </template>
    </el-dialog>

    <!-- 设置压缩对话框 -->
    <el-dialog v-model="compressDialogVisible" :title='$tr("设置 Pool 压缩")' width="460px" class="detail-dialog">
      <el-form label-width="90px">
        <el-form-item :label='$tr("Pool 名称")'>
          <el-input :model-value="compressForm.poolName" disabled />
        </el-form-item>
        <el-form-item :label='$tr("压缩算法")'>
          <el-select v-model="compressForm.algorithm" style="width: 100%">
            <el-option :label='$tr("lz4（最快）")' value="lz4" />
            <el-option label="snappy" value="snappy" />
            <el-option :label='$tr("zlib（压缩率高）")' value="zlib" />
            <el-option :label='$tr("zstd（推荐）")' value="zstd" />
          </el-select>
        </el-form-item>
        <el-form-item :label='$tr("压缩模式")'>
          <el-select v-model="compressForm.mode" style="width: 100%">
            <el-option :label='$tr("none（不压缩）")' value="none" />
            <el-option :label='$tr("passive（写入时尝试）")' value="passive" />
            <el-option :label='$tr("aggressive（尽量压缩）")' value="aggressive" />
            <el-option :label='$tr("force（强制压缩）")' value="force" />
          </el-select>
        </el-form-item>
      </el-form>
      <div class="size-tip">
        <el-icon><InfoFilled /></el-icon>
        <span>{{ $tr("压缩仅对新写入的数据生效，可显著节省空间（适合日志/文本类数据），会消耗少量 CPU。") }}</span>
      </div>
      <template #footer>
        <el-button @click="compressDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button :disabled="!cephCanWrite" type="primary" @click="submitCompress">{{ $tr("确定") }}</el-button>
      </template>
    </el-dialog>

    <!-- 创建 RBD 镜像对话框 -->
    <el-dialog v-model="rbdCreateDialogVisible" :title='$tr("创建 RBD 块设备镜像")' width="460px" class="detail-dialog">
      <el-form label-width="90px">
        <el-form-item :label='$tr("所属 Pool")'>
          <el-select v-model="rbdCreateForm.pool" style="width: 100%" filterable>
            <el-option v-for="p in cephPools" :key="p.name" :label="$tr(p.name)" :value="p.name" />
          </el-select>
        </el-form-item>
        <el-form-item :label='$tr("镜像名称")'>
          <el-input v-model="rbdCreateForm.name" :placeholder='$tr("如 my-disk")' />
        </el-form-item>
        <el-form-item :label='$tr("容量")'>
          <el-input v-model="rbdCreateForm.size" :placeholder='$tr("如 10G / 5T / 1024M")' />
        </el-form-item>
      </el-form>
      <div class="size-tip">
        <el-icon><InfoFilled /></el-icon>
        <span>{{ $tr("RBD 为块设备（thin provisioning 按需分配）。创建的镜像可通过 K8s 的 ceph-rbd StorageClass 以 PVC 方式使用。") }}</span>
      </div>
      <template #footer>
        <el-button @click="rbdCreateDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button :disabled="!cephCanWrite" type="primary" @click="submitCreateRBD">{{ $tr("创建") }}</el-button>
      </template>
    </el-dialog>

    <!-- 扩容 RBD 镜像对话框 -->
    <el-dialog v-model="rbdResizeDialogVisible" :title='$tr("调整 RBD 镜像容量")' width="460px" class="detail-dialog">
      <el-form label-width="90px">
        <el-form-item :label='$tr("镜像")'>
          <el-input :model-value="rbdResizeForm.pool + '/' + rbdResizeForm.image" disabled />
        </el-form-item>
        <el-form-item :label='$tr("目标容量")'>
          <el-input v-model="rbdResizeForm.size" :placeholder='$tr("如 20G / 10T")' />
        </el-form-item>
        <el-form-item :label='$tr("允许缩容")'>
          <el-switch v-model="rbdResizeForm.allowShrink" />
        </el-form-item>
      </el-form>
      <div class="size-tip">
        <el-icon><Warning /></el-icon>
        <span>{{ $tr("扩容安全；缩容需勾选\"允许缩容\"且目标容量须大于已使用量，否则数据会损坏。") }}</span>
      </div>
      <template #footer>
        <el-button @click="rbdResizeDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button :disabled="!cephCanWrite" type="primary" @click="submitRBDResize">{{ $tr("确定") }}</el-button>
      </template>
    </el-dialog>

    <!-- RBD 镜像详情 + 快照管理对话框 -->
    <el-dialog v-model="rbdDetailDialogVisible" :title="$tr('RBD 镜像详情: ' + rbdDetail.pool + '/' + rbdDetail.name)" width="720px" class="detail-dialog">
      <el-descriptions :column="3" border size="small" class="ceph-desc">
        <el-descriptions-item :label='$tr("容量")'>{{ formatBytes(rbdDetail.size) }}</el-descriptions-item>
        <el-descriptions-item :label='$tr("实际使用")'>{{ formatBytes(rbdDetail.usedSize) }}</el-descriptions-item>
        <el-descriptions-item :label='$tr("格式")'>v{{ rbdDetail.format }}</el-descriptions-item>
        <el-descriptions-item :label='$tr("对象数")'>{{ rbdDetail.objects }}</el-descriptions-item>
        <el-descriptions-item :label='$tr("对象大小")'>{{ formatBytes(rbdDetail.objectSize) }}</el-descriptions-item>
        <el-descriptions-item :label='$tr("快照数")'>{{ rbdDetail.snapCount }}</el-descriptions-item>
        <el-descriptions-item :label='$tr("特性")' :span="3">{{ rbdDetail.features || '-' }}</el-descriptions-item>
        <el-descriptions-item :label='$tr("父镜像")' :span="3">{{ rbdDetail.parent || '-' }}</el-descriptions-item>
      </el-descriptions>
      <div class="snap-header">
        <span class="sub-card-title">{{ $tr("快照管理") }}</span>
        <el-button :disabled="!cephCanWrite" size="small" type="primary" :icon="Plus" @click="openSnapCreate">{{ $tr("创建快照") }}</el-button>
      </div>
      <el-table :data="rbdSnapshots" stripe size="small" style="width: 100%">
        <el-table-column prop="name" :label='$tr("快照名")' min-width="120">
          <template #default="{ row }">
            <span class="osd-name">@{{ row.name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="size" :label='$tr("大小")' width="110">
          <template #default="{ row }">{{ formatBytes(row.size) }}</template>
        </el-table-column>
        <el-table-column :label='$tr("保护")' width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.protected ? 'success' : 'info'" size="small">
              {{ row.protected ? '已保护' : '未保护' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="timestamp" :label='$tr("创建时间")' min-width="180">
          <template #default="{ row }">{{ row.timestamp || '-' }}</template>
        </el-table-column>
        <el-table-column :label='$tr("操作")' width="220" fixed="right" align="center">
          <template #default="{ row }">
            <el-tooltip :content='$tr("回滚到该快照")' placement="top">
              <el-button :disabled="!cephCanWrite" size="small" type="warning" circle @click="handleRollbackSnap(row)">{{ $tr("回") }}</el-button>
            </el-tooltip>
            <el-tooltip :content='$tr("基于快照克隆（需已保护）")' placement="top">
              <el-button :disabled="!cephCanWrite" size="small" type="success" circle @click="openCloneDialog(row)">{{ $tr("克") }}</el-button>
            </el-tooltip>
            <el-tooltip :content='$tr("删除快照")' placement="top">
              <el-button :disabled="!cephCanWrite" size="small" type="danger" circle @click="handleDeleteSnap(row)">{{ $tr("删") }}</el-button>
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 创建 RBD 快照对话框 -->
    <el-dialog v-model="rbdSnapCreateDialogVisible" :title='$tr("创建快照")' width="420px" class="detail-dialog">
      <el-form label-width="90px">
        <el-form-item :label='$tr("镜像")'>
          <el-input :model-value="rbdSnapCreateForm.pool + '/' + rbdSnapCreateForm.image" disabled />
        </el-form-item>
        <el-form-item :label='$tr("快照名")'>
          <el-input v-model="rbdSnapCreateForm.snapName" :placeholder='$tr("如 snap-20260816")' />
        </el-form-item>
      </el-form>
      <div class="size-tip">
        <el-icon><InfoFilled /></el-icon>
        <span>{{ $tr("快照是镜像某一时刻的只读副本，可用于备份与恢复。克隆镜像前需将快照设为\"已保护\"。") }}</span>
      </div>
      <template #footer>
        <el-button @click="rbdSnapCreateDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button :disabled="!cephCanWrite" type="primary" @click="submitSnapCreate">{{ $tr("创建") }}</el-button>
      </template>
    </el-dialog>

    <!-- 克隆 RBD 镜像对话框 -->
    <el-dialog v-model="rbdCloneDialogVisible" :title='$tr("克隆 RBD 镜像")' width="460px" class="detail-dialog">
      <el-form label-width="90px">
        <el-form-item :label='$tr("源镜像")'>
          <el-input :model-value="rbdCloneForm.pool + '/' + rbdCloneForm.image + '@' + rbdCloneForm.snapName" disabled />
        </el-form-item>
        <el-form-item :label='$tr("目标 Pool")'>
          <el-select v-model="rbdCloneForm.targetPool" style="width: 100%" filterable>
            <el-option v-for="p in cephPools" :key="p.name" :label="$tr(p.name)" :value="p.name" />
          </el-select>
        </el-form-item>
        <el-form-item :label='$tr("目标名称")'>
          <el-input v-model="rbdCloneForm.targetName" :placeholder='$tr("如 clone-20260816")' />
        </el-form-item>
      </el-form>
      <div class="size-tip">
        <el-icon><Warning /></el-icon>
        <span>{{ $tr("克隆为写时复制（COW）子镜像，初始不占额外空间。源快照必须\"已保护\"（需先通过命令行执行 rbd snap protect）。") }}</span>
      </div>
      <template #footer>
        <el-button @click="rbdCloneDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button :disabled="!cephCanWrite" type="primary" @click="submitClone">{{ $tr("克隆") }}</el-button>
      </template>
    </el-dialog>

    <!-- 调整 OSD 权重对话框 -->
    <el-dialog v-model="osdReweightDialogVisible" :title='$tr("调整 OSD 权重")' width="420px" class="detail-dialog">
      <el-form label-width="90px">
        <el-form-item label="OSD">
          <el-input :model-value="'osd.' + osdReweightForm.id" disabled />
        </el-form-item>
        <el-form-item :label='$tr("权重 (0-1)")'>
          <el-input-number v-model="osdReweightForm.weight" :min="0" :max="1" :step="0.1" style="width: 100%" />
        </el-form-item>
      </el-form>
      <div class="size-tip">
        <el-icon><Warning /></el-icon>
        <span>{{ $tr("权重为 0 表示该 OSD 不接收新数据（配合 out 使用）；调低权重可减少其上的数据分布，用于维护或迁移。") }}</span>
      </div>
      <template #footer>
        <el-button @click="osdReweightDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button :disabled="!cephCanWrite" type="primary" @click="submitOSDReweight">{{ $tr("确定") }}</el-button>
      </template>
    </el-dialog>

    <!-- PVC 详情对话框 -->
    <el-dialog
      v-model="pvcDetailDialogVisible"
      :title='$tr("PVC 详情")'
      width="60%"
      class="detail-dialog"
    >
      <div class="detail-content">
        <div class="detail-item">
          <span class="detail-label">{{ $tr("名称:") }}</span>
          <span class="detail-value">{{ currentPVCForDetail.name || '-' }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("命名空间:") }}</span>
          <span class="detail-value">{{ currentPVCForDetail.namespace || '-' }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("状态:") }}</span>
          <el-tag :type="getStatusTagType(currentPVCForDetail.status)" size="small">
            {{ currentPVCForDetail.status || 'Unknown' }}
          </el-tag>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("容量:") }}</span>
          <span class="detail-value">{{ currentPVCForDetail.capacity || '-' }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("访问模式:") }}</span>
          <div class="access-modes">
            <el-tag
              v-for="mode in (currentPVCForDetail.accessModes || [])"
              :key="mode"
              size="small"
              type="info"
              class="mode-tag"
            >
              {{ mode }}
            </el-tag>
          </div>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("存储类:") }}</span>
          <span class="detail-value">{{ currentPVCForDetail.storageClass || currentPVCForDetail.storageClassName || '-' }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("创建时间:") }}</span>
          <span class="detail-value">{{ currentPVCForDetail.createdAt ? new Date(currentPVCForDetail.createdAt).toLocaleString() : (currentPVCForDetail.creationTimestamp ? new Date(currentPVCForDetail.creationTimestamp).toLocaleString() : '-') }}</span>
        </div>
        <div class="detail-item" v-if="currentPVCForDetail.volumeName">
          <span class="detail-label">{{ $tr("绑定的PV:") }}</span>
          <span class="detail-value">{{ currentPVCForDetail.volumeName }}</span>
        </div>
        <div class="detail-item" v-if="currentPVCForDetail.volumeMode">
          <span class="detail-label">{{ $tr("卷模式:") }}</span>
          <span class="detail-value">{{ currentPVCForDetail.volumeMode }}</span>
        </div>
      </div>
    </el-dialog>

    <!-- PV 详情对话框 -->
    <el-dialog
      v-model="pvDetailDialogVisible"
      :title='$tr("PV 详情")'
      width="60%"
      class="detail-dialog"
    >
      <div class="detail-content">
        <div class="detail-item">
          <span class="detail-label">{{ $tr("名称:") }}</span>
          <span class="detail-value">{{ currentPVForDetail.name || '-' }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("状态:") }}</span>
          <el-tag :type="getStatusTagType(currentPVForDetail.status)" size="small">
            {{ currentPVForDetail.status || 'Unknown' }}
          </el-tag>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("容量:") }}</span>
          <span class="detail-value">{{ currentPVForDetail.capacity || '-' }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("访问模式:") }}</span>
          <div class="access-modes">
            <el-tag
              v-for="mode in (currentPVForDetail.accessModes || [])"
              :key="mode"
              size="small"
              type="info"
              class="mode-tag"
            >
              {{ mode }}
            </el-tag>
          </div>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("回收策略:") }}</span>
          <span class="detail-value">{{ currentPVForDetail.reclaimPolicy || '-' }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("存储类:") }}</span>
          <span class="detail-value">{{ currentPVForDetail.storageClass || '-' }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("创建时间:") }}</span>
          <span class="detail-value">{{ currentPVForDetail.createdAt ? new Date(currentPVForDetail.createdAt).toLocaleString() : '-' }}</span>
        </div>
        <div class="detail-item" v-if="currentPVForDetail.volumeMode">
          <span class="detail-label">{{ $tr("卷模式:") }}</span>
          <span class="detail-value">{{ currentPVForDetail.volumeMode }}</span>
        </div>
        <div class="detail-item" v-if="currentPVForDetail.persistentVolumeSource">
          <span class="detail-label">{{ $tr("存储源:") }}</span>
          <div class="storage-source">
            <div v-if="currentPVForDetail.persistentVolumeSource.hostPath">
              <strong>HostPath:</strong> {{ currentPVForDetail.persistentVolumeSource.hostPath.path }}
              <br>
              <small>{{ $tr("类型:") }} {{ currentPVForDetail.persistentVolumeSource.hostPath.type }}</small>
            </div>
            <div v-else-if="currentPVForDetail.persistentVolumeSource.nfs">
              <strong>NFS:</strong> {{ currentPVForDetail.persistentVolumeSource.nfs.server }}:{{ currentPVForDetail.persistentVolumeSource.nfs.path }}
            </div>
            <div v-else>
              {{ Object.keys(currentPVForDetail.persistentVolumeSource)[0] }}
            </div>
          </div>
        </div>
        <div class="detail-item" v-if="currentPVForDetail.claimRef">
          <span class="detail-label">{{ $tr("绑定的PVC:") }}</span>
          <span class="detail-value">{{ currentPVForDetail.claimRef.namespace }}/{{ currentPVForDetail.claimRef.name }}</span>
        </div>
      </div>
    </el-dialog>

    <!-- StorageClass 详情对话框 -->
    <el-dialog
      v-model="storageClassDetailDialogVisible"
      :title='$tr("StorageClass 详情")'
      width="60%"
      class="detail-dialog"
    >
      <div class="detail-content">
        <div class="detail-item">
          <span class="detail-label">{{ $tr("名称:") }}</span>
          <span class="detail-value">{{ currentStorageClassForDetail.name }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("供应商:") }}</span>
          <span class="detail-value">{{ currentStorageClassForDetail.provisioner }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("回收策略:") }}</span>
          <span class="detail-value">{{ currentStorageClassForDetail.reclaimPolicy }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("支持扩容:") }}</span>
          <el-tag
            :type="currentStorageClassForDetail.allowVolumeExpansion ? 'success' : 'info'"
            size="small"
          >
            {{ currentStorageClassForDetail.allowVolumeExpansion ? $tr('是') : $tr('否') }}
          </el-tag>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("绑定模式:") }}</span>
          <span class="detail-value">{{ currentStorageClassForDetail.volumeBindingMode }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ $tr("创建时间:") }}</span>
          <span class="detail-value">{{ currentStorageClassForDetail.createdAt || currentStorageClassForDetail.creationTimestamp || '-' }}</span>
        </div>
      </div>
    </el-dialog>

    <!-- PVC YAML 编辑对话框 -->
    <PodYamlDialog
      :visible="pvcYamlDialogVisible"
      :yaml-content="currentPVCYaml"
      :resource-name="currentPVCForDetail.name || 'new-pvc'"
      :resource-type="'PVC'"
      :editable="true"
      @update:visible="pvcYamlDialogVisible = $event"
      @close="pvcYamlDialogVisible = false"
      @save="handlePVCYamlSave"
    />

    <!-- PV YAML 编辑对话框 -->
    <PodYamlDialog
      :visible="pvYamlDialogVisible"
      :yaml-content="currentPVYaml"
      :resource-name="currentPVForDetail.name || 'new-pv'"
      :resource-type="'PV'"
      :editable="true"
      @update:visible="pvYamlDialogVisible = $event"
      @close="pvYamlDialogVisible = false"
      @save="handlePVYamlSave"
    />

    <!-- StorageClass YAML 编辑对话框 -->
    <PodYamlDialog
      :visible="storageClassYamlDialogVisible"
      :yaml-content="currentStorageClassYaml"
      :resource-name="currentStorageClassForDetail.name || 'new-storageclass'"
      :resource-type="'StorageClass'"
      :editable="true"
      @update:visible="storageClassYamlDialogVisible = $event"
      @close="storageClassYamlDialogVisible = false"
      @save="handleStorageClassYamlSave"
    />

    <!-- PVC 扩容对话框 -->
    <el-dialog
      v-model="expandPVCDialogVisible"
      :title='$tr("PVC 扩容")'
      width="450px"
      :append-to-body="true"
    >
      <el-form label-width="100px">
        <el-form-item :label='$tr("PVC 名称")'>
          <el-input :model-value="expandTargetPVC.name" disabled />
        </el-form-item>
        <el-form-item :label='$tr("当前容量")'>
          <el-input :model-value="expandPVCForm.currentSize" disabled />
        </el-form-item>
        <el-form-item :label='$tr("新容量")' required>
          <el-input
            v-model="expandPVCForm.newSize"
            :placeholder='$tr("例如: 20Gi")'
          >
            <template #append>
              <el-select v-model="sizeUnit" style="width: 70px" disabled>
                <el-option label="Gi" value="Gi" />
              </el-select>
            </template>
          </el-input>
          <div style="font-size: 12px; color: var(--text-secondary); margin-top: 4px;">
            {{ $tr("新容量必须大于当前容量，例如：10Gi → 20Gi") }}
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="expandPVCDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button type="primary" @click="confirmExpandPVC">{{ $tr("确认扩容") }}</el-button>
      </template>
    </el-dialog>

    <!-- PVC 克隆/备份对话框 -->
    <el-dialog
      v-model="clonePVCDialogVisible"
      :title='$tr("PVC 克隆（备份迁移）")'
      width="500px"
      :append-to-body="true"
    >
      <el-form label-width="120px">
        <el-form-item :label='$tr("源 PVC")'>
          <el-input :model-value="cloneSourcePVC.name" disabled />
        </el-form-item>
        <el-form-item :label='$tr("源容量")'>
          <el-input :model-value="cloneSourcePVC.capacity || '-'" disabled />
        </el-form-item>
        <el-form-item :label='$tr("目标 PVC 名称")' required>
          <el-input
            v-model="clonePVCForm.targetName"
            :placeholder='$tr("请输入目标 PVC 名称")'
          />
        </el-form-item>
        <el-form-item :label='$tr("目标命名空间")'>
          <el-input
            v-model="clonePVCForm.targetNamespace"
            :placeholder='$tr("默认与源 PVC 相同")'
          />
        </el-form-item>
        <el-form-item :label='$tr("新容量（可选）")'>
          <el-input
            v-model="clonePVCForm.newSize"
            :placeholder='$tr("留空则与源 PVC 相同")'
          />
        </el-form-item>
      </el-form>
      <div style="padding: 8px 12px; background: var(--bg-card-alt); border-radius: 8px; margin-bottom: 8px; font-size: 13px; color: var(--text-secondary);">
        <el-icon style="vertical-align: middle"><InfoFilled /></el-icon>
        {{ $tr("提示：克隆操作将创建一个具有相同规格的新 PVC 声明。数据迁移需要配合数据复制工具（如 rsync）完成。") }}
      </div>
      <template #footer>
        <el-button @click="clonePVCDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button type="primary" @click="confirmClonePVC">{{ $tr("确认克隆") }}</el-button>
      </template>
    </el-dialog>

    <!-- 界面创建 PVC 对话框 -->
    <el-dialog :title='$tr("创建 PVC（界面化）")' v-model="createPVCDialogVisible" width="520px" :append-to-body="true">
      <el-form label-width="110px">
        <el-form-item :label='$tr("PVC 名称")' required>
          <el-input v-model="createPVCForm.name" :placeholder='$tr("例如: my-app-data")' />
        </el-form-item>
        <el-form-item :label='$tr("存储类")'>
          <el-select v-model="createPVCForm.storageClass" :placeholder='$tr("不选则使用默认存储类")' clearable filterable style="width: 100%">
            <el-option v-for="sc in storageClassList" :key="sc.name" :label="$tr(sc.name)" :value="sc.name" />
          </el-select>
        </el-form-item>
        <el-form-item :label='$tr("存储容量")' required>
          <el-input v-model="createPVCForm.storageSize" :placeholder='$tr("例如: 10Gi")' />
        </el-form-item>
        <el-form-item :label='$tr("访问模式")'>
          <el-select v-model="createPVCForm.accessMode" style="width: 100%">
            <el-option :label='$tr("ReadWriteOnce (单节点读写)")' value="ReadWriteOnce" />
            <el-option :label='$tr("ReadOnlyMany (多节点只读)")' value="ReadOnlyMany" />
            <el-option :label='$tr("ReadWriteMany (多节点读写)")' value="ReadWriteMany" />
          </el-select>
        </el-form-item>
        <el-form-item :label='$tr("卷模式")'>
          <el-select v-model="createPVCForm.volumeMode" style="width: 100%">
            <el-option :label='$tr("Filesystem (文件系统)")' value="Filesystem" />
            <el-option :label='$tr("Block (块设备)")' value="Block" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createPVCDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button type="primary" @click="submitCreatePVC">{{ $tr("创建") }}</el-button>
      </template>
    </el-dialog>

    <!-- 界面创建 PV 对话框 -->
    <el-dialog :title='$tr("创建 PV（界面化）")' v-model="createPVDialogVisible" width="620px" :append-to-body="true">
      <el-alert
        type="warning"
        :closable="false"
        show-icon
        style="margin-bottom: 14px"
        :title='$tr("静态 PV 创建")'
        :description='$tr("此功能用于创建静态 PV（HostPath / NFS / Local / CSI）。TopoLVM 等 CSI 动态供给存储请使用「创建 PVC」自动生成 PV，无需也不应手动创建 PV。")'
      />
      <el-form label-width="110px">
        <el-form-item :label='$tr("PV 名称")' required>
          <el-input v-model="createPVForm.name" :placeholder='$tr("例如: my-local-pv")' />
        </el-form-item>
        <el-form-item :label='$tr("卷类型")' required>
          <el-select v-model="createPVForm.volumeType" style="width: 100%">
            <el-option :label='$tr("HostPath - 本地卷（单机测试）")' value="hostPath" />
            <el-option :label='$tr("NFS - 共享文件存储")' value="nfs" />
            <el-option :label='$tr("Local - 本地静态盘（需 no-provisioner 存储类）")' value="local" />
            <el-option :label='$tr("CSI - 已有卷句柄（Ceph/Rook/TopoLVM）")' value="csi" />
          </el-select>
        </el-form-item>

        <!-- HostPath 参数 -->
        <template v-if="createPVForm.volumeType === 'hostPath'">
          <el-form-item :label='$tr("主机路径")' required>
            <el-input v-model="createPVForm.hostPath" placeholder="/mnt/data" />
          </el-form-item>
          <el-form-item :label='$tr("路径类型")'>
            <el-select v-model="createPVForm.hostPathType" style="width: 100%">
              <el-option label="DirectoryOrCreate" value="DirectoryOrCreate" />
              <el-option label="Directory" value="Directory" />
              <el-option label="FileOrCreate" value="FileOrCreate" />
            </el-select>
          </el-form-item>
        </template>

        <!-- NFS 参数 -->
        <template v-if="createPVForm.volumeType === 'nfs'">
          <el-form-item :label='$tr("NFS 服务器")' required>
            <el-input v-model="createPVForm.nfsServer" placeholder="192.168.80.51" />
          </el-form-item>
          <el-form-item :label='$tr("NFS 路径")' required>
            <el-input v-model="createPVForm.nfsPath" placeholder="/data/nfs" />
          </el-form-item>
          <el-form-item :label='$tr("只读")'>
            <el-switch v-model="createPVForm.nfsReadOnly" />
          </el-form-item>
        </template>

        <!-- Local 参数 -->
        <template v-if="createPVForm.volumeType === 'local'">
          <el-alert
            type="info"
            :closable="false"
            show-icon
            style="margin-bottom: 10px"
            :title='$tr("Local 静态盘说明")'
            :description='$tr("Local 静态 PV 需配合 kubernetes.io/no-provisioner 存储类，且节点上的本地路径必须真实存在。TopoLVM 存储请改用「创建 PVC」动态供给。")'
          />
          <el-form-item :label='$tr("绑定节点")' required>
            <el-select v-model="createPVForm.localNode" :placeholder='$tr("请选择节点")' filterable style="width: 100%">
              <el-option
                v-for="node in nodeList"
                :key="getNodeName(node)"
                :label="$tr(getNodeName(node))"
                :value="getNodeName(node)"
              />
            </el-select>
          </el-form-item>
          <el-form-item :label='$tr("本地路径")' required>
            <el-input v-model="createPVForm.localPath" placeholder="/mnt/disks/vol1" />
          </el-form-item>
          <el-form-item :label='$tr("文件系统")'>
            <el-input v-model="createPVForm.localFsType" placeholder="ext4" />
          </el-form-item>
        </template>

        <!-- CSI 参数 -->
        <template v-if="createPVForm.volumeType === 'csi'">
          <el-form-item :label='$tr("CSI 驱动")' required>
            <el-input v-model="createPVForm.csiDriver" placeholder="rook-ceph.rbd.csi.ceph.com" />
          </el-form-item>
          <el-form-item :label='$tr("卷句柄")' required>
            <el-input v-model="createPVForm.csiVolumeHandle" placeholder="0001-0009-rook-ceph-0000000000000001-..." />
          </el-form-item>
          <el-form-item :label='$tr("文件系统")'>
            <el-input v-model="createPVForm.csiFsType" placeholder="ext4" />
          </el-form-item>
        </template>

        <el-form-item :label='$tr("容量")' required>
          <el-input v-model="createPVForm.capacity" :placeholder='$tr("例如: 10Gi")' />
        </el-form-item>
        <el-form-item :label='$tr("访问模式")'>
          <el-select v-model="createPVForm.accessMode" style="width: 100%">
            <el-option label="ReadWriteOnce" value="ReadWriteOnce" />
            <el-option label="ReadOnlyMany" value="ReadOnlyMany" />
            <el-option label="ReadWriteMany" value="ReadWriteMany" />
          </el-select>
        </el-form-item>
        <el-form-item :label='$tr("回收策略")'>
          <el-select v-model="createPVForm.reclaimPolicy" style="width: 100%">
            <el-option :label='$tr("Retain (保留)")' value="Retain" />
            <el-option :label='$tr("Delete (删除)")' value="Delete" />
            <el-option :label='$tr("Recycle (回收)")' value="Recycle" />
          </el-select>
        </el-form-item>
        <el-form-item :label='$tr("存储类")'>
          <el-select v-model="createPVForm.storageClass" :placeholder='$tr("不选则使用默认存储类")' clearable filterable style="width: 100%">
            <el-option
              v-for="sc in storageClassList"
              :key="sc.name"
              :label="$tr(sc.name + '（' + (sc.provisioner || '') + '）')"
              :value="sc.name"
            />
          </el-select>
          <el-alert
            v-if="isDynamicStorageClass"
            type="error"
            :closable="false"
            show-icon
            style="margin-top: 8px"
            :title='$tr("存储类为动态供给（CSI）")'
            :description='$tr("该存储类由 CSI 控制器自动创建 PV，手动创建静态 PV 不会被正常绑定。TopoLVM 等动态供给存储请关闭本对话框，改用「创建 PVC」自动生成 PV。")'
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createPVDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button type="primary" @click="submitCreatePV">{{ $tr("创建") }}</el-button>
      </template>
    </el-dialog>

    <!-- 界面创建 StorageClass 对话框 -->
    <el-dialog :title='$tr("创建 StorageClass（界面化）")' v-model="createSCDialogVisible" width="560px" :append-to-body="true">
      <el-form label-width="130px">
        <el-form-item :label='$tr("名称")' required>
          <el-input v-model="createSCForm.name" :placeholder='$tr("例如: rook-ceph-block")' />
        </el-form-item>
        <el-form-item label="Provisioner" required>
          <el-select v-model="createSCForm.provisioner" filterable allow-create style="width: 100%">
            <el-option :label='$tr("kubernetes.io/no-provisioner (本地/静态)")' value="kubernetes.io/no-provisioner" />
            <el-option label="rook-ceph.rbd.csi.ceph.com (Ceph RBD)" value="rook-ceph.rbd.csi.ceph.com" />
            <el-option label="rook-ceph.cephfs.csi.ceph.com (CephFS)" value="rook-ceph.cephfs.csi.ceph.com" />
            <el-option label="nfs.csi.k8s.io (NFS CSI)" value="nfs.csi.k8s.io" />
            <el-option label="driver.longhorn.io (Longhorn)" value="driver.longhorn.io" />
            <el-option label="io.openebs.csi-mayastor (OpenEBS Mayastor)" value="io.openebs.csi-mayastor" />
            <el-option label="topolvm.io (TopoLVM)" value="topolvm.io" />
          </el-select>
        </el-form-item>
        <el-form-item :label='$tr("回收策略")'>
          <el-select v-model="createSCForm.reclaimPolicy" style="width: 100%">
            <el-option :label='$tr("Delete (删除)")' value="Delete" />
            <el-option :label='$tr("Retain (保留)")' value="Retain" />
          </el-select>
        </el-form-item>
        <el-form-item :label='$tr("卷绑定模式")'>
          <el-select v-model="createSCForm.volumeBindingMode" style="width: 100%">
            <el-option :label='$tr("Immediate (立即绑定)")' value="Immediate" />
            <el-option :label='$tr("WaitForFirstConsumer (延迟绑定)")' value="WaitForFirstConsumer" />
          </el-select>
        </el-form-item>
        <el-form-item :label='$tr("允许扩容")'>
          <el-switch v-model="createSCForm.allowVolumeExpansion" />
        </el-form-item>
        <el-form-item :label='$tr("参数")'>
          <el-input v-model="createSCForm.parameters" type="textarea" :rows="4" :placeholder='$tr("每行一个 key=value，例如：\nclusterID=rook-ceph\npool=replicapool\nimageFormat=2")' />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createSCDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button type="primary" @click="submitCreateSC">{{ $tr("创建") }}</el-button>
      </template>
    </el-dialog>

    <!-- 卷到卷数据迁移对话框 -->
    <el-dialog :title='$tr("卷数据迁移")' v-model="migrateDialogVisible" width="540px" :append-to-body="true">
      <el-form label-width="120px">
        <el-form-item :label='$tr("源 PVC")' required>
          <el-input v-model="migrateForm.sourcePvc" :placeholder='$tr("源 PVC 名称")' />
        </el-form-item>
        <el-form-item :label='$tr("源命名空间")' required>
          <el-input v-model="migrateForm.sourceNs" :placeholder='$tr("源命名空间")' />
        </el-form-item>
        <el-form-item :label='$tr("目标 PVC")' required>
          <el-input v-model="migrateForm.targetPvc" :placeholder='$tr("目标 PVC 名称")' />
        </el-form-item>
        <el-form-item :label='$tr("目标命名空间")'>
          <el-input v-model="migrateForm.targetNs" :placeholder='$tr("默认与源相同")' />
        </el-form-item>
        <el-form-item :label='$tr("自动创建目标")'>
          <el-switch v-model="migrateForm.createIfAbsent" />
          <span style="margin-left: 8px; font-size: 12px; color: var(--text-secondary);">{{ $tr("目标 PVC 不存在时自动创建") }}</span>
        </el-form-item>
        <el-form-item v-if="migrateForm.createIfAbsent" :label='$tr("目标容量")'>
          <el-input v-model="migrateForm.targetSize" :placeholder='$tr("留空与源相同")' />
        </el-form-item>
        <el-form-item v-if="migrateForm.createIfAbsent" :label='$tr("目标存储类")'>
          <el-select v-model="migrateForm.targetSc" clearable filterable :placeholder='$tr("选择存储类")' style="width: 100%">
            <el-option v-for="sc in storageClassList" :key="sc.name" :label="$tr(sc.name)" :value="sc.name" />
          </el-select>
        </el-form-item>
      </el-form>
      <el-alert type="info" :closable="false" show-icon :title='$tr("迁移说明")'>
        {{ $tr("迁移任务会创建一个 rsync Job，将源卷数据复制到目标卷。完成后请确认数据完整性，再执行一键换卷。") }}
      </el-alert>
      <template #footer>
        <el-button @click="migrateDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button type="primary" @click="submitMigrate">{{ $tr("开始迁移") }}</el-button>
      </template>
    </el-dialog>

    <!-- 一键换卷对话框 -->
    <el-dialog :title='$tr("一键换卷")' v-model="switchDialogVisible" width="540px" :append-to-body="true">
      <el-form label-width="120px">
        <el-form-item :label='$tr("工作负载类型")'>
          <el-select v-model="switchForm.workloadType" style="width: 100%">
            <el-option label="Deployment" value="deployment" />
            <el-option label="StatefulSet" value="statefulset" />
          </el-select>
        </el-form-item>
        <el-form-item :label='$tr("工作负载名称")' required>
          <el-input v-model="switchForm.workloadName" :placeholder='$tr("例如: my-app")' />
        </el-form-item>
        <el-form-item :label='$tr("旧 PVC 名称")'>
          <el-input v-model="switchForm.oldPvcName" :placeholder='$tr("可选，用于精确定位要替换的卷")' />
        </el-form-item>
        <el-form-item :label='$tr("新 PVC 名称")' required>
          <el-input v-model="switchForm.newPvcName" :placeholder='$tr("替换后的新 PVC 名称")' />
        </el-form-item>
      </el-form>
      <el-alert type="warning" :closable="false" show-icon :title='$tr("风险提示")'>
        {{ $tr("换卷后工作负载将滚动重启以挂载新卷，请确认数据已迁移完成且新 PVC 已存在。") }}
      </el-alert>
      <template #footer>
        <el-button @click="switchDialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button type="danger" @click="submitSwitch">{{ $tr("确认换卷") }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.k8s-storage-management {
  padding: 20px;
  min-height: 100vh;
  background: var(--bg-page);
}

.storage-card {
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
  border-color: var(--primary);
  box-shadow: 0 0 0 2px rgba(103, 126, 234, 0.2);
  background: rgba(255, 255, 255, 1);
}

.el-input :deep(.el-input__inner),
.el-select :deep(.el-input__inner) {
  background: transparent;
  border: none;
  color: var(--text-primary);
}

/* 标签样式 */
.el-tag {
  font-weight: 500;
  border-radius: 8px;
  border: none;
}

.storage-tabs {
  margin-top: 20px;
}

.storage-tabs :deep(.el-tabs__header) {
  margin-bottom: 20px;
}

.storage-tabs :deep(.el-tabs__item) {
  font-weight: 500;
  color: var(--text-regular);
}

.storage-tabs :deep(.el-tabs__item.is-active) {
  color: #409EFF;
  font-weight: 600;
}

/* ========== Ceph 存储样式 ========== */
.ceph-pane {
  min-height: 300px;
}

.ceph-status-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.ceph-stat-card {
  background: var(--bg-card-alt, #f5f7fa);
  border: 1px solid rgba(103, 126, 234, 0.15);
  border-radius: 12px;
  padding: 16px;
  text-align: center;
  transition: all 0.3s ease;
}

.ceph-stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(103, 126, 234, 0.15);
}

.stat-label {
  font-size: 12px;
  color: var(--text-secondary, #909399);
  margin-bottom: 8px;
}

.stat-value {
  font-size: 22px;
  font-weight: 700;
  color: var(--text-primary, #303133);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.stat-icon {
  color: #409EFF;
  font-size: 20px;
}

.stat-version {
  font-size: 15px;
  word-break: break-all;
}

.ceph-sub-card {
  margin-bottom: 16px;
  border-radius: 12px;
}

.sub-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.sub-card-title {
  font-weight: 600;
  color: var(--text-primary, #303133);
}

.capacity-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 24px;
}

.capacity-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 100px;
}

.cap-label {
  font-size: 12px;
  color: var(--text-secondary, #909399);
}

.cap-value {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary, #303133);
}

.capacity-bar {
  flex: 1;
  min-width: 260px;
}

.osd-name {
  font-family: 'JetBrains Mono', monospace;
  font-weight: 600;
}

.osd-state-tag {
  margin-left: 4px;
}

.size-tip {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 10px 12px;
  background: var(--bg-card-alt, #f5f7fa);
  border-radius: 8px;
  font-size: 12px;
  color: var(--text-secondary, #909399);
  line-height: 1.6;
  margin-top: 4px;
}

/* Ceph 子标签页 */
.ceph-sub-tabs {
  margin-top: 8px;
}

.ceph-sub-tabs :deep(.el-tabs__item) {
  font-weight: 500;
}

.ceph-sub-tabs :deep(.el-tabs__header) {
  margin-bottom: 14px;
}

.ceph-table {
  margin-top: 8px;
}

.form-tip {
  font-size: 12px;
  color: var(--text-secondary, #909399);
  line-height: 1.6;
}

/* 监控诊断 */
.monitor-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 14px;
}

.monitor-card {
  border-radius: 12px;
}

.monitor-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 4px;
  border-bottom: 1px dashed rgba(103, 126, 234, 0.12);
  font-size: 13px;
}

.monitor-item:last-child {
  border-bottom: none;
}

.monitor-label {
  color: var(--text-secondary, #909399);
  font-size: 12px;
}

/* RBD 详情 */
.ceph-desc {
  margin-bottom: 14px;
}

.snap-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin: 10px 0;
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
  color: var(--text-regular);
  font-weight: 500;
}

.resource-table {
  border-radius: 8px;
  overflow: hidden;
}

.pagination-container {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

.resource-table :deep(.el-table__header) {
  background: var(--bg-card-alt);
}

.resource-table :deep(.el-table__row:hover) {
  background-color: var(--bg-card-alt);
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

.access-modes {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.mode-tag {
  font-size: 11px;
}

.operation-buttons {
  display: flex;
  gap: 8px;
  justify-content: center;
}

/* 对话框样式 - 与k8s-clusters.vue保持一致 */
.detail-dialog :deep(.el-dialog) {
  border-radius: var(--radius-lg);
}

.detail-dialog :deep(.el-dialog__header) {
  background: var(--bg-card-alt);
  border-bottom: 1px solid var(--border);
  border-top-left-radius: var(--radius-lg);
  border-top-right-radius: var(--radius-lg);
  padding: 20px 24px;
}

.detail-dialog :deep(.el-dialog__title) {
  color: var(--text-primary);
  font-weight: 600;
}

.detail-dialog :deep(.el-dialog__body) {
  padding: 24px;
}

.detail-content {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.detail-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid var(--border-light);
}

.detail-label {
  font-weight: 600;
  color: var(--text-regular);
  min-width: 80px;
}

.detail-value {
  color: var(--text-primary);
  word-break: break-all;
}

.storage-source {
  background: var(--bg-card-alt);
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  border-left: 3px solid var(--primary);
  font-size: 13px;
  line-height: 1.6;
}

.storage-source strong {
  color: #409eff;
}

.storage-source small {
  color: var(--text-secondary);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .k8s-storage-management {
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
    grid-template-columns: 1fr;
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
}
</style>