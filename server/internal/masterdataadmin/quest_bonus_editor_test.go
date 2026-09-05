package masterdataadmin

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
)

func bonusTestFile(t *testing.T) (string, *memorydb.File) {
	t.Helper()
	path := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master-data asset is not installed")
	}
	file, err := memorydb.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return path, file
}

func bonusTestGroup(file *memorydb.File, table string, id int64) QuestBonusGroupInput {
	var rows [][]interface{}
	for _, row := range readRows(file, table) {
		if bonusInt(row, 0) == id {
			rows = append(rows, row)
		}
	}
	return QuestBonusGroupInput{Table: table, GroupID: id, Rows: bonusStringRows(rows)}
}

func TestQuestBonusCatalogFollowsEventChainAndKeepsExpiredMembers(t *testing.T) {
	path, _ := bonusTestFile(t)
	catalog, err := LoadTable(path, questBonusTable)
	if err != nil {
		t.Fatal(err)
	}
	editor := catalog.QuestBonusEditor
	if editor == nil || len(editor.Tables) != 8 || len(editor.Costumes) == 0 || len(editor.Weapons) == 0 {
		t.Fatalf("incomplete editor catalog")
	}
	for _, tc := range []struct {
		chapter int64
		bonuses []int64
	}{
		{551, []int64{201189, 201190, 201191}}, {573, []int64{201131, 201132, 201133}},
		{586, []int64{201192, 201193, 201194}}, {589, []int64{0}},
	} {
		count := 0
		seen := make(map[int64]bool)
		for _, quest := range editor.Quests {
			if quest.ChapterID == tc.chapter {
				count++
				seen[quest.BonusID] = true
			}
		}
		if count != 35 || len(seen) != len(tc.bonuses) {
			t.Fatalf("chapter %d: count=%d bonuses=%v", tc.chapter, count, seen)
		}
		for _, id := range tc.bonuses {
			if !seen[id] {
				t.Fatalf("chapter %d missing bonus %d", tc.chapter, id)
			}
		}
	}
	found := false
	for _, table := range editor.Tables {
		if table.Name != "m_quest_bonus_costume_setting_group" {
			continue
		}
		for _, row := range table.Rows {
			if row.Values["QuestBonusCostumeSettingGroupId"] == "201131" && row.Values["CostumeId"] == "32030" && row.Values["QuestBonusTermGroupId"] == "201131" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expired costume disappeared from editor")
	}
}

func TestQuestBonusGroupAssignmentPreviewAndRoundTrip(t *testing.T) {
	path, file := bonusTestFile(t)
	const newID int64 = 1900000000
	id := strconv.FormatInt(newID, 10)
	bonus := bonusTestGroup(file, questBonusTable, 201131)
	bonus.GroupID = newID
	bonus.Rows[0][0], bonus.Rows[0][4] = id, id
	costumes := bonusTestGroup(file, "m_quest_bonus_costume_setting_group", 201131)
	costumes.GroupID = newID
	// Replace one tier and include a new costume, retaining the other tiers.
	costumes.Rows = costumes.Rows[1:]
	for _, row := range costumes.Rows {
		row[0] = id
	}
	costumes.Rows = append(costumes.Rows, []string{id, "31029", "0", costumes.Rows[0][3], "0"})
	terms := bonusTestGroup(file, "m_quest_bonus_term_group", 201131)
	terms.Rows[0][3] = "4102444799000"
	var quest QuestBonusQuest
	for _, q := range questBonusQuests(file) {
		if q.ChapterID == 573 {
			quest = q
			break
		}
	}
	request := UpdateRequest{ExpectedVersion: file.Version(),
		Changes:          []Change{{Table: questTable, Row: quest.Row, Field: "QuestBonusId", Value: id}},
		QuestBonusGroups: []QuestBonusGroupInput{bonus, costumes, terms},
	}
	preview, err := PreviewUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.QuestBonusGroups) != 3 {
		t.Fatalf("preview=%+v", preview)
	}
	for _, group := range preview.QuestBonusGroups {
		if group.GroupID == newID && (len(group.Quests) != 1 || group.Quests[0].QuestID != quest.QuestID) {
			t.Fatalf("new group usage=%v", group.Quests)
		}
		if group.Table == terms.Table && len(group.Quests) <= 1 {
			t.Fatalf("shared term lost its other users: %v", group.Quests)
		}
	}
	if len(preview.OtherChanges) != 1 || preview.OtherChanges[0].Identity[0].Value != strconv.FormatInt(quest.QuestID, 10) || preview.OtherChanges[0].Changes[0].Before != strconv.FormatInt(quest.BonusID, 10) {
		t.Fatalf("assignment preview=%+v", preview.OtherChanges)
	}
	candidate, result, err := BuildUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.ChangedRows != len(bonus.Rows)+len(costumes.Rows)+2 {
		t.Fatalf("wrong change count: %+v", result)
	}
	output := filepath.Join(t.TempDir(), "bonus.bin.e")
	if err := os.WriteFile(output, candidate, 0600); err != nil {
		t.Fatal(err)
	}
	reloaded, err := LoadTable(output, questBonusTable)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Version != result.Version {
		t.Fatal("reload version differs")
	}
	found := false
	for _, q := range reloaded.QuestBonusEditor.Quests {
		if q.QuestID == quest.QuestID {
			found = true
			if q.BonusID != newID {
				t.Fatalf("assignment not persisted: %+v", q)
			}
		}
	}
	if !found {
		t.Fatal("quest missing after round trip")
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(bonusTestGroup(file, costumes.Table, 201131), bonusTestGroup(rebuilt, costumes.Table, 201131)) {
		t.Fatal("copy modified the shared source group")
	}
	if !reflect.DeepEqual(readRows(file, "m_weapon"), readRows(rebuilt, "m_weapon")) {
		t.Fatal("unrelated table changed")
	}
	if got := bonusTestGroup(rebuilt, terms.Table, terms.GroupID).Rows[0][3]; got != "4102444799000" {
		t.Fatalf("64-bit date truncated: %s", got)
	}
	if len(bonusTestGroup(rebuilt, costumes.Table, newID).Rows) != len(costumes.Rows) {
		t.Fatal("member additions/removals lost")
	}
	if _, _, err := BuildUpdate(output, request); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale version accepted: %v", err)
	}
}

