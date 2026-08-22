package runtime

import (
	"fmt"
	"log"

	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/importantitem"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/userdata"
)

// buildCatalogs runs the full Load*/Build*/Enrich* sequence against whatever
// memorydb currently holds and returns a fully populated *Catalogs. Called
// once at startup and again on every reload.
func buildCatalogs(gachaConfig *gacha.Config, gachaConfigHash string, gachaConfigExists bool, masterDataHash string, requireCompleteGacha bool) (*Catalogs, error) {
	log.Printf("master data loaded (%d tables)", memorydb.TableCount())

	gameConfig, err := masterdata.LoadGameConfig()
	if err != nil {
		return nil, fmt.Errorf("load game config: %w", err)
	}
	log.Printf("game config loaded (goldId=%d, skipTicketId=%d, rebirthGold=%d)",
		gameConfig.ConsumableItemIdForGold, gameConfig.ConsumableItemIdForQuestSkipTicket, gameConfig.CharacterRebirthConsumeGold)

	partsCatalog, err := masterdata.LoadPartsCatalog()
	if err != nil {
		return nil, fmt.Errorf("load parts catalog: %w", err)
	}
	log.Printf("parts catalog loaded: %d parts, %d rarities", len(partsCatalog.PartsById), len(partsCatalog.RarityByRarityType))

	conditionResolver, err := masterdata.LoadConditionResolver()
	if err != nil {
		return nil, fmt.Errorf("load condition resolver: %w", err)
	}

	questCatalog, err := masterdata.LoadQuestCatalog(partsCatalog, conditionResolver)
	if err != nil {
		return nil, fmt.Errorf("load quest catalog: %w", err)
	}
	missionCatalog, err := masterdata.LoadMissionCatalog()
	if err != nil {
		return nil, fmt.Errorf("load mission catalog: %w", err)
	}
	sideStoryCatalog := masterdata.LoadSideStoryCatalog()
	campaignCatalog, err := campaign.Load()
	if err != nil {
		return nil, fmt.Errorf("load campaign catalog: %w", err)
	}
	log.Printf("campaign catalog loaded: %d enhance, %d quest", campaignCatalog.EnhanceCount(), campaignCatalog.QuestCount())
	importantItemEffects, err := importantitem.Load()
	if err != nil {
		return nil, fmt.Errorf("load important item effects: %w", err)
	}
	log.Printf("important item effects loaded: %d server-side effects", importantItemEffects.EffectCount())
	characterRebirthCatalog, err := masterdata.LoadCharacterRebirthCatalog()
	if err != nil {
		return nil, fmt.Errorf("load character rebirth catalog: %w", err)
	}
	log.Printf("character rebirth catalog loaded: %d characters", len(characterRebirthCatalog.StepGroupByCharacterId))

	questHandler := questflow.NewQuestHandler(questCatalog, gameConfig, sideStoryCatalog, campaignCatalog, importantItemEffects, characterRebirthCatalog)
	userdata.SetQuestHandler(questHandler)

	gachaEntries, medalInfo, err := masterdata.LoadGachaCatalog()
	if err != nil {
		return nil, fmt.Errorf("load gacha catalog: %w", err)
	}
	masterdata.EnrichGachaUnlockConditions(gachaEntries, questCatalog)
	log.Printf("gacha catalog loaded: %d entries", len(gachaEntries))

	gachaPool, err := masterdata.LoadGachaPool()
	if err != nil {
		return nil, fmt.Errorf("load gacha pool: %w", err)
	}
	log.Printf("gacha pool loaded: costumes=%d rarities, weapons=%d rarities, materials=%d",
		len(gachaPool.CostumesByRarity), len(gachaPool.WeaponsByRarity), len(gachaPool.Materials))

	shopCatalog, err := masterdata.LoadShopCatalog()
	if err != nil {
		return nil, fmt.Errorf("load shop catalog: %w", err)
	}
	log.Printf("shop catalog loaded: %d items, %d content groups, %d exchange shops",
		len(shopCatalog.Items), len(shopCatalog.Contents), len(shopCatalog.ExchangeShopCells))

	gachaPool.PruneUnpairedCostumes()
	boxRewardKeys, err := masterdata.LoadGachaBoxRewardKeys()
	if err != nil {
		return nil, fmt.Errorf("load Gacha box reward references: %w", err)
	}
	premiumGacha, err := gacha.BuildPremiumCatalog(gachaConfig, gachaPool, gachaEntries, gacha.BuildOptions{
		RequireComplete:       requireCompleteGacha,
		CurrentMasterDataHash: masterDataHash,
		ValidBoxRewards:       boxRewardKeys,
	})
	if err != nil {
		return nil, fmt.Errorf("build configured Gacha pools: %w", err)
	}
	masterdata.EnrichCatalogPromotions(gachaEntries, gachaPool)
	gacha.ApplyConfiguredPromotions(gachaEntries, premiumGacha)
	gacha.ApplyConfiguredBoxes(gachaEntries, gachaConfig)

	dupExchange, err := masterdata.LoadDupExchange()
	if err != nil {
		return nil, fmt.Errorf("load dup exchange: %w", err)
	}
	dupAdded, err := masterdata.EnrichDupExchange(dupExchange, gachaPool)
	if err != nil {
		return nil, fmt.Errorf("enrich dup exchange: %w", err)
	}
	log.Printf("dup exchange loaded: %d entries (%d derived from fallback materials)", len(dupExchange), dupAdded)
	companionDupExchange, err := masterdata.LoadCompanionDupExchange()
	if err != nil {
		return nil, fmt.Errorf("load companion duplicate exchange: %w", err)
	}
	questHandler.Granter.CostumeDupExchange = dupExchange
	questHandler.Granter.CompanionDupExchange = companionDupExchange

	gachaHandler := gacha.NewGachaHandler(gachaPool, premiumGacha, gameConfig, questHandler.Granter, medalInfo, dupExchange)

	cageOrnamentCatalog := masterdata.LoadCageOrnamentCatalog()
	loginBonusCatalog := masterdata.LoadLoginBonusCatalog()
	characterViewerCatalog := masterdata.LoadCharacterViewerCatalog(conditionResolver)
	omikujiCatalog := masterdata.LoadOmikujiCatalog()

	materialCatalog, err := masterdata.LoadMaterialCatalog()
	if err != nil {
		return nil, fmt.Errorf("load material catalog: %w", err)
	}
	log.Printf("material catalog loaded: %d materials", len(materialCatalog.All))

	consumableItemCatalog, err := masterdata.LoadConsumableItemCatalog()
	if err != nil {
		return nil, fmt.Errorf("load consumable item catalog: %w", err)
	}
	log.Printf("consumable item catalog loaded: %d items", len(consumableItemCatalog.All))

	costumeCatalog, err := masterdata.LoadCostumeCatalog(materialCatalog)
	if err != nil {
		return nil, fmt.Errorf("load costume catalog: %w", err)
	}
	log.Printf("costume catalog loaded: %d costumes, %d materials, %d rarity curves", len(costumeCatalog.Costumes), len(costumeCatalog.Materials), len(costumeCatalog.ExpByRarity))

	weaponCatalog, err := masterdata.LoadWeaponCatalog(materialCatalog)
	if err != nil {
		return nil, fmt.Errorf("load weapon catalog: %w", err)
	}
	log.Printf("weapon catalog loaded: %d weapons, %d materials, %d enhance configs", len(weaponCatalog.Weapons), len(weaponCatalog.Materials), len(weaponCatalog.ExpByEnhanceId))

	exploreCatalog, err := masterdata.LoadExploreCatalog()
	if err != nil {
		return nil, fmt.Errorf("load explore catalog: %w", err)
	}
	log.Printf("explore catalog loaded: %d explores, %d grade assets", len(exploreCatalog.Explores), len(exploreCatalog.GradeAssets))

	gimmickCatalog, err := masterdata.LoadGimmickCatalog(conditionResolver, cageOrnamentCatalog)
	if err != nil {
		return nil, fmt.Errorf("load gimmick catalog: %w", err)
	}

	characterBoardCatalog, err := masterdata.LoadCharacterBoardCatalog()
	if err != nil {
		return nil, fmt.Errorf("load character board catalog: %w", err)
	}
	log.Printf("character board catalog loaded: %d panels, %d boards", len(characterBoardCatalog.PanelById), len(characterBoardCatalog.BoardById))

	companionCatalog, err := masterdata.LoadCompanionCatalog()
	if err != nil {
		return nil, fmt.Errorf("load companion catalog: %w", err)
	}
	log.Printf("companion catalog loaded: %d companions, %d categories", len(companionCatalog.CompanionById), len(companionCatalog.GoldCostByCategory))

	bigHuntCatalog := masterdata.LoadBigHuntCatalog()

	towerCatalog := masterdata.LoadTowerCatalog()

	labyrinthCatalog := masterdata.LoadLabyrinthCatalog()
	limitContentCatalog, err := masterdata.LoadLimitContentCatalog()
	if err != nil {
		return nil, fmt.Errorf("load limit content catalog: %w", err)
	}

	return &Catalogs{
		MasterDataHash:    masterDataHash,
		GameConfig:        gameConfig,
		Parts:             partsCatalog,
		Quest:             questCatalog,
		Mission:           missionCatalog,
		GachaEntries:      gachaEntries,
		GachaMedals:       medalInfo,
		GachaPool:         gachaPool,
		GachaConfig:       gachaConfig,
		GachaConfigHash:   gachaConfigHash,
		GachaConfigExists: gachaConfigExists,
		PremiumGacha:      premiumGacha,
		Shop:              shopCatalog,
		DupExchange:       dupExchange,
		ConditionResolver: conditionResolver,
		CageOrnament:      cageOrnamentCatalog,
		LoginBonus:        loginBonusCatalog,
		CharacterViewer:   characterViewerCatalog,
		Omikuji:           omikujiCatalog,
		Material:          materialCatalog,
		ConsumableItem:    consumableItemCatalog,
		Costume:           costumeCatalog,
		Weapon:            weaponCatalog,
		Explore:           exploreCatalog,
		Gimmick:           gimmickCatalog,
		CharacterBoard:    characterBoardCatalog,
		CharacterRebirth:  characterRebirthCatalog,
		Companion:         companionCatalog,
		SideStory:         sideStoryCatalog,
		BigHunt:           bigHuntCatalog,
		Tower:             towerCatalog,
		Labyrinth:         labyrinthCatalog,
		LimitContent:      limitContentCatalog,
		Campaign:          campaignCatalog,
		ImportantItems:    importantItemEffects,
		QuestHandler:      questHandler,
		GachaHandler:      gachaHandler,
	}, nil
}
