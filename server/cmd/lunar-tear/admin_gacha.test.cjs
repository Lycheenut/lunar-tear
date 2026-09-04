const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");
const vm = require("node:vm");

const source = readFileSync(path.join(__dirname, "admin.js"), "utf8");

function createEditor(reward = { possessionType: 5, possessionId: 100001 }) {
  const nodes = new Map();
  const element = (selector) => {
    if (!nodes.has(selector)) nodes.set(selector, {
      value: "", textContent: "", open: false, listeners: {},
      classList: { add() {}, remove() {}, toggle() {} },
      addEventListener(event, listener) { this.listeners[event] = listener; },
      focus() {},
      showModal() { this.open = true; }
    });
    return nodes.get(selector);
  };
  const box = {
    groupWeights: { limited: 10000, unlimited: 0 },
    limitedRewards: [{ ...reward, count: 1, maxCount: 1 }],
    unlimitedRewards: []
  };
  const catalog = {
    weapons: [], banners: [], medals: [], masterDataHash: "master-hash",
    config: {
      version: 1,
      banners: { 100001: { bannerAssetName: "limited_100001", startDatetime: 1000, endDatetime: 2000 } },
      chapterBanners: { 200000: box }
    }
  };
  const rewards = {
    materials: [{ possessionType: 5, possessionId: 100001 }],
    freeGems: [{ possessionType: 12, possessionId: 0 }]
  };
  const requests = [];
  const context = vm.createContext({
    document: { querySelector: element, querySelectorAll: () => [] },
    window: { addEventListener() {}, setTimeout() {}, clearTimeout() {} },
    sessionStorage: { getItem: () => null },
    localStorage: { getItem: () => null },
    fetch: async (url) => {
      requests.push(url);
      assert.ok(["/api/admin/gacha-config", "/api/admin/reward-reference"].includes(url), url);
      return { ok: true, json: async () => structuredClone(url.endsWith("gacha-config") ? catalog : rewards) };
    }
  });
  // Expose the real application state/functions without changing the served asset.
  vm.runInContext(source.replace(/\}\)\(\);\s*$/, `
    globalThis.editor = { state, loadSelectedTable, storeFieldChange };
  })();`), context);
  const editor = context.editor;
  editor.state.catalog = { tables: [{ name: "gacha", primary: true }] };
  element("#table-select").value = "gacha";
  return { ...editor, element, requests, rewards };
}

async function submitSchedule(editor) {
  const table = editor.state.catalog.tables[0];
  editor.storeFieldChange(table, table.rows[0], { name: "EndDatetime" }, "3000");
  await editor.element("#save").listeners.click();
}

for (const cached of [false, true]) {
  test(`operational Gacha schedule accepts existing box rewards with ${cached ? "cached" : "unloaded"} references`, async () => {
    const editor = createEditor();
    if (cached) editor.state.rewardCatalog = editor.rewards;
    await editor.loadSelectedTable();
    await submitSchedule(editor);
    assert.equal(editor.element("#gacha-publish-dialog").open, true, editor.element("#notice").textContent);
    assert.equal(editor.requests.filter((url) => url.endsWith("reward-reference")).length, cached ? 0 : 1);
    assert.equal(editor.state.gachaDraft.chapterBanners[200000].limitedRewards[0].possessionId, 100001);
  });
}

for (const reward of [{ possessionType: 12 }, { possessionType: 12, possessionId: 0 }]) {
  test(`operational Gacha schedule accepts free gems with ${"possessionId" in reward ? "zero" : "omitted"} ID`, async () => {
    const editor = createEditor(reward);
    editor.state.rewardCatalog = editor.rewards;
    await editor.loadSelectedTable();
    await submitSchedule(editor);
    assert.equal(editor.element("#gacha-publish-dialog").open, true, editor.element("#notice").textContent);
  });
}

test("operational Gacha schedule still rejects a missing reward", async () => {
  const editor = createEditor({ possessionType: 5, possessionId: 999999 });
  editor.state.rewardCatalog = editor.rewards;
  await editor.loadSelectedTable();
  await submitSchedule(editor);
  assert.equal(editor.element("#gacha-publish-dialog").open, false);
  assert.match(editor.element("#notice").textContent, /卡池 200000 第 1 箱的有限奖励 1 不在主数据奖励列表中/);
});

test("reloading published Gacha data reloads reward references before another schedule submission", async () => {
  const editor = createEditor();
  editor.state.rewardCatalog = editor.rewards;
  await editor.loadSelectedTable();
  editor.state.gachaCatalog = null;
  editor.state.gachaDraft = null;
  editor.state.rewardCatalog = null;
  await editor.loadSelectedTable();
  await submitSchedule(editor);
  assert.equal(editor.element("#gacha-publish-dialog").open, true, editor.element("#notice").textContent);
});
