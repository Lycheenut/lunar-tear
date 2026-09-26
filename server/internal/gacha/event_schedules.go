package gacha

import (
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

// ApplyEventSchedules lets existing activity schedules cover every ticket tier.
// An explicitly scheduled tier takes precedence over the base event schedule.
func ApplyEventSchedules(entries []store.GachaCatalogEntry, config *Config) {
	for i := range entries {
		entry := &entries[i]
		if entry.GachaLabelType != model.GachaLabelEvent {
			continue
		}
		schedule, ok := config.EventSchedules[entry.GachaId]
		if !ok {
			schedule, ok = config.EventSchedules[entry.EventGachaBaseId]
		}
		if ok {
			entry.StartDatetime = schedule.StartDatetime
			entry.EndDatetime = schedule.EndDatetime
		}
	}
}
