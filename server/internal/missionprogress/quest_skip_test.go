package missionprogress

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/store"
)

func TestSecretStoryCharacterQuestClearsIncludeSkipTickets(t *testing.T) {
	resolver := loadConditionResolver(t)
	missions, err := masterdata.LoadMissionCatalog()
	if err != nil {
		t.Fatal(err)
	}
	mission := missions.MissionById[500015]
	// Use Gayle's actual 100-clear condition, independent of feature unlocks and scheduling.
	mission.MissionUnlockConditionId, mission.MissionTermId = 0, 0

	for _, tt := range []struct {
		name                       string
		mode                       string
		previousDeck, selectedDeck int32
		count, wantProgress        int32
		restricted                 bool
	}{
		{name: "normal clear", mode: "normal", previousDeck: 2, count: 1, wantProgress: 95},
		{name: "single skip", previousDeck: 1, selectedDeck: 2, count: 1, wantProgress: 95},
		{name: "multiple skips", previousDeck: 1, selectedDeck: 2, count: 6, wantProgress: 100},
		{name: "bulk skips", mode: "bulk", previousDeck: 1, selectedDeck: 2, count: 6, wantProgress: 100},
		{name: "wrong selected character", previousDeck: 2, selectedDeck: 1, count: 6, wantProgress: 94},
		{name: "bulk wrong selected character", mode: "bulk", previousDeck: 2, selectedDeck: 1, count: 6, wantProgress: 94},
		{name: "omitted deck uses previous deck", previousDeck: 2, count: 6, wantProgress: 100},
		{name: "missing selected deck", previousDeck: 2, selectedDeck: 3, count: 6, wantProgress: 94},
		{name: "restricted normal clear", mode: "normal", previousDeck: 2, count: 1, wantProgress: 95, restricted: true},
		{name: "restricted skip", previousDeck: 1, selectedDeck: 2, count: 6, wantProgress: 100, restricted: true},
		{name: "restricted bulk skips", mode: "bulk", previousDeck: 1, selectedDeck: 2, count: 6, wantProgress: 100, restricted: true},
		{name: "restricted wrong character", previousDeck: 2, selectedDeck: 1, count: 6, wantProgress: 94, restricted: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			catalogs := testCatalog(mission,
				masterdata.EntityMMission{MissionId: 1, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCount), ClearConditionValue: 1000},
				masterdata.EntityMMission{MissionId: 2, MissionClearConditionType: int32(model.MissionClearConditionTypeQuestClearByCountWithoutSkip), ClearConditionValue: 1000},
			)
			catalogs.Quest = &masterdata.QuestCatalog{
				QuestById: map[int32]masterdata.EntityMQuest{
					10: {QuestId: 10, IsUsableSkipTicket: true},
					11: {QuestId: 11, IsUsableSkipTicket: true},
				},
				CostumeById: map[int32]masterdata.EntityMCostume{
					900: {CostumeId: 900, CharacterId: 1009},
					901: {CostumeId: 901, CharacterId: 1013},
				},
			}
			h := &questflow.QuestHandler{
				QuestCatalog: catalogs.Quest,
				Config:       &masterdata.GameConfig{ConsumableItemIdForQuestSkipTicket: 7},
				Granter:      &store.PossessionGranter{},
			}
			user := &store.UserState{}
			user.EnsureMaps()
			user.ConsumableItems[7] = 10
			for _, id := range []int32{10, 11} {
				user.Quests[id] = store.UserQuestState{
					QuestId: id, UserDeckNumber: tt.previousDeck, QuestStateType: model.UserQuestStateTypeCleared,
					ClearCount: 20, IsRewardGranted: true,
				}
			}
			user.Decks[store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: 1}] = store.DeckState{UserDeckCharacterUuid01: "wrong"}
			user.Decks[store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: 2}] = store.DeckState{UserDeckCharacterUuid01: "gayle"}
			if tt.restricted {
				for _, id := range []int32{10, 11} {
					quest := catalogs.Quest.QuestById[id]
					quest.QuestDeckRestrictionGroupId = 101
					catalogs.Quest.QuestById[id] = quest
				}
				for number := int32(1); number <= 2; number++ {
					user.Decks[store.DeckKey{DeckType: model.DeckTypeRestrictedQuest, UserDeckNumber: number}] = user.Decks[store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: number}]
				}
				// Ordinary decks deliberately contain the opposite character.
				user.Decks[store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: 1}] = store.DeckState{UserDeckCharacterUuid01: "gayle"}
				user.Decks[store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: 2}] = store.DeckState{UserDeckCharacterUuid01: "wrong"}
			}
			user.DeckCharacters["wrong"] = store.DeckCharacterState{UserCostumeUuid: "wrong-costume"}
			user.DeckCharacters["gayle"] = store.DeckCharacterState{UserCostumeUuid: "gayle-costume"}
			user.Costumes["wrong-costume"] = store.CostumeState{CostumeId: 901}
			user.Costumes["gayle-costume"] = store.CostumeState{CostumeId: 900}
			user.Missions[mission.MissionId] = store.UserMissionState{
				MissionId: mission.MissionId, ProgressValue: 94, MissionProgressStatusType: int32(model.MissionProgressStatusTypeInProgress),
			}
			Sync(catalogs, user, 100)
			before := store.CloneUserState(*user)

			var err error
			switch tt.mode {
			case "normal":
				h.HandleQuestFinish(user, 10, false, false, 200)
			case "bulk":
				_, err = h.HandleQuestSkipBulk(user, []int32{10, 11},
					[]int32{int32(model.QuestTypeMain), int32(model.QuestTypeEvent)}, []int32{0, 1},
					[]int32{2, tt.count - 2}, tt.selectedDeck, 200)
			default:
				_, err = h.HandleQuestSkip(user, 10, int32(model.QuestTypeMain), 0, tt.selectedDeck, tt.count, 200)
			}
			if err != nil {
				t.Fatal(err)
			}
			events := user.PendingMissionEvents
			user.PendingMissionEvents = nil
			Apply(catalogs, &before, user, events, 200)

			state := user.Missions[mission.MissionId]
			if state.ProgressValue != tt.wantProgress {
				t.Errorf("Secret Story progress = %d, want %d", state.ProgressValue, tt.wantProgress)
			}
			wantClear := tt.wantProgress >= mission.ClearConditionValue
			if got := resolver.Satisfied(511402, user); got != wantClear {
				t.Errorf("Secret Story character condition = %v, want %v", got, wantClear)
			}
			if wantClear && state.ClearDatetime != 200 {
				t.Errorf("clear datetime = %d, want 200", state.ClearDatetime)
			}
			if got := user.Missions[1].ProgressValue; got != 40+tt.count {
				t.Errorf("ordinary clear count = %d, want %d", got, 40+tt.count)
			}
			var wantWithoutSkip int32
			if tt.mode == "normal" {
				wantWithoutSkip = 1
			}
			if got := user.Missions[2].ProgressValue; got != wantWithoutSkip {
				t.Errorf("without-skip clear count = %d, want %d", got, wantWithoutSkip)
			}
			Sync(catalogs, user, 300)
			if got := user.Missions[mission.MissionId]; got != state {
				t.Errorf("refresh changed character mission: %+v -> %+v", state, got)
			}
		})
	}
}
