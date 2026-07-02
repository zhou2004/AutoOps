<template>
  <div class="fullscreen-terminal-container">
    <!-- 文件管理左侧边栏 -->
    <div :class="['file-manager', { 'is-expanded': showFileManager }]">
      <div class="fm-header">
        <span>文件管理</span>
        <div class="fm-actions">
           <el-icon @click="fetchFileList(currentPath)" class="action-icon" style="cursor: pointer; margin-right: 10px;" title="刷新"><Refresh /></el-icon>
           <el-icon @click="toggleFileManager" class="close-icon" style="cursor: pointer;"><Close /></el-icon>
        </div>
      </div>
      
      <!-- 路径导航 -->
      <div class="fm-path-bar" v-if="showFileManager">
        <span>{{ currentPath }}</span>
        <el-icon @click="goUpDir" class="action-icon" style="margin-left:auto;cursor:pointer;" title="返回上一级"><Top /></el-icon>
      </div>

      <div class="fm-content" v-loading="fileLoading" @drop.prevent="handleDrop" @dragover.prevent>
        <div class="file-list">
          <div v-for="item in fileList" :key="item.name" class="file-item" @dblclick="handleFileClick(item)">
            <div class="file-info">
              <el-icon class="file-icon" v-if="item.isDir"><Folder color="#e6a23c" /></el-icon>
              <el-icon class="file-icon" v-else><Document color="var(--text-secondary)" /></el-icon>
              <span class="file-name" :title="item.name">{{ item.name }}</span>
            </div>
            
            <div class="file-ops" v-if="!item.isDir">
              <el-icon @click="downloadFile(item)" title="下载"><Download /></el-icon>
              <el-icon @click="deleteFile(item)" class="danger-icon" title="删除"><Delete /></el-icon>
            </div>
            <div class="file-ops" v-else>
              <el-icon @click="deleteFile(item)" class="danger-icon" title="删除"><Delete /></el-icon>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 侧边栏展开按钮：改成了左侧边界鼠标悬停触发 -->
    <div class="fm-toggle-wrapper" v-if="!showFileManager">
      <div class="fm-toggle" @click="toggleFileManager">
        <el-icon><FolderOpened /></el-icon>
      </div>
    </div>

    <!-- 终端主区域 -->
    <div class="terminal-main" :style="{ width: showFileManager ? 'calc(100vw - 300px)' : '100vw' }">
      <div id="terminal" ref="terminalEl" class="terminal-wrapper"></div>
    </div>
  </div>
</template>

<script>
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import 'xterm/css/xterm.css'
import { getToken } from '@/utils/auth'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'

