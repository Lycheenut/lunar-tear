package masterdata

import (
	"path/filepath"
	"reflect"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
)

func TestBuildEventQuestIndexesUsesSequenceSortOrder(t *testing.T) {
	chapters := []EntityMEventQuestChapter{
		{EventQuestChapterId: 10, EventQuestSequenceGroupId: 20},
		{EventQuestChapterId: 11, EventQuestSequenceGroupId: 21},
	}
	groups := []EntityMEventQuestSequenceGroup{
		{EventQuestSequenceGroupId: 20, EventQuestSequenceId: 30},
		{EventQuestSequenceGroupId: 21, EventQuestSequenceId: 31},
	}
	sequences := []EntityMEventQuestSequence{
		{EventQuestSequenceId: 30, SortOrder: 2, QuestId: 200},
		{EventQuestSequenceId: 30, SortOrder: 1, QuestId: 100},
		{EventQuestSequenceId: 31, SortOrder: 1, QuestId: 100},
	}
	questsByChapter, questsBySortOrder := buildEventQuestIndexes(chapters, groups, sequences)
	if want := []int32{100, 200}; !reflect.DeepEqual(questsByChapter[10], want) {
		t.Fatalf("chapter quests = %v, want %v", questsByChapter[10], want)
	}
	if want := []int32{100}; !reflect.DeepEqual(questsByChapter[11], want) {
		t.Fatalf("shared chapter quests = %v, want %v", questsByChapter[11], want)
	}
	if want := []int32{200}; !reflect.DeepEqual(questsBySortOrder[10][2], want) {
		t.Fatalf("sort-order quests = %v, want %v", questsBySortOrder[10][2], want)
	}
}

func TestLoadQuestCatalogResolvesEventUnlockQuests(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	parts, err := LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := LoadConditionResolver()
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadQuestCatalog(parts, resolver)
	if err != nil {
		t.Fatal(err)
	}
	for questId, wantDifficulty := range map[int32]int32{2: 1, 10001: 2, 20001: 3} {
		if got := catalog.MainQuestDifficultyTypeByQuestId[questId]; got != wantDifficulty {
			t.Fatalf("main quest %d difficulty = %d, want %d", questId, got, wantDifficulty)
		}
		if catalog.RouteIdByQuestId[questId] == 0 || catalog.MainQuestChapterIdByQuestId[questId] == 0 {
			t.Fatalf("main quest %d is missing its route or chapter", questId)
		}
	}
	if len(catalog.EventChapterById) == 0 || len(catalog.EventUnlockConditions) == 0 {
		t.Fatal("event chapters or normalized unlock quests were not loaded")
	}
	labyrinth := LoadLabyrinthCatalog()
	for _, chapter := range labyrinth.ChaptersByOrder {
		for _, stageOrder := range chapter.StageOrders {
			questIds, ok := labyrinth.StageQuestIds(catalog, chapter.EventQuestChapterId, stageOrder)
			if !ok {
				t.Fatalf("labyrinth chapter %d stage %d has no quest mapping", chapter.EventQuestChapterId, stageOrder)
			}
			missionCount := 0
			for _, questId := range questIds {
				missionCount += len(catalog.MissionIdsByQuestId[questId])
			}
			tiers := labyrinth.AccumTiersByStage[labyrinthStageKey{chapter.EventQuestChapterId, stageOrder}]
			if len(tiers) > 0 && int(tiers[len(tiers)-1].QuestMissionClearCount) > missionCount {
				t.Fatalf("labyrinth chapter %d stage %d needs %d missions, only %d are mapped", chapter.EventQuestChapterId, stageOrder, tiers[len(tiers)-1].QuestMissionClearCount, missionCount)
			}
		}
	}
}
