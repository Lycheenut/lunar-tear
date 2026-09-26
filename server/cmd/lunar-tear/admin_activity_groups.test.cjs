const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");
const vm = require("node:vm");

const descendants = node => node.children.flatMap(child => [child, ...descendants(child)]);
function element(tagName = "div") {
  return {
    tagName, children: [], dataset: {}, attributes: {}, listeners: {}, scrollTop: 0, textContent: "", className: "",
    append(...children) { children.forEach(child => { child.parentNode = this; }); this.children.push(...children); },
    replaceChildren(...children) { this.children = children; },
    addEventListener(event, listener) { this.listeners[event] = listener; },
    setAttribute(name, value) { this.attributes[name] = value; },
    showModal() { this.open = true; },
    close() { this.open = false; this.listeners.close?.(); },
    remove() { if (this.parentNode) this.parentNode.children = this.parentNode.children.filter(child => child !== this); },
    get classList() { return {
      add: name => { if (!this.className.split(" ").includes(name)) this.className += ` ${name}`; },
      remove: name => { this.className = this.className.split(" ").filter(value => value !== name).join(" "); }
    }; },
    querySelectorAll(selector) {
      return descendants(this).filter(node => selector.startsWith(".")
        ? node.className.split(" ").includes(selector.slice(1))
        : Object.hasOwn(node.dataset, selector.slice(6, -1).replace(/-([a-z])/g, (_, letter) => letter.toUpperCase())));
    },
    querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
  };
}
const findText = (root, text) => descendants(root).find(node => node.textContent === text);
const check = (checkbox, value = true) => { checkbox.checked = value; checkbox.listeners.change(); };
const previewRows = body => descendants(body).filter(node => node.tagName === "tr" && node.children[0]?.tagName === "td").map(row => row.children.map(cell => cell.textContent));
const renameSelected = (root, value) => {
  const input = root.querySelector(".activity-group-header").children.find(node => node.textContent === "名称").children[0];
  input.value = value; input.listeners.input();
};

async function createEditor(premiumMembers = [], extraOptions = [], settings = {}) {
  const root = element(), body = element("body"), posts = [], requests = [], confirmations = [], notices = [];
  let allowDelete = true;
  const member = (kind, id) => ({ kind, id });
  let config = { version: 2, units: [
    { id: "premium:1", name: "Summons", type: 1, members: [member("premium", 1), ...premiumMembers] },
    { id: "chapter:2", name: "Record", type: 2, members: [member("chapter", 2)] },
    { id: "chapter:3", name: "Variation", type: 3, members: [member("chapter", 3)] }
  ], groups: [
    { id: "only", name: "Only premium", unitIds: ["premium:1"] },
    { id: "shared", name: "Shared", unitIds: ["premium:1", "chapter:2"] }
  ] };
  const kinds = Object.entries({ premium: [1], chapter: [2, 3], banner: [1, 2, 3], shop: [1, 2], event: [3], term: [1, 2, 3], medal: [1], mission: [2, 3], navi: [2, 3], tip: [2, 3] }).map(([kind, types]) => ({ kind, types }));
  const options = [
    { ...member("premium", 1), titles: { en: "Summons", ja: "記念ガチャ" }, previewPath: ["gacha", "limited_1", "banner.png"] },
    { ...member("chapter", 2), titles: { en: "Record" }, chapterType: 1 },
    { ...member("chapter", 3), titles: { en: "Variation" }, chapterType: 2 },
    { ...member("event", 4), titles: { en: "Event" }, relatedChapterId: 3 },
    { ...member("event", 5), titles: { en: "Unrelated" }, relatedChapterId: 99 },
    { ...member("banner", 6), titles: { en: "Summons banner" }, previewPath: ["gacha", "limited_1", "mom_banner.png"] },
    { ...member("term", 8), titles: { en: "Shards" }, endDatetime: 1800000000000 },
    { ...member("medal", 8), titles: { en: "Shards" }, endDatetime: 1800100000000 },
    { ...member("premium", 9), titles: { en: "Other summons" }, previewPath: ["gacha", "limited_9", "banner.png"] },
    { ...member("chapter", 10), titles: { en: "Other Record" }, chapterType: 1 },
    { ...member("chapter", 99), titles: { en: "Other Variation" }, chapterType: 2 },
    ...extraOptions
  ];
  const context = vm.createContext({
    document: { createElement: element, body },
    crypto: { randomUUID: () => "new-unit" },
    Option: function(text, value) { return Object.assign(element("option"), { textContent: text, value }); },
    window: { confirm: message => { confirmations.push(message); return allowDelete; } }
  });
  vm.runInContext(readFileSync(path.join(__dirname, "admin_activity_groups.js"), "utf8"), context);
  const editor = context.window.createActivityGroupEditor({
    root, localizedText: titles => titles?.ja || titles?.en || "", hasOtherChanges: settings.hasOtherChanges || (() => false),
    renderBannerPreview: option => Object.assign(element("img"), { src: option.previewPath.join("/") }),
    showNotice: (text, error) => notices.push({ text, error: Boolean(error) }), onPublished: async () => {},
    api: async (_, request) => {
      if (request) { requests.push(JSON.parse(request.body)); await settings.beforeSave?.(); config = JSON.parse(request.body).config; posts.push(config); }
      return { contentHash: "activity-hash", gachaConfigHash: "gacha-hash", masterDataHash: "master-hash", catalog: { config, kinds, options } };
    }
  });
  await editor.load();
  findText(root, "活动单位 · 3").listeners.click();
  const save = async () => {
    const count = requests.length;
    findText(root, "保存活动组配置").listeners.click();
    assert.equal(requests.length, count, "saving must wait for confirmation");
    const dialog = body.children.find(node => node.tagName === "dialog");
    assert.ok(dialog?.open);
    await findText(dialog, "确认保存").listeners.click();
  };
  return { root, body, editor, posts, requests, confirmations, notices, save, confirm: value => { allowDelete = value; } };
}

