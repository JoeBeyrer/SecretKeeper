package handlers

import (
    "database/sql"
    "encoding/base64"
    "encoding/json"
    "io"
    "net/http"
    "strings"

    "secret-keeper-app/backend/database"
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
