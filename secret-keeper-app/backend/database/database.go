package database

import (
	"database/sql"
	"log"
	_ "modernc.org/sqlite"
)

func InitDB(path string) *sql.DB {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	execOrFatal(db, `PRAGMA journal_mode=WAL;`)  // WAL makes fewer db locked issues
	execOrFatal(db, `PRAGMA foreign_keys = ON;`) // explicitly allow foreing keys

	execOrFatal(db, `
        CREATE TABLE IF NOT EXISTS users (
            id TEXT PRIMARY KEY,
            username TEXT UNIQUE NOT NULL,
            email TEXT UNIQUE NOT NULL,
            password_hash TEXT NOT NULL,
            created_at INTEGER NOT NULL,
            email_verified INTEGER NOT NULL DEFAULT 0
        )
    `)

	execOrFatal(db, `
        CREATE TABLE IF NOT EXISTS user_profiles (
            user_id TEXT PRIMARY KEY,
            display_name TEXT,
            bio TEXT,
            profile_picture_url TEXT,
            FOREIGN KEY (user_id) REFERENCES users(id)
        )
    `)

	execOrFatal(db, `
        CREATE TABLE IF NOT EXISTS sessions (
            id TEXT PRIMARY KEY,               
            user_id TEXT NOT NULL,
            created_at INTEGER NOT NULL,
            expires_at INTEGER NOT NULL,
            FOREIGN KEY (user_id) REFERENCES users(id)
        )
    `)

	execOrFatal(db, `
        CREATE TABLE IF NOT EXISTS email_verifications (
            id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            token TEXT NOT NULL,
            created_at INTEGER NOT NULL,
            expires_at INTEGER NOT NULL,
            new_email TEXT DEFAULT '',
            FOREIGN KEY (user_id) REFERENCES users(id)
        )
    `)


	execOrFatal(db, `
        CREATE TABLE IF NOT EXISTS password_resets (
            id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            token TEXT NOT NULL,
            created_at INTEGER NOT NULL,
            expires_at INTEGER NOT NULL,
            used INTEGER DEFAULT 0,
            FOREIGN KEY (user_id) REFERENCES users(id)
        )
    `)

	execOrFatal(db, `
        CREATE TABLE IF NOT EXISTS password_reset_audit (
            id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            token TEXT NOT NULL,
            created_at INTEGER NOT NULL,
            expires_at INTEGER NOT NULL,
            archived_at INTEGER NOT NULL,
            reason TEXT NOT NULL
        )
    `)

	execOrFatal(db, `
        CREATE TABLE IF NOT EXISTS conversations (
            id TEXT PRIMARY KEY,
            created_at INTEGER
        )
    `)

	execOrFatal(db, `
        CREATE TABLE IF NOT EXISTS conversation_members (
            conversation_id TEXT,
            user_id TEXT,
            joined_at INTEGER,
            PRIMARY KEY (conversation_id, user_id),
            FOREIGN KEY (conversation_id) REFERENCES conversations(id)
            FOREIGN KEY (user_id) REFERENCES users(id)
        )
    `)

	execOrFatal(db, `
        CREATE TABLE IF NOT EXISTS messages (
            id TEXT PRIMARY KEY,
            conversation_id TEXT,
            sender_id TEXT,
            ciphertext BLOB NOT NULL,
            created_at INTEGER,
            expires_at INTEGER,
            FOREIGN KEY (conversation_id) REFERENCES conversations(id),
            FOREIGN KEY (sender_id) REFERENCES users(id)
        )
    `)

    execOrFatal(db, `
        CREATE TABLE IF NOT EXISTS friendships (
            id TEXT PRIMARY KEY,
            requester_id TEXT NOT NULL,
            addressee_id TEXT NOT NULL,
            accepted INTEGER NOT NULL DEFAULT 0,
            created_at INTEGER NOT NULL,
            updated_at INTEGER NOT NULL,
            FOREIGN KEY (requester_id) REFERENCES users(id),
            FOREIGN KEY (addressee_id) REFERENCES users(id),
            UNIQUE (requester_id, addressee_id)
        )
    `)

    execOrFatal(db, `
        CREATE TABLE IF NOT EXISTS user_keys (
            user_id TEXT PRIMARY KEY,
            public_key TEXT NOT NULL,
            encrypted_private_key TEXT NOT NULL,
            FOREIGN KEY (user_id) REFERENCES users(id)
        )
    `)

    execOrFatal(db, `
        CREATE TABLE IF NOT EXISTS conversation_keys (
            conversation_id TEXT NOT NULL,
            user_id TEXT NOT NULL,
            encrypted_key TEXT NOT NULL,
            PRIMARY KEY (conversation_id, user_id),
            FOREIGN KEY (conversation_id) REFERENCES conversations(id),
            FOREIGN KEY (user_id) REFERENCES users(id)
        )
    `)

	log.Println("Database initialized")
	return db
}

func execOrFatal(db *sql.DB, query string) {
	if _, err := db.Exec(query); err != nil {
		log.Fatal(err)
	}
}