test("save preview shows names, types and member changes and cancellation preserves the draft", async () => {
  const { root, body, editor, posts, requests, save } = await createEditor();
  findText(root, "2. Record").listeners.click();
  renameSelected(root, "Renamed event");
  const type = descendants(root).find(node => node.attributes["aria-label"] === "单位类型");
  type.value = "3"; type.listeners.change();
  const chapter = descendants(root).find(node => node.attributes["aria-label"] === "选择変異（Variation）");
  chapter.value = "chapter:99"; chapter.listeners.change();
  findText(root, "活动组 · 2").listeners.click();
  findText(root, "Shared").listeners.click();
  renameSelected(root, "Renamed group");
  findText(root.querySelector(".activity-group-unit-table"), "移除").listeners.click();
  findText(root, "保存活动组配置").listeners.click();
  assert.equal(requests.length, 0);
  assert.deepEqual(previewRows(body), [
    ["活动组：Renamed group", "名称", "Shared", "Renamed group"],
    ["活动组：Renamed group", "移除成员", "1. 記念ガチャ", "—"],
    ["活动单位：2. Renamed event", "名称", "Record", "Renamed event"],
    ["活动单位：2. Renamed event", "单位类型", "記録（Record）", "変異（Variation）"],
    ["活动单位：2. Renamed event", "移除成员", "記録（Record）：2. Record", "—"],
    ["活动单位：2. Renamed event", "加入成员", "—", "変異（Variation）：99. Other Variation"]
  ]);
  findText(body, "取消").listeners.click();
  assert.equal(body.children.length, 0);
  assert.equal(editor.dirty(), true);
  assert.equal(posts.length, 0);
  await save();
  assert.equal(requests[0].expectedContentHash, "activity-hash");
  assert.equal(requests[0].expectedGachaConfigHash, "gacha-hash");
  assert.equal(requests[0].expectedMasterDataHash, "master-hash");
  assert.deepEqual(posts[0].groups[1].unitIds, ["chapter:2"]);
  assert.equal(posts[0].units[1].name, "Renamed event");
  assert.equal(editor.dirty(), false);
  assert.equal(body.children.length, 0);
});

