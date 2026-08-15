package service

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	giftExpirationFilterNone          int32 = 1
	giftExpirationFilterOnlyExpire    int32 = 2
	giftExpirationFilterOnlyNotExpire int32 = 3

	giftRewardKindNone      int32 = 1
	giftRewardKindGem       int32 = 2
	giftRewardKindGold      int32 = 3
	giftRewardKindWeapon    int32 = 4
	giftRewardKindCompanion int32 = 5
	giftRewardKindParts     int32 = 6
	giftRewardKindMaterial  int32 = 7
	giftRewardKindOther     int32 = 8
	giftRewardKindCostume   int32 = 9
)

type GiftServiceServer struct {
	pb.UnimplementedGiftServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewGiftServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *GiftServiceServer {
	return &GiftServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *GiftServiceServer) ReceiveGift(ctx context.Context, req *pb.ReceiveGiftRequest) (*pb.ReceiveGiftResponse, error) {
	log.Printf("[GiftService] ReceiveGift: giftUuids=%d", len(req.UserGiftUuid))

	userId := CurrentUserId(ctx, s.users, s.sessions)
	requested := make(map[string]struct{}, len(req.UserGiftUuid))
	for _, giftUuid := range req.UserGiftUuid {
		requested[giftUuid] = struct{}{}
	}
	received := make([]string, 0, len(requested))
	expired := make([]string, 0)
	overflow := make([]string, 0)
	cat := s.holder.Get()
	granter := cat.QuestHandler.Granter
	_, err := s.users.UpdateUsers([]int64{userId}, func(users map[int64]*store.UserState) error {
		user := users[userId]
		nowMillis := gametime.NowMillis()
		remaining := make([]store.NotReceivedGiftState, 0, len(user.Gifts.NotReceived))
		for _, gift := range user.Gifts.NotReceived {
			_, selected := requested[gift.UserGiftUuid]
			if isExpiredGift(gift, nowMillis) {
				if selected {
					expired = append(expired, gift.UserGiftUuid)
				}
				continue
			}
			if !selected {
				remaining = append(remaining, gift)
				continue
			}
			result := grantGift(user, gift.GiftCommon, granter, cat.GameConfig, nowMillis)
			switch result.Status {
			case store.GrantStatusOverflow:
				overflow = append(overflow, gift.UserGiftUuid)
				remaining = append(remaining, gift)
				continue
			case store.GrantStatusUnsupported:
				return status.Errorf(codes.FailedPrecondition, "gift %s has unsupported possession type %d", gift.UserGiftUuid, gift.GiftCommon.PossessionType)
			case store.GrantStatusInvalid:
				return status.Errorf(codes.FailedPrecondition, "gift %s has invalid count %d", gift.UserGiftUuid, gift.GiftCommon.Count)
			}
			received = append(received, gift.UserGiftUuid)
			user.Gifts.Received = append(user.Gifts.Received, store.ReceivedGiftState{
				GiftCommon:       gift.GiftCommon,
				ReceivedDatetime: nowMillis,
			})
		}
		user.Gifts.NotReceived = remaining
		user.Notifications.GiftNotReceiveCount = int32(len(user.Gifts.NotReceived))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("receive gifts: %w", err)
	}

	return &pb.ReceiveGiftResponse{
		ReceivedGiftUuid: received,
		ExpiredGiftUuid:  expired,
		OverflowGiftUuid: overflow,
	}, nil
}

func (s *GiftServiceServer) GetGiftList(ctx context.Context, req *pb.GetGiftListRequest) (*pb.GetGiftListResponse, error) {
	log.Printf("[GiftService] GetGiftList: rewardKinds=%v expirationType=%d ascending=%v nextCursor=%d previousCursor=%d getCount=%d",
		req.RewardKindType, req.ExpirationType, req.IsAscendingSort, req.NextCursor, req.PreviousCursor, req.GetCount)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	user, err := s.users.LoadUser(userId)
	if err != nil {
		return nil, fmt.Errorf("snapshot user: %w", err)
	}

	if req.NextCursor < 0 || req.PreviousCursor < 0 || req.GetCount < 0 {
		return nil, status.Error(codes.InvalidArgument, "cursors and getCount must not be negative")
	}
	if req.NextCursor > 0 && req.PreviousCursor > 0 {
		return nil, status.Error(codes.InvalidArgument, "nextCursor and previousCursor cannot both be set")
	}

	nowMillis := gametime.NowMillis()
	config := s.holder.Get().GameConfig
	gifts := make([]store.NotReceivedGiftState, 0, len(user.Gifts.NotReceived))
	for _, gift := range user.Gifts.NotReceived {
		if isExpiredGift(gift, nowMillis) ||
			!matchesGiftExpirationFilter(gift, req.ExpirationType) ||
			!matchesGiftRewardKindFilter(gift.GiftCommon, req.RewardKindType, config) {
			continue
		}
		gifts = append(gifts, gift)
	}
	sort.Slice(gifts, func(i, j int) bool {
		left, right := gifts[i].ExpirationDatetime, gifts[j].ExpirationDatetime
		if left == right {
			if gifts[i].GiftCommon.GrantDatetime == gifts[j].GiftCommon.GrantDatetime {
				return gifts[i].UserGiftUuid < gifts[j].UserGiftUuid
			}
			if req.IsAscendingSort {
				return gifts[i].GiftCommon.GrantDatetime < gifts[j].GiftCommon.GrantDatetime
			}
			return gifts[i].GiftCommon.GrantDatetime > gifts[j].GiftCommon.GrantDatetime
		}
		if req.IsAscendingSort {
			return left < right
		}
		return left > right
	})

	currentPage := int64(1)
	if req.NextCursor > 0 {
		currentPage = req.NextCursor
	} else if req.PreviousCursor > 0 {
		currentPage = req.PreviousCursor
	}
	start, end, totalPages, err := giftPageRange(len(gifts), req.GetCount, currentPage)
	if err != nil {
		return nil, err
	}
	gifts = gifts[start:end]

	items := make([]*pb.NotReceivedGift, 0, len(gifts))
	for _, gift := range gifts {
		items = append(items, &pb.NotReceivedGift{
			GiftCommon:         toProtoGiftCommon(gift.GiftCommon),
			ExpirationDatetime: timestamppb.New(time.UnixMilli(gift.ExpirationDatetime)),
			UserGiftUuid:       gift.UserGiftUuid,
		})
	}

	var nextCursor, previousCursor int64
	if currentPage < int64(totalPages) {
		nextCursor = currentPage + 1
	}
	if currentPage > 1 && totalPages > 0 {
		previousCursor = currentPage - 1
	}
	return &pb.GetGiftListResponse{
		Gift:           items,
		TotalPageCount: totalPages,
		NextCursor:     nextCursor,
		PreviousCursor: previousCursor,
	}, nil
}

func isExpiredGift(gift store.NotReceivedGiftState, nowMillis int64) bool {
	return gift.ExpirationDatetime != 0 && gift.ExpirationDatetime <= nowMillis
}

func claimableGiftCount(gifts []store.NotReceivedGiftState, nowMillis int64) int32 {
	var count int32
	for _, gift := range gifts {
		if !isExpiredGift(gift, nowMillis) {
			count++
		}
	}
	return count
}

func grantGift(user *store.UserState, gift store.GiftCommonState, granter *store.PossessionGranter, config *masterdata.GameConfig, nowMillis int64) store.GrantResult {
	before := *user
	candidate := store.CloneUserState(*user)
	result := granter.GrantFull(&candidate, model.PossessionType(gift.PossessionType), gift.PossessionId, gift.Count, nowMillis)
	if result.Status != store.GrantStatusGranted {
		return result
	}
	if exceedsPossessionLimits(before, candidate, config) {
		result.Status = store.GrantStatusOverflow
		return result
	}
	*user = candidate
	return result
}

func exceedsPossessionLimits(before, after store.UserState, config *masterdata.GameConfig) bool {
	if config == nil {
		return false
	}
	for id, count := range after.ConsumableItems {
		limit := config.PossessionCountLimitConsumableItem
		if id == config.ConsumableItemIdForGold {
			limit = config.PossessionCountLimitMoney
		}
		if exceedsCountLimit(before.ConsumableItems[id], count, limit) {
			return true
		}
	}
	for id, count := range after.Materials {
		if exceedsCountLimit(before.Materials[id], count, config.PossessionCountLimitMaterial) {
			return true
		}
	}
	for id, count := range after.ImportantItems {
		if exceedsCountLimit(before.ImportantItems[id], count, config.PossessionCountLimitImportantItem) {
			return true
		}
	}
	return exceedsCountLimit(len(before.Weapons), len(after.Weapons), int(config.PossessionCountLimitWeapon)) ||
		exceedsCountLimit(len(before.Parts), len(after.Parts), int(config.PossessionCountLimitParts))
}

func exceedsCountLimit[T ~int | ~int32](before, after, limit T) bool {
	return limit > 0 && after > limit && after > before
}

func matchesGiftExpirationFilter(gift store.NotReceivedGiftState, filter int32) bool {
	switch filter {
	case 0, giftExpirationFilterNone:
		return true
	case giftExpirationFilterOnlyExpire:
		return gift.ExpirationDatetime != 0
	case giftExpirationFilterOnlyNotExpire:
		return gift.ExpirationDatetime == 0
	default:
		return false
	}
}

func matchesGiftRewardKindFilter(gift store.GiftCommonState, filters []int32, config *masterdata.GameConfig) bool {
	if len(filters) == 0 {
		return true
	}
	kind := giftRewardKind(gift, config)
	for _, filter := range filters {
		if filter == 0 || filter == giftRewardKindNone || filter == kind {
			return true
		}
	}
	return false
}

func giftRewardKind(gift store.GiftCommonState, config *masterdata.GameConfig) int32 {
	possessionType := model.PossessionType(gift.PossessionType)
	switch possessionType {
	case model.PossessionTypePaidGem, model.PossessionTypeFreeGem:
		return giftRewardKindGem
	case model.PossessionTypeWeapon, model.PossessionTypeWeaponEnhanced:
		return giftRewardKindWeapon
	case model.PossessionTypeCompanion, model.PossessionTypeCompanionEnhanced:
		return giftRewardKindCompanion
	case model.PossessionTypeParts, model.PossessionTypePartsEnhanced:
		return giftRewardKindParts
	case model.PossessionTypeMaterial:
		return giftRewardKindMaterial
	case model.PossessionTypeCostume, model.PossessionTypeCostumeEnhanced:
		return giftRewardKindCostume
	case model.PossessionTypeConsumableItem:
		if config != nil && gift.PossessionId == config.ConsumableItemIdForGold {
			return giftRewardKindGold
		}
	}
	return giftRewardKindOther
}

func (s *GiftServiceServer) GetGiftReceiveHistoryList(ctx context.Context, req *emptypb.Empty) (*pb.GetGiftReceiveHistoryListResponse, error) {
	log.Printf("[GiftService] GetGiftReceiveHistoryList")
	userId := CurrentUserId(ctx, s.users, s.sessions)
	user, err := s.users.LoadUser(userId)
	if err != nil {
		return nil, fmt.Errorf("snapshot user: %w", err)
	}

	items := make([]*pb.ReceivedGift, 0, len(user.Gifts.Received))
	for _, gift := range user.Gifts.Received {
		items = append(items, &pb.ReceivedGift{
			GiftCommon:       toProtoGiftCommon(gift.GiftCommon),
			ReceivedDatetime: timestampOrNilGift(gift.ReceivedDatetime),
		})
	}
	return &pb.GetGiftReceiveHistoryListResponse{
		Gift: items,
	}, nil
}

func toProtoGiftCommon(gift store.GiftCommonState) *pb.GiftCommon {
	return &pb.GiftCommon{
		PossessionType:        gift.PossessionType,
		PossessionId:          gift.PossessionId,
		Count:                 gift.Count,
		GrantDatetime:         timestampOrNilGift(gift.GrantDatetime),
		DescriptionGiftTextId: gift.DescriptionGiftTextId,
		EquipmentData:         gift.EquipmentData,
	}
}

func timestampOrNilGift(unixMillis int64) *timestamppb.Timestamp {
	if unixMillis == 0 {
		return nil
	}
	return timestamppb.New(time.UnixMilli(unixMillis))
}

func pageCount(total, pageSize int) int32 {
	if total == 0 {
		return 0
	}
	if pageSize <= 0 {
		return 1
	}
	pages := total / pageSize
	if total%pageSize != 0 {
		pages++
	}
	return int32(pages)
}

func giftPageRange(total int, pageSize int32, currentPage int64) (start, end int, totalPages int32, err error) {
	totalPages = pageCount(total, int(pageSize))
	if currentPage < 1 {
		return 0, 0, totalPages, status.Error(codes.InvalidArgument, "cursor must identify a positive page")
	}
	if pageSize <= 0 {
		if currentPage != 1 {
			return 0, 0, totalPages, status.Error(codes.InvalidArgument, "cursor is outside the gift list")
		}
		return 0, total, totalPages, nil
	}
	if totalPages == 0 {
		if currentPage != 1 {
			return 0, 0, totalPages, status.Error(codes.InvalidArgument, "cursor is outside the gift list")
		}
		return 0, 0, totalPages, nil
	}
	if currentPage > int64(totalPages) {
		return 0, 0, totalPages, status.Error(codes.InvalidArgument, "cursor is outside the gift list")
	}
	start = int(currentPage-1) * int(pageSize)
	end = min(start+int(pageSize), total)
	return start, end, totalPages, nil
}
