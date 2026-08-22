package service

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const missionOptionDailySummon int32 = 100026

type GachaServiceServer struct {
	pb.UnimplementedGachaServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewGachaServiceServer(
	users store.UserRepository,
	sessions store.SessionRepository,
	holder *runtime.Holder,
) *GachaServiceServer {
	return &GachaServiceServer{
		users:    users,
		sessions: sessions,
		holder:   holder,
	}
}

func (s *GachaServiceServer) GetGachaList(ctx context.Context, req *pb.GetGachaListRequest) (*pb.GetGachaListResponse, error) {
	log.Printf("[GachaService] GetGachaList: labels=%v", req.GachaLabelType)

	cat := s.holder.Get()
	catalog := cat.GachaEntries
	handler := cat.GachaHandler
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	user, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		user.EnsureMaps()
		autoConvertExpiredMedals(user, catalog, handler, nowMillis)
	})
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	gachaList := make([]*pb.Gacha, 0, len(catalog))
	for _, entry := range catalog {
		if !gachaVisibleForUser(cat, &user, entry, nowMillis) {
			continue
		}
		if !matchesGachaLabel(req.GachaLabelType, entry.GachaLabelType) {
			continue
		}
		if entry.GachaLabelType == model.GachaLabelPortalCage || entry.GachaLabelType == model.GachaLabelRecycle {
			continue
		}
		bs := user.Gacha.BannerStates[entry.GachaId]
		entry = gachaForUser(cat, &user, entry, nowMillis)
		gachaList = append(gachaList, toProtoGacha(handler.EntryForState(entry, &bs), &bs))
	}

	return &pb.GetGachaListResponse{
		Gacha:               gachaList,
		ConvertedGachaMedal: toProtoConvertedGachaMedal(user.Gacha.ConvertedGachaMedal),
	}, nil
}

func autoConvertExpiredMedals(user *store.UserState, catalog []store.GachaCatalogEntry, handler *gacha.GachaHandler, nowMillis int64) {
	for _, entry := range catalog {
		if entry.GachaMedalId == 0 || gachaActiveAt(entry, nowMillis) {
			continue
		}

		medalInfo, ok := handler.MedalInfo[entry.GachaId]
		if !ok {
			continue
		}
		convertDatetime := medalInfo.AutoConvertDatetime
		if convertDatetime == 0 {
			convertDatetime = entry.EndDatetime
		}
		if convertDatetime == 0 || nowMillis < convertDatetime {
			continue
		}
		obtainItemId := int32(0)
		if handler.Config != nil {
			obtainItemId = handler.Config.ConsumableItemIdForMedal
		}
		if obtainItemId == 0 || obtainItemId == medalInfo.ConsumableItemId {
			continue
		}

		originalCount := user.ConsumableItems[medalInfo.ConsumableItemId]
		bs, exists := user.Gacha.BannerStates[entry.GachaId]
		if exists {
			bs.MedalCount = 0
			user.Gacha.BannerStates[entry.GachaId] = bs
		}
		if originalCount <= 0 {
			continue
		}

		conversionRate := medalInfo.ConversionRate
		if conversionRate <= 0 {
			conversionRate = 1
		}
		bookmarkCount := int32(min(int64(originalCount)*int64(conversionRate), int64(1<<31-1)))
		delete(user.ConsumableItems, medalInfo.ConsumableItemId)
		user.ConsumableItems[obtainItemId] = int32(min(
			int64(user.ConsumableItems[obtainItemId])+int64(bookmarkCount),
			int64(1<<31-1),
		))

		user.Gacha.ConvertedGachaMedal.ConvertedMedalPossession = append(
			user.Gacha.ConvertedGachaMedal.ConvertedMedalPossession,
			store.ConsumableItemState{
				ConsumableItemId: medalInfo.ConsumableItemId,
				Count:            originalCount,
			},
		)
		obtain := user.Gacha.ConvertedGachaMedal.ObtainPossession
		if obtain == nil || obtain.ConsumableItemId != obtainItemId {
			user.Gacha.ConvertedGachaMedal.ObtainPossession = &store.ConsumableItemState{ConsumableItemId: obtainItemId, Count: bookmarkCount}
		} else {
			obtain.Count = int32(min(int64(obtain.Count)+int64(bookmarkCount), int64(1<<31-1)))
		}

		log.Printf("[GachaService] auto-converted %d medals for gacha %d -> %d bookmarks (item %d)",
			originalCount, entry.GachaId, bookmarkCount, obtainItemId)
	}
}

