package masterdata

import (
	"fmt"
	"log"
	"sort"
	"sync"

	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

// MaxUserGimmickRows is the server's per-table cap for gimmick projections and the
// in-memory user.Gimmick.Sequences map. The client hard-caps each of its
// IUserGimmick* tables at 1024 rows; we project up to this many with a small
// safety margin so we never overflow client buffers. Shared by InitSequenceSchedule
// (limits user.Gimmick.Sequences map size) and the IUserGimmick / Ornament /
// Sequence projections.
const MaxUserGimmickRows = 1000

type gimmickScheduleEntry struct {
	ScheduleId      int32
	StartDatetime   int64
	EndDatetime     int64
	FirstSequenceId int32
	RequiredQuestId int32 // 0 = always active
	IsHidden        bool  // hidden-story or cage-memory gimmick: bypasses the quest gate
	Rank            int   // trim priority — see gimmickTypeRank
}

func readGimmickTable[T any](name, what string) ([]T, bool) {
	rows, err := utils.ReadTable[T](name)
	if err != nil {
		log.Printf("[gimmick] %s unavailable, %s empty: %v", name, what, err)
		return nil, false
	}
	return rows, true
}

func gimmickTypeRank(t model.GimmickType) int {
	switch t {
	case model.GimmickTypeReport: // hidden missions / stories
		return 0
	case model.GimmickTypeCageMemory: // lost archives
		return 1
	case model.GimmickTypeCageTreasureHunt: // treasure
		return 2
	case model.GimmickTypeBrokenObelisk, model.GimmickTypeFirstBrokenObelisk:
		return 3
	case model.GimmickTypeIronGrill:
		return 4
	case model.GimmickTypeRadioMessage:
		return 5
	case model.GimmickTypeMapOnlyCageTreasureHunt, model.GimmickTypeMapOnlyHideObelisk:
		return 6
	case model.GimmickTypeCageIntervalDropItem, model.GimmickTypeMapOnlyCageIntervalDrop:
		return 7 // birds — bottom
	}
	return 8
}

type gimmickTypeTables struct {
	byGimmick  map[int32]model.GimmickType
	bySequence map[int32]model.GimmickType
}

var gimmickTypes = sync.OnceValue(loadGimmickTypes)

func loadGimmickTypes() gimmickTypeTables {
	empty := gimmickTypeTables{
		byGimmick:  map[int32]model.GimmickType{},
		bySequence: map[int32]model.GimmickType{},
	}

	gimmicks, ok := readGimmickTable[EntityMGimmick]("m_gimmick", "type tables")
	if !ok {
		return empty
	}
	groups, ok := readGimmickTable[EntityMGimmickGroup]("m_gimmick_group", "type tables")
	if !ok {
		return empty
	}
	sequences, ok := readGimmickTable[EntityMGimmickSequence]("m_gimmick_sequence", "type tables")
	if !ok {
		return empty
	}

	byGimmick := make(map[int32]model.GimmickType, len(gimmicks))
	for _, g := range gimmicks {
		byGimmick[g.GimmickId] = model.GimmickType(g.GimmickType)
	}
	typeByGroup := make(map[int32]model.GimmickType, len(groups))
	for _, grp := range groups {
		if _, seen := typeByGroup[grp.GimmickGroupId]; seen {
			continue
		}
		if t, ok := byGimmick[grp.GimmickId]; ok {
			typeByGroup[grp.GimmickGroupId] = t
		}
	}
	bySequence := make(map[int32]model.GimmickType, len(sequences))
	for _, seq := range sequences {
		if t, ok := typeByGroup[seq.GimmickGroupId]; ok {
			bySequence[seq.GimmickSequenceId] = t
		}
	}
	return gimmickTypeTables{byGimmick: byGimmick, bySequence: bySequence}
}

func gimmickSequenceTypes() map[int32]model.GimmickType {
	return gimmickTypes().bySequence
}

func LoadGimmickSequenceRanks() map[int32]int {
	types := gimmickSequenceTypes()
	out := make(map[int32]int, len(types))
	for sid, t := range types {
		out[sid] = gimmickTypeRank(t)
	}
	return out
}

type SequenceReward struct {
	PossessionType int32
	PossessionId   int32
	Count          int32
}

type GimmickCatalog struct {
	schedules       []gimmickScheduleEntry
	hiddenSequences map[int32]bool             // GimmickSequenceId -> report/cage-memory
	sequenceRewards map[int32][]SequenceReward // GimmickSequenceId -> clear rewards
	gimmickTypes    map[int32]model.GimmickType
	cageMemoryItems map[int32]int32 // CageMemory GimmickId -> ImportantItemId (type 4)
	ornamentRewards map[GimmickOrnamentRef]SequenceReward
}

func LoadGimmickCatalog(resolver *ConditionResolver, cageOrnaments *CageOrnamentCatalog) (*GimmickCatalog, error) {
	rows, err := utils.ReadTable[EntityMGimmickSequenceSchedule]("m_gimmick_sequence_schedule")
	if err != nil {
		return nil, fmt.Errorf("load gimmick sequence schedule table: %w", err)
	}

	seqTypes := gimmickSequenceTypes()
	hiddenSeq := make(map[int32]bool, len(seqTypes))
	for sid, t := range seqTypes {
		if t == model.GimmickTypeReport || t == model.GimmickTypeCageMemory {
			hiddenSeq[sid] = true
		}
	}

	// Pick rule: prefer schedules with EndDatetime in the future (so the client's
	// in-cage filter doesn't drop them), then by lowest StartDatetime (longest
	// historical validity, tends to carry the simplest RequiredQuestId), tie-broken
	// by lowest ScheduleId for determinism. The future-end preference matters for
	// "Fickle Black Birds" (type 1) where real expiry dates vary; other types have
	// EndDatetime = 9999-03-31 so the preference is a no-op.
	now := gametime.NowMillis()
	bestBySeq := make(map[int32]gimmickScheduleEntry, len(rows))
	for _, r := range rows {
		entry := gimmickScheduleEntry{
			ScheduleId:      r.GimmickSequenceScheduleId,
			StartDatetime:   r.StartDatetime,
			EndDatetime:     r.EndDatetime,
			FirstSequenceId: r.FirstGimmickSequenceId,
			IsHidden:        hiddenSeq[r.FirstGimmickSequenceId],
			Rank:            gimmickTypeRank(seqTypes[r.FirstGimmickSequenceId]),
		}
		if r.ReleaseEvaluateConditionId != 0 {
			if qid, ok := resolver.RequiredQuestId(r.ReleaseEvaluateConditionId); ok {
				entry.RequiredQuestId = qid
			}
		}
		if existing, ok := bestBySeq[entry.FirstSequenceId]; ok {
			existingFuture := existing.EndDatetime > now
			entryFuture := entry.EndDatetime > now
			if existingFuture != entryFuture {
				// Future-end schedule wins over expired one.
				if existingFuture {
					continue
				}
			} else if existing.StartDatetime < entry.StartDatetime ||
				(existing.StartDatetime == entry.StartDatetime && existing.ScheduleId <= entry.ScheduleId) {
				continue
			}
		}
		bestBySeq[entry.FirstSequenceId] = entry
	}

	entries := make([]gimmickScheduleEntry, 0, len(bestBySeq))
	hiddenCount := 0
	for _, entry := range bestBySeq {
		if entry.IsHidden {
			hiddenCount++
		}
		entries = append(entries, entry)
	}
	dedupedCount := len(rows) - len(entries)

	// Sort by (Rank, ScheduleId) so ActiveScheduleKeys returns the priority order
	// directly: treasure/lost-archives/hidden-missions first, birds last. This is
	// what lets InitSequenceSchedule's 1000-row cap trim from the bottom.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Rank != entries[j].Rank {
			return entries[i].Rank < entries[j].Rank
		}
		return entries[i].ScheduleId < entries[j].ScheduleId
	})

	sequenceRewards := loadGimmickSequenceRewards()
	cageMemoryItems := loadCageMemoryImportantItems(gimmickTypes().byGimmick)
	ornamentRewards := loadGimmickOrnamentRewards(cageOrnaments)

	log.Printf("gimmick catalog loaded: %d schedules (%d hidden-content, %d duplicates dropped), %d reward sequences, %d cage-memory items, %d ornament rewards",
		len(entries), hiddenCount, dedupedCount, len(sequenceRewards), len(cageMemoryItems), len(ornamentRewards))
	return &GimmickCatalog{
		schedules:       entries,
		hiddenSequences: hiddenSeq,
		sequenceRewards: sequenceRewards,
		gimmickTypes:    gimmickTypes().byGimmick,
		cageMemoryItems: cageMemoryItems,
		ornamentRewards: ornamentRewards,
	}, nil
}

