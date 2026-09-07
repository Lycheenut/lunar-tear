package masterdataadmin

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
)

func originalDenRestore() QuestBonusRestoreInput {
	return QuestBonusRestoreInput{ChapterID: 501, SourceBonusID: 200031, RuleChapterID: 501, Weapons: []QuestBonusWeaponInput{
		{WeaponID: 250011, TemplateWeaponID: 250011}, {WeaponID: 240131, TemplateWeaponID: 340701},
		{WeaponID: 340151, TemplateWeaponID: 320501}, {WeaponID: 340211, TemplateWeaponID: 320501},
	}}
}

func TestQuestBonusRestorePreservesPhasesAndSynchronizesDates(t *testing.T) {
	path, file := bonusTestFile(t)
	request := UpdateRequest{ExpectedVersion: file.Version(), QuestBonusRestores: []QuestBonusRestoreInput{originalDenRestore()}}
	preview, err := PreviewUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.QuestBonusRestores) != 1 {
		t.Fatalf("missing activity preview: %+v", preview)
	}
	plan := preview.QuestBonusRestores[0]
	if len(plan.Groups) != 3 || !reflect.DeepEqual(plan.CostumeIDs, []int64{24002, 34002}) {
		t.Fatalf("wrong restoration: %+v", plan)
	}
	counts := map[int64]int{201081: 31, 201082: 2, 201083: 2}
	for _, group := range plan.Groups {
		if len(group.QuestIDs) != counts[group.BeforeBonusID] {
			t.Fatalf("phase changed: %+v", group)
		}
		for _, tier := range group.Weapons {
			if tier.LimitBreak != 4 {
				continue
			}
			want := int64(30)
			if tier.WeaponID == 240131 || tier.WeaponID == 240132 || tier.WeaponID == 250011 || tier.WeaponID == 250012 {
				want = 10
			}
			medals := map[int64]bool{181: true}
			if tier.WeaponID != 250011 && tier.WeaponID != 250012 {
				if group.BeforeBonusID >= 201082 {
					medals[182] = true
				}
				if group.BeforeBonusID == 201083 {
					medals[183] = true
				}
			}
			if len(tier.Rewards) != len(medals) {
				t.Fatalf("wrong medal eligibility: %+v", tier)
			}
			for _, reward := range tier.Rewards {
				if !medals[reward.PossessionID] || reward.Count != want {
					t.Fatalf("wrong quantity/currency: %+v", tier)
				}
			}
		}
	}
	candidate, _, err := BuildUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	p := newBonusRestorePlanner(rebuilt)
	for _, quest := range p.quests {
		if quest.ChapterID == 501 {
			terms := p.terms(quest.BonusID)
			if len(terms) != 1 || !terms[plan.TermGroupID] {
				t.Fatalf("member term not synchronized: %v", terms)
			}
		}
	}
	for _, table := range []string{bonusEffects, bonusDrops, "m_event_quest_chapter", "m_quest_pickup_reward_group"} {
		if !reflect.DeepEqual(readRows(file, table), readRows(rebuilt, table)) {
			t.Fatalf("unexpected edit to %s", table)
		}
	}
	original := newBonusRestorePlanner(file)
	for _, spec := range questBonusTableSpecs {
		for id, rows := range original.groups[spec.Name] {
			if !reflect.DeepEqual(rows, p.groups[spec.Name][id]) {
				t.Fatalf("shared source modified: %s:%d", spec.Name, id)
			}
		}
	}
	output := filepath.Join(t.TempDir(), "restored.bin.e")
	if err := os.WriteFile(output, candidate, 0600); err != nil {
		t.Fatal(err)
	}
	// Reapplying the same choices uses the current equivalent templates, with
	// no additional groups. Historical source IDs remain usable after reload.
	again := originalDenRestore()
	for i := range again.Weapons {
		again.Weapons[i].TemplateWeaponID = again.Weapons[i].WeaponID
	}
	if _, _, err := BuildUpdate(output, UpdateRequest{ExpectedVersion: rebuilt.Version(), QuestBonusRestores: []QuestBonusRestoreInput{again}}); err == nil || !strings.Contains(err.Error(), "unchanged") {
		t.Fatalf("non-idempotent restoration: %v", err)
	}
	var chapterRow int
	for i, row := range readRows(rebuilt, "m_event_quest_chapter") {
		if bonusInt(row, 0) == 501 {
			chapterRow = i
		}
	}
	newEnd := plan.EndDatetime + 86400000
	timed, _, err := BuildUpdate(output, UpdateRequest{ExpectedVersion: rebuilt.Version(), Changes: []Change{{Table: "m_event_quest_chapter", Row: chapterRow, Field: "EndDatetime", Value: bonusString(newEnd)}}})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := memorydb.OpenBytes(timed)
	if err != nil {
		t.Fatal(err)
	}
	if got := bonusTestGroup(updated, bonusTerms, plan.TermGroupID).Rows[0]; got[2] != bonusString(plan.StartDatetime) || got[3] != bonusString(newEnd) {
		t.Fatalf("date did not cascade: %v", got)
	}
	if !reflect.DeepEqual(readRows(rebuilt, questTable), readRows(updated, questTable)) {
		t.Fatal("private term synchronization unnecessarily reassigned quests")
	}
}

