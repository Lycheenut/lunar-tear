package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const missionOptionTitleScreen int32 = 395

type UserServiceServer struct {
	pb.UnimplementedUserServiceServer
	users      store.UserRepository
	sessions   store.SessionRepository
	holder     *runtime.Holder
	authURL    string
	noRegister bool
}

func NewUserServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder, authURL string, noRegister bool) *UserServiceServer {
	if authURL != "" && !strings.Contains(authURL, "://") {
		authURL = "http://" + authURL
	}
	return &UserServiceServer{users: users, sessions: sessions, holder: holder, authURL: authURL, noRegister: noRegister}
}

func (s *UserServiceServer) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.RegisterUserResponse, error) {
	if s.noRegister {
		ip := "invalid"

		if p, ok := peer.FromContext(ctx); ok {
			ip = p.Addr.String()
		}

		return nil, fmt.Errorf("Denied user registration: ip=%s uuid=%s", ip, req.Uuid)
	}

	platform := model.ClientPlatformFromContext(ctx)
	userId, err := s.users.CreateUser(req.Uuid, platform)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	log.Printf("[UserService] RegisterUser: uuid=%s terminalId=%s platform=%s -> userId=%d", req.Uuid, req.TerminalId, platform, userId)

	return &pb.RegisterUserResponse{
		UserId:    userId,
		Signature: fmt.Sprintf("sig_%d_%d", userId, gametime.Now().Unix()),
	}, nil
}

func (s *UserServiceServer) Auth(ctx context.Context, req *pb.AuthUserRequest) (*pb.AuthUserResponse, error) {
	platform := model.ClientPlatformFromContext(ctx)
	log.Printf("[UserService] Auth: uuid=%s platform=%s", req.Uuid, platform)

	session, err := s.sessions.CreateSession(req.Uuid, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	user, err := s.users.UpdateUser(session.UserId, func(user *store.UserState) {
		advanceLoginState(&user.Login, gametime.NowMillis())
		store.AddMissionCount(user, int32(model.MissionClearConditionTypeTitleTransitionByCount), 1, 0, missionOptionTitleScreen)
	})
	if err != nil {
		return nil, fmt.Errorf("update login state: %w", err)
	}

	return &pb.AuthUserResponse{
		SessionKey:     session.SessionKey,
		ExpireDatetime: timestamppb.New(session.ExpireAt),
		Signature:      req.Signature,
		UserId:         user.UserId,
	}, nil
}

func advanceLoginState(login *store.UserLoginState, nowMillis int64) {
	today := gametime.StartOfBusinessDayAtMillis(nowMillis)
	lastDay := gametime.StartOfBusinessDayAtMillis(login.LastLoginDatetime)

	if login.LastLoginDatetime == 0 {
		login.TotalLoginCount = 1
		login.ContinualLoginCount = 1
		login.MaxContinualLoginCount = max(login.MaxContinualLoginCount, 1)
	} else if today > lastDay {
		login.TotalLoginCount++
		if today-lastDay == int64(24*time.Hour/time.Millisecond) {
			login.ContinualLoginCount++
		} else {
			login.ContinualLoginCount = 1
		}
		login.MaxContinualLoginCount = max(login.MaxContinualLoginCount, login.ContinualLoginCount)
	}
	if nowMillis > login.LastLoginDatetime {
		login.LastLoginDatetime = nowMillis
	}
	login.LatestVersion = nowMillis
}

func (s *UserServiceServer) GameStart(ctx context.Context, _ *emptypb.Empty) (*pb.GameStartResponse, error) {
	log.Printf("[UserService] GameStart")

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-apb-session-key"); len(vals) > 0 {
			log.Printf("[UserService] GameStart session: %s", vals[0])
		}
	}

	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		user.GameStartDatetime = gametime.NowMillis()
	})

	return &pb.GameStartResponse{}, nil
}

func (s *UserServiceServer) TransferUser(ctx context.Context, req *pb.TransferUserRequest) (*pb.TransferUserResponse, error) {
	platform := model.ClientPlatformFromContext(ctx)

	log.Printf("[UserService] TransferUser: platform=%s", platform)

	userId, err := s.users.GetUserByUUID(req.Uuid)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &pb.TransferUserResponse{
		UserId:    userId,
		Signature: "transferred-sig",
	}, nil
}

