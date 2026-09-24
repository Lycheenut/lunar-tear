package questflow

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/vmihailenco/msgpack/v5"

	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/importantitem"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questdrop"
	"lunar-tear/server/internal/store"
)

func partsDropCampaign(t *testing.T, rate, count int32) *campaign.Catalog {
	t.Helper()
	installed := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	tables := map[string]any{
		"m_enhance_campaign":                 []masterdata.EntityMEnhanceCampaign{},
		"m_enhance_campaign_target_group":    []masterdata.EntityMEnhanceCampaignTargetGroup{},
		"m_beginner_campaign":                []masterdata.EntityMBeginnerCampaign{},
		"m_comeback_campaign":                []masterdata.EntityMComebackCampaign{},
		"m_quest_campaign_target_item_group": []masterdata.EntityMQuestCampaignTargetItemGroup{},
		"m_quest_campaign": []masterdata.EntityMQuestCampaign{
			{QuestCampaignId: 1, QuestCampaignTargetGroupId: 1, QuestCampaignEffectGroupId: 1, EndDatetime: 100000, TargetUserStatusType: int32(campaign.TargetUserStatusAll)},
		},
		"m_quest_campaign_target_group": []masterdata.EntityMQuestCampaignTargetGroup{
			{QuestCampaignTargetGroupId: 1, QuestCampaignTargetType: int32(campaign.QuestTargetWholeQuest)},
		},
		"m_quest_campaign_effect_group": []masterdata.EntityMQuestCampaignEffectGroup{
			{QuestCampaignEffectGroupId: 1, QuestCampaignEffectType: int32(campaign.QuestEffectDropRate), QuestCampaignEffectValue: rate},
			{QuestCampaignEffectGroupId: 1, QuestCampaignEffectType: int32(campaign.QuestEffectDropCount), QuestCampaignEffectValue: count},
		},
	}
	var body []byte
	header := make(map[string][2]int)
	for name, rows := range tables {
		data, err := msgpack.Marshal(rows)
		if err != nil {
			t.Fatal(err)
		}
		header[name] = [2]int{len(body), len(data)}
		body = append(body, data...)
	}
	raw, err := msgpack.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, body...)
	padding := aes.BlockSize - len(raw)%aes.BlockSize
	raw = append(raw, bytes.Repeat([]byte{byte(padding)}, padding)...)
	key, _ := hex.DecodeString("36436230313332314545356536624265")
	iv, _ := hex.DecodeString("45666341656634434165356536446141")
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(raw, raw)
	path := filepath.Join(t.TempDir(), "parts-campaign.bin.e")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := memorydb.Init(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := memorydb.Init(installed); err != nil {
			t.Fatal(err)
		}
	})
	catalog, err := campaign.Load()
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func partsDropHandler() *QuestHandler {
	catalog := &masterdata.QuestCatalog{
		QuestById: map[int32]masterdata.EntityMQuest{10: {QuestId: 10, QuestPickupRewardGroupId: 20, IsUsableSkipTicket: true}},
		BattleDropsByQuestId: map[int32][]masterdata.BattleDropInfo{
			10: {{QuestSceneId: 101, BattleDropCategoryId: 1}},
		},
		// The two high rarities share a reveal effect, but bonus rolls must keep
		// the actual parts rarity. Material rewards are not parts-pool entries.
		BattleDropEffectIdByRewardId: map[int32]int32{1001: 3, 1002: 3, 1003: 3, 1004: 3, 1005: 3},
		BattleDropRewardById: map[int32]masterdata.EntityMBattleDropReward{
			1001: {PossessionType: int32(model.PossessionTypeParts), PossessionId: 1, Count: 2},
			1002: {PossessionType: int32(model.PossessionTypeParts), PossessionId: 2, Count: 2},
			1003: {PossessionType: int32(model.PossessionTypeParts), PossessionId: 3, Count: 2},
			1004: {PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 4, Count: 2},
			1005: {PossessionType: int32(model.PossessionTypeParts), PossessionId: 5, Count: 1},
		},
		PickupRewardIdsByGroupId:          map[int32][]int32{20: {1001, 1002, 1002, 1002, 1003, 1004}},
		PickupRewardIdsByGroupAndEffectId: map[int32]map[int32][]int32{20: {3: {1001, 1002, 1002, 1002, 1003, 1004}}},
		PartsCatalog: &masterdata.PartsCatalog{
			PartsById: map[int32]masterdata.EntityMParts{
				1: {PartsId: 1, PartsGroupId: 1, RarityType: 40, PartsInitialLotteryId: 1},
				2: {PartsId: 2, PartsGroupId: 2, RarityType: 40, PartsInitialLotteryId: 1},
				3: {PartsId: 3, PartsGroupId: 3, RarityType: 30, PartsInitialLotteryId: 1},
				5: {PartsId: 5, PartsGroupId: 5, RarityType: 40, PartsInitialLotteryId: 1},
			},
		},
	}
	config := &masterdata.GameConfig{ConsumableItemIdForQuestSkipTicket: 7}
	return &QuestHandler{QuestCatalog: catalog, Granter: BuildGranter(catalog, config), Config: config}
}

