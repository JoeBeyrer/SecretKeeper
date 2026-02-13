package handlers

import (
	"database/sql"  // Required for the *sql.DB pointer
	"encoding/json" // Required to decode the JSON from Angular

	// Required to format the response string
	"log"
	"net/http" // Required for ResponseWriter, Request, and HandlerFunc
)

func LoginHandler(db *sql.DB) http.HandlerFunc {
	return func(reponse http.ResponseWriter, request *http.Request) {
		var userInfo struct {
			Username string `json:"username"` //has to be capitalized
			Password string `json:"password"`
		}
		var storedPassword string
		json.NewDecoder(request.Body).Decode(&userInfo) //takes request username and password and places them in userInfo struct
		err := db.QueryRow("SELECT hashed_password FROM authentication_data WHERE username = ?", userInfo.Username).Scan(&storedPassword)
		if err != nil {
			log.Printf("Error in database: %a", err)
		}
		if err != nil && storedPassword == userInfo.Password {

		}
	}
}
