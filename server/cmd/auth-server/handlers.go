package main

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"lunar-tear/server/internal/auth"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const maxOAuthFormBytes = 16 << 10

//go:embed login.html
var loginFS embed.FS

var loginTmpl = template.Must(template.ParseFS(loginFS, "login.html"))

// oauthRedirectTmpl drives the fbconnect:// hand-off via a renderer-initiated
// navigation instead of a server-side 302. Android WebView does NOT invoke
// WebViewClient.shouldOverrideUrlLoading for 302 redirects from POST form
// submissions to non-http schemes (documented Chromium WebView limitation,
// Stack Overflow #6738328 / Google issuetracker #36918490). Returning a 200
// HTML page with both <meta http-equiv="refresh"> and window.location.replace()
// makes the cross-scheme navigation renderer-initiated, which DOES invoke
// shouldOverrideUrlLoading, so the FB SDK can extract access_token from the
// URL fragment and complete its login flow. html/template auto-escapes {{.}}
// correctly for the meta URL-attribute context and the JS string-literal
// context inside <script>.
var oauthRedirectTmpl = template.Must(template.New("oauthRedirect").Parse(
	`<!doctype html><html><head><meta charset="utf-8">
<meta http-equiv="refresh" content="0;url={{.}}">
<script>window.location.replace({{.}});</script>
</head><body>
<noscript><a href="{{.}}">Continue</a></noscript>
</body></html>
`))

type Handlers struct {
	store             *auth.AuthStore
	tok               *auth.TokenService
	noRegister        bool
	allowedRedirects  map[string]struct{}
	restrictRedirects bool
}

func NewHandlers(store *auth.AuthStore, tok *auth.TokenService, noRegister bool, allowedRedirectURIs []string) *Handlers {
	allowedRedirects := make(map[string]struct{}, len(allowedRedirectURIs))
	for _, raw := range allowedRedirectURIs {
		canonical, ok := canonicalRedirectURI(raw)
		if !ok {
			log.Printf("ignoring invalid configured OAuth redirect URI %q", raw)
			continue
		}
		allowedRedirects[canonical] = struct{}{}
	}
	return &Handlers{
		store:             store,
		tok:               tok,
		noRegister:        noRegister,
		allowedRedirects:  allowedRedirects,
		restrictRedirects: len(allowedRedirectURIs) > 0,
	}
}

type loginPageData struct {
	RedirectURI string
	State       string
	Scope       string
	Error       string
	Username    string
}

func isOAuthPath(path string) bool {
	// Match /v{N}/dialog/oauth or /v{N}.{M}/dialog/oauth
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) != 3 {
		return false
	}
	return strings.HasPrefix(parts[0], "v") && parts[1] == "dialog" && parts[2] == "oauth"
}

func isMePath(path string) bool {
	p := strings.TrimPrefix(path, "/")
	if p == "me" {
		return true
	}
	parts := strings.Split(p, "/")
	return len(parts) == 2 && strings.HasPrefix(parts[0], "v") && parts[1] == "me"
}

