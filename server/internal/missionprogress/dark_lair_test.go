package missionprogress

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

func TestSecretStoryDarkLairUsesActualQuestAndDeck(t *testing.T) {
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
	config, err := masterdata.LoadGameConfig()
	if err != nil {
		t.Fatal(err)
	}
	catalogs := &runtime.Catalogs{Mission: missions, Quest: quests, ConditionResolver: resolver}
	h := &questflow.QuestHandler{
		QuestCatalog: quests, Granter: questflow.BuildGranter(quests, nil), Config: config,
	}
	const nowMillis = int64(1790000000000)

	for _, story := range []struct {
		name                                                               string
		missionId, conditionId, chapterId, questId, characterId            int32
		characterChapterId, characterQuestId, otherChapterId, otherQuestId int32
	}{
		{name: "Akeha", missionId: 500057, conditionId: 5173, chapterId: 99004, questId: 110040, characterId: 1015,
			characterChapterId: 904, characterQuestId: 120040, otherChapterId: 99005, otherQuestId: 110050},
		{name: "Argo", missionId: 500085, conditionId: 5205, chapterId: 99005, questId: 110050, characterId: 1006,
			characterChapterId: 905, characterQuestId: 120050, otherChapterId: 99004, otherQuestId: 110040},
	} {
		t.Run(story.name, func(t *testing.T) {
			quest := quests.QuestById[story.questId]
			// quest.name.110010 is 真暗ノ巣窟:上級; character QUEST 10 uses text 210.
			if quest.NameQuestTextId != 110010 || quest.QuestDeckRestrictionGroupId != 0 ||
				!quests.EventQuestBelongsToChapter(story.chapterId, story.questId) ||
				!quests.EventCharacterIdsByChapterId[story.chapterId][story.characterId] {
				t.Fatal("target must be the character's unrestricted Dark Lair: Hard")
			}
			var costumeId int32
			for id, costume := range quests.CostumeById {
				if costume.CharacterId == story.characterId && (costumeId == 0 || id < costumeId) {
					costumeId = id
				}
			}
			if costumeId == 0 {
				t.Fatal("missing required character costume")
			}
			for _, tt := range []struct {
				name                        string
				restricted, ordinary        []string
				chapterId, questId          int32
				skip                        bool
				retired, annihilated, clear bool
			}{
				{name: "selected ordinary deck", ordinary: []string{"required"}, clear: true},
				{name: "restricted deck does not shadow ordinary deck", restricted: []string{"other"}, ordinary: []string{"required"}, clear: true},
				{name: "wrong ordinary character", restricted: []string{"required"}, ordinary: []string{"other"}},
				{name: "no ordinary deck", restricted: []string{"required"}},
				{name: "not solo", ordinary: []string{"required", "other"}},
				{name: "wrong difficulty", ordinary: []string{"required"}, questId: story.questId - 1},
				{name: "coin quest", ordinary: []string{"required"}, questId: story.questId - 5},
				{name: "other character lair", ordinary: []string{"required"}, chapterId: story.otherChapterId, questId: story.otherQuestId},
				{name: "character QUEST 10", restricted: []string{"required"}, chapterId: story.characterChapterId, questId: story.characterQuestId},
				{name: "skip ticket", ordinary: []string{"required"}, skip: true},
				{name: "retreat", ordinary: []string{"required"}, retired: true},
				{name: "defeat", ordinary: []string{"required"}, annihilated: true},
			} {
				t.Run(tt.name, func(t *testing.T) {
					questId := story.questId
					if tt.questId != 0 {
						questId = tt.questId
					}
					chapterId := story.chapterId
					if tt.chapterId != 0 {
						chapterId = tt.chapterId
					}
					user := &store.UserState{}
					user.EnsureMaps()
					user.Status.Level, user.Status.StaminaMilliValue = 1, 100000
					for _, id := range quests.EventUnlockQuestIdsForChapter(chapterId) {
						user.Quests[id] = store.UserQuestState{QuestId: id, QuestStateType: model.UserQuestStateTypeCleared}
					}
					user.Quests[questId] = store.UserQuestState{
						QuestId: questId, UserDeckNumber: 2, QuestStateType: model.UserQuestStateTypeCleared,
						ClearCount: 1, IsRewardGranted: true,
					}
					user.ConsumableItems[config.ConsumableItemIdForQuestSkipTicket] = 1
					user.DeckCharacters["required"] = store.DeckCharacterState{UserCostumeUuid: "required"}
					user.DeckCharacters["other"] = store.DeckCharacterState{UserCostumeUuid: "other"}
					user.Costumes["required"] = store.CostumeState{CostumeId: costumeId}
					user.Costumes["other"] = store.CostumeState{CostumeId: 10100}
					// A different deck number must not supply the mission's character either.
					user.Decks[store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: 1}] = store.DeckState{UserDeckCharacterUuid01: "other"}
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
					Sync(catalogs, user, nowMillis)
					if resolver.Satisfied(story.conditionId, user) {
						t.Fatal("quest history alone satisfied the solo-character mission")
					}
					before := store.CloneUserState(*user)
					if tt.skip {
						if _, err := h.HandleQuestSkip(user, questId, int32(model.QuestTypeEvent), chapterId, 2, 1, nowMillis+100); err != nil {
							t.Fatal(err)
						}
					} else {
						if err := h.HandleEventQuestStart(user, chapterId, questId, true, 2, nowMillis); err != nil {
							t.Fatal(err)
						}
						if err := h.ValidateEventQuestContinuation(user, chapterId, questId, nowMillis+100); err != nil {
							t.Fatal(err)
						}
						h.HandleEventQuestFinish(user, chapterId, questId, tt.retired, tt.annihilated, nowMillis+100)
					}
					events := user.PendingMissionEvents
					user.PendingMissionEvents = nil
					Apply(catalogs, &before, user, events, nowMillis+100)
					if got := resolver.Satisfied(story.conditionId, user); got != tt.clear {
						t.Fatalf("Dark Lair Secret Story condition = %v, want %v; mission = %+v", got, tt.clear, user.Missions[story.missionId])
					}
					state := user.Missions[story.missionId]
					if tt.clear && (state.ProgressValue != 1 || state.ClearDatetime != nowMillis+100) {
						t.Fatalf("mission state = %+v, want one clear at %d", state, nowMillis+100)
					}
					Sync(catalogs, user, nowMillis+200)
					if user.Missions[story.missionId] != state {
						t.Fatal("refresh changed the recorded solo-character mission")
					}
				})
			}
		})
	}
}
