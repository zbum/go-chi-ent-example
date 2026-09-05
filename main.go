package main

import (
	"context"
	"encoding/json"
	"go-chi-ent-example/ent"
	"log"
	"net/http"
	"strconv"
	"time"

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
		r.Get("/users/{id}", getUser(client))
		r.Get("/users/{id}/cars", listCars(client))
		r.Post("/users/{id}/cars", createCar(client))

		r.Post("/users-with-car", createUserWithCar(client))
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

type UserCreateRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UserCreateWithCarRequest struct {
	UserCreateRequest
	Model string `json:"model"`
}

func createUser(client *ent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var createUserRequest UserCreateRequest
		err := json.NewDecoder(r.Body).Decode(&createUserRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		u, err := client.User.Create().
			SetName(createUserRequest.Name).
			SetEmail(createUserRequest.Email).
			Save(r.Context())

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(u)
	}
}

func createUserWithCar(client *ent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var createUserWithCarRequest UserCreateWithCarRequest
		err := json.NewDecoder(r.Body).Decode(&createUserWithCarRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		tx, err := client.Tx(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		u, err := tx.User.Create().
			SetName(createUserWithCarRequest.Name).
			SetEmail(createUserWithCarRequest.Email).
			Save(r.Context())

		if err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = tx.Car.Create().
			SetModel(createUserWithCarRequest.Model).
			SetRegisteredAt(time.Now()).
			SetOwner(u).
			Save(r.Context())

		if err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(u)
	}
}

func listUsers(client *ent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := client.User.Query().All(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		json.NewEncoder(w).Encode(users)
	}
}

func getUser(client *ent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		user, err := client.User.Get(r.Context(), id)
		if err != nil {
			if ent.IsNotFound(err) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json; charset=UTF-8")
		json.NewEncoder(w).Encode(user)
	}
}

type CarCreateRequest struct {
	Model        string    `json:"model"`
	RegisteredAt time.Time `json:"registered_at"`
}

func createCar(client *ent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var createCarRequest CarCreateRequest
		err = json.NewDecoder(r.Body).Decode(&createCarRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		car, err := client.Car.Create().
			SetModel(createCarRequest.Model).
			SetRegisteredAt(createCarRequest.RegisteredAt).
			SetOwnerID(id).
			Save(r.Context())

		if ent.IsNotFound(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Add("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(car)
	}
}

func listCars(client *ent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		u, err := client.User.Get(r.Context(), id)
		if err != nil {
			if ent.IsNotFound(err) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cars, err := u.QueryCars().All(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cars)
	}
}