test("save preview includes additions, deletions and the affected group membership", async () => {
  const { root, body, requests } = await createEditor();
  check(root.querySelectorAll(".activity-group-list-check")[1]);
  root.querySelector(".activity-group-batch-delete").listeners.click();
  findText(root, "新建活动单位").listeners.click();
  const source = descendants(root).find(node => node.attributes["aria-label"] === "选择Premium Gacha");
  source.value = "premium:9"; source.listeners.change();
  findText(root, "活动组 · 2").listeners.click();
  findText(root, "新建活动组").listeners.click();
  const unit = descendants(root).find(node => node.attributes["aria-label"] === "组合活动单位");
  unit.value = "unit-new-unit"; unit.listeners.change();
  findText(root, "加入活动组").listeners.click();
  findText(root, "保存活动组配置").listeners.click();
  const rows = previewRows(body);
  assert.equal(rows.length, 4);
  assert.deepEqual(rows[0], ["活动组：Shared", "移除成员", "2. Record", "—"]);
  assert.deepEqual(rows[1], ["活动组：新活动组", "新增", "—", "名称：新活动组\nunit-new-unit. 新活动单位"]);
  assert.deepEqual(rows[2], ["活动单位：2. Record", "删除", "名称：Record\n类型：記録（Record）\n記録（Record）：2. Record", "—"]);
  assert.deepEqual(rows[3], ["活动单位：unit-new-unit. 新活动单位", "新增", "—", "名称：新活动单位\n类型：Premium Gacha\nPremium Gacha：9. Other summons"]);
  assert.equal(requests.length, 0);
});

test("discard requires confirmation and cancel keeps the draft and selection", async () => {
  const { root, body, editor, requests, confirmations, confirm } = await createEditor();
  findText(root, "1. 記念ガチャ").listeners.click();
  renameSelected(root, "Unsaved name");
  check(root.querySelector(".activity-group-list-check"));
  confirm(false);
  findText(root, "放弃修改").listeners.click();
  assert.match(confirmations.at(-1), /放弃全部尚未保存的活动组和活动单位修改/);
  assert.equal(editor.dirty(), true);
  assert.ok(findText(root, "1. Unsaved name"));
  assert.equal(root.querySelector(".activity-group-list-check").checked, true);
  confirm(true);
  findText(root, "放弃修改").listeners.click();
  assert.equal(editor.dirty(), false);
  assert.ok(findText(root, "1. 記念ガチャ"));
  assert.equal(root.querySelector(".activity-group-list-check").checked, false);
  assert.equal(requests.length, 0);
  assert.equal(body.children.length, 0);
});

test("failed confirmed saves preserve the preview and draft for retry and block duplicate submits", async () => {
  let fail = true, release;
  const gate = new Promise(resolve => { release = resolve; });
  const { root, body, editor, requests, posts } = await createEditor([], [], { beforeSave: async () => { await gate; if (fail) throw new Error("配置已被修改，请刷新后重试"); } });
  findText(root, "1. 記念ガチャ").listeners.click();
  renameSelected(root, "Pending name");
  findText(root, "保存活动组配置").listeners.click();
  const dialog = body.children[0], submit = findText(dialog, "确认保存"), cancel = findText(dialog, "取消");
  const pending = submit.listeners.click();
  await submit.listeners.click();
  assert.equal(requests.length, 1);
  assert.equal(submit.disabled, true);
  assert.equal(cancel.disabled, true);
  let prevented = false;
  dialog.listeners.cancel({ preventDefault: () => { prevented = true; } });
  assert.equal(prevented, true);
  release(); await pending;
  assert.equal(editor.dirty(), true);
  assert.equal(dialog.open, true);
  assert.ok(findText(dialog, "配置已被修改，请刷新后重试"));
  assert.equal(submit.disabled, false);
  assert.equal(posts.length, 0);
  fail = false;
  await submit.listeners.click();
  assert.equal(requests.length, 2);
  assert.equal(posts.length, 1);
  assert.equal(posts[0].units[0].name, "Pending name");
  assert.equal(editor.dirty(), false);
  assert.equal(body.children.length, 0);
});

test("changes on another page prevent opening the save confirmation", async () => {
  const { root, body, requests, notices, editor } = await createEditor([], [], { hasOtherChanges: () => true });
  findText(root, "1. 記念ガチャ").listeners.click();
  renameSelected(root, "Pending name");
  findText(root, "保存活动组配置").listeners.click();
  assert.equal(body.children.length, 0);
  assert.equal(requests.length, 0);
  assert.equal(notices.at(-1).error, true);
  assert.equal(editor.dirty(), true);
});

