package masterdataadmin

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestActivitySpecsContainSelectedAndRelatedTables(t *testing.T) {
	if got, want := len(activityTableSpecs), 40; got != want {
		t.Fatalf("activity spec count = %d, want %d", got, want)
	}
	wantPrimary := map[string]bool{
		"m_beginner_campaign": true, "m_big_hunt_schedule": true,
		"m_comeback_campaign": true, "m_consumable_item_term": true,
		"m_dokan": true, "m_enhance_campaign": true, "m_event_quest_chapter": true,
		"m_event_quest_daily_group": true, "m_event_quest_labyrinth_season": true,
		"m_gacha_medal": true,
		"m_login_bonus": true, "m_maintenance": true, "m_mission_term": true, "m_mom_banner": true,
		"m_navi_cut_in": true,
		"m_omikuji":     true, "m_pvp_season": true, "m_quest_campaign": true,
		"m_shop": true, "m_shop_item_cell_term": true, "m_tip": true,
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
	wantDelivery := map[string]bool{
		"m_login_bonus_stamp": true, "m_mission_reward": true,
		"m_shop_item_content_possession": true,
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
		if spec.Delivery != wantDelivery[spec.Name] {
			t.Errorf("table %q delivery = %v, want %v", spec.Name, spec.Delivery, wantDelivery[spec.Name])
		}
		if !spec.Primary && !spec.Delivery && !wantRelated[spec.Name] {
			t.Errorf("unexpected related table %q", spec.Name)
		}
		if spec.Primary {
			primaryCount++
			if len(spec.Times) == 0 {
				t.Errorf("primary table %q has no datetime field", spec.Name)
			}
		}
		if len(spec.Fields) == 0 || !spec.Fields[0].PrimaryKey {
			t.Errorf("table %q has no primary key field", spec.Name)
		}
	}
	if primaryCount != len(wantPrimary) {
		t.Fatalf("primary table count = %d, want %d", primaryCount, len(wantPrimary))
	}
	if len(seen)-primaryCount-len(wantDelivery) != len(wantRelated) {
		t.Fatalf("related table count = %d, want %d", len(seen)-primaryCount-len(wantDelivery), len(wantRelated))
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
	if catalog.PrimaryCount != 21 || catalog.RelatedCount != 16 || catalog.DeliveryCount != 3 {
		t.Fatalf("loaded primary/related/delivery counts = %d/%d/%d, want 21/16/3", catalog.PrimaryCount, catalog.RelatedCount, catalog.DeliveryCount)
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

func TestGachaMedalActivityTableCanUpdateGachaLink(t *testing.T) {
	path, catalog := linkedUpdateTestCatalog(t)
	medal := catalogRowByID(t, catalog, "m_gacha_medal", "GachaMedalId", "8193")
	candidate, result, err := BuildUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes: []Change{{
			Table: "m_gacha_medal", Row: medal.Index, Field: "ShopTransitionGachaId", Value: "543",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ChangedCells != 1 || result.ChangedRows != 1 {
		t.Fatalf("Gacha Medal link update result = %+v, want one changed cell", result)
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	assertRawTimeByID(t, rebuilt, "m_gacha_medal", 0, 8193, 3, 543)
}

func TestLoginBonusStampIsDeliveryTableAndEditable(t *testing.T) {
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
	for _, candidate := range catalog.Tables {
		if candidate.Name == "m_login_bonus_stamp" {
			table = candidate
			break
		}
	}
	if !table.Delivery || table.Primary || len(table.Rows) == 0 {
		t.Fatalf("unexpected login bonus stamp table: delivery=%v primary=%v rows=%d", table.Delivery, table.Primary, len(table.Rows))
	}
	for index, field := range table.Fields {
		if field.PrimaryKey != (index < 3) {
			t.Fatalf("field %s primaryKey = %v, want %v", field.Name, field.PrimaryKey, index < 3)
		}
	}

	row := table.Rows[0]
	count, err := strconv.ParseInt(row.Values["RewardCount"], 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	candidate, _, err := BuildUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes:         []Change{{Table: table.Name, Row: row.Index, Field: "RewardCount", Value: count + 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	rows, _, err := rebuilt.TableRows(table.Name)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := valueAsInt64(rows[row.Index][5]); err != nil || got != count+1 {
		t.Fatalf("RewardCount = %d, %v; want %d", got, err, count+1)
	}
}

func TestMissionRewardIsDeliveryTableWithLocalizedSources(t *testing.T) {
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
	var termTable Table
	for _, candidate := range catalog.Tables {
		if candidate.Name == "m_mission_reward" {
			table = candidate
		}
		if candidate.Name == "m_mission_term" {
			termTable = candidate
		}
	}
	if !table.Delivery || table.Primary || len(table.Rows) == 0 {
		t.Fatalf("unexpected mission reward table: delivery=%v primary=%v rows=%d", table.Delivery, table.Primary, len(table.Rows))
	}
	for index, field := range table.Fields {
		if field.PrimaryKey != (index == 0) {
			t.Fatalf("field %s primaryKey = %v, want %v", field.Name, field.PrimaryKey, index == 0)
		}
	}
	if table.Fields[1].Name != "PossessionType" || table.Fields[1].Type != "PossessionType" || table.Fields[2].Name != "PossessionId" {
		t.Fatalf("mission reward fields do not expose the shared reward editor pair: %+v", table.Fields)
	}
	if len(catalog.MissionSources.Groups) == 0 || len(catalog.MissionSources.Missions) == 0 {
		t.Fatal("mission source catalog is empty")
	}
	groupByID := make(map[int64]MissionGroupSource)
	for _, group := range catalog.MissionSources.Groups {
		groupByID[group.MissionGroupID] = group
	}
	rewardIDs := make(map[string]bool)
	for _, row := range table.Rows {
		rewardIDs[row.Values["MissionRewardId"]] = true
	}
	localizedGroup := false
	localizedMission := false
	termIDs := make(map[int64]bool)
	for _, row := range termTable.Rows {
		termID, err := strconv.ParseInt(row.Values["MissionTermId"], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		termIDs[termID] = true
	}
	termSourceFound := false
	for _, group := range catalog.MissionSources.Groups {
		localizedGroup = localizedGroup || group.Names["en"] != "" && group.Names["ja"] != "" && group.Names["ko"] != ""
	}
	for _, mission := range catalog.MissionSources.Missions {
		if _, ok := groupByID[mission.MissionGroupID]; !ok {
			t.Fatalf("mission %d references missing group %d", mission.MissionID, mission.MissionGroupID)
		}
		if !rewardIDs[strconv.FormatInt(mission.MissionRewardID, 10)] {
			t.Fatalf("mission %d references missing reward %d", mission.MissionID, mission.MissionRewardID)
		}
		localizedMission = localizedMission || mission.Names["en"] != "" && mission.Names["ja"] != "" && mission.Names["ko"] != ""
		termSourceFound = termSourceFound || termIDs[mission.MissionTermID]
	}
	if !localizedGroup || !localizedMission || !termSourceFound {
		t.Fatalf("mission sources missing: group=%v mission=%v term=%v", localizedGroup, localizedMission, termSourceFound)
	}

	row := table.Rows[0]
	count, err := strconv.ParseInt(row.Values["Count"], 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	candidate, _, err := BuildUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes:         []Change{{Table: table.Name, Row: row.Index, Field: "Count", Value: count + 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	rows, _, err := rebuilt.TableRows(table.Name)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := valueAsInt64(rows[row.Index][3]); err != nil || got != count+1 {
		t.Fatalf("Count = %d, %v; want %d", got, err, count+1)
	}
}

func TestShopContentPossessionIsLocalizedDeliveryTableAndEditable(t *testing.T) {
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
	table := findCatalogTable(catalog, "m_shop_item_content_possession")
	if !table.Delivery || table.Primary || len(table.Rows) == 0 {
		t.Fatalf("unexpected shop content table: delivery=%v primary=%v rows=%d", table.Delivery, table.Primary, len(table.Rows))
	}
	for index, field := range table.Fields {
		if field.PrimaryKey != (index == 0) {
			t.Fatalf("field %s primaryKey = %v, want %v", field.Name, field.PrimaryKey, index == 0)
		}
	}
	if table.Fields[1].Name != "PossessionType" || table.Fields[1].Type != "PossessionType" || table.Fields[2].Name != "PossessionId" {
		t.Fatalf("shop content fields do not expose the shared reward editor pair: %+v", table.Fields)
	}

	var row Row
	for _, candidate := range table.Rows {
		if candidate.Titles["en"] != "" && len(candidate.ShopRelations) != 0 && len(candidate.ContentFootnotes) != 0 {
			row = candidate
			break
		}
	}
	if row.Values == nil {
		t.Fatal("no shop content row has a localized item name and shop relation")
	}
	count, err := strconv.ParseInt(row.Values["Count"], 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	candidate, _, err := BuildUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes:         []Change{{Table: table.Name, Row: row.Index, Field: "Count", Value: count + 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	rows, _, err := rebuilt.TableRows(table.Name)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := valueAsInt64(rows[row.Index][4]); err != nil || got != count+1 {
		t.Fatalf("Count = %d, %v; want %d", got, err, count+1)
	}
}

func TestShopEditorCatalogAndCompleteCellGroupReplacement(t *testing.T) {
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
	if len(catalog.ShopEditor.Shops) == 0 || len(catalog.ShopEditor.CellGroups) < 2 ||
		len(catalog.ShopEditor.Cells) == 0 || len(catalog.ShopEditor.Items) < 2 || len(catalog.ShopEditor.Stocks) == 0 {
		t.Fatalf("incomplete shop editor catalog: shops=%d groups=%d cells=%d items=%d",
			len(catalog.ShopEditor.Shops), len(catalog.ShopEditor.CellGroups),
			len(catalog.ShopEditor.Cells), len(catalog.ShopEditor.Items))
	}
	if catalog.ShopEditor.Cells[0].Row < 0 || catalog.ShopEditor.Items[0].Row < 0 {
		t.Fatal("shop editor rows must retain their physical table indexes")
	}

	groups := append([]ShopItemCellGroupInput(nil), catalog.ShopEditor.CellGroups[1:]...)
	cell := catalog.ShopEditor.Cells[0]
	priceItem := catalog.ShopEditor.Items[0]
	updatedPrice := priceItem.Price + 1
	updatedStockID := int64(0)
	if priceItem.ShopItemLimitedStockID == 0 {
		updatedStockID = catalog.ShopEditor.Stocks[0].ShopItemLimitedStockID
	}
	replacementItem := catalog.ShopEditor.Items[0].ShopItemID
	if replacementItem == cell.ShopItemID {
		replacementItem = catalog.ShopEditor.Items[1].ShopItemID
	}
	request := UpdateRequest{
		ExpectedVersion:    catalog.Version,
		ShopItemCellGroups: &groups,
		Changes: []Change{
			{Table: "m_shop_item_cell", Row: int(cell.Row), Field: "ShopItemId", Value: replacementItem},
			{Table: "m_shop_item", Row: int(priceItem.Row), Field: "Price", Value: updatedPrice},
			{Table: "m_shop_item", Row: int(priceItem.Row), Field: "ShopItemLimitedStockId", Value: updatedStockID},
		},
	}
	preview, err := PreviewUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.TableReplacements) != 1 || preview.TableReplacements[0].Table != shopItemCellGroupTable ||
		preview.TableReplacements[0].AfterRows != len(groups) {
		t.Fatalf("unexpected replacement preview: %+v", preview.TableReplacements)
	}

	candidate, result, err := BuildUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.ChangedCells < 3 || result.ChangedRows < 3 {
		t.Fatalf("unexpected update result: %+v", result)
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	groupRows, _, err := rebuilt.TableRows(shopItemCellGroupTable)
	if err != nil {
		t.Fatal(err)
	}
	if len(groupRows) != len(groups) {
		t.Fatalf("cell group row count = %d, want %d", len(groupRows), len(groups))
	}
	cellRows, _, err := rebuilt.TableRows("m_shop_item_cell")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := valueAsInt64(cellRows[cell.Row][2]); err != nil || got != replacementItem {
		t.Fatalf("ShopItemId = %d, %v; want %d", got, err, replacementItem)
	}
	itemRows, _, err := rebuilt.TableRows("m_shop_item")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := valueAsInt64(itemRows[priceItem.Row][6]); err != nil || got != updatedPrice {
		t.Fatalf("Price = %d, %v; want %d", got, err, updatedPrice)
	}
	if got, err := valueAsInt64(itemRows[priceItem.Row][9]); err != nil || got != updatedStockID {
		t.Fatalf("ShopItemLimitedStockId = %d, %v; want %d", got, err, updatedStockID)
	}
}

func TestShopItemCopyAndRestrictedDelete(t *testing.T) {
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
	if len(catalog.ShopEditor.Items) == 0 || len(catalog.ShopEditor.Cells) == 0 {
		t.Fatal("shop editor catalog is empty")
	}
	originalFile, err := memorydb.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	contentRows, _, err := originalFile.TableRows(shopItemContentPossessionTable)
	if err != nil {
		t.Fatal(err)
	}
	contentsByItemID := make(map[int64][]ShopItemContentPossessionInput)
	for _, row := range contentRows {
		content, ok := shopItemContentPossessionInputAt(row)
		if !ok {
			t.Fatal("malformed ShopItem Possession row")
		}
		contentsByItemID[int64(content.ShopItemID)] = append(contentsByItemID[int64(content.ShopItemID)], content)
	}
	var source, emptySource ShopEditorItem
	for _, item := range catalog.ShopEditor.Items {
		if source.ShopItemID == 0 && len(contentsByItemID[item.ShopItemID]) != 0 {
			source = item
		}
		if emptySource.ShopItemID == 0 && len(contentsByItemID[item.ShopItemID]) == 0 {
			emptySource = item
		}
	}
	if source.ShopItemID == 0 || emptySource.ShopItemID == 0 {
		t.Fatal("need ShopItems with and without Possession content")
	}
	newID := int64(1)
	for _, item := range catalog.ShopEditor.Items {
		if item.ShopItemID >= newID {
			newID = item.ShopItemID + 1
		}
	}
	if newID > math.MaxInt32 {
		t.Fatal("cannot allocate a test ShopItemId")
	}
	itemCopy := func(source ShopEditorItem, itemID int64) ShopItemInput {
		return ShopItemInput{
			ShopItemID: int32(itemID), NameShopTextID: int32(source.NameShopTextID),
			DescriptionShopTextID: int32(source.DescriptionShopTextID), ShopItemContentType: int32(source.ShopItemContentType),
			PriceType: int32(source.PriceType), PriceID: int32(source.PriceID), Price: int32(source.Price),
			RegularPrice: int32(source.RegularPrice), ShopPromotionType: int32(source.ShopPromotionType),
			ShopItemLimitedStockID: int32(source.ShopItemLimitedStockID), AssetCategoryID: int32(source.AssetCategoryID),
			AssetVariationID: int32(source.AssetVariationID), ShopItemDecorationType: int32(source.ShopItemDecorationType),
		}
	}
	copied := itemCopy(source, newID)
	copiedPossessions := append([]ShopItemContentPossessionInput(nil), contentsByItemID[source.ShopItemID]...)
	for index := range copiedPossessions {
		copiedPossessions[index].ShopItemID = int32(newID)
	}
	copiedPossessions[0].Count++
	request := UpdateRequest{
		ExpectedVersion: catalog.Version,
		ShopItems: &ShopItemStructuralUpdate{Copies: []ShopItemCopyInput{{
			SourceShopItemID: int32(source.ShopItemID), ShopItemInput: copied, Possessions: copiedPossessions,
		}}},
	}
	preview, err := PreviewUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.TableReplacements) != 2 || preview.TableReplacements[0].Table != shopItemTable ||
		preview.TableReplacements[1].Table != shopItemContentPossessionTable ||
		preview.TableReplacements[0].BeforeRows+1 != preview.TableReplacements[0].AfterRows {
		t.Fatalf("unexpected ShopItem replacement preview: %+v", preview.TableReplacements)
	}
	candidate, result, err := BuildUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	wantChangedCells := 13 + len(copiedPossessions)*5
	wantChangedRows := 1 + len(copiedPossessions)
	if result.ChangedCells != wantChangedCells || result.ChangedRows != wantChangedRows {
		t.Fatalf("copy result = %+v, want %d cells and %d rows", result, wantChangedCells, wantChangedRows)
	}
	candidateFile, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	rows, _, err := candidateFile.TableRows(shopItemTable)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(catalog.ShopEditor.Items)+1 {
		t.Fatalf("copied row count = %d, want %d", len(rows), len(catalog.ShopEditor.Items)+1)
	}
	last, ok := shopItemInputAt(rows[len(rows)-1])
	if !ok || last != copied {
		t.Fatalf("copied row = %+v, %v; want %+v", last, ok, copied)
	}
	candidateContentRows, _, err := candidateFile.TableRows(shopItemContentPossessionTable)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidateContentRows) != len(contentRows)+len(copiedPossessions) {
		t.Fatalf("copied Possession row count = %d, want %d", len(candidateContentRows), len(contentRows)+len(copiedPossessions))
	}
	for index, want := range copiedPossessions {
		got, parsed := shopItemContentPossessionInputAt(candidateContentRows[len(contentRows)+index])
		if !parsed || got != want {
			t.Fatalf("copied Possession %d = %+v, %v; want %+v", index, got, parsed, want)
		}
	}

	blockedID := catalog.ShopEditor.Cells[0].ShopItemID
	_, _, err = BuildUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		ShopItems:       &ShopItemStructuralUpdate{DeleteIDs: []int32{int32(blockedID)}},
	})
	if err == nil || !strings.Contains(err.Error(), "still referenced") {
		t.Fatalf("referenced delete error = %v, want reference rejection", err)
	}
	forged := copied
	forged.ShopItemID++
	forged.NameShopTextID++
	_, _, err = BuildUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		ShopItems: &ShopItemStructuralUpdate{Copies: []ShopItemCopyInput{{
			SourceShopItemID: int32(source.ShopItemID), ShopItemInput: forged, Possessions: copiedPossessions,
		}}},
	})
	if err == nil || !strings.Contains(err.Error(), "outside the ShopItem editor") {
		t.Fatalf("non-copy add error = %v, want restricted-field rejection", err)
	}

	_, _, err = buildUpdate(candidateFile, UpdateRequest{
		ExpectedVersion: candidateFile.Version(), ShopItems: &ShopItemStructuralUpdate{DeleteIDs: []int32{int32(newID)}},
	})
	if err == nil || !strings.Contains(err.Error(), shopItemContentPossessionTable) {
		t.Fatalf("copied content delete error = %v, want Possession reference rejection", err)
	}

	emptyID := newID + 1
	emptyCopy := itemCopy(emptySource, emptyID)
	emptyCandidate, _, err := BuildUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		ShopItems: &ShopItemStructuralUpdate{Copies: []ShopItemCopyInput{{
			SourceShopItemID: int32(emptySource.ShopItemID), ShopItemInput: emptyCopy, Possessions: []ShopItemContentPossessionInput{},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	emptyCandidateFile, err := memorydb.OpenBytes(emptyCandidate)
	if err != nil {
		t.Fatal(err)
	}
	deleted, deleteResult, err := buildUpdate(emptyCandidateFile, UpdateRequest{
		ExpectedVersion: emptyCandidateFile.Version(),
		ShopItems:       &ShopItemStructuralUpdate{DeleteIDs: []int32{int32(emptyID)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if deleteResult.ChangedCells != 13 || deleteResult.ChangedRows != 1 {
		t.Fatalf("delete result = %+v, want 13 cells and 1 row", deleteResult)
	}
	deletedFile, err := memorydb.OpenBytes(deleted)
	if err != nil {
		t.Fatal(err)
	}
	deletedRows, _, err := deletedFile.TableRows(shopItemTable)
	if err != nil {
		t.Fatal(err)
	}
	if len(deletedRows) != len(catalog.ShopEditor.Items) {
		t.Fatalf("deleted row count = %d, want %d", len(deletedRows), len(catalog.ShopEditor.Items))
	}
}

func TestShopItemCellAdditionAndRestrictedDelete(t *testing.T) {
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
	if len(catalog.ShopEditor.Cells) == 0 || len(catalog.ShopEditor.Items) == 0 || len(catalog.ShopEditor.CellGroups) == 0 {
		t.Fatal("shop editor catalog is incomplete")
	}
	newCellID := int64(1)
	for _, cell := range catalog.ShopEditor.Cells {
		if cell.ShopItemCellID >= newCellID {
			newCellID = cell.ShopItemCellID + 1
		}
	}
	if newCellID > math.MaxInt32 {
		t.Fatal("cannot allocate a test CellId")
	}
	added := ShopItemCellInput{
		ShopItemCellID: int32(newCellID), StepNumber: 1,
		ShopItemID: int32(catalog.ShopEditor.Items[0].ShopItemID),
	}
	request := UpdateRequest{
		ExpectedVersion: catalog.Version,
		ShopItemCells:   &ShopItemCellStructuralUpdate{Additions: []ShopItemCellInput{added}},
	}
	preview, err := PreviewUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.TableReplacements) != 1 || preview.TableReplacements[0].Table != shopItemCellTable ||
		preview.TableReplacements[0].BeforeRows+1 != preview.TableReplacements[0].AfterRows {
		t.Fatalf("unexpected Cell replacement preview: %+v", preview.TableReplacements)
	}
	candidate, result, err := BuildUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.ChangedCells != 3 || result.ChangedRows != 1 {
		t.Fatalf("Cell addition result = %+v, want 3 cells and 1 row", result)
	}
	candidateFile, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	rows, _, err := candidateFile.TableRows(shopItemCellTable)
	if err != nil {
		t.Fatal(err)
	}
	last, ok := shopItemCellInputAt(rows[len(rows)-1])
	if !ok || last != added {
		t.Fatalf("added Cell = %+v, %v; want %+v", last, ok, added)
	}

	referencedCellID := int64(catalog.ShopEditor.CellGroups[0].ShopItemCellID)
	var referencedCell ShopEditorCell
	for _, cell := range catalog.ShopEditor.Cells {
		if cell.ShopItemCellID == referencedCellID {
			referencedCell = cell
			break
		}
	}
	if referencedCell.ShopItemCellID == 0 || len(referencedCell.DeleteBlockers) == 0 {
		t.Fatal("CellGroup-referenced Cell must expose delete blockers")
	}
	_, _, err = BuildUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		ShopItemCells: &ShopItemCellStructuralUpdate{Deletes: []ShopItemCellKey{{
			ShopItemCellID: int32(referencedCell.ShopItemCellID), StepNumber: int32(referencedCell.StepNumber),
		}}},
	})
	if err == nil || !strings.Contains(err.Error(), "still referenced") {
		t.Fatalf("referenced Cell delete error = %v, want reference rejection", err)
	}

	deleted, deleteResult, err := buildUpdate(candidateFile, UpdateRequest{
		ExpectedVersion: candidateFile.Version(),
		ShopItemCells: &ShopItemCellStructuralUpdate{Deletes: []ShopItemCellKey{{
			ShopItemCellID: added.ShopItemCellID, StepNumber: added.StepNumber,
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if deleteResult.ChangedCells != 3 || deleteResult.ChangedRows != 1 {
		t.Fatalf("Cell delete result = %+v, want 3 cells and 1 row", deleteResult)
	}
	deletedFile, err := memorydb.OpenBytes(deleted)
	if err != nil {
		t.Fatal(err)
	}
	deletedRows, _, err := deletedFile.TableRows(shopItemCellTable)
	if err != nil {
		t.Fatal(err)
	}
	if len(deletedRows) != len(catalog.ShopEditor.Cells) {
		t.Fatalf("Cell row count after delete = %d, want %d", len(deletedRows), len(catalog.ShopEditor.Cells))
	}
}

func TestMissionRewardAssignmentCanBeUpdatedWithoutExposingMissionTable(t *testing.T) {
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
	if len(catalog.MissionSources.Missions) == 0 {
		t.Fatal("mission source catalog is empty")
	}
	for _, table := range catalog.Tables {
		if table.Name == "m_mission" {
			t.Fatal("m_mission must remain hidden from the general-purpose table catalog")
		}
	}

	source := catalog.MissionSources.Missions[0]
	var replacement int64
	for _, table := range catalog.Tables {
		if table.Name != "m_mission_reward" {
			continue
		}
		for _, row := range table.Rows {
			candidate, parseErr := strconv.ParseInt(row.Values["MissionRewardId"], 10, 64)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			if candidate != source.MissionRewardID {
				replacement = candidate
				break
			}
		}
	}
	if replacement == 0 {
		t.Fatal("no alternate mission reward id found")
	}
	request := UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes: []Change{{
			Table: "m_mission", Row: source.Row, Field: "MissionRewardId", Value: replacement,
		}},
	}
	preview, err := PreviewUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.OtherChanges) != 1 || len(preview.OtherChanges[0].Changes) != 1 {
		t.Fatalf("unexpected assignment preview: %+v", preview)
	}
	change := preview.OtherChanges[0].Changes[0]
	if change.Before != strconv.FormatInt(source.MissionRewardID, 10) || change.After != strconv.FormatInt(replacement, 10) {
		t.Fatalf("assignment preview = %+v", change)
	}

	candidate, result, err := BuildUpdate(path, request)
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
	rows, _, err := rebuilt.TableRows("m_mission")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := valueAsInt64(rows[source.Row][11]); err != nil || got != replacement {
		t.Fatalf("MissionRewardId = %d, %v; want %d", got, err, replacement)
	}
}

func TestMissionTermAssignmentCanBeUpdatedWithoutExposingMissionTable(t *testing.T) {
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
	if len(catalog.MissionSources.Missions) == 0 {
		t.Fatal("mission source catalog is empty")
	}
	for _, table := range catalog.Tables {
		if table.Name == "m_mission" {
			t.Fatal("m_mission must remain hidden from the general-purpose table catalog")
		}
	}

	source := catalog.MissionSources.Missions[0]
	var replacement int64
	for _, table := range catalog.Tables {
		if table.Name != "m_mission_term" {
			continue
		}
		for _, row := range table.Rows {
			candidate, parseErr := strconv.ParseInt(row.Values["MissionTermId"], 10, 64)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			if candidate != source.MissionTermID {
				replacement = candidate
				break
			}
		}
	}
	if replacement == 0 {
		t.Fatal("no alternate mission term id found")
	}
	request := UpdateRequest{
		ExpectedVersion: catalog.Version,
		Changes: []Change{{
			Table: "m_mission", Row: source.Row, Field: "MissionTermId", Value: replacement,
		}},
	}
	preview, err := PreviewUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.OtherChanges) != 1 || len(preview.OtherChanges[0].Changes) != 1 {
		t.Fatalf("unexpected assignment preview: %+v", preview)
	}
	change := preview.OtherChanges[0].Changes[0]
	if change.Before != strconv.FormatInt(source.MissionTermID, 10) || change.After != strconv.FormatInt(replacement, 10) {
		t.Fatalf("assignment preview = %+v", change)
	}

	candidate, result, err := BuildUpdate(path, request)
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
	rows, _, err := rebuilt.TableRows("m_mission")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := valueAsInt64(rows[source.Row][12]); err != nil || got != replacement {
		t.Fatalf("MissionTermId = %d, %v; want %d", got, err, replacement)
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
