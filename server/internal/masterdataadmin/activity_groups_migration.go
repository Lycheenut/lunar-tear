package masterdataadmin

import (
	"fmt"

	"lunar-tear/server/internal/activitygroup"
	"lunar-tear/server/internal/gacha"
)

// Split legacy mixed chapter units and remove sources outside Record/Variation.
// Keep explicit compatible members and custom names; never change schedules.
func migrateActivityGroups(existing *activitygroup.Config, catalog *ActivityGroupCatalog, gachaConfig *gacha.Config) (*activitygroup.Config, error) {
	result := &activitygroup.Config{Version: activitygroup.ConfigVersion, Units: []activitygroup.ActivityUnit{}, Groups: []activitygroup.ActivityGroup{}}
	options := make(map[string]ActivityMemberOption)
	for _, option := range catalog.Options {
		options[activityKey(option.ActivityMember)] = option
	}
	usedIDs := make(map[string]bool)
	for _, unit := range existing.Units {
		usedIDs[unit.ID] = true
	}
	replacements := make(map[string][]string)
	names := make(map[string]string)
	for _, unit := range existing.Units {
		for _, unitType := range []int{activitygroup.TypePremium, activitygroup.TypeRecord, activitygroup.TypeVariation} {
			sources := make(map[int64]bool)
			var source ActivityMemberOption
			for _, member := range unit.Members {
				option := options[activityKey(member)]
				if activitySourceType(option) == unitType {
					sources[member.ID] = true
					source = option
				}
			}
			if len(sources) == 0 {
				continue
			}
			updated := unit
			updated.Type = unitType
			updated.Members = []activitygroup.ActivityMember{}
			if len(replacements[unit.ID]) > 0 {
				for suffix := 1; ; suffix++ {
					updated.ID = fmt.Sprintf("%s:split-%d", unit.ID, suffix)
					if !usedIDs[updated.ID] {
						usedIDs[updated.ID] = true
						break
					}
				}
			}
			for _, member := range unit.Members {
				option, exists := options[activityKey(member)]
				if !exists || !activityMemberAllowed(option, unitType) || (member.Kind == "event" && !sources[option.RelatedChapterID]) {
					continue
				}
				updated.Members = append(updated.Members, member)
			}
			if unitType == activitygroup.TypePremium && len(sources) == 1 && unit.Name == gachaConfig.Banners[int32(source.ID)].BannerAssetName {
				for _, language := range supportedLanguages {
					if source.Titles[language] != "" {
						updated.Name = source.Titles[language]
						break
					}
				}
			}
			names[updated.ID] = updated.Name
			replacements[unit.ID] = append(replacements[unit.ID], updated.ID)
			result.Units = append(result.Units, updated)
		}
	}
	for _, group := range existing.Groups {
		updated := group
		updated.UnitIDs = []string{}
		for _, id := range group.UnitIDs {
			updated.UnitIDs = append(updated.UnitIDs, replacements[id]...)
		}
		if len(updated.UnitIDs) == 0 {
			continue
		}
		if len(group.UnitIDs) == 1 && group.ID == group.UnitIDs[0] {
			for _, oldUnit := range existing.Units {
				if oldUnit.ID == group.ID && oldUnit.Name == group.Name {
					updated.Name = names[updated.UnitIDs[0]]
					break
				}
			}
		}
		result.Groups = append(result.Groups, updated)
	}
	if err := ValidateActivityGroups(result, catalog); err != nil {
		return nil, err
	}
	return result, nil
}
