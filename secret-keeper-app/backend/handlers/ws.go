package handlers

import (
	"log"
	"net/http"
	"encoding/json"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"secret-keeper-app/backend/messaging"
	"secret-keeper-app/backend/models"
	"secret-keeper-app/backend/database"
)


var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func WebSocketHandler(hub *messaging.Hub, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Authenticate user via cookie
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

		// Upgrade connection
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
		hub.Unregister(c.UserID)
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
		hub.Unregister(c.UserID)
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		log.Println("Received:", string(message))

		var msg models.WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if msg.Type == "send_message" {

			id := uuid.New().String()
			convID := msg.ConversationID
			senderID := c.UserID
			ciphertext := msg.Ciphertext
			createdAt := time.Now().Unix()

			if !database.IsUserInConversation(db, senderID, convID) {
				continue
			}

			err := database.SaveMessage(db, id, convID, senderID, ciphertext, createdAt)
			if err != nil {
				log.Println("Failed to save message:", err)
				continue
			}

			senderUsername, err := database.GetUsernameByID(db, senderID)
			if err != nil {
				continue
			}

			outgoing := models.WSMessage{
				Type:           "new_message",
				ConversationID: convID,
				Ciphertext:     ciphertext,
				SenderID:       senderUsername,
			}

			jsonData, err := json.Marshal(outgoing)
			if err != nil {
				continue
			}

			// Get conversation members and broadcast to all of them
			members, err := database.GetConversationMembers(db, convID)
			if err != nil {
				continue
			}

			for _, userID := range members {
				hub.SendToUser(userID, jsonData)
			}
		}
	}
}