package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ShopServiceServer struct {
	pb.UnimplementedShopServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewShopServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *ShopServiceServer {
	return &ShopServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *ShopServiceServer) Buy(ctx context.Context, req *pb.BuyRequest) (*pb.BuyResponse, error) {
	log.Printf("[ShopService] Buy: shopId=%d items=%v", req.ShopId, req.ShopItems)

	cat := s.holder.Get()
	catalog := cat.Shop
	granter := cat.QuestHandler.Granter
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	if len(req.ShopItems) == 0 {
		return nil, status.Error(codes.InvalidArgument, "shop items are required")
	}
	shop, ok := catalog.Shops[req.ShopId]
	if !ok {
		return nil, status.Error(codes.NotFound, "shop not found")
	}
	if shop.ShopGroupType == model.ShopGroupTypePremiumShop {
		return nil, status.Error(codes.FailedPrecondition, "platform purchases are disabled")
	}
	if !catalog.IsShopOpen(req.ShopId, nowMillis) {
		return nil, status.Error(codes.FailedPrecondition, "shop is not open")
	}

	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		candidate := store.CloneUserState(*user)
		for shopItemId, qty := range req.ShopItems {
			if qty <= 0 {
				validationErr = status.Errorf(codes.InvalidArgument, "invalid quantity for shop item %d", shopItemId)
				return
			}
			item, ok := catalog.Items[shopItemId]
			if !ok {
				validationErr = status.Errorf(codes.NotFound, "shop item %d not found", shopItemId)
				return
			}
			if !catalog.IsItemAvailable(req.ShopId, shopItemId, nowMillis) {
				validationErr = status.Errorf(codes.FailedPrecondition, "shop item %d is not available in shop %d", shopItemId, req.ShopId)
				return
			}
			if shop.ShopGroupType == model.ShopGroupTypeItemShop && !replaceableLineupContains(candidate.ShopReplaceableLineup, shopItemId) {
				validationErr = status.Errorf(codes.FailedPrecondition, "shop item %d is not in the current lineup", shopItemId)
				return
			}
			additionalContents, levelAvailable := catalog.AdditionalContentsForLevel(shopItemId, candidate.Status.Level)
			if !levelAvailable {
				validationErr = status.Errorf(codes.FailedPrecondition, "user level does not satisfy shop item %d", shopItemId)
				return
			}
			si := candidate.ShopItems[shopItemId]
			si.ShopItemId = shopItemId
			if item.ShopItemLimitedStockId > 0 {
				rule, ok := catalog.LimitedStock[item.ShopItemLimitedStockId]
				if !ok || rule.MaxCount <= 0 {
					validationErr = status.Errorf(codes.FailedPrecondition, "shop item %d stock rule is unavailable", shopItemId)
					return
				}
				var resetErr error
				si, resetErr = resetShopItemStockIfDue(si, rule, nowMillis)
				if resetErr != nil {
					validationErr = status.Errorf(codes.FailedPrecondition, "shop item %d stock rule is invalid: %v", shopItemId, resetErr)
					return
				}
				if int64(si.BoughtCount)+int64(qty) > int64(rule.MaxCount) {
					validationErr = status.Errorf(codes.ResourceExhausted, "shop item %d stock limit exceeded", shopItemId)
					return
				}
			}

			totalPrice64 := int64(item.Price) * int64(qty)
			if totalPrice64 <= 0 || totalPrice64 > math.MaxInt32 {
				validationErr = status.Errorf(codes.InvalidArgument, "invalid total price for shop item %d", shopItemId)
				return
			}
			if err := store.DeductPrice(&candidate, item.PriceType, item.PriceId, int32(totalPrice64)); err != nil {
				validationErr = status.Errorf(codes.FailedPrecondition, "cannot buy shop item %d: %v", shopItemId, err)
				return
			}

			for _, content := range catalog.Contents[shopItemId] {
				if err := grantShopPossession(granter, &candidate, content.PossessionType, content.PossessionId, content.Count, qty, nowMillis); err != nil {
					validationErr = status.Errorf(codes.FailedPrecondition, "shop item %d has invalid content", shopItemId)
					return
				}
			}
			for _, content := range additionalContents {
				if err := grantShopPossession(granter, &candidate, content.PossessionType, content.PossessionId, content.Count, qty, nowMillis); err != nil {
					validationErr = status.Errorf(codes.FailedPrecondition, "shop item %d has invalid additional content", shopItemId)
					return
				}
			}

			applyShopContentEffects(catalog, &candidate, shopItemId, qty, nowMillis)

			si.BoughtCount += qty
			si.LatestBoughtCountChangedDatetime = nowMillis
			si.LatestVersion = nowMillis
			candidate.ShopItems[shopItemId] = si
		}
		if exceedsPossessionLimits(*user, candidate, cat.GameConfig) {
			validationErr = status.Error(codes.ResourceExhausted, "purchase exceeds possession limits")
			return
		}
		*user = candidate
	})
	if err != nil {
		return nil, fmt.Errorf("shop buy: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}
	return &pb.BuyResponse{
		OverflowPossession: []*pb.Possession{},
	}, nil
}

func (s *ShopServiceServer) RefreshUserData(ctx context.Context, req *pb.RefreshRequest) (*pb.RefreshResponse, error) {
	log.Printf("[ShopService] RefreshUserData: isGemUsed=%v", req.IsGemUsed)

	catalog := s.holder.Get().Shop
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	if catalog.ItemShopId == 0 || !catalog.IsShopOpen(catalog.ItemShopId, nowMillis) {
		return nil, status.Error(codes.FailedPrecondition, "replaceable shop is not open")
	}

	var validationErr error
	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		isFreeRefreshDue := len(user.ShopReplaceableLineup) == 0 || user.ShopReplaceable.LatestLineupUpdateDatetime < gametime.StartOfBusinessDayAtMillis(nowMillis)
		if !req.IsGemUsed && !isFreeRefreshDue {
			return
		}

		candidate := store.CloneUserState(*user)
		if req.IsGemUsed {
			nextCount := nextReplaceableRefreshCount(candidate.ShopReplaceable.LineupUpdateCount, isFreeRefreshDue)
			price, ok := catalog.ReplaceableGemPrice(nextCount)
			if !ok {
				validationErr = status.Error(codes.FailedPrecondition, "replaceable shop refresh price is unavailable")
				return
			}
			if err := store.DeductPrice(&candidate, model.PriceTypeGem, 0, price); err != nil {
				validationErr = status.Errorf(codes.FailedPrecondition, "cannot refresh replaceable shop: %v", err)
				return
			}
			candidate.ShopReplaceable.LineupUpdateCount = nextCount
		} else {
			candidate.ShopReplaceable.LineupUpdateCount = 0
		}

		pool := make([]int32, 0, len(catalog.ItemShopPool))
		for _, itemId := range catalog.ItemShopPool {
			if catalog.IsItemAvailable(catalog.ItemShopId, itemId, nowMillis) {
				pool = append(pool, itemId)
			}
		}
		if len(pool) == 0 {
			validationErr = status.Error(codes.FailedPrecondition, "replaceable shop has no available items")
			return
		}
		candidate.ShopReplaceableLineup = buildReplaceableLineup(pool, nowMillis)
		candidate.ShopReplaceable.LatestLineupUpdateDatetime = nowMillis
		candidate.ShopReplaceable.LatestVersion = nowMillis
		for _, itemId := range catalog.ItemShopPool {
			if si, ok := candidate.ShopItems[itemId]; ok {
				si.BoughtCount = 0
				si.LatestBoughtCountChangedDatetime = nowMillis
				si.LatestVersion = nowMillis
				candidate.ShopItems[itemId] = si
			}
		}
		*user = candidate
	})
	if err != nil {
		return nil, fmt.Errorf("shop refresh: %w", err)
	}
	if validationErr != nil {
		return nil, validationErr
	}

	return &pb.RefreshResponse{}, nil
}

