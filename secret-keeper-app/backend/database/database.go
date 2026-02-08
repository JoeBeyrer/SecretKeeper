package main

import (
	"database/sql"
	"log"
	_ "modernc.org/sqlite"
)

func main() {
	connection, db_error := sql.Open("sqlite", "user_management.db")
	if db_error != nil {
		log.Fatal(db_error)
	}
	db_error = connection.Ping()
	if db_error != nil {
		log.Fatal(db_error)
	}
	result, db_error := connection.Exec(
		`CREATE TABLE IF NOT EXISTS authentication_data (
			username TEXT NOT NULL,
			password TEXT NOT NULL
		)`,
	)
	initial_query, db_error := connection.Exec(
		`INSERT INTO authentication_data (username, password)
		VALUES ("admin","coolpassword")`,
	)
	_ = initial_query
	_ = result
	log.Printf("Succesfully created database")

	defer connection.Close()
}
