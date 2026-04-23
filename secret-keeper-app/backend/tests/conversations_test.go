package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"secret-keeper-app/backend/database"
	"secret-keeper-app/backend/handlers"
	"secret-keeper-app/backend/messaging"
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
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing room key, got %d", w.Code)
	} else {
		t.Log("missing room key correctly returned 400")
	}

	// Unknown member username should return 400
	body = `{"member_ids":["nobody"],"room_key":"testkey"}`
	req = requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w = httptest.NewRecorder()
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown member, got %d", w.Code)
	} else {
		t.Log("unknown member username correctly returned 400")
	}

	// Valid request should create a conversation
	body = `{"member_ids":["bob"],"room_key":"supersecretkey"}`
	req = requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w = httptest.NewRecorder()
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
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
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
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
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
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
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
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

	// Add a third user, create a named group conversation, and verify the shared name is returned
	body = `{"username":"carol","email":"carol@test.com","password":"password123"}`
	req = httptest.NewRequest("POST", "/api/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handlers.RegisterHandler(db)(w, req)
	db.Exec(`UPDATE users SET email_verified = 1 WHERE username = ?`, "carol")

	body = `{"member_ids":["bob","carol"],"room_key":"groupkey","group_name":"Weekend Plans"}`
	req = requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w = httptest.NewRecorder()
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for named group creation, got %d: %s", w.Code, w.Body.String())
	}

	req = requestWithUserID("GET", "/api/conversations/get", "", aliceID)
	w = httptest.NewRecorder()
	handlers.GetConversationsHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for named group get, got %d", w.Code)
	}
	var convSummaries []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&convSummaries)
	var foundNamedGroup bool
	for _, conv := range convSummaries {
		if conv["name"] == "Weekend Plans" {
			foundNamedGroup = true
			if memberCount, ok := conv["member_count"].(float64); !ok || int(memberCount) != 3 {
				t.Fatalf("expected named group member_count=3, got %#v", conv["member_count"])
			}
			break
		}
	}
	if !foundNamedGroup {
		t.Fatalf("expected named group conversation to appear in list, got %#v", convSummaries)
	}

	// Unauthorized
	req = httptest.NewRequest("GET", "/api/conversations/get", nil)
	w = httptest.NewRecorder()
	handlers.GetConversationsHandler(db)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	t.Log("unauthenticated get-conversations correctly returned 401")
}

