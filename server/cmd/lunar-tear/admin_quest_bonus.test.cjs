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
  assert.deepEqual(plain(draft.payload()),{changes:[],questBonusGroups:[],questBonusRestores:[{chapterId:501,sourceBonusId:30,mode:"replace",costumeIds:[31029,35031],ruleChapterId:501,weapons:[{weaponId:101,templateWeaponId:101},{weaponId:401,templateWeaponId:201}]}]});
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

test("supplement keeps the current roster and submits only checked additions", () => {
  const data=catalog(), snapshot=JSON.stringify(data), draft=new Draft(data);
  draft.setMode(501,"append"); draft.replace(501,30);
  const members=draft.members([30]), selected=draft.selections.get("501");
  assert.equal(draft.count(),0); assert.equal(draft.payload().questBonusRestores.length,0);
  draft.setMember(501,members.get("服装:31029"),true);
  draft.setMember(501,members.get("武器:1"),true);
  assert.equal(draft.selectedMembers(501).length,0,"existing IDs/families cannot be added again");
  draft.setMember(501,members.get("服装:35031"),true);
  assert.equal(draft.count(),1);
  assert.deepEqual(plain(draft.payload().questBonusRestores[0].weapons),[],"unchecked weapons need no rules");
  draft.setMember(501,members.get("武器:4"),true);
  assert.throws(()=>draft.payload(),/尚未选择武器规则/);
  selected.choices["武器:4"]="101";
  assert.deepEqual(plain(draft.payload().questBonusRestores[0]),{chapterId:501,sourceBonusId:30,mode:"append",costumeIds:[35031],ruleChapterId:501,weapons:[{weaponId:401,templateWeaponId:101}]});
  assert.deepEqual(plain(draft.summary(501)),{final:[2,4],added:[1,1],removed:[0,0]});
  draft.setMember(501,members.get("武器:4"),false);
  draft.setReference(501,589);
  assert.equal(draft.payload().questBonusRestores[0].groups,undefined,"costume-only edits need no external mapping");
  assert.equal(JSON.stringify(data),snapshot);
  draft.replace(501,10);
  assert.equal(draft.selectedMembers(501).length,0,"changing source clears selected additions");
  assert.deepEqual(plain(draft.selections.get("501").choices),{});
  draft.setMode(501,"replace");
  assert.equal(draft.selectedMembers(501).length,4);
  draft.setMember(501,draft.members([10]).get("武器:1"),false);
  draft.setMode(501,"replace");
  assert.equal(draft.selectedMembers(501).length,3,"clicking the active mode preserves the draft");
});

test("partial replacement and empty lists are explicit, including costume-only zero-bonus activities", () => {
  const draft=new Draft(catalog()); draft.replace(589,30);
  const selected=draft.selections.get("589"); selected.members.clear();
  draft.setMember(589,draft.members([30]).get("服装:35031"),true);
  assert.deepEqual(plain(draft.payload().questBonusRestores[0]),{chapterId:589,sourceBonusId:30,mode:"replace",costumeIds:[35031],ruleChapterId:589,weapons:[]});
  selected.members.clear();
  assert.deepEqual(plain(draft.payload().questBonusRestores[0].costumeIds),[]);
  draft.replace(501,30); draft.selections.get("501").members.clear();
  assert.deepEqual(plain(draft.summary(501)),{final:[0,0],added:[0,0],removed:[1,3]});
  assert.equal(draft.count(),2);
});

