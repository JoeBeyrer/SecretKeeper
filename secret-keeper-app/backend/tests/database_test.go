package main

import (
	"database/sql"
	"secret-keeper-app/backend/database"
	"testing"
	"time"
)

func Test_init_db_func(t *testing.T) {
	var id, username, email, password_hash string
	var created_at int64
	db := database.InitDB(":memory:")
	defer db.Close()

	if _, err := db.Exec(`
	 	CREATE TABLE IF NOT EXISTS users (
            id TEXT PRIMARY KEY,
            username TEXT UNIQUE NOT NULL,
            email TEXT UNIQUE NOT NULL,
            password_hash TEXT NOT NULL,
            created_at INTEGER NOT NULL
        )`); err != nil {
		t.Fatalf("create table failed because of %v", err)
	} else {
		t.Log("successfully created table")
	}

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
		WHERE session_id = ?`,
		sessionID,
	)

	err = db.QueryRow(`
		DELETE FROM sessions
		WHERE id = ?`,
		sessionID,
	).Scan(err)

	if err != sql.ErrNoRows {
		t.Fatalf("row still exists after deletion %v", err)
	} else {
		t.Log("successfully deleted session")
	}
}