func TestQuestBonusValidationAndRepairOfExistingMissingGroup(t *testing.T) {
	_, file := bonusTestFile(t)
	base := bonusTestGroup(file, "m_quest_bonus_costume_setting_group", 201131)
	for _, tc := range []struct {
		name     string
		mutate   func(*QuestBonusGroupInput)
		contains string
	}{
		{"duplicate tier", func(g *QuestBonusGroupInput) { g.Rows = append(g.Rows, g.Rows[0]) }, "duplicate primary key"},
		{"wrong group", func(g *QuestBonusGroupInput) { g.Rows[0][0] = "123" }, "different group"},
		{"missing costume", func(g *QuestBonusGroupInput) { g.Rows[0][1] = "2147483647" }, "missing m_costume"},
		{"missing effect", func(g *QuestBonusGroupInput) { g.Rows[0][3] = "2147483647" }, "missing m_quest_bonus_effect_group"},
		{"missing term", func(g *QuestBonusGroupInput) { g.Rows[0][4] = "2147483647" }, "missing m_quest_bonus_term_group"},
		{"invalid tier", func(g *QuestBonusGroupInput) { g.Rows[0][2] = "5" }, "limit break"},
		{"referenced deletion", func(g *QuestBonusGroupInput) { g.Rows = nil }, "missing m_quest_bonus_costume_setting_group"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			group := bonusTestGroup(file, base.Table, base.GroupID)
			tc.mutate(&group)
			_, _, err := buildUpdate(file, UpdateRequest{ExpectedVersion: file.Version(), QuestBonusGroups: []QuestBonusGroupInput{group}})
			if err == nil || !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("got %v, want %q", err, tc.contains)
			}
		})
	}
	repair := bonusTestGroup(file, base.Table, base.GroupID)
	repair.GroupID = 200735
	for _, row := range repair.Rows {
		row[0] = "200735"
	}
	if _, _, err := buildUpdate(file, UpdateRequest{ExpectedVersion: file.Version(), QuestBonusGroups: []QuestBonusGroupInput{repair}}); err != nil {
		t.Fatalf("existing missing group cannot be repaired: %v", err)
	}
	newBonus := bonusTestGroup(file, questBonusTable, 201131)
	newBonus.GroupID = 1900000000
	newBonus.Rows[0][0], newBonus.Rows[0][4] = "1900000000", "200735"
	if _, _, err := buildUpdate(file, UpdateRequest{ExpectedVersion: file.Version(), QuestBonusGroups: []QuestBonusGroupInput{newBonus}}); err == nil {
		t.Fatal("new reference to an existing missing group was accepted")
	}
	term := bonusTestGroup(file, "m_quest_bonus_term_group", 201131)
	term.Rows[0][3] = "1"
	if _, _, err := buildUpdate(file, UpdateRequest{ExpectedVersion: file.Version(), QuestBonusGroups: []QuestBonusGroupInput{term}}); err == nil {
		t.Fatal("reversed date range accepted")
	}
	if _, _, err := buildUpdate(file, UpdateRequest{ExpectedVersion: file.Version(), Changes: []Change{{Table: questTable, Row: 0, Field: "QuestBonusId", Value: "2147483647"}}}); err == nil {
		t.Fatal("dangling quest assignment accepted")
	}
}

