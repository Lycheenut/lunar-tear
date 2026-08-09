package masterdataadmin

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
)

func TestResolveAdditionalAssetTitles(t *testing.T) {
	resolver := &titleResolver{
		texts: localizationIndex{
			"en": {
				"gacha.title.limited_45":        "Celebratory Summons",
				"gacha.title.limitd_442":        "Merry Summons",
				"shop.name.101":                 "Medal Exchange",
				"mission.name.201":              "Anniversary Missions",
				"quest.event.chapter_title.301": "Record: The Festival",
				"consumable_item.name.110004":   "Gold Automata Medal",
				"important_item.name.401":       "Mystic Slab",
				"campaign.description.02.02.02": "Glorious success weapon-enhance rate up",
				"campaign.description.01.05.01": "Bonuses added to drops for certain quests.",
			},
			"ja": {"gacha.title.limited_45": "記念ガチャ"},
			"ko": {},
		},
		shopTextIDs:          map[int64]int64{55: 101},
		missionTermTextIDs:   map[int64]int64{77: 201},
		consumableTermKeys:   map[int64][]string{5: {"consumable_item.name.110004"}},
		importantEffectTexts: map[int64]int64{9: 401},
		enhanceTargetTypes:   map[int64][]int64{2: {2}},
		questEffects:         map[int64]questCampaignEffect{20: {effectType: 5, effectValue: 1000}},
		questTargetTypes:     map[int64][]int64{2: {1}},
	}

	tests := []struct {
		name  string
		table string
		row   []interface{}
		want  string
	}{
		{name: "gacha mom banner", table: "m_mom_banner", row: []interface{}{1, 0, 1, 45, "limited_45"}, want: "Celebratory Summons"},
		{name: "gacha typo fallback", table: "m_mom_banner", row: []interface{}{1, 0, 1, 442, "limited_442"}, want: "Merry Summons"},
		{name: "shop mom banner", table: "m_mom_banner", row: []interface{}{2, 0, 2, 55, "shop_mom_banner_55"}, want: "Medal Exchange"},
		{name: "mission mom banner", table: "m_mom_banner", row: []interface{}{3, 0, 22, 77, "mission_mom_banner_201"}, want: "Anniversary Missions"},
		{name: "event mom banner", table: "m_mom_banner", row: []interface{}{4, 0, 23, 88, "event_mom_banner_301"}, want: "Record: The Festival"},
		{name: "consumable item term", table: "m_consumable_item_term", row: []interface{}{5}, want: "Gold Automata Medal"},
		{name: "important item effect", table: "m_important_item_effect", row: []interface{}{9}, want: "Mystic Slab"},
		{name: "enhance campaign", table: "m_enhance_campaign", row: []interface{}{1, 2, 2, 400}, want: "Glorious success weapon-enhance rate up · ×3"},
		{name: "quest drop bonus", table: "m_quest_campaign", row: []interface{}{1, 2, 20}, want: "Bonuses added to drops for certain quests."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := resolver.resolve(test.table, test.row)["en"]; got != test.want {
				t.Fatalf("English title = %q, want %q", got, test.want)
			}
		})
	}
	if got := resolver.resolve("m_mom_banner", tests[0].row)["ja"]; got != "記念ガチャ" {
		t.Fatalf("Japanese gacha title = %q", got)
	}
}

