package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHelloJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/hello", nil)
	rec := httptest.NewRecorder()
	newRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != `{"message":"Hello, World!"}` {
		t.Fatalf("body = %q", body)
	}
}