func (s *UserServiceServer) SetUserName(ctx context.Context, req *pb.SetUserNameRequest) (*pb.SetUserNameResponse, error) {
	log.Printf("[UserService] SetUserName: %s", req.Name)
	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		nowMillis := gametime.NowMillis()
		user.Profile.Name = req.Name
		user.Profile.NameUpdateDatetime = nowMillis
	})
	return &pb.SetUserNameResponse{}, nil
}

func (s *UserServiceServer) SetUserMessage(ctx context.Context, req *pb.SetUserMessageRequest) (*pb.SetUserMessageResponse, error) {
	log.Printf("[UserService] SetUserMessage: %s", req.Message)
	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		nowMillis := gametime.NowMillis()
		user.Profile.Message = req.Message
		user.Profile.MessageUpdateDatetime = nowMillis
		recordUserMessageMissionEvents(user, req.Message)
	})
	return &pb.SetUserMessageResponse{}, nil
}

func recordUserMessageMissionEvents(user *store.UserState, message string) {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return
	}
	// Option 420 is used by missions whose localized text only asks the user
	// to update their profile comment, without prescribing a word.
	store.AddMissionCount(user, int32(model.MissionClearConditionTypeUserMessageMatchWord), 1, 0, 420)
	if strings.Contains(message, "あけましておめでとう") || strings.Contains(message, "お年玉") || strings.Contains(normalized, "a happy new year") {
		store.AddMissionCount(user, int32(model.MissionClearConditionTypeUserMessageMatchWord), 1, 0, 392)
	}
	if strings.Contains(message, "ママ") || strings.Contains(normalized, "mama") {
		store.AddMissionCount(user, int32(model.MissionClearConditionTypeUserMessageMatchWord), 1, 0, 393)
	}
}

func (s *UserServiceServer) SetUserFavoriteCostumeId(ctx context.Context, req *pb.SetUserFavoriteCostumeIdRequest) (*pb.SetUserFavoriteCostumeIdResponse, error) {
	log.Printf("[UserService] SetUserFavoriteCostumeId: %d", req.FavoriteCostumeId)
	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		nowMillis := gametime.NowMillis()
		user.Profile.FavoriteCostumeId = req.FavoriteCostumeId
		user.Profile.FavoriteCostumeIdUpdateDatetime = nowMillis
	})
	return &pb.SetUserFavoriteCostumeIdResponse{}, nil
}

func (s *UserServiceServer) GetUserProfile(ctx context.Context, req *pb.GetUserProfileRequest) (*pb.GetUserProfileResponse, error) {
	log.Printf("[UserService] GetUserProfile: playerId=%d", req.PlayerId)
	requesterUserId := CurrentUserId(ctx, s.users, s.sessions)
	targetUserId := requesterUserId
	if req.PlayerId != 0 {
		var err error
		targetUserId, err = s.users.GetUserByPlayerId(req.PlayerId)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, status.Error(codes.NotFound, "player not found")
			}
			return nil, fmt.Errorf("look up player: %w", err)
		}
	}
	user, err := s.users.LoadUser(targetUserId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "player not found")
		}
		return nil, fmt.Errorf("load player: %w", err)
	}

	isFriend := false
	if requesterUserId != 0 && requesterUserId != targetUserId {
		requester, loadErr := s.users.LoadUser(requesterUserId)
		if loadErr == nil {
			isFriend = requester.Friends[targetUserId].IsFriend
		}
	}

	return buildUserProfile(user, s.holder.Get(), isFriend), nil
}