let chromium;
try { ({ chromium } = require("playwright")); } catch (_) { /* Optional browser runtime in CI. */ }
test("roster checkboxes, modes and searchable rule inputs support selective supplementation", { skip: !chromium && "Install Playwright to run browser coverage" }, async t => {
  const browser=await chromium.launch({headless:true,...(process.platform==="win32"?{channel:"msedge"}:{})}); t.after(()=>browser.close());
  const page=await browser.newPage({viewport:{width:1440,height:900}}), errors=[];
  page.on("pageerror",error=>errors.push(error.message));
  const html=readFileSync(path.join(__dirname,"admin.html"),"utf8").replace(/<script\b[^>]*>[\s\S]*?<\/script>/g,"");
  await page.route("http://admin.test/**",route=>route.fulfill({contentType:route.request().url().endsWith(".css")?"text/css":"text/html",body:route.request().url().endsWith(".css")?readFileSync(path.join(__dirname,"admin.css"),"utf8"):html}));
  await page.goto("http://admin.test/admin/");
  await page.addScriptTag({path:path.join(__dirname,"admin_search_select.js")});
  await page.addScriptTag({path:path.join(__dirname,"admin_quest_bonus.js")});
  const main=readFileSync(path.join(__dirname,"admin.js"),"utf8").replace(/\}\)\(\);\s*$/,"globalThis.integration={state,renderCatalog,renderTable,showWorkspace}; globalThis.editor=questBonusEditor; })();");
  await page.addScriptTag({content:main});
  const data=catalog(); data.chapters=[501,589].map(id=>({values:{EventQuestChapterId:String(id),StartDatetime:"1",EndDatetime:"2"},titles:{ja:`活动 ${id}`}}));
  data.tables.forEach(table=>table.fields.forEach(field=>field.type="Int32"));
  const weaponRows=data.tables.find(t=>t.name==="m_quest_bonus_weapon_group").rows;
  for (const gid of [50,51,52]) for (const [level,effect] of [[2,20],[4,40]]) weaponRows.push({values:{QuestBonusWeaponGroupId:String(gid),WeaponId:"201",LimitBreakCountLowerLimit:String(level),QuestBonusEffectGroupId:String(effect),QuestBonusTermGroupId:"40"}});
  for (const amount of [20,40]) {
    data.tables.find(t=>t.name==="m_quest_bonus_effect_group").rows.push({values:{QuestBonusEffectGroupId:String(amount),SortOrder:"1",QuestBonusType:"3",QuestBonusEffectId:String(amount)}});
    data.tables.find(t=>t.name==="m_quest_bonus_drop_reward").rows.push({values:{QuestBonusEffectId:String(amount),PossessionType:"6",PossessionId:"181",AdditionalCount:String(amount)}});
  }
  await page.evaluate(data=>{
    const api=window.integration;
    api.state.catalog={...data,version:"0".repeat(64),languages:["ja"],tables:[...data.tables,{name:"m_test_related",entityName:"EntityMTestRelated",fields:[],rows:[]}]};
    api.state.language="ja"; api.state.section="related"; api.state.tableSelections.related="m_test_related";
    window.editor.load(data); api.showWorkspace(); api.renderCatalog(); api.renderTable();
  },data);
  const layout=()=>page.evaluate(()=>[".topbar","main",".summary-grid",".toolbar",".data-heading.table-section-only","#table-scroll",".savebar"].map(selector=>{
    const el=document.querySelector(selector),s=getComputedStyle(el);
    return [selector,s.display,s.padding,s.margin,s.fontSize,s.overflowY,s.maxHeight,el.getBoundingClientRect().width];
  }));
  const before=await layout();
  await page.locator("#search").fill("保留其他表筛选");
  await page.locator("#table-select").selectOption("m_quest_bonus");
  const activity=page.getByRole("combobox",{name:"目标活动",exact:true});
  await activity.waitFor();
  assert.deepEqual(await layout(),before,"QuestBonus must use the common table container and outer layout");
  assert.equal(await activity.evaluate(el=>el.closest("#table-search-label")!==null),true);
  assert.equal(await page.locator("#search").isVisible(),false);
  await activity.fill("501"); await page.getByRole("option",{name:"活动 501 · 501",exact:true}).click();
  assert.equal(await page.locator(".bonus-events, .bonus-diff-scroll").count(),0);
  assert.equal(await page.getByRole("combobox",{name:"预览突破档位"}).count(),0);
  const dates=page.locator(".bonus-activity-dates");
  assert.match(await dates.innerText(),/1970.*—.*1970/);
  assert.equal(await dates.evaluate(el=>el.previousElementSibling.tagName),"H2");
  assert.equal(await page.getByText("全部共鸣跟随活动").count(),0);
  await page.getByRole("button",{name:"保留当前并补充",exact:true}).click();
  const source=page.getByRole("combobox",{name:"历史名单来源",exact:true});
  await source.fill("30"); await page.getByRole("option",{name:/^30 ·/}).click();
  assert.equal(await source.inputValue(),"30 · 服装组 21 (兵器の祭典 等) / 武器组 60 (同名の武器 等)");
  for (const name of ["少女の祭典","神秘石の杖"]) {
    await source.fill(name); await page.getByRole("option",{name:/^30 ·/}).waitFor(); await source.press("Escape");
  }
  assert.equal(await page.getByRole("checkbox",{name:/兵器の祭典/}).isDisabled(),true);
  assert.equal(await page.getByRole("checkbox",{name:/· 101$/}).isDisabled(),true);
  assert.equal(await page.locator('.bonus-weapon-row input[role="combobox"]').count(),0);
  assert.equal(await page.evaluate(()=>window.editor.count()),0);
  await page.getByRole("checkbox",{name:/少女の祭典/}).check();
  assert.equal(await page.evaluate(()=>window.editor.payload().questBonusRestores[0].weapons.length),0);
  await page.getByRole("checkbox",{name:/· 401$/}).check();
  const rule=page.getByRole("combobox",{name:"武器规则 401",exact:true});
  await rule.fill("201"); await page.getByRole("option",{name:/^规则 2/}).click();
  assert.match(await page.locator(".bonus-summary").innerText(),/最终 2 套服装 \/ 4 种武器.*新增 1 套服装 \/ 1 种武器.*移除 0 套服装 \/ 0 种武器/);
  assert.equal(await page.locator('.bonus-roster:first-child strong[title$="201"]').locator("..").locator("small").innerText(),"物品 181 +10 / +10 / +20 / +20 / +40");
  assert.equal(await page.locator('.bonus-roster:nth-child(2) .bonus-weapon-row').filter({has:page.getByRole("checkbox",{name:/· 401$/})}).locator("small").innerText(),"物品 181 +10 / +10 / +20 / +20 / +40");
  assert.equal(await page.locator(".bonus-warning").count(),0);
  await page.getByRole("button",{name:"武器清空",exact:true}).click();
  assert.equal(await page.locator('.bonus-weapon-row input[role="combobox"]').count(),0);
  await page.getByRole("button",{name:"服装清空",exact:true}).click();
  assert.equal(await page.evaluate(()=>window.editor.count()),0);
  await page.getByRole("button",{name:"替换名单",exact:true}).click();
  assert.equal(await page.getByRole("checkbox").count(),4);
  assert.equal(await page.locator('input[type="checkbox"]:checked').count(),4);
  await page.getByRole("button",{name:"保留当前并补充",exact:true}).click();
  await page.getByRole("button",{name:"服装全选",exact:true}).click();
  await page.getByRole("button",{name:"武器全选",exact:true}).click();
  assert.equal(await page.locator('.bonus-weapon-row input[role="combobox"]').count(),1);
  await source.fill("10"); await page.getByRole("option",{name:/^10 ·/}).click();
  assert.equal(await page.evaluate(()=>window.editor.count()),0);
  await page.locator("#table-select").selectOption("m_test_related");
  await page.locator("#search").waitFor();
  assert.equal(await page.locator("#search").inputValue(),"保留其他表筛选");
  assert.equal(await activity.isVisible(),false);
  await page.locator("#table-select").selectOption("m_quest_bonus");
  await activity.waitFor(); assert.equal(await activity.inputValue(),"活动 501 · 501");
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  assert.deepEqual(errors,[]);
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
