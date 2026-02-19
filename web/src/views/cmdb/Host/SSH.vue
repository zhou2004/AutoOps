<template>
  <div class="ssh-container" ref="sshContainer">
    <div class="ssh-layout">
      <div class="tree-section">
        <el-card shadow="never" class="group-card">
          <div slot="header" class="card-header">
            <h3>资产分组</h3>
          </div>
          <el-input
            v-model="groupSearchText"
            placeholder="搜索分组"
            clearable
            size="medium"
            class="search-input"
            @input="handleGroupSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>

    <el-tree
      ref="groupTree"
      :data="groupList"
      :props="defaultProps"
      node-key="id"
      :expanded-keys="expandedKeys"
      :highlight-current="true"
      @node-click="handleGroupClick"
      @node-dblclick="handleGroupDblClick"
      @node-expand="handleNodeExpand"
      @node-collapse="handleNodeCollapse"
      class="group-tree"
      @dblclick.native="handleNativeDblClick"
    >
            <template v-slot="{ node, data }">
              <span :class="{ 'font-weight-bold': !data.parentId, 'host-node': data.isHost }">
                <template v-if="data.isHost">
                  <el-icon class="tree-icon"><Platform /></el-icon>
                </template>
                <template v-else-if="data.parentId">
                  <el-icon class="tree-icon"><FolderRemove /></el-icon>
                </template>
                <template v-else>
                  <el-icon v-if="expandedKeys.includes(node.key)" class="tree-icon"><FolderOpened /></el-icon>
                  <el-icon v-else class="tree-icon"><Folder /></el-icon>
                </template>
                {{ node.label }}
              </span>
            </template>
          </el-tree>
        </el-card>
      </div>
      
      <!-- 添加模拟终端卡片 -->
      <div class="terminal-section">
        <el-card shadow="never" class="terminal-card">
          <div slot="header" class="card-header">
            <h3>模拟终端</h3>
          </div>
          <div class="terminal-content">
            <div class="terminal-welcome">
              <h2>欢迎访问 devops 业务终端系统</h2>
            </div>
            <div class="terminal-prompt">
              <p>user@devops:~$ _</p>
            </div>
          </div>
        </el-card>
      </div>
    </div>
    <terminal ref="terminal" :current-host="currentHost" />
  </div>
</template>

<script>
import Terminal from './Terminal.vue'
import storage from '@/utils/storage'

