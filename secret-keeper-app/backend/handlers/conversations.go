package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"
    "log"
	"github.com/google/uuid"
	"secret-keeper-app/backend/database"
)

type createConvReq struct {
	MemberIDs []string `json:"member_ids"`
	RoomKey   string   `json:"room_key"`
}

type createConvResp struct {
	ConversationID string `json:"conversation_id"`
	Created        bool   `json:"created"`
}

func CreateConversationHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        var req createConvReq
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }

        // Resolve usernames to UUIDs
        var resolvedIDs []string
        for _, username := range req.MemberIDs {
            var resolvedID string
            err := db.QueryRow(`SELECT id FROM users WHERE username = ?`, username).Scan(&resolvedID)
            if err != nil {
                http.Error(w, "user not found: "+username, http.StatusBadRequest)
                return
            }
            resolvedIDs = append(resolvedIDs, resolvedID)
        }

        // Deduplicate members
        members := append(resolvedIDs, userID)
        seen := map[string]bool{}
        var uniqueMembers []string
        for _, id := range members {
            if !seen[id] {
                seen[id] = true
                uniqueMembers = append(uniqueMembers, id)
            }
        }

        // For 1-on-1 conversations (exactly 2 members), check if one already exists
        if len(uniqueMembers) == 2 {
            var existingID string
            err := db.QueryRow(`
                SELECT cm1.conversation_id FROM conversation_members cm1
                JOIN conversation_members cm2 ON cm1.conversation_id = cm2.conversation_id
                WHERE cm1.user_id = ? AND cm2.user_id = ?
                AND (SELECT COUNT(*) FROM conversation_members cm3 WHERE cm3.conversation_id = cm1.conversation_id) = 2
                LIMIT 1
            `, uniqueMembers[0], uniqueMembers[1]).Scan(&existingID)

            if err == nil {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusOK)
                json.NewEncoder(w).Encode(map[string]string{"conversation_id": existingID})
                return
            }
        }

		req.RoomKey = strings.TrimSpace(req.RoomKey)
		if req.RoomKey == "" {
			http.Error(w, "missing room key", http.StatusBadRequest)
			return
		}
		if len(req.RoomKey) <= 6 {
			http.Error(w, "room key must be longer than 6 characters", http.StatusBadRequest)
			return
		}

		roomKeyHash, err := database.HashConversationRoomKey(req.RoomKey)
		if err != nil {
			http.Error(w, "could not protect room key", http.StatusInternalServerError)
			return
		}

		// No existing conversation found - create a new one
		convID := uuid.New().String()
		now := time.Now().Unix()

		var pendingRoomKeyRecipientID any
		if len(uniqueMembers) == 2 {
			for _, id := range uniqueMembers {
				if id != userID {
					pendingRoomKeyRecipientID = id
					break
				}
			}
		}

		_, err = db.Exec(`
            INSERT INTO conversations (
                id,
                created_at,
                room_key_hash,
                pending_room_key,
                pending_room_key_recipient_id
            ) VALUES (?, ?, ?, ?, ?)
        `, convID, now, roomKeyHash, req.RoomKey, pendingRoomKeyRecipientID)
		if err != nil {
			http.Error(w, "could not create conversation", http.StatusInternalServerError)
			return
		}

		for _, id := range uniqueMembers {
			_, err := db.Exec(`INSERT INTO conversation_members (conversation_id, user_id, joined_at) VALUES (?, ?, ?)`, convID, id, now)
			if err != nil {
				http.Error(w, "could not add member", http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(createConvResp{ConversationID: convID, Created: true})
	}
}

type ConversationSummary struct {
    ID              string `json:"id"`
    Name            string `json:"name"`
    LastMessage     string `json:"last_message"`
    LastMessageTime int64  `json:"last_message_time"`
    MessageLifetime int `json:"message_lifetime"`
}

func GetConversationsHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        rows, err := db.Query(`
            SELECT
                c.id,
                (
                    SELECT GROUP_CONCAT(COALESCE(NULLIF(p.display_name, ''), u.username), ', ')
                    FROM conversation_members cm2
                    JOIN users u ON u.id = cm2.user_id
                    LEFT JOIN user_profiles p ON p.user_id = u.id
                    WHERE cm2.conversation_id = c.id AND cm2.user_id != ?
                ) AS name,
                (
                    SELECT m.ciphertext
                    FROM messages m
                    WHERE m.conversation_id = c.id
                    ORDER BY m.created_at DESC
                    LIMIT 1
                ) AS last_message,
                (
                    SELECT m.created_at
                    FROM messages m
                    WHERE m.conversation_id = c.id
                    ORDER BY m.created_at DESC
                    LIMIT 1
                ) AS last_message_time
            FROM conversations c
            JOIN conversation_members cm ON cm.conversation_id = c.id
            WHERE cm.user_id = ?
            ORDER BY COALESCE(last_message_time, 0) DESC
        `, userID, userID)
        if err != nil {
            http.Error(w, "could not fetch conversations", http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        var result []ConversationSummary
        for rows.Next() {
            var s ConversationSummary
            var name sql.NullString
            var lastMsg sql.NullString
            var lastTime sql.NullInt64
            if err := rows.Scan(&s.ID, &name, &lastMsg, &lastTime); err != nil {
                continue
            }
            s.Name = name.String
            if s.Name == "" {
                s.Name = s.ID[:8]
            }
            s.LastMessage = lastMsg.String
            s.LastMessageTime = lastTime.Int64
            result = append(result, s)
        }

        if result == nil {
            result = []ConversationSummary{}
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(result)
    }
}

func GetConversationMessagesHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        convID := r.PathValue("id")
        if convID == "" {
            http.Error(w, "missing conversation id", http.StatusBadRequest)
            return
        }

        if !database.IsUserInConversation(db, userID, convID) {
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }

        msgs, err := database.GetMessagesByConversation(db, convID, 200)
        if err != nil {
            http.Error(w, "could not fetch messages", http.StatusInternalServerError)
            return
        }

        if msgs == nil {
            msgs = []database.MessageRow{}
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(msgs)
    }
}

func VerifyConversationRoomKeyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserIDFromContext(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		convID := r.PathValue("id")
		if convID == "" {
			http.Error(w, "missing conversation id", http.StatusBadRequest)
			return
		}

		if !database.IsUserInConversation(db, userID, convID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		var body struct {
			RoomKey string `json:"room_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if body.RoomKey == "" {
			http.Error(w, "missing room key", http.StatusBadRequest)
			return
		}

		ok, err := database.VerifyConversationRoomKey(db, convID, body.RoomKey)
		if err == sql.ErrNoRows {
			http.Error(w, "room key verifier not set", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "could not verify room key", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "incorrect room key", http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}


type claimRoomKeyResp struct {
	RoomKey string `json:"room_key"`
}

func ClaimConversationRoomKeyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserIDFromContext(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		convID := r.PathValue("id")
		if convID == "" {
			http.Error(w, "missing conversation id", http.StatusBadRequest)
			return
		}

		if !database.IsUserInConversation(db, userID, convID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		roomKey, err := database.ClaimConversationRoomKey(db, convID, userID)
		if err == sql.ErrNoRows {
			http.Error(w, "no pending room key", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "could not claim room key", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(claimRoomKeyResp{RoomKey: roomKey})
	}
}

func SetMessageLifetimeHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID, ok := GetUserIDFromContext(r)
        if !ok {
            log.Println("[Lifetime] unauthorized")
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        convID := r.PathValue("id")
        if convID == "" {
            log.Println("[Lifetime] missing conversation id")
            http.Error(w, "missing conversation id", http.StatusBadRequest)
            return
        }

        if !database.IsUserInConversation(db, userID, convID) {
            log.Println("[Lifetime] forbidden - user not in conversation")
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }

        var body struct {
            MessageLifetime int `json:"message_lifetime"`
        }
        if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
            log.Println("[Lifetime] invalid request body:", err)
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }

        log.Printf("[Lifetime] setting lifetime for conversation %s to %d\n", convID, body.MessageLifetime)

        _, err := db.Exec(`UPDATE conversations SET message_lifetime = ? WHERE id = ?`, body.MessageLifetime, convID)
        if err != nil {
            log.Println("[Lifetime] db error:", err)
            http.Error(w, "could not update message lifetime", http.StatusInternalServerError)
            return
        }
        // Update expires_at for all existing messages in the conversation
        if body.MessageLifetime > 0 {
            _, err = db.Exec(`
                UPDATE messages 
                SET expires_at = created_at + ?
                WHERE conversation_id = ?
            `, body.MessageLifetime * 60, convID)
        } else {
            _, err = db.Exec(`
                UPDATE messages 
                SET expires_at = NULL
                WHERE conversation_id = ?
            `, convID)
        }
        if err != nil {
            log.Println("[Lifetime] failed to update message expiries:", err)
            http.Error(w, "could not update message expiries", http.StatusInternalServerError)
            return
        }
        log.Println("[Lifetime] success")
        w.WriteHeader(http.StatusNoContent)
    }
}