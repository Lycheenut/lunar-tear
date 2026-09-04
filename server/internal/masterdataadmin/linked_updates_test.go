package masterdataadmin

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestGachaMomBannerPreviewCascadesToMedalShopAndCurrencyTerm(t *testing.T) {
	path, catalog := linkedUpdateTestCatalog(t)
	banner := catalogRowByID(t, catalog, "m_mom_banner", "MomBannerId", "4")
	oldStart := mustParseInt64(t, banner.Values["StartDatetime"])
	end := mustParseInt64(t, banner.Values["EndDatetime"])
	redemptionEnd := end + 48*60*60*1000
	preview, err := PreviewUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes: []Change{{
			Table: "m_mom_banner", Row: banner.Index, Field: "StartDatetime",
			Value: strconv.FormatInt(oldStart+3600000, 10),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	impact := impactByKind(t, preview, "Gacha")
	targetByIdentity(t, impact, "m_gacha_medal", "GachaMedalId", "8003")
	assertGeneratedTarget(t, impact, "m_shop", "ShopId", "8003", "StartDatetime")
	assertGeneratedTarget(t, impact, "m_consumable_item_term", "ConsumableItemTermId", "8003", "StartDatetime")

	candidate, result, err := BuildUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes: []Change{{
			Table: "m_mom_banner", Row: banner.Index, Field: "StartDatetime",
			Value: strconv.FormatInt(oldStart+3600000, 10),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ChangedCells != 3 || result.ChangedRows != 3 {
		t.Fatalf("Gacha cascade result = %+v, want 3 cells across 3 rows", result)
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	assertRawTimeByID(t, rebuilt, "m_shop", 0, 8003, 9, oldStart+3600000)
	assertRawTimeByID(t, rebuilt, "m_shop", 0, 8003, 10, redemptionEnd)
	assertRawTimeByID(t, rebuilt, "m_consumable_item_term", 0, 8003, 1, oldStart+3600000)
	assertRawTimeByID(t, rebuilt, "m_consumable_item_term", 0, 8003, 2, redemptionEnd)
	assertRawTimeByID(t, rebuilt, "m_gacha_medal", 0, 8003, 4, redemptionEnd)
}

func TestGachaMomBannerEndCascadeAddsFortyEightHourRedemptionWindow(t *testing.T) {
	path, catalog := linkedUpdateTestCatalog(t)
	banner := catalogRowByID(t, catalog, "m_mom_banner", "MomBannerId", "4")
	newEnd := mustParseInt64(t, banner.Values["EndDatetime"]) + 3600000
	redemptionEnd := newEnd + 48*60*60*1000
	request := UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes: []Change{{
			Table: "m_mom_banner", Row: banner.Index, Field: "EndDatetime",
			Value: strconv.FormatInt(newEnd, 10),
		}},
	}

	preview, err := PreviewUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	impact := impactByKind(t, preview, "Gacha")
	assertGeneratedChangeValue(t,
		targetByIdentity(t, impact, "m_gacha_medal", "GachaMedalId", "8003"),
		"AutoConvertDatetime", strconv.FormatInt(redemptionEnd, 10))
	assertGeneratedChangeValue(t,
		targetByIdentity(t, impact, "m_shop", "ShopId", "8003"),
		"EndDatetime", strconv.FormatInt(redemptionEnd, 10))
	assertGeneratedChangeValue(t,
		targetByIdentity(t, impact, "m_consumable_item_term", "ConsumableItemTermId", "8003"),
		"EndDatetime", strconv.FormatInt(redemptionEnd, 10))

	candidate, result, err := BuildUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.ChangedCells != 4 || result.ChangedRows != 4 {
		t.Fatalf("Gacha end cascade result = %+v, want 4 cells across 4 rows", result)
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	assertRawTimeByID(t, rebuilt, "m_gacha_medal", 0, 8003, 4, redemptionEnd)
	assertRawTimeByID(t, rebuilt, "m_shop", 0, 8003, 10, redemptionEnd)
	assertRawTimeByID(t, rebuilt, "m_consumable_item_term", 0, 8003, 2, redemptionEnd)
}

func TestEventQuestPreviewIncludesAllCertainDownstreamTypes(t *testing.T) {
	path, catalog := linkedUpdateTestCatalog(t)
	chapter := catalogRowByID(t, catalog, "m_event_quest_chapter", "EventQuestChapterId", "508")
	oldEnd := mustParseInt64(t, chapter.Values["EndDatetime"])
	newEnd := oldEnd + 3600000
	redemptionEnd := newEnd + 48*60*60*1000
	preview, err := PreviewUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes: []Change{{
			Table: "m_event_quest_chapter", Row: chapter.Index, Field: "EndDatetime",
			Value: strconv.FormatInt(newEnd, 10),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	impact := impactByKind(t, preview, "EventQuestChapter")
	assertGeneratedTarget(t, impact, "m_shop", "ShopId", "6005", "EndDatetime")
	assertGeneratedChangeValue(t,
		targetByIdentity(t, impact, "m_shop", "ShopId", "6005"),
		"EndDatetime", strconv.FormatInt(redemptionEnd, 10))
	assertGeneratedChangeValue(t,
		targetByRelation(t, impact, "活动币有效期"),
		"EndDatetime", strconv.FormatInt(redemptionEnd, 10))
	assertGeneratedTarget(t, impact, "m_mom_banner", "MomBannerId", "33", "EndDatetime")
	assertGeneratedTarget(t, impact, "m_navi_cut_in", "NaviCutInId", "15", "EndDatetime")
	assertRelationExists(t, impact, "限时任务档期")
}

func TestEventQuestSingleTimeEditCopiesWholeScheduleAndSelectsOneNaviCutIn(t *testing.T) {
	path, catalog := linkedUpdateTestCatalog(t)
	chapter := catalogRowByID(t, catalog, "m_event_quest_chapter", "EventQuestChapterId", "508")
	finalTime := chapter.Values["EndDatetime"]
	preview, err := PreviewUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes: []Change{{
			Table: "m_event_quest_chapter", Row: chapter.Index, Field: "StartDatetime", Value: finalTime,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	impact := impactByKind(t, preview, "EventQuestChapter")
	naviTargets := targetsByRelation(impact, "活动 NaviCutIn")
	if len(naviTargets) != 1 || previewIdentityValue(naviTargets[0], "NaviCutInId") != "15" {
		t.Fatalf("selected NaviCutIns = %+v, want only NaviCutInId=15", naviTargets)
	}
	missionBanner := targetByIdentity(t, impact, "m_mom_banner", "MomBannerId", "91056")
	assertGeneratedChangeValue(t, missionBanner, "StartDatetime", finalTime)
	assertGeneratedChangeValue(t, missionBanner, "EndDatetime", finalTime)
}

func TestLoginBonusNonPairTimeEditStillCopiesWholeSchedule(t *testing.T) {
	path, catalog := linkedUpdateTestCatalog(t)
	loginBonus := catalogRowByID(t, catalog, "m_login_bonus", "LoginBonusId", "17")
	stampEnd := mustParseInt64(t, loginBonus.Values["StampReceiveEndDatetime"])
	preview, err := PreviewUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes: []Change{{
			Table: "m_login_bonus", Row: loginBonus.Index, Field: "StampReceiveEndDatetime",
			Value: strconv.FormatInt(stampEnd+3600000, 10),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	target := targetByIdentity(t, impactByKind(t, preview, "LoginBonus"), "m_mom_banner", "MomBannerId", "1")
	assertGeneratedChangeValue(t, target, "EndDatetime", loginBonus.Values["EndDatetime"])
}

func TestEventQuestLinkChangePreviewsOldAndNewShopsWithoutCascade(t *testing.T) {
	path, catalog := linkedUpdateTestCatalog(t)
	chapter := catalogRowByID(t, catalog, "m_event_quest_chapter", "EventQuestChapterId", "508")
	oldLinkID := chapter.Values["EventQuestLinkId"]
	var newLinkID, newShopID string
	for _, row := range catalogTableByName(t, catalog, "m_event_quest_link").Rows {
		if row.Values["DestinationDomainType"] != "3" || row.Values["EventQuestLinkId"] == oldLinkID {
			continue
		}
		newLinkID = row.Values["EventQuestLinkId"]
		newShopID = row.Values["DestinationDomainId"]
		break
	}
	if newLinkID == "" {
		t.Fatal("a second shop EventQuestLink was not found")
	}
	preview, err := PreviewUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes: []Change{{
			Table: "m_event_quest_chapter", Row: chapter.Index, Field: "EventQuestLinkId", Value: newLinkID,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	impact := impactByKind(t, preview, "EventQuestChapter")
	oldShop := targetByIdentity(t, impact, "m_shop", "ShopId", "6005")
	newShop := targetByIdentity(t, impact, "m_shop", "ShopId", newShopID)
	if len(oldShop.Changes) != 0 || oldShop.Note == "" || len(newShop.Changes) != 0 || newShop.Note == "" {
		t.Fatalf("link change should report old and new shops without cascading: old=%+v new=%+v", oldShop, newShop)
	}
}

func TestLoginBonusPreviewCascadesBannerButOnlyReportsSharedMamaMedalContent(t *testing.T) {
	path, catalog := linkedUpdateTestCatalog(t)
	loginBonus := catalogRowByID(t, catalog, "m_login_bonus", "LoginBonusId", "25")
	oldEnd := mustParseInt64(t, loginBonus.Values["EndDatetime"])
	preview, err := PreviewUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes: []Change{{
			Table: "m_login_bonus", Row: loginBonus.Index, Field: "EndDatetime",
			Value: strconv.FormatInt(oldEnd+3600000, 10),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	impact := impactByKind(t, preview, "LoginBonus")
	assertGeneratedTarget(t, impact, "m_mom_banner", "MomBannerId", "14", "EndDatetime")
	shop := targetByIdentity(t, impact, "m_shop", "ShopId", "102")
	if len(shop.Changes) != 0 || shop.Note == "" {
		t.Fatalf("shared monthly shop preview = %+v, want note without automatic changes", shop)
	}
	term := targetByIdentity(t, impact, "m_consumable_item_term", "ConsumableItemTermId", "7002")
	if len(term.Changes) != 0 || term.Note == "" {
		t.Fatalf("shared monthly term preview = %+v, want note without automatic changes", term)
	}
}

func TestEventQuestPreviewReportsMonthlyMamaMedalShopFromLinkedMission(t *testing.T) {
	path, catalog := linkedUpdateTestCatalog(t)
	file, err := memorydb.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	planner := &linkedUpdatePlanner{file: file, rows: make(map[string][][]interface{})}
	index, err := planner.relations()
	if err != nil {
		t.Fatal(err)
	}
	var chapterID int64
	for candidateID, terms := range index.missionTermsByChapter {
		for _, term := range terms {
			termRows, err := planner.tableRows(term.table)
			if err != nil {
				t.Fatal(err)
			}
			termID, _ := integerAt(termRows[term.row], 0)
			for _, currencyID := range index.monthlyCurrenciesByMissionTerm[termID] {
				if len(index.shopsByCurrency[currencyID]) != 0 {
					chapterID = candidateID
					break
				}
			}
			if chapterID != 0 {
				break
			}
		}
		if chapterID != 0 {
			break
		}
	}
	if chapterID == 0 {
		t.Fatal("an event chapter with a monthly Mama medal mission was not found")
	}
	chapter := catalogRowByID(t, catalog, "m_event_quest_chapter", "EventQuestChapterId", strconv.FormatInt(chapterID, 10))
	oldEnd := mustParseInt64(t, chapter.Values["EndDatetime"])
	preview, err := PreviewUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes: []Change{{
			Table: "m_event_quest_chapter", Row: chapter.Index, Field: "EndDatetime",
			Value: strconv.FormatInt(oldEnd+3600000, 10),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	shop := targetByRelation(t, impactByKind(t, preview, "EventQuestChapter"), "限时任务关联的月度妈妈兑换商店")
	if len(shop.Changes) != 0 || shop.Note == "" {
		t.Fatalf("shared monthly mission shop preview = %+v, want note without automatic changes", shop)
	}
}

func linkedUpdateTestCatalog(t *testing.T) (string, *Catalog) {
	t.Helper()
	path := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master-data asset is not installed")
	} else if err != nil {
		t.Fatal(err)
	}
	catalog, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return path, catalog
}

func catalogRowByID(t *testing.T, catalog *Catalog, tableName, fieldName, value string) Row {
	t.Helper()
	for _, table := range catalog.Tables {
		if table.Name != tableName {
			continue
		}
		for _, row := range table.Rows {
			if row.Values[fieldName] == value {
				return row
			}
		}
	}
	t.Fatalf("%s %s=%s not found", tableName, fieldName, value)
	return Row{}
}

func catalogTableByName(t *testing.T, catalog *Catalog, tableName string) Table {
	t.Helper()
	for _, table := range catalog.Tables {
		if table.Name == tableName {
			return table
		}
	}
	t.Fatalf("table %s not found", tableName)
	return Table{}
}

func impactByKind(t *testing.T, preview UpdatePreview, kind string) UpdateImpactPreview {
	t.Helper()
	for _, impact := range preview.Impacts {
		if impact.Kind == kind {
			return impact
		}
	}
	t.Fatalf("impact %q not found in %+v", kind, preview.Impacts)
	return UpdateImpactPreview{}
}

func assertGeneratedTarget(t *testing.T, impact UpdateImpactPreview, table, idField, idValue, field string) {
	t.Helper()
	target := targetByIdentity(t, impact, table, idField, idValue)
	for _, change := range target.Changes {
		if change.Field == field && change.Generated {
			return
		}
	}
	t.Fatalf("%s %s=%s has no generated %s change: %+v", table, idField, idValue, field, target)
}

func assertGeneratedChangeValue(t *testing.T, target RecordPreview, field, value string) {
	t.Helper()
	for _, change := range target.Changes {
		if change.Field == field && change.Generated && change.After == value {
			return
		}
	}
	t.Fatalf("%s has no generated %s=%s change: %+v", previewIdentityValue(target, ""), field, value, target)
}

func targetsByRelation(impact UpdateImpactPreview, relation string) []RecordPreview {
	var targets []RecordPreview
	for _, target := range impact.Downstream {
		if target.Relation == relation {
			targets = append(targets, target)
		}
	}
	return targets
}

func previewIdentityValue(target RecordPreview, field string) string {
	for _, identity := range target.Identity {
		if field == "" || identity.Name == field {
			return identity.Value
		}
	}
	return ""
}

func targetByIdentity(t *testing.T, impact UpdateImpactPreview, table, field, value string) RecordPreview {
	t.Helper()
	for _, target := range impact.Downstream {
		if target.Table != table {
			continue
		}
		for _, identity := range target.Identity {
			if identity.Name == field && identity.Value == value {
				return target
			}
		}
	}
	t.Fatalf("downstream %s %s=%s not found: %+v", table, field, value, impact.Downstream)
	return RecordPreview{}
}

func assertRelationExists(t *testing.T, impact UpdateImpactPreview, relation string) {
	t.Helper()
	for _, target := range impact.Downstream {
		if target.Relation == relation {
			return
		}
	}
	t.Fatalf("relation %q not found: %+v", relation, impact.Downstream)
}

func targetByRelation(t *testing.T, impact UpdateImpactPreview, relation string) RecordPreview {
	t.Helper()
	for _, target := range impact.Downstream {
		if target.Relation == relation {
			return target
		}
	}
	t.Fatalf("relation %q not found: %+v", relation, impact.Downstream)
	return RecordPreview{}
}

func mustParseInt64(t *testing.T, value string) int64 {
	t.Helper()
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func assertRawTimeByID(t *testing.T, file *memorydb.File, table string, idColumn int, id int64, timeColumn int, want int64) {
	t.Helper()
	rows, exists, err := file.TableRows(table)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("table %s is absent", table)
	}
	for _, row := range rows {
		rowID, _ := integerAt(row, idColumn)
		if rowID != id {
			continue
		}
		got, err := valueAsInt64(row[timeColumn])
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s id %d time = %d, want %d", table, id, got, want)
		}
		return
	}
	t.Fatalf("%s id %d not found", table, id)
}
