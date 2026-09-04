package gacha

import (
	"math"
	"testing"

	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestHandleDrawRejectsMissingPhaseAndInsufficientPrice(t *testing.T) {
	banner := &PremiumBannerPool{
		GachaId: 1,
		Groups: []PremiumGroup{{
			Id:        GroupWeaponOnly4,
			GrantType: GrantWeaponOnly,
			Rarity:    model.RaritySSRare,
			Weight:    1,
			NonPickup: []PoolItem{{WeaponId: 1, RarityType: model.RaritySSRare}},
		}},
	}
	h := &GachaHandler{Premium: &PremiumCatalog{Banners: map[int32]*PremiumBannerPool{1: banner}}}
	entry := store.GachaCatalogEntry{GachaId: 1, GachaLabelType: model.GachaLabelPremium, PricePhases: []store.GachaPricePhaseEntry{{PhaseId: 10, PriceType: model.PriceTypeGem, Price: 100, DrawCount: 1}}}
	user := &store.UserState{}
	user.EnsureMaps()
	if _, err := h.HandleDraw(user, entry, 999, 1); err == nil {
		t.Fatal("missing phase was accepted")
	}
	if _, err := h.HandleDraw(user, entry, 10, 1); err == nil {
		t.Fatal("insufficient gems were accepted")
	}
	if user.Gacha.BannerStates[1].DrawCount != 0 {
		t.Fatal("failed draw changed banner state")
	}
	if user.Gem != (store.UserGemState{}) {
		t.Fatal("failed draw changed gem balance")
	}
}

func TestHandleDrawRejectsOversizedRequestWithoutMutation(t *testing.T) {
	h := &GachaHandler{Pool: &masterdata.GachaCatalog{}, Granter: &store.PossessionGranter{}}
	entry := eventBoxEntry()
	entry.PricePhases[0].DrawCount = 10
	user := &store.UserState{}
	user.EnsureMaps()
	if _, err := h.HandleDraw(user, entry, entry.PricePhases[0].PhaseId, math.MaxInt32); err == nil {
		t.Fatal("oversized draw request was accepted")
	}
	if user.Materials[99] != 0 || len(user.Gacha.BannerStates) != 0 {
		t.Fatal("rejected draw mutated user state")
	}
}

func TestHandleDrawConsumesGuaranteedTickets(t *testing.T) {
	tests := []struct {
		name     string
		gachaId  int32
		ticketId int32
		rarity   model.RarityType
		weaponId int32
		groups   []PremiumGroup
	}{
		{
			name:     "three-star or higher",
			gachaId:  model.GachaIdGuaranteedThreeStarOrHigher,
			ticketId: model.ConsumableIdGuaranteedThreeStarOrHigherTicket,
			rarity:   model.RaritySRare,
			weaponId: 123,
			groups: []PremiumGroup{
				{Id: GroupWeaponOnly3, GrantType: GrantWeaponOnly, Star: 3, Rarity: model.RaritySRare, Weight: 1, NonPickup: []PoolItem{{WeaponId: 123, RarityType: model.RaritySRare}}},
				{Id: GroupWeaponOnly2, GrantType: GrantWeaponOnly, Star: 2, Rarity: model.RarityRare, Weight: 100, NonPickup: []PoolItem{{WeaponId: 122, RarityType: model.RarityRare}}},
			},
		},
		{
			name:     "four-star",
			gachaId:  model.GachaIdGuaranteedFourStar,
			ticketId: model.ConsumableIdGuaranteedFourStarTicket,
			rarity:   model.RaritySSRare,
			weaponId: 124,
			groups: []PremiumGroup{{
				Id: GroupWeaponOnly4, GrantType: GrantWeaponOnly, Star: 4, Rarity: model.RaritySSRare, Weight: 1, NonPickup: []PoolItem{{WeaponId: 124, RarityType: model.RaritySSRare}},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			banner := &PremiumBannerPool{GachaId: tt.gachaId, Groups: tt.groups}
			h := &GachaHandler{
				Premium: &PremiumCatalog{Banners: map[int32]*PremiumBannerPool{tt.gachaId: banner}},
				Granter: &store.PossessionGranter{},
			}
			phaseId := tt.gachaId*model.PhaseIdMultiplier + 1
			entry := store.GachaCatalogEntry{
				GachaId:        tt.gachaId,
				GachaLabelType: model.GachaLabelPremium,
				PricePhases: []store.GachaPricePhaseEntry{{
					PhaseId:        phaseId,
					PriceType:      model.PriceTypeConsumableItem,
					PriceId:        tt.ticketId,
					Price:          1,
					DrawCount:      1,
					FixedRarityMin: tt.rarity,
					FixedCount:     1,
				}},
			}
			user := &store.UserState{}
			user.EnsureMaps()
			user.ConsumableItems[tt.ticketId] = 1

			result, err := h.HandleDraw(user, entry, phaseId, 1)
			if err != nil {
				t.Fatal(err)
			}
			if user.ConsumableItems[tt.ticketId] != 0 {
				t.Fatalf("ticket %d was not consumed", tt.ticketId)
			}
			if len(result.Items) != 1 || result.Items[0].PossessionId != tt.weaponId || result.Items[0].RarityType != tt.rarity {
				t.Fatalf("draw result = %+v, want rarity %d weapon %d", result.Items, tt.rarity, tt.weaponId)
			}
		})
	}
}

func TestHandleDrawLimitsDailyGachaToOneExecutionPerBusinessDay(t *testing.T) {
	banner := &PremiumBannerPool{
		GachaId: model.GachaIdDaily,
		Groups: []PremiumGroup{{
			Id:        GroupWeaponOnly2,
			GrantType: GrantWeaponOnly,
			Rarity:    model.RarityRare,
			Weight:    GroupWeightTotal,
			NonPickup: []PoolItem{{WeaponId: 1, RarityType: model.RarityRare}},
		}},
	}
	h := &GachaHandler{
		Premium: &PremiumCatalog{Banners: map[int32]*PremiumBannerPool{model.GachaIdDaily: banner}},
		Granter: &store.PossessionGranter{},
	}
	phaseId := model.GachaIdDaily*model.PhaseIdMultiplier + 1
	entry := store.GachaCatalogEntry{
		GachaId:            model.GachaIdDaily,
		GachaLabelType:     model.GachaLabelPremium,
		GachaModeType:      model.GachaModeBasic,
		GachaAutoResetType: model.GachaAutoResetDaily,
		PricePhases: []store.GachaPricePhaseEntry{{
			PhaseId:        phaseId,
			DrawCount:      model.DailyGachaDrawCount,
			LimitExecCount: model.DailyGachaExecLimit,
		}},
	}
	user := &store.UserState{}
	user.EnsureMaps()

	result, err := h.HandleDraw(user, entry, phaseId, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != int(model.DailyGachaDrawCount) {
		t.Fatalf("daily draw count = %d, want %d", len(result.Items), model.DailyGachaDrawCount)
	}
	if _, err := h.HandleDraw(user, entry, phaseId, 1); err == nil {
		t.Fatal("daily Gacha accepted a second execution on the same business day")
	}

	state := user.Gacha.BannerStates[model.GachaIdDaily]
	state.BoxDrewCounts[model.DailyGachaDayCounterId] = gametime.BusinessDayKey(gametime.NowMillis()) - 1
	user.Gacha.BannerStates[model.GachaIdDaily] = state
	if _, err := h.HandleDraw(user, entry, phaseId, 1); err != nil {
		t.Fatalf("daily Gacha did not reset on the next business day: %v", err)
	}
	if got := user.Gacha.BannerStates[model.GachaIdDaily].DrawCount; got != model.DailyGachaDrawCount {
		t.Fatalf("daily draw count after reset = %d, want %d", got, model.DailyGachaDrawCount)
	}
}

func TestHandleDrawEnforcesStepUpOrderAndAdvancesFromFirstStep(t *testing.T) {
	h := &GachaHandler{Pool: &masterdata.GachaCatalog{}, Granter: &store.PossessionGranter{}}
	entry := eventBoxEntry()
	entry.GachaModeType = model.GachaModeStepup
	entry.MaxStepNumber = 2
	entry.PricePhases = []store.GachaPricePhaseEntry{
		{PhaseId: 11, DrawCount: 1, LimitExecCount: 1, StepNumber: 1},
		{PhaseId: 12, DrawCount: 1, LimitExecCount: 1, StepNumber: 2},
	}
	entry.BoxItems[0].MaxCount = 4
	user := &store.UserState{}
	user.EnsureMaps()

	if _, err := h.HandleDraw(user, entry, 12, 1); err == nil {
		t.Fatal("second step was accepted before first step")
	}
	if _, err := h.HandleDraw(user, entry, 11, 1); err != nil {
		t.Fatalf("first step failed: %v", err)
	}
	if got := user.Gacha.BannerStates[entry.GachaId].StepNumber; got != 2 {
		t.Fatalf("first step advanced to %d, want 2", got)
	}
	if _, err := h.HandleDraw(user, entry, 11, 1); err == nil {
		t.Fatal("first step was accepted twice")
	}
	if _, err := h.HandleDraw(user, entry, 12, 1); err != nil {
		t.Fatalf("second step failed: %v", err)
	}
	state := user.Gacha.BannerStates[entry.GachaId]
	if state.StepNumber != 1 || state.LoopCount != 1 {
		t.Fatalf("step-up did not loop correctly: %+v", state)
	}
}

func TestEventGachaDrawsOnlyItsConfiguredBoxItems(t *testing.T) {
	h := &GachaHandler{
		Pool:    &masterdata.GachaCatalog{Materials: []masterdata.GachaPoolItem{{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 10}}},
		Granter: &store.PossessionGranter{},
	}
	entry := eventBoxEntry()
	user := &store.UserState{}
	user.EnsureMaps()
	result, err := h.HandleDraw(user, entry, entry.PricePhases[0].PhaseId, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].PossessionId != 99 || user.Materials[99] != 1 {
		t.Fatalf("event box used the wrong pool: items=%v materials=%v", result.Items, user.Materials)
	}
	if user.Materials[10] != 0 {
		t.Fatal("global material pool leaked into event box")
	}
	if _, err := h.HandleDraw(user, entry, entry.PricePhases[0].PhaseId, 1); err == nil {
		t.Fatal("exhausted event box charged another draw")
	}
}

func TestHandleResetBoxInitializesPersistentBannerIdentity(t *testing.T) {
	h := &GachaHandler{}
	entry := eventBoxEntry()
	user := &store.UserState{}
	user.EnsureMaps()
	if err := h.HandleResetBox(user, entry); err != nil {
		t.Fatal(err)
	}
	state := user.Gacha.BannerStates[entry.GachaId]
	if state.GachaId != entry.GachaId || state.BoxNumber != 2 {
		t.Fatalf("unexpected reset state: %+v", state)
	}
}

func TestChapterDrawRenormalizesAfterMonthlyRewardCap(t *testing.T) {
	items := []store.GachaBoxItemEntry{
		{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 100, Count: 5, MaxCount: 1, CounterId: 1, Weight: 10},
		{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 200, Count: 2, CounterId: 2, Weight: 90},
	}
	drewCounts := make(map[int32]int32)
	var bounds []int
	result, err := drawChapterWithIntn(items, drewCounts, 2, 202608, func(bound int) int {
		bounds = append(bounds, bound)
		return 0
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 || result[0].PossessionId != 100 || result[0].Count != 5 || result[1].PossessionId != 200 || result[1].Count != 2 {
		t.Fatalf("chapter draw result = %+v", result)
	}
	if drewCounts[1] != 1 || drewCounts[model.ChapterGachaMonthCounterId] != 202608 {
		t.Fatalf("chapter counters = %v", drewCounts)
	}
	wantBounds := [...]int{100, 1, 90}
	if len(bounds) != len(wantBounds) {
		t.Fatalf("random bounds = %v, want %v", bounds, wantBounds)
	}
	for i := range wantBounds {
		if bounds[i] != wantBounds[i] {
			t.Fatalf("random bounds = %v, want %v", bounds, wantBounds)
		}
	}
}

func TestChapterDrawTreatsEachRemainingLimitedItemEqually(t *testing.T) {
	items := []store.GachaBoxItemEntry{
		{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 100, Count: 1, MaxCount: 3, CounterId: 1, Weight: 9999},
		{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 101, Count: 1, MaxCount: 1, CounterId: 2},
	}
	drewCounts := map[int32]int32{model.ChapterGachaMonthCounterId: 202608, 1: 1}
	result, err := drawChapterWithIntn(items, drewCounts, 1, 202608, func(bound int) int {
		if bound != 3 {
			t.Fatalf("limited draw bound = %d, want 3 remaining items", bound)
		}
		return 2
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].PossessionId != 101 {
		t.Fatalf("chapter draw result = %+v, want the third remaining item", result)
	}
}

func TestChapterDrawKeepsUnlimitedShareAtTwentyPercentWhileLimitedRewardsRemain(t *testing.T) {
	items := []store.GachaBoxItemEntry{
		{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 100, Count: 1, MaxCount: 1, CounterId: 1, Weight: 9999},
		{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 101, Count: 1, MaxCount: 1, CounterId: 2, Weight: 1},
		{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 200, Count: 1, CounterId: 3, Weight: 9999},
	}
	drewCounts := map[int32]int32{model.ChapterGachaMonthCounterId: 202608, 1: 1}
	wantBounds := [...]int{100, 9999, 100, 1}
	rolls := [...]int{80, 0, 79, 0}
	call := 0
	result, err := drawChapterWithIntn(items, drewCounts, 2, 202608, func(bound int) int {
		if call >= len(wantBounds) {
			t.Fatalf("unexpected random call with bound %d", bound)
		}
		if bound != wantBounds[call] {
			t.Fatalf("random call %d bound = %d, want %d", call+1, bound, wantBounds[call])
		}
		roll := rolls[call]
		call++
		return roll
	})
	if err != nil {
		t.Fatal(err)
	}
	if call != len(wantBounds) {
		t.Fatalf("random call count = %d, want %d", call, len(wantBounds))
	}
	if len(result) != 2 || result[0].PossessionId != 200 || result[1].PossessionId != 101 {
		t.Fatalf("chapter draw result = %+v, want unlimited then limited", result)
	}
}

func TestChapterDrawResetsCapsInNewBusinessMonth(t *testing.T) {
	items := []store.GachaBoxItemEntry{
		{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 100, Count: 1, MaxCount: 1, CounterId: 1, Weight: 10},
		{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 200, Count: 1, CounterId: 2, Weight: 90},
	}
	drewCounts := map[int32]int32{model.ChapterGachaMonthCounterId: 202607, 1: 1}
	result, err := drawChapterWithIntn(items, drewCounts, 1, 202608, func(int) int { return 0 })
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].PossessionId != 100 || drewCounts[1] != 1 || drewCounts[model.ChapterGachaMonthCounterId] != 202608 {
		t.Fatalf("new-month result=%+v counters=%v", result, drewCounts)
	}
}

func TestHandleChapterDrawConsumesItsTicketAndGrantsConfiguredQuantity(t *testing.T) {
	const ticketId int32 = 1008
	h := &GachaHandler{Granter: &store.PossessionGranter{}}
	entry := store.GachaCatalogEntry{
		GachaId:        200001,
		GachaLabelType: model.GachaLabelChapter,
		GachaModeType:  model.GachaModeBox,
		PricePhases: []store.GachaPricePhaseEntry{{
			PhaseId: 2000011, PriceType: model.PriceTypeConsumableItem, PriceId: ticketId, Price: 1, DrawCount: 1,
		}},
		BoxItems: []store.GachaBoxItemEntry{{
			PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 100004, Count: 4, MaxCount: 30, CounterId: 1, Weight: 10000,
		}},
	}
	user := &store.UserState{}
	user.EnsureMaps()
	user.ConsumableItems[ticketId] = 1
	result, err := h.HandleDraw(user, entry, entry.PricePhases[0].PhaseId, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].Count != 4 || user.Materials[100004] != 4 || user.ConsumableItems[ticketId] != 0 {
		t.Fatalf("result=%+v material=%d ticket=%d", result.Items, user.Materials[100004], user.ConsumableItems[ticketId])
	}
	if err := h.HandleResetBox(user, entry); err == nil {
		t.Fatal("manual reset was accepted for Chapter Gacha")
	}
}

func TestGrantItemsSendsWeaponsBeyondInventoryLimitToGiftBox(t *testing.T) {
	const nowMillis = int64(1234)
	granter := &store.PossessionGranter{}
	h := &GachaHandler{
		Config:  &masterdata.GameConfig{PossessionCountLimitWeapon: 2},
		Granter: granter,
	}
	user := &store.UserState{}
	user.EnsureMaps()
	granter.GrantWeapon(user, 100, nowMillis-1)

	h.grantItems(user, []DrawnItem{
		{PossessionType: int32(model.PossessionTypeWeapon), PossessionId: 101},
		{PossessionType: int32(model.PossessionTypeWeapon), PossessionId: 102},
	}, nowMillis)

	if len(user.Weapons) != 2 {
		t.Fatalf("weapon inventory count = %d, want 2", len(user.Weapons))
	}
	ownedWeaponIds := map[int32]bool{}
	for _, weapon := range user.Weapons {
		ownedWeaponIds[weapon.WeaponId] = true
	}
	if !ownedWeaponIds[100] || !ownedWeaponIds[101] || ownedWeaponIds[102] {
		t.Fatalf("unexpected inventory weapons: %v", ownedWeaponIds)
	}
	if len(user.Gifts.NotReceived) != 1 {
		t.Fatalf("gift count = %d, want 1", len(user.Gifts.NotReceived))
	}
	gift := user.Gifts.NotReceived[0]
	if gift.UserGiftUuid == "" ||
		gift.GiftCommon.PossessionType != int32(model.PossessionTypeWeapon) ||
		gift.GiftCommon.PossessionId != 102 ||
		gift.GiftCommon.Count != 1 ||
		gift.GiftCommon.GrantDatetime != nowMillis {
		t.Fatalf("unexpected overflow gift: %+v", gift)
	}
	if user.Notifications.GiftNotReceiveCount != 1 {
		t.Fatalf("gift notification count = %d, want 1", user.Notifications.GiftNotReceiveCount)
	}
}

func TestDupExchangesForGradeUsesTierCount(t *testing.T) {
	exchanges := []model.DupExchangeEntry{
		{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 501, Count: 10},
		{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 502, Count: 1},
	}
	tests := []struct {
		grade int32
		count int32
	}{
		{grade: 1, count: 20},
		{grade: 2, count: 16},
		{grade: 3, count: 14},
		{grade: 4, count: 12},
		{grade: 5, count: 10},
	}

	for _, tt := range tests {
		got := dupExchangesForGrade(exchanges, tt.grade)
		if got[0].Count != tt.count {
			t.Errorf("grade %d exchange count = %d, want %d", tt.grade, got[0].Count, tt.count)
		}
		if got[1].Count != 1 {
			t.Errorf("grade %d fixed bonus count = %d, want 1", tt.grade, got[1].Count)
		}
	}
	if exchanges[0].Count != 10 {
		t.Fatalf("source exchange count changed to %d", exchanges[0].Count)
	}
}

func TestDupGradeDistributionUsesConfiguredPercentages(t *testing.T) {
	counts := [5]int{}
	for roll := 0; roll < 100; roll++ {
		grade := dupGradeForRoll(roll)
		if grade < 1 || grade > 5 {
			t.Fatalf("roll %d produced invalid grade %d", roll, grade)
		}
		counts[grade-1]++
	}
	want := [5]int{3, 8, 14, 30, 45}
	if counts != want {
		t.Fatalf("grade counts = %v, want %v", counts, want)
	}
}

func TestDrawPremiumAppliesGuaranteePerExecution(t *testing.T) {
	banner := &PremiumBannerPool{
		GachaId: 1,
		Groups: []PremiumGroup{
			{Id: GroupWeaponOnly2, GrantType: GrantWeaponOnly, Star: 2, Rarity: model.RarityRare, Weight: 10000, NonPickup: []PoolItem{{WeaponId: 1, RarityType: model.RarityRare}}},
			{Id: GroupWeaponOnly4, GrantType: GrantWeaponOnly, Star: 4, Rarity: model.RaritySSRare, Weight: 1, NonPickup: []PoolItem{{WeaponId: 2, RarityType: model.RaritySSRare}}},
		},
	}
	h := &GachaHandler{Premium: &PremiumCatalog{Banners: map[int32]*PremiumBannerPool{1: banner}}}
	entry := store.GachaCatalogEntry{GachaId: 1}
	phase := store.GachaPricePhaseEntry{DrawCount: 10, FixedRarityMin: model.RaritySSRare, FixedCount: 1}

	items, err := h.drawPremium(entry, phase, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 20 {
		t.Fatalf("draw count = %d, want 20", len(items))
	}
	for _, index := range []int{9, 19} {
		if items[index].RarityType < model.RaritySSRare {
			t.Fatalf("execution ending at index %d was not guaranteed: %+v", index, items[index])
		}
	}
}

func eventBoxEntry() store.GachaCatalogEntry {
	return store.GachaCatalogEntry{
		GachaId:        1,
		GachaLabelType: model.GachaLabelEvent,
		GachaModeType:  model.GachaModeBox,
		PricePhases:    []store.GachaPricePhaseEntry{{PhaseId: 10, DrawCount: 1}},
		BoxItems:       []store.GachaBoxItemEntry{{PossessionType: int32(model.PossessionTypeMaterial), PossessionId: 99, RarityType: int32(model.RarityNormal), Count: 1, MaxCount: 1}},
	}
}
