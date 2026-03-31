package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"secret-keeper-app/backend/database"
	"secret-keeper-app/backend/handlers"
	"strings"
	"testing"
)

func Test_save_keys_handler(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	// Register alice
	body := `{"username":"alice","email":"alice@test.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.RegisterHandler(db)(w, req)
	db.Exec(`UPDATE users SET email_verified = 1 WHERE username = 'alice'`)

	var aliceID string
	db.QueryRow(`SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID)

	// Missing keys should return 400
	req = requestWithUserID("POST", "/api/keys/save", `{"public_key":"","encrypted_private_key":""}`, aliceID)
	w = httptest.NewRecorder()
	handlers.SaveKeysHandler(db)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty keys, got %d", w.Code)
	}
	t.Log("empty keys correctly returned 400")

	// Valid save should return 204
	req = requestWithUserID("POST", "/api/keys/save",
		`{"public_key":"my-public-key","encrypted_private_key":"my-enc-private-key"}`, aliceID)
	w = httptest.NewRecorder()
	handlers.SaveKeysHandler(db)(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for valid save, got %d: %s", w.Code, w.Body.String())
	}
	t.Log("valid key save returned 204")

	// Unauthenticated request should return 401
	req = httptest.NewRequest("POST", "/api/keys/save",
		strings.NewReader(`{"public_key":"k","encrypted_private_key":"k"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.SaveKeysHandler(db)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated request, got %d", w.Code)
	}
	t.Log("unauthenticated save-keys correctly returned 401")
}

func Test_get_keys_handler(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	body := `{"username":"alice","email":"alice@test.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.RegisterHandler(db)(w, req)
	db.Exec(`UPDATE users SET email_verified = 1 WHERE username = 'alice'`)

	var aliceID string
	db.QueryRow(`SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID)

	// No keys saved yet should return 404
	req = requestWithUserID("GET", "/api/keys/get", "", aliceID)
	w = httptest.NewRecorder()
	handlers.GetKeysHandler(db)(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 before any keys saved, got %d", w.Code)
	}
	t.Log("get-keys before save correctly returned 404")

	// Save keys first
	req = requestWithUserID("POST", "/api/keys/save",
		`{"public_key":"pub-abc","encrypted_private_key":"priv-xyz"}`, aliceID)
	w = httptest.NewRecorder()
	handlers.SaveKeysHandler(db)(w, req)

	// Now get should return them
	req = requestWithUserID("GET", "/api/keys/get", "", aliceID)
	w = httptest.NewRecorder()
	handlers.GetKeysHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after saving keys, got %d", w.Code)
	}
	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["public_key"] != "pub-abc" {
		t.Fatalf("expected public_key 'pub-abc', got '%s'", result["public_key"])
	}
	if result["encrypted_private_key"] != "priv-xyz" {
		t.Fatalf("expected encrypted_private_key 'priv-xyz', got '%s'", result["encrypted_private_key"])
	}
	t.Log("get-keys returned correct key pair")

	// Unauthenticated should return 401
	req = httptest.NewRequest("GET", "/api/keys/get", nil)
	w = httptest.NewRecorder()
	handlers.GetKeysHandler(db)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	t.Log("unauthenticated get-keys correctly returned 401")
}

func Test_get_public_key_handler(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	for _, u := range []struct{ name, email string }{
		{"alice", "alice@test.com"},
		{"bob", "bob@test.com"},
	} {
		body := `{"username":"` + u.name + `","email":"` + u.email + `","password":"password123"}`
		req := httptest.NewRequest("POST", "/api/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handlers.RegisterHandler(db)(w, req)
		db.Exec(`UPDATE users SET email_verified = 1 WHERE username = ?`, u.name)
	}

	var aliceID, bobID string
	db.QueryRow(`SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID)
	db.QueryRow(`SELECT id FROM users WHERE username = 'bob'`).Scan(&bobID)

	// Bob's public key not saved yet
	req := requestWithUserID("GET", "/api/users/bob/public-key", "", aliceID)
	req.SetPathValue("username", "bob")
	w := httptest.NewRecorder()
	handlers.GetPublicKeyHandler(db)(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 before bob saves keys, got %d", w.Code)
	}
	t.Log("get-public-key before save correctly returned 404")

	// Bob saves keys
	req = requestWithUserID("POST", "/api/keys/save",
		`{"public_key":"bobs-public-key","encrypted_private_key":"bobs-enc-private"}`, bobID)
	w = httptest.NewRecorder()
	handlers.SaveKeysHandler(db)(w, req)

	// Alice can now fetch Bob's public key
	req = requestWithUserID("GET", "/api/users/bob/public-key", "", aliceID)
	req.SetPathValue("username", "bob")
	w = httptest.NewRecorder()
	handlers.GetPublicKeyHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after bob saves keys, got %d: %s", w.Code, w.Body.String())
	}
	var result map[string]string
	json.NewDecoder(w.Body).Decode(&result)
	if result["public_key"] != "bobs-public-key" {
		t.Fatalf("expected 'bobs-public-key', got '%s'", result["public_key"])
	}
	if result["user_id"] != bobID {
		t.Fatalf("expected user_id '%s', got '%s'", bobID, result["user_id"])
	}
	t.Log("get-public-key returned correct key and user_id")

	// Unknown username should return 404
	req = requestWithUserID("GET", "/api/users/nobody/public-key", "", aliceID)
	req.SetPathValue("username", "nobody")
	w = httptest.NewRecorder()
	handlers.GetPublicKeyHandler(db)(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown username, got %d", w.Code)
	}
	t.Log("unknown username correctly returned 404")

	// Unauthenticated should return 401
	req = httptest.NewRequest("GET", "/api/users/bob/public-key", nil)
	req.SetPathValue("username", "bob")
	w = httptest.NewRecorder()
	handlers.GetPublicKeyHandler(db)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	t.Log("unauthenticated get-public-key correctly returned 401")
}
