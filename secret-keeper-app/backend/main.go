package main

import (
    "log"
    "net/http"
    "time"

    "github.com/rs/cors"
    "secret-keeper-app/backend/database"
    "secret-keeper-app/backend/handlers"
)

func main() {
    db := database.InitDB("./database/secretkeeper.db")
    defer db.Close()

    mux := http.NewServeMux()

    // Health check
    mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"status":"ok"}`))
    })

    // PUBLIC ROUTES
    mux.HandleFunc("/api/register", handlers.RegisterHandler(db))
    mux.HandleFunc("/api/login", handlers.LoginHandler(db, 24*time.Hour))

    // // PROTECTED ROUTES
    // auth := handlers.AuthMiddleware(db)
    // mux.Handle("/api/conversations/create", auth(http.HandlerFunc(handlers.CreateConversationHandler(db))))
    // mux.Handle("/api/messages/send", auth(http.HandlerFunc(handlers.SendMessageHandler(db))))

    // TEMPORARY FOR TESTING _ REMOVE OR COMMENT
    auth := handlers.AuthMiddleware(db)
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
        AllowedOrigins:   []string{"http://localhost:4200"},
        AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"Content-Type"},
        AllowCredentials: true,
    }).Handler(mux)

    server := &http.Server{
        Addr:         ":8080",
        Handler:      handler,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    log.Println("Server running on http://localhost:8080")
    log.Fatal(server.ListenAndServe())
}
