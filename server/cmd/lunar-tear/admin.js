(() => {
  "use strict";

  const $ = (selector) => document.querySelector(selector);
  const elements = {
    loginPanel: $("#login-panel"), loginForm: $("#login-form"), token: $("#token"),
    workspace: $("#workspace"), logout: $("#logout"), version: $("#version"),
    tableCount: $("#table-count"), rowCount: $("#row-count"), dirtyCount: $("#dirty-count"),
    timezone: $("#timezone"), tableSelect: $("#table-select"), typeFilters: $("#type-filters"), statusFilter: $("#status-filter"),
    languageSelect: $("#language-select"), search: $("#search"), refresh: $("#refresh"), notice: $("#notice"),
    entityName: $("#entity-name"), tableName: $("#table-name"), visibleCount: $("#visible-count"),
    tableScroll: $("#table-scroll"), head: $("#schedule-head"), body: $("#schedule-body"),
    grid: $("#schedule-grid"), viewButtons: document.querySelectorAll(".view-button"), empty: $("#empty-state"),
    saveSummary: $("#save-summary"), discard: $("#discard"), save: $("#save")
  };

  const state = {
    token: sessionStorage.getItem("lunar-admin-token") || "",
    language: localStorage.getItem("lunar-admin-language") || "en",
    view: localStorage.getItem("lunar-admin-view") === "grid" ? "grid" : "list",
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
      option.textContent = `${tableDisplayName(table)} · ${table.rows.length}`;
      option.title = table.name;
      elements.tableSelect.append(option);
    });
    if (state.catalog.tables.some((table) => table.name === previous)) elements.tableSelect.value = previous;
    renderLanguages();
    renderTypeFilters(currentTable());
    elements.version.textContent = `版本 ${state.catalog.version.slice(0, 12)}`;
    elements.version.title = state.catalog.version;
    elements.tableCount.textContent = state.catalog.tableCount.toLocaleString();
    elements.rowCount.textContent = state.catalog.rowCount.toLocaleString();
    elements.timezone.textContent = "UTC";
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

  function renderTypeFilters(table) {
    const previous = new Map([...elements.typeFilters.querySelectorAll("select")].map((select) => [select.dataset.field, select.value]));
    elements.typeFilters.replaceChildren();
    const fields = [...new Set((table?.rows || []).flatMap((row) => row.identity.map((field) => field.name)).filter((name) => name.endsWith("Type")))];
    fields.forEach((fieldName) => {
      const label = document.createElement("label");
      label.textContent = `类型 · ${fieldName}`;
      const select = document.createElement("select");
      select.dataset.field = fieldName;
      const all = document.createElement("option");
      all.value = "";
      all.textContent = "全部";
      select.append(all);
      const values = [...new Set(table.rows.map((row) => row.identity.find((field) => field.name === fieldName)?.value).filter((value) => value !== undefined))];
      values.sort(compareFieldValues);
      values.forEach((value) => {
        const option = document.createElement("option");
        option.value = value;
        option.textContent = typeOptionLabel(table.name, fieldName, value);
        select.append(option);
      });
      if (values.includes(previous.get(fieldName))) select.value = previous.get(fieldName);
      select.addEventListener("change", renderTable);
      label.append(select);
      elements.typeFilters.append(label);
    });
    elements.typeFilters.classList.toggle("hidden", fields.length === 0);
  }

  function compareFieldValues(left, right) {
    const leftNumber = Number(left);
    const rightNumber = Number(right);
    if (Number.isFinite(leftNumber) && Number.isFinite(rightNumber)) return leftNumber - rightNumber;
    return left.localeCompare(right);
  }

  function typeOptionLabel(tableName, fieldName, value) {
    return value;
  }

  function renderTable() {
    const table = currentTable();
    if (!table) return;
    elements.entityName.textContent = table.name;
    elements.tableName.textContent = tableDisplayName(table);
    elements.head.replaceChildren();
    elements.body.replaceChildren();
    elements.grid.replaceChildren();

    const query = elements.search.value.trim().toLocaleLowerCase();
    const statusFilter = elements.statusFilter.value;
    const typeFilters = [...elements.typeFilters.querySelectorAll("select")]
      .filter((select) => select.value !== "")
      .map((select) => ({ field: select.dataset.field, value: select.value }));
    const visibleRows = table.rows.filter((row) => {
      const status = rowStatus(table, row);
      if (statusFilter !== "all" && status !== statusFilter) return false;
      if (typeFilters.some((filter) => row.identity.find((field) => field.name === filter.field)?.value !== filter.value)) return false;
      if (!query) return true;
      const relationValues = (row.shopRelations || []).flatMap((relation) => [
        relation.shopId, relation.shopItemCellGroupId, ...relation.shopItemCellIds,
        ...Object.values(relation.shopTitles || {})
      ]);
      const haystack = [...Object.values(row.titles || {}), ...relationValues, ...row.identity.flatMap((field) => [field.name, field.value])].join(" ").toLocaleLowerCase();
      return haystack.includes(query);
    });

    const isGrid = state.view === "grid";
    elements.tableScroll.classList.toggle("hidden", isGrid);
    elements.grid.classList.toggle("hidden", !isGrid);
    syncViewToggle();
    if (isGrid) {
      visibleRows.forEach((row) => elements.grid.append(renderCard(table, row)));
    } else {
      const headerRow = document.createElement("tr");
      ["ID", "内容", "状态", "备注"].forEach((label) => headerRow.append(makeCell("th", label)));
      table.timeFields.forEach((field) => headerRow.append(makeCell("th", field)));
      elements.head.append(headerRow);
      visibleRows.forEach((row) => elements.body.append(renderRow(table, row)));
    }
    elements.visibleCount.textContent = `${visibleRows.length.toLocaleString()} 行`;
    elements.empty.classList.toggle("hidden", visibleRows.length !== 0);
  }

  function renderRow(table, row) {
    const tr = document.createElement("tr");
    const primary = row.identity[0];
    const localizedTitle = localizedText(row.titles);

    const idCell = makeCell("td", primary?.value || "-");
    idCell.className = "id-cell";
    idCell.title = primary?.name || "ID";
    tr.append(idCell);

    const contentCell = makeCell("td", localizedTitle || "-");
    contentCell.className = "content-cell";
    tr.append(contentCell);

    const statusCell = document.createElement("td");
    statusCell.className = "status-cell";
    statusCell.append(renderStatus(rowStatus(table, row)));
    tr.append(statusCell);

    const notesCell = document.createElement("td");
    notesCell.className = "notes-cell";
    notesCell.append(renderNotes(table, row));
    tr.append(notesCell);

    table.timeFields.forEach((field) => {
      const td = document.createElement("td");
      td.className = "time-column";
      td.append(renderTimeEditor(table, row, field));
      tr.append(td);
    });
    return tr;
  }

  function renderCard(table, row) {
    const card = document.createElement("article");
    card.className = "schedule-card";
    const primary = row.identity[0];
    const localizedTitle = localizedText(row.titles);

    const header = document.createElement("div");
    header.className = "card-header";
    const id = document.createElement("div");
    id.className = "card-id";
    const idLabel = document.createElement("span");
    idLabel.textContent = "ID";
    const idValue = document.createElement("strong");
    idValue.textContent = primary?.value || "-";
    idValue.title = primary?.name || "ID";
    id.append(idLabel, idValue);
    header.append(id, renderStatus(rowStatus(table, row)));

    const title = document.createElement("h3");
    title.textContent = localizedTitle || "-";
    card.append(header, title);

    const notesSection = document.createElement("section");
    notesSection.className = "card-section";
    const notesLabel = document.createElement("span");
    notesLabel.className = "card-label";
    notesLabel.textContent = "备注";
    notesSection.append(notesLabel, renderNotes(table, row));
    card.append(notesSection);

    const times = document.createElement("div");
    times.className = "card-times";
    table.timeFields.forEach((field) => {
      const group = document.createElement("label");
      group.textContent = field;
      group.append(renderTimeEditor(table, row, field));
      times.append(group);
    });
    card.append(times);
    return card;
  }

  function renderStatus(status) {
    const element = document.createElement("span");
    element.className = `status ${status}`;
    element.textContent = statusLabels[status];
    return element;
  }

  function renderNotes(table, row) {
    const notes = document.createElement("div");
    notes.className = "identity";
    row.identity.slice(1).forEach((field) => {
      if (table.name === "m_shop" && field.name === "ShopItemCellGroupId") return;
      if (field.name === "QuestScheduleCronExpression") {
        notes.append(renderCronExpression(field.value));
        return;
      }
      const meta = document.createElement("div");
      meta.className = "identity-meta";
      meta.textContent = `${field.name}=${displayText(field.value)}`;
      notes.append(meta);
    });
    if (table.name === "m_shop_item_cell_term") {
      const shopNames = renderShopNames(row.shopRelations);
      if (shopNames) notes.append(shopNames);
    }
    if (!notes.childElementCount) {
      const empty = document.createElement("span");
      empty.className = "notes-empty";
      empty.textContent = "-";
      notes.append(empty);
    }
    return notes;
  }

  function renderCronExpression(expression) {
    const wrapper = document.createElement("div");
    wrapper.className = "cron-expression";
    const readable = document.createElement("span");
    readable.className = "cron-readable";
    readable.textContent = humanizeCron(expression) || "自定义计划";
    const raw = document.createElement("code");
    raw.className = "cron-raw";
    raw.textContent = expression;
    wrapper.append(readable, raw);
    return wrapper;
  }

  function humanizeCron(expression) {
    const parts = expression.trim().split(/\s+/);
    if (parts.length !== 6) return "";
    const [seconds, minutes, hours, days, months, weekdays] = parts;
    if (seconds !== "*" || !["*", "0-59"].includes(minutes) || !["*", "1-31"].includes(days) || months !== "*") return "";

    const weekdayNames = { SUN: "日", MON: "一", TUE: "二", WED: "三", THU: "四", FRI: "五", SAT: "六" };
    let dayLabel = "每天";
    if (weekdays !== "*") {
      const names = weekdays.split(",").map((day) => weekdayNames[day]);
      if (names.some((name) => !name)) return "";
      dayLabel = names.length === 1 ? `每周${names[0]}` : `每周${names.join("、")}`;
    }

    let timeLabel = "全天";
    if (hours !== "*") {
      const match = /^(\d{1,2})(?:-(\d{1,2}))?$/.exec(hours);
      if (!match) return "";
      const start = Number(match[1]);
      const end = Number(match[2] ?? match[1]);
      if (start < 0 || end > 23 || start > end || minutes !== "0-59") return "";
      timeLabel = `${String(start).padStart(2, "0")}:00–${String(end).padStart(2, "0")}:59`;
    }
    return `${dayLabel} · ${timeLabel}（UTC）`;
  }

  function renderTimeEditor(table, row, field) {
    const wrapper = document.createElement("div");
    wrapper.className = "time-cell";
    const input = document.createElement("input");
    input.type = "datetime-local";
    input.step = "1";
    input.value = utcInputValue(effectiveValue(table.name, row, field));
    input.dataset.table = table.name;
    input.dataset.row = String(row.index);
    input.dataset.field = field;
    input.setAttribute("aria-label", `${field} UTC`);
    input.classList.toggle("changed", state.dirty.has(changeKey(table.name, row.index, field)));
    input.addEventListener("change", () => onTimeChange(table, row, field, input));
    wrapper.append(input);
    return wrapper;
  }

  function renderShopNames(relations = []) {
    const names = [...new Set(relations.map((relation) => localizedText(relation.shopTitles)).filter(Boolean))];
    if (!names.length) return null;
    const wrapper = document.createElement("div");
    wrapper.className = "shop-names";
    names.forEach((name) => {
      const item = document.createElement("div");
      item.className = "shop-name";
      item.textContent = name;
      wrapper.append(item);
    });
    return wrapper;
  }

  function localizedText(titles) {
    const text = titles?.[state.language] || titles?.[state.catalog.defaultLanguage] || Object.values(titles || {})[0] || "";
    return displayText(text);
  }

  function displayText(value) {
    return String(value ?? "").replace(/\\n/g, "\n");
  }

  function tableDisplayName(table) {
    if (table.entityName?.startsWith("EntityM")) return table.entityName.slice("EntityM".length);
    return table.name.replace(/^m_/, "").split("_").filter(Boolean)
      .map((part) => part.charAt(0).toUpperCase() + part.slice(1)).join("");
  }

  function onTimeChange(table, row, field, input) {
    const value = input.value ? Date.parse(`${input.value}Z`) : 0;
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

  function utcInputValue(milliseconds) {
    if (milliseconds === 0) return "";
    const date = new Date(milliseconds);
    if (Number.isNaN(date.getTime())) return "";
    return date.toISOString().slice(0, 19);
  }

  function syncViewToggle() {
    elements.viewButtons.forEach((button) => {
      const active = button.dataset.view === state.view;
      button.classList.toggle("active", active);
      button.setAttribute("aria-pressed", String(active));
    });
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

  elements.tableSelect.addEventListener("change", () => {
    renderTypeFilters(currentTable());
    renderTable();
  });
  elements.statusFilter.addEventListener("change", renderTable);
  elements.languageSelect.addEventListener("change", () => {
    state.language = elements.languageSelect.value;
    localStorage.setItem("lunar-admin-language", state.language);
    renderTable();
  });
  elements.search.addEventListener("input", renderTable);
  elements.viewButtons.forEach((button) => button.addEventListener("click", () => {
    state.view = button.dataset.view;
    localStorage.setItem("lunar-admin-view", state.view);
    renderTable();
  }));
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
