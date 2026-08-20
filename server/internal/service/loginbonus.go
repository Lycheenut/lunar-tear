package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/giftbox"
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

const (
	loginBonusStartConditionAll            = int32(0)
	loginBonusStartConditionComeback       = int32(4)
	loginBonusStartConditionBeginner       = int32(5)
	loginBonusStartConditionComebackGrade1 = int32(6)
)

var errLoginBonusExhausted = errors.New("login bonus exhausted")

func NewLoginBonusServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *LoginBonusServiceServer {
	return &LoginBonusServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *LoginBonusServiceServer) ReceiveStamp(ctx context.Context, req *emptypb.Empty) (*pb.ReceiveStampResponse, error) {
	log.Printf("[LoginBonusService] ReceiveStamp")
	userId := CurrentUserId(ctx, s.users, s.sessions)
	catalogs := s.holder.Get()
	var receipts []loginBonusReceipt
	_, err := s.users.UpdateUsers([]int64{userId}, func(users map[int64]*store.UserState) error {
		nowMillis := gametime.NowMillis()
		user := users[userId]
		ensureBeginnerCampaign(catalogs.Campaign, user, nowMillis)
		ensureComebackCampaign(catalogs.Campaign, user, nowMillis)
		syncLoginBonuses(catalogs.LoginBonus, catalogs.Campaign, user, nowMillis, false)
		var applyErr error
		receipts, applyErr = applyLoginBonusStamps(catalogs.LoginBonus, user, catalogs.GameConfig, nowMillis)
		return applyErr
	})
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	if len(receipts) == 0 {
		log.Printf("[LoginBonusService] no receivable stamps userId=%d", userId)
		return &pb.ReceiveStampResponse{}, nil
	}

	for _, receipt := range receipts {
		log.Printf("[LoginBonusService] bonusId=%d page %d->%d stamp %d->%d possType=%d possId=%d count=%d (-> gift box)",
			receipt.bonusId, receipt.oldPage, receipt.nextPage, receipt.oldStamp, receipt.nextStamp,
			receipt.reward.PossessionType, receipt.reward.PossessionId, receipt.reward.Count)
	}

	return &pb.ReceiveStampResponse{}, nil
}

type loginBonusReceipt struct {
	bonusId             int32
	oldPage, nextPage   int32
	oldStamp, nextStamp int32
	reward              masterdata.LoginBonusReward
}

func applyLoginBonusStamps(catalog *masterdata.LoginBonusCatalog, user *store.UserState, config *masterdata.GameConfig, nowMillis int64) ([]loginBonusReceipt, error) {
	receipts := make([]loginBonusReceipt, 0)
	for _, definition := range catalog.ActiveDefinitions(nowMillis) {
		lb, ok := user.LoginBonuses[definition.LoginBonusId]
		if !ok || isLoginBonusStampReceivedToday(lb, nowMillis) {
			continue
		}
		nextPage, nextStamp, reward, err := resolveNextStamp(catalog, lb)
		if errors.Is(err, errLoginBonusExhausted) {
			continue
		}
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, loginBonusReceipt{
			bonusId:   lb.LoginBonusId,
			oldPage:   lb.CurrentPageNumber,
			nextPage:  nextPage,
			oldStamp:  lb.CurrentStampNumber,
			nextStamp: nextStamp,
			reward:    reward,
		})
		giftbox.AddNotReceived(user, store.NotReceivedGiftState{
			GiftCommon: store.GiftCommonState{
				PossessionType: reward.PossessionType,
				PossessionId:   reward.PossessionId,
				Count:          reward.Count,
				GrantDatetime:  nowMillis,
			},
			ExpirationDatetime: nowMillis + int64(30*24*time.Hour/time.Millisecond),
		}, config)
		lb.CurrentPageNumber = nextPage
		lb.CurrentStampNumber = nextStamp
		lb.LatestRewardReceiveDatetime = nowMillis
		lb.LatestVersion = nowMillis
		user.LoginBonuses[lb.LoginBonusId] = lb
	}
	user.Notifications.GiftNotReceiveCount = int32(len(user.Gifts.NotReceived))
	return receipts, nil
}

func syncLoginBonuses(loginBonuses *masterdata.LoginBonusCatalog, campaigns *campaign.Catalog, user *store.UserState, nowMillis int64, resetComeback bool) {
	user.EnsureMaps()
	for _, definition := range loginBonuses.ActiveDefinitions(nowMillis) {
		if !loginBonusStartConditionEligible(definition.LoginBonusStartConditionId, campaigns, user, nowMillis) {
			continue
		}
		_, exists := user.LoginBonuses[definition.LoginBonusId]
		reset := resetComeback && (definition.LoginBonusStartConditionId == loginBonusStartConditionComeback ||
			definition.LoginBonusStartConditionId == loginBonusStartConditionComebackGrade1)
		if exists && !reset {
			continue
		}
		user.LoginBonuses[definition.LoginBonusId] = store.UserLoginBonusState{
			LoginBonusId:      definition.LoginBonusId,
			CurrentPageNumber: 1,
			LatestVersion:     nowMillis,
		}
	}
}

func loginBonusStartConditionEligible(conditionId int32, campaigns *campaign.Catalog, user *store.UserState, nowMillis int64) bool {
	if conditionId == loginBonusStartConditionAll {
		return true
	}
	if campaigns == nil {
		return false
	}
	cleared := campaignQuestCleared(user)
	switch conditionId {
	case loginBonusStartConditionComeback:
		return campaigns.IsComebackEnrollmentActive(
			user.ComebackCampaign.ComebackCampaignId, user.ComebackCampaign.ComebackDatetime, nowMillis, cleared,
		)
	case loginBonusStartConditionBeginner:
		return campaigns.IsBeginnerEnrollmentActive(
			user.BeginnerCampaign.BeginnerCampaignId, user.BeginnerCampaign.CampaignRegisterDatetime, nowMillis, cleared,
		)
	case loginBonusStartConditionComebackGrade1:
		return campaigns.IsComebackGradeGroupActive(
			user.ComebackCampaign.ComebackCampaignId, user.ComebackCampaign.ComebackDatetime, nowMillis, 1, cleared,
		)
	default:
		return false
	}
}

func isLoginBonusStampReceivedToday(lb store.UserLoginBonusState, nowMillis int64) bool {
	return lb.LatestRewardReceiveDatetime >= gametime.StartOfBusinessDayAtMillis(nowMillis) &&
		lb.LatestRewardReceiveDatetime <= nowMillis
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
			err = fmt.Errorf("%w: login bonus %d page %d stamp %d is the last",
				errLoginBonusExhausted, bonusId, curPage, curStamp)
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
