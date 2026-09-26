const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");
const vm = require("node:vm");

const descendants = node => node.children.flatMap(child => [child, ...descendants(child)]);
function element(tagName = "div") {
  return {
    tagName, children: [], dataset: {}, attributes: {}, listeners: {}, scrollTop: 0, textContent: "", className: "",
    append(...children) { this.children.push(...children); },
    replaceChildren(...children) { this.children = children; },
    addEventListener(event, listener) { this.listeners[event] = listener; },
    setAttribute(name, value) { this.attributes[name] = value; },
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

async function createEditor(premiumMembers = [], extraOptions = []) {
  const root = element(), posts = [], confirmations = [];
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
    document: { createElement: element },
    crypto: { randomUUID: () => "new-unit" },
    Option: function(text, value) { return Object.assign(element("option"), { textContent: text, value }); },
    window: { confirm: message => { confirmations.push(message); return allowDelete; } }
  });
  vm.runInContext(readFileSync(path.join(__dirname, "admin_activity_groups.js"), "utf8"), context);
  const editor = context.window.createActivityGroupEditor({
    root, localizedText: titles => titles?.ja || titles?.en || "", hasOtherChanges: () => false,
    renderBannerPreview: option => Object.assign(element("img"), { src: option.previewPath.join("/") }),
    showNotice: (_, error) => assert.equal(Boolean(error), false), onPublished: async () => {},
    api: async (_, request) => {
      if (request) { config = JSON.parse(request.body).config; posts.push(config); }
      return { catalog: { config, kinds, options } };
    }
  });
  await editor.load();
  findText(root, "活动单位 · 3").listeners.click();
  return { root, editor, posts, confirmations, confirm: value => { allowDelete = value; } };
}

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
  const { root, posts } = await createEditor([{ kind: "banner", id: 6 }]);
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
  await findText(root, "保存活动组配置").listeners.click();
  assert.deepEqual(posts[0].units[0].members, [{ kind: "premium", id: 9 }, { kind: "banner", id: 6 }]);
  assert.equal(posts[0].units[0].id, "premium:1");
  findText(root, "2. Record").listeners.click();
  assert.equal(root.querySelector(".activity-group-detail").scrollTop, 0);
});

test("changing the sole Variation chapter removes its former Event Gacha and offers the new chapter's gacha", async () => {
  const { root, posts } = await createEditor();
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
  await findText(root, "保存活动组配置").listeners.click();
  assert.deepEqual(posts[0].units[2].members, [{ kind: "chapter", id: 99 }]);
});

test("new units choose their primary entry directly without an add button", async () => {
  const { root, posts } = await createEditor();
  findText(root, "新建活动单位").listeners.click();
  const section = descendants(root).find(node => node.dataset.memberSection === "premium");
  const picker = descendants(section).find(node => node.tagName === "select");
  assert.equal(picker.value, "");
  assert.equal(findText(section, "添加条目"), undefined);
  picker.value = "premium:9"; picker.listeners.change();
  await findText(root, "保存活动组配置").listeners.click();
  assert.deepEqual(posts[0].units.at(-1).members, [{ kind: "premium", id: 9 }]);
});

