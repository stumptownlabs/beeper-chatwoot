//
// Implements the mautrix.Storer interface on StateStore
//

package store

import (
	log "github.com/sirupsen/logrus"
	"maunium.net/go/mautrix"
	mid "maunium.net/go/mautrix/id"
)

func (store *StateStore) SaveFilterID(userID mid.UserID, filterID string) {
	log.Debug("Upserting row into user_filter_ids")
	tx, err := store.DB.Begin()
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return
	}

	upsert := `
		INSERT INTO user_filter_ids (user_id, filter_id) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE
			SET filter_id = $2
	`
	if _, err := tx.Exec(upsert, userID, filterID); err != nil {
		tx.Rollback()
		log.Error(err)
		return
	}

	tx.Commit()
}

func (store *StateStore) LoadFilterID(userID mid.UserID) string {
	row := store.DB.QueryRow("SELECT filter_id FROM user_filter_ids WHERE user_id = $1", userID)
	var filterID string
	if err := row.Scan(&filterID); err != nil {
		return ""
	}
	return filterID
}

func (store *StateStore) SaveNextBatch(userID mid.UserID, nextBatchToken string) {
	log.Debug("Upserting row into user_batch_tokens")
	tx, err := store.DB.Begin()
	if err != nil {
		tx.Rollback()
		return
	}

	upsert := `
		INSERT INTO user_batch_tokens (user_id, next_batch_token) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE
			SET next_batch_token = $2
	`
	if _, err := tx.Exec(upsert, userID, nextBatchToken); err != nil {
		tx.Rollback()
		log.Error(err)
		return
	}

	tx.Commit()
}

func (store *StateStore) LoadNextBatch(userID mid.UserID) string {
	row := store.DB.QueryRow("SELECT next_batch_token FROM user_batch_tokens WHERE user_id = $1", userID)
	var batchToken string
	if err := row.Scan(&batchToken); err != nil {
		return ""
	}
	return batchToken
}

func (store *StateStore) GetRoomMembers(roomId mid.RoomID) []mid.UserID {
	rows, err := store.DB.Query("SELECT user_id FROM room_members WHERE room_id = $1", roomId)
	users := make([]mid.UserID, 0)
	if err != nil {
		return users
	}
	defer rows.Close()

	var userId mid.UserID
	for rows.Next() {
		if err := rows.Scan(&userId); err == nil {
			users = append(users, userId)
		}
	}
	return users
}

func (store *StateStore) GetNonBotRoomMembers(roomId mid.RoomID) []mid.UserID {
	rows, err := store.DB.Query(`
		SELECT user_id
		FROM room_members
		WHERE room_id = $1
		AND user_id != $2
	`, roomId, store.botUsername)
	users := make([]mid.UserID, 0)
	if err != nil {
		return users
	}
	defer rows.Close()

	var userId mid.UserID
	for rows.Next() {
		if err := rows.Scan(&userId); err == nil {
			users = append(users, userId)
		}
	}
	return users
}

func (store *StateStore) SaveRoom(room *mautrix.Room) {
	// This isn't really used at all.
}

func (store *StateStore) LoadRoom(roomId mid.RoomID) *mautrix.Room {
	// This isn't really used at all.
	return mautrix.NewRoom(roomId)
}
