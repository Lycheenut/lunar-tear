package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	sqlitestore "lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestFriendServiceLifecycleAndCheer(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	repo := sqlitestore.New(db, nil)
	userA := createFriendTestUser(t, repo, "friend-a", "Alice")
	userB := createFriendTestUser(t, repo, "friend-b", "Bob")
	userC := createFriendTestUser(t, repo, "friend-c", "Carol")
	ctxA := friendTestContext(t, repo, "friend-a")
	ctxB := friendTestContext(t, repo, "friend-b")
	ctxC := friendTestContext(t, repo, "friend-c")

	server := &FriendServiceServer{users: repo, sessions: repo}
	notifications := NewNotificationServiceServer(repo, repo)

	gotUser, err := server.GetUser(ctxA, &pb.GetUserRequest{PlayerId: userB.PlayerId})
	if err != nil {
		t.Fatal(err)
	}
	if gotUser.User.UserName != "Bob" || gotUser.User.PlayerId != userB.PlayerId {
		t.Fatalf("GetUser = %+v", gotUser.User)
	}
	recommended, err := server.SearchRecommendedUsers(ctxA, &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if len(recommended.Users) != 2 {
		t.Fatalf("recommended users = %d, want 2", len(recommended.Users))
	}

	if _, err := server.SendFriendRequest(ctxA, &pb.SendFriendRequestRequest{PlayerId: userB.PlayerId}); err != nil {
		t.Fatal(err)
	}
	header, err := notifications.GetHeaderNotification(ctxB, &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if header.FriendRequestReceiveCount != 1 {
		t.Fatalf("friend request notification = %d, want 1", header.FriendRequestReceiveCount)
	}
	requests, err := server.GetFriendRequestList(ctxB, &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if len(requests.User) != 1 || requests.User[0].PlayerId != userA.PlayerId {
		t.Fatalf("friend requests = %+v", requests.User)
	}
	if _, err := server.AcceptFriendRequest(ctxB, &pb.AcceptFriendRequestRequest{PlayerId: userA.PlayerId}); err != nil {
		t.Fatal(err)
	}
	assertMutualFriends(t, repo, userA.UserId, userB.UserId, true)

	friendList, err := server.GetFriendList(ctxA, &pb.GetFriendListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(friendList.FriendUser) != 1 || friendList.FriendUser[0].PlayerId != userB.PlayerId {
		t.Fatalf("friend list = %+v", friendList.FriendUser)
	}
	if _, err := server.CheerFriend(ctxA, &pb.CheerFriendRequest{PlayerId: userB.PlayerId}); err != nil {
		t.Fatal(err)
	}
	friendList, err = server.GetFriendList(ctxB, &pb.GetFriendListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(friendList.FriendUser) != 1 || !friendList.FriendUser[0].CheerReceived || friendList.FriendUser[0].StaminaReceived {
		t.Fatalf("cheer flags before receive = %+v", friendList.FriendUser)
	}
	setFriendTestStamina(t, repo, userB.UserId, 50000)
	if _, err := server.ReceiveCheer(ctxB, &pb.ReceiveCheerRequest{PlayerId: userA.PlayerId}); err != nil {
		t.Fatal(err)
	}
	updatedB, err := repo.LoadUser(userB.UserId)
	if err != nil {
		t.Fatal(err)
	}
	if updatedB.Status.StaminaMilliValue != 51000 {
		t.Fatalf("stamina = %d, want 51000", updatedB.Status.StaminaMilliValue)
	}
	if _, err := server.ReceiveCheer(ctxB, &pb.ReceiveCheerRequest{PlayerId: userA.PlayerId}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("duplicate receive status = %v, want FailedPrecondition", status.Code(err))
	}

	if _, err := server.DeleteFriend(ctxA, &pb.DeleteFriendRequest{PlayerId: userB.PlayerId}); err != nil {
		t.Fatal(err)
	}
	assertMutualFriends(t, repo, userA.UserId, userB.UserId, false)

	if _, err := server.SendFriendRequest(ctxA, &pb.SendFriendRequestRequest{PlayerId: userC.PlayerId}); err != nil {
		t.Fatal(err)
	}
	if _, err := server.DeclineFriendRequest(ctxC, &pb.DeclineFriendRequestRequest{PlayerIdOld: userA.PlayerId}); err != nil {
		t.Fatal(err)
	}
	header, err = notifications.GetHeaderNotification(ctxC, &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if header.FriendRequestReceiveCount != 0 {
		t.Fatalf("notification after decline = %d, want 0", header.FriendRequestReceiveCount)
	}

	makeFriendship(t, server, ctxA, ctxB, userA.PlayerId, userB.PlayerId)
	makeFriendship(t, server, ctxA, ctxC, userA.PlayerId, userC.PlayerId)
	bulkCheer, err := server.BulkCheerFriend(ctxA, &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bulkCheer.PlayerId) != 1 || bulkCheer.PlayerId[0] != userC.PlayerId {
		t.Fatalf("bulk cheered players = %v, want [%d] (deleted/re-added friend must stay limited)", bulkCheer.PlayerId, userC.PlayerId)
	}
	setFriendTestStamina(t, repo, userC.UserId, 50000)
	bulkReceive, err := server.BulkReceiveCheer(ctxC, &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bulkReceive.PlayerId) != 1 || bulkReceive.PlayerId[0] != userA.PlayerId {
		t.Fatalf("bulk received players = %v, want [%d]", bulkReceive.PlayerId, userA.PlayerId)
	}
	updatedC, err := repo.LoadUser(userC.UserId)
	if err != nil {
		t.Fatal(err)
	}
	if updatedC.Status.StaminaMilliValue != 51000 {
		t.Fatalf("bulk received stamina = %d, want 51000", updatedC.Status.StaminaMilliValue)
	}
}

func createFriendTestUser(t *testing.T, repo *sqlitestore.SQLiteStore, uuid, name string) store.UserState {
	t.Helper()
	userId, err := repo.CreateUser(uuid, model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	user, err := repo.UpdateUser(userId, func(user *store.UserState) {
		user.Profile.Name = name
		user.DeckTypeNotes[1] = store.DeckTypeNoteState{MaxDeckPower: 1234}
	})
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func friendTestContext(t *testing.T, repo *sqlitestore.SQLiteStore, uuid string) context.Context {
	t.Helper()
	session, err := repo.CreateSession(uuid, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-apb-session-key", session.SessionKey))
}

func assertMutualFriends(t *testing.T, repo *sqlitestore.SQLiteStore, leftId, rightId int64, want bool) {
	t.Helper()
	left, err := repo.LoadUser(leftId)
	if err != nil {
		t.Fatal(err)
	}
	right, err := repo.LoadUser(rightId)
	if err != nil {
		t.Fatal(err)
	}
	leftHas := left.Friends[rightId].IsFriend
	rightHas := right.Friends[leftId].IsFriend
	if leftHas != want || rightHas != want {
		t.Fatalf("mutual friendship = (%v,%v), want %v", leftHas, rightHas, want)
	}
}

func setFriendTestStamina(t *testing.T, repo *sqlitestore.SQLiteStore, userId int64, stamina int32) {
	t.Helper()
	_, err := repo.UpdateUser(userId, func(user *store.UserState) {
		user.Status.StaminaMilliValue = stamina
		user.Status.StaminaUpdateDatetime = gametime.NowMillis()
	})
	if err != nil {
		t.Fatal(err)
	}
}

func makeFriendship(t *testing.T, server *FriendServiceServer, senderCtx, receiverCtx context.Context, senderPlayerId, receiverPlayerId int64) {
	t.Helper()
	if _, err := server.SendFriendRequest(senderCtx, &pb.SendFriendRequestRequest{PlayerId: receiverPlayerId}); err != nil {
		t.Fatal(err)
	}
	if _, err := server.AcceptFriendRequest(receiverCtx, &pb.AcceptFriendRequestRequest{PlayerId: senderPlayerId}); err != nil {
		t.Fatal(err)
	}
}
