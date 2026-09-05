package masterdataadmin

import (
	"fmt"
	"sort"
	"strings"

	"lunar-tear/server/internal/masterdata/memorydb"
)

const questBonusTable = "m_quest_bonus"

// Only these small definitions can be replaced, and only within one ID group.
// The obsolete costume group is deliberately not used for costume membership.
var questBonusTableSpecs = []tableSpec{
	activityTable(questBonusTable, "EntityMQuestBonus", 1, false, field("QuestBonusId", 0, "int"), field("QuestBonusCharacterGroupId", 1, "int"), field("QuestBonusCostumeGroupId", 2, "int"), field("QuestBonusWeaponGroupId", 3, "int"), field("QuestBonusCostumeSettingGroupId", 4, "int"), field("QuestBonusAllyCharacterId", 5, "int")),
	activityTable("m_quest_bonus_costume_setting_group", "EntityMQuestBonusCostumeSettingGroup", 3, false, field("QuestBonusCostumeSettingGroupId", 0, "int"), field("CostumeId", 1, "int"), field("LimitBreakCountLowerLimit", 2, "int"), field("QuestBonusEffectGroupId", 3, "int"), field("QuestBonusTermGroupId", 4, "int")),
	activityTable("m_quest_bonus_weapon_group", "EntityMQuestBonusWeaponGroup", 3, false, field("QuestBonusWeaponGroupId", 0, "int"), field("WeaponId", 1, "int"), field("LimitBreakCountLowerLimit", 2, "int"), field("QuestBonusEffectGroupId", 3, "int"), field("QuestBonusTermGroupId", 4, "int")),
	activityTable("m_quest_bonus_effect_group", "EntityMQuestBonusEffectGroup", 2, false, field("QuestBonusEffectGroupId", 0, "int"), field("SortOrder", 1, "int"), field("QuestBonusType", 2, "QuestBonusType"), field("QuestBonusEffectId", 3, "int")),
	activityTable("m_quest_bonus_term_group", "EntityMQuestBonusTermGroup", 2, false, field("QuestBonusTermGroupId", 0, "int"), field("SortOrder", 1, "int"), field("StartDatetime", 2, "long"), field("EndDatetime", 3, "long")),
	activityTable("m_quest_bonus_ability", "EntityMQuestBonusAbility", 1, false, field("QuestBonusEffectId", 0, "int"), field("AbilityId", 1, "int"), field("Level", 2, "int")),
	activityTable("m_quest_bonus_exp", "EntityMQuestBonusExp", 1, false, field("QuestBonusEffectId", 0, "int"), field("ExpType", 1, "int"), field("BonusValuePermil", 2, "int")),
	activityTable("m_quest_bonus_drop_reward", "EntityMQuestBonusDropReward", 1, false, field("QuestBonusEffectId", 0, "int"), field("PossessionType", 1, "PossessionType"), field("PossessionId", 2, "int"), field("AdditionalCount", 3, "int")),
}

type QuestBonusOption struct {
	ID     int64             `json:"id"`
	Titles map[string]string `json:"titles,omitempty"`
}

type QuestBonusQuest struct {
	QuestID    int64 `json:"questId"`
	Row        int   `json:"row"`
	ChapterID  int64 `json:"chapterId"`
	Difficulty int64 `json:"difficulty"`
	SequenceID int64 `json:"sequenceId"`
	SortOrder  int64 `json:"sortOrder"`
	BonusID    int64 `json:"bonusId"`
}

type QuestBonusEditorCatalog struct {
	ReadOnlyLinks map[string][]string `json:"readOnlyLinks,omitempty"`
	Chapters      []Row               `json:"chapters"`
	Quests        []QuestBonusQuest   `json:"quests"`
	Tables        []Table             `json:"tables"`
	Costumes      []QuestBonusOption  `json:"costumes"`
	Weapons       []QuestBonusOption  `json:"weapons"`
}

type QuestBonusGroupInput struct {
	Table   string     `json:"table"`
	GroupID int64      `json:"groupId"`
	Rows    [][]string `json:"rows"`
}

