package giftbox

import (
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestSplitNotReceivedMaterialByPossessionLimit(t *testing.T) {
	config := &masterdata.GameConfig{PossessionCountLimitMaterial: 3}
	gift := store.NotReceivedGiftState{
		GiftCommon: store.GiftCommonState{
			PossessionType:        int32(model.PossessionTypeMaterial),
			PossessionId:          10,
			Count:                 7,
			GrantDatetime:         100,
			DescriptionGiftTextId: 20,
			EquipmentData:         []byte{1, 2},
		},
		ExpirationDatetime: 200,
		UserGiftUuid:       "original",
	}

	parts := SplitNotReceived(gift, config)
	if len(parts) != 3 {
		t.Fatalf("split gift count = %d, want 3", len(parts))
	}
	for i, want := range []int32{3, 3, 1} {
		if parts[i].GiftCommon.Count != want {
			t.Fatalf("part %d count = %d, want %d", i, parts[i].GiftCommon.Count, want)
		}
		if parts[i].GiftCommon.GrantDatetime != 100 || parts[i].GiftCommon.DescriptionGiftTextId != 20 || parts[i].ExpirationDatetime != 200 {
			t.Fatalf("part %d did not preserve gift metadata: %+v", i, parts[i])
		}
	}
	if parts[0].UserGiftUuid != "original" {
		t.Fatalf("first UUID = %q, want original", parts[0].UserGiftUuid)
	}
	if parts[1].UserGiftUuid == "" || parts[2].UserGiftUuid == "" || parts[1].UserGiftUuid == parts[2].UserGiftUuid {
		t.Fatalf("additional UUIDs are not unique: %q, %q", parts[1].UserGiftUuid, parts[2].UserGiftUuid)
	}
}

func TestSplitNotReceivedUsesConsumableAndMoneyLimits(t *testing.T) {
	config := &masterdata.GameConfig{
		ConsumableItemIdForGold:            99,
		PossessionCountLimitConsumableItem: 4,
		PossessionCountLimitMoney:          10,
	}
	consumable := store.NotReceivedGiftState{GiftCommon: store.GiftCommonState{
		PossessionType: int32(model.PossessionTypeConsumableItem),
		PossessionId:   98,
		Count:          9,
	}}
	parts := SplitNotReceived(consumable, config)
	if len(parts) != 3 || parts[0].GiftCommon.Count != 4 || parts[1].GiftCommon.Count != 4 || parts[2].GiftCommon.Count != 1 {
		t.Fatalf("consumable split = %+v, want counts [4 4 1]", parts)
	}

	gold := store.NotReceivedGiftState{GiftCommon: store.GiftCommonState{
		PossessionType: int32(model.PossessionTypeConsumableItem),
		PossessionId:   99,
		Count:          11,
	}}
	parts = SplitNotReceived(gold, config)
	if len(parts) != 2 || parts[0].GiftCommon.Count != 10 || parts[1].GiftCommon.Count != 1 {
		t.Fatalf("gold split = %+v, want counts [10 1]", parts)
	}
}

func TestAddNotReceivedLeavesOtherPossessionsWholeAndUpdatesNotification(t *testing.T) {
	user := &store.UserState{
		Gifts: store.GiftState{NotReceived: []store.NotReceivedGiftState{{UserGiftUuid: "existing"}}},
	}
	gift := store.NotReceivedGiftState{GiftCommon: store.GiftCommonState{
		PossessionType: int32(model.PossessionTypeWeapon),
		PossessionId:   1,
		Count:          5,
	}}

	AddNotReceived(user, gift, &masterdata.GameConfig{PossessionCountLimitMaterial: 1})
	if len(user.Gifts.NotReceived) != 2 || user.Gifts.NotReceived[1].GiftCommon.Count != 5 {
		t.Fatalf("not-received gifts = %+v, want unsplit weapon gift", user.Gifts.NotReceived)
	}
	if user.Notifications.GiftNotReceiveCount != 2 {
		t.Fatalf("gift notification count = %d, want 2", user.Notifications.GiftNotReceiveCount)
	}
}
