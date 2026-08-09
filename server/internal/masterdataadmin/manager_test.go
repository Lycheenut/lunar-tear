package masterdataadmin

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestActivitySpecsContainSelectedAndRelatedTables(t *testing.T) {
	if got, want := len(activityTableSpecs), 30; got != want {
		t.Fatalf("activity spec count = %d, want %d", got, want)
	}
	wantPrimary := map[string]bool{
		"m_big_hunt_schedule": true, "m_consumable_item_term": true,
		"m_enhance_campaign": true, "m_event_quest_chapter": true,
		"m_event_quest_daily_group": true, "m_event_quest_labyrinth_season": true,
		"m_login_bonus": true, "m_maintenance": true, "m_mom_banner": true,
		"m_omikuji": true, "m_pvp_season": true, "m_quest_campaign": true,
		"m_shop": true, "m_shop_item_cell_term": true,
	}
	wantRelated := map[string]bool{
		"m_enhance_campaign_target_group": true,
		"m_event_quest_link":              true, "m_event_quest_display_item_group": true,
		"m_event_quest_sequence_group":                true,
		"m_event_quest_daily_group_target_chapter":    true,
		"m_event_quest_daily_group_complete_reward":   true,
		"m_event_quest_daily_group_message":           true,
		"m_event_quest_labyrinth_season_reward_group": true,
		"m_maintenance_group":                         true, "m_pvp_season_grouping": true,
		"m_pvp_weekly_rank_reward_rank_group": true,
		"m_pvp_season_rank_reward_rank_group": true,
		"m_pvp_grade_group":                   true, "m_quest_campaign_target_group": true,
		"m_quest_campaign_effect_group": true, "m_shop_item_cell_group": true,
	}
	seen := make(map[string]bool, len(activityTableSpecs))
	primaryCount := 0
	for _, spec := range activityTableSpecs {
		if seen[spec.Name] {
			t.Fatalf("duplicate table spec %q", spec.Name)
		}
		seen[spec.Name] = true
		if spec.Primary != wantPrimary[spec.Name] {
			t.Errorf("table %q primary = %v, want %v", spec.Name, spec.Primary, wantPrimary[spec.Name])
		}
		if !spec.Primary && !wantRelated[spec.Name] {
			t.Errorf("unexpected related table %q", spec.Name)
		}
		if spec.Primary {
			primaryCount++
			if len(spec.pairs()) == 0 {
				t.Errorf("primary table %q has no start/end pair", spec.Name)
			}
		}
		if len(spec.Fields) == 0 || !spec.Fields[0].PrimaryKey {
			t.Errorf("table %q has no primary key field", spec.Name)
		}
	}
	if primaryCount != len(wantPrimary) {
		t.Fatalf("primary table count = %d, want %d", primaryCount, len(wantPrimary))
	}
	if len(seen)-primaryCount != len(wantRelated) {
		t.Fatalf("related table count = %d, want %d", len(seen)-primaryCount, len(wantRelated))
	}
}

func TestBuildUpdateAgainstCurrentMasterData(t *testing.T) {
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
	if catalog.TableCount != len(activityTableSpecs) {
		t.Fatalf("loaded %d activity tables, want %d", catalog.TableCount, len(activityTableSpecs))
	}
	if catalog.PrimaryCount != 14 || catalog.RelatedCount != 16 {
		t.Fatalf("loaded primary/related counts = %d/%d, want 14/16", catalog.PrimaryCount, catalog.RelatedCount)
	}
	if catalog.RowCount == 0 {
		t.Fatal("loaded catalog has no rows")
	}

	var table Table
	for _, candidate := range catalog.Tables {
		if len(candidate.Rows) > 0 && len(candidate.Pairs) > 0 {
			table = candidate
			break
		}
	}
	if table.Name == "" {
		t.Fatal("no editable schedule row found")
	}
	row := table.Rows[0]
	endField := table.Pairs[0].End
	current := row.Times[endField]
	updated := current + 1000
	if current == 0 || updated > maxDatetimeMillis {
		updated = 1
	}
	candidate, result, err := BuildUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes:         []Change{{Table: table.Name, Row: row.Index, Field: endField, Value: updated}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ChangedCells != 1 || result.ChangedRows != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	rows, exists, err := rebuilt.TableRows(table.Name)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("rebuilt table %q is absent", table.Name)
	}
	spec, ok := findSpec(table.Name)
	if !ok {
		t.Fatalf("spec %q is absent", table.Name)
	}
	field, ok := findTimeField(spec, endField)
	if !ok {
		t.Fatalf("field %q is absent", endField)
	}
	got, err := valueAsInt64(rows[row.Index][field.Index])
	if err != nil {
		t.Fatal(err)
	}
	if got != updated {
		t.Fatalf("rebuilt value = %d, want %d", got, updated)
	}
}

