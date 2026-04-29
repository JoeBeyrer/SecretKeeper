package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
	"secret-keeper-app/backend/database"
	"secret-keeper-app/backend/messaging"
	"secret-keeper-app/backend/models"
	"time"
	"strings"
)
var NotifyAsync = true

type createConvReq struct {
	MemberIDs []string `json:"member_ids"`
	RoomKey   string   `json:"room_key"`
	GroupName string   `json:"group_name"`
}

type createConvResp struct {
	ConversationID string `json:"conversation_id"`
	Created        bool   `json:"created"`
}

type editMessageReq struct {
	Ciphertext string `json:"ciphertext"`
}

type updateGroupNameReq struct {
	GroupName string `json:"group_name"`
}

type removeConversationMembersReq struct {
	MemberIDs []string `json:"member_ids"`
}

type addConversationMembersReq struct {
	MemberIDs []string `json:"member_ids"`
	RoomKey   string   `json:"room_key"`
}

var allowedMessageLifetimes = map[int]struct{}{
	0:      {},
	60:     {},
	1440:   {},
	10080:  {},
	43200:  {},
	525600: {},
}

func isAllowedMessageLifetime(lifetime int) bool {
	_, ok := allowedMessageLifetimes[lifetime]
	return ok
}

func notifyConversationMembers(db *sql.DB, hub *messaging.Hub, convID string) {
	if hub == nil || convID == "" {
		return
	}

	members, err := database.GetConversationMembers(db, convID)
	if err != nil {
		log.Println("[Conversation Notify] failed to get members:", err)
		return
	}

	notifyConversationUsers(hub, members, convID)
}

