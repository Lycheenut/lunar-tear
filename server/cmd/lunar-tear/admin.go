package main

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"lunar-tear/server/internal/masterdataadmin"
	"lunar-tear/server/internal/runtime"
)

//go:embed admin.html admin.css admin.js
var adminAssets embed.FS

// startAdmin serves the token-gated master-data API and its static management
// UI. The listener stays disabled when LUNAR_ADMIN_TOKEN is empty.
func startAdmin(listen, binPath string, holder *runtime.Holder) {
	token := os.Getenv("LUNAR_ADMIN_TOKEN")
	if token == "" {
		log.Println("[admin] disabled (no LUNAR_ADMIN_TOKEN set)")
		return
	}
	expected := []byte("Bearer " + token)
	authorized := func(r *http.Request) bool {
		got := []byte(r.Header.Get("Authorization"))
		return len(got) == len(expected) && subtle.ConstantTimeCompare(got, expected) == 1
	}

	var updateMu sync.Mutex
	mux := http.NewServeMux()
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/admin/", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/admin/", serveAdminAsset)
	mux.HandleFunc("/api/admin/master-data/schedules", func(w http.ResponseWriter, r *http.Request) {
		if !authorized(r) {
			writeAdminError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		switch r.Method {
		case http.MethodGet:
			catalog, err := masterdataadmin.Load(binPath)
			if err != nil {
				log.Printf("[admin] read schedules failed: %v", err)
				writeAdminError(w, http.StatusInternalServerError, "读取主数据失败")
				return
			}
			writeAdminJSON(w, http.StatusOK, catalog)
		case http.MethodPost:
			updateMu.Lock()
			defer updateMu.Unlock()

			var request masterdataadmin.UpdateRequest
			body := http.MaxBytesReader(w, r.Body, 2<<20)
			decoder := json.NewDecoder(body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				writeAdminError(w, http.StatusBadRequest, "请求格式无效: "+err.Error())
				return
			}
			candidate, result, err := masterdataadmin.BuildUpdate(binPath, request)
			if err != nil {
				if errors.Is(err, masterdataadmin.ErrVersionConflict) {
					writeAdminError(w, http.StatusConflict, "主数据已被其他操作更新，请刷新后重试")
					return
				}
				writeAdminError(w, http.StatusBadRequest, err.Error())
				return
			}
			temporary, err := writeCandidate(binPath, candidate)
			if err != nil {
				log.Printf("[admin] write candidate failed: %v", err)
				writeAdminError(w, http.StatusInternalServerError, "写入候选主数据失败")
				return
			}
			defer os.Remove(temporary)
			if err := holder.InstallAndReload(temporary); err != nil {
				log.Printf("[admin] install candidate failed: %v", err)
				writeAdminError(w, http.StatusInternalServerError, "候选主数据验证或热更新失败: "+err.Error())
				return
			}
			log.Printf("[admin] rebuilt and installed master data from %s: %d cells across %d rows", r.RemoteAddr, result.ChangedCells, result.ChangedRows)
			writeAdminJSON(w, http.StatusOK, result)
		default:
			w.Header().Set("Allow", "GET, POST")
			writeAdminError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
	mux.HandleFunc("/api/admin/master-data/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeAdminError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if !authorized(r) {
			writeAdminError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if err := holder.Reload(); err != nil {
			log.Printf("[admin] master-data reload failed: %v", err)
			writeAdminError(w, http.StatusInternalServerError, "master-data reload failed")
			return
		}
		log.Printf("[admin] master data reloaded by %s", r.RemoteAddr)
		writeAdminJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})

	server := &http.Server{
		Addr:              listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("[admin] management UI on http://%s/admin/ (token-gated API)", listen)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("[admin] listener failed: %v", err)
		}
	}()
}

func serveAdminAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var name, contentType string
	switch r.URL.Path {
	case "/admin/":
		name, contentType = "admin.html", "text/html; charset=utf-8"
	case "/admin/admin.css":
		name, contentType = "admin.css", "text/css; charset=utf-8"
	case "/admin/admin.js":
		name, contentType = "admin.js", "text/javascript; charset=utf-8"
	default:
		http.NotFound(w, r)
		return
	}
	content, err := adminAssets.ReadFile(name)
	if err != nil {
		http.Error(w, "asset unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; script-src 'self'; connect-src 'self'; img-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(content)
}

func writeCandidate(binPath string, data []byte) (path string, err error) {
	directory := filepath.Dir(binPath)
	file, err := os.CreateTemp(directory, ".master-data-admin-*.bin.e")
	if err != nil {
		return "", err
	}
	path = file.Name()
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(path)
		}
	}()
	if _, err = file.Write(data); err != nil {
		return "", err
	}
	if err = file.Sync(); err != nil {
		return "", err
	}
	return path, nil
}

func writeAdminJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("[admin] encode response failed: %v", err)
	}
}

func writeAdminError(w http.ResponseWriter, status int, message string) {
	writeAdminJSON(w, status, map[string]string{"error": fmt.Sprint(message)})
}