export default {
  components: {
    Terminal
  },
  data() {
    return {
      groupSearchText: '',
      expandedKeys: [],
      groupList: [],
      defaultProps: {
        children: 'children',
        label: 'name'
      },
      currentHost: null,
      showTerminalDrawer: false
    }
  },
  mounted() {
    window.addEventListener('resize', this.handleResize)
    // 在组件挂载后直接设置背景颜色
    this.setContainerBackground()
  },
  beforeUnmount() {
    window.removeEventListener('resize', this.handleResize)
  },
  created() {
    this.loadGroupList()
  },
  methods: {
    
    handleResize() {
      // 窗口大小调整处理
    },
    
    // 设置容器背景颜色的方法
    setContainerBackground() {
      if (this.$refs.sshContainer) {
        this.$refs.sshContainer.style.backgroundColor = '#f5f5f5'
        this.$refs.sshContainer.style.background = '#f5f5f5'
      }
      
      // 使用setTimeout确保DOM完全渲染后再设置
      setTimeout(() => {
        if (this.$refs.sshContainer) {
          this.$refs.sshContainer.style.backgroundColor = '#f5f5f5'
          this.$refs.sshContainer.style.background = '#f5f5f5'
        }
      }, 0)
    },
    
    async loadGroupList() {
      try {
        const response = await this.$api.getGroupListWithHosts()
        if (response.data.code === 200) {
          this.groupList = response.data.data.map(group => ({
            ...group,
            children: group.children ? group.children.map(child => ({
              ...child,
              children: child.hosts ? child.hosts.map(host => ({
                id: host.id,
                name: host.hostName,
                isHost: true,
                hostData: host
              })) : []
            })) : []
          }))
          
          if (this.groupList.length > 0) {
            this.expandedKeys = [this.groupList[0].id]
          }
        }
      } catch (error) {
        console.error('加载分组列表失败:', error)
        this.$message.error('加载分组列表失败')
      }
    },
    
    handleGroupClick(node, element) {
      console.log('点击事件触发测试')
      if (element.data.isHost) {
        console.log('点击的是主机节点')
        this.currentHost = {
          id: element.data.hostData.id,
          hostName: element.data.hostData.hostName,
          sshName: element.data.hostData.sshName,
          sshIp: element.data.hostData.sshIp,
          sshPort: element.data.hostData.sshPort
        }
      }
    },
    
    handleGroupDblClick(node, element) {
      if (element.data.isHost) {
        this.currentHost = {
          id: element.data.hostData.id,
          hostName: element.data.hostData.hostName,
          sshName: element.data.hostData.sshName,
          sshIp: element.data.hostData.sshIp,
          sshPort: element.data.hostData.sshPort
        }
        
        if (this.$refs.terminal) {
          this.$refs.terminal.show()
        }
      }
    },
    
    handleGroupSearch() {
      if (!this.groupSearchText) {
        this.expandedKeys = []
        return
      }
      
      try {
        const findMatchingGroup = (groups, searchText) => {
          for (const group of groups) {
            if (group.name.includes(searchText)) {
              return group
            }
            if (group.children && group.children.length > 0) {
              const found = findMatchingGroup(group.children, searchText)
              if (found) return found
            }
          }
          return null
        }
        
        const matchingGroup = findMatchingGroup(this.groupList, this.groupSearchText)
        if (matchingGroup) {
          const findPath = (groups, targetId, path = []) => {
            for (const group of groups) {
              if (group.id === targetId) {
                return [...path, group.id]
              }
              if (group.children && group.children.length > 0) {
                const foundPath = findPath(group.children, targetId, [...path, group.id])
                if (foundPath) return foundPath
              }
            }
            return null
          }
          
          const expandPath = findPath(this.groupList, matchingGroup.id)
          if (expandPath) {
            this.expandedKeys = expandPath.slice(0, -1)
            this.$nextTick(() => {
              this.$refs.groupTree.setCurrentKey(matchingGroup.id)
            })
          }
        } else {
          this.$message.warning('未找到匹配的分组')
        }
      } catch (error) {
        console.error('搜索分组失败:', error)
        this.$message.error('搜索分组失败')
      }
    },
    
    handleNodeExpand(data, node) {
      if (!this.expandedKeys.includes(node.key)) {
        this.expandedKeys.push(node.key)
      }
    },
    
    handleNodeCollapse(data, node) {
      this.expandedKeys = this.expandedKeys.filter(key => key !== node.key)
    },
    
    handleNativeDblClick(event) {
      const nodeEl = event.target.closest('.el-tree-node')
      if (nodeEl) {
        const nodeId = nodeEl.getAttribute('data-key')
        
        const findHost = (groups, id) => {
          for (const group of groups) {
            for (const child of group.children || []) {
              for (const host of child.children || []) {
                if (host.id == id) {
                  return host.hostData
                }
              }
            }
          }
          return null
        }
        
        const hostData = findHost(this.groupList, nodeId)
        if (hostData) {
          this.currentHost = {
            id: hostData.id,
            hostName: hostData.hostName,
            sshName: hostData.sshName,
            sshIp: hostData.sshIp,
            sshPort: hostData.sshPort
          }
          
          if (this.$refs.terminal) {
            this.$refs.terminal.show()
          }
        }
      }
    }
  }
}
</script>

<style scoped>
.ssh-container {
  height: calc(100vh - 120px);
  padding: 20px;
}

.ssh-layout {
  display: flex;
  height: 100%;
}

.tree-section {
  width: 300px;
}

/* 新增终端区域样式 */
.terminal-section {
  flex: 1;
  margin-left: 20px;
}

.group-card {
  height: 100%;
  border: none;
  background-color: #2a3f54;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.terminal-card {
  height: 100%;
  border: none;
  background-color: #2a3f54;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.card-header {
  background-color: #2a3f54;
  color: #00ff88;
  border-bottom: 1px solid #00ff88;
}

.card-header h3 {
  color: #00ff88;
}

.search-input {
  margin-bottom: 20px;
  width: 100%;
}

.group-tree {
  border: none;
  padding: 5px;
  background-color: #2a3f54;
  color: #00ff88;
}

.font-weight-bold {
  font-weight: bold;
  color: #00ff88;
}

