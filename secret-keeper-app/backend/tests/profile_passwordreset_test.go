package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"secret-keeper-app/backend/database"
	"secret-keeper-app/backend/handlers"
    "secret-keeper-app/backend/messaging"
	"strings"
	"testing"
	"time"
)

// Create a request with a user ID already in context (simulates AuthMiddleware)
func requestWithUserID(method, target, body, userID string) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	return handlers.SetTestUserID(req, userID) //from auth.go
}


func Test_forgot_password_handler(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	// Empty email body. Should get 400
	body := `{"email":""}`
	req := httptest.NewRequest("POST", "/api/password-reset/request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.ForgotPasswordHandler(db)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty email, got %d", w.Code)
	} else {
		t.Log("empty email correctly returned 400")
	}

	// Unknown email. Should get 200 with message
	body = `{"email":"ghost@nowhere.com"}`
	req = httptest.NewRequest("POST", "/api/password-reset/request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.ForgotPasswordHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for unknown email, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "If that email is registered") {
		t.Fatalf("expected generic message for unknown email, got: %s", w.Body.String())
	} else {
		t.Log("unknown email correctly returned 200 with generic message")
	}

	// Unverified email. Also 200 with message
	db.Exec(`INSERT INTO users (id, username, email, password_hash, created_at, email_verified)
		VALUES ('u1','unverified','unverified@test.com','hash',1700000000,0)`)
	body = `{"email":"unverified@test.com"}`
	req = httptest.NewRequest("POST", "/api/password-reset/request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.ForgotPasswordHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for unverified email, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "If that email is registered") {
		t.Fatalf("expected generic message for unverified email, got: %s", w.Body.String())
	} else {
		t.Log("unverified email correctly returned 200 with generic message")
	}

	// Confirm no token was created for unverified user
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM password_resets WHERE user_id = 'u1'`).Scan(&count)
	if count != 0 {
		t.Fatal("password reset token was created for unverified user — it should not be")
	} else {
		t.Log("no token created for unverified user, as expected")
	}
}

func Test_validate_reset_token_handler(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	// Empty token
	req := httptest.NewRequest("GET", "/api/password-reset/validate?token=", nil)
	w := httptest.NewRecorder()
	handlers.ValidateResetTokenHandler(db)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty token, got %d", w.Code)
	} else {
		t.Log("empty token correctly returned 400")
	}

	// Token does not exist in db
	req = httptest.NewRequest("GET", "/api/password-reset/validate?token=doesnotexist", nil)
	w = httptest.NewRecorder()
	handlers.ValidateResetTokenHandler(db)(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for unknown token, got %d", w.Code)
	} else {
		t.Log("unknown token correctly returned 422")
	}

	// Expired token
	db.Exec(`INSERT INTO users (id, username, email, password_hash, created_at, email_verified)
		VALUES ('u1','user1','user1@test.com','hash',1700000000,1)`)
	db.Exec(`INSERT INTO password_resets (id, user_id, token, created_at, expires_at, used)
		VALUES ('r1','u1','expiredtoken',1700000000,1700003600,0)`)
	req = httptest.NewRequest("GET", "/api/password-reset/validate?token=expiredtoken", nil)
	w = httptest.NewRecorder()
	handlers.ValidateResetTokenHandler(db)(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for expired token, got %d", w.Code)
	} else {
		t.Log("expired token correctly returned 422")
	}

	// Used token
	db.Exec(`INSERT INTO password_resets (id, user_id, token, created_at, expires_at, used)
		VALUES ('r2','u1','usedtoken',1700000000,9999999999,1)`)
	req = httptest.NewRequest("GET", "/api/password-reset/validate?token=usedtoken", nil)
	w = httptest.NewRecorder()
	handlers.ValidateResetTokenHandler(db)(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for used token, got %d", w.Code)
	} else {
		t.Log("used token correctly returned 422")
	}

	// Valid token
	db.Exec(`INSERT INTO password_resets (id, user_id, token, created_at, expires_at, used)
		VALUES ('r3','u1','validtoken',1700000000,9999999999,0)`)
	req = httptest.NewRequest("GET", "/api/password-reset/validate?token=validtoken", nil)
	w = httptest.NewRecorder()
	handlers.ValidateResetTokenHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid token, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"valid":true`) {
		t.Fatalf("expected valid:true in response, got %s", w.Body.String())
	} else {
		t.Log("valid token correctly returned 200 with valid:true")
	}
}

