package handlers

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "time"

    "github.com/google/uuid"
)

type createConvReq struct {
    MemberIDs []string `json:"member_ids"`
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

        // No existing conversation found — create a new one
        convID := uuid.New().String()
        now := time.Now().Unix()

        _, err := db.Exec(`INSERT INTO conversations (id, created_at) VALUES (?, ?)`, convID, now)
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
        json.NewEncoder(w).Encode(map[string]string{"conversation_id": convID})
    }
}
