package store

import (
	"fmt"
	"log"
	"math/rand"
	"sort"

	"github.com/google/uuid"

	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
)

func DeductPrice(user *UserState, priceType, priceId, amount int32) error {
	if amount <= 0 {
		return fmt.Errorf("invalid price amount %d", amount)
	}
	switch priceType {
	case model.PriceTypeConsumableItem:
		cur := user.ConsumableItems[priceId]
		if cur < amount {
			return fmt.Errorf("insufficient consumable %d: have %d, need %d", priceId, cur, amount)
		}
		user.ConsumableItems[priceId] = cur - amount
	case model.PriceTypeGem:
		total := int64(user.Gem.FreeGem) + int64(user.Gem.PaidGem)
		if total < int64(amount) {
			return fmt.Errorf("insufficient gems: have %d, need %d", total, amount)
		}
		if user.Gem.FreeGem >= amount {
			user.Gem.FreeGem -= amount
		} else {
			amount -= user.Gem.FreeGem
			user.Gem.FreeGem = 0
			user.Gem.PaidGem -= amount
		}
	case model.PriceTypePaidGem:
		if user.Gem.PaidGem < amount {
			return fmt.Errorf("insufficient paid gems: have %d, need %d", user.Gem.PaidGem, amount)
		}
		user.Gem.PaidGem -= amount
	case model.PriceTypePlatformPayment:
		return fmt.Errorf("platform payment is disabled")
	default:
		return fmt.Errorf("unsupported price type %d", priceType)
	}
	return nil
}

type PossessionCost struct {
	PossessionType model.PossessionType
	PossessionId   int32
	Count          int32
}

// DeductPossessions validates the complete cost list before changing user
// state, so callers never pay only a subset of a multi-resource cost.
func DeductPossessions(user *UserState, costs []PossessionCost) error {
	type possessionKey struct {
		PossessionType model.PossessionType
		PossessionId   int32
	}
	totals := make(map[possessionKey]int64, len(costs))
	for _, cost := range costs {
		if cost.Count < 0 {
			return fmt.Errorf("invalid possession cost type=%d id=%d count=%d", cost.PossessionType, cost.PossessionId, cost.Count)
		}
		if cost.Count == 0 {
			continue
		}
		switch cost.PossessionType {
		case model.PossessionTypeMaterial,
			model.PossessionTypeConsumableItem,
			model.PossessionTypePaidGem,
			model.PossessionTypeFreeGem,
			model.PossessionTypeImportantItem:
		default:
			return fmt.Errorf("unsupported possession cost type=%d", cost.PossessionType)
		}
		key := possessionKey{PossessionType: cost.PossessionType, PossessionId: cost.PossessionId}
		totals[key] += int64(cost.Count)
	}

	for key, count := range totals {
		var available int32
		switch key.PossessionType {
		case model.PossessionTypeMaterial:
			available = user.Materials[key.PossessionId]
		case model.PossessionTypeConsumableItem:
			available = user.ConsumableItems[key.PossessionId]
		case model.PossessionTypePaidGem:
			available = user.Gem.PaidGem
		case model.PossessionTypeFreeGem:
			available = user.Gem.FreeGem
		case model.PossessionTypeImportantItem:
			available = user.ImportantItems[key.PossessionId]
		}
		if int64(available) < count {
			return fmt.Errorf("insufficient possession type=%d id=%d: have %d, need %d", key.PossessionType, key.PossessionId, available, count)
		}
	}

	for key, count64 := range totals {
		count := int32(count64)
		switch key.PossessionType {
		case model.PossessionTypeMaterial:
			user.Materials[key.PossessionId] -= count
			if user.Materials[key.PossessionId] == 0 {
				delete(user.Materials, key.PossessionId)
			}
		case model.PossessionTypeConsumableItem:
			user.ConsumableItems[key.PossessionId] -= count
			if user.ConsumableItems[key.PossessionId] == 0 {
				delete(user.ConsumableItems, key.PossessionId)
			}
		case model.PossessionTypePaidGem:
			user.Gem.PaidGem -= count
		case model.PossessionTypeFreeGem:
			user.Gem.FreeGem -= count
		case model.PossessionTypeImportantItem:
			user.ImportantItems[key.PossessionId] -= count
			if user.ImportantItems[key.PossessionId] == 0 {
				delete(user.ImportantItems, key.PossessionId)
			}
		}
	}
	return nil
}

