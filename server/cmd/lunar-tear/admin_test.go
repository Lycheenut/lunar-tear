package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminContentSecurityPolicyAllowsBannerPreviews(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	response := httptest.NewRecorder()

	serveAdminAsset(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	policy := response.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "img-src 'self' https://assets.lycheenut.cc") {
		t.Fatalf("Content-Security-Policy does not allow banner previews: %q", policy)
	}
}

func TestAdminConfigurationModulesUseSubroutesAndLazyData(t *testing.T) {
	for _, path := range []string{
		"/admin/activities", "/admin/related", "/admin/delivery", "/admin/drops", "/admin/gacha",
	} {
		html := adminAssetBody(t, path)
		if !strings.Contains(html, `id="workspace"`) {
			t.Fatalf("GET %s did not serve the admin application", path)
		}
	}

	html := adminAssetBody(t, "/admin/")
	javascript := adminAssetBody(t, "/admin/admin.js")
	for _, required := range []string{
		`id="tab-drop"`, `id="quest-drop-filters"`,
	} {
		if !strings.Contains(html, required) {
			t.Fatalf("admin HTML is missing %s", required)
		}
	}
	for _, required := range []string{
		`api("/api/admin/master-data/catalog")`,
		"api(`/api/admin/master-data/table?name=${encodeURIComponent(requestedName)}`)",
		`new Option("请选择数据表", "")`,
		`"/admin/drops": "drop"`,
		`table.delivery && table.name !== "m_quest_pickup_reward_group"`,
		`api("/api/admin/gacha-config")`,
		`api("/api/admin/quest-drop-config")`,
	} {
		if !strings.Contains(javascript, required) {
			t.Fatalf("admin JavaScript is missing lazy route behavior %s", required)
		}
	}
}

func TestAdminSearchableSelectLoadsAroundSelectionAndExtendsAtEdges(t *testing.T) {
	javascript := adminAssetBody(t, "/admin/admin.js")
	for _, required := range []string{
		`const batchSize = controller.config.limit || 50`,
		`selectedIndex - 25`, `selectedIndex + 25`,
		`scrollIntoView({ block: "center" })`,
		`controller.windowStart - batchSize`,
		`controller.windowEnd + batchSize`,
		`options: lazySearchOptions(() => rewardSelectorOptions(references, definition))`,
	} {
		if !strings.Contains(javascript, required) {
			t.Fatalf("searchable select is missing windowing behavior %s", required)
		}
	}
}

func TestAdminQuestDropUsesSharedOuterPanelAndBottomSavebar(t *testing.T) {
	html := adminAssetBody(t, "/admin/")
	css := adminAssetBody(t, "/admin/admin.css")
	javascript := adminAssetBody(t, "/admin/admin.js")

	for _, required := range []string{
		`class="savebar drop-section-only hidden"`,
		`<strong id="quest-drop-save-summary">`,
		`id="quest-drop-discard"`, `id="quest-drop-save"`,
	} {
		if !strings.Contains(html, required) {
			t.Fatalf("quest drop bottom savebar is missing %s", required)
		}
	}
	headingStart := strings.Index(html, `class="quest-drop-heading"`)
	if headingStart < 0 {
		t.Fatal("quest drop heading is missing")
	}
	headingEnd := strings.Index(html[headingStart:], `</header>`)
	if headingEnd < 0 {
		t.Fatal("quest drop heading is not closed")
	}
	heading := html[headingStart : headingStart+headingEnd]
	if strings.Contains(heading, `id="quest-drop-discard"`) || strings.Contains(heading, `id="quest-drop-save"`) {
		t.Fatal("quest drop actions are still rendered in the page heading")
	}
	if !strings.Contains(css, `.table-scroll.quest-drop-mode { height: clamp(620px, calc(100vh - 300px), 920px); max-height: none; overflow: hidden; padding: 0;`) {
		t.Fatal("quest drop table still has the extra inset container padding")
	}
	if strings.Contains(css, `.quest-drop-save-summary {`) {
		t.Fatal("quest drop still has an in-panel save summary")
	}
	if !strings.Contains(javascript, `document.querySelectorAll(".drop-section-only")`) {
		t.Fatal("quest drop bottom savebar is not tied to its route")
	}
}

