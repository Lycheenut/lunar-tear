package service

import (
	"strings"
	"testing"
)

func TestNormalizeResourcesBaseURL(t *testing.T) {
	got, err := NormalizeResourcesBaseURL("https://assets.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != ResourcesBaseURLLength {
		t.Fatalf("normalized length = %d, want %d", len(got), ResourcesBaseURLLength)
	}
	if !strings.HasPrefix(got, "https://assets.example.com/r") {
		t.Fatalf("normalized URL = %q", got)
	}
}

func TestNormalizeResourcesBaseURLRejectsUnsafeValues(t *testing.T) {
	for _, raw := range []string{
		"ftp://assets.example.com",
		"https://assets.example.com?token=secret",
		"https://" + strings.Repeat("a", ResourcesBaseURLLength) + ".example",
	} {
		if _, err := NormalizeResourcesBaseURL(raw); err == nil {
			t.Errorf("NormalizeResourcesBaseURL(%q) unexpectedly succeeded", raw)
		}
	}
}
