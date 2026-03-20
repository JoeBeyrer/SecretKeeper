package main

import (
	"fmt"
	"secret-keeper-app/backend/database"
	"testing"
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
		fmt.Println("successfully created table")
	}

	if _, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES ("123213213", "testuser", "testuser@gmail.com", "$2y$12$XElWz9WPwSLK3y0jUP6KhOHepv.KF4zj6z4J3XXyYRye.VXnPsMA2", 1742467200)
	`); err != nil {
		t.Fatalf("insert into table failed because of %v", err)
	} else {
		fmt.Println("successfully inserted into table")
	}

	err := db.QueryRow(`
		SELECT id, username, email, password_hash, created_at FROM users WHERE username = ?`, "testuser",
	).Scan(&id, &username, &email, &password_hash, &created_at)

	if err != nil {
		t.Fatalf("select from table failed because of: %v", err)
	} else {
		fmt.Println("successfully selected items from table ")
	}

	if id != "123213213" || username != "testuser" || email != "testuser@gmail.com" || password_hash != "$2y$12$XElWz9WPwSLK3y0jUP6KhOHepv.KF4zj6z4J3XXyYRye.VXnPsMA2" || created_at != 1742467200 {
		t.Fatal("select from table succeeded but output was unexpected")
	} else {
		fmt.Println("successfully verified selected output")
	}
}
