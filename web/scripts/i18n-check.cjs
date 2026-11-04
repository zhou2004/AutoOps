const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const Module = require("node:module");
const ts = require("typescript");
const { nextTick, computed } = require("vue");
const source = path.join(__dirname, "../src/locales/index.ts");
const values = new Map();
global.localStorage = {
  getItem: key => values.get(key) ?? null,
  setItem: (key, value) => values.set(key, value)
};
global.document = { documentElement: { lang: "" } };
const compiled = ts.transpileModule(fs.readFileSync(source, "utf8"), {
  compilerOptions: {
    module: ts.ModuleKind.CommonJS,
    target: ts.ScriptTarget.ES2022,
    esModuleInterop: true
  }
});
const loaded = new Module(source, module);
loaded.filename = source;
loaded.paths = Module._nodeModulePaths(path.dirname(source));
loaded._compile(compiled.outputText, source);
const { tr, setLocale, localizeError } = loaded.exports;

test("language switches without changing business values; punctuation remains literal", async () => {
  assert.equal(tr("AI运维"), "AI运维");
  setLocale("en-US");
  await nextTick();
  assert.equal(tr("AI运维"), "AI Operations");
  assert.equal(document.documentElement.lang, "en-US");
  assert.equal(values.get("autoops-locale"), "en-US");
  assert.equal(
    tr('JSON格式，如 {"env": "prod"}'),
    'JSON, e.g. {"env": "prod"}'
  );
  assert.equal(
    tr("填写Go Template格式内容... 比如 {{.Alerts}}"),
    "Enter Go Template content, e.g. {{.Alerts}}"
  );
  assert.equal(tr("如: @all, 或手机号"), "E.g. @all or phone number");
  assert.equal(tr("| 共包含实例数:"), "| Total instances:");
  assert.equal(tr("生产服务器-user-supplied"), "生产服务器-user-supplied");
  assert.equal(
    localizeError("Diagnosis timed out / 诊断超时"),
    "Diagnosis timed out"
  );
  setLocale("zh-CN");
  await nextTick();
  assert.equal(tr("AI运维"), "AI运维");
  assert.equal(
    tr("填写Go Template格式内容... 比如 {{.Alerts}}"),
    "填写Go Template格式内容... 比如 {{.Alerts}}"
  );
  assert.equal(localizeError("Diagnosis timed out / 诊断超时"), "诊断超时");
});

test("database labels update in place and interpolation preserves user values", async () => {
  function domain(relative) {
    const filename = path.join(__dirname, relative);
    const result = ts.transpileModule(fs.readFileSync(filename, "utf8"), {
      compilerOptions: {
        module: ts.ModuleKind.CommonJS,
        target: ts.ScriptTarget.ES2022,
        esModuleInterop: true
      }
    });
    const instance = new Module(filename, module);
    instance.filename = filename;
    instance.paths = Module._nodeModulePaths(path.dirname(filename));
    instance.require = name =>
      name === "@/locales"
        ? loaded.exports
        : name === "@/utils/request"
          ? () => {
              throw new Error("No requests expected");
            }
          : Module.prototype.require.call(instance, name);
    instance._compile(result.outputText, filename);
    return instance.exports;
  }
  const {
    label,
    operationStatusLabel,
    capabilityLabel,
    preflightCheckLabel,
    operationMessage
  } = domain("../src/api/databasePlatform.ts");
  const { backupMethods, backupDisplayName } = domain(
    "../src/utils/databaseBackup.ts"
  );
  const capability = {
    action: "refresh",
    label: "刷新状态",
    labelEn: "Refresh status"
  };
  const view = computed(() => [
    label("healthy"),
    operationStatusLabel("unknown"),
    capabilityLabel(capability),
    backupMethods.logical.label
  ]);
  setLocale("zh-CN");
  assert.deepEqual(view.value, ["正常", "结果待核验", "刷新状态", "逻辑全量"]);
  setLocale("en-US");
  await nextTick();
  assert.deepEqual(view.value, [
    "Healthy",
    "Outcome unresolved",
    "Refresh status",
    "Logical full backup"
  ]);
  assert.equal(
    capabilityLabel({ ...capability, labelEn: undefined }),
    "Refresh status"
  );
  const name = "发布前备份 · {value1} $&";
  assert.equal(
    preflightCheckLabel(`受管实例 ${name}`),
    `Managed instance: ${name}`
  );
  assert.equal(
    operationMessage("实际结果由核验任务 #233 确认，未重放原切换或恢复命令"),
    "Actual results confirmed by verification task #233; the original switchover or recovery command was not replayed"
  );
  assert.equal(backupDisplayName({ name, method: "logical" }), name);
  assert.equal(tr("操作 {value0}", { value0: name }), `Actions for ${name}`);
  assert.equal(tr("操作 {value0}"), "Actions for {value0}");
  setLocale("zh-CN");
  await nextTick();
  assert.deepEqual(view.value, ["正常", "结果待核验", "刷新状态", "逻辑全量"]);
  assert.equal(tr("操作 {value0}", { value0: name }), `操作 ${name}`);
});
