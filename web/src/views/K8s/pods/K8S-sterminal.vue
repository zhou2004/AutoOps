<template>
  <div class="terminal-container">
    <el-card>
      <template #header>
        <div class="terminal-header">
          <div class="terminal-info">
            <el-icon class="terminal-icon">
              <Monitor />
            </el-icon>
            <span>终端 - {{ podName }}</span>
            <el-tag v-if="currentContainer" type="info" size="small">
              {{ currentContainer }}
            </el-tag>
          </div>
          <div class="terminal-controls">
            <el-select
              v-model="selectedContainer"
              placeholder="选择容器"
              size="small"
              style="width: 150px; margin-right: 10px;"
              @change="handleContainerChange"
            >
              <el-option
                v-for="container in containers"
                :key="container"
                :label="container"
                :value="container"
              />
            </el-select>
            <el-button
              type="primary"
              size="small"
              :loading="connecting"
              @click="connect"
              :disabled="!selectedContainer"
            >
              {{ isConnected ? '重连' : '连接' }}
            </el-button>
            <el-button
              type="danger"
              size="small"
              @click="disconnect"
              :disabled="!isConnected"
            >
              断开
            </el-button>
          </div>
        </div>
      </template>
      
      <div class="terminal-content">
        <div ref="terminalElement" class="xterm-container" @click="focusTerminal"></div>
        <div v-if="!isConnected" class="terminal-placeholder">
          <div class="placeholder-content">
            <el-icon class="placeholder-icon">
              <Monitor />
            </el-icon>
            <div class="placeholder-text">
              <h3>Web 终端</h3>
              <p>选择容器并点击连接按钮开始使用终端</p>
              <p v-if="!selectedContainer" class="tip">请先选择一个容器</p>
            </div>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Monitor } from '@element-plus/icons-vue'
import { Terminal as XTerm } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import k8sApi from '@/api/k8s'
import storage from '@/utils/storage'

const route = useRoute()
const router = useRouter()

// 从路由参数获取Pod信息
const clusterId = computed(() => route.params.clusterId)
const namespace = computed(() => route.params.namespace)
const podName = computed(() => route.params.podName)
const containerFromQuery = computed(() => route.query.container)

// 终端状态
const connecting = ref(false)
const isConnected = ref(false)
const selectedContainer = ref('')
const currentContainer = ref('')
const containers = ref([])

// 终端相关
const terminalElement = ref(null)
let terminal = null
let fitAddon = null
let websocket = null
let heartbeatInterval = null

// 初始化终端
const initTerminal = () => {
  if (!terminalElement.value) return
  
  terminal = new XTerm({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
    theme: {
      background: '#060101',
      foreground: '#ffffff',
      cursor: '#ffffff',
      selection: '#ffffff80'
    },
    // 确保终端可以接收输入
    disableStdin: false,
    convertEol: true,
    // 其他输入相关配置
    allowTransparency: false,
    rightClickSelectsWord: true,
    scrollback: 1000,
    cols: 80,
    rows: 24,
    windowsMode: false,
    lineHeight: 1.0,
    letterSpacing: 0,
    tabStopWidth: 8,
    bellStyle: 'none',
    drawBoldTextInBrightColors: true,
    cursorStyle: 'underline'
  })
  
  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  
  terminal.open(terminalElement.value)
  fitAddon.fit()
  
  // 设置终端焦点，确保可以接收输入
  setTimeout(() => {
    terminal.focus()
  }, 100)
  
  // 监听窗口大小变化 - 发送正确的JSON格式resize消息
  const resizeObserver = new ResizeObserver(() => {
    if (fitAddon) {
      fitAddon.fit()
      if (websocket && websocket.readyState === WebSocket.OPEN) {
        const resizeMessage = {
          operation: 'resize',
          data: {
            cols: terminal.cols,
            rows: terminal.rows
          }
        }
        websocket.send(JSON.stringify(resizeMessage))
      }
    }
  })
  resizeObserver.observe(terminalElement.value)
  
  // 监听键盘输入 - 发送JSON格式的消息
  terminal.onData((data) => {
    if (websocket && websocket.readyState === WebSocket.OPEN) {
      const inputMessage = {
        operation: 'stdin',
        data: data
      }
      websocket.send(JSON.stringify(inputMessage))
    }
  })
}

// 获取容器列表
const getContainers = async () => {
  try {
    const response = await k8sApi.getPodContainers(clusterId.value, namespace.value, podName.value)
    const responseData = response.data || response
    
    if (responseData.code === 200 && responseData.data) {
      containers.value = responseData.data.map(container => container.name || container)
    } else {
      console.warn('获取容器列表响应异常:', responseData)
      // 如果API失败，尝试从Pod详情获取
      await getContainersFromPodDetail()
    }
    
    // 如果URL中指定了容器，自动选择
    if (containerFromQuery.value && containers.value.includes(containerFromQuery.value)) {
      selectedContainer.value = containerFromQuery.value
    } else if (containers.value.length > 0) {
      selectedContainer.value = containers.value[0]
    }
  } catch (error) {
    console.error('获取容器列表失败:', error)
    ElMessage.error('获取容器列表失败')
    // 降级处理：从Pod详情获取容器信息
    await getContainersFromPodDetail()
  }
}

