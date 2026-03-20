package main

import (
	"secret-keeper-app/backend/database"
	"testing"
)

func Test_init_db_func(t *testing.T) {
	db := database.InitDB("db_init_test.db")
	defer db.Close()
	test_sql_statement := `
	 	CREATE TABLE IF NOT EXISTS users (
            id TEXT PRIMARY KEY,
            username TEXT UNIQUE NOT NULL,
            email TEXT UNIQUE NOT NULL,
            password_hash TEXT NOT NULL,
            created_at INTEGER NOT NULL
        )`
	if _, err := db.Exec(test_sql_statement); err != nil {
		t.Fatalf("test_sql_statement failed because of %e", err)
	}
	//need to destroy db at end
}
