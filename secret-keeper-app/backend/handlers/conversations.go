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

        convID := uuid.New().String()
        now := time.Now().Unix()

        _, err := db.Exec(`INSERT INTO conversations (id, created_at) VALUES (?, ?)`, convID, now)
        if err != nil {
            http.Error(w, "could not create conversation", http.StatusInternalServerError)
            return
        }

        // add the requester
        members := append(req.MemberIDs, userID)
        // deduplicate in case requester included themselves
        seen := map[string]bool{}
        for _, id := range members {
            if seen[id] { continue }
            seen[id] = true
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