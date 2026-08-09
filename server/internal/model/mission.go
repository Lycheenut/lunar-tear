package model

type MissionClearConditionType int32

const (
	MissionClearConditionTypeUnknown                                MissionClearConditionType = 0
	MissionClearConditionTypeQuestClearByCount                      MissionClearConditionType = 1
	MissionClearConditionTypeQuestClearById                         MissionClearConditionType = 2
	MissionClearConditionTypePossessionAddByCount                   MissionClearConditionType = 3
	MissionClearConditionTypeWeaponEnhanceByCount                   MissionClearConditionType = 5
	MissionClearConditionTypeWeaponEnhanceSkillByCount              MissionClearConditionType = 6
	MissionClearConditionTypeWeaponEvolveByCount                    MissionClearConditionType = 7
	MissionClearConditionTypeWeaponLimitBreakByCount                MissionClearConditionType = 8
	MissionClearConditionTypeCostumeEnhanceByCount                  MissionClearConditionType = 9
	MissionClearConditionTypeCostumeActiveSkillEnhanceByCount       MissionClearConditionType = 10
	MissionClearConditionTypeCostumeLimitBreakByCount               MissionClearConditionType = 11
	MissionClearConditionTypeCompanionEnhanceByCount                MissionClearConditionType = 12
	MissionClearConditionTypePartsEnhanceByCount                    MissionClearConditionType = 13
	MissionClearConditionTypePartsAddByCount                        MissionClearConditionType = 14
	MissionClearConditionTypePvpFinishByCount                       MissionClearConditionType = 15
	MissionClearConditionTypePvpFinishByWinCount                    MissionClearConditionType = 16
	MissionClearConditionTypePvpFinishByWinStreakCount              MissionClearConditionType = 17
	MissionClearConditionTypeGachaDrawByCount                       MissionClearConditionType = 18
	MissionClearConditionTypeGachaExecByCount                       MissionClearConditionType = 19
	MissionClearConditionTypeGachaDrawByGachaLabelType              MissionClearConditionType = 20
	MissionClearConditionTypeShopBuyByCount                         MissionClearConditionType = 21
	MissionClearConditionTypeUserLevel                              MissionClearConditionType = 22
	MissionClearConditionTypeUserLoginByCount                       MissionClearConditionType = 23
	MissionClearConditionTypeMissionClearByCount                    MissionClearConditionType = 25
	MissionClearConditionTypeExploreFinishByCount                   MissionClearConditionType = 26
	MissionClearConditionTypeSetFavoriteCharacter                   MissionClearConditionType = 27
	MissionClearConditionTypeMaxDeckPower                           MissionClearConditionType = 28
	MissionClearConditionTypeExploreHighScore                       MissionClearConditionType = 29
	MissionClearConditionTypeMissionClearForAllDaily                MissionClearConditionType = 30
	MissionClearConditionTypeMamaTapByCount                         MissionClearConditionType = 31
	MissionClearConditionTypeTowerWalkedDistance                    MissionClearConditionType = 32
	MissionClearConditionTypeCageOrnamentRewardAcquiredByCount      MissionClearConditionType = 33
	MissionClearConditionTypeDefeatBossCount                        MissionClearConditionType = 35
	MissionClearConditionTypeQuestBattleRetiredCount                MissionClearConditionType = 36
	MissionClearConditionTypeQuestBattleAnnihilatedCount            MissionClearConditionType = 37
	MissionClearConditionTypeCompleteTransferSettings               MissionClearConditionType = 38
	MissionClearConditionTypeLibraryElementCount                    MissionClearConditionType = 39
	MissionClearConditionTypeCostumeMaxLevel                        MissionClearConditionType = 40
	MissionClearConditionTypeCostumeSkillMaxLevel                   MissionClearConditionType = 41
	MissionClearConditionTypeCostumeAbilityMaxLevel                 MissionClearConditionType = 42
	MissionClearConditionTypeWeaponMaxLevel                         MissionClearConditionType = 43
	MissionClearConditionTypeWeaponSkillMaxLevel                    MissionClearConditionType = 44
	MissionClearConditionTypeWeaponAbilityMaxLevel                  MissionClearConditionType = 45
	MissionClearConditionTypeCompanionMaxLevel                      MissionClearConditionType = 46
	MissionClearConditionTypePartsMaxLevel                          MissionClearConditionType = 47
	MissionClearConditionTypeQuiz                                   MissionClearConditionType = 48
	MissionClearConditionTypePossessionComplete                     MissionClearConditionType = 49
	MissionClearConditionTypeCheerFriendByCount                     MissionClearConditionType = 50
	MissionClearConditionTypeBigHuntPlayByCount                     MissionClearConditionType = 51
	MissionClearConditionTypeBigHuntBossKnockDown                   MissionClearConditionType = 52
	MissionClearConditionTypeBigHuntHighScore                       MissionClearConditionType = 53
	MissionClearConditionTypeCharacterBoardPanelReleaseByCount      MissionClearConditionType = 54
	MissionClearConditionTypeDefeatWizardCount                      MissionClearConditionType = 55
	MissionClearConditionTypeUserMessageMatchWord                   MissionClearConditionType = 56
	MissionClearConditionTypeTitleTransitionByCount                 MissionClearConditionType = 57
	MissionClearConditionTypeWeaponProtectByCount                   MissionClearConditionType = 58
	MissionClearConditionTypeExploreScore                           MissionClearConditionType = 59
	MissionClearConditionTypeReportCount                            MissionClearConditionType = 60
	MissionClearConditionTypeRhythmInteractionTapCount              MissionClearConditionType = 61
	MissionClearConditionTypeQuestClearByCountWithoutSkip           MissionClearConditionType = 62
	MissionClearConditionTypeCharacterBoardFullOpen                 MissionClearConditionType = 63
	MissionClearConditionTypeWeaponCountWithLevelGE                 MissionClearConditionType = 64
	MissionClearConditionTypeConsumeStaminaAmount                   MissionClearConditionType = 65
	MissionClearConditionTypeCostumeAwakenCount                     MissionClearConditionType = 66
	MissionClearConditionTypeUserLoginByCountFromUnlock             MissionClearConditionType = 67
	MissionClearConditionTypeMissionClearForAllDailyBySubCategoryId MissionClearConditionType = 68
	MissionClearConditionTypePvpRank                                MissionClearConditionType = 69
	MissionClearConditionTypeWeaponAwakenCount                      MissionClearConditionType = 70
	MissionClearConditionTypeCharacterRebirthCount                  MissionClearConditionType = 71
	MissionClearConditionTypeCostumeLotteryEffectSlotUnlockCount    MissionClearConditionType = 72
	MissionClearConditionTypeCostumeLotteryEffectDrawCount          MissionClearConditionType = 73
	MissionClearConditionTypePvpFinishByWinStreakCountFromUnlock    MissionClearConditionType = 74
)

