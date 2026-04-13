package database

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"secret-keeper-app/backend/models"
)

// inserts a new pending friendship row (accepted = 0)
func SendFriendRequest(db *sql.DB, requesterID, addresseeID string) error {
	now := time.Now().Unix()
	id := uuid.New().String()

	_, err := db.Exec(`
		INSERT INTO friendships (id, requester_id, addressee_id, accepted, created_at, updated_at)
		VALUES (?, ?, ?, 0, ?, ?)
	`, id, requesterID, addresseeID, now, now)

	return err
}

// sets accepted = 1 for a pending request
func AcceptFriendRequest(db *sql.DB, addresseeID, requesterID string) error {
	now := time.Now().Unix()

	_, err := db.Exec(`
		UPDATE friendships
		SET accepted = 1, updated_at = ?
		WHERE requester_id = ? AND addressee_id = ? AND accepted = 0
	`, now, requesterID, addresseeID)

	return err
}

// deletes the pending row the caller sent (caller is the requester)
func RescindFriendRequest(db *sql.DB, requesterID, addresseeID string) error {
	_, err := db.Exec(`
		DELETE FROM friendships
		WHERE requester_id = ? AND addressee_id = ? AND accepted = 0
	`, requesterID, addresseeID)
	return err
}

// deletes the pending row
func DeclineFriendRequest(db *sql.DB, addresseeID, requesterID string) error {
	_, err := db.Exec(`
		DELETE FROM friendships
		WHERE requester_id = ? AND addressee_id = ? AND accepted = 0
	`, requesterID, addresseeID)

	return err
}

// deletes an accepted friendship between two users
func RemoveFriend(db *sql.DB, userID, otherUserID string) error {
	_, err := db.Exec(`
		DELETE FROM friendships
		WHERE accepted = 1
		  AND (
		    (requester_id = ? AND addressee_id = ?)
		    OR
		    (requester_id = ? AND addressee_id = ?)
		  )
	`, userID, otherUserID, otherUserID, userID)

	return err
}

// returns all accepted friends for the given user
func GetFriends(db *sql.DB, userID string) ([]models.FriendEntry, error) {
	rows, err := db.Query(`
		SELECT
			CASE WHEN requester_id = ? THEN addressee_id ELSE requester_id END AS friend_id,
			u.username,
			COALESCE(p.display_name, '') AS display_name
		FROM friendships f
		JOIN users u ON u.id = CASE WHEN f.requester_id = ? THEN f.addressee_id ELSE f.requester_id END
		LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE (requester_id = ? OR addressee_id = ?)
		  AND accepted = 1
	`, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []models.FriendEntry
	for rows.Next() {
		var f models.FriendEntry
		if err := rows.Scan(&f.UserID, &f.Username, &f.DisplayName); err != nil {
			return nil, err
		}
		f.Accepted = true
		friends = append(friends, f)
	}

	return friends, nil
}

// returns all pending requests involving the user
func GetPendingRequests(db *sql.DB, userID string) ([]models.FriendEntry, error) {
	rows, err := db.Query(`
		SELECT
			CASE WHEN requester_id = ? THEN addressee_id ELSE requester_id END AS other_id,
			u.username,
			COALESCE(p.display_name, '') AS display_name,
			CASE WHEN requester_id = ? THEN 'outgoing' ELSE 'incoming' END AS direction
		FROM friendships f
		JOIN users u ON u.id = CASE WHEN f.requester_id = ? THEN f.addressee_id ELSE f.requester_id END
		LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE (requester_id = ? OR addressee_id = ?)
		  AND accepted = 0
	`, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []models.FriendEntry
	for rows.Next() {
		var f models.FriendEntry
		if err := rows.Scan(&f.UserID, &f.Username, &f.DisplayName, &f.Direction); err != nil {
			return nil, err
		}
		f.Accepted = false
		requests = append(requests, f)
	}

	return requests, nil
}

// returns true if any row (pending or accepted) exists between two users.
func FriendshipExists(db *sql.DB, userA, userB string) (exists bool, accepted bool, direction string, err error) {
	var requesterID string
	var acceptedInt int

	err = db.QueryRow(`
		SELECT requester_id, accepted FROM friendships
		WHERE (requester_id = ? AND addressee_id = ?)
		   OR (requester_id = ? AND addressee_id = ?)
	`, userA, userB, userB, userA).Scan(&requesterID, &acceptedInt)

	if err == sql.ErrNoRows {
		return false, false, "", nil
	}
	if err != nil {
		return false, false, "", err
	}

	if requesterID == userA {
		direction = "outgoing"
	} else {
		direction = "incoming"
	}

	return true, acceptedInt == 1, direction, nil
}

// looks up a user ID from username
func GetUserIDByUsername(db *sql.DB, username string) (string, error) {
	var id string
	err := db.QueryRow(`SELECT id FROM users WHERE username = ?`, username).Scan(&id)
	return id, err
}

// SearchUsers returns up to 20 verified users whose username OR display name
// contains the query string (case-insensitive partial match), excluding only
// the caller. Each result carries the current friendship status with the caller.
func SearchUsers(db *sql.DB, callerID, query string) ([]models.UserSearchResult, error) {
	pattern := "%" + query + "%"

	rows, err := db.Query(`
		SELECT
			u.id,
			u.username,
			COALESCE(p.display_name, '') AS display_name,
			CASE
				WHEN f.id IS NULL                  THEN 'none'
				WHEN f.accepted = 1                THEN 'friend'
				WHEN f.requester_id = ?            THEN 'pending_outgoing'
				ELSE                                    'pending_incoming'
			END AS status
		FROM users u
		LEFT JOIN user_profiles p ON p.user_id = u.id
		LEFT JOIN friendships f ON (
			(f.requester_id = ? AND f.addressee_id = u.id)
			OR (f.addressee_id = ? AND f.requester_id = u.id)
		)
		WHERE u.email_verified = 1
		  AND u.id != ?
		  AND (u.username LIKE ? OR COALESCE(p.display_name, '') LIKE ?)
		ORDER BY u.username
		LIMIT 20
	`, callerID, callerID, callerID, callerID, pattern, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.UserSearchResult
	for rows.Next() {
		var r models.UserSearchResult
		if err := rows.Scan(&r.UserID, &r.Username, &r.DisplayName, &r.Status); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}