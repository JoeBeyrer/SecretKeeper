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

    db.Exec(`PRAGMA journal_mode=WAL;`) // fewer db locked issues

    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS authentication_data (
            username TEXT PRIMARY KEY,
            password TEXT NOT NULL
        )
    `)
    if err != nil {
        log.Fatal(err)
    }

    log.Println("Database initialized")
    return db
}
