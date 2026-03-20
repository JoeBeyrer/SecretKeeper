package handlers

import (
    "crypto/rand"
    "database/sql"
    "encoding/base64"
    "encoding/hex"
    "encoding/json"
    "io"
    "log"
    "net/http"
    "strings"
    "time"

    "secret-keeper-app/backend/auth"
    "secret-keeper-app/backend/database"
    "secret-keeper-app/backend/email"
    "github.com/google/uuid"
)


type profileResponse struct {
    Username           string `json:"username"`
    Email              string `json:"email"`
    DisplayName        string `json:"display_name"`
    Bio                string `json:"bio"`
    ProfilePictureURL  string `json:"profile_picture_url"`
}

type updateProfileReq struct {
    DisplayName  string `json:"display_name"`
    Bio          string `json:"bio"`
    ClearPicture bool   `json:"clear_picture"`
}

func GetProfileHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }

        userID, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        var username, email string
        err := db.QueryRow(`SELECT username, email FROM users WHERE id = ?`, userID).Scan(&username, &email)
        if err != nil {
            http.Error(w, "user not found", http.StatusNotFound)
            return
        }

        var displayName, bio, profilePictureURL string
        err = db.QueryRow(
            `SELECT display_name, bio, profile_picture_url FROM user_profiles WHERE user_id = ?`, userID,
        ).Scan(&displayName, &bio, &profilePictureURL)

        if err == sql.ErrNoRows {
            // Create a blank profile row on first fetch.
            db.Exec(`INSERT INTO user_profiles (user_id, display_name, bio, profile_picture_url) VALUES (?, '', '', '')`, userID)
        } else if err != nil {
            http.Error(w, "server error", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(profileResponse{
            Username:          username,
            Email:             email,
            DisplayName:       displayName,
            Bio:               bio,
            ProfilePictureURL: profilePictureURL,
        })
    }
}


// Updates display name and bio. Picture is handled separately.
func UpdateProfileHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPut {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }

        userID, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        var req updateProfileReq
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }

        if req.ClearPicture {
            _, err := db.Exec(`
                INSERT INTO user_profiles (user_id, display_name, bio, profile_picture_url)
                VALUES (?, ?, ?, '')
                ON CONFLICT(user_id) DO UPDATE SET
                    display_name = excluded.display_name,
                    bio = excluded.bio,
                    profile_picture_url = ''
            `, userID, req.DisplayName, req.Bio)
            if err != nil {
                http.Error(w, "server error", http.StatusInternalServerError)
                return
            }
        } else {
            _, err := db.Exec(`
                INSERT INTO user_profiles (user_id, display_name, bio, profile_picture_url)
                VALUES (?, ?, ?, '')
                ON CONFLICT(user_id) DO UPDATE SET
                    display_name = excluded.display_name,
                    bio = excluded.bio
            `, userID, req.DisplayName, req.Bio)
            if err != nil {
                http.Error(w, "server error", http.StatusInternalServerError)
                return
            }
        }

        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"message":"Profile updated successfully."}`))
    }
}

// Accepts an image file and converts it to a base64 data URL
// Max file size: 2 MB. Accepted types: jpeg, png, gif, webp.

func UploadProfilePictureHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }

        userID, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        // Limit request body to 2 MB.
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

        _, err = db.Exec(`
            INSERT INTO user_profiles (user_id, display_name, bio, profile_picture_url)
            VALUES (?, '', '', ?)
            ON CONFLICT(user_id) DO UPDATE SET profile_picture_url = excluded.profile_picture_url
        `, userID, dataURL)

        if err != nil {
            http.Error(w, "server error", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"message":"Profile picture updated successfully.","profile_picture_url":"` + strings.ReplaceAll(dataURL, `"`, `\"`) + `"}`))
    }
}

// Updates username, email, and/or password for the logged-in user.
type updateAccountReq struct {
    NewUsername string `json:"new_username"`
    NewEmail    string `json:"new_email"`
    NewPassword string `json:"new_password"`
}

func UpdateAccountHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPut {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }

        userID, ok := GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        var req updateAccountReq
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }

        if req.NewPassword != "" {
            if len(req.NewPassword) < 8 {
                http.Error(w, "new password must be at least 8 characters", http.StatusBadRequest)
                return
            }

            hashed, err := auth.HashPassword(req.NewPassword)
            if err != nil {
                http.Error(w, "server error", http.StatusInternalServerError)
                return
            }
            if _, err := db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, hashed, userID); err != nil {
                http.Error(w, "server error", http.StatusInternalServerError)
                return
            }
        }

        if req.NewUsername != "" {
            if len(req.NewUsername) < 3 {
                http.Error(w, "username must be at least 3 characters", http.StatusBadRequest)
                return
            }
            _, err := db.Exec(`UPDATE users SET username = ? WHERE id = ?`, req.NewUsername, userID)
            if err != nil {
                http.Error(w, "username already taken", http.StatusConflict)
                return
            }
        }

        // Email change — send verification to new address instead of updating directly.
        emailPending := false
        if req.NewEmail != "" {
            // Check the new email isn't the same as the current one.
            var currentEmail string
            db.QueryRow(`SELECT email FROM users WHERE id = ?`, userID).Scan(&currentEmail)
            if req.NewEmail == currentEmail {
                http.Error(w, "that is already your current email", http.StatusBadRequest)
                return
            }

            var existing string
            err := db.QueryRow(`SELECT id FROM users WHERE email = ? AND id != ?`, req.NewEmail, userID).Scan(&existing)
            if err == nil {
                http.Error(w, "email already in use", http.StatusConflict)
                return
            }

            // Delete any existing pending email change for this user.
            db.Exec(`DELETE FROM email_verifications WHERE user_id = ? AND new_email != ''`, userID)

            tokenBytes := make([]byte, 32)
            if _, err := rand.Read(tokenBytes); err != nil {
                http.Error(w, "server error", http.StatusInternalServerError)
                return
            }
            token := hex.EncodeToString(tokenBytes)
            verifyID := uuid.New().String()
            now := time.Now().Unix()
            expiresAt := time.Now().Add(24 * time.Hour).Unix()

            _, err = db.Exec(
                `INSERT INTO email_verifications (id, user_id, token, created_at, expires_at, new_email) VALUES (?, ?, ?, ?, ?, ?)`,
                verifyID, userID, token, now, expiresAt, req.NewEmail,
            )
            if err != nil {
                http.Error(w, "server error", http.StatusInternalServerError)
                return
            }

            if err := email.SendEmailChangeVerificationEmail(req.NewEmail, token); err != nil {
                log.Printf("[EMAIL CHANGE] send failed for user %s: %v", userID, err)
                db.Exec(`DELETE FROM email_verifications WHERE id = ?`, verifyID)
                http.Error(w, "failed to send verification email, please try again", http.StatusInternalServerError)
                return
            }

            log.Printf("[EMAIL CHANGE] verification sent to %s for user %s", req.NewEmail, userID)
            emailPending = true
        }

        w.Header().Set("Content-Type", "application/json")
        if emailPending {
            w.Write([]byte(`{"message":"Account updated. Check your new email address to confirm the email change."}`))
        } else {
            w.Write([]byte(`{"message":"Account updated successfully."}`))
        }
    }
}

func VerifyEmailChangeHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := r.URL.Query().Get("token")
        if token == "" {
            http.Error(w, "token required", http.StatusBadRequest)
            return
        }

        var verifyID, userID, newEmail string
        var expiresAt int64
        err := db.QueryRow(
            `SELECT id, user_id, expires_at, new_email FROM email_verifications WHERE token = ? AND new_email != ''`, token,
        ).Scan(&verifyID, &userID, &expiresAt, &newEmail)

        if err != nil {
            http.Error(w, "invalid or expired verification link", http.StatusUnprocessableEntity)
            return
        }
        if time.Now().Unix() > expiresAt {
            db.Exec(`DELETE FROM email_verifications WHERE id = ?`, verifyID)
            http.Error(w, "verification link has expired", http.StatusUnprocessableEntity)
            return
        }

        if _, err := db.Exec(`UPDATE users SET email = ? WHERE id = ?`, newEmail, userID); err != nil {
            http.Error(w, "server error", http.StatusInternalServerError)
            return
        }

        db.Exec(`DELETE FROM email_verifications WHERE id = ?`, verifyID)
        log.Printf("[EMAIL CHANGE] email updated to %s for user %s", newEmail, userID)

        http.Redirect(w, r, "http://localhost:4200/profile?email_updated=true", http.StatusFound)
    }
}

// Deletes the session from the database and clears the cookie.
func LogoutHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }

        cookie, err := r.Cookie("sk_session")
        if err != nil {
            http.Error(w, "not logged in", http.StatusUnauthorized)
            return
        }

        // Delete the session from the database.
        database.DeleteSession(db, cookie.Value)

        // Clear the cookie in the browser by setting MaxAge to -1.
        http.SetCookie(w, &http.Cookie{
            Name:     "sk_session",
            Value:    "",
            Path:     "/",
            HttpOnly: true,
            Secure:   false, // set to true when using HTTPS
            MaxAge:   -1,
        })

        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"message":"Logged out successfully."}`))
    }
}
