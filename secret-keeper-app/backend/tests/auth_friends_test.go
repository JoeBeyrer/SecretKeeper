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

func Test_send_friend_request_handler(t *testing.T) {
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

	// No context user — unauthorized
	req := httptest.NewRequest("POST", "/api/friends/request", strings.NewReader(`{"username":"bob"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.SendFriendRequestHandler(db)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no auth, got %d", w.Code)
	} else {
		t.Log("no auth correctly returned 401")
	}

	// Empty username in body
	req = requestWithUserID("POST", "/api/friends/request", `{}`, aliceID)
	w = httptest.NewRecorder()
	handlers.SendFriendRequestHandler(db)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty username, got %d", w.Code)
	} else {
		t.Log("empty username correctly returned 400")
	}

	// Unknown username
	req = requestWithUserID("POST", "/api/friends/request", `{"username":"nobody"}`, aliceID)
	w = httptest.NewRecorder()
	handlers.SendFriendRequestHandler(db)(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown user, got %d", w.Code)
	} else {
		t.Log("unknown username correctly returned 404")
	}

	// Send request to self
	req = requestWithUserID("POST", "/api/friends/request", `{"username":"alice"}`, aliceID)
	w = httptest.NewRecorder()
	handlers.SendFriendRequestHandler(db)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for self-request, got %d", w.Code)
	} else {
		t.Log("self-request correctly returned 400")
	}

	// Valid request
	req = requestWithUserID("POST", "/api/friends/request", `{"username":"bob"}`, aliceID)
	w = httptest.NewRecorder()
	handlers.SendFriendRequestHandler(db)(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for valid request, got %d", w.Code)
	} else {
		t.Log("valid friend request correctly returned 201")
	}

	// Confirm pending row in DB
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM friendships WHERE requester_id=? AND addressee_id=? AND accepted=0`, aliceID, bobID).Scan(&count)
	if count != 1 {
		t.Fatal("friendship row not created in DB")
	} else {
		t.Log("friendship row correctly inserted with accepted=0")
	}

	// Duplicate request conflict
	req = requestWithUserID("POST", "/api/friends/request", `{"username":"bob"}`, aliceID)
	w = httptest.NewRecorder()
	handlers.SendFriendRequestHandler(db)(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate request, got %d", w.Code)
	} else {
		t.Log("duplicate request correctly returned 409")
	}
}

func Test_accept_friend_request_handler(t *testing.T) {
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

	// Alice sends bob a request
	if err := database.SendFriendRequest(db, aliceID, bobID); err != nil {
		t.Fatalf("setup: SendFriendRequest: %v", err)
	}

	// No auth
	req := httptest.NewRequest("POST", "/api/friends/accept", strings.NewReader(`{"username":"alice"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.AcceptFriendRequestHandler(db)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no auth, got %d", w.Code)
	} else {
		t.Log("no auth correctly returned 401")
	}

	// Unknown requester
	req = requestWithUserID("POST", "/api/friends/accept", `{"username":"nobody"}`, bobID)
	w = httptest.NewRecorder()
	handlers.AcceptFriendRequestHandler(db)(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown user, got %d", w.Code)
	} else {
		t.Log("unknown requester correctly returned 404")
	}

	// Bob accepts alice's request
	req = requestWithUserID("POST", "/api/friends/accept", `{"username":"alice"}`, bobID)
	w = httptest.NewRecorder()
	handlers.AcceptFriendRequestHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid accept, got %d", w.Code)
	} else {
		t.Log("valid accept correctly returned 200")
	}

	// Confirm in DB
	var accepted int
	db.QueryRow(`SELECT accepted FROM friendships WHERE requester_id=? AND addressee_id=?`, aliceID, bobID).Scan(&accepted)
	if accepted != 1 {
		t.Fatal("friendship not marked accepted=1 in DB")
	} else {
		t.Log("friendship correctly marked accepted=1 in DB")
	}
}

func Test_decline_friend_request_handler(t *testing.T) {
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

	if err := database.SendFriendRequest(db, aliceID, bobID); err != nil {
		t.Fatalf("setup: SendFriendRequest: %v", err)
	}

	// No auth
	req := httptest.NewRequest("POST", "/api/friends/decline", strings.NewReader(`{"username":"alice"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.DeclineFriendRequestHandler(db)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no auth, got %d", w.Code)
	} else {
		t.Log("no auth correctly returned 401")
	}

	// Unknown requester
	req = requestWithUserID("POST", "/api/friends/decline", `{"username":"nobody"}`, bobID)
	w = httptest.NewRecorder()
	handlers.DeclineFriendRequestHandler(db)(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown user, got %d", w.Code)
	} else {
		t.Log("unknown requester correctly returned 404")
	}

	// Bob declines alice's request
	req = requestWithUserID("POST", "/api/friends/decline", `{"username":"alice"}`, bobID)
	w = httptest.NewRecorder()
	handlers.DeclineFriendRequestHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid decline, got %d", w.Code)
	} else {
		t.Log("valid decline correctly returned 200")
	}

	// Confirm deletion in DB
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM friendships WHERE requester_id=? AND addressee_id=?`, aliceID, bobID).Scan(&count)
	if count != 0 {
		t.Fatal("friendship row not deleted after decline")
	} else {
		t.Log("friendship row correctly deleted from DB after decline")
	}
}

func Test_remove_friend_handler(t *testing.T) {
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

	// Prepopulated an accepted friendship
	database.SendFriendRequest(db, aliceID, bobID)
	database.AcceptFriendRequest(db, bobID, aliceID)

	// No auth
	req := httptest.NewRequest("DELETE", "/api/friends/remove", strings.NewReader(`{"username":"bob"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers.RemoveFriendHandler(db)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no auth, got %d", w.Code)
	} else {
		t.Log("no auth correctly returned 401")
	}

	// Unknown user
	req = requestWithUserID("DELETE", "/api/friends/remove", `{"username":"nobody"}`, aliceID)
	w = httptest.NewRecorder()
	handlers.RemoveFriendHandler(db)(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown user, got %d", w.Code)
	} else {
		t.Log("unknown user correctly returned 404")
	}

	// Valid remove
	req = requestWithUserID("DELETE", "/api/friends/remove", `{"username":"bob"}`, aliceID)
	w = httptest.NewRecorder()
	handlers.RemoveFriendHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid remove, got %d", w.Code)
	} else {
		t.Log("valid remove correctly returned 200")
	}

	// Confirm friendship gone from DB
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM friendships WHERE (requester_id=? AND addressee_id=?) OR (requester_id=? AND addressee_id=?)`,
		aliceID, bobID, bobID, aliceID).Scan(&count)
	if count != 0 {
		t.Fatal("friendship row not deleted after remove")
	} else {
		t.Log("friendship row correctly deleted from DB after remove")
	}
}

func Test_get_friends_handler(t *testing.T) {
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

	// No auth
	req := httptest.NewRequest("GET", "/api/friends", nil)
	w := httptest.NewRecorder()
	handlers.GetFriendsHandler(db)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no auth, got %d", w.Code)
	} else {
		t.Log("no auth correctly returned 401")
	}

	// Empty list — no friends yet
	req = requestWithUserID("GET", "/api/friends", "", aliceID)
	w = httptest.NewRecorder()
	handlers.GetFriendsHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for empty friends, got %d", w.Code)
	}
	var empty []interface{}
	json.NewDecoder(w.Body).Decode(&empty)
	if len(empty) != 0 {
		t.Fatalf("expected empty array, got %d items", len(empty))
	} else {
		t.Log("empty friends list correctly returned [] with 200")
	}

	// Add friend then confirm they appear
	database.SendFriendRequest(db, aliceID, bobID)
	database.AcceptFriendRequest(db, bobID, aliceID)

	req = requestWithUserID("GET", "/api/friends", "", aliceID)
	w = httptest.NewRecorder()
	handlers.GetFriendsHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after adding friend, got %d", w.Code)
	}
	var friends []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&friends)
	if len(friends) != 1 {
		t.Fatalf("expected 1 friend, got %d", len(friends))
	}
	if friends[0]["username"] != "bob" {
		t.Fatalf("expected bob in friends list, got %v", friends[0]["username"])
	} else {
		t.Log("bob correctly appears in alice's friends list after acceptance")
	}
}

func Test_get_pending_requests_handler(t *testing.T) {
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

	// No auth
	req := httptest.NewRequest("GET", "/api/friends/pending", nil)
	w := httptest.NewRecorder()
	handlers.GetPendingRequestsHandler(db)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no auth, got %d", w.Code)
	} else {
		t.Log("no auth correctly returned 401")
	}

	// Empty list
	req = requestWithUserID("GET", "/api/friends/pending", "", aliceID)
	w = httptest.NewRecorder()
	handlers.GetPendingRequestsHandler(db)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for empty pending, got %d", w.Code)
	}
	var empty []interface{}
	json.NewDecoder(w.Body).Decode(&empty)
	if len(empty) != 0 {
		t.Fatalf("expected empty array, got %d", len(empty))
	} else {
		t.Log("empty pending list correctly returned [] with 200")
	}

	// Alice sends bob a request
	database.SendFriendRequest(db, aliceID, bobID)

	// Alice sees it as outgoing
	req = requestWithUserID("GET", "/api/friends/pending", "", aliceID)
	w = httptest.NewRecorder()
	handlers.GetPendingRequestsHandler(db)(w, req)
	var alicePending []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&alicePending)
	if len(alicePending) != 1 || alicePending[0]["direction"] != "outgoing" {
		t.Fatalf("alice expected 1 outgoing, got %+v", alicePending)
	} else {
		t.Log("alice correctly sees request as outgoing")
	}

	// Bob sees it as incoming
	req = requestWithUserID("GET", "/api/friends/pending", "", bobID)
	w = httptest.NewRecorder()
	handlers.GetPendingRequestsHandler(db)(w, req)
	var bobPending []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&bobPending)
	if len(bobPending) != 1 || bobPending[0]["direction"] != "incoming" {
		t.Fatalf("bob expected 1 incoming, got %+v", bobPending)
	} else {
		t.Log("bob correctly sees request as incoming")
	}
}


