package gacha

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

const (
	ConfigVersion              = 1
	GroupWeightTotal           = 10000
	DefaultBannerStartDatetime = int64(4070937600000) // 2099-01-01 16:00:00 +08:00
	DefaultBannerEndDatetime   = int64(4086576000000) // 2099-07-01 16:00:00 +08:00
)

type Availability string

const (
	AvailabilityStandard Availability = "standard"
	AvailabilityEvent    Availability = "event"
	AvailabilityLimited  Availability = "limited"
)

type GrantType string

const (
	GrantCharacterWeapon GrantType = "character_weapon"
	GrantWeaponOnly      GrantType = "weapon_only"
)

type RarityWeights struct {
	TwoStar   int `json:"2,omitempty"`
	ThreeStar int `json:"3"`
	FourStar  int `json:"4"`
}

type GroupWeights struct {
	CharacterWeapon RarityWeights `json:"characterWeapon"`
	WeaponOnly      RarityWeights `json:"weaponOnly"`
}

type LimitedSetConfig struct {
	DisplayName string `json:"displayName"`
}

type WeaponConfig struct {
	Availability Availability `json:"availability"`
	LimitedSet   string       `json:"limitedSet,omitempty"`
}

type BannerConfig struct {
	BannerAssetName string   `json:"bannerAssetName"`
	StartDatetime   int64    `json:"startDatetime"`
	EndDatetime     int64    `json:"endDatetime"`
	LimitedSets     []string `json:"limitedSets,omitempty"`
	PickupWeaponIds []int32  `json:"pickupWeaponIds,omitempty"`
}

type Config struct {
	Version              int                         `json:"version"`
	SourceMasterDataHash string                      `json:"sourceMasterDataHash"`
	GroupWeights         GroupWeights                `json:"groupWeights"`
	LimitedSets          map[string]LimitedSetConfig `json:"limitedSets"`
	Weapons              map[int32]WeaponConfig      `json:"weapons"`
	Banners              map[int32]BannerConfig      `json:"banners"`
	ChapterBanners       map[int32]BoxConfig         `json:"chapterBanners,omitempty"`
	EventBanners         map[int32]EventBoxConfig    `json:"eventBanners,omitempty"`
}

type PoolItem struct {
	WeaponId    int32
	CostumeId   int32
	CharacterId int32
	RarityType  model.RarityType
}

func (i PoolItem) DrawnItem() DrawnItem {
	if i.CostumeId != 0 {
		return DrawnItem{
			PossessionType: int32(model.PossessionTypeCostume),
			PossessionId:   i.CostumeId,
			RarityType:     i.RarityType,
			CharacterId:    i.CharacterId,
			Count:          1,
		}
	}
	return DrawnItem{
		PossessionType: int32(model.PossessionTypeWeapon),
		PossessionId:   i.WeaponId,
		RarityType:     i.RarityType,
		Count:          1,
	}
}

type GroupId string

const (
	GroupCharacterWeapon3 GroupId = "character_weapon_3"
	GroupCharacterWeapon4 GroupId = "character_weapon_4"
	GroupWeaponOnly2      GroupId = "weapon_only_2"
	GroupWeaponOnly3      GroupId = "weapon_only_3"
	GroupWeaponOnly4      GroupId = "weapon_only_4"
)

type PremiumGroup struct {
	Id        GroupId
	GrantType GrantType
	Star      int32
	Rarity    model.RarityType
	Weight    int
	Pickup    []PoolItem
	NonPickup []PoolItem
}

func (g PremiumGroup) ItemCount() int {
	return len(g.Pickup) + len(g.NonPickup)
}

type PremiumBannerPool struct {
	GachaId         int32
	Groups          []PremiumGroup
	ItemsByWeaponId map[int32]PoolItem
}

type PremiumCatalog struct {
	Config  *Config
	Banners map[int32]*PremiumBannerPool
}

type BuildOptions struct {
	RequireComplete       bool
	CurrentMasterDataHash string
	ValidBoxRewards       map[[2]int32]bool
}

