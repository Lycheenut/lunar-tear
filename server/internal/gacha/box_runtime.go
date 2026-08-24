package gacha

import (
	"math"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func (h *GachaHandler) configuredBox(entry store.GachaCatalogEntry, state *store.GachaBannerState) (BoxConfig, int32, bool) {
	if h.Premium == nil || h.Premium.Config == nil {
		return BoxConfig{}, 0, false
	}
	switch entry.GachaLabelType {
	case model.GachaLabelChapter:
		box, ok := h.Premium.Config.ChapterBanners[entry.GachaId]
		return box, 1, ok
	case model.GachaLabelEvent:
		event, ok := h.Premium.Config.EventBanners[entry.GachaId]
		if !ok || len(event.Boxes) == 0 {
			return BoxConfig{}, 0, false
		}
		boxNumber := currentBoxNumber(state, int32(len(event.Boxes)))
		return event.Boxes[boxNumber-1], int32(len(event.Boxes)), true
	default:
		return BoxConfig{}, 0, false
	}
}

func currentBoxNumber(state *store.GachaBannerState, boxCount int32) int32 {
	boxNumber := int32(1)
	if state != nil && state.BoxNumber > 0 {
		boxNumber = state.BoxNumber
	}
	if boxCount > 0 && boxNumber > boxCount {
		return boxCount
	}
	return boxNumber
}

// EntryForState resolves the configured Event box for a user and supplies the
// reset flags and featured rewards expected by the client.
func (h *GachaHandler) EntryForState(entry store.GachaCatalogEntry, state *store.GachaBannerState) store.GachaCatalogEntry {
	box, boxCount, configured := h.configuredBox(entry, state)
	if configured {
		entry.BoxItems = boxItems(box)
		entry.PromotionItems = boxPromotions(entry.BoxItems, entry.GachaLabelType == model.GachaLabelEvent)
		entry.BoxCount = boxCount
	}
	if entry.GachaLabelType != model.GachaLabelEvent || entry.BoxCount <= 0 {
		return entry
	}
	boxNumber := currentBoxNumber(state, entry.BoxCount)
	entry.IsCurrentBoxResettable = boxResettable(entry.BoxItems, state, boxNumber == entry.BoxCount)
	entry.IsResettableByAllTargets = true
	entry.IsInvalidReset = !entry.IsCurrentBoxResettable
	return entry
}

func boxResettable(items []store.GachaBoxItemEntry, state *store.GachaBannerState, lastBox bool) bool {
	if len(items) == 0 {
		return false
	}
	drewCounts := map[int32]int32(nil)
	if state != nil {
		drewCounts = state.BoxDrewCounts
	}
	foundRequired := false
	for i, item := range items {
		if item.MaxCount <= 0 || (!lastBox && !item.IsJackpot) {
			continue
		}
		foundRequired = true
		if drewCounts[chapterCounterId(item, i)] < item.MaxCount {
			return false
		}
	}
	return foundRequired
}

func availableConfiguredBoxDrawCount(items []store.GachaBoxItemEntry, weights BoxGroupWeights, state *store.GachaBannerState) int64 {
	var available int64
	for i, item := range items {
		if item.MaxCount <= 0 {
			if item.Weight > 0 && weights.Unlimited > 0 {
				return math.MaxInt64
			}
			continue
		}
		if weights.Limited <= 0 {
			continue
		}
		drewCount := int32(0)
		if state != nil {
			drewCount = state.BoxDrewCounts[chapterCounterId(item, i)]
		}
		remaining := int64(item.MaxCount) - int64(drewCount)
		if remaining > 0 {
			available += remaining
		}
	}
	return available
}
