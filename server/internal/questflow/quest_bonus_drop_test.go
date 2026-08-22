package questflow

import (
	"math"
	"path/filepath"
	"reflect"
	"testing"

	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/importantitem"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestImportantItemEffectsModifyMatchingBattleDrops(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatalf("initialize master data: %v", err)
	}
	effects, err := importantitem.Load()
	if err != nil {
		t.Fatalf("load important-item effects: %v", err)
	}

	const (
		questId = int32(10)
		groupId = int32(20)
	)
	catalog := &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{
			questId: {QuestId: questId, QuestPickupRewardGroupId: groupId},
		},
		BattleDropsByQuestId: map[int32][]masterdata.BattleDropInfo{
			questId: {{QuestSceneId: 101, BattleDropCategoryId: 1}},
		},
		PickupRewardIdsByGroupId: map[int32][]int32{
			groupId: {1001},
		},
		PickupRewardIdsByGroupAndEffectId: map[int32]map[int32][]int32{
			groupId: {1: {1001}},
		},
		BattleDropEffectIdByRewardId: map[int32]int32{1001: 1},
		BattleDropRewardById: map[int32]masterdata.EntityMBattleDropReward{
			1001: {
				BattleDropRewardId: 1001,
				PossessionType:     int32(model.PossessionTypeConsumableItem),
				PossessionId:       1008,
				Count:              2,
			},
		},
		QuestBonusById: map[int32]masterdata.EntityMQuestBonus{},
	}
	h := &QuestHandler{QuestCatalog: catalog, ImportantItemEffects: effects}
	const nowMillis = int64(1787241600000)
	user := &store.UserState{
		UserId:         99,
		ImportantItems: map[int32]int32{200001: 1},
		Quests: map[int32]store.UserQuestState{
			questId: {QuestId: questId, LatestStartDatetime: 123456789},
		},
	}

	matching := h.computeDropRewards(
		user,
		catalog.QuestById[questId],
		campaign.QuestTarget{QuestType: campaign.QuestTypeMainQuest, ChapterId: 2},
		nowMillis,
	)
	if len(matching) != 1 || matching[0].Count != 3 {
		t.Fatalf("matching important-item drop = %+v, want count 3", matching)
	}

	nonMatching := h.computeDropRewards(
		user,
		catalog.QuestById[questId],
		campaign.QuestTarget{QuestType: campaign.QuestTypeMainQuest, ChapterId: 1},
		nowMillis,
	)
	if len(nonMatching) != 1 || nonMatching[0].Count != 2 {
		t.Fatalf("non-matching important-item drop = %+v, want count 2", nonMatching)
	}
}