func TestBuildUpdateSupportsAllScalarKindsAndRejectsPrimaryKeys(t *testing.T) {
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
	var table Table
	var related Table
	for _, candidate := range catalog.Tables {
		if candidate.Name == "m_mom_banner" {
			table = candidate
		}
		if candidate.Name == "m_maintenance_group" {
			related = candidate
		}
	}
	if len(table.Rows) == 0 || len(related.Rows) == 0 {
		t.Fatal("test tables have no rows")
	}
	row := table.Rows[0]
	relatedRow := related.Rows[0]
	sortOrder, err := strconv.ParseInt(row.Values["SortOrderDesc"], 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	updatedAsset := row.Values["BannerAssetName"] + "_edited"
	updatedEmphasis := row.Values["IsEmphasis"] != "true"
	updatedPriority, err := strconv.ParseInt(relatedRow.Values["Priority"], 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	updatedPriority++
	updatedBlockValue := relatedRow.Values["BlockFunctionValue"] + "_edited"
	candidate, result, err := BuildUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes: []Change{
			{Table: table.Name, Row: row.Index, Field: "SortOrderDesc", Value: strconv.FormatInt(sortOrder+1, 10)},
			{Table: table.Name, Row: row.Index, Field: "BannerAssetName", Value: updatedAsset},
			{Table: table.Name, Row: row.Index, Field: "IsEmphasis", Value: strconv.FormatBool(updatedEmphasis)},
			{Table: related.Name, Row: relatedRow.Index, Field: "Priority", Value: strconv.FormatInt(updatedPriority, 10)},
			{Table: related.Name, Row: relatedRow.Index, Field: "BlockFunctionValue", Value: updatedBlockValue},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ChangedCells != 5 || result.ChangedRows != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	rows, _, err := rebuilt.TableRows(table.Name)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := valueAsInt64(rows[row.Index][1]); err != nil || got != sortOrder+1 {
		t.Fatalf("SortOrderDesc = %d, %v; want %d", got, err, sortOrder+1)
	}
	if got := rows[row.Index][4]; got != updatedAsset {
		t.Fatalf("BannerAssetName = %#v, want %#v", got, updatedAsset)
	}
	if got := rows[row.Index][5]; got != updatedEmphasis {
		t.Fatalf("IsEmphasis = %#v, want %v", got, updatedEmphasis)
	}
	relatedRows, _, err := rebuilt.TableRows(related.Name)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := valueAsInt64(relatedRows[relatedRow.Index][2]); err != nil || got != updatedPriority {
		t.Fatalf("Priority = %d, %v; want %d", got, err, updatedPriority)
	}
	if got := relatedRows[relatedRow.Index][5]; got != updatedBlockValue {
		t.Fatalf("BlockFunctionValue = %#v, want %#v", got, updatedBlockValue)
	}

	_, _, err = BuildUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes: []Change{{
			Table: table.Name, Row: row.Index, Field: "MomBannerId", Value: row.Values["MomBannerId"],
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "primary key") {
		t.Fatalf("primary key update error = %v", err)
	}
	_, _, err = BuildUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes: []Change{{
			Table: related.Name, Row: relatedRow.Index, Field: "ApiPath", Value: relatedRow.Values["ApiPath"] + "/edited",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "primary key") {
		t.Fatalf("composite primary key update error = %v", err)
	}
}

func findSpec(name string) (tableSpec, bool) {
	for _, spec := range activityTableSpecs {
		if spec.Name == name {
			return spec, true
		}
	}
	return tableSpec{}, false
}
