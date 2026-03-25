package main

import (
	"net/http"
	"net/http/httptest"
	"secret-keeper-app/backend/database"
	"secret-keeper-app/backend/handlers"
	"strings"
	"testing"
	"time"
)

func Test_register_handler_func(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()
	//invalid password length
	body := `{"username":"testuser","email":"test@gmail.com","password":"tooshrt"}`
	req := httptest.NewRequest("POST", "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.RegisterHandler(db)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatal("short password did not not return correct code")
	} else {
		t.Log("short password returned correct code ")
	}
	//empty username
	body = `{"username":"","email":"test@gmail.com","password":"password"}`
	req = httptest.NewRequest("POST", "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.RegisterHandler(db)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatal("empty username did not return correct code")
	} else {
		t.Log("empty username returned correct code")
	}
	//empty password
	body = `{"username":"testuser","email":"test@gmail.com","password":""}`
	req = httptest.NewRequest("POST", "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.RegisterHandler(db)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatal("empty password did not return correct code")
	} else {
		t.Log("empty password returned correct code")
	}
	//valid user
	body = `{"username":"testuser","email":"test@gmail.com","password":"password"}`
	req = httptest.NewRequest("POST", "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.RegisterHandler(db)(w, req)
	var username, email, password_hash string
	var email_verified int
	err := db.QueryRow(
		`SELECT username, email, password_hash, email_verified FROM users WHERE username = ?`, "testuser",
	).Scan(&username, &email, &password_hash, &email_verified)

	if err != nil {
		t.Fatalf("error getting username, email, password_hash, email_verified from users because of: %v", err)
	} else {
		t.Log("successfully got username, email, password_hash, email_verified from users")
	}

	if username != "testuser" {
		t.Fatalf("inputted username does not match db username, db username is: %v", username)
	} else {
		t.Log("inputted username matches db username")
	}

	if email != "test@gmail.com" {
		t.Fatalf("inputted email does not match db email, db email is: %v", email)
	} else {
		t.Log("inputted email matches db email")
	}

	if password_hash == "password" {
		t.Fatal("password_hash is stored in plain text and is not hashed")
	} else {
		t.Log("password is not stored in plain text")
	}

	if email_verified != 0 {
		t.Fatal("email_verified is not set as 0")
	} else {
		t.Log("email_verified correctly set to 0")
	}
	//email_verification table
	var token, userID string
	var created_at, expires_at int64

	err = db.QueryRow(
		`SELECT id FROM users WHERE username = ?`, "testuser",
	).Scan(&userID)

	if err != nil {
		t.Fatalf("error getting id from user table because of: %v", err)
	} else {
		t.Log("successfully got user_id from user table to check email_verifications table")
	}

	err = db.QueryRow(
		`SELECT token, created_at, expires_at FROM email_verifications WHERE user_id = ?`, userID,
	).Scan(&token, &created_at, &expires_at)

	if err != nil {
		t.Fatalf("error getting token, created_at, expires_at from email_verifications because of: %v", err)
	} else {
		t.Log("successfully got token, created_at, expires_at from email_verifications")
	}

	if token == "" {
		t.Fatal("token is empty")
	} else {
		t.Log("token is not empty")
	}

	if created_at > time.Now().Unix() {
		t.Fatal("created_at should not be in the future")
	} else {
		t.Log("created_at is in the past")
	}

	if expires_at < time.Now().Unix() {
		t.Fatal("expires_at should not be in the past")
	} else {
		t.Log("expires_at is in the future")
	}

	expected := `{"user_id":"` + userID + `","message":"Account created. Please check your email to verify your address before logging in."}`
	if w.Body.String() != expected {
		t.Fatalf("unexpected response body: %v", w.Body.String())
	} else {
		t.Log("expected response body")
	}

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", w.Code)
	} else {
		t.Log("got 201 code for correct response")
	}
	//duplicate username
	body = `{"username":"testuser","email":"test@gmail.com","password":"password"}`
	req = httptest.NewRequest("POST", "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.RegisterHandler(db)(w, req)

	if w.Code != http.StatusConflict {
		t.Fatal("duplicate username was accepted")
	} else {
		t.Log("duplicate username was rejected")
	}
}

func Test_verify_email_handler_func(t *testing.T) {
	var token string
	db := database.InitDB(":memory:")
	defer db.Close()

	//invalid token
	token = ""
	req := httptest.NewRequest("GET", "/api/verify-email?token="+token, nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.VerifyEmailHandler(db)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatal("empty token did not return correct code")
	} else {
		t.Log("empty token returned valid code")
	}

	//expired link (inserts a real user first so the foreign key is satisfied)
	if _, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, email_verified)
		VALUES ("9e99af6b-48e4-4eeb-951f-0cb27e03e32c", "testuser", "testuser@gmail.com", "hashedpassword", 1740067200, 0)
	`); err != nil {
		t.Fatalf("error inserting test user: %v", err)
	}

	token = "03feb4393fa18cbf70c49732c1419d3fc8e6842c072d6937edc96ba6aef27338"

	if _, err := db.Exec(`
		INSERT INTO email_verifications (id, user_id, token, created_at, expires_at)
		VALUES ("ev-0001", "9e99af6b-48e4-4eeb-951f-0cb27e03e32c", "03feb4393fa18cbf70c49732c1419d3fc8e6842c072d6937edc96ba6aef27338", 1740067200, 1740467200)
	`); err != nil {
		t.Fatal("error inserting test value into email_verifications table")
	} else {
		t.Log("successfully inserted test values into email_verifications table")
	}

	req = httptest.NewRequest("GET", "/api/verify-email?token="+token, nil)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.VerifyEmailHandler(db)(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatal("invalid or expired verification link does not return correct code")
	} else {
		t.Log("invalid or expired link returned correct code")
	}
}

func Test_login_handler_func(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	// Register verified user for login
	body := `{"username":"loginuser","email":"login@test.com","password":"loginpass"}`
	req := httptest.NewRequest("POST", "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.RegisterHandler(db)(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: register failed with %d", w.Code)
	}

	// Marking user as email_verified so login is allowed
	if _, err := db.Exec(`UPDATE users SET email_verified = 1 WHERE username = 'loginuser'`); err != nil {
		t.Fatalf("setup: could not verify email: %v", err)
	}

	// Wrong password
	body = `{"username":"loginuser","password":"wrongpass"}`
	req = httptest.NewRequest("POST", "/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.LoginHandler(db, 24*time.Hour)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", w.Code)
	} else {
		t.Log("wrong password correctly returned 401")
	}

	// Non-existent username
	body = `{"username":"nobody","password":"loginpass"}`
	req = httptest.NewRequest("POST", "/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.LoginHandler(db, 24*time.Hour)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unknown user, got %d", w.Code)
	} else {
		t.Log("non-existent username correctly returned 401")
	}

	// Unverified email for a second user
	body = `{"username":"unverified","email":"unverified@test.com","password":"loginpass"}`
	req = httptest.NewRequest("POST", "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.RegisterHandler(db)(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: register unverified user failed with %d", w.Code)
	}

	body = `{"username":"unverified","password":"loginpass"}`
	req = httptest.NewRequest("POST", "/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.LoginHandler(db, 24*time.Hour)(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for unverified email, got %d", w.Code)
	} else {
		t.Log("unverified email correctly returned 403")
	}

	// Valid login
	body = `{"username":"loginuser","password":"loginpass"}`
	req = httptest.NewRequest("POST", "/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.LoginHandler(db, 24*time.Hour)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid login, got %d", w.Code)
	} else {
		t.Log("valid login correctly returned 200")
	}

	// Confirm session cookie
	cookieFound := false
	for _, c := range w.Result().Cookies() {
		if c.Name == "sk_session" && c.Value != "" {
			cookieFound = true
		}
	}
	if !cookieFound {
		t.Fatal("sk_session cookie was not set after successful login")
	} else {
		t.Log("sk_session cookie correctly set after successful login")
	}

	// Confirm session in db
	var sessionID string
	for _, c := range w.Result().Cookies() {
		if c.Name == "sk_session" {
			sessionID = c.Value
		}
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE id = ?`, sessionID).Scan(&count)
	if count != 1 {
		t.Fatal("session was not created in the database after login")
	} else {
		t.Log("session correctly created in database after login")
	}
}
