package masterdataadmin

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/questdrop"
)

func TestActivitySpecsContainSelectedAndRelatedTables(t *testing.T) {
	if got, want := len(activityTableSpecs), 46; got != want {
		t.Fatalf("activity spec count = %d, want %d", got, want)
	}
	wantPrimary := map[string]bool{
		"m_beginner_campaign": true, "m_big_hunt_schedule": true,
		"m_big_hunt_score_reward_group_schedule":                  true,
		"m_big_hunt_weekly_attribute_score_reward_group_schedule": true,
		"m_comeback_campaign":                                     true, "m_consumable_item_term": true,
		"m_dokan": true, "m_enhance_campaign": true, "m_event_quest_chapter": true,
		"m_event_quest_daily_group": true, "m_event_quest_labyrinth_season": true,
		"m_login_bonus": true, "m_maintenance": true, "m_mission_term": true, "m_mom_banner": true,
		"m_navi_cut_in": true,
		"m_omikuji":     true, "m_pvp_season": true, "m_quest_campaign": true,
		"m_shop": true, "m_shop_item_cell_term": true, "m_tip": true,
	}
	wantRelated := map[string]bool{
		"m_big_hunt_boss_quest":           true,
		"m_big_hunt_score_reward_group":   true,
		"m_enhance_campaign_target_group": true,
		"m_event_quest_link":              true, "m_event_quest_display_item_group": true,
		"m_event_quest_sequence_group":                true,
		"m_event_quest_daily_group_target_chapter":    true,
		"m_event_quest_daily_group_complete_reward":   true,
		"m_event_quest_daily_group_message":           true,
		"m_event_quest_labyrinth_season_reward_group": true,
		"m_gacha_medal":       true,
		"m_maintenance_group": true, "m_pvp_season_grouping": true,
		"m_pvp_weekly_rank_reward_rank_group": true,
		"m_pvp_season_rank_reward_rank_group": true,
		"m_pvp_grade_group":                   true, "m_quest_campaign_target_group": true,
		"m_quest_campaign_effect_group": true, "m_shop_item_cell_group": true,
	}
	wantDelivery := map[string]bool{
		"m_big_hunt_reward_group": true,
		"m_login_bonus_stamp":     true, "m_mission_reward": true,
		"m_shop_item_content_possession": true, "m_quest_pickup_reward_group": true,
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
	if catalog.PrimaryCount != 22 || catalog.RelatedCount != 19 || catalog.DeliveryCount != 5 {
		t.Fatalf("loaded primary/related/delivery counts = %d/%d/%d, want 22/19/5", catalog.PrimaryCount, catalog.RelatedCount, catalog.DeliveryCount)
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

func TestLoadMetadataAndTableOnDemand(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master-data asset is not installed")
	} else if err != nil {
		t.Fatal(err)
	}

	metadata, err := LoadMetadata(path)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.TableCount == 0 || metadata.RowCount == 0 {
		t.Fatalf("metadata counts = %d tables, %d rows", metadata.TableCount, metadata.RowCount)
	}
	var selected Table
	for _, table := range metadata.Tables {
		if table.Rows != nil {
			t.Fatalf("metadata table %q unexpectedly includes row entries", table.Name)
		}
		if selected.Name == "" && table.RowCount > 0 {
			selected = table
		}
	}
	if selected.Name == "" {
		t.Fatal("metadata has no non-empty table")
	}

	loaded, err := LoadTable(path, selected.Name)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != metadata.Version {
		t.Fatalf("single-table version = %q, want %q", loaded.Version, metadata.Version)
	}
	if len(loaded.Tables) == 0 || loaded.Tables[0].Name != selected.Name {
		t.Fatalf("single-table response starts with %+v, want %q", loaded.Tables, selected.Name)
	}
	if len(loaded.Tables[0].Rows) != selected.RowCount {
		t.Fatalf("loaded %d rows, want %d", len(loaded.Tables[0].Rows), selected.RowCount)
	}
	if _, err := LoadTable(path, "m_not_an_admin_table"); err == nil {
		t.Fatal("unknown table unexpectedly loaded")
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

func TestQuestDropEditorCatalogSeparatesPickupPreviewsAndAcquisitionRoutes(t *testing.T) {
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
	table := findCatalogTable(catalog, questPickupRewardGroupTable)
	if !table.Delivery || len(table.Rows) == 0 {
		t.Fatalf("unexpected quest drop table: delivery=%v rows=%d", table.Delivery, len(table.Rows))
	}
	editor := catalog.QuestDropEditor
	if len(editor.Types) < 2 || len(editor.Chapters) == 0 || len(editor.Quests) == 0 || len(editor.Rewards) == 0 {
		t.Fatalf("incomplete quest drop catalog: types=%d chapters=%d quests=%d rewards=%d",
			len(editor.Types), len(editor.Chapters), len(editor.Quests), len(editor.Rewards))
	}
	expectedTypes := map[string]QuestDropType{
		"main":    {ID: "main", Value: 1, Label: "MAIN_QUEST"},
		"event-3": {ID: "event-3", Value: 3, Label: "DUNGEON"},
		"event-4": {ID: "event-4", Value: 4, Label: "DAY_OF_THE_WEEK"},
		"event-5": {ID: "event-5", Value: 5, Label: "GUERRILLA"},
		"event-6": {ID: "event-6", Value: 6, Label: "CHARACTER"},
		"event-7": {ID: "event-7", Value: 7, Label: "CHARACTER_QUEST"},
		"event-8": {ID: "event-8", Value: 8, Label: "CAGE"},
	}
	if len(editor.Types) != len(expectedTypes) {
		t.Fatalf("quest drop types=%d, want %d", len(editor.Types), len(expectedTypes))
	}
	for _, definition := range editor.Types {
		if expected, exists := expectedTypes[definition.ID]; !exists || definition != expected {
			t.Fatalf("unexpected quest drop type: %+v", definition)
		}
	}
	for index := 1; index < len(editor.Types); index++ {
		if editor.Types[index-1].Value >= editor.Types[index].Value {
			t.Fatalf("quest drop types are not ordered by ID: %d before %d",
				editor.Types[index-1].Value, editor.Types[index].Value)
		}
	}
	for index := 1; index < len(editor.Quests); index++ {
		if editor.Quests[index-1].QuestID > editor.Quests[index].QuestID {
			t.Fatalf("quest drop rows are not ordered by ID: %d before %d",
				editor.Quests[index-1].QuestID, editor.Quests[index].QuestID)
		}
	}
	chapters := make(map[string]bool, len(editor.Chapters))
	var mainChapters []QuestDropChapter
	for _, chapter := range editor.Chapters {
		if chapter.TypeID == "event-9" {
			t.Fatalf("activity chapter %d is configurable", chapter.ChapterID)
		}
		if chapter.TypeID == "main" {
			mainChapters = append(mainChapters, chapter)
		}
		chapters[chapter.TypeID+"/"+strconv.FormatInt(int64(chapter.ChapterID), 10)] = true
	}
	if len(mainChapters) != 30 {
		t.Fatalf("main story filter chapters=%d, want 30", len(mainChapters))
	}
	for index, chapter := range mainChapters {
		if chapter.ChapterID != int32(index+1) {
			t.Fatalf("main story filter chapter %d has ID %d", index, chapter.ChapterID)
		}
	}
	wantMainChapterNames := map[int32]string{
		1: "一章：風砂の章", 7: "第一夜：紅枯の章", 12: "第六夜：白秋の章",
		13: "陽ノ壱：朝暉の章", 14: "月ノ壱：宵闇の章", 25: "序ノ壱：青藍の章", 30: "三ノ幕：輪廻の章",
	}
	for chapterID, want := range wantMainChapterNames {
		if got := mainChapters[chapterID-1].Names["ja"]; got != want {
			t.Fatalf("main story chapter %d Japanese name=%q, want %q", chapterID, got, want)
		}
	}
	chapterByKey := make(map[string]QuestDropChapter, len(editor.Chapters))
	for _, chapter := range editor.Chapters {
		chapterByKey[chapter.TypeID+"/"+strconv.FormatInt(int64(chapter.ChapterID), 10)] = chapter
	}
	if chapters["event-4/10"] {
		t.Fatal("duplicate aggregate weekday chapter 10 is configurable")
	}
	wantGuerrillaChapterNames := map[int32]string{1: "小型剣", 2: "槍", 3: "大剣", 4: "格闘", 5: "杖", 6: "銃"}
	for chapterID, want := range wantGuerrillaChapterNames {
		chapter, exists := chapterByKey["event-5/"+strconv.FormatInt(int64(chapterID), 10)]
		if !exists || chapter.Names["ja"] != want {
			t.Fatalf("guerrilla chapter %d = %+v, want Japanese name %q", chapterID, chapter, want)
		}
	}
	for key, want := range map[string]string{
		"event-6/901": "リオン", "event-7/99001": "リオン",
		"event-6/913": "サリュ", "event-7/99014": "サリュ",
		"event-6/914": "プリエ", "event-7/99015": "プリエ",
		"event-6/915": "マリー", "event-7/99016": "マリー",
		"event-6/916": "ユリィ", "event-7/99017": "ユリィ",
		"event-6/917": "ユディル", "event-7/99018": "ユディル",
		"event-6/918": "サラーファ", "event-7/99019": "サラーファ",
		"event-6/919": "明城陽那", "event-7/99020": "明城陽那",
		"event-6/920": "暮染佑月", "event-7/99021": "暮染佑月",
		"event-6/921": "10H", "event-7/99022": "10H",
	} {
		if got := chapterByKey[key].Names["ja"]; got != want {
			t.Fatalf("character chapter %s Japanese name=%q, want %q", key, got, want)
		}
	}
	file, err := memorydb.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	type expectedPreviewReward struct {
		sortOrder int64
		rowIndex  int
		rewardID  int32
	}
	expectedPreviews := make(map[int32][]expectedPreviewReward)
	for rowIndex, row := range readRows(file, questPickupRewardGroupTable) {
		groupID, groupOK := integerAt(row, 0)
		sortOrder, sortOK := integerAt(row, 1)
		rewardID, rewardOK := integerAt(row, 2)
		if groupOK && sortOK && rewardOK {
			expectedPreviews[int32(groupID)] = append(expectedPreviews[int32(groupID)], expectedPreviewReward{
				sortOrder: sortOrder, rowIndex: rowIndex, rewardID: int32(rewardID),
			})
		}
	}
	for groupID := range expectedPreviews {
		sort.SliceStable(expectedPreviews[groupID], func(i, j int) bool {
			left, right := expectedPreviews[groupID][i], expectedPreviews[groupID][j]
			if left.sortOrder != right.sortOrder {
				return left.sortOrder < right.sortOrder
			}
			return left.rowIndex < right.rowIndex
		})
	}
	expectedRewardIDs := make(map[int32]bool)
	for _, row := range readRows(file, battleDropRewardTable) {
		if reward, ok := questDropRewardAt(row); ok {
			expectedRewardIDs[reward.BattleDropRewardID] = true
		}
	}
	rewards := make(map[int32]QuestDropReward, len(editor.Rewards))
	for index, reward := range editor.Rewards {
		if index > 0 && editor.Rewards[index-1].BattleDropRewardID >= reward.BattleDropRewardID {
			t.Fatalf("selectable rewards are not ordered by ID: %d before %d",
				editor.Rewards[index-1].BattleDropRewardID, reward.BattleDropRewardID)
		}
		rewards[reward.BattleDropRewardID] = reward
	}
	if len(rewards) != len(expectedRewardIDs) {
		t.Fatalf("selectable rewards=%d, want every BattleDropRewardId (%d)", len(rewards), len(expectedRewardIDs))
	}
	for rewardID := range expectedRewardIDs {
		if _, exists := rewards[rewardID]; !exists {
			t.Fatalf("BattleDropRewardId %d was omitted from the searchable selector", rewardID)
		}
	}
	groups := make(map[int32]QuestDropGroup, len(editor.Groups))
	for _, group := range editor.Groups {
		groups[group.QuestPickupRewardGroupID] = group
		expectedPreview := expectedPreviews[group.QuestPickupRewardGroupID]
		if len(group.PreviewRewardIDs) != len(expectedPreview) {
			t.Fatalf("group %d preview rewards=%d, want %d raw pickup rows",
				group.QuestPickupRewardGroupID, len(group.PreviewRewardIDs), len(expectedPreview))
		}
		for index, rewardID := range group.PreviewRewardIDs {
			if rewardID != expectedPreview[index].rewardID {
				t.Fatalf("group %d preview reward %d=%d, want %d",
					group.QuestPickupRewardGroupID, index, rewardID, expectedPreview[index].rewardID)
			}
		}
		seen := make(map[int32]bool, len(group.Rewards))
		for _, configuredReward := range group.Rewards {
			if _, exists := rewards[configuredReward.BattleDropRewardID]; !exists {
				t.Fatalf("group %d references omitted reward %d", group.QuestPickupRewardGroupID, configuredReward.BattleDropRewardID)
			}
			if seen[configuredReward.BattleDropRewardID] {
				t.Fatalf("group %d contains duplicate reward %d", group.QuestPickupRewardGroupID, configuredReward.BattleDropRewardID)
			}
			if configuredReward.Weight < 1 {
				t.Fatalf("group %d reward %d has invalid weight %d", group.QuestPickupRewardGroupID, configuredReward.BattleDropRewardID, configuredReward.Weight)
			}
			seen[configuredReward.BattleDropRewardID] = true
		}
	}
	foundRoutePossession := false
	foundDungeonQuest := false
	mainChapterByQuestID := make(map[int32]int32)
	weekdayStages := make(map[int32]map[int32]string)
	guerrillaStages := make(map[int32]map[int32]string)
	characterQuestStages := make(map[int32]map[int32]map[int32]string)
	wantStageNames := map[int32]string{1: "初級", 2: "中級", 3: "上級", 4: "超級"}
	wantCharacterQuestStageNames := map[int32]string{1: "初級", 2: "中級", 3: "上級"}
	for _, quest := range editor.Quests {
		if quest.TypeID == "event-1" || quest.TypeID == "event-2" || quest.TypeID == "event-9" ||
			quest.TypeID == "event-10" || quest.TypeID == "event-11" || quest.TypeID == "event-12" {
			t.Fatalf("event quest %d of excluded type %s is configurable", quest.QuestID, quest.TypeID)
		}
		if !chapters[quest.TypeID+"/"+strconv.FormatInt(int64(quest.ChapterID), 10)] {
			t.Fatalf("quest %d references an omitted chapter", quest.QuestID)
		}
		if len(quest.RoutePossessions) > 0 {
			foundRoutePossession = true
		}
		if quest.TypeID == "event-3" {
			foundDungeonQuest = true
		} else if quest.TypeID == "main" {
			mainChapterByQuestID[quest.QuestID] = quest.ChapterID
		} else if quest.TypeID == "event-4" && quest.QuestID < 400000 {
			if weekdayStages[quest.ChapterID] == nil {
				weekdayStages[quest.ChapterID] = make(map[int32]string)
			}
			weekdayStages[quest.ChapterID][quest.SortOrder] = quest.Names["ja"]
		} else if quest.TypeID == "event-5" {
			if guerrillaStages[quest.ChapterID] == nil {
				guerrillaStages[quest.ChapterID] = make(map[int32]string)
			}
			guerrillaStages[quest.ChapterID][quest.SortOrder] = quest.Names["ja"]
		} else if quest.TypeID == "event-7" {
			if characterQuestStages[quest.ChapterID] == nil {
				characterQuestStages[quest.ChapterID] = make(map[int32]map[int32]string)
			}
			if characterQuestStages[quest.ChapterID][quest.SubcategoryType] == nil {
				characterQuestStages[quest.ChapterID][quest.SubcategoryType] = make(map[int32]string)
			}
			characterQuestStages[quest.ChapterID][quest.SubcategoryType][quest.SortOrder] = quest.Names["ja"]
		}
	}
	if !foundDungeonQuest {
		t.Fatal("no Dungeon quest is configurable")
	}
	if _, exists := mainChapterByQuestID[1]; exists {
		t.Fatal("main-flow quest 1 is configurable")
	}
	for _, chapterID := range []int32{1, 2, 3, 4, 5, 6, 8} {
		if len(weekdayStages[chapterID]) != 4 {
			t.Fatalf("weekday chapter %d regular stages=%v, want four", chapterID, weekdayStages[chapterID])
		}
		for stage, want := range wantStageNames {
			if got := weekdayStages[chapterID][stage]; got != want {
				t.Fatalf("weekday chapter %d stage %d name=%q, want %q", chapterID, stage, got, want)
			}
		}
	}
	for chapterID := int32(1); chapterID <= 6; chapterID++ {
		if len(guerrillaStages[chapterID]) != 4 {
			t.Fatalf("guerrilla chapter %d stages=%v, want four", chapterID, guerrillaStages[chapterID])
		}
		for stage, want := range wantStageNames {
			if got := guerrillaStages[chapterID][stage]; got != want {
				t.Fatalf("guerrilla chapter %d stage %d name=%q, want %q", chapterID, stage, got, want)
			}
		}
	}
	if len(characterQuestStages) != 21 {
		t.Fatalf("character quest chapters=%d, want 21", len(characterQuestStages))
	}
	for chapterID, categories := range characterQuestStages {
		if len(categories) != 3 {
			t.Fatalf("character quest chapter %d categories=%v, want three", chapterID, categories)
		}
		for category := int32(1); category <= 3; category++ {
			if len(categories[category]) != 3 {
				t.Fatalf("character quest chapter %d category %d stages=%v, want three", chapterID, category, categories[category])
			}
			for stage, want := range wantCharacterQuestStageNames {
				if got := categories[category][stage]; got != want {
					t.Fatalf("character quest chapter %d category %d stage %d name=%q, want %q", chapterID, category, stage, got, want)
				}
			}
		}
	}
	wantMainChapterByQuestID := map[int32]int32{
		2: 1, 62: 7, 305: 13, 405: 14, 341: 19, 441: 20, 501: 25, 547: 30,
	}
	for questID, want := range wantMainChapterByQuestID {
		if got := mainChapterByQuestID[questID]; got != want {
			t.Fatalf("main quest %d filter chapter=%d, want %d", questID, got, want)
		}
	}
	if !foundRoutePossession {
		t.Fatal("no acquisition-route possessions are available for preview")
	}
}

func TestQuestDropConfigScopeOnlyAllowsCatalogQuests(t *testing.T) {
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
	questID := catalog.QuestDropEditor.Quests[0].QuestID
	valid := &questdrop.Config{Version: questdrop.ConfigVersion, Quests: map[int32]questdrop.QuestConfig{questID: {}}}
	if err := ValidateQuestDropConfigScope(catalog.QuestDropEditor, valid); err != nil {
		t.Fatalf("valid quest config scope: %v", err)
	}
	invalid := &questdrop.Config{Version: questdrop.ConfigVersion, Quests: map[int32]questdrop.QuestConfig{-1: {}}}
	if err := ValidateQuestDropConfigScope(catalog.QuestDropEditor, invalid); err == nil {
		t.Fatal("out-of-scope quest config was accepted")
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
	for left, right := 0, len(groups)-1; left < right; left, right = left+1, right-1 {
		groups[left], groups[right] = groups[right], groups[left]
	}
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
	assertRowsSortedByIntegerColumns(t, groupRows, 0, 1)
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
	for _, blocker := range source.DeleteBlockers {
		if blocker == shopItemContentPossessionTable {
			t.Fatalf("Possession content must not be reported as a ShopItem delete blocker: %+v", source.DeleteBlockers)
		}
	}
	usedItemIDs := make(map[int64]bool, len(catalog.ShopEditor.Items))
	maxItemID := int64(0)
	for _, item := range catalog.ShopEditor.Items {
		usedItemIDs[item.ShopItemID] = true
		if item.ShopItemID > maxItemID {
			maxItemID = item.ShopItemID
		}
	}
	newID := int64(1)
	for usedItemIDs[newID] {
		newID++
	}
	if newID >= maxItemID {
		t.Fatal("test master data has no ShopItemId gap below the current maximum")
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
	assertRowsSortedByIntegerColumns(t, rows, 0)
	foundCopy := false
	for _, row := range rows {
		item, ok := shopItemInputAt(row)
		if ok && item.ShopItemID == copied.ShopItemID {
			foundCopy = true
			if item != copied {
				t.Fatalf("copied row = %+v; want %+v", item, copied)
			}
			break
		}
	}
	if !foundCopy {
		t.Fatalf("copied ShopItemId %d is absent", copied.ShopItemID)
	}
	candidateContentRows, _, err := candidateFile.TableRows(shopItemContentPossessionTable)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidateContentRows) != len(contentRows)+len(copiedPossessions) {
		t.Fatalf("copied Possession row count = %d, want %d", len(candidateContentRows), len(contentRows)+len(copiedPossessions))
	}
	assertRowsSortedByIntegerColumns(t, candidateContentRows, 0, 3)
	gotCopiedPossessions := make([]ShopItemContentPossessionInput, 0, len(copiedPossessions))
	for _, row := range candidateContentRows {
		content, parsed := shopItemContentPossessionInputAt(row)
		if !parsed {
			t.Fatal("malformed copied Possession row")
		}
		if content.ShopItemID == copied.ShopItemID {
			gotCopiedPossessions = append(gotCopiedPossessions, content)
		}
	}
	if len(gotCopiedPossessions) != len(copiedPossessions) {
		t.Fatalf("copied Possession count = %d, want %d", len(gotCopiedPossessions), len(copiedPossessions))
	}
	for index, want := range copiedPossessions {
		if got := gotCopiedPossessions[index]; got != want {
			t.Fatalf("copied Possession %d = %+v; want %+v", index, got, want)
		}
	}

	blockedID := catalog.ShopEditor.Cells[0].ShopItemID
	var referencedItem ShopEditorItem
	for _, item := range catalog.ShopEditor.Items {
		if item.ShopItemID == blockedID {
			referencedItem = item
			break
		}
	}
	if referencedItem.ShopItemID == 0 || len(referencedItem.References) == 0 {
		t.Fatal("Cell-referenced ShopItem must expose reference records")
	}
	_, _, err = BuildUpdate(path, UpdateRequest{
		ExpectedVersion: catalog.Version,
		ShopItems:       &ShopItemStructuralUpdate{DeleteIDs: []int32{int32(blockedID)}},
	})
	if err == nil || !strings.Contains(err.Error(), "still referenced") {
		t.Fatalf("referenced delete error = %v, want reference rejection", err)
	}
	forged := copied
	forged.ShopItemID++
	for usedItemIDs[int64(forged.ShopItemID)] || forged.ShopItemID == copied.ShopItemID {
		forged.ShopItemID++
	}
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

	deletedWithPossessions, deleteWithPossessionsResult, err := buildUpdate(candidateFile, UpdateRequest{
		ExpectedVersion: candidateFile.Version(), ShopItems: &ShopItemStructuralUpdate{DeleteIDs: []int32{int32(newID)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantDeletedCells := 13 + len(copiedPossessions)*5
	wantDeletedRows := 1 + len(copiedPossessions)
	if deleteWithPossessionsResult.ChangedCells != wantDeletedCells || deleteWithPossessionsResult.ChangedRows != wantDeletedRows {
		t.Fatalf("delete with Possessions result = %+v, want %d cells and %d rows", deleteWithPossessionsResult, wantDeletedCells, wantDeletedRows)
	}
	deletedWithPossessionsFile, err := memorydb.OpenBytes(deletedWithPossessions)
	if err != nil {
		t.Fatal(err)
	}
	deletedWithPossessionItemRows, _, err := deletedWithPossessionsFile.TableRows(shopItemTable)
	if err != nil {
		t.Fatal(err)
	}
	if len(deletedWithPossessionItemRows) != len(catalog.ShopEditor.Items) {
		t.Fatalf("deleted ShopItem row count = %d, want %d", len(deletedWithPossessionItemRows), len(catalog.ShopEditor.Items))
	}
	deletedPossessionRows, _, err := deletedWithPossessionsFile.TableRows(shopItemContentPossessionTable)
	if err != nil {
		t.Fatal(err)
	}
	if len(deletedPossessionRows) != len(contentRows) {
		t.Fatalf("deleted Possession row count = %d, want %d", len(deletedPossessionRows), len(contentRows))
	}

	emptyID := newID + 1
	for usedItemIDs[emptyID] || emptyID == newID {
		emptyID++
	}
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
	usedCellIDs := make(map[int64]bool, len(catalog.ShopEditor.Cells))
	maxCellID := int64(0)
	for _, cell := range catalog.ShopEditor.Cells {
		usedCellIDs[cell.ShopItemCellID] = true
		if cell.ShopItemCellID > maxCellID {
			maxCellID = cell.ShopItemCellID
		}
	}
	newCellID := int64(1)
	for usedCellIDs[newCellID] {
		newCellID++
	}
	if newCellID >= maxCellID {
		t.Fatal("test master data has no CellId gap below the current maximum")
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
	assertRowsSortedByIntegerColumns(t, rows, 0, 1)
	foundAdded := false
	for _, row := range rows {
		cell, ok := shopItemCellInputAt(row)
		if ok && cell == added {
			foundAdded = true
			break
		}
	}
	if !foundAdded {
		t.Fatalf("added Cell %+v is absent", added)
	}

	referencedCellID := int64(catalog.ShopEditor.CellGroups[0].ShopItemCellID)
	var referencedCell ShopEditorCell
	for _, cell := range catalog.ShopEditor.Cells {
		if cell.ShopItemCellID == referencedCellID {
			referencedCell = cell
			break
		}
	}
	if referencedCell.ShopItemCellID == 0 || len(referencedCell.DeleteBlockers) == 0 || len(referencedCell.References) == 0 {
		t.Fatal("CellGroup-referenced Cell must expose delete blockers and reference records")
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

func assertRowsSortedByIntegerColumns(t *testing.T, rows [][]interface{}, columns ...int) {
	t.Helper()
	for rowIndex := 1; rowIndex < len(rows); rowIndex++ {
		for _, column := range columns {
			previous, err := valueAsInt64(rows[rowIndex-1][column])
			if err != nil {
				t.Fatalf("row %d column %d: %v", rowIndex-1, column, err)
			}
			current, err := valueAsInt64(rows[rowIndex][column])
			if err != nil {
				t.Fatalf("row %d column %d: %v", rowIndex, column, err)
			}
			if previous < current {
				break
			}
			if previous > current {
				t.Fatalf("rows are not sorted at indexes %d and %d by columns %v", rowIndex-1, rowIndex, columns)
			}
		}
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

func TestMissionRewardStructuralUpdateIsSortedAndReferenceSafe(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master-data asset is not installed")
	} else if err != nil {
		t.Fatal(err)
	}
	file, err := memorydb.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	currentRows, exists, err := file.TableRows(missionRewardTable)
	if err != nil || !exists {
		t.Fatalf("read %s: exists=%v err=%v", missionRewardTable, exists, err)
	}
	current := make([]MissionRewardInput, 0, len(currentRows))
	usedIDs := make(map[int32]bool, len(currentRows))
	var maximumID int32
	for _, row := range currentRows {
		reward, ok := missionRewardInputAt(row)
		if !ok {
			t.Fatalf("malformed %s row: %#v", missionRewardTable, row)
		}
		current = append(current, reward)
		usedIDs[reward.MissionRewardID] = true
		if reward.MissionRewardID > maximumID {
			maximumID = reward.MissionRewardID
		}
	}
	var newID int32
	for candidate := int32(1); candidate < maximumID; candidate++ {
		if !usedIDs[candidate] {
			newID = candidate
			break
		}
	}
	if newID == 0 {
		t.Fatal("no unused RewardId below the current maximum")
	}
	added := MissionRewardInput{
		MissionRewardID: newID, PossessionType: current[0].PossessionType,
		PossessionID: current[0].PossessionID, Count: 0,
	}
	withAddition := append(append([]MissionRewardInput(nil), current...), added)
	request := UpdateRequest{ExpectedVersion: file.Version(), MissionRewards: &withAddition}
	preview, err := PreviewUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.TableReplacements) != 1 || preview.TableReplacements[0].Table != missionRewardTable {
		t.Fatalf("unexpected table replacement preview: %+v", preview.TableReplacements)
	}
	candidate, result, err := BuildUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.ChangedCells != 4 || result.ChangedRows != 1 {
		t.Fatalf("addition result = %+v, want 4 cells in 1 row", result)
	}
	candidateFile, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	addedRows, _, err := candidateFile.TableRows(missionRewardTable)
	if err != nil {
		t.Fatal(err)
	}
	assertRowsSortedByIntegerColumns(t, addedRows, 0)
	found := false
	for _, row := range addedRows {
		reward, _ := missionRewardInputAt(row)
		found = found || reward.MissionRewardID == newID && reward.Count == 0
	}
	if !found {
		t.Fatalf("new RewardId %d with Count=0 was not retained", newID)
	}

	withoutAddition := append([]MissionRewardInput(nil), current...)
	_, deletionResult, err := buildUpdate(candidateFile, UpdateRequest{
		ExpectedVersion: candidateFile.Version(), MissionRewards: &withoutAddition,
	})
	if err != nil {
		t.Fatal(err)
	}
	if deletionResult.ChangedCells != 4 || deletionResult.ChangedRows != 1 {
		t.Fatalf("deletion result = %+v, want 4 cells in 1 row", deletionResult)
	}

	missionRows, exists, err := file.TableRows("m_mission")
	if err != nil || !exists {
		t.Fatalf("read m_mission: exists=%v err=%v", exists, err)
	}
	referenceCounts := make(map[int64]int)
	for _, row := range missionRows {
		rewardID, ok := integerAt(row, 11)
		if !ok {
			t.Fatalf("malformed m_mission row: %#v", row)
		}
		referenceCounts[rewardID]++
	}
	var singlyReferencedID int64
	var missionRow int
	for rowIndex, row := range missionRows {
		rewardID, _ := integerAt(row, 11)
		if referenceCounts[rewardID] == 1 {
			singlyReferencedID = rewardID
			missionRow = rowIndex
			break
		}
	}
	if singlyReferencedID == 0 {
		t.Fatal("no singly referenced RewardId found")
	}
	withoutReferenced := make([]MissionRewardInput, 0, len(current)-1)
	for _, reward := range current {
		if int64(reward.MissionRewardID) != singlyReferencedID {
			withoutReferenced = append(withoutReferenced, reward)
		}
	}
	blocked := UpdateRequest{ExpectedVersion: file.Version(), MissionRewards: &withoutReferenced}
	if _, _, err := BuildUpdate(path, blocked); err == nil || !strings.Contains(err.Error(), "still referenced") {
		t.Fatalf("referenced deletion error = %v, want still referenced", err)
	}
	var alternateID int32
	for _, reward := range current {
		if int64(reward.MissionRewardID) != singlyReferencedID {
			alternateID = reward.MissionRewardID
			break
		}
	}
	atomic := UpdateRequest{
		ExpectedVersion: file.Version(), MissionRewards: &withoutReferenced,
		Changes: []Change{{Table: "m_mission", Row: missionRow, Field: "MissionRewardId", Value: alternateID}},
	}
	if _, err := PreviewUpdate(path, atomic); err != nil {
		t.Fatalf("preview atomic reassignment and deletion: %v", err)
	}
	_, atomicResult, err := BuildUpdate(path, atomic)
	if err != nil {
		t.Fatal(err)
	}
	if atomicResult.ChangedRows != 2 || atomicResult.ChangedCells != 5 {
		t.Fatalf("atomic reassignment result = %+v, want 5 cells in 2 rows", atomicResult)
	}
}

func TestMissionRewardStructuralUpdateRejectsInvalidRows(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master-data asset is not installed")
	} else if err != nil {
		t.Fatal(err)
	}
	file, err := memorydb.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rows, _, err := file.TableRows(missionRewardTable)
	if err != nil {
		t.Fatal(err)
	}
	current := make([]MissionRewardInput, 0, len(rows))
	for _, row := range rows {
		reward, ok := missionRewardInputAt(row)
		if !ok {
			t.Fatalf("malformed %s row", missionRewardTable)
		}
		current = append(current, reward)
	}
	duplicate := append(append([]MissionRewardInput(nil), current...), current[0])
	if _, _, err := BuildUpdate(path, UpdateRequest{ExpectedVersion: file.Version(), MissionRewards: &duplicate}); err == nil || !strings.Contains(err.Error(), "duplicate RewardId") {
		t.Fatalf("duplicate RewardId error = %v", err)
	}
	negative := append([]MissionRewardInput(nil), current...)
	negative[0].Count = -1
	if _, _, err := BuildUpdate(path, UpdateRequest{ExpectedVersion: file.Version(), MissionRewards: &negative}); err == nil || !strings.Contains(err.Error(), "cannot be negative") {
		t.Fatalf("negative Count error = %v", err)
	}
	if _, _, err := BuildUpdate(path, UpdateRequest{
		ExpectedVersion: file.Version(),
		Changes:         []Change{{Table: missionRewardTable, Row: 0, Field: "Count", Value: -1}},
	}); err == nil || !strings.Contains(err.Error(), "cannot be negative") {
		t.Fatalf("negative scalar Count error = %v", err)
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
