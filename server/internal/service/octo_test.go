package service

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMasterDataURLServesCurrentDatabase(t *testing.T) {
	baseDir := t.TempDir()
	filePath := filepath.Join(baseDir, "assets", "release", "20240404193219.bin.e")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte("encrypted master data")
	if err := os.WriteFile(filePath, want, 0o644); err != nil {
		t.Fatal(err)
	}

	server := NewOctoHTTPServer("", baseDir)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/master-data/20240404193219_123", nil)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Body.Bytes(); !bytes.Equal(got, want) {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/octet-stream" {
		t.Fatalf("Content-Type = %q, want application/octet-stream", got)
	}
}
