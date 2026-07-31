package store

import (
	"errors"
	"time"

	"lunar-tear/server/internal/model"
)

var ErrNotFound = errors.New("store: not found")

type Clock func() time.Time

type UserRepository interface {
	CreateUser(uuid string, platform model.ClientPlatform) (int64, error)
	GetUserByUUID(uuid string) (int64, error)
	GetUserByPlayerId(playerId int64) (int64, error)
	LoadUser(userId int64) (UserState, error)
	UpdateUser(userId int64, mutate func(*UserState)) (UserState, error)
	UpdateUsers(userIds []int64, mutate func(map[int64]*UserState) error) (map[int64]UserState, error)
	DefaultUserId() (int64, error)
	SetFacebookId(userId int64, facebookId int64) error
	GetUserByFacebookId(facebookId int64) (int64, error)
	GetFacebookId(userId int64) (int64, error)
	ClearFacebookId(userId int64) error
	UpdateUUID(userId int64, newUuid string) error
}

type SessionRepository interface {
	CreateSession(uuid string, ttl time.Duration) (SessionState, error)
	ResolveUserId(sessionKey string) (int64, error)
}