for (const [unitName, unitIndex, kind, ids] of [
  ["1. 記念ガチャ", 0, "shop", [11, 12]],
  ["2. Record", 1, "shop", [11, 12]]
]) test(`${unitName} selects, replaces, and clears one ${kind} without changing other members`, async () => {
  const { root, posts } = await createEditor([], [
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
    await findText(root, "保存活动组配置").listeners.click();
    selected.push(posts.at(-1).units[unitIndex].members.filter(member => member.kind === kind));
    assert.equal(posts.at(-1).units[unitIndex].members.filter(member => member.kind !== kind).length, 1);
  }
  assert.deepEqual(selected, [[{ kind, id: ids[0] }], [{ kind, id: ids[1] }], []]);
});

test("Variation keeps all ticket pools as independent entries and removes only the chosen tier", async () => {
  const ids = [329001, 329011, 329021];
  const { root, posts } = await createEditor([], ids.map((id, index) => ({
    kind: "event", id, relatedChapterId: 3, titles: { ja: ["極光の覇王・銅", "極光の覇王・銀", "極光の覇王・金"][index] }
  })));
  findText(root, "3. Variation").listeners.click();
  for (const id of ids) {
    const section = descendants(root).find(node => node.dataset.memberSection === "event");
    const picker = descendants(section).find(node => node.tagName === "select");
    picker.value = `event:${id}`; picker.listeners.change();
    findText(section, "添加条目").listeners.click();
  }
  await findText(root, "保存活动组配置").listeners.click();
  assert.deepEqual(posts[0].units[2].members.slice(1), ids.map(id => ({ kind: "event", id })));
  let section = descendants(root).find(node => node.dataset.memberSection === "event");
  assert.equal(section.querySelectorAll(".activity-group-entry").length, 3);
  const silver = section.querySelectorAll(".activity-group-entry").find(row => findText(row, "329011. 極光の覇王・銀"));
  findText(silver, "移除").listeners.click();
  await findText(root, "保存活动组配置").listeners.click();
  assert.deepEqual(posts[1].units[2].members.slice(1), [329001, 329021].map(id => ({ kind: "event", id })));
  section = descendants(root).find(node => node.dataset.memberSection === "event");
  const picker = descendants(section).find(node => node.tagName === "select");
  assert.deepEqual(picker.children.map(option => option.value), ["", "event:4", "event:329011"]);
});

for (const [unitName, unitIndex] of [["2. Record", 1], ["3. Variation", 2]]) {
  test(`${unitName} keeps bronze, silver, and gold terms as independent removable entries`, async () => {
    const { root, posts } = await createEditor([], [14, 15, 16].map((id, index) => ({
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
    await findText(root, "保存活动组配置").listeners.click();
    assert.deepEqual(posts[0].units[unitIndex].members.filter(member => member.kind === "term"), [14, 15, 16].map(id => ({ kind: "term", id })));
    section = descendants(root).find(node => node.dataset.memberSection === "term");
    const silver = section.querySelectorAll(".activity-group-entry").find(row => findText(row, "15. Silver"));
    findText(silver, "移除").listeners.click();
    await findText(root, "保存活动组配置").listeners.click();
    assert.deepEqual(posts[1].units[unitIndex].members.filter(member => member.kind === "term"), [14, 16].map(id => ({ kind: "term", id })));
    section = descendants(root).find(node => node.dataset.memberSection === "term");
    const picker = descendants(section).find(node => node.tagName === "select");
    assert.deepEqual(picker.children.map(option => option.value), ["", "term:8", "term:15"]);
  });
}

test("batch deletion can be cancelled and removes groups emptied by multiple selected units", async () => {
  const { root, editor, posts, confirmations, confirm } = await createEditor();
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
  await findText(root, "保存活动组配置").listeners.click();
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
  const { root, posts } = await createEditor();
  findText(root, "活动组 · 2").listeners.click();
  check(root.querySelector(".activity-group-select-all"));
  root.querySelector(".activity-group-batch-delete").listeners.click();
  await findText(root, "保存活动组配置").listeners.click();
  assert.equal(posts[0].groups.length, 0);
  assert.equal(posts[0].units.length, 3);
});

test("deleting one selected unit preserves groups with remaining units", async () => {
  const { root, posts } = await createEditor();
  check(root.querySelector(".activity-group-list-check"));
  root.querySelector(".activity-group-batch-delete").listeners.click();
  await findText(root, "保存活动组配置").listeners.click();
  assert.deepEqual(posts[0].units.map(unit => unit.id), ["chapter:2", "chapter:3"]);
  assert.deepEqual(posts[0].groups, [{ id: "shared", name: "Shared", unitIds: ["chapter:2"] }]);
});

test("type filters intersect search, limit select-all, and match groups by their member units", async () => {
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
  const groupFilter = descendants(root).find(node => node.attributes["aria-label"] === "类型筛选");
  groupFilter.value = "2"; groupFilter.listeners.change();
  assert.deepEqual(root.querySelectorAll(".activity-group-list-item").map(node => node.title), ["shared"]);
});

test("shard terms add and remove same-ID conversion entries together without a separate picker", async () => {
  const { root, posts } = await createEditor();
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
  await findText(root, "保存活动组配置").listeners.click();
  assert.deepEqual(posts[0].units[0].members.slice(1), [{ kind: "term", id: 8 }, { kind: "medal", id: 8 }]);
  section = descendants(root).find(node => node.dataset.memberSection === "term");
  findText(section, "移除").listeners.click();
  await findText(root, "保存活动组配置").listeners.click();
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