type QuestBonusGroupPreview struct {
	Table   string            `json:"table"`
	GroupID int64             `json:"groupId"`
	Fields  []string          `json:"fields"`
	Before  [][]string        `json:"before"`
	After   [][]string        `json:"after"`
	Quests  []QuestBonusQuest `json:"quests"`
}

func bonusInt(row []interface{}, column int) int64 {
	value, _ := integerAt(row, column)
	return value
}

func loadQuestBonusEditor(file *memorydb.File, resolver *titleResolver) (QuestBonusEditorCatalog, error) {
	result := QuestBonusEditorCatalog{Quests: questBonusQuests(file)}
	graph, _ := bonusReferenceGraph(file)
	result.ReadOnlyLinks = make(map[string][]string)
	for source, targets := range graph {
		if strings.HasPrefix(source, "m_quest_bonus_character_group:") || strings.HasPrefix(source, "m_quest_bonus_costume_group:") || strings.HasPrefix(source, "m_quest_bonus_ally_character:") {
			result.ReadOnlyLinks[source] = targets
		}
	}
	spec, _ := tableSpecByName("m_event_quest_chapter")
	chapters, _, err := tableFromFile(file, resolver, spec, true)
	if err != nil {
		return result, err
	}
	result.Chapters = chapters.Rows
	for _, spec := range questBonusTableSpecs {
		table, exists, err := tableFromFile(file, resolver, spec, true)
		if err != nil {
			return result, err
		}
		if exists {
			result.Tables = append(result.Tables, table)
		}
	}
	for _, row := range readRows(file, "m_costume") {
		asset := fmt.Sprintf("ch%03d%03d", bonusInt(row, 4), bonusInt(row, 5))
		result.Costumes = append(result.Costumes, QuestBonusOption{bonusInt(row, 0), resolver.titlesForKeys([]string{"costume.name.replace." + asset, "costume.name." + asset})})
	}
	for _, row := range readRows(file, "m_weapon") {
		prefix := "wp"
		if bonusInt(row, 1) == 2 {
			prefix = "ac"
		}
		asset := fmt.Sprintf("%s%03d%03d", prefix, bonusInt(row, 2), bonusInt(row, 3))
		result.Weapons = append(result.Weapons, QuestBonusOption{bonusInt(row, 0), resolver.titlesForKeys([]string{"weapon.name.replace." + asset + ".1", "weapon.name." + asset + ".1", "weapon.name.replace." + asset + ".2", "weapon.name." + asset + ".2"})})
	}
	return result, nil
}

func questBonusQuests(file *memorydb.File) []QuestBonusQuest {
	quests := make(map[int64]QuestBonusQuest)
	for index, row := range readRows(file, questTable) {
		id := bonusInt(row, 0)
		quests[id] = QuestBonusQuest{QuestID: id, Row: index, BonusID: bonusInt(row, 19)}
	}
	groups := make(map[int64][][]interface{})
	for _, row := range readRows(file, "m_event_quest_sequence_group") {
		groups[bonusInt(row, 0)] = append(groups[bonusInt(row, 0)], row)
	}
	sequences := make(map[int64][][]interface{})
	for _, row := range readRows(file, "m_event_quest_sequence") {
		sequences[bonusInt(row, 0)] = append(sequences[bonusInt(row, 0)], row)
	}
	var result []QuestBonusQuest
	placed := make(map[int64]bool)
	for _, chapter := range readRows(file, "m_event_quest_chapter") {
		for _, group := range groups[bonusInt(chapter, 7)] {
			for _, sequence := range sequences[bonusInt(group, 2)] {
				quest, ok := quests[bonusInt(sequence, 2)]
				if !ok {
					continue
				}
				quest.ChapterID, quest.Difficulty = bonusInt(chapter, 0), bonusInt(group, 1)
				quest.SequenceID, quest.SortOrder = bonusInt(group, 2), bonusInt(sequence, 1)
				result = append(result, quest)
				placed[quest.QuestID] = true
			}
		}
	}
	// Keep non-event users in shared-group impact previews too.
	for id, quest := range quests {
		if !placed[id] && quest.BonusID != 0 {
			result = append(result, quest)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.ChapterID != b.ChapterID {
			return a.ChapterID < b.ChapterID
		}
		if a.Difficulty != b.Difficulty {
			return a.Difficulty < b.Difficulty
		}
		if a.SortOrder != b.SortOrder {
			return a.SortOrder < b.SortOrder
		}
		return a.QuestID < b.QuestID
	})
	return result
}

