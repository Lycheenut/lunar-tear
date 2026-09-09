const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");
const vm = require("node:vm");

const source = readFileSync(path.join(__dirname, "admin.js"), "utf8");
const catalogKeys = ["costumes", "weapons", "companions", "parts", "materials", "consumableItems",
  "enhancedCostumes", "enhancedWeapons", "enhancedCompanions", "enhancedParts", "paidGems", "freeGems",
  "importantItems", "thoughts", "missionPassPoints", "premiumItems"];

function createEditor() {
  const element = () => ({
    value: "", textContent: "", children: [], dataset: {}, listeners: {},
    get options() { return this.children; },
    classList: { add() {}, remove() {}, toggle() {} },
    append(...children) { this.children.push(...children); },
    replaceChildren(...children) { this.children = children; },
    addEventListener(event, listener) { this.listeners[event] = listener; },
    setAttribute() {}, focus() {}
  });
  const context = vm.createContext({
    document: { querySelector: element, querySelectorAll: () => [], createElement: element },
    window: { addEventListener() {}, setTimeout() {}, clearTimeout() {} },
    sessionStorage: { getItem: () => null }, localStorage: { getItem: () => null }
  });
  vm.runInContext(source.replace(/\}\)\(\);\s*$/, `
    // Leave the reward editing and serialization code intact; omit unrelated panel refreshes.
    renderTable = () => {};
    updateDirtyUI = () => {};
    refreshMissionRewardAssignmentDisplays = () => {};
    globalThis.editor = { state, rewardFieldPair, renderRewardTypeFieldEditor,
      rewardReferencesForPossessionType, rewardDefinitionForPossessionType,
      rewardSelectorOptions, missionRewardReplacementPayload, renderShopDraftPossessionField };
  })();`), context);
  const editor = context.editor;
  editor.state.catalog = { defaultLanguage: "en", tables: [] };
  editor.state.language = "en";
  editor.state.rewardCatalog = Object.fromEntries(catalogKeys.map((key, index) => [key, [{
    possessionType: index + 1, possessionId: [11, 12].includes(index + 1) ? 0 : 9000 + index,
    names: { en: `Reward ${key}`, ja: `日本語 ${key}` }
  }]]));
  return editor;
}

function rewardTable(name = "m_mission_reward", prefix = "") {
  return {
    name, fields: [
      { name: `${prefix}PossessionType`, type: "PossessionType", kind: "int32" },
      { name: `${prefix}PossessionId`, kind: "int32" }
    ],
    rows: [{ index: 0, values: {
      MissionRewardId: "1", [`${prefix}PossessionType`]: "5", [`${prefix}PossessionId`]: "123", Count: "3"
    } }]
  };
}

for (const [index, key] of catalogKeys.entries()) {
  test(`delivery editor selects, searches and serializes ${key}`, () => {
    const editor = createEditor();
    const table = rewardTable(), row = table.rows[0], possessionType = index + 1;
    const reference = editor.state.rewardCatalog[key][0];
    const pair = editor.rewardFieldPair(table, "PossessionType");
    const select = editor.renderRewardTypeFieldEditor(table, row, pair).children[0];
    assert.ok(select.options.some(option => option.value === String(possessionType)));
    select.value = String(possessionType);
    select.listeners.change();
    const payload = editor.missionRewardReplacementPayload(table)[0];
    assert.equal(payload.possessionType, possessionType);
    assert.equal(payload.possessionId, reference.possessionId);
    assert.equal(payload.count, 3);
    const references = editor.rewardReferencesForPossessionType(possessionType);
    const definition = editor.rewardDefinitionForPossessionType(possessionType);
    const option = editor.rewardSelectorOptions(references, definition)[0];
    assert.equal(option.value, String(reference.possessionId));
    assert.match(option.label, new RegExp(key));
    assert.match(option.searchText, /日本語/);
  });
}

test("login stamps use prefixed reward fields and do not offer mission-only points", () => {
  const editor = createEditor(), table = rewardTable("m_login_bonus_stamp", "Reward"), row = table.rows[0];
  const pair = editor.rewardFieldPair(table, "RewardPossessionId");
  const select = editor.renderRewardTypeFieldEditor(table, row, pair).children[0];
  assert.equal(select.options.some(option => option.value === "15"), false);
  select.value = "3";
  select.listeners.change();
  const changes = Array.from(editor.state.dirty.values());
  assert.ok(changes.some(change => change.field === "RewardPossessionType" && change.value === "3"));
  assert.ok(changes.some(change => change.field === "RewardPossessionId" && change.value === "9002"));
});

test("shop drafts select Companion and omit mission-only points", () => {
  const editor = createEditor(), possession = { possessionType: 5, possessionId: 123, count: 2 };
  const select = editor.renderShopDraftPossessionField({}, possession, 0, "PossessionType").children[0];
  assert.equal(select.options.some(option => option.value === "15"), false);
  select.value = "3";
  select.listeners.change();
  assert.deepEqual(possession, { possessionType: 3, possessionId: 9002, count: 2 });
});

test("empty catalogs do not create selectable rewards or discard unknown existing IDs", () => {
  const editor = createEditor(), table = rewardTable(), row = table.rows[0];
  editor.state.rewardCatalog.enhancedWeapons = [];
  row.values.PossessionType = "8";
  const pair = editor.rewardFieldPair(table, "PossessionType");
  const select = editor.renderRewardTypeFieldEditor(table, row, pair).children[0];
  assert.equal(select.options.filter(option => option.value === "8").length, 1);
  assert.match(select.options.find(option => option.value === "8").textContent, /强化武器/);
  assert.equal(editor.missionRewardReplacementPayload(table)[0].possessionId, 123);
  assert.equal(editor.state.dirty.size, 0);
  row.values.PossessionType = "5";
  assert.equal(editor.renderRewardTypeFieldEditor(table, row, pair).children[0].options.some(option => option.value === "8"), false);
});
