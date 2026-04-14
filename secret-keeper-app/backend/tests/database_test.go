package main

import (
	"database/sql"
	"io"
	"log"
	"os"
	"secret-keeper-app/backend/database"
	"secret-keeper-app/backend/handlers"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	handlers.SendVerificationEmail = func(to, token string) error { return nil }
	os.Exit(m.Run())
}

func Test_init_db_func(t *testing.T) {
	var id, username, email, password_hash string
	var created_at int64
	db := database.InitDB(":memory:")
	defer db.Close()

	if _, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("9e99af6b-48e4-4eeb-951f-0cb27e03e32c", "testuser", "testuser@gmail.com", "$2y$12$XElWz9WPwSLK3y0jUP6KhOHepv.KF4zj6z4J3XXyYRye.VXnPsMA2", 1742467200)
	`); err != nil {
		t.Fatalf("insert into table failed because of %v", err)
	} else {
		t.Log("successfully inserted into table")
	}

	err := db.QueryRow(`
		SELECT id, username, email, password_hash, created_at FROM users WHERE username = ?`, "testuser",
	).Scan(&id, &username, &email, &password_hash, &created_at)

	if err != nil {
		t.Fatalf("select from table failed because of: %v", err)
	} else {
		t.Log("successfully selected items from table")
	}

	if id != "9e99af6b-48e4-4eeb-951f-0cb27e03e32c" || username != "testuser" || email != "testuser@gmail.com" || password_hash != "$2y$12$XElWz9WPwSLK3y0jUP6KhOHepv.KF4zj6z4J3XXyYRye.VXnPsMA2" || created_at != 1742467200 {
		t.Fatal("select from table succeeded but output was unexpected")
	} else {
		t.Log("successfully verified selected output")
	}
}

func Test_create_session_func(t *testing.T) {
	var sessionID, userID string
	var created_at, expires_at int64
	db := database.InitDB(":memory:")
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("9e99af6b-48e4-4eeb-951f-0cb27e03e32c", "testuser", "testuser@gmail.com", "hashedpassword", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	sessionID, expires_at, err = database.CreateSession(db, "9e99af6b-48e4-4eeb-951f-0cb27e03e32c", 24*time.Hour)
	if err != nil {
		t.Fatalf("error when creating session %v", err)
	} else {
		t.Log("successfully created session")
	}

	err = db.QueryRow(`
		SELECT id, user_id, created_at, expires_at FROM sessions WHERE user_id = ?`, "9e99af6b-48e4-4eeb-951f-0cb27e03e32c",
	).Scan(&sessionID, &userID, &created_at, &expires_at)

	if err != nil {
		t.Fatalf("select from table failed because of: %v", err)
	} else {
		t.Log("successfully selected items from table")
	}

	if sessionID == "" || userID != "9e99af6b-48e4-4eeb-951f-0cb27e03e32c" || created_at == int64(0) || expires_at == 0 {
		t.Fatal("data selected from table does not match inputted data")
	} else {
		t.Log("data selected from table matches inputted data")
	}
}

func Test_delete_session_func(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("9e99af6b-48e4-4eeb-951f-0cb27e03e32c", "testuser", "testuser@gmail.com", "hashedpassword", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	sessionID, _, err := database.CreateSession(db, "9e99af6b-48e4-4eeb-951f-0cb27e03e32c", 24*time.Hour)
	if err != nil {
		t.Fatalf("error when creating session %v", err)
	} else {
		t.Log("successfully created session")
	}

	_, err = db.Exec(`
		DELETE FROM sessions
		WHERE id = ?`,
		sessionID,
	)
	if err != nil {
		t.Fatalf("error deleting because of: %v", err)
	} else {
		t.Log("successfully deleted session")
	}

	var id string
	err = db.QueryRow(`
		SELECT id FROM sessions WHERE id = ?`, sessionID,
	).Scan(&id)

	if err != sql.ErrNoRows {
		t.Fatalf("row still exists after deletion %v", err)
	} else {
		t.Log("successfully deleted session")
	}
}

func Test_send_friend_request_func(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("user1", "requester", "requester@gmail.com", "password", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert requester user: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("user2", "addressee", "addressee@gmail.com", "password", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert addressee user: %v", err)
	}

	err = database.SendFriendRequest(db, "user1", "user2")
	if err != nil {
		t.Fatalf("failed to send friend request: %v", err)
	} else {
		t.Log("successfully sent friend request")
	}

	var requesterID, addresseeID string
	var accepted int
	err = db.QueryRow(`
		SELECT requester_id, addressee_id, accepted
		FROM friendships
		WHERE requester_id = ? AND addressee_id = ?`,
		"user1", "user2",
	).Scan(&requesterID, &addresseeID, &accepted)

	if err == sql.ErrNoRows {
		t.Fatal("unable to get inserted row from friendships table")
	} else {
		t.Log("successfully got inserted rows from friendships table")
	}

	if requesterID != "user1" || addresseeID != "user2" || accepted != 0 {
		t.Fatal("selected information does not match friendships table inputted info")
	} else {
		t.Log("selected information matches friendships table inputted info")
	}
}

func Test_accept_friend_request_func(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("user1", "requester", "requester@gmail.com", "password", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert requester user: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("user2", "addressee", "addressee@gmail.com", "password", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert addressee user: %v", err)
	}

	err = database.SendFriendRequest(db, "user1", "user2")
	if err != nil {
		t.Fatalf("failed to send friend request: %v", err)
	} else {
		t.Log("successfully sent friend request")
	}

	err = database.AcceptFriendRequest(db, "user2", "user1")
	if err != nil {
		t.Fatalf("failed to accept friend request: %v", err)
	} else {
		t.Log("successfully accepted friend request")
	}

	var accepted int
	err = db.QueryRow(`
		SELECT accepted
		FROM friendships
		WHERE requester_id = ? AND addressee_id = ?`,
		"user1", "user2",
	).Scan(&accepted)

	if err == sql.ErrNoRows {
		t.Fatal("unable to get inserted row from friendships table")
	} else {
		t.Log("successfully got inserted rows from friendships table")
	}

	if accepted == 0 {
		t.Fatal("acceptfriendrequest function did not change accepted value")
	} else {
		t.Log("acceptfriendrequest changed accepted value")
	}
}

func Test_decline_friend_request_func(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("user1", "requester", "requester@gmail.com", "password", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert requester user: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("user2", "addressee", "addressee@gmail.com", "password", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert addressee user: %v", err)
	}

	err = database.SendFriendRequest(db, "user1", "user2")
	if err != nil {
		t.Fatalf("failed to send friend request: %v", err)
	} else {
		t.Log("successfully sent friend request")
	}

	err = database.DeclineFriendRequest(db, "user2", "user1")
	if err != nil {
		t.Fatalf("failed to decline friend request: %v", err)
	} else {
		t.Log("successfully declined friend request")
	}

	var accepted int
	err = db.QueryRow(`
		SELECT accepted
		FROM friendships
		WHERE requester_id = ? AND addressee_id = ?`,
		"user1", "user2",
	).Scan(&accepted)

	if err != sql.ErrNoRows {
		t.Fatal("friendship row still exists after deletion")
	} else {
		t.Log("successfully deleted friendship from friendships table")
	}
}

func Test_remove_friend_func(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("user1", "requester", "requester@gmail.com", "password", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert requester user: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("user2", "addressee", "addressee@gmail.com", "password", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert addressee user: %v", err)
	}

	err = database.SendFriendRequest(db, "user1", "user2")
	if err != nil {
		t.Fatalf("failed to send friend request: %v", err)
	} else {
		t.Log("successfully sent friend request")
	}

	err = database.AcceptFriendRequest(db, "user2", "user1")
	if err != nil {
		t.Fatalf("failed to accept friend request: %v", err)
	} else {
		t.Log("successfully accepted friend request")
	}

	err = database.RemoveFriend(db, "user2", "user1")
	if err != nil {
		t.Fatalf("remove friend function failed because of: %v", err)
	} else {
		t.Log("successfully removed friendship from friendships table")
	}

	var exists int
	err = db.QueryRow(`
		SELECT 1
		FROM friendships
		WHERE requester_id = ? AND addressee_id = ?`,
		"user1", "user2",
	).Scan(&exists)

	if err != sql.ErrNoRows {
		t.Fatal("friendship row still exists after removal")
	} else {
		t.Log("successfully removed friendship from friendships table")
	}
}

func Test_get_friends_func(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("user1", "requester", "requester@gmail.com", "password", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert requester user: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("user2", "addressee", "addressee@gmail.com", "password", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert addressee user: %v", err)
	}

	err = database.SendFriendRequest(db, "user1", "user2")
	if err != nil {
		t.Fatalf("failed to send friend request: %v", err)
	} else {
		t.Log("successfully sent friend request")
	}

	err = database.AcceptFriendRequest(db, "user2", "user1")
	if err != nil {
		t.Fatalf("failed to accept friend request: %v", err)
	} else {
		t.Log("successfully accepted friend request")
	}

	friends, err := database.GetFriends(db, "user1")
	if err != nil {
		t.Fatalf("failed to get friends: %v", err)
	} else {
		t.Log("successfully got friends")
	}

	if len(friends) != 1 {
		t.Fatalf("expected 1 friend, got %d", len(friends))
	}

	if friends[0].UserID != "user2" || friends[0].Username != "addressee" || friends[0].Accepted != true {
		t.Fatal("friend entry does not match expected values")
	} else {
		t.Log("friend entry matches expected values")
	}
}

func Test_get_pending_requests_func(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("user1", "requester", "requester@gmail.com", "password", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert requester user: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("user2", "addressee", "addressee@gmail.com", "password", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert addressee user: %v", err)
	}

	err = database.SendFriendRequest(db, "user1", "user2")
	if err != nil {
		t.Fatalf("failed to send friend request: %v", err)
	} else {
		t.Log("successfully sent friend request")
	}

	friends, err := database.GetPendingRequests(db, "user1")
	if err != nil {
		t.Fatalf("failed to get friends: %v", err)
	} else {
		t.Log("successfully got friends")
	}

	if len(friends) != 1 {
		t.Fatalf("expected 1 friend, got %d", len(friends))
	}

	if friends[0].UserID != "user2" || friends[0].Username != "addressee" || friends[0].Accepted == true || friends[0].Direction != "outgoing" {
		t.Fatal("friend entry does not match expected values")
	} else {
		t.Log("friend entry matches expected values")
	}
}

func Test_friendship_exists_func(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("user1", "requester", "requester@gmail.com", "password", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert requester user: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("user2", "addressee", "addressee@gmail.com", "password", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert addressee user: %v", err)
	}

	err = database.SendFriendRequest(db, "user1", "user2")
	if err != nil {
		t.Fatalf("failed to send friend request: %v", err)
	} else {
		t.Log("successfully sent friend request")
	}

	err = database.AcceptFriendRequest(db, "user2", "user1")
	if err != nil {
		t.Fatalf("failed to accept friend request: %v", err)
	} else {
		t.Log("successfully accepted friend request")
	}

	exists, accepted, direction, err := database.FriendshipExists(db, "user1", "user2")
	if err != nil {
		t.Fatalf("failed to get friendship status because of: %v", err)
	} else {
		t.Log("successfully got friendships status")
	}

	if exists != true || accepted != true || direction != "outgoing" {
		t.Fatal("friendship status does not match expected values")
	} else {
		t.Log("friendship status matches expected values")
	}
	//incoming direction
	exists, accepted, direction, err = database.FriendshipExists(db, "user2", "user1")
	if err != nil {
		t.Fatalf("failed to get friendship status because of: %v", err)
	} else {
		t.Log("successfully got friendships status")
	}

	if exists != true || accepted != true || direction != "incoming" {
		t.Fatal("friendship status does not match expected values")
	} else {
		t.Log("friendship status matches expected values")
	}
}

func Test_get_user_id_by_username_func(t *testing.T) {
	db := database.InitDB(":memory:")
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("user1", "requester", "requester@gmail.com", "password", 1740067200)
	`)
	if err != nil {
		t.Fatalf("failed to insert requester user: %v", err)
	}

	id, err := database.GetUserIDByUsername(db, "requester")
	if err != nil {
		t.Fatal("failed to get user id by username")
	} else {
		t.Log("got user id by username")
	}

	if id != "user1" {
		t.Fatal("returned the incorrect id by username")
	} else {
		t.Log("returned the correct id by username")
	}
	//nonexistent username
	_, err = database.GetUserIDByUsername(db, "nonexistent")
	if err != sql.ErrNoRows {
		t.Fatal("expected sql.ErrNoRows for nonexistent user")
	} else {
		t.Log("correctly returned sql.ErrNoRows for nonexistent user")
	}
}
