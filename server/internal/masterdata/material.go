package masterdata

import (
	"fmt"
	"log"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/utils"
)

func LoadParameterMap() ([]EntityMNumericalParameterMap, error) {
	rows, err := utils.ReadTable[EntityMNumericalParameterMap]("m_numerical_parameter_map")
	if err != nil {
		return nil, fmt.Errorf("load numerical parameter map table: %w", err)
	}
	return rows, nil
}

func BuildExpThresholds(paramMapRows []EntityMNumericalParameterMap, mapId int32) []int32 {
	maxKey := int32(0)
	for _, r := range paramMapRows {
		if r.NumericalParameterMapId == mapId && r.ParameterKey > maxKey {
			maxKey = r.ParameterKey
		}
	}
	thresholds := make([]int32, maxKey+1)
	for _, r := range paramMapRows {
		if r.NumericalParameterMapId == mapId {
			thresholds[r.ParameterKey] = r.ParameterValue
		}
	}
	return thresholds
}

type MaterialCatalog struct {
	All            map[int32]EntityMMaterial
	ByType         map[model.MaterialType]map[int32]EntityMMaterial
	SaleObtainById map[int32]EntityMMaterialSaleObtainPossession
}

func LoadMaterialCatalog() (*MaterialCatalog, error) {
	rows, err := utils.ReadTable[EntityMMaterial]("m_material")
	if err != nil {
		return nil, fmt.Errorf("load material table: %w", err)
	}

	catalog := &MaterialCatalog{
		All:            make(map[int32]EntityMMaterial, len(rows)),
		ByType:         make(map[model.MaterialType]map[int32]EntityMMaterial),
		SaleObtainById: make(map[int32]EntityMMaterialSaleObtainPossession),
	}
	for _, row := range rows {
		catalog.All[row.MaterialId] = row
		mt := model.MaterialType(row.MaterialType)
		if catalog.ByType[mt] == nil {
			catalog.ByType[mt] = make(map[int32]EntityMMaterial)
		}
		catalog.ByType[mt][row.MaterialId] = row
	}

	saleRows, err := utils.ReadTable[EntityMMaterialSaleObtainPossession]("m_material_sale_obtain_possession")
	if err != nil {
		log.Printf("material catalog: sale-obtain table unavailable, side rewards on sell will be skipped: %v", err)
	} else {
		for _, row := range saleRows {
			catalog.SaleObtainById[row.MaterialSaleObtainPossessionId] = row
		}
	}

	return catalog, nil
}
