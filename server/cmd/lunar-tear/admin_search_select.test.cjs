const assert = require("node:assert/strict");
const { test } = require("node:test");
const path = require("node:path");
const { readFileSync } = require("node:fs");
let chromium;
try { ({ chromium } = require("playwright")); } catch (_) { /* Optional browser runtime in CI. */ }

test("shared search selector supports large catalogs, keyboard, lazy visuals and modal overlays", { skip: !chromium && "Install Playwright to run browser coverage" }, async t => {
  const browser = await chromium.launch({ headless: true, ...(process.platform === "win32" ? { channel: "msedge" } : {}) });
  t.after(() => browser.close());
  const page = await browser.newPage({ viewport: { width: 1000, height: 700 } });
  const errors = []; page.on("pageerror", error => errors.push(error.message));
  await page.setContent('<label>活动<select id="native"></select></label><button id="outside">外部</button><dialog><select id="lazy"></select><button id="done">完成</button></dialog>');
  await page.addStyleTag({ path: path.join(__dirname, "admin.css") });
  await page.addStyleTag({ content: "#outside { position: fixed; right: 20px; bottom: 20px; }" });
  await page.evaluate(() => {
    const select = document.querySelector("#native");
    for (let i = 1; i <= 5000; i++) { const option = new Option(`活动 ${i}`, String(i)); select.append(option); }
    select.value = "2500"; window.changes = 0; select.addEventListener("change", () => window.changes++);
    document.querySelector("#done").onclick = () => document.querySelector("dialog").close();
  });
  await page.addScriptTag({ path: path.join(__dirname, "admin_search_select.js") });
  await page.waitForFunction(() => document.querySelector("#native").classList.contains("search-select-native"));
  const combo = page.getByRole("combobox", { name: "活动", exact: true });
  await combo.waitFor(); assert.equal(await combo.inputValue(), "活动 2500");
  await combo.click();
  await page.getByRole("option", { name: "活动 2500", exact: true }).waitFor();
  assert.ok(await page.getByRole("option").count() <= 13);
  await page.locator(".search-select-viewport").evaluate(el => { el.scrollTop = 4200 * 38; });
  await page.getByRole("option", { name: "活动 4201", exact: true }).waitFor();
  assert.ok(await page.getByRole("option").count() <= 13);
  await combo.fill("活动 4999"); await combo.press("Enter");
  assert.equal(await page.locator("#native").inputValue(), "4999");
  assert.equal(await page.evaluate(() => window.changes), 1);
  await combo.fill("does not exist"); await page.locator("#outside").click();
  assert.equal(await combo.inputValue(), "活动 4999");
  await page.evaluate(() => { document.querySelector("#native").value = "32"; });
  assert.equal(await combo.inputValue(), "活动 32");
  await combo.click(); await page.getByRole("option", { name: "活动 32", exact: true }).waitFor();
  await combo.press("ArrowDown"); await combo.press("Enter");
  assert.equal(await page.locator("#native").inputValue(), "33");
  await combo.click(); await combo.press("Escape"); assert.equal(await combo.getAttribute("aria-expanded"), "false");
  await page.evaluate(() => { document.querySelector("#native").disabled = true; });
  await page.waitForFunction(() => document.querySelector(".search-select input").disabled);
  await page.evaluate(() => {
    window.optionLoads = 0;
    const options = Array.from({ length: 10000 }, (_, i) => ({ value: String(i), label: `奖励 ${i}`, group: "武器", searchText: `weapon-${i}` }));
    const select = document.querySelector("#lazy"); select.append(new Option("奖励 7", "7"));
    window.AdminSearchSelect.enhance(select, { ariaLabel: "奖励", options() { window.optionLoads++; return options; }, renderOption(entry) { const span = document.createElement("span"); span.className = "reward-search-option"; span.textContent = entry.label; return span; } });
    document.querySelector("dialog").showModal();
  });
  const lazy = page.getByRole("combobox", { name: "奖励", exact: true });
  await lazy.fill("weapon-9999");
  assert.equal(await page.locator("dialog .search-select-menu").count(), 1);
  const option = page.getByRole("option", { name: "奖励 9999", exact: true });
  await option.click(); assert.equal(await page.locator("#lazy").inputValue(), "9999");
  assert.equal(await page.locator("#lazy option").count(), 1, "lazy options must not inflate the native DOM");
  await page.locator("#done").click();
  await page.evaluate(() => {
    document.querySelector("#native").disabled = false;
    window.AdminSearchSelect.enhance(document.querySelector("#native"), { ariaLabel: "活动" });
    for (const id of [32, 33]) document.querySelector(`#native option[value="${id}"]`).dataset.searchGroup = "equal";
  });
  assert.equal(await page.locator("#native").evaluate(el => el.closest(".search-select").querySelectorAll('input[role="combobox"]').length), 1);
  await combo.fill("32"); await page.getByRole("option", { name: "活动 32", exact: true }).click();
  assert.equal(await page.locator("#native").inputValue(), "32", "exact ID must remain selectable within identical groups");
  assert.deepEqual(errors, []);
});