func TestAdminTransientNoticesAndDatetimeYearRange(t *testing.T) {
	javascript := adminAssetBody(t, "/admin/admin.js")
	for _, required := range []string{
		`input.min = "0001-01-01T00:00:00"`,
		`input.max = "9999-12-31T23:59:59"`,
		`noticeTimer = window.setTimeout(() => clearNotice(currentID), timeout)`,
		`showNotice(message, false, { persistent: true })`,
		`clearNotice(completedNoticeID)`,
		`clearErrorNotice();`,
	} {
		if !strings.Contains(javascript, required) {
			t.Fatalf("admin transient state behavior is missing %s", required)
		}
	}
}

func TestAdminShopEditorUsesSearchableCardLayout(t *testing.T) {
	html := adminAssetBody(t, "/admin/")
	css := adminAssetBody(t, "/admin/admin.css")
	javascript := adminAssetBody(t, "/admin/admin.js")

	top := strings.Index(html, `class="shop-top-grid"`)
	items := strings.Index(html, `class="mission-reward-panel mission-reward-content-panel shop-item-panel"`)
	if top < 0 || items < 0 || top >= items {
		t.Fatal("Shop editor must place the CellGroup/Cell grid above the ShopItem panel")
	}
	for _, level := range []string{"L0 ·", "L1 ·", "L2 ·"} {
		if strings.Contains(html, level) {
			t.Fatalf("Shop editor still contains hierarchy label %q", level)
		}
	}
	for _, required := range []string{".shop-cell-group-grid", ".shop-cell-card-icon", ".shop-item-table"} {
		if !strings.Contains(css, required) {
			t.Fatalf("admin CSS is missing %s", required)
		}
	}
	for _, required := range []string{"createSearchableSelect(select", "shopCellGroupSearchOptions", "renderShopCellGroupIcon"} {
		if !strings.Contains(javascript, required) {
			t.Fatalf("admin JavaScript is missing %s", required)
		}
	}
}

func TestAdminShopItemUsesNamedSearchAndTransactionLayout(t *testing.T) {
	html := adminAssetBody(t, "/admin/")
	css := adminAssetBody(t, "/admin/admin.css")
	javascript := adminAssetBody(t, "/admin/admin.js")

	for _, required := range []string{
		`placeholder="搜索 ShopItemId 或名称"`,
		`<th>ShopItem</th><th>交易内容</th><th>库存配置</th>`,
	} {
		if !strings.Contains(html, required) {
			t.Fatalf("ShopItem HTML is missing %s", required)
		}
	}
	for _, required := range []string{
		".shop-cell-card-visual", ".shop-cell-card-id", ".shop-transaction-stack", ".shop-price-row.without-price-id",
	} {
		if !strings.Contains(css, required) {
			t.Fatalf("ShopItem CSS is missing %s", required)
		}
	}
	for _, required := range []string{
		"...Object.values(item.names || {})",
		`cellID.textContent = String(row.shopItemCellId)`,
		`transactionStack.append(contentSection, priceSection)`,
		`tr.append(identity, transaction, stock)`,
		"makeCell(\"span\", `重置周期 ${stock.autoResetPeriod}`)",
		`const includesPriceID = effectiveShopItemValue(item, "PriceType", item.priceType) === "1"`,
		`if (includesPriceID) controls.push(renderShopPriceIDEditor(item))`,
		`renderShopPriceIDEditor(item)`,
		`possessions: shopItemPossessionsForCopy(source, contentTable, itemID)`,
		`renderShopDraftPossessionField(item, row, rowIndex, field.name)`,
		`configureShopItemSelect(select, item, "ShopItemLimitedStockId"`,
		`function copyShopItem(source, contentTable)`,
		`function deleteShopItem(item)`,
		`function shopItemEffectiveBlockers(item, referencedItemIDs = null)`,
		`const possessionPrefix = `,
		`shopItems = shopItemStructuralPayload()`,
	} {
		if !strings.Contains(javascript, required) {
			t.Fatalf("ShopItem JavaScript is missing %s", required)
		}
	}
	if strings.Contains(javascript, "Cell ${row.shopItemCellId} · Group ${row.shopItemCellGroupId}") {
		t.Fatal("CellGroup card still renders the removed Cell/Group subtitle")
	}
	if strings.Contains(javascript, `blockers.push("m_shop_item_content_possession")`) {
		t.Fatal("ShopItem Possession content is still treated as a delete blocker")
	}
	for _, removed := range []string{"shop-content-sort-order", `"类型", "对象", "数量", "排序"`} {
		if strings.Contains(javascript, removed) || strings.Contains(css, removed) {
			t.Fatalf("ShopItem still renders removed SortOrder content: %s", removed)
		}
	}
}

