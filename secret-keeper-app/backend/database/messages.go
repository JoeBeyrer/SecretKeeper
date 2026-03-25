package database

import (
    "database/sql"
)

type MessageRow struct {
    ID string
    SenderID string
    Username string
    DisplayName string
    Ciphertext string
    CreatedAt int64
}

func SaveMessage(db *sql.DB, id, convID, senderID, ciphertext string, createdAt int64) error {
	_, err := db.Exec(`
		INSERT INTO messages (id, conversation_id, sender_id, ciphertext, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, convID, senderID, ciphertext, createdAt)

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
    rows, err := db.Query(`
        SELECT
            m.id,
            m.sender_id,
            u.username,
            COALESCE(p.display_name, u.username) AS display_name,
            m.ciphertext,
            m.created_at
        FROM messages m
        JOIN users u ON u.id = m.sender_id
        LEFT JOIN user_profiles p ON p.user_id = m.sender_id
        WHERE m.conversation_id = ?
        ORDER BY m.created_at ASC
        LIMIT ?
    `, conversationID, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var result []MessageRow
    for rows.Next() {
        var msg MessageRow
        if err := rows.Scan(&msg.ID, &msg.SenderID, &msg.Username, &msg.DisplayName, &msg.Ciphertext, &msg.CreatedAt); err != nil {
            return nil, err
        }
        result = append(result, msg)
    }
    return result, nil
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