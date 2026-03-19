package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"secret-keeper-app/backend/auth"
	"secret-keeper-app/backend/email"
	"github.com/google/uuid"
)

// Rate limiter tracking stored in memory: resets if the server restarts
type rateLimitEntry struct {
	count     int
	windowEnd time.Time
}

var (
	rateLimitMu      sync.Mutex
	rateLimitByEmail = make(map[string]*rateLimitEntry)
)

const (
	maxRequestsPerHour = 3
	rateLimitWindow    = time.Hour
)

func checkRateLimit(emailAddr string) bool {
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()

	now := time.Now()
	entry, exists := rateLimitByEmail[emailAddr]

	if !exists || now.After(entry.windowEnd) {
		rateLimitByEmail[emailAddr] = &rateLimitEntry{
			count:     1,
			windowEnd: now.Add(rateLimitWindow),
		}
		return true
	}

	if entry.count >= maxRequestsPerHour {
		return false
	}

	entry.count++
	return true
}


func archiveToken(db *sql.DB, resetID, userID, token string, createdAt, expiresAt int64, reason string) {
	archivedAt := time.Now().Unix()
	db.Exec(
		`INSERT INTO password_reset_audit (id, user_id, token, created_at, expires_at, archived_at, reason)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		resetID, userID, token, createdAt, expiresAt, archivedAt, reason,
	)
	db.Exec(`DELETE FROM password_resets WHERE id = ?`, resetID)
}

// StartTokenCleanup launches a background goroutine that sweeps expired unused
// tokens from password_resets into the audit table once per hour.
func StartTokenCleanup(db *sql.DB) {
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			cleanupExpiredTokens(db)
		}
	}()
}

func cleanupExpiredTokens(db *sql.DB) {
	now := time.Now().Unix()
	rows, err := db.Query(
		`SELECT id, user_id, token, created_at, expires_at FROM password_resets
		 WHERE expires_at < ? AND used = 0`, now,
	)
	if err != nil {
		log.Printf("[TOKEN CLEANUP] query error: %v", err)
		return
	}
	defer rows.Close()

	type expiredRow struct {
		id, userID, token    string
		createdAt, expiresAt int64
	}
	var expired []expiredRow

	for rows.Next() {
		var r expiredRow
		if err := rows.Scan(&r.id, &r.userID, &r.token, &r.createdAt, &r.expiresAt); err == nil {
			expired = append(expired, r)
		}
	}

	for _, r := range expired {
		archiveToken(db, r.id, r.userID, r.token, r.createdAt, r.expiresAt, "expired")
	}

	if len(expired) > 0 {
		log.Printf("[TOKEN CLEANUP] archived %d expired token(s)", len(expired))
	}
}

// Requests

type forgotPasswordReq struct {
	Email string `json:"email"`
}

type resetPasswordReq struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// ForgotPasswordHandler  POST /api/password-reset/request 
// 1. Rate-limits by email (max 3 per hour).
// 2. Checks the email exists and is verified before doing anything.
// 3. Generates a token, stores it, and emails the reset link.
// Always returns the same generic message to avoid leaking whether an email is registered.

func ForgotPasswordHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req forgotPasswordReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		genericResp := `{"message":"If that email is registered and verified, you will receive a reset link."}`

		// Rate limit — always return the generic message even when blocked so
		// an attacker doesn't know whether they've hit the limit on a real email.
		if !checkRateLimit(req.Email) {
			log.Printf("[PASSWORD RESET] rate limit hit for %s", req.Email)
			w.Write([]byte(genericResp))
			return
		}

		var userID string
		var emailVerified int
		err := db.QueryRow(
			`SELECT id, email_verified FROM users WHERE email = ?`, req.Email,
		).Scan(&userID, &emailVerified)

		if err != nil {
			// Email not found — return generic message.
			w.Write([]byte(genericResp))
			return
		}

		if emailVerified == 0 {
			// Email exists but is not verified. We still return the generic message
			// so we don't leak that the account exists but is unverified.
			log.Printf("[PASSWORD RESET] reset requested for unverified email %s", req.Email)
			w.Write([]byte(genericResp))
			return
		}

		// Archive any existing unused tokens for user so only the latest link works.
		rows, _ := db.Query(
			`SELECT id, user_id, token, created_at, expires_at FROM password_resets
			 WHERE user_id = ? AND used = 0`, userID,
		)
		if rows != nil {
			type oldToken struct {
				id, userID, token    string
				createdAt, expiresAt int64
			}
			var old []oldToken
			for rows.Next() {
				var t oldToken
				if err := rows.Scan(&t.id, &t.userID, &t.token, &t.createdAt, &t.expiresAt); err == nil {
					old = append(old, t)
				}
			}
			rows.Close()
			for _, t := range old {
				archiveToken(db, t.id, t.userID, t.token, t.createdAt, t.expiresAt, "superseded")
			}
		}

		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		token := hex.EncodeToString(tokenBytes)

		now := time.Now().Unix()
		expiresAt := time.Now().Add(1 * time.Hour).Unix()
		resetID := uuid.New().String()

		_, err = db.Exec(
			`INSERT INTO password_resets (id, user_id, token, created_at, expires_at, used)
			 VALUES (?, ?, ?, ?, ?, 0)`,
			resetID, userID, token, now, expiresAt,
		)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		// Send the email. On failure, delete the token row so the user can retry cleanly.
		if err := email.SendPasswordResetEmail(req.Email, token); err != nil {
			log.Printf("[PASSWORD RESET] email send failed for user %s: %v", userID, err)
			db.Exec(`DELETE FROM password_resets WHERE id = ?`, resetID)
			http.Error(w, "failed to send reset email, please try again", http.StatusInternalServerError)
			return
		}

		log.Printf("[PASSWORD RESET] email sent to %s (reset_id=%s)", req.Email, resetID)
		w.Write([]byte(genericResp))
	}
}


func ValidateResetTokenHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "token required", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		var expiresAt int64
		var used int
		err := db.QueryRow(
			`SELECT expires_at, used FROM password_resets WHERE token = ?`, token,
		).Scan(&expiresAt, &used)

		if err != nil || used == 1 || time.Now().Unix() > expiresAt {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(`{"valid":false}`))
			return
		}

		w.Write([]byte(`{"valid":true}`))
	}
}


func ResetPasswordHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req resetPasswordReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		if req.Token == "" || len(req.Password) < 8 {
			http.Error(w, "token and password (min 8 chars) are required", http.StatusBadRequest)
			return
		}

		var resetID, userID, token string
		var createdAt, expiresAt int64
		var used int
		err := db.QueryRow(
			`SELECT id, user_id, token, created_at, expires_at, used
			 FROM password_resets WHERE token = ?`, req.Token,
		).Scan(&resetID, &userID, &token, &createdAt, &expiresAt, &used)

		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnprocessableEntity)
			return
		}
		if used == 1 {
			http.Error(w, "token has already been used", http.StatusUnprocessableEntity)
			return
		}
		if time.Now().Unix() > expiresAt {
			// Archive the expired token before returning.
			archiveToken(db, resetID, userID, token, createdAt, expiresAt, "expired")
			http.Error(w, "token has expired", http.StatusUnprocessableEntity)
			return
		}

		hashed, err := auth.HashPassword(req.Password)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		if _, err = db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, hashed, userID); err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		// Archive all reset tokens for this user as used.
		rows, _ := db.Query(
			`SELECT id, user_id, token, created_at, expires_at FROM password_resets WHERE user_id = ?`, userID,
		)
		if rows != nil {
			type tokenRow struct {
				id, userID, token    string
				createdAt, expiresAt int64
			}
			var all []tokenRow
			for rows.Next() {
				var t tokenRow
				if err := rows.Scan(&t.id, &t.userID, &t.token, &t.createdAt, &t.expiresAt); err == nil {
					all = append(all, t)
				}
			}
			rows.Close()
			for _, t := range all {
				archiveToken(db, t.id, t.userID, t.token, t.createdAt, t.expiresAt, "used")
			}
		}

		// Kill all active sessions — forces re-login with the new password.
		db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"Password updated successfully. Please log in with your new password."}`))
	}
}