func TestCampaignTitlesIncludeConcreteEffect(t *testing.T) {
	resolver := &titleResolver{
		texts: localizationIndex{
			"en": {
				"campaign.description.02.01":    "Glorious success enhance rate up",
				"campaign.description.02.02.02": "Glorious success weapon-enhance rate up",
				"campaign.description.02.02.03": "Memoir enhance success rate up",
				"campaign.target.02.01":         "All Characters",
				"campaign.description.01.01.03": "Item drop rate x{0}",
				"campaign.description.01.03.01": "Stamina cost for quests halved",
				"campaign.description.01.04.02": "Gold acquired x{0}",
			},
			"ja": {},
			"ko": {},
		},
		enhanceTargetTypes: map[int64][]int64{10: {2}, 11: {3}, 12: {1}},
		questEffects: map[int64]questCampaignEffect{
			20: {effectType: 1, effectValue: 1500},
			21: {effectType: 3, effectValue: 500},
			22: {effectType: 4, effectValue: 2000},
		},
		questTargetTypes: map[int64][]int64{30: {3}, 31: {1}, 32: {2}},
	}

	tests := []struct {
		name  string
		table string
		row   []interface{}
		want  string
	}{
		{name: "weapon glorious success", table: "m_enhance_campaign", row: []interface{}{1, 10, 2, 400}, want: "Glorious success weapon-enhance rate up · ×3"},
		{name: "memoir success", table: "m_enhance_campaign", row: []interface{}{2, 11, 2, 500}, want: "Memoir enhance success rate up · +5%"},
		{name: "absolute glorious success", table: "m_enhance_campaign", row: []interface{}{6, 12, 1, 400}, want: "Glorious success enhance rate up (All Characters) · 4%"},
		{name: "drop rate", table: "m_quest_campaign", row: []interface{}{3, 30, 20}, want: "Item drop rate x2.5"},
		{name: "stamina", table: "m_quest_campaign", row: []interface{}{4, 31, 21}, want: "Stamina cost for quests halved · ×0.5"},
		{name: "gold", table: "m_quest_campaign", row: []interface{}{5, 32, 22}, want: "Gold acquired x3"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := resolver.resolve(test.table, test.row)["en"]; got != test.want {
				t.Fatalf("title = %q, want %q", got, test.want)
			}
		})
	}
}

func TestEventChapterTitlesCombinesDistinctRelations(t *testing.T) {
	resolver := &titleResolver{
		texts: localizationIndex{
			"en": {
				"quest.event.chapter_title.101": "Easy",
				"quest.event.chapter_title.102": "Normal",
				"quest.event.chapter_title.103": "Normal",
			},
			"ja": {
				"quest.event.chapter_title.101": "初級",
				"quest.event.chapter_title.102": "中級",
				"quest.event.chapter_title.103": "中級",
			},
			"ko": {},
		},
		chapterTextIDs: map[int64]int64{1: 101, 2: 102, 3: 103},
	}
	titles := resolver.eventChapterTitles([]int64{1, 2, 3})
	if got, want := titles["en"], "Easy / Normal"; got != want {
		t.Fatalf("English title = %q, want %q", got, want)
	}
	if got, want := titles["ja"], "初級 / 中級"; got != want {
		t.Fatalf("Japanese title = %q, want %q", got, want)
	}
	if _, exists := titles["ko"]; exists {
		t.Fatal("unexpected empty Korean title")
	}
}

func TestWeaponTitlesUsesAssetNameKey(t *testing.T) {
	resolver := &titleResolver{texts: localizationIndex{
		"en": {"weapon.name.wp001051.1": "Black Sunflower"},
		"ja": {},
		"ko": {},
	}}
	titles := weaponTitles(resolver, masterdata.EntityMWeapon{WeaponCategoryType: 1, WeaponType: 1, AssetVariationId: 51})
	if titles["en"] != "Black Sunflower" {
		t.Fatalf("weapon title = %q", titles["en"])
	}
}

func TestCostumeTitlesUsesSkeletonAndVariationKey(t *testing.T) {
	resolver := &titleResolver{texts: localizationIndex{
		"en": {"costume.name.ch008001": "Celebratory Assassin"},
		"ja": {},
		"ko": {},
	}}
	titles := costumeTitles(resolver, masterdata.EntityMCostume{ActorSkeletonId: 8, AssetVariationId: 1})
	if titles["en"] != "Celebratory Assassin" {
		t.Fatalf("costume title = %q", titles["en"])
	}
}