.tree-icon {
  margin-right: 5px;
  color: #00ff88;
}

/* 主机资产节点样式（第三级节点） */
.host-node {
  color: #ffff00 !important; /* 黄色 */
}

.host-node .tree-icon {
  color: #ffff00 !important; /* 黄色图标 */
}

/* 终端内容样式 */
.terminal-content {
  padding: 20px;
  background-color: #2a3f54;
  color: #00ff88;
  font-family: 'Courier New', monospace;
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
}

.terminal-welcome h2 {
  color: #00ff88;
  text-align: center;
  margin-bottom: 30px;
}

.terminal-prompt p {
  color: #00ff88;
  font-size: 16px;
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  border-right: 2px solid #00ff88;
  animation: blink 1s infinite;
}

@keyframes blink {
  0%, 100% { border-color: #00ff88; }
  50% { border-color: transparent; }
}

/* 样式穿透 */
:deep(.el-card__body) {
  background-color: #2a3f54;
}

:deep(.el-tree) {
  background-color: #2a3f54 !important;
}

:deep(.el-tree-node) {
  background-color: #2a3f54 !important;
}

:deep(.el-tree-node__content) {
  background-color: #2a3f54 !important;
  color: #00ff88;
}

:deep(.el-tree-node__content:hover) {
  background-color: #3a4f64 !important;
}

:deep(.el-tree-node:focus > .el-tree-node__content) {
  background-color: #3a4f64 !important;
}

:deep(.el-tree-node.is-current > .el-tree-node__content) {
  background-color: #3a4f64 !important;
}

/* 全面覆盖搜索框样式 */
:deep(.el-input__inner) {
  background-color: #3a4f64 !important;
  border-color: #00ff88 !important;
  color: #00ff88 !important;
}

:deep(.el-input__inner:hover) {
  background-color: #3a4f64 !important;
  border-color: #33ff99 !important;
}

:deep(.el-input__inner:focus) {
  background-color: #3a4f64 !important;
  border-color: #33ff99 !important;
  color: #00ff88 !important;
}

:deep(.el-input__prefix) {
  color: #00ff88 !important;
}

:deep(.el-input__suffix) {
  color: #00ff88 !important;
}

/* 修复搜索框焦点状态下的白色背景 */
:deep(.el-input__suffix-inner) {
  background-color: transparent !important;
}

/* 修复清空图标颜色 */
:deep(.el-input__suffix .el-icon-circle-close) {
  color: #00ff88 !important;
}

/* 修复输入框获得焦点时的边框阴影 */
:deep(.el-input.is-focus .el-input__inner) {
  border-color: #33ff99 !important;
  box-shadow: 0 0 0 2px rgba(0, 255, 136, 0.2) !important;
  background-color: #3a4f64 !important;
}

/* 修复输入框激活状态 */
:deep(.el-input--medium .el-input__inner) {
  background-color: #3a4f64 !important;
  border-color: #00ff88 !important;
  color: #00ff88 !important;
}

/* 修复输入框悬停状态 */
:deep(.el-input .el-input__inner:hover) {
  background-color: #3a4f64 !important;
  border-color: #33ff99 !important;
}

/* 修复搜索图标颜色 */
:deep(.el-input__icon) {
  color: #00ff88 !important;
}

/* 修复输入框容器背景 */
:deep(.el-input) {
  background-color: transparent !important;
}

/* 修复搜索图标容器背景 */
:deep(.el-input__prefix-inner) {
  background-color: transparent !important;
}

/* 修复suffix内部背景 */
:deep(.el-input__suffix-inner svg) {
  background-color: transparent !important;
}
</style>

<style>
/* 使用更强制的方式设置整个页面背景 */
.ssh-container,
div[class*="ssh-container"],
div[data-v-ssh-container] {
  background-color: #f5f5f5 !important;
  background: #f5f5f5 !important;
}

/* 针对可能的父级容器进行样式覆盖 */
.el-main .ssh-container,
.main-container .ssh-container,
.page-container .ssh-container {
  background-color: #f5f5f5 !important;
  background: #f5f5f5 !important;
}

/* 使用ID选择器和属性选择器增加优先级 */
#ssh-container,
[data-component="ssh-container"] {
  background-color: #f5f5f5 !important;
  background: #f5f5f5 !important;
}
</style>
