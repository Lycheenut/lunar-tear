package masterdataadmin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"lunar-tear/server/internal/activitygroup"
	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestActivityGroupsValidateMembersAndReferences(t *testing.T) {
	catalog := &ActivityGroupCatalog{Options: []ActivityMemberOption{
		{ActivityMember: activitygroup.ActivityMember{Kind: "chapter", ID: 1}, ChapterType: 2},
		{ActivityMember: activitygroup.ActivityMember{Kind: "chapter", ID: 2}, ChapterType: 1},
		{ActivityMember: activitygroup.ActivityMember{Kind: "event", ID: 3}, RelatedChapterID: 1},
		{ActivityMember: activitygroup.ActivityMember{Kind: "premium", ID: 4}},
		{ActivityMember: activitygroup.ActivityMember{Kind: "banner", ID: 5}},
		{ActivityMember: activitygroup.ActivityMember{Kind: "shop", ID: 6}},
		{ActivityMember: activitygroup.ActivityMember{Kind: "chapter", ID: 7}, ChapterType: 10},
	}}
	valid := activitygroup.Config{Version: activitygroup.ConfigVersion, Units: []activitygroup.ActivityUnit{
		{ID: "event", Name: "Event", Type: activitygroup.TypeVariation, Members: []activitygroup.ActivityMember{{Kind: "chapter", ID: 1}, {Kind: "event", ID: 3}}},
		{ID: "premium", Name: "Premium", Type: activitygroup.TypePremium, Members: []activitygroup.ActivityMember{{Kind: "premium", ID: 4}, {Kind: "banner", ID: 5}}},
	}, Groups: []activitygroup.ActivityGroup{{ID: "mixed", Name: "Mixed", UnitIDs: []string{"event", "premium"}}}}
	if err := ValidateActivityGroups(&valid, catalog); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*activitygroup.Config){
		"empty unit":   func(c *activitygroup.Config) { c.Units[0].Members = nil },
		"wrong source": func(c *activitygroup.Config) { c.Units[0].Type = 2 },
		"mixed chapter types": func(c *activitygroup.Config) {
			c.Units[0].Members = append(c.Units[0].Members, activitygroup.ActivityMember{Kind: "chapter", ID: 2})
		},
		"variation shop": func(c *activitygroup.Config) {
			c.Units[0].Members = append(c.Units[0].Members, activitygroup.ActivityMember{Kind: "shop", ID: 6})
		},
		"record event gacha": func(c *activitygroup.Config) {
			c.Units[0].Type = activitygroup.TypeRecord
			c.Units[0].Members[0].ID = 2
		},
		"unsupported chapter":      func(c *activitygroup.Config) { c.Units[0].Members[0].ID = 7 },
		"missing source":           func(c *activitygroup.Config) { c.Units[0].Members = c.Units[0].Members[1:] },
		"missing member":           func(c *activitygroup.Config) { c.Units[1].Members[0].ID = 99 },
		"duplicate member":         func(c *activitygroup.Config) { c.Units[1].Members = append(c.Units[1].Members, c.Units[1].Members[0]) },
		"duplicate unit":           func(c *activitygroup.Config) { c.Units[1].ID = "event" },
		"missing unit":             func(c *activitygroup.Config) { c.Groups[0].UnitIDs = []string{"missing"} },
		"empty group":              func(c *activitygroup.Config) { c.Groups[0].UnitIDs = nil },
		"duplicate unit reference": func(c *activitygroup.Config) { c.Groups[0].UnitIDs = []string{"event", "event"} },
		"duplicate group":          func(c *activitygroup.Config) { c.Groups = append(c.Groups, c.Groups[0]) },
	} {
		t.Run(name, func(t *testing.T) {
			raw, _ := json.Marshal(valid)
			var config activitygroup.Config
			_ = json.Unmarshal(raw, &config)
			mutate(&config)
			if err := ValidateActivityGroups(&config, catalog); err == nil {
				t.Fatal("invalid group configuration accepted")
			}
		})
	}
}

