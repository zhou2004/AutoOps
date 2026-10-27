// Reports untranslated static UI text without touching business data or source files.
// Usage: node scripts/i18n-audit.cjs [--json]
const fs = require("node:fs");
const path = require("node:path");
const { parse } = require("vue/compiler-sfc");
const catalog = require("../src/locales/en-US.json");
const files = [];
function walk(dir) {
  for (const item of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, item.name);
    if (item.isDirectory()) walk(p);
    else if (p.endsWith(".vue")) files.push(p);
  }
}
walk(path.join(__dirname, "../src"));
const missing = [];
const attributes = new Set([
  "label",
  "placeholder",
  "title",
  "description",
  "content",
  "empty-text",
  "active-text",
  "inactive-text",
  "confirm-button-text",
  "cancel-button-text",
  "aria-label"
]);
for (const file of files) {
  const { descriptor } = parse(fs.readFileSync(file, "utf8"));
  function record(text, line) {
    if (file.includes("LanguageSwitch") && text.trim() === "简体中文") return;
    if (/\p{Script=Han}/u.test(text))
      missing.push({
        file: path.relative(path.join(__dirname, ".."), file),
        line,
        text: text.trim(),
        cataloged: Object.hasOwn(catalog, text.trim())
      });
  }
  function visit(node) {
    if (node.type === 2) record(node.content, node.loc.start.line);
    if (node.type === 1)
      for (const prop of node.props)
        if (prop.type === 6 && attributes.has(prop.name) && prop.value)
          record(prop.value.content, prop.loc.start.line);
    node.children?.forEach(visit);
  }
  if (descriptor.template) visit(descriptor.template.ast);
}
if (process.argv.includes("--json"))
  console.log(JSON.stringify(missing, null, 2));
else
  console.log(
    JSON.stringify(
      {
        catalogEntries: Object.keys(catalog).length,
        vueFiles: files.length,
        remainingStaticOccurrences: missing.length,
        remainingFiles: new Set(missing.map(item => item.file)).size,
        note: "User content, logs and backend messages retain their source language. Use --json for static UI locations."
      },
      null,
      2
    )
  );

if (process.argv.includes("--strict") && missing.length) process.exitCode = 1;
