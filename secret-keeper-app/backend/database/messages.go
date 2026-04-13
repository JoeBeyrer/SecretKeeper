package database

import (
	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type MessageRow struct {
    ID string
    SenderID string
    Username string
    DisplayName string
    ProfilePictureURL string
    Ciphertext string
    CreatedAt int64
    Reactions []ReactionRow
}

type ReactionRow struct {
    MessageID   string
    UserID      string
    Username    string
    DisplayName string
    Emoji       string
}

func SaveMessage(db *sql.DB, id, convID, senderID, ciphertext string, createdAt int64) error {
    var messageLifetime int64
    db.QueryRow(`SELECT message_lifetime FROM conversations WHERE id = ?`, convID).Scan(&messageLifetime)
    var expiresAt *int64
    if messageLifetime > 0 {
        t := createdAt + (messageLifetime * 60)
        expiresAt = &t
    }
    _, err := db.Exec(`
        INSERT INTO messages (id, conversation_id, sender_id, ciphertext, created_at, expires_at)
        VALUES (?, ?, ?, ?, ?, ?)
    `, id, convID, senderID, ciphertext, createdAt, expiresAt)
    return err
}

func IsUserInConversation(db *sql.DB, userID, conversationID string) bool {
	var exists int

	err := db.QueryRow(`
		SELECT 1 FROM conversation_members
		WHERE conversation_id = ? AND user_id = ?
		LIMIT 1
	`, conversationID, userID).Scan(&exists)

	return err == nil
}

func GetUsernameByID(db *sql.DB, userID string) (string, error) {
	var username string
	err := db.QueryRow(`SELECT username FROM users WHERE id = ?`, userID).Scan(&username)
	return username, err
}

func GetDisplayNameByID(db *sql.DB, userID string) (string, error) {
	var displayName sql.NullString
	err := db.QueryRow(`SELECT display_name FROM user_profiles WHERE user_id = ?`, userID).Scan(&displayName)
	if err != nil {
		return "", err
	}
	return displayName.String, nil
}

func GetProfilePictureURLByID(db *sql.DB, userID string) (string, error) {
    var url sql.NullString
    err := db.QueryRow(
        `SELECT profile_picture_url FROM user_profiles WHERE user_id = ?`, userID,
    ).Scan(&url)
    if err != nil {
        return "", err
    }
    return url.String, nil
}

func GetConversationMembers(db *sql.DB, conversationID string) ([]string, error) {
	rows, err := db.Query(`
		SELECT user_id FROM conversation_members
		WHERE conversation_id = ?
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		members = append(members, userID)
	}

	return members, nil
}

func GetMessagesByConversation(db *sql.DB, conversationID string, limit int) ([]MessageRow, error) {
	now := time.Now().Unix()
    rows, err := db.Query(`
        SELECT
            m.id,
            m.sender_id,
            u.username,
            COALESCE(p.display_name, u.username) AS display_name,
            COALESCE(p.profile_picture_url, ''),
            m.ciphertext,
            m.created_at
        FROM messages m
        JOIN users u ON u.id = m.sender_id
        LEFT JOIN user_profiles p ON p.user_id = m.sender_id
        WHERE m.conversation_id = ?
          AND (m.expires_at IS NULL OR m.expires_at > ?)
        ORDER BY m.created_at ASC
        LIMIT ?
    `, conversationID, now, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var result []MessageRow
    for rows.Next() {
        var msg MessageRow
        if err := rows.Scan(&msg.ID, &msg.SenderID, &msg.Username, &msg.DisplayName, &msg.ProfilePictureURL, &msg.Ciphertext, &msg.CreatedAt); err != nil {
            return nil, err
        }
        result = append(result, msg)
    }
    return result, nil
}

func UpdateMessage(db *sql.DB, messageID, senderID, ciphertext string) (string, error) {
	var convID string
	err := db.QueryRow(`
        SELECT conversation_id
        FROM messages
        WHERE id = ? AND sender_id = ?
    `, messageID, senderID).Scan(&convID)
	if err != nil {
		return "", err
	}

	_, err = db.Exec(`
        UPDATE messages
        SET ciphertext = ?
        WHERE id = ? AND sender_id = ?
    `, ciphertext, messageID, senderID)
	if err != nil {
		return "", err
	}

	return convID, nil
}

func SaveUserKeys(db *sql.DB, userID, publicKey, encryptedPrivateKey string) error {
    _, err := db.Exec(`
        INSERT INTO user_keys (user_id, public_key, encrypted_private_key)
        VALUES (?, ?, ?)
        ON CONFLICT(user_id) DO UPDATE SET
            public_key = excluded.public_key,
            encrypted_private_key = excluded.encrypted_private_key
    `, userID, publicKey, encryptedPrivateKey)
    return err
}

func GetUserPublicKey(db *sql.DB, userID string) (string, error) {
    var key string
    err := db.QueryRow(`SELECT public_key FROM user_keys WHERE user_id = ?`, userID).Scan(&key)
    return key, err
}

func GetUserKeys(db *sql.DB, userID string) (publicKey, encryptedPrivateKey string, err error) {
    err = db.QueryRow(`
        SELECT public_key, encrypted_private_key FROM user_keys WHERE user_id = ?
    `, userID).Scan(&publicKey, &encryptedPrivateKey)
    return
}

func SaveConversationKey(db *sql.DB, conversationID, userID, encryptedKey string) error {
    _, err := db.Exec(`
        INSERT INTO conversation_keys (conversation_id, user_id, encrypted_key)
        VALUES (?, ?, ?)
        ON CONFLICT(conversation_id, user_id) DO UPDATE SET
            encrypted_key = excluded.encrypted_key
    `, conversationID, userID, encryptedKey)
    return err
}

func GetConversationKey(db *sql.DB, conversationID, userID string) (string, error) {
    var key string
    err := db.QueryRow(`
        SELECT encrypted_key FROM conversation_keys
        WHERE conversation_id = ? AND user_id = ?
    `, conversationID, userID).Scan(&key)
    return key, err
}

func ClaimConversationRoomKey(db *sql.DB, conversationID, userID string) (string, error) {
	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var roomKey sql.NullString
	var recipientID sql.NullString
	err = tx.QueryRow(`
        SELECT pending_room_key, pending_room_key_recipient_id
        FROM conversations
        WHERE id = ?
    `, conversationID).Scan(&roomKey, &recipientID)
	if err != nil {
		return "", err
	}

	if !roomKey.Valid || roomKey.String == "" || !recipientID.Valid || recipientID.String != userID {
		return "", sql.ErrNoRows
	}

	result, err := tx.Exec(`
        UPDATE conversations
        SET pending_room_key = NULL,
            pending_room_key_recipient_id = NULL
        WHERE id = ? AND pending_room_key_recipient_id = ?
    `, conversationID, userID)
	if err != nil {
		return "", err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	if rowsAffected == 0 {
		return "", sql.ErrNoRows
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return roomKey.String, nil
}

func HashConversationRoomKey(roomKey string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(roomKey), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyConversationRoomKey(db *sql.DB, conversationID, roomKey string) (bool, error) {
	var roomKeyHash sql.NullString
	err := db.QueryRow(`
        SELECT room_key_hash FROM conversations WHERE id = ?
    `, conversationID).Scan(&roomKeyHash)
	if err != nil {
		return false, err
	}
	if !roomKeyHash.Valid || roomKeyHash.String == "" {
		return false, sql.ErrNoRows
	}

	err = bcrypt.CompareHashAndPassword([]byte(roomKeyHash.String), []byte(roomKey))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func GetConversationIDsForUser(db *sql.DB, userID string) ([]string, error) {
	rows, err := db.Query(`SELECT conversation_id FROM conversation_members WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, nil
}

// ToggleReaction inserts the reaction if missing, deletes it if present.
// Returns true if the reaction was added, false if removed.
func ToggleReaction(db *sql.DB, messageID, userID, emoji string) (bool, error) {
	var exists int
	err := db.QueryRow(`
        SELECT 1 FROM message_reactions
        WHERE message_id = ? AND user_id = ? AND emoji = ?
    `, messageID, userID, emoji).Scan(&exists)

	if err == nil {
		_, delErr := db.Exec(`
            DELETE FROM message_reactions
            WHERE message_id = ? AND user_id = ? AND emoji = ?
        `, messageID, userID, emoji)
		return false, delErr
	}

	if err != sql.ErrNoRows {
		return false, err
	}

	_, insErr := db.Exec(`
        INSERT INTO message_reactions (message_id, user_id, emoji, created_at)
        VALUES (?, ?, ?, ?)
    `, messageID, userID, emoji, time.Now().Unix())
	return true, insErr
}

// GetConversationIDForMessage returns the conversation the message belongs to.
func GetConversationIDForMessage(db *sql.DB, messageID string) (string, error) {
	var convID string
	err := db.QueryRow(`SELECT conversation_id FROM messages WHERE id = ?`, messageID).Scan(&convID)
	return convID, err
}

// GetReactionsForConversation returns all reactions on messages in a conversation,
// grouped by message_id and ordered by created_at.
func GetReactionsForConversation(db *sql.DB, conversationID string) (map[string][]ReactionRow, error) {
	rows, err := db.Query(`
        SELECT
            r.message_id,
            r.user_id,
            u.username,
            COALESCE(p.display_name, u.username) AS display_name,
            r.emoji
        FROM message_reactions r
        JOIN messages m ON m.id = r.message_id
        JOIN users u ON u.id = r.user_id
        LEFT JOIN user_profiles p ON p.user_id = r.user_id
        WHERE m.conversation_id = ?
        ORDER BY r.created_at ASC
    `, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]ReactionRow)
	for rows.Next() {
		var r ReactionRow
		if err := rows.Scan(&r.MessageID, &r.UserID, &r.Username, &r.DisplayName, &r.Emoji); err != nil {
			return nil, err
		}
		result[r.MessageID] = append(result[r.MessageID], r)
	}
	return result, nil
}