// OrnamentReward resolves one-shot treasure ornaments (types 1 and 7) to the
// cage-ornament reward table. Interval-drop ornaments intentionally have no
// fabricated fallback reward.
func (c *GimmickCatalog) OrnamentReward(gimmickId, ornamentIndex int32) (SequenceReward, bool) {
	r, ok := c.ornamentRewards[GimmickOrnamentRef{GimmickId: gimmickId, OrnamentIndex: ornamentIndex}]
	return r, ok
}

// loadGimmickOrnamentRewards resolves (GimmickId, OrnamentIndex) to the
// CageOrnament reward table for one-shot type-1 and type-7 treasure gimmicks.
// Type 7 uses the view id directly; type 1 uses the chapter-scoped cage id:
//
//	m_gimmick (GimmickType == 1 or 7) -> GimmickOrnamentGroupId
//	m_gimmick_ornament (matching group) -> GimmickOrnamentViewId
//	m_cage_ornament (CageOrnamentId == ViewId, or 1_000_000+ChapterId*1_000+ViewId)
//	m_cage_ornament_reward (matching id) -> PossessionType / PossessionId / Count
func loadGimmickOrnamentRewards(cageOrnaments *CageOrnamentCatalog) map[GimmickOrnamentRef]SequenceReward {
	empty := map[GimmickOrnamentRef]SequenceReward{}
	if cageOrnaments == nil {
		return empty
	}

	gimmicks, ok := readGimmickTable[EntityMGimmick]("m_gimmick", "hidden-bird rewards")
	if !ok {
		return empty
	}
	ornaments, ok := readGimmickTable[EntityMGimmickOrnament]("m_gimmick_ornament", "hidden-bird rewards")
	if !ok {
		return empty
	}

	type gimmickRewardSource struct {
		gimmickId   int32
		gimmickType model.GimmickType
	}
	gimmicksByGroup := make(map[int32][]gimmickRewardSource)
	for _, g := range gimmicks {
		t := model.GimmickType(g.GimmickType)
		if t == model.GimmickTypeCageTreasureHunt || t == model.GimmickTypeMapOnlyCageTreasureHunt {
			gimmicksByGroup[g.GimmickOrnamentGroupId] = append(gimmicksByGroup[g.GimmickOrnamentGroupId], gimmickRewardSource{g.GimmickId, t})
		}
	}

	out := make(map[GimmickOrnamentRef]SequenceReward)
	missing := 0
	for _, o := range ornaments {
		gids, ok := gimmicksByGroup[o.GimmickOrnamentGroupId]
		if !ok {
			continue
		}
		for _, source := range gids {
			cageOrnamentId := o.GimmickOrnamentViewId
			if source.gimmickType == model.GimmickTypeCageTreasureHunt {
				cageOrnamentId = 1_000_000 + o.ChapterId*1_000 + o.GimmickOrnamentViewId
			}
			reward, ok := cageOrnaments.LookupReward(cageOrnamentId)
			if !ok {
				missing++
				continue
			}
			out[GimmickOrnamentRef{GimmickId: source.gimmickId, OrnamentIndex: o.GimmickOrnamentIndex}] = SequenceReward{
				PossessionType: reward.PossessionType,
				PossessionId:   reward.PossessionId,
				Count:          reward.Count,
			}
		}
	}
	if missing > 0 {
		log.Printf("[gimmick] %d treasure ornaments had no m_cage_ornament_reward row", missing)
	}
	return out
}

