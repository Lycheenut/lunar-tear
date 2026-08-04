package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	friendRecommendationLimit      = 20
	friendCheerStaminaMillis       = int32(1000)
	defaultFriendMax               = int32(100)
	defaultFriendReceiveCheerMax   = int32(100)
	defaultFriendReceiveRequestMax = int32(100)
	defaultFriendSendCheerMax      = int32(100)
)

type friendLimits struct {
	friendMax         int32
	receiveCheerMax   int32
	receiveRequestMax int32
	sendCheerMax      int32
}

type FriendServiceServer struct {
	pb.UnimplementedFriendServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	holder   *runtime.Holder
}

func NewFriendServiceServer(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder) *FriendServiceServer {
	return &FriendServiceServer{users: users, sessions: sessions, holder: holder}
}

func (s *FriendServiceServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	log.Printf("[FriendService] GetUser: playerId=%d", req.PlayerId)
	currentUserId := CurrentUserId(ctx, s.users, s.sessions)
	userId, err := s.userIdByPlayerId(req.PlayerId)
	if err != nil {
		return nil, err
	}
	if userId == currentUserId {
		return nil, status.Error(codes.InvalidArgument, "cannot search for yourself")
	}
	user, err := s.users.LoadUser(userId)
	if err != nil {
		return nil, fmt.Errorf("load user: %w", err)
	}
	return &pb.GetUserResponse{User: toFriendProtoUser(user)}, nil
}

func (s *FriendServiceServer) SearchRecommendedUsers(ctx context.Context, _ *emptypb.Empty) (*pb.SearchRecommendedUsersResponse, error) {
	log.Printf("[FriendService] SearchRecommendedUsers")
	currentUserId := CurrentUserId(ctx, s.users, s.sessions)
	current, err := s.users.LoadUser(currentUserId)
	if err != nil {
		return nil, fmt.Errorf("load current user: %w", err)
	}
	if activeFriendCount(current) >= s.limits().friendMax {
		return &pb.SearchRecommendedUsersResponse{Users: []*pb.User{}}, nil
	}

	userIds, err := s.users.ListUserIds()
	if err != nil {
		return nil, err
	}
	users := make([]*pb.User, 0, min(friendRecommendationLimit, len(userIds)))
	for _, userId := range userIds {
		if len(users) >= friendRecommendationLimit {
			break
		}
		if userId == currentUserId {
			continue
		}
		if current.Friends[userId].IsFriend {
			continue
		}
		if _, exists := current.FriendRequests[userId]; exists {
			continue
		}
		candidate, loadErr := s.users.LoadUser(userId)
		if loadErr != nil {
			continue
		}
		if _, requested := candidate.FriendRequests[currentUserId]; requested {
			continue
		}
		users = append(users, toFriendProtoUser(candidate))
	}
	return &pb.SearchRecommendedUsersResponse{Users: users}, nil
}

func (s *FriendServiceServer) GetFriendList(ctx context.Context, _ *pb.GetFriendListRequest) (*pb.GetFriendListResponse, error) {
	log.Printf("[FriendService] GetFriendList")
	currentUserId := CurrentUserId(ctx, s.users, s.sessions)
	current, err := s.users.LoadUser(currentUserId)
	if err != nil {
		return nil, fmt.Errorf("load current user: %w", err)
	}

	today := gametime.StartOfBusinessDayAtMillis(gametime.NowMillis())
	friendUsers := make([]*pb.FriendUser, 0, len(current.Friends))
	for _, friendUserId := range sortedFriendIds(current.Friends) {
		friend, loadErr := s.users.LoadUser(friendUserId)
		if loadErr != nil {
			continue
		}
		state := current.Friends[friendUserId]
		friendUsers = append(friendUsers, toFriendProtoFriendUser(friend, state, today))
	}
	return &pb.GetFriendListResponse{
		FriendUser:         friendUsers,
		SendCheerCount:     countDailySentCheers(current, today),
		ReceivedCheerCount: countDailyReceivedStamina(current, today),
	}, nil
}

