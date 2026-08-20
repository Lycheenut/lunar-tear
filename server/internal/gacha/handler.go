package gacha

import (
	"fmt"
	"math/rand"

	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/giftbox"
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
var dupGradePercentByGrade = [...]int{3, 8, 14, 30, 45}

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
	if entry.GachaLabelType == model.GachaLabelPremium {
		if h.Premium == nil || h.Premium.Banners[entry.GachaId] == nil {
			return nil, fmt.Errorf("premium gacha %d is not configured", entry.GachaId)
		}
	}

	bs := user.Gacha.BannerStates[entry.GachaId]
	bs.GachaId = entry.GachaId
	if bs.BoxDrewCounts != nil {
		cloned := make(map[int32]int32, len(bs.BoxDrewCounts))
		for counterId, drewCount := range bs.BoxDrewCounts {
			cloned[counterId] = drewCount
		}
		bs.BoxDrewCounts = cloned
	}
	entry = h.EntryForState(entry, &bs)
	if (entry.GachaLabelType == model.GachaLabelEvent || entry.GachaLabelType == model.GachaLabelChapter) && len(entry.BoxItems) == 0 {
		return nil, fmt.Errorf("box gacha %d has no catalog", entry.GachaId)
	}
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
	if entry.GachaLabelType == model.GachaLabelEvent && drawCount64 > h.availableBoxDrawCount(entry, &bs) {
		return nil, fmt.Errorf("event gacha %d has insufficient box items", entry.GachaId)
	}
	totalCost := int32(totalCost64)

	drawCount := int(drawCount64)
	nowMillis := gametime.NowMillis()

	var items []DrawnItem

	switch entry.GachaLabelType {
	case model.GachaLabelPremium:
		items, err = h.drawPremium(entry, phase, int(execCount))
		if err != nil {
			return nil, err
		}
	case model.GachaLabelChapter:
		items, err = h.drawChapter(entry, &bs, drawCount, nowMillis)
		if err != nil {
			return nil, err
		}
	case model.GachaLabelRecycle:
		items = h.drawMaterial(drawCount)
	case model.GachaLabelEvent:
		items, err = h.drawBox(entry, &bs, drawCount)
		if err != nil {
			return nil, err
		}
	default:
		items, err = h.drawPremium(entry, phase, int(execCount))
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
	if entry.GachaLabelType == model.GachaLabelChapter {
		return fmt.Errorf("chapter Gacha resets automatically each month")
	}
	bs := user.Gacha.BannerStates[entry.GachaId]
	bs.GachaId = entry.GachaId
	entry = h.EntryForState(entry, &bs)
	if entry.GachaLabelType != model.GachaLabelEvent || entry.BoxCount <= 0 {
		return fmt.Errorf("event Gacha %d has no configured boxes", entry.GachaId)
	}
	boxNumber := currentBoxNumber(&bs, entry.BoxCount)
	if !boxResettable(entry.BoxItems, &bs, boxNumber == entry.BoxCount) {
		if boxNumber == entry.BoxCount {
			return fmt.Errorf("last Event Gacha box can only reset after all limited rewards are drawn")
		}
		return fmt.Errorf("Event Gacha box %d can only advance after all jackpot rewards are drawn", boxNumber)
	}
	bs.BoxDrewCounts = make(map[int32]int32)
	bs.BoxNumber = boxNumber
	if bs.BoxNumber < entry.BoxCount {
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
		store.GrantPossession(user, model.PossessionType(item.PossessionType), item.PossessionId, positiveCount(item.Count))
	}

	user.Gacha.TodaysCurrentDrawCount = newCount
	user.Gacha.DailyMaxCount = maxCount
	user.Gacha.LastRewardDrawDate = nowMillis
	user.Gacha.RewardAvailable = newCount < maxCount

	return items, nil
}

func (h *GachaHandler) drawPremium(entry store.GachaCatalogEntry, phase store.GachaPricePhaseEntry, execCount int) ([]DrawnItem, error) {
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

	drawCountPerExecution := int(phase.DrawCount)
	result := make([]DrawnItem, 0, drawCountPerExecution*execCount)
	for range execCount {
		var execResult []DrawnItem
		var err error
		if entry.GachaId == model.GachaIdGuaranteedThreeStarOrHigher {
			execResult, err = DrawPremiumTenthSlot(bp, rateMultiplier)
		} else {
			execResult, err = DrawPremium(bp, drawCountPerExecution, fixedMin, fixedCount, rateMultiplier)
		}
		if err != nil {
			return nil, err
		}
		result = append(result, execResult...)
	}
	return result, nil
}

func (h *GachaHandler) drawMaterial(count int) []DrawnItem {
	return DrawReward(h.Pool.Materials, count)
}

func (h *GachaHandler) drawChapter(entry store.GachaCatalogEntry, bs *store.GachaBannerState, count int, nowMillis int64) ([]DrawnItem, error) {
	if bs.BoxDrewCounts == nil {
		bs.BoxDrewCounts = make(map[int32]int32)
	}
	box, _, configured := h.configuredBox(entry, bs)
	if !configured {
		return nil, fmt.Errorf("chapter Gacha %d has no configured box", entry.GachaId)
	}
	return drawChapterWithIntn(entry.BoxItems, box.GroupWeights, bs.BoxDrewCounts, count, gametime.BusinessMonthKey(nowMillis), rand.Intn)
}

func (h *GachaHandler) drawBox(entry store.GachaCatalogEntry, bs *store.GachaBannerState, count int) ([]DrawnItem, error) {
	if bs.BoxDrewCounts == nil {
		bs.BoxDrewCounts = make(map[int32]int32)
	}
	box, _, configured := h.configuredBox(entry, bs)
	if !configured {
		return nil, fmt.Errorf("event Gacha %d has no configured box", entry.GachaId)
	}
	return drawWeightedBoxWithIntn(entry.BoxItems, box.GroupWeights, bs.BoxDrewCounts, count, rand.Intn)
}

func (h *GachaHandler) availableBoxDrawCount(entry store.GachaCatalogEntry, bs *store.GachaBannerState) int64 {
	box, _, configured := h.configuredBox(entry, bs)
	if !configured {
		return 0
	}
	return availableConfiguredBoxDrawCount(entry.BoxItems, box.GroupWeights, bs)
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
		case model.PossessionTypeWeapon, model.PossessionTypeWeaponEnhanced:
			h.grantWeaponOrGift(user, item, nowMillis)
		default:
			if item.PossessionType != 0 {
				h.Granter.GrantFull(user, model.PossessionType(item.PossessionType), item.PossessionId, positiveCount(item.Count), nowMillis)
			}
		}
	}
	return dupInfos
}

