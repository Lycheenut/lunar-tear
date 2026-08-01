package sqlite

import (
	"database/sql"
	"fmt"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func (s *SQLiteStore) CreateUser(uuid string, platform model.ClientPlatform) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	nowMillis := s.clock().UnixMilli()

	res, err := tx.Exec(`INSERT INTO users (uuid, player_id, os_type, platform_type, user_restriction_type,
		register_datetime, game_start_datetime, latest_version, birth_year, birth_month,
		backup_token, charge_money_this_month) VALUES (?, 0, ?, ?, 0, ?, ?, 0, 2000, 1, 'mock-backup-token', 0)`,
		uuid, platform.OsType, platform.PlatformType, nowMillis, nowMillis)
	if err != nil {
		return 0, fmt.Errorf("insert user: %w", err)
	}
	userId, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}

	// player_id = user_id
	if _, err := tx.Exec(`UPDATE users SET player_id = ? WHERE user_id = ?`, userId, userId); err != nil {
		return 0, fmt.Errorf("update player_id: %w", err)
	}

	user := store.SeedUserState(userId, uuid, nowMillis, platform)
	if err := writeUserState(tx, userId, user); err != nil {
		return 0, fmt.Errorf("write seed state: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return userId, nil
}

func (s *SQLiteStore) GetUserByUUID(uuid string) (int64, error) {
	var userId int64
	err := s.db.QueryRow(`SELECT user_id FROM users WHERE uuid = ?`, uuid).Scan(&userId)
	if err == sql.ErrNoRows {
		return 0, store.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("query user: %w", err)
	}
	return userId, nil
}

func (s *SQLiteStore) GetUserByPlayerId(playerId int64) (int64, error) {
	var userId int64
	err := s.db.QueryRow(`SELECT user_id FROM users WHERE player_id = ? ORDER BY user_id LIMIT 1`, playerId).Scan(&userId)
	if err == sql.ErrNoRows {
		return 0, store.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("query user by player_id: %w", err)
	}
	return userId, nil
}

func (s *SQLiteStore) ListUserIds() ([]int64, error) {
	rows, err := s.db.Query(`SELECT user_id FROM users ORDER BY user_id`)
	if err != nil {
		return nil, fmt.Errorf("query user ids: %w", err)
	}
	defer rows.Close()

	var userIds []int64
	for rows.Next() {
		var userId int64
		if err := rows.Scan(&userId); err != nil {
			return nil, fmt.Errorf("scan user id: %w", err)
		}
		userIds = append(userIds, userId)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user ids: %w", err)
	}
	return userIds, nil
}

func (s *SQLiteStore) DefaultUserId() (int64, error) {
	var userId int64
	err := s.db.QueryRow(`SELECT min(user_id) FROM users`).Scan(&userId)
	if err != nil || userId == 0 {
		return 0, store.ErrNotFound
	}
	return userId, nil
}

// ImportUser replaces all data for u.UserId in the database with the
// contents of u.  Any pre-existing rows for that user are deleted first.
func (s *SQLiteStore) ImportUser(u *store.UserState) error {
	_, unlock := s.userLocks.lock([]int64{u.UserId})
	defer unlock()

	// A single-user snapshot cannot restore the other side of friendships or
	// outgoing requests. Drop cross-user social state instead of importing an
	// inconsistent half relationship.
	imported := store.CloneUserState(*u)
	imported.Friends = make(map[int64]store.FriendState)
	imported.FriendRequests = make(map[int64]int64)
	u = &imported

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	uid := u.UserId
	if _, err := tx.Exec(`DELETE FROM user_friend_requests WHERE requester_user_id = ?`, uid); err != nil {
		return fmt.Errorf("delete outgoing friend requests: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM user_friends WHERE friend_user_id = ?`, uid); err != nil {
		return fmt.Errorf("delete inbound friend relationships: %w", err)
	}

	for _, t := range userOwnedTables {
		if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE user_id = ?`, t), uid); err != nil {
			return fmt.Errorf("delete from %s: %w", t, err)
		}
	}
	if _, err := tx.Exec(`DELETE FROM users WHERE user_id = ?`, uid); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	if _, err := tx.Exec(`INSERT INTO users (user_id, uuid, player_id, os_type, platform_type,
		user_restriction_type, register_datetime, game_start_datetime, latest_version,
		birth_year, birth_month, backup_token, charge_money_this_month)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		uid, u.Uuid, u.PlayerId, u.OsType, u.PlatformType, u.UserRestrictionType,
		u.RegisterDatetime, u.GameStartDatetime, u.LatestVersion,
		u.BirthYear, u.BirthMonth, u.BackupToken, u.ChargeMoneyThisMonth); err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	if err := writeUserState(tx, uid, u); err != nil {
		return fmt.Errorf("write user state: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func (s *SQLiteStore) SetFacebookId(userId int64, facebookId int64) error {
	_, unlock := s.userLocks.lock([]int64{userId})
	defer unlock()

	_, err := s.db.Exec(`UPDATE users SET facebook_id = ? WHERE user_id = ?`, facebookId, userId)
	if err != nil {
		return fmt.Errorf("set facebook_id: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetUserByFacebookId(facebookId int64) (int64, error) {
	var userId int64
	err := s.db.QueryRow(`SELECT user_id FROM users WHERE facebook_id = ?`, facebookId).Scan(&userId)
	if err == sql.ErrNoRows {
		return 0, store.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("query user by facebook_id: %w", err)
	}
	return userId, nil
}

func (s *SQLiteStore) GetFacebookId(userId int64) (int64, error) {
	var fbId sql.NullInt64
	err := s.db.QueryRow(`SELECT facebook_id FROM users WHERE user_id = ?`, userId).Scan(&fbId)
	if err != nil {
		return 0, store.ErrNotFound
	}
	return fbId.Int64, nil
}

func (s *SQLiteStore) ClearFacebookId(userId int64) error {
	_, unlock := s.userLocks.lock([]int64{userId})
	defer unlock()

	_, err := s.db.Exec(`UPDATE users SET facebook_id = NULL WHERE user_id = ?`, userId)
	if err != nil {
		return fmt.Errorf("clear facebook_id: %w", err)
	}
	return nil
}

func (s *SQLiteStore) UpdateUUID(userId int64, newUuid string) error {
	_, unlock := s.userLocks.lock([]int64{userId})
	defer unlock()

	_, err := s.db.Exec(`UPDATE users SET uuid = ? WHERE user_id = ?`, newUuid, userId)
	if err != nil {
		return fmt.Errorf("update uuid: %w", err)
	}
	return nil
}

func (s *SQLiteStore) UpdateUser(userId int64, mutate func(*store.UserState)) (store.UserState, error) {
	updated, err := s.UpdateUsers([]int64{userId}, func(users map[int64]*store.UserState) error {
		mutate(users[userId])
		return nil
	})
	if err != nil {
		return store.UserState{}, err
	}
	return updated[userId], nil
}

func (s *SQLiteStore) UpdateUsers(userIds []int64, mutate func(map[int64]*store.UserState) error) (map[int64]store.UserState, error) {
	uniqueUserIds, unlock := s.userLocks.lock(userIds)
	defer unlock()

	before := make(map[int64]store.UserState, len(userIds))
	after := make(map[int64]*store.UserState, len(userIds))
	for _, userId := range uniqueUserIds {
		user, err := s.LoadUser(userId)
		if err != nil {
			return nil, err
		}
		before[userId] = user
		cloned := store.CloneUserState(user)
		after[userId] = &cloned
	}
	if err := mutate(after); err != nil {
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()
	for _, userId := range uniqueUserIds {
		old := before[userId]
		if err := diffAndSave(tx, userId, &old, after[userId]); err != nil {
			return nil, fmt.Errorf("diff and save user %d: %w", userId, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	updated := make(map[int64]store.UserState, len(after))
	for userId, user := range after {
		updated[userId] = *user
	}
	return updated, nil
}
