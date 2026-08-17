package missionprogress

import "lunar-tear/server/internal/model"

type optionTargetSet struct {
	optionGroupIds []int32
	targetIds      []int32
}

// optionTargetSets is recovered from the localized mission descriptions and
// possession-name resources. Weapon entries include both the base and evolved
// master IDs because both IDs represent the same named weapon family.
var optionTargetSets = []optionTargetSet{
	{[]int32{197, 199}, []int32{25002}},
	{[]int32{200, 202, 203}, []int32{250011, 250012}},
	{[]int32{205, 206, 207, 208}, []int32{340031, 340032}},
	{[]int32{209, 210, 211, 212}, []int32{250141, 250142}},
	{[]int32{217, 218, 219}, []int32{22003}},
	{[]int32{220, 221, 222}, []int32{220061, 220062}},
	{[]int32{225, 227, 229}, []int32{310131, 310132}},
	{[]int32{226, 228, 230}, []int32{220071, 220072}},
	{[]int32{237, 238, 239}, []int32{25003}},
	{[]int32{240, 241, 242}, []int32{250151, 250152}},
	{[]int32{244, 246, 248}, []int32{330061, 330062}},
	{[]int32{245, 247, 249}, []int32{210021, 210022}},
	{[]int32{254, 255, 256}, []int32{21003}},
	{[]int32{257, 258, 259}, []int32{210181, 210182}},
	{[]int32{264, 265, 266}, []int32{23005}},
	{[]int32{267, 268, 269}, []int32{230151, 230152}},
	{[]int32{271, 273, 275}, []int32{320071, 320072}},
	{[]int32{272, 274, 276}, []int32{230121, 230122}},
	{[]int32{281, 282, 283}, []int32{25006}},
	{[]int32{284, 285, 286}, []int32{250171, 250172}},
	{[]int32{291, 292, 293}, []int32{21004}},
	{[]int32{294, 295, 296}, []int32{210191, 210192}},
	{[]int32{298, 300, 302}, []int32{340271, 340272}},
	{[]int32{299, 301, 303}, []int32{250181, 250182}},
	{[]int32{305, 307, 309}, []int32{320161, 320162}},
	{[]int32{306, 308, 310}, []int32{230171, 230172}},
	{[]int32{315, 316, 317}, []int32{35008}},
	{[]int32{318, 319, 320}, []int32{350181, 350182}},
	{[]int32{335, 336, 337, 455, 456}, []int32{22005}},
	{[]int32{338, 339, 340}, []int32{220161, 220162}},
	{[]int32{345, 346, 347}, []int32{24005}},
	{[]int32{348, 349, 350}, []int32{240201, 240202}},
	{[]int32{352, 353, 354}, []int32{350211, 350212}},
	{[]int32{359, 360, 361}, []int32{24006}},
	{[]int32{362, 363, 364}, []int32{240221, 240222}},
	{[]int32{366, 367, 368}, []int32{310281, 310282}},
	{[]int32{373, 374}, []int32{35010}},
	{[]int32{375, 376}, []int32{350221, 350222}},
	{[]int32{379, 380, 381}, []int32{320221, 320222}},
	{[]int32{386, 387}, []int32{35012}},
	{[]int32{388, 389}, []int32{350261, 350262}},
	{[]int32{397, 398, 399}, []int32{340371, 340372}},
	{[]int32{404, 405}, []int32{22006}},
	{[]int32{406, 407}, []int32{220181, 220182}},
	{[]int32{410, 411, 412}, []int32{330301, 330302}},
	{[]int32{426, 427}, []int32{31013}},
	{[]int32{428, 429}, []int32{310321, 310322}},
	{[]int32{438, 439, 440}, []int32{350291, 350292}},
	{[]int32{445, 446}, []int32{24007}},
	{[]int32{447, 448}, []int32{240241, 240242}},
	{[]int32{457, 458}, []int32{220191, 220192}},
	{[]int32{465, 466, 467}, []int32{310361, 310362}},
	{[]int32{470, 471, 472}, []int32{330371, 330372}},
	{[]int32{477, 478}, []int32{350331, 350332}},
	{[]int32{487, 488}, []int32{22008}},
	{[]int32{489, 490}, []int32{220211, 220212}},
	{[]int32{493, 494, 495}, []int32{310391, 310392}},
	{[]int32{505, 506, 507}, []int32{350351, 350352}},
	{[]int32{513, 514}, []int32{25007}},
	{[]int32{515, 516}, []int32{250221, 250222}},
	{[]int32{526, 527, 528}, []int32{34029}},
	{[]int32{529, 530}, []int32{340521, 340522}},
	{[]int32{533, 534, 535}, []int32{320361, 320362}},
	{[]int32{541, 543}, []int32{320371, 320372}},
	{[]int32{553, 554}, []int32{25008}},
	{[]int32{555, 556}, []int32{250231, 250232}},
	{[]int32{559, 560, 561}, []int32{340561, 340562}},
	{[]int32{566, 567}, []int32{22009}},
	{[]int32{568, 569}, []int32{220231, 220232}},
	{[]int32{572, 573, 574}, []int32{310491, 310492}},
}

// These option-group IDs are campaign-specific labels, not Gacha master IDs.
// The target IDs are recovered from each mission's link/banner master data.
var gachaTargetsByOption = map[int32][]int32{
	600:       {552001},
	900002:    {9000073},
	101120601: {209},
}

func knownOptionTargets(conditionType model.MissionClearConditionType, optionGroupId int32) ([]int32, bool) {
	if !isEquipmentTargetCondition(conditionType) || optionGroupId == 0 {
		return nil, false
	}
	return findOptionTargets(optionTargetSets, optionGroupId)
}

func findOptionTargets(sets []optionTargetSet, optionGroupId int32) ([]int32, bool) {
	if optionGroupId == 0 {
		return nil, false
	}
	for _, set := range sets {
		for _, id := range set.optionGroupIds {
			if id == optionGroupId {
				return set.targetIds, true
			}
		}
	}
	return nil, false
}

func isEquipmentTargetCondition(conditionType model.MissionClearConditionType) bool {
	switch conditionType {
	case model.MissionClearConditionTypeWeaponEnhanceByCount,
		model.MissionClearConditionTypeWeaponEvolveByCount,
		model.MissionClearConditionTypeWeaponEnhanceSkillByCount,
		model.MissionClearConditionTypeWeaponLimitBreakByCount,
		model.MissionClearConditionTypeCostumeActiveSkillEnhanceByCount,
		model.MissionClearConditionTypeCostumeLimitBreakByCount,
		model.MissionClearConditionTypeWeaponProtectByCount,
		model.MissionClearConditionTypeCostumeMaxLevel,
		model.MissionClearConditionTypeWeaponMaxLevel,
		model.MissionClearConditionTypeCostumeAwakenCount:
		return true
	default:
		return false
	}
}

func containsTarget(targetIds []int32, targetId int32) bool {
	for _, id := range targetIds {
		if id == targetId {
			return true
		}
	}
	return false
}
