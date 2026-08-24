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
    languageSelect: $("#language-select"), tableSearchLabel: $("#table-search-label"),
    search: $("#search"), refresh: $("#refresh"), notice: $("#notice"),
    entityName: $("#entity-name"), tableName: $("#table-name"), visibleCount: $("#visible-count"),
    tableScroll: $("#table-scroll"), scheduleTable: $("#schedule-table"), head: $("#schedule-head"), body: $("#schedule-body"),
    missionRewardEditor: $("#mission-reward-editor"), missionRewardAssignmentBody: $("#mission-reward-assignment-body"),
    missionRewardAssignmentCount: $("#mission-reward-assignment-count"), missionRewardContentBody: $("#mission-reward-content-body"),
    missionRewardContentSearch: $("#mission-reward-content-search"), missionRewardContentCount: $("#mission-reward-content-count"),
    missionRewardContentUnreferenced: $("#mission-reward-content-unreferenced"),
    missionRewardContentAdd: $("#mission-reward-content-add"),
    missionRewardContentPageSize: $("#mission-reward-content-page-size"),
    missionRewardContentPagePrevious: $("#mission-reward-content-page-previous"),
    missionRewardContentPageInfo: $("#mission-reward-content-page-info"),
    missionRewardContentPageNext: $("#mission-reward-content-page-next"),
    missionReferenceDialog: $("#mission-reference-dialog"), missionReferenceEyebrow: $("#mission-reference-eyebrow"),
    missionReferenceTitle: $("#mission-reference-title"), missionReferenceSummary: $("#mission-reference-summary"),
    missionReferenceContent: $("#mission-reference-content"), missionReferenceClose: $("#mission-reference-close"),
    missionTermEditor: $("#mission-term-editor"), missionTermAssignmentBody: $("#mission-term-assignment-body"),
    missionTermAssignmentCount: $("#mission-term-assignment-count"), missionTermContentBody: $("#mission-term-content-body"),
    missionTermContentSearch: $("#mission-term-content-search"), missionTermContentCount: $("#mission-term-content-count"),
    missionTermContentPageSize: $("#mission-term-content-page-size"),
    missionTermContentPagePrevious: $("#mission-term-content-page-previous"),
    missionTermContentPageInfo: $("#mission-term-content-page-info"),
    missionTermContentPageNext: $("#mission-term-content-page-next"),
    shopEditor: $("#shop-editor"), shopCellGroupReferences: $("#shop-cell-group-references"),
    shopCellGroupCount: $("#shop-cell-group-count"), shopCellGroupAdd: $("#shop-cell-group-add"),
    shopCellGroupBody: $("#shop-cell-group-body"), shopCellSearch: $("#shop-cell-search"),
    shopCellUnreferenced: $("#shop-cell-unreferenced"), shopCellCount: $("#shop-cell-count"),
    shopCellAdd: $("#shop-cell-add"), shopCellBody: $("#shop-cell-body"),
    shopCellPageSize: $("#shop-cell-page-size"), shopCellPagePrevious: $("#shop-cell-page-previous"),
    shopCellPageInfo: $("#shop-cell-page-info"), shopCellPageNext: $("#shop-cell-page-next"),
    shopItemSearch: $("#shop-item-search"), shopItemUnreferenced: $("#shop-item-unreferenced"),
    shopItemCount: $("#shop-item-count"),
    shopItemBody: $("#shop-item-body"), shopItemPageSize: $("#shop-item-page-size"),
    shopItemPagePrevious: $("#shop-item-page-previous"), shopItemPageInfo: $("#shop-item-page-info"),
    shopItemPageNext: $("#shop-item-page-next"),
    questDropEditor: $("#quest-drop-editor"), questDropSearch: $("#quest-drop-search"),
    questDropCount: $("#quest-drop-count"), questDropBody: $("#quest-drop-body"),
    questDropPageSize: $("#quest-drop-page-size"), questDropPagePrevious: $("#quest-drop-page-previous"),
    questDropPageInfo: $("#quest-drop-page-info"), questDropPageNext: $("#quest-drop-page-next"),
    questDropSaveSummary: $("#quest-drop-save-summary"), questDropDiscard: $("#quest-drop-discard"),
    questDropSave: $("#quest-drop-save"), questDropCopyDialog: $("#quest-drop-copy-dialog"),
    questDropCopyTitle: $("#quest-drop-copy-title"), questDropCopySummary: $("#quest-drop-copy-summary"),
    questDropCopyField: $("#quest-drop-copy-field"), questDropCopySource: $("#quest-drop-copy-source"),
    questDropCopyError: $("#quest-drop-copy-error"), questDropCopyCancel: $("#quest-drop-copy-cancel"),
    questDropCopyConfirm: $("#quest-drop-copy-confirm"), questDropFilters: $("#quest-drop-filters"),
    empty: $("#empty-state"),
    saveSummary: $("#save-summary"), discard: $("#discard"), save: $("#save"),
    masterUpdateDialog: $("#master-update-dialog"), masterUpdateSummary: $("#master-update-summary"),
    masterUpdatePreview: $("#master-update-preview"), masterUpdateCancel: $("#master-update-cancel"),
    masterUpdateConfirm: $("#master-update-confirm"),
    tabMaster: $("#tab-master"), tabRelated: $("#tab-related"), tabDelivery: $("#tab-delivery"),
    tabDrop: $("#tab-drop"), tabGacha: $("#tab-gacha"), gachaEditor: $("#gacha-editor"),
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
    gachaPublishConfirm: $("#gacha-publish-confirm"),
    gachaKindPremium: $("#gacha-kind-premium"), gachaKindChapter: $("#gacha-kind-chapter"),
    gachaKindEvent: $("#gacha-kind-event"), premiumGachaConfig: $("#premium-gacha-config"),
    boxGachaConfig: $("#box-gacha-config"), boxGachaEyebrow: $("#box-gacha-eyebrow"),
    boxGachaTitle: $("#box-gacha-title"), boxGachaState: $("#box-gacha-state"),
    boxGachaBannerSelect: $("#box-gacha-banner-select"), boxGachaNumberLabel: $("#box-gacha-number-label"),
    boxGachaNumberSelect: $("#box-gacha-number-select"), boxGachaActions: $("#box-gacha-actions"),
    boxGachaAddBox: $("#box-gacha-add-box"), boxGachaRemoveBox: $("#box-gacha-remove-box"),
    boxGachaEmpty: $("#box-gacha-empty"), boxGachaEditorBody: $("#box-gacha-editor-body"),
    boxLimitedProbability: $("#box-limited-probability"), boxUnlimitedProbability: $("#box-unlimited-probability"),
    boxAddLimitedReward: $("#box-add-limited-reward"), boxAddUnlimitedReward: $("#box-add-unlimited-reward"),
    boxLimitedRewardBody: $("#box-limited-reward-body"), boxUnlimitedRewardBody: $("#box-unlimited-reward-body"),
    boxGachaRuleNote: $("#box-gacha-rule-note")
  };

  const state = {
    token: sessionStorage.getItem("lunar-admin-token") || "",
    language: localStorage.getItem("lunar-admin-language") || "en",
    mode: "simple",
    timeMode: "local",
    catalog: null,
    dirty: new Map(),
    section: "master",
    tableSelections: { master: "", related: "", delivery: "", drop: "m_quest_pickup_reward_group" },
    rewardCatalog: null,
    missionRewardContentPage: 1,
    missionRewardContentPageSize: 25,
    missionRewardContentPageCount: 1,
    missionRewardAdditions: [],
    missionRewardDeleteIDs: new Set(),
    missionRewardNextRow: -1,
    missionTermContentPage: 1,
    missionTermContentPageSize: 25,
    missionTermContentPageCount: 1,
    shopCellGroupDraft: [],
    shopCellGroupBaseline: "[]",
    shopCellGroupDirty: false,
    shopCellGroupSelection: "",
    shopCellPage: 1,
    shopCellPageSize: 25,
    shopCellPageCount: 1,
    shopCellAdditions: [],
    shopCellDeleteKeys: new Map(),
    shopItemPage: 1,
    shopItemPageSize: 10,
    shopItemPageCount: 1,
    shopItemCopies: [],
    shopItemDeleteIDs: new Set(),
    questDropDraft: new Map(),
    questDropBaseline: new Map(),
    questDropDirtyQuestIDs: new Set(),
    questDropRewardIndex: new Map(),
    questDropRewardReferenceIndex: new Map(),
    questDropGroupIndex: new Map(),
    questDropTypeFilter: "",
    questDropChapterFilter: "",
    questDropSubtypeFilter: "",
    questDropPage: 1,
    questDropPageSize: 10,
    questDropPageCount: 1,
    questDropCatalog: null,
    questDropCopyTargetID: 0,
    questDropCopySourceID: 0,
    questDropCopyConfirming: false,
    gachaCatalog: null,
    gachaDraft: null,
    gachaDirty: false,
    gachaKind: "premium",
    boxSelections: {},
    pendingMasterChanges: null
  };
  const statusLabels = { active: "进行中", upcoming: "未开始", expired: "已结束", disabled: "已禁用" };
  const languageLabels = { en: "English", ja: "日本語", ko: "한국어" };
  const missionCategoryLabels = {
    "1": "每日任务", "2": "挑战任务", "3": "特殊任务", "4": "WebView 任务",
    "5": "完成任务", "6": "商店购买任务", "7": "条件评估任务", "8": "妈妈积分任务",
    "9": "任务通行证 · 每日", "10": "任务通行证 · 特殊"
  };
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
    setBusy(true, "正在读取配置表元信息…");
    try {
      state.catalog = await api("/api/admin/master-data/catalog");
      state.gachaCatalog = null;
      state.questDropCatalog = null;
      state.rewardCatalog = null;
      state.gachaDraft = null;
      state.dirty.clear();
      state.pendingMasterChanges = null;
      sessionStorage.setItem("lunar-admin-token", state.token);
      showWorkspace();
      await switchAdminSection(sectionFromPath(), false);
      showNotice(`已读取 ${state.catalog.tableCount} 张配置表元信息；当前页面数据已按需加载。`);
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

  function sectionFromPath() {
    const sections = {
      "/admin/activities": "master",
      "/admin/related": "related",
      "/admin/delivery": "delivery",
      "/admin/drops": "drop",
      "/admin/gacha": "gacha"
    };
    return sections[window.location.pathname] || "master";
  }

  function sectionPath(section) {
    return {
      master: "/admin/activities",
      related: "/admin/related",
      delivery: "/admin/delivery",
      drop: "/admin/drops",
      gacha: "/admin/gacha"
    }[section] || "/admin/activities";
  }

  async function ensureRewardCatalog() {
    if (!state.rewardCatalog) state.rewardCatalog = await api("/api/admin/reward-reference");
  }

  async function loadSelectedTable() {
    const table = currentTable();
    if (!table || Array.isArray(table.rows)) return table;
    const requestedName = table.name;
    const payload = await api(`/api/admin/master-data/table?name=${encodeURIComponent(requestedName)}`);
    if (payload.version !== state.catalog.version) {
      throw new Error("主数据版本已变化，请刷新后重试。");
    }
    (payload.tables || []).forEach((loadedTable) => {
      const existing = state.catalog.tables.find((candidate) => candidate.name === loadedTable.name);
      if (existing) Object.assign(existing, loadedTable);
    });
    if (["m_mission_reward", "m_mission_term"].includes(requestedName)) {
      state.catalog.missionSources = payload.missionSources;
      resetMissionRewardDraft();
    }
    if (requestedName === "m_shop_item_content_possession") {
      state.catalog.shopEditor = payload.shopEditor;
      resetShopCellGroupDraft();
    }
    if (requestedName === "m_quest_pickup_reward_group") {
      state.catalog.questDropEditor = payload.questDropEditor;
    }
    return currentTable();
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
    if (state.section !== "drop") {
      const placeholder = new Option("请选择数据表", "");
      placeholder.disabled = true;
      elements.tableSelect.append(placeholder);
    }
    tables.forEach((table) => {
      const option = document.createElement("option");
      option.value = table.name;
      option.textContent = `${tableDisplayName(table)}（${Number(table.rowCount || 0).toLocaleString()} 行）`;
      option.title = table.name;
      elements.tableSelect.append(option);
    });
    elements.tableSelect.value = tables.some((table) => table.name === previous) ? previous : "";
    state.tableSelections[state.section] = elements.tableSelect.value;
    renderLanguages();
    elements.version.textContent = `版本 ${state.catalog.version.slice(0, 12)}`;
    elements.version.title = state.catalog.version;
    elements.tableCount.textContent = tables.length.toLocaleString();
    elements.tableCount.title = `${tables.length} 张配置表`;
    elements.rowCount.textContent = tables.reduce((count, table) => count + Number(table.rowCount || 0), 0).toLocaleString();
    elements.timezone.value = state.timeMode;
    updateDirtyUI();
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
    return tables.find((table) => table.name === elements.tableSelect.value);
  }

  function renderTableSelectionPrompt() {
    elements.entityName.textContent = "";
    elements.tableName.textContent = "请选择数据表";
    elements.visibleCount.textContent = "0 行";
    elements.head.replaceChildren();
    elements.body.replaceChildren();
    elements.scheduleTable.classList.add("hidden");
    elements.missionRewardEditor.classList.add("hidden");
    elements.missionTermEditor.classList.add("hidden");
    elements.shopEditor.classList.add("hidden");
    elements.questDropEditor.classList.add("hidden");
    elements.typeFilters.replaceChildren();
    elements.typeFilters.classList.add("hidden");
    elements.empty.textContent = "选择一个数据表后加载条目。";
    elements.empty.classList.remove("hidden");
  }

  function configurationTables() {
    if (!state.catalog) return [];
    if (state.section === "drop") return state.catalog.tables.filter((table) => table.name === "m_quest_pickup_reward_group");
    if (state.section === "delivery") return state.catalog.tables.filter((table) => table.delivery && table.name !== "m_quest_pickup_reward_group");
    if (state.section === "related") return state.catalog.tables.filter((table) => !table.primary && !table.delivery);
    return state.catalog.tables.filter((table) => table.primary && !table.delivery);
  }

  function displayedTableFields(table) {
    if (table.name === "m_login_bonus_stamp") {
      return table.fields.filter((field) => !["LoginBonusId", "LowerPageNumber"].includes(field.name));
    }
    if (["m_mission_reward", "m_mission_term"].includes(table.name)) {
      const idField = table.name === "m_mission_reward" ? "MissionRewardId" : "MissionTermId";
      return table.fields.filter((field) => field.name !== idField);
    }
    return table.fields;
  }

  function renderTypeFilters(table) {
    const previous = new Map([...elements.typeFilters.querySelectorAll("select")].map((select) => [select.dataset.field, select.value]));
    elements.typeFilters.replaceChildren();
    elements.questDropFilters.replaceChildren();
    elements.questDropFilters.classList.add("hidden");
    if (table?.name === "m_login_bonus_stamp") {
      renderLoginBonusStampFilters(table, previous);
      elements.typeFilters.classList.remove("hidden");
      return;
    }
    if (["m_mission_reward", "m_mission_term"].includes(table?.name)) {
      renderMissionSourceFilters(table, previous);
      elements.typeFilters.classList.remove("hidden");
      return;
    }
    if (table?.name === "m_shop_item_content_possession") {
      renderShopContentFilter(table, previous);
      elements.typeFilters.classList.remove("hidden");
      return;
    }
    if (table?.name === "m_quest_pickup_reward_group") {
      renderQuestDropFilters(elements.questDropFilters);
      elements.typeFilters.classList.add("hidden");
      elements.questDropFilters.classList.remove("hidden");
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

  function renderQuestDropFilters(container) {
    const editor = state.catalog?.questDropEditor || { types: [], chapters: [], quests: [] };
    const typeLabel = document.createElement("label");
    typeLabel.textContent = "副本类型";
    const typeSelect = document.createElement("select");
    typeSelect.dataset.field = "QuestDropType";
    [...editor.types].sort((left, right) => Number(left.value) - Number(right.value))
      .forEach((definition) => typeSelect.append(new Option(`${definition.value}. ${definition.label}`, definition.id)));
    if (!editor.types.some((definition) => definition.id === state.questDropTypeFilter)) {
      state.questDropTypeFilter = editor.types[0]?.id || "";
    }
    typeSelect.value = state.questDropTypeFilter;
    typeSelect.addEventListener("change", () => {
      state.questDropTypeFilter = typeSelect.value;
      state.questDropChapterFilter = "";
      state.questDropSubtypeFilter = "";
      state.questDropPage = 1;
      renderTypeFilters(currentTable());
      renderTable();
    });
    typeLabel.append(typeSelect);

    const chapterLabel = document.createElement("label");
    chapterLabel.textContent = "章节";
    const chapterSelect = document.createElement("select");
    chapterSelect.dataset.field = "QuestDropChapter";
    editor.chapters.filter((chapter) => chapter.typeId === state.questDropTypeFilter)
      .sort((left, right) => Number(left.chapterId) - Number(right.chapterId))
      .forEach((chapter) => {
        const value = `${chapter.typeId}:${chapter.chapterId}`;
        const name = localizedInlineText(chapter.names) || "未命名";
        chapterSelect.append(new Option(`${chapter.chapterId}. ${name}`, value));
      });
    if (![...chapterSelect.options].some((option) => option.value === state.questDropChapterFilter)) {
      state.questDropChapterFilter = chapterSelect.options[0]?.value || "";
    }
    chapterSelect.value = state.questDropChapterFilter;
    chapterSelect.addEventListener("change", () => {
      state.questDropChapterFilter = chapterSelect.value;
      state.questDropSubtypeFilter = "";
      state.questDropPage = 1;
      renderTypeFilters(currentTable());
      renderTable();
    });
    chapterLabel.append(chapterSelect);

    container.append(typeLabel, chapterLabel);
    const usesDifficultyFilter = state.questDropTypeFilter === "main";
    const usesSubcategoryFilter = state.questDropTypeFilter === "event-7";
    if (!usesDifficultyFilter && !usesSubcategoryFilter) {
      state.questDropSubtypeFilter = "";
      return;
    }

    const subtypeLabel = document.createElement("label");
    subtypeLabel.textContent = usesSubcategoryFilter ? "关卡类型" : "难度";
    const subtypeSelect = document.createElement("select");
    subtypeSelect.dataset.field = usesSubcategoryFilter ? "QuestDropSubcategory" : "QuestDropDifficulty";
    const subtypeValues = [...new Set(editor.quests
      .filter((quest) => quest.typeId === state.questDropTypeFilter
        && `${quest.typeId}:${quest.chapterId}` === state.questDropChapterFilter)
      .map((quest) => Number(usesSubcategoryFilter ? quest.subcategoryType : quest.difficultyType))
      .filter((value) => value > 0))]
      .sort((left, right) => left - right);
    subtypeValues.forEach((value) => {
      const name = usesSubcategoryFilter
        ? questDropCharacterQuestCategoryLabel(value)
        : questDropDifficultyLabel({ difficultyType: value });
      subtypeSelect.append(new Option(`${value}. ${name}`, String(value)));
    });
    if (!subtypeValues.some((value) => String(value) === state.questDropSubtypeFilter)) {
      state.questDropSubtypeFilter = subtypeValues.includes(1) ? "1" : String(subtypeValues[0] || "");
    }
    subtypeSelect.value = state.questDropSubtypeFilter;
    subtypeSelect.addEventListener("change", () => {
      state.questDropSubtypeFilter = subtypeSelect.value;
      state.questDropPage = 1;
      renderTable();
    });
    subtypeLabel.append(subtypeSelect);
    container.append(subtypeLabel);
  }

  const searchableSelectControllers = new WeakMap();

  function createSearchableSelect(select, config = {}) {
    let controller = searchableSelectControllers.get(select);
    if (!controller) {
      const wrapper = document.createElement("div");
      wrapper.className = "searchable-select";
      const input = document.createElement("input");
      input.type = "search";
      input.autocomplete = "off";
      input.setAttribute("role", "combobox");
      input.setAttribute("aria-autocomplete", "list");
      input.setAttribute("aria-expanded", "false");
      const list = document.createElement("div");
      list.className = "searchable-select-options hidden";
      list.setAttribute("role", "listbox");

      const nativeParent = select.parentNode;
      if (nativeParent) nativeParent.insertBefore(wrapper, select);
      select.classList.add("searchable-select-source");
      wrapper.append(input, list, select);

      const listGap = 6;
      const viewportMargin = 8;
      let trackingViewport = false;
      const positionList = () => {
        if (list.parentNode !== document.body || list.classList.contains("hidden")) return;
        const anchor = input.getBoundingClientRect();
        const viewportWidth = document.documentElement.clientWidth;
        const viewportHeight = document.documentElement.clientHeight;
        const width = Math.min(anchor.width, Math.max(0, viewportWidth - viewportMargin * 2));
        const left = Math.min(
          Math.max(anchor.left, viewportMargin),
          Math.max(viewportMargin, viewportWidth - viewportMargin - width)
        );
        list.style.right = "auto";
        list.style.left = `${left}px`;
        list.style.width = `${width}px`;
        const availableBelow = Math.max(0, viewportHeight - anchor.bottom - listGap - viewportMargin);
        const availableAbove = Math.max(0, anchor.top - listGap - viewportMargin);
        list.style.maxHeight = "none";
        const desiredHeight = Math.min(330, list.scrollHeight);
        const placeAbove = availableBelow < desiredHeight && availableAbove > availableBelow;
        const availableHeight = placeAbove ? availableAbove : availableBelow;

        list.style.maxHeight = `${Math.min(330, availableHeight)}px`;
        if (placeAbove) {
          list.style.top = "auto";
          list.style.bottom = `${viewportHeight - anchor.top + listGap}px`;
        } else {
          list.style.top = `${anchor.bottom + listGap}px`;
          list.style.bottom = "auto";
        }
      };
      const repositionList = () => {
        if (wrapper.isConnected) positionList();
        else close();
      };
      const stopTrackingViewport = () => {
        if (!trackingViewport) return;
        trackingViewport = false;
        window.removeEventListener("resize", repositionList);
        window.removeEventListener("scroll", repositionList, true);
        window.visualViewport?.removeEventListener("resize", repositionList);
      };
      const startTrackingViewport = () => {
        if (trackingViewport) return;
        trackingViewport = true;
        window.addEventListener("resize", repositionList);
        window.addEventListener("scroll", repositionList, true);
        window.visualViewport?.addEventListener("resize", repositionList);
      };
      const close = () => {
        list.classList.add("hidden");
        input.setAttribute("aria-expanded", "false");
        stopTrackingViewport();
        list.removeAttribute("style");
        if (wrapper.isConnected) wrapper.insertBefore(list, select);
        else list.remove();
      };
      const open = () => {
        if (list.parentNode !== document.body) document.body.append(list);
        list.classList.remove("hidden");
        input.setAttribute("aria-expanded", "true");
        startTrackingViewport();
        positionList();
      };
      const selectedOption = () => [...select.options].find((option) => option.value === select.value);
      const restoreSelection = () => {
        const option = selectedOption();
        input.value = option?.textContent?.trim() || "";
        input.title = input.value;
      };
      const availableOptions = () => {
        const configured = typeof controller.config.options === "function"
          ? controller.config.options()
          : controller.config.options;
        if (configured) {
          if (controller.optionSource !== configured) {
            controller.optionSource = configured;
            const groupOrder = new Map();
            controller.optionEntries = configured.map((entry, index) => {
              const group = String(entry.group || "");
              if (!groupOrder.has(group)) groupOrder.set(group, groupOrder.size);
              return {
                value: String(entry.value), label: String(entry.label), group,
                groupOrder: groupOrder.get(group), searchText: String(entry.searchText || ""),
                disabled: Boolean(entry.disabled), index
              };
            });
          }
          return controller.optionEntries;
        }
        return [...select.options].map((option, index) => ({
          option, value: option.value, label: option.textContent?.trim() || option.value,
          group: option.parentElement?.tagName === "OPTGROUP" ? option.parentElement.label : "",
          groupOrder: 0, searchText: option.dataset.searchText || "", disabled: option.disabled, index
        }));
      };
      const matchingOptions = (query = "") => {
        const normalized = query.trim().toLocaleLowerCase();
        const available = availableOptions();
        if (!normalized) return available.filter((entry) => !entry.disabled);
        const matches = [];
        available.forEach((entry) => {
          if (entry.disabled) return;
          const value = entry.value.toLocaleLowerCase();
          const searchText = `${entry.searchText} ${value} ${entry.label}`.toLocaleLowerCase();
          if (!searchText.includes(normalized)) return;
          const rank = value === normalized ? 0 : value.startsWith(normalized) ? 1 : entry.label.toLocaleLowerCase().startsWith(normalized) ? 2 : 3;
          matches.push({ ...entry, rank });
        });
        return matches.sort((left, right) => left.groupOrder - right.groupOrder || left.rank - right.rank || left.index - right.index);
      };
      const choose = (entry) => {
        if (!entry.option) {
          const option = document.createElement("option");
          option.value = entry.value;
          option.textContent = entry.label;
          select.replaceChildren(option);
        }
        select.value = entry.value;
        restoreSelection();
        close();
        select.dispatchEvent(new Event("change", { bubbles: true }));
      };
      const renderOptionWindow = (matches, start, end) => {
        controller.matches = matches;
        controller.windowStart = start;
        controller.windowEnd = end;
        list.replaceChildren();
        let previousGroup = null;
        matches.slice(start, end).forEach((entry) => {
          if (entry.group && entry.group !== previousGroup) {
            const group = document.createElement("div");
            group.className = "searchable-select-group";
            group.textContent = entry.group;
            list.append(group);
          }
          previousGroup = entry.group;
          const button = document.createElement("button");
          button.type = "button";
          button.className = "searchable-select-option";
          button.setAttribute("role", "option");
          button.setAttribute("aria-selected", String(entry.value === select.value));
          button.dataset.optionValue = entry.value;
          button.textContent = entry.label;
          button.title = entry.label;
          button.addEventListener("pointerdown", (event) => {
            event.preventDefault();
            choose(entry);
          });
          list.append(button);
        });
        if (!matches.length) {
          const note = document.createElement("div");
          note.className = "searchable-select-empty";
          note.textContent = controller.config.emptyText || "没有匹配项。";
          list.append(note);
        }
      };
      const renderOptions = (query = "", centerSelection = false) => {
        const matches = matchingOptions(query);
        const batchSize = controller.config.limit || 50;
        let start = 0;
        let end = Math.min(matches.length, batchSize);
        if (centerSelection) {
          const selectedIndex = matches.findIndex((entry) => entry.value === select.value);
          if (selectedIndex >= 0) {
            start = Math.max(0, selectedIndex - 25);
            end = Math.min(matches.length, selectedIndex + 25);
            if (end - start < batchSize) {
              start = Math.max(0, end - batchSize);
              end = Math.min(matches.length, start + batchSize);
            }
          }
        }
        renderOptionWindow(matches, start, end);
        if (centerSelection) {
          controller.positioning = true;
          requestAnimationFrame(() => {
            list.querySelector('[aria-selected="true"]')?.scrollIntoView({ block: "center" });
            requestAnimationFrame(() => { controller.positioning = false; });
          });
        }
        return matches;
      };

      input.addEventListener("focus", () => {
        input.select();
        renderOptions("", true);
        open();
      });
      input.addEventListener("input", () => {
        renderOptions(input.value);
        open();
      });
      list.addEventListener("scroll", () => {
        if (controller.positioning || !controller.matches.length) return;
        const batchSize = controller.config.limit || 50;
        if (list.scrollTop <= 8 && controller.windowStart > 0) {
          const previousHeight = list.scrollHeight;
          const previousTop = list.scrollTop;
          renderOptionWindow(
            controller.matches,
            Math.max(0, controller.windowStart - batchSize),
            controller.windowEnd
          );
          list.scrollTop = previousTop + list.scrollHeight - previousHeight;
          return;
        }
        if (list.scrollTop + list.clientHeight >= list.scrollHeight - 8
          && controller.windowEnd < controller.matches.length) {
          renderOptionWindow(
            controller.matches,
            controller.windowStart,
            Math.min(controller.matches.length, controller.windowEnd + batchSize)
          );
        }
      });
      input.addEventListener("keydown", (event) => {
        if (event.key === "Escape") {
          event.preventDefault();
          restoreSelection();
          close();
        }
        if (event.key === "Enter") {
          const match = matchingOptions(input.value)[0];
          if (match) {
            event.preventDefault();
            choose(match);
          }
        }
      });
      input.addEventListener("blur", () => {
        restoreSelection();
        close();
      });
      select.addEventListener("change", restoreSelection);

      controller = {
        wrapper,
        input,
        config: {},
        optionSource: null,
        optionEntries: [],
        matches: [],
        windowStart: 0,
        windowEnd: 0,
        positioning: false,
        sync() {
          input.placeholder = controller.config.placeholder || "搜索并选择";
          input.setAttribute("aria-label", controller.config.ariaLabel || input.placeholder);
          input.disabled = select.disabled || (!controller.config.options && select.options.length === 0);
          restoreSelection();
          close();
        }
      };
      searchableSelectControllers.set(select, controller);
    }
    controller.config = config;
    controller.sync();
    return controller.wrapper;
  }

  function createLazySearchSelect(value, label, options, onChange, config = {}) {
    const select = document.createElement("select");
    const selected = document.createElement("option");
    selected.value = String(value);
    selected.textContent = label;
    select.append(selected);
    select.value = String(value);
    select.addEventListener("change", () => onChange(select.value, select));
    const optionSource = typeof options === "function" ? lazySearchOptions(options) : options;
    const wrapper = createSearchableSelect(select, {
      ...config,
      options: optionSource,
      limit: config.limit || 50
    });
    return { wrapper, select, input: wrapper.querySelector("input") };
  }

  function lazySearchOptions(factory) {
    let cachedOptions;
    return () => (cachedOptions ||= factory());
  }

  function renderShopContentFilter(table, previous) {
    const editor = state.catalog?.shopEditor || { shops: [], cellGroups: [] };
    const options = shopCellGroupSearchOptions(editor);
    const optionIDs = new Set(options.map((option) => option.id));
    const previousID = previous.get("ShopItemCellGroupId");
    if (!optionIDs.has(state.shopCellGroupSelection)) {
      state.shopCellGroupSelection = optionIDs.has(previousID) ? previousID : options[0]?.id || "";
    }

    const label = document.createElement("label");
    label.className = "shop-group-filter";
    label.textContent = "CellGroup";
    const select = document.createElement("select");
    select.dataset.field = "ShopItemCellGroupId";
    select.dataset.sourceFilter = "shop-group";
    options.forEach((entry) => {
      const option = document.createElement("option");
      option.value = entry.id;
      option.textContent = entry.label;
      option.dataset.searchText = entry.searchText;
      select.append(option);
    });
    select.value = state.shopCellGroupSelection;
    select.addEventListener("change", () => {
      state.shopCellGroupSelection = select.value;
      renderTable();
    });
    label.append(createSearchableSelect(select, {
      placeholder: "搜索 CellGroup、商店 ID 或名称",
      emptyText: "没有匹配的 CellGroup。"
    }));
    elements.typeFilters.append(label);
  }

  function shopCellGroupSearchOptions(editor) {
    const shopsByGroup = new Map();
    editor.shops.forEach((shop) => {
      const groupID = String(shop.shopItemCellGroupId);
      if (!shopsByGroup.has(groupID)) shopsByGroup.set(groupID, []);
      shopsByGroup.get(groupID).push(shop);
    });
    const groupIDs = [...new Set([
      ...editor.cellGroups.map((row) => String(row.shopItemCellGroupId)),
      ...editor.shops.map((shop) => String(shop.shopItemCellGroupId))
    ])].sort(compareFieldValues);
    return groupIDs.map((groupID) => {
      const references = (shopsByGroup.get(groupID) || []).map((shop) => (
        idNameLabel(shop.shopId, localizedInlineText(shop.names) || "未命名商店")
      ));
      const label = references.length
        ? `${references.join("、")} · Group ${groupID}`
        : `未被商店引用 · Group ${groupID}`;
      return {
        id: groupID,
        label,
        searchText: `${groupID} ${label}`.toLocaleLowerCase()
      };
    });
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
      label.append(definition.field === "LoginBonusId"
        ? createSearchableSelect(select, { placeholder: "搜索 LoginBonusId 或名称" })
        : select);
      elements.typeFilters.append(label);
    });
  }

  function loginBonusSourceLabel(loginBonusID) {
    const loginBonusTable = state.catalog?.tables.find((table) => table.name === "m_login_bonus");
    const loginBonus = loginBonusTable?.rows.find((row) => row.values.LoginBonusId === loginBonusID);
    const name = localizedText(loginBonus?.titles) || loginBonus?.values.LoginBonusAssetName;
    return name ? idNameLabel(loginBonusID, name) : `${loginBonusID}`;
  }

  function renderMissionSourceFilters(table, previous) {
    const sourceCatalog = state.catalog?.missionSources || { groups: [], missions: [] };
    const sourceMissions = missionSourcesForTable(table);
    const sourceGroupIDs = new Set(sourceMissions.map((mission) => String(mission.missionGroupId)));
    const sourceGroups = sourceCatalog.groups.filter((group) => sourceGroupIDs.has(String(group.missionGroupId)));
    const categoryValues = [...new Set(sourceGroups.map((group) => String(group.missionCategoryType)))];
    categoryValues.sort(compareFieldValues);
    const categorySelect = appendMissionSourceFilter(
      "任务类别", "MissionCategoryType", categoryValues,
      (value) => idNameLabel(value, missionCategoryLabels[value] || "未知任务类别"),
      previous.get("MissionCategoryType"), () => {
        renderTypeFilters(table);
        renderTable();
      }
    );
    const categoryType = categorySelect?.value;
    const groups = sourceGroups.filter((group) => String(group.missionCategoryType) === categoryType);
    const groupValues = groups.map((group) => String(group.missionGroupId));
    if (groupValues.length > 1) {
      const groupByID = new Map(groups.map((group) => [String(group.missionGroupId), group]));
      appendMissionSourceFilter(
        "任务组", "MissionGroupId", groupValues,
        (value) => missionGroupSourceLabel(groupByID.get(value)),
        previous.get("MissionGroupId"), renderTable,
        { placeholder: "搜索任务组 ID 或名称" }
      );
    }
  }

  function missionSourcesForTable(table) {
    const sources = state.catalog?.missionSources?.missions || [];
    const sourceField = table.name === "m_mission_reward" ? "MissionRewardId" : "MissionTermId";
    const sourceIDs = new Set(table.rows.map((row) => row.values[sourceField]));
    return sources.filter((mission) => sourceIDs.has(String(
      table.name === "m_mission_reward" ? mission.missionRewardId : mission.missionTermId
    )));
  }

  function appendMissionSourceFilter(labelText, field, values, optionLabel, previous, onChange, searchableConfig = null) {
    if (!values.length) return null;
    const label = document.createElement("label");
    label.textContent = labelText;
    const select = document.createElement("select");
    select.dataset.field = field;
    select.dataset.sourceFilter = "mission";
    values.forEach((value) => {
      const option = document.createElement("option");
      option.value = value;
      option.textContent = optionLabel(value);
      select.append(option);
    });
    if (values.includes(previous)) select.value = previous;
    select.addEventListener("change", onChange);
    label.append(searchableConfig ? createSearchableSelect(select, searchableConfig) : select);
    elements.typeFilters.append(label);
    return select;
  }

  function missionGroupSourceLabel(group) {
    if (!group) return "未知任务组";
    const name = localizedText(group.names) || "未命名任务组";
    return idNameLabel(group.missionGroupId, name);
  }

  function selectedMissionSources(table) {
    const sources = missionSourcesForTable(table);
    const categorySelect = elements.typeFilters.querySelector('select[data-field="MissionCategoryType"][data-source-filter="mission"]');
    if (!categorySelect) return [];
    const groupSelect = elements.typeFilters.querySelector('select[data-field="MissionGroupId"][data-source-filter="mission"]');
    if (groupSelect) {
      return sources.filter((mission) => String(mission.missionGroupId) === groupSelect.value);
    }
    const groupIDs = new Set((state.catalog?.missionSources?.groups || [])
      .filter((group) => String(group.missionCategoryType) === categorySelect.value)
      .map((group) => String(group.missionGroupId)));
    return sources.filter((mission) => groupIDs.has(String(mission.missionGroupId)));
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
      if (definition) return idNameLabel(value, definition.label);
    }
    return value;
  }

  function renderTable() {
    const table = currentTable();
    if (!table) return;
    elements.empty.textContent = "当前筛选条件下没有内容。";
    const detailed = !table.primary || state.mode === "detail";
    const isMissionReward = table.name === "m_mission_reward";
    const isMissionTerm = table.name === "m_mission_term";
    const isMissionEditor = isMissionReward || isMissionTerm;
    const isShopEditor = table.name === "m_shop_item_content_possession";
    const isQuestDropEditor = table.name === "m_quest_pickup_reward_group";
    elements.entityName.textContent = table.name;
    elements.tableName.textContent = tableDisplayName(table);
    elements.modeControl.classList.toggle("hidden", !table.primary || isMissionEditor || isShopEditor || isQuestDropEditor);
    elements.scheduleTable.classList.toggle("detail-mode", detailed);
    elements.scheduleTable.classList.toggle("hidden", isMissionEditor || isShopEditor || isQuestDropEditor);
    elements.missionRewardEditor.classList.toggle("hidden", !isMissionReward);
    elements.missionTermEditor.classList.toggle("hidden", !isMissionTerm);
    elements.shopEditor.classList.toggle("hidden", !isShopEditor);
    elements.questDropEditor.classList.toggle("hidden", !isQuestDropEditor);
    elements.tableScroll.classList.toggle("mission-reward-mode", isMissionEditor || isShopEditor || isQuestDropEditor);
    elements.tableScroll.classList.toggle("mission-term-mode", isMissionTerm);
    elements.tableScroll.classList.toggle("shop-mode", isShopEditor);
    elements.tableScroll.classList.toggle("quest-drop-mode", isQuestDropEditor);
    syncModeToggle();
    elements.head.replaceChildren();
    elements.body.replaceChildren();

    const query = state.section === "delivery" ? "" : elements.search.value.trim().toLocaleLowerCase();
    const statusFilter = elements.statusFilter.value;
    const hasSchedule = (table.pairs || []).length > 0;
    const hasArtwork = table.name === "m_dokan";
    const hasContent = table.name !== "m_mission_term" && (table.name === "m_mom_banner"
      || table.rows.some((row) => Object.keys(row.titles || {}).length > 0
        || Object.keys(row.contentBody || {}).length > 0
        || (row.contentFootnotes || []).length > 0));
    const displayedFields = displayedTableFields(table);
    const selectedSources = isMissionEditor ? selectedMissionSources(table) : [];
    elements.statusFilterLabel.classList.toggle("hidden", !hasSchedule);
    const typeFilters = [...elements.typeFilters.querySelectorAll("select")]
      .filter((select) => select.value !== "" && !select.dataset.sourceFilter)
      .map((select) => ({ field: select.dataset.field, value: select.value }));
    if (isMissionReward) {
      renderMissionRewardEditor(table, displayedFields, selectedSources, query);
      return;
    }
    if (isMissionTerm) {
      renderMissionTermEditor(table, displayedFields, selectedSources, query);
      return;
    }
    if (isShopEditor) {
      renderShopEditor(table, query);
      return;
    }
    if (isQuestDropEditor) {
      renderQuestDropEditor();
      return;
    }
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
      visibleRows.forEach((row) => elements.body.append(renderDetailedRow(
        table, row, displayedFields, hasContent, hasArtwork, hasSchedule
      )));
    }
    elements.visibleCount.textContent = `${visibleRows.length.toLocaleString()} 行`;
    elements.empty.classList.toggle("hidden", visibleRows.length !== 0);
  }

  function resetShopCellGroupDraft() {
    state.shopCellGroupDraft = (state.catalog?.shopEditor?.cellGroups || []).map((row) => ({ ...row }));
    state.shopCellGroupBaseline = JSON.stringify(state.shopCellGroupDraft.map(shopCellGroupPayload));
    state.shopCellGroupDirty = false;
    const options = shopCellGroupSearchOptions(state.catalog?.shopEditor || { shops: [], cellGroups: [] });
    if (!options.some((option) => option.id === state.shopCellGroupSelection)) {
      state.shopCellGroupSelection = options[0]?.id || "";
    }
    state.shopCellPage = 1;
    state.shopItemPage = 1;
    state.shopItemCopies = [];
    state.shopItemDeleteIDs = new Set();
    state.shopCellAdditions = [];
    state.shopCellDeleteKeys = new Map();
  }

  function resetMissionRewardDraft() {
    state.missionRewardAdditions = [];
    state.missionRewardDeleteIDs = new Set();
    state.missionRewardNextRow = -1;
    state.missionRewardContentPage = 1;
  }

  function resetQuestDropDraft() {
    const editor = state.catalog?.questDropEditor || { quests: [], groups: [], rewards: [] };
    state.questDropGroupIndex = new Map(editor.groups.map((group) => [
      String(group.questPickupRewardGroupId), group
    ]));
    const configuredQuests = state.questDropCatalog?.config?.quests || {};
    state.questDropDraft = new Map();
    state.questDropBaseline = new Map();
    editor.quests.forEach((quest) => {
      const configured = configuredQuests[String(quest.questId)];
      const rewards = (configured?.rewards || [])
        .map((reward) => ({
          battleDropRewardId: Number(reward.battleDropRewardId),
          weight: Number(reward.weight),
          guaranteed: Boolean(reward.guaranteed)
        }));
      state.questDropDraft.set(String(quest.questId), rewards);
      state.questDropBaseline.set(String(quest.questId), JSON.stringify(rewards));
    });
    state.questDropDirtyQuestIDs = new Set();
    state.questDropRewardIndex = new Map(editor.rewards.map((reward) => [String(reward.battleDropRewardId), reward]));
    state.questDropRewardReferenceIndex = new Map();
    rewardDefinitions.forEach((definition) => {
      rewardReferencesForPossessionType(definition.possessionType).forEach((reference) => {
        state.questDropRewardReferenceIndex.set(`${reference.possessionType}:${reference.possessionId}`, reference);
      });
    });
    state.questDropPage = 1;
    updateQuestDropDirtyUI();
  }

  function questDropStructuralDirty() {
    return state.questDropDirtyQuestIDs.size > 0;
  }

  function markQuestDropChanged(questID) {
    const key = String(questID);
    const current = JSON.stringify(state.questDropDraft.get(key) || []);
    if (current === state.questDropBaseline.get(key)) state.questDropDirtyQuestIDs.delete(key);
    else state.questDropDirtyQuestIDs.add(key);
    updateDirtyUI();
    updateQuestDropDirtyUI();
  }

  function questDropRewardName(reward) {
    if (!reward) return "未知掉落";
    const definition = rewardDefinitionForPossessionType(reward.possessionType);
    const reference = state.questDropRewardReferenceIndex.get(`${reward.possessionType}:${reward.possessionId}`);
    if (reference && definition) return rewardReferenceName(reference, definition).replace(/\s*\n\s*/g, " ");
    const typeLabels = { "4": "回忆", "5": "道具", "6": "消耗品", "12": "免费宝石" };
    return `${typeLabels[String(reward.possessionType)] || `类型 ${reward.possessionType}`} ${reward.possessionId}`;
  }

  function questDropRewardLabel(reward) {
    if (!reward) return "未知掉落";
    return `${reward.battleDropRewardId}. ${questDropRewardName(reward)} ×${reward.count}`;
  }

  function questDropRewardSearchText(reward) {
    const reference = state.questDropRewardReferenceIndex.get(`${reward.possessionType}:${reward.possessionId}`);
    return `${reward.battleDropRewardId} ${reward.possessionType} ${reward.possessionId} ${reward.count} ${Object.values(reference?.names || {}).join(" ")}`;
  }

  function questDropRewardOptions() {
    const editor = state.catalog?.questDropEditor || { rewards: [] };
    return [...editor.rewards].sort((left, right) => Number(left.battleDropRewardId) - Number(right.battleDropRewardId))
      .map((reward) => ({
        value: reward.battleDropRewardId,
        label: questDropRewardLabel(reward),
        searchText: questDropRewardSearchText(reward)
      }));
  }

  function renderQuestDropRewardIcon(reward, className = "quest-drop-reward-icon") {
    const definition = rewardDefinitionForPossessionType(reward?.possessionType);
    const reference = reward && state.questDropRewardReferenceIndex.get(`${reward.possessionType}:${reward.possessionId}`);
    if (definition) return renderRewardIcon(reference, definition, className);
    return renderAssetIcon("", questDropRewardName(reward), reward?.possessionType === 4 ? "忆" : "奖", className);
  }

  function setQuestDropReward(questID, index, rewardID) {
    const rewards = state.questDropDraft.get(String(questID));
    if (!rewards || !state.questDropRewardIndex.has(String(rewardID))) return;
    const normalized = Number(rewardID);
    const guaranteed = rewards[index].guaranteed;
    if (rewards.some((reward, rewardIndex) => rewardIndex !== index
      && reward.guaranteed === guaranteed
      && reward.battleDropRewardId === normalized)) {
      showNotice(`奖励 ${normalized} 在同一${guaranteed ? "必定掉落" : "随机掉落"}组中只能配置一条。`, true);
      renderTable();
      return;
    }
    rewards[index].battleDropRewardId = normalized;
    markQuestDropChanged(questID);
    renderTable();
  }

  function setQuestDropWeight(questID, index, value) {
    const rewards = state.questDropDraft.get(String(questID));
    if (!rewards) return;
    rewards[index].weight = Number(value);
    markQuestDropChanged(questID);
  }

  function moveQuestDropReward(questID, index, offset) {
    const rewards = state.questDropDraft.get(String(questID));
    if (!rewards || !rewards[index]) return;
    const guaranteed = rewards[index].guaranteed;
    const groupIndexes = rewards.map((reward, rewardIndex) => ({ reward, rewardIndex }))
      .filter((entry) => entry.reward.guaranteed === guaranteed)
      .map((entry) => entry.rewardIndex);
    const groupIndex = groupIndexes.indexOf(index);
    const target = groupIndexes[groupIndex + offset];
    if (target === undefined) return;
    [rewards[index], rewards[target]] = [rewards[target], rewards[index]];
    markQuestDropChanged(questID);
    renderTable();
  }

  function removeQuestDropReward(questID, index) {
    const rewards = state.questDropDraft.get(String(questID));
    if (!rewards) return;
    rewards.splice(index, 1);
    markQuestDropChanged(questID);
    renderTable();
  }

  function addQuestDropReward(quest, guaranteed) {
    const rewards = state.questDropDraft.get(String(quest.questId));
    if (!rewards) return;
    const normalizedGuaranteed = Boolean(guaranteed);
    const used = new Set(rewards
      .filter((reward) => reward.guaranteed === normalizedGuaranteed)
      .map((reward) => String(reward.battleDropRewardId)));
    const option = questDropRewardOptions().find((candidate) => !used.has(String(candidate.value)));
    if (!option) {
      showNotice("当前没有可用的掉落奖励。", true);
      return;
    }
    rewards.push({ battleDropRewardId: Number(option.value), weight: 1, guaranteed: normalizedGuaranteed });
    markQuestDropChanged(quest.questId);
    renderTable();
  }

  function setQuestDropPreviewReward(questID, rewardID, included) {
    const key = String(questID);
    const rewards = state.questDropDraft.get(key);
    const normalized = Number(rewardID);
    if (!rewards || !state.questDropRewardIndex.has(String(normalized))) return;
    const index = rewards.findIndex((reward) => !reward.guaranteed && reward.battleDropRewardId === normalized);
    if (included && index < 0) rewards.push({ battleDropRewardId: normalized, weight: 1, guaranteed: false });
    if (!included && index >= 0) rewards.splice(index, 1);
    markQuestDropChanged(questID);
    renderTable();
  }

  function setQuestDropCopyError(message) {
    elements.questDropCopyError.textContent = message;
    elements.questDropCopyError.classList.toggle("hidden", !message);
  }

  function resetQuestDropCopyDialog() {
    state.questDropCopyTargetID = 0;
    state.questDropCopySourceID = 0;
    state.questDropCopyConfirming = false;
    elements.questDropCopySource.value = "";
    elements.questDropCopyField.classList.remove("hidden");
    elements.questDropCopyConfirm.textContent = "读取并复制";
    setQuestDropCopyError("");
  }

  function openQuestDropCopyDialog(quest) {
    resetQuestDropCopyDialog();
    state.questDropCopyTargetID = quest.questId;
    elements.questDropCopyTitle.textContent = "复制其他副本的掉落配置";
    elements.questDropCopySummary.textContent = `当前关卡：${quest.questId}`;
    elements.questDropCopyDialog.showModal();
    elements.questDropCopySource.focus();
  }

  function applyQuestDropCopy() {
    const targetQuestID = state.questDropCopyTargetID;
    const sourceQuestID = state.questDropCopySourceID;
    const sourceRewards = state.questDropDraft.get(String(sourceQuestID)) || [];
    state.questDropDraft.set(String(targetQuestID), sourceRewards.map((reward) => ({ ...reward })));
    markQuestDropChanged(targetQuestID);
    elements.questDropCopyDialog.close();
    renderTable();
    showNotice(`已将关卡 ${sourceQuestID} 的掉落配置复制到关卡 ${targetQuestID}。`);
  }

  function copyQuestDropRewards() {
    if (state.questDropCopyConfirming) {
      applyQuestDropCopy();
      return;
    }
    const normalized = elements.questDropCopySource.value.trim();
    if (!/^\d+$/.test(normalized)) {
      setQuestDropCopyError("请输入有效的来源 QuestId。");
      return;
    }
    const sourceQuestID = Number(normalized);
    const editor = state.catalog?.questDropEditor || { quests: [] };
    if (!editor.quests.some((candidate) => candidate.questId === sourceQuestID)) {
      setQuestDropCopyError(`来源关卡 ${sourceQuestID} 不存在或不属于可配置副本。`);
      return;
    }
    if (sourceQuestID === state.questDropCopyTargetID) {
      setQuestDropCopyError("来源关卡不能与当前关卡相同。");
      return;
    }
    const sourceRewards = state.questDropDraft.get(String(sourceQuestID)) || [];
    if (!sourceRewards.length) {
      setQuestDropCopyError(`来源关卡 ${sourceQuestID} 没有可复制的掉落配置。`);
      return;
    }
    state.questDropCopySourceID = sourceQuestID;
    const currentRewards = state.questDropDraft.get(String(state.questDropCopyTargetID)) || [];
    if (currentRewards.length) {
      state.questDropCopyConfirming = true;
      elements.questDropCopyTitle.textContent = "确认覆盖当前掉落？";
      elements.questDropCopySummary.textContent = `当前关卡 ${state.questDropCopyTargetID} 已配置 ${currentRewards.length} 条掉落；来源关卡 ${sourceQuestID} 有 ${sourceRewards.length} 条。`;
      elements.questDropCopyField.classList.add("hidden");
      elements.questDropCopyConfirm.textContent = "确认覆盖";
      setQuestDropCopyError("");
      return;
    }
    applyQuestDropCopy();
  }

  function questDropTypeLabel(typeID) {
    return state.catalog?.questDropEditor?.types.find((definition) => definition.id === typeID)?.label || typeID;
  }

  function questDropChapterLabel(quest) {
    const chapter = state.catalog?.questDropEditor?.chapters.find((candidate) =>
      candidate.typeId === quest.typeId && String(candidate.chapterId) === String(quest.chapterId));
    return localizedInlineText(chapter?.names) || String(quest.chapterId);
  }

  function questDropStageLabel(quest) {
    return localizedInlineText(quest.names) || String(quest.sortOrder);
  }

  function questDropDifficultyLabel(quest) {
    const labels = { "1": "Normal", "2": "Hard", "3": "Very Hard" };
    return labels[String(quest.difficultyType)] || `难度 ${quest.difficultyType}`;
  }

  function questDropCharacterQuestCategoryLabel(value) {
    const labels = { "1": "真暗ノ巣窟", "2": "真暗ノコイン", "3": "EXガチャチケット" };
    return labels[String(value)] || `分类 ${value}`;
  }

  function renderQuestDropPickupPreview(quest) {
    const preview = document.createElement("div");
    preview.className = "quest-drop-pickup-preview";
    const group = state.questDropGroupIndex.get(String(quest.questPickupRewardGroupId));
    const rewardIDs = (group?.previewRewardIds || group?.rewards?.map((reward) => reward.battleDropRewardId) || []).slice(0, 4);
    rewardIDs.forEach((rewardID) => {
      const reward = state.questDropRewardIndex.get(String(rewardID));
      const row = document.createElement("div");
      row.className = "quest-drop-preview-row";
      row.title = questDropRewardLabel(reward);
      const toggle = document.createElement("input");
      toggle.type = "checkbox";
      toggle.className = "quest-drop-preview-toggle";
      toggle.checked = (state.questDropDraft.get(String(quest.questId)) || [])
        .some((configuredReward) => !configuredReward.guaranteed
          && configuredReward.battleDropRewardId === Number(rewardID));
      toggle.setAttribute("aria-label", `将奖励预览 ${rewardID} 加入关卡 ${quest.questId} 的掉落内容`);
      toggle.addEventListener("change", () => setQuestDropPreviewReward(quest.questId, rewardID, toggle.checked));
      const detail = document.createElement("span");
      detail.textContent = `${rewardID}. ${questDropRewardName(reward)} ×${reward?.count ?? "?"}`;
      row.append(toggle, renderQuestDropRewardIcon(reward, "quest-drop-reward-icon quest-drop-preview-icon"), detail);
      preview.append(row);
    });
    if (!rewardIDs.length) {
      const empty = document.createElement("div");
      empty.className = "quest-drop-preview-empty";
      empty.textContent = "客户端没有配置奖励预览。";
      preview.append(empty);
    }
    return preview;
  }

  function renderQuestDropRoutePreview(quest) {
    const preview = document.createElement("div");
    preview.className = "quest-drop-route-preview";
    (quest.routePossessions || []).forEach((possession) => {
      const row = document.createElement("div");
      row.className = "quest-drop-preview-row";
      const detail = document.createElement("span");
      const definition = rewardDefinitionForPossessionType(possession.possessionType);
      const reference = state.questDropRewardReferenceIndex.get(`${possession.possessionType}:${possession.possessionId}`);
      detail.textContent = reference && definition ? rewardReferenceName(reference, definition).replace(/\s*\n\s*/g, " ") : definition?.label || "未知物品";
      row.append(renderQuestDropRewardIcon(possession, "quest-drop-reward-icon quest-drop-preview-icon"), detail);
      preview.append(row);
    });
    if (!quest.routePossessions?.length) {
      const empty = document.createElement("div");
      empty.className = "quest-drop-preview-empty";
      empty.textContent = "主数据中没有该关卡的获得路径。";
      preview.append(empty);
    }
    return preview;
  }

  function renderQuestDropGroup(quest, rewards, guaranteed, optionSource) {
    const groupLabel = guaranteed ? "必定掉落" : "随机掉落";
    const entries = rewards.map((configuredReward, index) => ({ configuredReward, index }))
      .filter((entry) => entry.configuredReward.guaranteed === guaranteed);
    const section = document.createElement("section");
    section.className = "quest-drop-group";
    const heading = document.createElement("header");
    const title = document.createElement("h4");
    title.textContent = groupLabel;
    const count = document.createElement("span");
    count.textContent = `${entries.length} 条`;
    heading.append(title, count);
    const list = document.createElement("div");
    list.className = "quest-drop-list";
    entries.forEach(({ configuredReward, index }, groupIndex) => {
      const rewardID = configuredReward.battleDropRewardId;
      const reward = state.questDropRewardIndex.get(String(rewardID));
      const item = document.createElement("div");
      item.className = `quest-drop-item${guaranteed ? " quest-drop-guaranteed-item" : ""}`;
      item.append(renderQuestDropRewardIcon(reward));
      const selector = createLazySearchSelect(
        rewardID,
        questDropRewardLabel(reward),
        optionSource,
        (value) => setQuestDropReward(quest.questId, index, value),
        { placeholder: "搜索掉落 ID、物品名或数量", ariaLabel: `关卡 ${quest.questId} 第 ${groupIndex + 1} 个${groupLabel}` }
      );
      selector.input.classList.toggle("changed", state.questDropDirtyQuestIDs.has(String(quest.questId)));
      item.append(selector.wrapper);
      if (!guaranteed) {
        const weight = document.createElement("label");
        weight.className = "quest-drop-weight";
        weight.textContent = "权重";
        const weightInput = document.createElement("input");
        weightInput.type = "number";
        weightInput.min = "1";
        weightInput.max = "2147483647";
        weightInput.step = "1";
        weightInput.value = String(configuredReward.weight);
        weightInput.setAttribute("aria-label", `关卡 ${quest.questId} 奖励 ${rewardID} 权重`);
        weightInput.addEventListener("input", () => setQuestDropWeight(quest.questId, index, weightInput.value));
        weight.append(weightInput);
        item.append(weight);
      }
      const actions = document.createElement("div");
      actions.className = "quest-drop-item-actions";
      [["↑", -1], ["↓", 1]].forEach(([label, offset]) => {
        const button = document.createElement("button");
        button.type = "button";
        button.className = "order-button";
        button.textContent = label;
        button.disabled = groupIndex + offset < 0 || groupIndex + offset >= entries.length;
        button.addEventListener("click", () => moveQuestDropReward(quest.questId, index, offset));
        actions.append(button);
      });
      const remove = document.createElement("button");
      remove.type = "button";
      remove.className = "button ghost quest-drop-remove";
      remove.textContent = "移除";
      remove.addEventListener("click", () => removeQuestDropReward(quest.questId, index));
      actions.append(remove);
      item.append(actions);
      list.append(item);
    });
    if (!entries.length) {
      const empty = document.createElement("div");
      empty.className = "quest-drop-list-empty";
      empty.textContent = `本关尚未配置${groupLabel}。`;
      list.append(empty);
    }
    const add = document.createElement("button");
    add.type = "button";
    add.className = "button ghost quest-drop-add";
    add.textContent = `添加${groupLabel}`;
    add.addEventListener("click", () => addQuestDropReward(quest, guaranteed));
    section.append(heading, list, add);
    return section;
  }

  function renderQuestDropEditor() {
    const editor = state.catalog?.questDropEditor || { quests: [] };
    const query = elements.questDropSearch.value.trim().toLocaleLowerCase();
    const visible = editor.quests.filter((quest) => {
      if (state.questDropTypeFilter && quest.typeId !== state.questDropTypeFilter) return false;
      if (state.questDropChapterFilter && `${quest.typeId}:${quest.chapterId}` !== state.questDropChapterFilter) return false;
      if (state.questDropSubtypeFilter) {
        if (state.questDropTypeFilter === "main"
          && String(quest.difficultyType) !== state.questDropSubtypeFilter) return false;
        if (state.questDropTypeFilter === "event-7"
          && String(quest.subcategoryType) !== state.questDropSubtypeFilter) return false;
      }
      if (!query) return true;
      return `${quest.questId} ${questDropChapterLabel(quest)} ${questDropStageLabel(quest)} ${questDropTypeLabel(quest.typeId)}`.toLocaleLowerCase().includes(query);
    });
    state.questDropPageCount = Math.max(1, Math.ceil(visible.length / state.questDropPageSize));
    state.questDropPage = Math.min(Math.max(1, state.questDropPage), state.questDropPageCount);
    const pageStart = (state.questDropPage - 1) * state.questDropPageSize;
    const page = visible.slice(pageStart, pageStart + state.questDropPageSize);
    elements.questDropBody.replaceChildren();
    page.forEach((quest) => {
      const row = document.createElement("tr");
      row.classList.toggle("quest-drop-changed", state.questDropDirtyQuestIDs.has(String(quest.questId)));
      const identity = document.createElement("td");
      const identityContent = document.createElement("div");
      identityContent.className = "quest-drop-identity";
      const heading = document.createElement("strong");
      heading.textContent = String(quest.questId);
      const chapter = document.createElement("span");
      chapter.textContent = `${questDropChapterLabel(quest)}-${questDropStageLabel(quest)}`;
      const dropCount = document.createElement("span");
      dropCount.textContent = `总掉落数 ${quest.dropCount ?? 0}`;
      const copy = document.createElement("button");
      copy.type = "button";
      copy.className = "button ghost quest-drop-copy";
      copy.textContent = "复制自其他副本";
      copy.setAttribute("aria-label", `为关卡 ${quest.questId} 复制其他副本的掉落配置`);
      copy.addEventListener("click", () => openQuestDropCopyDialog(quest));
      identityContent.append(heading, chapter, dropCount, copy);
      identity.append(identityContent);

      const content = document.createElement("td");
      const rewards = state.questDropDraft.get(String(quest.questId)) || [];
      let cachedOptions;
      const optionSource = () => (cachedOptions ||= questDropRewardOptions());
      const groups = document.createElement("div");
      groups.className = "quest-drop-groups";
      groups.append(
        renderQuestDropGroup(quest, rewards, false, optionSource),
        renderQuestDropGroup(quest, rewards, true, optionSource)
      );
      content.append(groups);
      const preview = document.createElement("td");
      preview.append(renderQuestDropPickupPreview(quest));
      const routePreview = document.createElement("td");
      routePreview.append(renderQuestDropRoutePreview(quest));
      row.append(identity, content, preview, routePreview);
      elements.questDropBody.append(row);
    });

    elements.questDropCount.textContent = `${visible.length.toLocaleString()} 个关卡`;
    elements.visibleCount.textContent = `${visible.length.toLocaleString()} 个关卡`;
    elements.questDropPageInfo.textContent = `第 ${state.questDropPage} / ${state.questDropPageCount} 页`;
    elements.questDropPagePrevious.disabled = state.questDropPage <= 1;
    elements.questDropPageNext.disabled = state.questDropPage >= state.questDropPageCount;
    elements.empty.classList.toggle("hidden", visible.length !== 0);
  }

  function questDropReplacementPayload() {
    const config = JSON.parse(JSON.stringify(state.questDropCatalog?.config || { version: 1, quests: {} }));
    config.version = 1;
    config.quests ||= {};
    [...state.questDropDirtyQuestIDs].sort(compareFieldValues).forEach((questID) => {
      config.quests[String(questID)] = {
        rewards: (state.questDropDraft.get(String(questID)) || []).map((reward) => ({
          battleDropRewardId: reward.battleDropRewardId,
          weight: reward.weight,
          guaranteed: reward.guaranteed
        }))
      };
    });
    return config;
  }

  function questDropValidationErrors() {
    const errors = [];
    state.questDropDraft.forEach((rewards, questID) => {
      const seen = new Set();
      rewards.forEach((reward) => {
        const key = `${Boolean(reward.guaranteed)}:${reward.battleDropRewardId}`;
        if (seen.has(key)) {
          errors.push(`关卡 ${questID} 的${reward.guaranteed ? "必定掉落" : "随机掉落"}奖励 ${reward.battleDropRewardId} 重复`);
        }
        seen.add(key);
        if (!Number.isSafeInteger(reward.weight) || reward.weight < 1 || reward.weight > 2147483647) {
          errors.push(`关卡 ${questID} 的奖励 ${reward.battleDropRewardId} 权重必须是 1–2147483647 的整数`);
        }
      });
    });
    return errors;
  }

  function updateQuestDropDirtyUI() {
    if (!elements.questDropSaveSummary) return;
    const errors = questDropValidationErrors();
    if (errors.length) elements.questDropSaveSummary.textContent = `无法发布：${errors[0]}${errors.length > 1 ? `（另有 ${errors.length - 1} 项）` : ""}`;
    else if (questDropStructuralDirty()) elements.questDropSaveSummary.textContent = `${state.questDropDirtyQuestIDs.size} 个关卡的掉落配置等待发布`;
    else elements.questDropSaveSummary.textContent = "没有待发布的关卡掉落修改";
    elements.questDropSave.disabled = !questDropStructuralDirty() || errors.length > 0;
    elements.questDropDiscard.disabled = !questDropStructuralDirty();
  }

  function missionRewardStructuralDirty() {
    return state.missionRewardAdditions.length > 0 || state.missionRewardDeleteIDs.size > 0;
  }

  function missionRewardEditorRows(table) {
    return [
      ...table.rows.filter((row) => !state.missionRewardDeleteIDs.has(String(row.values.MissionRewardId))),
      ...state.missionRewardAdditions
    ].sort((left, right) => compareFieldValues(left.values.MissionRewardId, right.values.MissionRewardId));
  }

  function missionRewardReplacementPayload(table) {
    return missionRewardEditorRows(table).map((row) => ({
      missionRewardId: Number(row.values.MissionRewardId),
      possessionType: Number(effectiveValue(table.name, row, "PossessionType")),
      possessionId: Number(effectiveValue(table.name, row, "PossessionId")),
      count: Number(effectiveValue(table.name, row, "Count"))
    }));
  }

  function missionRewardReferences(rewardID) {
    return (state.catalog?.missionSources?.missions || [])
      .filter((mission) => effectiveMissionRewardID(mission) === String(rewardID));
  }

  function clearMissionRewardRowChanges(row) {
    const prefix = `m_mission_reward\u0000${row.index}\u0000`;
    [...state.dirty.keys()].forEach((key) => {
      if (key.startsWith(prefix)) state.dirty.delete(key);
    });
  }

  function addMissionReward() {
    const table = state.catalog?.tables.find((candidate) => candidate.name === "m_mission_reward");
    if (!table) return;
    const existingIDs = new Set([
      ...table.rows.map((row) => String(row.values.MissionRewardId)),
      ...state.missionRewardAdditions.map((row) => String(row.values.MissionRewardId))
    ]);
    const suggested = [...existingIDs].reduce((maximum, id) => Math.max(maximum, Number(id)), 0) + 1;
    const input = prompt("请输入全新的 RewardId：", String(suggested));
    if (input === null) return;
    const rewardID = Number(input.trim());
    if (!Number.isInteger(rewardID) || rewardID <= 0 || rewardID > 2147483647) {
      showNotice("RewardId 必须是有效的正 32 位整数。", true);
      return;
    }
    if (existingIDs.has(String(rewardID))) {
      showNotice(`RewardId ${rewardID} 已存在。`, true);
      return;
    }
    const definition = rewardDefinitions.find((candidate) => rewardReferencesForPossessionType(candidate.possessionType).length);
    const reference = definition && rewardReferencesForPossessionType(definition.possessionType)[0];
    if (!definition || !reference) {
      showNotice("当前没有可用于新奖励的发放对象。", true);
      return;
    }
    state.missionRewardAdditions.push({
      index: state.missionRewardNextRow--,
      isNew: true,
      values: {
        MissionRewardId: String(rewardID),
        PossessionType: definition.possessionType,
        PossessionId: String(reference.possessionId),
        Count: "1"
      }
    });
    elements.missionRewardContentSearch.value = String(rewardID);
    state.missionRewardContentPage = 1;
    updateDirtyUI();
    renderTable();
    showNotice(`已新增 RewardId ${rewardID} 的草稿；保存时会提交完整奖励列表。`);
  }

  function deleteMissionReward(table, row) {
    const rewardID = String(row.values.MissionRewardId);
    const references = missionRewardReferences(rewardID);
    if (references.length) {
      showNotice(`RewardId ${rewardID} 仍被 ${references.length} 个任务引用，请先改派这些任务。`, true);
      return;
    }
    const possessionType = effectiveValue(table.name, row, "PossessionType");
    const warning = ["2", "13"].includes(possessionType)
      ? "\n\n该条目发放武器或服装，增删会同时影响 GachaPool 的任务发放排除结果。"
      : "";
    if (!confirm(`删除 RewardId ${rewardID}？${warning}`)) return;
    clearMissionRewardRowChanges(row);
    if (row.isNew) {
      state.missionRewardAdditions = state.missionRewardAdditions.filter((candidate) => candidate !== row);
    } else {
      state.missionRewardDeleteIDs.add(rewardID);
    }
    updateDirtyUI();
    renderTable();
  }

  function shopCellGroupPayload(row) {
    return {
      shopItemCellGroupId: Number(row.shopItemCellGroupId),
      shopItemCellId: Number(row.shopItemCellId),
      sortOrder: Number(row.sortOrder),
      shopItemCellTermId: Number(row.shopItemCellTermId)
    };
  }

  function shopEditorItems(editor = state.catalog?.shopEditor || { items: [] }) {
    return [
      ...editor.items.filter((item) => !state.shopItemDeleteIDs.has(Number(item.shopItemId))),
      ...state.shopItemCopies
    ].sort((left, right) => Number(left.shopItemId) - Number(right.shopItemId));
  }

  function shopCellKey(cell) {
    return `${Number(cell.shopItemCellId)}:${Number(cell.stepNumber)}`;
  }

  function shopEditorCells(editor = state.catalog?.shopEditor || { cells: [] }) {
    return [
      ...editor.cells.filter((cell) => !state.shopCellDeleteKeys.has(shopCellKey(cell))),
      ...state.shopCellAdditions
    ].sort((left, right) => Number(left.shopItemCellId) - Number(right.shopItemCellId)
      || Number(left.stepNumber) - Number(right.stepNumber));
  }

  function markShopCellGroupDirty() {
    state.shopCellGroupDirty = JSON.stringify(state.shopCellGroupDraft.map(shopCellGroupPayload))
      !== state.shopCellGroupBaseline;
    updateDirtyUI();
  }

  function selectedShopCellGroupID() {
    return state.shopCellGroupSelection;
  }

  function renderShopEditor(contentTable, query) {
    const editor = state.catalog?.shopEditor || { shops: [], cells: [], items: [] };
    const items = shopEditorItems(editor);
    const cells = shopEditorCells(editor);
    const groupID = selectedShopCellGroupID();
    const references = editor.shops.filter((shop) => String(shop.shopItemCellGroupId) === groupID);
    elements.shopCellGroupReferences.textContent = references.length
      ? `引用商店：${references.map((shop) => idNameLabel(shop.shopId, localizedInlineText(shop.names) || "未命名商店")).join("、")}`
      : "引用商店：无";

    const groupRows = state.shopCellGroupDraft.map((row, draftIndex) => ({ row, draftIndex }))
      .filter(({ row }) => String(row.shopItemCellGroupId) === groupID)
      .filter(({ row }) => !query || shopCellGroupSearchText(row).includes(query));
    elements.shopCellGroupBody.replaceChildren();
    groupRows.forEach(({ row, draftIndex }) => elements.shopCellGroupBody.append(renderShopCellGroupCard(row, draftIndex, contentTable)));
    if (!groupRows.length) {
      const empty = document.createElement("div");
      empty.className = "shop-cell-group-empty";
      empty.textContent = query ? "当前 CellGroup 中没有匹配项。" : "当前 CellGroup 尚无 Cell。";
      elements.shopCellGroupBody.append(empty);
    }
    elements.shopCellGroupCount.textContent = `${groupRows.length.toLocaleString()} 条`;
    elements.shopCellGroupAdd.disabled = !groupID || !cells.length;

    renderShopCellPanel(editor);
    renderShopItemPanel(editor, contentTable);
    elements.visibleCount.textContent = `${groupRows.length.toLocaleString()} 条 CellGroup 配置 · ${cells.length.toLocaleString()} 个 Cell · ${items.length.toLocaleString()} 个 ShopItem`;
    elements.statusFilterLabel.classList.add("hidden");
    elements.empty.classList.add("hidden");
  }

  function shopCellGroupSearchText(row) {
    const cell = shopEditorCells()
      .find((candidate) => String(candidate.shopItemCellId) === String(row.shopItemCellId));
    return [row.shopItemCellId, row.sortOrder, row.shopItemCellTermId, cell ? shopCellOptionLabel(cell) : ""]
      .join(" ").toLocaleLowerCase();
  }

  function renderShopCellGroupCard(row, draftIndex, contentTable) {
    const card = document.createElement("article");
    card.className = "shop-cell-card";
    const cellDefinition = shopEditorCells()
      .find((candidate) => String(candidate.shopItemCellId) === String(row.shopItemCellId));
    const itemID = cellDefinition ? effectiveShopCellItemID(cellDefinition) : "";
    const itemDefinition = shopEditorItems()
      .find((candidate) => String(candidate.shopItemId) === itemID);
    const visual = document.createElement("div");
    visual.className = "shop-cell-card-visual";
    const cellID = document.createElement("code");
    cellID.className = "shop-cell-card-id";
    cellID.textContent = String(row.shopItemCellId);
    visual.append(renderShopCellGroupIcon(itemID, contentTable), cellID);

    const heading = document.createElement("div");
    heading.className = "shop-cell-card-heading";
    const title = document.createElement("strong");
    title.textContent = itemDefinition
      ? idNameLabel(itemID, localizedInlineText(itemDefinition.names) || "未命名商品")
      : idNameLabel(itemID || "—", "未知商品");
    title.title = title.textContent;
    heading.append(title);

    const selectSlot = document.createElement("div");
    selectSlot.className = "shop-cell-card-select";
    const cellPicker = createShopCellSelector(row.shopItemCellId, (value) => {
      state.shopCellGroupDraft[draftIndex].shopItemCellId = Number(value);
      markShopCellGroupDirty();
      renderTable();
    });
    cellPicker.input.classList.toggle("changed", state.shopCellGroupDirty);
    selectSlot.append(cellPicker.wrapper);

    const meta = document.createElement("div");
    meta.className = "shop-cell-card-meta";
    meta.append(
      makeCell("span", `SortOrder ${row.sortOrder}`),
      makeCell("code", `Term ${row.shopItemCellTermId}`),
      makeCell("span", `${shopReadonlyTime(row.startDatetime)} → ${shopReadonlyTime(row.endDatetime)}`)
    );

    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "button ghost shop-remove-cell";
    remove.textContent = "移除";
    remove.addEventListener("click", () => {
      state.shopCellGroupDraft.splice(draftIndex, 1);
      markShopCellGroupDirty();
      renderTable();
    });
    card.append(visual, heading, selectSlot, meta, remove);
    return card;
  }

  function renderShopCellGroupIcon(itemID, contentTable) {
    const copied = state.shopItemCopies.find((item) => String(item.shopItemId) === String(itemID));
    if (copied?.possessions?.length) {
      const possession = [...copied.possessions].sort((left, right) => Number(left.sortOrder) - Number(right.sortOrder))[0];
      const definition = rewardDefinitionForPossessionType(possession.possessionType);
      const reference = rewardReferencesForPossessionType(possession.possessionType)
        .find((candidate) => String(candidate.possessionId) === String(possession.possessionId));
      return renderRewardIcon(reference, definition, "shop-cell-card-icon");
    }
    const rows = contentTable.rows
      .filter((row) => row.values.ShopItemId === String(itemID))
      .sort((left, right) => Number(effectiveValue(contentTable.name, left, "SortOrder"))
        - Number(effectiveValue(contentTable.name, right, "SortOrder")) || left.index - right.index);
    const row = rows[0];
    if (!row) {
      const missing = document.createElement("div");
      missing.className = "shop-cell-card-icon";
      missing.textContent = "商";
      return missing;
    }
    const possessionType = effectiveValue(contentTable.name, row, "PossessionType");
    const possessionID = effectiveValue(contentTable.name, row, "PossessionId");
    const definition = rewardDefinitionForPossessionType(possessionType);
    const reference = rewardReferencesForPossessionType(possessionType)
      .find((candidate) => String(candidate.possessionId) === possessionID);
    return renderRewardIcon(reference, definition, "shop-cell-card-icon");
  }

  function shopReadonlyTime(milliseconds) {
    const value = Number(milliseconds || 0);
    return value ? previewChangeValue(String(value), true) : "不限";
  }

  function shopCellSelectorOptions() {
    const options = new Map();
    const itemsByID = new Map(shopEditorItems().map((item) => [String(item.shopItemId), item]));
    shopEditorCells().forEach((cell) => {
      const value = String(cell.shopItemCellId);
      if (options.has(value)) return;
      const item = itemsByID.get(effectiveShopCellItemID(cell));
      const label = idNameLabel(cell.shopItemCellId, localizedInlineText(item?.names) || "未命名商品");
      options.set(value, { value, label, searchText: `${value} ${label}` });
    });
    return [...options.values()];
  }

  function createShopCellSelector(selectedID, onChange) {
    const selected = String(selectedID);
    const selectedCell = shopEditorCells().find((cell) => String(cell.shopItemCellId) === selected);
    const label = selectedCell ? shopCellOptionLabel(selectedCell) : idNameLabel(selected, "未知 Cell");
    return createLazySearchSelect(selected, label, shopCellSelectorOptions, onChange, {
      placeholder: "搜索 CellId 或商品名称", ariaLabel: "搜索并选择 Cell", limit: 50
    });
  }

  function shopCellOptionLabel(cell) {
    const itemID = effectiveShopCellItemID(cell);
    const item = shopEditorItems().find((candidate) => String(candidate.shopItemId) === itemID);
    const name = localizedInlineText(item?.names) || "未命名商品";
    return idNameLabel(cell.shopItemCellId, name);
  }

  function effectiveShopCellItemID(cell) {
    if (cell.isNew) return String(cell.shopItemId);
    return state.dirty.get(changeKey("m_shop_item_cell", Number(cell.row), "ShopItemId"))?.value
      ?? String(cell.shopItemId);
  }

  function addShopCellGroupRow() {
    const groupID = Number(selectedShopCellGroupID());
    const cells = shopEditorCells();
    if (!groupID || !cells.length) return;
    const groupRows = state.shopCellGroupDraft.filter((row) => Number(row.shopItemCellGroupId) === groupID);
    const used = new Set(groupRows.map((row) => String(row.shopItemCellId)));
    const cell = cells.find((candidate) => !used.has(String(candidate.shopItemCellId))) || cells[0];
    const template = groupRows.at(-1);
    const sortOrder = groupRows.reduce((maximum, row) => Math.max(maximum, Number(row.sortOrder)), 0) + 1;
    const added = {
      shopItemCellGroupId: groupID,
      shopItemCellId: Number(cell.shopItemCellId),
      sortOrder,
      shopItemCellTermId: Number(template?.shopItemCellTermId || 0),
      startDatetime: Number(template?.startDatetime || 0),
      endDatetime: Number(template?.endDatetime || 0)
    };
    let insertAt = state.shopCellGroupDraft.length;
    for (let index = state.shopCellGroupDraft.length - 1; index >= 0; index -= 1) {
      if (Number(state.shopCellGroupDraft[index].shopItemCellGroupId) === groupID) {
        insertAt = index + 1;
        break;
      }
    }
    state.shopCellGroupDraft.splice(insertAt, 0, added);
    markShopCellGroupDirty();
    renderTable();
    showNotice(`已添加 Cell ${added.shopItemCellId}；SortOrder 自动设为 ${added.sortOrder}，TermId 继承为 ${added.shopItemCellTermId}。`);
  }

  const shopCellBlockerLabels = {
    m_shop_item_cell_group: "CellGroup",
    m_shop_item_cell_limited_open: "CellLimitedOpen",
    shop_cell_group_draft: "当前 CellGroup 草稿"
  };

  function shopCellBlockerLabel(tableName) {
    return shopCellBlockerLabels[tableName] || tableName;
  }

  function shopCellEffectiveBlockers(cell) {
    const blockers = [...(cell.deleteBlockers || [])];
    const referencedByDraft = state.shopCellGroupDraft.some((row) => Number(row.shopItemCellId) === Number(cell.shopItemCellId));
    if (referencedByDraft && !blockers.includes("m_shop_item_cell_group")) blockers.push("shop_cell_group_draft");
    return blockers;
  }

  function shopCellHasReferences(cell) {
    return shopCellEffectiveBlockers(cell).length !== 0;
  }

  function shopCellReferences(cell) {
    const cellID = Number(cell.shopItemCellId);
    const groupReferences = state.shopCellGroupDraft.flatMap((row, index) => (
      Number(row.shopItemCellId) === cellID ? [{
        table: "m_shop_item_cell_group",
        row: index,
        identity: [
          { name: "ShopItemCellGroupId", value: String(row.shopItemCellGroupId) },
          { name: "ShopItemCellId", value: String(row.shopItemCellId) },
          { name: "SortOrder", value: String(row.sortOrder) },
          { name: "ShopItemCellTermId", value: String(row.shopItemCellTermId) }
        ]
      }] : []
    ));
    const otherReferences = (cell.references || [])
      .filter((reference) => reference.table !== "m_shop_item_cell_group");
    return [...groupReferences, ...otherReferences];
  }

  function addShopCell() {
    const existing = [...(state.catalog?.shopEditor?.cells || []), ...state.shopCellAdditions];
    const suggestedID = existing.reduce((maximum, cell) => Math.max(maximum, Number(cell.shopItemCellId)), 0) + 1;
    const cellText = prompt("请输入新 CellId：", String(suggestedID));
    if (cellText === null) return;
    const cellID = Number(cellText.trim());
    if (!Number.isInteger(cellID) || cellID <= 0 || cellID > 2147483647) {
      showNotice("CellId 必须是有效的正 32 位整数。", true);
      return;
    }
    const stepText = prompt("请输入 StepNumber：", "1");
    if (stepText === null) return;
    const stepNumber = Number(stepText.trim());
    if (!Number.isInteger(stepNumber) || stepNumber <= 0 || stepNumber > 2147483647) {
      showNotice("StepNumber 必须是有效的正 32 位整数。", true);
      return;
    }
    if (existing.some((cell) => Number(cell.shopItemCellId) === cellID && Number(cell.stepNumber) === stepNumber)) {
      showNotice(`Cell ${cellID} / Step ${stepNumber} 已存在。`, true);
      return;
    }
    const items = shopEditorItems();
    const targetItem = state.shopItemCopies.at(-1) || items[0];
    if (!targetItem) {
      showNotice("没有可供 Cell 引用的 ShopItem。", true);
      return;
    }
    state.shopCellAdditions.push({
      row: -1, shopItemCellId: cellID, stepNumber,
      shopItemId: Number(targetItem.shopItemId), deleteBlockers: [], isNew: true
    });
    elements.shopCellSearch.value = String(cellID);
    state.shopCellPage = 1;
    updateDirtyUI();
    renderTable();
    showNotice(`已新增 Cell ${cellID} / Step ${stepNumber}，请确认其 ShopItem。`);
  }

  function deleteShopCell(cell) {
    const blockers = shopCellEffectiveBlockers(cell);
    if (blockers.length) {
      showNotice(`Cell ${cell.shopItemCellId} 仍被 ${blockers.map(shopCellBlockerLabel).join("、")} 引用，无法删除。`, true);
      return;
    }
    if (cell.isNew) {
      if (!confirm(`取消新增 Cell ${cell.shopItemCellId} / Step ${cell.stepNumber}？`)) return;
      state.shopCellAdditions = state.shopCellAdditions.filter((candidate) => candidate !== cell);
    } else {
      if (!confirm(`删除 Cell ${cell.shopItemCellId} / Step ${cell.stepNumber}？`)) return;
      state.shopCellDeleteKeys.set(shopCellKey(cell), {
        shopItemCellId: Number(cell.shopItemCellId), stepNumber: Number(cell.stepNumber)
      });
      const prefix = `${shopItemCellTableName}\u0000${Number(cell.row)}\u0000`;
      [...state.dirty.keys()].filter((key) => key.startsWith(prefix)).forEach((key) => state.dirty.delete(key));
    }
    updateDirtyUI();
    renderTable();
  }

  function shopItemCellStructuralPayload() {
    return {
      additions: state.shopCellAdditions.map((cell) => ({
        shopItemCellId: Number(cell.shopItemCellId), stepNumber: Number(cell.stepNumber),
        shopItemId: Number(cell.shopItemId)
      })),
      deletes: [...state.shopCellDeleteKeys.values()]
    };
  }

  function shopItemCellStructuralDirty() {
    return state.shopCellAdditions.length !== 0 || state.shopCellDeleteKeys.size !== 0;
  }

  function renderShopCellPanel(editor) {
    const query = elements.shopCellSearch.value.trim().toLocaleLowerCase();
    const allCells = shopEditorCells(editor);
    const rows = allCells
      .filter((cell) => !elements.shopCellUnreferenced.checked || !shopCellHasReferences(cell))
      .filter((cell) => !query || String(cell.shopItemCellId).toLocaleLowerCase().includes(query));
    const page = shopPage(rows.length, "shopCellPage", state.shopCellPageSize);
    elements.shopCellBody.replaceChildren();
    rows.slice(page.start, page.end).forEach((cell) => elements.shopCellBody.append(renderShopCellRow(cell)));
    if (!rows.length) elements.shopCellBody.append(renderMissionRewardEmptyRow(4, query ? "没有匹配该 CellId 的 Cell。" : "没有 Cell。"));
    elements.shopCellCount.textContent = shopPageCountLabel(rows.length, allCells.length, page);
    syncShopPagination("shopCell", page);
  }

  function renderShopCellRow(cell) {
    const tr = document.createElement("tr");
    const id = document.createElement("td");
    const code = document.createElement("code");
    code.textContent = String(cell.shopItemCellId);
    id.append(code);
    const step = makeCell("td", String(cell.stepNumber));
    step.className = "shop-readonly";
    const item = document.createElement("td");
    const current = effectiveShopCellItemID(cell);
    const itemPicker = createShopItemSelector(current, (value, select) => {
      if (cell.isNew) {
        cell.shopItemId = Number(value);
        updateDirtyUI();
      } else {
        onFieldChange(
          { name: shopItemCellTableName },
          { index: Number(cell.row), values: { ShopItemId: String(cell.shopItemId) } },
          { name: "ShopItemId", kind: "int32", datetime: false }, select
        );
      }
      renderTable();
    });
    itemPicker.input.classList.toggle("changed", cell.isNew || state.dirty.has(changeKey(shopItemCellTableName, Number(cell.row), "ShopItemId")));
    item.append(itemPicker.wrapper);
    const actions = document.createElement("td");
    const actionStack = document.createElement("div");
    actionStack.className = "shop-cell-row-actions";
    const references = document.createElement("button");
    references.type = "button";
    references.className = "button ghost mission-reference-button";
    references.textContent = "查找引用";
    references.addEventListener("click", () => showShopCellReferences(cell));
    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "button ghost shop-item-delete";
    remove.textContent = cell.isNew ? "取消新增" : "删除";
    remove.disabled = shopCellHasReferences(cell);
    if (remove.disabled) remove.title = `无法删除：仍被 ${shopCellEffectiveBlockers(cell).map(shopCellBlockerLabel).join("、")} 引用`;
    remove.addEventListener("click", () => deleteShopCell(cell));
    actionStack.append(references, remove);
    actions.append(actionStack);
    tr.append(id, step, item, actions);
    return tr;
  }

  const shopItemCellTableName = "m_shop_item_cell";

  function shopItemSelectorOptions() {
    return shopEditorItems().map((item) => {
      const name = localizedInlineText(item.names) || "未命名商品";
      return {
        value: String(item.shopItemId), label: idNameLabel(item.shopItemId, name),
        searchText: `${item.shopItemId} ${name} ${Object.values(item.names || {}).join(" ")}`
      };
    });
  }

  function createShopItemSelector(selectedID, onChange) {
    const selected = String(selectedID);
    const selectedItem = shopEditorItems().find((item) => String(item.shopItemId) === selected);
    const label = selectedItem
      ? idNameLabel(selectedItem.shopItemId, localizedInlineText(selectedItem.names) || "未命名商品")
      : idNameLabel(selected, "未知商品");
    return createLazySearchSelect(selected, label, shopItemSelectorOptions, onChange, {
      placeholder: "搜索 ShopItemId 或名称", ariaLabel: "搜索并选择 ShopItem", limit: 50
    });
  }

  function renderShopItemPanel(editor, contentTable) {
    const query = elements.shopItemSearch.value.trim().toLocaleLowerCase();
    const allItems = shopEditorItems(editor);
    const referencedItemIDs = new Set(shopEditorCells(editor).map((cell) => effectiveShopCellItemID(cell)));
    const rows = allItems
      .filter((item) => !elements.shopItemUnreferenced.checked || !shopItemHasReferences(item, referencedItemIDs))
      .filter((item) => !query || [
        item.shopItemId,
        localizedInlineText(item.names),
        ...Object.values(item.names || {})
      ].join(" ").toLocaleLowerCase().includes(query));
    const page = shopPage(rows.length, "shopItemPage", state.shopItemPageSize);
    elements.shopItemBody.replaceChildren();
    rows.slice(page.start, page.end).forEach((item) => elements.shopItemBody.append(renderShopItemRow(item, contentTable)));
    if (!rows.length) elements.shopItemBody.append(renderMissionRewardEmptyRow(3, query ? "没有匹配该 ShopItemId 或名称的商品。" : "没有 ShopItem。"));
    elements.shopItemCount.textContent = shopPageCountLabel(rows.length, allItems.length, page);
    syncShopPagination("shopItem", page);
  }

  function renderShopItemRow(item, contentTable) {
    const tr = document.createElement("tr");
    const identity = document.createElement("td");
    const code = document.createElement("code");
    code.textContent = String(item.shopItemId);
    const name = document.createElement("span");
    name.className = "shop-item-name";
    name.textContent = localizedInlineText(item.names) || "未命名商品";
    const actions = document.createElement("div");
    actions.className = "shop-item-actions";
    const copy = document.createElement("button");
    copy.type = "button";
    copy.className = "button ghost";
    copy.textContent = "复制";
    copy.addEventListener("click", () => copyShopItem(item, contentTable));
    const references = document.createElement("button");
    references.type = "button";
    references.className = "button ghost";
    references.textContent = "查找引用";
    references.addEventListener("click", () => showShopItemReferences(item));
    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "button ghost shop-item-delete";
    remove.textContent = item.isNew ? "取消新增" : "删除";
    const blockers = shopItemEffectiveBlockers(item);
    remove.disabled = blockers.length !== 0;
    if (remove.disabled) remove.title = `无法删除：仍被 ${blockers.map(shopItemBlockerLabel).join("、")} 引用`;
    remove.addEventListener("click", () => deleteShopItem(item));
    actions.append(copy, references, remove);
    identity.append(code, name, actions);

    const transaction = document.createElement("td");
    const transactionStack = document.createElement("div");
    transactionStack.className = "shop-transaction-stack";
    const contentSection = document.createElement("section");
    contentSection.className = "shop-transaction-section";
    contentSection.append(makeCell("h4", "发放内容"));
    const contentStack = document.createElement("div");
    contentStack.className = "shop-stack";
    const rows = item.isNew
      ? item.possessions || []
      : contentTable.rows.filter((row) => row.values.ShopItemId === String(item.shopItemId));
    const fields = ["PossessionType", "PossessionId", "Count"]
      .map((name) => contentTable.fields.find((field) => field.name === name));
    if (rows.length) {
      const contentHeader = document.createElement("div");
      contentHeader.className = "shop-content-row shop-content-header";
      ["类型", "对象", "数量"].forEach((label) => contentHeader.append(makeCell("span", label)));
      contentStack.append(contentHeader);
    }
    rows.forEach((row, rowIndex) => {
      const content = document.createElement("div");
      content.className = "shop-content-row";
      fields.forEach((field) => {
        content.append(item.isNew
          ? renderShopDraftPossessionField(item, row, rowIndex, field.name)
          : renderFieldEditor(contentTable, row, field));
      });
      contentStack.append(content);
    });
    if (!rows.length) {
      const empty = document.createElement("span");
      empty.className = "shop-content-empty";
      empty.textContent = "无 Possession 发放内容";
      contentStack.append(empty);
    }
    contentSection.append(contentStack);
    const priceSection = document.createElement("section");
    priceSection.className = "shop-transaction-section shop-price-section";
    priceSection.append(makeCell("h4", "价格"), renderShopPriceEditor(item));
    transactionStack.append(contentSection, priceSection);
    transaction.append(transactionStack);

    const stock = document.createElement("td");
    stock.append(renderShopStockEditor(item));

    tr.append(identity, transaction, stock);
    return tr;
  }

  const shopItemBlockerLabels = {
    m_shop_item_cell: "Cell",
    m_shop_item_content_effect: "Effect 发放内容",
    m_shop_item_content_mission: "Mission 发放内容",
    m_shop_item_user_level_condition: "等级附加内容",
    shop_item_cell_draft: "当前 Cell 草稿"
  };

  function shopItemBlockerLabel(tableName) {
    return shopItemBlockerLabels[tableName] || tableName;
  }

  function shopItemEffectiveBlockers(item, referencedItemIDs = null) {
    const blockers = [...(item.deleteBlockers || [])];
    const referencedByCell = referencedItemIDs
      ? referencedItemIDs.has(String(item.shopItemId))
      : shopEditorCells().some((cell) => Number(effectiveShopCellItemID(cell)) === Number(item.shopItemId));
    if (referencedByCell && !blockers.includes("m_shop_item_cell")) blockers.push("shop_item_cell_draft");
    return blockers;
  }

  function shopItemHasReferences(item, referencedItemIDs = null) {
    return shopItemEffectiveBlockers(item, referencedItemIDs).length !== 0;
  }

  function shopItemReferences(item) {
    const itemID = Number(item.shopItemId);
    const cellReferences = shopEditorCells().flatMap((cell) => (
      Number(effectiveShopCellItemID(cell)) === itemID ? [{
        table: "m_shop_item_cell",
        row: Number(cell.row),
        identity: [
          { name: "ShopItemCellId", value: String(cell.shopItemCellId) },
          { name: "StepNumber", value: String(cell.stepNumber) },
          { name: "ShopItemId", value: String(item.shopItemId) }
        ]
      }] : []
    ));
    const otherReferences = (item.references || [])
      .filter((reference) => reference.table !== "m_shop_item_cell");
    return [...cellReferences, ...otherReferences];
  }

  function renderShopStockEditor(item) {
    const stack = document.createElement("div");
    stack.className = "shop-stack";
    const select = document.createElement("select");
    const current = effectiveShopItemValue(item, "ShopItemLimitedStockId", item.shopItemLimitedStockId);
    const unlimited = document.createElement("option");
    unlimited.value = "0";
    unlimited.textContent = idNameLabel(0, "不限库存");
    select.append(unlimited);
    (state.catalog?.shopEditor?.stocks || []).forEach((stock) => {
      const option = document.createElement("option");
      option.value = String(stock.shopItemLimitedStockId);
      option.textContent = `${stock.shopItemLimitedStockId}. 上限 ${stock.maxCount}`;
      select.append(option);
    });
    if (![...select.options].some((option) => option.value === current)) {
      const option = document.createElement("option");
      option.value = current;
      option.textContent = idNameLabel(current, "未知库存配置");
      select.append(option);
    }
    select.value = current;
    configureShopItemSelect(select, item, "ShopItemLimitedStockId", item.shopItemLimitedStockId, () => renderTable());
    stack.append(select);

    const detail = document.createElement("div");
    detail.className = "shop-stack shop-readonly shop-stock-detail";
    const stock = (state.catalog?.shopEditor?.stocks || [])
      .find((candidate) => String(candidate.shopItemLimitedStockId) === current);
    if (stock) {
      detail.append(
        makeCell("span", `上限 ${stock.maxCount}`),
        makeCell("span", `重置类型 ${stock.autoResetType}`),
        makeCell("span", `重置周期 ${stock.autoResetPeriod}`)
      );
    } else {
      detail.textContent = current === "0" ? "不限库存" : "库存配置不存在";
    }
    stack.append(detail);
    return stack;
  }

  function renderShopPriceEditor(item) {
    const stack = document.createElement("div");
    stack.className = "shop-stack";
    const includesPriceID = effectiveShopItemValue(item, "PriceType", item.priceType) === "1";
    const header = document.createElement("div");
    header.className = "shop-content-row shop-content-header shop-price-row";
    const labels = includesPriceID
      ? ["PriceType", "PriceId", "Price", "RegularPrice"]
      : ["PriceType", "Price", "RegularPrice"];
    labels.forEach((label) => header.append(makeCell("span", label)));
    const fields = document.createElement("div");
    fields.className = "shop-content-row shop-price-row";
    const controls = [renderShopPriceTypeSelect(item)];
    if (includesPriceID) controls.push(renderShopPriceIDEditor(item));
    controls.push(
      renderShopItemInput(item, "Price", item.price, "价格"),
      renderShopItemInput(item, "RegularPrice", item.regularPrice, "原价")
    );
    if (!includesPriceID) {
      header.classList.add("without-price-id");
      fields.classList.add("without-price-id");
    }
    fields.append(...controls);
    stack.append(header, fields);
    return stack;
  }

  const shopPriceTypeDisplays = {
    "1": { name: "消耗品", glyph: "消" },
    "2": { name: "免费宝石", glyph: "石" },
    "3": { name: "付费宝石", glyph: "石" },
    "4": { name: "平台支付", glyph: "￥" }
  };

  function renderShopPriceTypeSelect(item) {
    const select = document.createElement("select");
    const current = effectiveShopItemValue(item, "PriceType", item.priceType);
    [...new Set(["1", "2", "3", "4", current])].sort(compareFieldValues).forEach((value) => {
      const option = document.createElement("option");
      option.value = value;
      option.textContent = idNameLabel(value, shopPriceTypeDisplays[value]?.name || "未知类型");
      select.append(option);
    });
    select.value = current;
    configureShopItemSelect(select, item, "PriceType", item.priceType, () => renderTable());
    return select;
  }

  function renderShopPriceIDEditor(item) {
    const wrapper = document.createElement("div");
    wrapper.className = "field-editor reward-id-field-editor";
    const priceType = effectiveShopItemValue(item, "PriceType", item.priceType);
    let icon = renderShopPriceIcon(priceType, effectiveShopItemValue(item, "PriceId", item.priceId));
    const select = renderShopPriceIDSelect(item, () => {
      const nextIcon = renderShopPriceIcon(priceType, select.value);
      icon.replaceWith(nextIcon);
      icon = nextIcon;
    });
    const selectSlot = document.createElement("div");
    selectSlot.className = "reward-field-select";
    selectSlot.append(select);
    wrapper.append(icon, selectSlot);
    return wrapper;
  }

  function renderShopPriceIcon(priceType, priceID) {
    if (priceType === "1") {
      const definition = rewardDefinitionForPossessionType("6");
      const reference = rewardReferencesForPossessionType("6")
        .find((candidate) => String(candidate.possessionId) === String(priceID));
      return renderRewardIcon(reference, definition, "reward-field-icon");
    }
    const display = shopPriceTypeDisplays[priceType] || { name: "未知价格类型", glyph: "价" };
    return renderRewardIcon(null, { fallbackName: display.name, glyph: display.glyph }, "reward-field-icon");
  }

  function renderShopPriceIDSelect(item, afterChange) {
    const select = document.createElement("select");
    const priceType = effectiveShopItemValue(item, "PriceType", item.priceType);
    const current = effectiveShopItemValue(item, "PriceId", item.priceId);
    if (priceType === "1") {
      const definition = rewardDefinitionForPossessionType("6");
      const references = rewardReferencesForPossessionType("6");
      references.forEach((reference) => {
        const option = document.createElement("option");
        option.value = String(reference.possessionId);
        option.textContent = rewardReferenceOptionLabel(reference, definition);
        select.append(option);
      });
    } else {
      const option = document.createElement("option");
      option.value = "0";
      option.textContent = idNameLabel(0, "不使用 PriceId");
      select.append(option);
    }
    if (![...select.options].some((option) => option.value === current)) {
      const option = document.createElement("option");
      option.value = current;
      option.textContent = idNameLabel(current, "当前 PriceId");
      select.append(option);
    }
    select.value = current;
    configureShopItemSelect(select, item, "PriceId", item.priceId, afterChange);
    return select;
  }

  const shopItemFieldProperties = {
    PriceType: "priceType",
    PriceId: "priceId",
    Price: "price",
    RegularPrice: "regularPrice",
    ShopItemLimitedStockId: "shopItemLimitedStockId"
  };

  function renderShopDraftPossessionField(item, possession, rowIndex, fieldName) {
    if (fieldName === "PossessionType") {
      const wrapper = document.createElement("div");
      wrapper.className = "field-editor reward-type-field-editor";
      const select = document.createElement("select");
      const current = String(possession.possessionType);
      rewardDefinitions.forEach((definition) => {
        if (!rewardReferencesForPossessionType(definition.possessionType).length) return;
        const option = document.createElement("option");
        option.value = definition.possessionType;
        option.textContent = idNameLabel(definition.possessionType, definition.label);
        select.append(option);
      });
      if (![...select.options].some((option) => option.value === current)) {
        const option = document.createElement("option");
        option.value = current;
        option.textContent = idNameLabel(current, "未知类型");
        select.append(option);
      }
      select.value = current;
      select.classList.add("changed");
      select.addEventListener("change", () => {
        possession.possessionType = Number(select.value);
        const references = rewardReferencesForPossessionType(select.value);
        if (!references.some((reference) => Number(reference.possessionId) === Number(possession.possessionId)) && references.length) {
          possession.possessionId = Number(references[0].possessionId);
        }
        renderTable();
      });
      wrapper.append(select);
      return wrapper;
    }
    if (fieldName === "PossessionId") {
      const wrapper = document.createElement("div");
      wrapper.className = "field-editor reward-id-field-editor";
      const select = document.createElement("select");
      const possessionType = String(possession.possessionType);
      const current = String(possession.possessionId);
      const definition = rewardDefinitionForPossessionType(possessionType);
      const references = rewardReferencesForPossessionType(possessionType);
      const reference = references.find((candidate) => String(candidate.possessionId) === current);
      populateRewardIDSelect(select, references, current, definition);
      select.classList.add("changed");
      select.addEventListener("change", () => {
        possession.possessionId = Number(select.value);
        renderTable();
      });
      const selectSlot = document.createElement("div");
      selectSlot.className = "reward-field-select";
      const searchable = createSearchableSelect(select, {
        options: lazySearchOptions(() => rewardSelectorOptions(references, definition)),
        placeholder: "搜索奖励对象 ID 或名称",
        ariaLabel: "搜索并选择奖励对象",
        limit: 50
      });
      searchable.querySelector("input").classList.add("changed");
      selectSlot.append(searchable);
      wrapper.append(renderRewardIcon(reference, definition, "reward-field-icon"), selectSlot);
      return wrapper;
    }
    const wrapper = document.createElement("div");
    wrapper.className = "field-editor";
    const input = document.createElement("input");
    input.type = "text";
    input.inputMode = "numeric";
    input.value = String(possession.count);
    input.classList.add("changed");
    input.setAttribute("aria-label", `${item.shopItemId} 发放内容 ${rowIndex + 1} 数量`);
    input.addEventListener("input", () => {
      const value = input.value.trim();
      if (!/^\d+$/.test(value) || BigInt(value) <= 0n || BigInt(value) > 2147483647n) {
        input.classList.add("invalid");
        showNotice("Count 必须是有效的正 32 位整数。", true);
        return;
      }
      input.classList.remove("invalid");
      possession.count = Number(value);
      updateDirtyUI();
    });
    wrapper.append(input);
    return wrapper;
  }

  function shopItemPossessionsForCopy(source, contentTable, shopItemID) {
    if (source.isNew) {
      return (source.possessions || []).map((row) => ({ ...row, shopItemId: shopItemID }));
    }
    return contentTable.rows
      .filter((row) => row.values.ShopItemId === String(source.shopItemId))
      .map((row) => ({
        shopItemId: shopItemID,
        possessionType: Number(effectiveValue(contentTable.name, row, "PossessionType")),
        possessionId: Number(effectiveValue(contentTable.name, row, "PossessionId")),
        sortOrder: Number(row.values.SortOrder),
        count: Number(effectiveValue(contentTable.name, row, "Count"))
      }));
  }

  function copyShopItem(source, contentTable) {
    const existing = [...(state.catalog?.shopEditor?.items || []), ...state.shopItemCopies];
    const suggested = existing.reduce((maximum, item) => Math.max(maximum, Number(item.shopItemId)), 0) + 1;
    const input = prompt(`复制 ShopItem ${source.shopItemId}。请输入新 ShopItemId：`, String(suggested));
    if (input === null) return;
    const itemID = Number(input.trim());
    if (!Number.isInteger(itemID) || itemID <= 0 || itemID > 2147483647) {
      showNotice("ShopItemId 必须是有效的正 32 位整数。", true);
      return;
    }
    if (existing.some((item) => Number(item.shopItemId) === itemID)) {
      showNotice(`ShopItemId ${itemID} 已存在。`, true);
      return;
    }
    const copied = {
      ...source,
      row: -1,
      sourceShopItemId: Number(source.sourceShopItemId || source.shopItemId),
      shopItemId: itemID,
      priceType: Number(effectiveShopItemValue(source, "PriceType", source.priceType)),
      priceId: Number(effectiveShopItemValue(source, "PriceId", source.priceId)),
      price: Number(effectiveShopItemValue(source, "Price", source.price)),
      regularPrice: Number(effectiveShopItemValue(source, "RegularPrice", source.regularPrice)),
      shopItemLimitedStockId: Number(effectiveShopItemValue(source, "ShopItemLimitedStockId", source.shopItemLimitedStockId)),
      possessions: shopItemPossessionsForCopy(source, contentTable, itemID),
      names: { ...(source.names || {}) },
      deleteBlockers: [],
      isNew: true
    };
    state.shopItemCopies.push(copied);
    elements.shopItemSearch.value = String(itemID);
    state.shopItemPage = 1;
    updateDirtyUI();
    renderTable();
    showNotice(`已复制为 ShopItem ${itemID}，并继承其 Possession 发放内容。`);
  }

  function deleteShopItem(item) {
    const blockers = shopItemEffectiveBlockers(item);
    if (blockers.length) {
      showNotice(`ShopItem ${item.shopItemId} 仍被 ${blockers.map(shopItemBlockerLabel).join("、")} 引用，无法删除。`, true);
      return;
    }
    if (item.isNew) {
      if (!confirm(`取消新增 ShopItem ${item.shopItemId}？`)) return;
      state.shopItemCopies = state.shopItemCopies.filter((candidate) => candidate !== item);
      updateDirtyUI();
      renderTable();
      return;
    }
    if (!confirm(`删除孤立 ShopItem ${item.shopItemId}？此操作会在应用变更时再次检查引用。`)) return;
    state.shopItemDeleteIDs.add(Number(item.shopItemId));
    const prefix = `m_shop_item\u0000${Number(item.row)}\u0000`;
    [...state.dirty.keys()].filter((key) => key.startsWith(prefix)).forEach((key) => state.dirty.delete(key));
    const possessionTable = state.catalog?.tables?.find((table) => table.name === "m_shop_item_content_possession");
    (possessionTable?.rows || [])
      .filter((row) => Number(row.values.ShopItemId) === Number(item.shopItemId))
      .forEach((row) => {
        const possessionPrefix = `m_shop_item_content_possession\u0000${Number(row.index)}\u0000`;
        [...state.dirty.keys()].filter((key) => key.startsWith(possessionPrefix)).forEach((key) => state.dirty.delete(key));
      });
    updateDirtyUI();
    renderTable();
    showNotice(`已将孤立 ShopItem ${item.shopItemId} 标记为删除。`);
  }

  function shopItemPayload(item) {
    return {
      sourceShopItemId: Number(item.sourceShopItemId),
      shopItemId: Number(item.shopItemId),
      nameShopTextId: Number(item.nameShopTextId),
      descriptionShopTextId: Number(item.descriptionShopTextId),
      shopItemContentType: Number(item.shopItemContentType),
      priceType: Number(effectiveShopItemValue(item, "PriceType", item.priceType)),
      priceId: Number(effectiveShopItemValue(item, "PriceId", item.priceId)),
      price: Number(effectiveShopItemValue(item, "Price", item.price)),
      regularPrice: Number(effectiveShopItemValue(item, "RegularPrice", item.regularPrice)),
      shopPromotionType: Number(item.shopPromotionType),
      shopItemLimitedStockId: Number(effectiveShopItemValue(item, "ShopItemLimitedStockId", item.shopItemLimitedStockId)),
      assetCategoryId: Number(item.assetCategoryId),
      assetVariationId: Number(item.assetVariationId),
      shopItemDecorationType: Number(item.shopItemDecorationType),
      possessions: (item.possessions || []).map((row) => ({
        shopItemId: Number(item.shopItemId),
        possessionType: Number(row.possessionType),
        possessionId: Number(row.possessionId),
        sortOrder: Number(row.sortOrder),
        count: Number(row.count)
      }))
    };
  }

  function shopItemStructuralPayload() {
    return {
      copies: state.shopItemCopies.map(shopItemPayload),
      deleteIds: [...state.shopItemDeleteIDs].sort((left, right) => left - right)
    };
  }

  function shopItemStructuralDirty() {
    return state.shopItemCopies.length !== 0 || state.shopItemDeleteIDs.size !== 0;
  }

  function configureShopItemSelect(select, item, fieldName, original, afterChange) {
    if (item.isNew) {
      select.classList.add("changed");
      select.addEventListener("change", () => {
        item[shopItemFieldProperties[fieldName]] = Number(select.value);
        updateDirtyUI();
        afterChange?.();
      });
      return;
    }
    const key = changeKey("m_shop_item", Number(item.row), fieldName);
    select.classList.toggle("changed", state.dirty.has(key));
    select.addEventListener("change", () => {
      onFieldChange(
        { name: "m_shop_item" },
        { index: Number(item.row), values: { [fieldName]: String(original) } },
        { name: fieldName, kind: "int32", datetime: false }, select
      );
      afterChange?.();
    });
  }

  function renderShopItemInput(item, fieldName, original, label) {
    const input = document.createElement("input");
    input.type = "text";
    input.inputMode = "numeric";
    input.value = effectiveShopItemValue(item, fieldName, original);
    input.placeholder = label;
    input.setAttribute("aria-label", `${item.shopItemId} ${label}`);
    input.classList.toggle("changed", item.isNew || state.dirty.has(changeKey("m_shop_item", Number(item.row), fieldName)));
    if (item.isNew) {
      input.addEventListener("input", () => {
        const value = input.value.trim();
        if (!/^-?\d+$/.test(value) || BigInt(value) < -2147483648n || BigInt(value) > 2147483647n
          || ["Price", "RegularPrice"].includes(fieldName) && Number(value) < 0) {
          input.classList.add("invalid");
          showNotice(`${fieldName} 必须是有效的非负 32 位整数。`, true);
          return;
        }
        input.classList.remove("invalid");
        item[shopItemFieldProperties[fieldName]] = Number(value);
        updateDirtyUI();
      });
    } else {
      input.addEventListener("input", () => onFieldChange(
        { name: "m_shop_item" },
        { index: Number(item.row), values: { [fieldName]: String(original) } },
        { name: fieldName, kind: "int32", datetime: false }, input
      ));
    }
    return input;
  }

  function effectiveShopItemValue(item, fieldName, original) {
    if (item.isNew) return String(item[shopItemFieldProperties[fieldName]]);
    return state.dirty.get(changeKey("m_shop_item", Number(item.row), fieldName))?.value ?? String(original);
  }

  function shopPage(rowCount, stateField, pageSize) {
    const pageCount = Math.max(1, Math.ceil(rowCount / pageSize));
    state[stateField] = Math.min(Math.max(1, state[stateField]), pageCount);
    const start = (state[stateField] - 1) * pageSize;
    return { page: state[stateField], pageCount, start, end: Math.min(start + pageSize, rowCount) };
  }

  function shopPageCountLabel(filtered, total, page) {
    const rowCount = filtered === total ? `${total.toLocaleString()} 行` : `${filtered.toLocaleString()} / ${total.toLocaleString()} 行`;
    return filtered ? `${rowCount} · ${page.start + 1}–${page.end}` : rowCount;
  }

  function syncShopPagination(prefix, page) {
    state[`${prefix}PageCount`] = page.pageCount;
    elements[`${prefix}PageInfo`].textContent = `第 ${page.page.toLocaleString()} / ${page.pageCount.toLocaleString()} 页`;
    elements[`${prefix}PagePrevious`].disabled = page.page === 1;
    elements[`${prefix}PageNext`].disabled = page.page === page.pageCount;
  }

  function renderMissionRewardEditor(table, fields, sources, query) {
    const visibleSources = sources.filter((source) => {
      if (!query) return true;
      const rewardID = effectiveMissionRewardID(source);
      return [
        source.missionId,
        rewardID,
        ...Object.values(source.names || {}),
        missionRewardOptionLabel(table, rewardID)
      ].join(" ").toLocaleLowerCase().includes(query);
    });
    elements.missionRewardAssignmentBody.replaceChildren();
    visibleSources.forEach((source) => elements.missionRewardAssignmentBody.append(renderMissionRewardAssignmentRow(table, source)));
    if (!visibleSources.length) elements.missionRewardAssignmentBody.append(renderMissionRewardEmptyRow(3, "当前筛选条件下没有任务。"));
    const rewardIDQuery = elements.missionRewardContentSearch.value.trim().toLocaleLowerCase();
    const rewardRows = missionRewardEditorRows(table);
    const unreferencedOnly = elements.missionRewardContentUnreferenced.checked;
    const referencedRewardIDs = unreferencedOnly
      ? new Set((state.catalog?.missionSources?.missions || []).map(effectiveMissionRewardID))
      : null;
    const visibleRewardRows = rewardRows.filter((row) => (
      (!referencedRewardIDs || !referencedRewardIDs.has(String(row.values.MissionRewardId)))
      && (!rewardIDQuery || String(row.values.MissionRewardId).toLocaleLowerCase().includes(rewardIDQuery))
    ));
    const pageCount = Math.max(1, Math.ceil(visibleRewardRows.length / state.missionRewardContentPageSize));
    state.missionRewardContentPage = Math.min(Math.max(1, state.missionRewardContentPage), pageCount);
    state.missionRewardContentPageCount = pageCount;
    const pageStart = (state.missionRewardContentPage - 1) * state.missionRewardContentPageSize;
    const pageEnd = Math.min(pageStart + state.missionRewardContentPageSize, visibleRewardRows.length);
    elements.missionRewardContentBody.replaceChildren();
    visibleRewardRows.slice(pageStart, pageEnd)
      .forEach((row) => elements.missionRewardContentBody.append(renderMissionRewardContentRow(table, fields, row)));
    if (!visibleRewardRows.length) elements.missionRewardContentBody.append(renderMissionRewardEmptyRow(
      5, rewardIDQuery || unreferencedOnly ? "没有匹配当前筛选条件的奖励内容。" : "当前没有奖励内容。"
    ));
    elements.missionRewardAssignmentCount.textContent = `${visibleSources.length.toLocaleString()} 个任务`;
    const rewardRowCount = rewardIDQuery || unreferencedOnly
      ? `${visibleRewardRows.length.toLocaleString()} / ${rewardRows.length.toLocaleString()} 行`
      : `${rewardRows.length.toLocaleString()} 行`;
    elements.missionRewardContentCount.textContent = visibleRewardRows.length
      ? `${rewardRowCount} · ${pageStart + 1}–${pageEnd}`
      : rewardRowCount;
    elements.missionRewardContentPageInfo.textContent = `第 ${state.missionRewardContentPage.toLocaleString()} / ${pageCount.toLocaleString()} 页`;
    elements.missionRewardContentPagePrevious.disabled = state.missionRewardContentPage === 1;
    elements.missionRewardContentPageNext.disabled = state.missionRewardContentPage === pageCount;
    elements.visibleCount.textContent = `${visibleSources.length.toLocaleString()} 个任务 · ${rewardRowCount.replace(" 行", " 条奖励内容")}`;
    elements.empty.classList.add("hidden");
  }

  function renderMissionRewardAssignmentRow(table, source) {
    const tr = document.createElement("tr");
    const missionID = document.createElement("td");
    const missionIDValue = document.createElement("code");
    missionIDValue.textContent = String(source.missionId);
    missionID.append(missionIDValue);
    const description = makeCell("td", localizedInlineText(source.names) || "未命名任务");
    description.className = "mission-reward-description";
    const reward = document.createElement("td");
    const editor = document.createElement("div");
    editor.className = "mission-reward-assignment-editor";
    const rewardID = effectiveMissionRewardID(source);
    const preview = renderMissionRewardInlinePreview(table, rewardID);
    const select = document.createElement("select");
    select.className = "mission-reward-assignment-select";
    populateMissionRewardSelect(select, table, rewardID, false);
    select.dataset.table = "m_mission";
    select.dataset.row = String(source.row);
    select.dataset.field = "MissionRewardId";
    select.setAttribute("aria-label", `${description.textContent} 的 RewardId`);
    select.classList.toggle("changed", state.dirty.has(changeKey("m_mission", source.row, "MissionRewardId")));
    const populate = () => populateMissionRewardSelect(select, table, select.value, true);
    select.addEventListener("focus", populate);
    select.addEventListener("pointerdown", populate);
    select.addEventListener("change", () => {
      onFieldChange(
        { name: "m_mission" },
        { index: source.row, values: { MissionRewardId: String(source.missionRewardId) } },
        { name: "MissionRewardId", kind: "int32", datetime: false },
        select
      );
      renderTable();
    });
    editor.append(preview, select);
    reward.append(editor);
    tr.append(missionID, description, reward);
    return tr;
  }

  function renderMissionRewardContentRow(table, fields, row) {
    const tr = document.createElement("tr");
    const rewardID = document.createElement("td");
    const rewardIDValue = document.createElement("code");
    rewardIDValue.textContent = row.values.MissionRewardId;
    rewardID.append(rewardIDValue);
    tr.append(rewardID);
    fields.forEach((field) => {
      const cell = document.createElement("td");
      cell.append(renderFieldEditor(table, row, field));
      tr.append(cell);
    });
    const referenceCell = document.createElement("td");
    const referenceButton = document.createElement("button");
    referenceButton.type = "button";
    referenceButton.className = "button ghost mission-reference-button";
    referenceButton.textContent = "查找引用";
    referenceButton.setAttribute("aria-label", `查找 RewardId ${row.values.MissionRewardId} 的任务引用`);
    referenceButton.addEventListener("click", () => showMissionReferences("reward", row.values.MissionRewardId));
    const deleteButton = document.createElement("button");
    deleteButton.type = "button";
    deleteButton.className = "button ghost mission-reward-delete";
    deleteButton.textContent = "删除";
    const references = missionRewardReferences(row.values.MissionRewardId);
    deleteButton.disabled = references.length > 0;
    deleteButton.title = references.length ? `仍被 ${references.length} 个任务引用；请先改派` : "删除该奖励内容";
    deleteButton.addEventListener("click", () => deleteMissionReward(table, row));
    const actions = document.createElement("div");
    actions.className = "mission-reward-row-actions";
    actions.append(referenceButton, deleteButton);
    referenceCell.append(actions);
    tr.append(referenceCell);
    return tr;
  }

  function showMissionReferences(referenceType, referenceID) {
    const isTerm = referenceType === "term";
    const idLabel = isTerm ? "TermId" : "RewardId";
    const referenceLabel = isTerm ? "期限" : "奖励";
    const effectiveReferenceID = isTerm ? effectiveMissionTermID : effectiveMissionRewardID;
    const references = (state.catalog?.missionSources?.missions || [])
      .filter((mission) => effectiveReferenceID(mission) === String(referenceID));
    const groupByID = new Map((state.catalog?.missionSources?.groups || [])
      .map((group) => [String(group.missionGroupId), group]));
    const categories = new Map();
    references.forEach((mission) => {
      const group = groupByID.get(String(mission.missionGroupId));
      const categoryType = String(group?.missionCategoryType ?? "unknown");
      if (!categories.has(categoryType)) categories.set(categoryType, new Map());
      const groups = categories.get(categoryType);
      const groupID = String(mission.missionGroupId);
      if (!groups.has(groupID)) groups.set(groupID, { group, missions: [] });
      groups.get(groupID).missions.push(mission);
    });

    const groupCount = new Set(references.map((mission) => String(mission.missionGroupId))).size;
    elements.missionReferenceEyebrow.textContent = isTerm ? "MISSION TERM REFERENCES" : "MISSION REWARD REFERENCES";
    elements.missionReferenceTitle.textContent = `${idLabel} ${referenceID} 的任务引用`;
    elements.missionReferenceSummary.textContent = references.length
      ? `${references.length.toLocaleString()} 个任务 · ${categories.size.toLocaleString()} 个类型 · ${groupCount.toLocaleString()} 个任务组`
      : `当前没有任务引用此${referenceLabel}`;
    elements.missionReferenceContent.replaceChildren();

    if (!references.length) {
      const empty = document.createElement("div");
      empty.className = "impact-section impact-no-change";
      empty.textContent = `没有找到引用该 ${idLabel} 的任务。`;
      elements.missionReferenceContent.append(empty);
    } else {
      categories.forEach((groups, categoryType) => {
        const category = document.createElement("section");
        category.className = "impact-group";
        const categoryHeading = document.createElement("header");
        const categoryTitle = document.createElement("strong");
        categoryTitle.textContent = missionCategoryLabels[categoryType] || `任务类型 ${categoryType}`;
        const categoryCount = document.createElement("span");
        const missionCount = [...groups.values()].reduce((total, entry) => total + entry.missions.length, 0);
        categoryCount.textContent = `${missionCount.toLocaleString()} 个任务 · ${groups.size.toLocaleString()} 个任务组`;
        categoryHeading.append(categoryTitle, categoryCount);
        category.append(categoryHeading);
        groups.forEach(({ group, missions }, groupID) => {
          const groupLabel = group ? missionGroupSourceLabel(group) : idNameLabel(groupID, "未知任务组");
          category.append(renderImpactSection(
            `${groupLabel} · ${missions.length.toLocaleString()} 个任务`,
            missions.map((mission) => ({
              table: "m_mission",
              tableLabel: "Mission",
              row: mission.row,
              identity: [{ name: "MissionId", value: String(mission.missionId) }],
              titles: mission.names,
              omitChanges: true
            }))
          ));
        });
        elements.missionReferenceContent.append(category);
      });
    }
    elements.missionReferenceDialog.showModal();
  }

  function showShopCellReferences(cell) {
    showShopReferences("Cell", cell.shopItemCellId, shopCellReferences(cell), shopCellBlockerLabel);
  }

  function showShopItemReferences(item) {
    showShopReferences("ShopItem", item.shopItemId, shopItemReferences(item), shopItemBlockerLabel);
  }

  function showShopReferences(entityLabel, entityID, references, tableLabel) {
    const groups = new Map();
    references.forEach((reference) => {
      if (!groups.has(reference.table)) groups.set(reference.table, []);
      groups.get(reference.table).push(reference);
    });
    elements.missionReferenceEyebrow.textContent = `${entityLabel.toUpperCase()} REFERENCES`;
    elements.missionReferenceTitle.textContent = `${entityLabel} ${entityID} 的引用`;
    elements.missionReferenceSummary.textContent = references.length
      ? `${references.length.toLocaleString()} 条引用 · ${groups.size.toLocaleString()} 张表`
      : `当前没有内容引用此 ${entityLabel}`;
    elements.missionReferenceContent.replaceChildren();
    if (!references.length) {
      const empty = document.createElement("div");
      empty.className = "impact-section impact-no-change";
      empty.textContent = `没有找到引用该 ${entityLabel} 的配置。`;
      elements.missionReferenceContent.append(empty);
    } else {
      groups.forEach((records, table) => {
        const group = document.createElement("section");
        group.className = "impact-group";
        const heading = document.createElement("header");
        heading.append(
          makeCell("strong", tableLabel(table)),
          makeCell("span", `${records.length.toLocaleString()} 条引用`)
        );
        group.append(heading, renderImpactSection("引用记录", records.map((record) => ({
          ...record,
          tableLabel: tableLabel(table),
          omitChanges: true
        }))));
        elements.missionReferenceContent.append(group);
      });
    }
    elements.missionReferenceDialog.showModal();
  }

  function renderMissionRewardEmptyRow(columnCount, message) {
    const tr = document.createElement("tr");
    const cell = makeCell("td", message);
    cell.className = "mission-reward-empty";
    cell.colSpan = columnCount;
    tr.append(cell);
    return tr;
  }

  function effectiveMissionRewardID(source) {
    return state.dirty.get(changeKey("m_mission", source.row, "MissionRewardId"))?.value
      ?? String(source.missionRewardId);
  }

  function missionRewardRows(table, rewardID) {
    return missionRewardEditorRows(table).filter((row) => row.values.MissionRewardId === String(rewardID));
  }

  function missionRewardIDs(table) {
    return [...new Set(missionRewardEditorRows(table).map((row) => row.values.MissionRewardId))].sort(compareFieldValues);
  }

  function populateMissionRewardSelect(select, table, selectedID, expanded) {
    if (expanded && select.dataset.expanded === "true") return;
    const rewardIDs = missionRewardIDs(table);
    const options = expanded ? rewardIDs : rewardIDs.includes(String(selectedID)) ? [String(selectedID)] : [];
    select.replaceChildren();
    options.forEach((rewardID) => {
      const option = document.createElement("option");
      option.value = rewardID;
      option.textContent = missionRewardOptionLabel(table, rewardID);
      select.append(option);
    });
    if (!options.includes(String(selectedID))) {
      const unknown = document.createElement("option");
      unknown.value = String(selectedID);
      unknown.textContent = idNameLabel(selectedID, "未知奖励");
      select.append(unknown);
    }
    select.value = String(selectedID);
    select.title = select.options[select.selectedIndex]?.textContent || "";
    select.dataset.expanded = String(expanded);
  }

  function missionRewardOptionLabel(table, rewardID) {
    const summaries = missionRewardRows(table, rewardID).map((row) => {
      const possessionType = effectiveValue(table.name, row, "PossessionType");
      const possessionID = effectiveValue(table.name, row, "PossessionId");
      const count = effectiveValue(table.name, row, "Count");
      const definition = rewardDefinitionForPossessionType(possessionType);
      const reference = rewardReferencesForPossessionType(possessionType)
        .find((candidate) => String(candidate.possessionId) === possessionID);
      return `${rewardReferenceName(reference, definition).replace(/\s*\n\s*/g, " ")} ×${count}`;
    });
    return idNameLabel(rewardID, summaries.join(" + ") || "未定义奖励");
  }

  function renderMissionRewardInlinePreview(table, rewardID) {
    const preview = document.createElement("div");
    preview.className = "mission-reward-inline-preview";
    preview.dataset.missionRewardPreview = String(rewardID);
    const rows = missionRewardRows(table, rewardID);
    rows.forEach((row) => {
      const possessionType = effectiveValue(table.name, row, "PossessionType");
      const possessionID = effectiveValue(table.name, row, "PossessionId");
      const definition = rewardDefinitionForPossessionType(possessionType);
      const reference = rewardReferencesForPossessionType(possessionType)
        .find((candidate) => String(candidate.possessionId) === possessionID);
      preview.append(renderRewardIcon(reference, definition, "mission-reward-inline-icon"));
    });
    if (!rows.length) {
      const missing = document.createElement("span");
      missing.className = "mission-reward-inline-icon";
      missing.textContent = "奖";
      preview.append(missing);
    }
    return preview;
  }

  function refreshMissionRewardAssignmentDisplays(table, rewardID) {
    document.querySelectorAll(".mission-reward-assignment-select").forEach((select) => {
      const option = [...select.options].find((candidate) => candidate.value === String(rewardID));
      if (option) {
        option.textContent = missionRewardOptionLabel(table, rewardID);
        if (select.value === String(rewardID)) select.title = option.textContent;
      }
    });
    document.querySelectorAll("[data-mission-reward-preview]").forEach((preview) => {
      if (preview.dataset.missionRewardPreview !== String(rewardID)) return;
      preview.replaceWith(renderMissionRewardInlinePreview(table, rewardID));
    });
  }

  function renderMissionTermEditor(table, fields, sources, query) {
    const statusFilter = elements.statusFilter.value;
    const visibleSources = sources.filter((source) => {
      const termRow = missionTermRow(table, effectiveMissionTermID(source));
      if (statusFilter !== "all" && (!termRow || rowStatus(table, termRow) !== statusFilter)) return false;
      if (!query) return true;
      const termID = effectiveMissionTermID(source);
      return [
        source.missionId,
        termID,
        ...Object.values(source.names || {}),
        missionTermOptionLabel(table, termID)
      ].join(" ").toLocaleLowerCase().includes(query);
    });
    elements.missionTermAssignmentBody.replaceChildren();
    visibleSources.forEach((source) => elements.missionTermAssignmentBody.append(renderMissionTermAssignmentRow(table, source)));
    if (!visibleSources.length) elements.missionTermAssignmentBody.append(renderMissionRewardEmptyRow(3, "当前筛选条件下没有任务。"));

    const termIDQuery = elements.missionTermContentSearch.value.trim().toLocaleLowerCase();
    const visibleTermRows = table.rows.filter((row) => (
      (statusFilter === "all" || rowStatus(table, row) === statusFilter)
      && (!termIDQuery || String(row.values.MissionTermId).toLocaleLowerCase().includes(termIDQuery))
    ));
    const pageCount = Math.max(1, Math.ceil(visibleTermRows.length / state.missionTermContentPageSize));
    state.missionTermContentPage = Math.min(Math.max(1, state.missionTermContentPage), pageCount);
    state.missionTermContentPageCount = pageCount;
    const pageStart = (state.missionTermContentPage - 1) * state.missionTermContentPageSize;
    const pageEnd = Math.min(pageStart + state.missionTermContentPageSize, visibleTermRows.length);
    elements.missionTermContentBody.replaceChildren();
    visibleTermRows.slice(pageStart, pageEnd)
      .forEach((row) => elements.missionTermContentBody.append(renderMissionTermContentRow(table, fields, row)));
    if (!visibleTermRows.length) elements.missionTermContentBody.append(renderMissionRewardEmptyRow(
      5, termIDQuery ? "没有匹配该 TermId 的期限定义。" : "当前没有期限定义。"
    ));

    elements.missionTermAssignmentCount.textContent = `${visibleSources.length.toLocaleString()} 个任务`;
    const termRowCount = termIDQuery || statusFilter !== "all"
      ? `${visibleTermRows.length.toLocaleString()} / ${table.rows.length.toLocaleString()} 行`
      : `${table.rows.length.toLocaleString()} 行`;
    elements.missionTermContentCount.textContent = visibleTermRows.length
      ? `${termRowCount} · ${pageStart + 1}–${pageEnd}`
      : termRowCount;
    elements.missionTermContentPageInfo.textContent = `第 ${state.missionTermContentPage.toLocaleString()} / ${pageCount.toLocaleString()} 页`;
    elements.missionTermContentPagePrevious.disabled = state.missionTermContentPage === 1;
    elements.missionTermContentPageNext.disabled = state.missionTermContentPage === pageCount;
    elements.visibleCount.textContent = `${visibleSources.length.toLocaleString()} 个任务 · ${termRowCount.replace(" 行", " 条期限定义")}`;
    elements.empty.classList.add("hidden");
  }

  function renderMissionTermAssignmentRow(table, source) {
    const tr = document.createElement("tr");
    const missionID = document.createElement("td");
    const missionIDValue = document.createElement("code");
    missionIDValue.textContent = String(source.missionId);
    missionID.append(missionIDValue);
    const description = makeCell("td", localizedInlineText(source.names) || "未命名任务");
    description.className = "mission-reward-description";
    const term = document.createElement("td");
    const editor = document.createElement("div");
    editor.className = "mission-reward-assignment-editor mission-term-assignment-editor";
    const termID = effectiveMissionTermID(source);
    const preview = renderMissionTermInlinePreview(table, termID);
    const select = document.createElement("select");
    select.className = "mission-term-assignment-select";
    populateMissionTermSelect(select, table, termID, false);
    select.dataset.table = "m_mission";
    select.dataset.row = String(source.row);
    select.dataset.field = "MissionTermId";
    select.setAttribute("aria-label", `${description.textContent} 的 TermId`);
    select.classList.toggle("changed", state.dirty.has(changeKey("m_mission", source.row, "MissionTermId")));
    const populate = () => populateMissionTermSelect(select, table, select.value, true);
    select.addEventListener("focus", populate);
    select.addEventListener("pointerdown", populate);
    select.addEventListener("change", () => {
      onFieldChange(
        { name: "m_mission" },
        { index: source.row, values: { MissionTermId: String(source.missionTermId) } },
        { name: "MissionTermId", kind: "int32", datetime: false },
        select
      );
      renderTable();
    });
    editor.append(preview, select);
    term.append(editor);
    tr.append(missionID, description, term);
    return tr;
  }

  function renderMissionTermContentRow(table, fields, row) {
    const tr = document.createElement("tr");
    const termID = document.createElement("td");
    const termIDValue = document.createElement("code");
    termIDValue.textContent = row.values.MissionTermId;
    termID.append(termIDValue);
    const status = document.createElement("td");
    const statusValue = renderStatus(rowStatus(table, row));
    statusValue.dataset.missionTermStatusRow = String(row.index);
    status.append(statusValue);
    tr.append(termID, status);
    fields.forEach((field) => {
      const cell = document.createElement("td");
      cell.append(renderFieldEditor(table, row, field));
      tr.append(cell);
    });
    const referenceCell = document.createElement("td");
    const referenceButton = document.createElement("button");
    referenceButton.type = "button";
    referenceButton.className = "button ghost mission-reference-button";
    referenceButton.textContent = "查找引用";
    referenceButton.setAttribute("aria-label", `查找 TermId ${row.values.MissionTermId} 的任务引用`);
    referenceButton.addEventListener("click", () => showMissionReferences("term", row.values.MissionTermId));
    referenceCell.append(referenceButton);
    tr.append(referenceCell);
    return tr;
  }

  function effectiveMissionTermID(source) {
    return state.dirty.get(changeKey("m_mission", source.row, "MissionTermId"))?.value
      ?? String(source.missionTermId);
  }

  function missionTermRow(table, termID) {
    return table.rows.find((row) => row.values.MissionTermId === String(termID));
  }

  function missionTermIDs(table) {
    return table.rows.map((row) => row.values.MissionTermId).sort(compareFieldValues);
  }

  function populateMissionTermSelect(select, table, selectedID, expanded) {
    if (expanded && select.dataset.expanded === "true") return;
    const termIDs = missionTermIDs(table);
    const options = expanded ? termIDs : termIDs.includes(String(selectedID)) ? [String(selectedID)] : [];
    select.replaceChildren();
    options.forEach((termID) => {
      const option = document.createElement("option");
      option.value = termID;
      option.textContent = missionTermOptionLabel(table, termID);
      select.append(option);
    });
    if (!options.includes(String(selectedID))) {
      const unknown = document.createElement("option");
      unknown.value = String(selectedID);
      unknown.textContent = idNameLabel(selectedID, "未知期限");
      select.append(unknown);
    }
    select.value = String(selectedID);
    select.title = select.options[select.selectedIndex]?.textContent || "";
    select.dataset.expanded = String(expanded);
  }

  function missionTermOptionLabel(table, termID) {
    const row = missionTermRow(table, termID);
    if (!row) return idNameLabel(termID, "未知期限");
    const start = previewChangeValue(effectiveValue(table.name, row, "StartDatetime"), true);
    const end = previewChangeValue(effectiveValue(table.name, row, "EndDatetime"), true);
    return idNameLabel(termID, `${start} → ${end}`);
  }

  function renderMissionTermInlinePreview(table, termID) {
    const preview = document.createElement("div");
    preview.className = "mission-reward-inline-preview mission-term-inline-preview";
    preview.dataset.missionTermPreview = String(termID);
    const row = missionTermRow(table, termID);
    if (row) {
      preview.append(renderStatus(rowStatus(table, row)));
    } else {
      const missing = document.createElement("span");
      missing.className = "status disabled";
      missing.textContent = "未定义";
      preview.append(missing);
    }
    return preview;
  }

  function refreshMissionTermAssignmentDisplays(table, row) {
    const termID = row.values.MissionTermId;
    document.querySelectorAll(".mission-term-assignment-select").forEach((select) => {
      const option = [...select.options].find((candidate) => candidate.value === String(termID));
      if (option) {
        option.textContent = missionTermOptionLabel(table, termID);
        if (select.value === String(termID)) select.title = option.textContent;
      }
    });
    document.querySelectorAll("[data-mission-term-preview]").forEach((preview) => {
      if (preview.dataset.missionTermPreview !== String(termID)) return;
      preview.replaceWith(renderMissionTermInlinePreview(table, termID));
    });
    const status = document.querySelector(`[data-mission-term-status-row="${row.index}"]`);
    if (status) {
      const replacement = renderStatus(rowStatus(table, row));
      replacement.dataset.missionTermStatusRow = String(row.index);
      status.replaceWith(replacement);
    }
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
        input.min = "0001-01-01T00:00:00";
        input.max = "9999-12-31T23:59:59";
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
      option.textContent = idNameLabel(definition.possessionType, definition.label);
      select.append(option);
    });
    if (![...select.options].some((option) => option.value === currentType)) {
      const unknown = document.createElement("option");
      unknown.value = currentType;
      unknown.textContent = idNameLabel(currentType, "未知类型");
      select.append(unknown);
    }
    select.value = currentType;
    configureFieldInput(select, table, row, pair.typeField);
    select.addEventListener("change", () => {
      onFieldChange(table, row, pair.typeField, select);
      if (table.name === "m_mission_reward" && row.isNew && select.value === "2") {
        showNotice("新增武器奖励会影响 GachaPool 的任务发放排除结果，请一并确认卡池变化。", true);
      }
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
    populateRewardIDSelect(select, references, currentID, definition);
    configureFieldInput(select, table, row, pair.idField);

    let icon = renderRewardIcon(reference, definition, "reward-field-icon");
    select.addEventListener("change", () => {
      onFieldChange(table, row, pair.idField, select);
      searchInput.classList.toggle("changed", state.dirty.has(changeKey(table.name, row.index, pair.idField.name)));
      reference = references.find((candidate) => String(candidate.possessionId) === select.value);
      const nextIcon = renderRewardIcon(reference, definition, "reward-field-icon");
      icon.replaceWith(nextIcon);
      icon = nextIcon;
      if (table.name === "m_shop_item_content_possession") renderTable();
    });
    const selectSlot = document.createElement("div");
    selectSlot.className = "reward-field-select";
    const searchable = createSearchableSelect(select, {
      options: lazySearchOptions(() => rewardSelectorOptions(references, definition)),
      placeholder: "搜索奖励对象 ID 或名称",
      ariaLabel: "搜索并选择奖励对象",
      limit: 50
    });
    const searchInput = searchable.querySelector("input");
    searchInput.classList.toggle("changed", state.dirty.has(changeKey(table.name, row.index, pair.idField.name)));
    selectSlot.append(searchable);
    wrapper.append(icon, selectSlot);
    return wrapper;
  }

  function populateRewardIDSelect(select, references, selectedID, definition) {
    select.replaceChildren();
    const selected = references.find((reference) => String(reference.possessionId) === selectedID);
    if (selected) {
      const option = document.createElement("option");
      option.value = String(selected.possessionId);
      option.textContent = rewardReferenceOptionLabel(selected, definition);
      select.append(option);
    } else {
      const unknown = document.createElement("option");
      unknown.value = selectedID;
      unknown.textContent = idNameLabel(selectedID, "未知奖励");
      select.append(unknown);
    }
    select.value = selectedID;
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

  function idNameLabel(id, name) {
    return `${id}. ${name}`;
  }

  function displayContentText(value) {
    return displayText(value).replace(/<br>/g, "\n");
  }

  function displayText(value) {
    return String(value ?? "").replace(/\\n/g, "\n");
  }

  function renderMasterUpdatePreview(preview) {
    const replacementCount = (preview.tableReplacements || []).length;
    elements.masterUpdateSummary.textContent = `${preview.requestedChanges} 个字段修改将生成 ${preview.generatedChanges} 个确定的下游修改${replacementCount ? `，并整表替换 ${replacementCount} 张表` : ""}，共影响 ${preview.changedRows} 行。`;
    elements.masterUpdatePreview.replaceChildren();

    (preview.tableReplacements || []).forEach((replacement) => {
      const group = document.createElement("section");
      group.className = "impact-group";
      const header = document.createElement("header");
      const title = document.createElement("strong");
      title.textContent = "整表替换";
      const count = document.createElement("span");
      count.textContent = `${replacement.changedRows} 行变化`;
      header.append(title, count);
      const detail = document.createElement("div");
      detail.className = "impact-section impact-no-change";
      detail.textContent = `${previewTableName(replacement.table)}：${replacement.beforeRows} 行 → ${replacement.afterRows} 行；提交的是修改后的完整列表。`;
      group.append(header, detail);
      elements.masterUpdatePreview.append(group);
    });

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
    identity.textContent = `${record.tableLabel || previewTableName(record.table)} · ${identityText}`;
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
    if (record.omitChanges) return container;
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
    const shopValidationMessage = validateShopFreeInput(table.name, field.name, value);
    if (shopValidationMessage) {
      input.classList.add("invalid");
      showNotice(shopValidationMessage, true);
      return;
    }
    input.classList.remove("invalid");
    clearErrorNotice();
    storeFieldChange(table, row, field, value, input);
    if (table.name === "m_mission_reward") {
      refreshMissionRewardAssignmentDisplays(table, row.values.MissionRewardId);
    }
    if (table.name === "m_mission_term") {
      refreshMissionTermAssignmentDisplays(table, row);
    }
  }

  function validateShopFreeInput(tableName, fieldName, value) {
    const number = Number(value);
    if (tableName === "m_mission_reward" && fieldName === "Count" && number < 0) {
      return "Count 不能为负数。";
    }
    if (tableName === "m_shop_item" && ["Price", "RegularPrice"].includes(fieldName) && number < 0) {
      return `${fieldName} 不能为负数。`;
    }
    if (tableName === "m_shop_item_content_possession" && fieldName === "Count" && number <= 0) {
      return "Count 必须大于 0。";
    }
    if (tableName === "m_shop_item_content_possession" && fieldName === "SortOrder" && number < 0) {
      return "SortOrder 不能为负数。";
    }
    return "";
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
    const masterCount = masterDirtyCount();
    const count = masterCount + (questDropStructuralDirty() ? 1 : 0);
    elements.dirtyCount.textContent = count.toLocaleString();
    const groupSummary = state.shopCellGroupDirty ? "，含 1 张 CellGroup 完整列表" : "";
    const itemSummary = shopItemStructuralDirty()
      ? `，含 ${state.shopItemCopies.length} 个复制、${state.shopItemDeleteIDs.size} 个删除`
      : "";
    const cellSummary = shopItemCellStructuralDirty()
      ? `，含 ${state.shopCellAdditions.length} 个 Cell 新增、${state.shopCellDeleteKeys.size} 个删除`
      : "";
    const missionRewardSummary = missionRewardStructuralDirty()
      ? `，含 ${state.missionRewardAdditions.length} 个 Reward 新增、${state.missionRewardDeleteIDs.size} 个删除`
      : "";
    const questDropSummary = questDropStructuralDirty() ? "；关卡掉落请使用页面内的发布按钮" : "";
    elements.saveSummary.textContent = masterCount
      ? `${state.dirty.size} 个字段等待应用${groupSummary}${cellSummary}${itemSummary}${missionRewardSummary}${questDropSummary}`
      : questDropStructuralDirty() ? "关卡掉落修改请使用页面内的发布按钮" : "没有待应用的修改";
    elements.save.disabled = masterCount === 0;
    elements.discard.disabled = count === 0;
  }

  function masterDirtyCount() {
    return state.dirty.size + (state.shopCellGroupDirty ? 1 : 0)
      + (shopItemCellStructuralDirty() ? 1 : 0) + (shopItemStructuralDirty() ? 1 : 0)
      + (missionRewardStructuralDirty() ? 1 : 0);
  }

  function makeCell(tag, text) {
    const cell = document.createElement(tag);
    cell.textContent = text;
    return cell;
  }

  let noticeTimer = 0;
  let noticeID = 0;
  let noticeIsError = false;
  let busyDepth = 0;
  let busyNoticeID = 0;

  function clearNotice(expectedID = 0) {
    if (expectedID && expectedID !== noticeID) return;
    if (noticeTimer) window.clearTimeout(noticeTimer);
    noticeTimer = 0;
    noticeID += 1;
    noticeIsError = false;
    elements.notice.textContent = "";
    elements.notice.classList.add("hidden");
    elements.notice.classList.remove("error");
  }

  function clearErrorNotice() {
    if (noticeIsError) clearNotice();
  }

  function showNotice(message, error = false, options = {}) {
    if (!message) {
      clearNotice();
      return 0;
    }
    if (noticeTimer) window.clearTimeout(noticeTimer);
    noticeTimer = 0;
    const currentID = ++noticeID;
    noticeIsError = error;
    elements.notice.textContent = message;
    elements.notice.classList.remove("hidden");
    elements.notice.classList.toggle("error", error);
    if (!options.persistent) {
      const timeout = options.timeout ?? (error ? 6000 : 4500);
      noticeTimer = window.setTimeout(() => clearNotice(currentID), timeout);
    }
    return currentID;
  }

  function setBusy(busy, message = "") {
    if (busy) {
      busyDepth += 1;
      if (message) busyNoticeID = showNotice(message, false, { persistent: true });
    } else {
      busyDepth = Math.max(0, busyDepth - 1);
      if (busyDepth === 0 && busyNoticeID) {
        const completedNoticeID = busyNoticeID;
        busyNoticeID = 0;
        clearNotice(completedNoticeID);
      }
    }
    const isBusy = busyDepth > 0;
    elements.save.disabled = isBusy || masterDirtyCount() === 0;
    elements.questDropSave.disabled = isBusy || !questDropStructuralDirty() || questDropValidationErrors().length > 0;
    elements.questDropDiscard.disabled = isBusy || !questDropStructuralDirty();
    elements.refresh.disabled = isBusy;
  }

  const availabilityLabels = { standard: "常驻", event: "活动", limited: "限定" };
  const weaponAttributeLabels = { 1: "暗", 2: "火", 3: "光", 5: "水", 6: "风" };
  const weaponTypeLabels = { 1: "小剑", 2: "枪", 3: "大剑", 4: "拳", 5: "杖", 6: "铳" };
  const rewardDefinitions = [
    { key: "material", catalogKey: "materials", possessionType: "5", label: "道具", fallbackName: "未命名道具", glyph: "具" },
    { key: "weapon", catalogKey: "weapons", possessionType: "2", label: "武器", fallbackName: "未命名武器", glyph: "武" },
    { key: "companion", catalogKey: "companions", possessionType: "3", label: "伙伴", fallbackName: "未命名伙伴", glyph: "伙" },
    { key: "consumable", catalogKey: "consumableItems", possessionType: "6", label: "消耗品", fallbackName: "未命名消耗品", glyph: "消" },
    { key: "important_item", catalogKey: "importantItems", possessionType: "13", label: "重要道具", fallbackName: "未命名重要道具", glyph: "重" },
    { key: "free_gem", catalogKey: "freeGems", possessionType: "12", label: "免费宝石", fallbackName: "免费宝石", glyph: "石" }
  ];
  const rewardPageSizes = [25, 50, 100];
  const gachaGroupDefinitions = [
    { id: "character_weapon_4", grantType: "character_weapon", star: 4, label: "4星角色武器" },
    { id: "weapon_only_4", grantType: "weapon_only", star: 4, label: "4星武器" },
    { id: "character_weapon_3", grantType: "character_weapon", star: 3, label: "3星角色武器" },
    { id: "weapon_only_3", grantType: "weapon_only", star: 3, label: "3星武器" },
    { id: "weapon_only_2", grantType: "weapon_only", star: 2, label: "2星武器", calculated: true }
  ];

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
    return idNameLabel(reference.possessionId, name);
  }

  function rewardSelectorOptions(references, definition) {
    return references.map((reference) => ({
      value: String(reference.possessionId),
      label: rewardReferenceOptionLabel(reference, definition),
      searchText: `${reference.possessionId} ${Object.values(reference.names || {}).join(" ")} ${Object.values(reference.costumeNames || {}).join(" ")}`
    }));
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

  async function switchAdminSection(section, navigate = true) {
    state.section = ["master", "related", "delivery", "drop", "gacha"].includes(section) ? section : "master";
    if (navigate && window.location.pathname !== sectionPath(state.section)) {
      window.history.pushState({}, "", sectionPath(state.section));
    } else if (!navigate && window.location.pathname === "/admin/") {
      window.history.replaceState({}, "", sectionPath(state.section));
    }
    const isGacha = state.section === "gacha";
    const isDrop = state.section === "drop";
    const isDelivery = state.section === "delivery";
    document.querySelectorAll(".master-only").forEach((element) => element.classList.toggle("hidden", isGacha));
    document.querySelectorAll(".table-section-only").forEach((element) => element.classList.toggle("hidden", isGacha || isDrop));
    document.querySelectorAll(".drop-section-only").forEach((element) => element.classList.toggle("hidden", !isDrop));
    elements.gachaEditor.classList.toggle("hidden", !isGacha);
    elements.tableSearchLabel.classList.toggle("hidden", isDelivery);
    if (isDrop) elements.tableSearchLabel.classList.add("hidden");
    elements.tabMaster.classList.toggle("active", state.section === "master");
    elements.tabRelated.classList.toggle("active", state.section === "related");
    elements.tabDelivery.classList.toggle("active", isDelivery);
    elements.tabDrop.classList.toggle("active", isDrop);
    elements.tabGacha.classList.toggle("active", isGacha);
    elements.tabMaster.setAttribute("aria-pressed", String(state.section === "master"));
    elements.tabRelated.setAttribute("aria-pressed", String(state.section === "related"));
    elements.tabDelivery.setAttribute("aria-pressed", String(isDelivery));
    elements.tabDrop.setAttribute("aria-pressed", String(isDrop));
    elements.tabGacha.setAttribute("aria-pressed", String(isGacha));

    if (!state.catalog) return;
    setBusy(true, isGacha ? "正在读取 Gacha 配置…" : isDrop ? "正在读取掉落配置…" : "正在读取当前数据表…");
    try {
      if (isGacha) {
        const gachaRequest = state.gachaCatalog
          ? Promise.resolve(state.gachaCatalog)
          : api("/api/admin/gacha-config").then((catalog) => { state.gachaCatalog = catalog; });
        await Promise.all([gachaRequest, ensureRewardCatalog()]);
        if (!state.gachaDraft) resetGachaDraft();
        renderGachaEditor();
        return;
      }

      renderCatalog();
      if (!currentTable()) {
        renderTableSelectionPrompt();
        return;
      }
      const tableWasLoaded = Array.isArray(currentTable()?.rows);
      if (isDrop) {
        const dropRequest = state.questDropCatalog
          ? Promise.resolve(state.questDropCatalog)
          : api("/api/admin/quest-drop-config").then((catalog) => { state.questDropCatalog = catalog; });
        await Promise.all([loadSelectedTable(), dropRequest, ensureRewardCatalog()]);
        if (!tableWasLoaded) resetQuestDropDraft();
      } else {
        await Promise.all([
          loadSelectedTable(),
          isDelivery ? ensureRewardCatalog() : Promise.resolve()
        ]);
      }
      renderTypeFilters(currentTable());
      renderTable();
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

  function resetGachaDraft() {
    state.gachaDraft = JSON.parse(JSON.stringify(state.gachaCatalog?.config || {}));
    state.gachaDraft.version ||= 1;
    state.gachaDraft.limitedSets ||= {};
    state.gachaDraft.weapons ||= {};
    state.gachaDraft.banners ||= {};
    state.gachaDraft.chapterBanners ||= {};
    state.gachaDraft.eventBanners ||= {};
    state.gachaDraft.groupWeights ||= {
      characterWeapon: { "3": 500, "4": 200 },
      weaponOnly: { "2": 8000, "3": 1000, "4": 300 }
    };
    state.gachaDraft.groupWeights.characterWeapon ||= { "3": 500, "4": 200 };
    state.gachaDraft.groupWeights.weaponOnly ||= { "2": 8000, "3": 1000, "4": 300 };
    recalculateTwoStarWeaponProbability();
    recalculateAllBoxUnlimitedProbabilities();
    state.gachaDraft.sourceMasterDataHash = state.gachaCatalog?.masterDataHash || "";
    state.gachaDirty = false;
  }

  function renderGachaEditor() {
    if (!state.gachaCatalog || !state.gachaDraft) return;
    const isPremium = state.gachaKind === "premium";
    elements.premiumGachaConfig.classList.toggle("hidden", !isPremium);
    elements.boxGachaConfig.classList.toggle("hidden", isPremium);
    [[elements.gachaKindPremium, "premium"], [elements.gachaKindChapter, "chapter"], [elements.gachaKindEvent, "event"]].forEach(([button, kind]) => {
      const active = state.gachaKind === kind;
      button.classList.toggle("active", active);
      button.setAttribute("aria-pressed", String(active));
    });
    if (isPremium) {
      renderGachaLanguages();
      renderGachaLimitedSets();
      renderGachaWeapons();
      renderGachaBanners();
      renderGachaGroupProbabilities();
      renderGachaBannerEditor();
    } else {
      renderBoxGachaEditor();
    }
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
    const costumeName = gachaLocalizedText(weapon.costumeNames) || (weapon.costumeId ? `Costume #${weapon.costumeId}` : "—");
    const costumeCell = document.createElement("td");
    if (weapon.costumeIconPath) {
      const costumeReference = document.createElement("div");
      costumeReference.className = "gacha-costume-reference";
      const label = document.createElement("span");
      label.textContent = costumeName;
      costumeReference.append(
        renderAssetIcon(weapon.costumeIconPath, costumeName, "装", "gacha-costume-icon"),
        label
      );
      costumeCell.append(costumeReference);
    } else {
      costumeCell.textContent = costumeName;
    }
    tr.append(
      makeCell("td", String(weapon.weaponId)),
      iconCell,
      makeCell("td", weaponName),
      costumeCell,
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
      option.textContent = idNameLabel(banner.gachaId, name);
      elements.gachaBannerSelect.append(option);
    });
    if (state.gachaCatalog.banners.some((banner) => String(banner.gachaId) === previous)) elements.gachaBannerSelect.value = previous;
    createSearchableSelect(elements.gachaBannerSelect, { placeholder: "搜索卡池 ID 或名称" });
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
      label.append(input, document.createTextNode(idNameLabel(id, definition.displayName)));
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

  function recalculateBoxUnlimitedProbability(box) {
    box.groupWeights ||= {};
    box.groupWeights.limited = Number(box.groupWeights.limited ?? 8000);
    box.groupWeights.unlimited = 10000 - box.groupWeights.limited;
  }

  function recalculateAllBoxUnlimitedProbabilities() {
    Object.values(state.gachaDraft.chapterBanners).forEach(recalculateBoxUnlimitedProbability);
    Object.values(state.gachaDraft.eventBanners).forEach((event) => {
      (event.boxes || []).forEach(recalculateBoxUnlimitedProbability);
    });
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

  function setGachaKind(kind) {
    state.gachaKind = ["premium", "chapter", "event"].includes(kind) ? kind : "premium";
    renderGachaEditor();
  }

  function boxBannersForCurrentKind() {
    const labelType = state.gachaKind === "chapter" ? 3 : 2;
    return (state.gachaCatalog.boxBanners || []).filter((banner) => banner.gachaLabelType === labelType);
  }

  function currentBoxBanner() {
    const banners = boxBannersForCurrentKind();
    const selected = Number(elements.boxGachaBannerSelect.value);
    return banners.find((banner) => banner.gachaId === selected) || banners[0];
  }

  function selectedBoxNumber(banner, boxCount) {
    if (!banner || boxCount <= 0) return 0;
    const key = `${state.gachaKind}:${banner.gachaId}`;
    const selected = Number(state.boxSelections[key] || elements.boxGachaNumberSelect.value || 1);
    return Math.min(Math.max(selected, 1), boxCount);
  }

	  function boxConfigForBanner(banner) {
	    if (!banner) return { box: null, boxNumber: 0, boxCount: 0 };
	    if (state.gachaKind === "chapter") {
	      const box = state.gachaDraft.chapterBanners[String(banner.gachaId)] || null;
	      return { box, boxNumber: box ? 1 : 0, boxCount: box ? 1 : 0 };
	    }
    const event = state.gachaDraft.eventBanners[String(banner.gachaId)];
    const boxCount = event?.boxes?.length || 0;
    const boxNumber = selectedBoxNumber(banner, boxCount);
    return { box: boxNumber ? event.boxes[boxNumber - 1] : null, boxNumber, boxCount };
  }

  function renderBoxGachaEditor() {
    const banners = boxBannersForCurrentKind();
    const previousBanner = elements.boxGachaBannerSelect.value;
    elements.boxGachaBannerSelect.replaceChildren();
    banners.forEach((banner) => {
      const option = document.createElement("option");
      option.value = String(banner.gachaId);
      const name = gachaLocalizedText(banner.titles) || banner.bannerAssetName || (state.gachaKind === "chapter" ? `Chapter ${banner.relatedMainQuestChapterId}` : `Event ${banner.relatedEventQuestChapterId}`);
      option.textContent = idNameLabel(banner.gachaId, name);
      elements.boxGachaBannerSelect.append(option);
    });
    if (banners.some((banner) => String(banner.gachaId) === previousBanner)) elements.boxGachaBannerSelect.value = previousBanner;
    createSearchableSelect(elements.boxGachaBannerSelect, { placeholder: "搜索卡池 ID 或名称" });

    const banner = currentBoxBanner();
    const event = state.gachaKind === "event";
    elements.boxGachaEyebrow.textContent = event ? "EVENT BOX CONFIG" : "CHAPTER BOX CONFIG";
    elements.boxGachaTitle.textContent = event ? "Event Gacha 奖励箱" : "Chapter Gacha 奖励箱";
	    elements.boxGachaActions.classList.remove("hidden");
	    elements.boxGachaNumberLabel.classList.toggle("hidden", !event);
    document.querySelectorAll(".jackpot-column").forEach((column) => column.classList.toggle("hidden", !event));

    const selection = boxConfigForBanner(banner);
    elements.boxGachaNumberSelect.replaceChildren();
    for (let number = 1; number <= selection.boxCount; number++) {
      const option = document.createElement("option");
      option.value = String(number);
      option.textContent = `第 ${number} 箱`;
      elements.boxGachaNumberSelect.append(option);
    }
    if (selection.boxNumber) elements.boxGachaNumberSelect.value = String(selection.boxNumber);
	    elements.boxGachaAddBox.textContent = event ? "新增箱子" : "创建 Chapter 配置";
	    elements.boxGachaAddBox.disabled = !event && Boolean(selection.box);
	    elements.boxGachaRemoveBox.textContent = event ? "删除当前箱子" : "删除 Chapter 配置";
	    elements.boxGachaRemoveBox.disabled = !selection.box;
	    elements.boxGachaState.textContent = selection.box ? `${selection.boxCount} 箱 · 当前第 ${selection.boxNumber} 箱` : "未配置";
	    elements.boxGachaEmpty.textContent = event
	      ? "当前 Event Gacha 尚未配置箱子。点击“新增箱子”开始配置。"
	      : "当前 Chapter Gacha 尚未配置奖励。点击“创建 Chapter 配置”开始配置。";
    elements.boxGachaEmpty.classList.toggle("hidden", Boolean(selection.box));
    elements.boxGachaEditorBody.classList.toggle("hidden", !selection.box);
    if (!selection.box) return;

    recalculateBoxUnlimitedProbability(selection.box);
    selection.box.limitedRewards ||= [];
    selection.box.unlimitedRewards ||= [];
    elements.boxLimitedProbability.value = formatGroupProbability(Number(selection.box.groupWeights.limited || 0));
    elements.boxUnlimitedProbability.value = formatGroupProbability(Number(selection.box.groupWeights.unlimited || 0));
    elements.boxLimitedRewardBody.replaceChildren();
    selection.box.limitedRewards.forEach((reward, index) => elements.boxLimitedRewardBody.append(renderBoxRewardRow("limited", index, reward, event)));
    elements.boxUnlimitedRewardBody.replaceChildren();
    selection.box.unlimitedRewards.forEach((reward, index) => elements.boxUnlimitedRewardBody.append(renderBoxRewardRow("unlimited", index, reward, event)));
    elements.boxGachaRuleNote.textContent = event
      ? (selection.boxNumber === selection.boxCount ? "末箱：有限奖励库存全部抽完后才可重置本箱。" : "非末箱：所有标记为“大奖”的有限奖励抽完后才可解锁下一箱。")
      : "Chapter Gacha 每月自动重置有限奖励库存，不允许手动重置。";
    refreshBoxProbabilityPreviews();
  }

  function currentEditableBox() {
    return boxConfigForBanner(currentBoxBanner()).box;
  }

  function setRewardFromReference(reward, reference) {
    reward.possessionType = Number(reference.possessionType);
    reward.possessionId = Number(reference.possessionId);
    reward.rarityType = Number(reference.rarityType || 0);
  }

  function newBoxReward(limited, event, offset = 0) {
    const references = state.rewardCatalog?.materials || [];
    const reference = references[Math.min(offset, Math.max(references.length - 1, 0))] || { possessionType: 5, possessionId: 100001, rarityType: 1 };
    return {
      possessionType: Number(reference.possessionType), possessionId: Number(reference.possessionId), rarityType: Number(reference.rarityType || 0),
      count: 1, ...(limited ? { maxCount: 1 } : {}), weight: 100, featured: false, ...(limited && event ? { jackpot: true } : {})
    };
  }

  function newBoxConfig(event) {
    return {
      groupWeights: { limited: 8000, unlimited: 2000 },
      limitedRewards: [newBoxReward(true, event, 0)],
      unlimitedRewards: [newBoxReward(false, event, 1)]
    };
  }

  function renderBoxRewardRow(group, index, reward, event) {
    const limited = group === "limited";
    const tr = document.createElement("tr");
    const rewardCell = document.createElement("td");
    const editor = document.createElement("div");
    editor.className = "box-reward-selector";
    const typeSelect = document.createElement("select");
    rewardDefinitions.forEach((definition) => {
      if (!rewardReferencesForPossessionType(definition.possessionType).length) return;
      const option = document.createElement("option");
      option.value = definition.possessionType;
      option.textContent = definition.label;
      typeSelect.append(option);
    });
    typeSelect.value = String(reward.possessionType);
    const itemSelect = document.createElement("select");
    const renderOptions = () => {
      itemSelect.replaceChildren();
      const references = rewardReferencesForPossessionType(typeSelect.value);
      references.forEach((reference) => {
        const option = document.createElement("option");
        option.value = String(reference.possessionId);
        option.textContent = idNameLabel(
          reference.possessionId || "—",
          gachaLocalizedText(reference.names) || rewardDefinitionForPossessionType(reference.possessionType)?.fallbackName || "未命名奖励"
        );
        itemSelect.append(option);
      });
      itemSelect.value = String(reward.possessionId || 0);
      if (!itemSelect.value && references.length) {
        setRewardFromReference(reward, references[0]);
        itemSelect.value = String(reward.possessionId || 0);
      }
    };
    renderOptions();
    typeSelect.addEventListener("change", () => {
      const reference = rewardReferencesForPossessionType(typeSelect.value)[0];
      if (reference) setRewardFromReference(reward, reference);
      markBoxGachaDirty(true);
    });
    itemSelect.addEventListener("change", () => {
      const reference = rewardReferencesForPossessionType(typeSelect.value).find((item) => item.possessionId === Number(itemSelect.value));
      if (reference) setRewardFromReference(reward, reference);
      markBoxGachaDirty(false);
    });
    editor.append(typeSelect, itemSelect);
    rewardCell.append(editor);
    tr.append(rewardCell);

    const appendNumber = (field, label, min, onInput) => {
      const cell = document.createElement("td");
      const input = document.createElement("input");
      input.type = "number";
      input.min = String(min);
      input.step = "1";
      input.value = String(reward[field] ?? min);
      input.setAttribute("aria-label", label);
      input.addEventListener("input", () => {
        reward[field] = Math.round(Number(input.value || 0));
        markBoxGachaDirty(false);
        if (onInput) onInput();
      });
      cell.append(input);
      tr.append(cell);
    };
    appendNumber("count", "单次获得数量", 1);
    if (limited) appendNumber("maxCount", "有限库存", 1);
    appendNumber("weight", "奖励权重", 1, refreshBoxProbabilityPreviews);

    const probabilityCell = document.createElement("td");
    const probability = document.createElement("strong");
    probability.className = "box-probability-preview";
    probability.dataset.boxProbability = `${group}:${index}`;
    probabilityCell.append(probability);
    tr.append(probabilityCell);

    const featuredCell = document.createElement("td");
    const featured = document.createElement("input");
    featured.type = "checkbox";
    featured.checked = Boolean(reward.featured);
    featured.setAttribute("aria-label", "Featured 奖励");
    featured.addEventListener("change", () => { reward.featured = featured.checked; markBoxGachaDirty(false); });
    featuredCell.append(featured);
    tr.append(featuredCell);

    if (limited) {
      const jackpotCell = document.createElement("td");
      jackpotCell.className = "jackpot-column";
      const jackpot = document.createElement("input");
      jackpot.type = "checkbox";
      jackpot.checked = Boolean(reward.jackpot);
      jackpot.disabled = !event;
      jackpot.setAttribute("aria-label", "Event Gacha 大奖");
      jackpot.addEventListener("change", () => { reward.jackpot = jackpot.checked; markBoxGachaDirty(false); });
      jackpotCell.append(jackpot);
      jackpotCell.classList.toggle("hidden", !event);
      tr.append(jackpotCell);
    }

    const removeCell = document.createElement("td");
    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "chip-action box-remove-reward";
    remove.textContent = "删除";
    remove.addEventListener("click", () => {
      const box = currentEditableBox();
      box[limited ? "limitedRewards" : "unlimitedRewards"].splice(index, 1);
      markBoxGachaDirty(true);
    });
    removeCell.append(remove);
    tr.append(removeCell);
    return tr;
  }

  function refreshBoxProbabilityPreviews() {
    const box = currentEditableBox();
    if (!box) return;
    const limitedWeight = Number(box.groupWeights?.limited || 0);
    const unlimitedWeight = Number(box.groupWeights?.unlimited || 0);
    const groupTotal = limitedWeight + unlimitedWeight;
    [["limited", box.limitedRewards || [], limitedWeight], ["unlimited", box.unlimitedRewards || [], unlimitedWeight]].forEach(([group, rewards, groupWeight]) => {
      const rewardTotal = rewards.reduce((sum, reward) => sum + Math.max(0, Number(reward.weight || 0)), 0);
      rewards.forEach((reward, index) => {
        const target = document.querySelector(`[data-box-probability="${group}:${index}"]`);
        if (!target) return;
        const probability = groupTotal > 0 && rewardTotal > 0 ? groupWeight / groupTotal * Math.max(0, Number(reward.weight || 0)) / rewardTotal * 100 : 0;
        target.textContent = `${probability.toFixed(4).replace(/0+$/, "").replace(/\.$/, "")}%`;
      });
    });
  }

  function markBoxGachaDirty(rerender) {
    state.gachaDirty = true;
    if (rerender) renderBoxGachaEditor();
    else updateGachaDirtyUI();
  }

  function addBoxReward(group) {
    const box = currentEditableBox();
    if (!box) return;
    const limited = group === "limited";
    box[limited ? "limitedRewards" : "unlimitedRewards"].push(newBoxReward(limited, state.gachaKind === "event"));
    markBoxGachaDirty(true);
  }

	  function addConfiguredBox() {
	    const banner = currentBoxBanner();
	    if (!banner) return;
	    const key = String(banner.gachaId);
	    if (state.gachaKind === "chapter") {
	      if (state.gachaDraft.chapterBanners[key]) return;
	      state.gachaDraft.chapterBanners[key] = newBoxConfig(false);
	      markBoxGachaDirty(true);
	      return;
	    }
	    const event = state.gachaDraft.eventBanners[key] ||= { boxes: [] };
    event.boxes ||= [];
    event.boxes.push(newBoxConfig(true));
    state.boxSelections[`event:${banner.gachaId}`] = event.boxes.length;
    markBoxGachaDirty(true);
  }

	  function removeConfiguredBox() {
	    const banner = currentBoxBanner();
	    if (!banner) return;
	    if (state.gachaKind === "chapter") {
	      const key = String(banner.gachaId);
	      if (!state.gachaDraft.chapterBanners[key] || !confirm(`删除 Chapter Gacha ${banner.gachaId} 的奖励配置？`)) return;
	      delete state.gachaDraft.chapterBanners[key];
	      markBoxGachaDirty(true);
	      return;
	    }
	    const event = state.gachaDraft.eventBanners[String(banner.gachaId)];
    if (!event?.boxes?.length || !confirm(`删除 Event Gacha ${banner.gachaId} 的当前箱子？`)) return;
    const number = selectedBoxNumber(banner, event.boxes.length);
    event.boxes.splice(number - 1, 1);
    if (!event.boxes.length) delete state.gachaDraft.eventBanners[String(banner.gachaId)];
    state.boxSelections[`event:${banner.gachaId}`] = Math.max(1, number - 1);
    markBoxGachaDirty(true);
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

  function appendBoxValidationErrors(errors) {
    const knownRewards = new Set(rewardDefinitions.flatMap((definition) =>
      rewardReferencesForPossessionType(definition.possessionType).map((reward) => `${reward.possessionType}:${reward.possessionId}`)
    ));
    const validateBox = (gachaId, box, boxNumber, event) => {
      const limitedWeight = Number(box?.groupWeights?.limited);
      const unlimitedWeight = Number(box?.groupWeights?.unlimited);
      if (![limitedWeight, unlimitedWeight].every((weight) => Number.isInteger(weight) && weight >= 0) || limitedWeight + unlimitedWeight !== 10000) {
        errors.push(`卡池 ${gachaId} 第 ${boxNumber} 箱的有限、无限奖励组概率合计必须为 100%`);
      }
      const groups = [["有限", box?.limitedRewards || [], true, limitedWeight], ["无限", box?.unlimitedRewards || [], false, unlimitedWeight]];
      groups.forEach(([label, rewards, limited, groupWeight]) => {
        if (groupWeight > 0 && !rewards.length) errors.push(`卡池 ${gachaId} 第 ${boxNumber} 箱的${label}奖励组有概率但没有奖励`);
        rewards.forEach((reward, index) => {
          if (!knownRewards.has(`${reward.possessionType}:${reward.possessionId}`)) errors.push(`卡池 ${gachaId} 第 ${boxNumber} 箱的${label}奖励 ${index + 1} 不在主数据奖励列表中`);
          if (!Number.isInteger(Number(reward.count)) || Number(reward.count) <= 0) errors.push(`卡池 ${gachaId} 第 ${boxNumber} 箱的${label}奖励 ${index + 1} 单次数量必须为正整数`);
          if (limited && (!Number.isInteger(Number(reward.maxCount)) || Number(reward.maxCount) <= 0)) errors.push(`卡池 ${gachaId} 第 ${boxNumber} 箱的有限奖励 ${index + 1} 库存必须为正整数`);
          if (!Number.isInteger(Number(reward.weight)) || Number(reward.weight) <= 0) errors.push(`卡池 ${gachaId} 第 ${boxNumber} 箱的${label}奖励 ${index + 1} 权重必须为正整数`);
          if ((!event || !limited) && reward.jackpot) errors.push(`卡池 ${gachaId} 第 ${boxNumber} 箱只有 Event 有限奖励可以设为大奖`);
        });
      });
      if (event && limitedWeight <= 0) errors.push(`Event Gacha ${gachaId} 第 ${boxNumber} 箱的有限奖励组概率必须大于 0`);
      if (event && !(box?.limitedRewards || []).some((reward) => reward.jackpot)) errors.push(`Event Gacha ${gachaId} 第 ${boxNumber} 箱至少需要一个大奖`);
    };
    Object.entries(state.gachaDraft.chapterBanners || {}).forEach(([gachaId, box]) => validateBox(gachaId, box, 1, false));
    Object.entries(state.gachaDraft.eventBanners || {}).forEach(([gachaId, event]) => {
      if (!event.boxes?.length) errors.push(`Event Gacha ${gachaId} 至少需要一个箱子`);
      (event.boxes || []).forEach((box, index) => validateBox(gachaId, box, index + 1, true));
    });
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
    appendBoxValidationErrors(errors);
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
    state.questDropCatalog = null;
    state.gachaDraft = null;
    state.gachaDirty = false;
    state.dirty.clear();
    state.shopCellGroupDraft = [];
    state.shopCellGroupBaseline = "[]";
    state.shopCellGroupDirty = false;
    state.shopCellGroupSelection = "";
    state.shopItemCopies = [];
    state.shopItemDeleteIDs = new Set();
    state.shopCellAdditions = [];
    state.shopCellDeleteKeys = new Map();
    resetMissionRewardDraft();
    state.questDropDraft = new Map();
    state.questDropBaseline = new Map();
    state.questDropDirtyQuestIDs = new Set();
    sessionStorage.removeItem("lunar-admin-token");
    showLogin();
  });

  const activateSection = (section, navigate = true) => {
    switchAdminSection(section, navigate).catch(() => { /* notice is already shown */ });
  };
  elements.tabMaster.addEventListener("click", () => activateSection("master"));
  elements.tabRelated.addEventListener("click", () => activateSection("related"));
  elements.tabDelivery.addEventListener("click", () => activateSection("delivery"));
  elements.tabDrop.addEventListener("click", () => activateSection("drop"));
  elements.tabGacha.addEventListener("click", () => activateSection("gacha"));
  window.addEventListener("popstate", () => activateSection(sectionFromPath(), false));
  elements.gachaKindPremium.addEventListener("click", () => setGachaKind("premium"));
  elements.gachaKindChapter.addEventListener("click", () => setGachaKind("chapter"));
  elements.gachaKindEvent.addEventListener("click", () => setGachaKind("event"));

  elements.tableSelect.addEventListener("change", async () => {
    state.tableSelections[state.section] = elements.tableSelect.value;
    state.missionRewardContentPage = 1;
    state.missionTermContentPage = 1;
    state.shopCellPage = 1;
    state.shopItemPage = 1;
    state.questDropPage = 1;
    setBusy(true, "正在读取当前数据表…");
    try {
      await Promise.all([
        loadSelectedTable(),
        state.section === "delivery" ? ensureRewardCatalog() : Promise.resolve()
      ]);
      renderTypeFilters(currentTable());
      renderTable();
    } catch (error) {
      showNotice(error.message, true);
    } finally {
      setBusy(false);
    }
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
  });

  elements.missionRewardContentPageSize.addEventListener("change", () => {
    const pageSize = Number(elements.missionRewardContentPageSize.value);
    state.missionRewardContentPageSize = rewardPageSizes.includes(pageSize) ? pageSize : 25;
    state.missionRewardContentPage = 1;
    renderTable();
  });
  elements.missionRewardContentAdd.addEventListener("click", addMissionReward);
  elements.missionRewardContentSearch.addEventListener("input", () => {
    state.missionRewardContentPage = 1;
    renderTable();
  });
  elements.missionRewardContentUnreferenced.addEventListener("change", () => {
    state.missionRewardContentPage = 1;
    renderTable();
  });
  elements.missionRewardContentPagePrevious.addEventListener("click", () => {
    if (state.missionRewardContentPage <= 1) return;
    state.missionRewardContentPage -= 1;
    renderTable();
  });
  elements.missionRewardContentPageNext.addEventListener("click", () => {
    if (state.missionRewardContentPage >= state.missionRewardContentPageCount) return;
    state.missionRewardContentPage += 1;
    renderTable();
  });
  elements.missionReferenceClose.addEventListener("click", () => elements.missionReferenceDialog.close());
  elements.missionTermContentPageSize.addEventListener("change", () => {
    const pageSize = Number(elements.missionTermContentPageSize.value);
    state.missionTermContentPageSize = rewardPageSizes.includes(pageSize) ? pageSize : 25;
    state.missionTermContentPage = 1;
    renderTable();
  });
  elements.missionTermContentSearch.addEventListener("input", () => {
    state.missionTermContentPage = 1;
    renderTable();
  });
  elements.missionTermContentPagePrevious.addEventListener("click", () => {
    if (state.missionTermContentPage <= 1) return;
    state.missionTermContentPage -= 1;
    renderTable();
  });
  elements.missionTermContentPageNext.addEventListener("click", () => {
    if (state.missionTermContentPage >= state.missionTermContentPageCount) return;
    state.missionTermContentPage += 1;
    renderTable();
  });
  elements.shopCellGroupAdd.addEventListener("click", addShopCellGroupRow);
  elements.shopCellAdd.addEventListener("click", addShopCell);
  elements.shopCellSearch.addEventListener("input", () => {
    state.shopCellPage = 1;
    renderTable();
  });
  elements.shopCellUnreferenced.addEventListener("change", () => {
    state.shopCellPage = 1;
    renderTable();
  });
  elements.shopCellPageSize.addEventListener("change", () => {
    const pageSize = Number(elements.shopCellPageSize.value);
    state.shopCellPageSize = rewardPageSizes.includes(pageSize) ? pageSize : 25;
    state.shopCellPage = 1;
    renderTable();
  });
  elements.shopCellPagePrevious.addEventListener("click", () => {
    if (state.shopCellPage <= 1) return;
    state.shopCellPage -= 1;
    renderTable();
  });
  elements.shopCellPageNext.addEventListener("click", () => {
    if (state.shopCellPage >= state.shopCellPageCount) return;
    state.shopCellPage += 1;
    renderTable();
  });
  elements.shopItemSearch.addEventListener("input", () => {
    state.shopItemPage = 1;
    renderTable();
  });
  elements.shopItemUnreferenced.addEventListener("change", () => {
    state.shopItemPage = 1;
    renderTable();
  });
  elements.shopItemPageSize.addEventListener("change", () => {
    const pageSize = Number(elements.shopItemPageSize.value);
    state.shopItemPageSize = [10, 25, 50].includes(pageSize) ? pageSize : 10;
    state.shopItemPage = 1;
    renderTable();
  });
  elements.shopItemPagePrevious.addEventListener("click", () => {
    if (state.shopItemPage <= 1) return;
    state.shopItemPage -= 1;
    renderTable();
  });
  elements.shopItemPageNext.addEventListener("click", () => {
    if (state.shopItemPage >= state.shopItemPageCount) return;
    state.shopItemPage += 1;
    renderTable();
  });
  elements.questDropSearch.addEventListener("input", () => {
    state.questDropPage = 1;
    renderTable();
  });
  elements.questDropPageSize.addEventListener("change", () => {
    const pageSize = Number(elements.questDropPageSize.value);
    state.questDropPageSize = [10, 25, 50].includes(pageSize) ? pageSize : 10;
    state.questDropPage = 1;
    renderTable();
  });
  elements.questDropPagePrevious.addEventListener("click", () => {
    if (state.questDropPage <= 1) return;
    state.questDropPage -= 1;
    renderTable();
  });
  elements.questDropPageNext.addEventListener("click", () => {
    if (state.questDropPage >= state.questDropPageCount) return;
    state.questDropPage += 1;
    renderTable();
  });
  elements.questDropCopyCancel.addEventListener("click", () => elements.questDropCopyDialog.close());
  elements.questDropCopyConfirm.addEventListener("click", copyQuestDropRewards);
  elements.questDropCopySource.addEventListener("keydown", (event) => {
    if (event.key !== "Enter") return;
    event.preventDefault();
    copyQuestDropRewards();
  });
  elements.questDropCopyDialog.addEventListener("close", resetQuestDropCopyDialog);
  elements.questDropDiscard.addEventListener("click", () => {
    if (!confirm("放弃尚未发布的关卡掉落修改？")) return;
    resetQuestDropDraft();
    updateDirtyUI();
    renderTable();
    showNotice("已放弃关卡掉落修改。");
  });
  elements.questDropSave.addEventListener("click", async () => {
    const errors = questDropValidationErrors();
    if (errors.length) {
      showNotice(errors[0], true);
      return;
    }
    if (!confirm(`发布 ${state.questDropDirtyQuestIDs.size} 个关卡的掉落配置？`)) return;
    setBusy(true, "正在校验、写入并热更新关卡掉落配置…");
    try {
      await api("/api/admin/quest-drop-config", {
        method: "POST",
        body: JSON.stringify({
          expectedContentHash: state.questDropCatalog.contentHash,
          config: questDropReplacementPayload()
        })
      });
      await loadCatalog();
      showNotice("关卡掉落配置发布成功，新的关卡请求已使用新权重。");
    } catch (error) {
      showNotice(error.message, true);
      if (error.status === 409) {
        try { await loadCatalog(); } catch (_) { /* keep the conflict notice */ }
      }
    } finally {
      setBusy(false);
      updateQuestDropDirtyUI();
      updateDirtyUI();
    }
  });
  elements.search.addEventListener("input", renderTable);
  elements.modeButtons.forEach((button) => button.addEventListener("click", () => {
    state.mode = button.dataset.mode === "detail" ? "detail" : "simple";
    renderTable();
  }));
  elements.refresh.addEventListener("click", async () => {
    if ((masterDirtyCount() || questDropStructuralDirty() || state.gachaDirty) && !confirm("刷新会放弃尚未应用的修改，是否继续？")) return;
    try { await loadCatalog(); } catch (_) { /* notice is already shown */ }
  });
  elements.discard.addEventListener("click", () => {
    if (!confirm("放弃全部尚未应用的修改？")) return;
    state.dirty.clear();
    resetShopCellGroupDraft();
    resetMissionRewardDraft();
    resetQuestDropDraft();
    state.pendingMasterChanges = null;
    updateDirtyUI();
    renderTable();
    showNotice("已放弃本次修改。");
  });
  elements.save.addEventListener("click", async () => {
    const rewardStructural = missionRewardStructuralDirty();
    const changes = [...state.dirty.values()].filter((change) => !(rewardStructural && change.table === "m_mission_reward"));
    if (!changes.length && !rewardStructural && !state.shopCellGroupDirty && !shopItemCellStructuralDirty() && !shopItemStructuralDirty()) return;
    const request = { expectedVersion: state.catalog.version, changes };
    if (rewardStructural) {
      const table = state.catalog.tables.find((candidate) => candidate.name === "m_mission_reward");
      request.missionRewards = missionRewardReplacementPayload(table);
    }
    if (state.shopCellGroupDirty) {
      request.shopItemCellGroups = state.shopCellGroupDraft.map(shopCellGroupPayload);
    }
    if (shopItemCellStructuralDirty()) request.shopItemCells = shopItemCellStructuralPayload();
    if (shopItemStructuralDirty()) request.shopItems = shopItemStructuralPayload();
    setBusy(true, "正在计算确定链路及变更预览…");
    try {
      const preview = await api("/api/admin/master-data/schedules/preview", {
        method: "POST",
        body: JSON.stringify(request)
      });
      state.pendingMasterChanges = request;
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
    const request = state.pendingMasterChanges;
    if (!request || (!request.changes?.length && !request.missionRewards && !request.shopItemCellGroups && !request.shopItemCells && !request.shopItems)) return;
    elements.masterUpdateDialog.returnValue = "confirm";
    elements.masterUpdateDialog.close();
    state.pendingMasterChanges = null;
    setBusy(true, "正在重建、验证并热更新上游及关联主数据…");
    try {
      const result = await api("/api/admin/master-data/schedules", {
        method: "POST",
        body: JSON.stringify(request)
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
  elements.boxGachaBannerSelect.addEventListener("change", renderBoxGachaEditor);
  elements.boxGachaNumberSelect.addEventListener("change", () => {
    const banner = currentBoxBanner();
    if (banner) state.boxSelections[`${state.gachaKind}:${banner.gachaId}`] = Number(elements.boxGachaNumberSelect.value || 1);
    renderBoxGachaEditor();
  });
	  elements.boxGachaAddBox.addEventListener("click", addConfiguredBox);
	  elements.boxGachaRemoveBox.addEventListener("click", removeConfiguredBox);
  elements.boxAddLimitedReward.addEventListener("click", () => addBoxReward("limited"));
  elements.boxAddUnlimitedReward.addEventListener("click", () => addBoxReward("unlimited"));
  elements.boxLimitedProbability.addEventListener("input", () => {
    const box = currentEditableBox();
    if (!box) return;
    box.groupWeights.limited = Math.round(Number(elements.boxLimitedProbability.value || 0) * 100);
    recalculateBoxUnlimitedProbability(box);
    elements.boxUnlimitedProbability.value = formatGroupProbability(box.groupWeights.unlimited);
    markBoxGachaDirty(false);
    refreshBoxProbabilityPreviews();
  });
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
    showNotice("正在校验、写入并热更新 Gacha 配置…", false, { persistent: true });
    try {
      await api("/api/admin/gacha-config", {
        method: "POST",
        body: JSON.stringify({ expectedContentHash: state.gachaCatalog.contentHash, config: state.gachaDraft })
      });
      await loadCatalog();
      showNotice("Gacha 配置发布成功，新的抽取请求已使用新版本。");
    } catch (error) {
      showNotice(error.message, true);
      if (error.status === 409) {
        try { await loadCatalog(); } catch (_) { /* keep the conflict notice */ }
      }
    } finally {
      updateGachaDirtyUI();
    }
  });

  window.addEventListener("beforeunload", (event) => {
    if (!masterDirtyCount() && !questDropStructuralDirty() && !state.gachaDirty) return;
    event.preventDefault();
    event.returnValue = "";
  });

  if (state.token) loadCatalog().catch(() => showLogin());
  else showLogin();
})();