test("activity selection preserves the sidebar node and scroll position, with localized ID-first names", async () => {
  const { root, editor } = await createEditor();
  const list = root.querySelector(".activity-group-sidebar"); list.scrollTop = 420;
  check(root.querySelector(".activity-group-list-check"));
  findText(root, "1. 記念ガチャ").listeners.click();
  assert.equal(root.querySelector(".activity-group-sidebar"), list);
  assert.equal(list.scrollTop, 420);
  assert.equal(root.querySelector(".activity-group-list-check").checked, true);
  assert.equal(root.querySelector(".activity-group-select-all").indeterminate, true);
  assert.equal(editor.dirty(), false);
});

test("Record and Variation sections restrict chapter and Event Gacha choices", async () => {
  const { root } = await createEditor();
  findText(root, "2. Record").listeners.click();
  assert.ok(findText(root, "活动商店"));
  assert.equal(findText(root, "Event Gacha"), undefined);
  const chapterPicker = descendants(root).find(node => node.tagName === "select" && node.attributes["aria-label"] === "选择記録（Record）");
  assert.deepEqual(chapterPicker.children.map(option => option.value), ["", "chapter:2", "chapter:10"]);
  assert.equal(chapterPicker.value, "chapter:2");
  findText(root, "3. Variation").listeners.click();
  assert.equal(findText(root, "活动商店"), undefined);
  const picker = descendants(root).find(node => node.tagName === "select" && node.attributes["aria-label"] === "添加Event Gacha");
  assert.deepEqual(picker.children.map(option => option.value), ["", "event:4"]);
  assert.equal(picker.children[1].dataset.searchLabel, "4. Event");
});

test("primary selection replaces the source, keeps other members, and preserves both scroll positions", async () => {
  const { root, posts, save } = await createEditor([{ kind: "banner", id: 6 }]);
  findText(root, "1. 記念ガチャ").listeners.click();
  const section = descendants(root).find(node => node.dataset.memberSection === "premium");
  assert.equal(findText(section, "移除"), undefined);
  assert.equal(findText(section, "添加条目"), undefined);
  const picker = descendants(section).find(node => node.tagName === "select");
  assert.equal(picker.value, "premium:1");
  assert.equal(picker.children[0].disabled, true);
  root.querySelector(".activity-group-sidebar").scrollTop = 420;
  root.querySelector(".activity-group-detail").scrollTop = 180;
  picker.value = "premium:9"; picker.listeners.change();
  assert.equal(root.querySelector(".activity-group-sidebar").scrollTop, 420);
  assert.equal(root.querySelector(".activity-group-detail").scrollTop, 180);
  await save();
  assert.deepEqual(posts[0].units[0].members, [{ kind: "premium", id: 9 }, { kind: "banner", id: 6 }]);
  assert.equal(posts[0].units[0].id, "premium:1");
  findText(root, "2. Record").listeners.click();
  assert.equal(root.querySelector(".activity-group-detail").scrollTop, 0);
});

test("changing the sole Variation chapter removes its former Event Gacha and offers the new chapter's gacha", async () => {
  const { root, posts, save } = await createEditor();
  findText(root, "3. Variation").listeners.click();
  let section = descendants(root).find(node => node.dataset.memberSection === "event");
  const eventPicker = descendants(section).find(node => node.tagName === "select");
  eventPicker.value = "event:4"; eventPicker.listeners.change();
  findText(section, "添加条目").listeners.click();
  section = descendants(root).find(node => node.dataset.memberSection === "chapter");
  assert.equal(findText(section, "移除"), undefined);
  const picker = descendants(section).find(node => node.tagName === "select");
  picker.value = "chapter:99"; picker.listeners.change();
  const replacementPicker = descendants(root).find(node => node.attributes["aria-label"] === "添加Event Gacha");
  assert.deepEqual(replacementPicker.children.map(option => option.value), ["", "event:5"]);
  await save();
  assert.deepEqual(posts[0].units[2].members, [{ kind: "chapter", id: 99 }]);
});

