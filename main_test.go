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

func TestGroup(t *testing.T) {
	ctx := t.Context()

	client, err := ent.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("fail to create sqlite client %v \n", err)
	}
	if err := client.Schema.Create(t.Context()); err != nil {
		t.Fatalf("fail to create schema %v \n", err)
	}

	u, err := client.User.Create().SetEmail("jibum.jung@gmail.com").SetName("manty").Save(ctx)
	if u == nil || err != nil {
		t.Fatalf("%v %v", u, err)
	}

	g, err := client.Group.Create().SetName("GitHub").AddUsers(u).Save(ctx)
	if g == nil || err != nil {
		t.Fatalf("%v %v", g, err)
	}

	users, err := g.QueryUsers().All(ctx)
	if len(users) != 1 || err != nil {
		t.Fatalf("%v %v", users, err)
	}

	groups, err := u.QueryGroups().All(ctx)
	if groups == nil || err != nil {
		t.Fatalf("%v %v", groups, err)
	}

}

func TestRollback(t *testing.T) {
	body := strings.NewReader(`{"name":"tx-fail","email":"tx-fail@example.com","model":""}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/users-with-car", body)
	req.Header.Set("content-type", "application/json")
	rec := httptest.NewRecorder()

	client, err := ent.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("fail to create sqlite client %v \n", err)
	}
	if err := client.Schema.Create(t.Context()); err != nil {
		t.Fatalf("fail to create schema %v \n", err)
	}

	newRouter(client).ServeHTTP(rec, req)

	if rec.Code == http.StatusOK || rec.Code == http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	newRouter(client).ServeHTTP(listRec, listReq)
	if strings.Contains(listRec.Body.String(), "tx-fail@example.com") {
		t.Fatalf("user should not exist: %s", listRec.Body.String())
	}
}
