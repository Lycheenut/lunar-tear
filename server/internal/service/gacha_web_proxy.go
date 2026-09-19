package service

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// IsGachaWebPath recognizes the client's localized and unprefixed WebView paths.
func IsGachaWebPath(path string) bool {
	if strings.HasPrefix(path, "/web/") {
		path = strings.TrimPrefix(path, "/web")
		for _, language := range []string{"ja", "en", "ko"} {
			path = strings.TrimPrefix(path, "/"+language)
		}
	}
	return path == "/gacha-details" || path == "/gacha-rate" || path == "/gacha/box"
}

// SetGachaWebBackend routes only Gacha pages to the authoritative game process.
// The target comes from server configuration, never the client's serverAddress query.
func (s *OctoHTTPServer) SetGachaWebBackend(baseURL string) error {
	target, err := url.Parse(baseURL)
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") || target.User != nil || target.RawQuery != "" || target.Fragment != "" {
		return fmt.Errorf("invalid Gacha web backend URL")
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{ResponseHeaderTimeout: 15 * time.Second}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, _ error) {
		http.Error(w, "Gacha details are temporarily unavailable. Please reopen this page.", http.StatusBadGateway)
	}
	s.gachaWeb = proxy
	return nil
}
