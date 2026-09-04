package masterdataadmin

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

type GachaWeaponReference struct {
	WeaponId        int32             `json:"weaponId"`
	WeaponNames     map[string]string `json:"weaponNames,omitempty"`
	IconPath        string            `json:"iconPath"`
	CostumeId       int32             `json:"costumeId,omitempty"`
	CostumeNames    map[string]string `json:"costumeNames,omitempty"`
	CostumeIconPath string            `json:"costumeIconPath,omitempty"`
	WeaponType      int32             `json:"weaponType"`
	AttributeType   int32             `json:"attributeType"`
	Star            int32             `json:"star"`
	GrantType       gacha.GrantType   `json:"grantType"`
	Eligible        bool              `json:"eligible"`
}

type GachaBannerReference struct {
	GachaId         int32             `json:"gachaId"`
	Titles          map[string]string `json:"titles,omitempty"`
	BannerAssetName string            `json:"bannerAssetName,omitempty"`
	StartDatetime   int64             `json:"startDatetime"`
	EndDatetime     int64             `json:"endDatetime"`
}

type GachaBoxBannerReference struct {
	GachaId                    int32             `json:"gachaId"`
	GachaLabelType             int32             `json:"gachaLabelType"`
	Titles                     map[string]string `json:"titles,omitempty"`
	BannerAssetName            string            `json:"bannerAssetName,omitempty"`
	StartDatetime              int64             `json:"startDatetime"`
	EndDatetime                int64             `json:"endDatetime"`
	RelatedMainQuestChapterId  int32             `json:"relatedMainQuestChapterId,omitempty"`
	RelatedEventQuestChapterId int32             `json:"relatedEventQuestChapterId,omitempty"`
	RequiredConsumableItemId   int32             `json:"requiredConsumableItemId,omitempty"`
	ConfiguredBoxCount         int32             `json:"configuredBoxCount"`
}

type GachaEditorCatalog struct {
	ContentHash     string                    `json:"contentHash"`
	MasterDataHash  string                    `json:"masterDataHash"`
	ConfigExists    bool                      `json:"configExists"`
	DefaultLanguage string                    `json:"defaultLanguage"`
	Languages       []string                  `json:"languages"`
	Config          *gacha.Config             `json:"config"`
	Weapons         []GachaWeaponReference    `json:"weapons"`
	Banners         []GachaBannerReference    `json:"banners"`
	BoxBanners      []GachaBoxBannerReference `json:"boxBanners"`
	Warnings        []string                  `json:"warnings,omitempty"`
}

// BuildGachaMomBannerUpdate builds the binary master-data candidate that keeps
// existing limited MomBanner rows synchronized with gacha.json schedules.
// Assets without a MomBanner row remain valid Gacha entries and need no row to
// be synthesized here.
func BuildGachaMomBannerUpdate(masterDataPath string, config *gacha.Config) ([]byte, UpdateResult, error) {
	if config == nil {
		return nil, UpdateResult{}, fmt.Errorf("Gacha config is nil")
	}
	if _, _, err := gacha.EncodeConfig(config); err != nil {
		return nil, UpdateResult{}, err
	}
	catalog, err := LoadTable(masterDataPath, "m_mom_banner")
	if err != nil {
		return nil, UpdateResult{}, err
	}
	var table *Table
	for index := range catalog.Tables {
		if catalog.Tables[index].Name == "m_mom_banner" {
			table = &catalog.Tables[index]
			break
		}
	}
	if table == nil {
		return nil, UpdateResult{}, fmt.Errorf("table %q is absent from the current master data", "m_mom_banner")
	}

	bannerByAsset := make(map[string]gacha.BannerConfig, len(config.Banners))
	for _, banner := range config.Banners {
		if strings.HasPrefix(banner.BannerAssetName, model.BannerPrefixLimited) {
			bannerByAsset[banner.BannerAssetName] = banner
		}
	}
	changes := make([]Change, 0)
	for _, row := range table.Rows {
		if row.Values["DestinationDomainType"] != fmt.Sprint(model.MomBannerDomainGacha) {
			continue
		}
		banner, ok := bannerByAsset[row.Values["BannerAssetName"]]
		if !ok {
			continue
		}
		if row.Times["StartDatetime"] != banner.StartDatetime {
			changes = append(changes, Change{Table: table.Name, Row: row.Index, Field: "StartDatetime", Value: banner.StartDatetime})
		}
		if row.Times["EndDatetime"] != banner.EndDatetime {
			changes = append(changes, Change{Table: table.Name, Row: row.Index, Field: "EndDatetime", Value: banner.EndDatetime})
		}
	}
	if len(changes) == 0 {
		return nil, UpdateResult{Version: catalog.Version}, nil
	}
	return BuildUpdate(masterDataPath, UpdateRequest{ExpectedVersion: catalog.Version, Changes: changes})
}

