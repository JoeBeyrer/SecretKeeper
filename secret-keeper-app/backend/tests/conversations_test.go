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

func Test_create_conversation_handler(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	// Register alice and bob
	for _, u := range []struct{ name, email string }{
		{"alice", "alice@test.com"},
		{"bob", "bob@test.com"},
	} {
		body := `{"username":"` + u.name + `","email":"` + u.email + `","password":"password123"}`
		req := httptest.NewRequest("POST", "/api/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handlers.RegisterHandler(db)(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("setup: register %s got %d", u.name, w.Code)
		}
		db.Exec(`UPDATE users SET email_verified = 1 WHERE username = ?`, u.name)
	}

	var aliceID, bobID string
	db.QueryRow(`SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID)
	db.QueryRow(`SELECT id FROM users WHERE username = 'bob'`).Scan(&bobID)

	// Missing room key should return 400
	body := `{"member_ids":["bob"]}`
	req := requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w := httptest.NewRecorder()
	handlers.CreateConversationHandler(db)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing room key, got %d", w.Code)
	} else {
		t.Log("missing room key correctly returned 400")
	}

	// Unknown member username should return 400
	body = `{"member_ids":["nobody"],"room_key":"testkey"}`
	req = requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w = httptest.NewRecorder()
	handlers.CreateConversationHandler(db)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown member, got %d", w.Code)
	} else {
		t.Log("unknown member username correctly returned 400")
	}

	// Valid request should create a conversation
	body = `{"member_ids":["bob"],"room_key":"supersecretkey"}`
	req = requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w = httptest.NewRecorder()
	handlers.CreateConversationHandler(db)(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for valid creation, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	convID, ok := resp["conversation_id"].(string)
	if !ok || convID == "" {
		t.Fatal("expected non-empty conversation_id in response")
	}
	if created, _ := resp["created"].(bool); !created {
		t.Fatal("expected created=true for new conversation")
	}
	t.Logf("conversation created with id: %s", convID)

	// Duplicate request should return existing conversation without created=true
	req = requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w = httptest.NewRecorder()
	handlers.CreateConversationHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for existing conversation, got %d", w.Code)
	}
	var resp2 map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp2)
	if resp2["conversation_id"] != convID {
		t.Fatalf("expected same conversation_id %s, got %v", convID, resp2["conversation_id"])
	}
	t.Log("duplicate conversation request correctly returned existing conversation")

	// Unauthorized request should return 401
	req = httptest.NewRequest("POST", "/api/conversations/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.CreateConversationHandler(db)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated request, got %d", w.Code)
	}
	t.Log("unauthenticated request correctly returned 401")
}

func Test_get_conversations_handler(t *testing.T) {
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

	var aliceID string
	db.QueryRow(`SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID)

	// Empty list before any conversations exist
	req := requestWithUserID("GET", "/api/conversations/get", "", aliceID)
	w := httptest.NewRecorder()
	handlers.GetConversationsHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var convs []interface{}
	json.NewDecoder(w.Body).Decode(&convs)
	if len(convs) != 0 {
		t.Fatalf("expected empty list, got %d conversations", len(convs))
	}
	t.Log("empty conversations list returned correctly")

	// Create a conversation then verify it appears in the list
	body := `{"member_ids":["bob"],"room_key":"testkey"}`
	req = requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w = httptest.NewRecorder()
	handlers.CreateConversationHandler(db)(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: failed to create conversation: %d", w.Code)
	}

	req = requestWithUserID("GET", "/api/conversations/get", "", aliceID)
	w = httptest.NewRecorder()
	handlers.GetConversationsHandler(db)(w, req)
	json.NewDecoder(w.Body).Decode(&convs)
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(convs))
	}
	t.Log("created conversation appears in list correctly")

	// Unauthorized
	req = httptest.NewRequest("GET", "/api/conversations/get", nil)
	w = httptest.NewRecorder()
	handlers.GetConversationsHandler(db)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	t.Log("unauthenticated get-conversations correctly returned 401")
}