export default {
  name: 'FullscreenTerminal',
  data() {
    return {
      term: null,
      fitAddon: null,
      socket: null,
      hostId: this.$route.query.id,
      hostName: this.$route.query.hostName || 'Terminal',
      _lastInputTime: 0,
      showFileManager: false,
      currentPath: '/',
      fileList: [],
      fileLoading: false
    }
  },
  mounted() {
    document.title = `Terminal - ${this.hostName}`
    document.body.style.backgroundColor = '#000'
    document.body.style.margin = '0'
    
    this.$nextTick(() => {
      this.initTerminal()
      this.connectTerminal()
    })
    window.addEventListener('resize', this.handleResize)
    // 监听滚轮事件实现按住Ctrl字体缩放
    window.addEventListener('wheel', this.handleWheel, { passive: false })
  },
  beforeUnmount() {
    this.disconnectTerminal()
    window.removeEventListener('resize', this.handleResize)
    window.removeEventListener('wheel', this.handleWheel)
    if (this.term) {
      this.term.dispose()
    }
    document.body.style.backgroundColor = ''
    document.body.style.margin = ''
  },
  methods: {
    toggleFileManager() {
      this.showFileManager = !this.showFileManager
      if (this.showFileManager && this.fileList.length === 0) {
        this.fetchFileList(this.currentPath)
      }
      setTimeout(() => {
        this.handleResize() // 展开/收起后重新计算终端大小
      }, 300)
    },
    
    async fetchFileList(path) {
      if (!this.hostId) return
      this.fileLoading = true
      try {
        const res = await request({
          url: '/api/v1/cmdb/hostssh/files',
          method: 'get',
          params: { hostId: this.hostId, path: path }
        })
        if (res.data?.code === 200) {
          this.currentPath = path
          // 文件夹排前面，然后按名字排序
          let files = res.data.data || []
          files.sort((a, b) => {
            if (a.isDir === b.isDir) return a.name.localeCompare(b.name)
            return a.isDir ? -1 : 1
          })
          this.fileList = files
        } else {
          ElMessage.error(res.data?.message || '获取文件列表失败')
        }
      } catch (error) {
        console.error(error)
      } finally {
        this.fileLoading = false
      }
    },

    handleFileClick(item) {
      if (item.isDir) {
        const newPath = this.currentPath.endsWith('/') 
          ? this.currentPath + item.name 
          : this.currentPath + '/' + item.name
        this.fetchFileList(newPath)
      }
    },

    goUpDir() {
      if (this.currentPath === '/' || this.currentPath === '') return
      let parts = this.currentPath.split('/').filter(p => p)
      parts.pop()
      const newPath = '/' + parts.join('/')
      this.fetchFileList(newPath)
    },

    async downloadFile(item) {
      const filePath = this.currentPath.endsWith('/') 
        ? this.currentPath + item.name 
        : this.currentPath + '/' + item.name
        
      try {
        const res = await request({
          url: '/cmdb/hostssh/download',
          method: 'get',
          params: { hostId: this.hostId, path: filePath },
          responseType: 'blob'
        })
        
        // 由于拦截器可能会处理401等情况，如果没有正常返回数据则不处理下载
        if (!res || !res.data) return
        // 创建下载链接
        const blob = new Blob([res.data])
        const downloadUrl = window.URL.createObjectURL(blob)
        const link = document.createElement('a')
        link.style.display = 'none'
        link.href = downloadUrl
        link.setAttribute('download', item.name)
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
        window.URL.revokeObjectURL(downloadUrl)
      } catch (error) {
        console.error('下载失败', error)
        ElMessage.error('下载失败，请检查网络或后端日志')
      }
    },

    deleteFile(item) {
      const filePath = this.currentPath.endsWith('/') 
        ? this.currentPath + item.name 
        : this.currentPath + '/' + item.name
      ElMessageBox.confirm(`确定要删除 ${item.isDir ? '目录' : '文件'} ${item.name} 吗？`, '提示', {
        type: 'warning'
      }).then(async () => {
        try {
          const res = await request({
            url: '/api/v1/cmdb/hostssh/file',
            method: 'delete',
            params: { hostId: this.hostId, path: filePath }
          })
          if (res.data?.code === 200) {
            ElMessage.success('删除成功')
            this.fetchFileList(this.currentPath)
          } else {
            ElMessage.error(res.data?.message || '删除失败')
          }
        } catch (error) {
          console.error(error)
        }
      }).catch(() => {})
    },

    async handleDrop(e) {
      const files = e.dataTransfer.files
      if (files.length > 0) {
        const file = files[0]
        const formData = new FormData()
        formData.append('file', file)
        formData.append('destPath', this.currentPath)
        
        try {
          ElMessage.info(`准备上传: ${file.name}...`)
          const res = await request({
            url: `/api/v1/cmdb/hostssh/upload/${this.hostId}`,
            method: 'post',
            data: formData,
            headers: { 'Content-Type': 'multipart/form-data' },
            timeout: 60000
          })
          if (res.data?.code === 200) {
            ElMessage.success(`上传 ${file.name} 成功`)
            this.fetchFileList(this.currentPath)
          } else {
            ElMessage.error(res.data?.message || '上传失败')
          }
        } catch (err) {
          ElMessage.error('上传异常: ' + err.message)
        }
      }
    },

    handleWheel(e) {
      if (e.ctrlKey && this.term) {
        e.preventDefault()
        let fontSize = this.term.options.fontSize || 14
        if (e.deltaY < 0) {
          fontSize = Math.min(fontSize + 1, 36) // 放大
        } else {
          fontSize = Math.max(fontSize - 1, 10) // 缩小
        }
        this.term.options.fontSize = fontSize
        this.fitAddon.fit()
        // 发送终端大小变更给后端
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
          this.socket.send(JSON.stringify({
            type: 'resize',
            cols: this.term.cols,
            rows: this.term.rows
          }))
        }
      }
    },

    initTerminal() {
      if (!this.$refs.terminalEl) return
      
      this.term = new Terminal({
        cursorBlink: true,
        theme: {
          background: '#000000',
          foreground: '#ffffff',
          cursor: '#ffffff'
        },
        fontFamily: 'Consolas, Liberation Mono, Menlo, Courier, monospace',
        fontSize: 14,
        allowTransparency: true
      })
      
      this.fitAddon = new FitAddon()
      this.term.loadAddon(this.fitAddon)
      this.term.open(this.$refs.terminalEl)
      this.fitAddon.fit()

      // 绑定快捷键 Ctrl+C / Ctrl+V
      this.term.attachCustomKeyEventHandler(e => {
        if (e.ctrlKey && e.type === 'keydown') {
          // Ctrl+C：如果有选中文本则复制，否则就发送SIGINT到底层系统
          if (e.key === 'c' && this.term.hasSelection()) {
            navigator.clipboard.writeText(this.term.getSelection())
            this.term.clearSelection()
            return false // 阻止默认及其它处理
          }
          // Ctrl+V：粘贴文本
          if (e.key === 'v') {
            navigator.clipboard.readText().then(text => {
              if (this.socket && this.socket.readyState === WebSocket.OPEN) {
                this.socket.send(text)
              }
            }).catch(err => {
              console.error('Failed to read clipboard contents: ', err);
            })
            return false
          }
        }
        return true
      })
      
      this.term.writeln(`Connecting to ${this.hostName}...`)
      this.term.writeln(`[Tips] 终端支持 Ctrl+鼠标滚轮 缩放字体；支持 Ctrl+C 复制选中文本；支持 Ctrl+V 粘贴。`)
      this.term.writeln('')
      
      this.term.onData(data => {
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
          if (Date.now() - this._lastInputTime < 30) return
          this._lastInputTime = Date.now()
          this.socket.send(data)
        }
      })
    },
    
    connectTerminal() {
      if (!this.hostId) {
        this.term.writeln('\r\nError: Host ID is missing.')
        return
      }

      const token = getToken()
      const getWsBaseUrl = () => {
        const baseUrl = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '')
        if (baseUrl.startsWith('http')) {
          return baseUrl.replace(/^http/, 'ws')
        }
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        return `${protocol}//${window.location.host}${baseUrl}`
      }

      const wsUrl = `${getWsBaseUrl()}/api/v1/cmdb/hostssh/connect/${this.hostId}?token=${encodeURIComponent(token)}`
      
      this.socket = new WebSocket(wsUrl)
      
      this.socket.onopen = () => {
        // [修复Bug]: 删除了这里的 JSON.stringify({ type: 'auth', token })，避免其打印在终端上
        this.term.writeln('\r\nConnected!')
        this.term.focus()
      }
      
      this.socket.onmessage = (event) => {
        this.term.write(event.data)
      }
      
      this.term.onResize((size) => {
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
          this.socket.send(JSON.stringify({
            type: 'resize',
            cols: size.cols,
            rows: size.rows
          }))
        }
      })
      
      this.socket.onclose = (event) => {
        this.term.writeln(`\r\nConnection closed. (Code: ${event.code})`)
      }
      
      this.socket.onerror = (error) => {
        this.term.writeln('\r\nWebSocket connection failed.')
      }
    },
    
    disconnectTerminal() {
      if (this.socket) {
        this.socket.close()
        this.socket = null
      }
    },
    
    handleResize() {
      if (this.fitAddon) {
        this.fitAddon.fit()
        // 通知后端终端尺寸变化
        if (this.term && this.socket && this.socket.readyState === WebSocket.OPEN) {
          this.socket.send(JSON.stringify({
            type: 'resize',
            cols: this.term.cols,
            rows: this.term.rows
          }))
        }
      }
    }
  }
}
</script>

