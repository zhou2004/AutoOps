const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const Module = require("node:module");
const ts = require("typescript");

function load(relative, dependencies = {}, base = "") {
  const filename = path.join(__dirname, relative);
  const source = fs
    .readFileSync(filename, "utf8")
    .replaceAll("import.meta.env.VITE_API_BASE_URL", JSON.stringify(base));
  const compiled = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    }
  });
  const instance = new Module(filename, module);
  instance.filename = filename;
  instance.paths = Module._nodeModulePaths(path.dirname(filename));
  instance.require = name =>
    dependencies[name] || Module.prototype.require.call(instance, name);
  instance._compile(compiled.outputText, filename);
  return instance.exports;
}
const urls = load("../src/api/platformUrl.ts");

test("DNS discovery and diagnostics use stored accounts without sending credentials", async () => {
  const calls = [];
  const controller = new AbortController();
  const api = load("../src/api/webservice.ts", {
    "./platformPermissions": {
      platformRequest: async (url, method, body, timeout, signal) => {
        calls.push({ url, method, body, signal });
        return { valid: true, writeAccess: "unverified" };
      }
    }
  });
  await api.webDNSCredentials("cloudflare", controller.signal);
  await api.webDNSZones(7, controller.signal);
  const check = await api.webCheckDNSAccount(7, "*.example.com", controller.signal);
  assert.deepEqual(calls.map(({ url, method, body }) => ({ url, method, body })), [
    { url: "/web/dns-credentials?provider=cloudflare", method: "GET", body: undefined },
    { url: "/web/dns-accounts/7/zones", method: "GET", body: undefined },
    { url: "/web/dns-accounts/7/check", method: "POST", body: { domain: "*.example.com" } }
  ]);
  assert.ok(calls.every(call => call.signal === controller.signal));
  assert.equal(check.writeAccess, "unverified");
});

test("HTTP and WebSocket share a single versioned API prefix", () => {
  for (const [configured, expected] of [
    ["", "/api/v1"],
    ["/api/v1/", "/api/v1"],
    ["https://ops.test/", "https://ops.test/api/v1"],
    ["https://ops.test/gateway/api/v1/", "https://ops.test/gateway/api/v1"],
    ["/gateway", "/gateway/api/v1"]
  ])
    assert.equal(urls.apiV1Base(configured), expected);
});

test("Web requests preserve query parameters and options paginate", async () => {
  const calls = [];
  const api = load("../src/api/webservice.ts", {
    "./platformPermissions": {
      platformRequest: async (url, method, body, timeout) => {
        calls.push({ url, method, body, timeout });
        return { list: [{ id: calls.length }], total: 2 };
      }
    }
  });
  const selected = await api.webOptions("sources");
  assert.deepEqual(
    selected.map(r => r.id),
    [1, 2]
  );
  assert.equal(calls[0].url, "/web/sources?page=1&size=100&view=options");
  assert.equal(calls[1].url, "/web/sources?page=2&size=100&view=options");
  await api.webList("audit", {
    groupId: 18,
    clusterId: 0,
    name: "发布 & TLS",
    status: "unknown",
    environment: "test",
    page: 2,
    size: 20,
    empty: ""
  });
  const params = new URL("https://ops.test" + calls[2].url).searchParams;
  assert.equal(params.get("name"), "发布 & TLS");
  assert.equal(params.get("status"), "unknown");
  assert.equal(params.get("groupId"), "18");
  assert.equal(params.has("clusterId"), false);
  assert.equal(params.get("page"), "2");
});

test("log query and download keep identical time boundaries and offsets", () => {
  const api = load("../src/api/webservice.ts", { "./platformPermissions": {} });
  const range = [
    new Date("2026-11-09T08:00:00Z"),
    new Date("2026-11-09T09:00:00Z")
  ];
  const query = api.webLogQuery({
    logId: "registered",
    range,
    rotation: ".2.gz",
    status: "503",
    offset: 500,
    lines: 500
  });
  const download = new URLSearchParams(query + "&download=json");
  assert.equal(download.get("from"), range[0].toISOString());
  assert.equal(download.get("to"), range[1].toISOString());
  assert.equal(download.get("offset"), "500");
  assert.equal(download.get("rotation"), ".2.gz");
  assert.equal(download.has("range"), false);
});