func (s *GachaServiceServer) GetGacha(ctx context.Context, req *pb.GetGachaRequest) (*pb.GetGachaResponse, error) {
	log.Printf("[GachaService] GetGacha: ids=%v", req.GachaId)

	cat := s.holder.Get()
	catalog := cat.GachaEntries
	nowMillis := gametime.NowMillis()

	userId := CurrentUserId(ctx, s.users, s.sessions)
	user, err := s.users.LoadUser(userId)
	if err != nil {
		return nil, fmt.Errorf("snapshot user: %w", err)
	}

	byId := make(map[int32]*pb.Gacha, len(req.GachaId))
	for _, wantedId := range req.GachaId {
		for _, entry := range catalog {
			if entry.GachaId != wantedId {
				continue
			}
			if !gachaVisibleForUser(cat, &user, entry, nowMillis) {
				break
			}
			entry = gachaForUser(cat, &user, entry, nowMillis)
			bs := user.Gacha.BannerStates[entry.GachaId]
			byId[wantedId] = toProtoGacha(cat.GachaHandler.EntryForState(entry, &bs), &bs)
			break
		}
	}

	return &pb.GetGachaResponse{
		Gacha: byId,
	}, nil
}

func (s *GachaServiceServer) Draw(ctx context.Context, req *pb.DrawRequest) (*pb.DrawResponse, error) {
	log.Printf("[GachaService] Draw: gachaId=%d phaseId=%d execCount=%d", req.GachaId, req.GachaPricePhaseId, req.ExecCount)

	cat := s.holder.Get()
	entry := findCatalogEntry(cat.GachaEntries, req.GachaId)
	nowMillis := gametime.NowMillis()
	if entry == nil || !gachaVisible(cat, *entry, nowMillis) {
		return nil, fmt.Errorf("gacha %d not found", req.GachaId)
	}
	handler := cat.GachaHandler

	userId := CurrentUserId(ctx, s.users, s.sessions)
	execCount := req.ExecCount
	if execCount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "exec count must be positive")
	}

	var drawResult *gacha.DrawResult
	var drawErr error
	ownedCostumes := map[int32]bool{}
	acquiredWeapons := map[int32]bool{}
	updatedUser, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		if !gachaUnlocked(cat, user, *entry, nowMillis) {
			drawErr = status.Error(codes.FailedPrecondition, "gacha is locked")
			return
		}
		for _, c := range user.Costumes {
			ownedCostumes[c.CostumeId] = true
		}
		acquiredWeapons = acquiredWeaponIds(user)
		drawResult, drawErr = handler.HandleDraw(user, *entry, req.GachaPricePhaseId, execCount)
		if drawErr != nil {
			return
		}
		store.AddMissionCount(user, int32(model.MissionClearConditionTypeGachaExecByCount), execCount, entry.GachaId, entry.GachaId)
	})
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	if drawErr != nil {
		return nil, status.Error(codes.FailedPrecondition, drawErr.Error())
	}

	for i, item := range drawResult.Items {
		if bonus, ok := drawResult.BonusItems[i]; ok {
			log.Printf("[GachaService] drawn[%d]: type=%d id=%d rarity=%d + bonus type=%d id=%d rarity=%d",
				i, item.PossessionType, item.PossessionId, item.RarityType,
				bonus.PossessionType, bonus.PossessionId, bonus.RarityType)
		} else {
			log.Printf("[GachaService] drawn[%d]: type=%d id=%d rarity=%d",
				i, item.PossessionType, item.PossessionId, item.RarityType)
		}
	}

	gachaResults := make([]*pb.DrawGachaOddsItem, 0, len(drawResult.Items))
	dupMap := make(map[int]gacha.DuplicateInfo)
	for _, d := range drawResult.DuplicateInfos {
		dupMap[d.Index] = d
	}
	bonusDupMap := make(map[int]gacha.DuplicateInfo)
	for _, d := range drawResult.BonusDuplicateInfos {
		bonusDupMap[d.Index] = d
	}

	costumePT := int32(model.PossessionTypeCostume)
	weaponPT := int32(model.PossessionTypeWeapon)
	isMaterialDraw := model.IsMaterialBanner(entry.GachaLabelType)

	for i, item := range drawResult.Items {
		isNew := !isOwnedByType(item, ownedCostumes, acquiredWeapons, updatedUser)
		if item.PossessionType == weaponPT {
			isNew = isNewWeaponInDraw(item.PossessionId, acquiredWeapons)
		}

		var oddsItem *pb.DrawGachaOddsItem

		if isMaterialDraw {
			oddsItem = &pb.DrawGachaOddsItem{
				GachaItem: &pb.GachaItem{
					PossessionType: item.PossessionType,
					PossessionId:   item.PossessionId,
					Count:          gachaItemCount(item),
					IsNew:          isNew,
				},
				GachaItemBonus: &pb.GachaItem{},
			}
		} else if bonus, hasBonusWeapon := drawResult.BonusItems[i]; hasBonusWeapon {
			bonusIsNew := isNewWeaponInDraw(bonus.PossessionId, acquiredWeapons)
			oddsItem = &pb.DrawGachaOddsItem{
				GachaItem: &pb.GachaItem{
					PossessionType: costumePT,
					PossessionId:   item.PossessionId,
					Count:          gachaItemCount(item),
					IsNew:          isNew,
				},
				GachaItemBonus: &pb.GachaItem{
					PossessionType: weaponPT,
					PossessionId:   bonus.PossessionId,
					Count:          1,
					IsNew:          bonusIsNew,
				},
			}
		} else {
			oddsItem = &pb.DrawGachaOddsItem{
				GachaItem: &pb.GachaItem{
					PossessionType: item.PossessionType,
					PossessionId:   item.PossessionId,
					Count:          gachaItemCount(item),
					IsNew:          isNew,
				},
				GachaItemBonus: &pb.GachaItem{},
			}
		}

		oddsItem.MedalBonus = &pb.GachaBonus{}
		if drawResult.MedalBonus > 0 && entry.MedalConsumableItemId != 0 {
			oddsItem.MedalBonus = &pb.GachaBonus{
				PossessionType: int32(model.PossessionTypeConsumableItem),
				PossessionId:   entry.MedalConsumableItemId,
				Count:          0,
			}
		}

		if dup, ok := dupMap[i]; ok {
			applyDuplicationBonus(oddsItem, dup)
		}
		if bdup, ok := bonusDupMap[i]; ok {
			applyDuplicationBonus(oddsItem, bdup)
		}
		oddsItem.IsTarget = item.IsTarget

		gachaResults = append(gachaResults, oddsItem)
	}

	var bonuses []*pb.GachaBonus
	for _, b := range drawResult.Bonuses {
		bonuses = append(bonuses, &pb.GachaBonus{
			PossessionType: b.PossessionType,
			PossessionId:   b.PossessionId,
			Count:          b.Count,
		})
	}

	bs := updatedUser.Gacha.BannerStates[entry.GachaId]
	nextEntry := gachaForUser(cat, &updatedUser, *entry, nowMillis)
	// Draw succeeded while this Gacha was unlocked. Keep the response-side
	// NextGacha usable for the result production even when the draw consumed
	// the last required ticket; subsequent list requests will hide it.
	nextEntry.IsUserGachaUnlock = true
	nextGacha := toProtoGacha(handler.EntryForState(nextEntry, &bs), &bs)

	return &pb.DrawResponse{
		NextGacha:          nextGacha,
		GachaResult:        gachaResults,
		GachaBonus:         bonuses,
		MenuGachaBadgeInfo: []*pb.MenuGachaBadgeInfo{},
	}, nil
}

