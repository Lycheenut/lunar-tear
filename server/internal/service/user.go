package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

type UserServiceServer struct {
	pb.UnimplementedUserServiceServer
	users      store.UserRepository
	sessions   store.SessionRepository
	holder     *runtime.Holder
	authURL    string
	authClient *http.Client
	noRegister bool
}

func NewUserServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder, authURL string, noRegister bool) *UserServiceServer {
	if authURL != "" && !strings.Contains(authURL, "://") {
		authURL = "http://" + authURL
	}
	return &UserServiceServer{
		users:      users,
		sessions:   sessions,
		holder:     holder,
		authURL:    strings.TrimRight(authURL, "/"),
		authClient: &http.Client{Timeout: 5 * time.Second},
		noRegister: noRegister,
	}
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
	user, err := s.users.LoadUser(session.UserId)
	if err != nil {
		return nil, fmt.Errorf("load user: %w", err)
	}

	return &pb.AuthUserResponse{
		SessionKey:     session.SessionKey,
		ExpireDatetime: timestamppb.New(session.ExpireAt),
		Signature:      req.Signature,
		UserId:         user.UserId,
	}, nil
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
	})
	return &pb.SetUserMessageResponse{}, nil
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
	userId := req.PlayerId
	if userId == 0 {
		userId = CurrentUserId(ctx, s.users, s.sessions)
	}
	user, err := s.users.LoadUser(userId)
	if err != nil {
		return &pb.GetUserProfileResponse{}, nil
	}

	deckCharacters := []*pb.ProfileDeckCharacter{}
	if deck, ok := user.Decks[store.DeckKey{DeckType: model.DeckTypeQuest, UserDeckNumber: 1}]; ok && deck.UserDeckCharacterUuid01 != "" {
		if deckCharacter, ok := user.DeckCharacters[deck.UserDeckCharacterUuid01]; ok {
			costumeId := int32(0)
			if costume, ok := user.Costumes[deckCharacter.UserCostumeUuid]; ok {
				costumeId = costume.CostumeId
			}
			mainWeaponId := int32(0)
			mainWeaponLevel := int32(0)
			if weapon, ok := user.Weapons[deckCharacter.MainUserWeaponUuid]; ok {
				mainWeaponId = weapon.WeaponId
				mainWeaponLevel = weapon.Level
			}
			deckCharacters = append(deckCharacters, &pb.ProfileDeckCharacter{
				CostumeId:       costumeId,
				MainWeaponId:    mainWeaponId,
				MainWeaponLevel: mainWeaponLevel,
			})
		}
	}

	return &pb.GetUserProfileResponse{
		Level:             user.Status.Level,
		Name:              user.Profile.Name,
		FavoriteCostumeId: user.Profile.FavoriteCostumeId,
		Message:           user.Profile.Message,
		IsFriend:          false,
		LatestUsedDeck: &pb.ProfileDeck{
			Power:         100,
			DeckCharacter: deckCharacters,
		},
		PvpInfo: &pb.ProfilePvpInfo{},
		GamePlayHistory: &pb.GamePlayHistory{
			HistoryItem:              []*pb.PlayHistoryItem{},
			HistoryCategoryGraphItem: []*pb.PlayHistoryCategoryGraphItem{},
		},
	}, nil
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
	return &pb.GetUserGamePlayNoteResponse{}, nil
}

func (s *UserServiceServer) resolveAuthToken(ctx context.Context, token string) (facebookId int64, err error) {
	if s.authURL == "" {
		return 0, status.Error(codes.FailedPrecondition, "auth server not configured (--auth-url)")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.authURL+"/me", nil)
	if err != nil {
		return 0, status.Errorf(codes.Internal, "create auth request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := s.authClient.Do(req)
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

	fbId, err := s.resolveAuthToken(ctx, req.Token)
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

	fbId, err := s.resolveAuthToken(ctx, req.Token)
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
