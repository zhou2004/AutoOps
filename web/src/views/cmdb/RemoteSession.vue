<template>
  <div class="asset-page remote-session">
    <header class="asset-page-header">
      <div>
        <span class="asset-eyebrow">REMOTE SESSION</span>
        <h1>{{ profile?.name || route.query.name || "远程会话" }}</h1>
        <p>
          {{
            profile
              ? `${profile.protocol.toUpperCase()} · ${profile.address}:${profile.port}`
              : "正在加载连接信息"
          }}
        </p>
      </div>
      <div class="asset-actions">
        <el-tag :type="state === 'connected' ? 'success' : 'info'">{{
          stateLabels[state]
        }}</el-tag
        ><el-tag v-if="profile?.readOnly" type="warning">{{ $tr("只读") }}</el-tag
        ><el-button :disabled="state === 'connecting'" @click="connect"
          >{{ $tr("重新连接") }}</el-button
        ><el-button :disabled="state !== 'connected'" @click="disconnect"
          >{{ $tr("断开") }}</el-button
        ><el-button @click="fullscreen">{{ $tr("全屏") }}</el-button>
      </div>
    </header>
    <el-alert
      v-if="error"
      :title="$tr(error)"
      type="error"
      :closable="false"
      show-icon
    />
    <div
      ref="viewport"
      class="remote-viewport"
      tabindex="0"
      :aria-label='$tr("远程桌面或终端")'
      @blur="releaseKeys"
    >
      <div ref="display" class="remote-display" />
    </div>
    <p class="asset-muted">
      {{
        profile?.protocol === "ssh"
          ? "终端输入会直接发送到远端。"
          : "点击桌面后操作键盘与鼠标。"
      }}
      {{ $tr("关闭此页面将结束会话。") }}
    </p>
  </div>
</template>

<script setup>
import { tr } from "@/locales";

import { onMounted, onBeforeUnmount, ref, nextTick } from "vue";
import { useRoute } from "vue-router";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import Guacamole from "guacamole-common-js";
import "@xterm/xterm/css/xterm.css";
import api from "@/api/cmdb";
const route = useRoute(),
  viewport = ref(),
  display = ref(),
  profile = ref(null),
  error = ref(""),
  state = ref("disconnected");
const stateLabels = {
  connected: "已连接",
  connecting: "连接中",
  disconnected: "已断开"
};
let socket,
  terminal,
  fit,
  client,
  keyboard,
  observer,
  generation = 0,
  disposed = false;