func TestActivityGroupGenerationIsOneTimeAndDoesNotReschedule(t *testing.T) {
	path, _ := linkedUpdateTestCatalog(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	config := gacha.DefaultConfig()
	config.Banners[999001] = gacha.BannerConfig{BannerAssetName: "limited_588", GachaMedalId: 8193, StartDatetime: 1000, EndDatetime: 2000}
	beforeConfig, _ := json.Marshal(config)
	generated, err := GenerateActivityGroups(path, nil, config, nil)
	if err != nil {
		t.Fatal(err)
	}
	afterConfig, _ := json.Marshal(config)
	if string(beforeConfig) != string(afterConfig) {
		t.Fatal("generation mutated input or schedules")
	}
	if len(generated.Groups) == 0 {
		t.Fatal("no initial groups generated")
	}
	var chapter, premium *activitygroup.ActivityUnit
	for i := range generated.Units {
		unit := &generated.Units[i]
		if unit.ID == "chapter:508" {
			chapter = unit
		}
		if unit.ID == "premium:999001" {
			premium = unit
		}
	}
	if chapter == nil || premium == nil {
		t.Fatal("missing chapter or configured premium unit")
	}
	for _, member := range []activitygroup.ActivityMember{{Kind: "shop", ID: 6005}, {Kind: "banner", ID: 33}, {Kind: "navi", ID: 15}} {
		assertActivityMember(t, chapter, member)
	}
	assertActivityMember(t, premium, activitygroup.ActivityMember{Kind: "medal", ID: 8193})
	// Configured Gacha IDs can differ from legacy banner destinations. Asset
	// and configured medal IDs, rather than those destinations, seed membership.
	if len(premium.Members) < 3 {
		t.Fatalf("premium membership = %+v", premium.Members)
	}
	again, err := GenerateActivityGroups("missing.bin", generated, config, nil)
	if err != nil || again != generated {
		t.Fatalf("existing configuration was regenerated: %v", err)
	}
	generated = &activitygroup.Config{Version: activitygroup.ConfigVersion, Units: []activitygroup.ActivityUnit{}, Groups: []activitygroup.ActivityGroup{}}
	again, err = GenerateActivityGroups("missing.bin", generated, config, nil)
	if err != nil || len(again.Groups) != 0 {
		t.Fatal("empty saved configuration was regenerated")
	}
	after, _ := os.ReadFile(path)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("generation changed master data")
	}
}

func assertActivityMember(t *testing.T, unit *activitygroup.ActivityUnit, expected activitygroup.ActivityMember) {
	t.Helper()
	for _, member := range unit.Members {
		if member == expected {
			return
		}
	}
	t.Fatalf("%s missing member %+v", unit.ID, expected)
}

