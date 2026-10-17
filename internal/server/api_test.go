package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"specview/internal/catalog"
	"specview/internal/project"
)

func apiFixture(t *testing.T) (*catalog.Store, http.Handler, func(string, string), string) {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "changes/demo"), 0755)
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	write := func(name, body string) {
		t.Helper()
		full := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(full), 0755)
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("changes/demo/tasks.md", "- [ ] Task")
	s, err := catalog.New(root, project.Project{Name: "demo", Path: dir, OpenSpec: dir})
	if err != nil {
		t.Fatal(err)
	}
	return s, Handler(s, fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("frontend")}}), write, dir
}
func req(h http.Handler, method, url string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://127.0.0.1:4321"+url, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestAPI(t *testing.T) {
	store, h, write, dir := apiFixture(t)
	if w := req(h, "GET", "/api/catalog"); w.Code != 200 || !strings.Contains(w.Body.String(), `"total":1`) {
		t.Fatal(w)
	}
	document := fmt.Sprintf("/api/document?path=changes/demo/tasks.md&revision=%d", store.Snapshot().Revision)
	if w := req(h, "GET", document); w.Code != 200 || !strings.Contains(w.Body.String(), "Task") {
		t.Fatal(w)
	}
	write("changes/demo/tasks.md", "- [x] Task")
	store.Refresh()
	if w := req(h, "GET", document); w.Code != 409 {
		t.Fatal(w)
	}
	document = fmt.Sprintf("/api/document?path=changes/demo/tasks.md&revision=%d", store.Snapshot().Revision)
	if w := req(h, "GET", document); w.Code != 200 || !strings.Contains(w.Body.String(), "[x]") {
		t.Fatal(w)
	}
	if w := req(h, "GET", "/api/catalog"); !strings.Contains(w.Body.String(), `"done":1`) {
		t.Fatal(w)
	}
	os.Remove(filepath.Join(dir, "changes/demo/tasks.md"))
	store.Refresh()
	document = fmt.Sprintf("/api/document?path=changes/demo/tasks.md&revision=%d", store.Snapshot().Revision)
	if w := req(h, "GET", document); w.Code != 404 {
		t.Fatal(w)
	}
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		if w := req(h, method, "/api/catalog"); w.Code != 405 {
			t.Fatal(w)
		}
	}
	for _, url := range []string{"/api/document?path=../.env&revision=3", "/api/document?path=/etc/passwd&revision=3"} {
		if w := req(h, "GET", url); w.Code != 403 {
			t.Fatal(w)
		}
	}
	for _, url := range []string{"/.env", "/changes/demo/tasks.md", "/api/missing", "/assets/"} {
		if w := req(h, "GET", url); w.Code != 404 {
			t.Fatalf("%s %d", url, w.Code)
		}
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://evil.example/api/catalog", nil)
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w)
	}
	if w := req(h, "HEAD", "/api/catalog"); w.Body.Len() != 0 || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal(w)
	}
	if w := req(h, "GET", "/"); w.Code != 200 || w.Body.String() != "frontend" {
		t.Fatal(w)
	}
}

// Exercise streaming through a pipe without requiring a listening socket.
type streamWriter struct {
	header  http.Header
	writer  *io.PipeWriter
	once    sync.Once
	started chan struct{}
}

func (w *streamWriter) Header() http.Header { return w.header }
func (w *streamWriter) WriteHeader(int)     {}
func (w *streamWriter) Write(b []byte) (int, error) {
	w.once.Do(func() { close(w.started) })
	return w.writer.Write(b)
}
func (w *streamWriter) Flush() {}
func TestSSE(t *testing.T) {
	store, h, write, _ := apiFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader, writer := io.Pipe()
	defer reader.Close()
	w := &streamWriter{header: http.Header{}, writer: writer, started: make(chan struct{})}
	r := httptest.NewRequest("GET", "http://127.0.0.1/api/events", nil).WithContext(ctx)
	done := make(chan struct{})
	go func() { h.ServeHTTP(w, r); writer.Close(); close(done) }()
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() || scanner.Text() != "data: 1" {
		t.Fatal(scanner.Text())
	}
	scanner.Scan()
	write("changes/demo/tasks.md", "- [x] Task")
	store.Refresh()
	if !scanner.Scan() || scanner.Text() != "data: 2" {
		t.Fatal(scanner.Text())
	}
	cancel()
	reader.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SSE did not close")
	}
	revisions, unsub := store.Subscribe()
	if rev := <-revisions; rev != 2 {
		t.Fatal(rev)
	}
	for i := 0; i < 5; i++ {
		write("changes/demo/tasks.md", fmt.Sprintf("- [x] Task %d", i))
		store.Refresh()
	}
	if rev := <-revisions; rev != store.Snapshot().Revision {
		t.Fatal("slow subscriber missed latest", rev)
	}
	unsub()
}
func TestWatch(t *testing.T) {
	store, _, write, dir := apiFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go store.Watch(ctx, 10*time.Millisecond)
	wait := func(old uint64) {
		t.Helper()
		limit := time.Now().Add(time.Second)
		for store.Snapshot().Revision <= old {
			if time.Now().After(limit) {
				t.Fatal("no refresh")
			}
			time.Sleep(time.Millisecond)
		}
	}
	old := store.Snapshot().Revision
	write("changes/demo/tasks.md", "- [x] Task")
	wait(old)
	old = store.Snapshot().Revision
	write("changes/demo/temp.md", "atomic")
	os.Rename(filepath.Join(dir, "changes/demo/temp.md"), filepath.Join(dir, "changes/demo/tasks.md"))
	wait(old)
	old = store.Snapshot().Revision
	write("changes/demo/specs/identity/user-auth/spec.md", "# New")
	wait(old)
	old = store.Snapshot().Revision
	os.MkdirAll(filepath.Join(dir, "changes/archive"), 0755)
	os.Rename(filepath.Join(dir, "changes/demo"), filepath.Join(dir, "changes/archive/demo"))
	wait(old)
	if _, ok := store.Snapshot().Documents["changes/archive/demo/tasks.md"]; !ok {
		t.Fatal("archive not updated")
	}
	old = store.Snapshot().Revision
	os.Remove(filepath.Join(dir, "changes/archive/demo/tasks.md"))
	wait(old)
	b, err := json.Marshal(store.Snapshot())
	if err != nil || strings.Contains(string(b), "atomic") {
		t.Fatal("content leaked into catalog", err)
	}
}