type groupDefinition struct {
	id        GroupId
	grantType GrantType
	star      int32
	rarity    model.RarityType
}

var groupDefinitions = []groupDefinition{
	{GroupCharacterWeapon4, GrantCharacterWeapon, 4, model.RaritySSRare},
	{GroupWeaponOnly4, GrantWeaponOnly, 4, model.RaritySSRare},
	{GroupCharacterWeapon3, GrantCharacterWeapon, 3, model.RaritySRare},
	{GroupWeaponOnly3, GrantWeaponOnly, 3, model.RaritySRare},
	{GroupWeaponOnly2, GrantWeaponOnly, 2, model.RarityRare},
}

func DefaultConfig() *Config {
	return &Config{
		Version: ConfigVersion,
		GroupWeights: GroupWeights{
			CharacterWeapon: RarityWeights{ThreeStar: 500, FourStar: 200},
			WeaponOnly:      RarityWeights{TwoStar: 8000, ThreeStar: 1000, FourStar: 300},
		},
		LimitedSets:    make(map[string]LimitedSetConfig),
		Weapons:        make(map[int32]WeaponConfig),
		Banners:        make(map[int32]BannerConfig),
		ChapterBanners: make(map[int32]BoxConfig),
		EventBanners:   make(map[int32]EventBoxConfig),
	}
}

func ReadConfig(path string) (*Config, string, bool, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultConfig(), ContentHash(nil), false, nil
	}
	if err != nil {
		return nil, "", false, fmt.Errorf("read Gacha config: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return nil, "", true, fmt.Errorf("decode Gacha config: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, "", true, err
	}
	normalizeConfig(&config)
	return &config, ContentHash(raw), true, nil
}

func EncodeConfig(config *Config) ([]byte, string, error) {
	if config == nil {
		return nil, "", fmt.Errorf("Gacha config is nil")
	}
	normalizeConfig(config)
	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, "", fmt.Errorf("encode Gacha config: %w", err)
	}
	raw = append(raw, '\n')
	return raw, ContentHash(raw), nil
}

// ConfigWithoutAutomaticEventWeapons returns a copy that omits root weapons
// excluded from Gacha by master data. Unknown IDs are preserved so normal
// validation can still reject stale or mistyped configuration.
func ConfigWithoutAutomaticEventWeapons(config *Config, source *masterdata.GachaCatalog) *Config {
	if config == nil || source == nil {
		return config
	}
	automaticEvent := make(map[int32]bool)
	for weaponId := range source.ConfigurableWeaponById {
		if _, eligible := source.EligibleWeaponById[weaponId]; !eligible {
			automaticEvent[weaponId] = true
		}
	}

	result := *config
	result.Weapons = make(map[int32]WeaponConfig, len(config.Weapons))
	for weaponId, weapon := range config.Weapons {
		if !automaticEvent[weaponId] {
			result.Weapons[weaponId] = weapon
		}
	}
	result.Banners = make(map[int32]BannerConfig, len(config.Banners))
	for gachaId, banner := range config.Banners {
		filtered := BannerConfig{
			BannerAssetName: banner.BannerAssetName,
			StartDatetime:   banner.StartDatetime,
			EndDatetime:     banner.EndDatetime,
			LimitedSets:     append([]string(nil), banner.LimitedSets...),
		}
		for _, weaponId := range banner.PickupWeaponIds {
			if !automaticEvent[weaponId] {
				filtered.PickupWeaponIds = append(filtered.PickupWeaponIds, weaponId)
			}
		}
		result.Banners[gachaId] = filtered
	}
	return &result
}

