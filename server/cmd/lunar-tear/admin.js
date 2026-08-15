(() => {
  "use strict";

  const $ = (selector) => document.querySelector(selector);
  const imagePreviewBaseURL = "https://assets.lycheenut.cc/assets/ui";
  const elements = {
    loginPanel: $("#login-panel"), loginForm: $("#login-form"), token: $("#token"),
    workspace: $("#workspace"), logout: $("#logout"), version: $("#version"),
    tableCount: $("#table-count"), rowCount: $("#row-count"), dirtyCount: $("#dirty-count"),
    timezone: $("#timezone"), tableSelect: $("#table-select"), typeFilters: $("#type-filters"),
    modeControl: $("#detail-mode-control"), modeButtons: document.querySelectorAll(".mode-button"),
    statusFilter: $("#status-filter"), statusFilterLabel: $("#status-filter-label"),
    languageSelect: $("#language-select"), search: $("#search"), refresh: $("#refresh"), notice: $("#notice"),
    entityName: $("#entity-name"), tableName: $("#table-name"), visibleCount: $("#visible-count"),
    tableScroll: $("#table-scroll"), scheduleTable: $("#schedule-table"), head: $("#schedule-head"), body: $("#schedule-body"),
    empty: $("#empty-state"),
    saveSummary: $("#save-summary"), discard: $("#discard"), save: $("#save"),
    masterUpdateDialog: $("#master-update-dialog"), masterUpdateSummary: $("#master-update-summary"),
    masterUpdatePreview: $("#master-update-preview"), masterUpdateCancel: $("#master-update-cancel"),
    masterUpdateConfirm: $("#master-update-confirm"),
    tabMaster: $("#tab-master"), tabRelated: $("#tab-related"), tabDelivery: $("#tab-delivery"),
    tabGacha: $("#tab-gacha"), gachaEditor: $("#gacha-editor"), masterLayout: $("#master-layout"),
    rewardReference: $("#reward-reference"), rewardVisibleCount: $("#reward-visible-count"),
    rewardType: $("#reward-type"), rewardSearch: $("#reward-search"),
    rewardMaterialTypeLabel: $("#reward-material-type-label"), rewardMaterialType: $("#reward-material-type"),
    rewardWeaponFilters: $("#reward-weapon-filters"), rewardWeaponAttribute: $("#reward-weapon-attribute"),
    rewardWeaponType: $("#reward-weapon-type"), rewardWeaponGrant: $("#reward-weapon-grant"),
    rewardReferenceList: $("#reward-reference-list"), rewardReferenceEmpty: $("#reward-reference-empty"),
    rewardPageSize: $("#reward-page-size"), rewardPagePrevious: $("#reward-page-previous"),
    rewardPageInfo: $("#reward-page-info"), rewardPageNext: $("#reward-page-next"),
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
    gachaGroupProbabilities: $("#gacha-group-probabilities"),
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
    mode: "simple",
    timeMode: "local",
    catalog: null,
    dirty: new Map(),
    section: "master",
    tableSelections: { master: "", related: "", delivery: "" },
    rewardCatalog: null,
    rewardPage: 1,
    rewardPageSize: 25,
    rewardPageCount: 1,
    gachaCatalog: null,
    gachaDraft: null,
    gachaDirty: false,
    pendingMasterChanges: null
  };
  const statusLabels = { active: "进行中", upcoming: "未开始", expired: "已结束", disabled: "已禁用" };
  const languageLabels = { en: "English", ja: "日本語", ko: "한국어" };
  const simpleFieldNames = {
    m_beginner_campaign: ["BeginnerCampaignId", "GrantCampaignTermDayCount", "CampaignUnlockQuestId"],
    m_big_hunt_schedule: ["BigHuntScheduleId"],
    m_comeback_campaign: ["ComebackCampaignId", "ComebackJudgeDayCount", "GrantCampaignTermDayCount", "CampaignUnlockQuestId", "ComebackCampaignGradeGroupId"],
    m_consumable_item_term: ["ConsumableItemTermId"],
    m_dokan: ["DokanId", "SortOrder", "DokanType"],
    m_enhance_campaign: ["EnhanceCampaignId", "EnhanceCampaignTargetGroupId", "EnhanceCampaignEffectType", "EnhanceCampaignEffectValue", "TargetUserStatusType"],
    m_event_quest_chapter: ["EventQuestChapterId", "EventQuestType", "SortOrder", "NameEventQuestTextId", "BannerAssetId"],
    m_event_quest_daily_group: ["EventQuestDailyGroupId"],
    m_event_quest_labyrinth_season: ["EventQuestChapterId", "SeasonNumber"],
    m_login_bonus: ["LoginBonusId", "SortOrder", "LoginBonusStartConditionId", "LoginBonusAssetName"],
    m_maintenance: ["MaintenanceId"],
    m_mission_term: ["MissionTermId"],
    m_mom_banner: ["MomBannerId", "SortOrderDesc", "DestinationDomainType", "DestinationDomainId", "BannerAssetName"],
    m_navi_cut_in: ["NaviCutInId", "RelatedCutInFunctionType", "SortOrder", "NaviCutInContentGroupId", "RelatedCutInFunctionValue"],
    m_omikuji: ["OmikujiId"],
    m_pvp_season: ["PvpSeasonId", "NameAssetPath"],
    m_quest_campaign: ["QuestCampaignId", "QuestCampaignTargetGroupId", "QuestCampaignEffectGroupId", "TargetUserStatusType"],
    m_shop: ["ShopId", "ShopGroupType", "SortOrderInShopGroup", "NameShopTextId", "ShopItemCellGroupId"],
    m_shop_item_cell_term: ["ShopItemCellTermId"],
    m_tip: ["TipId", "TitleTipTextId", "ContentTipTextId"]
  };

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
      const [catalog, gachaCatalog, rewardCatalog] = await Promise.all([
        api("/api/admin/master-data/schedules"),
        api("/api/admin/gacha-config"),
        api("/api/admin/reward-reference")
      ]);
      state.catalog = catalog;
      state.gachaCatalog = gachaCatalog;
      state.rewardCatalog = rewardCatalog;
      resetGachaDraft();
      state.dirty.clear();
      state.pendingMasterChanges = null;
      sessionStorage.setItem("lunar-admin-token", state.token);
      showWorkspace();
      renderCatalog();
      renderGachaEditor();
      initializeRewardFilters();
      renderRewardReference();
      switchAdminSection(state.section);
      showNotice(`已读取 ${state.catalog.tableCount} 张配置表、${state.catalog.rowCount} 行内容，以及 ${rewardReferenceCount()} 个奖励对象。`);
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
    const tables = configurationTables();
    const previous = state.tableSelections[state.section] || elements.tableSelect.value;
    elements.tableSelect.replaceChildren();
    tables.forEach((table) => {
      const option = document.createElement("option");
      option.value = table.name;
      option.textContent = `${tableDisplayName(table)}（${table.rows.length} 行）`;
      option.title = table.name;
      elements.tableSelect.append(option);
    });
    if (tables.some((table) => table.name === previous)) elements.tableSelect.value = previous;
    state.tableSelections[state.section] = elements.tableSelect.value;
    renderLanguages();
    renderTypeFilters(currentTable());
    elements.version.textContent = `版本 ${state.catalog.version.slice(0, 12)}`;
    elements.version.title = state.catalog.version;
    elements.tableCount.textContent = tables.length.toLocaleString();
    elements.tableCount.title = `${tables.length} 张配置表`;
    elements.rowCount.textContent = tables.reduce((count, table) => count + table.rows.length, 0).toLocaleString();
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
    const tables = configurationTables();
    return tables.find((table) => table.name === elements.tableSelect.value) || tables[0];
  }

  function configurationTables() {
    if (!state.catalog) return [];
    if (state.section === "delivery") return state.catalog.tables.filter((table) => table.delivery);
    if (state.section === "related") return state.catalog.tables.filter((table) => !table.primary && !table.delivery);
    return state.catalog.tables.filter((table) => table.primary && !table.delivery);
  }

  function displayedTableFields(table) {
    if (table.name !== "m_login_bonus_stamp") return table.fields;
    return table.fields.filter((field) => !["LoginBonusId", "LowerPageNumber"].includes(field.name));
  }

  function renderTypeFilters(table) {
    const previous = new Map([...elements.typeFilters.querySelectorAll("select")].map((select) => [select.dataset.field, select.value]));
    elements.typeFilters.replaceChildren();
    if (table?.name === "m_login_bonus_stamp") {
      renderLoginBonusStampFilters(table, previous);
      elements.typeFilters.classList.remove("hidden");
      return;
    }
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

  function renderLoginBonusStampFilters(table, previous) {
    [
      { field: "LoginBonusId", label: "奖励来源 · LoginBonusId", optionLabel: loginBonusSourceLabel },
      { field: "LowerPageNumber", label: "页码 · LowerPageNumber", defaultValue: "1" }
    ].forEach((definition) => {
      const label = document.createElement("label");
      label.textContent = definition.label;
      const select = document.createElement("select");
      select.dataset.field = definition.field;
      const values = [...new Set(table.rows.map((row) => row.values[definition.field]).filter((value) => value !== undefined))];
      values.sort(compareFieldValues);
      values.forEach((value) => {
        const option = document.createElement("option");
        option.value = value;
        option.textContent = definition.optionLabel ? definition.optionLabel(value) : value;
        select.append(option);
      });
      const preferred = previous.get(definition.field) || definition.defaultValue;
      if (values.includes(preferred)) select.value = preferred;
      select.addEventListener("change", renderTable);
      label.append(select);
      elements.typeFilters.append(label);
    });
  }

  function loginBonusSourceLabel(loginBonusID) {
    const loginBonusTable = state.catalog?.tables.find((table) => table.name === "m_login_bonus");
    const loginBonus = loginBonusTable?.rows.find((row) => row.values.LoginBonusId === loginBonusID);
    const name = localizedText(loginBonus?.titles) || loginBonus?.values.LoginBonusAssetName;
    return name ? `${name}（${loginBonusID}）` : `${loginBonusID}`;
  }

  function compareFieldValues(left, right) {
    const leftNumber = Number(left);
    const rightNumber = Number(right);
    if (Number.isFinite(leftNumber) && Number.isFinite(rightNumber)) return leftNumber - rightNumber;
    return left.localeCompare(right);
  }

  function typeOptionLabel(tableName, fieldName, value) {
    if (fieldName.endsWith("PossessionType")) {
      const definition = rewardDefinitionForPossessionType(value);
      if (definition) return `${definition.label}（${value}）`;
    }
    return value;
  }

  function renderTable() {
    const table = currentTable();
    if (!table) return;
    const detailed = !table.primary || state.mode === "detail";
    elements.entityName.textContent = table.name;
    elements.tableName.textContent = tableDisplayName(table);
    elements.modeControl.classList.toggle("hidden", !table.primary);
    elements.scheduleTable.classList.toggle("detail-mode", detailed);
    syncModeToggle();
    elements.head.replaceChildren();
    elements.body.replaceChildren();

    const query = elements.search.value.trim().toLocaleLowerCase();
    const statusFilter = elements.statusFilter.value;
    const hasSchedule = (table.pairs || []).length > 0;
    const hasArtwork = table.name === "m_dokan";
    const hasContent = table.name === "m_mom_banner"
      || table.rows.some((row) => Object.keys(row.titles || {}).length > 0
        || Object.keys(row.contentBody || {}).length > 0
        || (row.contentFootnotes || []).length > 0);
    const displayedFields = displayedTableFields(table);
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
      const footnoteValues = (row.contentFootnotes || []).flatMap((footnote) => Object.values(footnote || {}));
      const artworkValues = (row.dokanImages || []).flatMap((image) => [image.contentIndex, image.imageId]);
      const haystack = [...Object.values(row.titles || {}), ...Object.values(row.contentBody || {}),
      ...footnoteValues, ...artworkValues, ...relationValues, ...fieldValues].join(" ").toLocaleLowerCase();
      return haystack.includes(query);
    });

    if (!detailed) {
      const headerRow = document.createElement("tr");
      ["ID", "内容"].forEach((label) => headerRow.append(makeCell("th", label)));
      if (hasArtwork) headerRow.append(makeCell("th", "配图"));
      ["状态", "备注"].forEach((label) => headerRow.append(makeCell("th", label)));
      simpleTimeFields(table).forEach((field) => headerRow.append(makeCell("th", field.name)));
      elements.head.append(headerRow);
      visibleRows.forEach((row) => elements.body.append(renderSimpleRow(table, row)));
    } else {
      const headerRow = document.createElement("tr");
      if (hasContent) headerRow.append(makeCell("th", "内容"));
      if (hasArtwork) headerRow.append(makeCell("th", "配图"));
      if (hasSchedule) headerRow.append(makeCell("th", "状态"));
      displayedFields.forEach((field) => {
        const header = makeCell("th", field.name);
        header.dataset.field = field.name;
        header.title = `${field.type}${field.primaryKey ? " · 主键（只读）" : ""}`;
        headerRow.append(header);
      });
      elements.head.append(headerRow);
      visibleRows.forEach((row) => elements.body.append(renderDetailedRow(table, row, displayedFields, hasContent, hasArtwork, hasSchedule)));
    }
    elements.visibleCount.textContent = `${visibleRows.length.toLocaleString()} 行`;
    elements.empty.classList.toggle("hidden", visibleRows.length !== 0);
  }

  function renderDetailedRow(table, row, fields, hasContent, hasArtwork, hasSchedule) {
    const tr = document.createElement("tr");
    if (hasContent) {
      const contentCell = renderContentCell(table, row);
      contentCell.classList.add("content-cell", "detailed-content-cell");
      tr.append(contentCell);
    }
    if (hasArtwork) tr.append(renderDokanImagesCell(row));
    if (hasSchedule) {
      const statusCell = document.createElement("td");
      statusCell.className = "status-cell";
      statusCell.append(renderStatus(rowStatus(table, row)));
      tr.append(statusCell);
    }
    fields.forEach((field) => {
      const td = document.createElement("td");
      td.className = field.primaryKey ? "id-cell field-column" : "field-column";
      td.dataset.field = field.name;
      td.append(renderFieldEditor(table, row, field));
      tr.append(td);
    });
    return tr;
  }

  function renderSimpleRow(table, row) {
    const tr = document.createElement("tr");
    const primary = row.identity[0];

    const idCell = makeCell("td", primary?.value || "-");
    idCell.className = "id-cell";
    idCell.title = primary?.name || "ID";
    tr.append(idCell);

    const contentCell = renderContentCell(table, row);
    contentCell.classList.add("content-cell");
    tr.append(contentCell);

    if (table.name === "m_dokan") tr.append(renderDokanImagesCell(row));

    const statusCell = document.createElement("td");
    statusCell.className = "status-cell";
    statusCell.append(renderStatus(rowStatus(table, row)));
    tr.append(statusCell);

    const notesCell = document.createElement("td");
    notesCell.className = "notes-cell";
    notesCell.append(renderSimpleNotes(table, row));
    tr.append(notesCell);

    simpleTimeFields(table).forEach((field) => {
      const td = document.createElement("td");
      td.className = "time-column";
      td.append(renderSimpleTimeEditor(table, row, field));
      tr.append(td);
    });
    return tr;
  }

  function renderSimpleNotes(table, row) {
    const notes = document.createElement("div");
    notes.className = "identity";
    (simpleFieldNames[table.name] || []).slice(1).forEach((fieldName) => {
      if (table.name === "m_shop" && fieldName === "ShopItemCellGroupId") return;
      const meta = document.createElement("div");
      meta.className = "identity-meta";
      meta.textContent = `${fieldName}=${displayText(effectiveValue(table.name, row, fieldName))}`;
      notes.append(meta);
    });
    if (!notes.childElementCount) {
      const empty = document.createElement("span");
      empty.className = "notes-empty";
      empty.textContent = "-";
      notes.append(empty);
    }
    return notes;
  }

  function simpleTimeFields(table) {
    return table.fields.filter((field) => field.datetime);
  }

  function renderSimpleTimeEditor(table, row, field) {
    const editor = renderFieldEditor(table, row, field);
    editor.classList.add("time-cell");
    return editor;
  }

  function renderContentCell(table, row) {
    if (table.name === "m_mom_banner") return renderMomBannerContentCell(row);
    if (table.name === "m_tip") return renderTipContentCell(row);

    const cell = document.createElement("td");
    const title = document.createElement("div");
    title.className = "content-title";
    title.textContent = localizedText(row.titles) || "-";
    cell.append(title);
    const footnotes = [...new Set((row.contentFootnotes || []).map(localizedInlineText).filter(Boolean))];
    if (footnotes.length) {
      const note = document.createElement("div");
      note.className = "content-footnote";
      note.textContent = footnotes.join(" · ");
      cell.append(note);
    }
    return cell;
  }

  function renderTipContentCell(row) {
    const cell = document.createElement("td");
    cell.className = "tip-content";
    cell.append(
      renderTipContentSection("标题", localizedText(row.titles) || "-", "tip-content-title"),
      renderTipContentSection("正文", localizedText(row.contentBody) || "-", "tip-content-body")
    );
    return cell;
  }

  function renderTipContentSection(labelText, text, textClass) {
    const section = document.createElement("div");
    section.className = "tip-content-section";
    const label = document.createElement("span");
    label.className = "tip-content-label";
    label.textContent = labelText;
    const value = document.createElement("div");
    value.className = textClass;
    value.textContent = text;
    section.append(label, value);
    return section;
  }

  function renderMomBannerContentCell(row) {
    const cell = document.createElement("td");
    const tooltip = [...new Set([
      localizedText(row.titles),
      ...(row.contentFootnotes || []).map(localizedInlineText)
    ].filter(Boolean))].join("\n") || "无文本说明";
    const previewURLs = momBannerPreviewURLs(row);
    if (!previewURLs.length) {
      cell.append(renderMomBannerPreviewMissing(tooltip));
      return cell;
    }

    const image = document.createElement("img");
    image.className = "mom-banner-preview";
    image.alt = tooltip;
    image.title = tooltip;
    image.loading = "lazy";
    image.decoding = "async";
    let previewIndex = 0;
    image.addEventListener("error", () => {
      previewIndex += 1;
      if (previewIndex < previewURLs.length) {
        image.src = previewURLs[previewIndex];
        return;
      }
      image.replaceWith(renderMomBannerPreviewMissing(tooltip));
    });
    image.src = previewURLs[previewIndex];
    cell.append(image);
    return cell;
  }

  function momBannerPreviewURLs(row) {
    const languages = [...new Set([state.language, state.catalog.defaultLanguage, "en"].filter(Boolean))];
    return [...new Set(languages.map((language) => {
      const segments = momBannerPreviewPath(row, language);
      if (!segments) return "";
      return `${imagePreviewBaseURL}/${segments.map(encodeURIComponent).join("/")}`;
    }).filter(Boolean))];
  }

  function momBannerPreviewPath(row, language) {
    const domainType = Number(effectiveValue("m_mom_banner", row, "DestinationDomainType"));
    const domainId = effectiveValue("m_mom_banner", row, "DestinationDomainId");
    const assetName = effectiveValue("m_mom_banner", row, "BannerAssetName");
    if (domainType === 1 && assetName) {
      return ["gacha", language, assetName, "mom_banner.png"];
    }
    if (domainType === 21) {
      const loginBonusAssetName = relatedTableValue(
        "m_login_bonus", "LoginBonusId", domainId, "LoginBonusAssetName"
      );
      return loginBonusAssetName
        ? ["login_bonus", language, "banner", `${loginBonusAssetName}.png`]
        : null;
    }
    if (domainType === 25) {
      const bannerAssetId = relatedTableValue(
        "m_event_quest_chapter", "EventQuestChapterId", domainId, "BannerAssetId"
      );
      return /^\d+$/.test(bannerAssetId)
        ? ["quest", language, "mom_banner", `event_mom_banner_${bannerAssetId.padStart(3, "0")}.png`]
        : null;
    }
    return assetName ? ["mom_banner", language, `mom_banner_${assetName}.png`] : null;
  }

  function relatedTableValue(tableName, idField, idValue, valueField) {
    const table = state.catalog.tables.find((candidate) => candidate.name === tableName);
    const expectedId = String(idValue ?? "");
    const row = table?.rows.find((candidate) => String(effectiveValue(tableName, candidate, idField) ?? "") === expectedId);
    return row ? String(effectiveValue(tableName, row, valueField) ?? "") : "";
  }

  function renderMomBannerPreviewMissing(tooltip) {
    const missing = document.createElement("span");
    missing.className = "mom-banner-preview-missing";
    missing.textContent = "预览不可用";
    missing.title = tooltip;
    return missing;
  }

  function renderDokanImagesCell(row) {
    const cell = document.createElement("td");
    cell.className = "dokan-images-cell";
    const images = [...(row.dokanImages || [])].sort((left, right) => (
      Number(left.contentIndex) - Number(right.contentIndex)
    ));
    if (!images.length) {
      const empty = document.createElement("span");
      empty.className = "notes-empty";
      empty.textContent = "无配图";
      cell.append(empty);
      return cell;
    }
    if (images.length === 1) {
      cell.append(renderDokanImage(images[0]));
      return cell;
    }
    cell.append(renderDokanCarousel(images));
    return cell;
  }

  function renderDokanCarousel(images) {
    const carousel = document.createElement("div");
    carousel.className = "dokan-carousel";
    carousel.tabIndex = 0;
    carousel.setAttribute("role", "region");
    carousel.setAttribute("aria-label", `Dokan 配图，共 ${images.length} 张`);

    const viewport = document.createElement("div");
    viewport.className = "dokan-carousel-viewport";
    const slides = images.map((entry) => renderDokanImage(entry));
    slides.forEach((slide) => viewport.append(slide));

    const controls = document.createElement("div");
    controls.className = "dokan-carousel-controls";
    const previous = document.createElement("button");
    previous.type = "button";
    previous.className = "dokan-carousel-button";
    previous.textContent = "‹";
    previous.setAttribute("aria-label", "上一张配图");
    const position = document.createElement("span");
    position.className = "dokan-carousel-position";
    position.setAttribute("aria-live", "polite");
    const next = document.createElement("button");
    next.type = "button";
    next.className = "dokan-carousel-button";
    next.textContent = "›";
    next.setAttribute("aria-label", "下一张配图");
    controls.append(previous, position, next);

    const dots = document.createElement("div");
    dots.className = "dokan-carousel-dots";
    const dotButtons = images.map((entry, index) => {
      const dot = document.createElement("button");
      dot.type = "button";
      dot.className = "dokan-carousel-dot";
      dot.setAttribute("aria-label", `查看第 ${index + 1} 张配图，播放序号 ${entry.contentIndex}`);
      dots.append(dot);
      return dot;
    });

    let current = 0;
    const show = (index) => {
      current = (index + slides.length) % slides.length;
      slides.forEach((slide, slideIndex) => {
        slide.hidden = slideIndex !== current;
        slide.setAttribute("aria-hidden", String(slideIndex !== current));
      });
      dotButtons.forEach((dot, dotIndex) => {
        if (dotIndex === current) dot.setAttribute("aria-current", "true");
        else dot.removeAttribute("aria-current");
      });
      position.textContent = `${current + 1} / ${slides.length}`;
    };
    previous.addEventListener("click", () => show(current - 1));
    next.addEventListener("click", () => show(current + 1));
    dotButtons.forEach((dot, index) => dot.addEventListener("click", () => show(index)));
    carousel.addEventListener("keydown", (event) => {
      if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
      event.preventDefault();
      show(current + (event.key === "ArrowLeft" ? -1 : 1));
    });

    carousel.append(viewport, controls, dots);
    show(0);
    return carousel;
  }

  function renderDokanImage(entry) {
    const figure = document.createElement("figure");
    figure.className = "dokan-image";
    const urls = dokanImagePreviewURLs(entry.imageId);
    const image = document.createElement("img");
    image.className = "dokan-image-preview";
    image.alt = `Dokan ImageId ${entry.imageId}`;
    image.title = dokanImageTooltip(entry);
    image.loading = "lazy";
    image.decoding = "async";
    let previewIndex = 0;
    image.addEventListener("error", () => {
      previewIndex += 1;
      if (previewIndex < urls.length) {
        image.src = urls[previewIndex];
        return;
      }
      image.replaceWith(renderMomBannerPreviewMissing(dokanImageTooltip(entry)));
    });
    image.src = urls[previewIndex];
    figure.append(image);
    return figure;
  }

  function dokanImageTooltip(entry) {
    const filename = dokanImagePreviewPath(entry.imageId, state.language).at(-1);
    return `#${entry.contentIndex} · ImageId ${entry.imageId}\n${filename}`;
  }

  function dokanImagePreviewURLs(imageId) {
    const languages = [...new Set([
      state.language, state.catalog.defaultLanguage, "en", "ja", "ko"
    ].filter(Boolean))];
    return languages.map((language) => {
      const segments = dokanImagePreviewPath(imageId, language);
      return `${imagePreviewBaseURL}/${segments.map(encodeURIComponent).join("/")}`;
    });
  }

  function dokanImagePreviewPath(imageId, language) {
    const assetName = `prm${String(imageId).padStart(3, "0")}.png`;
    return ["mom_promotion", language, "banner", assetName];
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

    const rewardPair = rewardFieldPair(table, field.name);
    if (rewardPair) return renderRewardFieldEditor(table, row, field, rewardPair);

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

  function rewardFieldPair(table, fieldName) {
    const typeField = table.fields.find((candidate) => {
      if (candidate.type !== "PossessionType" || !candidate.name.endsWith("Type")) return false;
      const idFieldName = `${candidate.name.slice(0, -"Type".length)}Id`;
      return candidate.name === fieldName || idFieldName === fieldName;
    });
    if (!typeField) return null;
    const idField = table.fields.find((candidate) => (
      candidate.name === `${typeField.name.slice(0, -"Type".length)}Id`
    ));
    return idField ? { typeField, idField } : null;
  }

  function renderRewardFieldEditor(table, row, field, pair) {
    return field.name === pair.typeField.name
      ? renderRewardTypeFieldEditor(table, row, pair)
      : renderRewardIDFieldEditor(table, row, pair);
  }

  function renderRewardTypeFieldEditor(table, row, pair) {
    const wrapper = document.createElement("div");
    wrapper.className = "field-editor reward-type-field-editor";
    const select = document.createElement("select");
    const currentType = effectiveValue(table.name, row, pair.typeField.name);
    rewardDefinitions.forEach((definition) => {
      if (!rewardReferencesForPossessionType(definition.possessionType).length) return;
      const option = document.createElement("option");
      option.value = definition.possessionType;
      option.textContent = `${definition.label}（${definition.possessionType}）`;
      select.append(option);
    });
    if (![...select.options].some((option) => option.value === currentType)) {
      const unknown = document.createElement("option");
      unknown.value = currentType;
      unknown.textContent = `未知类型（${currentType}）`;
      select.append(unknown);
    }
    select.value = currentType;
    configureFieldInput(select, table, row, pair.typeField);
    select.addEventListener("change", () => {
      onFieldChange(table, row, pair.typeField, select);
      const references = rewardReferencesForPossessionType(select.value);
      const currentID = effectiveValue(table.name, row, pair.idField.name);
      if (!references.some((reference) => String(reference.possessionId) === currentID) && references.length) {
        const originalID = row.values[pair.idField.name];
        const replacement = references.find((reference) => String(reference.possessionId) === originalID) || references[0];
        storeFieldChange(table, row, pair.idField, String(replacement.possessionId));
      }
      renderTable();
    });
    wrapper.append(select);
    return wrapper;
  }

  function renderRewardIDFieldEditor(table, row, pair) {
    const wrapper = document.createElement("div");
    wrapper.className = "field-editor reward-id-field-editor";
    const select = document.createElement("select");
    const possessionType = effectiveValue(table.name, row, pair.typeField.name);
    const currentID = effectiveValue(table.name, row, pair.idField.name);
    const definition = rewardDefinitionForPossessionType(possessionType);
    const references = rewardReferencesForPossessionType(possessionType);
    let reference = references.find((candidate) => String(candidate.possessionId) === currentID);
    populateRewardIDSelect(select, references, currentID, definition, false);
    configureFieldInput(select, table, row, pair.idField);
    const populate = () => populateRewardIDSelect(select, references, select.value, definition, true);
    select.addEventListener("focus", populate);
    select.addEventListener("pointerdown", populate);

    let icon = renderRewardIcon(reference, definition, "reward-field-icon");
    select.addEventListener("change", () => {
      onFieldChange(table, row, pair.idField, select);
      reference = references.find((candidate) => String(candidate.possessionId) === select.value);
      const nextIcon = renderRewardIcon(reference, definition, "reward-field-icon");
      icon.replaceWith(nextIcon);
      icon = nextIcon;
    });
    const selectSlot = document.createElement("div");
    selectSlot.className = "reward-field-select";
    selectSlot.append(select);
    wrapper.append(icon, selectSlot);
    return wrapper;
  }

  function populateRewardIDSelect(select, references, selectedID, definition, expanded) {
    if (expanded && select.dataset.expanded === "true") return;
    select.replaceChildren();
    const selected = references.find((reference) => String(reference.possessionId) === selectedID);
    const options = expanded ? references : selected ? [selected] : [];
    options.forEach((reference) => {
      const option = document.createElement("option");
      option.value = String(reference.possessionId);
      option.textContent = rewardReferenceOptionLabel(reference, definition);
      select.append(option);
    });
    if (!selected) {
      const unknown = document.createElement("option");
      unknown.value = selectedID;
      unknown.textContent = `未知奖励（ID ${selectedID}）`;
      select.append(unknown);
    }
    select.value = selectedID;
    select.dataset.expanded = String(expanded);
  }

  function configureFieldInput(input, table, row, field) {
    input.dataset.table = table.name;
    input.dataset.row = String(row.index);
    input.dataset.field = field.name;
    input.setAttribute("aria-label", field.name);
    input.classList.toggle("changed", state.dirty.has(changeKey(table.name, row.index, field.name)));
  }

  function localizedText(titles) {
    const text = titles?.[state.language] || titles?.[state.catalog.defaultLanguage] || Object.values(titles || {})[0] || "";
    return displayContentText(text);
  }

  function localizedInlineText(titles) {
    return localizedText(titles).replace(/\s*\n\s*/g, " ");
  }

  function displayContentText(value) {
    return displayText(value).replace(/<br>/g, "\n");
  }

  function displayText(value) {
    return String(value ?? "").replace(/\\n/g, "\n");
  }

  function renderMasterUpdatePreview(preview) {
    elements.masterUpdateSummary.textContent = `${preview.requestedChanges} 个上游/直接修改将生成 ${preview.generatedChanges} 个确定的下游修改，共影响 ${preview.changedRows} 行。`;
    elements.masterUpdatePreview.replaceChildren();

    (preview.impacts || []).forEach((impact) => {
      const group = document.createElement("section");
      group.className = "impact-group";
      const header = document.createElement("header");
      const title = document.createElement("strong");
      title.textContent = impact.kind;
      const count = document.createElement("span");
      count.textContent = `${(impact.downstream || []).length} 条确定下游`;
      header.append(title, count);
      group.append(header);

      group.append(renderImpactSection("修改的上游内容", [impact.upstream]));
      if ((impact.downstream || []).length) {
        group.append(renderImpactSection("受影响的下游内容", impact.downstream));
      } else {
        const empty = document.createElement("div");
        empty.className = "impact-section impact-no-change";
        empty.textContent = "没有找到可由确定链路关联的下游内容。";
        group.append(empty);
      }
      elements.masterUpdatePreview.append(group);
    });

    if ((preview.otherChanges || []).length) {
      const group = document.createElement("section");
      group.className = "impact-group";
      const header = document.createElement("header");
      const title = document.createElement("strong");
      title.textContent = "其他直接修改";
      const count = document.createElement("span");
      count.textContent = `${preview.otherChanges.length} 行`;
      header.append(title, count);
      group.append(header, renderImpactSection("本次一并提交", preview.otherChanges));
      elements.masterUpdatePreview.append(group);
    }
  }

  function renderImpactSection(titleText, records) {
    const section = document.createElement("div");
    section.className = "impact-section";
    const title = document.createElement("div");
    title.className = "impact-section-title";
    title.textContent = titleText;
    section.append(title);
    records.forEach((record) => section.append(renderImpactRecord(record)));
    return section;
  }

  function renderImpactRecord(record) {
    const container = document.createElement("article");
    container.className = "impact-record";
    const heading = document.createElement("div");
    heading.className = "impact-record-heading";
    if (record.relation) {
      const relation = document.createElement("span");
      relation.className = "impact-relation";
      relation.textContent = record.relation;
      heading.append(relation);
    }
    const identity = document.createElement("span");
    identity.className = "impact-identity";
    const identityText = (record.identity || []).map((entry) => `${entry.name}=${entry.value}`).join(", ") || `row=${record.row}`;
    identity.textContent = `${previewTableName(record.table)} · ${identityText}`;
    heading.append(identity);
    const titleText = previewLocalizedTitle(record.titles);
    if (titleText) {
      const title = document.createElement("span");
      title.className = "impact-title";
      title.textContent = titleText;
      heading.append(title);
    }
    container.append(heading);

    if (record.note) {
      const note = document.createElement("p");
      note.className = "impact-note";
      note.textContent = record.note;
      container.append(note);
    }
    const changes = document.createElement("div");
    changes.className = "impact-changes";
    if (!(record.changes || []).length) {
      const unchanged = document.createElement("span");
      unchanged.className = "impact-no-change";
      unchanged.textContent = "仅展示确定关联，本次不自动修改字段。";
      changes.append(unchanged);
    } else {
      record.changes.forEach((change) => changes.append(renderImpactChange(change)));
    }
    container.append(changes);
    return container;
  }

  function renderImpactChange(change) {
    const row = document.createElement("div");
    row.className = "impact-change";
    const field = document.createElement("span");
    field.className = "change-field";
    field.textContent = change.field;
    if (change.generated) {
      const badge = document.createElement("span");
      badge.className = "cascade-badge";
      badge.textContent = "自动级联";
      field.append(badge);
    }
    const before = document.createElement("code");
    before.textContent = previewChangeValue(change.before, change.datetime);
    const arrow = document.createElement("span");
    arrow.className = "change-arrow";
    arrow.textContent = "→";
    const after = document.createElement("code");
    after.textContent = previewChangeValue(change.after, change.datetime);
    row.append(field, before, arrow, after);
    return row;
  }

  function previewChangeValue(value, datetime) {
    if (!datetime) return displayText(value);
    const milliseconds = Number(value);
    if (!Number.isSafeInteger(milliseconds)) return displayText(value);
    if (milliseconds === 0) return "0（禁用）";
    const input = timeInputValue(milliseconds).replace("T", " ");
    return `${input} ${timeModeLabel()}`;
  }

  function previewLocalizedTitle(titles) {
    return displayContentText(titles?.[state.language]
      || titles?.[state.catalog.defaultLanguage]
      || Object.values(titles || {})[0]
      || "").replace(/\s*\n\s*/g, " ");
  }

  function previewTableName(tableName) {
    const table = state.catalog.tables.find((candidate) => candidate.name === tableName);
    return table ? tableDisplayName(table) : tableName;
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
    storeFieldChange(table, row, field, value, input);
  }

  function storeFieldChange(table, row, field, value, input = null) {
    const key = changeKey(table.name, row.index, field.name);
    if (value === row.values[field.name]) state.dirty.delete(key);
    else state.dirty.set(key, { table: table.name, row: row.index, field: field.name, value });
    input?.classList.toggle("changed", state.dirty.has(key));
    updateDirtyUI();
  }

  function rowStatus(table, row) {
    const pair = (table.pairs || [])[0];
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

  function syncModeToggle() {
    elements.modeButtons.forEach((button) => {
      const active = button.dataset.mode === state.mode;
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
  const materialTypeLabels = {
    10: "武器强化", 20: "服装强化", 30: "伙伴强化", 40: "武器技能强化",
    50: "服装技能强化", 60: "通用技能强化", 70: "武器进化", 80: "武器突破",
    90: "服装突破", 100: "传承的石碑", 110: "服装觉醒", 120: "升华",
    130: "精炼", 140: "天命"
  };
  const rewardDefinitions = [
    { key: "material", catalogKey: "materials", possessionType: "5", label: "道具", fallbackName: "未命名道具", glyph: "具" },
    { key: "weapon", catalogKey: "weapons", possessionType: "2", label: "武器", fallbackName: "未命名武器", glyph: "武" },
    { key: "companion", catalogKey: "companions", possessionType: "3", label: "伙伴", fallbackName: "未命名伙伴", glyph: "伙" },
    { key: "consumable", catalogKey: "consumableItems", possessionType: "6", label: "消耗品", fallbackName: "未命名消耗品", glyph: "消" },
    { key: "free_gem", catalogKey: "freeGems", possessionType: "12", label: "免费宝石", fallbackName: "免费宝石", glyph: "石" }
  ];
  const rewardTypes = rewardDefinitions.map((definition) => definition.key);
  const rewardPageSizes = [25, 50, 100];
  const gachaGroupDefinitions = [
    { id: "character_weapon_4", grantType: "character_weapon", star: 4, label: "4星角色武器" },
    { id: "weapon_only_4", grantType: "weapon_only", star: 4, label: "4星武器" },
    { id: "character_weapon_3", grantType: "character_weapon", star: 3, label: "3星角色武器" },
    { id: "weapon_only_3", grantType: "weapon_only", star: 3, label: "3星武器" },
    { id: "weapon_only_2", grantType: "weapon_only", star: 2, label: "2星武器", calculated: true }
  ];

  function initializeRewardFilters() {
    if (!state.rewardCatalog) return;
    if (!rewardTypes.includes(elements.rewardType.value)) {
      elements.rewardType.value = state.rewardCatalog.defaultType || "material";
    }
    populateRewardFilter(
      elements.rewardMaterialType,
      state.rewardCatalog.materials.map((item) => item.materialType),
      (value) => materialTypeLabels[value] || `类型 ${value}`
    );
    populateRewardFilter(
      elements.rewardWeaponAttribute,
      state.rewardCatalog.weapons.map((item) => item.attributeType),
      (value) => weaponAttributeLabels[value] || `属性 ${value}`
    );
    populateRewardFilter(
      elements.rewardWeaponType,
      state.rewardCatalog.weapons.map((item) => item.weaponType),
      (value) => weaponTypeLabels[value] || `类型 ${value}`
    );
  }

  function populateRewardFilter(select, values, labelForValue) {
    const previous = select.value;
    select.replaceChildren();
    const all = document.createElement("option");
    all.value = "";
    all.textContent = "全部";
    select.append(all);
    [...new Set(values.map(Number).filter(Number.isFinite))].sort((left, right) => left - right).forEach((value) => {
      const option = document.createElement("option");
      option.value = String(value);
      option.textContent = `${labelForValue(value)}（${value}）`;
      select.append(option);
    });
    if ([...select.options].some((option) => option.value === previous)) select.value = previous;
  }

  function rewardDefinitionForPossessionType(possessionType) {
    return rewardDefinitions.find((definition) => definition.possessionType === String(possessionType));
  }

  function rewardReferencesForPossessionType(possessionType) {
    const definition = rewardDefinitionForPossessionType(possessionType);
    return definition ? state.rewardCatalog?.[definition.catalogKey] || [] : [];
  }

  function rewardReferenceName(reference, definition) {
    return localizedText(reference?.names) || definition?.fallbackName || "未命名奖励";
  }

  function rewardReferenceOptionLabel(reference, definition) {
    const name = rewardReferenceName(reference, definition).replace(/\s*\n\s*/g, " ");
    return `${name}（${reference.possessionId}）`;
  }

  function renderAssetIcon(iconPath, alt, glyph, className) {
    const visual = document.createElement("div");
    visual.className = className;
    const fallback = () => {
      const fallbackGlyph = document.createElement("span");
      fallbackGlyph.textContent = glyph;
      visual.replaceChildren(fallbackGlyph);
    };
    if (!iconPath) {
      fallback();
      return visual;
    }
    const image = document.createElement("img");
    image.alt = alt;
    image.loading = "lazy";
    image.decoding = "async";
    image.addEventListener("error", fallback, { once: true });
    image.src = `${imagePreviewBaseURL}/${iconPath.split("/").map(encodeURIComponent).join("/")}`;
    visual.append(image);
    return visual;
  }

  function renderRewardIcon(reference, definition, className) {
    return renderAssetIcon(
      reference?.iconPath,
      rewardReferenceName(reference, definition),
      definition?.glyph || "奖",
      className
    );
  }

  function renderRewardReference() {
    if (!state.rewardCatalog) return;
    const rewardType = elements.rewardType.value;
    const isWeapon = rewardType === "weapon";
    elements.rewardMaterialTypeLabel.classList.toggle("hidden", rewardType !== "material");
    elements.rewardWeaponFilters.classList.toggle("hidden", !isWeapon);

    const query = elements.rewardSearch.value.trim().toLocaleLowerCase();
    const materialType = elements.rewardMaterialType.value;
    const attributeType = elements.rewardWeaponAttribute.value;
    const weaponType = elements.rewardWeaponType.value;
    const grantCharacter = elements.rewardWeaponGrant.value;
    const definition = rewardDefinitions.find((candidate) => candidate.key === rewardType);
    const source = definition ? state.rewardCatalog[definition.catalogKey] || [] : [];
    const visible = source.filter((item) => {
      if (rewardType === "material" && materialType && String(item.materialType) !== materialType) return false;
      if (isWeapon && attributeType && String(item.attributeType) !== attributeType) return false;
      if (isWeapon && weaponType && String(item.weaponType) !== weaponType) return false;
      if (isWeapon && grantCharacter && String(Boolean(item.grantsCharacter)) !== grantCharacter) return false;
      if (!query) return true;
      return `${item.possessionId} ${localizedText(item.names)} ${Object.values(item.names || {}).join(" ")}`
        .toLocaleLowerCase().includes(query);
    });

    const pageCount = Math.max(1, Math.ceil(visible.length / state.rewardPageSize));
    state.rewardPage = Math.min(Math.max(1, state.rewardPage), pageCount);
    state.rewardPageCount = pageCount;
    const pageStart = (state.rewardPage - 1) * state.rewardPageSize;
    const pageEnd = Math.min(pageStart + state.rewardPageSize, visible.length);
    const rendered = visible.slice(pageStart, pageEnd);
    elements.rewardReferenceList.replaceChildren();
    rendered.forEach((item) => elements.rewardReferenceList.append(renderRewardReferenceCard(item, rewardType)));
    elements.rewardReferenceList.scrollTop = 0;
    elements.rewardVisibleCount.textContent = visible.length
      ? `${visible.length.toLocaleString()} 项 · ${pageStart + 1}–${pageEnd}`
      : "0 项";
    elements.rewardPageInfo.textContent = `第 ${state.rewardPage.toLocaleString()} / ${pageCount.toLocaleString()} 页`;
    elements.rewardPagePrevious.disabled = state.rewardPage === 1;
    elements.rewardPageNext.disabled = state.rewardPage === pageCount;
    elements.rewardReferenceEmpty.classList.toggle("hidden", visible.length !== 0);
  }

  function resetRewardPageAndRender() {
    state.rewardPage = 1;
    renderRewardReference();
  }

  function rewardReferenceCount() {
    if (!state.rewardCatalog) return 0;
    return rewardDefinitions
      .reduce((count, definition) => count + (state.rewardCatalog[definition.catalogKey]?.length || 0), 0)
      .toLocaleString();
  }

  function renderRewardReferenceCard(item, rewardType) {
    const card = document.createElement("article");
    card.className = "reward-reference-card";
    const definition = rewardDefinitions.find((candidate) => candidate.key === rewardType);
    const visual = renderRewardIcon(item, definition, "reward-reference-icon");
    const name = rewardReferenceName(item, definition);

    const content = document.createElement("div");
    content.className = "reward-reference-content";
    const title = document.createElement("strong");
    title.textContent = name;
    const summary = document.createElement("span");
    if (rewardType === "weapon") {
      summary.textContent = `${weaponAttributeLabels[item.attributeType] || `属性 ${item.attributeType}`} · ${weaponTypeLabels[item.weaponType] || `类型 ${item.weaponType}`} · ${item.grantsCharacter ? "赠送角色" : "不赠送角色"}`;
    } else if (rewardType === "material") {
      summary.textContent = materialTypeLabels[item.materialType] || `类型 ${item.materialType}`;
    } else if (rewardType === "companion") {
      summary.textContent = weaponAttributeLabels[item.attributeType] || `属性 ${item.attributeType}`;
    } else if (rewardType === "consumable") {
      summary.textContent = `消耗品类型 ${item.consumableType}`;
    } else {
      summary.textContent = "免费宝石";
    }
    const identifiers = document.createElement("code");
    identifiers.textContent = `Type=${item.possessionType} · ID=${item.possessionId}`;
    content.append(title, summary, identifiers);
    card.append(visual, content);
    return card;
  }

  function switchAdminSection(section) {
    state.section = ["master", "related", "delivery", "gacha"].includes(section) ? section : "master";
    const isGacha = state.section === "gacha";
    const isDelivery = state.section === "delivery";
    document.querySelectorAll(".master-only").forEach((element) => element.classList.toggle("hidden", isGacha));
    elements.gachaEditor.classList.toggle("hidden", !isGacha);
    elements.rewardReference.classList.toggle("hidden", !isDelivery);
    elements.masterLayout.classList.toggle("with-reward-reference", isDelivery);
    elements.tabMaster.classList.toggle("active", state.section === "master");
    elements.tabRelated.classList.toggle("active", state.section === "related");
    elements.tabDelivery.classList.toggle("active", isDelivery);
    elements.tabGacha.classList.toggle("active", isGacha);
    elements.tabMaster.setAttribute("aria-pressed", String(state.section === "master"));
    elements.tabRelated.setAttribute("aria-pressed", String(state.section === "related"));
    elements.tabDelivery.setAttribute("aria-pressed", String(isDelivery));
    elements.tabGacha.setAttribute("aria-pressed", String(isGacha));
    if (isGacha) renderGachaEditor();
    else if (state.catalog) {
      renderCatalog();
      if (isDelivery) renderRewardReference();
    }
  }

  function resetGachaDraft() {
    state.gachaDraft = JSON.parse(JSON.stringify(state.gachaCatalog?.config || {}));
    state.gachaDraft.version ||= 1;
    state.gachaDraft.limitedSets ||= {};
    state.gachaDraft.weapons ||= {};
    state.gachaDraft.banners ||= {};
    state.gachaDraft.groupWeights ||= {
      characterWeapon: { "3": 500, "4": 200 },
      weaponOnly: { "2": 8000, "3": 1000, "4": 300 }
    };
    state.gachaDraft.groupWeights.characterWeapon ||= { "3": 500, "4": 200 };
    state.gachaDraft.groupWeights.weaponOnly ||= { "2": 8000, "3": 1000, "4": 300 };
    recalculateTwoStarWeaponProbability();
    state.gachaDraft.sourceMasterDataHash = state.gachaCatalog?.masterDataHash || "";
    state.gachaDirty = false;
  }

  function renderGachaEditor() {
    if (!state.gachaCatalog || !state.gachaDraft) return;
    renderGachaLanguages();
    renderGachaLimitedSets();
    renderGachaWeapons();
    renderGachaBanners();
    renderGachaGroupProbabilities();
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
    const weaponName = gachaLocalizedText(weapon.weaponNames) || `#${weapon.weaponId}`;
    const iconCell = document.createElement("td");
    iconCell.append(renderAssetIcon(weapon.iconPath, weaponName, "武", "gacha-weapon-icon"));
    tr.append(
      makeCell("td", String(weapon.weaponId)),
      iconCell,
      makeCell("td", weaponName),
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

  function setGroupWeight(definition, weight) {
    const weights = definition.grantType === "character_weapon" ? state.gachaDraft.groupWeights.characterWeapon : state.gachaDraft.groupWeights.weaponOnly;
    weights[String(definition.star)] = weight;
  }

  function recalculateTwoStarWeaponProbability() {
    const allocated = gachaGroupDefinitions
      .filter((definition) => !definition.calculated)
      .reduce((sum, definition) => sum + groupWeight(definition), 0);
    state.gachaDraft.groupWeights.weaponOnly["2"] = 10000 - allocated;
  }

  function formatGroupProbability(weight) {
    return (weight / 100).toFixed(2).replace(/\.00$/, "").replace(/(\.\d)0$/, "$1");
  }

  function syncCalculatedGroupProbability() {
    const calculated = gachaGroupDefinitions.find((definition) => definition.calculated);
    const input = elements.gachaGroupProbabilities.querySelector(`[data-group-id="${calculated.id}"]`);
    if (!input) return;
    const weight = groupWeight(calculated);
    input.value = formatGroupProbability(weight);
    input.closest(".gacha-probability-row").classList.toggle("invalid", weight < 0);
  }

  function renderGachaGroupProbabilities() {
    elements.gachaGroupProbabilities.replaceChildren();
    gachaGroupDefinitions.forEach((definition) => {
      const row = document.createElement("label");
      row.className = `gacha-probability-row${definition.calculated ? " calculated" : ""}`;
      const name = document.createElement("span");
      name.textContent = definition.label;
      const inputWrapper = document.createElement("span");
      inputWrapper.className = "gacha-probability-input";
      const input = document.createElement("input");
      input.type = "number";
      input.min = "0";
      input.max = "100";
      input.step = "0.01";
      input.value = formatGroupProbability(groupWeight(definition));
      input.dataset.groupId = definition.id;
      input.readOnly = Boolean(definition.calculated);
      input.setAttribute("aria-label", `${definition.label}概率`);
      if (!definition.calculated) {
        input.addEventListener("input", () => {
          const percentage = Number(input.value || 0);
          if (!Number.isFinite(percentage)) return;
          setGroupWeight(definition, Math.round(percentage * 100));
          recalculateTwoStarWeaponProbability();
          syncCalculatedGroupProbability();
          state.gachaDirty = true;
          updateGachaDirtyUI();
        });
      }
      const suffix = document.createElement("span");
      suffix.textContent = "%";
      inputWrapper.append(input, suffix);
      row.append(name, inputWrapper);
      elements.gachaGroupProbabilities.append(row);
    });
    syncCalculatedGroupProbability();
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
    const editableWeights = weights.filter(({ definition }) => !definition.calculated);
    if (editableWeights.some(({ weight }) => !Number.isInteger(weight) || weight < 0)) errors.push("四个可配置概率组必须是非负的 0.01% 倍数");
    if (groupWeight(gachaGroupDefinitions.find((definition) => definition.calculated)) < 0) errors.push("其他四个概率组合计不能超过 100%");
    if (weights.reduce((sum, { weight }) => sum + weight, 0) !== 10000) errors.push("五组概率合计必须为 100%");
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
    state.rewardCatalog = null;
    state.gachaCatalog = null;
    state.gachaDraft = null;
    state.gachaDirty = false;
    state.dirty.clear();
    sessionStorage.removeItem("lunar-admin-token");
    showLogin();
  });

  elements.tabMaster.addEventListener("click", () => switchAdminSection("master"));
  elements.tabRelated.addEventListener("click", () => switchAdminSection("related"));
  elements.tabDelivery.addEventListener("click", () => switchAdminSection("delivery"));
  elements.tabGacha.addEventListener("click", () => switchAdminSection("gacha"));

  elements.tableSelect.addEventListener("change", () => {
    state.tableSelections[state.section] = elements.tableSelect.value;
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
    renderTypeFilters(currentTable());
    renderTable();
    renderGachaEditor();
    renderRewardReference();
  });

  elements.rewardType.addEventListener("change", resetRewardPageAndRender);
  elements.rewardSearch.addEventListener("input", resetRewardPageAndRender);
  elements.rewardMaterialType.addEventListener("change", resetRewardPageAndRender);
  elements.rewardWeaponAttribute.addEventListener("change", resetRewardPageAndRender);
  elements.rewardWeaponType.addEventListener("change", resetRewardPageAndRender);
  elements.rewardWeaponGrant.addEventListener("change", resetRewardPageAndRender);
  elements.rewardPageSize.addEventListener("change", () => {
    const pageSize = Number(elements.rewardPageSize.value);
    state.rewardPageSize = rewardPageSizes.includes(pageSize) ? pageSize : 25;
    resetRewardPageAndRender();
  });
  elements.rewardPagePrevious.addEventListener("click", () => {
    if (state.rewardPage <= 1) return;
    state.rewardPage -= 1;
    renderRewardReference();
  });
  elements.rewardPageNext.addEventListener("click", () => {
    if (state.rewardPage >= state.rewardPageCount) return;
    state.rewardPage += 1;
    renderRewardReference();
  });
  elements.search.addEventListener("input", renderTable);
  elements.modeButtons.forEach((button) => button.addEventListener("click", () => {
    state.mode = button.dataset.mode === "detail" ? "detail" : "simple";
    renderTable();
  }));
  elements.refresh.addEventListener("click", async () => {
    if ((state.dirty.size || state.gachaDirty) && !confirm("刷新会放弃尚未应用的修改，是否继续？")) return;
    try { await loadCatalog(); } catch (_) { /* notice is already shown */ }
  });
  elements.discard.addEventListener("click", () => {
    if (!confirm("放弃全部尚未应用的修改？")) return;
    state.dirty.clear();
    state.pendingMasterChanges = null;
    updateDirtyUI();
    renderTable();
    showNotice("已放弃本次修改。");
  });
  elements.save.addEventListener("click", async () => {
    const changes = [...state.dirty.values()];
    if (!changes.length) return;
    setBusy(true, "正在计算确定链路及变更预览…");
    try {
      const preview = await api("/api/admin/master-data/schedules/preview", {
        method: "POST",
        body: JSON.stringify({ expectedVersion: state.catalog.version, changes })
      });
      state.pendingMasterChanges = changes;
      renderMasterUpdatePreview(preview);
      elements.masterUpdateDialog.showModal();
      showNotice("已生成上游及确定下游的变更预览，请确认后应用。");
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
  elements.masterUpdateCancel.addEventListener("click", () => {
    state.pendingMasterChanges = null;
    elements.masterUpdateDialog.close();
    showNotice("已取消应用，修改仍保留在编辑器中。");
  });
  elements.masterUpdateDialog.addEventListener("close", () => {
    if (elements.masterUpdateDialog.returnValue !== "confirm") state.pendingMasterChanges = null;
    elements.masterUpdateDialog.returnValue = "";
  });
  elements.masterUpdateConfirm.addEventListener("click", async () => {
    const changes = state.pendingMasterChanges;
    if (!changes?.length) return;
    elements.masterUpdateDialog.returnValue = "confirm";
    elements.masterUpdateDialog.close();
    state.pendingMasterChanges = null;
    setBusy(true, "正在重建、验证并热更新上游及关联主数据…");
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
