package questflow

import (
	"fmt"
	"reflect"
	"testing"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questdrop"
	"lunar-tear/server/internal/store"
)

func TestParseAutoSaleRulesClientRanks(t *testing.T) {
	for _, test := range []struct {
		value string
		want  map[int32]bool
	}{
		{"0,1,2,3,4", map[int32]bool{1: true, 2: true, 3: true, 4: true, 5: true}},
		{"0", map[int32]bool{1: true}},
		{"1", map[int32]bool{2: true}},
		{"2", map[int32]bool{3: true}},
		{"3", map[int32]bool{4: true}},
		{"4", map[int32]bool{5: true}},
		{"", map[int32]bool{}},
		{"5,6,9", map[int32]bool{}},
	} {
		t.Run(test.value, func(t *testing.T) {
			raritySet, rankSet := parseAutoSaleRules(map[int32]store.AutoSaleSettingState{
				1: {PossessionAutoSaleItemType: 1, PossessionAutoSaleItemValue: "10,20,30"},
				2: {PossessionAutoSaleItemType: 2, PossessionAutoSaleItemValue: test.value},
			})
			if !reflect.DeepEqual(raritySet, map[int32]bool{10: true, 20: true, 30: true}) {
				t.Fatalf("rarities = %v, want 10, 20, 30", raritySet)
			}
			if !reflect.DeepEqual(rankSet, test.want) {
				t.Fatalf("ranks = %v, want %v", rankSet, test.want)
			}
		})
	}
}

func TestPartsDropAutoSaleClientSettingsWithCountCampaign(t *testing.T) {
	for _, multiplier := range []int32{1, 5} {
		t.Run(fmt.Sprintf("x%d", multiplier), func(t *testing.T) {
			campaigns := partsDropCampaign(t, 0, (multiplier-1)*1000)
			for _, mode := range []string{"finish", "skip"} {
				for _, rarity := range []int32{10, 20, 30, 40} {
					for rank := int32(1); rank <= 5; rank++ {
						t.Run(fmt.Sprintf("%s/rarity%d/rank%d", mode, rarity, rank), func(t *testing.T) {
							h := partsDropHandler()
							h.Campaigns = campaigns
							h.BattleDropRewardById[1003] = masterdata.EntityMBattleDropReward{
								PossessionType: int32(model.PossessionTypeParts), PossessionId: 3, Count: 1,
							}
							h.DropRewardsByQuestID = map[int32][]questdrop.Reward{10: {{BattleDropRewardID: 1003, Weight: 1}}}
							// A single variant makes every rank's sale outcome deterministic.
							h.Granter.PartsById[3] = store.PartsRef{PartsGroupId: 3, RarityType: rarity, PartsInitialLotteryId: rank}
							h.Granter.PartsSellPriceL1ByRarity[rarity] = 100
							h.Granter.GoldConsumableItemId = 99
							user := store.SeedUserState(99, "parts", 1, model.ClientPlatform{})
							user.AutoSaleSettings = map[int32]store.AutoSaleSettingState{
								1: {PossessionAutoSaleItemType: 1, PossessionAutoSaleItemValue: "10,20,30"},
								2: {PossessionAutoSaleItemType: 2, PossessionAutoSaleItemValue: "0,1,2,3,4"},
							}
							user.Quests[10] = store.UserQuestState{QuestId: 10, QuestStateType: model.UserQuestStateTypeCleared, IsRewardGranted: true}
							user.ConsumableItems[7] = 1
							var outcome FinishOutcome
							if mode == "skip" {
								var err error
								outcome, err = h.HandleQuestSkip(user, 10, int32(model.QuestTypeEvent), 0, 0, 1, 1000)
								if err != nil {
									t.Fatal(err)
								}
							} else {
								outcome = h.HandleEventQuestFinish(user, 0, 10, false, false, 1000)
							}
							wantSold := rarity != 40
							wantKept, wantGold := int(multiplier), int32(0)
							if wantSold {
								wantKept, wantGold = 0, multiplier*100
							}
							if len(user.Parts) != wantKept || user.ConsumableItems[99] != wantGold {
								t.Fatalf("kept=%d gold=%d, want kept=%d gold=%d", len(user.Parts), user.ConsumableItems[99], wantKept, wantGold)
							}
							if len(outcome.DropRewards) != int(multiplier) {
								t.Fatalf("rewards=%d, want %d", len(outcome.DropRewards), multiplier)
							}
							for _, drop := range outcome.DropRewards {
								if drop.Count != 1 || drop.IsAutoSale != wantSold {
									t.Fatalf("drop=%+v, want count=1 autoSale=%v", drop, wantSold)
								}
							}
							if wantSold && len(user.PartsStatusSubs) != 0 {
								t.Fatalf("auto-sold parts left %d sub-status rows", len(user.PartsStatusSubs))
							}
						})
					}
				}
			}
		})
	}
}
