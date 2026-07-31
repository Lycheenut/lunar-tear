package masterdata

import (
	"fmt"
	"strconv"

	"lunar-tear/server/internal/utils"
)

type GameConfig struct {
	ConsumableItemIdForGold               int32
	ConsumableItemIdForMedal              int32
	ConsumableItemIdForRareMedal          int32
	ConsumableItemIdForArenaCoin          int32
	ConsumableItemIdForExploreTicket      int32
	ConsumableItemIdForMomPoint           int32
	ConsumableItemIdForPremiumGachaTicket int32
	ConsumableItemIdForQuestSkipTicket    int32

	CharacterRebirthAvailableCount int32
	CharacterRebirthConsumeGold    int32

	CostumeAwakenAvailableCount     int32
	CostumeLimitBreakAvailableCount int32

	MaterialSameWeaponExpCoefficientPermil int32

	UserStaminaRecoverySecond int32
	RewardGachaDailyMaxCount  int32
	QuestSkipMaxCountAtOnce   int32

	WeaponLimitBreakAvailableCount int32

	CostumeLotteryEffectUnlockSlotConsumeGold int32
	CostumeLotteryEffectDrawSlotConsumeGold   int32

	PossessionCountLimitMaterial       int32
	PossessionCountLimitConsumableItem int32
	PossessionCountLimitMoney          int32
	PossessionCountLimitWeapon         int32
	PossessionCountLimitParts          int32
	PossessionCountLimitImportantItem  int32
}

func LoadGameConfig() (*GameConfig, error) {
	rows, err := utils.ReadTable[EntityMConfig]("m_config")
	if err != nil {
		return nil, fmt.Errorf("load config table: %w", err)
	}

	kv := make(map[string]string, len(rows))
	for _, r := range rows {
		kv[r.ConfigKey] = r.Value
	}

	cfg := &GameConfig{}

	cfg.ConsumableItemIdForGold = parseInt32(kv, "CONSUMABLE_ITEM_ID_FOR_GOLD")
	cfg.ConsumableItemIdForMedal = parseInt32(kv, "CONSUMABLE_ITEM_ID_FOR_MEDAL")
	cfg.ConsumableItemIdForRareMedal = parseInt32(kv, "CONSUMABLE_ITEM_ID_FOR_RARE_MEDAL")
	cfg.ConsumableItemIdForArenaCoin = parseInt32(kv, "CONSUMABLE_ITEM_ID_FOR_ARENA_COIN")
	cfg.ConsumableItemIdForExploreTicket = parseInt32(kv, "CONSUMABLE_ITEM_ID_FOR_EXPLORE_TICKET")
	cfg.ConsumableItemIdForMomPoint = parseInt32(kv, "CONSUMABLE_ITEM_ID_FOR_MOM_POINT")
	cfg.ConsumableItemIdForPremiumGachaTicket = parseInt32(kv, "CONSUMABLE_ITEM_ID_FOR_PREMIUM_GACHA_TICKET")
	cfg.ConsumableItemIdForQuestSkipTicket = parseInt32(kv, "CONSUMABLE_ITEM_ID_FOR_QUEST_SKIP_TICKET")

	cfg.CharacterRebirthAvailableCount = parseInt32(kv, "CHARACTER_REBIRTH_AVAILABLE_COUNT")
	cfg.CharacterRebirthConsumeGold = parseInt32(kv, "CHARACTER_REBIRTH_CONSUME_GOLD")

	cfg.CostumeAwakenAvailableCount = parseInt32(kv, "COSTUME_AWAKEN_AVAILABLE_COUNT")
	cfg.CostumeLimitBreakAvailableCount = parseInt32(kv, "COSTUME_LIMIT_BREAK_AVAILABLE_COUNT")

	cfg.MaterialSameWeaponExpCoefficientPermil = parseInt32(kv, "MATERIAL_SAME_WEAPON_EXP_COEFFICIENT_PERMIL")

	cfg.UserStaminaRecoverySecond = parseInt32(kv, "USER_STAMINA_RECOVERY_SECOND")
	cfg.RewardGachaDailyMaxCount = parseInt32(kv, "REWARD_GACHA_DAILY_MAX_COUNT")
	cfg.QuestSkipMaxCountAtOnce = parseInt32(kv, "QUEST_SKIP_MAX_COUNT_AT_ONCE")

	cfg.WeaponLimitBreakAvailableCount = parseInt32(kv, "WEAPON_LIMIT_BREAK_AVAILABLE_COUNT")

	cfg.CostumeLotteryEffectUnlockSlotConsumeGold = parseInt32(kv, "COSTUME_LOTTERY_EFFECT_UNLOCK_SLOT_CONSUME_GOLD")
	cfg.CostumeLotteryEffectDrawSlotConsumeGold = parseInt32(kv, "COSTUME_LOTTERY_EFFECT_DRAW_SLOT_CONSUME_GOLD")

	cfg.PossessionCountLimitMaterial = parseInt32(kv, "POSSESSION_COUNT_LIMIT_MATERIAL")
	cfg.PossessionCountLimitConsumableItem = parseInt32(kv, "POSSESSION_COUNT_LIMIT_CONSUMABLE_ITEM")
	cfg.PossessionCountLimitMoney = parseInt32(kv, "POSSESSION_COUNT_LIMIT_MONEY")
	cfg.PossessionCountLimitWeapon = parseInt32(kv, "POSSESSION_COUNT_LIMIT_WEAPON")
	cfg.PossessionCountLimitParts = parseInt32(kv, "POSSESSION_COUNT_LIMIT_PARTS")
	cfg.PossessionCountLimitImportantItem = parseInt32(kv, "POSSESSION_COUNT_LIMIT_IMPORTANT_ITEM")

	return cfg, nil
}

func parseInt32(kv map[string]string, key string) int32 {
	s, ok := kv[key]
	if !ok {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0
	}
	return int32(v)
}