// ApplyConfiguredPremiumBanners replaces master-data premium banners with the
// limited_* inventory declared by gacha.json. m_mom_banner is intentionally not
// consulted: MamaBanner responses are downstream of these catalog entries.
func ApplyConfiguredPremiumBanners(config *Config, entries []store.GachaCatalogEntry, medals map[int32]masterdata.GachaMedalInfo) []store.GachaCatalogEntry {
	result := make([]store.GachaCatalogEntry, 0, len(entries)+len(config.Banners))
	for _, entry := range entries {
		if entry.GachaLabelType != model.GachaLabelPremium || model.IsGuaranteedTicketGacha(entry.GachaId) {
			result = append(result, entry)
		}
	}

	gachaIds := make([]int32, 0, len(config.Banners))
	for gachaId := range config.Banners {
		if model.IsGuaranteedTicketGacha(gachaId) {
			continue
		}
		gachaIds = append(gachaIds, gachaId)
	}
	sort.Slice(gachaIds, func(i, j int) bool { return gachaIds[i] < gachaIds[j] })
	for _, gachaId := range gachaIds {
		banner := config.Banners[gachaId]
		medal := medals[gachaId]
		result = append(result, store.GachaCatalogEntry{
			GachaId:               gachaId,
			IsMamaBanner:          true,
			GachaLabelType:        model.GachaLabelPremium,
			GachaModeType:         model.GachaModeBasic,
			GachaAutoResetType:    model.GachaAutoResetNone,
			IsUserGachaUnlock:     true,
			StartDatetime:         banner.StartDatetime,
			EndDatetime:           banner.EndDatetime,
			GachaMedalId:          medal.GachaMedalId,
			MedalConsumableItemId: medal.ConsumableItemId,
			GachaDecorationType:   model.GachaDecorationFestival,
			SortOrder:             gachaId,
			BannerAssetName:       banner.BannerAssetName,
			GroupId:               gachaId,
			CeilingCount:          model.PityCeilingCount,
			PricePhases:           premiumBasicPricePhases(gachaId),
		})
	}
	applyGuaranteedTicketAvailability(result)
	return result
}

func premiumBasicPricePhases(gachaId int32) []store.GachaPricePhaseEntry {
	return []store.GachaPricePhaseEntry{
		{
			PhaseId:   gachaId*model.PhaseIdMultiplier + 1,
			PriceType: model.PriceTypeGem,
			DrawCount: 1,
		},
		{
			PhaseId:        gachaId*model.PhaseIdMultiplier + 2,
			PriceType:      model.PriceTypeGem,
			DrawCount:      model.PremiumMultiPullCount,
			FixedRarityMin: model.RaritySRare,
			FixedCount:     1,
		},
		{
			PhaseId:   gachaId*model.PhaseIdMultiplier + 3,
			PriceType: model.PriceTypeConsumableItem,
			PriceId:   model.ConsumableIdPremiumTicket,
			DrawCount: 1,
		},
	}
}

func applyGuaranteedTicketAvailability(entries []store.GachaCatalogEntry) {
	maxSortOrder := int32(0)
	var startDatetime int64
	var endDatetime int64
	for _, entry := range entries {
		if model.IsGuaranteedTicketGacha(entry.GachaId) {
			continue
		}
		if entry.SortOrder > maxSortOrder {
			maxSortOrder = entry.SortOrder
		}
		if entry.StartDatetime > 0 && (startDatetime == 0 || entry.StartDatetime < startDatetime) {
			startDatetime = entry.StartDatetime
		}
		if entry.EndDatetime > endDatetime {
			endDatetime = entry.EndDatetime
		}
	}
	for i := range entries {
		if !model.IsGuaranteedTicketGacha(entries[i].GachaId) {
			continue
		}
		maxSortOrder++
		entries[i].SortOrder = maxSortOrder
		entries[i].StartDatetime = startDatetime
		entries[i].EndDatetime = endDatetime
	}
}