func (s *GachaServiceServer) ResetBoxGacha(ctx context.Context, req *pb.ResetBoxGachaRequest) (*pb.ResetBoxGachaResponse, error) {
	log.Printf("[GachaService] ResetBoxGacha: gachaId=%d", req.GachaId)

	cat := s.holder.Get()
	entry := findCatalogEntry(cat.GachaEntries, req.GachaId)
	nowMillis := gametime.NowMillis()
	if entry == nil || !gachaVisible(cat, *entry, nowMillis) {
		return nil, fmt.Errorf("gacha %d not found", req.GachaId)
	}
	handler := cat.GachaHandler

	userId := CurrentUserId(ctx, s.users, s.sessions)
	var resetErr error
	updatedUser, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		if !gachaUnlocked(cat, user, *entry, nowMillis) {
			resetErr = status.Error(codes.FailedPrecondition, "gacha is locked")
			return
		}
		if entry.GachaModeType != model.GachaModeBox {
			resetErr = status.Error(codes.FailedPrecondition, "gacha is not a box gacha")
			return
		}
		resetErr = handler.HandleResetBox(user, *entry)
	})
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	if resetErr != nil {
		return nil, resetErr
	}

	bs := updatedUser.Gacha.BannerStates[entry.GachaId]

	return &pb.ResetBoxGachaResponse{
		Gacha: toProtoGacha(handler.EntryForState(*entry, &bs), &bs),
	}, nil
}