func TestBattleDropPlanMatchesFinishRewards(t *testing.T) {
	const (
		questId = int32(10)
		groupId = int32(20)
	)
	catalog := &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{
			questId: {QuestId: questId, QuestPickupRewardGroupId: groupId},
		},
		BattleDropsByQuestId: map[int32][]masterdata.BattleDropInfo{
			questId: {
				{QuestSceneId: 101, BattleDropCategoryId: 1},
				{QuestSceneId: 102, BattleDropCategoryId: 2},
				{QuestSceneId: 103, BattleDropCategoryId: 3},
			},
		},
		PickupRewardIdsByGroupId: map[int32][]int32{
			groupId: {1001, 1002, 2001},
		},
		PickupRewardIdsByGroupAndEffectId: map[int32]map[int32][]int32{
			groupId: {
				1: {1001, 1002},
				3: {2001},
			},
		},
		BattleDropEffectIdByRewardId: map[int32]int32{
			1001: 1,
			1002: 1,
			2001: 3,
		},
		BattleDropRewardById: map[int32]masterdata.EntityMBattleDropReward{
			1001: {BattleDropRewardId: 1001, PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 501, Count: 1},
			1002: {BattleDropRewardId: 1002, PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 502, Count: 2},
			2001: {BattleDropRewardId: 2001, PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 601, Count: 1},
		},
		QuestBonusById: map[int32]masterdata.EntityMQuestBonus{},
	}
	h := &QuestHandler{QuestCatalog: catalog}
	user := &store.UserState{
		UserId: 99,
		Quests: map[int32]store.UserQuestState{
			questId: {QuestId: questId, LatestStartDatetime: 123456789},
		},
	}

	plan := h.BattleDropRewards(user, questId)
	if repeated := h.BattleDropRewards(user, questId); !reflect.DeepEqual(repeated, plan) {
		t.Fatalf("battle drop plan changed between calls: first=%v second=%v", plan, repeated)
	}
	if len(plan) != 3 {
		t.Fatalf("battle drop plan has %d entries, want one for each of 3 candidates", len(plan))
	}

	drops := h.computeDropRewards(user, catalog.QuestById[questId], campaign.QuestTarget{}, 123456790)
	if len(drops) != len(plan) {
		t.Fatalf("finish drops = %v, want %d planned drops", drops, len(plan))
	}
	for i, planned := range plan {
		definition := catalog.BattleDropRewardById[planned.BattleDropRewardId]
		got := drops[i]
		if got.RewardEffectId != planned.BattleDropEffectId ||
			got.PossessionType != model.PossessionType(definition.PossessionType) ||
			got.PossessionId != definition.PossessionId || got.Count != definition.Count {
			t.Fatalf("drop %d = %+v, does not match plan %+v / definition %+v", i, got, planned, definition)
		}
	}
}

func TestBattleDropSelectsUniformlyWithinRevealedRarity(t *testing.T) {
	const (
		questId = int32(10)
		groupId = int32(20)
	)
	h := &QuestHandler{QuestCatalog: &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{
			questId: {QuestId: questId, QuestPickupRewardGroupId: groupId},
		},
		BattleDropsByQuestId: map[int32][]masterdata.BattleDropInfo{
			questId: {{QuestSceneId: 101, BattleDropCategoryId: 1}},
		},
		PickupRewardIdsByGroupId: map[int32][]int32{
			groupId: {1001, 1002, 2001},
		},
		PickupRewardIdsByGroupAndEffectId: map[int32]map[int32][]int32{
			groupId: {1: {1001, 1002}, 3: {2001}},
		},
		BattleDropEffectIdByRewardId: map[int32]int32{1001: 1, 1002: 1, 2001: 3},
	}}
	user := &store.UserState{UserId: 99}
	counts := map[int32]int{}
	for seed := int64(1); seed <= 10_000; seed++ {
		plan := h.battleDropPlan(user, questId, seed)
		if len(plan) == 1 && plan[0].BattleDropEffectId == 1 {
			counts[plan[0].BattleDropRewardId]++
		}
	}
	normalTotal := counts[1001] + counts[1002]
	if normalTotal == 0 {
		t.Fatal("normal-rarity subset was never selected")
	}
	imbalance := math.Abs(float64(counts[1001]-counts[1002])) / float64(normalTotal)
	if imbalance > 0.08 {
		t.Fatalf("normal-rarity selections are too imbalanced: %v", counts)
	}
}