func LoadGachaEditorCatalog(
	masterDataPath string,
	pool *masterdata.GachaCatalog,
	weaponCatalog *masterdata.WeaponCatalog,
	costumeCatalog *masterdata.CostumeCatalog,
	entries []store.GachaCatalogEntry,
	config *gacha.Config,
	contentHash, masterDataHash string,
	configExists bool,
) (*GachaEditorCatalog, error) {
	file, err := memorydb.OpenFile(masterDataPath)
	if err != nil {
		return nil, err
	}
	texts := loadLocalizationIndex(masterDataPath)
	resolver := newTitleResolver(file, texts)
	result := &GachaEditorCatalog{
		ContentHash:     contentHash,
		MasterDataHash:  masterDataHash,
		ConfigExists:    configExists,
		DefaultLanguage: "en",
		Languages:       append([]string(nil), supportedLanguages...),
		Config:          gacha.ConfigWithoutAutomaticEventWeapons(config, pool),
	}

	weaponIds := make([]int32, 0, len(pool.EligibleWeaponById))
	for weaponId := range pool.EligibleWeaponById {
		weaponIds = append(weaponIds, weaponId)
	}
	sort.Slice(weaponIds, func(i, j int) bool { return weaponIds[i] < weaponIds[j] })
	for _, weaponId := range weaponIds {
		poolItem := pool.EligibleWeaponById[weaponId]
		weapon := weaponCatalog.Weapons[weaponId]
		reference := GachaWeaponReference{
			WeaponId:      weaponId,
			WeaponNames:   weaponTitles(resolver, weapon),
			IconPath:      rewardWeaponIconPath(weapon),
			WeaponType:    weapon.WeaponType,
			AttributeType: weapon.AttributeType,
			Star:          rarityStar(poolItem.RarityType),
			GrantType:     gacha.GrantWeaponOnly,
		}
		reference.Eligible = true
		if costume, paired := pool.CostumeByWeaponId[weaponId]; paired {
			reference.GrantType = gacha.GrantCharacterWeapon
			reference.CostumeId = costume.PossessionId
			if master, ok := costumeCatalog.Costumes[costume.PossessionId]; ok {
				reference.CostumeNames = costumeTitles(resolver, master)
				reference.CostumeIconPath = costumeIconPath(master)
			}
		}
		result.Weapons = append(result.Weapons, reference)
	}

	seenBanners := make(map[int32]bool)
	for _, entry := range entries {
		if entry.GachaLabelType == model.GachaLabelChapter || entry.GachaLabelType == model.GachaLabelEvent {
			titles := resolver.byKey("gacha.title." + entry.BannerAssetName)
			if entry.GachaLabelType == model.GachaLabelEvent {
				titles = resolver.byKey(fmt.Sprintf("quest.event.chapter_title.%d", entry.DescriptionTextId))
			}
			boxCount := entry.BoxCount
			if event, ok := result.Config.EventBanners[entry.GachaId]; ok {
				boxCount = int32(len(event.Boxes))
			}
			result.BoxBanners = append(result.BoxBanners, GachaBoxBannerReference{
				GachaId:                    entry.GachaId,
				GachaLabelType:             entry.GachaLabelType,
				Titles:                     titles,
				BannerAssetName:            entry.BannerAssetName,
				StartDatetime:              entry.StartDatetime,
				EndDatetime:                entry.EndDatetime,
				RelatedMainQuestChapterId:  entry.RelatedMainQuestChapterId,
				RelatedEventQuestChapterId: entry.RelatedEventQuestChapterId,
				RequiredConsumableItemId:   entry.RequiredConsumableItemId,
				ConfiguredBoxCount:         boxCount,
			})
			continue
		}
		if entry.GachaLabelType != model.GachaLabelPremium {
			continue
		}
		if !strings.HasPrefix(entry.BannerAssetName, model.BannerPrefixLimited) {
			continue
		}
		if seenBanners[entry.GachaId] {
			continue
		}
		seenBanners[entry.GachaId] = true
		titles := resolver.byKey("gacha.title." + entry.BannerAssetName)
		if len(titles) == 0 {
			if suffix, ok := strings.CutPrefix(entry.BannerAssetName, "limited_"); ok {
				titles = resolver.byKey("gacha.title.limitd_" + suffix)
			}
		}
		result.Banners = append(result.Banners, GachaBannerReference{
			GachaId:         entry.GachaId,
			Titles:          titles,
			BannerAssetName: entry.BannerAssetName,
			StartDatetime:   entry.StartDatetime,
			EndDatetime:     entry.EndDatetime,
		})
	}
	sort.Slice(result.Banners, func(i, j int) bool { return result.Banners[i].GachaId < result.Banners[j].GachaId })
	sort.Slice(result.BoxBanners, func(i, j int) bool {
		if result.BoxBanners[i].GachaLabelType != result.BoxBanners[j].GachaLabelType {
			return result.BoxBanners[i].GachaLabelType < result.BoxBanners[j].GachaLabelType
		}
		return result.BoxBanners[i].GachaId < result.BoxBanners[j].GachaId
	})

	if !configExists {
		result.Warnings = append(result.Warnings, "Gacha 配置尚未发布：可抽取武器按常驻处理，且无限定、无 Pickup；Chapter 与 Event 奖励箱在配置前不会开放。")
	} else if config.SourceMasterDataHash != masterDataHash {
		result.Warnings = append(result.Warnings, "Gacha 配置基于旧版主数据；新增可抽取武器已按常驻处理，请检查后重新发布。")
	}
	return result, nil
}

