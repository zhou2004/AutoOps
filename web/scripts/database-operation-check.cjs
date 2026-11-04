const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const Module = require("node:module");
const ts = require("typescript");
const { parse, compileScript } = require("vue/compiler-sfc");
const { reactive, effectScope, nextTick, ref } = require("vue");

function load(source, code, mocks) {
  const compiled = ts.transpileModule(code, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022,
      esModuleInterop: true
    }
  });
  const loaded = new Module(source, module);
  loaded.filename = source;
  loaded.paths = Module._nodeModulePaths(path.dirname(source));
  loaded.require = name =>
    mocks[name] ??
    (name === "@/locales"
      ? {
          tr: (text, params = {}) =>
            String(text ?? "").replace(/\{(\w+)\}/g, (placeholder, key) =>
              Object.hasOwn(params, key) ? String(params[key]) : placeholder
            ),
          locale: ref("zh-CN")
        }
      : Module.prototype.require.call(loaded, name));
  loaded._compile(compiled.outputText, source);
  return loaded.exports;
}

const planResponse = (id, extra = {}) => ({
  id,
  revision: 1,
  action: "refresh",
  executable: true,
  expiresAt: "2030-01-01T00:00:00Z",
  checks: [],
  ...extra
});
const flush = () => new Promise(resolve => setImmediate(resolve));
function dialog(t, request, confirm = async () => "confirm") {
  const source = path.join(__dirname, "../src/api/databasePlatform.ts");
  const api = load(source, fs.readFileSync(source, "utf8"), {
    "@/utils/request": async options => ({
      data: { code: 200, data: await request(options) }
    })
  });
  const component = path.join(
    __dirname,
    "../src/views/database/components/OperationDialog.vue"
  );
  const backupSource = path.join(__dirname, "../src/utils/databaseBackup.ts");
  const backupHelpers = load(
    backupSource,
    fs.readFileSync(backupSource, "utf8"),
    {}
  );
  const { descriptor } = parse(fs.readFileSync(component, "utf8"), {
    filename: component
  });
  const script = compileScript(descriptor, { id: "operation-regression" });
  const definition = load(component, script.content, {
    "@/api/databasePlatform": api,
    "@/utils/databaseBackup": backupHelpers,
    "@/api/cmdb": {},
    "element-plus": { ElMessageBox: { confirm } }
  }).default;
  const props = reactive({
    modelValue: false,
    action: "refresh",
    label: "刷新状态",
    cluster: {
      id: 4,
      name: "test",
      engine: "mysql",
      topology: "mysql-gtid",
      members: []
    }
  });
  const emitted = [],
    scope = effectScope();
  const vm = scope.run(() =>
    definition.setup(props, {
      expose() {},
      emit: (...args) => emitted.push(args)
    })
  );
  t.after(() => scope.stop());
  return {
    vm,
    props,
    emitted,
    api,
    async open() {
      props.modelValue = true;
      await nextTick();
      await flush();
    }
  };
}

test("HTTP browser without crypto can prepare and submit a server-issued key", async t => {
  const crypto = Object.getOwnPropertyDescriptor(globalThis, "crypto");
  Object.defineProperty(globalThis, "crypto", {
    configurable: true,
    value: undefined
  });
  t.after(() =>
    crypto
      ? Object.defineProperty(globalThis, "crypto", crypto)
      : delete globalThis.crypto
  );
  const submitted = [];
  const d = dialog(t, async options => {
    if (options.url.endsWith("operation-plans"))
      return planResponse(42, { idempotencyKey: "db-plan-42" });
    submitted.push(options.data);
    return { id: 100 };
  });
  await d.open();
  assert.equal(d.vm.error.value, "");
  assert.equal(d.vm.key.value, "db-plan-42");
  await d.vm.submit();
  assert.deepEqual(submitted, [
    { planId: 42, idempotencyKey: "db-plan-42", confirmed: true }
  ]);
  assert.deepEqual(d.emitted, [["submitted", 100]]);
});

test("older API fallback and response-loss retry preserve exactly the same key", async t => {
  const submitted = [];
  let planID = 1;
  const d = dialog(t, async options => {
    if (options.url.endsWith("operation-plans")) return planResponse(planID);
    submitted.push(options.data);
    if (submitted.length === 1)
      throw new Error("response lost after acceptance");
    return { id: 100 };
  });
  await d.open();
  await d.vm.submit();
  assert.match(d.vm.error.value, /response lost/);
  await d.vm.submit();
  assert.deepEqual(submitted[0], submitted[1]);
  assert.equal(submitted[0].idempotencyKey, "db-plan-1");
  planID = 2;
  await d.vm.preflight();
  assert.equal(d.vm.key.value, "db-plan-2");
  assert.notEqual(d.vm.key.value, submitted[0].idempotencyKey);
});

