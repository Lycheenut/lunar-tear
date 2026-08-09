package service

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

func TestBuildUserProfileUsesLatestFullDeckAndRealHistory(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Status.Level = 12
	user.Login.TotalLoginCount = 6
	user.Battle.FinishCount = 4
	user.Quests[10] = store.UserQuestState{QuestId: 10, UserDeckNumber: 1, LatestStartDatetime: 100, ClearCount: 2}
	user.Quests[20] = store.UserQuestState{QuestId: 20, UserDeckNumber: 2, LatestStartDatetime: 200, ClearCount: 3, QuestStateType: model.UserQuestStateTypeCleared}

	var deckCharacterIds [3]string
	for i := range 3 {
		costumeUuid := "costume-" + string(rune('a'+i))
		weaponUuid := "weapon-" + string(rune('a'+i))
		deckCharacterUuid := "deck-character-" + string(rune('a'+i))
		deckCharacterIds[i] = deckCharacterUuid
		user.Costumes[costumeUuid] = store.CostumeState{UserCostumeUuid: costumeUuid, CostumeId: int32(100 + i)}
		user.Weapons[weaponUuid] = store.WeaponState{UserWeaponUuid: weaponUuid, WeaponId: int32(200 + i), Level: int32(10 + i)}
		user.DeckCharacters[deckCharacterUuid] = store.DeckCharacterState{
			UserDeckCharacterUuid: deckCharacterUuid,
			UserCostumeUuid:       costumeUuid,
			MainUserWeaponUuid:    weaponUuid,
		}
	}
	user.Decks[store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: 2}] = store.DeckState{
		DeckType: model.DeckTypeQuest, UserDeckNumber: 2, Power: 4321,
		UserDeckCharacterUuid01: deckCharacterIds[0],
		UserDeckCharacterUuid02: deckCharacterIds[1],
		UserDeckCharacterUuid03: deckCharacterIds[2],
	}
	catalogs := &runtime.Catalogs{
		Quest: &masterdata.QuestCatalog{
			QuestById:   map[int32]masterdata.EntityMQuest{10: {}, 20: {}},
			MissionById: map[int32]masterdata.EntityMQuestMission{},
		},
		Costume:   &masterdata.CostumeCatalog{Costumes: map[int32]masterdata.EntityMCostume{100: {}, 101: {}, 102: {}}},
		Weapon:    &masterdata.WeaponCatalog{Weapons: map[int32]masterdata.EntityMWeapon{200: {}, 201: {}, 202: {}}},
		Companion: &masterdata.CompanionCatalog{CompanionById: map[int32]masterdata.EntityMCompanion{}},
	}

	user.Profile.CurrentPvpRank = 7
	user.Profile.CurrentPvpGradeId = 8
	user.Profile.MaxPvpSeasonRank = 9
	profile := buildUserProfile(*user, catalogs, true)
	if profile.LatestUsedDeck.Power != 4321 {
		t.Fatalf("deck power = %d, want 4321", profile.LatestUsedDeck.Power)
	}
	if got := len(profile.LatestUsedDeck.DeckCharacter); got != 3 {
		t.Fatalf("profile deck characters = %d, want 3", got)
	}
	if !profile.IsFriend {
		t.Fatal("friend state was not reflected in profile")
	}
	if profile.PvpInfo.CurrentRank != 7 || profile.PvpInfo.CurrentGradeId != 8 || profile.PvpInfo.MaxSeasonRank != 9 {
		t.Fatalf("pvp info = %+v", profile.PvpInfo)
	}
	if got := gamePlayHistoryValue(*user, playHistoryQuestClear); got != 5 {
		t.Fatalf("quest clear history = %d, want 5", got)
	}
	if got := len(profile.GamePlayHistory.HistoryItem); got == 0 {
		t.Fatal("game play history is empty")
	}
	if got := profile.GamePlayHistory.HistoryCategoryGraphItem[0].ProgressPermil; got != 500 {
		t.Fatalf("quest graph progress = %d, want 500", got)
	}
}

func TestRecordUserMessageMissionEventsUsesTextResourceWords(t *testing.T) {
	tests := []struct {
		message string
		groups  []int32
	}{
		{message: "A Happy New Year", groups: []int32{420, 392}},
		{message: "お年玉をありがとう", groups: []int32{420, 392}},
		{message: "ママ", groups: []int32{420, 393}},
		{message: "ordinary comment", groups: []int32{420}},
	}
	for _, tt := range tests {
		user := &store.UserState{}
		recordUserMessageMissionEvents(user, tt.message)
		if len(user.PendingMissionEvents) != len(tt.groups) {
			t.Fatalf("message %q emitted groups %+v, want %v", tt.message, user.PendingMissionEvents, tt.groups)
		}
		for i, group := range tt.groups {
			if user.PendingMissionEvents[i].OptionGroupId != group {
				t.Fatalf("message %q event %d group = %d, want %d", tt.message, i, user.PendingMissionEvents[i].OptionGroupId, group)
			}
		}
	}
}
