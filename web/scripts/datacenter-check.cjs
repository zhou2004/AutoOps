// Browser smoke test; all API calls are intercepted. Never writes live inventory.
// PLAYWRIGHT_MODULE=/path/to/playwright CHROMIUM_PATH=/path/to/chrome node scripts/datacenter-check.cjs
const assert = require("node:assert/strict");
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || "playwright");
const baseURL = process.env.BASE_URL || "http://127.0.0.1:8080";
const rooms = [
  {
    id: 11,
    name: "已定位测试机房",
    city: "北京",
    country: "中国",
    longitude: 116.4,
    latitude: 39.9,
    status: 1
  },
  { id: 12, name: "待定位测试机房", longitude: null, latitude: null, status: 1 }
];
const rack = {
  id: 31,
  idcId: 11,
  name: "R01",
  unitNum: 42,
  usedUnit: 2,
  widthMm: 600,
  depthMm: 1000,
  layoutX: 0,
  layoutZ: 0,
  rotation: 0,
  unpositioned: 0,
  conflictUnits: [],
  invalidPlacements: 0,
  devices: [
    {
      id: 44,
      cabinetId: 31,
      kind: "physical",
      name: "测试服务器",
      sn: "TEST-44",
      unitPosition: 4,
      unitHeight: 2,
      status: 1
    }
  ],
  occupiedUnits: [4, 5]
};
(async () => {
  const browser = await chromium.launch({
    executablePath: process.env.CHROMIUM_PATH,
    args: [
      "--no-sandbox",
      "--enable-unsafe-swiftshader",
      "--use-angle=swiftshader"
    ]
  });
  try {
    const context = await browser.newContext({
      viewport: { width: 1600, height: 1000 }
    });
    await context.addInitScript(() => {
      localStorage.setItem(
        "authorized-token",
        JSON.stringify("intercepted-test-token")
      );
      localStorage.setItem(
        "user-info",
        JSON.stringify({
          username: "preview",
          roles: ["admin"],
          permissions: ["*:*:*"]
        })
      );
      document.cookie = "multiple-tabs=true; path=/";
    });
    const page = await context.newPage(),
      errors = [],
      writes = [];
    let inventory = rooms;
    page.on("pageerror", e => errors.push(e.message));
    await context.route("**/api/v1/**", route => {
      const request = route.request(),
        path = new URL(request.url()).pathname;
      if (request.method() !== "GET") writes.push(path);
      let data = {};
      if (path.endsWith("/system/permissions/me"))
        data = {
          userId: 1,
          username: "preview",
          isAdmin: true,
          permissions: ["cmdb:read", "cmdb:write"]
        };
      else if (path.endsWith("/menu/vo/list")) data = [];
      else if (/\/cmdb\/idc(\/all)?$/.test(path))
        data = { list: inventory, total: inventory.length };
      else if (path.endsWith("/cmdb/cabinet"))
        data = { list: [rack], total: 1 };
      else if (path.endsWith("/topology"))
        data = {
          room: rooms.find(r => path.includes(`/${r.id}/`)) || rooms[0],
          cabinets: [rack],
          unplaced: []
        };
      return route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 200, data })
      });
    });
    await page.goto(baseURL + "/cmdb/datacenter");
    await page.locator(".idc-item").first().waitFor();
    assert.equal(await page.locator(".idc-item").count(), 2);
    assert.equal(
      await page.locator(".globe-pin").count(),
      1,
      "unlocated room must not get a fake coordinate"
    );
    assert.equal(
      await page
        .getByRole("menuitem", { name: "机房管理", exact: true })
        .count(),
      1
    );
    assert.equal(
      await page.getByRole("menuitem", { name: /机房3D|机房与机柜/ }).count(),
      0
    );
    await page.getByRole("button", { name: "暂停自转", exact: true }).click();
    await page
      .getByRole("button", { name: "进入 已定位测试机房", exact: true })
      .click();
    await page.locator(".rack-choice").first().waitFor();
    await page.locator(".rack-choice").first().click();
    await page.getByRole("button", { name: "关闭柜门", exact: true }).waitFor();
    await page.getByRole("button", { name: "关闭柜门", exact: true }).click();
    await page.getByRole("button", { name: "打开柜门", exact: true }).click();
    await page.locator(".u-device").filter({ hasText: "测试服务器" }).click();
    await page.locator(".facility-device-drawer").waitFor();
    assert.ok(
      await page.locator(".rack-scene canvas").evaluate(canvas => {
        const rect = canvas.getBoundingClientRect();
        return (
          document.elementFromPoint(
            rect.x + rect.width * 0.45,
            rect.y + rect.height * 0.65
          ) === canvas
        );
      }),
      "non-modal device detail must allow interaction with the model"
    );

    assert.match(
      await page.locator(".facility-device-drawer").innerText(),
      /4–5 U/
    );
    await page.getByRole("button", { name: "收回设备", exact: true }).waitFor();
    await page.keyboard.press("Escape");
    await page.getByRole("button", { name: "返回机房", exact: true }).click();
    await page.getByRole("button", { name: "气流示意", exact: true }).click();
    assert.match(
      await page.locator(".scene-bottom").innerText(),
      /未接入传感器/
    );
    await page
      .getByRole("button", { name: "机房与机柜台账", exact: true })
      .click();
    await page
      .getByRole("button", { name: "3D 视图", exact: true })
      .first()
      .click();
    await page.locator(".rack-choice").first().waitFor();
    assert.match(
      page.url(),
      /idcId=11/,
      "same-component ledger drilldown must switch tabs"
    );
    await page.goto(baseURL + "/system/machine");
    await page.locator(".inventory-content").waitFor();
    assert.match(page.url(), /cmdb\/datacenter\?tab=inventory/);
    await page.getByRole("button", { name: "可视化总览", exact: true }).click();
    await page
      .locator(".idc-item")
      .filter({ hasText: "待定位测试机房" })
      .click();
    await page.locator(".rack-choice").first().waitFor();
    assert.match(page.url(), /idcId=12/);
    inventory = [];
    await page.goto(baseURL + "/cmdb/datacenter");
    await page.locator(".directory-empty").waitFor();
    assert.equal(await page.locator(".globe-pin").count(), 0);
    await page
      .getByRole("button", { name: "体验示例场景", exact: true })
      .click();
    assert.equal(await page.locator(".idc-item").count(), 5);
    await page.locator(".idc-item").first().click();
    await page.locator(".rack-choice").first().waitFor();
    assert.equal(await page.locator(".rack-choice").count(), 10);
    assert.equal(
      await page.getByRole("button", { name: "布局", exact: true }).count(),
      0
    );
    await page.locator(".rack-choice").first().click();
    await page
      .locator(".u-device")
      .filter({ hasText: "汇聚交换机-1-1" })
      .click();
    await page.locator(".facility-device-drawer").waitFor();
    assert.match(
      await page.locator(".facility-device-drawer").innerText(),
      /示例设备/
    );
    assert.equal(
      await page.getByRole("button", { name: "前往资产管理" }).count(),
      0
    );
    await page.keyboard.press("Escape");
    await page.getByRole("button", { name: "返回世界地图" }).click();
    async function assertFits(selector) {
      await page.waitForTimeout(450);
      const d = await page.locator(selector).evaluate(el => {
        const box = el.getBoundingClientRect(),
          wrap = el.closest(".el-scrollbar__wrap");
        return {
          top: box.top,
          bottom: box.bottom,
          height: box.height,
          viewport: innerHeight,
          overflow: document.documentElement.scrollWidth > innerWidth + 1,
          scroll: wrap ? wrap.scrollHeight - wrap.clientHeight : 0
        };
      });
      assert.ok(
        d.top >= 0 && d.bottom <= d.viewport && d.height > 180 && !d.overflow,
        JSON.stringify(d)
      );
      assert.ok(
        d.scroll <= 2,
        "visualization forces outer-page scrolling: " + JSON.stringify(d)
      );
    }
    for (const size of [
      { width: 1366, height: 768 },
      { width: 1280, height: 600 },
      { width: 1920, height: 1080 },
      { width: 390, height: 844 }
    ]) {
      await page.setViewportSize(size);
      await assertFits(".idc-globe");
    }
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.getByRole("button", { name: "IDC 目录", exact: true }).click();
    await page.locator(".idc-item").first().click();
    await page.locator(".rack-scene").waitFor();
    await assertFits(".rack-scene");
    await page.getByRole("button", { name: "机柜目录", exact: true }).click();
    await page.locator(".rack-choice").first().click();
    await page.getByRole("button", { name: "U 位面板", exact: true }).click();
    await page
      .locator(".u-device")
      .filter({ hasText: "汇聚交换机-1-1" })
      .click();
    await page.locator(".facility-device-drawer").waitFor();
    await page.keyboard.press("Escape");
    await page.getByRole("button", { name: "U 位面板", exact: true }).click();
    for (const size of [
      { width: 1280, height: 600 },
      { width: 1366, height: 768 },
      { width: 1920, height: 1080 }
    ]) {
      await page.setViewportSize(size);
      await assertFits(".rack-scene");
    }
    assert.equal(
      await page.locator(".facility-stat-grid:visible").count(),
      0,
      "statistics must start collapsed"
    );
    const before = await page.locator(".rack-scene").boundingBox();
    await page.getByRole("button", { name: "统计详情", exact: true }).click();
    await page
      .getByRole("dialog", { name: "机房统计详情", exact: true })
      .waitFor();
    assert.match(
      await page
        .getByRole("dialog", { name: "机房统计详情", exact: true })
        .innerText(),
      /240/
    );
    assert.deepEqual(
      await page.locator(".rack-scene").boundingBox(),
      before,
      "statistics dialog changed scene size"
    );
    await page.keyboard.press("Escape");
    const lightColor = await page
      .locator(".rack-list")
      .evaluate(el => getComputedStyle(el).backgroundColor);
    await page.evaluate(() => {
      document.documentElement.classList.add("dark");
      document.documentElement.dataset.theme = "dark";
    });
    await page.waitForTimeout(400);
    assert.equal(
      await page.locator(".facility-shell").getAttribute("data-facility-theme"),
      "dark"
    );
    assert.notEqual(
      await page
        .locator(".rack-list")
        .evaluate(el => getComputedStyle(el).backgroundColor),
      lightColor
    );
    await page
      .locator(".u-device")
      .filter({ hasText: "汇聚交换机-1-1" })
      .click();
    assert.equal(
      await page
        .locator(".facility-device-drawer")
        .getAttribute("data-facility-theme"),
      "dark"
    );
    await page.keyboard.press("Escape");
    await page.evaluate(() => {
      document.documentElement.classList.remove("dark");
      document.documentElement.dataset.theme = "light";
    });
    await page.waitForTimeout(400);
    assert.equal(
      await page
        .locator(".rack-list")
        .evaluate(el => getComputedStyle(el).backgroundColor),
      lightColor
    );
    assert.equal(
      await page
        .locator(".facility-actions")
        .getByRole("button", { name: /全屏|亮色|暗色/ })
        .count(),
      0
    );
    await page.locator(".language-switch:visible").first().click();
    await page.getByRole("menuitem", { name: "English", exact: true }).click();
    await page
      .getByRole("button", { name: "Room overview", exact: true })
      .waitFor();
    await page.getByRole("button", { name: "Statistics", exact: true }).click();
    await page
      .getByRole("dialog", { name: "Room statistics", exact: true })
      .waitFor();
    assert.match(
      await page.locator(".facility-stat-grid:visible").innerText(),
      /Occupied rack units/
    );
    await page.keyboard.press("Escape");
    await assertFits(".rack-scene");
    await page
      .getByRole("button", { name: "← Back to world map", exact: true })
      .click();
    await page
      .getByRole("button", { name: "Pause rotation", exact: true })
      .waitFor();
    await page
      .getByRole("button", { name: "Rooms & racks", exact: true })
      .click();

    await page
      .getByRole("button", { name: "+ Add data center", exact: true })
      .click();
    await page
      .getByRole("dialog", { name: "Add data center", exact: true })
      .waitFor();
    assert.match(
      await page
        .getByRole("dialog", { name: "Add data center", exact: true })
        .innerText(),
      /Latitude/
    );
    await page.getByRole("button", { name: "Cancel", exact: true }).click();
    await page.locator(".language-switch:visible").first().click();
    await page.getByRole("menuitem", { name: "简体中文", exact: true }).click();
    // Simulate missing WebGL. The directory and U lists must stay usable.
    await page.addInitScript(() => {
      const original = HTMLCanvasElement.prototype.getContext;
      HTMLCanvasElement.prototype.getContext = function (type, ...args) {
        return /webgl/.test(type) ? null : original.call(this, type, ...args);
      };
    });
    inventory = rooms;
    await page.goto(baseURL + "/cmdb/datacenter");
    await page.locator(".globe-failure").waitFor();
    await page.locator(".idc-item").first().click();
    await page.locator(".scene-failure").waitFor();
    assert.equal(await page.locator(".rack-choice").count(), 1);
    await page.locator(".u-device").filter({ hasText: "测试服务器" }).click();
    await page.locator(".facility-device-drawer").waitFor();
    await page.goto(baseURL + "/cmdb/datacenter?tab=inventory");
    await page.locator(".inventory-content").waitFor();
    assert.deepEqual(errors, [], "uncaught browser errors");
    assert.deepEqual(writes, [], "visualization unexpectedly wrote inventory");
    console.log(
      "PASS: unified navigation, globe drilldown, rack doors, device inspection, real/demo separation, ledger and legacy routes, viewport fit, responsive panels, light/dark themes, collapsed statistics, removed page controls, live language switching, mobile/reduced motion, WebGL fallback; no writes or uncaught errors."
    );
  } finally {
    await browser.close();
  }
})().catch(error => {
  console.error(error);
  process.exitCode = 1;
});