func weaponTitles(resolver *titleResolver, weapon masterdata.EntityMWeapon) map[string]string {
	assetName := rewardWeaponAssetName(weapon)
	for _, key := range []string{
		"weapon.name.replace." + assetName + ".1",
		"weapon.name." + assetName + ".1",
		"weapon.name.replace." + assetName + ".2",
		"weapon.name." + assetName + ".2",
	} {
		if titles := resolver.byKey(key); len(titles) > 0 {
			return titles
		}
	}
	return nil
}

func costumeTitles(resolver *titleResolver, costume masterdata.EntityMCostume) map[string]string {
	assetName := costumeAssetName(costume)
	for _, key := range []string{"costume.name.replace." + assetName, "costume.name." + assetName} {
		if titles := resolver.byKey(key); len(titles) > 0 {
			return titles
		}
	}
	return nil
}

func costumeAssetName(costume masterdata.EntityMCostume) string {
	return fmt.Sprintf("ch%03d%03d", costume.ActorSkeletonId, costume.AssetVariationId)
}

func costumeIconPath(costume masterdata.EntityMCostume) string {
	assetName := costumeAssetName(costume)
	return path.Join("costume", assetName, assetName+"_standard.png")
}

func rarityStar(rarity model.RarityType) int32 {
	switch rarity {
	case model.RarityRare:
		return 2
	case model.RaritySRare:
		return 3
	case model.RaritySSRare:
		return 4
	default:
		return 0
	}
}
