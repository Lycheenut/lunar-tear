package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminContentSecurityPolicyAllowsBannerPreviews(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	response := httptest.NewRecorder()

	serveAdminAsset(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	policy := response.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "img-src 'self' https://assets.lycheenut.cc") {
		t.Fatalf("Content-Security-Policy does not allow banner previews: %q", policy)
	}
}