func buildUserProfile(user store.UserState, catalogs *runtime.Catalogs, isFriend bool) *pb.GetUserProfileResponse {
	deck, _ := latestUsedQuestDeck(user)
	deckCharacters := make([]*pb.ProfileDeckCharacter, 0, 3)
	for _, deckCharacterUuid := range []string{
		deck.UserDeckCharacterUuid01,
		deck.UserDeckCharacterUuid02,
		deck.UserDeckCharacterUuid03,
	} {
		deckCharacter, ok := user.DeckCharacters[deckCharacterUuid]
		if !ok || deckCharacterUuid == "" {
			continue
		}
		costume := user.Costumes[deckCharacter.UserCostumeUuid]
		weapon := user.Weapons[deckCharacter.MainUserWeaponUuid]
		deckCharacters = append(deckCharacters, &pb.ProfileDeckCharacter{
			CostumeId:       costume.CostumeId,
			MainWeaponId:    weapon.WeaponId,
			MainWeaponLevel: weapon.Level,
		})
	}

	return &pb.GetUserProfileResponse{
		Level:             user.Status.Level,
		Name:              user.Profile.Name,
		FavoriteCostumeId: user.Profile.FavoriteCostumeId,
		Message:           user.Profile.Message,
		IsFriend:          isFriend,
		LatestUsedDeck: &pb.ProfileDeck{
			Power:         deck.Power,
			DeckCharacter: deckCharacters,
		},
		PvpInfo: &pb.ProfilePvpInfo{
			CurrentRank:    user.Profile.CurrentPvpRank,
			CurrentGradeId: user.Profile.CurrentPvpGradeId,
			MaxSeasonRank:  user.Profile.MaxPvpSeasonRank,
		},
		GamePlayHistory: buildGamePlayHistory(user, catalogs),
	}
}

func latestUsedQuestDeck(user store.UserState) (store.DeckState, bool) {
	deckNumber := int32(0)
	latestStart := int64(-1)
	for _, quest := range user.Quests {
		if quest.UserDeckNumber > 0 && quest.LatestStartDatetime > latestStart {
			deckNumber = quest.UserDeckNumber
			latestStart = quest.LatestStartDatetime
		}
	}
	if deckNumber > 0 {
		if deck, ok := user.Decks[store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: deckNumber}]; ok {
			return deck, true
		}
	}
	if deck, ok := user.Decks[store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: 1}]; ok {
		return deck, true
	}
	var selected store.DeckState
	found := false
	selectedNumber := int32(0)
	for key, deck := range user.Decks {
		if key.DeckType != model.DeckTypeQuest || found && key.UserDeckNumber >= selectedNumber {
			continue
		}
		selected = deck
		selectedNumber = key.UserDeckNumber
		found = true
	}
	return selected, found
}

const (
	playHistoryTotalLogin int32 = iota + 1
	playHistoryQuestClear
	playHistoryBattleFinish
	playHistoryCostumeOwned
	playHistoryWeaponOwned
	playHistoryCompanionOwned
	playHistoryPartsOwned
	playHistoryGachaDraw
	playHistoryMissionClear
)

func buildGamePlayHistory(user store.UserState, catalogs *runtime.Catalogs) *pb.GamePlayHistory {
	items := make([]*pb.PlayHistoryItem, 0, 9)
	for id := playHistoryTotalLogin; id <= playHistoryMissionClear; id++ {
		items = append(items, &pb.PlayHistoryItem{HistoryItemId: id, Count: gamePlayHistoryValue(user, id)})
	}

	questTotal, questMissionTotal, costumeTotal, weaponTotal, companionTotal := 0, 0, 0, 0, 0
	if catalogs != nil {
		if catalogs.Quest != nil {
			questTotal = len(catalogs.Quest.QuestById)
			questMissionTotal = len(catalogs.Quest.MissionById)
		}
		if catalogs.Costume != nil {
			costumeTotal = len(catalogs.Costume.Costumes)
		}
		if catalogs.Weapon != nil {
			weaponTotal = len(catalogs.Weapon.Weapons)
		}
		if catalogs.Companion != nil {
			companionTotal = len(catalogs.Companion.CompanionById)
		}
	}

	clearedQuests := 0
	for _, quest := range user.Quests {
		if quest.QuestStateType == model.UserQuestStateTypeCleared {
			clearedQuests++
		}
	}
	clearedQuestMissions := 0
	for _, mission := range user.QuestMissions {
		if mission.IsClear {
			clearedQuestMissions++
		}
	}
	ownedWeaponIds := make(map[int32]struct{}, len(user.Weapons))
	for _, weapon := range user.Weapons {
		ownedWeaponIds[weapon.WeaponId] = struct{}{}
	}
	graph := []*pb.PlayHistoryCategoryGraphItem{
		{CategoryTypeId: 1, ProgressPermil: progressPermil(clearedQuests, questTotal)},
		{CategoryTypeId: 2, ProgressPermil: progressPermil(clearedQuestMissions, questMissionTotal)},
		{CategoryTypeId: 3, ProgressPermil: progressPermil(len(user.Costumes), costumeTotal)},
		{CategoryTypeId: 4, ProgressPermil: progressPermil(len(ownedWeaponIds), weaponTotal)},
		{CategoryTypeId: 5, ProgressPermil: progressPermil(len(user.Companions), companionTotal)},
	}
	return &pb.GamePlayHistory{HistoryItem: items, HistoryCategoryGraphItem: graph}
}