test("standard-envelope 401 clears the session and never yields page data", async () => {
  let removed = 0;
  const api = load(
    "../src/api/platformPermissions.ts",
    {
      "./platformUrl": urls,
      "@/utils/auth": {
        getToken: () => "test-token",
        removeToken: () => removed++
      }
    },
    "/api/v1"
  );
  const original = global.fetch;
  try {
    global.fetch = async (url, init) => {
      assert.equal(url, "/api/v1/web/sources");
      assert.equal(init.headers.Authorization, "Bearer test-token");
      return new Response(JSON.stringify({ code: 401, message: "expired" }), {
        status: 200
      });
    };
    await assert.rejects(api.platformRequest("/web/sources"), /expired/);
    assert.equal(removed, 1);
  } finally {
    global.fetch = original;
  }
});

test("changing a target propagates cancellation to the HTTP request", async () => {
  const api = load("../src/api/platformPermissions.ts", {
    "./platformUrl": urls,
    "@/utils/auth": { getToken: () => "test", removeToken() {} }
  });
  const original = global.fetch;
  const abort = new AbortController();
  let signal;
  try {
    global.fetch = async (_, init) => {
      signal = init.signal;
      return new Response(JSON.stringify({ code: 200, data: [] }));
    };
    await api.platformRequest(
      "/web/instances/1/logs",
      "GET",
      undefined,
      10000,
      abort.signal
    );
    abort.abort();
    assert.equal(signal.aborted, true);
  } finally {
    global.fetch = original;
  }
});

test("native validation preserves raw files, deletion and the selected node baselines", async () => {
  const calls = [];
  const api = load("../src/api/webservice.ts", {
    "./platformPermissions": {
      platformRequest: async (url, method, body, timeout, signal) => {
        calls.push({ url, method, body, timeout, signal });
        return [
          { instanceId: 52, valid: false, message: 'unknown directive "bad"' }
        ];
      }
    }
  });
  const files = [
    {
      path: "/etc/nginx/conf.d/site.conf",
      content: '# keep comments\nserver { set $label "生产站点"; }\n',
      mode: 420
    },
    { path: "/etc/nginx/conf.d/removed.conf", content: "", deleted: true }
  ];
  const scope = {
    instanceIds: [52],
    baselines: { 52: "sha256-before-edit" },
    variables: { 52: { UPSTREAM: "127.0.0.1:8081" } }
  };
  const result = await api.webValidateClusterConfig(7, files, scope);
  assert.deepEqual(calls[0], {
    url: "/web/clusters/7/config/validate",
    method: "POST",
    body: { files, ...scope },
    timeout: 180000,
    signal: undefined
  });
  assert.equal(result[0].valid, false);
  assert.equal(result[0].message, 'unknown directive "bad"');
  await api.webValidateClusterConfig(7, files);
  assert.deepEqual(calls[1].body, { files });
  const abort = new AbortController();
  await api.webInstanceConfig(52, abort.signal);
  assert.equal(calls[2].url, "/web/instances/52/config");
  assert.equal(calls[2].signal, abort.signal);
});

test("certificate exports select immutable versions and never request private keys by default", async () => {
  const calls = [];
  const api = load("../src/api/webservice.ts", {
    "./platformPermissions": {
      platformRequest: async (url, method, body) => {
        calls.push({ url, method, body });
        return { certificate: "public-chain", versionId: 18 };
      }
    }
  });
  const exported = await api.webExportCertificate(4, 18);
  assert.equal(exported.certificate, "public-chain");
  assert.deepEqual(calls[0], {
    url: "/web/certificates/4/versions/18/export?key=false",
    method: "GET",
    body: undefined
  });
  await api.webExportCertificate(4, 17, true);
  assert.equal(calls[1].url, "/web/certificates/4/versions/17/export?key=true");
  await api.webExportCertificate(4, 0);
  assert.equal(calls[2].url, "/web/certificates/4/versions/0/export?key=false");
});

test("certificate import sends secret material only in the authenticated JSON body", async () => {
  const platform = load("../src/api/platformPermissions.ts", {
    "./platformUrl": urls,
    "@/utils/auth": { getToken: () => "test", removeToken() {} }
  });
  const api = load("../src/api/webservice.ts", {
    "./platformPermissions": platform
  });
  const original = global.fetch;
  const body = {
    name: "Imported certificate",
    groupId: 18,
    environment: "test",
    certificate: "test certificate chain",
    privateKey: "test private material"
  };
  try {
    global.fetch = async (url, init) => {
      assert.equal(url, "/api/v1/web/certificates/import");
      assert.equal(init.method, "POST");
      assert.equal(init.headers.Authorization, "Bearer test");
      assert.deepEqual(JSON.parse(init.body), body);
      return new Response(
        JSON.stringify({
          code: 200,
          data: { certificate: { id: 4 }, version: { id: 18 } }
        })
      );
    };
    assert.deepEqual(
      await api.webRequest("/certificates/import", "POST", body),
      {
        certificate: { id: 4 },
        version: { id: 18 }
      }
    );
  } finally {
    global.fetch = original;
  }
});


