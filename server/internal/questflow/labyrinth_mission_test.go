package questflow

import (
	"path/filepath"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestLabyrinthMissionCountsFollowRequiredWeaponPreference(t *testing.T) {
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
	labyrinth := masterdata.LoadLabyrinthCatalog()
	if len(labyrinth.ChaptersByOrder) == 0 {
		t.Fatal("no labyrinth chapters loaded")
	}

	for _, tt := range []struct {
		name               string
		requiredPreference bool
		equipPreferred     bool
		wantCount          int32
	}{
		{name: "required preference with preferred weapon", requiredPreference: true, equipPreferred: true, wantCount: 3},
		{name: "required preference with another weapon", requiredPreference: true, wantCount: 3},
		{name: "wrong preference with preferred weapon", equipPreferred: true, wantCount: 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Use real quest/mission definitions with synthetic inventory and rewards.
			h := questMissionPowerBonusTestHandler(false, 0)
			h.QuestById = catalog.QuestById
			h.MissionById = catalog.MissionById
			h.MissionIdsByQuestId = catalog.MissionIdsByQuestId
			user := questMissionPowerBonusTestUser(model.DeckTypeQuest, 1000000)
			user.Costumes["costume"] = store.CostumeState{CostumeId: 900}
			user.Weapons["weapon"] = store.WeaponState{WeaponId: 901}
			user.DeckCharacters["dc"] = store.DeckCharacterState{UserCostumeUuid: "costume", MainUserWeaponUuid: "weapon"}
			key := store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: 2}
			deck := user.Decks[key]
			deck.UserDeckCharacterUuid01 = "dc"
			user.Decks[key] = deck

			for _, chapter := range labyrinth.ChaptersByOrder {
				for _, stage := range chapter.StageOrders {
					questIds, ok := labyrinth.StageQuestIds(catalog, chapter.EventQuestChapterId, stage)
					if !ok {
						t.Fatalf("chapter %d stage %d has no quests", chapter.EventQuestChapterId, stage)
					}
					for _, questId := range questIds {
						// "Clear with a character whose preferred weapon is: {0}"
						// specifies the costume's preference, not its equipped weapon.
						var preference, attribute int32
						for _, id := range catalog.MissionIdsByQuestId[questId] {
							mission := catalog.MissionById[id]
							switch model.QuestMissionConditionType(mission.QuestMissionConditionType) {
							case model.QuestMissionConditionTypeCostumeSkillfulWeaponAnyCharacter:
								preference = mission.ConditionValue
							case model.QuestMissionConditionTypeSpecifiedAttributeMainWeaponAllCharacter:
								attribute = mission.ConditionValue
							}
						}
						if preference == 0 {
							t.Fatalf("quest %d has no preferred-weapon mission", questId)
						}
						if !tt.requiredPreference {
							preference = preference%6 + 1
						}
						weaponType := preference
						if !tt.equipPreferred {
							weaponType = preference%6 + 1
						}
						h.CostumeById = map[int32]masterdata.EntityMCostume{900: {CostumeId: 900, SkillfulWeaponType: preference}}
						h.WeaponById = map[int32]masterdata.EntityMWeapon{901: {WeaponId: 901, WeaponType: weaponType, AttributeType: attribute}}
						user.Quests[questId] = store.UserQuestState{QuestId: questId, UserDeckNumber: 2, QuestStateType: model.UserQuestStateTypeActive, LatestStartDatetime: 10}
						user.Battle.LastFinishedAt = 11
						user.Battle.LastUserPartyCount = 1
						user.Battle.MissionDetail = store.BattleMissionDetailState{
							IsValid: true, MaxDamage: 1000000, CostumeResultCount: 1,
							CostumeResults: [3]store.CostumeBattleResultState{{IsAlive: true, MaxHp: 100, RemainingHp: 80}},
						}
						outcome := h.HandleEventQuestFinish(user, chapter.EventQuestChapterId, questId, false, false, 12)
						if got := len(outcome.ClearedQuestMissionIds); got != int(tt.wantCount) {
							t.Errorf("quest %d newly cleared missions = %v, want %d", questId, outcome.ClearedQuestMissionIds, tt.wantCount)
						}
						if got := h.ClearedQuestMissionCount(user, []int32{questId}); got != tt.wantCount {
							t.Errorf("quest %d stored mission count = %d, want %d", questId, got, tt.wantCount)
						}
					}
					if got, want := h.ClearedQuestMissionCount(user, questIds), int32(len(questIds))*tt.wantCount; got != want {
						t.Errorf("chapter %d stage %d mission count = %d, want %d", chapter.EventQuestChapterId, stage, got, want)
					}
				}
			}
		})
	}
}