func Test_reset_password_handler(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	db.Exec(`INSERT INTO users (id, username, email, password_hash, created_at, email_verified)
		VALUES ('u1','resetuser','reset@test.com','oldhash',1700000000,1)`)

	// Missing token
	body := `{"token":"","password":"newpassword"}`
	req := httptest.NewRequest("POST", "/api/password-reset/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.ResetPasswordHandler(db)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing token, got %d", w.Code)
	} else {
		t.Log("missing token correctly returned 400")
	}

	// Password too short
	body = `{"token":"sometoken","password":"short"}`
	req = httptest.NewRequest("POST", "/api/password-reset/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.ResetPasswordHandler(db)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for short password, got %d", w.Code)
	} else {
		t.Log("short password correctly returned 400")
	}

	// Invalid token
	body = `{"token":"nonexistenttoken","password":"newpassword"}`
	req = httptest.NewRequest("POST", "/api/password-reset/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.ResetPasswordHandler(db)(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid token, got %d", w.Code)
	} else {
		t.Log("invalid token correctly returned 422")
	}

	// Expired token
	db.Exec(`INSERT INTO password_resets (id, user_id, token, created_at, expires_at, used)
		VALUES ('r1','u1','expiredresettoken',1700000000,1700003600,0)`)
	body = `{"token":"expiredresettoken","password":"newpassword"}`
	req = httptest.NewRequest("POST", "/api/password-reset/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.ResetPasswordHandler(db)(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for expired token, got %d", w.Code)
	} else {
		t.Log("expired token correctly returned 422")
	}

	// Valid reset
	db.Exec(`INSERT INTO password_resets (id, user_id, token, created_at, expires_at, used)
		VALUES ('r2','u1','validresettoken',1700000000,9999999999,0)`)
	body = `{"token":"validresettoken","password":"brandnewpass"}`
	req = httptest.NewRequest("POST", "/api/password-reset/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.ResetPasswordHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid reset, got %d", w.Code)
	} else {
		t.Log("valid reset correctly returned 200")
	}

	// Confirm password hash changed in db
	var newHash string
	db.QueryRow(`SELECT password_hash FROM users WHERE id = 'u1'`).Scan(&newHash)
	if newHash == "oldhash" {
		t.Fatal("password hash was not updated in database")
	} else {
		t.Log("password hash was updated in database")
	}

	// Confirm token was archived
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM password_resets WHERE token = 'validresettoken'`).Scan(&count)
	if count != 0 {
		t.Fatal("token was not archived after use")
	} else {
		t.Log("token was correctly archived after use")
	}
}

func Test_get_profile_handler(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	db.Exec(`INSERT INTO users (id, username, email, password_hash, created_at, email_verified)
		VALUES ('u1','profileuser','profile@test.com','hash',1700000000,1)`)

	// No user in context — unauthorized
	req := httptest.NewRequest("GET", "/api/profile", nil)
	w := httptest.NewRecorder()
	handlers.GetProfileHandler(db)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without user in context, got %d", w.Code)
	} else {
		t.Log("no user in context correctly returned 401")
	}

	// Valid request
	req = requestWithUserID("GET", "/api/profile", "", "u1")
	w = httptest.NewRecorder()
	handlers.GetProfileHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "profileuser") {
		t.Fatalf("expected username in response, got %s", body)
	}
	if !strings.Contains(body, "profile@test.com") {
		t.Fatalf("expected email in response, got %s", body)
	} else {
		t.Log("profile correctly returned username and email")
	}

	// Confirm blank profile row created on first fetch
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM user_profiles WHERE user_id = 'u1'`).Scan(&count)
	if count != 1 {
		t.Fatal("blank profile row was not created on first fetch")
	} else {
		t.Log("blank profile row correctly created on first fetch")
	}
}

