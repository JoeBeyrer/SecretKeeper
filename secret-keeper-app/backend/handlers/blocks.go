package handlers

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "time"
    "github.com/google/uuid"
	"secret-keeper-app/backend/database"
)

func BlockUser(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        BlockerID, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        var body struct {
            BlockeeID string `json:"blockee_id"`
        }
        if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
            http.Error(w, "Invalid request", http.StatusBadRequest)
            return
        }
        if BlockerID == body.BlockeeID {
            http.Error(w, "Cannot block yourself", http.StatusBadRequest)
            return
        }
        id := uuid.New().String()
        _, err := db.ExecContext(r.Context(), `
            INSERT INTO blocks (id, blocker_id, blockee_id, created_at)
            VALUES (?, ?, ?, ?)
        `, id, BlockerID, body.BlockeeID, time.Now().Unix())
        if err != nil {
            http.Error(w, "Already blocked", http.StatusConflict)
            return
        }
        database.RemoveFriend(db, BlockerID, body.BlockeeID)

        w.WriteHeader(http.StatusCreated)
    }
}

func UnblockUser(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        blockerID, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        blockeeID := r.PathValue("blockee_id")
        if blockeeID == "" {
            http.Error(w, "blockee_id required", http.StatusBadRequest)
            return
        }
        _, err := db.ExecContext(r.Context(), `
            DELETE FROM blocks
            WHERE blocker_id = ? AND blockee_id = ?
        `, blockerID, blockeeID)
        if err != nil {
            http.Error(w, "Error unblocking user", http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusNoContent)
    }
}