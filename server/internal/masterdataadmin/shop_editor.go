package masterdataadmin

import (
	"fmt"
	"sort"

	"lunar-tear/server/internal/masterdata/memorydb"
)

const shopItemCellGroupTable = "m_shop_item_cell_group"

type ShopEditorCatalog struct {
	Shops      []ShopEditorShop         `json:"shops"`
	CellGroups []ShopItemCellGroupInput `json:"cellGroups"`
	Cells      []ShopEditorCell         `json:"cells"`
	Items      []ShopEditorItem         `json:"items"`
}

type ShopEditorShop struct {
	ShopID              int64             `json:"shopId"`
	ShopItemCellGroupID int64             `json:"shopItemCellGroupId"`
	Names               map[string]string `json:"names,omitempty"`
}

type ShopItemCellGroupInput struct {
	ShopItemCellGroupID int32 `json:"shopItemCellGroupId"`
	ShopItemCellID      int32 `json:"shopItemCellId"`
	SortOrder           int32 `json:"sortOrder"`
	ShopItemCellTermID  int32 `json:"shopItemCellTermId"`
	StartDatetime       int64 `json:"startDatetime,omitempty"`
	EndDatetime         int64 `json:"endDatetime,omitempty"`
}

type ShopEditorCell struct {
	Row            int64 `json:"row"`
	ShopItemCellID int64 `json:"shopItemCellId"`
	StepNumber     int64 `json:"stepNumber"`
	ShopItemID     int64 `json:"shopItemId"`
}

type ShopEditorItem struct {
	Row                    int64             `json:"row"`
	ShopItemID             int64             `json:"shopItemId"`
	PriceType              int64             `json:"priceType"`
	PriceID                int64             `json:"priceId"`
	Price                  int64             `json:"price"`
	RegularPrice           int64             `json:"regularPrice"`
	ShopItemLimitedStockID int64             `json:"shopItemLimitedStockId"`
	StockMaxCount          int64             `json:"stockMaxCount,omitempty"`
	StockAutoResetType     int64             `json:"stockAutoResetType,omitempty"`
	StockAutoResetPeriod   int64             `json:"stockAutoResetPeriod,omitempty"`
	Names                  map[string]string `json:"names,omitempty"`
}

type shopEditorStock struct {
	maxCount        int64
	autoResetType   int64
	autoResetPeriod int64
}

