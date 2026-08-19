package masterdataadmin

import (
	"fmt"
	"sort"

	"lunar-tear/server/internal/masterdata/memorydb"
)

const shopItemCellGroupTable = "m_shop_item_cell_group"
const shopItemCellTable = "m_shop_item_cell"
const shopItemTable = "m_shop_item"
const shopItemContentPossessionTable = "m_shop_item_content_possession"

var shopItemCellReferenceTables = []struct {
	name   string
	column int
}{
	{name: shopItemCellGroupTable, column: 1},
	{name: "m_shop_item_cell_limited_open", column: 0},
}

var shopItemReferenceTables = []struct {
	name   string
	column int
}{
	{name: "m_shop_item_cell", column: 2},
	{name: "m_shop_item_content_effect", column: 0},
	{name: "m_shop_item_content_mission", column: 0},
	{name: "m_shop_item_user_level_condition", column: 0},
}

type ShopEditorCatalog struct {
	Shops      []ShopEditorShop         `json:"shops"`
	CellGroups []ShopItemCellGroupInput `json:"cellGroups"`
	Cells      []ShopEditorCell         `json:"cells"`
	Items      []ShopEditorItem         `json:"items"`
	Stocks     []ShopEditorStock        `json:"stocks"`
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
	Row            int64    `json:"row"`
	ShopItemCellID int64    `json:"shopItemCellId"`
	StepNumber     int64    `json:"stepNumber"`
	ShopItemID     int64    `json:"shopItemId"`
	DeleteBlockers []string `json:"deleteBlockers,omitempty"`
}

type ShopItemCellInput struct {
	ShopItemCellID int32 `json:"shopItemCellId"`
	StepNumber     int32 `json:"stepNumber"`
	ShopItemID     int32 `json:"shopItemId"`
}

type ShopItemCellKey struct {
	ShopItemCellID int32 `json:"shopItemCellId"`
	StepNumber     int32 `json:"stepNumber"`
}

type ShopItemCellStructuralUpdate struct {
	Additions []ShopItemCellInput `json:"additions,omitempty"`
	Deletes   []ShopItemCellKey   `json:"deletes,omitempty"`
}

type ShopItemInput struct {
	ShopItemID             int32 `json:"shopItemId"`
	NameShopTextID         int32 `json:"nameShopTextId"`
	DescriptionShopTextID  int32 `json:"descriptionShopTextId"`
	ShopItemContentType    int32 `json:"shopItemContentType"`
	PriceType              int32 `json:"priceType"`
	PriceID                int32 `json:"priceId"`
	Price                  int32 `json:"price"`
	RegularPrice           int32 `json:"regularPrice"`
	ShopPromotionType      int32 `json:"shopPromotionType"`
	ShopItemLimitedStockID int32 `json:"shopItemLimitedStockId"`
	AssetCategoryID        int32 `json:"assetCategoryId"`
	AssetVariationID       int32 `json:"assetVariationId"`
	ShopItemDecorationType int32 `json:"shopItemDecorationType"`
}

type ShopItemStructuralUpdate struct {
	Copies    []ShopItemCopyInput `json:"copies,omitempty"`
	DeleteIDs []int32             `json:"deleteIds,omitempty"`
}

type ShopItemCopyInput struct {
	SourceShopItemID int32 `json:"sourceShopItemId"`
	ShopItemInput
	Possessions []ShopItemContentPossessionInput `json:"possessions"`
}

type ShopItemContentPossessionInput struct {
	ShopItemID     int32 `json:"shopItemId"`
	PossessionType int32 `json:"possessionType"`
	PossessionID   int32 `json:"possessionId"`
	SortOrder      int32 `json:"sortOrder"`
	Count          int32 `json:"count"`
}

type ShopEditorItem struct {
	Row                    int64             `json:"row"`
	ShopItemID             int64             `json:"shopItemId"`
	NameShopTextID         int64             `json:"nameShopTextId"`
	DescriptionShopTextID  int64             `json:"descriptionShopTextId"`
	ShopItemContentType    int64             `json:"shopItemContentType"`
	PriceType              int64             `json:"priceType"`
	PriceID                int64             `json:"priceId"`
	Price                  int64             `json:"price"`
	RegularPrice           int64             `json:"regularPrice"`
	ShopPromotionType      int64             `json:"shopPromotionType"`
	ShopItemLimitedStockID int64             `json:"shopItemLimitedStockId"`
	AssetCategoryID        int64             `json:"assetCategoryId"`
	AssetVariationID       int64             `json:"assetVariationId"`
	ShopItemDecorationType int64             `json:"shopItemDecorationType"`
	Names                  map[string]string `json:"names,omitempty"`
	DeleteBlockers         []string          `json:"deleteBlockers,omitempty"`
}