func gamePlayHistoryValue(user store.UserState, historyTypeId int32) int64 {
	switch historyTypeId {
	case playHistoryTotalLogin:
		return int64(user.Login.TotalLoginCount)
	case playHistoryQuestClear:
		var count int64
		for _, quest := range user.Quests {
			count += int64(quest.ClearCount)
		}
		return count
	case playHistoryBattleFinish:
		return int64(user.Battle.FinishCount)
	case playHistoryCostumeOwned:
		return int64(len(user.Costumes))
	case playHistoryWeaponOwned:
		return int64(len(user.Weapons))
	case playHistoryCompanionOwned:
		return int64(len(user.Companions))
	case playHistoryPartsOwned:
		return int64(len(user.Parts))
	case playHistoryGachaDraw:
		var count int64
		for _, banner := range user.Gacha.BannerStates {
			count += int64(banner.DrawCount)
		}
		return count
	case playHistoryMissionClear:
		var count int64
		for _, mission := range user.Missions {
			if mission.MissionProgressStatusType >= int32(model.MissionProgressStatusTypeClear) {
				count++
			}
		}
		return count
	default:
		return 0
	}
}

func progressPermil(current, total int) int32 {
	if current <= 0 || total <= 0 {
		return 0
	}
	return int32(min(1000, current*1000/total))
}

func (s *UserServiceServer) SetBirthYearMonth(ctx context.Context, req *pb.SetBirthYearMonthRequest) (*pb.SetBirthYearMonthResponse, error) {
	log.Printf("[UserService] SetBirthYearMonth: %d/%d", req.BirthYear, req.BirthMonth)
	userId := CurrentUserId(ctx, s.users, s.sessions)
	_, _ = s.users.UpdateUser(userId, func(user *store.UserState) {
		user.BirthYear = req.BirthYear
		user.BirthMonth = req.BirthMonth
	})
	return &pb.SetBirthYearMonthResponse{}, nil
}

func (s *UserServiceServer) GetBirthYearMonth(ctx context.Context, _ *emptypb.Empty) (*pb.GetBirthYearMonthResponse, error) {
	userId := CurrentUserId(ctx, s.users, s.sessions)
	user, err := s.users.LoadUser(userId)
	if err != nil {
		return &pb.GetBirthYearMonthResponse{BirthYear: 2000, BirthMonth: 1}, nil
	}
	return &pb.GetBirthYearMonthResponse{BirthYear: user.BirthYear, BirthMonth: user.BirthMonth}, nil
}

func (s *UserServiceServer) GetChargeMoney(ctx context.Context, _ *emptypb.Empty) (*pb.GetChargeMoneyResponse, error) {
	userId := CurrentUserId(ctx, s.users, s.sessions)
	user, err := s.users.LoadUser(userId)
	if err != nil {
		return &pb.GetChargeMoneyResponse{ChargeMoneyThisMonth: 0}, nil
	}
	return &pb.GetChargeMoneyResponse{ChargeMoneyThisMonth: user.ChargeMoneyThisMonth}, nil
}

func (s *UserServiceServer) SetUserSetting(ctx context.Context, req *pb.SetUserSettingRequest) (*pb.SetUserSettingResponse, error) {
	log.Printf("[UserService] SetUserSetting: isNotifyPurchaseAlert=%v", req.IsNotifyPurchaseAlert)
	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		user.Setting.IsNotifyPurchaseAlert = req.IsNotifyPurchaseAlert
	})
	return &pb.SetUserSettingResponse{}, nil
}

func (s *UserServiceServer) GetAndroidArgs(ctx context.Context, req *pb.GetAndroidArgsRequest) (*pb.GetAndroidArgsResponse, error) {
	return &pb.GetAndroidArgsResponse{Nonce: "Mama", ApiKey: "1234567890"}, nil
}

