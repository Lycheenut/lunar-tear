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

async function createEditor(premiumMembers = []) {
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
    { ...member("medal", 8), titles: { en: "Shards" }, endDatetime: 1800100000000 }
  ];
  const context = vm.createContext({
    document: { createElement: element },
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
  const list = root.querySelector(".activity-group-list"); list.scrollTop = 420;
  check(root.querySelector(".activity-group-list-check"));
  findText(root, "1. 記念ガチャ").listeners.click();
  assert.equal(root.querySelector(".activity-group-list"), list);
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
  let chapterPicker = descendants(root).find(node => node.tagName === "select" && node.attributes["aria-label"] === "添加記録（Record）");
  assert.equal(chapterPicker.children.length, 1, "selected Record is omitted and Variation is not offered");
  findText(root, "3. Variation").listeners.click();
  assert.equal(findText(root, "活动商店"), undefined);
  const picker = descendants(root).find(node => node.tagName === "select" && node.attributes["aria-label"] === "添加Event Gacha");
  assert.deepEqual(picker.children.map(option => option.value), ["", "event:4"]);
  assert.equal(picker.children[1].dataset.searchLabel, "4. Event");
});

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
  const premiumCard = premiumSection.querySelector(".activity-group-banner-card");
  assert.equal(premiumCard.children[0].src, "gacha/limited_1/banner.png");
  assert.ok(findText(premiumCard, "1. 記念ガチャ"));
  assert.equal(editor.dirty(), false);
});
