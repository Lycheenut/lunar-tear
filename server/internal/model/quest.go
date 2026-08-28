package model

import "fmt"

type QuestType int32

const (
	QuestTypeUnknown QuestType = 0
	QuestTypeMain    QuestType = 1
	QuestTypeEvent   QuestType = 2
	QuestTypeExtra   QuestType = 3
	QuestTypeBigHunt QuestType = 4
)

type QuestFlowType int32

const (
	QuestFlowTypeUnknown                QuestFlowType = 0
	QuestFlowTypeMainFlow               QuestFlowType = 1
	QuestFlowTypeSubFlow                QuestFlowType = 2
	QuestFlowTypeReplayFlow             QuestFlowType = 3
	QuestFlowTypeAnotherRouteReplayFlow QuestFlowType = 4
)

func IsReplayQuestFlowType(t int32) bool {
	return t == int32(QuestFlowTypeReplayFlow) ||
		t == int32(QuestFlowTypeAnotherRouteReplayFlow)
}

func (t QuestFlowType) String() string {
	switch t {
	case QuestFlowTypeUnknown:
		return "unknown"
	case QuestFlowTypeMainFlow:
		return "main-flow"
	case QuestFlowTypeSubFlow:
		return "sub-flow"
	case QuestFlowTypeReplayFlow:
		return "replay-flow"
	case QuestFlowTypeAnotherRouteReplayFlow:
		return "another-route-replay-flow"
	default:
		return fmt.Sprintf("unknown-quest-flow(%d)", int32(t))
	}
}

type QuestResultType int32

const (
	QuestResultTypeUnknown    QuestResultType = 0
	QuestResultTypeNone       QuestResultType = 1
	QuestResultTypeHalfResult QuestResultType = 2
	QuestResultTypeFullResult QuestResultType = 3
)

type MissionProgressStatusType int32

const (
	MissionProgressStatusTypeUnknown        MissionProgressStatusType = 0
	MissionProgressStatusTypeInProgress     MissionProgressStatusType = 1
	MissionProgressStatusTypeClear          MissionProgressStatusType = 2
	MissionProgressStatusTypeRewardReceived MissionProgressStatusType = 9
)

type QuestSceneType int32

const (
	QuestSceneTypeUnknown      QuestSceneType = 0
	QuestSceneTypeTower        QuestSceneType = 1
	QuestSceneTypePictureBook  QuestSceneType = 2
	QuestSceneTypeField        QuestSceneType = 3
	QuestSceneTypeNovel        QuestSceneType = 4
	QuestSceneTypeLimitContent QuestSceneType = 5
)

type QuestMissionConditionType int

