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
				"shop.item.name.501":            "Mama Medal",
				"mission.name.201":              "Anniversary Missions",
				"quest.event.chapter_title.301": "Record: The Festival",
				"consumable_item.name.110004":   "Gold Automata Medal",
				"important_item.name.401":       "Mystic Slab",
				"campaign.description.02.02.02": "Glorious success weapon-enhance rate up",
				"campaign.description.01.05.01": "Bonuses added to drops for certain quests.",
				"tip.1000":                      "Enhancing Weapons",
				"tip.11000":                     "Weapons gain strength when enhanced.",
			},
			"ja": {"gacha.title.limited_45": "記念ガチャ"},
			"ko": {},
		},
		shopTextIDs:          map[int64]int64{55: 101},
		shopItemTextIDs:      map[int64]int64{500: 501},
		missionTermTextIDs:   map[int64]int64{77: 201},
		consumableTermKeys:   map[int64][]string{5: {"consumable_item.name.110004"}},
		importantEffectTexts: map[int64]int64{9: 401},
		enhanceTargets:       map[int64][]campaignTarget{2: {{targetType: 2}}},
		questEffects:         map[int64][]questCampaignEffect{20: {{effectType: 5, effectValue: 1000}}},
		questTargets:         map[int64][]campaignTarget{2: {{targetType: 1}}},
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
		{name: "mission term", table: "m_mission_term", row: []interface{}{77}, want: "Anniversary Missions"},
		{name: "shop item content", table: "m_shop_item_content_possession", row: []interface{}{500}, want: "Mama Medal"},
		{name: "event mom banner", table: "m_mom_banner", row: []interface{}{4, 0, 23, 88, "event_mom_banner_301"}, want: "Record: The Festival"},
		{name: "consumable item term", table: "m_consumable_item_term", row: []interface{}{5}, want: "Gold Automata Medal"},
		{name: "important item effect", table: "m_important_item_effect", row: []interface{}{9}, want: "Mystic Slab"},
		{name: "enhance campaign", table: "m_enhance_campaign", row: []interface{}{1, 2, 2, 400}, want: "Glorious success weapon-enhance rate up"},
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
	tipRow := []interface{}{1000, 1000, 11000}
	if got, want := resolver.resolve("m_tip", tipRow)["en"], "Enhancing Weapons"; got != want {
		t.Fatalf("Tip title = %q, want %q", got, want)
	}
	if got, want := resolver.resolveContentBody("m_tip", tipRow)["en"], "Weapons gain strength when enhanced."; got != want {
		t.Fatalf("Tip body = %q, want %q", got, want)
	}
}

func TestBigHuntScheduleTitleUsesSeasonAssetID(t *testing.T) {
	resolver := &titleResolver{}
	for language, want := range map[string]string{
		"en": "Season 8",
		"ja": "シーズン 8",
		"ko": "시즌 8",
	} {
		if got := resolver.resolve("m_big_hunt_schedule", []interface{}{18, 0, 0, 0, 8})[language]; got != want {
			t.Fatalf("%s title = %q, want %q", language, got, want)
		}
	}
	if titles := resolver.resolve("m_big_hunt_schedule", []interface{}{1, 0, 0, 0, 0}); len(titles) != 0 {
		t.Fatalf("zero season asset title = %v, want empty", titles)
	}
}

func TestDokanTitleIncludesEveryContentTextInOrder(t *testing.T) {
	resolver := &titleResolver{
		dokanTitles: translatedText{
			10: {"en": "First line", "ja": "一行目"},
			20: {"en": "Second line\nwith detail", "ja": "二行目\n詳細"},
			30: {"en": "Third line", "ja": "三行目"},
		},
		dokanGroupTexts: map[int64][]dokanContentText{
			100: {
				{contentIndex: 1, textID: 10, imageID: 101},
				{contentIndex: 2, textID: 20},
				{contentIndex: 3, textID: 30, imageID: 103},
			},
		},
	}

	row := []interface{}{1, 1, 1, int64(100), int64(200), 100}
	if got, want := resolver.resolve("m_dokan", row)["en"], "First line\nSecond line\nwith detail\nThird line"; got != want {
		t.Fatalf("English Dokan title = %q, want %q", got, want)
	}
	if got, want := resolver.resolve("m_dokan", row)["ja"], "一行目\n二行目\n詳細\n三行目"; got != want {
		t.Fatalf("Japanese Dokan title = %q, want %q", got, want)
	}
	images := resolver.resolveDokanImages("m_dokan", row)
	if got, want := images, []DokanImage{{ContentIndex: 1, ImageID: 101}, {ContentIndex: 3, ImageID: 103}}; !equalDokanImages(got, want) {
		t.Fatalf("Dokan images = %+v, want %+v", got, want)
	}
}

