package service

import (
	"context"
	"fmt"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

type BannerServiceServer struct {
	pb.UnimplementedBannerServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewBannerServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *BannerServiceServer {
	return &BannerServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *BannerServiceServer) GetMamaBanner(ctx context.Context, req *pb.GetMamaBannerRequest) (*pb.GetMamaBannerResponse, error) {
	cageUpdate, err := parseCageMeasurableValues(req.CageMeasurableValues)
	if err != nil {
		return nil, err
	}
	catalogs := s.holder.Get()
	nowMillis := gametime.NowMillis()
	if cageUpdate.present {
		userId := CurrentUserId(ctx, s.users, s.sessions)
		if _, err := s.users.UpdateUser(userId, func(user *store.UserState) {
			cageUpdate.apply(catalogs, user, nowMillis)
		}); err != nil {
			return nil, fmt.Errorf("sync cage measurable values: %w", err)
		}
	}

	catalog := catalogs.GachaEntries
	var termLimited []*pb.GachaBanner
	var latestChapter *pb.GachaBanner
	for _, entry := range catalog {
		if !entry.IsMamaBanner || !gachaActiveAt(entry, nowMillis) {
			continue
		}
		if entry.GachaLabelType == model.GachaLabelPortalCage || entry.GachaLabelType == model.GachaLabelRecycle {
			continue
		}
		b := &pb.GachaBanner{
			GachaLabelType: entry.GachaLabelType,
			GachaAssetName: entry.BannerAssetName,
			GachaId:        entry.GachaId,
		}
		switch entry.GachaLabelType {
		case model.GachaLabelEvent, model.GachaLabelPremium:
			termLimited = append(termLimited, b)
		case model.GachaLabelChapter:
			if latestChapter == nil || entry.GachaId > latestChapter.GachaId {
				latestChapter = b
			}
		}
	}
	return &pb.GetMamaBannerResponse{
		TermLimitedGacha:   termLimited,
		LatestChapterGacha: latestChapter,
		IsExistUnreadPop:   false,
	}, nil
}