func Test_update_profile_handler(t *testing.T) {
	db := database.InitDB(":memory:")
	hub := messaging.NewHub()
	defer db.Close()

	db.Exec(`INSERT INTO users (id, username, email, password_hash, created_at, email_verified)
		VALUES ('u1','profileuser','profile@test.com','hash',1700000000,1)`)

	// No user in context
	req := httptest.NewRequest("PUT", "/api/profile/update", strings.NewReader(`{"display_name":"Test","bio":"Hello"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.UpdateProfileHandler(db, hub)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without user in context, got %d", w.Code)
	} else {
		t.Log("no user in context correctly returned 401")
	}

	// Valid update
	req = requestWithUserID("PUT", "/api/profile/update", `{"display_name":"Test User","bio":"My bio"}`, "u1")
	w = httptest.NewRecorder()
	handlers.UpdateProfileHandler(db, hub)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid update, got %d", w.Code)
	} else {
		t.Log("valid profile update correctly returned 200")
	}

	// Confirm values saved in db
	var displayName, bio string
	db.QueryRow(`SELECT display_name, bio FROM user_profiles WHERE user_id = 'u1'`).Scan(&displayName, &bio)
	if displayName != "Test User" {
		t.Fatalf("display_name not saved correctly, got: %s", displayName)
	}
	if bio != "My bio" {
		t.Fatalf("bio not saved correctly, got: %s", bio)
	} else {
		t.Log("display_name and bio correctly saved to database")
	}

	// Clear picture flag
	req = requestWithUserID("PUT", "/api/profile/update", `{"display_name":"Test User","bio":"My bio","clear_picture":true}`, "u1")
	w = httptest.NewRecorder()
	handlers.UpdateProfileHandler(db, hub)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for clear picture, got %d", w.Code)
	} else {
		t.Log("clear picture correctly returned 200")
	}

	var pic string
	db.QueryRow(`SELECT profile_picture_url FROM user_profiles WHERE user_id = 'u1'`).Scan(&pic)
	if pic != "" {
		t.Fatalf("profile picture not cleared, got: %s", pic)
	} else {
		t.Log("profile picture correctly cleared")
	}
}

func Test_update_account_handler(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	db.Exec(`INSERT INTO users (id, username, email, password_hash, created_at, email_verified)
		VALUES ('u1','accountuser','account@test.com','oldhash',1700000000,1)`)

	// No user in context
	req := httptest.NewRequest("PUT", "/api/account", strings.NewReader(`{"new_username":"newname"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.UpdateAccountHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without user in context, got %d", w.Code)
	} else {
		t.Log("no user in context correctly returned 401")
	}

	// Password too short
	req = requestWithUserID("PUT", "/api/account", `{"new_password":"short"}`, "u1")
	w = httptest.NewRecorder()
	handlers.UpdateAccountHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for short password, got %d", w.Code)
	} else {
		t.Log("short password correctly returned 400")
	}

	// Username too short
	req = requestWithUserID("PUT", "/api/account", `{"new_username":"ab"}`, "u1")
	w = httptest.NewRecorder()
	handlers.UpdateAccountHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for short username, got %d", w.Code)
	} else {
		t.Log("short username correctly returned 400")
	}

	// Valid username update
	req = requestWithUserID("PUT", "/api/account", `{"new_username":"brandnewname"}`, "u1")
	w = httptest.NewRecorder()
	handlers.UpdateAccountHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid username update, got %d", w.Code)
	} else {
		t.Log("valid username update correctly returned 200")
	}

	var username string
	db.QueryRow(`SELECT username FROM users WHERE id = 'u1'`).Scan(&username)
	if username != "brandnewname" {
		t.Fatalf("username not updated in db, got: %s", username)
	} else {
		t.Log("username correctly updated in database")
	}

	// Valid password update
	req = requestWithUserID("PUT", "/api/account", `{"new_password":"newpassword123"}`, "u1")
	w = httptest.NewRecorder()
	handlers.UpdateAccountHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid password update, got %d", w.Code)
	} else {
		t.Log("valid password update correctly returned 200")
	}

	var hash string
	db.QueryRow(`SELECT password_hash FROM users WHERE id = 'u1'`).Scan(&hash)
	if hash == "oldhash" {
		t.Fatal("password hash was not updated")
	} else {
		t.Log("password hash correctly updated in database")
	}

	// Same email rejected
	req = requestWithUserID("PUT", "/api/account", `{"new_email":"account@test.com"}`, "u1")
	w = httptest.NewRecorder()
	handlers.UpdateAccountHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for same email, got %d", w.Code)
	} else {
		t.Log("same email correctly returned 400")
	}
}

