package sqlite

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/migrations"
)

func TestFriendStateRoundTripAndImportClearsSocialState(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	repo := New(db, nil)
	requesterId, err := repo.CreateUser("requester", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	targetId, err := repo.CreateUser("target", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateUsers([]int64{requesterId, targetId}, func(users map[int64]*store.UserState) error {
		users[requesterId].Friends[targetId] = store.FriendState{IsFriend: true}
		users[targetId].Friends[requesterId] = store.FriendState{IsFriend: true}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	requester, err := repo.LoadUser(requesterId)
	if err != nil {
		t.Fatal(err)
	}
	if friend, ok := requester.Friends[targetId]; !ok || !friend.IsFriend {
		t.Fatal("friend relationship was not persisted")
	}
	if err := repo.ImportUser(&requester); err != nil {
		t.Fatal(err)
	}
	requester, err = repo.LoadUser(requesterId)
	if err != nil {
		t.Fatal(err)
	}
	target, err := repo.LoadUser(targetId)
	if err != nil {
		t.Fatal(err)
	}
	if len(requester.Friends) != 0 || len(requester.FriendRequests) != 0 {
		t.Fatalf("imported social state was retained: friends=%d requests=%d", len(requester.Friends), len(requester.FriendRequests))
	}
	if _, exists := target.Friends[requesterId]; exists {
		t.Fatal("inbound friendship survived single-user import")
	}
}

func TestUpdateUsersSerializesReadModifyWrite(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := New(db, nil)
	leftId, err := repo.CreateUser("left", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	rightId, err := repo.CreateUser("right", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateUsers([]int64{leftId, rightId}, func(users map[int64]*store.UserState) error {
		users[leftId].Friends[rightId] = store.FriendState{IsFriend: true}
		users[rightId].Friends[leftId] = store.FriendState{IsFriend: true}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	firstEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondStarted := make(chan struct{})
	secondEntered := make(chan struct{})
	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, updateErr := repo.UpdateUsers([]int64{leftId, rightId}, func(users map[int64]*store.UserState) error {
			close(firstEntered)
			<-releaseFirst
			leftFriend := users[leftId].Friends[rightId]
			leftFriend.CheerSentDatetime = 1
			users[leftId].Friends[rightId] = leftFriend
			rightFriend := users[rightId].Friends[leftId]
			rightFriend.CheerReceivedDatetime = 1
			users[rightId].Friends[leftId] = rightFriend
			return nil
		})
		errCh <- updateErr
	}()
	<-firstEntered
	go func() {
		defer wg.Done()
		close(secondStarted)
		_, updateErr := repo.UpdateUsers([]int64{rightId, leftId}, func(users map[int64]*store.UserState) error {
			close(secondEntered)
			rightFriend := users[rightId].Friends[leftId]
			rightFriend.CheerSentDatetime = 2
			users[rightId].Friends[leftId] = rightFriend
			leftFriend := users[leftId].Friends[rightId]
			leftFriend.CheerReceivedDatetime = 2
			users[leftId].Friends[rightId] = leftFriend
			return nil
		})
		errCh <- updateErr
	}()
	<-secondStarted

	enteredBeforeRelease := false
	select {
	case <-secondEntered:
		enteredBeforeRelease = true
	case <-time.After(50 * time.Millisecond):
	}
	close(releaseFirst)
	wg.Wait()
	close(errCh)
	for updateErr := range errCh {
		if updateErr != nil {
			t.Fatal(updateErr)
		}
	}
	if enteredBeforeRelease {
		t.Fatal("second update entered its mutation before the first update committed")
	}

	left, err := repo.LoadUser(leftId)
	if err != nil {
		t.Fatal(err)
	}
	right, err := repo.LoadUser(rightId)
	if err != nil {
		t.Fatal(err)
	}
	if friend := left.Friends[rightId]; friend.CheerSentDatetime != 1 || friend.CheerReceivedDatetime != 2 {
		t.Fatalf("left friend state = %+v", friend)
	}
	if friend := right.Friends[leftId]; friend.CheerSentDatetime != 2 || friend.CheerReceivedDatetime != 1 {
		t.Fatalf("right friend state = %+v", friend)
	}
}

func TestUpdateUserDoesNotSerializeDifferentUsers(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := New(db, nil)
	leftID, err := repo.CreateUser("parallel-left", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}
	rightID, err := repo.CreateUser("parallel-right", model.ClientPlatform{})
	if err != nil {
		t.Fatal(err)
	}

	leftEntered := make(chan struct{})
	releaseLeft := make(chan struct{})
	rightEntered := make(chan struct{})
	errCh := make(chan error, 2)
	go func() {
		_, updateErr := repo.UpdateUser(leftID, func(user *store.UserState) {
			close(leftEntered)
			<-releaseLeft
			user.Status.Exp++
		})
		errCh <- updateErr
	}()
	<-leftEntered
	go func() {
		_, updateErr := repo.UpdateUser(rightID, func(user *store.UserState) {
			close(rightEntered)
			user.Status.Exp++
		})
		errCh <- updateErr
	}()

	select {
	case <-rightEntered:
	case <-time.After(time.Second):
		close(releaseLeft)
		t.Fatal("an unrelated user update was blocked by the first user's callback")
	}
	close(releaseLeft)
	for range 2 {
		if updateErr := <-errCh; updateErr != nil {
			t.Fatal(updateErr)
		}
	}
}
