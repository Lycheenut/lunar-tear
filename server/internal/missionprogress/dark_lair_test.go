package missionprogress

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/store"
)

func TestSecretStoryDarkLairUsesRestrictedDeck(t *testing.T) {
	resolver := loadConditionResolver(t)
	missions, err := masterdata.LoadMissionCatalog()
	if err != nil {
		t.Fatal(err)
	}
	parts, err := masterdata.LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	quests, err := masterdata.LoadQuestCatalog(parts, resolver)
	if err != nil {
		t.Fatal(err)
	}

	for _, story := range []struct {
		name                                                    string
		missionId, conditionId, chapterId, questId, characterId int32
	}{
		{name: "Akeha", missionId: 500057, conditionId: 5173, chapterId: 904, questId: 120040, characterId: 1015},
		{name: "Argo", missionId: 500085, conditionId: 5205, chapterId: 905, questId: 120050, characterId: 1006},
	} {
		t.Run(story.name, func(t *testing.T) {
			mission := missions.MissionById[story.missionId]
			// Exercise the real Secret Story condition independently of feature unlocks and scheduling.
			mission.MissionUnlockConditionId, mission.MissionTermId = 0, 0
			if quests.QuestById[story.questId].QuestDeckRestrictionGroupId == 0 {
				t.Fatal("Dark Lair must use a restricted deck")
			}
			for _, tt := range []struct {
				name                        string
				restricted, ordinary        []string
				wrongDifficulty, skip       bool
				retired, annihilated, clear bool
			}{
				{name: "selected restricted deck", restricted: []string{"required"}, ordinary: []string{"other"}, clear: true},
				{name: "no ordinary deck", restricted: []string{"required"}, clear: true},
				{name: "wrong restricted character", restricted: []string{"other"}, ordinary: []string{"required"}},
				{name: "not solo", restricted: []string{"required", "other"}, ordinary: []string{"required"}},
				{name: "wrong difficulty", restricted: []string{"required"}, wrongDifficulty: true},
				{name: "skip ticket", restricted: []string{"required"}, skip: true},
				{name: "retreat", restricted: []string{"required"}, retired: true},
				{name: "defeat", restricted: []string{"required"}, annihilated: true},
			} {
				t.Run(tt.name, func(t *testing.T) {
					questId := story.questId
					if tt.wrongDifficulty {
						questId-- // Dark Lair: Normal precedes Hard in each character's sequence.
					}
					catalogs := testCatalog(mission)
					// Retain the real quest restriction while keeping rewards and stamina synthetic.
					catalogs.Quest = &masterdata.QuestCatalog{
						QuestById: map[int32]masterdata.EntityMQuest{questId: {
							QuestId: questId, IsUsableSkipTicket: true,
							QuestDeckRestrictionGroupId: quests.QuestById[questId].QuestDeckRestrictionGroupId,
						}},
						CostumeById: map[int32]masterdata.EntityMCostume{
							900: {CostumeId: 900, CharacterId: story.characterId},
							901: {CostumeId: 901, CharacterId: 1009},
						},
					}
					h := &questflow.QuestHandler{
						QuestCatalog: catalogs.Quest, Granter: &store.PossessionGranter{},
						Config: &masterdata.GameConfig{ConsumableItemIdForQuestSkipTicket: 7},
					}
					user := &store.UserState{}
					user.EnsureMaps()
					user.Quests[questId] = store.UserQuestState{
						QuestId: questId, UserDeckNumber: 2, QuestStateType: model.UserQuestStateTypeCleared,
						ClearCount: 1, IsRewardGranted: true,
					}
					user.ConsumableItems[7] = 1
					user.DeckCharacters["required"] = store.DeckCharacterState{UserCostumeUuid: "required"}
					user.DeckCharacters["other"] = store.DeckCharacterState{UserCostumeUuid: "other"}
					user.Costumes["required"] = store.CostumeState{CostumeId: 900}
					user.Costumes["other"] = store.CostumeState{CostumeId: 901}
					// A different deck number must not supply the mission's character either.
					user.Decks[store.DeckKey{DeckType: model.DeckTypeRestrictedQuest, UserDeckNumber: 1}] = store.DeckState{UserDeckCharacterUuid01: "other"}
					for deckType, characters := range map[model.DeckType][]string{
						model.DeckTypeQuest: tt.ordinary, model.DeckTypeRestrictedQuest: tt.restricted,
					} {
						if len(characters) == 0 {
							continue
						}
						deck := store.DeckState{UserDeckCharacterUuid01: characters[0]}
						if len(characters) > 1 {
							deck.UserDeckCharacterUuid02 = characters[1]
						}
						user.Decks[store.DeckKey{DeckType: deckType, UserDeckNumber: 2}] = deck
					}
					Sync(catalogs, user, 100)
					if resolver.Satisfied(story.conditionId, user) {
						t.Fatal("quest history alone satisfied the solo-character mission")
					}
					before := store.CloneUserState(*user)
					if tt.skip {
						if _, err := h.HandleQuestSkip(user, questId, int32(model.QuestTypeEvent), story.chapterId, 2, 1, 200); err != nil {
							t.Fatal(err)
						}
					} else {
						h.HandleEventQuestFinish(user, story.chapterId, questId, tt.retired, tt.annihilated, 200)
					}
					events := user.PendingMissionEvents
					user.PendingMissionEvents = nil
					Apply(catalogs, &before, user, events, 200)
					if got := resolver.Satisfied(story.conditionId, user); got != tt.clear {
						t.Fatalf("Dark Lair Secret Story condition = %v, want %v; mission = %+v", got, tt.clear, user.Missions[story.missionId])
					}
					state := user.Missions[story.missionId]
					if tt.clear && (state.ProgressValue != 1 || state.ClearDatetime != 200) {
						t.Fatalf("mission state = %+v, want one clear at 200", state)
					}
					Sync(catalogs, user, 300)
					if user.Missions[story.missionId] != state {
						t.Fatal("refresh changed the recorded solo-character mission")
					}
				})
			}
		})
	}
}
