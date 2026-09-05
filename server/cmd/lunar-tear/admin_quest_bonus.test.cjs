const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");
const vm = require("node:vm");

const context = vm.createContext({ window: {} });
vm.runInContext(readFileSync(path.join(__dirname, "admin_quest_bonus.js"), "utf8"), context);
const Draft = context.window.QuestBonusDraft;
const bonus = "m_quest_bonus", costume = "m_quest_bonus_costume_setting_group", term = "m_quest_bonus_term_group";
const plain = (value) => JSON.parse(JSON.stringify(value));

function catalog() {
  const table = (name, fields, rows) => ({ name, fields: fields.map((name) => ({ name })), rows: rows.map((values) => ({ values: Object.fromEntries(fields.map((field, i) => [field, String(values[i])])) })) });
  return {
    tables: [
      table(bonus, ["QuestBonusId", "QuestBonusCostumeSettingGroupId"], [[10, 20], [11, 20]]),
      table(costume, ["QuestBonusCostumeSettingGroupId", "CostumeId", "LimitBreakCountLowerLimit", "QuestBonusEffectGroupId", "QuestBonusTermGroupId"], [[20, 31029, 0, 30, 40], [20, 35031, 0, 30, 0]]),
      table(term, ["QuestBonusTermGroupId", "SortOrder", "StartDatetime", "EndDatetime"], [[40, 1, 100, 200]])
    ],
    quests: [
      { questId: 1, row: 0, chapterId: 573, difficulty: 1, bonusId: 10 },
      { questId: 2, row: 1, chapterId: 573, difficulty: 2, bonusId: 11 },
      { questId: 3, row: 2, chapterId: 589, difficulty: 1, bonusId: 0 }
    ], chapters: [], costumes: [], weapons: []
  };
}

test("shared terms include both difficulties, while the zero-bonus rerun stays separate", () => {
  const draft = new Draft(catalog());
  assert.deepEqual(plain(draft.affected(term, 40).map((quest) => quest.questId)), [1, 2]);
  assert.equal(draft.termStatus(40, 201), "已过期／停用");
  assert.equal(draft.termStatus(40, 100), "生效中");
  assert.equal(draft.termStatus(40, 200), "生效中");
  assert.equal(draft.termStatus(40, 99), "未开始");
  assert.equal(draft.termStatus(0), "不限期");
  assert.equal(draft.termStatus(999), "期限引用缺失");
  assert.equal(draft.rows(costume, 20).length, 2);
  draft.setGroup(costume, 20, [...draft.rows(costume, 20)].reverse());
  assert.equal(draft.count(), 0, "reordering unchanged members must not create an unsavable draft");
});

test("copy and reassignment submit only the new group and selected quest", () => {
  const data = catalog(), draft = new Draft(data);
  draft.setGroup(costume, 21, draft.rows(costume, 20).map((row) => ["21", ...row.slice(1)]));
  draft.setGroup(bonus, 12, [["12", "21"]]);
  draft.assign(data.quests[2], "12");
  const payload = plain(draft.payload());
  assert.deepEqual(payload.changes, [{ table: "m_quest", row: 2, field: "QuestBonusId", value: "12" }]);
  assert.equal(payload.questBonusGroups.length, 2);
  assert.equal(draft.rows(costume, 20)[0][0], "20");
  assert.deepEqual(plain(draft.affected(costume, 21).map((quest) => quest.questId)), [3]);
  assert.equal(draft.affected(term, 40).length, 3);
  draft.assign(data.quests[2], "0");
  draft.setGroup(bonus, 12, []);
  draft.setGroup(costume, 21, []);
  assert.equal(draft.count(), 0);
});

test("deleted groups stay empty in the draft and read-only legacy users contribute to impacts", () => {
  const data = catalog();
  data.readOnlyLinks = { "m_quest_bonus_character_group:90": ["m_quest_bonus_term_group:40"] };
  data.tables[0].fields.push({ name: "QuestBonusCharacterGroupId" });
  data.tables[0].rows.forEach((row) => { row.values.QuestBonusCharacterGroupId = "90"; });
  const draft = new Draft(data);
  draft.setGroup(costume, 20, []);
  assert.equal(draft.rows(costume, 20).length, 0);
  assert.equal(draft.affected(term, 40).length, 2);
  assert.equal(plain(draft.payload()).questBonusGroups[0].rows.length, 0);
});

test("discard works when the other specialized catalogs have not been loaded", () => {
  const nodes = new Map();
  const element = (selector) => {
    if (!nodes.has(selector)) nodes.set(selector, {
      value: "", textContent: "", listeners: {},
      classList: { add() {}, remove() {}, toggle() {} },
      addEventListener(event, listener) { this.listeners[event] = listener; },
      focus() {}
    });
    return nodes.get(selector);
  };
  const sandbox = vm.createContext({
    document: { querySelector: element, querySelectorAll: () => [] },
    window: { addEventListener() {}, setTimeout() {}, clearTimeout() {} },
    sessionStorage: { getItem: () => null }, localStorage: { getItem: () => null }, confirm: () => true
  });
  vm.runInContext(readFileSync(path.join(__dirname, "admin_quest_bonus.js"), "utf8"), sandbox);
  const main = readFileSync(path.join(__dirname, "admin.js"), "utf8");
  vm.runInContext(main.replace(/\}\)\(\);\s*$/, "globalThis.integration = { state }; })();"), sandbox);
  sandbox.integration.state.catalog = {
    tables: [], shopEditor: { shops: null, cellGroups: null },
    questDropEditor: { quests: null, groups: null, rewards: null }
  };
  element("#discard").listeners.click();
  assert.equal(element("#save").disabled, true);
  assert.match(element("#notice").textContent, /已放弃/);
});