func (s *GachaServiceServer) GetRewardGacha(ctx context.Context, req *emptypb.Empty) (*pb.GetRewardGachaResponse, error) {
	log.Printf("[GachaService] GetRewardGacha")
	userId := CurrentUserId(ctx, s.users, s.sessions)
	user, err := s.users.LoadUser(userId)
	if err != nil {
		return nil, fmt.Errorf("snapshot user: %w", err)
	}

	maxCount := s.holder.Get().GachaHandler.Config.RewardGachaDailyMaxCount
	if maxCount <= 0 {
		maxCount = model.DefaultDailyDrawLimit
	}

	todayStart := gametime.StartOfBusinessDayMillis()
	drawCount := user.Gacha.TodaysCurrentDrawCount
	if user.Gacha.LastRewardDrawDate < todayStart {
		drawCount = 0
	}

	return &pb.GetRewardGachaResponse{
		Available:              drawCount < maxCount,
		TodaysCurrentDrawCount: drawCount,
		DailyMaxCount:          maxCount,
	}, nil
}

func (s *GachaServiceServer) RewardDraw(ctx context.Context, req *pb.RewardDrawRequest) (*pb.RewardDrawResponse, error) {
	log.Printf("[GachaService] RewardDraw: placement=%q reward=%q amount=%q", req.PlacementName, req.RewardName, req.RewardAmount)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	handler := s.holder.Get().GachaHandler

	var items []gacha.DrawnItem
	ownedCostumes := map[int32]bool{}
	acquiredWeapons := map[int32]bool{}
	updatedUser, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		for _, c := range user.Costumes {
			ownedCostumes[c.CostumeId] = true
		}
		acquiredWeapons = acquiredWeaponIds(user)
		var drawErr error
		items, drawErr = handler.HandleRewardDraw(user, 1)
		if drawErr != nil {
			log.Printf("[GachaService] RewardDraw error: %v", drawErr)
			return
		}
		store.AddMissionCount(user, int32(model.MissionClearConditionTypeGachaDrawByCount), int32(len(items)), 0, missionOptionDailySummon)
		store.AddMissionCount(user, int32(model.MissionClearConditionTypeGachaExecByCount), 1, 0, 0)
	})
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	results := make([]*pb.RewardGachaItem, 0, len(items))
	for _, item := range items {
		results = append(results, &pb.RewardGachaItem{
			PossessionType: item.PossessionType,
			PossessionId:   item.PossessionId,
			Count:          gachaItemCount(item),
			IsNew:          !isOwnedByType(item, ownedCostumes, acquiredWeapons, updatedUser),
		})
	}

	return &pb.RewardDrawResponse{
		RewardGachaResult: results,
	}, nil
}