func GrantPossession(user *UserState, possessionType model.PossessionType, possessionId, count int32) {
	switch possessionType {
	case model.PossessionTypeMaterial:
		user.Materials[possessionId] += count
	case model.PossessionTypeConsumableItem:
		user.ConsumableItems[possessionId] += count
	case model.PossessionTypePaidGem:
		user.Gem.PaidGem += count
	case model.PossessionTypeFreeGem:
		user.Gem.FreeGem += count
	case model.PossessionTypeImportantItem:
		user.ImportantItems[possessionId] += count
	case model.PossessionTypePremiumItem:
		user.PremiumItems[possessionId] = gametime.NowMillis()
	default:
		log.Printf("[GrantPossession] unhandled type=%d id=%d count=%d", possessionType, possessionId, count)
	}
}

type CostumeRef struct {
	CharacterId int32
}

type CostumeEnhancedRef struct {
	CostumeId int32
	Level     int32
	Exp       int32
}

type WeaponRef struct {
	WeaponSkillGroupId                 int32
	WeaponAbilityGroupId               int32
	WeaponStoryReleaseConditionGroupId int32
}

type WeaponStoryReleaseCond struct {
	StoryIndex                      int32
	WeaponStoryReleaseConditionType model.WeaponStoryReleaseConditionType
	ConditionValue                  int32
}

type PartsRef struct {
	PartsGroupId                  int32
	RarityType                    int32
	PartsInitialLotteryId         int32
	PartsStatusMainLotteryGroupId int32
	PartsStatusSubLotteryGroupId  int32
}

// PartsStatusSubDef carries the per-lottery-id sub-status shape needed at
// grant time. Held here so the store package does not import masterdata.
type PartsStatusSubDef struct {
	StatusKindType           int32
	StatusCalculationType    int32
	StatusChangeInitialValue int32
	StatusFunc               func(level int32) int32
}

type PossessionGranter struct {
	CostumeById         map[int32]CostumeRef
	CostumeEnhancedById map[int32]CostumeEnhancedRef
	WeaponById          map[int32]WeaponRef
	WeaponSkillSlots    map[int32][]int32
	WeaponAbilitySlots  map[int32][]int32
	ReleaseConditions   map[int32][]WeaponStoryReleaseCond

	PartsById                            map[int32]PartsRef
	DefaultPartsStatusMainByLotteryGroup map[int32]int32
	PartsVariantsByGroupRarity           map[int32]map[int32][]int32
	PartsSubStatusPool                   map[int32][]int32
	PartsSubStatusDefs                   map[int32]PartsStatusSubDef

	PartsSellPriceL1ByRarity map[int32]int32
	GoldConsumableItemId     int32
	CostumeDupExchange       map[int32][]model.DupExchangeEntry
	CompanionDupExchange     map[int32][]model.DupExchangeEntry
}

type GrantStatus int

const (
	GrantStatusGranted GrantStatus = iota
	GrantStatusOverflow
	GrantStatusUnsupported
	GrantStatusInvalid
)

type GrantResult struct {
	Status                GrantStatus
	ChangedStoryWeaponIds []int32
}