func TestActivityGroupScheduleUsesExplicitMembersAndRedemptionWindow(t *testing.T) {
	path, _ := linkedUpdateTestCatalog(t)
	config := gacha.DefaultConfig()
	config.Banners[588] = gacha.BannerConfig{BannerAssetName: "limited_588", StartDatetime: 1000, EndDatetime: 2000}
	config.Banners[589] = gacha.BannerConfig{BannerAssetName: "limited_589", StartDatetime: 3000, EndDatetime: 4000}
	groups := &activitygroup.Config{Version: activitygroup.ConfigVersion, Units: []activitygroup.ActivityUnit{
		{ID: "chapter", Name: "Chapter", Type: activitygroup.TypeVariation, Members: []activitygroup.ActivityMember{{Kind: "chapter", ID: 300}, {Kind: "event", ID: 300001}, {Kind: "term", ID: 8003}, {Kind: "tip", ID: 1000}}},
		{ID: "premium", Name: "Premium", Type: activitygroup.TypePremium, Members: []activitygroup.ActivityMember{{Kind: "premium", ID: 588}, {Kind: "shop", ID: 6005}, {Kind: "term", ID: 8003}}},
	}, Groups: []activitygroup.ActivityGroup{{ID: "combined", Name: "Combined", UnitIDs: []string{"chapter", "premium"}}}}
	config.EventSchedules = map[int32]gacha.EventSchedule{300001: {StartDatetime: 1000, EndDatetime: 2000}}
	entries := []store.GachaCatalogEntry{{GachaId: 300001, GachaLabelType: model.GachaLabelEvent, RelatedEventQuestChapterId: 300}}
	catalog, err := LoadActivityGroups(path, groups, config, entries)
	if err != nil {
		t.Fatal(err)
	}
	const start, end = int64(1800000000000), int64(1800100000000)
	candidate, updated, preview, err := BuildActivitySchedule(path, groups, config, catalog, "combined", start, end)
	if err != nil {
		t.Fatal(err)
	}
	file, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	assertRawTimeByID(t, file, "m_event_quest_chapter", 0, 300, 8, start)
	assertRawTimeByID(t, file, "m_event_quest_chapter", 0, 300, 9, end)
	assertRawTimeByID(t, file, "m_shop", 0, 6005, 9, start)
	assertRawTimeByID(t, file, "m_shop", 0, 6005, 10, end+claimRedemptionGraceMillis)
	assertRawTimeByID(t, file, "m_consumable_item_term", 0, 8003, 2, end+claimRedemptionGraceMillis)
	assertRawTimeByID(t, file, "m_gacha_medal", 0, 8003, 4, end+claimRedemptionGraceMillis)
	assertRawTimeByID(t, file, "m_tip", 0, 1000, 6, end)
	if updated.Banners[588].EndDatetime != end || updated.EventSchedules[300001].EndDatetime != end+claimRedemptionGraceMillis {
		t.Fatal("Gacha schedules were not updated")
	}
	if !reflect.DeepEqual(updated.Banners[589], config.Banners[589]) || config.Banners[588].EndDatetime != 2000 {
		t.Fatal("unselected Gacha or input changed")
	}
	if config.EventSchedules[300001].EndDatetime != 2000 {
		t.Fatal("input activity schedules changed")
	}
	seen := map[string]bool{}
	for _, change := range preview {
		id := activityKey(change.ActivityMember) + change.Field
		if seen[id] {
			t.Fatal("shared member updated more than once")
		}
		seen[id] = true
	}
	if len(preview) != 13 {
		t.Fatalf("preview has %d fields, want 13", len(preview))
	}
	groups.Units[1].Members = append(groups.Units[1].Members, activitygroup.ActivityMember{Kind: "medal", ID: 8003})
	_, _, withExplicitConversion, err := BuildActivitySchedule(path, groups, config, catalog, "combined", start, end)
	if err != nil || !reflect.DeepEqual(preview, withExplicitConversion) {
		t.Fatalf("explicit conversion changed the bound schedule: %v", err)
	}
	groups.Groups[0].UnitIDs = []string{"chapter"}
	_, _, chapterOnly, err := BuildActivitySchedule(path, groups, config, catalog, "combined", start, end)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range chapterOnly {
		if change.Kind == "medal" {
			t.Fatal("non-Premium currency was bound to an automatic conversion")
		}
	}
	for _, invalid := range [][2]int64{{0, end}, {end, start}, {start, maxDatetimeMillis}} {
		if _, _, _, err := BuildActivitySchedule(path, groups, config, catalog, "combined", invalid[0], invalid[1]); err == nil {
			t.Fatal("invalid date accepted")
		}
	}
}