type ShopEditorStock struct {
	ShopItemLimitedStockID int64 `json:"shopItemLimitedStockId"`
	MaxCount               int64 `json:"maxCount"`
	AutoResetType          int64 `json:"autoResetType"`
	AutoResetPeriod        int64 `json:"autoResetPeriod"`
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

	cellDeleteBlockers := loadShopItemCellDeleteBlockerIndex(file)
	for rowIndex, row := range readRows(file, shopItemCellTable) {
		cellID, cellOK := integerAt(row, 0)
		step, stepOK := integerAt(row, 1)
		itemID, itemOK := integerAt(row, 2)
		if cellOK && stepOK && itemOK {
			result.Cells = append(result.Cells, ShopEditorCell{
				Row: int64(rowIndex), ShopItemCellID: cellID, StepNumber: step, ShopItemID: itemID,
				DeleteBlockers: cellDeleteBlockers[cellID],
			})
		}
	}

	for _, row := range readRows(file, "m_shop_item_limited_stock") {
		stockID, idOK := integerAt(row, 0)
		maxCount, maxOK := integerAt(row, 1)
		resetType, typeOK := integerAt(row, 2)
		resetPeriod, periodOK := integerAt(row, 3)
		if idOK && maxOK && typeOK && periodOK {
			stock := ShopEditorStock{
				ShopItemLimitedStockID: stockID, MaxCount: maxCount,
				AutoResetType: resetType, AutoResetPeriod: resetPeriod,
			}
			result.Stocks = append(result.Stocks, stock)
		}
	}
	deleteBlockers := loadShopItemDeleteBlockerIndex(file)
	for rowIndex, row := range readRows(file, "m_shop_item") {
		item, ok := shopItemInputAt(row)
		if !ok {
			continue
		}
		result.Items = append(result.Items, ShopEditorItem{
			Row: int64(rowIndex), ShopItemID: int64(item.ShopItemID), NameShopTextID: int64(item.NameShopTextID),
			DescriptionShopTextID: int64(item.DescriptionShopTextID), ShopItemContentType: int64(item.ShopItemContentType),
			PriceType: int64(item.PriceType), PriceID: int64(item.PriceID), Price: int64(item.Price),
			RegularPrice: int64(item.RegularPrice), ShopPromotionType: int64(item.ShopPromotionType),
			ShopItemLimitedStockID: int64(item.ShopItemLimitedStockID), AssetCategoryID: int64(item.AssetCategoryID),
			AssetVariationID: int64(item.AssetVariationID), ShopItemDecorationType: int64(item.ShopItemDecorationType),
			Names:          resolver.byKey(fmt.Sprintf("shop.item.name.%d", resolver.shopItemTextIDs[int64(item.ShopItemID)])),
			DeleteBlockers: deleteBlockers[int64(item.ShopItemID)],
		})
	}

	sort.SliceStable(result.Shops, func(i, j int) bool { return result.Shops[i].ShopID < result.Shops[j].ShopID })
	sort.SliceStable(result.Cells, func(i, j int) bool { return result.Cells[i].ShopItemCellID < result.Cells[j].ShopItemCellID })
	sort.SliceStable(result.Items, func(i, j int) bool { return result.Items[i].ShopItemID < result.Items[j].ShopItemID })
	sort.SliceStable(result.Stocks, func(i, j int) bool {
		return result.Stocks[i].ShopItemLimitedStockID < result.Stocks[j].ShopItemLimitedStockID
	})
	return result
}

