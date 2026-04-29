package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"secret-keeper-app/backend/database"
	"secret-keeper-app/backend/messaging"
	"secret-keeper-app/backend/models"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func WebSocketHandler(hub *messaging.Hub, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("sk_session")
		if err != nil {
			http.Error(w, "Unauthorized: no cookie", http.StatusUnauthorized)
			return
		}

		userID, err := database.GetUserIDForSession(db, cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized: invalid session", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}

		client := &messaging.Client{
			UserID: userID,
			Conn:   conn,
			Send:   make(chan []byte, 256), // buffered to avoid blocking hub
		}

		hub.Register(client)

		go writePump(client, hub)
		go readPump(client, hub, db)
	}
}

func writePump(c *messaging.Client, hub *messaging.Hub) {
	defer func() {
		hub.Unregister(c.UserID, c)
		c.Conn.Close()
	}()

	for msg := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			break
		}
	}
}

func readPump(c *messaging.Client, hub *messaging.Hub, db *sql.DB) {
	defer func() {
		hub.Unregister(c.UserID, c)
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg models.WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if msg.Type != "send_message" {
			continue
		}

		id := uuid.New().String()
		convID := msg.ConversationID
		senderID := c.UserID
		ciphertext := msg.Ciphertext
		createdAt := time.Now().Unix()

		if !database.IsUserInConversation(db, senderID, convID) {
			continue
		}

		members, err := database.GetConversationMembers(db, convID)
		if err != nil {
			continue
		}
		var recipientID string
		for _, memberID := range members {
			if memberID != senderID {
				recipientID = memberID
				break
			}
		}

		var blocked bool
		err = db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM blocks
				WHERE blocker_id = ? AND blockee_id = ?
			)
		`, recipientID, senderID).Scan(&blocked)
		if err != nil {
			log.Println("Failed to check block status:", err)
			continue
		}
		if blocked {
			continue
		}

		err = database.SaveMessage(db, id, convID, senderID, ciphertext, createdAt)
		if err != nil {
			log.Println("Failed to save message:", err)
			continue
		}

		senderUsername, err := database.GetUsernameByID(db, senderID)
		if err != nil {
			continue
		}

		senderDisplayName, _ := database.GetDisplayNameByID(db, senderID)
		senderPictureURL, _ := database.GetProfilePictureURLByID(db, senderID)

		// Read back the expires_at that SaveMessage computed so the broadcast
		// carries the exact expiry timestamp. Clients use this to render per-message
		// expiry labels immediately, without needing to reload message history.
		var savedExpiresAt *int64
		var rawExpiry *int64
		if scanErr := db.QueryRow(
			`SELECT expires_at FROM messages WHERE id = ?`, id,
		).Scan(&rawExpiry); scanErr == nil {
			savedExpiresAt = rawExpiry
		}

		outgoing := models.WSMessage{
			Type:              "new_message",
			ConversationID:    convID,
			Ciphertext:        ciphertext,
			SenderID:          senderUsername,
			UserID:            senderID,
			DisplayName:       senderDisplayName,
			ProfilePictureURL: senderPictureURL,
			MessageID:         id,
			ExpiresAt:         savedExpiresAt,
		}

		jsonData, err := json.Marshal(outgoing)
		if err != nil {
			continue
		}

		for _, userID := range members {
			hub.SendToUser(userID, jsonData)
		}

		if msg.ClientMessageID != "" {
			ack, err := json.Marshal(models.WSMessage{
				Type:            "message_ack",
				ConversationID:  convID,
				MessageID:       id,
				ClientMessageID: msg.ClientMessageID,
			})
			if err == nil {
				hub.SendToUser(senderID, ack)
			}
		}
	}
}
