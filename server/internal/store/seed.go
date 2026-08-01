package store

import (
	"lunar-tear/server/internal/model"
)

const (
	starterMissionId         = int32(1)
	starterMainQuestRouteId  = int32(1)
	starterMainQuestSeasonId = int32(1)

	defaultBirthYear            = int32(2000)
	defaultBirthMonth           = int32(1)
	defaultBackupToken          = "mock-backup-token"
	defaultChargeMoneyThisMonth = int64(0)
)

func SeedUserState(userId int64, uuid string, nowMillis int64, platform model.ClientPlatform) *UserState {
	user := &UserState{
		UserId:               userId,
		Uuid:                 uuid,
		PlayerId:             userId,
		OsType:               platform.OsType,
		PlatformType:         platform.PlatformType,
		UserRestrictionType:  0,
		RegisterDatetime:     nowMillis,
		GameStartDatetime:    nowMillis,
		LatestVersion:        0,
		BirthYear:            defaultBirthYear,
		BirthMonth:           defaultBirthMonth,
		BackupToken:          defaultBackupToken,
		ChargeMoneyThisMonth: defaultChargeMoneyThisMonth,
		Setting: UserSettingState{
			IsNotifyPurchaseAlert: false,
			LatestVersion:         0,
		},
		Status: UserStatusState{
			Level:                 1,
			Exp:                   0,
			StaminaMilliValue:     50000,
			StaminaUpdateDatetime: nowMillis,
			LatestVersion:         0,
		},
		Gem: UserGemState{
			PaidGem: 0,
			FreeGem: 0,
		},
		Profile: UserProfileState{
			Name:                            "",
			NameUpdateDatetime:              0,
			Message:                         "",
			MessageUpdateDatetime:           nowMillis,
			FavoriteCostumeId:               0,
			FavoriteCostumeIdUpdateDatetime: nowMillis,
			LatestVersion:                   0,
		},
		Login: UserLoginState{
			TotalLoginCount:           1,
			ContinualLoginCount:       1,
			MaxContinualLoginCount:    1,
			LastLoginDatetime:         nowMillis,
			LastComebackLoginDatetime: 0,
			LatestVersion:             0,
		},
		LoginBonus: UserLoginBonusState{
			LoginBonusId:                1,
			CurrentPageNumber:           1,
			CurrentStampNumber:          0,
			LatestRewardReceiveDatetime: 0,
			LatestVersion:               0,
		},
		Tutorials: map[int32]TutorialProgressState{
			1: {TutorialType: 1},
		},
		Battle: BattleState{},
		Gifts: GiftState{
			NotReceived: []NotReceivedGiftState{},
			Received:    []ReceivedGiftState{},
		},
		Gacha: GachaState{
			ConvertedGachaMedal: ConvertedGachaMedalState{
				ConvertedMedalPossession: []ConsumableItemState{},
			},
			BannerStates: make(map[int32]GachaBannerState),
		},
		MainQuest: MainQuestState{
			CurrentMainQuestRouteId: starterMainQuestRouteId,
			MainQuestSeasonId:       starterMainQuestSeasonId,
		},
		Notifications: NotificationState{
			GiftNotReceiveCount: 1,
		},
		Friends:                  make(map[int64]FriendState),
		FriendRequests:           make(map[int64]int64),
		Characters:               make(map[int32]CharacterState),
		Costumes:                 make(map[string]CostumeState),
		Weapons:                  make(map[string]WeaponState),
		Companions:               make(map[string]CompanionState),
		DeckCharacters:           make(map[string]DeckCharacterState),
		Decks:                    make(map[DeckKey]DeckState),
		TripleDecks:              make(map[DeckKey]TripleDeckState),
		DeckSubWeapons:           make(map[string][]string),
		DeckParts:                make(map[string][]string),
		Quests:                   make(map[int32]UserQuestState),
		QuestMissions:            make(map[QuestMissionKey]UserQuestMissionState),
		SideStoryQuests:          make(map[int32]SideStoryQuestProgress),
		QuestLimitContentStatus:  make(map[int32]QuestLimitContentStatus),
		BigHuntMaxScores:         make(map[int32]BigHuntMaxScore),
		BigHuntStatuses:          make(map[int32]BigHuntStatus),
		BigHuntScheduleMaxScores: make(map[BigHuntScheduleScoreKey]BigHuntScheduleMaxScore),
		BigHuntWeeklyMaxScores:   make(map[BigHuntWeeklyScoreKey]BigHuntWeeklyMaxScore),
		BigHuntWeeklyStatuses:    make(map[int64]BigHuntWeeklyStatus),
		WeaponStories:            make(map[int32]WeaponStoryState),
		Missions: map[int32]UserMissionState{
			starterMissionId: {
				MissionId:                 starterMissionId,
				StartDatetime:             nowMillis,
				MissionProgressStatusType: int32(model.MissionProgressStatusTypeInProgress),
			},
		},
		Gimmick: GimmickState{
			Progress:         make(map[GimmickKey]GimmickProgressState),
			OrnamentProgress: make(map[GimmickOrnamentKey]GimmickOrnamentProgressState),
			Sequences:        make(map[GimmickSequenceKey]GimmickSequenceState),
			Unlocks:          make(map[GimmickKey]GimmickUnlockState),
		},
		CageOrnamentRewards:        make(map[int32]CageOrnamentRewardState),
		CageOrnamentAccesses:       make(map[int32]CageOrnamentAccessState),
		DeckLimitContentRestricted: make(map[string]DeckLimitContentRestrictedState),
		TowerAccumulationRewards:   make(map[int32]TowerAccumulationRewardState),
		LabyrinthSeasons:           make(map[int32]LabyrinthSeasonState),
		LabyrinthStages:            make(map[LabyrinthStageKey]LabyrinthStageState),
		ConsumableItems:            make(map[int32]int32),
		Materials:                  make(map[int32]int32),
		Thoughts:                   make(map[string]ThoughtState),
		Parts:                      make(map[string]PartsState),
		PartsGroupNotes:            make(map[int32]PartsGroupNoteState),
		PartsPresets:               make(map[int32]PartsPresetState),
		PartsPresetTags:            make(map[int32]PartsPresetTagState),
		PartsStatusSubs:            make(map[PartsStatusSubKey]PartsStatusSubState),
		ImportantItems:             make(map[int32]int32),
		CostumeActiveSkills:        make(map[string]CostumeActiveSkillState),
		WeaponSkills:               make(map[string][]WeaponSkillState),
		WeaponAbilities:            make(map[string][]WeaponAbilityState),
		DeckTypeNotes:              make(map[model.DeckType]DeckTypeNoteState),
		WeaponNotes:                make(map[int32]WeaponNoteState),
		NaviCutInPlayed:            make(map[int32]bool),
		ViewedMovies:               make(map[int32]int64),
		ContentsStories:            make(map[int32]int64),
		DrawnOmikuji:               make(map[int32]int64),
		PremiumItems:               make(map[int32]int64),
		DokanConfirmed:             make(map[int32]bool),
		ShopItems:                  make(map[int32]UserShopItemState),
		ShopReplaceableLineup:      make(map[int32]UserShopReplaceableLineupState),
		ExploreScores:              make(map[int32]ExploreScoreState),

		CharacterBoards:         make(map[int32]CharacterBoardState),
		CharacterBoardAbilities: make(map[CharacterBoardAbilityKey]CharacterBoardAbilityState),
		CharacterBoardStatusUps: make(map[CharacterBoardStatusUpKey]CharacterBoardStatusUpState),

		CostumeAwakenStatusUps:           make(map[CostumeAwakenStatusKey]CostumeAwakenStatusUpState),
		CostumeLevelBonusReleaseStatuses: make(map[int32]CostumeLevelBonusReleaseStatusState),
		CostumeLotteryEffects:            make(map[CostumeLotteryEffectKey]CostumeLotteryEffectState),
		CostumeLotteryEffectAbilities:    make(map[CostumeLotteryEffectKey]CostumeLotteryEffectAbilityState),
		CostumeLotteryEffectStatusUps:    make(map[CostumeLotteryEffectStatusKey]CostumeLotteryEffectStatusUpState),
		CostumeLotteryEffectPending:      make(map[string]CostumeLotteryEffectPendingState),
		AutoSaleSettings:                 make(map[int32]AutoSaleSettingState),
		CharacterRebirths:                make(map[int32]CharacterRebirthState),
	}
	user.EnsureMaps()
	return user
}