<style scoped>
.fullscreen-terminal-container {
  display: flex;
  height: 100vh;
  width: 100vw;
  background-color: #000;
  overflow: hidden;
  margin: 0;
  padding: 0;
  position: relative;
}

/* 左侧文件管理器 */
.file-manager {
  width: 0;
  background-color: #1e1e1e;
  color: #fff;
  border-right: 1px solid #333;
  display: flex;
  flex-direction: column;
  transition: width 0.3s ease;
  overflow: hidden;
  white-space: nowrap;
}

.file-manager.is-expanded {
  width: 300px;
}

.fm-header {
  height: 40px;
  min-height: 40px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 15px;
  background-color: #252526;
  border-bottom: 1px solid #333;
  font-size: 14px;
}

.fm-actions {
  display: flex;
  align-items: center;
}

.fm-path-bar {
  background-color: #2d2d2d;
  padding: 8px 15px;
  font-size: 12px;
  color: #ccc;
  display: flex;
  align-items: center;
  border-bottom: 1px solid #333;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.close-icon:hover, .action-icon:hover {
  color: #409EFF;
}

.fm-content {
  flex: 1;
  padding: 0;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}

.file-list {
  display: flex;
  flex-direction: column;
}

.file-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 15px;
  border-bottom: 1px solid #333;
  cursor: pointer;
  transition: background 0.2s;
}

