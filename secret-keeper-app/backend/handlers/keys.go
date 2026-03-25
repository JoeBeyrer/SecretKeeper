package handlers

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "secret-keeper-app/backend/database"
)

func SaveKeysHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        var body struct {
            PublicKey           string `json:"public_key"`
            EncryptedPrivateKey string `json:"encrypted_private_key"`
        }
        if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }
        if body.PublicKey == "" || body.EncryptedPrivateKey == "" {
            http.Error(w, "missing keys", http.StatusBadRequest)
            return
        }

        if err := database.SaveUserKeys(db, userID, body.PublicKey, body.EncryptedPrivateKey); err != nil {
            http.Error(w, "could not save keys", http.StatusInternalServerError)
            return
        }

        w.WriteHeader(http.StatusNoContent)
    }
}

func GetKeysHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        publicKey, encryptedPrivateKey, err := database.GetUserKeys(db, userID)
        if err == sql.ErrNoRows {
            http.Error(w, "no keys found", http.StatusNotFound)
            return
        }
        if err != nil {
            http.Error(w, "could not fetch keys", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{
            "public_key":            publicKey,
            "encrypted_private_key": encryptedPrivateKey,
        })
    }
}

func GetPublicKeyHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        _, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        username := r.PathValue("username")
        if username == "" {
            http.Error(w, "missing username", http.StatusBadRequest)
            return
        }

        var userID string
        err := db.QueryRow(`SELECT id FROM users WHERE username = ?`, username).Scan(&userID)
        if err != nil {
            http.Error(w, "user not found", http.StatusNotFound)
            return
        }

        publicKey, err := database.GetUserPublicKey(db, userID)
        if err != nil {
            http.Error(w, "no public key found for user", http.StatusNotFound)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{
			"public_key": publicKey,
			"user_id":    userID,
		})
    }
}

func SaveConversationKeyHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        convID := r.PathValue("id")
        if !database.IsUserInConversation(db, userID, convID) {
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }

        var body struct {
            Keys []struct {
                UserID       string `json:"user_id"`
                EncryptedKey string `json:"encrypted_key"`
            } `json:"keys"`
        }
        if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }

        for _, k := range body.Keys {
            if err := database.SaveConversationKey(db, convID, k.UserID, k.EncryptedKey); err != nil {
                http.Error(w, "could not save key", http.StatusInternalServerError)
                return
            }
        }

        w.WriteHeader(http.StatusNoContent)
    }
}

func GetConversationKeyHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        convID := r.PathValue("id")
        if !database.IsUserInConversation(db, userID, convID) {
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }

        key, err := database.GetConversationKey(db, convID, userID)
        if err == sql.ErrNoRows {
            http.Error(w, "no key found", http.StatusNotFound)
            return
        }
        if err != nil {
            http.Error(w, "could not fetch key", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{
            "encrypted_key": key,
        })
    }
}