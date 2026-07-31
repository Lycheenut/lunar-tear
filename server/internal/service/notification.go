package service

import (
	"context"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/store"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type NotificationServiceServer struct {
	pb.UnimplementedNotificationServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
}

func NewNotificationServiceServer(users store.UserRepository, sessions store.SessionRepository) *NotificationServiceServer {
	return &NotificationServiceServer{users: users, sessions: sessions}
}

func (s *NotificationServiceServer) GetHeaderNotification(ctx context.Context, req *emptypb.Empty) (*pb.GetHeaderNotificationResponse, error) {
	log.Printf("[NotificationService] GetHeaderNotification")
	userId := CurrentUserId(ctx, s.users, s.sessions)
	user, err := s.users.LoadUser(userId)
	if err != nil {
		return &pb.GetHeaderNotificationResponse{
			GiftNotReceiveCount:       0,
			FriendRequestReceiveCount: 0,
			IsExistUnreadInformation:  false,
		}, nil
	}
	return &pb.GetHeaderNotificationResponse{
		GiftNotReceiveCount:       claimableGiftCount(user.Gifts.NotReceived, gametime.NowMillis()),
		FriendRequestReceiveCount: int32(len(user.FriendRequests)),
		IsExistUnreadInformation:  user.Notifications.IsExistUnreadInformation,
	}, nil
}