func (c *GimmickCatalog) GimmickType(gimmickId int32) model.GimmickType {
	return c.gimmickTypes[gimmickId]
}

// CageMemoryImportantItem returns the ImportantItemId (type 4) that the library uses
// to mark a tapped cage memory as collected, given the world-gimmick id. The mapping
// is derived from m_gimmick_additional_asset texture suffixes — see
// loadCageMemoryImportantItems.
func (c *GimmickCatalog) CageMemoryImportantItem(gimmickId int32) (int32, bool) {
	id, ok := c.cageMemoryItems[gimmickId]
	return id, ok
}

// importantItemTypeCageMemory mirrors EntityMImportantItem.ImportantItemType==4 — the
// CageMemory entry that the library's HasCageMemory check resolves to.
const importantItemTypeCageMemory int32 = 4

func loadCageMemoryImportantItems(typeByGimmick map[int32]model.GimmickType) map[int32]int32 {
	empty := map[int32]int32{}

	ornaments, ok := readGimmickTable[EntityMGimmickOrnament]("m_gimmick_ornament", "cage-memory items")
	if !ok {
		return empty
	}
	chapters, ok := readGimmickTable[EntityMMainQuestChapter]("m_main_quest_chapter", "cage-memory items")
	if !ok {
		return empty
	}
	routes, ok := readGimmickTable[EntityMMainQuestRoute]("m_main_quest_route", "cage-memory items")
	if !ok {
		return empty
	}
	cageMemories, ok := readGimmickTable[EntityMCageMemory]("m_cage_memory", "cage-memory items")
	if !ok {
		return empty
	}
	items, ok := readGimmickTable[EntityMImportantItem]("m_important_item", "cage-memory items")
	if !ok {
		return empty
	}

	chapterByOrnamentGroup := make(map[int32]int32, len(ornaments))
	for _, o := range ornaments {
		if _, seen := chapterByOrnamentGroup[o.GimmickOrnamentGroupId]; seen {
			continue
		}
		chapterByOrnamentGroup[o.GimmickOrnamentGroupId] = o.ChapterId
	}
	routeByChapter := make(map[int32]int32, len(chapters))
	for _, c := range chapters {
		routeByChapter[c.MainQuestChapterId] = c.MainQuestRouteId
	}
	seasonByRoute := make(map[int32]int32, len(routes))
	for _, r := range routes {
		seasonByRoute[r.MainQuestRouteId] = r.MainQuestSeasonId
	}
	cmsBySeason := make(map[int32][]int32)
	for _, c := range cageMemories {
		cmsBySeason[c.MainQuestSeasonId] = append(cmsBySeason[c.MainQuestSeasonId], c.CageMemoryId)
	}
	for s := range cmsBySeason {
		sort.Slice(cmsBySeason[s], func(i, j int) bool { return cmsBySeason[s][i] < cmsBySeason[s][j] })
	}
	itemByCageMemory := make(map[int32]int32)
	for _, it := range items {
		if it.ImportantItemType == importantItemTypeCageMemory && it.CageMemoryId != 0 {
			itemByCageMemory[it.CageMemoryId] = it.ImportantItemId
		}
	}

	gimmicksByRoute := make(map[int32][]int32)
	for gid, t := range typeByGimmick {
		if t != model.GimmickTypeCageMemory {
			continue
		}
		chapter, ok := chapterByOrnamentGroup[gid]
		if !ok {
			log.Printf("[gimmick] cage-memory %d has no ornament row, skipping mapping", gid)
			continue
		}
		route, ok := routeByChapter[chapter]
		if !ok {
			log.Printf("[gimmick] cage-memory %d chapter %d has no route, skipping mapping", gid, chapter)
			continue
		}
		gimmicksByRoute[route] = append(gimmicksByRoute[route], gid)
	}
	for r := range gimmicksByRoute {
		sort.Slice(gimmicksByRoute[r], func(i, j int) bool { return gimmicksByRoute[r][i] < gimmicksByRoute[r][j] })
	}

	out := make(map[int32]int32)
	for route, gids := range gimmicksByRoute {
		season, ok := seasonByRoute[route]
		if !ok {
			log.Printf("[gimmick] route %d has no season, skipping %d cage-memory gimmicks", route, len(gids))
			continue
		}
		seasonCms := cmsBySeason[season]
		for i, gid := range gids {
			if i >= len(seasonCms) {
				log.Printf("[gimmick] route %d (season %d) has %d cage-memory gimmicks but only %d cage memories; gimmick %d skipped",
					route, season, len(gids), len(seasonCms), gid)
				continue
			}
			cageMemoryId := seasonCms[i]
			itemId, ok := itemByCageMemory[cageMemoryId]
			if !ok {
				log.Printf("[gimmick] cage memory %d (gimmick %d) has no m_important_item row (type 4), skipping",
					cageMemoryId, gid)
				continue
			}
			out[gid] = itemId
		}
	}
	return out
}

