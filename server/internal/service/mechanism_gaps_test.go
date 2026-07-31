package service

import (
	"math"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

func TestValidateLoginBonusTerm(t *testing.T) {
	day := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC).UnixMilli()
	term := masterdata.LoginBonusTerm{
		StartDatetime:           day - int64(time.Hour/time.Millisecond),
		EndDatetime:             day + int64(time.Hour/time.Millisecond),
		StampReceiveEndDatetime: day + int64(2*time.Hour/time.Millisecond),
	}
	lb := store.UserLoginBonusState{LoginBonusId: 1}

	if err := validateLoginBonusTerm(term, lb, day); err != nil {
		t.Fatalf("valid term rejected: %v", err)
	}

	cases := []struct {
		name string
		term masterdata.LoginBonusTerm
		lb   store.UserLoginBonusState
		now  int64
	}{
		{name: "not started", term: term, lb: lb, now: term.StartDatetime - 1},
		{name: "ended", term: term, lb: lb, now: term.EndDatetime},
		{name: "stamp receive ended", term: masterdata.LoginBonusTerm{StampReceiveEndDatetime: day}, lb: lb, now: day},
		{name: "already received today", term: term, lb: store.UserLoginBonusState{LoginBonusId: 1, LatestRewardReceiveDatetime: day - 1000}, now: day},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateLoginBonusTerm(tc.term, tc.lb, tc.now); status.Code(err) != codes.FailedPrecondition {
				t.Fatalf("status = %v, want FailedPrecondition (err=%v)", status.Code(err), err)
			}
		})
	}
}

func TestAdvanceLoginStateCountsCalendarDays(t *testing.T) {
	day := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

	sameDay := store.UserLoginState{
		TotalLoginCount:        5,
		ContinualLoginCount:    3,
		MaxContinualLoginCount: 4,
		LastLoginDatetime:      day.Add(-time.Hour).UnixMilli(),
	}
	advanceLoginState(&sameDay, day.UnixMilli())
	assertLoginCounts(t, sameDay, 5, 3, 4)

	consecutive := store.UserLoginState{
		TotalLoginCount:        5,
		ContinualLoginCount:    4,
		MaxContinualLoginCount: 4,
		LastLoginDatetime:      day.Add(-24 * time.Hour).UnixMilli(),
	}
	advanceLoginState(&consecutive, day.UnixMilli())
	assertLoginCounts(t, consecutive, 6, 5, 5)

	broken := store.UserLoginState{
		TotalLoginCount:        5,
		ContinualLoginCount:    4,
		MaxContinualLoginCount: 7,
		LastLoginDatetime:      day.Add(-48 * time.Hour).UnixMilli(),
	}
	advanceLoginState(&broken, day.UnixMilli())
	assertLoginCounts(t, broken, 6, 1, 7)
}

func assertLoginCounts(t *testing.T, login store.UserLoginState, total, continual, maximum int32) {
	t.Helper()
	if login.TotalLoginCount != total || login.ContinualLoginCount != continual || login.MaxContinualLoginCount != maximum {
		t.Fatalf("login counts = (%d,%d,%d), want (%d,%d,%d)",
			login.TotalLoginCount, login.ContinualLoginCount, login.MaxContinualLoginCount,
			total, continual, maximum)
	}
}

func TestGrantGiftGrantsAssetsAndRejectsOverflow(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Materials[10] = 8
	granter := &store.PossessionGranter{}
	config := &masterdata.GameConfig{PossessionCountLimitMaterial: 10}

	if result := grantGift(user, store.GiftCommonState{
		PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 10, Count: 3,
	}, granter, config, 1000); result.Status != store.GrantStatusOverflow {
		t.Fatalf("overflowing gift status = %v, want overflow", result.Status)
	}
	if got := user.Materials[10]; got != 8 {
		t.Fatalf("material changed after overflow: got %d, want 8", got)
	}
	if result := grantGift(user, store.GiftCommonState{
		PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 10, Count: 2,
	}, granter, config, 1000); result.Status != store.GrantStatusGranted {
		t.Fatalf("gift at inventory limit status = %v, want granted", result.Status)
	}
	if got := user.Materials[10]; got != 10 {
		t.Fatalf("material count = %d, want 10", got)
	}
}