func questBonusSpec(name string) (tableSpec, bool) {
	for _, spec := range questBonusTableSpecs {
		if spec.Name == name {
			return spec, true
		}
	}
	return tableSpec{}, false
}

func bonusRowKey(spec tableSpec, row []interface{}) string {
	var parts []string
	for _, field := range spec.Fields {
		if field.PrimaryKey {
			parts = append(parts, fmt.Sprint(row[field.Index]))
		}
	}
	return strings.Join(parts, ":")
}

func bonusStringRows(rows [][]interface{}) [][]string {
	result := make([][]string, 0, len(rows))
	for _, row := range rows {
		values := make([]string, len(row))
		for i, v := range row {
			values[i] = fmt.Sprint(v)
		}
		result = append(result, values)
	}
	return result
}

func prepareQuestBonusGroups(file *memorydb.File, inputs []QuestBonusGroupInput, changes []Change) (map[string][][]interface{}, int, int, []QuestBonusGroupPreview, error) {
	replacements := make(map[string][][]interface{})
	seen := make(map[string]bool)
	cells, changedRows, totalRows := 0, 0, 0
	var previews []QuestBonusGroupPreview
	for _, input := range inputs {
		spec, ok := questBonusSpec(input.Table)
		if !ok {
			return nil, 0, 0, nil, fmt.Errorf("table %q is not a quest bonus definition", input.Table)
		}
		key := fmt.Sprintf("%s:%d", input.Table, input.GroupID)
		if input.GroupID <= 0 || input.GroupID > 2147483647 || seen[key] {
			return nil, 0, 0, nil, fmt.Errorf("invalid or duplicate quest bonus group %s", key)
		}
		seen[key] = true
		totalRows += len(input.Rows)
		if totalRows > 10000 || len(inputs) > 1000 {
			return nil, 0, 0, nil, fmt.Errorf("too many quest bonus rows")
		}
		for _, change := range changes {
			if change.Table == input.Table {
				return nil, 0, 0, nil, fmt.Errorf("cannot mix cell edits and group replacement for %s", input.Table)
			}
		}
		current, loaded := replacements[input.Table]
		if !loaded {
			var exists bool
			var err error
			current, exists, err = file.TableRows(input.Table)
			if err != nil {
				return nil, 0, 0, nil, err
			}
			if !exists {
				return nil, 0, 0, nil, fmt.Errorf("table %s is absent", input.Table)
			}
		}
		var before, after, retained [][]interface{}
		for _, row := range current {
			if bonusInt(row, 0) == input.GroupID {
				before = append(before, row)
			} else {
				retained = append(retained, row)
			}
		}
		keys := make(map[string]bool)
		for _, values := range input.Rows {
			if len(values) != len(spec.Fields) {
				return nil, 0, 0, nil, fmt.Errorf("%s: expected %d columns", key, len(spec.Fields))
			}
			row := make([]interface{}, len(values))
			for i, field := range spec.Fields {
				value, err := parseChangeValue(field, values[i])
				if err != nil {
					return nil, 0, 0, nil, fmt.Errorf("%s %s: %w", key, field.Name, err)
				}
				row[i] = value
				if bonusInt(row, i) < 0 {
					return nil, 0, 0, nil, fmt.Errorf("%s %s cannot be negative", key, field.Name)
				}
			}
			if bonusInt(row, 0) != input.GroupID {
				return nil, 0, 0, nil, fmt.Errorf("%s: row belongs to a different group", key)
			}
			rowKey := bonusRowKey(spec, row)
			if keys[rowKey] {
				return nil, 0, 0, nil, fmt.Errorf("%s: duplicate primary key %s", key, rowKey)
			}
			keys[rowKey] = true
			if strings.HasSuffix(input.Table, "costume_setting_group") || strings.HasSuffix(input.Table, "weapon_group") {
				if bonusInt(row, 2) > 4 {
					return nil, 0, 0, nil, fmt.Errorf("%s: limit break must be between 0 and 4", key)
				}
			}
			if input.Table == "m_quest_bonus_effect_group" && (bonusInt(row, 2) < 1 || bonusInt(row, 2) > 3) {
				return nil, 0, 0, nil, fmt.Errorf("%s: unsupported quest bonus type", key)
			}
			if input.Table == "m_quest_bonus_term_group" && bonusInt(row, 3) != 0 && bonusInt(row, 2) > bonusInt(row, 3) {
				return nil, 0, 0, nil, fmt.Errorf("%s: StartDatetime must not be after EndDatetime", key)
			}
			after = append(after, row)
		}
		oldByKey := make(map[string][]interface{})
		for _, row := range before {
			oldByKey[bonusRowKey(spec, row)] = row
		}
		groupChanges := 0
		for _, row := range after {
			rowKey := bonusRowKey(spec, row)
			old, exists := oldByKey[rowKey]
			n := 0
			for i, value := range row {
				if !exists || fmt.Sprint(old[i]) != fmt.Sprint(value) {
					n++
				}
			}
			if n > 0 {
				groupChanges++
				cells += n
			}
			delete(oldByKey, rowKey)
		}
		groupChanges += len(oldByKey)
		cells += len(oldByKey) * len(spec.Fields)
		if groupChanges == 0 {
			continue
		}
		changedRows += groupChanges
		replacement := append(retained, after...)
		sort.Slice(replacement, func(i, j int) bool {
			for _, field := range spec.Fields {
				if !field.PrimaryKey {
					continue
				}
				a, b := bonusInt(replacement[i], field.Index), bonusInt(replacement[j], field.Index)
				if a != b {
					return a < b
				}
			}
			return false
		})
		replacements[input.Table] = replacement
		preview := QuestBonusGroupPreview{Table: input.Table, GroupID: input.GroupID, Before: bonusStringRows(before), After: bonusStringRows(after)}
		for _, field := range spec.Fields {
			preview.Fields = append(preview.Fields, field.Name)
		}
		previews = append(previews, preview)
	}
	return replacements, cells, changedRows, previews, nil
}