func notifyConversationUsers(hub *messaging.Hub, userIDs []string, convID string) {
	if hub == nil || convID == "" || len(userIDs) == 0 {
		return
	}

	notification, err := json.Marshal(models.WSMessage{
		Type:           "messages_updated",
		ConversationID: convID,
	})
	if err != nil {
		log.Println("[Conversation Notify] failed to marshal notification:", err)
		return
	}

	seen := make(map[string]struct{}, len(userIDs))
	for _, userID := range userIDs {
		if userID == "" {
			continue
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		hub.SendToUser(userID, notification)
	}
}

func deleteConversationDataTx(tx *sql.Tx, convID string) error {
	queries := []string{
		`DELETE FROM conversation_pending_room_keys WHERE conversation_id = ?`,
		`DELETE FROM conversation_keys WHERE conversation_id = ?`,
		`DELETE FROM messages WHERE conversation_id = ?`,
		`DELETE FROM conversation_members WHERE conversation_id = ?`,
		`DELETE FROM conversations WHERE id = ?`,
	}

	for _, query := range queries {
		if _, err := tx.Exec(query, convID); err != nil {
			return err
		}
	}

	return nil
}

func CreateConversationHandler(db *sql.DB, hub *messaging.Hub) http.HandlerFunc {
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
            if resolvedID == userID {
                http.Error(w, "you cannot create a conversation with yourself", http.StatusBadRequest)
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

		if req.RoomKey == "" {
			http.Error(w, "missing room key", http.StatusBadRequest)
			return
		}

		roomKeyHash, err := database.HashConversationRoomKey(req.RoomKey)
		if err != nil {
			http.Error(w, "could not protect room key", http.StatusInternalServerError)
			return
		}

		groupName := ""
		if len(uniqueMembers) > 2 {
			groupName = strings.TrimSpace(req.GroupName)
		}

		// No existing conversation found - create a new one
		convID := uuid.New().String()
		now := time.Now().Unix()

		_, err = db.Exec(`
            INSERT INTO conversations (
                id,
                created_at,
                room_key_hash,
                group_name
            ) VALUES (?, ?, ?, ?)
        `, convID, now, roomKeyHash, groupName)
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

			if id == userID {
				continue
			}

			_, err = db.Exec(`
                INSERT INTO conversation_pending_room_keys (conversation_id, user_id, room_key)
                VALUES (?, ?, ?)
            `, convID, id, req.RoomKey)
			if err != nil {
				http.Error(w, "could not store room key", http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(createConvResp{ConversationID: convID, Created: true})

		// Notify all members so their conversation list refreshes immediately,
		// without requiring a page reload.
        if NotifyAsync {
            go notifyConversationMembers(db, hub, convID)
        } else {
            notifyConversationMembers(db, hub, convID)
        }
	}
}

type ConversationSummary struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	LastMessage       string `json:"last_message"`
	LastMessageTime   int64  `json:"last_message_time"`
	MessageLifetime   int    `json:"message_lifetime"`
	MemberCount       int    `json:"member_count"`
	ProfilePictureURL string `json:"profile_picture_url"`
	OtherUsername     string `json:"other_username"`
	OtherUserID       string `json:"other_user_id"`
}

type ConversationMemberSummary struct {
	UserID            string `json:"user_id"`
	Username          string `json:"username"`
	DisplayName       string `json:"display_name"`
	ProfilePictureURL string `json:"profile_picture_url"`
	FriendshipStatus  string `json:"friendship_status"`
}

func GetConversationsHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
		now := time.Now().Unix()
        rows, err := db.Query(`
            SELECT
                c.id,
                c.message_lifetime,
                (
                    SELECT COUNT(*)
                    FROM conversation_members cm_count
                    WHERE cm_count.conversation_id = c.id
                ) AS member_count,
                COALESCE(
                    NULLIF(TRIM(c.group_name), ''),
                    (
                        SELECT GROUP_CONCAT(COALESCE(NULLIF(p.display_name, ''), u.username), ', ')
                        FROM conversation_members cm2
                        JOIN users u ON u.id = cm2.user_id
                        LEFT JOIN user_profiles p ON p.user_id = u.id
                        WHERE cm2.conversation_id = c.id AND cm2.user_id != ?
                    )
                ) AS name,
                (
                    SELECT m.ciphertext
                    FROM messages m
                    WHERE m.conversation_id = c.id
                      AND (m.expires_at IS NULL OR m.expires_at > ?)
                    ORDER BY m.created_at DESC
                    LIMIT 1
                ) AS last_message,
                (
                    SELECT m.created_at
                    FROM messages m
                    WHERE m.conversation_id = c.id
                      AND (m.expires_at IS NULL OR m.expires_at > ?)
                    ORDER BY m.created_at DESC
                    LIMIT 1
                ) AS last_message_time,
                CASE WHEN (SELECT COUNT(*) FROM conversation_members WHERE conversation_id = c.id) = 2
                    THEN COALESCE((
                        SELECT up.profile_picture_url
                        FROM conversation_members cm3
                        JOIN user_profiles up ON up.user_id = cm3.user_id
                        WHERE cm3.conversation_id = c.id AND cm3.user_id != ?
                        LIMIT 1
                    ), '')
                    ELSE COALESCE(c.group_picture_url, '')
                END AS profile_picture_url,
                CASE WHEN (SELECT COUNT(*) FROM conversation_members WHERE conversation_id = c.id) = 2
                    THEN COALESCE((
                        SELECT u2.username
                        FROM conversation_members cm4
                        JOIN users u2 ON u2.id = cm4.user_id
                        WHERE cm4.conversation_id = c.id AND cm4.user_id != ?
                        LIMIT 1
                    ), '')
                    ELSE ''
                END AS other_username,
                CASE WHEN (SELECT COUNT(*) FROM conversation_members WHERE conversation_id = c.id) = 2
                    THEN COALESCE((
                        SELECT cm5.user_id
                        FROM conversation_members cm5
                        WHERE cm5.conversation_id = c.id AND cm5.user_id != ?
                        LIMIT 1
                    ), '')
                    ELSE ''
                END AS other_user_id
            FROM conversations c
            JOIN conversation_members cm ON cm.conversation_id = c.id
            WHERE cm.user_id = ?
            ORDER BY COALESCE(last_message_time, 0) DESC
        `, userID, now, now, userID, userID, userID, userID)
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
            var picURL sql.NullString
            var otherUsername sql.NullString
            var otherUserID sql.NullString
            if err := rows.Scan(&s.ID, &s.MessageLifetime, &s.MemberCount, &name, &lastMsg, &lastTime, &picURL, &otherUsername, &otherUserID); err != nil {
                continue
            }
            s.Name = name.String
            if s.Name == "" {
                s.Name = s.ID[:8]
            }
            s.LastMessage = lastMsg.String
            s.LastMessageTime = lastTime.Int64
            s.ProfilePictureURL = picURL.String
            s.OtherUsername = otherUsername.String
            s.OtherUserID = otherUserID.String
            result = append(result, s)
        }

        if result == nil {
            result = []ConversationSummary{}
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(result)
    }
}

