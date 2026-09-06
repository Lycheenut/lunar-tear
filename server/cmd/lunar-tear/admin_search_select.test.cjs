const assert = require("node:assert/strict");
const { test } = require("node:test");
const path = require("node:path");
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