// 从工作负载获取容器列表（降级方案）
const getContainersFromPodDetail = async () => {
  try {
    // 从Pod名称推断Deployment名称
    const podNameParts = podName.value.split('-')
    let deploymentName = null
    if (podNameParts.length >= 3) {
      deploymentName = podNameParts.slice(0, -2).join('-')
      console.log('🔍 从Pod名称推断Deployment:', deploymentName)
    }

    if (deploymentName) {
      const response = await k8sApi.getWorkloadPods(
        clusterId.value,
        namespace.value,
        'deployment',
        deploymentName
      )
      const responseData = response.data || response

      if (responseData.code === 200 && responseData.data) {
        // 从Pod列表中找到当前Pod
        const currentPod = responseData.data.find(pod => pod.name === podName.value)
        if (currentPod) {
          if (currentPod.spec && currentPod.spec.containers) {
            containers.value = currentPod.spec.containers.map(container => container.name)
          } else if (currentPod.containers) {
            containers.value = currentPod.containers.map(container => container.name)
          }
          return
        }
      }
    }

    // 如果无法通过工作负载API获取，使用默认容器名
    containers.value = ['main']
  } catch (error) {
    console.error('从工作负载获取容器列表失败:', error)
    // 最后的降级：使用默认容器名
    containers.value = ['main']
  }
}

// 连接终端
const connect = async () => {
  if (!selectedContainer.value) {
    ElMessage.error('请选择容器')
    return
  }

  if (websocket) {
    websocket.close()
  }

  connecting.value = true
  
  try {
    const token = storage.getItem('token')
    if (!token) {
      ElMessage.error('未找到token，请先登录')
      return
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    // 使用当前页面的host，支持Docker部署
    const wsUrl = `${protocol}//${window.location.host}/api/v1/k8s/cluster/${clusterId.value}/namespaces/${namespace.value}/pods/${podName.value}/terminal?containerName=${selectedContainer.value}&token=${encodeURIComponent(token)}`
    
    console.log('🔌 WebSocket连接URL:', wsUrl)
    console.log('🔑 使用token:', token ? '***' + token.slice(-4) : 'null')
    
    websocket = new WebSocket(wsUrl)
    
    // 添加连接超时处理
    const connectionTimeout = setTimeout(() => {
      if (websocket.readyState === WebSocket.CONNECTING) {
        console.warn('⏰ WebSocket连接超时，主动关闭')
        websocket.close()
        connecting.value = false
        ElMessage.error('连接超时，请检查网络或服务器状态')
      }
    }, 10000) // 10秒超时
    
    websocket.onopen = () => {
      clearTimeout(connectionTimeout)
      connecting.value = false
      isConnected.value = true
      currentContainer.value = selectedContainer.value
      ElMessage.success('终端连接成功')
      
      // 清空终端并显示欢迎信息
      if (terminal) {
        terminal.clear()
        terminal.writeln('\x1B[1;32m🖥️  K8S Pod 终端连接成功\x1B[0m')
        terminal.writeln('\x1B[1;34m📦 Pod: ' + podName.value + '\x1B[0m')
        terminal.writeln('\x1B[1;33m🐳 容器: ' + selectedContainer.value + '\x1B[0m')
        terminal.writeln('\x1B[1;36m⚡ 等待服务器终端初始化...\x1B[0m')
        terminal.writeln('')
        
        // 发送初始终端大小 - 使用正确的JSON格式
        const resizeMessage = {
          operation: 'resize',
          data: {
            cols: terminal.cols,
            rows: terminal.rows
          }
        }
        console.log('📐 连接成功，发送初始终端大小:', resizeMessage)
        websocket.send(JSON.stringify(resizeMessage))
        
        // 暂时禁用心跳检测，避免干扰
        // startHeartbeat()
        console.log('🚫 已禁用心跳检测，避免干扰K8S终端')
        
        // 确保终端获得焦点
        setTimeout(() => {
          terminal.focus()
          console.log('🖥️ WebSocket连接成功，终端重新获得焦点')
        }, 200)
      }
    }
    
    websocket.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data)
        console.log('📨 收到消息:', message.operation)
        
        if ((message.operation === 'stdout' || message.operation === 'stderr') && message.data) {
          console.log('✍️ 写入终端')
          terminal.write(message.data)
        }
      } catch (error) {
        // 非JSON消息直接写入
        console.log('📝 非JSON消息，直接写入')
        terminal.write(event.data)
      }
    }
    
    websocket.onerror = (error) => {
      console.error('❌ WebSocket连接错误:', error)
      console.error('🔍 错误详情:', {
        type: error.type,
        target: error.target,
        readyState: error.target?.readyState,
        url: error.target?.url
      })
      ElMessage.error('终端连接出错，请检查网络或后端服务')
      connecting.value = false
      isConnected.value = false
    }
    
    websocket.onclose = (event) => {
      console.log('🔌 WebSocket连接关闭:', {
        code: event.code,
        reason: event.reason,
        wasClean: event.wasClean
      })
      isConnected.value = false
      connecting.value = false
      
      // 停止心跳检测
      stopHeartbeat()
      
      // 提供更友好的错误信息
      const getCloseReason = (code, reason) => {
        switch (code) {
          case 1000:
            return '正常关闭'
          case 1001:
            return '服务器关闭或重启'
          case 1002:
            return '协议错误'
          case 1003:
            return '不支持的数据类型'
          case 1005:
            return '连接异常关闭'
          case 1006:
            return '连接异常断开，可能是网络问题'
          case 1007:
            return '数据格式错误'
          case 1008:
            return '违反协议策略'
          case 1009:
            return '数据过大'
          case 1010:
            return '缺少扩展支持'
          case 1011:
            return '服务器内部错误'
          case 1015:
            return 'TLS握手失败'
          default:
            return reason || '连接中断'
        }
      }
      
      if (event.code === 1000) {
        ElMessage.info('终端连接已正常断开')
      } else if (event.code === 1006) {
        ElMessage.error('终端连接异常断开，请检查网络连接')
      } else {
        const reasonText = getCloseReason(event.code, event.reason)
        ElMessage.warning(`终端连接关闭: ${reasonText}`)
      }
    }
    
  } catch (error) {
    console.error('❌ 创建WebSocket失败:', error)
    console.error('🔍 创建失败详情:', {
      name: error.name,
      message: error.message,
      stack: error.stack
    })
    ElMessage.error(`连接终端失败: ${error.message}`)
    connecting.value = false
    isConnected.value = false
  }
}

