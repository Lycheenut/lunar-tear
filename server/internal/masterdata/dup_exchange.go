package masterdata

import (
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/utils"
)

func LoadDupExchange() (map[int32][]model.DupExchangeEntry, error) {
	result := make(map[int32][]model.DupExchangeEntry)

	costumeRows, err := utils.ReadTable[EntityMCostumeDuplicationExchangePossessionGroup]("m_costume_duplication_exchange_possession_group")
	if err != nil {
		return nil, err
	}
	for _, r := range costumeRows {
		result[r.CostumeId] = append(result[r.CostumeId], model.DupExchangeEntry{
			PossessionType: r.PossessionType,
			PossessionId:   r.PossessionId,
			Count:          r.Count,
		})
	}

	return result, nil
}

func LoadCompanionDupExchange() (map[int32][]model.DupExchangeEntry, error) {
	rows, err := utils.ReadTable[EntityMCompanionDuplicationExchangePossessionGroup]("m_companion_duplication_exchange_possession_group")
	if err != nil {
		return nil, err
	}

	result := make(map[int32][]model.DupExchangeEntry)
	for _, row := range rows {
		result[row.CompanionId] = append(result[row.CompanionId], model.DupExchangeEntry{
			PossessionType: row.PossessionType,
			PossessionId:   row.PossessionId,
			Count:          row.Count,
		})
	}
	return result, nil
}

const dupExchangeFallbackCount int32 = 10

func EnrichDupExchange(dupMap map[int32][]model.DupExchangeEntry, pool *GachaCatalog) (int, error) {
	lbRows, err := utils.ReadTable[EntityMCostumeLimitBreakMaterialGroup]("m_costume_limit_break_material_group")
	if err != nil {
		return 0, err
	}
	groupToMaterial := make(map[int32]int32, len(lbRows))
	for _, r := range lbRows {
		groupToMaterial[r.CostumeLimitBreakMaterialGroupId] = r.MaterialId
	}

	costumeRows, err := utils.ReadTable[EntityMCostume]("m_costume")
	if err != nil {
		return 0, err
	}
	costumeLBGroup := make(map[int32]int32, len(costumeRows))
	for _, r := range costumeRows {
		costumeLBGroup[r.CostumeId] = r.CostumeLimitBreakMaterialGroupId
	}

	added := 0
	for costumeId := range pool.CostumeById {
		if _, exists := dupMap[costumeId]; exists {
			continue
		}
		matId := groupToMaterial[costumeLBGroup[costumeId]]
		if matId == 0 {
			continue
		}
		dupMap[costumeId] = []model.DupExchangeEntry{{
			PossessionType: int32(model.PossessionTypeMaterial),
			PossessionId:   matId,
			Count:          dupExchangeFallbackCount,
		}}
		added++
	}
	return added, nil
}
