package masterdata

import (
	"fmt"

	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

type CharacterBoardCatalog struct {
	PanelById               map[int32]EntityMCharacterBoardPanel
	PanelsByBoardId         map[int32][]EntityMCharacterBoardPanel
	ReleaseCostsByGroupId   map[int32][]EntityMCharacterBoardPanelReleasePossessionGroup
	ReleaseEffectsByGroupId map[int32][]EntityMCharacterBoardPanelReleaseEffectGroup
	StatusUpById            map[int32]EntityMCharacterBoardStatusUp
	AbilityById             map[int32]EntityMCharacterBoardAbility
	AbilityMaxLevel         map[store.CharacterBoardAbilityKey]int32
	EffectTargetsByGroupId  map[int32][]EntityMCharacterBoardEffectTargetGroup
	BoardById               map[int32]EntityMCharacterBoard
	CharacterIdByBoardId    map[int32]int32
	MissionOptionByBoardId  map[int32]int32
}

func LoadCharacterBoardCatalog() (*CharacterBoardCatalog, error) {
	panels, err := utils.ReadTable[EntityMCharacterBoardPanel]("m_character_board_panel")
	if err != nil {
		return nil, fmt.Errorf("load character board panel table: %w", err)
	}

	costs, err := utils.ReadTable[EntityMCharacterBoardPanelReleasePossessionGroup]("m_character_board_panel_release_possession_group")
	if err != nil {
		return nil, fmt.Errorf("load character board release possession table: %w", err)
	}

	effects, err := utils.ReadTable[EntityMCharacterBoardPanelReleaseEffectGroup]("m_character_board_panel_release_effect_group")
	if err != nil {
		return nil, fmt.Errorf("load character board release effect table: %w", err)
	}

	boards, err := utils.ReadTable[EntityMCharacterBoard]("m_character_board")
	if err != nil {
		return nil, fmt.Errorf("load character board table: %w", err)
	}
	assignments, err := utils.ReadTable[EntityMCharacterBoardAssignment]("m_character_board_assignment")
	if err != nil {
		return nil, fmt.Errorf("load character board assignment table: %w", err)
	}
	groups, err := utils.ReadTable[EntityMCharacterBoardGroup]("m_character_board_group")
	if err != nil {
		return nil, fmt.Errorf("load character board group table: %w", err)
	}

	statusUps, err := utils.ReadTable[EntityMCharacterBoardStatusUp]("m_character_board_status_up")
	if err != nil {
		return nil, fmt.Errorf("load character board status up table: %w", err)
	}

	abilities, err := utils.ReadTable[EntityMCharacterBoardAbility]("m_character_board_ability")
	if err != nil {
		return nil, fmt.Errorf("load character board ability table: %w", err)
	}

	abilityMaxLevels, err := utils.ReadTable[EntityMCharacterBoardAbilityMaxLevel]("m_character_board_ability_max_level")
	if err != nil {
		return nil, fmt.Errorf("load character board ability max level table: %w", err)
	}

	targets, err := utils.ReadTable[EntityMCharacterBoardEffectTargetGroup]("m_character_board_effect_target_group")
	if err != nil {
		return nil, fmt.Errorf("load character board effect target table: %w", err)
	}
	catalog := &CharacterBoardCatalog{
		PanelById:               make(map[int32]EntityMCharacterBoardPanel, len(panels)),
		PanelsByBoardId:         make(map[int32][]EntityMCharacterBoardPanel),
		ReleaseCostsByGroupId:   make(map[int32][]EntityMCharacterBoardPanelReleasePossessionGroup),
		ReleaseEffectsByGroupId: make(map[int32][]EntityMCharacterBoardPanelReleaseEffectGroup),
		StatusUpById:            make(map[int32]EntityMCharacterBoardStatusUp, len(statusUps)),
		AbilityById:             make(map[int32]EntityMCharacterBoardAbility, len(abilities)),
		AbilityMaxLevel:         make(map[store.CharacterBoardAbilityKey]int32, len(abilityMaxLevels)),
		EffectTargetsByGroupId:  make(map[int32][]EntityMCharacterBoardEffectTargetGroup),
		BoardById:               make(map[int32]EntityMCharacterBoard, len(boards)),
		CharacterIdByBoardId:    make(map[int32]int32, len(boards)),
		MissionOptionByBoardId:  make(map[int32]int32, len(boards)),
	}

	for _, p := range panels {
		catalog.PanelById[p.CharacterBoardPanelId] = p
		catalog.PanelsByBoardId[p.CharacterBoardId] = append(catalog.PanelsByBoardId[p.CharacterBoardId], p)
	}
	for _, c := range costs {
		catalog.ReleaseCostsByGroupId[c.CharacterBoardPanelReleasePossessionGroupId] = append(
			catalog.ReleaseCostsByGroupId[c.CharacterBoardPanelReleasePossessionGroupId], c)
	}
	for _, e := range effects {
		catalog.ReleaseEffectsByGroupId[e.CharacterBoardPanelReleaseEffectGroupId] = append(
			catalog.ReleaseEffectsByGroupId[e.CharacterBoardPanelReleaseEffectGroupId], e)
	}
	for _, b := range boards {
		catalog.BoardById[b.CharacterBoardId] = b
	}
	characterByCategoryId := make(map[int32]int32, len(assignments))
	missionOptionBaseByCategoryId := make(map[int32]int32, len(assignments))
	for _, assignment := range assignments {
		if existing := characterByCategoryId[assignment.CharacterBoardCategoryId]; existing != 0 && existing != assignment.CharacterId {
			return nil, fmt.Errorf("character board category %d has multiple character assignments", assignment.CharacterBoardCategoryId)
		}
		characterByCategoryId[assignment.CharacterBoardCategoryId] = assignment.CharacterId
		if assignment.CharacterBoardAssignmentType == 1 && assignment.SortOrder > 0 {
			missionOptionBaseByCategoryId[assignment.CharacterBoardCategoryId] = 310001 + (assignment.SortOrder-1)*2
		}
	}
	categoryByGroupId := make(map[int32]int32, len(groups))
	groupTypeByGroupId := make(map[int32]int32, len(groups))
	for _, group := range groups {
		categoryByGroupId[group.CharacterBoardGroupId] = group.CharacterBoardCategoryId
		groupTypeByGroupId[group.CharacterBoardGroupId] = group.CharacterBoardGroupType
	}
	for _, board := range boards {
		categoryId := categoryByGroupId[board.CharacterBoardGroupId]
		characterId := characterByCategoryId[categoryId]
		if characterId == 0 {
			return nil, fmt.Errorf("character board %d has no character assignment", board.CharacterBoardId)
		}
		catalog.CharacterIdByBoardId[board.CharacterBoardId] = characterId
		if base := missionOptionBaseByCategoryId[categoryId]; base != 0 {
			catalog.MissionOptionByBoardId[board.CharacterBoardId] = base + groupTypeByGroupId[board.CharacterBoardGroupId] - 1
		}
	}
	for _, s := range statusUps {
		catalog.StatusUpById[s.CharacterBoardStatusUpId] = s
	}
	for _, a := range abilities {
		catalog.AbilityById[a.CharacterBoardAbilityId] = a
	}
	for _, m := range abilityMaxLevels {
		catalog.AbilityMaxLevel[store.CharacterBoardAbilityKey{
			CharacterId: m.CharacterId,
			AbilityId:   m.AbilityId,
		}] = m.MaxLevel
	}
	for _, t := range targets {
		catalog.EffectTargetsByGroupId[t.CharacterBoardEffectTargetGroupId] = append(
			catalog.EffectTargetsByGroupId[t.CharacterBoardEffectTargetGroupId], t)
	}
	return catalog, nil
}

func IsCharacterBoardPanelReleased(board store.CharacterBoardState, sortOrder int32) bool {
	if sortOrder <= 0 || sortOrder > 128 {
		return false
	}
	field := (sortOrder - 1) / 32
	mask := int32(1 << uint((sortOrder-1)%32))
	switch field {
	case 0:
		return board.PanelReleaseBit1&mask != 0
	case 1:
		return board.PanelReleaseBit2&mask != 0
	case 2:
		return board.PanelReleaseBit3&mask != 0
	case 3:
		return board.PanelReleaseBit4&mask != 0
	}
	return false
}