func (h *Handlers) HandleOAuth(w http.ResponseWriter, r *http.Request) {
	if isMePath(r.URL.Path) {
		h.HandleMe(w, r)
		return
	}

	if !isOAuthPath(r.URL.Path) {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.oauthGet(w, r)
	case http.MethodPost:
		h.oauthPost(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) oauthGet(w http.ResponseWriter, r *http.Request) {
	redirectURI := r.URL.Query().Get("redirect_uri")
	if !h.isAllowedRedirectURI(redirectURI) {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	data := loginPageData{
		RedirectURI: redirectURI,
		State:       r.URL.Query().Get("state"),
		Scope:       r.URL.Query().Get("scope"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := loginTmpl.Execute(w, data); err != nil {
		log.Printf("render login page: %v", err)
	}
}

func (h *Handlers) oauthPost(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxOAuthFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	action := r.FormValue("action")
	redirectURI := r.FormValue("redirect_uri")
	state := r.FormValue("state")
	scope := r.FormValue("scope")
	if !h.isAllowedRedirectURI(redirectURI) {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}

	renderErr := func(msg string) {
		data := loginPageData{
			RedirectURI: redirectURI,
			State:       state,
			Scope:       scope,
			Error:       msg,
			Username:    username,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if err := loginTmpl.Execute(w, data); err != nil {
			log.Printf("render login page: %v", err)
		}
	}

	if username == "" || password == "" {
		renderErr("Username and password are required.")
		return
	}

	var user auth.AuthUser
	var err error

	switch action {
	case "register":
		if h.noRegister {
			renderErr("This server does not accept user registrations.")
			return
		}

		user, err = h.store.CreateUser(username, password)
		if err == auth.ErrUserExists {
			renderErr("Username is already taken.")
			return
		}
		if err != nil {
			log.Printf("create user: %v", err)
			renderErr("Server error. Try again.")
			return
		}
		log.Printf("registered user %q (id=%d)", user.Username, user.ID)

	case "login":
		user, err = h.store.VerifyUser(username, password)
		if err == auth.ErrInvalidCreds {
			renderErr("Invalid username or password.")
			return
		}
		if err != nil {
			log.Printf("verify user: %v", err)
			renderErr("Server error. Try again.")
			return
		}
		log.Printf("authenticated user %q (id=%d)", user.Username, user.ID)

	default:
		renderErr("Invalid action.")
		return
	}

	token, err := h.tok.Generate(user)
	if err != nil {
		log.Printf("generate token: %v", err)
		renderErr("Server error. Try again.")
		return
	}

	payload := fmt.Sprintf(`{"user_id":"%d"}`, user.ID)
	b64 := base64.RawURLEncoding.EncodeToString([]byte(payload))

	fragment := url.Values{}
	fragment.Set("access_token", token)
	fragment.Set("token_type", "bearer")
	fragment.Set("expires_in", strconv.FormatInt(int64(auth.TokenTTL.Seconds()), 10))
	fragment.Set("signed_request", "0."+b64)
	// iOS FBSDKLoginManager treats an empty granted_scopes set as a cancelled login
	// (LoginManager.swift -> getSuccessResult -> getCancelledResult). Echo back the
	// scope the SDK sent so parameters.permissions is non-empty and the SDK fires
	// its success path. Android tolerates either way.
	if scope != "" {
		fragment.Set("granted_scopes", scope)
		fragment.Set("denied_scopes", "")
	}
	if state != "" {
		fragment.Set("state", state)
	}

	target, err := buildOAuthRedirectTarget(redirectURI, fragment)
	if err != nil {
		log.Printf("build OAuth redirect: %v", err)
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	callback, _ := url.Parse(redirectURI)
	log.Printf("redirecting to OAuth callback scheme=%s host=%s", callback.Scheme, callback.Host)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := oauthRedirectTmpl.Execute(w, target); err != nil {
		log.Printf("render oauth redirect: %v", err)
	}
}

func (h *Handlers) HandleCheckUsername(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	w.Header().Set("Content-Type", "application/json")
	if username == "" {
		json.NewEncoder(w).Encode(map[string]bool{"exists": false})
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"exists": h.store.UserExists(username)})
}

func (h *Handlers) HandleMe(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("access_token")
	if token == "" {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = auth[7:]
		}
	}

	if token == "" {
		http.Error(w, `{"error":{"message":"missing access_token","type":"OAuthException","code":190}}`, http.StatusUnauthorized)
		return
	}

	claims, err := h.tok.Validate(token)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":{"message":"%s","type":"OAuthException","code":190}}`, err), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":   strconv.FormatInt(claims.Sub, 10),
		"name": claims.Name,
	})
}

func (h *Handlers) isAllowedRedirectURI(raw string) bool {
	canonical, ok := canonicalRedirectURI(raw)
	if !ok {
		return false
	}
	if h.restrictRedirects {
		_, ok := h.allowedRedirects[canonical]
		return ok
	}
	return true
}

func canonicalRedirectURI(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", false
	}
	if u.User != nil || u.Opaque != "" || u.RawQuery != "" || u.Fragment != "" || u.Port() != "" {
		return "", false
	}

	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	allowed := false
	switch {
	case scheme == "fbconnect":
		allowed = host == "success" ||
			(strings.HasPrefix(host, "cct.") && len(strings.TrimPrefix(host, "cct.")) > 0)
	case strings.HasPrefix(scheme, "fb") && allASCIIDigits(strings.TrimPrefix(scheme, "fb")):
		allowed = host == "authorize"
	}
	if !allowed {
		return "", false
	}
	u.Scheme = scheme
	u.Host = host
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String(), true
}

func allASCIIDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func buildOAuthRedirectTarget(redirectURI string, values url.Values) (string, error) {
	if _, ok := canonicalRedirectURI(redirectURI); !ok {
		return "", fmt.Errorf("redirect URI is not allowed")
	}
	u, err := url.Parse(redirectURI)
	if err != nil {
		return "", err
	}
	u.RawQuery = values.Encode()
	return u.String(), nil
}