func (g *PossessionGranter) GrantFull(user *UserState, possessionType model.PossessionType, possessionId, count int32, nowMillis int64) GrantResult {
	if count <= 0 {
		return GrantResult{Status: GrantStatusInvalid}
	}
	result := GrantResult{Status: GrantStatusGranted}
	switch possessionType {
	case model.PossessionTypeCostume:
		for range count {
			g.GrantCostume(user, possessionId, nowMillis)
		}
	case model.PossessionTypeCostumeEnhanced:
		for range count {
			g.GrantEnhancedCostume(user, possessionId, nowMillis)
		}
	case model.PossessionTypeWeapon, model.PossessionTypeWeaponEnhanced:
		for range count {
			result.ChangedStoryWeaponIds = append(result.ChangedStoryWeaponIds, g.GrantWeapon(user, possessionId, nowMillis)...)
		}
	case model.PossessionTypeCompanion, model.PossessionTypeCompanionEnhanced:
		for range count {
			g.GrantCompanion(user, possessionId, nowMillis)
		}
	case model.PossessionTypeParts, model.PossessionTypePartsEnhanced:
		for range count {
			g.GrantParts(user, possessionId, nowMillis)
		}
	case model.PossessionTypeThought:
		for range count {
			g.GrantThought(user, possessionId, nowMillis)
		}
	case model.PossessionTypeMaterial,
		model.PossessionTypeConsumableItem,
		model.PossessionTypePaidGem,
		model.PossessionTypeFreeGem,
		model.PossessionTypeImportantItem,
		model.PossessionTypePremiumItem:
		GrantPossession(user, possessionType, possessionId, count)
	default:
		result.Status = GrantStatusUnsupported
	}
	if result.Status == GrantStatusGranted {
		AddMissionCount(user, int32(model.MissionClearConditionTypePossessionAddByCount), count, possessionId, int32(possessionType))
	}
	return result
}

func (g *PossessionGranter) GrantCostume(user *UserState, costumeId int32, nowMillis int64) {
	g.grantCostume(user, costumeId, 1, 0, nowMillis)
}

func (g *PossessionGranter) GrantEnhancedCostume(user *UserState, costumeEnhancedId int32, nowMillis int64) {
	enhanced, ok := g.CostumeEnhancedById[costumeEnhancedId]
	if !ok {
		g.GrantCostume(user, costumeEnhancedId, nowMillis)
		return
	}
	g.grantCostume(user, enhanced.CostumeId, enhanced.Level, enhanced.Exp, nowMillis)
}

func (g *PossessionGranter) grantCostume(user *UserState, costumeId, level, exp int32, nowMillis int64) {
	for _, row := range user.Costumes {
		if row.CostumeId == costumeId {
			grantDuplicateExchange(user, g.CostumeDupExchange[costumeId])
			return
		}
	}
	if cm, ok := g.CostumeById[costumeId]; ok {
		if _, exists := user.Characters[cm.CharacterId]; !exists {
			user.Characters[cm.CharacterId] = CharacterState{
				CharacterId: cm.CharacterId,
				Level:       1,
			}
		}
	}
	key := uuid.New().String()
	user.Costumes[key] = CostumeState{
		UserCostumeUuid:     key,
		CostumeId:           costumeId,
		Level:               level,
		Exp:                 exp,
		HeadupDisplayViewId: 1,
		AcquisitionDatetime: nowMillis,
	}
	user.CostumeActiveSkills[key] = CostumeActiveSkillState{
		UserCostumeUuid:     key,
		Level:               1,
		AcquisitionDatetime: nowMillis,
	}
}

func (g *PossessionGranter) GrantCompanion(user *UserState, companionId int32, nowMillis int64) {
	for _, row := range user.Companions {
		if row.CompanionId == companionId {
			grantDuplicateExchange(user, g.CompanionDupExchange[companionId])
			return
		}
	}
	key := uuid.New().String()
	user.Companions[key] = CompanionState{
		UserCompanionUuid:   key,
		CompanionId:         companionId,
		Level:               1,
		HeadupDisplayViewId: 1,
		AcquisitionDatetime: nowMillis,
	}
}

func grantDuplicateExchange(user *UserState, exchange []model.DupExchangeEntry) {
	for _, item := range exchange {
		GrantPossession(user, model.PossessionType(item.PossessionType), item.PossessionId, item.Count)
	}
}