func Test_verify_email_change_handler(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	db.Exec(`INSERT INTO users (id, username, email, password_hash, created_at, email_verified)
		VALUES ('u1','changeuser','old@test.com','hash',1700000000,1)`)

	// Empty token
	req := httptest.NewRequest("GET", "/api/account/verify-email-change?token=", nil)
	w := httptest.NewRecorder()
	handlers.VerifyEmailChangeHandler(db)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty token, got %d", w.Code)
	} else {
		t.Log("empty token correctly returned 400")
	}

	// Token does not exist
	req = httptest.NewRequest("GET", "/api/account/verify-email-change?token=nosuchtoken", nil)
	w = httptest.NewRecorder()
	handlers.VerifyEmailChangeHandler(db)(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for unknown token, got %d", w.Code)
	} else {
		t.Log("unknown token correctly returned 422")
	}

	// Expired token
	db.Exec(`INSERT INTO email_verifications (id, user_id, token, created_at, expires_at, new_email)
		VALUES ('v1','u1','expiredchangetoken',1700000000,1700003600,'expired@test.com')`)
	req = httptest.NewRequest("GET", "/api/account/verify-email-change?token=expiredchangetoken", nil)
	w = httptest.NewRecorder()
	handlers.VerifyEmailChangeHandler(db)(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for expired token, got %d", w.Code)
	} else {
		t.Log("expired token correctly returned 422")
	}

	// Valid token
	db.Exec(`INSERT INTO email_verifications (id, user_id, token, created_at, expires_at, new_email)
		VALUES ('v2','u1','validchangetoken',1700000000,9999999999,'new@test.com')`)
	req = httptest.NewRequest("GET", "/api/account/verify-email-change?token=validchangetoken", nil)
	w = httptest.NewRecorder()
	handlers.VerifyEmailChangeHandler(db)(w, req)
	// Handler redirects on success (302)
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect for valid token, got %d", w.Code)
	} else {
		t.Log("valid token correctly returned 302 redirect")
	}

	// Confirm email was updated in db
	var email string
	db.QueryRow(`SELECT email FROM users WHERE id = 'u1'`).Scan(&email)
	if email != "new@test.com" {
		t.Fatalf("email not updated in db, got: %s", email)
	} else {
		t.Log("email correctly updated in database after change verification")
	}

	// Confirm the verification token was deleted
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM email_verifications WHERE token = 'validchangetoken'`).Scan(&count)
	if count != 0 {
		t.Fatal("email_verifications row was not deleted after use")
	} else {
		t.Log("email_verifications row correctly deleted after use")
	}
}

func Test_logout_handler(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	db.Exec(`INSERT INTO users (id, username, email, password_hash, created_at, email_verified)
		VALUES ('u1','logoutuser','logout@test.com','hash',1700000000,1)`)

	// No cookie — unauthorized
	req := httptest.NewRequest("POST", "/api/logout", nil)
	w := httptest.NewRecorder()
	handlers.LogoutHandler(db)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without cookie, got %d", w.Code)
	} else {
		t.Log("no cookie correctly returned 401")
	}

	// Create a real session then log out
	sessionID, _, err := database.CreateSession(db, "u1", 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	req = httptest.NewRequest("POST", "/api/logout", nil)
	req.AddCookie(&http.Cookie{Name: "sk_session", Value: sessionID})
	w = httptest.NewRecorder()
	handlers.LogoutHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid logout, got %d", w.Code)
	} else {
		t.Log("valid logout correctly returned 200")
	}

	// Confirm session deleted
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE id = ?`, sessionID).Scan(&count)
	if count != 0 {
		t.Fatal("session was not deleted from database after logout")
	} else {
		t.Log("session correctly deleted from database after logout")
	}

	// Confirm cookie was cleared
	cookieCleared := false
	for _, c := range w.Result().Cookies() {
		if c.Name == "sk_session" && c.MaxAge == -1 {
			cookieCleared = true
		}
	}
	if !cookieCleared {
		t.Fatal("session cookie was not cleared in response")
	} else {
		t.Log("session cookie correctly cleared in response")
	}
}

func Test_upload_profile_picture_handler(t *testing.T) {
	db := database.InitDB(":memory:")
	hub := messaging.NewHub()
	defer db.Close()

	db.Exec(`INSERT INTO users (id, username, email, password_hash, created_at, email_verified)
		VALUES ('u1','picuser','pic@test.com','hash',1700000000,1)`)

	// No user in context
	req := httptest.NewRequest("POST", "/api/profile/picture", nil)
	w := httptest.NewRecorder()
	handlers.UploadProfilePictureHandler(db, hub)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without user in context, got %d", w.Code)
	} else {
		t.Log("no user in context correctly returned 401")
	}

	// Valid jpeg upload
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": []string{`form-data; name="picture"; filename="test.jpg"`},
		"Content-Type":        []string{"image/jpeg"},
	})
	part.Write([]byte("fakeimagebytes"))
	writer.Close()

	req2 := httptest.NewRequest("POST", "/api/profile/picture", &buf)
	req2.Header.Set("Content-Type", writer.FormDataContentType())
	req2 = handlers.SetTestUserID(req2, "u1")
	w = httptest.NewRecorder()
	handlers.UploadProfilePictureHandler(db, hub)(w, req2)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid jpeg upload, got %d: %s", w.Code, w.Body.String())
	} else {
		t.Log("valid jpeg upload correctly returned 200")
	}

	// Confirm picture URL saved in db
	var picURL string
	db.QueryRow(`SELECT profile_picture_url FROM user_profiles WHERE user_id = 'u1'`).Scan(&picURL)
	if !strings.HasPrefix(picURL, "data:image/jpeg;base64,") {
		t.Fatalf("profile picture URL not saved correctly, got: %s", picURL)
	} else {
		t.Log("profile picture URL correctly saved as base64 data URL")
	}
}
