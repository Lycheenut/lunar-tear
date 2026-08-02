(() => {
  "use strict";

  const $ = (selector) => document.querySelector(selector);
  const elements = {
    loginPanel: $("#login-panel"), loginForm: $("#login-form"), token: $("#token"),
    workspace: $("#workspace"), logout: $("#logout"), version: $("#version"),
    tableCount: $("#table-count"), rowCount: $("#row-count"), dirtyCount: $("#dirty-count"),
    timezone: $("#timezone"), tableSelect: $("#table-select"), statusFilter: $("#status-filter"),
    languageSelect: $("#language-select"), search: $("#search"), refresh: $("#refresh"), notice: $("#notice"),
    entityName: $("#entity-name"), tableName: $("#table-name"), visibleCount: $("#visible-count"),
    head: $("#schedule-head"), body: $("#schedule-body"), empty: $("#empty-state"),
    saveSummary: $("#save-summary"), discard: $("#discard"), save: $("#save")
  };

  const state = {
    token: sessionStorage.getItem("lunar-admin-token") || "",
    language: localStorage.getItem("lunar-admin-language") || "en",
    catalog: null,
    dirty: new Map()
  };
  const statusLabels = { active: "进行中", upcoming: "未开始", expired: "已结束", disabled: "已禁用" };
  const languageLabels = { en: "English", ja: "日本語", ko: "한국어" };

  async function api(path, options = {}) {
    const response = await fetch(path, {
      ...options,
      headers: {
        "Authorization": `Bearer ${state.token}`,
        ...(options.body ? { "Content-Type": "application/json" } : {}),
        ...(options.headers || {})
      }
    });
    let payload;
    try { payload = await response.json(); } catch (_) { payload = {}; }
    if (!response.ok) {
      const error = new Error(payload.error || `请求失败（HTTP ${response.status}）`);
      error.status = response.status;
      throw error;
    }
    return payload;
  }

  async function loadCatalog() {
    setBusy(true, "正在读取主数据…");
    try {
      state.catalog = await api("/api/admin/master-data/schedules");
      state.dirty.clear();
      sessionStorage.setItem("lunar-admin-token", state.token);
      showWorkspace();
      renderCatalog();
      showNotice(`已读取 ${state.catalog.tableCount} 张限时表、${state.catalog.rowCount} 行内容。`);
    } catch (error) {
      if (error.status === 401) {
        state.token = "";
        sessionStorage.removeItem("lunar-admin-token");
        showLogin();
      }
      showNotice(error.message, true);
      throw error;
    } finally {
      setBusy(false);
    }
  }

  function showWorkspace() {
    elements.loginPanel.classList.add("hidden");
    elements.workspace.classList.remove("hidden");
    elements.logout.classList.remove("hidden");
  }

  function showLogin() {
    elements.workspace.classList.add("hidden");
    elements.loginPanel.classList.remove("hidden");
    elements.logout.classList.add("hidden");
    elements.version.textContent = "尚未连接";
    elements.token.value = "";
    elements.token.focus();
  }

  function renderCatalog() {
    const previous = elements.tableSelect.value;
    elements.tableSelect.replaceChildren();
    state.catalog.tables.forEach((table) => {
      const option = document.createElement("option");
      option.value = table.name;
      option.textContent = `${table.name} · ${table.rows.length}`;
      elements.tableSelect.append(option);
    });
    if (state.catalog.tables.some((table) => table.name === previous)) elements.tableSelect.value = previous;
    renderLanguages();
    elements.version.textContent = `版本 ${state.catalog.version.slice(0, 12)}`;
    elements.version.title = state.catalog.version;
    elements.tableCount.textContent = state.catalog.tableCount.toLocaleString();
    elements.rowCount.textContent = state.catalog.rowCount.toLocaleString();
    elements.timezone.textContent = Intl.DateTimeFormat().resolvedOptions().timeZone || "浏览器本地时区";
    updateDirtyUI();
    renderTable();
  }

  function renderLanguages() {
    const languages = state.catalog.languages?.length ? state.catalog.languages : [state.catalog.defaultLanguage || "en"];
    if (!languages.includes(state.language)) state.language = state.catalog.defaultLanguage || "en";
    elements.languageSelect.replaceChildren();
    languages.forEach((language) => {
      const option = document.createElement("option");
      option.value = language;
      option.textContent = languageLabels[language] || language;
      elements.languageSelect.append(option);
    });
    elements.languageSelect.value = state.language;
  }

  function currentTable() {
    return state.catalog?.tables.find((table) => table.name === elements.tableSelect.value) || state.catalog?.tables[0];
  }

  function renderTable() {
    const table = currentTable();
    if (!table) return;
    elements.entityName.textContent = table.entityName;
    elements.tableName.textContent = table.name;
    elements.head.replaceChildren();
    elements.body.replaceChildren();

    const headerRow = document.createElement("tr");
    headerRow.append(makeCell("th", "内容 / 状态"));
    table.timeFields.forEach((field) => headerRow.append(makeCell("th", field)));
    elements.head.append(headerRow);

    const query = elements.search.value.trim().toLocaleLowerCase();
    const statusFilter = elements.statusFilter.value;
    const visibleRows = table.rows.filter((row) => {
      const status = rowStatus(table, row);
      if (statusFilter !== "all" && status !== statusFilter) return false;
      if (!query) return true;
      const relationValues = (row.shopRelations || []).flatMap((relation) => [
        relation.shopId, relation.shopItemCellGroupId, ...relation.shopItemCellIds,
        ...Object.values(relation.shopTitles || {})
      ]);
      const haystack = [row.index, ...Object.values(row.titles || {}), ...relationValues, ...row.identity.flatMap((field) => [field.name, field.value])].join(" ").toLocaleLowerCase();
      return haystack.includes(query);
    });

    visibleRows.forEach((row) => elements.body.append(renderRow(table, row)));
    elements.visibleCount.textContent = `${visibleRows.length.toLocaleString()} 行`;
    elements.empty.classList.toggle("hidden", visibleRows.length !== 0);
  }

  function renderRow(table, row) {
    const tr = document.createElement("tr");
    const identityCell = document.createElement("td");
    const identity = document.createElement("div");
    identity.className = "identity";
    const title = document.createElement("div");
    title.className = "identity-main";
    const primary = row.identity[0];
    const localizedTitle = localizedText(row.titles);
    title.textContent = localizedTitle || (primary ? `${primary.name} = ${primary.value}` : `Row ${row.index}`);
    const status = rowStatus(table, row);
    const statusEl = document.createElement("span");
    statusEl.className = `status ${status}`;
    statusEl.textContent = statusLabels[status];
    title.append(statusEl);
    identity.append(title);
    row.identity.slice(localizedTitle ? 0 : 1).forEach((field) => {
      const meta = document.createElement("div");
      meta.className = "identity-meta";
      meta.textContent = `${field.name}=${field.value}`;
      identity.append(meta);
    });
    if (row.shopRelations?.length) {
      const relations = document.createElement("div");
      relations.className = "shop-relations";
      row.shopRelations.forEach((relation) => relations.append(renderShopRelation(relation, localizedTitle)));
      identity.append(relations);
    }
    const index = document.createElement("div");
    index.className = "identity-meta";
    index.textContent = `row ${row.index}`;
    identity.append(index);
    identityCell.append(identity);
    tr.append(identityCell);

    table.timeFields.forEach((field) => {
      const td = document.createElement("td");
      const wrapper = document.createElement("div");
      wrapper.className = "time-cell";
      const input = document.createElement("input");
      input.type = "datetime-local";
      input.step = "1";
      input.value = localInputValue(effectiveValue(table.name, row, field));
      input.dataset.table = table.name;
      input.dataset.row = String(row.index);
      input.dataset.field = field;
      input.classList.toggle("changed", state.dirty.has(changeKey(table.name, row.index, field)));
      const exact = document.createElement("span");
      exact.className = "iso-time";
      exact.textContent = exactTime(effectiveValue(table.name, row, field));
      input.addEventListener("change", () => onTimeChange(table, row, field, input, exact));
      wrapper.append(input, exact);
      td.append(wrapper);
      tr.append(td);
    });
    return tr;
  }

  function renderShopRelation(relation, rowTitle) {
    const wrapper = document.createElement("div");
    wrapper.className = "shop-relation";
    const path = document.createElement("div");
    path.className = "relation-path";
    path.textContent = `SHOP ${relation.shopId} → CELL GROUP ${relation.shopItemCellGroupId}`;
    wrapper.append(path);
    const shopTitle = localizedText(relation.shopTitles);
    if (shopTitle && shopTitle !== rowTitle) {
      const title = document.createElement("div");
      title.className = "relation-title";
      title.textContent = shopTitle;
      wrapper.append(title);
    }
    const cells = document.createElement("div");
    cells.className = "relation-cells";
    cells.textContent = `CELLS ${summarizeIds(relation.shopItemCellIds)}`;
    cells.title = relation.shopItemCellIds.join(", ");
    wrapper.append(cells);
    return wrapper;
  }

  function localizedText(titles) {
    return titles?.[state.language] || titles?.[state.catalog.defaultLanguage] || Object.values(titles || {})[0] || "";
  }

  function summarizeIds(ids, limit = 8) {
    const shown = ids.slice(0, limit).join(", ");
    return ids.length > limit ? `${shown} · +${ids.length - limit}` : shown;
  }

  function onTimeChange(table, row, field, input, exact) {
    const value = input.value ? new Date(input.value).getTime() : 0;
    if (!Number.isSafeInteger(value) || value < 0) {
      input.classList.add("invalid");
      showNotice("请输入有效的日期时间。", true);
      return;
    }
    input.classList.remove("invalid");
    const key = changeKey(table.name, row.index, field);
    if (value === row.times[field]) state.dirty.delete(key);
    else state.dirty.set(key, { table: table.name, row: row.index, field, value });
    input.classList.toggle("changed", state.dirty.has(key));
    exact.textContent = exactTime(value);
    updateDirtyUI();
  }

  function rowStatus(table, row) {
    const pair = table.pairs[0];
    if (!pair) return "expired";
    const start = effectiveValue(table.name, row, pair.start);
    const end = effectiveValue(table.name, row, pair.end);
    const now = Date.now();
    if (end === 0) return "disabled";
    if (now < start) return "upcoming";
    if (now <= end) return "active";
    return "expired";
  }

  function effectiveValue(table, row, field) {
    return state.dirty.get(changeKey(table, row.index, field))?.value ?? row.times[field];
  }

  function changeKey(table, row, field) { return `${table}\u0000${row}\u0000${field}`; }

  function localInputValue(milliseconds) {
    if (milliseconds === 0) return "";
    const date = new Date(milliseconds);
    if (Number.isNaN(date.getTime())) return "";
    const part = (value) => String(value).padStart(2, "0");
    return `${date.getFullYear()}-${part(date.getMonth() + 1)}-${part(date.getDate())}T${part(date.getHours())}:${part(date.getMinutes())}:${part(date.getSeconds())}`;
  }

  function exactTime(milliseconds) {
    if (milliseconds === 0) return "0 · 禁用";
    try { return `${milliseconds} · ${new Date(milliseconds).toISOString()}`; }
    catch (_) { return String(milliseconds); }
  }

  function updateDirtyUI() {
    const count = state.dirty.size;
    elements.dirtyCount.textContent = count.toLocaleString();
    elements.saveSummary.textContent = count ? `${count} 个时间值等待应用` : "没有待应用的修改";
    elements.save.disabled = count === 0;
    elements.discard.disabled = count === 0;
  }

  function makeCell(tag, text) {
    const cell = document.createElement(tag);
    cell.textContent = text;
    return cell;
  }

  function showNotice(message, error = false) {
    elements.notice.textContent = message;
    elements.notice.classList.remove("hidden");
    elements.notice.classList.toggle("error", error);
  }

  function setBusy(busy, message = "") {
    elements.save.disabled = busy || state.dirty.size === 0;
    elements.refresh.disabled = busy;
    if (message) showNotice(message);
  }

  elements.loginForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    state.token = elements.token.value;
    try { await loadCatalog(); } catch (_) { /* notice is already shown */ }
  });

  elements.logout.addEventListener("click", () => {
    state.token = "";
    state.catalog = null;
    state.dirty.clear();
    sessionStorage.removeItem("lunar-admin-token");
    showLogin();
  });

  elements.tableSelect.addEventListener("change", renderTable);
  elements.statusFilter.addEventListener("change", renderTable);
  elements.languageSelect.addEventListener("change", () => {
    state.language = elements.languageSelect.value;
    localStorage.setItem("lunar-admin-language", state.language);
    renderTable();
  });
  elements.search.addEventListener("input", renderTable);
  elements.refresh.addEventListener("click", async () => {
    if (state.dirty.size && !confirm("刷新会放弃尚未应用的修改，是否继续？")) return;
    try { await loadCatalog(); } catch (_) { /* notice is already shown */ }
  });
  elements.discard.addEventListener("click", () => {
    if (!confirm("放弃全部尚未应用的修改？")) return;
    state.dirty.clear();
    updateDirtyUI();
    renderTable();
    showNotice("已放弃本次修改。");
  });
  elements.save.addEventListener("click", async () => {
    const changes = [...state.dirty.values()];
    if (!changes.length) return;
    if (!confirm(`将 ${changes.length} 个修改重建为新的主数据并立即更新当前服务。是否继续？`)) return;
    setBusy(true, "正在重建、验证并热更新主数据…");
    try {
      const result = await api("/api/admin/master-data/schedules", {
        method: "POST",
        body: JSON.stringify({ expectedVersion: state.catalog.version, changes })
      });
      await loadCatalog();
      showNotice(`应用成功：更新 ${result.changedRows} 行、${result.changedCells} 个时间值。`);
    } catch (error) {
      showNotice(error.message, true);
      if (error.status === 409) {
        state.dirty.clear();
        try { await loadCatalog(); } catch (_) { /* keep the conflict notice */ }
      }
    } finally {
      setBusy(false);
      updateDirtyUI();
    }
  });

  window.addEventListener("beforeunload", (event) => {
    if (!state.dirty.size) return;
    event.preventDefault();
    event.returnValue = "";
  });

  if (state.token) loadCatalog().catch(() => showLogin());
  else showLogin();
})();