type MissionUnlockConditionType int32

const (
	MissionUnlockConditionTypeUnknown                              MissionUnlockConditionType = 0
	MissionUnlockConditionTypeGrant                                MissionUnlockConditionType = 1
	MissionUnlockConditionTypeQuestClear                           MissionUnlockConditionType = 2
	MissionUnlockConditionTypeMissionClearById                     MissionUnlockConditionType = 3
	MissionUnlockConditionTypeMissionClearForAllDaily              MissionUnlockConditionType = 4
	MissionUnlockConditionTypeWebviewPanelMissionClearByPageNumber MissionUnlockConditionType = 5
	MissionUnlockConditionTypeEvaluate                             MissionUnlockConditionType = 6
)

func (t MissionUnlockConditionType) IsKnown() bool {
	return t >= MissionUnlockConditionTypeGrant && t <= MissionUnlockConditionTypeEvaluate
}

func (t MissionClearConditionType) IsFromUnlock() bool {
	return t == MissionClearConditionTypeUserLoginByCountFromUnlock ||
		t == MissionClearConditionTypePvpFinishByWinStreakCountFromUnlock
}

func (t MissionClearConditionType) IsKnown() bool {
	switch t {
	case MissionClearConditionTypeQuestClearByCount,
		MissionClearConditionTypeQuestClearById,
		MissionClearConditionTypePossessionAddByCount,
		MissionClearConditionTypeWeaponEnhanceByCount,
		MissionClearConditionTypeWeaponEnhanceSkillByCount,
		MissionClearConditionTypeWeaponEvolveByCount,
		MissionClearConditionTypeWeaponLimitBreakByCount,
		MissionClearConditionTypeCostumeEnhanceByCount,
		MissionClearConditionTypeCostumeActiveSkillEnhanceByCount,
		MissionClearConditionTypeCostumeLimitBreakByCount,
		MissionClearConditionTypeCompanionEnhanceByCount,
		MissionClearConditionTypePartsEnhanceByCount,
		MissionClearConditionTypePartsAddByCount,
		MissionClearConditionTypePvpFinishByCount,
		MissionClearConditionTypePvpFinishByWinCount,
		MissionClearConditionTypePvpFinishByWinStreakCount,
		MissionClearConditionTypeGachaDrawByCount,
		MissionClearConditionTypeGachaExecByCount,
		MissionClearConditionTypeGachaDrawByGachaLabelType,
		MissionClearConditionTypeShopBuyByCount,
		MissionClearConditionTypeUserLevel,
		MissionClearConditionTypeUserLoginByCount,
		MissionClearConditionTypeMissionClearByCount,
		MissionClearConditionTypeExploreFinishByCount,
		MissionClearConditionTypeSetFavoriteCharacter,
		MissionClearConditionTypeMaxDeckPower,
		MissionClearConditionTypeExploreHighScore,
		MissionClearConditionTypeMissionClearForAllDaily,
		MissionClearConditionTypeMamaTapByCount,
		MissionClearConditionTypeTowerWalkedDistance,
		MissionClearConditionTypeCageOrnamentRewardAcquiredByCount,
		MissionClearConditionTypeDefeatBossCount,
		MissionClearConditionTypeQuestBattleRetiredCount,
		MissionClearConditionTypeQuestBattleAnnihilatedCount,
		MissionClearConditionTypeCompleteTransferSettings,
		MissionClearConditionTypeLibraryElementCount,
		MissionClearConditionTypeCostumeMaxLevel,
		MissionClearConditionTypeCostumeSkillMaxLevel,
		MissionClearConditionTypeCostumeAbilityMaxLevel,
		MissionClearConditionTypeWeaponMaxLevel,
		MissionClearConditionTypeWeaponSkillMaxLevel,
		MissionClearConditionTypeWeaponAbilityMaxLevel,
		MissionClearConditionTypeCompanionMaxLevel,
		MissionClearConditionTypePartsMaxLevel,
		MissionClearConditionTypeQuiz,
		MissionClearConditionTypePossessionComplete,
		MissionClearConditionTypeCheerFriendByCount,
		MissionClearConditionTypeBigHuntPlayByCount,
		MissionClearConditionTypeBigHuntBossKnockDown,
		MissionClearConditionTypeBigHuntHighScore,
		MissionClearConditionTypeCharacterBoardPanelReleaseByCount,
		MissionClearConditionTypeDefeatWizardCount,
		MissionClearConditionTypeUserMessageMatchWord,
		MissionClearConditionTypeTitleTransitionByCount,
		MissionClearConditionTypeWeaponProtectByCount,
		MissionClearConditionTypeExploreScore,
		MissionClearConditionTypeReportCount,
		MissionClearConditionTypeRhythmInteractionTapCount,
		MissionClearConditionTypeQuestClearByCountWithoutSkip,
		MissionClearConditionTypeCharacterBoardFullOpen,
		MissionClearConditionTypeWeaponCountWithLevelGE,
		MissionClearConditionTypeConsumeStaminaAmount,
		MissionClearConditionTypeCostumeAwakenCount,
		MissionClearConditionTypeUserLoginByCountFromUnlock,
		MissionClearConditionTypeMissionClearForAllDailyBySubCategoryId,
		MissionClearConditionTypePvpRank,
		MissionClearConditionTypeWeaponAwakenCount,
		MissionClearConditionTypeCharacterRebirthCount,
		MissionClearConditionTypeCostumeLotteryEffectSlotUnlockCount,
		MissionClearConditionTypeCostumeLotteryEffectDrawCount,
		MissionClearConditionTypePvpFinishByWinStreakCountFromUnlock:
		return true
	default:
		return false
	}
}
