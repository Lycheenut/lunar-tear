package masterdata

import (
	"fmt"

	"lunar-tear/server/internal/utils"
)

type ConsumableItemCatalog struct {
	All            map[int32]EntityMConsumableItem
	EffectByItemId map[int32]EntityMConsumableItemEffect
}

func LoadConsumableItemCatalog() (*ConsumableItemCatalog, error) {
	rows, err := utils.ReadTable[EntityMConsumableItem]("m_consumable_item")
	if err != nil {
		return nil, fmt.Errorf("load consumable item table: %w", err)
	}
	effects, err := utils.ReadTable[EntityMConsumableItemEffect]("m_consumable_item_effect")
	if err != nil {
		return nil, fmt.Errorf("load consumable item effect table: %w", err)
	}

	catalog := &ConsumableItemCatalog{
		All:            make(map[int32]EntityMConsumableItem, len(rows)),
		EffectByItemId: make(map[int32]EntityMConsumableItemEffect, len(effects)),
	}
	for _, row := range rows {
		catalog.All[row.ConsumableItemId] = row
	}
	for _, e := range effects {
		catalog.EffectByItemId[e.ConsumableItemId] = e
	}
	return catalog, nil
}
