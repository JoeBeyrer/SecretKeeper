package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"secret-keeper-app/backend/database"
	"secret-keeper-app/backend/models"
)


func SendFriendRequestHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		requesterID, ok := GetUserIDFromContext(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			Username string `json:"username"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
			http.Error(w, "username required", http.StatusBadRequest)
			return
		}

		addresseeID, err := database.GetUserIDByUsername(db, req.Username)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		if addresseeID == requesterID {
			http.Error(w, "you cannot add yourself", http.StatusBadRequest)
			return
		}

		exists, accepted, _, err := database.FriendshipExists(db, requesterID, addresseeID)
		if err == nil && exists {
			if accepted {
				http.Error(w, "already friends", http.StatusConflict)
			} else {
				http.Error(w, "friend request already pending", http.StatusConflict)
			}
			return
		}

		if err := database.SendFriendRequest(db, requesterID, addresseeID); err != nil {
			http.Error(w, "could not send friend request", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"message":"Friend request sent."}`))
	}
}

func AcceptFriendRequestHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		addresseeID, ok := GetUserIDFromContext(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			Username string `json:"username"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
			http.Error(w, "username required", http.StatusBadRequest)
			return
		}

		requesterID, err := database.GetUserIDByUsername(db, req.Username)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		if err := database.AcceptFriendRequest(db, addresseeID, requesterID); err != nil {
			http.Error(w, "could not accept friend request", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"Friend request accepted."}`))
	}
}

func DeclineFriendRequestHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		addresseeID, ok := GetUserIDFromContext(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			Username string `json:"username"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
			http.Error(w, "username required", http.StatusBadRequest)
			return
		}

		requesterID, err := database.GetUserIDByUsername(db, req.Username)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		if err := database.DeclineFriendRequest(db, addresseeID, requesterID); err != nil {
			http.Error(w, "could not decline friend request", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"Friend request declined."}`))
	}
}

func RemoveFriendHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID, ok := GetUserIDFromContext(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			Username string `json:"username"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
			http.Error(w, "username required", http.StatusBadRequest)
			return
		}

		otherID, err := database.GetUserIDByUsername(db, req.Username)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		if err := database.RemoveFriend(db, userID, otherID); err != nil {
			http.Error(w, "could not remove friend", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"Friend removed."}`))
	}
}

func GetFriendsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID, ok := GetUserIDFromContext(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		friends, err := database.GetFriends(db, userID)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		if friends == nil {
			friends = []models.FriendEntry{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(friends)
	}
}

func SearchUsersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		callerID, ok := GetUserIDFromContext(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		query := r.URL.Query().Get("q")
		if len(query) < 1 {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[]`))
			return
		}

		results, err := database.SearchUsers(db, callerID, query)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		if results == nil {
			results = []models.UserSearchResult{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}

func GetPendingRequestsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID, ok := GetUserIDFromContext(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		requests, err := database.GetPendingRequests(db, userID)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		if requests == nil {
			requests = []models.FriendEntry{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(requests)
	}
}