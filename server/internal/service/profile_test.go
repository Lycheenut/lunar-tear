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
	user.Costumes["costume-duplicate"] = store.CostumeState{UserCostumeUuid: "costume-duplicate", CostumeId: 100}
	user.Weapons["weapon-duplicate"] = store.WeaponState{UserWeaponUuid: "weapon-duplicate", WeaponId: 200}
	user.Companions["companion-a"] = store.CompanionState{UserCompanionUuid: "companion-a", CompanionId: 300}
	user.Companions["companion-duplicate"] = store.CompanionState{UserCompanionUuid: "companion-duplicate", CompanionId: 300}
	user.Parts["parts-a"] = store.PartsState{UserPartsUuid: "parts-a", PartsId: 400}
	user.Parts["parts-b"] = store.PartsState{UserPartsUuid: "parts-b", PartsId: 401}
	user.Missions[1] = store.UserMissionState{MissionId: 1, MissionProgressStatusType: int32(model.MissionProgressStatusTypeClear)}
	user.Missions[2] = store.UserMissionState{MissionId: 2, MissionProgressStatusType: int32(model.MissionProgressStatusTypeRewardReceived)}
	user.Missions[3] = store.UserMissionState{MissionId: 3, MissionProgressStatusType: int32(model.MissionProgressStatusTypeInProgress)}
	user.Gacha.BannerStates[1000] = store.GachaBannerState{GachaId: 1000, DrawCount: 7}
	user.Gacha.BannerStates[2000] = store.GachaBannerState{GachaId: 2000, DrawCount: 11}
	user.Gacha.BannerStates[3000] = store.GachaBannerState{GachaId: 3000, DrawCount: 13}
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
		GachaEntries: []store.GachaCatalogEntry{
			{GachaId: 1000, GachaLabelType: model.GachaLabelChapter},
			{GachaId: 2000, GachaLabelType: model.GachaLabelEvent},
			{GachaId: 3000, GachaLabelType: model.GachaLabelPremium},
		},
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
	wantHistory := []struct {
		id    int32
		count int64
	}{
		{id: 1, count: 7},
		{id: 2, count: 11},
		{id: 12, count: 2},
		{id: 14, count: 3},
		{id: 15, count: 3},
		{id: 16, count: 1},
		{id: 17, count: 2},
		{id: 18, count: 6},
	}
	if got := len(profile.GamePlayHistory.HistoryItem); got != len(wantHistory) {
		t.Fatalf("game play history item count = %d, want %d", got, len(wantHistory))
	}
	for i, want := range wantHistory {
		got := profile.GamePlayHistory.HistoryItem[i]
		if got.HistoryItemId != want.id || got.Count != want.count {
			t.Fatalf("game play history item %d = (id %d, count %d), want (id %d, count %d)", i, got.HistoryItemId, got.Count, want.id, want.count)
		}
	}
	if got := len(profile.GamePlayHistory.HistoryCategoryGraphItem); got != 5 {
		t.Fatalf("history graph item count = %d, want 5", got)
	}
	for i, got := range profile.GamePlayHistory.HistoryCategoryGraphItem {
		if got.CategoryTypeId != int32(i+1) || got.ProgressPermil != 0 {
			t.Fatalf("history graph item %d = (id %d, progress %d), want (id %d, progress 0)", i, got.CategoryTypeId, got.ProgressPermil, i+1)
		}
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