func TestQuestBonusWeaponAndEffectChainCanBeCreatedTogether(t *testing.T) {
	_, file := bonusTestFile(t)
	const id = "1900000000"
	weapon := bonusTestGroup(file, "m_quest_bonus_weapon_group", 201131)
	effectID, _ := strconv.ParseInt(weapon.Rows[0][3], 10, 64)
	effect := bonusTestGroup(file, "m_quest_bonus_effect_group", effectID)
	dropIndex := -1
	for i, row := range effect.Rows {
		if row[2] == "3" {
			dropIndex = i
			break
		}
	}
	if dropIndex < 0 {
		t.Fatal("weapon fixture lacks a drop effect")
	}
	dropID, _ := strconv.ParseInt(effect.Rows[dropIndex][3], 10, 64)
	drop := bonusTestGroup(file, "m_quest_bonus_drop_reward", dropID)
	drop.GroupID, drop.Rows[0][0] = 1900000000, id
	drop.Rows[0][3] = "99"
	effect.GroupID = 1900000000
	for _, row := range effect.Rows {
		row[0] = id
	}
	effect.Rows[dropIndex][3] = id
	weapon.GroupID = 1900000000
	weapon.Rows = [][]string{{id, "330591", "0", id, "0"}}
	bonus := bonusTestGroup(file, questBonusTable, 201131)
	bonus.GroupID, bonus.Rows[0][0], bonus.Rows[0][3] = 1900000000, id, id
	request := UpdateRequest{ExpectedVersion: file.Version(), QuestBonusGroups: []QuestBonusGroupInput{bonus, weapon, effect, drop}}
	candidate, _, err := buildUpdate(file, request)
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range request.QuestBonusGroups {
		if !reflect.DeepEqual(bonusTestGroup(rebuilt, group.Table, group.GroupID), group) {
			t.Fatalf("chain definition did not survive: %s", group.Table)
		}
	}
	// Deleting an effect still referenced by the original weapon group must fail.
	deletion := QuestBonusGroupInput{Table: effect.Table, GroupID: effectID}
	if _, _, err := buildUpdate(file, UpdateRequest{ExpectedVersion: file.Version(), QuestBonusGroups: []QuestBonusGroupInput{deletion}}); err == nil {
		t.Fatal("referenced effect deletion accepted")
	}
	request.QuestBonusGroups = append(request.QuestBonusGroups, weapon)
	if _, _, err := buildUpdate(file, request); err == nil {
		t.Fatal("duplicate replacement group accepted")
	}
	request.QuestBonusGroups = []QuestBonusGroupInput{bonus}
	request.Changes = []Change{{Table: questBonusTable, Row: 0, Field: "QuestBonusWeaponGroupId", Value: "0"}}
	if _, _, err := buildUpdate(file, request); err == nil || !strings.Contains(err.Error(), "cannot mix") {
		t.Fatalf("mixed edits accepted: %v", err)
	}
}
