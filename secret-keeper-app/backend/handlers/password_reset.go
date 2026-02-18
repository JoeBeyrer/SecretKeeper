package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"secret-keeper-app/backend/auth"
	"secret-keeper-app/backend/email"
	"github.com/google/uuid"
)

type forgotPasswordReq struct {
	Email string `json:"email"`
}

type resetPasswordReq struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}


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

		
		genericResp := `{"message":"If that email is registered you will receive a reset link."}`

		var userID string
		err := db.QueryRow(`SELECT id FROM users WHERE email = ?`, req.Email).Scan(&userID)
		if err != nil {
			w.Write([]byte(genericResp))
			return
		}

	
		db.Exec(`UPDATE password_resets SET used = 1 WHERE user_id = ? AND used = 0`, userID)

		//Generate a secure random token.
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
			`INSERT INTO password_resets (id, user_id, token, created_at, expires_at, used) VALUES (?, ?, ?, ?, ?, 0)`,
			resetID, userID, token, now, expiresAt,
		)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		//Send the email. Rollback the token row if sending fails
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

		var resetID, userID string
		var expiresAt int64
		var used int
		err := db.QueryRow(
			`SELECT id, user_id, expires_at, used FROM password_resets WHERE token = ?`,
			req.Token,
		).Scan(&resetID, &userID, &expiresAt, &used)

		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnprocessableEntity)
			return
		}
		if used == 1 {
			http.Error(w, "token has already been used", http.StatusUnprocessableEntity)
			return
		}
		if time.Now().Unix() > expiresAt {
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

		//Kept for audit trail
		db.Exec(`UPDATE password_resets SET used = 1 WHERE user_id = ?`, userID)

		//Kill all active sessions — forces re-login with the new password
		db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"Password updated successfully. Please log in with your new password."}`))
	}
}
