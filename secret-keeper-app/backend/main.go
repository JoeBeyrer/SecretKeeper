package main

import (
    "log"
    "net/http"
    "time"

    "github.com/joho/godotenv" //adding this for .env file loading for linux
    "github.com/rs/cors"
    "secret-keeper-app/backend/database"
    "secret-keeper-app/backend/handlers"
    "secret-keeper-app/backend/messaging"
)

func main() {
    if err := godotenv.Load(); err != nil {
        log.Println("There isn't a .env variable (you can probably ignore this if not running linux terminal):", err)
    } // .env loading

    db := database.InitDB("./database/secretkeeper.db")
    defer db.Close()

    // Background goroutine that sweeps expired reset tokens
    handlers.StartTokenCleanup(db)

    mux := http.NewServeMux()

    hub := messaging.NewHub() // messaging

    // Background goroutine that sweeps expired messages
    database.CleanupMessages(db, hub)

    // Health check
    mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status":"ok"}`))
    })

    // PUBLIC ROUTES
    mux.HandleFunc("/api/register", handlers.RegisterHandler(db))
    mux.HandleFunc("/api/login", handlers.LoginHandler(db, 24*time.Hour))
    mux.HandleFunc("/api/verify-email", handlers.VerifyEmailHandler(db))
    mux.HandleFunc("/api/verify-email-change", handlers.VerifyEmailChangeHandler(db))
    mux.HandleFunc("/api/logout", handlers.LogoutHandler(db))

    // PASSWORD RESET ROUTES
    mux.HandleFunc("/api/password-reset/request", handlers.ForgotPasswordHandler(db))
    mux.HandleFunc("/api/password-reset/validate", handlers.ValidateResetTokenHandler(db))
    mux.HandleFunc("/api/password-reset/confirm", handlers.ResetPasswordHandler(db))

    // PROTECTED ROUTES
    auth := handlers.AuthMiddleware(db)
    mux.Handle("/ws", auth(http.HandlerFunc(handlers.WebSocketHandler(hub, db))))
    mux.Handle("/api/profile", auth(http.HandlerFunc(handlers.GetProfileHandler(db))))
    mux.Handle("/api/profile/update", auth(http.HandlerFunc(handlers.UpdateProfileHandler(db, hub))))
    mux.Handle("/api/profile/picture", auth(http.HandlerFunc(handlers.UploadProfilePictureHandler(db, hub))))
    mux.Handle("/api/profile/by-username/{username}", auth(http.HandlerFunc(handlers.GetProfileByUsernameHandler(db))))
    mux.Handle("/api/account", auth(http.HandlerFunc(handlers.UpdateAccountHandler(db, hub))))

    // CONVERSATION ROUTES
    mux.Handle("/api/conversations/create", auth(http.HandlerFunc(handlers.CreateConversationHandler(db, hub))))
    mux.Handle("/api/conversations/get", auth(http.HandlerFunc(handlers.GetConversationsHandler(db))))
    mux.Handle("/api/conversations/{id}/messages", auth(http.HandlerFunc(handlers.GetConversationMessagesHandler(db))))
    mux.Handle("/api/conversations/{id}/verify-room-key", auth(http.HandlerFunc(handlers.VerifyConversationRoomKeyHandler(db))))
    mux.Handle("/api/conversations/{id}/claim-room-key", auth(http.HandlerFunc(handlers.ClaimConversationRoomKeyHandler(db))))
    mux.Handle("/api/conversations/{id}/lifetime", auth(http.HandlerFunc(handlers.SetMessageLifetimeHandler(db, hub))))
    mux.Handle("/api/messages/{id}/react", auth(http.HandlerFunc(handlers.ToggleMessageReactionHandler(db, hub))))
    mux.Handle("/api/messages/{id}", auth(http.HandlerFunc(handlers.MessageHandler(db, hub))))
    mux.Handle("/api/conversations/{id}/leave", auth(http.HandlerFunc(handlers.LeaveConversationHandler(db, hub))))
    mux.Handle("/api/conversations/{id}/members", auth(http.HandlerFunc(handlers.GetConversationMembersHandler(db))))
    mux.Handle("/api/conversations/{id}/group-name", auth(http.HandlerFunc(handlers.UpdateGroupNameHandler(db, hub))))
    mux.Handle("/api/conversations/{id}/group-picture", auth(http.HandlerFunc(handlers.UploadGroupPictureHandler(db, hub))))
    mux.Handle("/api/conversations/{id}/members/remove", auth(http.HandlerFunc(handlers.RemoveConversationMembersHandler(db, hub))))
    mux.Handle("/api/conversations/{id}/members/add", auth(http.HandlerFunc(handlers.AddConversationMembersHandler(db, hub))))
    
    // FRIENDS ROUTES
  	mux.Handle("/api/friends", auth(http.HandlerFunc(handlers.GetFriendsHandler(db))))
  	mux.Handle("/api/friends/requests", auth(http.HandlerFunc(handlers.GetPendingRequestsHandler(db))))
  	mux.Handle("/api/friends/request", auth(http.HandlerFunc(handlers.SendFriendRequestHandler(db))))
  	mux.Handle("/api/friends/accept", auth(http.HandlerFunc(handlers.AcceptFriendRequestHandler(db))))
  	mux.Handle("/api/friends/decline", auth(http.HandlerFunc(handlers.DeclineFriendRequestHandler(db))))
  	mux.Handle("/api/friends/rescind", auth(http.HandlerFunc(handlers.RescindFriendRequestHandler(db))))
  	mux.Handle("/api/friends/remove", auth(http.HandlerFunc(handlers.RemoveFriendHandler(db))))

    // USER SEARCH
    mux.Handle("/api/users/search", auth(http.HandlerFunc(handlers.SearchUsersHandler(db))))

    // BLOCKED ROUTES
    mux.Handle("/api/blocks/block/{blockee_id}", auth(http.HandlerFunc(handlers.BlockUser(db))))
    mux.Handle("/api/blocks/unblock/{blockee_id}", auth(http.HandlerFunc(handlers.UnblockUser(db))))

    // ENCRYPTION ROUTES
    mux.Handle("/api/keys/save", auth(http.HandlerFunc(handlers.SaveKeysHandler(db))))
    mux.Handle("/api/keys/get", auth(http.HandlerFunc(handlers.GetKeysHandler(db))))
    mux.Handle("/api/users/{username}/public-key", auth(http.HandlerFunc(handlers.GetPublicKeyHandler(db))))
    mux.Handle("/api/conversations/{id}/keys", auth(http.HandlerFunc(handlers.SaveConversationKeyHandler(db))))
    mux.Handle("/api/conversations/{id}/key", auth(http.HandlerFunc(handlers.GetConversationKeyHandler(db))))

    // TEMPORARY FOR TESTING _ REMOVE OR COMMENT
    mux.Handle("/api/test-auth", auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID, ok := handlers.GetUserIDFromContext(r)
        if !ok {
            http.Error(w, "no user in context", http.StatusInternalServerError)
            return
        }

        w.Write([]byte("Authenticated user ID: " + userID))
    })))

    // CORS
    handler := cors.New(cors.Options{
        AllowedOrigins: []string{"http://localhost:4200"},
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
        AllowedHeaders: []string{"Content-Type"},
        AllowCredentials: true,
    }).Handler(mux)

    server := &http.Server{
        Addr: ":8080",
        Handler: handler,
        ReadTimeout: 10 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout: 60 * time.Second,
    }

    log.Println("Server running on http://localhost:8080")
    log.Fatal(server.ListenAndServe())
}
