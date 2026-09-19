package main

import (
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

// loadGranter loads only the tables needed by the login-unlock bundle. It does
// not initialize the server runtime or its protobuf-dependent client projections.
func loadGranter(masterPath string) (*store.PossessionGranter, error) {
	if err := memorydb.Init(masterPath); err != nil {
		return nil, err
	}
	costumes, err := memorydb.ReadTable[masterdata.EntityMCostume]("m_costume")
	if err != nil {
		return nil, err
	}
	weapons, err := memorydb.ReadTable[masterdata.EntityMWeapon]("m_weapon")
	if err != nil {
		return nil, err
	}
	skills, err := memorydb.ReadTable[masterdata.EntityMWeaponSkillGroup]("m_weapon_skill_group")
	if err != nil {
		return nil, err
	}
	abilities, err := memorydb.ReadTable[masterdata.EntityMWeaponAbilityGroup]("m_weapon_ability_group")
	if err != nil {
		return nil, err
	}
	stories, err := memorydb.ReadTable[masterdata.EntityMWeaponStoryReleaseConditionGroup]("m_weapon_story_release_condition_group")
	if err != nil {
		return nil, err
	}
	duplicates, err := masterdata.LoadDupExchange()
	if err != nil {
		return nil, err
	}
	// Reuse the server's book/awakening-stone fallback for the gifted costume.
	// EnrichDupExchange only needs the costume IDs, not a populated gacha pool.
	if _, err := masterdata.EnrichDupExchange(duplicates, &masterdata.GachaCatalog{
		CostumeById: map[int32]masterdata.GachaPoolItem{24008: {}},
	}); err != nil {
		return nil, err
	}
	g := &store.PossessionGranter{
		CostumeById:        make(map[int32]store.CostumeRef, len(costumes)),
		WeaponById:         make(map[int32]store.WeaponRef, len(weapons)),
		WeaponSkillSlots:   make(map[int32][]int32),
		WeaponAbilitySlots: make(map[int32][]int32),
		ReleaseConditions:  make(map[int32][]store.WeaponStoryReleaseCond),
		CostumeDupExchange: duplicates,
	}
	for _, costume := range costumes {
		g.CostumeById[costume.CostumeId] = store.CostumeRef{CharacterId: costume.CharacterId}
	}
	for _, weapon := range weapons {
		g.WeaponById[weapon.WeaponId] = store.WeaponRef{
			WeaponSkillGroupId:                 weapon.WeaponSkillGroupId,
			WeaponAbilityGroupId:               weapon.WeaponAbilityGroupId,
			WeaponStoryReleaseConditionGroupId: weapon.WeaponStoryReleaseConditionGroupId,
		}
	}
	for _, skill := range skills {
		g.WeaponSkillSlots[skill.WeaponSkillGroupId] = append(g.WeaponSkillSlots[skill.WeaponSkillGroupId], skill.SlotNumber)
	}
	for _, ability := range abilities {
		g.WeaponAbilitySlots[ability.WeaponAbilityGroupId] = append(g.WeaponAbilitySlots[ability.WeaponAbilityGroupId], ability.SlotNumber)
	}
	for _, story := range stories {
		g.ReleaseConditions[story.WeaponStoryReleaseConditionGroupId] = append(g.ReleaseConditions[story.WeaponStoryReleaseConditionGroupId], store.WeaponStoryReleaseCond{
			StoryIndex:                      story.StoryIndex,
			WeaponStoryReleaseConditionType: model.WeaponStoryReleaseConditionType(story.WeaponStoryReleaseConditionType),
			ConditionValue:                  story.ConditionValue,
		})
	}
	return g, nil
}
