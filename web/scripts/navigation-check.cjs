// Run: node scripts/navigation-check.cjs. Uses the actual route/access definitions.
const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const Module = require("node:module");
const ts = require("typescript");
const { createRouter, createMemoryHistory } = require("vue-router");
function load(relative, mocks = {}) {
  const filename = path.join(__dirname, "../src", relative);
  const loaded = new Module(filename, module);
  loaded.filename = filename;
  loaded.paths = Module._nodeModulePaths(path.dirname(filename));
  loaded.require = name =>
    Object.hasOwn(mocks, name) ? mocks[name] : module.require(name);
  loaded._compile(
    ts.transpileModule(fs.readFileSync(filename, "utf8"), {
      compilerOptions: {
        module: ts.ModuleKind.CommonJS,
        target: ts.ScriptTarget.ES2022
      }
    }).outputText,
    filename
  );
  return loaded.exports;
}
const routes = ["cmdb", "database", "k8s", "system"].map(
  name => load(`router/modules/${name}.ts`).default
);
const access = load("utils/platformAccess.ts", {
  "@/utils/auth": { getToken: () => "fixture" },
  "@/api/platformPermissions": {
    myPermissions: async () => ({ isAdmin: false, permissions: [] })
  }
});
function identity(permissions, admin = false) {
  Object.assign(access.platformAccess, {
    ready: true,
    isAdmin: admin,
    permissions
  });
}

test("database operators get independent source management without CMDB permissions", () => {
  identity(["database:read"]);
  const menus = access.filterPlatformMenus(routes);
  assert.equal(menus.some(r => r.path === "/cmdb"), false);
  const database = menus.find(r => r.path === "/database");
  assert.deepEqual(database.children.map(r => r.path), ["/database/sources"]);
  assert.equal(database.children[0].meta.showParent, true);
  assert.equal(access.canVisitPlatform("/database/workbench"), true);
  identity(["cmdb:read"]);
  assert.equal(access.canVisitPlatform("/database/sources"), false);
});

test("Docker-only operator sees Docker under containers without Kubernetes or asset administration", () => {
  identity(["docker:read"]);
  assert.equal(access.canVisitPlatform("/k8s/docker"), true);
  assert.equal(access.canVisitPlatform("/k8s/list"), false);
  assert.equal(
    access.canVisitPlatform("/k8s/docker/terminal/host/1/app"),
    false
  );
  assert.equal(access.canVisitPlatform("/system/authorization"), false);
  const menus = access.filterPlatformMenus(routes);
  assert.equal(
    menus.some(r => r.path === "/cmdb"),
    false
  );
  const containers = menus.find(r => r.path === "/k8s");
  assert.deepEqual(
    containers.children.map(r => r.path),
    ["/k8s/docker"]
  );
  assert.equal(containers.redirect, "/k8s/docker");
  identity(["docker:read", "docker:execute"]);
  assert.equal(
    access.canVisitPlatform("/k8s/docker/terminal/host/1/app"),
    true
  );
  assert.equal(access.canVisitPlatform("/k8s/terminal/1/default/app"), false);
});

test("administrators get one authorization entry and hidden redirects stay out of menus", () => {
  identity([], true);
  const paths = access
    .filterPlatformMenus(routes)
    .flatMap(r => r.children.map(c => c.path));
  assert.equal(paths.filter(p => p === "/system/authorization").length, 1);
  for (const old of [
    "/cmdb/docker",
    "/cmdb/asset-permission",
    "/system/k8s-permission",
    "/system/asset-permission",
    "/cmdb/credential-permission",
    "/cmdb/user-group",
    "/system/role-permission"
  ]) {
    assert.equal(paths.includes(old), false, old);
  }
});

test("old bookmarks redirect to the correct tab and preserve Docker terminal parameters", async () => {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: routes
      .flatMap(r => r.children)
      .map(r => ({
        ...r,
        component: r.redirect ? undefined : { render: () => null }
      }))
  });
  await router.push("/cmdb/db?groupId=18");
  assert.equal(router.currentRoute.value.path, "/database/sources");
  assert.equal(router.currentRoute.value.query.groupId, "18");
  await router.push("/cmdb/dbdetails?id=10");
  assert.equal(router.currentRoute.value.path, "/database/workbench");
  assert.equal(router.currentRoute.value.query.id, "10");
  for (const [old, tab] of [
    ["/cmdb/asset-permission", "assets"],
    ["/system/asset-permission", "assets"],
    ["/system/k8s-permission", "containers"],
    ["/cmdb/user-group", "groups"],
    ["/cmdb/credential-permission", "assets"]
  ]) {
    await router.push(old);
    assert.equal(router.currentRoute.value.path, "/system/authorization");
    assert.equal(router.currentRoute.value.query.tab, tab);
  }
  await router.push("/system/role-permission?roleId=2");
  assert.equal(router.currentRoute.value.path, "/system/api-keys");
  assert.equal(router.currentRoute.value.query.roleId, undefined);
  await router.push("/cmdb/docker?asset=host:1");
  assert.equal(router.currentRoute.value.path, "/k8s/docker");
  assert.equal(router.currentRoute.value.query.asset, "host:1");
  await router.push("/cmdb/docker/terminal/host/1/web?connectionId=31");
  assert.equal(
    router.currentRoute.value.path,
    "/k8s/docker/terminal/host/1/web"
  );
  assert.equal(router.currentRoute.value.query.connectionId, "31");
});
