package database

import (
    "database/sql"
    "encoding/json"
    "log"
    "time"

    "secret-keeper-app/backend/messaging"
    "secret-keeper-app/backend/models"
)

func CleanupMessages(db *sql.DB, hub *messaging.Hub) {
    go func() {
        ticker := time.NewTicker(time.Minute)
        defer ticker.Stop()
        for range ticker.C {
            CleanExpiredMessages(db, hub)
        }
    }()
}

func CleanExpiredMessages(db *sql.DB, hub *messaging.Hub) {
    now := time.Now().Unix()

    rows, err := db.Query(`
        SELECT DISTINCT conversation_id FROM messages
        WHERE expires_at IS NOT NULL AND expires_at <= ?
    `, now)
    if err != nil {
        log.Println("Error fetching expired messages:", err)
        return
    }

    var affectedConvs []string
    for rows.Next() {
        var convID string
        if err := rows.Scan(&convID); err != nil {
            continue
        }
        affectedConvs = append(affectedConvs, convID)
    }
    rows.Close()

    _, err = db.Exec(`
        DELETE FROM messages
        WHERE expires_at IS NOT NULL AND expires_at <= ?
    `, now)
    if err != nil {
        log.Println("Error cleaning expired messages:", err)
        return
    }

	for _, convID := range affectedConvs {
		log.Println("[Cleanup] notifying conversation:", convID)
		members, err := GetConversationMembers(db, convID)
		if err != nil {
			log.Println("[Cleanup] failed to get members:", err)
			continue
		}
		log.Println("[Cleanup] members to notify:", members)
		notification, _ := json.Marshal(models.WSMessage{
			Type: "messages_updated",
			ConversationID: convID,
		})
		for _, userID := range members {
			log.Println("[Cleanup] sending to user:", userID)
			hub.SendToUser(userID, notification)
		}
	}
}
