package main

import (
    "log"
    "net/http"
    "github.com/rs/cors"
    "secret-keeper-app/backend/database"
    "time"
)

func main() {
    db := database.InitDB("./database/secretkeeper.db")
    defer db.Close()

    mux := http.NewServeMux()

    // API routes
    mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
            log.Println("write error:", err)
        }
    })

    // Wrap the mux with CORS middleware
    handler := cors.New(cors.Options{
        AllowedOrigins: []string{"http://localhost:4200"},
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders: []string{"Content-Type", "Authorization"},
        AllowCredentials: true,
    }).Handler(mux)

    log.Println("Server running on http://localhost:8080")
    server := &http.Server{
        Addr:         ":8080",
        Handler:      handler,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    log.Fatal(server.ListenAndServe())
}