func Test_get_conversation_messages_handler(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	for _, u := range []struct{ name, email string }{
		{"alice", "alice@test.com"},
		{"bob", "bob@test.com"},
		{"carol", "carol@test.com"},
	} {
		body := `{"username":"` + u.name + `","email":"` + u.email + `","password":"password123"}`
		req := httptest.NewRequest("POST", "/api/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handlers.RegisterHandler(db)(w, req)
		db.Exec(`UPDATE users SET email_verified = 1 WHERE username = ?`, u.name)
	}

	var aliceID, carolID string
	db.QueryRow(`SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID)
	db.QueryRow(`SELECT id FROM users WHERE username = 'carol'`).Scan(&carolID)

	// Create conversation between alice and bob
	body := `{"member_ids":["bob"],"room_key":"testkey"}`
	req := requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w := httptest.NewRecorder()
	handlers.CreateConversationHandler(db)(w, req)
	var convResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&convResp)
	convID := convResp["conversation_id"].(string)

	// Alice can fetch messages
	req = requestWithUserID("GET", "/api/conversations/"+convID+"/messages", "", aliceID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.GetConversationMessagesHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for message fetch, got %d", w.Code)
	}
	var msgs []interface{}
	json.NewDecoder(w.Body).Decode(&msgs)
	if len(msgs) != 0 {
		t.Fatalf("expected empty message list, got %d", len(msgs))
	}
	t.Log("empty message list returned correctly")

	// Carol should get 403
	req = requestWithUserID("GET", "/api/conversations/"+convID+"/messages", "", carolID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.GetConversationMessagesHandler(db)(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-member, got %d", w.Code)
	}
	t.Log("non-member correctly returned 403")

	// Save a message directly and confirm it comes back through the handler
	if err := database.SaveMessage(db, "msg-001", convID, aliceID, "hello encrypted", 1700000000); err != nil {
		t.Fatalf("SaveMessage: %v", err)
	}

	req = requestWithUserID("GET", "/api/conversations/"+convID+"/messages", "", aliceID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.GetConversationMessagesHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after saving message, got %d", w.Code)
	}
	var msgsAfter []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&msgsAfter)
	if len(msgsAfter) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgsAfter))
	}
	if msgsAfter[0]["Ciphertext"] != "hello encrypted" {
		t.Fatalf("wrong ciphertext: %v", msgsAfter[0]["Ciphertext"])
	} else {
		t.Log("saved message correctly appears with correct ciphertext")
	}
}

func Test_verify_conversation_room_key_handler(t *testing.T) {
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

	var aliceID string
	db.QueryRow(`SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID)

	// Create conversation
	body := `{"member_ids":["bob"],"room_key":"correct-room-key"}`
	req := requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w := httptest.NewRecorder()
	handlers.CreateConversationHandler(db)(w, req)
	var convResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&convResp)
	convID := convResp["conversation_id"].(string)

	// Correct key should return 204
	req = requestWithUserID("POST", "/api/conversations/"+convID+"/verify-room-key",
		`{"room_key":"correct-room-key"}`, aliceID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.VerifyConversationRoomKeyHandler(db)(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for correct key, got %d: %s", w.Code, w.Body.String())
	}
	t.Log("correct room key verification returned 204")

	// Wrong key should return 401
	req = requestWithUserID("POST", "/api/conversations/"+convID+"/verify-room-key",
		`{"room_key":"wrong-key"}`, aliceID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.VerifyConversationRoomKeyHandler(db)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong key, got %d", w.Code)
	}
	t.Log("wrong room key correctly returned 401")

	// Missing room_key field should return 400
	req = requestWithUserID("POST", "/api/conversations/"+convID+"/verify-room-key",
		`{"room_key":""}`, aliceID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.VerifyConversationRoomKeyHandler(db)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty room key, got %d", w.Code)
	}
	t.Log("empty room key correctly returned 400")
}

func Test_claim_conversation_room_key_handler(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	for _, u := range []struct{ name, email string }{
		{"alice", "alice@test.com"},
		{"bob", "bob@test.com"},
		{"carol", "carol@test.com"},
	} {
		body := `{"username":"` + u.name + `","email":"` + u.email + `","password":"password123"}`
		req := httptest.NewRequest("POST", "/api/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handlers.RegisterHandler(db)(w, req)
		db.Exec(`UPDATE users SET email_verified = 1 WHERE username = ?`, u.name)
	}

	var aliceID, bobID, carolID string
	db.QueryRow(`SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID)
	db.QueryRow(`SELECT id FROM users WHERE username = 'bob'`).Scan(&bobID)
	db.QueryRow(`SELECT id FROM users WHERE username = 'carol'`).Scan(&carolID)

	// Alice creates conversation with bob
	body := `{"member_ids":["bob"],"room_key":"the-real-room-key"}`
	req := requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w := httptest.NewRecorder()
	handlers.CreateConversationHandler(db)(w, req)
	var convResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&convResp)
	convID := convResp["conversation_id"].(string)

	// Bob can claim the pending room key
	req = requestWithUserID("POST", "/api/conversations/"+convID+"/claim-room-key", "", bobID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.ClaimConversationRoomKeyHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for bob claiming key, got %d: %s", w.Code, w.Body.String())
	}
	var keyResp map[string]string
	json.NewDecoder(w.Body).Decode(&keyResp)
	if keyResp["room_key"] != "the-real-room-key" {
		t.Fatalf("expected 'the-real-room-key', got '%s'", keyResp["room_key"])
	}
	t.Log("bob successfully claimed the pending room key")

	// Claiming again should return 404
	req = requestWithUserID("POST", "/api/conversations/"+convID+"/claim-room-key", "", bobID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.ClaimConversationRoomKeyHandler(db)(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for already-claimed key, got %d", w.Code)
	}
	t.Log("second claim correctly returned 404")

	// Carol should get 403
	req = requestWithUserID("POST", "/api/conversations/"+convID+"/claim-room-key", "", carolID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.ClaimConversationRoomKeyHandler(db)(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-member carol, got %d", w.Code)
	}
	t.Log("non-member correctly returned 403")
}


func Test_edit_message_handler(t *testing.T) {
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
		if w.Code != http.StatusCreated {
			t.Fatalf("setup: register %s got %d", u.name, w.Code)
		}
		db.Exec(`UPDATE users SET email_verified = 1 WHERE username = ?`, u.name)
	}

	var aliceID, bobID string
	db.QueryRow(`SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID)
	db.QueryRow(`SELECT id FROM users WHERE username = 'bob'`).Scan(&bobID)

	body := `{"member_ids":["bob"],"room_key":"supersecretkey"}`
	req := requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w := httptest.NewRecorder()
	handlers.CreateConversationHandler(db)(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: failed to create conversation: %d", w.Code)
	}

	var convResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&convResp); err != nil {
		t.Fatalf("setup: decode conversation response: %v", err)
	}
	convID := convResp["conversation_id"].(string)

	if err := database.SaveMessage(db, "msg-edit-1", convID, aliceID, "original-ciphertext", 1700000000); err != nil {
		t.Fatalf("setup: SaveMessage: %v", err)
	}

	handler := handlers.MessageHandler(db, messaging.NewHub())

	req = requestWithUserID("PATCH", "/api/messages/msg-edit-1", `{}`, aliceID)
	req.SetPathValue("id", "msg-edit-1")
	w = httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing ciphertext, got %d", w.Code)
	}

	req = requestWithUserID("PATCH", "/api/messages/msg-edit-1", `{"ciphertext":"bob-edit"}`, bobID)
	req.SetPathValue("id", "msg-edit-1")
	w = httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for editing another user's message, got %d", w.Code)
	}

	req = requestWithUserID("PATCH", "/api/messages/msg-edit-1", `{"ciphertext":"updated-ciphertext"}`, aliceID)
	req.SetPathValue("id", "msg-edit-1")
	w = httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for valid edit, got %d: %s", w.Code, w.Body.String())
	}

	var ciphertext string
	if err := db.QueryRow(`SELECT ciphertext FROM messages WHERE id = ?`, "msg-edit-1").Scan(&ciphertext); err != nil {
		t.Fatalf("query updated message: %v", err)
	}
	if ciphertext != "updated-ciphertext" {
		t.Fatalf("expected ciphertext to be updated, got %q", ciphertext)
	}
}