func (g *PossessionGranter) GrantThought(user *UserState, thoughtId int32, nowMillis int64) {
	key := uuid.New().String()
	user.Thoughts[key] = ThoughtState{
		UserThoughtUuid:     key,
		ThoughtId:           thoughtId,
		AcquisitionDatetime: nowMillis,
		LatestVersion:       nowMillis,
	}
}

func (g *PossessionGranter) GrantParts(user *UserState, requestedPartsId int32, nowMillis int64) {
	chosenPartsId, chosenRef, ok := g.rollPartsVariant(requestedPartsId)
	if !ok {
		g.grantBareParts(user, requestedPartsId, nowMillis)
		return
	}
	g.createParts(user, chosenPartsId, chosenRef, nowMillis)
}

// The rolled variant sets both rarity and rank, so the auto-sale decision can
// only happen after the roll. Returns the rolled variant id and whether it sold.
func (g *PossessionGranter) GrantOrSellPartsDrop(user *UserState, requestedPartsId int32, raritySet, rankSet map[int32]bool, nowMillis int64) (int32, bool) {
	chosenPartsId, chosenRef, ok := g.rollPartsVariant(requestedPartsId)
	if !ok {
		g.grantBareParts(user, requestedPartsId, nowMillis)
		return requestedPartsId, false
	}
	rarity := chosenRef.RarityType
	rank := chosenRef.PartsInitialLotteryId
	if price, ok := g.PartsSellPriceL1ByRarity[rarity]; ok && raritySet[rarity] && rankSet[rank] {
		user.ConsumableItems[g.GoldConsumableItemId] += price
		log.Printf("[GrantParts] auto-sold chosen=%d rarity=%d rank=%d -> %d gold", chosenPartsId, rarity, rank, price)
		return chosenPartsId, true
	}
	g.createParts(user, chosenPartsId, chosenRef, nowMillis)
	return chosenPartsId, false
}

func (g *PossessionGranter) grantBareParts(user *UserState, partsId int32, nowMillis int64) {
	key := uuid.New().String()
	user.Parts[key] = PartsState{
		UserPartsUuid:       key,
		PartsId:             partsId,
		Level:               1,
		AcquisitionDatetime: nowMillis,
	}
	log.Printf("[GrantParts] unknown partsId=%d, granted as-is with no variant roll", partsId)
}

// rollPartsVariant picks one of a parts group's 5 variants at random; the five
// carry distinct PartsInitialLotteryId 1..5, which is the part's rank.
func (g *PossessionGranter) rollPartsVariant(requestedPartsId int32) (int32, PartsRef, bool) {
	ref, refOk := g.PartsById[requestedPartsId]
	if !refOk {
		return requestedPartsId, PartsRef{}, false
	}
	chosenPartsId := requestedPartsId
	chosenRef := ref
	if variants := g.PartsVariantsByGroupRarity[ref.PartsGroupId][ref.RarityType]; len(variants) == 5 {
		chosenPartsId = variants[rand.Intn(len(variants))]
		chosenRef = g.PartsById[chosenPartsId]
	} else {
		log.Printf("[GrantParts] no 5-variant set for group=%d rarity=%d (have %d), granting requested=%d", ref.PartsGroupId, ref.RarityType, len(variants), requestedPartsId)
	}
	return chosenPartsId, chosenRef, true
}

// PickUniquePartsSubStatus selects an unused lottery entry for a part. Exact
// lottery ids are unique; flat and percentage variants have different ids and
// therefore remain independently eligible.
func PickUniquePartsSubStatus(pool []int32, user *UserState, partsUuid string) (int32, bool) {
	used := make(map[int32]bool)
	for key, status := range user.PartsStatusSubs {
		if key.UserPartsUuid == partsUuid {
			used[status.PartsStatusSubLotteryId] = true
		}
	}
	available := make([]int32, 0, len(pool))
	for _, id := range pool {
		if !used[id] {
			available = append(available, id)
		}
	}
	if len(available) == 0 {
		return 0, false
	}
	return available[rand.Intn(len(available))], true
}

