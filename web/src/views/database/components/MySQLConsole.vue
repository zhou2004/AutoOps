<template>
  <section class="mysql-console">
    <header class="console-toolbar">
      <div>
        <span class="console-dot" /> MySQL {{ $tr("控制台") }}
        <el-tag size="small" type="info">{{
          database || $tr("未选择数据库")
        }}</el-tag>
      </div>
      <div>
        <el-button size="small" @click="clear">{{ $tr("清屏") }}</el-button
        ><el-button size="small" :disabled="busy" @click="run">{{
          $tr("执行")
        }}</el-button>
      </div>
    </header>
    <div class="console-layout">
      <div
        ref="element"
        class="console-screen"
        :aria-label="$tr('MySQL SQL 终端')"
      />
    </div>
  </section>
</template>
<script setup>
import {
  ref,
  computed,
  onMounted,
  onBeforeUnmount,
  watch,
  nextTick
} from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import { ElMessageBox } from "element-plus";
import api from "@/api/cmdb";
import { databaseResult } from "@/api/mysqlWorkbench";
import { tr } from "@/locales";
const props = defineProps({
  source: { type: Object, required: true },
  database: { type: String, default: "" },
  catalog: { type: Array, default: () => [] },
  tables: { type: Array, default: () => [] },
  columns: { type: Array, default: () => [] },
  canWrite: Boolean,
  active: Boolean
});
const emit = defineEmits(["database-change", "executed"]);
const element = ref(),
  input = ref(""),
  cursorPosition = ref(0),
  busy = ref(false);
let terminal,
  fit,
  observer,
  disposed = false,
  cursor = 0,
  history = [],
  historyAt = 0;
