(() => {
  "use strict";

  const $ = (selector) => document.querySelector(selector);
  const elements = {
    loginPanel: $("#login-panel"), loginForm: $("#login-form"), token: $("#token"),
    workspace: $("#workspace"), logout: $("#logout"), version: $("#version"),
    tableCount: $("#table-count"), rowCount: $("#row-count"), dirtyCount: $("#dirty-count"),
    timezone: $("#timezone"), tableSelect: $("#table-select"), typeFilters: $("#type-filters"),
    statusFilter: $("#status-filter"), statusFilterLabel: $("#status-filter-label"),
    languageSelect: $("#language-select"), search: $("#search"), refresh: $("#refresh"), notice: $("#notice"),
    entityName: $("#entity-name"), tableName: $("#table-name"), visibleCount: $("#visible-count"),
    tableScroll: $("#table-scroll"), head: $("#schedule-head"), body: $("#schedule-body"),
    grid: $("#schedule-grid"), viewButtons: document.querySelectorAll(".view-button"), empty: $("#empty-state"),
    saveSummary: $("#save-summary"), discard: $("#discard"), save: $("#save"),
    tabMaster: $("#tab-master"), tabGacha: $("#tab-gacha"), gachaEditor: $("#gacha-editor"),
    gachaStandardCount: $("#gacha-standard-count"), gachaOverrideCount: $("#gacha-override-count"),
    gachaBannerCount: $("#gacha-banner-count"), gachaPickupCount: $("#gacha-pickup-count"), gachaWarnings: $("#gacha-warnings"),
    gachaLanguageSelect: $("#gacha-language-select"), gachaLimitedSetId: $("#gacha-limited-set-id"),
    gachaLimitedSetName: $("#gacha-limited-set-name"), gachaAddLimitedSet: $("#gacha-add-limited-set"),
    gachaLimitedSets: $("#gacha-limited-sets"), gachaWeaponSearch: $("#gacha-weapon-search"),
    gachaAvailabilityFilter: $("#gacha-availability-filter"), gachaStarFilter: $("#gacha-star-filter"),
    gachaAttributeFilter: $("#gacha-attribute-filter"),
    gachaWeaponTypeFilter: $("#gacha-weapon-type-filter"), gachaGrantFilter: $("#gacha-grant-filter"),
    gachaWeaponBody: $("#gacha-weapon-body"), gachaWeaponEmpty: $("#gacha-weapon-empty"),
    gachaBannerSelect: $("#gacha-banner-select"), gachaBannerState: $("#gacha-banner-state"),
    gachaBannerLimitedSets: $("#gacha-banner-limited-sets"), gachaPickupSearch: $("#gacha-pickup-search"),
    gachaPickupStarFilter: $("#gacha-pickup-star-filter"), gachaPickupAttributeFilter: $("#gacha-pickup-attribute-filter"),
    gachaPickupWeaponTypeFilter: $("#gacha-pickup-weapon-type-filter"), gachaPickupGrantFilter: $("#gacha-pickup-grant-filter"),
    gachaPickupBody: $("#gacha-pickup-body"),
    gachaSaveSummary: $("#gacha-save-summary"), gachaDiscard: $("#gacha-discard"), gachaSave: $("#gacha-save"),
    gachaPublishDialog: $("#gacha-publish-dialog"), gachaPublishCancel: $("#gacha-publish-cancel"),
    gachaPublishConfirm: $("#gacha-publish-confirm")
  };

  const state = {
    token: sessionStorage.getItem("lunar-admin-token") || "",
    language: localStorage.getItem("lunar-admin-language") || "en",
    view: localStorage.getItem("lunar-admin-view") === "grid" ? "grid" : "list",
    timeMode: "local",
    catalog: null,
    dirty: new Map(),
    section: "master",
    gachaCatalog: null,
    gachaDraft: null,
    gachaDirty: false
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
      const [catalog, gachaCatalog] = await Promise.all([
        api("/api/admin/master-data/schedules"),
        api("/api/admin/gacha-config")
      ]);
      state.catalog = catalog;
      state.gachaCatalog = gachaCatalog;
      resetGachaDraft();
      state.dirty.clear();
      sessionStorage.setItem("lunar-admin-token", state.token);
      showWorkspace();
      renderCatalog();
      renderGachaEditor();
      switchAdminSection(state.section);
      showNotice(`已读取 ${state.catalog.primaryCount} 张限时主表、${state.catalog.relatedCount} 张直接关联表、${state.catalog.rowCount} 行内容，以及 ${state.gachaCatalog.weapons.length} 件 Gacha 武器。`);
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
    [[true, "限时主表"], [false, "直接关联表"]].forEach(([primary, label]) => {
      const group = document.createElement("optgroup");
      group.label = label;
      state.catalog.tables.filter((table) => table.primary === primary).forEach((table) => {
        const option = document.createElement("option");
        option.value = table.name;
        option.textContent = `${tableDisplayName(table)} · ${table.rows.length}`;
        option.title = table.name;
        group.append(option);
      });
      elements.tableSelect.append(group);
    });
    if (state.catalog.tables.some((table) => table.name === previous)) elements.tableSelect.value = previous;
    renderLanguages();
    renderTypeFilters(currentTable());
    elements.version.textContent = `版本 ${state.catalog.version.slice(0, 12)}`;
    elements.version.title = state.catalog.version;
    elements.tableCount.textContent = `${state.catalog.primaryCount} + ${state.catalog.relatedCount}`;
    elements.tableCount.title = `${state.catalog.primaryCount} 张限时主表，${state.catalog.relatedCount} 张直接关联表`;
    elements.rowCount.textContent = state.catalog.rowCount.toLocaleString();
    elements.timezone.value = state.timeMode;
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
    const fields = (table?.fields || []).filter((field) => field.type.endsWith("Type"));
    fields.forEach((field) => {
      const label = document.createElement("label");
      label.textContent = `类型 · ${field.name}`;
      const select = document.createElement("select");
      select.dataset.field = field.name;
      const all = document.createElement("option");
      all.value = "";
      all.textContent = "全部";
      select.append(all);
      const values = [...new Set(table.rows.map((row) => row.values[field.name]).filter((value) => value !== undefined))];
      values.sort(compareFieldValues);
      values.forEach((value) => {
        const option = document.createElement("option");
        option.value = value;
        option.textContent = typeOptionLabel(table.name, field.name, value);
        select.append(option);
      });
      if (values.includes(previous.get(field.name))) select.value = previous.get(field.name);
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
    elements.tableName.textContent = `${tableDisplayName(table)} · ${table.primary ? "限时主表" : "直接关联表"}`;
    elements.head.replaceChildren();
    elements.body.replaceChildren();
    elements.grid.replaceChildren();

    const query = elements.search.value.trim().toLocaleLowerCase();
    const statusFilter = elements.statusFilter.value;
    const hasSchedule = table.pairs.length > 0;
    const hasTitles = table.rows.some((row) => Object.keys(row.titles || {}).length > 0);
    elements.statusFilterLabel.classList.toggle("hidden", !hasSchedule);
    const typeFilters = [...elements.typeFilters.querySelectorAll("select")]
      .filter((select) => select.value !== "")
      .map((select) => ({ field: select.dataset.field, value: select.value }));
    const visibleRows = table.rows.filter((row) => {
      if (hasSchedule && statusFilter !== "all" && rowStatus(table, row) !== statusFilter) return false;
      if (typeFilters.some((filter) => effectiveValue(table.name, row, filter.field) !== filter.value)) return false;
      if (!query) return true;
      const relationValues = (row.shopRelations || []).flatMap((relation) => [
        relation.shopId, relation.shopItemCellGroupId, ...relation.shopItemCellIds,
        ...Object.values(relation.shopTitles || {})
      ]);
      const fieldValues = table.fields.flatMap((field) => [field.name, effectiveValue(table.name, row, field.name)]);
      const haystack = [...Object.values(row.titles || {}), ...relationValues, ...fieldValues].join(" ").toLocaleLowerCase();
      return haystack.includes(query);
    });

    const isGrid = state.view === "grid";
    elements.tableScroll.classList.toggle("hidden", isGrid);
    elements.grid.classList.toggle("hidden", !isGrid);
    syncViewToggle();
    if (isGrid) {
      visibleRows.forEach((row) => elements.grid.append(renderCard(table, row, hasTitles, hasSchedule)));
    } else {
      const headerRow = document.createElement("tr");
      if (hasTitles) headerRow.append(makeCell("th", "内容"));
      if (hasSchedule) headerRow.append(makeCell("th", "状态"));
      table.fields.forEach((field) => {
        const header = makeCell("th", field.name);
        header.title = `${field.type}${field.primaryKey ? " · 主键（只读）" : ""}`;
        headerRow.append(header);
      });
      elements.head.append(headerRow);
      visibleRows.forEach((row) => elements.body.append(renderRow(table, row, hasTitles, hasSchedule)));
    }
    elements.visibleCount.textContent = `${visibleRows.length.toLocaleString()} 行`;
    elements.empty.classList.toggle("hidden", visibleRows.length !== 0);
  }

  function renderRow(table, row, hasTitles, hasSchedule) {
    const tr = document.createElement("tr");
    if (hasTitles) {
      const contentCell = makeCell("td", localizedText(row.titles) || "-");
      contentCell.className = "content-cell";
      tr.append(contentCell);
    }
    if (hasSchedule) {
      const statusCell = document.createElement("td");
      statusCell.className = "status-cell";
      statusCell.append(renderStatus(rowStatus(table, row)));
      tr.append(statusCell);
    }
    table.fields.forEach((field) => {
      const td = document.createElement("td");
      td.className = field.primaryKey ? "id-cell field-column" : "field-column";
      td.append(renderFieldEditor(table, row, field));
      tr.append(td);
    });
    return tr;
  }

  function renderCard(table, row, hasTitles, hasSchedule) {
    const card = document.createElement("article");
    card.className = "schedule-card";

    const header = document.createElement("div");
    header.className = "card-header";
    const id = document.createElement("div");
    id.className = "card-id";
    const idLabel = document.createElement("span");
    idLabel.textContent = row.identity.map((field) => field.name).join(" / ") || "ROW";
    const idValue = document.createElement("strong");
    idValue.textContent = row.identity.map((field) => field.value).join(" / ") || String(row.index);
    id.append(idLabel, idValue);
    header.append(id);
    if (hasSchedule) header.append(renderStatus(rowStatus(table, row)));
    card.append(header);
    if (hasTitles) {
      const title = document.createElement("h3");
      title.textContent = localizedText(row.titles) || "-";
      card.append(title);
    }

    const fields = document.createElement("div");
    fields.className = "card-times card-fields";
    table.fields.forEach((field) => {
      const group = document.createElement("label");
      group.textContent = `${field.name}${field.primaryKey ? " · 主键" : ""}`;
      group.title = field.type;
      group.append(renderFieldEditor(table, row, field));
      fields.append(group);
    });
    card.append(fields);
    return card;
  }

  function renderStatus(status) {
    const element = document.createElement("span");
    element.className = `status ${status}`;
    element.textContent = statusLabels[status];
    return element;
  }

  function renderFieldEditor(table, row, field) {
    const wrapper = document.createElement("div");
    wrapper.className = "field-editor";
    const current = effectiveValue(table.name, row, field.name);
    if (field.primaryKey) {
      const value = document.createElement("code");
      value.className = "readonly-field";
      value.textContent = displayText(current);
      wrapper.append(value);
      return wrapper;
    }

    let input;
    if (field.kind === "bool") {
      input = document.createElement("select");
      [["true", "true"], ["false", "false"]].forEach(([value, label]) => {
        const option = document.createElement("option");
        option.value = value;
        option.textContent = label;
        input.append(option);
      });
      input.value = current;
    } else {
      input = document.createElement("input");
      input.type = field.datetime ? "datetime-local" : "text";
      if (field.datetime) {
        input.step = "1";
        input.value = timeInputValue(Number(current));
      } else {
        input.value = current;
        if (field.kind === "int32" || field.kind === "int64") input.inputMode = "numeric";
      }
    }
    input.dataset.table = table.name;
    input.dataset.row = String(row.index);
    input.dataset.field = field.name;
    input.setAttribute("aria-label", field.datetime ? `${field.name} ${timeModeLabel()}` : field.name);
    input.classList.toggle("changed", state.dirty.has(changeKey(table.name, row.index, field.name)));
    const eventName = field.datetime || field.kind === "bool" ? "change" : "input";
    input.addEventListener(eventName, () => onFieldChange(table, row, field, input));
    wrapper.append(input);
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

  function onFieldChange(table, row, field, input) {
    let value = input.value;
    if (field.datetime) {
      const milliseconds = parseTimeInput(value);
      if (!Number.isSafeInteger(milliseconds) || milliseconds < 0) {
        input.classList.add("invalid");
        showNotice("请输入有效的日期时间。", true);
        return;
      }
      value = String(milliseconds);
    } else if (field.kind === "int32" || field.kind === "int64") {
      const integer = value.trim();
      if (!/^-?\d+$/.test(integer)) {
        input.classList.add("invalid");
        showNotice(`${field.name} 必须是整数。`, true);
        return;
      }
      const parsed = BigInt(integer);
      const minimum = field.kind === "int32" ? -2147483648n : -9223372036854775808n;
      const maximum = field.kind === "int32" ? 2147483647n : 9223372036854775807n;
      if (parsed < minimum || parsed > maximum) {
        input.classList.add("invalid");
        showNotice(`${field.name} 超出 ${field.kind === "int32" ? "32" : "64"} 位整数范围。`, true);
        return;
      }
      value = parsed.toString();
      input.value = value;
    }
    if (field.kind === "bool" && value !== "true" && value !== "false") {
      input.classList.add("invalid");
      showNotice(`${field.name} 必须是 true 或 false。`, true);
      return;
    }
    input.classList.remove("invalid");
    const key = changeKey(table.name, row.index, field.name);
    if (value === row.values[field.name]) state.dirty.delete(key);
    else state.dirty.set(key, { table: table.name, row: row.index, field: field.name, value });
    input.classList.toggle("changed", state.dirty.has(key));
    updateDirtyUI();
  }

  function rowStatus(table, row) {
    const pair = table.pairs[0];
    if (!pair) return "expired";
    const start = Number(effectiveValue(table.name, row, pair.start));
    const end = Number(effectiveValue(table.name, row, pair.end));
    const now = Date.now();
    if (end === 0) return "disabled";
    if (now < start) return "upcoming";
    if (now <= end) return "active";
    return "expired";
  }

  function effectiveValue(table, row, field) {
    return state.dirty.get(changeKey(table, row.index, field))?.value ?? row.values[field];
  }

  function changeKey(table, row, field) { return `${table}\u0000${row}\u0000${field}`; }

  function timeModeLabel() {
    return state.timeMode === "utc" ? "UTC" : "本机时间";
  }

  function parseTimeInput(value) {
    if (!value) return 0;
    return state.timeMode === "utc" ? Date.parse(`${value}Z`) : new Date(value).getTime();
  }

  function timeInputValue(milliseconds) {
    if (milliseconds === 0) return "";
    const date = new Date(milliseconds);
    if (Number.isNaN(date.getTime())) return "";
    if (state.timeMode === "local") {
      const localMilliseconds = milliseconds - date.getTimezoneOffset() * 60 * 1000;
      return new Date(localMilliseconds).toISOString().slice(0, 19);
    }
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
    elements.saveSummary.textContent = count ? `${count} 个字段等待应用` : "没有待应用的修改";
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

  const availabilityLabels = { standard: "常驻", event: "活动", limited: "限定" };
  const weaponAttributeLabels = { 1: "暗", 2: "火", 3: "光", 5: "水", 6: "风" };
  const weaponTypeLabels = { 1: "小剑", 2: "枪", 3: "大剑", 4: "拳", 5: "杖", 6: "铳" };
  const gachaGroupDefinitions = [
    { id: "character_weapon_4", grantType: "character_weapon", star: 4, label: "赠送角色 · 4星" },
    { id: "weapon_only_4", grantType: "weapon_only", star: 4, label: "仅武器 · 4星" },
    { id: "character_weapon_3", grantType: "character_weapon", star: 3, label: "赠送角色 · 3星" },
    { id: "weapon_only_3", grantType: "weapon_only", star: 3, label: "仅武器 · 3星" },
    { id: "character_weapon_2", grantType: "character_weapon", star: 2, label: "赠送角色 · 2星" },
    { id: "weapon_only_2", grantType: "weapon_only", star: 2, label: "仅武器 · 2星" }
  ];

  function switchAdminSection(section) {
    state.section = section === "gacha" ? "gacha" : "master";
    const isGacha = state.section === "gacha";
    document.querySelectorAll(".master-only").forEach((element) => element.classList.toggle("hidden", isGacha));
    elements.gachaEditor.classList.toggle("hidden", !isGacha);
    elements.tabMaster.classList.toggle("active", !isGacha);
    elements.tabGacha.classList.toggle("active", isGacha);
    elements.tabMaster.setAttribute("aria-pressed", String(!isGacha));
    elements.tabGacha.setAttribute("aria-pressed", String(isGacha));
    if (isGacha) renderGachaEditor();
  }

  function resetGachaDraft() {
    state.gachaDraft = JSON.parse(JSON.stringify(state.gachaCatalog?.config || {}));
    state.gachaDraft.version ||= 1;
    state.gachaDraft.limitedSets ||= {};
    state.gachaDraft.weapons ||= {};
    state.gachaDraft.banners ||= {};
    state.gachaDraft.groupWeights ||= {
      characterWeapon: { "2": 0, "3": 500, "4": 200 },
      weaponOnly: { "2": 8000, "3": 1000, "4": 300 }
    };
    state.gachaDraft.sourceMasterDataHash = state.gachaCatalog?.masterDataHash || "";
    state.gachaDirty = false;
  }

  function renderGachaEditor() {
    if (!state.gachaCatalog || !state.gachaDraft) return;
    renderGachaLanguages();
    renderGachaLimitedSets();
    renderGachaWeapons();
    renderGachaBanners();
    renderGachaBannerEditor();
    renderGachaWarnings();
    updateGachaDirtyUI();
  }

  function renderGachaLanguages() {
    const languages = state.gachaCatalog.languages?.length ? state.gachaCatalog.languages : [state.gachaCatalog.defaultLanguage || "en"];
    elements.gachaLanguageSelect.replaceChildren();
    languages.forEach((language) => {
      const option = document.createElement("option");
      option.value = language;
      option.textContent = languageLabels[language] || language;
      elements.gachaLanguageSelect.append(option);
    });
    if (!languages.includes(state.language)) state.language = state.gachaCatalog.defaultLanguage || "en";
    elements.gachaLanguageSelect.value = state.language;
  }

  function renderGachaWarnings() {
    const warnings = [];
    if (!state.gachaCatalog.configExists) {
      warnings.push("Gacha 配置尚未发布，当前所有可抽取武器按常驻处理，且无限定、无 Pickup。");
    } else if (!state.gachaDirty && state.gachaCatalog.config?.sourceMasterDataHash !== state.gachaCatalog.masterDataHash) {
      warnings.push("Gacha 配置基于旧版主数据；新增可抽取武器已按常驻处理，请检查后重新发布。");
    }
    elements.gachaWarnings.textContent = warnings.join("\n");
    elements.gachaWarnings.classList.toggle("hidden", warnings.length === 0);
  }

  function renderGachaLimitedSets() {
    elements.gachaLimitedSets.replaceChildren();
    const entries = Object.entries(state.gachaDraft.limitedSets).sort(([left], [right]) => left.localeCompare(right));
    if (!entries.length) {
      const empty = document.createElement("span");
      empty.className = "notes-empty";
      empty.textContent = "尚未定义限定集合。";
      elements.gachaLimitedSets.append(empty);
      return;
    }
    entries.forEach(([id, definition]) => {
      const chip = document.createElement("span");
      chip.className = "limited-set-chip";
      const name = document.createElement("strong");
      name.textContent = definition.displayName;
      const key = document.createElement("code");
      key.textContent = id;
      const rename = document.createElement("button");
      rename.className = "chip-action";
      rename.type = "button";
      rename.textContent = "重命名";
      rename.addEventListener("click", () => renameLimitedSet(id));
      const remove = document.createElement("button");
      remove.className = "chip-action";
      remove.type = "button";
      remove.textContent = "删除";
      remove.addEventListener("click", () => removeLimitedSet(id));
      chip.append(name, key, rename, remove);
      elements.gachaLimitedSets.append(chip);
    });
  }

  function addLimitedSet() {
    const id = elements.gachaLimitedSetId.value.trim();
    const displayName = elements.gachaLimitedSetName.value.trim();
    if (!id || !displayName) {
      showNotice("请填写限定集合稳定键和显示名称。", true);
      return;
    }
    if (!/^[a-zA-Z0-9_.-]+$/.test(id)) {
      showNotice("限定集合稳定键只能包含英文、数字、下划线、点和连字符。", true);
      return;
    }
    if (state.gachaDraft.limitedSets[id]) {
      showNotice(`限定集合 ${id} 已存在。`, true);
      return;
    }
    state.gachaDraft.limitedSets[id] = { displayName };
    elements.gachaLimitedSetId.value = "";
    elements.gachaLimitedSetName.value = "";
    markGachaDirty();
  }

  function renameLimitedSet(id) {
    const current = state.gachaDraft.limitedSets[id]?.displayName || id;
    const displayName = (prompt("新的显示名称：", current) || "").trim();
    if (!displayName || displayName === current) return;
    state.gachaDraft.limitedSets[id].displayName = displayName;
    markGachaDirty();
  }

  function removeLimitedSet(id) {
    if (!confirm(`删除限定集合 ${id}？引用它的武器会恢复为常驻，卡池引用也会被移除。`)) return;
    delete state.gachaDraft.limitedSets[id];
    Object.entries(state.gachaDraft.weapons).forEach(([weaponId, definition]) => {
      if (definition.limitedSet === id) delete state.gachaDraft.weapons[weaponId];
    });
    Object.values(state.gachaDraft.banners).forEach((banner) => {
      banner.limitedSets = (banner.limitedSets || []).filter((value) => value !== id);
    });
    pruneAllBannerPickups();
    markGachaDirty();
  }

  function visibleGachaWeapons() {
    const query = elements.gachaWeaponSearch.value.trim().toLocaleLowerCase();
    const availability = elements.gachaAvailabilityFilter.value;
    const star = elements.gachaStarFilter.value;
    const attributeType = elements.gachaAttributeFilter.value;
    const weaponType = elements.gachaWeaponTypeFilter.value;
    const grantType = elements.gachaGrantFilter.value;
    return state.gachaCatalog.weapons.filter((weapon) => {
      if (!weapon.eligible) return false;
      const effective = effectiveWeaponAvailability(weapon);
      if (availability !== "all" && effective !== availability) return false;
      if (star !== "all" && weapon.star !== Number(star)) return false;
      if (attributeType !== "all" && weapon.attributeType !== Number(attributeType)) return false;
      if (weaponType !== "all" && weapon.weaponType !== Number(weaponType)) return false;
      if (grantType !== "all" && weapon.grantType !== grantType) return false;
      if (!query) return true;
      const haystack = [weapon.weaponId, ...Object.values(weapon.weaponNames || {}), ...Object.values(weapon.costumeNames || {})].join(" ").toLocaleLowerCase();
      return haystack.includes(query);
    });
  }

  function effectiveWeaponAvailability(weapon) {
    if (!weapon.eligible) return "event";
    return state.gachaDraft.weapons[String(weapon.weaponId)]?.availability || "standard";
  }

  function starSymbols(star) {
    return "★".repeat(Math.max(0, Number(star) || 0));
  }

  function renderGachaWeapons() {
    elements.gachaWeaponBody.replaceChildren();
    const visible = visibleGachaWeapons();
    visible.forEach((weapon) => elements.gachaWeaponBody.append(renderGachaWeaponRow(weapon)));
    elements.gachaWeaponEmpty.classList.toggle("hidden", visible.length !== 0);

    const standard = state.gachaCatalog.weapons.filter((weapon) => weapon.eligible && effectiveWeaponAvailability(weapon) === "standard").length;
    const overridden = state.gachaCatalog.weapons.filter((weapon) => weapon.eligible && effectiveWeaponAvailability(weapon) !== "standard").length;
    elements.gachaStandardCount.textContent = standard.toLocaleString();
    elements.gachaOverrideCount.textContent = overridden.toLocaleString();
  }

  function renderGachaWeaponRow(weapon) {
    const tr = document.createElement("tr");
    tr.append(
      makeCell("td", String(weapon.weaponId)),
      makeCell("td", gachaLocalizedText(weapon.weaponNames) || `#${weapon.weaponId}`),
      makeCell("td", gachaLocalizedText(weapon.costumeNames) || (weapon.costumeId ? `Costume #${weapon.costumeId}` : "—")),
      makeCell("td", weaponAttributeLabels[weapon.attributeType] || `#${weapon.attributeType}`),
      makeCell("td", weaponTypeLabels[weapon.weaponType] || `#${weapon.weaponType}`)
    );

    const groupCell = document.createElement("td");
    const group = document.createElement("span");
    group.className = "star-rating";
    group.textContent = starSymbols(weapon.star);
    group.title = `${weapon.star} 星`;
    groupCell.append(group);
    tr.append(groupCell);

    const definition = state.gachaDraft.weapons[String(weapon.weaponId)];
    const availabilityCell = document.createElement("td");
    const availability = document.createElement("select");
    const availabilityOptions = weapon.eligible ? [["standard", "常驻"], ["event", "活动"], ["limited", "限定"]] : [["event", "活动（自动）"]];
    availabilityOptions.forEach(([value, label]) => {
      const option = document.createElement("option");
      option.value = value;
      option.textContent = label;
      availability.append(option);
    });
    availability.value = effectiveWeaponAvailability(weapon);
    availability.disabled = !weapon.eligible;
    availability.addEventListener("change", () => setWeaponAvailability(weapon, availability.value));
    availabilityCell.append(availability);
    if (!weapon.eligible) {
      const note = document.createElement("div");
      note.className = "unavailable";
      note.textContent = "主数据已排除，自动视为活动";
      availabilityCell.append(note);
    }
    tr.append(availabilityCell);

    const setCell = document.createElement("td");
    const limitedSet = document.createElement("select");
    const empty = document.createElement("option");
    empty.value = "";
    empty.textContent = "请选择";
    limitedSet.append(empty);
    Object.entries(state.gachaDraft.limitedSets).sort(([left], [right]) => left.localeCompare(right)).forEach(([id, value]) => {
      const option = document.createElement("option");
      option.value = id;
      option.textContent = value.displayName;
      limitedSet.append(option);
    });
    limitedSet.value = definition?.limitedSet || "";
    limitedSet.disabled = effectiveWeaponAvailability(weapon) !== "limited";
    limitedSet.addEventListener("change", () => {
      state.gachaDraft.weapons[String(weapon.weaponId)].limitedSet = limitedSet.value;
      pruneAllBannerPickups();
      markGachaDirty();
    });
    setCell.append(limitedSet);
    tr.append(setCell);
    return tr;
  }

  function setWeaponAvailability(weapon, availability) {
    const key = String(weapon.weaponId);
    if (!availability || availability === "standard") {
      delete state.gachaDraft.weapons[key];
    } else if (availability === "limited") {
      const existing = state.gachaDraft.weapons[key];
      const firstSet = Object.keys(state.gachaDraft.limitedSets).sort()[0] || "";
      state.gachaDraft.weapons[key] = { availability, limitedSet: existing?.limitedSet || firstSet };
    } else {
      state.gachaDraft.weapons[key] = { availability };
    }
    pruneAllBannerPickups();
    markGachaDirty();
  }

  function renderGachaBanners() {
    const previous = elements.gachaBannerSelect.value;
    elements.gachaBannerSelect.replaceChildren();
    state.gachaCatalog.banners.forEach((banner) => {
      const option = document.createElement("option");
      option.value = String(banner.gachaId);
      const name = gachaLocalizedText(banner.titles) || banner.bannerAssetName || "未命名卡池";
      option.textContent = `${banner.gachaId} · ${name}`;
      elements.gachaBannerSelect.append(option);
    });
    if (state.gachaCatalog.banners.some((banner) => String(banner.gachaId) === previous)) elements.gachaBannerSelect.value = previous;
    elements.gachaBannerCount.textContent = state.gachaCatalog.banners.length.toLocaleString();
  }

  function currentGachaBanner() {
    const id = Number(elements.gachaBannerSelect.value);
    return state.gachaCatalog.banners.find((banner) => banner.gachaId === id) || state.gachaCatalog.banners[0];
  }

  function bannerConfig(gachaId, create = false) {
    const key = String(gachaId);
    if (!state.gachaDraft.banners[key] && create) state.gachaDraft.banners[key] = { limitedSets: [], pickupWeaponIds: [] };
    return state.gachaDraft.banners[key] || { limitedSets: [], pickupWeaponIds: [] };
  }

  function renderGachaBannerEditor() {
    const banner = currentGachaBanner();
    if (!banner) return;
    const configured = Object.prototype.hasOwnProperty.call(state.gachaDraft.banners, String(banner.gachaId));
    const definition = bannerConfig(banner.gachaId);
    elements.gachaBannerState.textContent = configured ? "已配置" : "默认配置";
    elements.gachaPickupCount.textContent = (definition.pickupWeaponIds || []).length.toLocaleString();
    renderBannerLimitedSets(banner);
    renderPickupWeapons(banner);
  }

  function renderBannerLimitedSets(banner) {
    elements.gachaBannerLimitedSets.replaceChildren();
    const entries = Object.entries(state.gachaDraft.limitedSets).sort(([left], [right]) => left.localeCompare(right));
    if (!entries.length) {
      const empty = document.createElement("span");
      empty.className = "notes-empty";
      empty.textContent = "没有可选限定集合。";
      elements.gachaBannerLimitedSets.append(empty);
      return;
    }
    const selected = new Set(bannerConfig(banner.gachaId).limitedSets || []);
    entries.forEach(([id, definition]) => {
      const label = document.createElement("label");
      const input = document.createElement("input");
      input.type = "checkbox";
      input.checked = selected.has(id);
      input.addEventListener("change", () => {
        const current = bannerConfig(banner.gachaId, true);
        const values = new Set(current.limitedSets || []);
        if (input.checked) values.add(id);
        else values.delete(id);
        current.limitedSets = [...values].sort();
        pruneBannerPickups(banner.gachaId);
        markGachaDirty();
      });
      label.append(input, document.createTextNode(`${definition.displayName} · ${id}`));
      elements.gachaBannerLimitedSets.append(label);
    });
  }

  function candidateWeaponsForBanner(gachaId) {
    const allowed = new Set(bannerConfig(gachaId).limitedSets || []);
    return state.gachaCatalog.weapons.filter((weapon) => {
      if (!weapon.eligible) return false;
      const definition = state.gachaDraft.weapons[String(weapon.weaponId)];
      if (!definition) return true;
      if (definition.availability === "event") return false;
      if (definition.availability === "standard") return true;
      return definition.availability === "limited" && allowed.has(definition.limitedSet);
    });
  }

  function renderPickupWeapons(banner) {
    elements.gachaPickupBody.replaceChildren();
    const definition = bannerConfig(banner.gachaId);
    const pickupIds = definition.pickupWeaponIds || [];
    const pickupSet = new Set(pickupIds);
    const allReferences = new Map(state.gachaCatalog.weapons.map((weapon) => [weapon.weaponId, weapon]));
    const candidates = candidateWeaponsForBanner(banner.gachaId);
    const candidateSet = new Set(candidates.map((weapon) => weapon.weaponId));
    const pickupOrder = new Map(pickupIds.map((weaponId, index) => [weaponId, index]));
    const orderedById = new Map(candidates.map((weapon) => [weapon.weaponId, weapon]));
    pickupIds.forEach((weaponId) => { if (allReferences.has(weaponId)) orderedById.set(weaponId, allReferences.get(weaponId)); });
    const availabilityRanks = { limited: 0, standard: 1, event: 2 };
    const ordered = [...orderedById.values()].sort((left, right) => {
      const availabilityDifference = (availabilityRanks[effectiveWeaponAvailability(left)] ?? 3) - (availabilityRanks[effectiveWeaponAvailability(right)] ?? 3);
      if (availabilityDifference) return availabilityDifference;
      const leftPickupOrder = pickupOrder.get(left.weaponId);
      const rightPickupOrder = pickupOrder.get(right.weaponId);
      if (leftPickupOrder !== undefined && rightPickupOrder !== undefined) return leftPickupOrder - rightPickupOrder;
      if (leftPickupOrder !== undefined) return -1;
      if (rightPickupOrder !== undefined) return 1;
      return left.weaponId - right.weaponId;
    });
    const query = elements.gachaPickupSearch.value.trim().toLocaleLowerCase();
    const star = elements.gachaPickupStarFilter.value;
    const attributeType = elements.gachaPickupAttributeFilter.value;
    const weaponType = elements.gachaPickupWeaponTypeFilter.value;
    const grantType = elements.gachaPickupGrantFilter.value;
    ordered.filter((weapon) => {
      if (star !== "all" && weapon.star !== Number(star)) return false;
      if (attributeType !== "all" && weapon.attributeType !== Number(attributeType)) return false;
      if (weaponType !== "all" && weapon.weaponType !== Number(weaponType)) return false;
      if (grantType !== "all" && weapon.grantType !== grantType) return false;
      if (!query) return true;
      return [weapon.weaponId, ...Object.values(weapon.weaponNames || {}), ...Object.values(weapon.costumeNames || {})].join(" ").toLocaleLowerCase().includes(query);
    }).forEach((weapon) => {
      const tr = document.createElement("tr");
      if (!candidateSet.has(weapon.weaponId)) tr.className = "invalid";
      const checkCell = document.createElement("td");
      const checkbox = document.createElement("input");
      checkbox.type = "checkbox";
      checkbox.checked = pickupSet.has(weapon.weaponId);
      checkbox.disabled = !candidateSet.has(weapon.weaponId) && !checkbox.checked;
      checkbox.addEventListener("change", () => {
        const current = bannerConfig(banner.gachaId, true);
        if (checkbox.checked) current.pickupWeaponIds = [...(current.pickupWeaponIds || []), weapon.weaponId];
        else current.pickupWeaponIds = (current.pickupWeaponIds || []).filter((id) => id !== weapon.weaponId);
        markGachaDirty();
      });
      checkCell.append(checkbox);
      const nameCell = document.createElement("td");
      nameCell.className = "pickup-weapon-name";
      const name = document.createElement("strong");
      name.textContent = gachaLocalizedText(weapon.weaponNames) || `#${weapon.weaponId}`;
      nameCell.append(name);
      const costumeName = weapon.costumeId ? gachaLocalizedText(weapon.costumeNames) || `Costume #${weapon.costumeId}` : "—";
      const groupCell = makeCell("td", starSymbols(weapon.star));
      groupCell.className = "star-rating";
      groupCell.title = `${weapon.star} 星`;
      const orderCell = document.createElement("td");
      if (pickupSet.has(weapon.weaponId)) {
        const index = pickupIds.indexOf(weapon.weaponId);
        const actions = document.createElement("div");
        actions.className = "order-actions";
        [["↑", -1], ["↓", 1]].forEach(([label, offset]) => {
          const button = document.createElement("button");
          button.type = "button";
          button.className = "order-button";
          button.textContent = label;
          button.disabled = index + offset < 0 || index + offset >= pickupIds.length;
          button.addEventListener("click", () => movePickup(banner.gachaId, index, index + offset));
          actions.append(button);
        });
        orderCell.append(actions);
      }
      tr.append(
        checkCell,
        makeCell("td", String(weapon.weaponId)),
        nameCell,
        makeCell("td", costumeName),
        makeCell("td", weaponAttributeLabels[weapon.attributeType] || `#${weapon.attributeType}`),
        makeCell("td", weaponTypeLabels[weapon.weaponType] || `#${weapon.weaponType}`),
        groupCell,
        orderCell
      );
      elements.gachaPickupBody.append(tr);
    });
  }

  function movePickup(gachaId, from, to) {
    const current = bannerConfig(gachaId, true);
    const ids = [...(current.pickupWeaponIds || [])];
    [ids[from], ids[to]] = [ids[to], ids[from]];
    current.pickupWeaponIds = ids;
    markGachaDirty();
  }

  function pruneBannerPickups(gachaId) {
    const current = state.gachaDraft.banners[String(gachaId)];
    if (!current) return;
    const candidates = new Set(candidateWeaponsForBanner(gachaId).map((weapon) => weapon.weaponId));
    current.pickupWeaponIds = (current.pickupWeaponIds || []).filter((weaponId) => candidates.has(weaponId));
  }

  function pruneAllBannerPickups() {
    Object.keys(state.gachaDraft.banners).forEach((gachaId) => pruneBannerPickups(Number(gachaId)));
  }

  function groupWeight(definition) {
    const weights = definition.grantType === "character_weapon" ? state.gachaDraft.groupWeights.characterWeapon : state.gachaDraft.groupWeights.weaponOnly;
    return Number(weights?.[String(definition.star)] || 0);
  }

  function groupStatsForBanner(banner) {
    const definition = bannerConfig(banner.gachaId);
    const pickupSet = new Set(definition.pickupWeaponIds || []);
    const candidates = candidateWeaponsForBanner(banner.gachaId);
    const groups = gachaGroupDefinitions.map((group) => {
      const items = candidates.filter((weapon) => weapon.star === group.star && weapon.grantType === group.grantType);
      const pickup = items.filter((weapon) => pickupSet.has(weapon.weaponId));
      return { ...group, weight: groupWeight(group), itemCount: items.length, pickupCount: pickup.length, nonPickupCount: items.length - pickup.length };
    });
    const totalWeight = groups.reduce((sum, group) => sum + group.weight, 0);
    const tenthWeights = Object.fromEntries(groups.map((group) => [group.id, group.weight]));
    groups.filter((group) => group.star === 2).forEach((group) => {
      const target = groups.find((candidate) => candidate.star === 3 && candidate.grantType === group.grantType);
      if (target) tenthWeights[target.id] += group.weight;
      tenthWeights[group.id] = 0;
    });
    const tenthTotalWeight = Object.values(tenthWeights).reduce((sum, weight) => sum + weight, 0);
    return { groups, candidates, totalWeight, tenthWeights, tenthTotalWeight };
  }

  function gachaValidationErrors() {
    const errors = [];
    const weights = gachaGroupDefinitions.map((definition) => ({ definition, weight: groupWeight(definition) }));
    if (weights.some(({ weight }) => !Number.isInteger(weight) || weight < 0)) errors.push("六组权重必须是非负整数");
    if (weights.reduce((sum, { weight }) => sum + weight, 0) <= 0) errors.push("六组权重合计必须大于 0");
    state.gachaCatalog.weapons.forEach((weapon) => {
      const definition = state.gachaDraft.weapons[String(weapon.weaponId)];
      if (!definition) return;
      if (!weapon.eligible && definition.availability !== "event") errors.push(`武器 ${weapon.weaponId} 不可抽取，只能设为活动`);
      if (definition.availability === "limited" && !state.gachaDraft.limitedSets[definition.limitedSet]) errors.push(`限定武器 ${weapon.weaponId} 尚未选择有效限定集合`);
    });
    Object.entries(state.gachaDraft.limitedSets).forEach(([id, definition]) => {
      if (!id.trim() || !definition.displayName?.trim()) errors.push(`限定集合 ${id || "<空>"} 缺少稳定键或显示名称`);
    });
    state.gachaCatalog.banners.forEach((banner) => {
      const stats = groupStatsForBanner(banner);
      stats.groups.forEach((group) => {
        if (group.weight > 0 && group.itemCount === 0) errors.push(`卡池 ${banner.gachaId} 的 ${group.label} 没有候选武器`);
        if ((stats.tenthWeights[group.id] || 0) > 0 && group.star >= 3 && group.itemCount === 0) errors.push(`卡池 ${banner.gachaId} 的十连末位 ${group.label} 没有候选武器`);
        if (group.pickupCount > 0 && group.nonPickupCount === 0) errors.push(`卡池 ${banner.gachaId} 的 ${group.label} 全部为 Pickup`);
      });
      const candidates = new Set(stats.candidates.map((weapon) => weapon.weaponId));
      (bannerConfig(banner.gachaId).pickupWeaponIds || []).forEach((weaponId) => {
        if (!candidates.has(weaponId)) errors.push(`卡池 ${banner.gachaId} 的 Pickup 武器 ${weaponId} 不在候选池`);
      });
    });
    return errors;
  }

  function updateGachaDirtyUI() {
    const errors = gachaValidationErrors();
    if (errors.length) elements.gachaSaveSummary.textContent = `无法发布：${errors[0]}${errors.length > 1 ? `（另有 ${errors.length - 1} 项）` : ""}`;
    else elements.gachaSaveSummary.textContent = state.gachaDirty ? "Gacha 配置修改等待发布" : "没有待发布的 Gacha 修改";
    elements.gachaSave.disabled = !state.gachaDirty || errors.length > 0;
    elements.gachaDiscard.disabled = !state.gachaDirty;
  }

  function markGachaDirty() {
    state.gachaDirty = true;
    renderGachaEditor();
  }

  function gachaLocalizedText(titles) {
    const text = titles?.[state.language] || titles?.[state.gachaCatalog.defaultLanguage] || Object.values(titles || {})[0] || "";
    return displayText(text);
  }

  elements.loginForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    state.token = elements.token.value;
    try { await loadCatalog(); } catch (_) { /* notice is already shown */ }
  });

  elements.logout.addEventListener("click", () => {
    state.token = "";
    state.catalog = null;
    state.gachaCatalog = null;
    state.gachaDraft = null;
    state.gachaDirty = false;
    state.dirty.clear();
    sessionStorage.removeItem("lunar-admin-token");
    showLogin();
  });

  elements.tabMaster.addEventListener("click", () => switchAdminSection("master"));
  elements.tabGacha.addEventListener("click", () => switchAdminSection("gacha"));

  elements.tableSelect.addEventListener("change", () => {
    renderTypeFilters(currentTable());
    renderTable();
  });
  elements.statusFilter.addEventListener("change", renderTable);
  elements.timezone.addEventListener("change", () => {
    state.timeMode = elements.timezone.value === "utc" ? "utc" : "local";
    renderTable();
  });
  elements.languageSelect.addEventListener("change", () => {
    state.language = elements.languageSelect.value;
    localStorage.setItem("lunar-admin-language", state.language);
    renderTable();
    renderGachaEditor();
  });
  elements.search.addEventListener("input", renderTable);
  elements.viewButtons.forEach((button) => button.addEventListener("click", () => {
    state.view = button.dataset.view;
    localStorage.setItem("lunar-admin-view", state.view);
    renderTable();
  }));
  elements.refresh.addEventListener("click", async () => {
    if ((state.dirty.size || state.gachaDirty) && !confirm("刷新会放弃尚未应用的修改，是否继续？")) return;
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
      showNotice(`应用成功：更新 ${result.changedRows} 行、${result.changedCells} 个字段。`);
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

  elements.gachaLanguageSelect.addEventListener("change", () => {
    state.language = elements.gachaLanguageSelect.value;
    localStorage.setItem("lunar-admin-language", state.language);
    if (elements.languageSelect.querySelector(`option[value="${CSS.escape(state.language)}"]`)) elements.languageSelect.value = state.language;
    renderGachaEditor();
    renderTable();
  });
  elements.gachaAddLimitedSet.addEventListener("click", addLimitedSet);
  elements.gachaWeaponSearch.addEventListener("input", renderGachaWeapons);
  elements.gachaAvailabilityFilter.addEventListener("change", renderGachaWeapons);
  elements.gachaStarFilter.addEventListener("change", renderGachaWeapons);
  elements.gachaAttributeFilter.addEventListener("change", renderGachaWeapons);
  elements.gachaWeaponTypeFilter.addEventListener("change", renderGachaWeapons);
  elements.gachaGrantFilter.addEventListener("change", renderGachaWeapons);
  elements.gachaBannerSelect.addEventListener("change", renderGachaBannerEditor);
  const renderCurrentPickupWeapons = () => {
    const banner = currentGachaBanner();
    if (banner) renderPickupWeapons(banner);
  };
  elements.gachaPickupSearch.addEventListener("input", renderCurrentPickupWeapons);
  elements.gachaPickupStarFilter.addEventListener("change", renderCurrentPickupWeapons);
  elements.gachaPickupAttributeFilter.addEventListener("change", renderCurrentPickupWeapons);
  elements.gachaPickupWeaponTypeFilter.addEventListener("change", renderCurrentPickupWeapons);
  elements.gachaPickupGrantFilter.addEventListener("change", renderCurrentPickupWeapons);
  elements.gachaDiscard.addEventListener("click", () => {
    if (!confirm("放弃全部尚未发布的 Gacha 修改？")) return;
    resetGachaDraft();
    renderGachaEditor();
    showNotice("已放弃本次 Gacha 修改。");
  });
  elements.gachaSave.addEventListener("click", async () => {
    const validationErrors = gachaValidationErrors();
    if (validationErrors.length) {
      showNotice(validationErrors[0], true);
      return;
    }
    elements.gachaPublishDialog.showModal();
  });
  elements.gachaPublishCancel.addEventListener("click", () => elements.gachaPublishDialog.close());
  elements.gachaPublishConfirm.addEventListener("click", async () => {
    elements.gachaPublishDialog.close();
    elements.gachaSave.disabled = true;
    elements.gachaDiscard.disabled = true;
    showNotice("正在校验、写入并热更新 Gacha 配置…");
    try {
      await api("/api/admin/gacha-config", {
        method: "POST",
        body: JSON.stringify({ expectedContentHash: state.gachaCatalog.contentHash, config: state.gachaDraft })
      });
      await loadCatalog();
      state.section = "gacha";
      switchAdminSection("gacha");
      showNotice("Gacha 配置发布成功，新的抽取请求已使用新版本。");
    } catch (error) {
      showNotice(error.message, true);
      if (error.status === 409) {
        try { await loadCatalog(); } catch (_) { /* keep the conflict notice */ }
        switchAdminSection("gacha");
      }
    } finally {
      updateGachaDirtyUI();
    }
  });

  window.addEventListener("beforeunload", (event) => {
    if (!state.dirty.size && !state.gachaDirty) return;
    event.preventDefault();
    event.returnValue = "";
  });

  if (state.token) loadCatalog().catch(() => showLogin());
  else showLogin();
})();