func GetConversationMembersHandler(db *sql.DB) http.HandlerFunc {
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

		rows, err := db.Query(`
            SELECT
                u.id,
                u.username,
                COALESCE(NULLIF(p.display_name, ''), u.username) AS display_name,
                COALESCE(p.profile_picture_url, '') AS profile_picture_url
            FROM conversation_members cm
            JOIN users u ON u.id = cm.user_id
            LEFT JOIN user_profiles p ON p.user_id = u.id
            WHERE cm.conversation_id = ?
            ORDER BY CASE WHEN cm.user_id = ? THEN 0 ELSE 1 END,
                     LOWER(COALESCE(NULLIF(p.display_name, ''), u.username)) ASC,
                     LOWER(u.username) ASC
        `, convID, userID)
		if err != nil {
			http.Error(w, "could not load conversation members", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		members := []ConversationMemberSummary{}
		for rows.Next() {
			var member ConversationMemberSummary
			if err := rows.Scan(&member.UserID, &member.Username, &member.DisplayName, &member.ProfilePictureURL); err != nil {
				http.Error(w, "could not load conversation members", http.StatusInternalServerError)
				return
			}

			if member.UserID == userID {
				member.FriendshipStatus = "self"
			} else {
				exists, accepted, direction, err := database.FriendshipExists(db, userID, member.UserID)
				if err != nil {
					http.Error(w, "could not load friendship status", http.StatusInternalServerError)
					return
				}

				switch {
				case accepted:
					member.FriendshipStatus = "friend"
				case exists && direction == "outgoing":
					member.FriendshipStatus = "pending_outgoing"
				case exists && direction == "incoming":
					member.FriendshipStatus = "pending_incoming"
				default:
					member.FriendshipStatus = "none"
				}
			}

			members = append(members, member)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(members)
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

        reactionsByMessage, err := database.GetReactionsForConversation(db, convID)
        if err != nil {
            log.Println("[Conversations] failed to load reactions:", err)
            reactionsByMessage = map[string][]database.ReactionRow{}
        }
        for i := range msgs {
            if rs, ok := reactionsByMessage[msgs[i].ID]; ok {
                msgs[i].Reactions = rs
            } else {
                msgs[i].Reactions = []database.ReactionRow{}
            }
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(msgs)
    }
}

func ToggleMessageReactionHandler(db *sql.DB, hub *messaging.Hub) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        messageID := r.PathValue("id")
        if messageID == "" {
            http.Error(w, "missing message id", http.StatusBadRequest)
            return
        }

        var body struct {
            Emoji string `json:"emoji"`
        }
        if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }
        if body.Emoji == "" {
            http.Error(w, "missing emoji", http.StatusBadRequest)
            return
        }

        convID, err := database.GetConversationIDForMessage(db, messageID)
        if err != nil {
            http.Error(w, "message not found", http.StatusNotFound)
            return
        }

        if !database.IsUserInConversation(db, userID, convID) {
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }

        if _, err := database.ToggleReaction(db, messageID, userID, body.Emoji); err != nil {
            log.Println("[Reactions] toggle failed:", err)
            http.Error(w, "could not toggle reaction", http.StatusInternalServerError)
            return
        }

        notifyConversationMembers(db, hub, convID)

        w.WriteHeader(http.StatusNoContent)
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

func LeaveConversationHandler(db *sql.DB, hub *messaging.Hub) http.HandlerFunc {
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

		rows, err := db.Query(`
			SELECT user_id
			FROM conversation_members
			WHERE conversation_id = ?
		`, convID)
		if err != nil {
			http.Error(w, "could not load conversation members", http.StatusInternalServerError)
			return
		}

		var members []string
		for rows.Next() {
			var memberID string
			if err := rows.Scan(&memberID); err != nil {
				rows.Close()
				http.Error(w, "could not load conversation members", http.StatusInternalServerError)
				return
			}
			members = append(members, memberID)
		}
		rows.Close()

		if len(members) == 0 {
			http.Error(w, "conversation not found", http.StatusNotFound)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "could not leave conversation", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		deleteEntireConversation := len(members) <= 2
		if deleteEntireConversation {
			if err := deleteConversationDataTx(tx, convID); err != nil {
				http.Error(w, "could not delete conversation", http.StatusInternalServerError)
				return
			}
		} else {
			var leavingName string
			if err := tx.QueryRow(`
				SELECT COALESCE(NULLIF(p.display_name, ''), u.username)
				FROM users u
				LEFT JOIN user_profiles p ON p.user_id = u.id
				WHERE u.id = ?
			`, userID).Scan(&leavingName); err != nil {
				http.Error(w, "could not load leaving user", http.StatusInternalServerError)
				return
			}

			if err := database.SaveSystemMessageTx(tx, uuid.New().String(), convID, leavingName+" has left the conversation", time.Now().Unix()); err != nil {
				http.Error(w, "could not record leave message", http.StatusInternalServerError)
				return
			}

			queries := []string{
				`DELETE FROM conversation_pending_room_keys WHERE conversation_id = ? AND user_id = ?`,
				`DELETE FROM conversation_keys WHERE conversation_id = ? AND user_id = ?`,
				`DELETE FROM conversation_members WHERE conversation_id = ? AND user_id = ?`,
			}
			for _, query := range queries {
				if _, err := tx.Exec(query, convID, userID); err != nil {
					http.Error(w, "could not leave conversation", http.StatusInternalServerError)
					return
				}
			}

			if len(members)-1 <= 2 {
				if _, err := tx.Exec(`UPDATE conversations SET group_name = '' WHERE id = ?`, convID); err != nil {
					http.Error(w, "could not update conversation", http.StatusInternalServerError)
					return
				}
			}
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "could not leave conversation", http.StatusInternalServerError)
			return
		}

		notifyConversationUsers(hub, members, convID)
		w.WriteHeader(http.StatusNoContent)
	}
}

func UpdateGroupNameHandler(db *sql.DB, hub *messaging.Hub) http.HandlerFunc {
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

		var body updateGroupNameReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		newGroupName := strings.TrimSpace(body.GroupName)
		if newGroupName == "" {
			http.Error(w, "group name is required", http.StatusBadRequest)
			return
		}

		if len([]rune(newGroupName)) > 80 {
			http.Error(w, "group name must be 80 characters or fewer", http.StatusBadRequest)
			return
		}

		var memberCount int
		if err := db.QueryRow(`
			SELECT COUNT(*)
			FROM conversation_members
			WHERE conversation_id = ?
		`, convID).Scan(&memberCount); err != nil {
			http.Error(w, "could not load conversation members", http.StatusInternalServerError)
			return
		}
		if memberCount <= 2 {
			http.Error(w, "group name can only be changed for group conversations", http.StatusBadRequest)
			return
		}

		var currentGroupName sql.NullString
		if err := db.QueryRow(`SELECT group_name FROM conversations WHERE id = ?`, convID).Scan(&currentGroupName); err != nil {
			http.Error(w, "conversation not found", http.StatusNotFound)
			return
		}
		if strings.TrimSpace(currentGroupName.String) == newGroupName {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "could not update group name", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		if _, err := tx.Exec(`UPDATE conversations SET group_name = ? WHERE id = ?`, newGroupName, convID); err != nil {
			http.Error(w, "could not update group name", http.StatusInternalServerError)
			return
		}

		changeMessage := `Group name changed to "` + newGroupName + `"`
		if err := database.SaveSystemMessageTx(tx, uuid.New().String(), convID, changeMessage, time.Now().Unix()); err != nil {
			http.Error(w, "could not record group name change", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "could not update group name", http.StatusInternalServerError)
			return
		}

		notifyConversationMembers(db, hub, convID)
		w.WriteHeader(http.StatusNoContent)
	}
}


func RemoveConversationMembersHandler(db *sql.DB, hub *messaging.Hub) http.HandlerFunc {
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

		var body removeConversationMembersReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		seen := map[string]struct{}{}
		targetIDs := make([]string, 0, len(body.MemberIDs))
		for _, rawID := range body.MemberIDs {
			targetID := strings.TrimSpace(rawID)
			if targetID == "" {
				continue
			}
			if _, exists := seen[targetID]; exists {
				continue
			}
			seen[targetID] = struct{}{}
			targetIDs = append(targetIDs, targetID)
		}

		if len(targetIDs) == 0 {
			http.Error(w, "at least one member is required", http.StatusBadRequest)
			return
		}

		membersBefore, err := database.GetConversationMembers(db, convID)
		if err != nil {
			http.Error(w, "could not load conversation members", http.StatusInternalServerError)
			return
		}
		if len(membersBefore) <= 2 {
			http.Error(w, "members can only be removed from group conversations", http.StatusBadRequest)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "could not update conversation", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		var actorName string
		if err := tx.QueryRow(`
			SELECT COALESCE(NULLIF(p.display_name, ''), u.username)
			FROM users u
			LEFT JOIN user_profiles p ON p.user_id = u.id
			WHERE u.id = ?
		`, userID).Scan(&actorName); err != nil {
			http.Error(w, "could not load acting user", http.StatusInternalServerError)
			return
		}

		type removalTarget struct {
			UserID string
			Name   string
		}
		targets := make([]removalTarget, 0, len(targetIDs))
		for _, targetID := range targetIDs {
			if targetID == userID {
				http.Error(w, "use leave conversation to remove yourself", http.StatusBadRequest)
				return
			}

			var targetName string
			err := tx.QueryRow(`
				SELECT COALESCE(NULLIF(p.display_name, ''), u.username)
				FROM conversation_members cm
				JOIN users u ON u.id = cm.user_id
				LEFT JOIN user_profiles p ON p.user_id = u.id
				WHERE cm.conversation_id = ? AND cm.user_id = ?
			`, convID, targetID).Scan(&targetName)
			if err == sql.ErrNoRows {
				http.Error(w, "member not found in conversation", http.StatusBadRequest)
				return
			}
			if err != nil {
				http.Error(w, "could not load conversation members", http.StatusInternalServerError)
				return
			}

			targets = append(targets, removalTarget{UserID: targetID, Name: targetName})
		}

		remainingMembers := len(membersBefore) - len(targets)
		if remainingMembers < 2 {
			http.Error(w, "group conversations must keep at least two members", http.StatusBadRequest)
			return
		}

		now := time.Now().Unix()
		for _, target := range targets {
			if err := database.SaveSystemMessageTx(tx, uuid.New().String(), convID, actorName+" removed "+target.Name+" from the conversation", now); err != nil {
				http.Error(w, "could not record member removal", http.StatusInternalServerError)
				return
			}

			queries := []string{
				`DELETE FROM conversation_pending_room_keys WHERE conversation_id = ? AND user_id = ?`,
				`DELETE FROM conversation_keys WHERE conversation_id = ? AND user_id = ?`,
				`DELETE FROM conversation_members WHERE conversation_id = ? AND user_id = ?`,
			}
			for _, query := range queries {
				if _, err := tx.Exec(query, convID, target.UserID); err != nil {
					http.Error(w, "could not remove member", http.StatusInternalServerError)
					return
				}
			}
		}

		if remainingMembers <= 2 {
			if _, err := tx.Exec(`UPDATE conversations SET group_name = '' WHERE id = ?`, convID); err != nil {
				http.Error(w, "could not update conversation", http.StatusInternalServerError)
				return
			}
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "could not update conversation", http.StatusInternalServerError)
			return
		}

		notifyConversationUsers(hub, membersBefore, convID)
		w.WriteHeader(http.StatusNoContent)
	}
}


