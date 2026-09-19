package service

import (
	"html/template"
	"net/url"
	"strings"
	"testing"

	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/masterdata"
)

func TestGachaWebPairedCostumeIconsAndPickupOrder(t *testing.T) {
	web, _, session, cat := newGachaWebTestServer(t)
	web.icons = map[string]template.URL{
		"attribute_water":       "colored-water.png",
		"proper_attribute_fire": "gray-fire.png",
		"weapon_icon_sword":     "sword.png",
		"weapon_icon_spear":     "spear.png",
	}
	cat.Weapon.Weapons[100003] = masterdata.EntityMWeapon{WeaponType: 1, AttributeType: 5, AssetVariationId: 60}
	cat.Costume = &masterdata.CostumeCatalog{
		Costumes:         map[int32]masterdata.EntityMCostume{7: {CharacterId: 1, ActorSkeletonId: 1, AssetVariationId: 2, RarityType: 30, SkillfulWeaponType: 2}},
		ProperAttributes: map[int32]int32{7: 2},
	}
	web.names["en"]["costume.name.ch001002"] = "Test Costume"
	web.names["en"]["character.name.1"] = "Test Character"
	cat.GachaHandler.Premium.Banners[100].Groups[0].Pickup = []gacha.PoolItem{{WeaponId: 100003, CostumeId: 7, RarityType: 30}}
	response := requestGachaPage(web, "/web/en/gacha-rate?gachaId=100&sessionKey="+url.QueryEscape(session.SessionKey))
	body := response.Body.String()
	if response.Code != 200 {
		t.Fatalf("page status = %d: %s", response.Code, body)
	}
	if strings.Contains(body, `id="search"`) || strings.Count(body, `class="rate-summary"`) != 2 || strings.Count(body, `class="individual-rates"`) != 2 {
		t.Fatal("page must show separate draw sections without a search field")
	}
	if strings.Count(body, `rowspan="2"`) != 2 || strings.Count(body, `class="costume-row pickup"`) != 2 {
		t.Fatal("paired costumes must share one probability cell in each table")
	}
	if strings.Index(body, `data-weapon-id="100003"`) > strings.Index(body, `data-weapon-id="100002"`) {
		t.Fatal("Pickup was not placed first")
	}
	if strings.Count(body, `data-weapon-id="100001"`) != 1 {
		t.Fatal("two-star weapons should only appear in the normal draw table")
	}
	for _, section := range strings.Split(body, `class="costume-row pickup"`)[1:] {
		row := strings.SplitN(section, "</tr>", 2)[0]
		attribute, weapon, character := strings.Index(row, "proper_attribute_fire"), strings.Index(row, "weapon_icon_spear"), strings.Index(row, "Test Character")
		if attribute < 0 || weapon <= attribute || character <= weapon || strings.Contains(row, "attribute_water") {
			t.Fatal("costume must show its own gray attribute before its preferred weapon and character name")
		}
	}
}
