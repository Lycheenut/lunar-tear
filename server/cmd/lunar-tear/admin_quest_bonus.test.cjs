const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");
const vm = require("node:vm");
const context = vm.createContext({ window: {} });
vm.runInContext(readFileSync(path.join(__dirname, "admin_quest_bonus.js"), "utf8"), context);
const Draft = context.window.QuestBonusDraft;
const plain = value => JSON.parse(JSON.stringify(value));
function catalog() {
  const table = (name, fields, rows) => ({ name, fields: fields.map(name => ({ name })), rows: rows.map(values => ({ values: Object.fromEntries(fields.map((field, i) => [field, String(values[i])])) })) });
  return {
    tables: [
      table("m_quest_bonus", ["QuestBonusId", "QuestBonusCharacterGroupId", "QuestBonusCostumeGroupId", "QuestBonusWeaponGroupId", "QuestBonusCostumeSettingGroupId", "QuestBonusAllyCharacterId"], [[10, 7, 8, 50, 20, 9], [11, 7, 8, 51, 20, 9], [12, 7, 8, 52, 20, 9], [30, 0, 0, 60, 21, 0]]),
      table("m_quest_bonus_costume_setting_group", ["QuestBonusCostumeSettingGroupId", "CostumeId", "LimitBreakCountLowerLimit", "QuestBonusEffectGroupId", "QuestBonusTermGroupId"], [[20, 31029, 0, 30, 40], [21, 31029, 0, 30, 0], [21, 35031, 0, 30, 0]]),
      table("m_quest_bonus_weapon_group", ["QuestBonusWeaponGroupId", "WeaponId", "LimitBreakCountLowerLimit", "QuestBonusEffectGroupId", "QuestBonusTermGroupId"], [
        [50,101,0,1,40],[50,102,0,1,40],[50,201,0,1,40],[50,301,0,1,40],
        [51,101,0,2,40],[51,102,0,2,40],[51,201,0,1,40],[51,301,0,2,40],
        [52,101,0,3,40],[52,102,0,3,40],[52,201,0,1,40],[52,301,0,3,40],
        [60,101,0,9,0],[60,102,0,9,0],[60,401,0,9,0],[60,402,0,9,0]
      ]),
      table("m_quest_bonus_effect_group", ["QuestBonusEffectGroupId", "SortOrder", "QuestBonusType", "QuestBonusEffectId"], [[1,1,3,1],[2,1,3,1],[2,2,3,2],[3,1,3,1],[3,2,3,2],[3,3,3,3],[9,1,3,9]]),
      table("m_quest_bonus_drop_reward", ["QuestBonusEffectId", "PossessionType", "PossessionId", "AdditionalCount"], [[1,6,181,10],[2,6,182,10],[3,6,183,10],[9,6,9,5]])
    ],
    quests: [
      {questId:1,row:0,chapterId:501,difficulty:1,bonusId:10,medalIds:[181]},
      {questId:2,row:1,chapterId:501,difficulty:4,bonusId:11,medalIds:[181,182]},
      {questId:3,row:2,chapterId:501,difficulty:4,bonusId:12,medalIds:[181,182,183]},
      {questId:4,row:3,chapterId:589,difficulty:1,bonusId:0,medalIds:[249]},
      {questId:5,row:4,chapterId:589,difficulty:4,bonusId:0,medalIds:[249,250]},
      {questId:6,row:5,chapterId:589,difficulty:4,bonusId:0,medalIds:[249,250,251]}
    ], chapters: [], medals: [],
    costumes: [{id:31029,titles:{ja:"兵器の祭典"}},{id:35031,titles:{ja:"少女の祭典"}}],
    weapons: [101,102,201,301,401,402].map(id=>({id,evolutionGroupId:Math.floor(id/100),evolutionOrder:id%100,titles:{ja:id<400?"同名の武器":"神秘石の杖"}}))
  };
}

test("historical roster imports every member and auto-matches only actual weapon families", () => {
  const data=catalog(), snapshot=JSON.stringify(data), draft=new Draft(data);
  draft.replace(501,30);
  const selected=draft.selections.get("501");
  assert.equal(selected.choices["武器:1"],"101");
  assert.equal(selected.choices["武器:4"],undefined);
  assert.throws(()=>draft.payload(),/尚未选择/);
  selected.choices["武器:4"]="201";
  assert.deepEqual(plain(draft.payload()),{changes:[],questBonusGroups:[],questBonusRestores:[{chapterId:501,sourceBonusId:30,ruleChapterId:501,weapons:[{weaponId:101,templateWeaponId:101},{weaponId:401,templateWeaponId:201}]}]});
  assert.equal(draft.count(),1);
  assert.equal(draft.targetGroups(501).length,3);
  assert.equal(JSON.stringify(data),snapshot);
  assert.equal(draft.members([30]).size,4);
  assert.deepEqual(plain(draft.members([30]).get("武器:4").itemIDs),["401","402"]);
  assert.throws(()=>draft.replace(501,999),/已有加成/);
  assert.equal(draft.selections.get("501"),selected);
  draft.replace(501,""); assert.equal(draft.count(),0);
});

test("profiles distinguish silver/gold phases, evolution coverage, and equal-name families", () => {
  const draft=new Draft(catalog()), profiles=draft.profiles(501);
  assert.equal(profiles.length,3,"same effects with different evolution coverage must stay distinct");
  assert.deepEqual(plain(profiles.find(p=>p.id==="201").medalIDs),[181]);
  assert.deepEqual(plain(profiles.find(p=>p.id==="101").medalIDs),[181,182,183]);
  assert.equal(draft.members([10]).size,4,"same title must not merge distinct weapon families");
  assert.equal(draft.templateRows(12,402,201)[0].WeaponId,"201","later forms inherit nearest available earlier template form");
  assert.deepEqual(plain(draft.rewards(9)),[{possessionType:6,possessionId:9,count:5}]);
});

test("zero-bonus activity requires explicit reference phase and currency mappings", () => {
  const draft=new Draft(catalog()); draft.replace(589,30);
  assert.throws(()=>draft.payload(),/缺少现有加成/);
  draft.setReference(589,501);
  const selected=draft.selections.get("589"); selected.choices["武器:4"]="201";
  assert.throws(()=>draft.payload(),/关卡分组/);
  draft.targetGroups(589).forEach((group,i)=>selected.groups[group.key]=String(10+i));
  assert.throws(()=>draft.payload(),/奖章对应/);
  selected.currencies={181:"249",182:"250",183:"251"};
  const input=draft.payload().questBonusRestores[0];
  assert.deepEqual(plain(input.groups),[{questIds:[4],ruleBonusId:10},{questIds:[5],ruleBonusId:11},{questIds:[6],ruleBonusId:12}]);
  assert.deepEqual(plain(input.currencies),[{fromId:181,toId:249},{fromId:182,toId:250},{fromId:183,toId:251}]);
});

test("shared quest activities are rejected before changing the draft", () => {
  const data=catalog(); data.quests.push({...data.quests[0],chapterId:999}); const draft=new Draft(data);
  assert.throws(()=>draft.replace(501,30),/共用关卡/); assert.equal(draft.count(),0);
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
