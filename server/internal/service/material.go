package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

type MaterialServiceServer struct {
	pb.UnimplementedMaterialServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewMaterialServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *MaterialServiceServer {
	return &MaterialServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *MaterialServiceServer) Sell(ctx context.Context, req *pb.MaterialSellRequest) (*pb.MaterialSellResponse, error) {
	log.Printf("[MaterialService] Sell: %d item(s)", len(req.MaterialPossession))

	cat := s.holder.Get()
	catalog := cat.Material
	config := cat.GameConfig
	userId := CurrentUserId(ctx, s.users, s.sessions)

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		totalGold := int32(0)
		for _, item := range req.MaterialPossession {
			mat, ok := catalog.All[item.MaterialId]
			if !ok {
				log.Printf("[MaterialService] Sell: unknown materialId=%d, skipping", item.MaterialId)
				continue
			}

			cur := user.Materials[item.MaterialId]
			if cur < item.Count {
				log.Printf("[MaterialService] Sell: insufficient materialId=%d have=%d need=%d", item.MaterialId, cur, item.Count)
				continue
			}

			user.Materials[item.MaterialId] -= item.Count
			if user.Materials[item.MaterialId] <= 0 {
				delete(user.Materials, item.MaterialId)
			}

			gold := mat.SellPrice * item.Count
			totalGold += gold
			log.Printf("[MaterialService] Sell: materialId=%d x%d -> %d gold", item.MaterialId, item.Count, gold)

			if mat.MaterialSaleObtainPossessionId != 0 {
				if row, ok := catalog.SaleObtainById[mat.MaterialSaleObtainPossessionId]; ok {
					grantCount := row.Count * item.Count
					store.GrantPossession(user, model.PossessionType(row.PossessionType), row.PossessionId, grantCount)
					log.Printf("[MaterialService] Sell: materialId=%d x%d -> SaleObtain type=%d id=%d +%d", item.MaterialId, item.Count, row.PossessionType, row.PossessionId, grantCount)
				}
			}
		}

		if totalGold > 0 {
			user.ConsumableItems[config.ConsumableItemIdForGold] += totalGold
			log.Printf("[MaterialService] Sell: total gold +%d", totalGold)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("material sell: %w", err)
	}

	return &pb.MaterialSellResponse{}, nil
}
