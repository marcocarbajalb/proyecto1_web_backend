package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"seriestracker/internal/db"
	"seriestracker/internal/handlers"
	"seriestracker/internal/middleware"
)

func main() {
	dbPath := getenv("DB_PATH", "series.db")
	port := getenv("PORT", "8080")

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	series := &handlers.SeriesHandler{DB: database}
	uploads := &handlers.UploadHandler{DB: database}
	ratings := &handlers.RatingHandler{DB: database}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /series", series.List)
	mux.HandleFunc("GET /series/{id}", series.Get)
	mux.HandleFunc("POST /series", series.Create)
	mux.HandleFunc("PUT /series/{id}", series.Update)
	mux.HandleFunc("DELETE /series/{id}", series.Delete)

	mux.HandleFunc("POST /series/{id}/image", uploads.UploadSeriesImage)
	mux.HandleFunc("GET /uploads/{filename}", handlers.ServeUpload)

	mux.HandleFunc("GET /series/{id}/rating", ratings.Get)
	mux.HandleFunc("PUT /series/{id}/rating", ratings.Set)
	mux.HandleFunc("DELETE /series/{id}/rating", ratings.Delete)

	handler := middleware.CORS(mux)

	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
	    log.Fatalf("server: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}