package giftbox

import (
	"github.com/google/uuid"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

// AddNotReceived adds a gift to the gift box, splitting stackable possessions
// so that every record can be claimed without exceeding its possession limit.
func AddNotReceived(user *store.UserState, gift store.NotReceivedGiftState, config *masterdata.GameConfig) {
	user.Gifts.NotReceived = append(user.Gifts.NotReceived, SplitNotReceived(gift, config)...)
	user.Notifications.GiftNotReceiveCount = int32(len(user.Gifts.NotReceived))
}

// SplitNotReceived preserves the original UUID on the first record and assigns
// new UUIDs to any additional records.
func SplitNotReceived(gift store.NotReceivedGiftState, config *masterdata.GameConfig) []store.NotReceivedGiftState {
	if gift.UserGiftUuid == "" {
		gift.UserGiftUuid = uuid.NewString()
	}
	limit := CountLimit(gift.GiftCommon.PossessionType, gift.GiftCommon.PossessionId, config)
	if limit <= 0 || gift.GiftCommon.Count <= limit {
		return []store.NotReceivedGiftState{gift}
	}

	count := gift.GiftCommon.Count
	parts := make([]store.NotReceivedGiftState, 0, (count-1)/limit+1)
	for count > 0 {
		part := gift
		part.GiftCommon.Count = min(count, limit)
		if len(parts) > 0 {
			part.UserGiftUuid = uuid.NewString()
		}
		parts = append(parts, part)
		count -= part.GiftCommon.Count
	}
	return parts
}

// CountLimit returns the per-record limit for stackable gift-box possessions.
// Gold is a consumable item in the protocol, but uses the separate money cap.
func CountLimit(possessionType, possessionId int32, config *masterdata.GameConfig) int32 {
	if config == nil {
		return 0
	}
	switch model.PossessionType(possessionType) {
	case model.PossessionTypeMaterial:
		return config.PossessionCountLimitMaterial
	case model.PossessionTypeConsumableItem:
		if possessionId == config.ConsumableItemIdForGold {
			return config.PossessionCountLimitMoney
		}
		return config.PossessionCountLimitConsumableItem
	default:
		return 0
	}
}
