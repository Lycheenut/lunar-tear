package missionprogress

import (
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

// Repository decorates every user mutation with mission reconciliation. This
// keeps mission wiring out of dozens of unrelated gRPC service constructors.
type Repository struct {
	store.UserRepository
	store.SessionRepository
	holder *runtime.Holder
}

func NewRepository(base interface {
	store.UserRepository
	store.SessionRepository
}, holder *runtime.Holder) *Repository {
	return &Repository{UserRepository: base, SessionRepository: base, holder: holder}
}

func (r *Repository) CreateUser(uuid string, platform model.ClientPlatform) (int64, error) {
	userId, err := r.UserRepository.CreateUser(uuid, platform)
	if err != nil || r.holder == nil {
		return userId, err
	}
	_, err = r.UpdateUser(userId, func(*store.UserState) {})
	return userId, err
}

func (r *Repository) UpdateUser(userId int64, mutate func(*store.UserState)) (store.UserState, error) {
	return r.UserRepository.UpdateUser(userId, func(user *store.UserState) {
		before := store.CloneUserState(*user)
		before.PendingMissionEvents = nil
		user.PendingMissionEvents = nil
		mutate(user)
		events := append([]store.MissionEvent(nil), user.PendingMissionEvents...)
		user.PendingMissionEvents = nil
		if r.holder != nil {
			Apply(r.holder.Get(), &before, user, events, gametime.NowMillis())
		}
	})
}

func (r *Repository) UpdateUsers(userIds []int64, mutate func(map[int64]*store.UserState) error) (map[int64]store.UserState, error) {
	return r.UserRepository.UpdateUsers(userIds, func(users map[int64]*store.UserState) error {
		before := make(map[int64]store.UserState, len(users))
		for id, user := range users {
			before[id] = store.CloneUserState(*user)
			copy := before[id]
			copy.PendingMissionEvents = nil
			before[id] = copy
			user.PendingMissionEvents = nil
		}
		if err := mutate(users); err != nil {
			return err
		}
		if r.holder == nil {
			return nil
		}
		catalogs := r.holder.Get()
		nowMillis := gametime.NowMillis()
		for id, user := range users {
			events := append([]store.MissionEvent(nil), user.PendingMissionEvents...)
			user.PendingMissionEvents = nil
			old := before[id]
			Apply(catalogs, &old, user, events, nowMillis)
		}
		return nil
	})
}

func (r *Repository) SetFacebookId(userId, facebookId int64) error {
	if err := r.UserRepository.SetFacebookId(userId, facebookId); err != nil {
		return err
	}
	_, err := r.UpdateUser(userId, func(*store.UserState) {})
	return err
}
