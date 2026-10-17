package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"specview/internal/catalog"
)

func Handler(store *catalog.Store, assets fs.FS) http.Handler {
	files := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if host != "127.0.0.1" && host != "localhost" {
			http.Error(w, "local requests only", http.StatusForbidden)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "read-only", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		switch r.URL.Path {
		case "/api/catalog":
			writeJSON(w, r, store.Snapshot())
		case "/api/document":
			s := store.Snapshot()
			rev, err := strconv.ParseUint(r.URL.Query().Get("revision"), 10, 64)
			if err != nil {
				http.Error(w, "revision is required", 400)
				return
			}
			if rev != s.Revision {
				http.Error(w, "revision changed; reload catalog", 409)
				return
			}
			name := r.URL.Query().Get("path")
			if !fs.ValidPath(name) || strings.ContainsAny(name, "\\:") {
				http.Error(w, "invalid document path", 403)
				return
			}
			doc, ok := s.Documents[name]
			if !ok {
				http.NotFound(w, r)
				return
			}
			if doc.Error != "" {
				http.Error(w, doc.Error, 422)
				return
			}
			writeJSON(w, r, map[string]any{"revision": s.Revision, "path": doc.Path, "source": doc.Source})
		case "/api/events":
			if r.Method == http.MethodHead {
				w.Header().Set("Content-Type", "text/event-stream")
				return
			}
			events(w, r, store)
		default:
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			requested := strings.TrimPrefix(r.URL.Path, "/")
			if requested == "" {
				requested = "index.html"
			}
			if !fs.ValidPath(requested) || path.Clean(requested) != requested {
				http.NotFound(w, r)
				return
			}
			info, err := fs.Stat(assets, requested)
			if err != nil || info.IsDir() {
				http.NotFound(w, r)
				return
			}
			files.ServeHTTP(w, r)
		}
	})
}
func writeJSON(w http.ResponseWriter, r *http.Request, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodHead {
		json.NewEncoder(w).Encode(value)
	}
}
func events(w http.ResponseWriter, r *http.Request, store *catalog.Store) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	revisions, unsubscribe := store.Subscribe()
	defer unsubscribe()
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	for {
		controller := http.NewResponseController(w)
		controller.SetWriteDeadline(time.Now().Add(15 * time.Second))
		select {
		case <-r.Context().Done():
			return
		case rev := <-revisions:
			if _, err := fmt.Fprintf(w, "data: %d\n\n", rev); err != nil {
				return
			}
			flusher.Flush()
		case <-tick.C:
			if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
