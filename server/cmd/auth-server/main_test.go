package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadTokenSecretPersistsGeneratedValue(t *testing.T) {
	secretPath := filepath.Join(t.TempDir(), "auth.secret")

	first, generated, err := loadTokenSecret("", secretPath)
	if err != nil {
		t.Fatal(err)
	}
	if !generated {
		t.Fatal("expected a newly generated secret")
	}
	if len(first) != 32 {
		t.Fatalf("generated secret length = %d, want 32", len(first))
	}

	second, generated, err := loadTokenSecret("", secretPath)
	if err != nil {
		t.Fatal(err)
	}
	if generated {
		t.Fatal("existing secret was unexpectedly regenerated")
	}
	if !bytes.Equal(first, second) {
		t.Fatal("persisted secret changed between loads")
	}

	if runtime.GOOS != "windows" {
		if info, err := os.Stat(secretPath); err != nil {
			t.Fatal(err)
		} else if info.Mode().Perm()&0077 != 0 {
			t.Fatalf("secret permissions = %o, want no group/other access", info.Mode().Perm())
		}
	}
}

func TestLoadTokenSecretRejectsShortExplicitSecret(t *testing.T) {
	if _, _, err := loadTokenSecret("too-short", ""); err == nil {
		t.Fatal("expected a short explicit secret to be rejected")
	}
}

func TestSecurityHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	securityHeaders(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(recorder, request)

	for header, want := range map[string]string{
		"Cache-Control":          "no-store",
		"Referrer-Policy":        "no-referrer",
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
	} {
		if got := recorder.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestHealthz(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	handleHealthz(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Body.String(); got != "ok\n" {
		t.Fatalf("body = %q, want %q", got, "ok\n")
	}
}