func (s *FriendServiceServer) GetFriendRequestList(ctx context.Context, _ *emptypb.Empty) (*pb.GetFriendRequestListResponse, error) {
	log.Printf("[FriendService] GetFriendRequestList")
	currentUserId := CurrentUserId(ctx, s.users, s.sessions)
	current, err := s.users.LoadUser(currentUserId)
	if err != nil {
		return nil, fmt.Errorf("load current user: %w", err)
	}

	requesterIds := make([]int64, 0, len(current.FriendRequests))
	for userId := range current.FriendRequests {
		requesterIds = append(requesterIds, userId)
	}
	sort.Slice(requesterIds, func(i, j int) bool {
		left, right := current.FriendRequests[requesterIds[i]], current.FriendRequests[requesterIds[j]]
		if left == right {
			return requesterIds[i] < requesterIds[j]
		}
		return left > right
	})
	users := make([]*pb.User, 0, len(requesterIds))
	for _, requesterUserId := range requesterIds {
		requester, loadErr := s.users.LoadUser(requesterUserId)
		if loadErr == nil {
			users = append(users, toFriendProtoUser(requester))
		}
	}
	return &pb.GetFriendRequestListResponse{User: users}, nil
}

func (s *FriendServiceServer) SendFriendRequest(ctx context.Context, req *pb.SendFriendRequestRequest) (*pb.SendFriendRequestResponse, error) {
	log.Printf("[FriendService] SendFriendRequest: playerId=%d", req.PlayerId)
	currentUserId := CurrentUserId(ctx, s.users, s.sessions)
	targetUserId, err := s.userIdByPlayerId(req.PlayerId)
	if err != nil {
		return nil, err
	}
	if targetUserId == currentUserId {
		return nil, status.Error(codes.InvalidArgument, "cannot send a friend request to yourself")
	}
	limits := s.limits()
	_, err = s.users.UpdateUsers([]int64{currentUserId, targetUserId}, func(users map[int64]*store.UserState) error {
		current, target := users[currentUserId], users[targetUserId]
		if current.Friends[targetUserId].IsFriend {
			return status.Error(codes.AlreadyExists, "already friends")
		}
		if _, exists := target.FriendRequests[currentUserId]; exists {
			return status.Error(codes.AlreadyExists, "friend request already sent")
		}
		if _, exists := current.FriendRequests[targetUserId]; exists {
			return status.Error(codes.AlreadyExists, "an incoming friend request already exists")
		}
		if activeFriendCount(*current) >= limits.friendMax || activeFriendCount(*target) >= limits.friendMax {
			return status.Error(codes.ResourceExhausted, "friend list is full")
		}
		if int32(len(target.FriendRequests)) >= limits.receiveRequestMax {
			return status.Error(codes.ResourceExhausted, "friend request inbox is full")
		}
		target.FriendRequests[currentUserId] = gametime.NowMillis()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pb.SendFriendRequestResponse{}, nil
}

func (s *FriendServiceServer) AcceptFriendRequest(ctx context.Context, req *pb.AcceptFriendRequestRequest) (*pb.AcceptFriendRequestResponse, error) {
	log.Printf("[FriendService] AcceptFriendRequest: playerId=%d", req.PlayerId)
	currentUserId := CurrentUserId(ctx, s.users, s.sessions)
	requesterUserId, err := s.userIdByPlayerId(req.PlayerId)
	if err != nil {
		return nil, err
	}
	if requesterUserId == currentUserId {
		return nil, status.Error(codes.InvalidArgument, "cannot accept yourself as a friend")
	}
	limits := s.limits()
	_, err = s.users.UpdateUsers([]int64{currentUserId, requesterUserId}, func(users map[int64]*store.UserState) error {
		current, requester := users[currentUserId], users[requesterUserId]
		if _, exists := current.FriendRequests[requesterUserId]; !exists {
			return status.Error(codes.FailedPrecondition, "friend request not found")
		}
		if activeFriendCount(*current) >= limits.friendMax || activeFriendCount(*requester) >= limits.friendMax {
			return status.Error(codes.ResourceExhausted, "friend list is full")
		}
		delete(current.FriendRequests, requesterUserId)
		currentFriend := current.Friends[requesterUserId]
		currentFriend.IsFriend = true
		current.Friends[requesterUserId] = currentFriend
		requesterFriend := requester.Friends[currentUserId]
		requesterFriend.IsFriend = true
		requester.Friends[currentUserId] = requesterFriend
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pb.AcceptFriendRequestResponse{}, nil
}

func (s *FriendServiceServer) DeclineFriendRequest(ctx context.Context, req *pb.DeclineFriendRequestRequest) (*pb.DeclineFriendRequestResponse, error) {
	log.Printf("[FriendService] DeclineFriendRequest: old=%d playerIds=%v", req.PlayerIdOld, req.PlayerId)
	currentUserId := CurrentUserId(ctx, s.users, s.sessions)
	playerIds := append([]int64(nil), req.PlayerId...)
	if req.PlayerIdOld != 0 {
		playerIds = append(playerIds, req.PlayerIdOld)
	}
	requesterUserIds := make(map[int64]struct{}, len(playerIds))
	for _, playerId := range playerIds {
		requesterUserId, err := s.users.GetUserByPlayerId(playerId)
		if errors.Is(err, store.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("look up player %d: %w", playerId, err)
		}
		requesterUserIds[requesterUserId] = struct{}{}
	}
	_, err := s.users.UpdateUser(currentUserId, func(current *store.UserState) {
		for requesterUserId := range requesterUserIds {
			delete(current.FriendRequests, requesterUserId)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("decline friend request: %w", err)
	}
	return &pb.DeclineFriendRequestResponse{}, nil
}

func (s *FriendServiceServer) DeleteFriend(ctx context.Context, req *pb.DeleteFriendRequest) (*pb.DeleteFriendResponse, error) {
	log.Printf("[FriendService] DeleteFriend: playerId=%d", req.PlayerId)
	currentUserId := CurrentUserId(ctx, s.users, s.sessions)
	friendUserId, err := s.userIdByPlayerId(req.PlayerId)
	if err != nil {
		return nil, err
	}
	if friendUserId == currentUserId {
		return nil, status.Error(codes.InvalidArgument, "cannot delete yourself as a friend")
	}
	_, err = s.users.UpdateUsers([]int64{currentUserId, friendUserId}, func(users map[int64]*store.UserState) error {
		current, friend := users[currentUserId], users[friendUserId]
		currentFriend, exists := current.Friends[friendUserId]
		if !exists || !currentFriend.IsFriend {
			return status.Error(codes.FailedPrecondition, "friend not found")
		}
		friendCurrent, exists := friend.Friends[currentUserId]
		if !exists || !friendCurrent.IsFriend {
			return status.Error(codes.FailedPrecondition, "friend relationship is inconsistent")
		}
		currentFriend.IsFriend = false
		friendCurrent.IsFriend = false
		current.Friends[friendUserId] = currentFriend
		friend.Friends[currentUserId] = friendCurrent
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pb.DeleteFriendResponse{}, nil
}

func (s *FriendServiceServer) CheerFriend(ctx context.Context, req *pb.CheerFriendRequest) (*pb.CheerFriendResponse, error) {
	log.Printf("[FriendService] CheerFriend: playerId=%d", req.PlayerId)
	currentUserId := CurrentUserId(ctx, s.users, s.sessions)
	friendUserId, err := s.userIdByPlayerId(req.PlayerId)
	if err != nil {
		return nil, err
	}
	if friendUserId == currentUserId {
		return nil, status.Error(codes.InvalidArgument, "cannot cheer yourself")
	}
	limit := s.limits().sendCheerMax
	_, err = s.users.UpdateUsers([]int64{currentUserId, friendUserId}, func(users map[int64]*store.UserState) error {
		return sendCheer(users[currentUserId], users[friendUserId], gametime.NowMillis(), limit)
	})
	if err != nil {
		return nil, err
	}
	return &pb.CheerFriendResponse{}, nil
}

func (s *FriendServiceServer) BulkCheerFriend(ctx context.Context, _ *emptypb.Empty) (*pb.BulkCheerFriendResponse, error) {
	log.Printf("[FriendService] BulkCheerFriend")
	currentUserId := CurrentUserId(ctx, s.users, s.sessions)
	current, err := s.users.LoadUser(currentUserId)
	if err != nil {
		return nil, fmt.Errorf("load current user: %w", err)
	}
	friendUserIds := sortedFriendIds(current.Friends)
	userIds := append([]int64{currentUserId}, friendUserIds...)
	limit := s.limits().sendCheerMax
	cheeredPlayerIds := make([]int64, 0, len(friendUserIds))
	_, err = s.users.UpdateUsers(userIds, func(users map[int64]*store.UserState) error {
		current := users[currentUserId]
		nowMillis := gametime.NowMillis()
		for _, friendUserId := range friendUserIds {
			if countDailySentCheers(*current, gametime.StartOfBusinessDayAtMillis(nowMillis)) >= limit {
				break
			}
			friend := users[friendUserId]
			if friend == nil {
				continue
			}
			if err := sendCheer(current, friend, nowMillis, limit); err == nil {
				cheeredPlayerIds = append(cheeredPlayerIds, friend.PlayerId)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pb.BulkCheerFriendResponse{PlayerId: cheeredPlayerIds}, nil
}

func (s *FriendServiceServer) ReceiveCheer(ctx context.Context, req *pb.ReceiveCheerRequest) (*pb.ReceiveCheerResponse, error) {
	log.Printf("[FriendService] ReceiveCheer: playerId=%d", req.PlayerId)
	currentUserId := CurrentUserId(ctx, s.users, s.sessions)
	friendUserId, err := s.userIdByPlayerId(req.PlayerId)
	if err != nil {
		return nil, err
	}
	if friendUserId == currentUserId {
		return nil, status.Error(codes.InvalidArgument, "cannot receive cheer from yourself")
	}
	limit := s.limits().receiveCheerMax
	_, err = s.users.UpdateUsers([]int64{currentUserId}, func(users map[int64]*store.UserState) error {
		current := users[currentUserId]
		return receiveCheer(current, friendUserId, gametime.NowMillis(), s.maxStaminaMillis(*current), limit)
	})
	if err != nil {
		return nil, err
	}
	return &pb.ReceiveCheerResponse{}, nil
}

func (s *FriendServiceServer) BulkReceiveCheer(ctx context.Context, _ *emptypb.Empty) (*pb.BulkReceiveCheerResponse, error) {
	log.Printf("[FriendService] BulkReceiveCheer")
	currentUserId := CurrentUserId(ctx, s.users, s.sessions)
	current, err := s.users.LoadUser(currentUserId)
	if err != nil {
		return nil, fmt.Errorf("load current user: %w", err)
	}
	friendUserIds := sortedFriendIds(current.Friends)
	friendPlayerIds := make(map[int64]int64, len(friendUserIds))
	for _, friendUserId := range friendUserIds {
		friend, loadErr := s.users.LoadUser(friendUserId)
		if loadErr == nil {
			friendPlayerIds[friendUserId] = friend.PlayerId
		}
	}
	limit := s.limits().receiveCheerMax
	receivedPlayerIds := make([]int64, 0, len(friendUserIds))
	_, err = s.users.UpdateUsers([]int64{currentUserId}, func(users map[int64]*store.UserState) error {
		current := users[currentUserId]
		nowMillis := gametime.NowMillis()
		maxStamina := s.maxStaminaMillis(*current)
		for _, friendUserId := range friendUserIds {
			playerId, exists := friendPlayerIds[friendUserId]
			if !exists {
				continue
			}
			if err := receiveCheer(current, friendUserId, nowMillis, maxStamina, limit); err == nil {
				receivedPlayerIds = append(receivedPlayerIds, playerId)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pb.BulkReceiveCheerResponse{PlayerId: receivedPlayerIds}, nil
}

func (s *FriendServiceServer) userIdByPlayerId(playerId int64) (int64, error) {
	if playerId == 0 {
		return 0, status.Error(codes.InvalidArgument, "playerId is required")
	}
	userId, err := s.users.GetUserByPlayerId(playerId)
	if errors.Is(err, store.ErrNotFound) {
		return 0, status.Error(codes.NotFound, "player not found")
	}
	if err != nil {
		return 0, fmt.Errorf("look up player: %w", err)
	}
	return userId, nil
}

func (s *FriendServiceServer) limits() friendLimits {
	limits := friendLimits{
		friendMax:         defaultFriendMax,
		receiveCheerMax:   defaultFriendReceiveCheerMax,
		receiveRequestMax: defaultFriendReceiveRequestMax,
		sendCheerMax:      defaultFriendSendCheerMax,
	}
	if s.holder == nil {
		return limits
	}
	catalogs := s.holder.Get()
	if catalogs == nil || catalogs.GameConfig == nil {
		return limits
	}
	config := catalogs.GameConfig
	if config.UserFriendMaxNumber > 0 {
		limits.friendMax = config.UserFriendMaxNumber
	}
	if config.UserFriendReceiveCheerMaxNumber > 0 {
		limits.receiveCheerMax = config.UserFriendReceiveCheerMaxNumber
	}
	if config.UserFriendReceiveRequestMaxNumber > 0 {
		limits.receiveRequestMax = config.UserFriendReceiveRequestMaxNumber
	}
	if config.UserFriendSendCheerMaxNumber > 0 {
		limits.sendCheerMax = config.UserFriendSendCheerMaxNumber
	}
	return limits
}

func (s *FriendServiceServer) maxStaminaMillis(user store.UserState) int32 {
	if s.holder != nil {
		catalogs := s.holder.Get()
		if catalogs != nil && catalogs.Quest != nil {
			if maxStamina := catalogs.Quest.MaxStaminaByLevel[user.Status.Level]; maxStamina > 0 {
				return maxStamina * 1000
			}
		}
	}
	return max(user.Status.StaminaMilliValue, int32(50000))
}

func sendCheer(sender, receiver *store.UserState, nowMillis int64, dailyLimit int32) error {
	today := gametime.StartOfBusinessDayAtMillis(nowMillis)
	senderFriend, exists := sender.Friends[receiver.UserId]
	if !exists || !senderFriend.IsFriend {
		return status.Error(codes.FailedPrecondition, "friend not found")
	}
	receiverFriend, exists := receiver.Friends[sender.UserId]
	if !exists || !receiverFriend.IsFriend {
		return status.Error(codes.FailedPrecondition, "friend relationship is inconsistent")
	}
	if senderFriend.CheerSentDatetime >= today {
		return status.Error(codes.FailedPrecondition, "cheer already sent today")
	}
	if countDailySentCheers(*sender, today) >= dailyLimit {
		return status.Error(codes.ResourceExhausted, "daily cheer send limit reached")
	}
	senderFriend.CheerSentDatetime = nowMillis
	receiverFriend.CheerReceivedDatetime = nowMillis
	sender.Friends[receiver.UserId] = senderFriend
	receiver.Friends[sender.UserId] = receiverFriend
	return nil
}

func receiveCheer(user *store.UserState, friendUserId, nowMillis int64, maxStaminaMillis, dailyLimit int32) error {
	today := gametime.StartOfBusinessDayAtMillis(nowMillis)
	friend, exists := user.Friends[friendUserId]
	if !exists || !friend.IsFriend {
		return status.Error(codes.FailedPrecondition, "friend not found")
	}
	if friend.CheerReceivedDatetime < today {
		return status.Error(codes.FailedPrecondition, "no cheer available")
	}
	if friend.StaminaReceivedDatetime >= today {
		return status.Error(codes.FailedPrecondition, "cheer stamina already received today")
	}
	if countDailyReceivedStamina(*user, today) >= dailyLimit {
		return status.Error(codes.ResourceExhausted, "daily cheer receive limit reached")
	}
	if maxStaminaMillis <= 0 {
		return status.Error(codes.FailedPrecondition, "stamina limit is unavailable")
	}
	store.RecoverStamina(user, friendCheerStaminaMillis, maxStaminaMillis, nowMillis)
	user.Status.LatestVersion = nowMillis
	friend.StaminaReceivedDatetime = nowMillis
	user.Friends[friendUserId] = friend
	return nil
}

func countDailySentCheers(user store.UserState, today int64) int32 {
	var count int32
	for _, friend := range user.Friends {
		if friend.CheerSentDatetime >= today {
			count++
		}
	}
	return count
}

func countDailyReceivedStamina(user store.UserState, today int64) int32 {
	var count int32
	for _, friend := range user.Friends {
		if friend.StaminaReceivedDatetime >= today {
			count++
		}
	}
	return count
}

func sortedFriendIds(friends map[int64]store.FriendState) []int64 {
	ids := make([]int64, 0, len(friends))
	for userId, friend := range friends {
		if !friend.IsFriend {
			continue
		}
		ids = append(ids, userId)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func activeFriendCount(user store.UserState) int32 {
	var count int32
	for _, friend := range user.Friends {
		if friend.IsFriend {
			count++
		}
	}
	return count
}

func toFriendProtoUser(user store.UserState) *pb.User {
	return &pb.User{
		PlayerId:          user.PlayerId,
		UserName:          user.Profile.Name,
		LastLoginDatetime: friendTimestamp(user.Login.LastLoginDatetime),
		MaxDeckPower:      friendMaxDeckPower(user),
		FavoriteCostumeId: user.Profile.FavoriteCostumeId,
		Level:             user.Status.Level,
	}
}

func toFriendProtoFriendUser(user store.UserState, friend store.FriendState, today int64) *pb.FriendUser {
	return &pb.FriendUser{
		PlayerId:          user.PlayerId,
		UserName:          user.Profile.Name,
		LastLoginDatetime: friendTimestamp(user.Login.LastLoginDatetime),
		MaxDeckPower:      friendMaxDeckPower(user),
		FavoriteCostumeId: user.Profile.FavoriteCostumeId,
		Level:             user.Status.Level,
		CheerReceived:     friend.CheerReceivedDatetime >= today,
		CheerSent:         friend.CheerSentDatetime >= today,
		StaminaReceived:   friend.StaminaReceivedDatetime >= today,
	}
}

func friendMaxDeckPower(user store.UserState) int32 {
	var power int32
	for _, note := range user.DeckTypeNotes {
		power = max(power, note.MaxDeckPower)
	}
	for _, deck := range user.Decks {
		power = max(power, deck.Power)
	}
	return power
}

func friendTimestamp(millis int64) *timestamppb.Timestamp {
	if millis == 0 {
		return nil
	}
	return timestamppb.New(time.UnixMilli(millis))
}