func AddConversationMembersHandler(db *sql.DB, hub *messaging.Hub) http.HandlerFunc {
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

		var body addConversationMembersReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.RoomKey) == "" {
			http.Error(w, "missing room key", http.StatusBadRequest)
			return
		}

		membersBefore, err := database.GetConversationMembers(db, convID)
		if err != nil {
			http.Error(w, "could not load conversation members", http.StatusInternalServerError)
			return
		}
		if len(membersBefore) <= 2 {
			http.Error(w, "members can only be added to group conversations", http.StatusBadRequest)
			return
		}

		roomKeyOK, err := database.VerifyConversationRoomKey(db, convID, body.RoomKey)
		if err == sql.ErrNoRows {
			http.Error(w, "room key verifier not set", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "could not verify room key", http.StatusInternalServerError)
			return
		}
		if !roomKeyOK {
			http.Error(w, "incorrect room key", http.StatusUnauthorized)
			return
		}

		seenUsernames := map[string]struct{}{}
		targetUsernames := make([]string, 0, len(body.MemberIDs))
		for _, rawUsername := range body.MemberIDs {
			username := strings.TrimSpace(rawUsername)
			if username == "" {
				continue
			}
			if _, exists := seenUsernames[username]; exists {
				continue
			}
			seenUsernames[username] = struct{}{}
			targetUsernames = append(targetUsernames, username)
		}
		if len(targetUsernames) == 0 {
			http.Error(w, "at least one member is required", http.StatusBadRequest)
			return
		}

		existingMembers := map[string]struct{}{}
		for _, memberID := range membersBefore {
			existingMembers[memberID] = struct{}{}
		}

		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "could not update conversation", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		var actorName string
		if err := tx.QueryRow(`
			SELECT COALESCE(NULLIF(p.display_name, ''), u.username)
			FROM users u
			LEFT JOIN user_profiles p ON p.user_id = u.id
			WHERE u.id = ?
		`, userID).Scan(&actorName); err != nil {
			http.Error(w, "could not load acting user", http.StatusInternalServerError)
			return
		}

		type additionTarget struct {
			UserID   string
			Name     string
			Username string
		}
		targets := make([]additionTarget, 0, len(targetUsernames))
		for _, username := range targetUsernames {
			var target additionTarget
			err := tx.QueryRow(`
				SELECT
					u.id,
					COALESCE(NULLIF(p.display_name, ''), u.username),
					u.username
				FROM users u
				LEFT JOIN user_profiles p ON p.user_id = u.id
				WHERE u.username = ?
			`, username).Scan(&target.UserID, &target.Name, &target.Username)
			if err == sql.ErrNoRows {
				http.Error(w, "user not found: "+username, http.StatusBadRequest)
				return
			}
			if err != nil {
				http.Error(w, "could not load user", http.StatusInternalServerError)
				return
			}

			if target.UserID == userID {
				http.Error(w, "you cannot add yourself to the conversation", http.StatusBadRequest)
				return
			}
			if _, exists := existingMembers[target.UserID]; exists {
				http.Error(w, "user is already in the conversation", http.StatusBadRequest)
				return
			}

			exists, accepted, _, err := database.FriendshipExists(db, userID, target.UserID)
			if err != nil {
				http.Error(w, "could not verify friendship", http.StatusInternalServerError)
				return
			}
			if !exists || !accepted {
				http.Error(w, "you can only add accepted friends", http.StatusBadRequest)
				return
			}

			targets = append(targets, target)
		}

		now := time.Now().Unix()
		addedUserIDs := make([]string, 0, len(targets))
		for _, target := range targets {
			if _, err := tx.Exec(`INSERT INTO conversation_members (conversation_id, user_id, joined_at) VALUES (?, ?, ?)`, convID, target.UserID, now); err != nil {
				http.Error(w, "could not add member", http.StatusInternalServerError)
				return
			}

			if _, err := tx.Exec(`
				INSERT INTO conversation_pending_room_keys (conversation_id, user_id, room_key)
				VALUES (?, ?, ?)
			`, convID, target.UserID, body.RoomKey); err != nil {
				http.Error(w, "could not store room key", http.StatusInternalServerError)
				return
			}

			if err := database.SaveSystemMessageTx(tx, uuid.New().String(), convID, actorName+" added "+target.Name+" to the conversation", now); err != nil {
				http.Error(w, "could not record member addition", http.StatusInternalServerError)
				return
			}

			addedUserIDs = append(addedUserIDs, target.UserID)
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "could not update conversation", http.StatusInternalServerError)
			return
		}

		notifyConversationUsers(hub, append(membersBefore, addedUserIDs...), convID)
		w.WriteHeader(http.StatusNoContent)
	}
}




