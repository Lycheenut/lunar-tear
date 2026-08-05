package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type LoginBonusServiceServer struct {
	pb.UnimplementedLoginBonusServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewLoginBonusServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *LoginBonusServiceServer {
	return &LoginBonusServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *LoginBonusServiceServer) ReceiveStamp(ctx context.Context, req *emptypb.Empty) (*pb.ReceiveStampResponse, error) {
	log.Printf("[LoginBonusService] ReceiveStamp")
	userId := CurrentUserId(ctx, s.users, s.sessions)
	catalog := s.holder.Get().LoginBonus
	var receipt loginBonusReceipt
	_, err := s.users.UpdateUsers([]int64{userId}, func(users map[int64]*store.UserState) error {
		var applyErr error
		receipt, applyErr = applyLoginBonusStamp(catalog, users[userId], gametime.NowMillis())
		return applyErr
	})
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	log.Printf("[LoginBonusService] bonusId=%d page %d->%d stamp %d->%d possType=%d possId=%d count=%d (-> gift box)",
		receipt.bonusId, receipt.oldPage, receipt.nextPage, receipt.oldStamp, receipt.nextStamp,
		receipt.reward.PossessionType, receipt.reward.PossessionId, receipt.reward.Count)

	return &pb.ReceiveStampResponse{}, nil
}

type loginBonusReceipt struct {
	bonusId             int32
	oldPage, nextPage   int32
	oldStamp, nextStamp int32
	reward              masterdata.LoginBonusReward
}

func applyLoginBonusStamp(catalog *masterdata.LoginBonusCatalog, user *store.UserState, nowMillis int64) (loginBonusReceipt, error) {
	if err := validateLoginBonusReceive(catalog, user.LoginBonus, nowMillis); err != nil {
		return loginBonusReceipt{}, err
	}
	nextPage, nextStamp, reward, err := resolveNextStamp(catalog, user.LoginBonus)
	if err != nil {
		return loginBonusReceipt{}, err
	}
	receipt := loginBonusReceipt{
		bonusId:   user.LoginBonus.LoginBonusId,
		oldPage:   user.LoginBonus.CurrentPageNumber,
		nextPage:  nextPage,
		oldStamp:  user.LoginBonus.CurrentStampNumber,
		nextStamp: nextStamp,
		reward:    reward,
	}
	user.Gifts.NotReceived = append(user.Gifts.NotReceived, store.NotReceivedGiftState{
		GiftCommon: store.GiftCommonState{
			PossessionType: reward.PossessionType,
			PossessionId:   reward.PossessionId,
			Count:          reward.Count,
			GrantDatetime:  nowMillis,
		},
		ExpirationDatetime: nowMillis + int64(30*24*time.Hour/time.Millisecond),
		UserGiftUuid:       uuid.New().String(),
	})
	user.Notifications.GiftNotReceiveCount = int32(len(user.Gifts.NotReceived))
	user.LoginBonus.CurrentPageNumber = nextPage
	user.LoginBonus.CurrentStampNumber = nextStamp
	user.LoginBonus.LatestRewardReceiveDatetime = nowMillis
	user.LoginBonus.LatestVersion = nowMillis
	return receipt, nil
}

func validateLoginBonusReceive(catalog *masterdata.LoginBonusCatalog, lb store.UserLoginBonusState, nowMillis int64) error {
	term, ok := catalog.LookupTerm(lb.LoginBonusId)
	if !ok {
		return status.Errorf(codes.FailedPrecondition, "login bonus %d is not configured", lb.LoginBonusId)
	}
	return validateLoginBonusTerm(term, lb, nowMillis)
}

func validateLoginBonusTerm(term masterdata.LoginBonusTerm, lb store.UserLoginBonusState, nowMillis int64) error {
	if term.StartDatetime != 0 && nowMillis < term.StartDatetime {
		return status.Errorf(codes.FailedPrecondition, "login bonus %d has not started", lb.LoginBonusId)
	}
	if term.EndDatetime != 0 && nowMillis >= term.EndDatetime {
		return status.Errorf(codes.FailedPrecondition, "login bonus %d has ended", lb.LoginBonusId)
	}
	if term.StampReceiveEndDatetime != 0 && nowMillis >= term.StampReceiveEndDatetime {
		return status.Errorf(codes.FailedPrecondition, "login bonus %d stamp receiving has ended", lb.LoginBonusId)
	}
	if lb.LatestRewardReceiveDatetime >= gametime.StartOfBusinessDayAtMillis(nowMillis) {
		return status.Error(codes.FailedPrecondition, "login bonus stamp already received today")
	}
	return nil
}

func resolveNextStamp(catalog *masterdata.LoginBonusCatalog, lb store.UserLoginBonusState) (nextPage, nextStamp int32, reward masterdata.LoginBonusReward, err error) {
	bonusId := lb.LoginBonusId
	curPage := lb.CurrentPageNumber
	curStamp := lb.CurrentStampNumber

	nextPage = curPage
	nextStamp = curStamp + 1
	var ok bool
	reward, ok = catalog.LookupStampReward(bonusId, nextPage, nextStamp)
	if !ok {
		nextPage = curPage + 1
		nextStamp = 1
		total := catalog.TotalPageCount(bonusId)
		if total > 0 && nextPage > total {
			err = status.Errorf(codes.FailedPrecondition,
				"login bonus %d exhausted (page %d stamp %d is the last)",
				bonusId, curPage, curStamp)
			return
		}
		reward, ok = catalog.LookupStampReward(bonusId, nextPage, nextStamp)
		if !ok {
			err = status.Errorf(codes.FailedPrecondition,
				"no reward found for login bonus %d page %d stamp %d",
				bonusId, nextPage, nextStamp)
			return
		}
	}
	return
}
