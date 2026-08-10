package campaign

import (
	"path/filepath"
	"testing"
	"time"

	"lunar-tear/server/internal/masterdata/memorydb"
)

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

func TestEnhanceRateUsesBeginnerAndComebackUserStatus(t *testing.T) {
	now := int64(100) * millisecondsPerDay
	c := &Catalog{
		enhance: []enhanceRow{
			statusEnhanceRow(EnhanceEffectAdditionalPerm, 80, EnhanceTargetCostumeAll, TargetUserStatusBeginner),
			statusEnhanceRow(EnhanceEffectAdditionalPerm, 40, EnhanceTargetCostumeAll, TargetUserStatusComeback),
		},
		beginner: []beginnerRow{{
			judgeStartMillis: 1,
			judgeEndMillis:   200 * millisecondsPerDay,
			grantDays:        7,
			unlockQuestId:    1,
		}},
		comeback: []comebackRow{{
			judgeStartMillis: 1,
			judgeEndMillis:   200 * millisecondsPerDay,
			judgeDays:        28,
			grantDays:        7,
			unlockQuestId:    1,
		}},
	}
	unlocked := func(questId int32) bool { return questId == 1 }

	beginnerFilter := c.FilterForUser(UserStatusContext{
		NowMillis:                    now,
		RegisterDatetime:             now - millisecondsPerDay,
		IsCampaignUnlockQuestCleared: unlocked,
	})
	if got := c.CostumeRateBonus(CostumeTarget{}, beginnerFilter).Apply(20); got != 100 {
		t.Fatalf("beginner great-success rate = %d, want 100", got)
	}

	comebackFilter := c.FilterForUser(UserStatusContext{
		NowMillis:                    now,
		RegisterDatetime:             now - 100*millisecondsPerDay,
		LastComebackLoginDatetime:    now - millisecondsPerDay,
		IsCampaignUnlockQuestCleared: unlocked,
	})
	if got := c.CostumeRateBonus(CostumeTarget{}, comebackFilter).Apply(20); got != 60 {
		t.Fatalf("comeback great-success rate = %d, want 60", got)
	}

	ordinaryFilter := c.FilterForUser(UserStatusContext{
		NowMillis:        now,
		RegisterDatetime: now - 100*millisecondsPerDay,
	})
	if got := c.CostumeRateBonus(CostumeTarget{}, ordinaryFilter).Apply(20); got != 20 {
		t.Fatalf("ordinary great-success rate = %d, want 20", got)
	}
}

func TestComebackLoginUsesJudgePeriodAndAbsenceDays(t *testing.T) {
	now := int64(100) * millisecondsPerDay
	c := &Catalog{comeback: []comebackRow{{
		judgeStartMillis: 1,
		judgeEndMillis:   200 * millisecondsPerDay,
		judgeDays:        28,
		grantDays:        7,
		unlockQuestId:    1,
	}}}
	unlocked := func(questId int32) bool { return questId == 1 }

	if !c.IsComebackLogin(now, now-28*millisecondsPerDay, unlocked) {
		t.Fatal("28-day absence was not recognized as a comeback")
	}
	if c.IsComebackLogin(now, now-27*millisecondsPerDay, unlocked) {
		t.Fatal("27-day absence was recognized as a comeback")
	}
	if c.IsComebackLogin(now, now-28*millisecondsPerDay, func(int32) bool { return false }) {
		t.Fatal("locked comeback campaign was recognized")
	}
	if c.IsComebackLogin(201*millisecondsPerDay, now, unlocked) {
		t.Fatal("login outside the comeback judge period was recognized")
	}
}

func TestCurrentMasterDataEnhanceCampaignUserStatus(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatalf("init master data: %v", err)
	}
	c, err := Load()
	if err != nil {
		t.Fatalf("load campaigns: %v", err)
	}
	now := time.Date(2026, time.August, 10, 13, 30, 0, 0, time.UTC).UnixMilli()
	unlocked := func(questId int32) bool { return questId == 1 }

	beginnerFilter := c.FilterForUser(UserStatusContext{
		NowMillis:                    now,
		RegisterDatetime:             now - millisecondsPerDay,
		IsCampaignUnlockQuestCleared: unlocked,
	})
	if got := c.CostumeRateBonus(CostumeTarget{}, beginnerFilter).Apply(20); got != 100 {
		t.Fatalf("current beginner costume rate = %d, want 100", got)
	}
	if got := c.WeaponRateBonus(WeaponTarget{}, beginnerFilter).Apply(20); got != 100 {
		t.Fatalf("current beginner weapon rate = %d, want 100", got)
	}

	comebackFilter := c.FilterForUser(UserStatusContext{
		NowMillis:                    now,
		RegisterDatetime:             now - 100*millisecondsPerDay,
		LastComebackLoginDatetime:    now - millisecondsPerDay,
		IsCampaignUnlockQuestCleared: unlocked,
	})
	if got := c.CostumeRateBonus(CostumeTarget{}, comebackFilter).Apply(20); got != 60 {
		t.Fatalf("current comeback costume rate = %d, want 60", got)
	}
	if got := c.WeaponRateBonus(WeaponTarget{}, comebackFilter).Apply(20); got != 60 {
		t.Fatalf("current comeback weapon rate = %d, want 60", got)
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

func statusEnhanceRow(effect EnhanceCampaignEffectType, value int32, target EnhanceCampaignTargetType, status TargetUserStatusType) enhanceRow {
	return enhanceRow{
		effectType: effect, effectValue: value,
		targets:     []enhanceMatch{{t: target}},
		startMillis: 1, endMillis: 200 * millisecondsPerDay, userStatus: status,
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
