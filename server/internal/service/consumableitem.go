package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

type ConsumableItemServiceServer struct {
	pb.UnimplementedConsumableItemServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewConsumableItemServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *ConsumableItemServiceServer {
	return &ConsumableItemServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *ConsumableItemServiceServer) UseEffectItem(ctx context.Context, req *pb.ConsumableItemUseEffectItemRequest) (*pb.ConsumableItemUseEffectItemResponse, error) {
	log.Printf("[ConsumableItemService] UseEffectItem: consumableItemId=%d count=%d", req.ConsumableItemId, req.Count)

	cat := s.holder.Get()
	catalog := cat.ConsumableItem
	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		if _, ok := catalog.All[req.ConsumableItemId]; !ok {
			log.Printf("[ConsumableItemService] UseEffectItem: unknown consumableItemId=%d", req.ConsumableItemId)
			return
		}
		cur := user.ConsumableItems[req.ConsumableItemId]
		if cur < req.Count {
			log.Printf("[ConsumableItemService] UseEffectItem: insufficient consumableItemId=%d have=%d need=%d", req.ConsumableItemId, cur, req.Count)
			return
		}

		user.ConsumableItems[req.ConsumableItemId] -= req.Count
		if user.ConsumableItems[req.ConsumableItemId] <= 0 {
			delete(user.ConsumableItems, req.ConsumableItemId)
		}

		maxStaminaMillis := cat.Shop.MaxStaminaMillis[user.Status.Level]
		if effect, ok := catalog.EffectByItemId[req.ConsumableItemId]; ok {
			switch effect.EffectTargetType {
			case model.EffectTargetStaminaRecovery:
				millis := store.ResolveStaminaEffectMillis(effect.EffectValueType, effect.EffectValue, maxStaminaMillis)
				store.RecoverStamina(user, millis*req.Count, maxStaminaMillis, nowMillis)
			default:
				log.Printf("[ConsumableItemService] UseEffectItem: unhandled effect targetType=%d valueType=%d value=%d itemId=%d",
					effect.EffectTargetType, effect.EffectValueType, effect.EffectValue, req.ConsumableItemId)
			}
		}
	})
	if err != nil {
		return nil, fmt.Errorf("consumable item use effect item: %w", err)
	}

	return &pb.ConsumableItemUseEffectItemResponse{}, nil
}

func (s *ConsumableItemServiceServer) Sell(ctx context.Context, req *pb.ConsumableItemSellRequest) (*pb.ConsumableItemSellResponse, error) {
	log.Printf("[ConsumableItemService] Sell: %d item(s)", len(req.ConsumableItemPossession))

	cat := s.holder.Get()
	catalog := cat.ConsumableItem
	config := cat.GameConfig
	userId := CurrentUserId(ctx, s.users, s.sessions)

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		totalGold := int32(0)
		for _, item := range req.ConsumableItemPossession {
			row, ok := catalog.All[item.ConsumableItemId]
			if !ok {
				log.Printf("[ConsumableItemService] Sell: unknown consumableItemId=%d, skipping", item.ConsumableItemId)
				continue
			}

			cur := user.ConsumableItems[item.ConsumableItemId]
			if cur < item.Count {
				log.Printf("[ConsumableItemService] Sell: insufficient consumableItemId=%d have=%d need=%d", item.ConsumableItemId, cur, item.Count)
				continue
			}

			user.ConsumableItems[item.ConsumableItemId] -= item.Count
			if user.ConsumableItems[item.ConsumableItemId] <= 0 {
				delete(user.ConsumableItems, item.ConsumableItemId)
			}

			gold := row.SellPrice * item.Count
			totalGold += gold
			log.Printf("[ConsumableItemService] Sell: consumableItemId=%d x%d -> %d gold", item.ConsumableItemId, item.Count, gold)
		}

		if totalGold > 0 {
			user.ConsumableItems[config.ConsumableItemIdForGold] += totalGold
			log.Printf("[ConsumableItemService] Sell: total gold +%d", totalGold)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("consumable item sell: %w", err)
	}

	return &pb.ConsumableItemSellResponse{}, nil
}