func loadGimmickSequenceRewards() map[int32][]SequenceReward {
	empty := map[int32][]SequenceReward{}

	sequences, ok := readGimmickTable[EntityMGimmickSequence]("m_gimmick_sequence", "sequence rewards")
	if !ok {
		return empty
	}
	rewardGroups, ok := readGimmickTable[EntityMGimmickSequenceRewardGroup]("m_gimmick_sequence_reward_group", "sequence rewards")
	if !ok {
		return empty
	}

	rewardsByGroup := make(map[int32][]SequenceReward)
	for _, rg := range rewardGroups {
		if rg.PossessionType == 0 || rg.PossessionId == 0 {
			continue
		}
		rewardsByGroup[rg.GimmickSequenceRewardGroupId] = append(
			rewardsByGroup[rg.GimmickSequenceRewardGroupId], SequenceReward{
				PossessionType: rg.PossessionType,
				PossessionId:   rg.PossessionId,
				Count:          rg.Count,
			})
	}

	rewardsBySequence := make(map[int32][]SequenceReward, len(sequences))
	for _, seq := range sequences {
		if rewards := rewardsByGroup[seq.GimmickSequenceRewardGroupId]; len(rewards) > 0 {
			rewardsBySequence[seq.GimmickSequenceId] = rewards
		}
	}
	return rewardsBySequence
}