test("mission selectors are searchable before interaction and table navigation retains the common layout", { skip: !chromium && "Install Playwright to run browser coverage" }, async t => {
  const browser = await chromium.launch({ headless: true, ...(process.platform === "win32" ? { channel: "msedge" } : {}) });
  t.after(() => browser.close());
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  const errors = []; page.on("pageerror", error => errors.push(error.message));
  const html = readFileSync(path.join(__dirname, "admin.html"), "utf8").replace(/<script\b[^>]*>[\s\S]*?<\/script>/g, "");
  await page.route("http://admin.test/**", route => route.fulfill({ contentType: route.request().url().endsWith(".css") ? "text/css" : "text/html", body: route.request().url().endsWith(".css") ? readFileSync(path.join(__dirname, "admin.css"), "utf8") : html }));
  await page.goto("http://admin.test/admin/");
  await page.addScriptTag({ path: path.join(__dirname, "admin_search_select.js") });
  const main = readFileSync(path.join(__dirname, "admin.js"), "utf8").replace(/\}\)\(\);\s*$/, "globalThis.adminTest = { state, renderCatalog, renderMissionRewardAssignmentRow, renderMissionTermAssignmentRow, refreshMissionRewardAssignmentDisplays }; })();");
  await page.addScriptTag({ content: main });
  const labels = await page.evaluate(() => {
    const { state, renderCatalog } = window.adminTest;
    state.catalog = { version: "0".repeat(64), defaultLanguage: "ja", languages: ["ja"], tables: [
      { name: "m_zeta", entityName: "EntityMZeta", rowCount: 999 },
      { name: "m_quest_bonus", entityName: "EntityMQuestBonus", rowCount: 300 },
      ...Array.from({ length: 12 }, (_, index) => ({ name: `m_alpha_${index}`, entityName: `EntityMAlpha${String(11 - index).padStart(2, "0")}`, rowCount: 200 }))
    ] };
    state.section = "related"; state.language = "ja"; renderCatalog();
    document.querySelector("#workspace").classList.remove("hidden"); document.querySelector("#login-panel").classList.add("hidden");
    return [...document.querySelector("#table-select").options].filter(option => option.value).map(option => option.textContent);
  });
  await page.evaluate(() => window.AdminSearchSelect.refresh());
  assert.equal(await page.locator("#table-select").evaluate(select => select.closest(".search-select")), null);
  assert.deepEqual(labels, [...labels].sort((a, b) => a.localeCompare(b, "en", { sensitivity: "base" })));
  assert.ok(labels.includes("QuestBonus")); assert.ok(labels.every(label => !/m_|行|（/.test(label)));
  assert.equal(await page.locator("#table-select option[title]").count(), 0);
  const layout = () => page.evaluate(() => [".topbar", "main", ".summary-grid", ".data-heading", ".savebar"].map(selector => {
    const element = document.querySelector(selector), style = getComputedStyle(element);
    return [selector, style.display, style.padding, style.margin, style.fontSize, element.getBoundingClientRect().width];
  }));
  const before = await layout();
  await page.locator("#quest-bonus-editor").evaluate(element => element.classList.remove("hidden"));
  assert.deepEqual(await layout(), before, "QuestBonus must not restyle the surrounding page");
  await page.evaluate(() => {
    const api = window.adminTest;
    api.state.rewardCatalog = { consumableItems: [{ possessionId: 1, names: { ja: "回復薬" } }] };
    const rewardTable = { name: "m_mission_reward", rows: [301, 302].map((id, index) => ({ index, values: { MissionRewardId: String(id), PossessionType: "6", PossessionId: "1", Count: String(index + 1) } })) };
    const termTable = { name: "m_mission_term", rows: [41, 42].map((id, index) => ({ index, values: { MissionTermId: String(id), StartDatetime: "1000", EndDatetime: "2000" } })) };
    window.rewardTable = rewardTable;
    document.querySelector("#mission-reward-assignment-body").append(api.renderMissionRewardAssignmentRow(rewardTable, { row: 0, missionId: 1, missionRewardId: 301, names: { ja: "テスト報酬" } }));
    document.querySelector("#mission-term-assignment-body").append(api.renderMissionTermAssignmentRow(termTable, { row: 0, missionId: 1, missionTermId: 41, names: { ja: "テスト期限" } }));
    for (const id of ["#mission-reward-editor", "#mission-term-editor"]) document.querySelector(id).classList.remove("hidden");
    document.querySelector("#quest-bonus-editor").classList.add("hidden");
  });
  assert.equal(await page.locator("#mission-reward-assignment-body input[role=combobox]").count(), 1);
  assert.equal(await page.locator("#mission-term-assignment-body input[role=combobox]").count(), 1);
  assert.equal(await page.locator(".mission-reward-assignment-select option").count(), 1);
  const reward = page.getByRole("combobox", { name: "テスト報酬 的 RewardId", exact: true });
  await reward.fill("302 回復薬"); await page.getByRole("option", { name: "302. 回復薬 ×2", exact: true }).click();
  assert.equal(await page.locator(".mission-reward-assignment-select").inputValue(), "302");
  await page.evaluate(() => { window.rewardTable.rows[1].values.Count = "9"; window.adminTest.refreshMissionRewardAssignmentDisplays(window.rewardTable, "302"); });
  await page.waitForFunction(() => document.querySelector("#mission-reward-assignment-body input").value.includes("×9"));
  await reward.fill("302 回復薬"); await page.getByRole("option", { name: "302. 回復薬 ×9", exact: true }).waitFor();
  await reward.press("Escape");
  const term = page.getByRole("combobox", { name: "テスト期限 的 TermId", exact: true });
  await term.fill("42"); await term.press("Enter");
  assert.equal(await page.locator(".mission-term-assignment-select").inputValue(), "42");
  assert.deepEqual(errors, []);
});