func TestPartsDropRateRerollsWithinRarity(t *testing.T) {
	for _, configured := range []bool{false, true} {
		name := "master-data"
		if configured {
			name = "configured"
		}
		t.Run(name, func(t *testing.T) {
			h := partsDropHandler()
			h.Campaigns = partsDropCampaign(t, 1000, 1000)
			if configured {
				h.DropRewardsByQuestID = map[int32][]questdrop.Reward{10: {
					{BattleDropRewardID: 1001, Weight: 1},
					{BattleDropRewardID: 1002, Weight: 3},
					{BattleDropRewardID: 1003, Weight: 1},
					{BattleDropRewardID: 1004, Weight: 1},
				}}
			}
			user := store.SeedUserState(99, "parts", 1, model.ClientPlatform{})
			counts := map[int32]int{}
			var different bool
			for seed := int64(1); seed <= 1200; seed++ {
				plan := h.battleDropPlan(user, 10, seed)
				original := h.BattleDropRewardById[plan[0].BattleDropRewardId]
				drops := h.computeDropRewardsForRun(user, h.QuestById[10], campaign.QuestTarget{}, 1000, seed)
				if repeat := h.computeDropRewardsForRun(user, h.QuestById[10], campaign.QuestTarget{}, 1000, seed); !reflect.DeepEqual(drops, repeat) {
					t.Fatal("same run produced different rewards")
				}
				if original.PossessionType == int32(model.PossessionTypeMaterial) {
					if len(drops) != 1 || drops[0].Count != 8 {
						t.Fatalf("material multiplier changed: %+v", drops)
					}
					continue
				}
				if len(drops) != 2 || drops[0].PossessionId != original.PossessionId {
					t.Fatalf("drops = %+v, want original reward %+v plus an independent roll", drops, original)
				}
				rarity := h.PartsById[original.PossessionId].RarityType
				for _, drop := range drops {
					if drop.PossessionType != model.PossessionTypeParts || drop.Count != 4 || drop.RewardEffectId != plan[0].BattleDropEffectId || h.PartsById[drop.PossessionId].RarityType != rarity || drop.PossessionId == 5 {
						t.Fatalf("invalid parts reroll: %+v, original=%+v", drop, original)
					}
				}
				if rarity == 40 {
					counts[drops[1].PossessionId]++
					different = different || drops[0].PossessionId != drops[1].PossessionId
				}
			}
			if !different {
				t.Fatal("bonus rolls only duplicated the original parts")
			}
			if ratio := float64(counts[1]) / float64(counts[1]+counts[2]); math.Abs(ratio-0.25) > 0.06 {
				t.Fatalf("bonus roll weights changed: %v", counts)
			}
		})
	}
}