func (c *GimmickCatalog) IsHiddenSequence(sequenceId int32) bool {
	return c.hiddenSequences[sequenceId]
}

func (c *GimmickCatalog) SequenceRewards(sequenceId int32) []SequenceReward {
	return c.sequenceRewards[sequenceId]
}

func (c *GimmickCatalog) ActiveScheduleKeys(user store.UserState, nowMillis int64) []store.GimmickSequenceKey {
	keys := make([]store.GimmickSequenceKey, 0, len(c.schedules))
	for _, s := range c.schedules {
		if nowMillis < s.StartDatetime {
			continue // future schedules still skipped
		}
		if !s.IsHidden && s.RequiredQuestId != 0 {
			q, ok := user.Quests[s.RequiredQuestId]
			if !ok || q.QuestStateType != model.UserQuestStateTypeCleared {
				continue
			}
		}
		keys = append(keys, store.GimmickSequenceKey{
			GimmickSequenceScheduleId: s.ScheduleId,
			GimmickSequenceId:         s.FirstSequenceId,
		})
	}
	return keys
}

type GimmickOrnamentRef struct {
	GimmickId     int32
	OrnamentIndex int32
}

func LoadGimmickOrnamentRefs() map[int32][]GimmickOrnamentRef {
	empty := map[int32][]GimmickOrnamentRef{}

	sequences, ok := readGimmickTable[EntityMGimmickSequence]("m_gimmick_sequence", "ornament refs")
	if !ok {
		return empty
	}
	groups, ok := readGimmickTable[EntityMGimmickGroup]("m_gimmick_group", "ornament refs")
	if !ok {
		return empty
	}
	gimmicks, ok := readGimmickTable[EntityMGimmick]("m_gimmick", "ornament refs")
	if !ok {
		return empty
	}
	ornaments, ok := readGimmickTable[EntityMGimmickOrnament]("m_gimmick_ornament", "ornament refs")
	if !ok {
		return empty
	}

	indicesByOrnamentGroup := make(map[int32][]int32)
	for _, o := range ornaments {
		indicesByOrnamentGroup[o.GimmickOrnamentGroupId] = append(
			indicesByOrnamentGroup[o.GimmickOrnamentGroupId], o.GimmickOrnamentIndex)
	}
	ornamentGroupByGimmick := make(map[int32]int32, len(gimmicks))
	for _, g := range gimmicks {
		ornamentGroupByGimmick[g.GimmickId] = g.GimmickOrnamentGroupId
	}
	gimmicksByGroup := make(map[int32][]int32)
	for _, grp := range groups {
		gimmicksByGroup[grp.GimmickGroupId] = append(gimmicksByGroup[grp.GimmickGroupId], grp.GimmickId)
	}

	refsBySequence := make(map[int32][]GimmickOrnamentRef, len(sequences))
	for _, seq := range sequences {
		var refs []GimmickOrnamentRef
		for _, gimmickId := range gimmicksByGroup[seq.GimmickGroupId] {
			for _, ornamentIndex := range indicesByOrnamentGroup[ornamentGroupByGimmick[gimmickId]] {
				refs = append(refs, GimmickOrnamentRef{GimmickId: gimmickId, OrnamentIndex: ornamentIndex})
			}
		}
		if len(refs) > 0 {
			refsBySequence[seq.GimmickSequenceId] = refs
		}
	}
	log.Printf("gimmick ornament refs loaded: %d sequences", len(refsBySequence))
	return refsBySequence
}

