package masterdata

import (
	"fmt"
	"sort"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/utils"
)

type ExchangeShopCell struct {
	SortOrder  int32
	ShopItemId int32
}

type ShopCell struct {
	LimitedOpenId int32
	StartDatetime int64
	EndDatetime   int64
}

type ShopLimitedStockRule struct {
	MaxCount        int32
	AutoResetType   int32
	AutoResetPeriod int32
}

type ShopCatalog struct {
	Items             map[int32]EntityMShopItem
	Contents          map[int32][]EntityMShopItemContentPossession
	Effects           map[int32][]EntityMShopItemContentEffect
	MaxStaminaMillis  map[int32]int32 // level -> max stamina in millis
	LimitedStock      map[int32]ShopLimitedStockRule
	Shops             map[int32]EntityMShop
	CellsByShop       map[int32]map[int32][]ShopCell // shopId -> itemId -> cells
	UserLevelsByItem  map[int32][]EntityMShopItemUserLevelCondition
	AdditionalContent map[int32][]EntityMShopItemAdditionalContent
	ReplaceableGems   []EntityMShopReplaceableGem // sorted by refresh-count lower limit
	ItemShopId        int32
	ItemShopPool      []int32                      // shop item IDs for the replaceable item shop, sorted by cell sort order
	ExchangeShopCells map[int32][]ExchangeShopCell // shopId -> sorted cells for exchange shops
}

func (c *ShopCatalog) IsShopOpen(shopId int32, nowMillis int64) bool {
	shop, ok := c.Shops[shopId]
	return ok && shop.LimitedOpenId == 0 && inShopTerm(shop.StartDatetime, shop.EndDatetime, nowMillis)
}

func (c *ShopCatalog) IsItemAvailable(shopId, shopItemId int32, nowMillis int64) bool {
	if !c.IsShopOpen(shopId, nowMillis) {
		return false
	}
	for _, cell := range c.CellsByShop[shopId][shopItemId] {
		if cell.LimitedOpenId == 0 && inShopTerm(cell.StartDatetime, cell.EndDatetime, nowMillis) {
			return true
		}
	}
	return false
}

func (c *ShopCatalog) AdditionalContentsForLevel(shopItemId, userLevel int32) ([]EntityMShopItemAdditionalContent, bool) {
	conditions := c.UserLevelsByItem[shopItemId]
	if len(conditions) == 0 {
		return nil, true
	}
	for _, condition := range conditions {
		if (condition.UserLevelLowerLimit == 0 || userLevel >= condition.UserLevelLowerLimit) &&
			(condition.UserLevelUpperLimit == 0 || userLevel <= condition.UserLevelUpperLimit) {
			return c.AdditionalContent[condition.ShopItemAdditionalContentId], true
		}
	}
	return nil, false
}

func (c *ShopCatalog) ReplaceableGemPrice(refreshCount int32) (int32, bool) {
	var price int32
	found := false
	for _, row := range c.ReplaceableGems {
		if refreshCount < row.LineupUpdateCountLowerLimit {
			break
		}
		price = row.NecessaryGem
		found = true
	}
	return price, found
}

func inShopTerm(startDatetime, endDatetime, nowMillis int64) bool {
	return (startDatetime == 0 || nowMillis >= startDatetime) &&
		(endDatetime == 0 || nowMillis <= endDatetime)
}