func (g *PossessionGranter) createParts(user *UserState, chosenPartsId int32, chosenRef PartsRef, nowMillis int64) {
	mainStatId := g.DefaultPartsStatusMainByLotteryGroup[chosenRef.PartsStatusMainLotteryGroupId]
	if _, exists := user.PartsGroupNotes[chosenRef.PartsGroupId]; !exists {
		user.PartsGroupNotes[chosenRef.PartsGroupId] = PartsGroupNoteState{
			PartsGroupId:             chosenRef.PartsGroupId,
			FirstAcquisitionDatetime: nowMillis,
			LatestVersion:            nowMillis,
		}
	}
	key := uuid.New().String()
	user.Parts[key] = PartsState{
		UserPartsUuid:       key,
		PartsId:             chosenPartsId,
		Level:               1,
		PartsStatusMainId:   mainStatId,
		AcquisitionDatetime: nowMillis,
	}

	initialCount := chosenRef.PartsInitialLotteryId
	pool := g.PartsSubStatusPool[chosenRef.PartsStatusSubLotteryGroupId]
	if initialCount > 1 && len(pool) > 0 {
		for i := int32(0); i < initialCount-1; i++ {
			pickId, picked := PickUniquePartsSubStatus(pool, user, key)
			if !picked {
				break
			}
			def, ok := g.PartsSubStatusDefs[pickId]
			if !ok {
				continue
			}
			val := def.StatusChangeInitialValue
			if def.StatusFunc != nil {
				val = def.StatusFunc(1)
			}
			user.PartsStatusSubs[PartsStatusSubKey{UserPartsUuid: key, StatusIndex: i + 1}] = PartsStatusSubState{
				UserPartsUuid:           key,
				StatusIndex:             i + 1,
				PartsStatusSubLotteryId: pickId,
				Level:                   1,
				StatusKindType:          def.StatusKindType,
				StatusCalculationType:   def.StatusCalculationType,
				StatusChangeValue:       val,
				LatestVersion:           nowMillis,
			}
		}
	}

	log.Printf("[GrantParts] chosen=%d group=%d rarity=%d preUnlockedSubs=%d", chosenPartsId, chosenRef.PartsGroupId, chosenRef.RarityType, initialCount-1)
}

func (g *PossessionGranter) GrantWeapon(user *UserState, weaponId int32, nowMillis int64) []int32 {
	key := uuid.New().String()
	user.Weapons[key] = WeaponState{
		UserWeaponUuid:      key,
		WeaponId:            weaponId,
		Level:               1,
		AcquisitionDatetime: nowMillis,
	}
	if _, exists := user.WeaponNotes[weaponId]; !exists {
		user.WeaponNotes[weaponId] = WeaponNoteState{
			WeaponId:                 weaponId,
			MaxLevel:                 1,
			MaxLimitBreakCount:       0,
			FirstAcquisitionDatetime: nowMillis,
			LatestVersion:            nowMillis,
		}
	}
	weapon, ok := g.WeaponById[weaponId]
	if !ok {
		return nil
	}

	g.populateWeaponSkillsAbilities(user, key, weapon)
	if weapon.WeaponStoryReleaseConditionGroupId != 0 {
		changed := false
		for _, cond := range g.ReleaseConditions[weapon.WeaponStoryReleaseConditionGroupId] {
			switch cond.WeaponStoryReleaseConditionType {
			case model.WeaponStoryReleaseConditionTypeAcquisition:
				if grantWeaponStoryUnlock(user, weaponId, cond.StoryIndex, nowMillis) {
					changed = true
				}
			case model.WeaponStoryReleaseConditionTypeQuestClear:
				if qs, ok := user.Quests[cond.ConditionValue]; ok && qs.QuestStateType == model.UserQuestStateTypeCleared {
					if grantWeaponStoryUnlock(user, weaponId, cond.StoryIndex, nowMillis) {
						changed = true
					}
				}
			}
		}
		if changed {
			return []int32{weaponId}
		}
	}
	return nil
}