func Test_get_conversation_members_handler(t *testing.T) {
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

	var aliceID, bobID string
	db.QueryRow(`SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID)
	db.QueryRow(`SELECT id FROM users WHERE username = 'bob'`).Scan(&bobID)

	if err := database.SendFriendRequest(db, aliceID, bobID); err != nil {
		t.Fatalf("setup: send friend request: %v", err)
	}
	if err := database.AcceptFriendRequest(db, bobID, aliceID); err != nil {
		t.Fatalf("setup: accept friend request: %v", err)
	}

	body := `{"member_ids":["bob","carol"],"room_key":"groupkey","group_name":"Weekend Plans"}`
	req := requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w := httptest.NewRecorder()
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: failed to create conversation: %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("setup: decode conversation response: %v", err)
	}
	convID := resp["conversation_id"].(string)

	req = requestWithUserID("GET", "/api/conversations/"+convID+"/members", "", aliceID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.GetConversationMembersHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from get members, got %d: %s", w.Code, w.Body.String())
	}

	var members []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&members); err != nil {
		t.Fatalf("decode members response: %v", err)
	}
	if len(members) != 3 {
		t.Fatalf("expected 3 members, got %d", len(members))
	}

	statuses := map[string]string{}
	for _, member := range members {
		statuses[member["username"].(string)] = member["friendship_status"].(string)
	}

	if statuses["alice"] != "self" {
		t.Fatalf("expected alice friendship_status=self, got %q", statuses["alice"])
	}
	if statuses["bob"] != "friend" {
		t.Fatalf("expected bob friendship_status=friend, got %q", statuses["bob"])
	}
	if statuses["carol"] != "none" {
		t.Fatalf("expected carol friendship_status=none, got %q", statuses["carol"])
	}

	req = requestWithUserID("GET", "/api/conversations/"+convID+"/members", "", "not-in-group")
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.GetConversationMembersHandler(db)(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-member, got %d", w.Code)
	}
}

func Test_update_group_name_handler(t *testing.T) {
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

	var aliceID string
	db.QueryRow(`SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID)

	body := `{"member_ids":["bob","carol"],"room_key":"groupkey","group_name":"Weekend Plans"}`
	req := requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w := httptest.NewRecorder()
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: failed to create conversation: %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("setup: decode conversation response: %v", err)
	}
	convID := resp["conversation_id"].(string)

	req = requestWithUserID("PATCH", "/api/conversations/"+convID+"/group-name", `{"group_name":"Road Trip Crew"}`, aliceID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.UpdateGroupNameHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for group name update, got %d: %s", w.Code, w.Body.String())
	}

	var groupName string
	if err := db.QueryRow(`SELECT group_name FROM conversations WHERE id = ?`, convID).Scan(&groupName); err != nil {
		t.Fatalf("load updated group name: %v", err)
	}
	if groupName != "Road Trip Crew" {
		t.Fatalf("expected updated group name, got %q", groupName)
	}

	var notice string
	var senderID sql.NullString
	if err := db.QueryRow(`
		SELECT ciphertext, sender_id
		FROM messages
		WHERE conversation_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, convID).Scan(&notice, &senderID); err != nil {
		t.Fatalf("load rename notice: %v", err)
	}
	if notice != `Group name changed to "Road Trip Crew"` {
		t.Fatalf("expected rename notice, got %q", notice)
	}
	if senderID.Valid {
		t.Fatalf("expected rename notice to be a system message, got sender %q", senderID.String)
	}

	req = requestWithUserID("PATCH", "/api/conversations/"+convID+"/group-name", `{"group_name":"   "}`, aliceID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.UpdateGroupNameHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for blank group name, got %d", w.Code)
	}
}

func Test_remove_conversation_members_handler(t *testing.T) {
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
		if w.Code != http.StatusCreated {
			t.Fatalf("setup: register %s got %d", u.name, w.Code)
		}
		db.Exec(`UPDATE users SET email_verified = 1 WHERE username = ?`, u.name)
	}

	var aliceID, bobID, carolID string
	db.QueryRow(`SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID)
	db.QueryRow(`SELECT id FROM users WHERE username = 'bob'`).Scan(&bobID)
	db.QueryRow(`SELECT id FROM users WHERE username = 'carol'`).Scan(&carolID)

	body := `{"member_ids":["bob","carol"],"room_key":"groupsecret","group_name":"Weekend Plans"}`
	req := requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w := httptest.NewRecorder()
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	convID := resp["conversation_id"].(string)

	if _, err := db.Exec(`INSERT INTO conversation_keys (conversation_id, user_id, encrypted_key) VALUES (?, ?, ?)`, convID, bobID, "bob-key"); err != nil {
		t.Fatalf("insert bob conversation key: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO conversation_keys (conversation_id, user_id, encrypted_key) VALUES (?, ?, ?)`, convID, carolID, "carol-key"); err != nil {
		t.Fatalf("insert carol conversation key: %v", err)
	}

	req = requestWithUserID("PATCH", "/api/conversations/"+convID+"/members/remove", `{"member_ids":["`+carolID+`"]}`, aliceID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.RemoveConversationMembersHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for member removal, got %d: %s", w.Code, w.Body.String())
	}

	remainingMembers, err := database.GetConversationMembers(db, convID)
	if err != nil {
		t.Fatalf("GetConversationMembers: %v", err)
	}
	if len(remainingMembers) != 2 {
		t.Fatalf("expected two remaining members after removal, got %d", len(remainingMembers))
	}
	for _, memberID := range remainingMembers {
		if memberID == carolID {
			t.Fatalf("expected carol to be removed from the conversation")
		}
	}

	var groupName string
	if err := db.QueryRow(`SELECT COALESCE(group_name, '') FROM conversations WHERE id = ?`, convID).Scan(&groupName); err != nil {
		t.Fatalf("load group name: %v", err)
	}
	if groupName != "" {
		t.Fatalf("expected group name to clear when only two members remain, got %q", groupName)
	}

	var removalNotice string
	var senderID sql.NullString
	if err := db.QueryRow(`
		SELECT ciphertext, sender_id
		FROM messages
		WHERE conversation_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, convID).Scan(&removalNotice, &senderID); err != nil {
		t.Fatalf("load removal notice: %v", err)
	}
	if removalNotice != "alice removed carol from the conversation" {
		t.Fatalf("expected removal notice, got %q", removalNotice)
	}
	if senderID.Valid {
		t.Fatalf("expected removal notice to be a system message, got sender %q", senderID.String)
	}

	var carolMembershipCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ? AND user_id = ?`, convID, carolID).Scan(&carolMembershipCount); err != nil {
		t.Fatalf("count carol membership: %v", err)
	}
	if carolMembershipCount != 0 {
		t.Fatalf("expected carol membership to be removed, got %d rows", carolMembershipCount)
	}

	var carolKeyCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_keys WHERE conversation_id = ? AND user_id = ?`, convID, carolID).Scan(&carolKeyCount); err != nil {
		t.Fatalf("count carol conversation key: %v", err)
	}
	if carolKeyCount != 0 {
		t.Fatalf("expected carol conversation key to be removed, got %d rows", carolKeyCount)
	}

	var bobKeyCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_keys WHERE conversation_id = ? AND user_id = ?`, convID, bobID).Scan(&bobKeyCount); err != nil {
		t.Fatalf("count bob conversation key: %v", err)
	}
	if bobKeyCount != 1 {
		t.Fatalf("expected bob conversation key to remain, got %d rows", bobKeyCount)
	}
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
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
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
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
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