func SetMessageLifetimeHandler(db *sql.DB, hub *messaging.Hub) http.HandlerFunc {
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

        if !isAllowedMessageLifetime(body.MessageLifetime) {
			log.Println("[Lifetime] invalid lifetime value:", body.MessageLifetime)
			http.Error(w, "invalid message lifetime", http.StatusBadRequest)
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

		now := time.Now().Unix()
		result, err := db.Exec(`
			DELETE FROM messages
			WHERE conversation_id = ?
			  AND expires_at IS NOT NULL
			  AND expires_at <= ?
		`, convID, now)
		if err != nil {
			log.Println("[Lifetime] failed to purge expired messages:", err)
			http.Error(w, "could not purge expired messages", http.StatusInternalServerError)
			return
		}

		if rowsDeleted, err := result.RowsAffected(); err == nil && rowsDeleted > 0 {
			log.Printf("[Lifetime] purged %d newly expired message(s) for conversation %s\n", rowsDeleted, convID)
		}

		notifyConversationMembers(db, hub, convID)
		log.Println("[Lifetime] success")
		w.WriteHeader(http.StatusNoContent)
	}
}

func MessageHandler(db *sql.DB, hub *messaging.Hub) http.HandlerFunc {
	editHandler := EditMessageHandler(db, hub)
	deleteHandler := DeleteMessageHandler(db, hub)

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPatch:
			editHandler(w, r)
		case http.MethodDelete:
			deleteHandler(w, r)
		default:
			w.Header().Set("Allow", "DELETE, PATCH")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func EditMessageHandler(db *sql.DB, hub *messaging.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserIDFromContext(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		messageID := r.PathValue("id")
		if messageID == "" {
			http.Error(w, "missing message id", http.StatusBadRequest)
			return
		}

		var req editMessageReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		if req.Ciphertext == "" {
			http.Error(w, "missing ciphertext", http.StatusBadRequest)
			return
		}

		convID, err := database.UpdateMessage(db, messageID, userID, req.Ciphertext)
		if err == sql.ErrNoRows {
			http.Error(w, "message not found or not yours", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "could not edit message", http.StatusInternalServerError)
			return
		}

		notifyConversationMembers(db, hub, convID)
		w.WriteHeader(http.StatusNoContent)
	}
}

func DeleteMessageHandler(db *sql.DB, hub *messaging.Hub) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        messageID := r.PathValue("id")
        if messageID == "" {
            http.Error(w, "missing message id", http.StatusBadRequest)
            return
        }
        // Get convID before deleting so we can notify members after
        convID, err := database.GetConversationIDForMessage(db, messageID)
        if err != nil {
            http.Error(w, "message not found", http.StatusNotFound)
            return
        }

        if !database.IsUserInConversation(db, userID, convID) {
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }

        result, err := db.Exec(`
            DELETE FROM messages
            WHERE sender_id = ? AND id = ?
        `, userID, messageID)
        if err != nil {
            http.Error(w, "could not delete message", http.StatusInternalServerError)
            return
        }

        rows, _ := result.RowsAffected()
        if rows == 0 {
            http.Error(w, "message not found or not yours", http.StatusNotFound)
            return
        }

        notifyConversationMembers(db, hub, convID)
        w.WriteHeader(http.StatusNoContent)
    }
}

