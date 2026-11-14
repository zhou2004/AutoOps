<template>
  <Teleport to="body" :disabled="!expanded">
    <div ref="host" class="log-console-panel" :class="{ expanded }">
      <div class="console-bar">
        <span class="console-title"
          ><span class="console-dot" :class="{ live: follow }" />{{
            title || $tr("日志内容")
          }}</span
        >
        <div class="console-tools">
          <el-button text size="small" @click="scrollToBottom">{{
            $tr("滚动到底部")
          }}</el-button>
          <el-button text size="small" @click="expanded = !expanded">{{
            $tr(expanded ? "退出全屏" : "全屏")
          }}</el-button>
        </div>
      </div>
      <div
        ref="viewport"
        class="console-viewport"
        :aria-label="$tr('日志内容')"
      >
        <div ref="element" class="console-terminal" />
      </div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import { ref, watch, nextTick, onMounted, onBeforeUnmount } from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
const props = withDefaults(
  defineProps<{
    text?: string;
    title?: string;
    follow?: boolean;
    wrap?: boolean;
  }>(),
  { text: "", title: "", follow: false, wrap: true }
);
const host = ref<HTMLElement>(),
  viewport = ref<HTMLElement>(),
  element = ref<HTMLElement>(),
  expanded = ref(false);
let terminal: Terminal | undefined,
  fitAddon: FitAddon | undefined,
  observer: ResizeObserver | undefined;
let rendered = "",
  pendingFrame = 0,
  alive = true;
// Render log data literally: OSC/CSI sequences from requests must never control the terminal.
function literal(value: string) {
  return value.replace(/[\x00-\x08\x0b-\x1f\x7f-\x9f]/g, c =>
    c === "\r" ? "" : `\\x${c.charCodeAt(0).toString(16).padStart(2, "0")}`
  );
}
function fit() {
  if (!element.value || !viewport.value || !terminal || !alive) return;
  const width = viewport.value.clientWidth;
  if (width < 20 || viewport.value.clientHeight < 20) return;
  const length = props.wrap
    ? 0
    : Math.min(
        2000,
        props.text
          .split("\n")
          .reduce((max, line) => Math.max(max, line.length), 0)
      );
  element.value.style.width = `${Math.max(width, length * 8 + 30)}px`;
  fitAddon?.fit();
}
function render() {
  pendingFrame = 0;
  if (!terminal || !alive) return;
  fit();
  const next = props.text;
  const append = next.startsWith(rendered) ? next.slice(rendered.length) : next;
  if (!next.startsWith(rendered)) terminal.reset();
  rendered = next;
  terminal.write(literal(append), () => {
    if (alive && props.follow) terminal?.scrollToBottom();
  });
}
function schedule() {
  if (!pendingFrame) pendingFrame = requestAnimationFrame(render);
}
function scrollToBottom() {
  terminal?.scrollToBottom();
}
function key(event: KeyboardEvent) {
  if (event.key === "Escape") expanded.value = false;
}
onMounted(() => {
  terminal = new Terminal({
    disableStdin: true,
    cursorBlink: false,
    convertEol: true,
    fontSize: 13,
    lineHeight: 1.4,
    fontFamily: "'Courier New', Consolas, monospace",
    scrollback: 10000,
    theme: {
      background: "#1e1e1e",
      foreground: "#d4d4d4",
      selectionBackground: "#ffffff40"
    }
  });
  fitAddon = new FitAddon();
  terminal.loadAddon(fitAddon);
  terminal.open(element.value!);
  observer = new ResizeObserver(schedule);
  observer.observe(viewport.value!);
  schedule();
  window.addEventListener("keydown", key);
});
watch(() => [props.text, props.wrap], schedule);
watch(expanded, () => nextTick(schedule));
onBeforeUnmount(() => {
  alive = false;
  observer?.disconnect();
  cancelAnimationFrame(pendingFrame);
  terminal?.dispose();
  window.removeEventListener("keydown", key);
});
defineExpose({ scrollToBottom, fit });
</script>
<style scoped>
.log-console-panel {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 260px;
  height: 100%;
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  overflow: hidden;
  background: #1e1e1e;
}
.console-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
  padding: 6px 12px;
  color: #d4d4d4;
  background: #252526;
  border-bottom: 1px solid #393939;
}
.console-title {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 8px;
  font:
    12px ui-monospace,
    monospace;
  overflow-wrap: anywhere;
  flex: 1;
}
.console-dot {
  flex-shrink: 0;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #8b8b8b;
}
.console-dot.live {
  background: #67c23a;
}
.console-tools {
  display: flex;
  gap: 4px;
}
.console-tools :deep(.el-button.is-text) {
  --el-button-text-color: #d4d4d4;
  --el-button-hover-text-color: #ffffff;
  --el-button-hover-bg-color: #393939;
  color: #d4d4d4;
  margin: 0;
}
.console-viewport {
  flex: 1;
  min-height: 0;
  min-width: 0;
  overflow: auto;
  padding: 10px 0 10px 10px;
  box-sizing: border-box;
}
.console-terminal {
  height: 100%;
  min-height: 0;
}
.console-terminal :deep(.xterm) {
  height: 100%;
  padding-right: 10px;
}
.console-terminal :deep(.xterm-viewport::-webkit-scrollbar) {
  width: 8px;
  height: 8px;
}
.console-terminal :deep(.xterm-viewport::-webkit-scrollbar-thumb) {
  background: #555;
  border-radius: 4px;
}
.console-terminal :deep(.xterm-viewport::-webkit-scrollbar-track) {
  background: #2d2d2d;
}
.expanded {
  position: fixed;
  inset: 12px;
  z-index: 3100;
  height: auto;
  max-height: none;
  min-height: 0;
  box-shadow: 0 0 0 20px #0008;
}
</style>