func TestQuestBonusRestoreRequiresCompleteMappings(t *testing.T) {
	path, file := bonusTestFile(t)
	for _, tc := range []struct {
		name    string
		mutate  func(*QuestBonusRestoreInput)
		message string
	}{
		{"missing weapon rule", func(r *QuestBonusRestoreInput) { r.Weapons = r.Weapons[:1] }, "尚未选择规则"},
		{"source missing", func(r *QuestBonusRestoreInput) { r.SourceBonusID = 2147483647 }, "来源加成"},
		{"foreign template", func(r *QuestBonusRestoreInput) { r.Weapons[0].TemplateWeaponID = 240011 }, "进化形态"},
		{"zero bonus needs reference", func(r *QuestBonusRestoreInput) { r.ChapterID = 589; r.RuleChapterID = 589 }, "缺少现有加成"},
		{"external reference needs all quests", func(r *QuestBonusRestoreInput) { r.RuleChapterID = 573 }, "全部目标关卡"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := originalDenRestore()
			tc.mutate(&input)
			_, _, err := BuildUpdate(path, UpdateRequest{ExpectedVersion: file.Version(), QuestBonusRestores: []QuestBonusRestoreInput{input}})
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("got %v, want %s", err, tc.message)
			}
		})
	}
}

func TestQuestBonusMedalsFollowPickupReferences(t *testing.T) {
	_, file := bonusTestFile(t)
	_, medals := questBonusMedals(file)
	for qid, want := range map[int64][]int64{201319: {181}, 201320: {181, 182}, 201322: {181, 182, 183}} {
		if !reflect.DeepEqual(medals[qid], want) {
			t.Fatalf("quest %d medals=%v, want %v", qid, medals[qid], want)
		}
	}
}

func TestQuestBonusRestoreExternalRulesRemapActualMedals(t *testing.T) {
	path, file := bonusTestFile(t)
	input := originalDenRestore()
	input.ChapterID = 589 // All quests initially have QuestBonusId=0.
	input.Currencies = []QuestBonusCurrencyInput{{181, 249}, {182, 250}, {183, 251}}
	_, medals := questBonusMedals(file)
	groups := map[int][]int64{}
	for _, q := range questBonusQuests(file) {
		if q.ChapterID == 589 {
			groups[len(medals[q.QuestID])] = append(groups[len(medals[q.QuestID])], q.QuestID)
		}
	}
	for count := 1; count <= 3; count++ {
		input.Groups = append(input.Groups, QuestBonusRuleInput{QuestIDs: groups[count], RuleBonusID: 201080 + int64(count)})
	}
	request := UpdateRequest{ExpectedVersion: file.Version(), QuestBonusRestores: []QuestBonusRestoreInput{input}}
	preview, err := PreviewUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.QuestBonusRestores[0].Groups) != 3 {
		t.Fatal("zero bonus lost medal phases")
	}
	for _, group := range preview.QuestBonusRestores[0].Groups {
		for _, tier := range group.Weapons {
			for _, reward := range tier.Rewards {
				if reward.PossessionID < 249 || reward.PossessionID > 251 {
					t.Fatalf("old currency leaked: %+v", reward)
				}
				for _, id := range group.QuestIDs {
					found := false
					for _, actual := range medals[id] {
						found = found || actual == reward.PossessionID
					}
					if !found {
						t.Fatalf("unavailable medal %d in quest %d", reward.PossessionID, id)
					}
				}
			}
		}
	}
	candidate, _, err := BuildUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"m_battle_drop_reward", "m_quest_pickup_reward_group", "m_event_quest_chapter"} {
		if !reflect.DeepEqual(readRows(file, table), readRows(rebuilt, table)) {
			t.Fatalf("base data changed: %s", table)
		}
	}
	input.Currencies[2].ToID = 183
	request.QuestBonusRestores[0] = input
	if _, _, err := BuildUpdate(path, request); err == nil || !strings.Contains(err.Error(), "实际奖章") {
		t.Fatalf("invalid target medal accepted: %v", err)
	}
}

