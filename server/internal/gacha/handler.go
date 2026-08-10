package gacha

import (
	"fmt"
	"math/rand"

	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

type DrawResult struct {
	Items               []DrawnItem
	BonusItems          map[int]DrawnItem
	Bonuses             []store.GachaBonusEntry
	DuplicateInfos      []DuplicateInfo
	BonusDuplicateInfos []DuplicateInfo
	MedalBonus          int32
}

type DuplicateInfo struct {
	Index   int
	Grade   int32
	Bonuses []model.DupExchangeEntry
}

type GachaHandler struct {
	Pool        *masterdata.GachaCatalog
	Premium     *PremiumCatalog
	Config      *masterdata.GameConfig
	Granter     *store.PossessionGranter
	MedalInfo   map[int32]masterdata.GachaMedalInfo
	DupExchange map[int32][]model.DupExchangeEntry
}

const (
	maxDrawCountPerRequest int64 = 1000
	maxInt32Value          int64 = 1<<31 - 1
	baseDupExchangeCount   int32 = 10
)

var dupExchangeCountByGrade = [...]int32{20, 16, 14, 12, 10}

func NewGachaHandler(
	pool *masterdata.GachaCatalog,
	premium *PremiumCatalog,
	config *masterdata.GameConfig,
	granter *store.PossessionGranter,
	medalInfo map[int32]masterdata.GachaMedalInfo,
	dupExchange map[int32][]model.DupExchangeEntry,
) *GachaHandler {
	return &GachaHandler{
		Pool:        pool,
		Premium:     premium,
		Config:      config,
		Granter:     granter,
		MedalInfo:   medalInfo,
		DupExchange: dupExchange,
	}
}

func (h *GachaHandler) HandleDraw(
	user *store.UserState,
	entry store.GachaCatalogEntry,
	phaseId int32,
	execCount int32,
) (*DrawResult, error) {
	phase, err := findPhase(entry, phaseId)
	if err != nil {
		return nil, err
	}
	if execCount <= 0 {
		return nil, fmt.Errorf("exec count must be positive")
	}
	if phase.LimitExecCount > 0 && execCount > phase.LimitExecCount {
		return nil, fmt.Errorf("exec count %d exceeds phase limit %d", execCount, phase.LimitExecCount)
	}
	if entry.GachaLabelType == model.GachaLabelEvent && len(entry.BoxItems) == 0 {
		return nil, fmt.Errorf("event gacha %d has no box catalog", entry.GachaId)
	}
	if entry.GachaLabelType == model.GachaLabelPremium {
		if h.Premium == nil || h.Premium.Banners[entry.GachaId] == nil {
			return nil, fmt.Errorf("premium gacha %d is not configured", entry.GachaId)
		}
	}

	bs := user.Gacha.BannerStates[entry.GachaId]
	bs.GachaId = entry.GachaId
	if entry.GachaModeType == model.GachaModeStepup {
		currentStep := bs.StepNumber
		if currentStep <= 0 {
			currentStep = 1
			bs.StepNumber = currentStep
		}
		if entry.MaxStepNumber <= 0 || execCount != 1 || phase.StepNumber != currentStep {
			return nil, fmt.Errorf("step-up gacha %d requires step %d", entry.GachaId, currentStep)
		}
	}

	totalCost64 := int64(phase.Price) * int64(execCount)
	drawCount64 := int64(phase.DrawCount) * int64(execCount)
	if totalCost64 < 0 || totalCost64 > maxInt32Value {
		return nil, fmt.Errorf("gacha cost is out of range")
	}
	if drawCount64 <= 0 || drawCount64 > maxDrawCountPerRequest {
		return nil, fmt.Errorf("gacha draw count is out of range")
	}
	if entry.GachaLabelType == model.GachaLabelEvent && drawCount64 > availableBoxDrawCount(entry, bs) {
		return nil, fmt.Errorf("event gacha %d has insufficient box items", entry.GachaId)
	}
	totalCost := int32(totalCost64)

	drawCount := int(drawCount64)
	nowMillis := gametime.NowMillis()

	var items []DrawnItem

	switch entry.GachaLabelType {
	case model.GachaLabelPremium:
		items, err = h.drawPremium(entry, phase, drawCount)
		if err != nil {
			return nil, err
		}
	case model.GachaLabelChapter, model.GachaLabelRecycle:
		items = h.drawMaterial(drawCount)
	case model.GachaLabelEvent:
		items = h.drawBox(entry, &bs, drawCount)
	default:
		items, err = h.drawPremium(entry, phase, drawCount)
		if err != nil {
			return nil, err
		}
	}
	if totalCost > 0 {
		if err := store.DeductPrice(user, phase.PriceType, phase.PriceId, totalCost); err != nil {
			return nil, err
		}
	}

	if entry.GachaModeType == model.GachaModeStepup {
		bs.StepNumber++
		if bs.StepNumber > entry.MaxStepNumber {
			bs.StepNumber = 1
			if bs.LoopCount < int32(maxInt32Value) {
				bs.LoopCount++
			}
		}
	}

	var medalBonus int32
	if entry.GachaMedalId != 0 {
		medalBonus = int32(drawCount)
		bs.MedalCount = int32(min(int64(bs.MedalCount)+int64(medalBonus), int64(model.MedalCountCap)))
	}

	bs.DrawCount = int32(min(int64(bs.DrawCount)+int64(drawCount), maxInt32Value))
	user.Gacha.BannerStates[entry.GachaId] = bs

	dupInfos := h.grantItems(user, items, nowMillis)

	bonusMap := h.generateBonusItems(entry, items)
	bonusSlice := make([]DrawnItem, 0, len(bonusMap))
	for _, b := range bonusMap {
		bonusSlice = append(bonusSlice, b)
	}
	bonusDupInfos := h.grantItems(user, bonusSlice, nowMillis)

	result := &DrawResult{
		Items:               items,
		BonusItems:          bonusMap,
		DuplicateInfos:      dupInfos,
		BonusDuplicateInfos: bonusDupInfos,
		MedalBonus:          medalBonus,
	}

	for _, p := range phase.Bonuses {
		store.GrantPossession(user, model.PossessionType(p.PossessionType), p.PossessionId, p.Count)
		result.Bonuses = append(result.Bonuses, p)
	}

	if medalBonus > 0 && entry.MedalConsumableItemId != 0 {
		store.GrantPossession(user, model.PossessionTypeConsumableItem, entry.MedalConsumableItemId, medalBonus)
	}

	return result, nil
}

func (h *GachaHandler) HandleResetBox(
	user *store.UserState,
	entry store.GachaCatalogEntry,
) error {
	bs := user.Gacha.BannerStates[entry.GachaId]
	bs.GachaId = entry.GachaId
	bs.BoxDrewCounts = make(map[int32]int32)
	if bs.BoxNumber <= 0 {
		bs.BoxNumber = 1
	}
	if bs.BoxNumber < int32(maxInt32Value) {
		bs.BoxNumber++
	}
	user.Gacha.BannerStates[entry.GachaId] = bs
	return nil
}

func clampDailyDraw(lastDate, todayStart int64, currentCount, maxCount, requested int32) (clamped, newCount int32, reset bool) {
	if lastDate < todayStart {
		currentCount = 0
		reset = true
	}
	remaining := maxCount - currentCount
	if remaining <= 0 {
		return 0, currentCount, reset
	}
	if requested > remaining {
		requested = remaining
	}
	return requested, currentCount + requested, reset
}

func (h *GachaHandler) HandleRewardDraw(
	user *store.UserState,
	count int32,
) ([]DrawnItem, error) {
	nowMillis := gametime.NowMillis()
	todayStart := gametime.StartOfBusinessDayMillis()

	maxCount := h.Config.RewardGachaDailyMaxCount
	if maxCount <= 0 {
		maxCount = model.DefaultDailyDrawLimit
	}

	clamped, newCount, _ := clampDailyDraw(
		user.Gacha.LastRewardDrawDate, todayStart,
		user.Gacha.TodaysCurrentDrawCount, maxCount, count,
	)
	if clamped <= 0 {
		return nil, fmt.Errorf("daily reward draw limit reached")
	}

	items := DrawReward(h.Pool.Materials, int(clamped))

	for _, item := range items {
		store.GrantPossession(user, model.PossessionType(item.PossessionType), item.PossessionId, 1)
	}

	user.Gacha.TodaysCurrentDrawCount = newCount
	user.Gacha.DailyMaxCount = maxCount
	user.Gacha.LastRewardDrawDate = nowMillis
	user.Gacha.RewardAvailable = newCount < maxCount

	return items, nil
}

func (h *GachaHandler) drawPremium(entry store.GachaCatalogEntry, phase store.GachaPricePhaseEntry, count int) ([]DrawnItem, error) {
	fixedMin := phase.FixedRarityMin
	fixedCount := int(phase.FixedCount)

	bp := h.Premium.Banners[entry.GachaId]

	rateMultiplier := 1.0
	if entry.GachaModeType == model.GachaModeStepup {
		switch phase.StepNumber {
		case 1, 3:
			rateMultiplier = model.StepUpRateBoost
		case 5:
			rateMultiplier = model.StepUpRateMaxBoost
		}
	}

	return DrawPremium(bp, count, fixedMin, fixedCount, rateMultiplier)
}

func (h *GachaHandler) drawMaterial(count int) []DrawnItem {
	return DrawReward(h.Pool.Materials, count)
}

func (h *GachaHandler) drawBox(entry store.GachaCatalogEntry, bs *store.GachaBannerState, count int) []DrawnItem {
	if bs.BoxDrewCounts == nil {
		bs.BoxDrewCounts = make(map[int32]int32)
	}

	boxItems := h.buildBoxPool(entry)
	for i := range boxItems {
		boxItems[i].DrewCount = bs.BoxDrewCounts[boxItems[i].PossessionId]
	}

	result := DrawBox(boxItems, count)

	for _, item := range result {
		bs.BoxDrewCounts[item.PossessionId]++
	}

	return result
}

func availableBoxDrawCount(entry store.GachaCatalogEntry, bs store.GachaBannerState) int64 {
	var available int64
	for _, item := range entry.BoxItems {
		remaining := int64(item.MaxCount) - int64(bs.BoxDrewCounts[item.PossessionId])
		if remaining > 0 {
			available += remaining
		}
	}
	return available
}

func (h *GachaHandler) buildBoxPool(entry store.GachaCatalogEntry) []BoxItem {
	if len(entry.BoxItems) > 0 {
		items := make([]BoxItem, 0, len(entry.BoxItems))
		for _, item := range entry.BoxItems {
			items = append(items, BoxItem{PossessionType: item.PossessionType, PossessionId: item.PossessionId, RarityType: model.RarityType(item.RarityType), Count: item.Count, MaxCount: item.MaxCount})
		}
		return items
	}
	var items []BoxItem
	for _, mat := range h.Pool.Materials {
		items = append(items, BoxItem{
			PossessionType: mat.PossessionType,
			PossessionId:   mat.PossessionId,
			RarityType:     mat.RarityType,
			Count:          1,
			MaxCount:       model.BoxItemDefaultMax,
		})
		if len(items) >= model.BoxPoolMaxItems {
			break
		}
	}
	if len(items) < model.BoxPoolMinItems {
		items = append(items, BoxItem{
			PossessionType: int32(model.PossessionTypeMaterial),
			PossessionId:   model.BoxFallbackItemId,
			RarityType:     model.RarityNormal,
			Count:          1,
			MaxCount:       model.BoxFallbackItemMax,
		})
	}
	return items
}

func (h *GachaHandler) grantItems(user *store.UserState, items []DrawnItem, nowMillis int64) []DuplicateInfo {
	var dupInfos []DuplicateInfo
	for i, item := range items {
		switch model.PossessionType(item.PossessionType) {
		case model.PossessionTypeCostume:
			if dup, ok := h.tryCostumeDupExchange(user, item, i); ok {
				dupInfos = append(dupInfos, dup)
				continue
			}
			h.Granter.GrantCostume(user, item.PossessionId, nowMillis)
		case model.PossessionTypeWeapon:
			h.Granter.GrantWeapon(user, item.PossessionId, nowMillis)
		default:
			if item.PossessionType != 0 {
				h.Granter.GrantFull(user, model.PossessionType(item.PossessionType), item.PossessionId, 1, nowMillis)
			}
		}
	}
	return dupInfos
}

func (h *GachaHandler) tryCostumeDupExchange(user *store.UserState, item DrawnItem, index int) (DuplicateInfo, bool) {
	for _, c := range user.Costumes {
		if c.CostumeId == item.PossessionId {
			grade := int32(rand.Intn(model.DupGradeRange) + int(model.DupGradeMin))
			exchanges := dupExchangesForGrade(h.DupExchange[item.PossessionId], grade)
			for _, ex := range exchanges {
				store.GrantPossession(user, model.PossessionType(ex.PossessionType), ex.PossessionId, ex.Count)
			}
			return DuplicateInfo{Index: index, Grade: grade, Bonuses: exchanges}, true
		}
	}
	return DuplicateInfo{}, false
}

func dupExchangesForGrade(exchanges []model.DupExchangeEntry, grade int32) []model.DupExchangeEntry {
	graded := append([]model.DupExchangeEntry(nil), exchanges...)
	count := dupExchangeCountByGrade[grade-model.DupGradeMin]
	for i := range graded {
		if graded[i].Count == baseDupExchangeCount {
			graded[i].Count = count
		}
	}
	return graded
}

func (h *GachaHandler) generateBonusItems(entry store.GachaCatalogEntry, mainItems []DrawnItem) map[int]DrawnItem {
	bonus := make(map[int]DrawnItem)
	for i, item := range mainItems {
		if item.PossessionType != int32(model.PossessionTypeCostume) {
			continue
		}
		wid, ok := h.Pool.CostumeWeaponMap[item.PossessionId]
		if !ok {
			continue
		}
		w, ok := h.Pool.WeaponById[wid]
		if !ok {
			continue
		}
		bonus[i] = DrawnItem{
			PossessionType: w.PossessionType,
			PossessionId:   w.PossessionId,
			RarityType:     w.RarityType,
		}
	}
	return bonus
}

func findPhase(entry store.GachaCatalogEntry, phaseId int32) (store.GachaPricePhaseEntry, error) {
	for _, p := range entry.PricePhases {
		if p.PhaseId == phaseId {
			return p, nil
		}
	}
	return store.GachaPricePhaseEntry{}, fmt.Errorf("price phase %d not found for gacha %d", phaseId, entry.GachaId)
}
