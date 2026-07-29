package service

import (
	"fmt"
	"net/url"
	"strings"
)

const ResourcesBaseURLLength = len(resourcesURLOriginal)

// NormalizeResourcesBaseURL validates a public HTTP(S) resource base URL and
// pads it with a path segment so it can replace the fixed-width URL in list.bin.
func NormalizeResourcesBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse resources base URL: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("resources base URL must use http or https and include a host")
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("resources base URL must not include user info, a query, or a fragment")
	}

	base := strings.TrimRight(u.String(), "/")
	if len(base) > ResourcesBaseURLLength {
		return "", fmt.Errorf("resources base URL is %d bytes; maximum is %d", len(base), ResourcesBaseURLLength)
	}
	if len(base) == ResourcesBaseURLLength {
		return base, nil
	}
	padding := ResourcesBaseURLLength - len(base) - 1
	if padding < 1 {
		return "", fmt.Errorf("resources base URL must leave room for a padding path segment")
	}
	return base + "/" + strings.Repeat("r", padding), nil
}