func Test_claim_group_conversation_room_key_handler(t *testing.T) {
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

	var aliceID, bobID string
	db.QueryRow(`SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID)
	db.QueryRow(`SELECT id FROM users WHERE username = 'bob'`).Scan(&bobID)

	body := `{"member_ids":["bob","carol"],"room_key":"shared-group-room-key"}`
	req := requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w := httptest.NewRecorder()
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating group conversation, got %d: %s", w.Code, w.Body.String())
	}

	var convResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&convResp)
	convID := convResp["conversation_id"].(string)

	for _, claimantID := range []string{bobID, carolID} {
		req = requestWithUserID("POST", "/api/conversations/"+convID+"/claim-room-key", "", claimantID)
		req.SetPathValue("id", convID)
		w = httptest.NewRecorder()
		handlers.ClaimConversationRoomKeyHandler(db)(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 when claiming group room key, got %d: %s", w.Code, w.Body.String())
		}

		var keyResp map[string]string
		json.NewDecoder(w.Body).Decode(&keyResp)
		if keyResp["room_key"] != "shared-group-room-key" {
			t.Fatalf("expected shared-group-room-key, got %q", keyResp["room_key"])
		}
	}

	req = requestWithUserID("POST", "/api/conversations/"+convID+"/claim-room-key", "", bobID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.ClaimConversationRoomKeyHandler(db)(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after room key was already claimed, got %d", w.Code)
	}
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

	var aliceID, bobID string
	db.QueryRow(`SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID)
	db.QueryRow(`SELECT id FROM users WHERE username = 'bob'`).Scan(&bobID)

	// Alice creates conversation with bob
	body := `{"member_ids":["bob"],"room_key":"the-real-room-key"}`
	req := requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w := httptest.NewRecorder()
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
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
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
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
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
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

func Test_leave_two_person_conversation_handler(t *testing.T) {
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
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	convID := resp["conversation_id"].(string)

	if err := database.SaveMessage(db, "msg-1", convID, aliceID, "ciphertext", 1700000000); err != nil {
		t.Fatalf("SaveMessage: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO conversation_keys (conversation_id, user_id, encrypted_key) VALUES (?, ?, ?)`, convID, aliceID, "alice-key"); err != nil {
		t.Fatalf("insert alice conversation key: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO conversation_keys (conversation_id, user_id, encrypted_key) VALUES (?, ?, ?)`, convID, bobID, "bob-key"); err != nil {
		t.Fatalf("insert bob conversation key: %v", err)
	}

	req = requestWithUserID("POST", "/api/conversations/"+convID+"/leave", "", aliceID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.LeaveConversationHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for leave, got %d: %s", w.Code, w.Body.String())
	}

	checks := []struct {
		query string
		label string
	}{
		{`SELECT COUNT(*) FROM conversations WHERE id = ?`, "conversation"},
		{`SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ?`, "conversation members"},
		{`SELECT COUNT(*) FROM messages WHERE conversation_id = ?`, "messages"},
		{`SELECT COUNT(*) FROM conversation_pending_room_keys WHERE conversation_id = ?`, "pending room keys"},
		{`SELECT COUNT(*) FROM conversation_keys WHERE conversation_id = ?`, "conversation keys"},
	}
	for _, check := range checks {
		var count int
		if err := db.QueryRow(check.query, convID).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", check.label, err)
		}
		if count != 0 {
			t.Fatalf("expected %s cleanup for %s, got %d remaining row(s)", check.label, convID, count)
		}
	}
}

func Test_leave_group_conversation_handler(t *testing.T) {
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
		if w.Code != http.StatusCreated {
			t.Fatalf("setup: register %s got %d", u.name, w.Code)
		}
		db.Exec(`UPDATE users SET email_verified = 1 WHERE username = ?`, u.name)
	}

	var aliceID, bobID string
	db.QueryRow(`SELECT id FROM users WHERE username = 'alice'`).Scan(&aliceID)
	db.QueryRow(`SELECT id FROM users WHERE username = 'bob'`).Scan(&bobID)

	body := `{"member_ids":["bob","carol"],"room_key":"groupsecret","group_name":"Weekend Plans"}`
	req := requestWithUserID("POST", "/api/conversations/create", body, aliceID)
	w := httptest.NewRecorder()
	handlers.CreateConversationHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	convID := resp["conversation_id"].(string)

	if err := database.SaveMessage(db, "msg-group-1", convID, bobID, "ciphertext", 1700000000); err != nil {
		t.Fatalf("SaveMessage: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO conversation_keys (conversation_id, user_id, encrypted_key) VALUES (?, ?, ?)`, convID, aliceID, "alice-key"); err != nil {
		t.Fatalf("insert alice conversation key: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO conversation_keys (conversation_id, user_id, encrypted_key) VALUES (?, ?, ?)`, convID, bobID, "bob-key"); err != nil {
		t.Fatalf("insert bob conversation key: %v", err)
	}

	req = requestWithUserID("POST", "/api/conversations/"+convID+"/leave", "", aliceID)
	req.SetPathValue("id", convID)
	w = httptest.NewRecorder()
	handlers.LeaveConversationHandler(db, messaging.NewHub())(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for leave, got %d: %s", w.Code, w.Body.String())
	}

	var conversationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversations WHERE id = ?`, convID).Scan(&conversationCount); err != nil {
		t.Fatalf("count conversation: %v", err)
	}
	if conversationCount != 1 {
		t.Fatalf("expected group conversation to remain, got %d rows", conversationCount)
	}

	var aliceMembershipCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ? AND user_id = ?`, convID, aliceID).Scan(&aliceMembershipCount); err != nil {
		t.Fatalf("count alice membership: %v", err)
	}
	if aliceMembershipCount != 0 {
		t.Fatalf("expected alice to be removed from group conversation, got %d membership rows", aliceMembershipCount)
	}

	remainingMembers, err := database.GetConversationMembers(db, convID)
	if err != nil {
		t.Fatalf("GetConversationMembers: %v", err)
	}
	if len(remainingMembers) != 2 {
		t.Fatalf("expected two remaining group members, got %d", len(remainingMembers))
	}

	var messageCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM messages WHERE conversation_id = ?`, convID).Scan(&messageCount); err != nil {
		t.Fatalf("count messages: %v", err)
	}
	if messageCount != 2 {
		t.Fatalf("expected group messages plus leave notice to remain, got %d rows", messageCount)
	}

	var leaveNotice string
	var leaveSender sql.NullString
	if err := db.QueryRow(`
		SELECT ciphertext, sender_id
		FROM messages
		WHERE conversation_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, convID).Scan(&leaveNotice, &leaveSender); err != nil {
		t.Fatalf("load leave notice: %v", err)
	}
	if leaveNotice != "alice has left the conversation" {
		t.Fatalf("expected leave notice message, got %q", leaveNotice)
	}
	if leaveSender.Valid {
		t.Fatalf("expected leave notice to have no sender, got %q", leaveSender.String)
	}

	var remainingGroupName string
	if err := db.QueryRow(`SELECT COALESCE(group_name, '') FROM conversations WHERE id = ?`, convID).Scan(&remainingGroupName); err != nil {
		t.Fatalf("load remaining group name: %v", err)
	}
	if remainingGroupName != "" {
		t.Fatalf("expected group name to clear when only two members remain, got %q", remainingGroupName)
	}

	var aliceKeyCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_keys WHERE conversation_id = ? AND user_id = ?`, convID, aliceID).Scan(&aliceKeyCount); err != nil {
		t.Fatalf("count alice conversation key: %v", err)
	}
	if aliceKeyCount != 0 {
		t.Fatalf("expected alice conversation key to be removed, got %d rows", aliceKeyCount)
	}

	var bobKeyCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_keys WHERE conversation_id = ? AND user_id = ?`, convID, bobID).Scan(&bobKeyCount); err != nil {
		t.Fatalf("count bob conversation key: %v", err)
	}
	if bobKeyCount != 1 {
		t.Fatalf("expected remaining member conversation key to stay, got %d rows", bobKeyCount)
	}

	var bobPendingCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_pending_room_keys WHERE conversation_id = ? AND user_id = ?`, convID, bobID).Scan(&bobPendingCount); err != nil {
		t.Fatalf("count bob pending room key: %v", err)
	}
	if bobPendingCount != 1 {
		t.Fatalf("expected other members' pending room keys to remain, got %d rows", bobPendingCount)
	}

	var carolPendingCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_pending_room_keys WHERE conversation_id = ? AND user_id = ?`, convID, carolID).Scan(&carolPendingCount); err != nil {
		t.Fatalf("count carol pending room key: %v", err)
	}
	if carolPendingCount != 1 {
		t.Fatalf("expected other members' pending room keys to remain, got %d rows", carolPendingCount)
	}
}