func TestAdminShopCellStructureLazySelectorsAndUnreferencedFilters(t *testing.T) {
	html := adminAssetBody(t, "/admin/")
	javascript := adminAssetBody(t, "/admin/admin.js")

	for _, required := range []string{
		`id="shop-cell-add"`, `id="shop-cell-unreferenced"`, `id="shop-item-unreferenced"`,
		`<th>CellId</th><th>StepNumber</th><th>ShopItemId</th><th>操作</th>`,
	} {
		if !strings.Contains(html, required) {
			t.Fatalf("Shop Cell HTML is missing %s", required)
		}
	}
	for _, required := range []string{
		`function createLazySearchSelect(`,
		`options,`,
		`limit: config.limit || 50`,
		`function createShopCellSelector(`,
		`function createShopItemSelector(`,
		`function shopItemCellStructuralPayload()`,
		`request.shopItemCells = shopItemCellStructuralPayload()`,
		`elements.shopCellUnreferenced.checked`,
		`elements.shopItemUnreferenced.checked`,
	} {
		if !strings.Contains(javascript, required) {
			t.Fatalf("Shop Cell JavaScript is missing %s", required)
		}
	}
	for _, obsolete := range []string{"populateShopCellSelect", "populateShopItemSelect"} {
		if strings.Contains(javascript, obsolete) {
			t.Fatalf("Shop selectors still eagerly populate native options through %s", obsolete)
		}
	}
}

func TestAdminDeliveryUsesSearchableRewardsAndReferenceLookups(t *testing.T) {
	html := adminAssetBody(t, "/admin/")
	javascript := adminAssetBody(t, "/admin/admin.js")

	if strings.Contains(html, "奖励对象对照表") || strings.Contains(html, `id="reward-reference"`) {
		t.Fatal("delivery editor still renders the removed reward reference table")
	}
	for _, required := range []string{
		`id="table-search-label"`,
		`elements.tableSearchLabel.classList.toggle("hidden", isDelivery)`,
		`const query = state.section === "delivery" ? ""`,
		`function rewardSelectorOptions(references, definition)`,
		`placeholder: "搜索奖励对象 ID 或名称"`,
		`function showShopCellReferences(cell)`,
		`function showShopItemReferences(item)`,
		`references.textContent = "查找引用"`,
	} {
		if !strings.Contains(html, required) && !strings.Contains(javascript, required) {
			t.Fatalf("delivery editor is missing %s", required)
		}
	}
}

