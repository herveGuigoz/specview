package web

import (
	"io/fs"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"specview/internal/catalog"
	"specview/internal/project"
	"specview/internal/server"
)

func TestEmbeddedFrontend(t *testing.T) {
	assets := Assets()
	html, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		t.Fatal("frontend missing: run make build", err)
	}
	if strings.Contains(string(html), "fonts.googleapis.com") {
		t.Fatal("remote fonts")
	}
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "changes/c"), 0755)
	os.WriteFile(filepath.Join(dir, "changes/c/proposal.md"), []byte("# Proposal"), 0644)
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	store, err := catalog.New(root, project.Project{Name: "project", OpenSpec: dir, Path: filepath.Dir(dir)})
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Handler(store, assets)
	count := 0
	err = fs.WalkDir(assets, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		count++
		url := "http://127.0.0.1/" + name
		if name == "index.html" {
			url = "http://127.0.0.1/"
		}
		r := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 200 || w.Body.Len() == 0 {
			t.Fatalf("asset %s: %d", name, w.Code)
		}
		if strings.HasSuffix(name, ".css") && !strings.Contains(w.Header().Get("Content-Type"), "text/css") {
			t.Fatal("CSS MIME", w.Header())
		}
		if strings.HasSuffix(name, ".js") && !strings.Contains(w.Header().Get("Content-Type"), "javascript") {
			t.Fatal("JS MIME", w.Header())
		}
		return nil
	})
	if err != nil || count < 3 {
		t.Fatal("incomplete embedded frontend", count, err)
	}
}