func shopItemInputAt(row []interface{}) (ShopItemInput, bool) {
	values := make([]int64, 13)
	for column := range values {
		value, ok := integerAt(row, column)
		if !ok {
			return ShopItemInput{}, false
		}
		values[column] = value
	}
	return ShopItemInput{
		ShopItemID: int32(values[0]), NameShopTextID: int32(values[1]), DescriptionShopTextID: int32(values[2]),
		ShopItemContentType: int32(values[3]), PriceType: int32(values[4]), PriceID: int32(values[5]),
		Price: int32(values[6]), RegularPrice: int32(values[7]), ShopPromotionType: int32(values[8]),
		ShopItemLimitedStockID: int32(values[9]), AssetCategoryID: int32(values[10]),
		AssetVariationID: int32(values[11]), ShopItemDecorationType: int32(values[12]),
	}, true
}

func shopItemRows(rows []ShopItemInput) [][]interface{} {
	result := make([][]interface{}, 0, len(rows))
	for _, row := range rows {
		result = append(result, []interface{}{
			row.ShopItemID, row.NameShopTextID, row.DescriptionShopTextID, row.ShopItemContentType,
			row.PriceType, row.PriceID, row.Price, row.RegularPrice, row.ShopPromotionType,
			row.ShopItemLimitedStockID, row.AssetCategoryID, row.AssetVariationID, row.ShopItemDecorationType,
		})
	}
	return result
}

func shopItemContentPossessionInputAt(row []interface{}) (ShopItemContentPossessionInput, bool) {
	values := make([]int64, 5)
	for column := range values {
		value, ok := integerAt(row, column)
		if !ok {
			return ShopItemContentPossessionInput{}, false
		}
		values[column] = value
	}
	return ShopItemContentPossessionInput{
		ShopItemID: int32(values[0]), PossessionType: int32(values[1]), PossessionID: int32(values[2]),
		SortOrder: int32(values[3]), Count: int32(values[4]),
	}, true
}

func shopItemContentPossessionRows(rows []ShopItemContentPossessionInput) [][]interface{} {
	result := make([][]interface{}, 0, len(rows))
	for _, row := range rows {
		result = append(result, []interface{}{
			row.ShopItemID, row.PossessionType, row.PossessionID, row.SortOrder, row.Count,
		})
	}
	return result
}

func shopItemCellInputAt(row []interface{}) (ShopItemCellInput, bool) {
	cellID, cellOK := integerAt(row, 0)
	step, stepOK := integerAt(row, 1)
	itemID, itemOK := integerAt(row, 2)
	if !cellOK || !stepOK || !itemOK {
		return ShopItemCellInput{}, false
	}
	return ShopItemCellInput{
		ShopItemCellID: int32(cellID), StepNumber: int32(step), ShopItemID: int32(itemID),
	}, true
}

func shopItemCellRows(rows []ShopItemCellInput) [][]interface{} {
	result := make([][]interface{}, 0, len(rows))
	for _, row := range rows {
		result = append(result, []interface{}{row.ShopItemCellID, row.StepNumber, row.ShopItemID})
	}
	return result
}

func loadShopItemCellDeleteBlockerIndex(file *memorydb.File) map[int64][]string {
	result := make(map[int64][]string)
	for _, reference := range shopItemCellReferenceTables {
		seen := make(map[int64]bool)
		for _, row := range readRows(file, reference.name) {
			cellID, ok := integerAt(row, reference.column)
			if !ok || seen[cellID] {
				continue
			}
			seen[cellID] = true
			result[cellID] = append(result[cellID], reference.name)
		}
	}
	return result
}

func shopItemCellDeleteBlockers(file *memorydb.File, shopItemCellID int64) ([]string, error) {
	var result []string
	for _, reference := range shopItemCellReferenceTables {
		rows, exists, err := file.TableRows(reference.name)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		for _, row := range rows {
			cellID, ok := integerAt(row, reference.column)
			if !ok {
				return nil, fmt.Errorf("table %q contains a malformed row", reference.name)
			}
			if cellID == shopItemCellID {
				result = append(result, reference.name)
				break
			}
		}
	}
	return result, nil
}