func LoadShopCatalog() (*ShopCatalog, error) {
	items, err := utils.ReadTable[EntityMShopItem]("m_shop_item")
	if err != nil {
		return nil, fmt.Errorf("load shop item table: %w", err)
	}
	contents, err := utils.ReadTable[EntityMShopItemContentPossession]("m_shop_item_content_possession")
	if err != nil {
		return nil, fmt.Errorf("load shop content possession table: %w", err)
	}
	effects, err := utils.ReadTable[EntityMShopItemContentEffect]("m_shop_item_content_effect")
	if err != nil {
		return nil, fmt.Errorf("load shop content effect table: %w", err)
	}
	userLevels, err := utils.ReadTable[EntityMUserLevel]("m_user_level")
	if err != nil {
		return nil, fmt.Errorf("load user level table: %w", err)
	}
	stockRows, err := utils.ReadTable[EntityMShopItemLimitedStock]("m_shop_item_limited_stock")
	if err != nil {
		return nil, fmt.Errorf("load shop item limited stock table: %w", err)
	}
	levelRows, err := utils.ReadTable[EntityMShopItemUserLevelCondition]("m_shop_item_user_level_condition")
	if err != nil {
		return nil, fmt.Errorf("load shop item user level condition table: %w", err)
	}
	additionalContentRows, err := utils.ReadTable[EntityMShopItemAdditionalContent]("m_shop_item_additional_content")
	if err != nil {
		return nil, fmt.Errorf("load shop item additional content table: %w", err)
	}
	gemRows, err := utils.ReadTable[EntityMShopReplaceableGem]("m_shop_replaceable_gem")
	if err != nil {
		return nil, fmt.Errorf("load shop replaceable gem table: %w", err)
	}

	catalog := &ShopCatalog{
		Items:             make(map[int32]EntityMShopItem, len(items)),
		Contents:          make(map[int32][]EntityMShopItemContentPossession, len(contents)),
		Effects:           make(map[int32][]EntityMShopItemContentEffect, len(effects)),
		MaxStaminaMillis:  make(map[int32]int32, len(userLevels)),
		LimitedStock:      make(map[int32]ShopLimitedStockRule, len(stockRows)),
		Shops:             make(map[int32]EntityMShop),
		CellsByShop:       make(map[int32]map[int32][]ShopCell),
		UserLevelsByItem:  make(map[int32][]EntityMShopItemUserLevelCondition),
		AdditionalContent: make(map[int32][]EntityMShopItemAdditionalContent),
		ReplaceableGems:   gemRows,
	}
	for _, row := range items {
		catalog.Items[row.ShopItemId] = row
	}
	for _, row := range contents {
		catalog.Contents[row.ShopItemId] = append(catalog.Contents[row.ShopItemId], row)
	}
	for _, row := range effects {
		catalog.Effects[row.ShopItemId] = append(catalog.Effects[row.ShopItemId], row)
	}
	for _, ul := range userLevels {
		catalog.MaxStaminaMillis[ul.UserLevel] = ul.MaxStamina * 1000
	}
	for _, row := range stockRows {
		catalog.LimitedStock[row.ShopItemLimitedStockId] = ShopLimitedStockRule{
			MaxCount:        row.MaxCount,
			AutoResetType:   row.ShopItemAutoResetType,
			AutoResetPeriod: row.ShopItemAutoResetPeriod,
		}
	}
	for _, row := range levelRows {
		catalog.UserLevelsByItem[row.ShopItemId] = append(catalog.UserLevelsByItem[row.ShopItemId], row)
	}
	for shopItemId, rows := range catalog.UserLevelsByItem {
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].UserLevelLowerLimit < rows[j].UserLevelLowerLimit
		})
		for i, row := range rows {
			lower, upper := shopUserLevelBounds(row)
			if lower > upper {
				return nil, fmt.Errorf("shop item %d has invalid user level range", shopItemId)
			}
			if i > 0 {
				_, previousUpper := shopUserLevelBounds(rows[i-1])
				if lower <= previousUpper {
					return nil, fmt.Errorf("shop item %d has overlapping user level ranges", shopItemId)
				}
			}
		}
		catalog.UserLevelsByItem[shopItemId] = rows
	}
	for _, row := range additionalContentRows {
		catalog.AdditionalContent[row.ShopItemAdditionalContentId] = append(catalog.AdditionalContent[row.ShopItemAdditionalContentId], row)
	}
	for shopItemId, rows := range catalog.UserLevelsByItem {
		for _, row := range rows {
			if row.ShopItemAdditionalContentId > 0 && len(catalog.AdditionalContent[row.ShopItemAdditionalContentId]) == 0 {
				return nil, fmt.Errorf("shop item %d references missing additional content %d", shopItemId, row.ShopItemAdditionalContentId)
			}
		}
	}
	sort.Slice(catalog.ReplaceableGems, func(i, j int) bool {
		return catalog.ReplaceableGems[i].LineupUpdateCountLowerLimit < catalog.ReplaceableGems[j].LineupUpdateCountLowerLimit
	})

	shops, err := utils.ReadTable[EntityMShop]("m_shop")
	if err != nil {
		return nil, fmt.Errorf("load shop table: %w", err)
	}
	cellGroups, err := utils.ReadTable[EntityMShopItemCellGroup]("m_shop_item_cell_group")
	if err != nil {
		return nil, fmt.Errorf("load shop item cell group table: %w", err)
	}
	cells, err := utils.ReadTable[EntityMShopItemCell]("m_shop_item_cell")
	if err != nil {
		return nil, fmt.Errorf("load shop item cell table: %w", err)
	}

	cellById := make(map[int32]EntityMShopItemCell, len(cells))
	for _, c := range cells {
		cellById[c.ShopItemCellId] = c
	}
	limitedOpenRows, err := utils.ReadTable[EntityMShopItemCellLimitedOpen]("m_shop_item_cell_limited_open")
	if err != nil {
		return nil, fmt.Errorf("load shop item cell limited open table: %w", err)
	}
	limitedOpenByCellId := make(map[int32]int32, len(limitedOpenRows))
	for _, row := range limitedOpenRows {
		limitedOpenByCellId[row.ShopItemCellId] = row.LimitedOpenId
	}
	terms, err := utils.ReadTable[EntityMShopItemCellTerm]("m_shop_item_cell_term")
	if err != nil {
		return nil, fmt.Errorf("load shop item cell term table: %w", err)
	}
	termById := make(map[int32]EntityMShopItemCellTerm, len(terms))
	for _, term := range terms {
		termById[term.ShopItemCellTermId] = term
	}

	cellGroupByCGId := make(map[int32][]EntityMShopItemCellGroup, len(cellGroups))
	for _, cg := range cellGroups {
		cellGroupByCGId[cg.ShopItemCellGroupId] = append(cellGroupByCGId[cg.ShopItemCellGroupId], cg)
	}

	catalog.ExchangeShopCells = make(map[int32][]ExchangeShopCell)
	for _, s := range shops {
		catalog.Shops[s.ShopId] = s
		entries := cellGroupByCGId[s.ShopItemCellGroupId]
		if len(entries) == 0 {
			continue
		}
		catalog.CellsByShop[s.ShopId] = make(map[int32][]ShopCell)
		for _, cg := range entries {
			cell, ok := cellById[cg.ShopItemCellId]
			if !ok {
				continue
			}
			term := termById[cg.ShopItemCellTermId]
			catalog.CellsByShop[s.ShopId][cell.ShopItemId] = append(catalog.CellsByShop[s.ShopId][cell.ShopItemId], ShopCell{
				LimitedOpenId: limitedOpenByCellId[cell.ShopItemCellId],
				StartDatetime: term.StartDatetime,
				EndDatetime:   term.EndDatetime,
			})
		}

		switch s.ShopGroupType {
		case model.ShopGroupTypeItemShop:
			var poolCells []ExchangeShopCell
			for _, cg := range entries {
				if cell, ok := cellById[cg.ShopItemCellId]; ok {
					poolCells = append(poolCells, ExchangeShopCell{cg.SortOrder, cell.ShopItemId})
				}
			}
			sort.Slice(poolCells, func(i, j int) bool { return poolCells[i].SortOrder < poolCells[j].SortOrder })
			if catalog.ItemShopId == 0 {
				catalog.ItemShopId = s.ShopId
				catalog.ItemShopPool = make([]int32, len(poolCells))
				for i, pc := range poolCells {
					catalog.ItemShopPool[i] = pc.ShopItemId
				}
			}

		case model.ShopGroupTypeExchangeShop:
			var sc []ExchangeShopCell
			for _, cg := range entries {
				if cell, ok := cellById[cg.ShopItemCellId]; ok {
					sc = append(sc, ExchangeShopCell{cg.SortOrder, cell.ShopItemId})
				}
			}
			sort.Slice(sc, func(i, j int) bool { return sc[i].SortOrder < sc[j].SortOrder })
			catalog.ExchangeShopCells[s.ShopId] = sc
		}
	}

	return catalog, nil
}

func shopUserLevelBounds(row EntityMShopItemUserLevelCondition) (int64, int64) {
	lower := int64(row.UserLevelLowerLimit)
	upper := int64(row.UserLevelUpperLimit)
	if lower == 0 {
		lower = -1 << 63
	}
	if upper == 0 {
		upper = 1<<63 - 1
	}
	return lower, upper
}
