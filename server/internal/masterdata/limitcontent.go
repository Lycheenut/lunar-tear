package masterdata

import (
	"fmt"
	"sort"

	"lunar-tear/server/internal/utils"
)

type LimitContentCatalog struct {
	ContentsByChapter map[int32][]EntityMEventQuestLimitContent
	RestrictionsById  map[int32][]EntityMEventQuestLimitContentDeckRestriction
	TargetTypesById   map[int32][]int32
}

func LoadLimitContentCatalog() (*LimitContentCatalog, error) {
	relations, err := utils.ReadTable[EntityMEventQuestChapterLimitContentRelation]("m_event_quest_chapter_limit_content_relation")
	if err != nil {
		return nil, fmt.Errorf("load limit content relations: %w", err)
	}
	contents, err := utils.ReadTable[EntityMEventQuestLimitContent]("m_event_quest_limit_content")
	if err != nil {
		return nil, fmt.Errorf("load limit content: %w", err)
	}
	restrictions, err := utils.ReadTable[EntityMEventQuestLimitContentDeckRestriction]("m_event_quest_limit_content_deck_restriction")
	if err != nil {
		return nil, fmt.Errorf("load limit content deck restrictions: %w", err)
	}
	targets, err := utils.ReadTable[EntityMEventQuestLimitContentDeckRestrictionTarget]("m_event_quest_limit_content_deck_restriction_target")
	if err != nil {
		return nil, fmt.Errorf("load limit content restriction targets: %w", err)
	}

	contentById := make(map[int32]EntityMEventQuestLimitContent, len(contents))
	for _, content := range contents {
		contentById[content.EventQuestLimitContentId] = content
	}
	catalog := &LimitContentCatalog{ContentsByChapter: make(map[int32][]EntityMEventQuestLimitContent), RestrictionsById: make(map[int32][]EntityMEventQuestLimitContentDeckRestriction), TargetTypesById: make(map[int32][]int32)}
	for _, relation := range relations {
		if content, ok := contentById[relation.EventQuestLimitContentId]; ok {
			catalog.ContentsByChapter[relation.EventQuestChapterId] = append(catalog.ContentsByChapter[relation.EventQuestChapterId], content)
		}
	}
	for _, restriction := range restrictions {
		catalog.RestrictionsById[restriction.EventQuestLimitContentDeckRestrictionId] = append(catalog.RestrictionsById[restriction.EventQuestLimitContentDeckRestrictionId], restriction)
	}
	for _, target := range targets {
		catalog.TargetTypesById[target.EventQuestLimitContentDeckRestrictionTargetId] = append(catalog.TargetTypesById[target.EventQuestLimitContentDeckRestrictionTargetId], target.LimitContentDeckRestrictionType)
	}
	return catalog, nil
}

func (c *LimitContentCatalog) ActiveRestrictionTypes(chapterId int32, nowMillis int64) []int32 {
	seen := make(map[int32]bool)
	for _, content := range c.ContentsByChapter[chapterId] {
		if nowMillis < content.StartDatetime || nowMillis >= content.EndDatetime {
			continue
		}
		for _, restriction := range c.RestrictionsById[content.EventQuestLimitContentDeckRestrictionId] {
			if nowMillis < restriction.StartDatetime || nowMillis >= restriction.EndDatetime {
				continue
			}
			for _, targetType := range c.TargetTypesById[restriction.EventQuestLimitContentDeckRestrictionTargetId] {
				if targetType != 0 {
					seen[targetType] = true
				}
			}
		}
	}
	types := make([]int32, 0, len(seen))
	for targetType := range seen {
		types = append(types, targetType)
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })
	return types
}