func (s *ShopServiceServer) GetCesaLimit(_ context.Context, _ *emptypb.Empty) (*pb.GetCesaLimitResponse, error) {
	log.Printf("[ShopService] GetCesaLimit")
	return &pb.GetCesaLimitResponse{
		CesaLimit: []*pb.CesaLimit{},
	}, nil
}

func (s *ShopServiceServer) CreatePurchaseTransaction(ctx context.Context, req *pb.CreatePurchaseTransactionRequest) (*pb.CreatePurchaseTransactionResponse, error) {
	log.Printf("[ShopService] CreatePurchaseTransaction: shopId=%d shopItemId=%d productId=%s",
		req.ShopId, req.ShopItemId, req.ProductId)
	return nil, status.Error(codes.FailedPrecondition, "platform purchases are disabled")
}

func (s *ShopServiceServer) PurchaseGooglePlayStoreProduct(ctx context.Context, req *pb.PurchaseGooglePlayStoreProductRequest) (*pb.PurchaseGooglePlayStoreProductResponse, error) {
	log.Printf("[ShopService] PurchaseGooglePlayStoreProduct: txId=%s", req.PurchaseTransactionId)
	return nil, status.Error(codes.FailedPrecondition, "platform purchases are disabled")
}

func nextReplaceableRefreshCount(currentCount int32, isNewDay bool) int32 {
	if isNewDay {
		return 1
	}
	return currentCount + 1
}