func (g *PossessionGranter) populateWeaponSkillsAbilities(user *UserState, weaponUuid string, weapon WeaponRef) {
	if slots, ok := g.WeaponSkillSlots[weapon.WeaponSkillGroupId]; ok {
		skills := make([]WeaponSkillState, len(slots))
		for i, slot := range slots {
			skills[i] = WeaponSkillState{
				UserWeaponUuid: weaponUuid,
				SlotNumber:     slot,
				Level:          1,
			}
		}
		user.WeaponSkills[weaponUuid] = skills
	}
	if slots, ok := g.WeaponAbilitySlots[weapon.WeaponAbilityGroupId]; ok {
		abilities := make([]WeaponAbilityState, len(slots))
		for i, slot := range slots {
			abilities[i] = WeaponAbilityState{
				UserWeaponUuid: weaponUuid,
				SlotNumber:     slot,
				Level:          1,
			}
		}
		user.WeaponAbilities[weaponUuid] = abilities
	}
}

func GrantWeaponStoryUnlock(user *UserState, weaponId, storyIndex int32, nowMillis int64) bool {
	return grantWeaponStoryUnlock(user, weaponId, storyIndex, nowMillis)
}

func grantWeaponStoryUnlock(user *UserState, weaponId, storyIndex int32, nowMillis int64) bool {
	hasWeapon := false
	for _, row := range user.Weapons {
		if row.WeaponId == weaponId {
			hasWeapon = true
			break
		}
	}
	if !hasWeapon {
		log.Printf("[grantWeaponStoryUnlock] skipping weaponId=%d (weapon not in user.Weapons)", weaponId)
		return false
	}
	if user.WeaponStories == nil {
		user.WeaponStories = make(map[int32]WeaponStoryState)
	}
	cur := user.WeaponStories[weaponId]
	if storyIndex <= cur.ReleasedMaxStoryIndex {
		return false
	}
	user.WeaponStories[weaponId] = WeaponStoryState{
		WeaponId:              weaponId,
		ReleasedMaxStoryIndex: storyIndex,
		LatestVersion:         nowMillis,
	}
	return true
}

func EnsureDefaultDeck(user *UserState, nowMillis int64) {
	if len(user.Costumes) == 0 || len(user.Decks) > 0 {
		return
	}

	const rionCostumeId = int32(10100)
	const rionWeaponId = int32(101001)

	var costumeUuid, weaponUuid string
	for k, v := range user.Costumes {
		if v.CostumeId == rionCostumeId {
			costumeUuid = k
			break
		}
	}
	for k, v := range user.Weapons {
		if v.WeaponId == rionWeaponId {
			weaponUuid = k
			break
		}
	}

	dcUuid := uuid.New().String()
	user.DeckCharacters[dcUuid] = DeckCharacterState{
		UserDeckCharacterUuid: dcUuid,
		UserCompanionUuid:     "",
		UserCostumeUuid:       costumeUuid,
		MainUserWeaponUuid:    weaponUuid,
		Power:                 100,
		LatestVersion:         nowMillis,
	}
	user.Decks[DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: 1}] = DeckState{
		DeckType:                model.DeckTypeQuest,
		UserDeckNumber:          1,
		UserDeckCharacterUuid01: dcUuid,
		Name:                    "Deck 1",
		Power:                   100,
		LatestVersion:           nowMillis,
	}

	if _, exists := user.DeckTypeNotes[model.DeckTypeQuest]; !exists {
		user.DeckTypeNotes[model.DeckTypeQuest] = DeckTypeNoteState{
			DeckType:      model.DeckTypeQuest,
			MaxDeckPower:  100,
			LatestVersion: nowMillis,
		}
	}
}

func FirstSortedKey[V any](m map[string]V) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys[0]
}

