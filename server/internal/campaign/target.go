package campaign

import "lunar-tear/server/internal/model"

type EnhanceCampaignEffectType int32

const (
	EnhanceEffectUnknown        EnhanceCampaignEffectType = 0
	EnhanceEffectProbability    EnhanceCampaignEffectType = 1
	EnhanceEffectAdditionalPerm EnhanceCampaignEffectType = 2
)

type EnhanceCampaignTargetType int32

const (
	EnhanceTargetUnknown               EnhanceCampaignTargetType = 0
	EnhanceTargetCostumeAll            EnhanceCampaignTargetType = 1
	EnhanceTargetWeaponAll             EnhanceCampaignTargetType = 2
	EnhanceTargetPartsAll              EnhanceCampaignTargetType = 3
	EnhanceTargetCostumeCharacterId    EnhanceCampaignTargetType = 11
	EnhanceTargetCostumeSkillfulWeapon EnhanceCampaignTargetType = 12
	EnhanceTargetCostumeId             EnhanceCampaignTargetType = 13
	EnhanceTargetWeaponTypeId          EnhanceCampaignTargetType = 21
	EnhanceTargetWeaponAttributeTypeId EnhanceCampaignTargetType = 22
	EnhanceTargetWeaponId              EnhanceCampaignTargetType = 23
	EnhanceTargetPartsSeriesId         EnhanceCampaignTargetType = 31
	EnhanceTargetPartsId               EnhanceCampaignTargetType = 32
)

type QuestCampaignEffectType int32

const (
	QuestEffectUnknown         QuestCampaignEffectType = 0
	QuestEffectDropRate        QuestCampaignEffectType = 1
	QuestEffectDropCount       QuestCampaignEffectType = 2
	QuestEffectStaminaConsume  QuestCampaignEffectType = 3
	QuestEffectClearRewardGold QuestCampaignEffectType = 4
	QuestEffectDropItemAdd     QuestCampaignEffectType = 5
)

type QuestCampaignTargetType int32

const (
	QuestTargetUnknown            QuestCampaignTargetType = 0
	QuestTargetWholeQuest         QuestCampaignTargetType = 1
	QuestTargetQuestType          QuestCampaignTargetType = 2
	QuestTargetEventQuestType     QuestCampaignTargetType = 3
	QuestTargetMainQuestChapterId QuestCampaignTargetType = 4
	QuestTargetMainQuestQuestId   QuestCampaignTargetType = 5
	QuestTargetSubQuestChapterId  QuestCampaignTargetType = 6
	QuestTargetSubQuestQuestId    QuestCampaignTargetType = 7
)

type QuestType int32

const (
	QuestTypeUnknown    QuestType = 0
	QuestTypeMainQuest  QuestType = 1
	QuestTypeEventQuest QuestType = 2
	QuestTypeExtraQuest QuestType = 3
	QuestTypeBigHunt    QuestType = 4
)

type TargetUserStatusType int32

const (
	TargetUserStatusUnknown  TargetUserStatusType = 0
	TargetUserStatusAll      TargetUserStatusType = 1
	TargetUserStatusComeback TargetUserStatusType = 2
	TargetUserStatusBeginner TargetUserStatusType = 3
)

type Filter struct {
	NowMillis  int64
	UserStatus TargetUserStatusType
}

type PartsTarget struct {
	PartsId      int32
	PartsGroupId int32
	Rarity       model.RarityType
}

type CostumeTarget struct {
	CostumeId          int32
	CharacterId        int32
	SkillfulWeaponType int32
}

type WeaponTarget struct {
	WeaponId      int32
	WeaponType    int32
	AttributeType int32
}

type QuestTarget struct {
	QuestId                 int32
	QuestType               QuestType
	EventQuestType          int32
	ChapterId               int32
	MainQuestDifficultyType int32
}
