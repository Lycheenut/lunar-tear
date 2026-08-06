package masterdataadmin

import (
	"fmt"
	"sort"
	"strings"

	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

type GachaWeaponReference struct {
	WeaponId      int32             `json:"weaponId"`
	WeaponNames   map[string]string `json:"weaponNames,omitempty"`
	CostumeId     int32             `json:"costumeId,omitempty"`
	CostumeNames  map[string]string `json:"costumeNames,omitempty"`
	WeaponType    int32             `json:"weaponType"`
	AttributeType int32             `json:"attributeType"`
	Star          int32             `json:"star"`
	GrantType     gacha.GrantType   `json:"grantType"`
	Eligible      bool              `json:"eligible"`
}

type GachaBannerReference struct {
	GachaId         int32             `json:"gachaId"`
	Titles          map[string]string `json:"titles,omitempty"`
	BannerAssetName string            `json:"bannerAssetName,omitempty"`
	StartDatetime   int64             `json:"startDatetime"`
	EndDatetime     int64             `json:"endDatetime"`
}

type GachaEditorCatalog struct {
	ContentHash     string                 `json:"contentHash"`
	MasterDataHash  string                 `json:"masterDataHash"`
	ConfigExists    bool                   `json:"configExists"`
	DefaultLanguage string                 `json:"defaultLanguage"`
	Languages       []string               `json:"languages"`
	Config          *gacha.Config          `json:"config"`
	Weapons         []GachaWeaponReference `json:"weapons"`
	Banners         []GachaBannerReference `json:"banners"`
	Warnings        []string               `json:"warnings,omitempty"`
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
			}
		}
		result.Weapons = append(result.Weapons, reference)
	}

	seenBanners := make(map[int32]bool)
	for _, entry := range entries {
		if entry.GachaLabelType != model.GachaLabelPremium {
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

	if !configExists {
		result.Warnings = append(result.Warnings, "Gacha 配置尚未发布，当前所有可抽取武器按常驻处理，且无限定、无 Pickup。")
	} else if config.SourceMasterDataHash != masterDataHash {
		result.Warnings = append(result.Warnings, "Gacha 配置基于旧版主数据；新增可抽取武器已按常驻处理，请检查后重新发布。")
	}
	return result, nil
}

func weaponTitles(resolver *titleResolver, weapon masterdata.EntityMWeapon) map[string]string {
	prefix := "wp"
	if weapon.WeaponCategoryType == 2 {
		prefix = "mw"
	}
	assetName := fmt.Sprintf("%s%03d%03d", prefix, weapon.WeaponType, weapon.AssetVariationId)
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
	assetName := fmt.Sprintf("ch%03d%03d", costume.ActorSkeletonId, costume.AssetVariationId)
	for _, key := range []string{"costume.name.replace." + assetName, "costume.name." + assetName} {
		if titles := resolver.byKey(key); len(titles) > 0 {
			return titles
		}
	}
	return nil
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
