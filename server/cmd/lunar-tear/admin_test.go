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
		`<th>ShopItem</th><th>交易内容</th><th>库存（只读）</th>`,
	} {
		if !strings.Contains(html, required) {
			t.Fatalf("ShopItem HTML is missing %s", required)
		}
	}
	for _, required := range []string{
		".shop-cell-card-visual", ".shop-cell-card-id", ".shop-transaction-stack", ".shop-price-row",
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
		"makeCell(\"span\", `重置周期 ${item.stockAutoResetPeriod}`)",
		`renderShopPriceIDEditor(item)`,
	} {
		if !strings.Contains(javascript, required) {
			t.Fatalf("ShopItem JavaScript is missing %s", required)
		}
	}
	if strings.Contains(javascript, "Cell ${row.shopItemCellId} · Group ${row.shopItemCellGroupId}") {
		t.Fatal("CellGroup card still renders the removed Cell/Group subtitle")
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
		"MissionReward group":  `table.name === "m_mission_reward" ? { placeholder: "搜索任务组 ID 或名称" } : null`,
	} {
		if !strings.Contains(javascript, required) {
			t.Fatalf("%s does not use the searchable select", name)
		}
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