func buildShopItemCellReplacement(file *memorydb.File, update *ShopItemCellStructuralUpdate, edits []memorydb.CellEdit) ([][]interface{}, int, int, bool, error) {
	if update == nil || len(update.Additions) == 0 && len(update.Deletes) == 0 {
		return nil, 0, 0, false, nil
	}
	current, exists, err := file.TableRows(shopItemCellTable)
	if err != nil {
		return nil, 0, 0, false, err
	}
	if !exists {
		return nil, 0, 0, false, fmt.Errorf("table %q is absent from the current master data", shopItemCellTable)
	}
	rows := make([][]interface{}, len(current))
	existingKeys := make(map[[2]int64]bool, len(current))
	for index, row := range current {
		cell, ok := shopItemCellInputAt(row)
		if !ok {
			return nil, 0, 0, false, fmt.Errorf("table %q contains a malformed row", shopItemCellTable)
		}
		key := [2]int64{int64(cell.ShopItemCellID), int64(cell.StepNumber)}
		if existingKeys[key] {
			return nil, 0, 0, false, fmt.Errorf("table %q contains duplicate key %v", shopItemCellTable, key)
		}
		existingKeys[key] = true
		rows[index] = append([]interface{}(nil), row...)
	}
	for _, edit := range edits {
		if edit.Table == shopItemCellTable {
			rows[edit.Row][edit.Column] = edit.Value
		}
	}

	deleteKeys := make(map[[2]int64]bool, len(update.Deletes))
	for _, deleted := range update.Deletes {
		key := [2]int64{int64(deleted.ShopItemCellID), int64(deleted.StepNumber)}
		if deleteKeys[key] {
			return nil, 0, 0, false, fmt.Errorf("duplicate Cell key %v in deletes", key)
		}
		if !existingKeys[key] {
			return nil, 0, 0, false, fmt.Errorf("Cell %v does not exist", key)
		}
		blockers, blockerErr := shopItemCellDeleteBlockers(file, key[0])
		if blockerErr != nil {
			return nil, 0, 0, false, blockerErr
		}
		if len(blockers) != 0 {
			return nil, 0, 0, false, fmt.Errorf("CellId %d is still referenced by %v", key[0], blockers)
		}
		deleteKeys[key] = true
	}

	additionKeys := make(map[[2]int64]bool, len(update.Additions))
	for _, added := range update.Additions {
		key := [2]int64{int64(added.ShopItemCellID), int64(added.StepNumber)}
		if added.ShopItemCellID <= 0 || added.StepNumber <= 0 {
			return nil, 0, 0, false, fmt.Errorf("CellId and StepNumber must be positive")
		}
		if existingKeys[key] || additionKeys[key] {
			return nil, 0, 0, false, fmt.Errorf("Cell %v already exists", key)
		}
		additionKeys[key] = true
	}

	replacement := make([][]interface{}, 0, len(rows)-len(deleteKeys)+len(update.Additions))
	for _, row := range rows {
		cell, _ := shopItemCellInputAt(row)
		key := [2]int64{int64(cell.ShopItemCellID), int64(cell.StepNumber)}
		if !deleteKeys[key] {
			replacement = append(replacement, row)
		}
	}
	replacement = append(replacement, shopItemCellRows(update.Additions)...)
	sort.SliceStable(replacement, func(i, j int) bool {
		left, _ := shopItemCellInputAt(replacement[i])
		right, _ := shopItemCellInputAt(replacement[j])
		if left.ShopItemCellID != right.ShopItemCellID {
			return left.ShopItemCellID < right.ShopItemCellID
		}
		return left.StepNumber < right.StepNumber
	})
	changedRows := len(update.Additions) + len(update.Deletes)
	return replacement, changedRows * 3, changedRows, true, nil
}

func loadShopItemDeleteBlockerIndex(file *memorydb.File) map[int64][]string {
	result := make(map[int64][]string)
	for _, reference := range shopItemReferenceTables {
		seen := make(map[int64]bool)
		for _, row := range readRows(file, reference.name) {
			itemID, ok := integerAt(row, reference.column)
			if !ok || seen[itemID] {
				continue
			}
			seen[itemID] = true
			result[itemID] = append(result[itemID], reference.name)
		}
	}
	return result
}

func shopItemDeleteBlockers(file *memorydb.File, shopItemID int64) ([]string, error) {
	var result []string
	for _, reference := range shopItemReferenceTables {
		rows, exists, err := file.TableRows(reference.name)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		for _, row := range rows {
			itemID, ok := integerAt(row, reference.column)
			if !ok {
				return nil, fmt.Errorf("table %q contains a malformed row", reference.name)
			}
			if itemID == shopItemID {
				result = append(result, reference.name)
				break
			}
		}
	}
	return result, nil
}

