<template>
  <div class="docker-terminal-page">
    <el-card shadow="never" class="terminal-card">
      <template #header>
        <div class="terminal-header">
          <div class="terminal-info">
            <el-icon class="terminal-icon"><Monitor /></el-icon>
            <span>{{ $tr("容器终端 -") }} {{ containerName }}</span>
            <el-tag type="info" size="small">{{ assetType }} #{{ assetId }}</el-tag>
          </div>
          <div class="terminal-controls">
            <el-select v-model="shell" size="small" style="width: 130px; margin-right: 8px">
              <el-option label="/bin/sh" value="/bin/sh" />
              <el-option label="/bin/bash" value="/bin/bash" />
              <el-option label="/bin/ash" value="/bin/ash" />
            </el-select>
            <el-button type="primary" size="small" :loading="connecting" @click="connect" :disabled="!connectionId">
              {{ isConnected ? "重连" : $tr("连接") }}
            </el-button>
            <el-button type="danger" size="small" @click="disconnect" :disabled="!isConnected">{{ $tr("断开") }}</el-button>
            <el-button type="success" size="small" style="margin-left: 10px" @click="fileVisible = true" :disabled="!connectionId">{{ $tr("文件管理") }}</el-button>
            <el-button type="info" size="small" style="margin-left: 10px" @click="goBack">{{ $tr("返回") }}</el-button>
          </div>
        </div>
      </template>

      <div class="terminal-content">
        <div ref="terminalElement" class="xterm-container" @click="focusTerminal"></div>
        <div v-if="!isConnected" class="terminal-placeholder">
          <div class="placeholder-content">
            <el-icon class="placeholder-icon"><Monitor /></el-icon>
            <div class="placeholder-text">
              <h3>{{ $tr("容器 Web 终端") }}</h3>
              <p>{{ $tr("点击「连接」按钮进入容器终端") }}</p>
              <p v-if="!connectionId" class="tip">{{ $tr("缺少连接参数，请从 Docker 管理页进入") }}</p>
            </div>
          </div>
        </div>
      </div>
    </el-card>

    <el-dialog v-model="fileVisible" :title="`文件管理 - ${containerName}`" width="900px" destroy-on-close>
      <docker-file-manager
        v-if="fileVisible"
        :asset-type="assetType"
        :asset-id="assetId"
        :container="container"
        :connection-id="connectionId"
      />
    </el-dialog>
  </div>
</template>

<script setup>
import { tr } from "@/locales";

import { ref, onMounted, onUnmounted, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import { Monitor } from "@element-plus/icons-vue";
import { Terminal as XTerm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import { getToken } from "@/utils/auth";
import DockerFileManager from "./DockerFileManager.vue";

const route = useRoute();
const router = useRouter();

const assetType = computed(() => String(route.params.assetType || "host"));
const assetId = computed(() => Number(route.params.assetId || 0));
const container = computed(() => String(route.params.container || ""));
const connectionId = computed(() => route.query.connectionId || "");
const containerName = computed(() => String(route.query.name || container.value));

const shell = ref("/bin/sh");
const fileVisible = ref(false);
const connecting = ref(false);
const isConnected = ref(false);
const terminalElement = ref(null);

let terminal = null;
let fitAddon = null;
let websocket = null;

const initTerminal = () => {
  if (!terminalElement.value) return;
  terminal = new XTerm({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
    theme: { background: "#060101", foreground: "#ffffff", cursor: "#ffffff" },
    scrollback: 1000,
    cols: 80,
    rows: 24
  });
  fitAddon = new FitAddon();
  terminal.loadAddon(fitAddon);
  terminal.open(terminalElement.value);
  fitAddon.fit();
  setTimeout(() => terminal.focus(), 100);

  const resizeObserver = new ResizeObserver(() => {
    if (fitAddon) {
      fitAddon.fit();
      if (websocket && websocket.readyState === WebSocket.OPEN) {
        websocket.send(JSON.stringify({ type: "resize", cols: terminal.cols, rows: terminal.rows }));
      }
    }
  });
  resizeObserver.observe(terminalElement.value);

  terminal.onData(data => {
    if (websocket && websocket.readyState === WebSocket.OPEN) {
      websocket.send(data);
    }
  });
};

const getWsBaseUrl = () => {
  const baseUrl = (import.meta.env.VITE_API_BASE_URL || "").replace(/\/$/, "");
  if (baseUrl.startsWith("http")) return baseUrl.replace(/^http/, "ws");
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}${baseUrl}`;
};

const connect = () => {
  if (!connectionId.value) {
    ElMessage.error(tr("缺少连接参数"));
    return;
  }
  if (websocket) websocket.close();
  connecting.value = true;
  const token = getToken();
  const wsUrl = `${getWsBaseUrl()}/api/v1/cmdb/docker/${assetType.value}/${assetId.value}/containers/${encodeURIComponent(container.value)}/terminal?connectionId=${connectionId.value}&shell=${encodeURIComponent(shell.value)}&token=${encodeURIComponent(token || "")}`;
  websocket = new WebSocket(wsUrl);

  const timeout = setTimeout(() => {
    if (websocket.readyState === WebSocket.CONNECTING) {
      websocket.close();
      connecting.value = false;
      ElMessage.error(tr("连接超时，请检查网络或服务状态"));
    }
  }, 10000);

  websocket.onopen = () => {
    clearTimeout(timeout);
    connecting.value = false;
    isConnected.value = true;
    if (terminal) {
      terminal.clear();
      terminal.writeln("\x1B[1;32m🐳 Docker 容器终端连接成功\x1B[0m");
      terminal.writeln(`\x1B[1;36m容器: ${containerName.value}\x1B[0m`);
      terminal.writeln("");
      websocket.send(JSON.stringify({ type: "resize", cols: terminal.cols, rows: terminal.rows }));
      setTimeout(() => terminal.focus(), 200);
    }
  };

  websocket.onmessage = event => {
    if (terminal) terminal.write(event.data);
  };

  websocket.onerror = () => {
    ElMessage.error(tr("终端连接出错"));
    connecting.value = false;
    isConnected.value = false;
  };

  websocket.onclose = () => {
    isConnected.value = false;
    connecting.value = false;
  };
};

const disconnect = () => {
  if (websocket) websocket.close();
  websocket = null;
  isConnected.value = false;
};

const focusTerminal = () => terminal && terminal.focus();
const goBack = () => router.back();

onMounted(initTerminal);
onUnmounted(() => {
  if (websocket) websocket.close();
  if (terminal) terminal.dispose();
});
</script>

<style scoped>
.docker-terminal-page {
  padding: 22px;
}
.terminal-card :deep(.el-card__body) {
  padding: 0;
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
  font-weight: 600;
}
.terminal-icon {
  font-size: 20px;
  color: #409eff;
}
.terminal-controls {
  display: flex;
  align-items: center;
}
.terminal-content {
  position: relative;
  height: calc(100vh - 220px);
  min-height: 420px;
  background: #060101;
}
.xterm-container {
  width: 100%;
  height: 100%;
  padding: 8px;
}
.terminal-placeholder {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #060101;
}
.placeholder-content {
  text-align: center;
  color: #8c9bab;
}
.placeholder-icon {
  font-size: 42px;
}
.placeholder-text h3 {
  color: #d7e3f4;
  margin: 8px 0;
}
.tip {
  color: #e6a23c;
}
</style>