// 心跳检测 - 仅监控连接状态，不发送心跳包避免干扰K8S终端
const startHeartbeat = () => {
  stopHeartbeat() // 确保不重复启动
  heartbeatInterval = setInterval(() => {
    if (websocket && websocket.readyState !== WebSocket.OPEN) {
      console.warn('💔 检测到WebSocket连接异常，状态:', websocket.readyState)
      // 连接异常，停止心跳检测
      stopHeartbeat()
    } else if (websocket && websocket.readyState === WebSocket.OPEN) {
      console.log('💓 WebSocket连接正常')
    }
  }, 10000) // 每10秒检查一次连接状态
}

const stopHeartbeat = () => {
  if (heartbeatInterval) {
    clearInterval(heartbeatInterval)
    heartbeatInterval = null
    console.log('💔 心跳检测已停止')
  }
}

// 断开连接
const disconnect = () => {
  stopHeartbeat()
  if (websocket) {
    websocket.close()
  }
  if (terminal) {
    terminal.clear()
  }
}

// 切换容器
const handleContainerChange = () => {
  if (isConnected.value) {
    ElMessage.info('切换容器需要重新连接')
    disconnect()
  }
}

// 手动设置终端焦点
const focusTerminal = () => {
  if (terminal) {
    terminal.focus()
    console.log('🖥️ 手动设置终端焦点')
  }
}

onMounted(() => {
  initTerminal()
  getContainers()
})

onUnmounted(() => {
  disconnect()
  stopHeartbeat()
  if (terminal) {
    terminal.dispose()
  }
})
</script>

<style scoped>
.terminal-container {
  height: 100vh;
  padding: 20px;
  background: #f5f5f5;
}

.terminal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.terminal-info {
  display: flex;
  align-items: center;
  gap: 10px;
}

.terminal-icon {
  color: #409eff;
}

.terminal-controls {
  display: flex;
  align-items: center;
}

.terminal-content {
  height: calc(100vh - 200px);
  position: relative;
}

.xterm-container {
  height: 100%;
  width: 100%;
  background-color: #060101;
  padding: 10px;
  border-radius: 4px;
  overflow: hidden;
}

.terminal-placeholder {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fafafa;
  border: 2px dashed #dcdfe6;
  border-radius: 4px;
}

.placeholder-content {
  text-align: center;
  color: #909399;
}

.placeholder-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.placeholder-text h3 {
  margin: 0 0 8px 0;
  font-size: 18px;
  color: #303133;
}

.placeholder-text p {
  margin: 4px 0;
}

.placeholder-text .tip {
  color: #f56c6c;
  font-weight: bold;
}
</style>