test("SSE log stream parses split UTF-8/CRLF events, uses one authenticated request and closes reader", async () => {
 const oldFetch = global.fetch; let calls=0, cancelled=0; const events=[];
 const bytes=new TextEncoder().encode('id: 1\r\nevent: log\r\ndata: {"text":"中文日志\\n"}\r\n\r\nevent: end\ndata: {"message":"done"}\n\n');
 global.fetch=async (url, init) => {
  calls++;assert.ok(url.includes('/web/instances/3/logs/stream?logId=abc'));assert.equal(init.headers.Authorization,'Bearer session-token');assert.ok(!url.includes('token='));
  let offset=0;return new Response(new ReadableStream({pull(controller){if(offset<bytes.length){controller.enqueue(bytes.slice(offset,++offset));}else controller.close();},cancel(){cancelled++;}}),{headers:{'Content-Type':'text/event-stream'}});
 };
 const api=load('../src/api/webLogStream.ts',{'@/locales':{tr:x=>x},'./platformPermissions':{platformApiBase:'/api/v1'},'@/utils/auth':{getToken:()=> 'session-token',removeToken(){}}});
 try {await api.webLogStream(3,'logId=abc',new AbortController().signal,e=>events.push(e));assert.equal(calls,1);assert.equal(events[0].data.text,'中文日志\n');assert.equal(events[1].event,'end');}
 finally{global.fetch=oldFetch;}
});
test("SSE envelope 401 invalidates authentication without reconnecting", async () => {
 const oldFetch=global.fetch;let removed=0,calls=0;
 global.fetch=async()=>{calls++;return new Response(JSON.stringify({code:401,message:'expired'}),{headers:{'Content-Type':'application/json'}})};
 const api=load('../src/api/webLogStream.ts',{'@/locales':{tr:x=>x},'./platformPermissions':{platformApiBase:'/api/v1'},'@/utils/auth':{getToken:()=> 'token',removeToken(){removed++}}});
 try{await assert.rejects(api.webLogStream(1,'logId=a',new AbortController().signal,()=>{}),/expired/);assert.equal(removed,1);assert.equal(calls,1);}finally{global.fetch=oldFetch;}
});

test("live log buffer bounds UTF-8 bytes without splitting CJK or emoji", () => {
  const { appendLogText } = load("../src/api/webLogStream.ts", {
    "./platformPermissions": {}, "@/locales": { tr: s => s }, "@/utils/auth": {}
  });
  const next = appendLogText("日志🙂".repeat(300000), "\n最新日志🙂\n");
  assert.ok(Buffer.byteLength(next, "utf8") <= 2 * 1024 * 1024);
  assert.ok(next.endsWith("\n最新日志🙂\n"));
  assert.ok(!next.includes("\ufffd"));
  const rows = appendLogText("old\n".repeat(15000), "new\n");
  assert.equal(rows.split("\n").filter(Boolean).length, 10000);
  assert.ok(rows.endsWith("new\n"));
  assert.equal(appendLogText("old\n".repeat(15000), "partial").split("\n").length, 10000);
});

test("cluster create/edit submits details and explicit membership deltas in one request", async () => {
  const calls = [];
  const api = load("../src/api/webservice.ts", { "./platformPermissions": {
    platformRequest: async (url, method, body) => { calls.push({url,method,body}); return {id:12,revision:3}; }
  }});
  await api.webSaveCluster({name:"new cluster",groupId:10,spec:{entries:[]}}, {add:[7,8],remove:[]});
  await api.webSaveCluster({id:12,name:"renamed",revision:3,groupId:10}, {add:[9],remove:[7]});
  assert.equal(calls.length,2);
  assert.equal(calls[0].url,"/web/clusters"); assert.equal(calls[0].method,"POST");
  assert.deepEqual(calls[0].body.memberChanges,{add:[7,8],remove:[]});
  assert.equal(calls[1].url,"/web/clusters/12"); assert.equal(calls[1].method,"PUT");
  assert.equal(calls[1].body.revision,3);
  assert.deepEqual(calls[1].body.memberChanges,{add:[9],remove:[7]});
  assert.equal("instanceIds" in calls[1].body,false);
});