test("new units choose their primary entry directly without an add button", async () => {
  const { root, posts, save } = await createEditor();
  findText(root, "新建活动单位").listeners.click();
  const section = descendants(root).find(node => node.dataset.memberSection === "premium");
  const picker = descendants(section).find(node => node.tagName === "select");
  assert.equal(picker.value, "");
  assert.equal(findText(section, "添加条目"), undefined);
  picker.value = "premium:9"; picker.listeners.change();
  await save();
  assert.deepEqual(posts[0].units.at(-1).members, [{ kind: "premium", id: 9 }]);
});

for (const [unitName, unitIndex, kind, ids] of [
  ["1. 記念ガチャ", 0, "shop", [11, 12]],
  ["2. Record", 1, "shop", [11, 12]]
]) test(`${unitName} selects, replaces, and clears one ${kind} without changing other members`, async () => {
  const { root, posts, save } = await createEditor([], [
    { kind: "shop", id: 11, titles: { en: "Exchange" } },
    { kind: "shop", id: 12, titles: { en: "Other exchange" } },
    { kind: "event", id: 13, titles: { en: "Other event" }, relatedChapterId: 3 }
  ]);
  findText(root, unitName).listeners.click();
  const selected = [];
  for (const id of [...ids, null]) {
    const section = descendants(root).find(node => node.dataset.memberSection === kind);
    assert.equal(findText(section, "添加条目"), undefined);
    assert.equal(findText(section, "移除"), undefined);
    const picker = descendants(section).find(node => node.tagName === "select");
    assert.equal(Boolean(picker.children[0].disabled), false);
    picker.value = id === null ? "" : `${kind}:${id}`; picker.listeners.change();
    await save();
    selected.push(posts.at(-1).units[unitIndex].members.filter(member => member.kind === kind));
    assert.equal(posts.at(-1).units[unitIndex].members.filter(member => member.kind !== kind).length, 1);
  }
  assert.deepEqual(selected, [[{ kind, id: ids[0] }], [{ kind, id: ids[1] }], []]);
});

test("Variation keeps all ticket pools as independent entries and removes only the chosen tier", async () => {
  const ids = [329001, 329011, 329021];
  const { root, posts, save } = await createEditor([], ids.map((id, index) => ({
    kind: "event", id, relatedChapterId: 3, titles: { ja: ["極光の覇王・銅", "極光の覇王・銀", "極光の覇王・金"][index] }
  })));
  findText(root, "3. Variation").listeners.click();
  for (const id of ids) {
    const section = descendants(root).find(node => node.dataset.memberSection === "event");
    const picker = descendants(section).find(node => node.tagName === "select");
    picker.value = `event:${id}`; picker.listeners.change();
    findText(section, "添加条目").listeners.click();
  }
  await save();
  assert.deepEqual(posts[0].units[2].members.slice(1), ids.map(id => ({ kind: "event", id })));
  let section = descendants(root).find(node => node.dataset.memberSection === "event");
  assert.equal(section.querySelectorAll(".activity-group-entry").length, 3);
  const silver = section.querySelectorAll(".activity-group-entry").find(row => findText(row, "329011. 極光の覇王・銀"));
  findText(silver, "移除").listeners.click();
  await save();
  assert.deepEqual(posts[1].units[2].members.slice(1), [329001, 329021].map(id => ({ kind: "event", id })));
  section = descendants(root).find(node => node.dataset.memberSection === "event");
  const picker = descendants(section).find(node => node.tagName === "select");
  assert.deepEqual(picker.children.map(option => option.value), ["", "event:4", "event:329011"]);
});