func Test_get_conversation_messages_handler_attachment_ciphertext(t *testing.T) {
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
		if w.Code != http.StatusCreated {
			t.Fatalf("setup: register %s got %d", u.name, w.Code)
		}
		db.Exec(`UPDATE users SET email_verified = 1 WHERE username = ?`, u.name)
	}

	var aliceID string
	db.QueryRow(`SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID)

	body := `{"member_ids":["bob"],"room_key":"supersecretkey"}`
	req := requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w := httptest.NewRecorder()
	handlers.CreateConversationHandler(db)(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: failed to create conversation: %d", w.Code)
	}

	var convResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&convResp); err != nil {
		t.Fatalf("setup: decode conversation response: %v", err)
	}
	convID := convResp["conversation_id"].(string)

	richCiphertext := "encrypted-rich-message-payload"
	if err := database.SaveMessage(db, "msg-file-1", convID, aliceID, richCiphertext, 1700000001); err != nil {
		t.Fatalf("setup: SaveMessage: %v", err)
	}

	req = requestWithUserID("GET", "/api/conversations/"+convID+"/messages", "", aliceID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.GetConversationMessagesHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for message fetch, got %d", w.Code)
	}

	var msgs []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&msgs); err != nil {
		t.Fatalf("decode messages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0]["Ciphertext"] != richCiphertext {
		t.Fatalf("expected ciphertext %q, got %v", richCiphertext, msgs[0]["Ciphertext"])
	}
}