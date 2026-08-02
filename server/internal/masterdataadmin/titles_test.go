package masterdataadmin

import "testing"

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
				"campaign.name.02.02.01":        "Glorious Success Rate Up Campaign",
				"campaign.name.01.05.01":        "Drop Bonus Campaign",
			},
			"ja": {"gacha.title.limited_45": "記念ガチャ"},
			"ko": {},
		},
		shopTextIDs:          map[int64]int64{55: 101},
		missionTermTextIDs:   map[int64]int64{77: 201},
		consumableTermKeys:   map[int64][]string{5: {"consumable_item.name.110004"}},
		importantEffectTexts: map[int64]int64{9: 401},
		questEffectTypes:     map[int64]int64{20: 5},
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
		{name: "enhance campaign", table: "m_enhance_campaign", row: []interface{}{1, 2, 2}, want: "Glorious Success Rate Up Campaign"},
		{name: "quest drop bonus", table: "m_quest_campaign", row: []interface{}{1, 2, 20}, want: "Drop Bonus Campaign"},
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