func (s *UserServiceServer) GetBackupToken(ctx context.Context, req *pb.GetBackupTokenRequest) (*pb.GetBackupTokenResponse, error) {
	userId := CurrentUserId(ctx, s.users, s.sessions)
	user, err := s.users.LoadUser(userId)
	if err != nil {
		return &pb.GetBackupTokenResponse{BackupToken: "mock-backup-token"}, nil
	}
	return &pb.GetBackupTokenResponse{BackupToken: user.BackupToken}, nil
}

func (s *UserServiceServer) CheckTransferSetting(ctx context.Context, _ *emptypb.Empty) (*pb.CheckTransferSettingResponse, error) {
	return &pb.CheckTransferSettingResponse{}, nil
}

func (s *UserServiceServer) GetUserGamePlayNote(ctx context.Context, req *pb.GetUserGamePlayNoteRequest) (*pb.GetUserGamePlayNoteResponse, error) {
	userId := CurrentUserId(ctx, s.users, s.sessions)
	user, err := s.users.LoadUser(userId)
	if err != nil {
		return nil, fmt.Errorf("load user: %w", err)
	}
	value := gamePlayHistoryValue(user, req.GamePlayHistoryTypeId)
	if value > int64(^uint32(0)>>1) {
		value = int64(^uint32(0) >> 1)
	}
	return &pb.GetUserGamePlayNoteResponse{ProgressValue: int32(value)}, nil
}

func (s *UserServiceServer) resolveAuthToken(token string) (facebookId int64, err error) {
	if s.authURL == "" {
		return 0, status.Error(codes.FailedPrecondition, "auth server not configured (--auth-url)")
	}

	resp, err := http.Get(s.authURL + "/me?access_token=" + token)
	if err != nil {
		return 0, status.Errorf(codes.Internal, "auth server unreachable: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	var body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, status.Errorf(codes.Internal, "decode auth response: %v", err)
	}
	if body.ID == "" {
		return 0, status.Error(codes.Unauthenticated, "auth server returned empty id")
	}

	id, err := strconv.ParseInt(body.ID, 10, 64)
	if err != nil {
		return 0, status.Errorf(codes.Internal, "invalid auth id %q: %v", body.ID, err)
	}
	return id, nil
}

func (s *UserServiceServer) SetFacebookAccount(ctx context.Context, req *pb.SetFacebookAccountRequest) (*pb.SetFacebookAccountResponse, error) {
	log.Printf("[UserService] SetFacebookAccount")

	fbId, err := s.resolveAuthToken(req.Token)
	if err != nil {
		return nil, err
	}

	userId := CurrentUserId(ctx, s.users, s.sessions)
	if err := s.users.SetFacebookId(userId, fbId); err != nil {
		return nil, fmt.Errorf("set facebook id: %w", err)
	}
	log.Printf("[UserService] linked facebook_id=%d to user_id=%d", fbId, userId)
	return &pb.SetFacebookAccountResponse{}, nil
}

func (s *UserServiceServer) UnsetFacebookAccount(ctx context.Context, _ *emptypb.Empty) (*pb.UnsetFacebookAccountResponse, error) {
	log.Printf("[UserService] UnsetFacebookAccount")

	userId := CurrentUserId(ctx, s.users, s.sessions)
	if err := s.users.ClearFacebookId(userId); err != nil {
		return nil, fmt.Errorf("clear facebook id: %w", err)
	}
	log.Printf("[UserService] unlinked facebook from user_id=%d", userId)
	return &pb.UnsetFacebookAccountResponse{}, nil
}

func (s *UserServiceServer) TransferUserByFacebook(ctx context.Context, req *pb.TransferUserByFacebookRequest) (*pb.TransferUserByFacebookResponse, error) {
	log.Printf("[UserService] TransferUserByFacebook: uuid=%s", req.Uuid)

	fbId, err := s.resolveAuthToken(req.Token)
	if err != nil {
		return nil, err
	}

	userId, err := s.users.GetUserByFacebookId(fbId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "no account linked to this login")
	}

	if err := s.users.UpdateUUID(userId, req.Uuid); err != nil {
		return nil, fmt.Errorf("update uuid: %w", err)
	}

	log.Printf("[UserService] transferred facebook_id=%d -> user_id=%d with new uuid=%s", fbId, userId, req.Uuid)

	return &pb.TransferUserByFacebookResponse{
		UserId:    userId,
		Signature: fmt.Sprintf("fb_transfer_%d_%d", userId, gametime.Now().Unix()),
	}, nil
}