func findCatalogEntry(catalog []store.GachaCatalogEntry, gachaId int32) *store.GachaCatalogEntry {
	for i := range catalog {
		if catalog[i].GachaId == gachaId {
			return &catalog[i]
		}
	}
	return nil
}

func matchesGachaLabel(labels []int32, label int32) bool {
	if len(labels) == 0 {
		return true
	}
	for _, candidate := range labels {
		if candidate == label {
			return true
		}
	}
	return false
}

func gachaActiveAt(entry store.GachaCatalogEntry, nowMillis int64) bool {
	if entry.IsInactive {
		return false
	}
	if (entry.GachaLabelType == model.GachaLabelEvent || entry.GachaLabelType == model.GachaLabelChapter) && entry.BoxCount <= 0 {
		return false
	}
	if entry.StartDatetime != 0 && nowMillis < entry.StartDatetime {
		return false
	}
	if entry.EndDatetime != 0 && nowMillis >= entry.EndDatetime {
		return false
	}
	return true
}

func gachaVisible(cat *runtime.Catalogs, entry store.GachaCatalogEntry, nowMillis int64) bool {
	if !gachaActiveAt(entry, nowMillis) {
		return false
	}
	if entry.RelatedEventQuestChapterId != 0 {
		chapter, ok := cat.Quest.EventChapterById[entry.RelatedEventQuestChapterId]
		if !ok || nowMillis < chapter.StartDatetime || (chapter.EndDatetime > 0 && nowMillis >= chapter.EndDatetime) {
			return false
		}
	}
	return true
}

func gachaVisibleForUser(cat *runtime.Catalogs, user *store.UserState, entry store.GachaCatalogEntry, nowMillis int64) bool {
	if !gachaVisible(cat, entry, nowMillis) {
		return false
	}
	if entry.GachaLabelType == model.GachaLabelChapter && !gachaUnlocked(cat, user, entry, nowMillis) {
		return false
	}
	return entry.RequiredConsumableItemId == 0 || user.ConsumableItems[entry.RequiredConsumableItemId] > 0
}

func gachaForUser(cat *runtime.Catalogs, user *store.UserState, entry store.GachaCatalogEntry, nowMillis int64) store.GachaCatalogEntry {
	entry.IsUserGachaUnlock = gachaUnlocked(cat, user, entry, nowMillis)
	if entry.GachaAutoResetType == model.GachaAutoResetMonthly {
		entry.NextAutoResetDatetime = gametime.StartOfNextBusinessMonthAtMillis(nowMillis)
	}
	return entry
}

