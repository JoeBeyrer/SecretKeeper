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
