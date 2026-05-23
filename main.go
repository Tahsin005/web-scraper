package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Tahsin005/web-scraper/internal/database"
	"github.com/Tahsin005/web-scraper/internal/handlers"
	"github.com/Tahsin005/web-scraper/internal/scraper"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load(".env")
	portString := os.Getenv("PORT")
	if portString == "" {
		log.Fatal("PORT is not found in the environment")
	}

	dbUrl := os.Getenv("DB_URL")
	if dbUrl == "" {
		log.Fatal("DB_URL is not found in the environment")
	}

	conn, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal("Can't connect to the database: " + err.Error())
	}

	db := database.New(conn)

	h := &handlers.Handler{
		DB: db,
	}

	go scraper.StartScraping(db, 10, time.Minute)

	router := chi.NewRouter()

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://*", "https://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	v1Router := chi.NewRouter()
	v1Router.Get("/healthz", h.HandlerReadiness)
	v1Router.Post("/users", h.HandlerCreateUser)
	v1Router.Get("/users", h.AuthedHandler(h.HandlerGetUserByAPIKey))
	v1Router.Post("/feeds", h.AuthedHandler(h.HandlerCreateFeed))
	v1Router.Get("/feeds", h.HandlerGetFeeds)
	v1Router.Post("/feed_follows", h.AuthedHandler(h.HandlerCreateFeedFollows))
	v1Router.Get("/feed_follows", h.AuthedHandler(h.HandlerGetFeedFollows))
	v1Router.Delete("/feed_follows/{feedFollowID}", h.AuthedHandler(h.HandlerDeleteFeedFollow))
	v1Router.Get("/posts", h.AuthedHandler(h.HandlerGetPostsForUser))

	router.Mount("/v1", v1Router)

	srv := &http.Server{
		Addr:    ":" + portString,
		Handler: router,
	}

	log.Printf("Server is listening on port %s", portString)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