func TestAdminQuestDropEditorUsesInlineRewardAndAcquisitionPreviews(t *testing.T) {
	html := adminAssetBody(t, "/admin/")
	css := adminAssetBody(t, "/admin/admin.css")
	javascript := adminAssetBody(t, "/admin/admin.js")

	for _, required := range []string{
		`id="quest-drop-editor"`, `id="quest-drop-search"`,
		`id="quest-drop-copy-dialog"`, `id="quest-drop-copy-source"`, `id="quest-drop-copy-confirm"`,
		`<h3>QuestPickupRewardGroup</h3>`,
		`<th>QuestId</th><th>掉落内容</th><th>奖励预览</th><th>获得路径预览</th>`,
		`id="quest-drop-page-info"`,
	} {
		if !strings.Contains(html, required) {
			t.Fatalf("quest drop HTML is missing %s", required)
		}
	}
	for _, required := range []string{
		".quest-drop-editor", ".quest-drop-item", ".quest-drop-main",
		".quest-drop-pickup-preview", ".quest-drop-route-preview", ".quest-drop-preview-row",
		".quest-drop-preview-toggle", ".quest-drop-groups", ".quest-drop-group", ".quest-drop-guaranteed-item",
		".quest-drop-identity", ".quest-drop-copy", ".quest-drop-copy-field",
		".quest-drop-table th:first-child { width: 240px; }",
		".quest-drop-table th:nth-child(2) { width: 520px; }",
		".quest-drop-table th:nth-child(4) { width: 230px; }",
	} {
		if !strings.Contains(css, required) {
			t.Fatalf("quest drop CSS is missing %s", required)
		}
	}
	for _, required := range []string{
		`typeLabel.textContent = "副本类型"`, `chapterLabel.textContent = "章节"`,
		`const usesDifficultyFilter = state.questDropTypeFilter === "main"`,
		`const usesSubcategoryFilter = state.questDropTypeFilter === "event-7"`,
		`if (!usesDifficultyFilter && !usesSubcategoryFilter)`,
		`subtypeLabel.textContent = usesSubcategoryFilter ? "关卡类型" : "难度"`,
		`const labels = { "1": "真暗ノ巣窟", "2": "真暗ノコイン", "3": "EXガチャチケット" }`,
		"new Option(`${definition.value}. ${definition.label}`, definition.id)",
		"Number(left.value) - Number(right.value)",
		"Number(left.chapterId) - Number(right.chapterId)",
		"Number(left.battleDropRewardId) - Number(right.battleDropRewardId)",
		`state.questDropTypeFilter = editor.types[0]?.id || ""`,
		`state.questDropChapterFilter = chapterSelect.options[0]?.value || ""`,
		`questDropSubtypeFilter: ""`, `subtypeValues.includes(1) ? "1"`,
		`function questDropRewardOptions()`, `function renderQuestDropPickupPreview(quest)`,
		`function renderQuestDropRoutePreview(quest)`, `row.append(identity, content, preview, routePreview)`,
		`function setQuestDropPreviewReward(questID, rewardID, included)`, `toggle.type = "checkbox"`,
		`toggle.addEventListener("change", () => setQuestDropPreviewReward`,
		"detail.textContent = `${rewardID}. ${questDropRewardName(reward)} ×${reward?.count ?? \"?\"}`",
		`function questDropStageLabel(quest)`,
		`identityContent.className = "quest-drop-identity"`,
		"chapter.textContent = `${questDropChapterLabel(quest)}-${questDropStageLabel(quest)}`",
		"dropCount.textContent = `总掉落数 ${quest.dropCount ?? 0}`",
		`function questDropReplacementPayload()`, `api("/api/admin/quest-drop-config"`,
		`function renderQuestDropGroup(quest, rewards, guaranteed, optionSource)`,
		`const groupLabel = guaranteed ? "必定掉落" : "随机掉落"`, `weightInput.type = "number"`,
		`renderQuestDropGroup(quest, rewards, false, optionSource)`,
		`renderQuestDropGroup(quest, rewards, true, optionSource)`,
		"add.textContent = `添加${groupLabel}`", `guaranteed: reward.guaranteed`, `在同一${guaranteed ? "必定掉落" : "随机掉落"}组中只能配置一条`,
		`reward.guaranteed === normalizedGuaranteed`, "const key = `${Boolean(reward.guaranteed)}:${reward.battleDropRewardId}`",
		`function openQuestDropCopyDialog(quest)`, `function copyQuestDropRewards()`, `copy.textContent = "复制自其他副本"`,
		`sourceRewards.map((reward) => ({ ...reward }))`, `没有可复制的掉落配置`,
		`当前关卡相同`, `已配置 ${currentRewards.length} 条掉落`, `elements.questDropCopyConfirm.textContent = "确认覆盖"`,
		`table.name === "m_quest_pickup_reward_group"`,
	} {
		if !strings.Contains(javascript, required) {
			t.Fatalf("quest drop JavaScript is missing %s", required)
		}
	}
	for _, obsolete := range []string{
		`recommendedPossessions`, `Pickup / 获得途径（并集）`, `id="quest-route-quest-select"`,
		`id="quest-route-possession-list"`, `.quest-drop-workspace`, `.quest-route-preview`,
		`if (table.name === "m_quest_pickup_reward_group") return "关卡掉落"`,
		`默认 PickupGroup`, "heading.textContent = `Quest ${quest.questId}`",
		"id.textContent = `Reward ${rewardID}`", ` · Drop ${reward.battleDropRewardId}`,
		`state.questDropGroupIndex.get(String(quest.questPickupRewardGroupId))?.rewards || []`,
		"`${possession.possessionType}:${possession.possessionId}.",
		`function setQuestDropGuaranteed(`, `guaranteedInput.type = "checkbox"`, `.quest-drop-guaranteed input`,
		`questDropDifficultyFilter`,
	} {
		if strings.Contains(html, obsolete) || strings.Contains(css, obsolete) || strings.Contains(javascript, obsolete) {
			t.Fatalf("quest drop editor still contains obsolete presentation: %s", obsolete)
		}
	}
}

func TestAdminEntityLabelsUseIDFirstFormat(t *testing.T) {
	javascript := adminAssetBody(t, "/admin/admin.js")
	if !strings.Contains(javascript, "return `${id}. ${name}`;") {
		t.Fatal("admin JavaScript is missing the ID-first entity label formatter")
	}
	for _, obsolete := range []string{
		"${name}（${reference.possessionId}）",
		"${definition.label}（${definition.possessionType}）",
		"${banner.gachaId} · ${name}",
		"${definition.displayName} · ${id}",
		"未命名商品\"}（${item.shopItemId}）",
	} {
		if strings.Contains(javascript, obsolete) {
			t.Fatalf("admin JavaScript still uses a name-first entity label: %s", obsolete)
		}
	}
}