func loadShopEditor(file *memorydb.File, resolver *titleResolver) ShopEditorCatalog {
	result := ShopEditorCatalog{}
	for _, row := range readRows(file, "m_shop") {
		shopID, shopOK := integerAt(row, 0)
		groupID, groupOK := integerAt(row, 7)
		if !shopOK || !groupOK {
			continue
		}
		result.Shops = append(result.Shops, ShopEditorShop{
			ShopID: shopID, ShopItemCellGroupID: groupID, Names: resolver.resolve("m_shop", row),
		})
	}

	terms := make(map[int64][2]int64)
	for _, row := range readRows(file, "m_shop_item_cell_term") {
		termID, idOK := integerAt(row, 0)
		start, startOK := integerAt(row, 1)
		end, endOK := integerAt(row, 2)
		if idOK && startOK && endOK {
			terms[termID] = [2]int64{start, end}
		}
	}
	for _, row := range readRows(file, shopItemCellGroupTable) {
		groupID, groupOK := integerAt(row, 0)
		cellID, cellOK := integerAt(row, 1)
		sortOrder, sortOK := integerAt(row, 2)
		termID, termOK := integerAt(row, 3)
		if !groupOK || !cellOK || !sortOK || !termOK {
			continue
		}
		term := terms[termID]
		result.CellGroups = append(result.CellGroups, ShopItemCellGroupInput{
			ShopItemCellGroupID: int32(groupID), ShopItemCellID: int32(cellID),
			SortOrder: int32(sortOrder), ShopItemCellTermID: int32(termID),
			StartDatetime: term[0], EndDatetime: term[1],
		})
	}

	for rowIndex, row := range readRows(file, "m_shop_item_cell") {
		cellID, cellOK := integerAt(row, 0)
		step, stepOK := integerAt(row, 1)
		itemID, itemOK := integerAt(row, 2)
		if cellOK && stepOK && itemOK {
			result.Cells = append(result.Cells, ShopEditorCell{
				Row: int64(rowIndex), ShopItemCellID: cellID, StepNumber: step, ShopItemID: itemID,
			})
		}
	}

	stocks := make(map[int64]shopEditorStock)
	for _, row := range readRows(file, "m_shop_item_limited_stock") {
		stockID, idOK := integerAt(row, 0)
		maxCount, maxOK := integerAt(row, 1)
		resetType, typeOK := integerAt(row, 2)
		resetPeriod, periodOK := integerAt(row, 3)
		if idOK && maxOK && typeOK && periodOK {
			stocks[stockID] = shopEditorStock{maxCount: maxCount, autoResetType: resetType, autoResetPeriod: resetPeriod}
		}
	}
	for rowIndex, row := range readRows(file, "m_shop_item") {
		itemID, itemOK := integerAt(row, 0)
		priceType, typeOK := integerAt(row, 4)
		priceID, priceIDOK := integerAt(row, 5)
		price, priceOK := integerAt(row, 6)
		regularPrice, regularOK := integerAt(row, 7)
		stockID, stockOK := integerAt(row, 9)
		if !itemOK || !typeOK || !priceIDOK || !priceOK || !regularOK || !stockOK {
			continue
		}
		stock := stocks[stockID]
		result.Items = append(result.Items, ShopEditorItem{
			Row: int64(rowIndex), ShopItemID: itemID, PriceType: priceType, PriceID: priceID,
			Price: price, RegularPrice: regularPrice, ShopItemLimitedStockID: stockID,
			StockMaxCount: stock.maxCount, StockAutoResetType: stock.autoResetType,
			StockAutoResetPeriod: stock.autoResetPeriod,
			Names:                resolver.byKey(fmt.Sprintf("shop.item.name.%d", resolver.shopItemTextIDs[itemID])),
		})
	}

	sort.SliceStable(result.Shops, func(i, j int) bool { return result.Shops[i].ShopID < result.Shops[j].ShopID })
	sort.SliceStable(result.Cells, func(i, j int) bool { return result.Cells[i].ShopItemCellID < result.Cells[j].ShopItemCellID })
	sort.SliceStable(result.Items, func(i, j int) bool { return result.Items[i].ShopItemID < result.Items[j].ShopItemID })
	return result
}

func buildShopItemCellGroupReplacement(file *memorydb.File, replacement *[]ShopItemCellGroupInput) (int, int, error) {
	if replacement == nil {
		return 0, 0, nil
	}
	current, exists, err := file.TableRows(shopItemCellGroupTable)
	if err != nil {
		return 0, 0, err
	}
	if !exists {
		return 0, 0, fmt.Errorf("table %q is absent from the current master data", shopItemCellGroupTable)
	}
	counts := make(map[[4]int64]int, len(current))
	for _, row := range current {
		key, ok := shopItemCellGroupKey(row)
		if !ok {
			return 0, 0, fmt.Errorf("table %q contains a malformed row", shopItemCellGroupTable)
		}
		counts[key]++
	}
	for _, row := range shopItemCellGroupRows(*replacement) {
		key, _ := shopItemCellGroupKey(row)
		counts[key]--
	}
	removed, added := 0, 0
	for _, difference := range counts {
		if difference > 0 {
			removed += difference
		} else {
			added -= difference
		}
	}
	changedRows := removed
	if added > changedRows {
		changedRows = added
	}
	return changedRows * 4, changedRows, nil
}

func shopItemCellGroupKey(row []interface{}) ([4]int64, bool) {
	var key [4]int64
	for column := range key {
		value, ok := integerAt(row, column)
		if !ok {
			return [4]int64{}, false
		}
		key[column] = value
	}
	return key, true
}

func shopItemCellGroupRows(rows []ShopItemCellGroupInput) [][]interface{} {
	result := make([][]interface{}, 0, len(rows))
	for _, row := range rows {
		result = append(result, []interface{}{
			row.ShopItemCellGroupID, row.ShopItemCellID, row.SortOrder, row.ShopItemCellTermID,
		})
	}
	return result
}