func buildShopItemReplacement(file *memorydb.File, update *ShopItemStructuralUpdate, edits []memorydb.CellEdit) ([][]interface{}, int, int, bool, error) {
	if update == nil || (len(update.Copies) == 0 && len(update.DeleteIDs) == 0) {
		return nil, 0, 0, false, nil
	}
	current, exists, err := file.TableRows(shopItemTable)
	if err != nil {
		return nil, 0, 0, false, err
	}
	if !exists {
		return nil, 0, 0, false, fmt.Errorf("table %q is absent from the current master data", shopItemTable)
	}
	rows := make([][]interface{}, len(current))
	existingIDs := make(map[int64]bool, len(current))
	existingItems := make(map[int64]ShopItemInput, len(current))
	for index, row := range current {
		item, ok := shopItemInputAt(row)
		if !ok {
			return nil, 0, 0, false, fmt.Errorf("table %q contains a malformed row", shopItemTable)
		}
		if existingIDs[int64(item.ShopItemID)] {
			return nil, 0, 0, false, fmt.Errorf("table %q contains duplicate ShopItemId %d", shopItemTable, item.ShopItemID)
		}
		existingIDs[int64(item.ShopItemID)] = true
		existingItems[int64(item.ShopItemID)] = item
		rows[index] = append([]interface{}(nil), row...)
	}
	for _, edit := range edits {
		if edit.Table == shopItemTable {
			rows[edit.Row][edit.Column] = edit.Value
		}
	}

	deleteIDs := make(map[int64]bool, len(update.DeleteIDs))
	for _, value := range update.DeleteIDs {
		itemID := int64(value)
		if deleteIDs[itemID] {
			return nil, 0, 0, false, fmt.Errorf("duplicate ShopItemId %d in deleteIds", itemID)
		}
		if !existingIDs[itemID] {
			return nil, 0, 0, false, fmt.Errorf("ShopItemId %d does not exist", itemID)
		}
		blockers, blockerErr := shopItemDeleteBlockers(file, itemID)
		if blockerErr != nil {
			return nil, 0, 0, false, blockerErr
		}
		if len(blockers) != 0 {
			return nil, 0, 0, false, fmt.Errorf("ShopItemId %d is still referenced by %v", itemID, blockers)
		}
		deleteIDs[itemID] = true
	}

	copyIDs := make(map[int64]bool, len(update.Copies))
	for _, copied := range update.Copies {
		itemID := int64(copied.ShopItemID)
		if itemID <= 0 {
			return nil, 0, 0, false, fmt.Errorf("copied ShopItemId must be positive")
		}
		if existingIDs[itemID] || copyIDs[itemID] {
			return nil, 0, 0, false, fmt.Errorf("ShopItemId %d already exists", itemID)
		}
		source, sourceExists := existingItems[int64(copied.SourceShopItemID)]
		if !sourceExists {
			return nil, 0, 0, false, fmt.Errorf("source ShopItemId %d does not exist", copied.SourceShopItemID)
		}
		if !shopItemReadOnlyFieldsEqual(source, copied.ShopItemInput) {
			return nil, 0, 0, false, fmt.Errorf("copied ShopItemId %d changes fields outside the ShopItem editor", itemID)
		}
		copyIDs[itemID] = true
	}

	replacement := make([][]interface{}, 0, len(rows)-len(deleteIDs)+len(update.Copies))
	for _, row := range rows {
		itemID, _ := integerAt(row, 0)
		if !deleteIDs[itemID] {
			replacement = append(replacement, row)
		}
	}
	copies := make([]ShopItemInput, 0, len(update.Copies))
	for _, copied := range update.Copies {
		copies = append(copies, copied.ShopItemInput)
	}
	replacement = append(replacement, shopItemRows(copies)...)
	sort.SliceStable(replacement, func(i, j int) bool {
		left, _ := integerAt(replacement[i], 0)
		right, _ := integerAt(replacement[j], 0)
		return left < right
	})
	changedRows := len(update.Copies) + len(update.DeleteIDs)
	return replacement, changedRows * 13, changedRows, true, nil
}