const keywords = [
  "SELECT",
  "SELECT * FROM",
  "SHOW DATABASES;",
  "SHOW TABLES;",
  "SHOW VARIABLES;",
  "SHOW STATUS;",
  "SHOW CREATE TABLE",
  "DESCRIBE",
  "EXPLAIN",
  "INSERT INTO",
  "UPDATE",
  "DELETE FROM",
  "WHERE",
  "ORDER BY",
  "GROUP BY",
  "LIMIT",
  "COUNT(*)",
  "USE",
  "help",
  "clear"
];
const prefix = computed(
  () =>
    input.value
      .slice(0, cursorPosition.value)
      .match(/[\w\u0080-\uffff]+$/)?.[0] || ""
);
const suggestions = computed(() => {
  const before = input.value.slice(0, cursorPosition.value),
    word = prefix.value.toLowerCase();
  let choices = /\bUSE\s+\S*$/i.test(before)
    ? props.catalog
    : /\b(FROM|JOIN|INTO|UPDATE|TABLE|DESCRIBE|DESC)\s+\S*$/i.test(before)
      ? props.tables.map(t => t.name)
      : [
          ...keywords,
          ...props.tables.map(t => t.name),
          ...props.columns.map(c => c.name)
        ];
  return [...new Set(choices)]
    .filter(s => !word || s.toLowerCase().startsWith(word))
    .slice(0, 9);
});
const completion = computed(() => {
  if (!prefix.value || cursorPosition.value !== input.value.length) return "";
  return (
    suggestions.value.find(item => item.length > prefix.value.length) || ""
  );
});
function clean(value) {
  return String(value ?? "NULL").replace(/[\x00-\x1f\x7f-\x9f]/g, " ");
}
function prompt() {
  return `mysql [${clean(props.database || "default")}]> `;
}
function render() {
  cursorPosition.value = cursor;
  if (!terminal || busy.value || disposed) return;
  const p = prompt(),
    width = Math.max(12, terminal.cols - p.length - 2),
    start = Math.max(0, cursor - width + 1),
    visible = input.value.slice(start, start + width);
  terminal.write("\r\x1b[2K\x1b[38;5;81m" + p + "\x1b[0m" + clean(visible));
  const back = visible.length - (cursor - start);
  if (back > 0) terminal.write(`\x1b[${back}D`);
  const ghost = clean(completion.value.slice(prefix.value.length)).slice(
    0,
    Math.max(0, width - visible.length)
  );
  if (ghost) {
    // Preview only: keep the input and cursor unchanged until Tab is pressed.
    // Disable wrapping while drawing so wide identifiers cannot spill onto a new line.
    terminal.write(
      "\x1b7\x1b[?7l\x1b[38;2;93;111;137m" + ghost + "\x1b[0m\x1b[?7h\x1b8"
    );
  }
}
function setInput(value) {
  input.value = value;
  cursor = value.length;
  render();
}
function complete(value) {
  if (busy.value) return;
  const start = cursor - prefix.value.length;
  input.value =
    input.value.slice(0, start) + value + " " + input.value.slice(cursor);
  cursor = start + value.length + 1;
  render();
  terminal?.focus();
}
function clear() {
  terminal?.clear();
  render();
  terminal?.focus();
}
function showResult(data) {
  if (data.columns) {
    const columns = data.columns,
      rows = data.rows || [];
    const widths = columns.map(c =>
      Math.min(
        36,
        Math.max(
          c.length,
          ...rows.slice(0, 100).map(row => clean(row[c]).length)
        )
      )
    );
    const line = values =>
      values
        .map((v, i) => clean(v).slice(0, widths[i]).padEnd(widths[i]))
        .join(" | ");
    terminal.writeln(line(columns));
    terminal.writeln(widths.map(n => "─".repeat(n)).join("─┼─"));
    for (const row of rows) terminal.writeln(line(columns.map(c => row[c])));
    terminal.writeln(
      `\x1b[32m${data.returnedRows} rows (${data.durationMs} ms)${data.truncated ? " — " + tr("结果已截断") : ""}\x1b[0m`
    );
  } else
    terminal.writeln(
      `\x1b[32mQuery OK, ${data.affectedRows} rows affected (${data.durationMs} ms)\x1b[0m`
    );
}
async function run() {
  if (busy.value || !input.value.trim() || !terminal) return;
  const statement = input.value.trim();
  const selectedDatabase = props.database;
  terminal.write("\r\x1b[2K");
  terminal.writeln(prompt() + clean(statement));
  input.value = "";
  cursor = 0;
  history.push(statement);
  history = history.slice(-100);
  historyAt = history.length;
  if (/^(clear|\\c);?$/i.test(statement)) {
    clear();
    return;
  }
  if (/^help;?$/i.test(statement)) {
    terminal.writeln(
      tr(
        "支持 SELECT / SHOW / DESCRIBE / EXPLAIN / WITH 和 INSERT / UPDATE / DELETE。USE 切换当前数据库。"
      )
    );
    terminal.writeln(
      tr("Tab 补全；方向键浏览历史；Ctrl+L 清屏；Ctrl+C 清除输入。")
    );
    render();
    return;
  }
  const use = statement.match(/^USE\s+(?:`((?:``|[^`])+)`|([^;\s]+))\s*;?$/i);
  if (use) {
    const db = (use[1] || use[2]).replace(/``/g, "`");
    if (props.catalog.includes(db)) {
      emit("database-change", db);
      terminal.writeln("Database changed");
    } else terminal.writeln(tr("数据库不存在或没有访问权限"));
    await nextTick();
    render();
    return;
  }
  busy.value = true;
  try {
    const operation = statement
      .replace(/^\s*(?:\/\*[\s\S]*?\*\/\s*|--[^\n]*\n\s*)*/, "")
      .match(/^[a-z]+/i)?.[0]
      .toUpperCase();
    let data;
    if (
      ["SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN", "WITH"].includes(
        operation
      )
    )
      data = await databaseResult(
        api.queryDatabase(props.source.id, {
          database: selectedDatabase,
          sql: statement
        })
      );
    else if (["INSERT", "UPDATE", "DELETE"].includes(operation)) {
      if (!props.canWrite) throw Error(tr("没有此数据源的写操作权限"));
      const { value } = await ElMessageBox.prompt(
        `${tr("请输入完整数据源名称确认写操作")}：${props.source.name}`,
        tr("执行 SQL"),
        { inputValidator: v => v === props.source.name || tr("名称不匹配") }
      );
      data = await databaseResult(
        api.executeDatabaseWrite(props.source.id, {
          database: selectedDatabase,
          sql: statement,
          confirmName: value
        })
      );
    } else
      throw Error(
        tr("此控制台支持单条查询与 DML；结构及参数变更请使用工作台对应功能。")
      );
    if (!disposed) {
      showResult(data);
      emit("executed");
    }
  } catch (e) {
    if (!disposed)
      terminal.writeln(
        "\x1b[31m" + clean(e?.message || tr("操作已取消")) + "\x1b[0m"
      );
  } finally {
    busy.value = false;
    if (!disposed) {
      terminal.writeln("");
      render();
      terminal.focus();
    }
  }
}
function onData(data) {
  if (busy.value) return;
  if (data === "\r") {
    void run();
    return;
  }
  if (data === "\t") {
    if (completion.value) complete(completion.value);
    return;
  }
  if (data === "\x1b[A") {
    historyAt = Math.max(0, historyAt - 1);
    setInput(history[historyAt] || "");
    return;
  }
  if (data === "\x1b[B") {
    historyAt = Math.min(history.length, historyAt + 1);
    setInput(history[historyAt] || "");
    return;
  }
  if (data === "\x1b[D") {
    cursor = Math.max(0, cursor - 1);
    render();
    return;
  }
  if (data === "\x1b[C") {
    cursor = Math.min(input.value.length, cursor + 1);
    render();
    return;
  }
  if (data === "\x7f") {
    if (cursor > 0) {
      input.value =
        input.value.slice(0, cursor - 1) + input.value.slice(cursor);
      cursor--;
    }
    render();
    return;
  }
  if (data === "\x03") {
    setInput("");
    return;
  }
  if (data === "\x0c") {
    clear();
    return;
  }
  if (data === "\x01") {
    cursor = 0;
    render();
    return;
  }
  if (data === "\x05") {
    cursor = input.value.length;
    render();
    return;
  }
  if (data.startsWith("\x1b")) return;
  const text = data
    .replace(/\r\n?/g, "\n")
    .replace(/[\x00-\x09\x0b-\x1f\x7f-\x9f]/g, " ");
  if (input.value.length + text.length > 65536) return;
  input.value = input.value.slice(0, cursor) + text + input.value.slice(cursor);
  cursor += text.length;
  render();
}
watch(
  [
    () => props.database,
    () => props.catalog,
    () => props.tables,
    () => props.columns
  ],
  () => nextTick(render)
);
watch(
  () => props.active,
  async active => {
    if (active) {
      await nextTick();
      fit?.fit();
      render();
      terminal?.focus();
    }
  }
);
onMounted(() => {
  terminal = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
    theme: { background: "#090e18", foreground: "#d7e3f4", cursor: "#6ec6ff" },
    scrollback: 3000,
    convertEol: true
  });
  fit = new FitAddon();
  terminal.loadAddon(fit);
  terminal.open(element.value);
  fit.fit();
  terminal.onData(onData);
  terminal.writeln("\x1b[1;36mMySQL SQL Console\x1b[0m");
  terminal.writeln(
    tr(
      "输入 SQL 并按 Enter 执行；灰色虚影为补全提示，Tab 接受，↑↓ 查看历史。粘贴内容不会自动执行。"
    )
  );
  terminal.writeln("");
  render();
  observer = new ResizeObserver(() => {
    if (element.value?.clientWidth && element.value?.clientHeight) {
      fit.fit();
      render();
    }
  });
  observer.observe(element.value);
});
onBeforeUnmount(() => {
  disposed = true;
  observer?.disconnect();
  terminal?.dispose();
});
defineExpose({ insert: setInput, focus: () => terminal?.focus() });
</script>
<style scoped>
.mysql-console {
  border: 1px solid var(--border);
  border-radius: 12px;
  overflow: hidden;
}
.console-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: var(--bg-card-alt);
}
.console-toolbar > div {
  display: flex;
  align-items: center;
  gap: 10px;
}
.console-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #22c55e;
}
.console-layout {
  background: #090e18;
}
.console-screen {
  height: 480px;
  min-width: 0;
  padding: 12px;
}
@media (max-width: 900px) {
  .console-screen {
    height: 420px;
  }
}
</style>
