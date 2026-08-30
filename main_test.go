package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-chi-ent-example/ent"

	_ "github.com/mattn/go-sqlite3"
)

func TestHelloJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/hello", nil)
	rec := httptest.NewRecorder()
	newRouter(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != `{"message":"Hello, World!"}` {
		t.Fatalf("body = %q", body)
	}
}

func TestGetUser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/users/99", nil)
	rec := httptest.NewRecorder()

	client, err := ent.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("fail to create sqlite client %v \n", err)
	}
	if err := client.Schema.Create(t.Context()); err != nil {
		t.Fatalf("fail to create schema %v \n", err)
	}

	newRouter(client).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}

}
