const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const Module = require("node:module");
const ts = require("typescript");
const { nextTick } = require("vue");
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
