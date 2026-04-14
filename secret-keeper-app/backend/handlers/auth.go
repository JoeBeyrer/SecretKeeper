package handlers

import (
    "context"
    "crypto/rand"
    "database/sql"
    "encoding/hex"
    "encoding/json"
    "log"
    "net/http"
    "time"

    "secret-keeper-app/backend/auth"
    "secret-keeper-app/backend/database"
    "secret-keeper-app/backend/email"
    "github.com/google/uuid"
)

type contextKey string
const userIDKey contextKey = "userID"

type registerReq struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

var SendVerificationEmail = email.SendVerificationEmail

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
        _, err = db.Exec(`INSERT INTO users (id, username, email, password_hash, created_at, email_verified) VALUES (?, ?, ?, ?, ?, 0)`, id, req.Username, req.Email, hashed, now)
        if err != nil {
            http.Error(w, "could not create user", http.StatusConflict)
            return
        }

        tokenBytes := make([]byte, 32)
        if _, err := rand.Read(tokenBytes); err != nil {
            http.Error(w, "server error", http.StatusInternalServerError)
            return
        }
        token := hex.EncodeToString(tokenBytes)
        verifyID := uuid.New().String()
        expiresAt := time.Now().Add(24 * time.Hour).Unix()

        _, err = db.Exec(`INSERT INTO email_verifications (id, user_id, token, created_at, expires_at) VALUES (?, ?, ?, ?, ?)`, verifyID, id, token, now, expiresAt)
        if err != nil {
            log.Printf("[REGISTER] failed to store verification token for user %s: %v", id, err)
        } else {
            if err := SendVerificationEmail(req.Email, token); err != nil {
                log.Printf("[REGISTER] verification email failed for user %s: %v", id, err)
            } else {
                log.Printf("[REGISTER] verification email sent to %s", req.Email)
            }
        }

        w.WriteHeader(http.StatusCreated)
        w.Write([]byte(`{"user_id":"` + id + `","message":"Account created. Please check your email to verify your address before logging in."}`))
    }
}

func VerifyEmailHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := r.URL.Query().Get("token")
        if token == "" {
            http.Error(w, "token required", http.StatusBadRequest)
            return
        }

        var verifyID, userID string
        var expiresAt int64
        err := db.QueryRow(
            `SELECT id, user_id, expires_at FROM email_verifications WHERE token = ?`, token,
        ).Scan(&verifyID, &userID, &expiresAt)

        if err != nil {
            http.Error(w, "invalid or expired verification link", http.StatusUnprocessableEntity)
            return
        }
        if time.Now().Unix() > expiresAt {
            db.Exec(`DELETE FROM email_verifications WHERE id = ?`, verifyID)
            http.Error(w, "verification link has expired", http.StatusUnprocessableEntity)
            return
        }

        if _, err := db.Exec(`UPDATE users SET email_verified = 1 WHERE id = ?`, userID); err != nil {
            http.Error(w, "server error", http.StatusInternalServerError)
            return
        }

        db.Exec(`DELETE FROM email_verifications WHERE id = ?`, verifyID)

        log.Printf("[VERIFY EMAIL] user %s verified successfully", userID)

        // Redirect to the login page
        http.Redirect(w, r, "http://localhost:4200/login?verified=true", http.StatusFound)
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
        var emailVerified int
        err := db.QueryRow(`SELECT id, password_hash, email_verified FROM users WHERE username = ?`, req.Username).Scan(&userID, &pwHash, &emailVerified)
        if err != nil {
            http.Error(w, "invalid credentials", http.StatusUnauthorized)
            return
        }

        if err := auth.CheckPasswordHash(pwHash, req.Password); err != nil {
            http.Error(w, "invalid credentials", http.StatusUnauthorized)
            return
        }

        if emailVerified == 0 {
            http.Error(w, "please verify your email address before logging in", http.StatusForbidden)
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

// SetTestUserID is a test helper injecting a userID into a request context -
// simulating what AuthMiddleware does.
func SetTestUserID(r *http.Request, userID string) *http.Request {
    ctx := context.WithValue(r.Context(), userIDKey, userID)
    return r.WithContext(ctx)
}
