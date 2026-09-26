package masterdata

import (
	"fmt"

	"lunar-tear/server/internal/store"
)

// Ticket families from the installed consumable_item and gacha_title assets.
// Quest links only identify one pool, and some older links name a ticket from
// the preceding event. Do not infer higher tiers from adjacent item IDs.
var eventGachaTickets = map[int32][]int32{
	311001: {6011, 6012},
	312001: {6013, 6014},
	313001: {6015, 6016},
	314001: {6017, 6018, 6019},
	315001: {6020, 6021, 6022},
	316001: {6023, 6024, 6025},
	317001: {6029, 6030, 6031},
	318001: {6026, 6027, 6028},
	321001: {6033, 6034, 6035},
	322001: {6036, 6037, 6038},
	323001: {6039, 6040, 6041},
	324001: {6042, 6043, 6044},
	325001: {6046, 6047, 6048},
	327001: {6049, 6050, 6051},
	328001: {6052, 6053, 6054},
	329001: {6055, 6056, 6057},
	330001: {6058, 6059, 6060},
	331001: {6061, 6062, 6063},
}

func eventGachaTierEntries(base store.GachaCatalogEntry) []store.GachaCatalogEntry {
	base.EventGachaBaseId = base.GachaId
	tickets := eventGachaTickets[base.GachaId]
	if len(tickets) == 0 {
		return []store.GachaCatalogEntry{base}
	}
	names := []string{"bronze", "silver", "gold"}
	entries := make([]store.GachaCatalogEntry, 0, len(tickets))
	for i, ticketId := range tickets {
		entry := base
		// Keep the existing lowest-tier ID. The other IDs follow the asset
		// families (_01, _11, _21) and own independent box/reset counters.
		entry.GachaId = base.GachaId + int32(i)*10
		entry.GroupId = entry.GachaId
		entry.EventGachaTicketTier = names[i]
		entry.RequiredConsumableItemId = ticketId
		entry.PricePhases = buildChapterPricePhases(entry.GachaId, ticketId)
		entry.BannerAssetName = fmt.Sprintf("event_%d_%02d", base.GachaId/1000, 1+i*10)
		entries = append(entries, entry)
	}
	return entries
}