// UploadGroupPictureHandler accepts a multipart image upload and stores it as a
// base64 data URL on the conversation. Any member of the group may change it.
func UploadGroupPictureHandler(db *sql.DB, hub *messaging.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

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

		// Only group conversations (member_count > 2) get a group picture.
		var memberCount int
		if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ?`, convID).Scan(&memberCount); err != nil {
			http.Error(w, "could not load conversation", http.StatusInternalServerError)
			return
		}
		if memberCount <= 2 {
			http.Error(w, "group picture can only be set for group conversations", http.StatusBadRequest)
			return
		}

		if r.Method == http.MethodDelete {
			if _, err := db.Exec(`UPDATE conversations SET group_picture_url = '' WHERE id = ?`, convID); err != nil {
				http.Error(w, "server error", http.StatusInternalServerError)
				return
			}
			notifyConversationMembers(db, hub, convID)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
		if err := r.ParseMultipartForm(2 << 20); err != nil {
			http.Error(w, "file too large (max 2 MB)", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("picture")
		if err != nil {
			http.Error(w, "picture field required", http.StatusBadRequest)
			return
		}
		defer file.Close()

		contentType := header.Header.Get("Content-Type")
		allowed := map[string]bool{
			"image/jpeg": true,
			"image/png":  true,
			"image/gif":  true,
			"image/webp": true,
		}
		if !allowed[contentType] {
			http.Error(w, "only jpeg, png, gif, and webp images are accepted", http.StatusBadRequest)
			return
		}

		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		encoded := base64.StdEncoding.EncodeToString(data)
		dataURL := "data:" + contentType + ";base64," + encoded

		if _, err := db.Exec(`UPDATE conversations SET group_picture_url = ? WHERE id = ?`, dataURL, convID); err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"Group picture updated.","group_picture_url":"` + strings.ReplaceAll(dataURL, `"`, `\"`) + `"}`))

		// Notify all members so their conversation list and header refresh.
		notifyConversationMembers(db, hub, convID)
	}
}