func (h *GachaHandler) grantWeaponOrGift(user *store.UserState, item DrawnItem, nowMillis int64) {
	limit := int32(0)
	if h.Config != nil {
		limit = h.Config.PossessionCountLimitWeapon
	}
	if limit <= 0 || int64(len(user.Weapons)) < int64(limit) {
		h.Granter.GrantWeapon(user, item.PossessionId, nowMillis)
		return
	}

	giftbox.AddNotReceived(user, store.NotReceivedGiftState{
		GiftCommon: store.GiftCommonState{
			PossessionType: item.PossessionType,
			PossessionId:   item.PossessionId,
			Count:          1,
			GrantDatetime:  nowMillis,
		},
	}, h.Config)
}

func (h *GachaHandler) tryCostumeDupExchange(user *store.UserState, item DrawnItem, index int) (DuplicateInfo, bool) {
	for _, c := range user.Costumes {
		if c.CostumeId == item.PossessionId {
			grade := dupGradeForRoll(rand.Intn(100))
			exchanges := dupExchangesForGrade(h.DupExchange[item.PossessionId], grade)
			for _, ex := range exchanges {
				store.GrantPossession(user, model.PossessionType(ex.PossessionType), ex.PossessionId, ex.Count)
			}
			return DuplicateInfo{Index: index, Grade: grade, Bonuses: exchanges}, true
		}
	}
	return DuplicateInfo{}, false
}

func dupGradeForRoll(roll int) int32 {
	cumulative := 0
	for i, percent := range dupGradePercentByGrade {
		cumulative += percent
		if roll < cumulative {
			return model.DupGradeMin + int32(i)
		}
	}
	return model.DupGradeMin + int32(len(dupGradePercentByGrade)-1)
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
			Count:          1,
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
