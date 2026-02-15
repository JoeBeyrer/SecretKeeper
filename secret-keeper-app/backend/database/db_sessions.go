package database

import (
    "database/sql"
    "time"

    "github.com/google/uuid"
)

func CreateSession(db *sql.DB, userID string, ttl time.Duration) (string, int64, error) {
    sessionID := uuid.New().String()
    now := time.Now().Unix()
    expires := int64(0)
    if ttl > 0 {
        expires = time.Now().Add(ttl).Unix()
    }
    _, err := db.Exec(`INSERT INTO sessions (id, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`, sessionID, userID, now, expires)
    if err != nil {
        return "", 0, err
    }
    return sessionID, expires, nil
}

func DeleteSession(db *sql.DB, sessionID string) error {
    _, err := db.Exec(`DELETE FROM sessions WHERE id = ?`, sessionID)
    return err
}

func GetUserIDForSession(db *sql.DB, sessionID string) (string, error) {
    var userID string
    var expiresAt int64

    err := db.QueryRow(`
        SELECT user_id, expires_at 
        FROM sessions 
        WHERE id = ?`,
        sessionID,
    ).Scan(&userID, &expiresAt)

    if err != nil {
        return "", err
    }

    if expiresAt != 0 && time.Now().Unix() > expiresAt {
        return "", sql.ErrNoRows
    }

    return userID, nil
}

