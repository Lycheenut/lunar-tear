package masterdataadmin

import (
	"encoding/json"
	"os"
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
	}}
	valid := activitygroup.Config{Version: 1, Units: []activitygroup.ActivityUnit{
		{ID: "event", Name: "Event", Type: 1, Members: []activitygroup.ActivityMember{{Kind: "chapter", ID: 1}, {Kind: "chapter", ID: 2}, {Kind: "event", ID: 3}}},
		{ID: "premium", Name: "Premium", Type: 2, Members: []activitygroup.ActivityMember{{Kind: "premium", ID: 4}, {Kind: "banner", ID: 5}}},
	}, Groups: []activitygroup.ActivityGroup{{ID: "mixed", Name: "Mixed", UnitIDs: []string{"event", "premium"}}}}
	if err := ValidateActivityGroups(&valid, catalog); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*activitygroup.Config){
		"empty unit":               func(c *activitygroup.Config) { c.Units[0].Members = nil },
		"wrong source":             func(c *activitygroup.Config) { c.Units[0].Type = 2 },
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
	generated = &activitygroup.Config{Version: 1, Units: []activitygroup.ActivityUnit{}, Groups: []activitygroup.ActivityGroup{}}
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
	groups := &activitygroup.Config{Version: 1, Units: []activitygroup.ActivityUnit{
		{ID: "chapter", Name: "Chapter", Type: 1, Members: []activitygroup.ActivityMember{{Kind: "chapter", ID: 300}, {Kind: "event", ID: 300001}, {Kind: "shop", ID: 6005}, {Kind: "term", ID: 8003}, {Kind: "tip", ID: 1000}}},
		{ID: "premium", Name: "Premium", Type: 2, Members: []activitygroup.ActivityMember{{Kind: "premium", ID: 588}, {Kind: "shop", ID: 6005}, {Kind: "medal", ID: 8003}}},
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
	for _, invalid := range [][2]int64{{0, end}, {end, start}, {start, maxDatetimeMillis}} {
		if _, _, _, err := BuildActivitySchedule(path, groups, config, catalog, "combined", invalid[0], invalid[1]); err == nil {
			t.Fatal("invalid date accepted")
		}
	}
}