const (
	QuestMissionConditionTypeUnknown                                   QuestMissionConditionType = 0
	QuestMissionConditionTypeLessThanOrEqualXPeopleNotAlive            QuestMissionConditionType = 1
	QuestMissionConditionTypeMaxDamage                                 QuestMissionConditionType = 2
	QuestMissionConditionTypeSpecifiedCostumeIsInDeck                  QuestMissionConditionType = 3
	QuestMissionConditionTypeSpecifiedCharacterIsInDeck                QuestMissionConditionType = 4
	QuestMissionConditionTypeSpecifiedAttributeMainWeaponIsInDeck      QuestMissionConditionType = 5
	QuestMissionConditionTypeGreaterThanOrEqualXCostumeSkillUseCount   QuestMissionConditionType = 6
	QuestMissionConditionTypeGreaterThanOrEqualXWeaponSkillUseCount    QuestMissionConditionType = 7
	QuestMissionConditionTypeGreaterThanOrEqualXCompanionSkillUseCount QuestMissionConditionType = 8
	QuestMissionConditionTypeCostumeSkillfulWeaponAllCharacter         QuestMissionConditionType = 9
	QuestMissionConditionTypeCostumeSkillfulWeaponAnyCharacter         QuestMissionConditionType = 10
	QuestMissionConditionTypeCostumeRarityEqAllCharacter               QuestMissionConditionType = 11
	QuestMissionConditionTypeCostumeRarityGeAllCharacter               QuestMissionConditionType = 12
	QuestMissionConditionTypeCostumeRarityLeAllCharacter               QuestMissionConditionType = 13
	QuestMissionConditionTypeCostumeRarityEqAnyCharacter               QuestMissionConditionType = 14
	QuestMissionConditionTypeCostumeRarityGeAnyCharacter               QuestMissionConditionType = 15
	QuestMissionConditionTypeCostumeRarityLeAnyCharacter               QuestMissionConditionType = 16
	QuestMissionConditionTypeWeaponEvolutionGroupId                    QuestMissionConditionType = 17
	QuestMissionConditionTypeSpecifiedAttributeWeaponIsInDeck          QuestMissionConditionType = 18
	QuestMissionConditionTypeSpecifiedAttributeMainWeaponAllCharacter  QuestMissionConditionType = 19
	QuestMissionConditionTypeSpecifiedAttributeWeaponAllCharacter      QuestMissionConditionType = 20
	QuestMissionConditionTypeWeaponManSkillfulWeaponAllCharacter       QuestMissionConditionType = 21
	QuestMissionConditionTypeWeaponSkillfulWeaponAllCharacter          QuestMissionConditionType = 22
	QuestMissionConditionTypeWeaponManSkillfulWeaponAnyCharacter       QuestMissionConditionType = 23
	QuestMissionConditionTypeWeaponSkillfulWeaponAnyCharacter          QuestMissionConditionType = 24
	QuestMissionConditionTypeWeaponRarityEqAllCharacter                QuestMissionConditionType = 25
	QuestMissionConditionTypeWeaponRarityGeAllCharacter                QuestMissionConditionType = 26
	QuestMissionConditionTypeWeaponRarityLeAllCharacter                QuestMissionConditionType = 27
	QuestMissionConditionTypeWeaponMainRarityEqAllCharacter            QuestMissionConditionType = 28
	QuestMissionConditionTypeWeaponMainRarityGeAllCharacter            QuestMissionConditionType = 29
	QuestMissionConditionTypeWeaponMainRarityLeAllCharacter            QuestMissionConditionType = 30
	QuestMissionConditionTypeWeaponRarityEqAnyCharacter                QuestMissionConditionType = 31
	QuestMissionConditionTypeWeaponRarityGeAnyCharacter                QuestMissionConditionType = 32
	QuestMissionConditionTypeWeaponRarityLeAnyCharacter                QuestMissionConditionType = 33
	QuestMissionConditionTypeWeaponMainRarityEqAnyCharacter            QuestMissionConditionType = 34
	QuestMissionConditionTypeWeaponMainRarityGeAnyCharacter            QuestMissionConditionType = 35
	QuestMissionConditionTypeWeaponMainRarityLeAnyCharacter            QuestMissionConditionType = 36
	QuestMissionConditionTypeCompanionId                               QuestMissionConditionType = 37
	QuestMissionConditionTypeCompanionAttribute                        QuestMissionConditionType = 38
	QuestMissionConditionTypeCompanionCategory                         QuestMissionConditionType = 39
	QuestMissionConditionTypePartsId                                   QuestMissionConditionType = 40
	QuestMissionConditionTypePartsGroupId                              QuestMissionConditionType = 41
	QuestMissionConditionTypePartsRarityEq                             QuestMissionConditionType = 42
	QuestMissionConditionTypePartsRarityGe                             QuestMissionConditionType = 43
	QuestMissionConditionTypePartsRarityLe                             QuestMissionConditionType = 44
	QuestMissionConditionTypeDeckPowerGe                               QuestMissionConditionType = 45
	QuestMissionConditionTypeDeckPowerLe                               QuestMissionConditionType = 46
	QuestMissionConditionTypeDeckCostumeNumEq                          QuestMissionConditionType = 47
	QuestMissionConditionTypeDeckCostumeNumGe                          QuestMissionConditionType = 48
	QuestMissionConditionTypeDeckCostumeNumLe                          QuestMissionConditionType = 49
	QuestMissionConditionTypeCriticalCountGe                           QuestMissionConditionType = 50
	QuestMissionConditionTypeMinHpPercentageGe                         QuestMissionConditionType = 51
	QuestMissionConditionTypeComboCountGe                              QuestMissionConditionType = 52
	QuestMissionConditionTypeComboMaxDamageGe                          QuestMissionConditionType = 53
	QuestMissionConditionTypeLessThanOrEqualXCostumeSkillUseCount      QuestMissionConditionType = 54
	QuestMissionConditionTypeLessThanOrEqualXWeaponSkillUseCount       QuestMissionConditionType = 55
	QuestMissionConditionTypeLessThanOrEqualXCompanionSkillUseCount    QuestMissionConditionType = 56
	QuestMissionConditionTypeWithoutRecoverySkill                      QuestMissionConditionType = 57
	QuestMissionConditionTypeWithoutCostumeSkill                       QuestMissionConditionType = 58
	QuestMissionConditionTypeWithoutWeaponSkill                        QuestMissionConditionType = 59
	QuestMissionConditionTypeWithoutCompanionSkill                     QuestMissionConditionType = 60
	QuestMissionConditionTypeCharacterContainAll                       QuestMissionConditionType = 61
	QuestMissionConditionTypeCharacterContainAny                       QuestMissionConditionType = 62
	QuestMissionConditionTypeCostumeContainAll                         QuestMissionConditionType = 63
	QuestMissionConditionTypeCostumeContainAny                         QuestMissionConditionType = 64
	QuestMissionConditionTypeCostumeSkillfulWeaponContainAll           QuestMissionConditionType = 65
	QuestMissionConditionTypeCostumeSkillfulWeaponContainAny           QuestMissionConditionType = 66
	QuestMissionConditionTypeAttributeMainWeaponContainAll             QuestMissionConditionType = 67
	QuestMissionConditionTypeAttributeMainWeaponContainAny             QuestMissionConditionType = 68
	QuestMissionConditionTypeAttributeWeaponContainAll                 QuestMissionConditionType = 69
	QuestMissionConditionTypeAttributeWeaponContainAny                 QuestMissionConditionType = 70
	QuestMissionConditionTypeWeaponManSkillfulWeaponContainAll         QuestMissionConditionType = 71
	QuestMissionConditionTypeWeaponManSkillfulWeaponContainAny         QuestMissionConditionType = 72
	QuestMissionConditionTypeWeaponSkillfulWeaponContainAll            QuestMissionConditionType = 73
	QuestMissionConditionTypeWeaponSkillfulWeaponContainAny            QuestMissionConditionType = 74
	QuestMissionConditionTypeComplete                                  QuestMissionConditionType = 9999
)