for (const [unitName, unitIndex] of [["2. Record", 1], ["3. Variation", 2]]) {
  test(`${unitName} keeps bronze, silver, and gold terms as independent removable entries`, async () => {
    const { root, posts, save } = await createEditor([], [14, 15, 16].map((id, index) => ({
      kind: "term", id, titles: { en: ["Bronze", "Silver", "Gold"][index] }, endDatetime: 1800000000000 + index * 1000
    })));
    findText(root, unitName).listeners.click();
    for (const id of [14, 15, 16]) {
      const section = descendants(root).find(node => node.dataset.memberSection === "term");
      const picker = descendants(section).find(node => node.tagName === "select");
      picker.value = `term:${id}`; picker.listeners.change();
      findText(section, "添加条目").listeners.click();
    }
    let section = descendants(root).find(node => node.dataset.memberSection === "term");
    assert.equal(section.querySelectorAll(".activity-group-entry").length, 3);
    await save();
    assert.deepEqual(posts[0].units[unitIndex].members.filter(member => member.kind === "term"), [14, 15, 16].map(id => ({ kind: "term", id })));
    section = descendants(root).find(node => node.dataset.memberSection === "term");
    const silver = section.querySelectorAll(".activity-group-entry").find(row => findText(row, "15. Silver"));
    findText(silver, "移除").listeners.click();
    await save();
    assert.deepEqual(posts[1].units[unitIndex].members.filter(member => member.kind === "term"), [14, 16].map(id => ({ kind: "term", id })));
    section = descendants(root).find(node => node.dataset.memberSection === "term");
    const picker = descendants(section).find(node => node.tagName === "select");
    assert.deepEqual(picker.children.map(option => option.value), ["", "term:8", "term:15"]);
  });
}

test("batch deletion can be cancelled and removes groups emptied by multiple selected units", async () => {
  const { root, editor, posts, confirmations, confirm, save } = await createEditor();
  assert.equal(root.querySelector(".activity-group-batch-delete").disabled, true);
  assert.equal(root.querySelector(".activity-group-list-delete"), null);
  check(root.querySelectorAll(".activity-group-list-check")[0]);
  check(root.querySelectorAll(".activity-group-list-check")[1]);
  assert.equal(root.querySelector(".activity-group-batch-delete").textContent, "删除所选（2）");
  confirm(false);
  root.querySelector(".activity-group-batch-delete").listeners.click();
  assert.equal(editor.dirty(), false);
  assert.equal(root.querySelectorAll(".activity-group-list-check").filter(checkbox => checkbox.checked).length, 2);
  confirm(true);
  root.querySelector(".activity-group-batch-delete").listeners.click();
  assert.match(confirmations.at(-1), /所选的 2 个活动单位/);
  assert.match(confirmations.at(-1), /变空的 2 个活动组/);
  assert.equal(root.querySelector(".activity-group-batch-delete").disabled, true);
  await save();
  assert.deepEqual(posts[0].units.map(unit => unit.id), ["chapter:3"]);
  assert.deepEqual(posts[0].groups, []);
  assert.equal(editor.dirty(), false);
});

test("select all applies to filtered results while selections survive filtering and stay separate by mode", async () => {
  const { root, editor } = await createEditor();
  check(root.querySelector(".activity-group-list-check"));
  const search = descendants(root).find(node => node.attributes["aria-label"] === "搜索活动组或单位");
  search.value = "chapter:"; search.listeners.input();
  check(root.querySelector(".activity-group-select-all"));
  assert.equal(root.querySelector(".activity-group-batch-delete").textContent, "删除所选（3）");
  assert.equal(root.querySelector(".activity-group-select-all").checked, true);
  check(root.querySelector(".activity-group-select-all"), false);
  assert.equal(root.querySelector(".activity-group-batch-delete").textContent, "删除所选（1）");
  search.value = "missing"; search.listeners.input();
  assert.equal(root.querySelector(".activity-group-select-all").disabled, true);
  assert.equal(root.querySelector(".activity-group-batch-delete").disabled, false);
  findText(root, "活动组 · 2").listeners.click();
  assert.equal(root.querySelector(".activity-group-batch-delete").disabled, true);
  findText(root, "活动单位 · 3").listeners.click();
  assert.equal(root.querySelector(".activity-group-list-check").checked, true);
  assert.equal(editor.dirty(), false);
  findText(root, "放弃修改").listeners.click();
  assert.equal(root.querySelector(".activity-group-batch-delete").disabled, true);
});

test("batch deleting groups leaves their units intact", async () => {
  const { root, posts, save } = await createEditor();
  findText(root, "活动组 · 2").listeners.click();
  check(root.querySelector(".activity-group-select-all"));
  root.querySelector(".activity-group-batch-delete").listeners.click();
  await save();
  assert.equal(posts[0].groups.length, 0);
  assert.equal(posts[0].units.length, 3);
});

