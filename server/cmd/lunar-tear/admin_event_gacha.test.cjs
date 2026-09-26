const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");
const vm = require("node:vm");

const source = readFileSync(path.join(__dirname, "admin.js"), "utf8");

function createEditor() {
  const nodes = new Map();
  function node() {
    const classes = new Set();
    return {
      value: "", textContent: "", children: [], listeners: {},
      classList: {
        add(name) { classes.add(name); }, remove(name) { classes.delete(name); },
        contains(name) { return classes.has(name); },
        toggle(name, enabled) { if (enabled) classes.add(name); else classes.delete(name); }
      },
      addEventListener(event, fn) { this.listeners[event] = fn; },
      append(child) { this.children.push(child); if (!this.value) this.value = child.value; },
      replaceChildren() { this.children = []; this.value = ""; },
      close() {}, focus() {}
    };
  }
  const element = (id) => {
    if (!nodes.has(id)) nodes.set(id, node());
    return nodes.get(id);
  };
  const requests = [];
  const context = vm.createContext({
    document: { querySelector: element, querySelectorAll: () => [], createElement: node },
    window: { addEventListener() {}, setTimeout() {}, clearTimeout() {} },
    sessionStorage: { getItem: () => null }, localStorage: { getItem: () => null },
    confirm: () => true,
    fetch: async (url, options) => {
      requests.push({ url, options });
      return { ok: true, json: async () => ({}) };
    }
  });
  // Keep real selection, box editing, validation and publish handlers. Stub
  // unrelated reward-row rendering and post-publish network reload.
  vm.runInContext(source.replace(/\}\)\(\);\s*$/, `
    createSearchableSelect = () => {};
    renderBoxRewardRow = () => ({});
    refreshBoxProbabilityPreviews = () => {};
    updateGachaDirtyUI = () => {};
    reloadPublishedGachaConfig = async () => {};
    globalThis.editor = { state, renderBoxGachaEditor, boxBannersForCurrentKind,
      addConfiguredBox, removeConfiguredBox, appendBoxValidationErrors };
  })();`), context);
  const editor = context.editor;
  const box = (id) => ({
    groupWeights: { limited: 10000, unlimited: 0 },
    limitedRewards: [{ possessionType: 5, possessionId: id, count: 1, maxCount: 1, jackpot: true }],
    unlimitedRewards: []
  });
  Object.assign(editor.state, {
    language: "en", gachaKind: "event", boxSelections: {},
    gachaCatalog: { contentHash: "hash", defaultLanguage: "en", boxBanners: [
      ...["bronze", "silver", "gold"].map((tier, i) => ({
        gachaId: 329001 + i * 10, gachaLabelType: 2, eventGachaBaseId: 329001,
        eventGachaTicketTier: tier, requiredConsumableItemId: 6055 + i,
        titles: { en: `Aurora Dynast ${tier}` }, ticketNames: { en: `${tier} ticket` }
      })),
      { gachaId: 332001, gachaLabelType: 2, requiredConsumableItemId: 6065 },
      { gachaId: 200001, gachaLabelType: 3 }
    ] },
    gachaDraft: { chapterBanners: {}, eventBanners: {
      329001: { boxes: [box(100001), box(100001)] },
      329011: { boxes: [box(100002)] }
    } },
    rewardCatalog: { materials: [100001, 100002].map((id) => ({ possessionType: 5, possessionId: id })) }
  });
  return { ...editor, element, requests };
}

test("tier filter distinguishes pools and switching retains each pool's box selection", () => {
  const editor = createEditor();
  editor.renderBoxGachaEditor();
  assert.equal(editor.element("#box-gacha-banner-select").children.length, 4);
  assert.match(editor.element("#box-gacha-banner-select").children[1].textContent, /银票池.*6056/);
  editor.element("#box-gacha-number-select").value = "2";
  editor.element("#box-gacha-number-select").listeners.change();
  editor.element("#box-gacha-tier-select").value = "silver";
  editor.element("#box-gacha-tier-select").listeners.change();
  assert.equal(editor.element("#box-gacha-banner-select").value, "329011");
  assert.equal(editor.element("#box-gacha-number-select").value, "1");
  assert.match(editor.element("#box-gacha-ticket-info").textContent, /6056.*silver ticket/);
  editor.element("#box-gacha-tier-select").value = "bronze";
  editor.renderBoxGachaEditor();
  assert.equal(editor.element("#box-gacha-number-select").value, "2");
  editor.element("#box-gacha-tier-select").value = "single";
  assert.equal(editor.boxBannersForCurrentKind()[0].gachaId, 332001);
  editor.state.gachaKind = "chapter";
  assert.equal(editor.boxBannersForCurrentKind()[0].gachaId, 200001);
});

test("creating, deleting and publishing a higher tier preserves other tiers", async () => {
  const editor = createEditor();
  const original = JSON.stringify(editor.state.gachaDraft.eventBanners);
  editor.element("#box-gacha-tier-select").value = "gold";
  editor.renderBoxGachaEditor();
  assert.equal(editor.element("#box-gacha-editor-body").classList.contains("hidden"), true);
  editor.addConfiguredBox();
  assert.equal(editor.state.gachaDraft.eventBanners[329021].boxes.length, 1);
  const errors = [];
  editor.appendBoxValidationErrors(errors);
  assert.deepEqual(errors, []);
  await editor.element("#gacha-publish-confirm").listeners.click();
  const request = editor.requests.find(({ options }) => options.method === "POST");
  assert.equal(request.url, "/api/admin/gacha-config");
  const payload = JSON.parse(request.options.body);
  assert.equal(payload.expectedContentHash, "hash");
  assert.deepEqual(Object.keys(payload.config.eventBanners), ["329001", "329011", "329021"]);
  assert.equal(payload.config.eventBanners[329001].boxes.length, 2);
  editor.removeConfiguredBox();
  assert.equal(JSON.stringify(editor.state.gachaDraft.eventBanners), original);
});

test("validation checks invalid gold rewards even while viewing bronze", () => {
  const editor = createEditor();
  editor.state.gachaDraft.eventBanners[329021] = { boxes: [{
    groupWeights: { limited: 10000, unlimited: 0 },
    limitedRewards: [{ possessionType: 5, possessionId: 999999, count: 1, maxCount: 1 }]
  }] };
  editor.element("#box-gacha-tier-select").value = "bronze";
  const errors = [];
  editor.appendBoxValidationErrors(errors);
  assert.ok(errors.some((message) => message.includes("329021") && message.includes("不在主数据")));
  assert.ok(errors.some((message) => message.includes("329021") && message.includes("大奖")));
});
