<template>
  <div class="docker-file-manager">
    <div class="file-toolbar">
      <div class="breadcrumb-container" :class="{ 'is-root': currentPath === '/' }">
        <el-icon class="home-icon" @click="navigateTo(0)"><Location /></el-icon>
        <el-breadcrumb separator="/" style="flex: 1; margin-left: 8px">
          <el-breadcrumb-item
            v-for="(part, index) in pathParts"
            :key="index"
            @click="navigateTo(index)"
            class="breadcrumb-item"
          >
            {{ part || "根目录" }}
          </el-breadcrumb-item>
        </el-breadcrumb>
      </div>

      <div class="toolbar-actions">
        <el-button type="primary" plain size="small" :icon="Refresh" @click="fetchFiles" :loading="loading">{{ $tr("刷新") }}</el-button>
        <el-button type="primary" plain size="small" :icon="FolderAdd" @click="showCreateDirDialog">{{ $tr("新建目录") }}</el-button>
        <el-upload class="upload-demo" action="#" :http-request="customUpload" :show-file-list="false">
          <el-button type="success" size="small" :icon="Upload">{{ $tr("上传文件") }}</el-button>
        </el-upload>
      </div>
    </div>

    <el-table
      v-loading="loading"
      :data="fileList"
      height="480px"
      border
      stripe
      highlight-current-row
      @row-dblclick="handleRowDblClick"
    >
      <el-table-column prop="name" :label='$tr("名称")' min-width="240" show-overflow-tooltip>
        <template #default="scope">
          <div style="display: flex; align-items: center; cursor: pointer" class="file-name-cell">
            <el-icon class="file-icon" :color="scope.row.isDir ? '#e6a23c' : '#409EFF'">
              <Folder v-if="scope.row.isDir" />
              <Document v-else />
            </el-icon>
            <span class="file-name-text" :class="{ 'is-directory': scope.row.isDir }">{{ scope.row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="size" :label='$tr("大小")' width="120" align="right">
        <template #default="scope">
          <span class="file-size-text">{{ scope.row.isDir ? "-" : formatSize(scope.row.size) }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="modTime" :label='$tr("修改时间")' width="180" align="center">
        <template #default="scope">
          <span class="file-time-text">{{ scope.row.modTime || "-" }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="mode" :label='$tr("权限")' width="120" align="center">
        <template #default="scope">
          <el-tag size="small" type="info" effect="plain">{{ scope.row.mode || "-" }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label='$tr("操作")' width="160" fixed="right" align="center">
        <template #default="scope">
          <div class="action-buttons">
            <el-tooltip v-if="!scope.row.isDir" :content='$tr("查看/编辑")' placement="top" :show-after="200">
              <el-button type="primary" circle plain size="small" :icon="View" @click="viewAndEditFile(scope.row)" />
            </el-tooltip>
            <el-tooltip v-if="!scope.row.isDir" :content='$tr("下载")' placement="top" :show-after="200">
              <el-button type="success" circle plain size="small" :icon="Download" @click="downloadFile(scope.row)" />
            </el-tooltip>
            <el-popconfirm :title='$tr("确定要删除这个文件/目录吗？")' confirm-button-type="danger" @confirm="deleteFile(scope.row)" width="200px">
              <template #reference>
                <div class="delete-btn-wrapper">
                  <el-tooltip :content='$tr("删除")' placement="top" :show-after="200">
                    <el-button type="danger" circle plain size="small" :icon="Delete" />
                  </el-tooltip>
                </div>
              </template>
            </el-popconfirm>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="editorVisible" :title="`编辑文件: ${currentEditFileName}`" width="800px" top="5vh" append-to-body destroy-on-close>
      <div v-loading="editorLoading" class="editor-container">
        <div class="editor-header">
          <el-icon class="header-icon"><Document /></el-icon>
          <span class="file-path">{{ currentEditFilePath }}</span>
        </div>
        <el-input v-model="fileContent" type="textarea" :rows="22" spellcheck="false" :placeholder='$tr("请输入文件内容...")' />
      </div>
      <template #footer>
        <div class="dialog-footer-custom">
          <el-button @click="editorVisible = false" plain>{{ $tr("取 消") }}</el-button>
          <el-button type="primary" @click="saveFileContent" :loading="savingFile">{{ $tr("保 存") }}</el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog v-model="createDirVisible" :title='$tr("新建目录")' width="400px" append-to-body destroy-on-close>
      <el-input v-model="newDirName" :placeholder='$tr("请输入目录名称")' @keyup.enter="handleCreateDir" />
      <template #footer>
        <span class="dialog-footer-custom">
          <el-button @click="createDirVisible = false" plain>{{ $tr("取 消") }}</el-button>
          <el-button type="primary" @click="handleCreateDir" :loading="creatingDir">{{ $tr("确 认") }}</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { tr } from "@/locales";

import { ref, computed, onMounted } from "vue";
import { ElMessage } from "element-plus";
import { Folder, Document, Refresh, Upload, Download, Delete, Location, View, FolderAdd } from "@element-plus/icons-vue";
import api from "@/api/cmdb";

const props = defineProps({
  assetType: { type: String, required: true },
  assetId: { type: [String, Number], required: true },
  container: { type: String, required: true },
  connectionId: { type: [String, Number], required: true }
});

const currentPath = ref("/");
const fileList = ref([]);
const loading = ref(false);

const editorVisible = ref(false);
const editorLoading = ref(false);
const savingFile = ref(false);
const fileContent = ref("");
const currentEditFileName = ref("");
const currentEditFilePath = ref("");

const createDirVisible = ref(false);
const newDirName = ref("");
const creatingDir = ref(false);

const pathParts = computed(() => {
  const parts = currentPath.value.split("/").filter(p => p);
  return [""].concat(parts);
});

function unpack(res) {
  if (res.data?.code !== 200) throw new Error(res.data?.message || tr("操作失败"));
  return res.data.data;
}

const fetchFiles = async () => {
  loading.value = true;
  try {
    const data = unpack(
      await api.getDockerContainerFiles(props.assetType, props.assetId, props.container, {
        connectionId: props.connectionId,
        path: currentPath.value
      })
    );
    fileList.value = (data.list || []).sort((a, b) => {
      if (a.isDir && !b.isDir) return -1;
      if (!a.isDir && b.isDir) return 1;
      return String(a.name).localeCompare(String(b.name));
    });
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    loading.value = false;
  }
};

const navigateTo = index => {
  currentPath.value = index === 0 ? "/" : "/" + pathParts.value.slice(1, index + 1).join("/");
  fetchFiles();
};

const handleRowDblClick = row => {
  if (row.isDir) {
    currentPath.value = currentPath.value.endsWith("/") ? currentPath.value + row.name : `${currentPath.value}/${row.name}`;
    fetchFiles();
  }
};

const getFullFilePath = fileName =>
  currentPath.value.endsWith("/") ? `${currentPath.value}${fileName}` : `${currentPath.value}/${fileName}`;

const viewAndEditFile = async row => {
  const filePath = getFullFilePath(row.name);
  currentEditFileName.value = row.name;
  currentEditFilePath.value = filePath;
  editorVisible.value = true;
  editorLoading.value = true;
  fileContent.value = "";
  try {
    fileContent.value = unpack(
      await api.getDockerContainerFileContent(props.assetType, props.assetId, props.container, {
        connectionId: props.connectionId,
        path: filePath
      })
    );
  } catch (e) {
    ElMessage.error(e.message);
    editorVisible.value = false;
  } finally {
    editorLoading.value = false;
  }
};

const saveFileContent = async () => {
  savingFile.value = true;
  try {
    unpack(
      await api.updateDockerContainerFile(props.assetType, props.assetId, props.container, {
        connectionId: props.connectionId,
        path: currentEditFilePath.value,
        content: fileContent.value
      })
    );
    ElMessage.success(tr("保存成功"));
    editorVisible.value = false;
    fetchFiles();
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    savingFile.value = false;
  }
};

const showCreateDirDialog = () => {
  newDirName.value = "";
  createDirVisible.value = true;
};

const handleCreateDir = async () => {
  if (!newDirName.value.trim()) {
    ElMessage.warning(tr("请输入目录名称"));
    return;
  }
  creatingDir.value = true;
  try {
    unpack(
      await api.createDockerContainerDir(props.assetType, props.assetId, props.container, {
        connectionId: props.connectionId,
        path: getFullFilePath(newDirName.value.trim())
      })
    );
    ElMessage.success(tr("目录创建成功"));
    createDirVisible.value = false;
    fetchFiles();
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    creatingDir.value = false;
  }
};

const customUpload = async ({ file }) => {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("path", getFullFilePath(file.name));
  try {
    unpack(
      await api.uploadDockerContainerFile(props.assetType, props.assetId, props.container, formData, {
        connectionId: props.connectionId
      })
    );
    ElMessage.success(tr("上传成功"));
    fetchFiles();
  } catch (e) {
    ElMessage.error(e.message);
  }
};

const downloadFile = async row => {
  const filePath = getFullFilePath(row.name);
  try {
    const res = await api.downloadDockerContainerFile(props.assetType, props.assetId, props.container, {
      connectionId: props.connectionId,
      path: filePath
    });
    const blob = new Blob([res.data]);
    const downloadUrl = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = downloadUrl;
    a.download = row.name;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    window.URL.revokeObjectURL(downloadUrl);
  } catch (e) {
    ElMessage.error(tr("下载失败"));
  }
};

const deleteFile = async row => {
  const filePath = getFullFilePath(row.name);
  try {
    unpack(
      await api.deleteDockerContainerFile(props.assetType, props.assetId, props.container, {
        connectionId: props.connectionId,
        path: filePath
      })
    );
    ElMessage.success(tr("删除成功"));
    fetchFiles();
  } catch (e) {
    ElMessage.error(e.message);
  }
};

const formatSize = size => {
  const n = Number(size);
  if (Number.isNaN(n)) return size;
  if (n >= 1024 * 1024 * 1024) return (n / (1024 * 1024 * 1024)).toFixed(2) + " GB";
  if (n >= 1024 * 1024) return (n / (1024 * 1024)).toFixed(2) + " MB";
  if (n >= 1024) return (n / 1024).toFixed(2) + " KB";
  return n + " B";
};

onMounted(fetchFiles);
</script>

<style scoped>
.docker-file-manager {
  padding: 4px;
}
.file-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.breadcrumb-container {
  display: flex;
  align-items: center;
  flex: 1;
}
.home-icon {
  cursor: pointer;
  font-size: 18px;
  color: var(--text-secondary);
}
.breadcrumb-item {
  cursor: pointer;
}
.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.file-name-cell {
  gap: 6px;
}
.file-icon {
  font-size: 16px;
}
.is-directory {
  color: #e6a23c;
  font-weight: 600;
}
.editor-container {
  padding: 8px;
}
.editor-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  color: var(--text-secondary);
}
</style>