func shopItemReadOnlyFieldsEqual(left, right ShopItemInput) bool {
	return left.NameShopTextID == right.NameShopTextID &&
		left.DescriptionShopTextID == right.DescriptionShopTextID &&
		left.ShopItemContentType == right.ShopItemContentType &&
		left.ShopPromotionType == right.ShopPromotionType &&
		left.AssetCategoryID == right.AssetCategoryID &&
		left.AssetVariationID == right.AssetVariationID &&
		left.ShopItemDecorationType == right.ShopItemDecorationType
}

func buildShopItemContentPossessionReplacement(file *memorydb.File, update *ShopItemStructuralUpdate, edits []memorydb.CellEdit) ([][]interface{}, int, int, bool, error) {
	if update == nil || len(update.Copies) == 0 && len(update.DeleteIDs) == 0 {
		return nil, 0, 0, false, nil
	}
	current, exists, err := file.TableRows(shopItemContentPossessionTable)
	if err != nil {
		return nil, 0, 0, false, err
	}
	if !exists {
		return nil, 0, 0, false, fmt.Errorf("table %q is absent from the current master data", shopItemContentPossessionTable)
	}
	rows := make([][]interface{}, len(current))
	byShopItemID := make(map[int64][]ShopItemContentPossessionInput)
	for index, row := range current {
		content, ok := shopItemContentPossessionInputAt(row)
		if !ok {
			return nil, 0, 0, false, fmt.Errorf("table %q contains a malformed row", shopItemContentPossessionTable)
		}
		byShopItemID[int64(content.ShopItemID)] = append(byShopItemID[int64(content.ShopItemID)], content)
		rows[index] = append([]interface{}(nil), row...)
	}
	for _, edit := range edits {
		if edit.Table == shopItemContentPossessionTable {
			rows[edit.Row][edit.Column] = edit.Value
		}
	}

	var added []ShopItemContentPossessionInput
	for _, copied := range update.Copies {
		source := byShopItemID[int64(copied.SourceShopItemID)]
		if len(copied.Possessions) != len(source) {
			return nil, 0, 0, false, fmt.Errorf("copied ShopItemId %d must retain all Possession rows", copied.ShopItemID)
		}
		for index, possession := range copied.Possessions {
			if possession.ShopItemID != copied.ShopItemID || possession.SortOrder != source[index].SortOrder {
				return nil, 0, 0, false, fmt.Errorf("copied ShopItemId %d changes the Possession row structure", copied.ShopItemID)
			}
			added = append(added, possession)
		}
	}
	deleteIDs := make(map[int64]bool, len(update.DeleteIDs))
	for _, shopItemID := range update.DeleteIDs {
		deleteIDs[int64(shopItemID)] = true
	}
	replacement := make([][]interface{}, 0, len(rows)+len(added))
	removed := 0
	for _, row := range rows {
		shopItemID, _ := integerAt(row, 0)
		if deleteIDs[shopItemID] {
			removed++
			continue
		}
		replacement = append(replacement, row)
	}
	changedRows := removed + len(added)
	if changedRows == 0 {
		return nil, 0, 0, false, nil
	}
	replacement = append(replacement, shopItemContentPossessionRows(added)...)
	sort.SliceStable(replacement, func(i, j int) bool {
		left, _ := shopItemContentPossessionInputAt(replacement[i])
		right, _ := shopItemContentPossessionInputAt(replacement[j])
		if left.ShopItemID != right.ShopItemID {
			return left.ShopItemID < right.ShopItemID
		}
		return left.SortOrder < right.SortOrder
	})
	return replacement, changedRows * 5, changedRows, true, nil
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
	ordered := append([]ShopItemCellGroupInput(nil), rows...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].ShopItemCellGroupID != ordered[j].ShopItemCellGroupID {
			return ordered[i].ShopItemCellGroupID < ordered[j].ShopItemCellGroupID
		}
		return ordered[i].ShopItemCellID < ordered[j].ShopItemCellID
	})
	result := make([][]interface{}, 0, len(ordered))
	for _, row := range ordered {
		result = append(result, []interface{}{
			row.ShopItemCellGroupID, row.ShopItemCellID, row.SortOrder, row.ShopItemCellTermID,
		})
	}
	return result
}