func TestQuestBonusInitialDateEditIsolatesAllMemberTerms(t *testing.T) {
	path, file := bonusTestFile(t)
	var index int
	var start, end int64
	for i, row := range readRows(file, "m_event_quest_chapter") {
		if bonusInt(row, 0) == 501 {
			index, start, end = i, bonusInt(row, 8), bonusInt(row, 9)
		}
	}
	request := UpdateRequest{ExpectedVersion: file.Version(), Changes: []Change{{Table: "m_event_quest_chapter", Row: index, Field: "EndDatetime", Value: bonusString(end + 86400000)}}}
	preview, err := PreviewUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	plan := preview.QuestBonusRestores[0]
	if !plan.ScheduleOnly || plan.StartDatetime != start || plan.EndDatetime != end+86400000 {
		t.Fatalf("invalid date preview: %+v", plan)
	}
	candidate, _, err := BuildUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	p := newBonusRestorePlanner(rebuilt)
	for _, q := range p.quests {
		if q.ChapterID == 501 {
			terms := p.terms(q.BonusID)
			if len(terms) != 1 || !terms[plan.TermGroupID] {
				t.Fatalf("unsynchronized member: %v", terms)
			}
		} else if p.terms(q.BonusID)[plan.TermGroupID] {
			t.Fatalf("activity term shared with %d", q.ChapterID)
		}
	}
}

func TestQuestBonusAppendSelectedMembersPreservesCurrentPhases(t *testing.T) {
	path, file := bonusTestFile(t)
	input := QuestBonusRestoreInput{ChapterID: 573, SourceBonusID: 201121, Mode: "append", CostumeIDs: []int64{31029, 33027}, RuleChapterID: 573,
		Weapons: []QuestBonusWeaponInput{{310621, 350621}, {330591, 350621}}}
	request := UpdateRequest{ExpectedVersion: file.Version(), QuestBonusRestores: []QuestBonusRestoreInput{input}}
	preview, err := PreviewUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	plan := preview.QuestBonusRestores[0]
	if plan.Mode != "append" || !reflect.DeepEqual(plan.CostumeIDs, []int64{31029, 32030, 33027, 35031}) || len(plan.Groups) != 3 {
		t.Fatalf("wrong supplemented roster: %+v", plan)
	}
	candidate, _, err := BuildUpdate(path, request)
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := memorydb.OpenBytes(candidate)
	if err != nil {
		t.Fatal(err)
	}
	before, after := newBonusRestorePlanner(file), newBonusRestorePlanner(rebuilt)
	counts := map[int64]int{201131: 31, 201132: 2, 201133: 2}
	for _, group := range plan.Groups {
		if len(group.QuestIDs) != counts[group.BeforeBonusID] {
			t.Fatalf("phase assignments changed: %+v", group)
		}
		old := before.groups[questBonusTable][group.BeforeBonusID][0]
		updated := after.groups[questBonusTable][group.AfterBonusID][0]
		for _, link := range []struct {
			table  string
			column int
		}{{bonusCostumes, 4}, {bonusWeapons, 3}} {
			for _, row := range before.groups[link.table][bonusNumber(old[link.column])] {
				matches := 0
				for _, next := range after.groups[link.table][bonusNumber(updated[link.column])] {
					if reflect.DeepEqual(row[1:4], next[1:4]) && next[4] == bonusString(plan.TermGroupID) {
						matches++
					}
				}
				if matches != 1 {
					t.Fatalf("current phase member lost, altered or duplicated: %s %v", link.table, row)
				}
			}
		}
		families := make(map[int64]bool)
		for _, tier := range group.Weapons {
			family := after.family[tier.WeaponID]
			families[family] = true
			if family != 31062 && family != 33059 {
				continue
			}
			found := false
			for _, template := range before.weaponPreview(bonusNumber(old[3])) {
				if template.WeaponID == 350620+after.order[tier.WeaponID] && template.LimitBreak == tier.LimitBreak && reflect.DeepEqual(template.Rewards, tier.Rewards) {
					found = true
				}
			}
			if !found {
				t.Fatalf("new weapon lost reference phase/evolution/tier rules: %+v", tier)
			}
		}
		if len(families) != 5 || !families[31062] || !families[33059] {
			t.Fatalf("unchecked source weapons leaked: %v", families)
		}
	}
	for _, spec := range questBonusTableSpecs {
		for id, rows := range before.groups[spec.Name] {
			if !reflect.DeepEqual(rows, after.groups[spec.Name][id]) {
				t.Fatalf("shared definition modified: %s:%d", spec.Name, id)
			}
		}
	}
	output := filepath.Join(t.TempDir(), "supplemented.bin.e")
	if err := os.WriteFile(output, candidate, 0600); err != nil {
		t.Fatal(err)
	}
	request.ExpectedVersion = rebuilt.Version()
	if _, _, err := BuildUpdate(output, request); err == nil || !strings.Contains(err.Error(), "unchanged") {
		t.Fatalf("repeating additions must be a no-op: %v", err)
	}
}