type bonusReference struct {
	table    string
	column   int
	target   string
	optional bool
}

var bonusReferences = []bonusReference{
	{questTable, 19, questBonusTable, true},
	{questBonusTable, 1, "m_quest_bonus_character_group", true},
	// QuestBonusCostumeGroupId is obsolete; many valid modern rows retain
	// IDs whose legacy groups no longer exist. Preserve it without validating it.
	{questBonusTable, 3, "m_quest_bonus_weapon_group", true},
	{questBonusTable, 4, "m_quest_bonus_costume_setting_group", true},
	{questBonusTable, 5, "m_quest_bonus_ally_character", true},
	{"m_quest_bonus_costume_setting_group", 1, "m_costume", false},
	{"m_quest_bonus_costume_setting_group", 3, "m_quest_bonus_effect_group", false},
	{"m_quest_bonus_costume_setting_group", 4, "m_quest_bonus_term_group", true},
	{"m_quest_bonus_weapon_group", 1, "m_weapon", false},
	{"m_quest_bonus_weapon_group", 3, "m_quest_bonus_effect_group", false},
	{"m_quest_bonus_weapon_group", 4, "m_quest_bonus_term_group", true},
	{"m_quest_bonus_ability", 1, "m_ability", false},
	{"m_quest_bonus_character_group", 2, "m_quest_bonus_effect_group", false},
	{"m_quest_bonus_character_group", 3, "m_quest_bonus_term_group", true},
	{"m_quest_bonus_costume_group", 2, "m_quest_bonus_effect_group", false},
	{"m_quest_bonus_costume_group", 3, "m_quest_bonus_term_group", true},
	{"m_quest_bonus_ally_character", 1, "m_quest_bonus_effect_group", false},
	{"m_quest_bonus_ally_character", 2, "m_quest_bonus_term_group", true},
}

