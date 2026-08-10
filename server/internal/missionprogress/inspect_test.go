package missionprogress

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestInspectMissionParameters(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	missions, err := memorydb.ReadTable[masterdata.EntityMMission]("m_mission")
	if err != nil {
		t.Fatal(err)
	}
	types := make(map[int32][]masterdata.EntityMMission)
	for _, mission := range missions {
		if mission.MissionClearConditionOptionGroupId == 0 && mission.MissionClearConditionOptionDetailGroupId == 0 {
			continue
		}
		types[mission.MissionClearConditionType] = append(types[mission.MissionClearConditionType], mission)
	}
	var conditionTypes []int
	for conditionType := range types {
		conditionTypes = append(conditionTypes, int(conditionType))
	}
	sort.Ints(conditionTypes)
	for _, rawType := range conditionTypes {
		options := make(map[int32]bool)
		details := make(map[int32]bool)
		for _, mission := range types[int32(rawType)] {
			options[mission.MissionClearConditionOptionGroupId] = true
			details[mission.MissionClearConditionOptionDetailGroupId] = true
		}
		t.Logf("type=%d rows=%d options=%s details=%s", rawType, len(types[int32(rawType)]), summarizeIds(options), summarizeIds(details))
	}
}

func TestInspectQuestClearOptions(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	parts, err := masterdata.LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := masterdata.LoadConditionResolver()
	if err != nil {
		t.Fatal(err)
	}
	quests, err := masterdata.LoadQuestCatalog(parts, resolver)
	if err != nil {
		t.Fatal(err)
	}
	missions, err := masterdata.LoadMissionCatalog()
	if err != nil {
		t.Fatal(err)
	}
	eventTypes := make(map[int32]bool)
	for _, eventType := range quests.EventQuestTypeByChapterId {
		eventTypes[eventType] = true
	}
	t.Logf("event types=%s", summarizeIds(eventTypes))
	classes := map[string]map[int32]bool{
		"main-or-event-type": {}, "event-chapter": {}, "quest-id": {}, "unknown": {},
	}
	for _, mission := range missions.OrderedMissions {
		if mission.MissionClearConditionType != 1 || mission.MissionClearConditionOptionGroupId == 0 {
			continue
		}
		option := mission.MissionClearConditionOptionGroupId
		switch {
		case option == 4 || option == 5 || eventTypes[option]:
			classes["main-or-event-type"][option] = true
		case quests.EventChapterById[option].EventQuestChapterId != 0:
			classes["event-chapter"][option] = true
		case quests.QuestById[option].QuestId != 0:
			classes["quest-id"][option] = true
		default:
			classes["unknown"][option] = true
		}
	}
	for _, class := range []string{"main-or-event-type", "event-chapter", "quest-id", "unknown"} {
		t.Logf("%s=%s", class, summarizeIds(classes[class]))
	}
	for _, option := range []int32{4, 5, 6, 8, 9, 10, 11, 12} {
		var samples []string
		for _, mission := range missions.OrderedMissions {
			if mission.MissionClearConditionType == 1 && mission.MissionClearConditionOptionGroupId == option && len(samples) < 12 {
				samples = append(samples, fmt.Sprintf("id=%d/value=%d/link=%d/text=%d", mission.MissionId, mission.ClearConditionValue, mission.MissionLinkId, mission.NameMissionTextId))
			}
		}
		t.Logf("option=%d eventTypeChapters=%d missions=%s", option, countEventChaptersOfType(quests, option), strings.Join(samples, " "))
	}
}

func countEventChaptersOfType(quests *masterdata.QuestCatalog, eventType int32) int {
	var count int
	for _, candidate := range quests.EventQuestTypeByChapterId {
		if candidate == eventType {
			count++
		}
	}
	return count
}

func summarizeIds(ids map[int32]bool) string {
	delete(ids, 0)
	values := make([]int, 0, len(ids))
	for id := range ids {
		values = append(values, int(id))
	}
	sort.Ints(values)
	parts := make([]string, 0, min(len(values), 21))
	for i, value := range values {
		if len(values) > 20 && i >= 10 && i < len(values)-10 {
			if i == 10 {
				parts = append(parts, fmt.Sprintf("...(%d total)...", len(values)))
			}
			continue
		}
		parts = append(parts, fmt.Sprint(value))
	}
	return "[" + strings.Join(parts, ",") + "]"
}
