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
