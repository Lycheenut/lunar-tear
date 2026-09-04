package questflow

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestHandleQuestSkipAwardsSelectedDeckExp(t *testing.T) {
	const (
		questId      int32 = 10
		previousDeck int32 = 1
		selectedDeck int32 = 2
		characterExp int32 = 100
		costumeExp   int32 = 200
		skipTicketId int32 = 7
	)
	h := &QuestHandler{
		QuestCatalog: &masterdata.QuestCatalog{
			QuestById: map[int32]masterdata.EntityMQuest{
				questId: {
					QuestId:            questId,
					Stamina:            1,
					IsUsableSkipTicket: true,
					CharacterExp:       characterExp,
					CostumeExp:         costumeExp,
				},
			},
			CostumeById: map[int32]masterdata.EntityMCostume{
				101: {CostumeId: 101, CharacterId: 1, RarityType: 20},
				102: {CostumeId: 102, CharacterId: 2, RarityType: 20},
				103: {CostumeId: 103, CharacterId: 3, RarityType: 20},
			},
			CharacterExpThresholds: []int32{0, 1_000},
			CostumeExpByRarity:     map[int32][]int32{20: {0, 1_000}},
			MaxStaminaByLevel:      map[int32]int32{1: 100},
			MissionIdsByQuestId:    map[int32][]int32{},
			BattleDropsByQuestId:   map[int32][]masterdata.BattleDropInfo{},
		},
		Config:  &masterdata.GameConfig{ConsumableItemIdForQuestSkipTicket: skipTicketId},
		Granter: &store.PossessionGranter{},
	}
	user := store.SeedUserState(1, "skip-exp", 1, model.ClientPlatform{})
	user.Status.Level = 1
	user.Status.StaminaMilliValue = 100_000
	user.ConsumableItems[skipTicketId] = 1
	user.Quests[questId] = store.UserQuestState{
		QuestId:        questId,
		QuestStateType: model.UserQuestStateTypeCleared,
		UserDeckNumber: previousDeck,
	}
	user.Decks[store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: previousDeck}] = store.DeckState{
		DeckType:                model.DeckTypeQuest,
		UserDeckNumber:          previousDeck,
		UserDeckCharacterUuid01: "previous-unit",
	}
	user.DeckCharacters["previous-unit"] = store.DeckCharacterState{
		UserDeckCharacterUuid: "previous-unit",
		UserCostumeUuid:       "costume-2",
	}
	user.Decks[store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: selectedDeck}] = store.DeckState{
		DeckType:                model.DeckTypeQuest,
		UserDeckNumber:          selectedDeck,
		UserDeckCharacterUuid01: "unit-1",
		UserDeckCharacterUuid02: "unit-2",
		UserDeckCharacterUuid03: "unit-3",
	}
	for i := int32(1); i <= 3; i++ {
		unitUuid := "unit-" + string(rune('0'+i))
		costumeUuid := "costume-" + string(rune('0'+i))
		user.DeckCharacters[unitUuid] = store.DeckCharacterState{
			UserDeckCharacterUuid: unitUuid,
			UserCostumeUuid:       costumeUuid,
		}
		user.Costumes[costumeUuid] = store.CostumeState{
			UserCostumeUuid: costumeUuid,
			CostumeId:       100 + i,
			Level:           1,
		}
		user.Characters[i] = store.CharacterState{CharacterId: i, Level: 1}
	}

	if _, err := h.HandleQuestSkip(user, questId, int32(model.QuestTypeMain), 0, selectedDeck, 1, 100); err != nil {
		t.Fatalf("skip quest failed: %v", err)
	}

	if got := user.Quests[questId].UserDeckNumber; got != selectedDeck {
		t.Fatalf("recorded deck number = %d, want %d", got, selectedDeck)
	}
	for i := int32(1); i <= 3; i++ {
		if got := user.Characters[i].Exp; got != characterExp {
			t.Errorf("character %d exp = %d, want %d", i, got, characterExp)
		}
		costumeUuid := "costume-" + string(rune('0'+i))
		if got := user.Costumes[costumeUuid].Exp; got != costumeExp {
			t.Errorf("costume %d exp = %d, want %d", i, got, costumeExp)
		}
	}
}
