const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");
const vm = require("node:vm");

const source = readFileSync(path.join(__dirname, "admin.js"), "utf8");

function createEditor() {
  const element = (tagName = "div") => {
    const classes = new Set();
    return {
      tagName, value: "", textContent: "", children: [], dataset: {}, listeners: {},
      classList: {
        add: value => classes.add(value), remove: value => classes.delete(value),
        contains: value => classes.has(value),
        toggle: (value, enabled) => enabled ? classes.add(value) : classes.delete(value)
      },
      append(...children) { this.children.push(...children); },
      replaceChildren(...children) { this.children = children; },
      addEventListener(event, listener) { this.listeners[event] = listener; },
      setAttribute() {}, focus() {}, showModal() {}
    };
  };
  const context = vm.createContext({
    document: { querySelector: () => element(), querySelectorAll: () => [], createElement: element },
    window: {
      addEventListener() {}, setTimeout() {}, clearTimeout() {},
      AdminSearchSelect: { enhance: select => ({ querySelector: () => select }) }
    },
    sessionStorage: { getItem: () => null }, localStorage: { getItem: () => null },
    fetch: async (url, options) => {
      assert.equal(url, "/api/admin/master-data/schedules/preview");
      return { ok: true, json: async () => JSON.parse(options.body) };
    }
  });
  vm.runInContext(source.replace(/\}\)\(\);\s*$/, `
    // Exercise CellGroup rendering, events and saving without unrelated panels.
    renderShopCellPanel = renderShopItemPanel = updateDirtyUI = setBusy = () => {};
    renderMasterUpdatePreview = () => {};
    renderTable = () => renderShopEditor({ rows: [] }, elements.search.value);
    globalThis.editor = { state, elements, renderTable, resetShopCellGroupDraft };
  })();`), context);
  const editor = context.editor;
  editor.state.catalog = {
    version: "test-version", tables: [],
    shopEditor: {
      shops: [], items: [], cells: [],
      cellGroups: [
        { shopItemCellGroupId: 1, shopItemCellId: 101, sortOrder: 10, shopItemCellTermId: 8 },
        { shopItemCellGroupId: 2, shopItemCellId: 201, sortOrder: 0, shopItemCellTermId: 9 },
        { shopItemCellGroupId: 1, shopItemCellId: 102, sortOrder: 2, shopItemCellTermId: 8 },
        { shopItemCellGroupId: 1, shopItemCellId: 103, sortOrder: 2, shopItemCellTermId: 8 }
      ]
    }
  };
  editor.resetShopCellGroupDraft();
  editor.renderTable();
  return editor;
}

const cards = editor => editor.elements.shopCellGroupBody.children;
const cellIDs = editor => cards(editor).map(card => Number(card.children[0].children[1].textContent));
const sortInput = card => card.children[3].children[0].children.find(child => child.tagName === "input");

test("CellGroup sorts numerically and stably without changing the draft or dirty state", () => {
  const editor = createEditor();
  assert.deepEqual(cellIDs(editor), [102, 103, 101]);
  assert.deepEqual(Array.from(editor.state.shopCellGroupDraft, row => row.shopItemCellId), [101, 201, 102, 103]);
  assert.equal(editor.state.shopCellGroupDirty, false);
  editor.elements.search.value = "103";
  editor.renderTable();
  assert.deepEqual(cellIDs(editor), [103]);
});

test("editing SortOrder reorders the correct row and includes it in the save request", async () => {
  const editor = createEditor();
  const input = sortInput(cards(editor)[0]);
  assert.ok(input, "CellGroup must expose a SortOrder input");
  input.value = "20";
  input.listeners.change();
  assert.deepEqual(cellIDs(editor), [103, 101, 102]);
  assert.equal(editor.state.shopCellGroupDraft[2].sortOrder, 20);
  assert.equal(editor.state.shopCellGroupDraft[0].sortOrder, 10);
  assert.equal(editor.state.shopCellGroupDirty, true);
  await editor.elements.save.listeners.click();
  const payload = JSON.parse(JSON.stringify(editor.state.pendingMasterChanges.shopItemCellGroups));
  assert.deepEqual(payload[2], { shopItemCellGroupId: 1, shopItemCellId: 102, sortOrder: 20, shopItemCellTermId: 8 });
  assert.equal(payload[1].shopItemCellGroupId, 2);

  const restored = sortInput(cards(editor)[2]);
  restored.value = "2";
  restored.listeners.change();
  assert.equal(editor.state.shopCellGroupDirty, false);
  assert.deepEqual(cellIDs(editor), [102, 103, 101]);
});

test("removing a sorted, filtered CellGroup card removes its original draft row", () => {
  const editor = createEditor();
  editor.elements.search.value = "103";
  editor.renderTable();
  cards(editor)[0].children.at(-1).listeners.click();
  assert.deepEqual(Array.from(editor.state.shopCellGroupDraft, row => row.shopItemCellId), [101, 201, 102]);
});

test("invalid SortOrder values leave the draft unchanged and valid integers recover", () => {
  const editor = createEditor();
  const input = sortInput(cards(editor)[0]);
  assert.ok(input, "CellGroup must expose a SortOrder input");
  for (const value of ["", "1.5", "abc", "2147483648", "-2147483649"]) {
    input.value = value;
    input.listeners.change();
    assert.equal(input.classList.contains("invalid"), true, value);
    assert.equal(editor.state.shopCellGroupDraft[2].sortOrder, 2);
    assert.equal(editor.state.shopCellGroupDirty, false);
  }
  input.value = "0";
  input.listeners.change();
  assert.equal(input.classList.contains("invalid"), false);
  assert.equal(editor.state.shopCellGroupDraft[2].sortOrder, 0);
  assert.equal(editor.state.shopCellGroupDirty, true);
});
