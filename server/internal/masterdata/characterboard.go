package masterdata

import (
	"fmt"

	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

type CharacterBoardAssignmentRow struct {
	CharacterId                  int32 `json:"CharacterId"`
	CharacterBoardCategoryId     int32 `json:"CharacterBoardCategoryId"`
	SortOrder                    int32 `json:"SortOrder"`
	CharacterBoardAssignmentType int32 `json:"CharacterBoardAssignmentType"`
}

type CharacterBoardGroupRow struct {
	CharacterBoardGroupId    int32 `json:"CharacterBoardGroupId"`
	CharacterBoardCategoryId int32 `json:"CharacterBoardCategoryId"`
	SortOrder                int32 `json:"SortOrder"`
	CharacterBoardGroupType  int32 `json:"CharacterBoardGroupType"`
	TextAssetId              int32 `json:"TextAssetId"`
}

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