func LoadHiddenGimmickSequenceIDs() map[int32]bool {
	types := gimmickSequenceTypes()
	out := make(map[int32]bool, len(types))
	for sid, t := range types {
		if t == model.GimmickTypeReport || t == model.GimmickTypeCageMemory || t == model.GimmickTypeMapOnlyCageTreasureHunt {
			out[sid] = true
		}
	}
	return out
}

func LoadBirdGimmickIDs() map[int32]bool {
	byGimmick := gimmickTypes().byGimmick
	out := make(map[int32]bool, len(byGimmick))
	for gid, t := range byGimmick {
		if t == model.GimmickTypeCageIntervalDropItem || t == model.GimmickTypeMapOnlyCageIntervalDrop {
			out[gid] = true
		}
	}
	return out
}

func LoadGimmickSequenceChains() map[int32][]int32 {
	empty := map[int32][]int32{}

	sequences, ok := readGimmickTable[EntityMGimmickSequence]("m_gimmick_sequence", "sequence chains")
	if !ok {
		return empty
	}
	groups, ok := readGimmickTable[EntityMGimmickSequenceGroup]("m_gimmick_sequence_group", "sequence chains")
	if !ok {
		return empty
	}

	membersByGroup := make(map[int32][]int32)
	for _, g := range groups {
		membersByGroup[g.GimmickSequenceGroupId] = append(membersByGroup[g.GimmickSequenceGroupId], g.GimmickSequenceId)
	}
	nextGroupBySequence := make(map[int32]int32, len(sequences))
	for _, seq := range sequences {
		nextGroupBySequence[seq.GimmickSequenceId] = seq.NextGimmickSequenceGroupId
	}

	chains := make(map[int32][]int32, len(sequences))
	for _, seq := range sequences {
		start := seq.GimmickSequenceId
		seen := map[int32]bool{start: true}
		chain := []int32{start}
		for queue := []int32{start}; len(queue) > 0; {
			cur := queue[0]
			queue = queue[1:]
			nextGroup := nextGroupBySequence[cur]
			if nextGroup == 0 {
				continue
			}
			for _, member := range membersByGroup[nextGroup] {
				if !seen[member] {
					seen[member] = true
					chain = append(chain, member)
					queue = append(queue, member)
				}
			}
		}
		chains[start] = chain
	}
	return chains
}