func ContentHash(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func FileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func BuildPremiumCatalog(config *Config, source *masterdata.GachaCatalog, entries []store.GachaCatalogEntry, options BuildOptions) (*PremiumCatalog, error) {
	if config == nil {
		return nil, fmt.Errorf("Gacha config is nil")
	}
	normalizeConfig(config)
	if err := validateConfigShape(config, source, entries, options); err != nil {
		return nil, err
	}

	catalog := &PremiumCatalog{Config: config, Banners: make(map[int32]*PremiumBannerPool)}

	weaponIds := make([]int32, 0, len(source.EligibleWeaponById))
	for weaponId := range source.EligibleWeaponById {
		weaponIds = append(weaponIds, weaponId)
	}
	sort.Slice(weaponIds, func(i, j int) bool { return weaponIds[i] < weaponIds[j] })

	for _, entry := range entries {
		if entry.GachaLabelType != model.GachaLabelPremium {
			continue
		}
		bannerConfig := config.Banners[entry.GachaId]
		if model.IsGuaranteedTicketGacha(entry.GachaId) {
			bannerConfig = BannerConfig{}
		}
		allowedSets := stringSet(bannerConfig.LimitedSets)
		pickupSet := int32Set(bannerConfig.PickupWeaponIds)
		groups := make([]PremiumGroup, len(groupDefinitions))
		groupIndex := make(map[[2]int32]int, len(groupDefinitions))
		for i, definition := range groupDefinitions {
			groups[i] = PremiumGroup{
				Id:        definition.id,
				GrantType: definition.grantType,
				Star:      definition.star,
				Rarity:    definition.rarity,
				Weight:    config.GroupWeights.weight(definition.grantType, definition.star),
			}
			groupIndex[groupKey(definition.grantType, definition.rarity)] = i
		}

		banner := &PremiumBannerPool{
			GachaId:         entry.GachaId,
			Groups:          groups,
			ItemsByWeaponId: make(map[int32]PoolItem),
		}
		for _, weaponId := range weaponIds {
			weaponConfig, configured := config.Weapons[weaponId]
			if configured && weaponConfig.Availability == AvailabilityEvent {
				continue
			}
			if configured && weaponConfig.Availability == AvailabilityLimited && !allowedSets[weaponConfig.LimitedSet] {
				continue
			}
			weapon := source.EligibleWeaponById[weaponId]
			item := PoolItem{WeaponId: weaponId, RarityType: weapon.RarityType}
			grantType := GrantWeaponOnly
			if costume, paired := source.CostumeByWeaponId[weaponId]; paired {
				item.CostumeId = costume.PossessionId
				item.CharacterId = costume.CharacterId
				grantType = GrantCharacterWeapon
			}
			index := groupIndex[groupKey(grantType, weapon.RarityType)]
			if pickupSet[weaponId] {
				banner.Groups[index].Pickup = append(banner.Groups[index].Pickup, item)
			} else {
				banner.Groups[index].NonPickup = append(banner.Groups[index].NonPickup, item)
			}
			banner.ItemsByWeaponId[weaponId] = item
		}

		for _, pickupId := range bannerConfig.PickupWeaponIds {
			if _, ok := banner.ItemsByWeaponId[pickupId]; !ok {
				return nil, fmt.Errorf("Gacha %d pickup weapon %d is not in the final banner pool", entry.GachaId, pickupId)
			}
		}
		for _, group := range banner.Groups {
			if group.Weight > 0 && group.ItemCount() == 0 {
				return nil, fmt.Errorf("Gacha %d group %s has positive weight but no candidates", entry.GachaId, group.Id)
			}
			if len(group.Pickup) > 0 && len(group.NonPickup) == 0 {
				return nil, fmt.Errorf("Gacha %d group %s contains pickup items but no non-pickup items", entry.GachaId, group.Id)
			}
		}
		baseWeights := make([]int, len(banner.Groups))
		for i, group := range banner.Groups {
			baseWeights[i] = group.Weight
		}
		for i, tenthWeight := range transferTwoStarWeightsToThreeStar(banner.Groups, baseWeights) {
			if tenthWeight > 0 && banner.Groups[i].ItemCount() == 0 {
				return nil, fmt.Errorf("Gacha %d tenth-draw group %s has transferred weight but no candidates", entry.GachaId, banner.Groups[i].Id)
			}
		}
		for _, phase := range entry.PricePhases {
			if phase.FixedCount <= 0 {
				continue
			}
			available := false
			for _, group := range banner.Groups {
				if group.Weight > 0 && group.ItemCount() > 0 && group.Rarity >= phase.FixedRarityMin {
					available = true
					break
				}
			}
			if !available {
				return nil, fmt.Errorf("Gacha %d phase %d has no positive-weight group for fixed rarity %d", entry.GachaId, phase.PhaseId, phase.FixedRarityMin)
			}
		}
		catalog.Banners[entry.GachaId] = banner
	}
	return catalog, nil
}

func ApplyConfiguredPromotions(entries []store.GachaCatalogEntry, catalog *PremiumCatalog) {
	if catalog == nil || catalog.Config == nil {
		return
	}
	for i := range entries {
		if entries[i].GachaLabelType != model.GachaLabelPremium {
			continue
		}
		entries[i].PromotionItems = nil
		if model.IsGuaranteedTicketGacha(entries[i].GachaId) {
			// The Japanese confirm_sr/confirm_ssr assets have no appeal image, and
			// the client cannot initialize a Gacha with zero appeal parts.
			entries[i].PromotionItems = guaranteedTicketPromotions(entries[i].GachaId, catalog.Banners[entries[i].GachaId])
			continue
		}
		banner := catalog.Banners[entries[i].GachaId]
		if banner == nil {
			continue
		}
		for _, weaponId := range catalog.Config.Banners[entries[i].GachaId].PickupWeaponIds {
			item, ok := banner.ItemsByWeaponId[weaponId]
			if !ok {
				continue
			}
			promotion := store.GachaPromotionItem{IsTarget: true}
			if item.CostumeId != 0 {
				promotion.PossessionType = int32(model.PossessionTypeCostume)
				promotion.PossessionId = item.CostumeId
				promotion.BonusPossessionType = int32(model.PossessionTypeWeapon)
				promotion.BonusPossessionId = item.WeaponId
			} else {
				promotion.PossessionType = int32(model.PossessionTypeWeapon)
				promotion.PossessionId = item.WeaponId
			}
			entries[i].PromotionItems = append(entries[i].PromotionItems, promotion)
		}
	}
}

var guaranteedThreeStarPromotionWeaponIds = []int32{
	220021, // Dissenting Traveler / 異存たる冒険者
	210031, // Dissenting Exile / 異存たる亡命者
}

var guaranteedFourStarPromotionWeaponIds = []int32{
	320081, // Abstract Assassin / 形而上の剣客
	350161, // Abstract Gunman / 形而上の射手
	330001, // Abstract Hunter / 形而上の狩人
}

func guaranteedTicketPromotions(gachaId int32, banner *PremiumBannerPool) []store.GachaPromotionItem {
	if banner == nil {
		return nil
	}
	var weaponIds []int32
	switch gachaId {
	case model.GachaIdGuaranteedThreeStarOrHigher:
		weaponIds = guaranteedThreeStarPromotionWeaponIds
	case model.GachaIdGuaranteedFourStar:
		weaponIds = guaranteedFourStarPromotionWeaponIds
	default:
		return nil
	}
	promotions := make([]store.GachaPromotionItem, 0, len(weaponIds))
	for _, weaponId := range weaponIds {
		if item, ok := banner.ItemsByWeaponId[weaponId]; ok {
			promotions = append(promotions, promotionFromPoolItem(item))
		}
	}
	return promotions
}

func promotionFromPoolItem(item PoolItem) store.GachaPromotionItem {
	promotion := store.GachaPromotionItem{IsTarget: true}
	if item.CostumeId != 0 {
		promotion.PossessionType = int32(model.PossessionTypeCostume)
		promotion.PossessionId = item.CostumeId
		promotion.BonusPossessionType = int32(model.PossessionTypeWeapon)
		promotion.BonusPossessionId = item.WeaponId
	} else {
		promotion.PossessionType = int32(model.PossessionTypeWeapon)
		promotion.PossessionId = item.WeaponId
	}
	return promotion
}

func validateConfigShape(config *Config, source *masterdata.GachaCatalog, entries []store.GachaCatalogEntry, options BuildOptions) error {
	if config.Version != ConfigVersion {
		return fmt.Errorf("unsupported Gacha config version %d", config.Version)
	}
	if options.RequireComplete && options.CurrentMasterDataHash != "" && config.SourceMasterDataHash != options.CurrentMasterDataHash {
		return fmt.Errorf("sourceMasterDataHash does not match the current master data")
	}
	if _, ok := config.GroupWeights.weaponOnlyTwoStarRemainder(); !ok {
		return fmt.Errorf("Gacha group probabilities other than 2-star weapon must be non-negative and total at most 100%%")
	}
	totalWeight := 0
	for _, definition := range groupDefinitions {
		weight := config.GroupWeights.weight(definition.grantType, definition.star)
		if weight < 0 {
			return fmt.Errorf("group %s has a negative weight", definition.id)
		}
		totalWeight += weight
	}
	if totalWeight != GroupWeightTotal {
		return fmt.Errorf("Gacha group weights must total %d", GroupWeightTotal)
	}

	for id, limitedSet := range config.LimitedSets {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("limited set id must not be empty")
		}
		if strings.TrimSpace(limitedSet.DisplayName) == "" {
			return fmt.Errorf("limited set %q has an empty display name", id)
		}
	}
	for weaponId, weaponConfig := range config.Weapons {
		if _, ok := source.ConfigurableWeaponById[weaponId]; !ok {
			return fmt.Errorf("configured weapon %d is not a configurable root weapon", weaponId)
		}
		switch weaponConfig.Availability {
		case "", AvailabilityStandard:
			if weaponConfig.LimitedSet != "" {
				return fmt.Errorf("standard weapon %d must not specify limitedSet", weaponId)
			}
			if _, ok := source.EligibleWeaponById[weaponId]; !ok {
				return fmt.Errorf("standard weapon %d is not eligible for Gacha", weaponId)
			}
		case AvailabilityEvent:
			if weaponConfig.LimitedSet != "" {
				return fmt.Errorf("event weapon %d must not specify limitedSet", weaponId)
			}
		case AvailabilityLimited:
			if _, ok := config.LimitedSets[weaponConfig.LimitedSet]; !ok {
				return fmt.Errorf("limited weapon %d references unknown limited set %q", weaponId, weaponConfig.LimitedSet)
			}
			if _, ok := source.EligibleWeaponById[weaponId]; !ok {
				return fmt.Errorf("limited weapon %d is not eligible for Gacha", weaponId)
			}
		default:
			return fmt.Errorf("weapon %d has invalid availability %q", weaponId, weaponConfig.Availability)
		}
	}
	premiumGachaIds := make(map[int32]bool)
	for _, entry := range entries {
		if entry.GachaLabelType == model.GachaLabelPremium {
			premiumGachaIds[entry.GachaId] = true
		}
	}
	for gachaId, banner := range config.Banners {
		if !premiumGachaIds[gachaId] {
			return fmt.Errorf("configured Gacha %d is not a premium weapon banner", gachaId)
		}
		if banner.StartDatetime <= 0 || banner.EndDatetime <= 0 {
			return fmt.Errorf("Gacha %d must have a start and end timestamp", gachaId)
		}
		if banner.EndDatetime != 0 && banner.EndDatetime < banner.StartDatetime {
			return fmt.Errorf("Gacha %d ends before it starts", gachaId)
		}
		if !model.IsGuaranteedTicketGacha(gachaId) {
			assetGachaId, err := strconv.ParseInt(strings.TrimPrefix(banner.BannerAssetName, model.BannerPrefixLimited), 10, 32)
			if err != nil || !strings.HasPrefix(banner.BannerAssetName, model.BannerPrefixLimited) || int32(assetGachaId) != gachaId {
				return fmt.Errorf("Gacha %d has invalid limited banner asset %q", gachaId, banner.BannerAssetName)
			}
		}
		seenSet := make(map[string]bool)
		for _, limitedSet := range banner.LimitedSets {
			if _, ok := config.LimitedSets[limitedSet]; !ok {
				return fmt.Errorf("Gacha %d references unknown limited set %q", gachaId, limitedSet)
			}
			if seenSet[limitedSet] {
				return fmt.Errorf("Gacha %d repeats limited set %q", gachaId, limitedSet)
			}
			seenSet[limitedSet] = true
		}
		seenPickup := make(map[int32]bool)
		for _, weaponId := range banner.PickupWeaponIds {
			if seenPickup[weaponId] {
				return fmt.Errorf("Gacha %d repeats pickup weapon %d", gachaId, weaponId)
			}
			seenPickup[weaponId] = true
			weaponConfig, configured := config.Weapons[weaponId]
			if !configured {
				if _, eligible := source.EligibleWeaponById[weaponId]; !eligible {
					return fmt.Errorf("Gacha %d pickup weapon %d is not eligible for Gacha", gachaId, weaponId)
				}
				continue
			}
			if weaponConfig.Availability == AvailabilityEvent {
				return fmt.Errorf("Gacha %d pickup weapon %d is event-only", gachaId, weaponId)
			}
		}
	}
	return validateBoxConfigs(config, entries, options.ValidBoxRewards)
}

