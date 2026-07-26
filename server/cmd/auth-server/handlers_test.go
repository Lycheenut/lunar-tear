package main

import (
	"bytes"
	"database/sql"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"lunar-tear/server/internal/auth"
)

func TestAllowedRedirectURI(t *testing.T) {
	tests := []struct {
		uri  string
		want bool
	}{
		{"fbconnect://success", true},
		{"fbconnect://cct.com.example.game", true},
		{"fb123456789://authorize/", true},
		{"https://attacker.example/callback", false},
		{"fbconnect://cct.example?next=https://attacker.example", false},
		{"fbconnect://", false},
		{"fbnotdigits://authorize", false},
	}
	for _, test := range tests {
		handler := &Handlers{}
		if got := handler.isAllowedRedirectURI(test.uri); got != test.want {
			t.Errorf("isAllowedRedirectURI(%q) = %v, want %v", test.uri, got, test.want)
		}
	}
}

func TestConfiguredRedirectAllowlistIsExact(t *testing.T) {
	handler := &Handlers{allowedRedirects: map[string]struct{}{
		"fbconnect://cct.com.example.game": {},
	}, restrictRedirects: true}
	if !handler.isAllowedRedirectURI("fbconnect://cct.com.example.game") {
		t.Fatal("configured redirect was rejected")
	}
	if handler.isAllowedRedirectURI("fbconnect://cct.attacker.game") {
		t.Fatal("unconfigured redirect was accepted")
	}
}

func TestOAuthRejectsUntrustedRedirect(t *testing.T) {
	handler := newTestHandlers(t)
	request := httptest.NewRequest(
		http.MethodGet,
		"/v14.0/dialog/oauth?redirect_uri=https://attacker.example/callback",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.HandleOAuth(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestOAuthLoginDoesNotLogAccessToken(t *testing.T) {
	handler := newTestHandlers(t)
	if _, err := handler.store.CreateUser("alice", "correct horse battery staple"); err != nil {
		t.Fatal(err)
	}

	form := url.Values{
		"username":     {"alice"},
		"password":     {"correct horse battery staple"},
		"action":       {"login"},
		"redirect_uri": {"fbconnect://success"},
		"state":        {"test-state"},
	}
	request := httptest.NewRequest(http.MethodPost, "/v14.0/dialog/oauth", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	var logs bytes.Buffer
	previousWriter := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previousWriter) })

	handler.HandleOAuth(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), "access_token") {
		t.Fatal("OAuth response does not contain an access token")
	}
	if strings.Contains(logs.String(), "access_token") {
		t.Fatalf("access token leaked to logs: %s", logs.String())
	}
}

func newTestHandlers(t *testing.T) *Handlers {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store, err := auth.NewAuthStore(db)
	if err != nil {
		t.Fatal(err)
	}
	return NewHandlers(store, auth.NewTokenService(bytes.Repeat([]byte{1}, 32)), false, nil)
}