// Reference keys include the full source primary key: introducing another row
// pointing to an already-missing group must still be rejected.
func bonusReferenceGraph(file *memorydb.File) (map[string][]string, map[string]bool) {
	graph, missing := make(map[string][]string), make(map[string]bool)
	targets := make(map[string]map[int64]bool)
	add := func(table string, row []interface{}, column int, target string, optional bool) {
		id := bonusInt(row, column)
		if id == 0 && optional {
			return
		}
		if targets[target] == nil {
			targets[target] = make(map[int64]bool)
			for _, r := range readRows(file, target) {
				targets[target][bonusInt(r, 0)] = true
			}
		}
		source := fmt.Sprintf("%s:%d", table, bonusInt(row, 0))
		destination := fmt.Sprintf("%s:%d", target, id)
		graph[source] = append(graph[source], destination)
		if !targets[target][id] || id <= 0 {
			identity := fmt.Sprint(bonusInt(row, 0))
			if spec, ok := questBonusSpec(table); ok {
				identity = bonusRowKey(spec, row)
			}
			missing[fmt.Sprintf("%s[%s] column %d -> missing %s", table, identity, column, destination)] = true
		}
	}
	for _, ref := range bonusReferences {
		for _, row := range readRows(file, ref.table) {
			add(ref.table, row, ref.column, ref.target, ref.optional)
		}
	}
	for _, row := range readRows(file, "m_quest_bonus_effect_group") {
		target := map[int64]string{1: "m_quest_bonus_ability", 2: "m_quest_bonus_exp", 3: "m_quest_bonus_drop_reward"}[bonusInt(row, 2)]
		if target != "" {
			add("m_quest_bonus_effect_group", row, 3, target, false)
		}
	}
	return graph, missing
}

func validateQuestBonusReferences(before, after *memorydb.File) error {
	_, oldMissing := bonusReferenceGraph(before)
	_, newMissing := bonusReferenceGraph(after)
	var errors []string
	for key := range newMissing {
		if !oldMissing[key] {
			errors = append(errors, key)
		}
	}
	sort.Strings(errors)
	if len(errors) > 0 {
		return fmt.Errorf("quest bonus reference: %s", errors[0])
	}
	return nil
}

func bonusDependsOn(graph map[string][]string, source, target string, seen map[string]bool) bool {
	if source == target {
		return true
	}
	if seen[source] {
		return false
	}
	seen[source] = true
	for _, child := range graph[source] {
		if bonusDependsOn(graph, child, target, seen) {
			return true
		}
	}
	return false
}

func appendQuestBonusPreview(preview *UpdatePreview, before, after *memorydb.File, request UpdateRequest) error {
	_, _, _, groups, err := prepareQuestBonusGroups(before, request.QuestBonusGroups, request.Changes)
	if err != nil {
		return err
	}
	oldGraph, _ := bonusReferenceGraph(before)
	newGraph, _ := bonusReferenceGraph(after)
	for i := range groups {
		target := fmt.Sprintf("%s:%d", groups[i].Table, groups[i].GroupID)
		seen := make(map[string]bool)
		for index, file := range []*memorydb.File{before, after} {
			graph := oldGraph
			if index == 1 {
				graph = newGraph
			}
			for _, quest := range questBonusQuests(file) {
				key := fmt.Sprintf("%d:%d:%d", quest.ChapterID, quest.Difficulty, quest.QuestID)
				if !seen[key] && bonusDependsOn(graph, fmt.Sprintf("%s:%d", questBonusTable, quest.BonusID), target, make(map[string]bool)) {
					groups[i].Quests = append(groups[i].Quests, quest)
					seen[key] = true
				}
			}
		}
	}
	preview.QuestBonusGroups = groups
	return nil
}
