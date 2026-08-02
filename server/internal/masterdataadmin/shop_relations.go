package masterdataadmin

import (
	"fmt"
	"sort"

	"lunar-tear/server/internal/masterdata/memorydb"
)

type ShopRelation struct {
	ShopID              int64             `json:"shopId"`
	ShopItemCellGroupID int64             `json:"shopItemCellGroupId"`
	ShopItemCellIDs     []int64           `json:"shopItemCellIds"`
	ShopTitles          map[string]string `json:"shopTitles,omitempty"`
}

type shopReference struct {
	shopID      int64
	cellGroupID int64
	titleTextID int64
}

func (r *titleResolver) loadShopRelations(file *memorydb.File) {
	shopsByGroup := make(map[int64][]shopReference)
	for _, row := range readRows(file, "m_shop") {
		shopID, shopOK := integerAt(row, 0)
		titleTextID, titleOK := integerAt(row, 4)
		cellGroupID, groupOK := integerAt(row, 7)
		if !shopOK || !titleOK || !groupOK || shopID == 0 || cellGroupID == 0 {
			continue
		}
		shopsByGroup[cellGroupID] = append(shopsByGroup[cellGroupID], shopReference{
			shopID: shopID, cellGroupID: cellGroupID, titleTextID: titleTextID,
		})
	}

	byShop := make(map[int64]map[int64]*ShopRelation)
	byTerm := make(map[int64]map[int64]*ShopRelation)
	for _, row := range readRows(file, "m_shop_item_cell_group") {
		cellGroupID, groupOK := integerAt(row, 0)
		cellID, cellOK := integerAt(row, 1)
		termID, termOK := integerAt(row, 3)
		if !groupOK || !cellOK || cellID == 0 {
			continue
		}
		for _, shop := range shopsByGroup[cellGroupID] {
			r.addShopRelation(byShop, shop.shopID, shop, cellID)
			if termOK && termID != 0 {
				r.addShopRelation(byTerm, termID, shop, cellID)
			}
		}
	}
	r.shopRelationsByShop = freezeShopRelations(byShop)
	r.shopRelationsByTerm = freezeShopRelations(byTerm)
}

func (r *titleResolver) addShopRelation(target map[int64]map[int64]*ShopRelation, ownerID int64, shop shopReference, cellID int64) {
	if target[ownerID] == nil {
		target[ownerID] = make(map[int64]*ShopRelation)
	}
	relation := target[ownerID][shop.shopID]
	if relation == nil {
		relation = &ShopRelation{
			ShopID:              shop.shopID,
			ShopItemCellGroupID: shop.cellGroupID,
			ShopTitles:          r.byKey(fmt.Sprintf("shop.name.%d", shop.titleTextID)),
		}
		target[ownerID][shop.shopID] = relation
	}
	relation.ShopItemCellIDs = append(relation.ShopItemCellIDs, cellID)
}

func freezeShopRelations(source map[int64]map[int64]*ShopRelation) map[int64][]ShopRelation {
	result := make(map[int64][]ShopRelation, len(source))
	for ownerID, relationsByShop := range source {
		shopIDs := make([]int64, 0, len(relationsByShop))
		for shopID := range relationsByShop {
			shopIDs = append(shopIDs, shopID)
		}
		sort.Slice(shopIDs, func(i, j int) bool { return shopIDs[i] < shopIDs[j] })
		for _, shopID := range shopIDs {
			relation := relationsByShop[shopID]
			sort.Slice(relation.ShopItemCellIDs, func(i, j int) bool {
				return relation.ShopItemCellIDs[i] < relation.ShopItemCellIDs[j]
			})
			relation.ShopItemCellIDs = compactInt64s(relation.ShopItemCellIDs)
			result[ownerID] = append(result[ownerID], *relation)
		}
	}
	return result
}

func compactInt64s(values []int64) []int64 {
	if len(values) < 2 {
		return values
	}
	write := 1
	for read := 1; read < len(values); read++ {
		if values[read] != values[write-1] {
			values[write] = values[read]
			write++
		}
	}
	return values[:write]
}

func (r *titleResolver) resolveShopRelations(table string, row []interface{}) []ShopRelation {
	var relations []ShopRelation
	switch table {
	case "m_shop":
		if shopID, ok := integerAt(row, 0); ok {
			relations = r.shopRelationsByShop[shopID]
		}
	case "m_shop_item_cell_term":
		if termID, ok := integerAt(row, 0); ok {
			relations = r.shopRelationsByTerm[termID]
		}
	}
	return cloneShopRelations(relations)
}

func cloneShopRelations(source []ShopRelation) []ShopRelation {
	if len(source) == 0 {
		return nil
	}
	result := make([]ShopRelation, len(source))
	for index, relation := range source {
		result[index] = relation
		result[index].ShopItemCellIDs = append([]int64(nil), relation.ShopItemCellIDs...)
		result[index].ShopTitles = cloneTitles(relation.ShopTitles)
	}
	return result
}
