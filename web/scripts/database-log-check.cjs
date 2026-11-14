const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const Module = require("node:module");
const ts = require("typescript");
const filename = require("node:path").resolve(
  __dirname,
  "../src/api/databaseLogStream.ts"
);
let removed = 0;
const source = ts.transpileModule(fs.readFileSync(filename, "utf8"), {
  compilerOptions: {
    module: ts.ModuleKind.CommonJS,
    target: ts.ScriptTarget.ES2022
  }
}).outputText;
const loaded = new Module(filename, module);
loaded.require = name =>
  ({
    "@/locales": { tr: text => text },
    "./platformPermissions": { platformApiBase: "https://test.invalid/api/v1" },
    "@/utils/auth": {
      getToken: () => "test-token",
      removeToken: () => removed++
    }
  })[name];
loaded._compile(source, filename);
const { databaseLogStream, appendLogText } = loaded.exports;

test("SSE handles split UTF-8, CRLF, LF, comments and authorized requests", async () => {
  const old = global.fetch;
  let options;
  global.fetch = async (url, request) => {
    options = request;
    assert.match(url, /database\/clusters\/7\/logs\/stream/);
    assert.ok(!url.includes("test-token"));
    const bytes = new TextEncoder().encode(
      ': hello\r\n\r\nevent: ready\r\ndata: {"path":"mysql.log"}\r\n\r\nevent: log\ndata: {"text":"中文🙂"}\n\nevent: end\ndata: {}\n\n'
    );
    return new Response(
      new ReadableStream({
        start(c) {
          for (let i = 0; i < bytes.length; i += 2)
            c.enqueue(bytes.slice(i, i + 2));
          c.close();
        }
      }),
      { headers: { "content-type": "text/event-stream" } }
    );
  };
  try {
    const result = [];
    const controller = new AbortController();
    await databaseLogStream(7, "memberId=2", controller.signal, e =>
      result.push(e)
    );
    assert.deepEqual(
      result.map(e => e.event),
      ["ready", "log", "end"]
    );
    assert.equal(result[1].data.text, "中文🙂");
    assert.equal(options.headers.Authorization, "Bearer test-token");
    assert.equal(options.signal, controller.signal);
  } finally {
    global.fetch = old;
  }
});
test("API errors are surfaced and expired auth is removed", async () => {
  const old = global.fetch;
  global.fetch = async () =>
    new Response(JSON.stringify({ code: 401, message: "expired" }), {
      status: 401,
      headers: { "content-type": "application/json" }
    });
  try {
    await assert.rejects(
      databaseLogStream(1, "", new AbortController().signal, () => {}),
      /expired/
    );
    assert.ok(removed > 0);
  } finally {
    global.fetch = old;
  }
});
test("log buffer bounds lines and UTF-8 bytes", () => {
  let value = appendLogText("", "中文🙂\n".repeat(20000));
  assert.ok(value.split("\n").length <= 10001);
  assert.ok(Buffer.byteLength(value) <= 2 * 1024 * 1024);
  value = appendLogText("", "中文🙂".repeat(500000));
  assert.ok(Buffer.byteLength(value) <= 2 * 1024 * 1024);
  assert.ok(!value.includes("\ufffd"));
});
test("stopping the reader cancels its response body", async () => {
  const old = global.fetch;
  let cancelled = false;
  global.fetch = async () =>
    new Response(
      new ReadableStream({
        start(c) {
          c.enqueue(
            new TextEncoder().encode('event: log\ndata: {"text":"x"}\n\n')
          );
        },
        cancel() {
          cancelled = true;
        }
      }),
      { headers: { "content-type": "text/event-stream" } }
    );
  try {
    await assert.rejects(
      databaseLogStream(1, "", new AbortController().signal, () => {
        throw new Error("stop");
      }),
      /stop/
    );
    assert.equal(cancelled, true);
  } finally {
    global.fetch = old;
  }
});
test("unexpected stream EOF is reported instead of looking successful", async () => {
  const old = global.fetch;
  global.fetch = async () =>
    new Response('event: log\ndata: {"text":"partial"}\n\n', {
      headers: { "content-type": "text/event-stream" }
    });
  try {
    await assert.rejects(
      databaseLogStream(1, "", new AbortController().signal, () => {}),
      /日志连接已中断/
    );
  } finally {
    global.fetch = old;
  }
});