func TestOriginalQuestBonusUsesBestEquippedWeaponTier(t *testing.T) {
	const questId = int32(10)
	catalog := &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{
			questId: {QuestId: questId, QuestBonusId: 30},
		},
		QuestBonusById: map[int32]masterdata.EntityMQuestBonus{
			30: {QuestBonusId: 30, QuestBonusCharacterGroupId: 41, QuestBonusWeaponGroupId: 40},
		},
		QuestBonusWeaponRowsByGroupId: map[int32][]masterdata.EntityMQuestBonusWeaponGroup{
			40: {
				{QuestBonusWeaponGroupId: 40, WeaponId: 100, LimitBreakCountLowerLimit: 0, QuestBonusEffectGroupId: 50},
				{QuestBonusWeaponGroupId: 40, WeaponId: 100, LimitBreakCountLowerLimit: 2, QuestBonusEffectGroupId: 51},
			},
		},
		QuestBonusCharacterRowsByGroupId: map[int32][]masterdata.EntityMQuestBonusCharacterGroup{
			41: {{QuestBonusCharacterGroupId: 41, CharacterId: 7, QuestBonusEffectGroupId: 60}},
		},
		QuestBonusEffectsByGroupId: map[int32][]masterdata.EntityMQuestBonusEffectGroup{
			50: {{QuestBonusEffectGroupId: 50, QuestBonusType: questBonusTypeDropReward, QuestBonusEffectId: 500}},
			51: {{QuestBonusEffectGroupId: 51, QuestBonusType: questBonusTypeDropReward, QuestBonusEffectId: 501}},
			60: {{QuestBonusEffectGroupId: 60, QuestBonusType: questBonusTypeExp, QuestBonusEffectId: 600}},
		},
		QuestBonusDropByEffectId: map[int32]masterdata.EntityMQuestBonusDropReward{
			500: {QuestBonusEffectId: 500, PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 900, AdditionalCount: 1},
			501: {QuestBonusEffectId: 501, PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 900, AdditionalCount: 4},
		},
		QuestBonusExpByEffectId: map[int32]masterdata.EntityMQuestBonusExp{
			600: {QuestBonusEffectId: 600, ExpType: questBonusExpCostume, BonusValuePermil: 500},
		},
		WeaponEvolutionByWeaponId: map[int32]masterdata.WeaponEvolutionInfo{
			100: {GroupId: 1, Order: 1},
			101: {GroupId: 1, Order: 2},
		},
		CostumeById: map[int32]masterdata.EntityMCostume{
			200: {CostumeId: 200, CharacterId: 7},
		},
	}
	h := &QuestHandler{QuestCatalog: catalog}
	user := &store.UserState{
		Quests: map[int32]store.UserQuestState{
			questId: {QuestId: questId, UserDeckNumber: 1},
		},
		Decks: map[store.DeckKey]store.DeckState{
			{DeckType: model.DeckTypeQuest, UserDeckNumber: 1}: {
				DeckType: model.DeckTypeQuest, UserDeckNumber: 1, UserDeckCharacterUuid01: "deck-character",
			},
		},
		DeckCharacters: map[string]store.DeckCharacterState{
			"deck-character": {UserDeckCharacterUuid: "deck-character", UserCostumeUuid: "costume", MainUserWeaponUuid: "evolved"},
		},
		DeckSubWeapons: map[string][]string{
			"deck-character": {"evolved", "base"},
		},
		Weapons: map[string]store.WeaponState{
			"evolved": {UserWeaponUuid: "evolved", WeaponId: 101, LimitBreakCount: 2},
			"base":    {UserWeaponUuid: "base", WeaponId: 100, LimitBreakCount: 0},
		},
		Costumes: map[string]store.CostumeState{
			"costume": {UserCostumeUuid: "costume", CostumeId: 200},
		},
	}

	drops := h.questBonusDropRewards(user, catalog.QuestById[questId], 1)
	wantDrops := []RewardGrant{{
		PossessionType: model.PossessionTypeMaterial,
		PossessionId:   900,
		Count:          5,
		RewardEffectId: questBonusRewardEffectId,
	}}
	if !reflect.DeepEqual(drops, wantDrops) {
		t.Fatalf("quest bonus drops = %+v, want %+v", drops, wantDrops)
	}

	characterBonus, costumeBonus := h.questBonusExpPermilByCostume(user, catalog.QuestById[questId], 1)
	if len(characterBonus) != 0 || !reflect.DeepEqual(costumeBonus, map[string]int32{"costume": 500}) {
		t.Fatalf("quest bonus exp = character %v costume %v", characterBonus, costumeBonus)
	}
}
