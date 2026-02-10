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


    execOrFatal(db, `PRAGMA journal_mode=WAL;`) // WAL makes fewer db locked issues
    execOrFatal(db, `PRAGMA foreign_keys = ON;`) // explicitly allow foreing keys



    execOrFatal(db, `
        CREATE TABLE IF NOT EXISTS authentication_data (
            username TEXT PRIMARY KEY,
            password_hash TEXT NOT NULL
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
            FOREIGN KEY (sender_id) REFERENCES authentication_data(username)
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