func TestPartsDropSkipGrantsEveryReward(t *testing.T) {
	h := partsDropHandler()
	h.Campaigns = partsDropCampaign(t, 1000, 1000)
	h.DropRewardsByQuestID = map[int32][]questdrop.Reward{10: {
		{BattleDropRewardID: 1001, Weight: 1}, {BattleDropRewardID: 1002, Weight: 1},
	}}
	user := store.SeedUserState(99, "parts", 1, model.ClientPlatform{})
	user.ConsumableItems[7] = 3
	user.Quests[10] = store.UserQuestState{QuestId: 10, QuestStateType: model.UserQuestStateTypeCleared}
	outcome, err := h.HandleQuestSkip(user, 10, int32(model.QuestTypeEvent), 0, 0, 3, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(user.Parts) != 24 || len(outcome.DropRewards) != 24 || user.ConsumableItems[7] != 0 {
		t.Fatalf("parts=%d rewards=%d tickets=%d, want 24, 24, 0", len(user.Parts), len(outcome.DropRewards), user.ConsumableItems[7])
	}
	for _, drop := range outcome.DropRewards {
		if drop.Count != 1 || drop.IsAutoSale {
			t.Fatalf("parts reward must represent one independently generated item: %+v", drop)
		}
	}
}

func TestPartsDropRewardsMatchIndependentlyRolledInventory(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	parts, err := masterdata.LoadPartsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	campaigns := partsDropCampaign(t, 1000, 1000)
	for _, mode := range []string{"finish", "skip"} {
		t.Run(mode, func(t *testing.T) {
			h := partsDropHandler()
			h.PartsCatalog = parts
			h.Granter = BuildGranter(h.QuestCatalog, h.Config)
			h.Campaigns = campaigns
			h.BattleDropRewardById[1001] = masterdata.EntityMBattleDropReward{
				PossessionType: int32(model.PossessionTypeParts), PossessionId: 16, Count: 1,
			}
			h.DropRewardsByQuestID = map[int32][]questdrop.Reward{10: {{BattleDropRewardID: 1001, Weight: 1}}}
			user := store.SeedUserState(99, "parts", 1, model.ClientPlatform{})
			user.Quests[10] = store.UserQuestState{QuestId: 10, QuestStateType: model.UserQuestStateTypeCleared, IsRewardGranted: true}
			user.ConsumableItems[7] = 16
			reported := map[int32]int{}
			reportedEquipment := map[string]int{}
			for range 16 {
				var outcome FinishOutcome
				if mode == "skip" {
					outcome, err = h.HandleQuestSkip(user, 10, int32(model.QuestTypeEvent), 0, 0, 1, 1000)
					if err != nil {
						t.Fatal(err)
					}
				} else {
					outcome = h.HandleEventQuestFinish(user, 0, 10, false, false, 1000)
				}
				if len(outcome.DropRewards) != 4 {
					t.Fatalf("rewards=%d, want 4 independently rolled copies", len(outcome.DropRewards))
				}
				for _, drop := range outcome.DropRewards {
					if drop.Count != 1 || drop.IsAutoSale {
						t.Fatalf("unexpected parts reward: %+v", drop)
					}
					reported[drop.PossessionId]++
					reportedEquipment[fmt.Sprintf("%d:%x", drop.PossessionId, drop.EquipmentData)]++
				}
			}
			actual := map[int32]int{}
			actualEquipment := map[string]int{}
			for uuid, part := range user.Parts {
				actual[part.PartsId]++
				actualEquipment[fmt.Sprintf("%d:%x", part.PartsId, partsRewardEquipmentData(user, uuid))]++
				var subCount int32
				for key, sub := range user.PartsStatusSubs {
					if key.UserPartsUuid == uuid {
						subCount++
						def := parts.PartsStatusSubById[sub.PartsStatusSubLotteryId]
						r := def.Initial
						if sub.StatusChangeValue < r.Min || sub.StatusChangeValue > r.Max || (sub.StatusChangeValue-r.Min)%r.Step != 0 || sub.StatusKindType != def.StatusKindType || sub.StatusCalculationType != def.StatusCalculationType {
							t.Fatalf("drop sub-status %+v does not match initial range %+v", sub, r)
						}
					}
				}
				if want := parts.PartsById[part.PartsId].PartsInitialLotteryId - 1; subCount != want {
					t.Fatalf("parts %d has %d sub statuses, rank requires %d", part.PartsId, subCount, want)
				}
			}
			if len(actual) < 2 {
				t.Fatal("all copies inherited the same rank")
			}
			if !reflect.DeepEqual(reported, actual) {
				t.Fatalf("reward ranks=%v, inventory ranks=%v", reported, actual)
			}
			if !reflect.DeepEqual(reportedEquipment, actualEquipment) {
				t.Fatal("reward equipment snapshots do not match the independently granted parts")
			}
		})
	}
}

func TestPartsRewardEquipmentDataClientWireFormat(t *testing.T) {
	user := store.SeedUserState(99, "parts", 1, model.ClientPlatform{})
	user.Parts["drop"] = store.PartsState{Level: 1, PartsStatusMainId: 24}
	user.PartsStatusSubs[store.PartsStatusSubKey{UserPartsUuid: "drop", StatusIndex: 2}] = store.PartsStatusSubState{
		StatusIndex: 2, PartsStatusSubLotteryId: 8, Level: 1, StatusKindType: 7, StatusCalculationType: 1, StatusChangeValue: 250,
	}
	user.PartsStatusSubs[store.PartsStatusSubKey{UserPartsUuid: "drop", StatusIndex: 1}] = store.PartsStatusSubState{
		StatusIndex: 1, PartsStatusSubLotteryId: 4, Level: 1, StatusKindType: 2, StatusCalculationType: 1, StatusChangeValue: 250,
	}
	user.PartsStatusSubs[store.PartsStatusSubKey{UserPartsUuid: "other", StatusIndex: 1}] = store.PartsStatusSubState{StatusIndex: 1}
	// apb.api.gift.Parts from the client protocol: level 1, main status 24,
	// then two PartsStatusSub messages with fields 1..6 in status-index order.
	const want = "080110181a0d0801100418012002280130fa011a0d0802100818012007280130fa01"
	if got := hex.EncodeToString(partsRewardEquipmentData(user, "drop")); got != want {
		t.Fatalf("equipment wire data = %s, want %s", got, want)
	}
}

func TestPartsDropImportantItemRateAddsDraws(t *testing.T) {
	if err := memorydb.Init(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")); err != nil {
		t.Fatal(err)
	}
	effects, err := importantitem.Load()
	if err != nil {
		t.Fatal(err)
	}
	h := partsDropHandler()
	h.ImportantItemEffects = effects
	h.PartsById[8] = masterdata.EntityMParts{PartsId: 8, PartsGroupId: 8, RarityType: 30}
	h.BattleDropRewardById[1001] = masterdata.EntityMBattleDropReward{PossessionType: int32(model.PossessionTypeParts), PossessionId: 3, Count: 2}
	h.BattleDropRewardById[1002] = masterdata.EntityMBattleDropReward{PossessionType: int32(model.PossessionTypeParts), PossessionId: 8, Count: 2}
	h.DropRewardsByQuestID = map[int32][]questdrop.Reward{10: {
		{BattleDropRewardID: 1001, Weight: 1}, {BattleDropRewardID: 1002, Weight: 1},
	}}
	user := store.SeedUserState(99, "parts", 1, model.ClientPlatform{})
	for _, test := range []struct {
		item, eventType int32
		wantDraws       int
	}{
		{0, 3, 1},
		{200012, 3, 2}, // +50%, retaining DropRateMul's existing rounding.
		{200019, 3, 2}, // +100%.
		{200019, 4, 1}, // Bonus does not target this event type.
	} {
		user.ImportantItems = map[int32]int32{test.item: 1}
		target := campaign.QuestTarget{QuestType: campaign.QuestTypeEventQuest, EventQuestType: test.eventType}
		drops := h.computeDropRewardsForRun(user, h.QuestById[10], target, 1787241600000, 1)
		if len(drops) != test.wantDraws {
			t.Fatalf("item=%d event=%d drops=%+v, want %d draws", test.item, test.eventType, drops, test.wantDraws)
		}
		for _, drop := range drops {
			if drop.Count != 2 {
				t.Fatalf("important-item drop rate multiplied quantity: %+v", drop)
			}
		}
	}
}

func TestPartsDropAutoSaleIsIndependentForEveryCopy(t *testing.T) {
	h := partsDropHandler()
	h.Granter.PartsVariantsByGroupRarity[1][40] = []int32{101, 102, 103, 104, 105}
	for rank := int32(1); rank <= 5; rank++ {
		h.Granter.PartsById[100+rank] = store.PartsRef{PartsGroupId: 1, RarityType: 40, PartsInitialLotteryId: rank}
	}
	h.Granter.PartsSellPriceL1ByRarity[40] = 100
	h.Granter.GoldConsumableItemId = 99
	user := store.SeedUserState(99, "parts", 1, model.ClientPlatform{})
	drops := h.grantDropRewards(user, []RewardGrant{
		{PossessionType: model.PossessionTypeParts, PossessionId: 101, Count: 256},
	}, map[int32]bool{40: true}, map[int32]bool{3: true}, 1000)
	var sold int
	for _, drop := range drops {
		if drop.Count != 1 {
			t.Fatalf("parts copies were combined: %+v", drop)
		}
		if drop.IsAutoSale {
			sold++
			if drop.PossessionId != 103 {
				t.Fatalf("sold wrong rank: %+v", drop)
			}
		}
	}
	if len(drops) != 256 || sold == 0 || len(user.Parts) == 0 || sold+len(user.Parts) != 256 || user.ConsumableItems[99] != int32(sold)*100 {
		t.Fatalf("drops=%d sold=%d kept=%d gold=%d", len(drops), sold, len(user.Parts), user.ConsumableItems[99])
	}
	for _, part := range user.Parts {
		if part.PartsId == 103 {
			t.Fatal("auto-sold rank was also added to inventory")
		}
	}
}