test("failed or invalid preparation cannot expose a submittable plan with an empty/stale key", async t => {
  let fail = false,
    submitted = 0;
  const d = dialog(t, async options => {
    if (options.url.endsWith("operation-plans"))
      return planResponse(42, { idempotencyKey: fail ? "" : "db-plan-42" });
    submitted++;
    return { id: 100 };
  });
  await d.open();
  fail = true;
  await d.vm.preflight();
  assert.equal(d.vm.plan.value, undefined);
  assert.equal(d.vm.key.value, "");
  await d.vm.submit();
  assert.equal(submitted, 0);
  for (const id of [0, -1, undefined, NaN, 1.2, Number.MAX_SAFE_INTEGER + 1])
    assert.throws(() => d.api.operationPlanKey(planResponse(id)));
});

test("a late preflight response from a closed dialog cannot replace a new plan", async t => {
  let resolveOld,
    requests = 0;
  const d = dialog(t, async () => {
    requests++;
    if (requests === 1)
      return new Promise(resolve => {
        resolveOld = resolve;
      });
    return planResponse(52);
  });
  await d.open();
  d.props.modelValue = false;
  await nextTick();
  await d.open();
  resolveOld(planResponse(51));
  await flush();
  assert.equal(d.vm.plan.value.id, 52);
  assert.equal(d.vm.key.value, "db-plan-52");
});

test("confirmation blocks submission; cancel and close preserve the prepared plan", async t => {
  let submitted = 0,
    confirmation,
    confirmCalls = 0;
  const d = dialog(
    t,
    async options => {
      if (options.url.endsWith("operation-plans")) return planResponse(42);
      submitted++;
      return { id: 100 };
    },
    (message, title, options) => {
      confirmCalls++;
      assert.match(message, /test.*刷新状态/);
      assert.equal(title, "执行确认");
      assert.equal(options.confirmButtonText, "确认执行");
      return new Promise((resolve, reject) => {
        confirmation = { resolve, reject };
      });
    }
  );
  await d.open();
  for (const reason of ["cancel", "close"]) {
    const pending = d.vm.submit();
    assert.equal(d.vm.busy.value, true);
    await d.vm.submit();
    assert.equal(submitted, 0);
    confirmation.reject(reason);
    await pending;
    assert.equal(d.vm.error.value, "");
    assert.equal(d.vm.busy.value, false);
    assert.equal(d.vm.plan.value.id, 42);
    assert.equal(d.vm.key.value, "db-plan-42");
    assert.deepEqual(d.emitted, []);
  }
  assert.equal(confirmCalls, 2, "duplicate click opened another confirmation");
  const accepted = d.vm.submit();
  assert.equal(submitted, 0);
  confirmation.resolve("confirm");
  await accepted;
  assert.equal(submitted, 1);
  assert.deepEqual(d.emitted, [["submitted", 100]]);
});

test("accepting a stale confirmation cannot submit a newly opened operation", async t => {
  let accept,
    requests = 0,
    submitted = 0;
  const d = dialog(
    t,
    async options => {
      if (options.url.endsWith("operation-plans"))
        return planResponse(++requests);
      submitted++;
      return { id: 100 };
    },
    () =>
      new Promise(resolve => {
        accept = resolve;
      })
  );
  await d.open();
  const pending = d.vm.submit();
  d.props.modelValue = false;
  await nextTick();
  await d.open();
  accept("confirm");
  await pending;
  assert.equal(submitted, 0);
  assert.equal(d.vm.plan.value.id, 2);
  assert.equal(d.vm.busy.value, false);
});

test("recovery time selection requires a complete overlapping archive window", () => {
  const source = path.join(__dirname, "../src/utils/databaseBackup.ts");
  const { recoveryWindow } = load(source, fs.readFileSync(source, "utf8"), {});
  const artifact = (start, end) => ({
    engine: "mysql",
    method: "binlog-archive",
    archiveCheck: true,
    backupStop: start,
    archiveThrough: new Date(end * 1000).toISOString()
  });
  const backup = {
    status: "complete",
    manifest: {
      members: [{ id: 1 }, { id: 2 }],
      artifacts: { a: artifact(100, 300), b: artifact(200, 400) }
    }
  };
  assert.deepEqual(recoveryWindow(backup), { start: 200000, end: 300000 });
  backup.manifest.artifacts.b = artifact(301, 400);
  assert.equal(
    recoveryWindow(backup),
    undefined,
    "disjoint shard windows were combined"
  );
  backup.manifest.artifacts.b = artifact(200, 400);
  backup.manifest.artifacts.b.method = "logical";
  assert.equal(
    recoveryWindow(backup),
    undefined,
    "logical backup exposed PITR"
  );
  delete backup.manifest.artifacts.b;
  assert.equal(recoveryWindow(backup), undefined, "missing shard was ignored");
  backup.status = "failed";
  backup.manifest.members = [{ id: 1 }];
  assert.equal(recoveryWindow(backup), undefined, "failed backup exposed PITR");
});
