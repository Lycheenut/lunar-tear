package model

const MaterialIdDarkKarmaStone int32 = 315002

type MaterialType int32

const (
	MaterialTypeUnknown                    MaterialType = 0
	MaterialTypeWeaponEnhancement          MaterialType = 10
	MaterialTypeCostumeEnhancement         MaterialType = 20
	MaterialTypeCompanionEnhancement       MaterialType = 30
	MaterialTypeWeaponSkillEnhancement     MaterialType = 40
	MaterialTypeCostumeSkillEnhancement    MaterialType = 50
	MaterialTypeCommonSkillEnhancement     MaterialType = 60
	MaterialTypeWeaponEvolution            MaterialType = 70
	MaterialTypeWeaponLimitBreak           MaterialType = 80
	MaterialTypeCostumeLimitBreak          MaterialType = 90
	MaterialTypeCharacterBoardRelease      MaterialType = 100
	MaterialTypeCostumeAwaken              MaterialType = 110
	MaterialTypeCharacterRebirth           MaterialType = 120
	MaterialTypeWeaponAwaken               MaterialType = 130
	MaterialTypeCostumeLotteryEffectUnlock MaterialType = 140
)
