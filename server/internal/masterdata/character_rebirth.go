package masterdata

import (
	"fmt"

	"lunar-tear/server/internal/utils"
)

type StepKey struct {
	GroupId            int32
	BeforeRebirthCount int32
}

type CharacterRebirthCatalog struct {
	StepGroupByCharacterId map[int32]int32
	CharacterIdBySortOrder map[int32]int32
	StepByGroupAndCount    map[StepKey]EntityMCharacterRebirthStepGroup
	MaterialsByGroupId     map[int32][]EntityMCharacterRebirthMaterialGroup
}

func (c *CharacterRebirthCatalog) CostumeLevelLimitUp(characterId, rebirthCount int32) int32 {
	if c == nil || rebirthCount <= 0 {
		return 0
	}
	stepGroupId, ok := c.StepGroupByCharacterId[characterId]
	if !ok {
		return 0
	}
	var total int32
	for i := range rebirthCount {
		step, ok := c.StepByGroupAndCount[StepKey{GroupId: stepGroupId, BeforeRebirthCount: i}]
		if !ok {
			continue
		}
		total += step.CostumeLevelLimitUp
	}
	return total
}

func LoadCharacterRebirthCatalog() (*CharacterRebirthCatalog, error) {
	rebirthRows, err := utils.ReadTable[EntityMCharacterRebirth]("m_character_rebirth")
	if err != nil {
		return nil, fmt.Errorf("load character rebirth table: %w", err)
	}

	stepRows, err := utils.ReadTable[EntityMCharacterRebirthStepGroup]("m_character_rebirth_step_group")
	if err != nil {
		return nil, fmt.Errorf("load character rebirth step group table: %w", err)
	}

	materialRows, err := utils.ReadTable[EntityMCharacterRebirthMaterialGroup]("m_character_rebirth_material_group")
	if err != nil {
		return nil, fmt.Errorf("load character rebirth material group table: %w", err)
	}

	stepGroupByCharacterId := make(map[int32]int32, len(rebirthRows))
	characterIdBySortOrder := make(map[int32]int32, len(rebirthRows))
	for _, r := range rebirthRows {
		stepGroupByCharacterId[r.CharacterId] = r.CharacterRebirthStepGroupId
		characterIdBySortOrder[r.SortOrder] = r.CharacterId
	}

	stepByGroupAndCount := make(map[StepKey]EntityMCharacterRebirthStepGroup, len(stepRows))
	for _, s := range stepRows {
		stepByGroupAndCount[StepKey{GroupId: s.CharacterRebirthStepGroupId, BeforeRebirthCount: s.BeforeRebirthCount}] = s
	}

	materialsByGroupId := make(map[int32][]EntityMCharacterRebirthMaterialGroup)
	for _, m := range materialRows {
		materialsByGroupId[m.CharacterRebirthMaterialGroupId] = append(materialsByGroupId[m.CharacterRebirthMaterialGroupId], m)
	}

	return &CharacterRebirthCatalog{
		StepGroupByCharacterId: stepGroupByCharacterId,
		CharacterIdBySortOrder: characterIdBySortOrder,
		StepByGroupAndCount:    stepByGroupAndCount,
		MaterialsByGroupId:     materialsByGroupId,
	}, nil
}