func gachaUnlocked(cat *runtime.Catalogs, user *store.UserState, entry store.GachaCatalogEntry, nowMillis int64) bool {
	if !entry.IsUserGachaUnlock {
		return false
	}
	if !chapterGachaRouteSelected(cat, user, entry) {
		return false
	}
	if entry.RequiredConsumableItemId != 0 && user.ConsumableItems[entry.RequiredConsumableItemId] <= 0 {
		return false
	}
	if entry.RelatedEventQuestChapterId != 0 {
		return cat.QuestHandler.EventChapterAvailable(user, entry.RelatedEventQuestChapterId, nowMillis) == nil
	}
	for _, condition := range entry.UnlockConditions {
		switch condition.GachaUnlockConditionType {
		case model.GachaUnlockNone:
		case model.GachaUnlockUserRankGreaterOrEqual:
			if user.Status.Level < condition.ConditionValue {
				return false
			}
		case model.GachaUnlockWithinHoursFromGameStart:
			if nowMillis >= user.GameStartDatetime+int64(condition.ConditionValue)*int64(time.Hour/time.Millisecond) {
				return false
			}
		case model.GachaUnlockMainQuestClear:
			if user.Quests[condition.ConditionValue].QuestStateType != model.UserQuestStateTypeCleared {
				return false
			}
		case model.GachaUnlockMainFunctionReleased:
			if user.Tutorials[condition.ConditionValue].ProgressPhase <= 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func chapterGachaRouteSelected(cat *runtime.Catalogs, user *store.UserState, entry store.GachaCatalogEntry) bool {
	if entry.GachaLabelType != model.GachaLabelChapter || entry.RelatedMainQuestChapterId == 0 ||
		cat.Quest == nil || cat.QuestHandler == nil {
		return true
	}
	routeId := cat.Quest.MainQuestRouteIdByChapterId[entry.RelatedMainQuestChapterId]
	seasonId := cat.Quest.SeasonIdByRouteId[routeId]
	if routeId == 0 || seasonId <= 1 || len(cat.Quest.RoutesBySeason[seasonId]) <= 1 {
		return true
	}
	return cat.QuestHandler.SeasonRoutesFor(user)[seasonId] == routeId
}

func toProtoGacha(entry store.GachaCatalogEntry, bs *store.GachaBannerState) *pb.Gacha {
	g := &pb.Gacha{
		GachaId:                    entry.GachaId,
		GachaLabelType:             entry.GachaLabelType,
		GachaModeType:              entry.GachaModeType,
		GachaAutoResetType:         entry.GachaAutoResetType,
		GachaAutoResetPeriod:       entry.GachaAutoResetPeriod,
		NextAutoResetDatetime:      safeTimestamp(entry.NextAutoResetDatetime),
		GachaUnlockCondition:       make([]*pb.GachaUnlockCondition, 0, len(entry.UnlockConditions)),
		IsUserGachaUnlock:          entry.IsUserGachaUnlock,
		StartDatetime:              safeTimestamp(entry.StartDatetime),
		EndDatetime:                safeTimestamp(entry.EndDatetime),
		RelatedMainQuestChapterId:  entry.RelatedMainQuestChapterId,
		RelatedEventQuestChapterId: entry.RelatedEventQuestChapterId,
		PromotionMovieAssetId:      entry.PromotionMovieAssetId,
		GachaMedalId:               entry.GachaMedalId,
		GachaDecorationType:        entry.GachaDecorationType,
		SortOrder:                  entry.SortOrder,
		IsInactive:                 entry.IsInactive,
		InformationId:              entry.InformationId,
	}
	for _, condition := range entry.UnlockConditions {
		g.GachaUnlockCondition = append(g.GachaUnlockCondition, &pb.GachaUnlockCondition{GachaUnlockConditionType: condition.GachaUnlockConditionType, ConditionValue: condition.ConditionValue})
	}

	g.GachaPricePhase = buildProtoPricePhases(entry, bs)

	promotionItems := buildProtoPromotionItems(entry, bs)

	switch entry.GachaModeType {
	case model.GachaModeBox:
		boxNumber := int32(1)
		if bs != nil && bs.BoxNumber > 0 {
			boxNumber = bs.BoxNumber
		}
		phaseId := int32(0)
		if len(entry.PricePhases) > 0 {
			phaseId = entry.PricePhases[0].PhaseId
		}
		g.GachaMode = &pb.Gacha_GachaModeBoxComposition{
			GachaModeBoxComposition: &pb.GachaModeBoxComposition{
				GachaBoxGroupId:                 entry.GroupId,
				BoxNumber:                       max(entry.BoxCount, 1),
				CurrentBoxNumber:                boxNumber,
				IsCurrentBoxResettable:          entry.IsCurrentBoxResettable,
				NaviCharacterCommentAssetName:   "production",
				GachaAssetName:                  entry.BannerAssetName,
				GachaPricePhaseId:               phaseId,
				PromotionGachaOddsItem:          promotionItems,
				IsResettableByDrawingAllTargets: entry.IsResettableByAllTargets,
				IsInvalidReset:                  entry.IsInvalidReset,
				GachaDescriptionTextId:          entry.DescriptionTextId,
			},
		}
	case model.GachaModeStepup:
		stepNumber := int32(1)
		loopCount := int32(0)
		if bs != nil {
			if bs.StepNumber > 0 {
				stepNumber = bs.StepNumber
			}
			loopCount = bs.LoopCount
		}
		g.GachaMode = &pb.Gacha_GachaModeStepupComposition{
			GachaModeStepupComposition: &pb.GachaModeStepupComposition{
				GachaStepGroupId:              entry.GroupId,
				StepNumber:                    1,
				CurrentStepNumber:             stepNumber,
				NaviCharacterCommentAssetName: "production",
				GachaAssetName:                entry.BannerAssetName,
				PromotionGachaOddsItem:        promotionItems,
				CurrentLoopCount:              loopCount,
			},
		}
	default:
		g.GachaMode = &pb.Gacha_GachaModeBasic{
			GachaModeBasic: &pb.GachaModeBasic{
				NaviCharacterCommentAssetName: "production",
				GachaAssetName:                entry.BannerAssetName,
				PromotionGachaOddsItem:        promotionItems,
			},
		}
	}

	return g
}

func buildProtoPricePhases(entry store.GachaCatalogEntry, bs *store.GachaBannerState) []*pb.GachaPricePhase {
	phases := make([]*pb.GachaPricePhase, 0, len(entry.PricePhases))

	for _, p := range entry.PricePhases {
		isEnabled := true
		if entry.GachaModeType == model.GachaModeStepup && bs != nil {
			currentStep := bs.StepNumber
			if currentStep <= 0 {
				currentStep = 1
			}
			isEnabled = p.StepNumber == currentStep
		}

		var bonuses []*pb.GachaBonus
		for _, b := range p.Bonuses {
			bonuses = append(bonuses, &pb.GachaBonus{
				PossessionType: b.PossessionType,
				PossessionId:   b.PossessionId,
				Count:          b.Count,
			})
		}

		limitExec := p.LimitExecCount
		if limitExec <= 0 {
			limitExec = 999
		}

		phases = append(phases, &pb.GachaPricePhase{
			GachaPricePhaseId: p.PhaseId,
			IsEnabled:         isEnabled,
			EndDatetime:       safeTimestamp(entry.EndDatetime),
			PriceType:         p.PriceType,
			PriceId:           p.PriceId,
			Price:             p.Price,
			RegularPrice:      p.RegularPrice,
			DrawCount:         p.DrawCount,
			LimitExecCount:    limitExec,
			EachMaxExecCount:  p.DrawCount,
			GachaBonus:        bonuses,
			GachaOddsFixedRarity: &pb.GachaOddsFixedRarity{
				FixedRarityTypeLowerLimit: p.FixedRarityMin,
				FixedCount:                p.FixedCount,
			},
			GachaBadgeType: model.GachaBadgeTypeNone,
		})
	}

	return phases
}

func buildProtoPromotionItems(entry store.GachaCatalogEntry, bs *store.GachaBannerState) []*pb.GachaOddsItem {
	if len(entry.PromotionItems) == 0 {
		return nil
	}
	isMaterial := model.IsMaterialBanner(entry.GachaLabelType)

	items := make([]*pb.GachaOddsItem, 0, len(entry.PromotionItems))
	for i, pi := range entry.PromotionItems {
		count := pi.Count
		if count <= 0 {
			count = 1
		}
		maxDrawableCount := pi.MaxDrawableCount
		if maxDrawableCount <= 0 {
			maxDrawableCount = 999
		}
		var drewCount int32
		if bs != nil && (entry.GachaLabelType != model.GachaLabelChapter || bs.BoxDrewCounts[model.ChapterGachaMonthCounterId] == gametime.BusinessMonthKey(gametime.NowMillis())) {
			drewCount = bs.BoxDrewCounts[pi.CounterId]
		}
		bonus := &pb.GachaItem{}
		if !isMaterial && pi.BonusPossessionType != 0 {
			bonus = &pb.GachaItem{
				PossessionType: pi.BonusPossessionType,
				PossessionId:   pi.BonusPossessionId,
				Count:          1,
				PromotionOrder: int32(i + 1),
			}
		}
		items = append(items, &pb.GachaOddsItem{
			GachaItem: &pb.GachaItem{
				PossessionType: pi.PossessionType,
				PossessionId:   pi.PossessionId,
				Count:          count,
				PromotionOrder: int32(i + 1),
			},
			GachaItemBonus:   bonus,
			MaxDrawableCount: maxDrawableCount,
			DrewCount:        drewCount,
			IsTarget:         pi.IsTarget,
		})
	}
	return items
}

func gachaItemCount(item gacha.DrawnItem) int32 {
	if item.Count > 0 {
		return item.Count
	}
	return 1
}

func toProtoConvertedGachaMedal(state store.ConvertedGachaMedalState) *pb.ConvertedGachaMedal {
	items := make([]*pb.ConsumableItemPossession, 0, len(state.ConvertedMedalPossession))
	for _, item := range state.ConvertedMedalPossession {
		items = append(items, &pb.ConsumableItemPossession{
			ConsumableItemId: item.ConsumableItemId,
			Count:            item.Count,
		})
	}

	obtain := &pb.ConsumableItemPossession{
		ConsumableItemId: 0,
		Count:            0,
	}
	if state.ObtainPossession != nil {
		obtain.ConsumableItemId = state.ObtainPossession.ConsumableItemId
		obtain.Count = state.ObtainPossession.Count
	}

	return &pb.ConvertedGachaMedal{
		ConvertedMedalPossession: items,
		ObtainPossession:         obtain,
	}
}

func safeTimestamp(unixMillis int64) *timestamppb.Timestamp {
	if unixMillis == 0 {
		return &timestamppb.Timestamp{Seconds: 0}
	}
	return timestamppb.New(time.UnixMilli(unixMillis))
}

func applyDuplicationBonus(oddsItem *pb.DrawGachaOddsItem, dup gacha.DuplicateInfo) {
	if oddsItem.DuplicationBonusGrade == 0 {
		oddsItem.DuplicationBonusGrade = dup.Grade
	}
	for _, b := range dup.Bonuses {
		oddsItem.DuplicationBonus = append(oddsItem.DuplicationBonus, &pb.GachaBonus{
			PossessionType: b.PossessionType,
			PossessionId:   b.PossessionId,
			Count:          b.Count,
		})
	}
}

func acquiredWeaponIds(user *store.UserState) map[int32]bool {
	weaponIds := make(map[int32]bool, len(user.WeaponNotes))
	for weaponId := range user.WeaponNotes {
		weaponIds[weaponId] = true
	}
	return weaponIds
}

func isNewWeaponInDraw(weaponId int32, acquiredWeapons map[int32]bool) bool {
	isNew := !acquiredWeapons[weaponId]
	acquiredWeapons[weaponId] = true
	return isNew
}

func isOwnedByType(item gacha.DrawnItem, costumes, weapons map[int32]bool, user store.UserState) bool {
	switch item.PossessionType {
	case int32(model.PossessionTypeCostume):
		return costumes[item.PossessionId]
	case int32(model.PossessionTypeWeapon):
		return weapons[item.PossessionId]
	case int32(model.PossessionTypeMaterial):
		return user.Materials[item.PossessionId] > 0
	case int32(model.PossessionTypeWeaponEnhanced):
		return user.ConsumableItems[item.PossessionId] > 0
	}
	return false
}
