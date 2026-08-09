package campaign

import "testing"

func TestEnhanceRateUsesBestMatchingCampaignWithoutStacking(t *testing.T) {
	c := &Catalog{enhance: []enhanceRow{
		activeEnhanceRow(EnhanceEffectAdditionalPerm, 40, EnhanceTargetPartsAll, 0),
		activeEnhanceRow(EnhanceEffectAdditionalPerm, 80, EnhanceTargetPartsSeriesId, 7),
		activeEnhanceRow(EnhanceEffectProbability, 60, EnhanceTargetPartsId, 9),
	}}

	got := c.PartsRateBonus(PartsTarget{PartsId: 9, PartsGroupId: 7}, activeFilter()).Apply(20)
	if got != 100 {
		t.Fatalf("success rate = %d, want 100", got)
	}
}

func TestEnhanceRateTargetsCostumeAndWeaponGreatSuccess(t *testing.T) {
	c := &Catalog{enhance: []enhanceRow{
		activeEnhanceRow(EnhanceEffectAdditionalPerm, 40, EnhanceTargetCostumeCharacterId, 10),
		activeEnhanceRow(EnhanceEffectProbability, 80, EnhanceTargetWeaponAttributeTypeId, 20),
	}}

	if got := c.CostumeRateBonus(CostumeTarget{CharacterId: 10}, activeFilter()).Apply(20); got != 60 {
		t.Fatalf("costume great-success rate = %d, want 60", got)
	}
	if got := c.WeaponRateBonus(WeaponTarget{AttributeType: 20}, activeFilter()).Apply(20); got != 80 {
		t.Fatalf("weapon great-success rate = %d, want 80", got)
	}
}

func TestQuestCampaignModifiersUseStrongestMatchingEffect(t *testing.T) {
	c := &Catalog{quest: []questRow{
		activeQuestRow(QuestEffectDropRate, 500),
		activeQuestRow(QuestEffectDropRate, 1000),
		activeQuestRow(QuestEffectDropCount, 500),
		activeQuestRow(QuestEffectStaminaConsume, 500),
		activeQuestRow(QuestEffectClearRewardGold, 2000),
	}}
	target := QuestTarget{QuestId: 10, QuestType: QuestTypeMainQuest}
	filter := activeFilter()

	if got := c.QuestDropRate(target, filter).Apply(3); got != 6 {
		t.Fatalf("drop-rate result = %d, want 6", got)
	}
	if got := c.QuestDropCount(target, filter).Apply(3); got != 4 {
		t.Fatalf("drop-count result = %d, want 4", got)
	}
	if got := c.QuestStamina(target, filter).Apply(11); got != 5 {
		t.Fatalf("stamina result = %d, want 5", got)
	}
	if got := c.QuestGold(target, filter).Apply(10); got != 30 {
		t.Fatalf("gold result = %d, want 30", got)
	}
}

func activeEnhanceRow(effect EnhanceCampaignEffectType, value int32, target EnhanceCampaignTargetType, targetValue int32) enhanceRow {
	return enhanceRow{
		effectType: effect, effectValue: value,
		targets:     []enhanceMatch{{t: target, v: targetValue}},
		startMillis: 1, endMillis: 200, userStatus: TargetUserStatusAll,
	}
}

func activeQuestRow(effect QuestCampaignEffectType, value int32) questRow {
	return questRow{
		effectType: effect, effectValue: value,
		targets:     []questMatch{{t: QuestTargetWholeQuest}},
		startMillis: 1, endMillis: 200, userStatus: TargetUserStatusAll,
	}
}

func activeFilter() Filter {
	return Filter{NowMillis: 100, UserStatus: TargetUserStatusAll}
}