function unpack(res) {
  if (res.data.code !== 200) throw new Error(res.data.message || tr("操作失败"));
  return res.data.data;
}
function wsUrl(path) {
  const base = import.meta.env.VITE_API_BASE_URL || window.location.origin;
  const url = new URL(
    path,
    /^https?:\/\//.test(base) ? base : window.location.origin
  );
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  return url.toString();
}
function releaseKeys() {
  keyboard?.reset();
}
function disconnect() {
  generation++;
  releaseKeys();
  if (keyboard) {
    keyboard.onkeydown = null;
    keyboard.onkeyup = null;
    keyboard = null;
  }
  if (socket) {
    socket.onclose = null;
    socket.onerror = null;
    socket.onmessage = null;
    socket.close();
    socket = null;
  }
  if (client) {
    client.onstatechange = null;
    client.onerror = null;
    client.disconnect();
    client = null;
  }
  terminal?.dispose();
  terminal = null;
  fit = null;
  display.value?.replaceChildren();
  state.value = "disconnected";
}
function resize() {
  if (terminal) {
    fit?.fit();
    if (socket?.readyState === WebSocket.OPEN)
      socket.send(
        JSON.stringify({
          type: "resize",
          cols: terminal.cols,
          rows: terminal.rows
        })
      );
  }
  if (client) {
    const remote = client.getDisplay();
    const width = remote.getWidth();
    if (width) remote.scale(Math.min(1, viewport.value.clientWidth / width));
    client.sendSize(viewport.value.clientWidth, viewport.value.clientHeight);
  }
}
async function connect() {
  disconnect();
  error.value = "";
  state.value = "connecting";
  const current = generation;
  try {
    const type = String(route.query.assetType || ""),
      id = Number(route.query.assetId),
      connectionId = Number(route.query.connectionId);
    if (!["host", "physical", "network"].includes(type) || !id || !connectionId)
      throw new Error(tr("连接参数无效"));
    const list = unpack(await api.getAssetConnections(type, id));
    if (disposed || current !== generation) return;
    profile.value = (list.connections || []).find(p => p.id === connectionId);
    if (!profile.value) throw new Error(tr("连接档案不存在或没有访问权限"));
    const ticket = unpack(
      await api.createConnectionTicket(type, id, connectionId)
    );
    if (disposed || current !== generation) return;
    await nextTick();
    if (ticket.protocol === "ssh") {
      const decoder = new TextDecoder();
      terminal = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: "Consolas, monospace",
        theme: { background: "#0b1220", foreground: "#e2e8f0" }
      });
      fit = new FitAddon();
      terminal.loadAddon(fit);
      terminal.open(display.value);
      fit.fit();
      socket = new WebSocket(wsUrl(ticket.websocketPath));
      socket.binaryType = "arraybuffer";
      socket.onopen = () => {
        resize();
      };
      socket.onmessage = e => {
        if (typeof e.data !== "string") {
          terminal?.write(decoder.decode(e.data, { stream: true }));
          return;
        }
        try {
          const event = JSON.parse(e.data);
          if (event.type === "ready") {
            state.value = "connected";
            resize();
            terminal?.focus();
          } else if (event.type === "error") {
            error.value = event.message || "SSH 会话建立失败";
            state.value = "disconnected";
          }
        } catch {
          error.value = "终端通信格式无效，请刷新页面并重新连接。";
          state.value = "disconnected";
        }
      };
      socket.onclose = () => {
        if (state.value === "connecting" && !error.value)
          error.value = "终端通道在 SSH 会话就绪前关闭，请重新连接。";
        state.value = "disconnected";
      };
      socket.onerror = () => {
        error.value =
          "无法建立终端通道，请检查后端服务、代理的 WebSocket 转发及登录状态。";
        state.value = "disconnected";
      };
      terminal.onData(data => {
        if (
          state.value === "connected" &&
          socket?.readyState === WebSocket.OPEN
        )
          socket.send(JSON.stringify({ type: "input", data }));
      });
    } else {
      const desktopUrl = new URL(wsUrl(ticket.websocketPath));
      const tunnelParams = desktopUrl.searchParams.toString();
      desktopUrl.search = "";
      client = new Guacamole.Client(
        new Guacamole.WebSocketTunnel(desktopUrl.toString())
      );
      const el = client.getDisplay().getElement();
      el.tabIndex = 0;
      el.addEventListener("blur", releaseKeys);
      display.value.appendChild(el);
      client.onerror = e => {
        error.value = e.message || "远程桌面连接失败";
        state.value = "disconnected";
      };
      client.onstatechange = s => {
        if (s === 3) {
          state.value = "connected";
          resize();
          el.focus();
        } else if (s === 5) state.value = "disconnected";
      };
      client.getDisplay().onresize = resize;
      if (!profile.value.readOnly) {
        const mouse = new Guacamole.Mouse(el);
        mouse.onEach(["mousedown", "mouseup", "mousemove"], event => {
          el.focus();
          client?.sendMouseState(event.state, true);
        });
        keyboard = new Guacamole.Keyboard(el);
        keyboard.onkeydown = key => {
          client?.sendKeyEvent(1, key);
          return false;
        };
        keyboard.onkeyup = key => client?.sendKeyEvent(0, key);
      }
      client.connect(tunnelParams);
    }
  } catch (e) {
    if (current === generation && !disposed) {
      error.value = e.message || "连接失败";
      state.value = "disconnected";
    }
  }
}
async function fullscreen() {
  if (viewport.value?.requestFullscreen)
    await viewport.value.requestFullscreen().catch(() => {});
  resize();
}
onMounted(() => {
  observer = new ResizeObserver(resize);
  observer.observe(viewport.value);
  window.addEventListener("beforeunload", disconnect);
  connect();
});
onBeforeUnmount(() => {
  disposed = true;
  observer?.disconnect();
  window.removeEventListener("beforeunload", disconnect);
  disconnect();
});
</script>