func TestNaviCutInTitleIncludesEveryContentTextInOrder(t *testing.T) {
	resolver := &titleResolver{
		naviCutInTitles: translatedText{
			10: {"en": "First line", "ja": "一行目"},
			20: {"en": "Second line", "ja": "二行目"},
		},
		naviCutInGroupTexts: map[int64][]naviCutInContentText{
			100: {
				{contentIndex: 1, textID: 10},
				{contentIndex: 2, textID: 20},
			},
		},
	}

	row := []interface{}{1, 1, 1, int64(100), int64(200), 100, 0}
	if got, want := resolver.resolve("m_navi_cut_in", row)["en"], "First line\nSecond line"; got != want {
		t.Fatalf("English title = %q, want %q", got, want)
	}
	if got, want := resolver.resolve("m_navi_cut_in", row)["ja"], "一行目\n二行目"; got != want {
		t.Fatalf("Japanese title = %q, want %q", got, want)
	}
}

func equalDokanImages(left, right []DokanImage) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
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
		enhanceTargets: map[int64][]campaignTarget{
			10: {{targetType: 2}}, 11: {{targetType: 3}}, 12: {{targetType: 1}},
		},
		questEffects: map[int64][]questCampaignEffect{
			20: {{effectType: 1, effectValue: 1500}},
			21: {{effectType: 3, effectValue: 500}},
			22: {{effectType: 4, effectValue: 2000}},
		},
		questTargets: map[int64][]campaignTarget{
			30: {{targetType: 3}}, 31: {{targetType: 1}}, 32: {{targetType: 2}},
		},
	}

	tests := []struct {
		name  string
		table string
		row   []interface{}
		want  string
	}{
		{name: "weapon glorious success", table: "m_enhance_campaign", row: []interface{}{1, 10, 2, 400}, want: "Glorious success weapon-enhance rate up"},
		{name: "memoir success", table: "m_enhance_campaign", row: []interface{}{2, 11, 2, 500}, want: "Memoir enhance success rate up"},
		{name: "absolute glorious success", table: "m_enhance_campaign", row: []interface{}{6, 12, 1, 400}, want: "Glorious success enhance rate up (All Characters)"},
		{name: "drop rate", table: "m_quest_campaign", row: []interface{}{3, 30, 20}, want: "Item drop rate x2.5"},
		{name: "stamina", table: "m_quest_campaign", row: []interface{}{4, 31, 21}, want: "Stamina cost for quests halved"},
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

func TestContentFootnotes(t *testing.T) {
	resolver := &titleResolver{
		texts: localizationIndex{
			"en": {
				"character.name.1001":           "2B",
				"campaign.target.02.03":         "All Memoirs",
				"quest.event.chapter_title.301": "Record: The Festival",
			},
			"ja": {},
			"ko": {},
		},
		chapterTextIDs:       map[int64]int64{30: 301},
		enhanceTargets:       map[int64][]campaignTarget{10: {{targetType: 23, targetValue: 20}}, 11: {{targetType: 11, targetValue: 1001}}, 12: {{targetType: 3}}},
		questEffects:         map[int64][]questCampaignEffect{20: {{effectType: 3, effectValue: 500}}},
		questTargets:         map[int64][]campaignTarget{30: {{targetType: 7, targetValue: 40}}},
		weaponTitlesByID:     map[int64]map[string]string{20: {"en": "Black Sunflower"}},
		eventChaptersByQuest: map[int64][]int64{40: {30}},
		maintenanceAPIs:      map[int64][]string{50: {"apb.api.gacha.GachaService/Draw", "apb.api.pvp.PvpService/GetRanking"}},
	}

	enhance := resolver.resolveContentFootnotes("m_enhance_campaign", []interface{}{1, 10, 2, 400, int64(100), int64(200), 2}, nil)
	if got, want := footnoteTexts(enhance, "en"), []string{"×3", "Black Sunflower", "Returning Users"}; !equalStrings(got, want) {
		t.Fatalf("enhance footnotes = %q, want %q", got, want)
	}
	if got, want := footnoteTexts(enhance, "ja"), []string{"×3", "Black Sunflower", "カムバックユーザー"}; !equalStrings(got, want) {
		t.Fatalf("Japanese enhance footnotes = %q, want %q", got, want)
	}
	character := resolver.resolveContentFootnotes("m_enhance_campaign", []interface{}{2, 11, 1, 400}, nil)
	if got, want := footnoteTexts(character, "en"), []string{"4%", "2B"}; !equalStrings(got, want) {
		t.Fatalf("character footnotes = %q, want %q", got, want)
	}
	memoir := resolver.resolveContentFootnotes("m_enhance_campaign", []interface{}{3, 12, 2, 500}, nil)
	if got, want := footnoteTexts(memoir, "en"), []string{"+5%", "All Memoirs"}; !equalStrings(got, want) {
		t.Fatalf("memoir footnotes = %q, want %q", got, want)
	}
	quest := resolver.resolveContentFootnotes("m_quest_campaign", []interface{}{4, 30, 20, int64(100), int64(200), 3}, nil)
	if got, want := footnoteTexts(quest, "en"), []string{"×0.5", "Record: The Festival", "New Users"}; !equalStrings(got, want) {
		t.Fatalf("quest footnotes = %q, want %q", got, want)
	}
	beginner := resolver.resolveContentFootnotes("m_beginner_campaign", []interface{}{1}, nil)
	if got, want := footnoteTexts(beginner, "en"), []string{"New Users"}; !equalStrings(got, want) {
		t.Fatalf("beginner footnotes = %q, want %q", got, want)
	}
	comeback := resolver.resolveContentFootnotes("m_comeback_campaign", []interface{}{2}, nil)
	if got, want := footnoteTexts(comeback, "en"), []string{"Returning Users"}; !equalStrings(got, want) {
		t.Fatalf("comeback footnotes = %q, want %q", got, want)
	}
	loginBonus := resolver.resolveContentFootnotes("m_login_bonus", []interface{}{24, 3, 6}, nil)
	if got, want := footnoteTexts(loginBonus, "en"), []string{"Returning Users (Grade Group 1)"}; !equalStrings(got, want) {
		t.Fatalf("login bonus footnotes = %q, want %q", got, want)
	}
	if got, want := footnoteTexts(loginBonus, "ja"), []string{"カムバックユーザー（グレードグループ1）"}; !equalStrings(got, want) {
		t.Fatalf("Japanese login bonus footnotes = %q, want %q", got, want)
	}
	for condition, want := range map[int64]string{0: "All Users", 4: "Returning Users", 5: "New Users"} {
		footnotes := resolver.resolveContentFootnotes("m_login_bonus", []interface{}{1, 1, condition}, nil)
		if got := footnoteTexts(footnotes, "en"); !equalStrings(got, []string{want}) {
			t.Fatalf("login bonus condition %d footnotes = %q, want %q", condition, got, want)
		}
	}
	maintenanceRow := []interface{}{5, int64(100), int64(200), 50}
	if got, want := resolver.resolve("m_maintenance", maintenanceRow)["en"], "apb.api.gacha.GachaService/Draw / apb.api.pvp.PvpService/GetRanking"; got != want {
		t.Fatalf("maintenance title = %q, want %q", got, want)
	}
	if footnotes := resolver.resolveContentFootnotes("m_maintenance", maintenanceRow, nil); len(footnotes) != 0 {
		t.Fatalf("maintenance footnotes = %q, want none", footnoteTexts(footnotes, "en"))
	}
	shop := resolver.resolveContentFootnotes("m_shop_item_cell_term", []interface{}{5}, []ShopRelation{
		{ShopTitles: map[string]string{"en": "Medal Exchange"}},
		{ShopTitles: map[string]string{"en": "Event Exchange"}},
	})
	if got, want := footnoteTexts(shop, "en"), []string{"Medal Exchange / Event Exchange"}; !equalStrings(got, want) {
		t.Fatalf("shop footnotes = %q, want %q", got, want)
	}
}

func footnoteTexts(footnotes []map[string]string, language string) []string {
	values := make([]string, 0, len(footnotes))
	for _, footnote := range footnotes {
		values = append(values, footnote[language])
	}
	return values
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
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