type WeaponStoryReleaseConditionType int32

const (
	WeaponStoryReleaseConditionTypeUnknown                      WeaponStoryReleaseConditionType = 0
	WeaponStoryReleaseConditionTypeAcquisition                  WeaponStoryReleaseConditionType = 1
	WeaponStoryReleaseConditionTypeReachSpecifiedLevel          WeaponStoryReleaseConditionType = 2
	WeaponStoryReleaseConditionTypeReachInitialMaxLevel         WeaponStoryReleaseConditionType = 3
	WeaponStoryReleaseConditionTypeReachOnceEvolvedMaxLevel     WeaponStoryReleaseConditionType = 4
	WeaponStoryReleaseConditionTypeReachSpecifiedEvolutionCount WeaponStoryReleaseConditionType = 5
	WeaponStoryReleaseConditionTypeQuestClear                   WeaponStoryReleaseConditionType = 6
	WeaponStoryReleaseConditionTypeMainFlowSceneProgress        WeaponStoryReleaseConditionType = 7
)

type UserQuestStateType int32

const (
	UserQuestStateTypeUnknown    UserQuestStateType = 0
	UserQuestStateTypeActive     UserQuestStateType = 1
	UserQuestStateTypeCleared    UserQuestStateType = 2
	UserQuestStateTypeChallenged UserQuestStateType = 4
)

type QuestReleaseConditionType int32

const (
	QuestReleaseConditionTypeQuestClear     QuestReleaseConditionType = 4
	QuestReleaseConditionTypeQuestChallenge QuestReleaseConditionType = 7
)

type ConditionOperationType int32

const (
	ConditionOperationTypeAnd ConditionOperationType = 1
	ConditionOperationTypeOr  ConditionOperationType = 2
)

type SideStoryQuestStateType int32

const (
	SideStoryQuestStateUnknown SideStoryQuestStateType = 0
	SideStoryQuestStateActive  SideStoryQuestStateType = 2
	SideStoryQuestStateCleared SideStoryQuestStateType = 3
)

type SideStorySceneIdType int32

const (
	SideStorySceneInvalid                SideStorySceneIdType = 0
	SideStorySceneIntroduction           SideStorySceneIdType = 1
	SideStoryScenePlayGeneralQuest       SideStorySceneIdType = 2
	SideStorySceneUnlockLastQuest        SideStorySceneIdType = 3
	SideStoryScenePlayLastQuest          SideStorySceneIdType = 4
	SideStorySceneOutroduction           SideStorySceneIdType = 5
	SideStorySceneShowCostumeAcquisition SideStorySceneIdType = 6
	SideStoryScenePlayFreeQuest          SideStorySceneIdType = 7
)
