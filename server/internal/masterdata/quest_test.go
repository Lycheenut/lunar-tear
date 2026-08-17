package masterdata

import (
	"path/filepath"
	"reflect"
	"testing"

	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
)

func TestBattleDropEffectIdUsesRewardRarity(t *testing.T) {
	materialRarity := map[int32]int32{101: model.RarityNormal, 102: model.RarityRare, 103: model.RaritySRare}
	partsById := map[int32]EntityMParts{201: {PartsId: 201, RarityType: model.RarityRare}}
	consumableType := map[int32]int32{301: consumableItemTypeGachaTicket, 302: 100}
	tests := []struct {
		name   string
		reward EntityMBattleDropReward
		want   int32
	}{
		{name: "normal material", reward: EntityMBattleDropReward{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 101}, want: battleDropEffectNormal},
		{name: "rare material", reward: EntityMBattleDropReward{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 102}, want: battleDropEffectRare},
		{name: "high material", reward: EntityMBattleDropReward{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 103}, want: battleDropEffectHigh},
		{name: "rare parts", reward: EntityMBattleDropReward{PossessionType: int32(model.PossessionTypeParts), PossessionId: 201}, want: battleDropEffectRare},
		{name: "ticket", reward: EntityMBattleDropReward{PossessionType: int32(model.PossessionTypeConsumableItem), PossessionId: 301}, want: battleDropEffectHigh},
		{name: "currency", reward: EntityMBattleDropReward{PossessionType: int32(model.PossessionTypeConsumableItem), PossessionId: 302}, want: battleDropEffectNormal},
		{name: "gem", reward: EntityMBattleDropReward{PossessionType: int32(model.PossessionTypeFreeGem), PossessionId: 1}, want: battleDropEffectHigh},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := battleDropEffectId(test.reward, materialRarity, partsById, consumableType); got != test.want {
				t.Fatalf("battleDropEffectId() = %d, want %d", got, test.want)
			}
		})
	}
}

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
	questsByChapter, questsBySortOrder, questsByDifficulty := buildEventQuestIndexes(chapters, groups, sequences)
	if want := []int32{100, 200}; !reflect.DeepEqual(questsByChapter[10], want) {
		t.Fatalf("chapter quests = %v, want %v", questsByChapter[10], want)
	}
	if want := []int32{100}; !reflect.DeepEqual(questsByChapter[11], want) {
		t.Fatalf("shared chapter quests = %v, want %v", questsByChapter[11], want)
	}
	if want := []int32{200}; !reflect.DeepEqual(questsBySortOrder[10][2], want) {
		t.Fatalf("sort-order quests = %v, want %v", questsBySortOrder[10][2], want)
	}
	if want := []int32{100, 200}; !reflect.DeepEqual(questsByDifficulty[10][0], want) {
		t.Fatalf("difficulty quests = %v, want %v", questsByDifficulty[10][0], want)
	}
}

func TestBuildBossCountByQuestIdUsesEnemyTypes(t *testing.T) {
	scenes := []EntityMQuestScene{{QuestSceneId: 1, QuestId: 10}, {QuestSceneId: 2, QuestId: 20}}
	sceneBattles := []EntityMQuestSceneBattle{{QuestSceneId: 1, BattleGroupId: 100}, {QuestSceneId: 2, BattleGroupId: 200}}
	battleGroups := []EntityMBattleGroup{
		{BattleGroupId: 100, WaveNumber: 1, BattleId: 1000},
		{BattleGroupId: 100, WaveNumber: 2, BattleId: 1001},
		{BattleGroupId: 200, WaveNumber: 1, BattleId: 2000},
	}
	battles := []EntityMBattle{
		{BattleId: 1000, BattleNpcId: 1, DeckType: 1, BattleNpcDeckNumber: 1},
		{BattleId: 1001, BattleNpcId: 2, DeckType: 1, BattleNpcDeckNumber: 1},
		{BattleId: 2000, BattleNpcId: 3, DeckType: 1, BattleNpcDeckNumber: 1},
	}
	decks := []EntityMBattleNpcDeck{
		{BattleNpcId: 1, DeckType: 1, BattleNpcDeckNumber: 1, BattleNpcDeckCharacterUuid01: "normal", BattleNpcDeckCharacterUuid02: "boss-a"},
		{BattleNpcId: 2, DeckType: 1, BattleNpcDeckNumber: 1, BattleNpcDeckCharacterUuid01: "boss-b", BattleNpcDeckCharacterUuid02: "boss-c"},
		{BattleNpcId: 3, DeckType: 1, BattleNpcDeckNumber: 1, BattleNpcDeckCharacterUuid01: "normal-only"},
	}
	types := []EntityMBattleNpcDeckCharacterType{
		{BattleNpcId: 1, BattleNpcDeckCharacterUuid: "normal", BattleEnemyType: 1},
		{BattleNpcId: 1, BattleNpcDeckCharacterUuid: "boss-a", BattleEnemyType: 2},
		{BattleNpcId: 2, BattleNpcDeckCharacterUuid: "boss-b", BattleEnemyType: 2},
		{BattleNpcId: 2, BattleNpcDeckCharacterUuid: "boss-c", BattleEnemyType: 2},
		{BattleNpcId: 3, BattleNpcDeckCharacterUuid: "normal-only", BattleEnemyType: 1},
	}

	got := buildBossCountByQuestId(scenes, sceneBattles, battleGroups, battles, decks, types)
	if got[10] != 3 {
		t.Fatalf("quest 10 boss count = %d, want 3", got[10])
	}
	if got[20] != 0 {
		t.Fatalf("quest 20 boss count = %d, want 0", got[20])
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
	for groupId, rewardIds := range catalog.PickupRewardIdsByGroupId {
		classifiedCount := 0
		for effectId, subset := range catalog.PickupRewardIdsByGroupAndEffectId[groupId] {
			if effectId < battleDropEffectNormal || effectId > battleDropEffectHigh {
				t.Fatalf("pickup reward group %d has invalid drop effect %d", groupId, effectId)
			}
			classifiedCount += len(subset)
		}
		if classifiedCount != len(rewardIds) {
			t.Fatalf("pickup reward group %d classified %d of %d rewards", groupId, classifiedCount, len(rewardIds))
		}
	}
	for questId, wantDifficulty := range map[int32]int32{2: 1, 10001: 2, 20001: 3} {
		if got := catalog.MainQuestDifficultyTypeByQuestId[questId]; got != wantDifficulty {
			t.Fatalf("main quest %d difficulty = %d, want %d", questId, got, wantDifficulty)
		}
		if catalog.RouteIdByQuestId[questId] == 0 || catalog.MainQuestChapterIdByQuestId[questId] == 0 {
			t.Fatalf("main quest %d is missing its route or chapter", questId)
		}
		chapterId := catalog.MainQuestChapterIdByQuestId[questId]
		if got := catalog.MainQuestRouteIdByChapterId[chapterId]; got != catalog.RouteIdByQuestId[questId] {
			t.Fatalf("main quest chapter %d route = %d, want %d", chapterId, got, catalog.RouteIdByQuestId[questId])
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