func buildReplaceableLineup(pool []int32, nowMillis int64) map[int32]store.UserShopReplaceableLineupState {
	lineup := make(map[int32]store.UserShopReplaceableLineupState, len(pool))
	for i, itemId := range pool {
		slot := int32(i + 1)
		lineup[slot] = store.UserShopReplaceableLineupState{
			SlotNumber:    slot,
			ShopItemId:    itemId,
			LatestVersion: nowMillis,
		}
	}
	return lineup
}

func replaceableLineupContains(lineup map[int32]store.UserShopReplaceableLineupState, shopItemId int32) bool {
	for _, row := range lineup {
		if row.ShopItemId == shopItemId {
			return true
		}
	}
	return false
}

func grantShopPossession(granter *store.PossessionGranter, user *store.UserState, possessionType, possessionId, count, quantity int32, nowMillis int64) error {
	totalCount := int64(count) * int64(quantity)
	if totalCount <= 0 || totalCount > math.MaxInt32 {
		return fmt.Errorf("invalid content count")
	}
	result := granter.GrantFull(user, model.PossessionType(possessionType), possessionId, int32(totalCount), nowMillis)
	if result.Status != store.GrantStatusGranted {
		return fmt.Errorf("grant status %d", result.Status)
	}
	return nil
}

func resetShopItemStockIfDue(item store.UserShopItemState, rule masterdata.ShopLimitedStockRule, nowMillis int64) (store.UserShopItemState, error) {
	var boundaryMillis int64
	switch rule.AutoResetType {
	case model.ShopItemAutoResetNone:
		return item, nil
	case model.ShopItemAutoResetWeekly:
		if rule.AutoResetPeriod < 1 || rule.AutoResetPeriod > 7 {
			return item, fmt.Errorf("invalid weekly reset day %d", rule.AutoResetPeriod)
		}
		now := gametime.InBusinessLocation(nowMillis)
		currentWeekday := int32(now.Weekday())
		if currentWeekday == 0 {
			currentWeekday = 7
		}
		daysSinceReset := (currentWeekday - rule.AutoResetPeriod + 7) % 7
		boundaryMillis = time.Date(now.Year(), now.Month(), now.Day()-int(daysSinceReset), 0, 0, 0, 0, gametime.BusinessLocation()).UnixMilli()
	case model.ShopItemAutoResetMonthly:
		if rule.AutoResetPeriod < 1 || rule.AutoResetPeriod > 31 {
			return item, fmt.Errorf("invalid monthly reset day %d", rule.AutoResetPeriod)
		}
		now := gametime.InBusinessLocation(nowMillis)
		boundary := monthlyShopResetBoundary(now, int(rule.AutoResetPeriod))
		boundaryMillis = boundary.UnixMilli()
	default:
		return item, fmt.Errorf("unsupported reset type %d", rule.AutoResetType)
	}
	if item.LatestBoughtCountChangedDatetime < boundaryMillis {
		item.BoughtCount = 0
	}
	return item, nil
}

func monthlyShopResetBoundary(now time.Time, resetDay int) time.Time {
	now = now.In(gametime.BusinessLocation())
	year, month := now.Year(), now.Month()
	day := min(resetDay, daysInMonth(year, month))
	boundary := time.Date(year, month, day, 0, 0, 0, 0, gametime.BusinessLocation())
	if now.Before(boundary) {
		month--
		if month == 0 {
			month = 12
			year--
		}
		day = min(resetDay, daysInMonth(year, month))
		boundary = time.Date(year, month, day, 0, 0, 0, 0, gametime.BusinessLocation())
	}
	return boundary
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, gametime.BusinessLocation()).Day()
}

func applyShopContentEffects(catalog *masterdata.ShopCatalog, user *store.UserState, shopItemId, qty int32, nowMillis int64) {
	for _, effect := range catalog.Effects[shopItemId] {
		switch effect.EffectTargetType {
		case model.EffectTargetStaminaRecovery:
			maxMillis := catalog.MaxStaminaMillis[user.Status.Level]
			millis := store.ResolveStaminaEffectMillis(effect.EffectValueType, effect.EffectValue, maxMillis)
			store.RecoverStamina(user, millis*qty, maxMillis, nowMillis)
		default:
			log.Printf("[ShopService] unhandled effect: shopItemId=%d targetType=%d", shopItemId, effect.EffectTargetType)
		}
	}
}
