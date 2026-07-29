package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveAuthTokenUsesAuthorizationHeader(t *testing.T) {
	var receivedAuthorization string
	var receivedQuery string
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthorization = r.Header.Get("Authorization")
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"42","name":"alice"}`))
	}))
	defer authServer.Close()

	service := NewUserServiceServer(nil, nil, nil, authServer.URL, true)
	id, err := service.resolveAuthToken(context.Background(), "sensitive-token")
	if err != nil {
		t.Fatal(err)
	}
	if id != 42 {
		t.Fatalf("id = %d, want 42", id)
	}
	if receivedAuthorization != "Bearer sensitive-token" {
		t.Fatalf("Authorization = %q", receivedAuthorization)
	}
	if receivedQuery != "" {
		t.Fatalf("token leaked into query string %q", receivedQuery)
	}
}
