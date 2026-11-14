const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const Module = require("node:module");
const ts = require("typescript");

const filename = path.join(__dirname, "../src/views/webservice/nativeEditorIndent.ts");
const instance = new Module(filename, module);
instance._compile(ts.transpileModule(fs.readFileSync(filename, "utf8"), {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
}).outputText, filename);
const { indentConfiguration } = instance.exports;
function apply(value, start, end = start, outdent = false) {
  const edit = indentConfiguration(value, start, end, outdent);
  if (!edit) return { value, start, end };
  return {
    value: value.slice(0, edit.from) + edit.text + value.slice(edit.to),
    start: edit.selectionStart,
    end: edit.selectionEnd
  };
}

test("Tab aligns a caret to the next four-column stop without rewriting the file", () => {
  assert.deepEqual(apply("ab|\r\n# untouched", 2), {
    value: "ab  |\r\n# untouched", start: 4, end: 4
  });
  assert.deepEqual(apply("\tab", 3), { value: "\tab  ", start: 5, end: 5 });
});

test("multiline selections indent all selected lines and preserve CRLF and unknown directives", () => {
  const value = "http {\r\n  custom_directive on;\r\n  # comment\r\n}\r\n";
  const start = value.indexOf("custom"), end = value.indexOf("comment") + "comment".length;
  const result = apply(value, start, end);
  assert.equal(result.value, "http {\r\n      custom_directive on;\r\n      # comment\r\n}\r\n");
  assert.equal(result.value.slice(result.start, result.end), "custom_directive on;\r\n      # comment");
  assert.deepEqual(apply(result.value, result.start, result.end, true), { value, start, end });
});

test("a selection ending at the next line start does not indent that next line", () => {
  assert.deepEqual(apply("one\ntwo\nthree", 0, 8), {
    value: "    one\n    two\nthree", start: 4, end: 16
  });
});

test("Shift+Tab handles tabs, partial indentation and a caret inside indentation", () => {
  assert.deepEqual(apply("\tlisten 80;\n  server_name test;", 0, 31, true), {
    value: "listen 80;\nserver_name test;", start: 0, end: 28
  });
  assert.deepEqual(apply("    listen 80;", 2, 2, true), {
    value: "listen 80;", start: 0, end: 0
  });
});

test("empty first lines and trailing newlines retain their positions", () => {
  assert.deepEqual(apply("\nnext", 0), { value: "    \nnext", start: 4, end: 4 });
  assert.deepEqual(apply("a\nb\n", 0, 4), { value: "    a\n    b\n", start: 4, end: 12 });
  assert.equal(indentConfiguration("server {}", 0, 9, true), undefined);
});