func normalizeConfig(config *Config) {
	config.GroupWeights.CharacterWeapon.TwoStar = 0
	config.GroupWeights.calculateWeaponOnlyTwoStar()
	if config.LimitedSets == nil {
		config.LimitedSets = make(map[string]LimitedSetConfig)
	}
	if config.Weapons == nil {
		config.Weapons = make(map[int32]WeaponConfig)
	}
	for weaponId, weapon := range config.Weapons {
		if (weapon.Availability == "" || weapon.Availability == AvailabilityStandard) && weapon.LimitedSet == "" {
			delete(config.Weapons, weaponId)
		}
	}
	if config.Banners == nil {
		config.Banners = make(map[int32]BannerConfig)
	}
	for gachaId, banner := range config.Banners {
		if banner.StartDatetime == 0 && banner.EndDatetime == 0 {
			banner.StartDatetime = DefaultBannerStartDatetime
			banner.EndDatetime = DefaultBannerEndDatetime
			config.Banners[gachaId] = banner
		}
	}
	if config.ChapterBanners == nil {
		config.ChapterBanners = make(map[int32]BoxConfig)
	}
	if config.EventBanners == nil {
		config.EventBanners = make(map[int32]EventBoxConfig)
	}
}

func (weights GroupWeights) weaponOnlyTwoStarRemainder() (int, bool) {
	remaining := GroupWeightTotal
	for _, weight := range []int{
		weights.CharacterWeapon.ThreeStar,
		weights.CharacterWeapon.FourStar,
		weights.WeaponOnly.ThreeStar,
		weights.WeaponOnly.FourStar,
	} {
		if weight < 0 || weight > remaining {
			return -1, false
		}
		remaining -= weight
	}
	return remaining, true
}

func (weights *GroupWeights) calculateWeaponOnlyTwoStar() {
	remaining, _ := weights.weaponOnlyTwoStarRemainder()
	weights.WeaponOnly.TwoStar = remaining
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("decode Gacha config: multiple JSON values")
		}
		return fmt.Errorf("decode Gacha config: %w", err)
	}
	return nil
}

func (weights GroupWeights) weight(grantType GrantType, star int32) int {
	selected := weights.WeaponOnly
	if grantType == GrantCharacterWeapon {
		selected = weights.CharacterWeapon
	}
	switch star {
	case 2:
		return selected.TwoStar
	case 3:
		return selected.ThreeStar
	case 4:
		return selected.FourStar
	default:
		return 0
	}
}

func groupKey(grantType GrantType, rarity model.RarityType) [2]int32 {
	grant := int32(0)
	if grantType == GrantCharacterWeapon {
		grant = 1
	}
	return [2]int32{grant, rarity}
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func int32Set(values []int32) map[int32]bool {
	set := make(map[int32]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}
