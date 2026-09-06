package questflow

import (
	"path/filepath"
	"slices"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestSecretStoryHiddenQuestMission(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	resolver, err := masterdata.LoadConditionResolver()
	if err != nil {
		t.Fatal(err)
	}
	parts, err := masterdata.LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := masterdata.LoadQuestCatalog(parts, resolver)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name                                  string
		questId, characterId                  int32
		skip, retired, annihilated, wantClear bool
	}{
		{name: "Gayle on 10F", questId: 210010, characterId: 1009, wantClear: true},
		{name: "wrong character despite power bonus", questId: 210010, characterId: 1022},
		{name: "wrong floor", questId: 210001, characterId: 1009},
		{name: "skip ticket", questId: 210010, characterId: 1009, skip: true},
		{name: "retreat", questId: 210010, characterId: 1009, retired: true},
		{name: "defeat", questId: 210010, characterId: 1009, annihilated: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := questMissionPowerBonusTestHandler(true, 0)
			// Keep rewards synthetic while loading the actual hidden mission
			// mappings and definitions used by the screenshot's condition.
			quest := h.QuestById[10]
			quest.QuestId = tt.questId
			h.QuestById[tt.questId] = quest
			h.MissionIdsByQuestId[tt.questId] = h.MissionIdsByQuestId[10]
			h.InvisibleMissionIdsByQuestId = catalog.InvisibleMissionIdsByQuestId
			for _, id := range catalog.InvisibleMissionIdsByQuestId[tt.questId] {
				h.MissionById[id] = catalog.MissionById[id]
			}
			h.CostumeById = map[int32]masterdata.EntityMCostume{900: {CostumeId: 900, CharacterId: tt.characterId}}
			h.WeaponById = map[int32]masterdata.EntityMWeapon{901: {WeaponId: 901}}
			user := questMissionPowerBonusTestUser(model.DeckTypeQuest, 130000)
			user.Quests[tt.questId] = store.UserQuestState{QuestId: tt.questId, UserDeckNumber: 2, QuestStateType: model.UserQuestStateTypeCleared, ClearCount: 1, IsRewardGranted: true, LatestStartDatetime: 10}
			deckKey := store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: 2}
			deck := user.Decks[deckKey]
			deck.UserDeckCharacterUuid01 = "dc"
			user.Decks[deckKey] = deck
			user.DeckCharacters["dc"] = store.DeckCharacterState{UserCostumeUuid: "costume", MainUserWeaponUuid: "weapon"}
			user.Costumes["costume"] = store.CostumeState{CostumeId: 900}
			user.Weapons["weapon"] = store.WeaponState{WeaponId: 901}

			if resolver.Satisfied(511901, user) {
				t.Fatal("quest history alone satisfied hidden mission")
			}
			if tt.skip {
				if _, err := h.applyQuestSkip(user, tt.questId, 2, 1, h.targetForEvent(210, tt.questId), 12); err != nil {
					t.Fatal(err)
				}
			} else {
				h.HandleEventQuestFinish(user, 210, tt.questId, tt.retired, tt.annihilated, 12)
			}
			if got := resolver.Satisfied(511901, user); got != tt.wantClear {
				t.Fatalf("Gayle's Secret Story unlock = %v, want %v", got, tt.wantClear)
			}
			if tt.wantClear {
				if resolver.Satisfied(5119, user) {
					t.Fatal("Secret Story unlocked without the weapon limit-break condition")
				}
				user.Missions[500018] = store.UserMissionState{MissionId: 500018, MissionProgressStatusType: int32(model.MissionProgressStatusTypeClear)}
				if !resolver.Satisfied(5119, user) {
					t.Fatal("Secret Story stayed locked after both conditions cleared")
				}
				state := user.QuestMissions[store.QuestMissionKey{QuestId: 210010, QuestMissionId: 50005}]
				if state.ProgressValue != 1 || state.LatestClearDatetime != 12 || state.LatestVersion != 12 {
					t.Fatalf("hidden mission state = %+v", state)
				}
				if count := h.ClearedQuestMissionCount(user, []int32{210010}); count != 2 {
					t.Fatalf("hidden missions changed visible star count: %d", count)
				}
				h.HandleEventQuestFinish(user, 210, tt.questId, false, false, 13)
				if user.QuestMissions[store.QuestMissionKey{QuestId: 210010, QuestMissionId: 50005}] != state {
					t.Fatal("repeat clear rewrote completed hidden mission")
				}
				if user.ConsumableItems[100] != 2 || user.ConsumableItems[101] != 3 {
					t.Fatal("hidden/repeated clear changed mission rewards")
				}
			}
		})
	}
}

func TestHiddenBattleMissionsRequireCurrentResults(t *testing.T) {
	for _, tt := range []struct {
		name                     string
		valid                    bool
		finishedAt, hp, recovery int64
		weaponSkills             int32
		want                     []int32
	}{
		{name: "missing results"},
		{name: "stale results", valid: true, finishedAt: 9, hp: 100},
		{name: "full HP without healing or weapon skills", valid: true, finishedAt: 11, hp: 100, want: []int32{50006, 50008, 50010}},
		{name: "damaged", valid: true, finishedAt: 11, hp: 99, want: []int32{50008, 50010}},
		{name: "healed", valid: true, finishedAt: 11, hp: 100, recovery: 1, want: []int32{50006, 50010}},
		{name: "used weapon skill", valid: true, finishedAt: 11, hp: 100, weaponSkills: 1, want: []int32{50006, 50008}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := questMissionPowerBonusTestHandler(true, 0)
			h.InvisibleMissionIdsByQuestId = map[int32][]int32{10: {50006, 50008, 50010}}
			h.MissionById[50006] = masterdata.EntityMQuestMission{QuestMissionId: 50006, QuestMissionConditionType: int32(model.QuestMissionConditionTypeMinHpPercentageGe), ConditionValue: 100}
			h.MissionById[50008] = masterdata.EntityMQuestMission{QuestMissionId: 50008, QuestMissionConditionType: int32(model.QuestMissionConditionTypeWithoutRecoverySkill)}
			h.MissionById[50010] = masterdata.EntityMQuestMission{QuestMissionId: 50010, QuestMissionConditionType: int32(model.QuestMissionConditionTypeLessThanOrEqualXWeaponSkillUseCount)}
			user := questMissionPowerBonusTestUser(model.DeckTypeQuest, 130000)
			user.Battle.LastFinishedAt = tt.finishedAt
			user.Battle.MissionDetail = store.BattleMissionDetailState{
				IsValid: tt.valid, TotalRecoverPoint: tt.recovery, WeaponSkillUseCount: tt.weaponSkills,
				CostumeResultCount: 1, CostumeResults: [3]store.CostumeBattleResultState{{MaxHp: 100, RemainingHp: tt.hp}},
			}
			outcome := h.HandleQuestFinish(user, 10, false, false, 12)
			if !outcome.IsBigWin {
				t.Fatal("hidden missions prevented visible mission completion")
			}
			for _, id := range h.InvisibleMissionIdsByQuestId[10] {
				if user.QuestMissions[store.QuestMissionKey{QuestId: 10, QuestMissionId: id}].IsClear != slices.Contains(tt.want, id) {
					t.Errorf("hidden mission %d clear state did not match battle results", id)
				}
			}
		})
	}
}
