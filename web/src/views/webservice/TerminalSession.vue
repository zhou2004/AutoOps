<template>
  <div>
    <div class="toolbar">
      <el-tag :type="connected ? 'success' : 'info'">{{
        $tr(connected ? "已连接" : "已断开")
      }}</el-tag
      ><el-button :loading="connecting" @click="connect">{{
        $tr("重新连接")
      }}</el-button
      ><el-button @click="disconnect">{{ $tr("断开") }}</el-button>
    </div>
    <el-alert v-if="error" :title="error" type="error" />
    <div ref="element" class="terminal" />
  </div>
</template>
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import { tr } from "@/locales";
import { webRequest } from "@/api/webservice";
import { platformApiBase } from "@/api/platformPermissions";
const props = defineProps<{ instanceId: number }>();
const element = ref<HTMLElement>(),
  connected = ref(false),
  connecting = ref(false),
  error = ref("");
let term: Terminal,
  fit: FitAddon,
  ws: WebSocket | undefined,
  observer: ResizeObserver,
  gen = 0;
function disconnect() {
  gen++;
  ws?.close();
  ws = undefined;
  connected.value = false;
}
function resize() {
  fit?.fit();
  if (ws?.readyState === WebSocket.OPEN)
    ws.send(
      JSON.stringify({ type: "resize", rows: term.rows, cols: term.cols })
    );
}
async function connect() {
  disconnect();
  const request = gen;
  connecting.value = true;
  error.value = "";
  try {
    const ticket = await webRequest("/terminal-tickets", "POST", {
      instanceId: props.instanceId
    });
    if (request !== gen) return;
    const url = new URL(
      `${platformApiBase}/web/terminal/connect`,
      location.origin
    );
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    url.searchParams.set("ticket", ticket.ticket);
    ws = new WebSocket(url);
    ws.binaryType = "arraybuffer";
    ws.onopen = () => {
      if (request !== gen) return;
      connected.value = true;
      resize();
    };
    ws.onmessage = e => {
      if (request !== gen) return;
      term.write(
        e.data instanceof ArrayBuffer ? new Uint8Array(e.data) : String(e.data)
      );
    };
    ws.onclose = () => {
      if (request !== gen) return;
      connected.value = false;
    };
    ws.onerror = () => {
      if (request !== gen) return;
      error.value = tr("终端连接失败");
    };
  } catch (e) {
    if (request === gen) error.value = (e as Error).message;
  } finally {
    if (request === gen) connecting.value = false;
  }
}
onMounted(() => {
  term = new Terminal({ cursorBlink: true, scrollback: 3000, fontSize: 14 });
  fit = new FitAddon();
  term.loadAddon(fit);
  term.open(element.value!);
  term.onData(data => {
    if (ws?.readyState === WebSocket.OPEN)
      ws.send(JSON.stringify({ type: "input", data }));
  });
  observer = new ResizeObserver(resize);
  observer.observe(element.value!);
  connect();
});
onUnmounted(() => {
  disconnect();
  observer?.disconnect();
  term?.dispose();
});
</script>
<style scoped>
.toolbar {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-bottom: 12px;
}
.terminal {
  height: 60vh;
  min-height: 320px;
  background: #000;
  padding: 12px;
  border-radius: 8px;
  overflow: hidden;
}
</style>