func TestAdminSearchableSelectAppliedToRequestedFilters(t *testing.T) {
	css := adminAssetBody(t, "/admin/admin.css")
	javascript := adminAssetBody(t, "/admin/admin.js")

	for _, required := range []string{".searchable-select", ".searchable-select-options", ".searchable-select-source"} {
		if !strings.Contains(css, required) {
			t.Fatalf("admin CSS is missing %s", required)
		}
	}
	for name, required := range map[string]string{
		"premium Gacha banner": `createSearchableSelect(elements.gachaBannerSelect`,
		"Box Gacha banner":     `createSearchableSelect(elements.boxGachaBannerSelect`,
		"LoginBonusId":         `definition.field === "LoginBonusId"`,
		"Mission groups":       `{ placeholder: "搜索任务组 ID 或名称" }`,
	} {
		if !strings.Contains(javascript, required) {
			t.Fatalf("%s does not use the searchable select", name)
		}
	}
}

func TestAdminChapterBoxToolbarReservesActionWidth(t *testing.T) {
	css := adminAssetBody(t, "/admin/admin.css")

	for _, required := range []string{
		`.box-gacha-toolbar > * { min-width: 0; }`,
		`.box-gacha-toolbar:has(#box-gacha-number-label.hidden) { grid-template-columns: minmax(260px, 1fr) auto; }`,
	} {
		if !strings.Contains(css, required) {
			t.Fatalf("admin CSS is missing Chapter Box toolbar layout rule %s", required)
		}
	}
}

func TestAdminMissionTermUsesRequestedLayout(t *testing.T) {
	css := adminAssetBody(t, "/admin/admin.css")

	for _, required := range []string{
		`.mission-term-editor {`,
		`grid-template-columns: minmax(0, 1.25fr) minmax(500px, 1fr);`,
		`.toolbar:has(#detail-mode-control.hidden)`,
		`minmax(300px, 1fr) max-content`,
	} {
		if !strings.Contains(css, required) {
			t.Fatalf("admin CSS is missing requested MissionTerm layout rule %s", required)
		}
	}
}

func TestAdminMissionRewardUsesRestrictedStructuralUpdates(t *testing.T) {
	html := adminAssetBody(t, "/admin/")
	css := adminAssetBody(t, "/admin/admin.css")
	javascript := adminAssetBody(t, "/admin/admin.js")

	for _, required := range []string{
		`id="mission-reward-content-add"`,
		`id="mission-reward-content-unreferenced"`,
		`<th>RewardId</th><th>PossessionType</th><th>PossessionId</th><th>Count</th><th>操作</th>`,
	} {
		if !strings.Contains(html, required) {
			t.Fatalf("MissionReward HTML is missing %s", required)
		}
	}
	for _, required := range []string{
		".mission-reward-row-actions", ".mission-reward-delete:disabled", ".mission-reward-unreferenced-filter",
	} {
		if !strings.Contains(css, required) {
			t.Fatalf("MissionReward CSS is missing %s", required)
		}
	}
	for _, required := range []string{
		`function missionRewardReplacementPayload(table)`,
		`function missionRewardReferences(rewardID)`,
		`function addMissionReward()`,
		`function deleteMissionReward(table, row)`,
		`deleteButton.disabled = references.length > 0`,
		`const unreferencedOnly = elements.missionRewardContentUnreferenced.checked`,
		`!referencedRewardIDs.has(String(row.values.MissionRewardId))`,
		`request.missionRewards = missionRewardReplacementPayload(table)`,
		`tableName === "m_mission_reward" && fieldName === "Count" && number < 0`,
	} {
		if !strings.Contains(javascript, required) {
			t.Fatalf("MissionReward JavaScript is missing %s", required)
		}
	}
}

func TestAdminRewardCatalogIncludesImportantItems(t *testing.T) {
	javascript := adminAssetBody(t, "/admin/admin.js")

	if !strings.Contains(javascript, `{ key: "important_item", catalogKey: "importantItems", possessionType: "13"`) {
		t.Fatal("reward definitions are missing ImportantItem possession type 13")
	}
}

func adminAssetBody(t *testing.T, path string) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	serveAdminAsset(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want %d", path, response.Code, http.StatusOK)
	}
	return response.Body.String()
}