test("deleting one selected unit preserves groups with remaining units", async () => {
  const { root, posts, save } = await createEditor();
  check(root.querySelector(".activity-group-list-check"));
  root.querySelector(".activity-group-batch-delete").listeners.click();
  await save();
  assert.deepEqual(posts[0].units.map(unit => unit.id), ["chapter:2", "chapter:3"]);
  assert.deepEqual(posts[0].groups, [{ id: "shared", name: "Shared", unitIds: ["chapter:2"] }]);
});

test("unit type filters intersect search and limit select-all without filtering groups", async () => {
  const { root, editor } = await createEditor();
  const filter = descendants(root).find(node => node.attributes["aria-label"] === "类型筛选");
  filter.value = "2"; filter.listeners.change();
  assert.deepEqual(root.querySelectorAll(".activity-group-list-item").map(node => node.title), ["chapter:2"]);
  check(root.querySelector(".activity-group-select-all"));
  assert.equal(root.querySelector(".activity-group-batch-delete").textContent, "删除所选（1）");
  const search = descendants(root).find(node => node.attributes["aria-label"] === "搜索活动组或单位");
  search.value = "Summons"; search.listeners.input();
  assert.equal(root.querySelectorAll(".activity-group-list-item").length, 0);
  assert.equal(editor.dirty(), false);
  findText(root, "活动组 · 2").listeners.click();
  assert.equal(descendants(root).find(node => node.attributes["aria-label"] === "类型筛选"), undefined);
  assert.deepEqual(root.querySelectorAll(".activity-group-list-item").map(node => node.textContent), ["Only premium", "Shared"]);
  findText(root, "活动单位 · 3").listeners.click();
  assert.equal(descendants(root).find(node => node.attributes["aria-label"] === "类型筛选").value, "2");
  assert.deepEqual(root.querySelectorAll(".activity-group-list-item").map(node => node.title), ["chapter:2"]);
});

test("shard terms add and remove same-ID conversion entries together without a separate picker", async () => {
  const { root, posts, save } = await createEditor();
  findText(root, "1. 記念ガチャ").listeners.click();
  let section = descendants(root).find(node => node.dataset.memberSection === "term");
  const picker = descendants(section).find(node => node.tagName === "select");
  assert.deepEqual(picker.children.map(node => node.value), ["", "term:8"]);
  assert.equal(picker.children[1].textContent, "8. Shards");
  picker.value = "term:8"; picker.listeners.change();
  findText(section, "添加条目").listeners.click();
  section = descendants(root).find(node => node.dataset.memberSection === "term");
  assert.equal(section.querySelectorAll(".activity-group-entry").length, 1);
  assert.ok(descendants(section).some(node => node.textContent.startsWith("自动转换时间：")));
  assert.equal(descendants(section).some(node => /（有效期）|（自动转换）/.test(node.textContent)), false);
  await save();
  assert.deepEqual(posts[0].units[0].members.slice(1), [{ kind: "term", id: 8 }, { kind: "medal", id: 8 }]);
  section = descendants(root).find(node => node.dataset.memberSection === "term");
  findText(section, "移除").listeners.click();
  await save();
  assert.deepEqual(posts[1].units[0].members, [{ kind: "premium", id: 1 }]);
});

test("legacy conversion-only selections remain visible as shard terms and banner cards render previews", async () => {
  const { root, editor } = await createEditor([{ kind: "medal", id: 8 }, { kind: "banner", id: 6 }]);
  findText(root, "1. 記念ガチャ").listeners.click();
  assert.ok(findText(root, "8. Shards"));
  const bannerSection = descendants(root).find(node => node.dataset.memberSection === "banner");
  const card = bannerSection.querySelector(".activity-group-banner-card");
  assert.equal(card.children[0].src, "gacha/limited_1/mom_banner.png");
  assert.ok(findText(card, "6. Summons banner"));
  const premiumSection = descendants(root).find(node => node.dataset.memberSection === "premium");
  const premiumRow = premiumSection.querySelector(".activity-group-premium-row");
  assert.equal(premiumRow.children[0].src, "gacha/limited_1/banner.png");
  assert.ok(findText(premiumRow, "1. 記念ガチャ"));
  assert.equal(editor.dirty(), false);
});