.file-item:hover {
  background-color: #37373d;
}

.file-item:hover .file-ops {
  opacity: 1;
}

.file-info {
  display: flex;
  align-items: center;
  overflow: hidden;
}

.file-icon {
  margin-right: 8px;
  font-size: 16px;
}

.file-name {
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-ops {
  display: flex;
  gap: 8px;
  opacity: 0.6;
  transition: opacity 0.2s;
}

.file-item:hover .file-ops {
  opacity: 1;
}

.file-ops .el-icon {
  font-size: 16px;
}

.file-ops .el-icon:hover {
  color: #409EFF;
}

.danger-icon:hover {
  color: #F56C6C !important;
}

/* 侧边栏热区 */
.fm-toggle-wrapper {
  position: absolute;
  top: 0;
  left: 0;
  width: 40px;
  height: 100%;
  z-index: 9;
}

.fm-toggle-wrapper:hover .fm-toggle {
  opacity: 1;
  visibility: visible;
}

/* 侧边栏开关按钮 — 始终半透明可见 */
.fm-toggle {
  position: absolute;
  top: 15px;
  left: 15px;
  z-index: 10;
  color: #ccc;
  cursor: pointer;
  background: rgba(40, 40, 40, 0.7);
  padding: 8px;
  border-radius: 4px;
  border: 1px solid #555;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s;
  opacity: 0.6;
}

.fm-toggle:hover {
  color: #fff;
  background: rgba(60, 60, 60, 0.9);
  border-color: #888;
  opacity: 1;
}

/* 终端容器 */
.terminal-main {
  transform: translateZ(0); /* 强制GPU加速开启独立渲染层，避免影响重绘 */
  transition: width 0.3s ease;
  height: 100%;
}

.terminal-wrapper {
  height: 100%;
  width: 100%;
  padding: 5px;
  box-sizing: border-box;
}

:deep(.xterm) {
  padding: 5px;
  width: 100% !important;
  height: 100% !important;
}

:deep(.xterm-viewport) {
  width: 100% !important;
  overflow-y: auto;
}
</style>