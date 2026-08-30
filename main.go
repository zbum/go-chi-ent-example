package main

import (
	"context"
	"encoding/json"
	"go-chi-ent-example/ent"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	_ "github.com/mattn/go-sqlite3"
)

func newRouter(client *ent.Client) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	r.Route("/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("pong"))
		})
		r.Get("/hello", helloJSON)

		r.Post("/users", createUser(client))
		r.Get("/users", listUsers(client))
	})
	return r
}

func main() {

	client, err := ent.Open("sqlite3", "file:dev.db?_fk=1")
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}
	defer client.Close()

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	http.ListenAndServe(":3000", newRouter(client))
}

func helloJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Hello, World!",
	})
}

func createUser(client *ent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		u, err := client.User.Create().
			SetName(body["Name"]).
			SetEmail(body["Email"]).
			Save(r.Context())
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	}
}