func ApplyDeckReplacement(user *UserState, deckType model.DeckType, userDeckNumber int32, slots []DeckCharacterInput, nowMillis int64) {
	deckKey := DeckKey{DeckType: deckType, UserDeckNumber: userDeckNumber}
	deck := user.Decks[deckKey]
	deck.DeckType = deckType
	deck.UserDeckNumber = userDeckNumber
	if deck.Name == "" {
		deck.Name = fmt.Sprintf("Deck %d", userDeckNumber)
	}
	if deck.Power == 0 {
		deck.Power = 100
	}

	for _, oldUuid := range []string{deck.UserDeckCharacterUuid01, deck.UserDeckCharacterUuid02, deck.UserDeckCharacterUuid03} {
		if oldUuid == "" {
			continue
		}
		delete(user.DeckCharacters, oldUuid)
		delete(user.DeckSubWeapons, oldUuid)
		delete(user.DeckParts, oldUuid)
	}

	var newUuids [3]string
	for i := range 3 {
		if i >= len(slots) || slots[i].UserCostumeUuid == "" {
			continue
		}
		slot := slots[i]
		dcUuid := uuid.New().String()
		user.DeckCharacters[dcUuid] = DeckCharacterState{
			UserDeckCharacterUuid: dcUuid,
			UserCostumeUuid:       slot.UserCostumeUuid,
			MainUserWeaponUuid:    slot.MainUserWeaponUuid,
			UserCompanionUuid:     slot.UserCompanionUuid,
			UserThoughtUuid:       slot.UserThoughtUuid,
			DressupCostumeId:      slot.DressupCostumeId,
			LatestVersion:         nowMillis,
		}
		user.DeckSubWeapons[dcUuid] = slot.SubWeaponUuids
		user.DeckParts[dcUuid] = slot.PartsUuids
		newUuids[i] = dcUuid
	}

	deck.UserDeckCharacterUuid01 = newUuids[0]
	deck.UserDeckCharacterUuid02 = newUuids[1]
	deck.UserDeckCharacterUuid03 = newUuids[2]
	deck.LatestVersion = nowMillis
	user.Decks[deckKey] = deck
}

func RemoveDeckData(user *UserState, deckType model.DeckType, userDeckNumber int32) {
	deckKey := DeckKey{DeckType: deckType, UserDeckNumber: userDeckNumber}
	deck, ok := user.Decks[deckKey]
	if !ok {
		return
	}
	for _, dcUuid := range []string{deck.UserDeckCharacterUuid01, deck.UserDeckCharacterUuid02, deck.UserDeckCharacterUuid03} {
		if dcUuid == "" {
			continue
		}
		delete(user.DeckCharacters, dcUuid)
		delete(user.DeckSubWeapons, dcUuid)
		delete(user.DeckParts, dcUuid)
	}
	delete(user.Decks, deckKey)
}

func ReadDeckSlots(user *UserState, deckType model.DeckType, userDeckNumber int32) []DeckCharacterInput {
	deckKey := DeckKey{DeckType: deckType, UserDeckNumber: userDeckNumber}
	deck, ok := user.Decks[deckKey]
	if !ok {
		return nil
	}
	slots := make([]DeckCharacterInput, 3)
	for i, dcUuid := range []string{deck.UserDeckCharacterUuid01, deck.UserDeckCharacterUuid02, deck.UserDeckCharacterUuid03} {
		if dcUuid == "" {
			continue
		}
		dc, ok := user.DeckCharacters[dcUuid]
		if !ok {
			continue
		}
		slots[i] = DeckCharacterInput{
			UserCostumeUuid:    dc.UserCostumeUuid,
			MainUserWeaponUuid: dc.MainUserWeaponUuid,
			SubWeaponUuids:     user.DeckSubWeapons[dcUuid],
			PartsUuids:         user.DeckParts[dcUuid],
			UserCompanionUuid:  dc.UserCompanionUuid,
			UserThoughtUuid:    dc.UserThoughtUuid,
			DressupCostumeId:   dc.DressupCostumeId,
		}
	}
	return slots
}