func TestGrantGiftDoesNotBlockOnUnrelatedExistingOverflow(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	user.Materials[10] = 11
	config := &masterdata.GameConfig{PossessionCountLimitMaterial: 10}
	if result := grantGift(user, store.GiftCommonState{
		PossessionType: int32(model.PossessionTypeFreeGem), Count: 5,
	}, &store.PossessionGranter{}, config, 1000); result.Status != store.GrantStatusGranted {
		t.Fatalf("unrelated existing overflow status = %v, want granted", result.Status)
	}
	if got := user.Gem.FreeGem; got != 5 {
		t.Fatalf("free gems = %d, want 5", got)
	}
}

func TestGiftRejectsUnsupportedPossessionWithoutCallingItOverflow(t *testing.T) {
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	result := grantGift(user, store.GiftCommonState{
		PossessionType: int32(model.PossessionTypeMissionPassPoint),
		PossessionId:   1,
		Count:          10,
	}, &store.PossessionGranter{}, &masterdata.GameConfig{}, 1000)
	if result.Status != store.GrantStatusUnsupported {
		t.Fatalf("mission pass gift status = %v, want unsupported", result.Status)
	}
}

func TestGiftPageRangeRejectsUntrustedCursor(t *testing.T) {
	if _, _, _, err := giftPageRange(0, 2, math.MaxInt64); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("large cursor status = %v, want InvalidArgument", status.Code(err))
	}
	start, end, pages, err := giftPageRange(5, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if start != 4 || end != 5 || pages != 3 {
		t.Fatalf("page range = (%d,%d,%d), want (4,5,3)", start, end, pages)
	}
}

func TestClaimableGiftCountExcludesExpiredGifts(t *testing.T) {
	gifts := []store.NotReceivedGiftState{
		{ExpirationDatetime: 999},
		{ExpirationDatetime: 1001},
		{},
	}
	if got := claimableGiftCount(gifts, 1000); got != 2 {
		t.Fatalf("claimable gift count = %d, want 2", got)
	}
}

func TestGiftFiltersMatchClientEnums(t *testing.T) {
	config := &masterdata.GameConfig{ConsumableItemIdForGold: 99}
	cases := []struct {
		possessionType model.PossessionType
		possessionId   int32
		want           int32
	}{
		{model.PossessionTypeFreeGem, 0, giftRewardKindGem},
		{model.PossessionTypeConsumableItem, 99, giftRewardKindGold},
		{model.PossessionTypeWeapon, 1, giftRewardKindWeapon},
		{model.PossessionTypeCompanion, 1, giftRewardKindCompanion},
		{model.PossessionTypeParts, 1, giftRewardKindParts},
		{model.PossessionTypeMaterial, 1, giftRewardKindMaterial},
		{model.PossessionTypeImportantItem, 1, giftRewardKindOther},
		{model.PossessionTypeCostume, 1, giftRewardKindCostume},
	}
	for _, tc := range cases {
		gift := store.GiftCommonState{PossessionType: int32(tc.possessionType), PossessionId: tc.possessionId}
		if got := giftRewardKind(gift, config); got != tc.want {
			t.Errorf("type %d id %d kind = %d, want %d", tc.possessionType, tc.possessionId, got, tc.want)
		}
	}

	expiring := store.NotReceivedGiftState{ExpirationDatetime: 100}
	permanent := store.NotReceivedGiftState{}
	if !matchesGiftExpirationFilter(expiring, giftExpirationFilterOnlyExpire) ||
		matchesGiftExpirationFilter(permanent, giftExpirationFilterOnlyExpire) ||
		!matchesGiftExpirationFilter(permanent, giftExpirationFilterOnlyNotExpire) {
		t.Fatal("expiration filters did not match client enum semantics")
	}
}

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