func TestQuestBonusSelectiveListsAndEmptySupplement(t *testing.T) {
	_, file := bonusTestFile(t)
	for _, tc := range []struct {
		name     string
		input    QuestBonusRestoreInput
		costumes []int64
	}{
		{"costumes only on zero bonus", QuestBonusRestoreInput{ChapterID: 589, SourceBonusID: 201121, Mode: "replace", CostumeIDs: []int64{31029}, RuleChapterID: 573}, []int64{31029}},
		{"explicit empty replacement", QuestBonusRestoreInput{ChapterID: 573, SourceBonusID: 201121, Mode: "replace", CostumeIDs: []int64{}}, []int64{}},
		{"empty supplement", QuestBonusRestoreInput{ChapterID: 573, SourceBonusID: 201121, Mode: "append", CostumeIDs: []int64{}}, nil},
		{"existing members only", QuestBonusRestoreInput{ChapterID: 573, SourceBonusID: 201131, Mode: "append", CostumeIDs: []int64{35031}, Weapons: []QuestBonusWeaponInput{{350621, 250011}}}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request, previews, err := planQuestBonusUpdates(file, UpdateRequest{QuestBonusRestores: []QuestBonusRestoreInput{tc.input}})
			if err != nil {
				t.Fatal(err)
			}
			if tc.costumes == nil {
				if len(previews) != 0 || len(request.Changes) != 0 || len(request.QuestBonusGroups) != 0 {
					t.Fatal("no-op supplement generated changes")
				}
				return
			}
			if len(previews) != 1 || !reflect.DeepEqual(previews[0].CostumeIDs, tc.costumes) {
				t.Fatalf("partial costume selection ignored: %+v", previews)
			}
			for _, group := range previews[0].Groups {
				if len(group.Weapons) != 0 {
					t.Fatal("unchecked weapons included")
				}
			}
		})
	}
	var chapterRow int
	for i, row := range readRows(file, "m_event_quest_chapter") {
		if bonusInt(row, 0) == 573 {
			chapterRow = i
			break
		}
	}
	end := bonusInt(readRows(file, "m_event_quest_chapter")[chapterRow], 9) + 86400000
	_, previews, err := planQuestBonusUpdates(file, UpdateRequest{Changes: []Change{{Table: "m_event_quest_chapter", Row: chapterRow, Field: "EndDatetime", Value: bonusString(end)}}, QuestBonusRestores: []QuestBonusRestoreInput{{ChapterID: 573, SourceBonusID: 201121, Mode: "append", CostumeIDs: []int64{}}}})
	if err != nil || len(previews) != 1 || !previews[0].ScheduleOnly || previews[0].EndDatetime != end {
		t.Fatalf("empty supplement swallowed date edits: %+v %v", previews, err)
	}
}

func TestQuestBonusSelectiveSourceValidation(t *testing.T) {
	_, file := bonusTestFile(t)
	for _, tc := range []struct {
		name   string
		mutate func(*QuestBonusRestoreInput)
	}{
		{"invalid mode", func(r *QuestBonusRestoreInput) { r.Mode = "merge" }},
		{"missing explicit costume selection", func(r *QuestBonusRestoreInput) { r.CostumeIDs = nil }},
		{"foreign costume", func(r *QuestBonusRestoreInput) { r.CostumeIDs = []int64{35031} }},
		{"duplicate costume", func(r *QuestBonusRestoreInput) { r.CostumeIDs = []int64{31029, 31029} }},
		{"foreign weapon", func(r *QuestBonusRestoreInput) { r.Weapons = []QuestBonusWeaponInput{{350621, 350621}} }},
		{"duplicate weapon family", func(r *QuestBonusRestoreInput) {
			r.Weapons = []QuestBonusWeaponInput{{310621, 350621}, {310622, 350621}}
		}},
		{"missing selected weapon rule", func(r *QuestBonusRestoreInput) { r.Weapons = []QuestBonusWeaponInput{{310621, 0}} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := QuestBonusRestoreInput{ChapterID: 573, SourceBonusID: 201121, Mode: "append", CostumeIDs: []int64{31029}}
			tc.mutate(&input)
			if _, _, err := planQuestBonusUpdates(file, UpdateRequest{QuestBonusRestores: []QuestBonusRestoreInput{input}}); err == nil {
				t.Fatal("invalid selection accepted")
			}
		})
	}
}
