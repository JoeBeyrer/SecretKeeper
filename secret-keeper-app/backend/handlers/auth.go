package handlers

import (
    "database/sql"
    "encoding/json"
    "context"
    "net/http"
    "time"

    "secret-keeper-app/backend/auth"
    "secret-keeper-app/backend/database"
    "github.com/google/uuid"
)

type contextKey string
const userIDKey contextKey = "userID"

type registerReq struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

func RegisterHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req registerReq
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }

        // basic validation
        // might want to change since validation happens before sending request
        if req.Username == "" || req.Password == "" || len(req.Password) < 8 {
            http.Error(w, "invalid username or password (min 8 chars)", http.StatusBadRequest)
            return
        }

        hashed, err := auth.HashPassword(req.Password)
        if err != nil {
            http.Error(w, "server error", http.StatusInternalServerError)
            return
        }

        id := uuid.New().String()
        now := time.Now().Unix()
        _, err = db.Exec(`INSERT INTO users (id, username, email, password_hash, created_at) VALUES (?, ?, ?, ?, ?)`, id, req.Username, req.Email, hashed, now)
        if err != nil {
            http.Error(w, "could not create user", http.StatusConflict)
            return
        }

        w.WriteHeader(http.StatusCreated)
        w.Write([]byte(`{"user_id":"` + id + `"}`))
    }
}

type loginReq struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

func LoginHandler(db *sql.DB, sessionTTL time.Duration) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req loginReq
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }

        var userID string
        var pwHash []byte
        err := db.QueryRow(`SELECT id, password_hash FROM users WHERE username = ?`, req.Username).Scan(&userID, &pwHash)
        if err != nil {
            http.Error(w, "invalid credentials", http.StatusUnauthorized)
            return
        }

        if err := auth.CheckPasswordHash(pwHash, req.Password); err != nil {
            http.Error(w, "invalid credentials", http.StatusUnauthorized)
            return
        }

        sessionID, expires, err := database.CreateSession(db, userID, sessionTTL)
        if err != nil {
            http.Error(w, "could not create session", http.StatusInternalServerError)
            return
        }

        // set secure cookie
        cookie := &http.Cookie{
            Name:     "sk_session",
            Value:    sessionID,
            Path:     "/",
            HttpOnly: true,
            Secure:   false, // must be HTTPS - set to False only for testing
            SameSite: http.SameSiteLaxMode,
            Expires:  time.Unix(expires, 0),
        }
        http.SetCookie(w, cookie)
    }
}

func AuthMiddleware(db *sql.DB) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

            cookie, err := r.Cookie("sk_session")
            if err != nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }

            sessionID := cookie.Value

            userID, err := database.GetUserIDForSession(db, sessionID)
            if err != nil {
                http.Error(w, "invalid session", http.StatusUnauthorized)
                return
            }

            ctx := context.WithValue(r.Context(), userIDKey, userID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func GetUserIDFromContext(r *http.Request) (string, bool) {
    userID, ok := r.Context().Value(userIDKey).(string)
    return userID, ok
}