func TestActivityBannerPreviewPathsResolveDestinations(t *testing.T) {
	path, _ := linkedUpdateTestCatalog(t)
	file, err := memorydb.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := file.RebuildTables(nil, map[string][][]interface{}{
		"m_login_bonus":         {{20, 0, 0, 0, 0, 0, 0, "login_20"}},
		"m_event_quest_chapter": {{30, 1, 0, 0, 7}},
		"m_mom_banner": {
			{1, 0, 1, 99, "limited_45"},
			{2, 0, 21, 20, "ignored"},
			{3, 0, 25, 30, "ignored"},
			{4, 0, 22, 90, "mission_90"},
			{5, 0, 21, 999, "missing"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	file, err = memorydb.OpenBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := activityBannerPreviewPaths(file)
	want := map[int64][]string{
		1: {"gacha", "limited_45", "mom_banner.png"},
		2: {"login_bonus", "banner", "login_20.png"},
		3: {"quest", "mom_banner", "event_mom_banner_007.png"},
		4: {"mom_banner", "mom_banner_mission_90.png"},
	}
	if err != nil || !reflect.DeepEqual(paths, want) {
		t.Fatalf("banner paths = %+v, err = %v", paths, err)
	}
}

func TestActivityCatalogRestrictsSourcesAndResolvesPremiumTitles(t *testing.T) {
	path, _ := linkedUpdateTestCatalog(t)
	cacheKey, _ := filepath.Abs(filepath.Join(filepath.Dir(filepath.Dir(path)), "revisions", "0", "assetbundle"))
	previous, existed := localizationCache.Load(cacheKey)
	localizationCache.Store(cacheKey, localizationIndex{"en": {"gacha.title.limitd_588": "Celebratory Summons"}, "ja": {"gacha.title.limitd_588": "記念ガチャ"}})
	t.Cleanup(func() {
		if existed {
			localizationCache.Store(cacheKey, previous)
		} else {
			localizationCache.Delete(cacheKey)
		}
	})
	config := gacha.DefaultConfig()
	config.Banners[588] = gacha.BannerConfig{BannerAssetName: "limited_588"}
	catalog, err := LoadActivityGroups(path, nil, config, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, option := range catalog.Options {
		if option.Kind == "chapter" && option.ChapterType != 1 && option.ChapterType != 2 {
			t.Fatalf("unsupported chapter offered: %+v", option)
		}
		if option.Kind == "premium" && option.ID == 588 {
			found = true
			if option.Titles["en"] != "Celebratory Summons" || option.Titles["ja"] != "記念ガチャ" {
				t.Fatalf("missing Premium titles: %+v", option.Titles)
			}
			if !reflect.DeepEqual(option.PreviewPath, []string{"gacha", "limited_588", "banner.png"}) {
				t.Fatalf("Premium preview path = %v", option.PreviewPath)
			}
		}
	}
	if !found {
		t.Fatal("Premium option missing")
	}
}

func TestMigrateActivityGroupsSplitsTypesAndPreservesCustomNames(t *testing.T) {
	catalog := &ActivityGroupCatalog{Options: []ActivityMemberOption{
		{ActivityMember: activitygroup.ActivityMember{Kind: "chapter", ID: 1}, ChapterType: 1},
		{ActivityMember: activitygroup.ActivityMember{Kind: "chapter", ID: 2}, ChapterType: 2},
		{ActivityMember: activitygroup.ActivityMember{Kind: "chapter", ID: 3}, ChapterType: 10},
		{ActivityMember: activitygroup.ActivityMember{Kind: "event", ID: 20}, RelatedChapterID: 2},
		{ActivityMember: activitygroup.ActivityMember{Kind: "shop", ID: 30}},
		{ActivityMember: activitygroup.ActivityMember{Kind: "banner", ID: 40}},
		{ActivityMember: activitygroup.ActivityMember{Kind: "premium", ID: 588}, Titles: map[string]string{"en": "Celebratory Summons"}},
	}}
	legacy := &activitygroup.Config{Version: 1, Units: []activitygroup.ActivityUnit{
		{ID: "mixed", Name: "Custom name", Type: 1, Members: []activitygroup.ActivityMember{{Kind: "chapter", ID: 1}, {Kind: "chapter", ID: 2}, {Kind: "event", ID: 20}, {Kind: "shop", ID: 30}, {Kind: "banner", ID: 40}}},
		{ID: "unsupported", Name: "Other", Type: 1, Members: []activitygroup.ActivityMember{{Kind: "chapter", ID: 3}}},
		{ID: "premium:588", Name: "limited_588", Type: 2, Members: []activitygroup.ActivityMember{{Kind: "premium", ID: 588}}},
	}, Groups: []activitygroup.ActivityGroup{
		{ID: "combined", Name: "Custom group", UnitIDs: []string{"mixed", "unsupported"}},
		{ID: "unsupported", Name: "Other", UnitIDs: []string{"unsupported"}},
		{ID: "premium:588", Name: "limited_588", UnitIDs: []string{"premium:588"}},
	}}
	config := gacha.DefaultConfig()
	config.Banners[588] = gacha.BannerConfig{BannerAssetName: "limited_588"}
	before, _ := json.Marshal(legacy)
	migrated, err := migrateActivityGroups(legacy, catalog, config)
	if err != nil {
		t.Fatal(err)
	}
	if len(migrated.Units) != 3 || len(migrated.Groups) != 2 || len(migrated.Groups[0].UnitIDs) != 2 {
		t.Fatalf("unexpected migration: %+v", migrated)
	}
	if migrated.Units[0].Type != activitygroup.TypeRecord || migrated.Units[1].Type != activitygroup.TypeVariation || migrated.Units[2].Type != activitygroup.TypePremium {
		t.Fatal("incorrect type mapping")
	}
	assertActivityMember(t, &migrated.Units[0], activitygroup.ActivityMember{Kind: "shop", ID: 30})
	assertActivityMember(t, &migrated.Units[1], activitygroup.ActivityMember{Kind: "event", ID: 20})
	if migrated.Units[0].Name != "Custom name" || migrated.Groups[0].Name != "Custom group" || migrated.Groups[1].Name != "Celebratory Summons" {
		t.Fatal("names not preserved/restored")
	}
	after, _ := json.Marshal(legacy)
	if string(before) != string(after) {
		t.Fatal("migration changed input")
	}
	if again, err := GenerateActivityGroups("missing.bin", migrated, config, nil); err != nil || again != migrated {
		t.Fatal("version 2 was regenerated")
	}
}
