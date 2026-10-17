package catalog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecovery(t *testing.T) {
	root, p, write := fixture(t)
	write("changes/c/tasks.md", "- [ ] task")
	store, err := New(root, p)
	if err != nil {
		t.Fatal(err)
	}
	first := store.Snapshot().Revision
	// A same-size change with the original mtime still invalidates the snapshot.
	file := filepath.Join(p.OpenSpec, "changes/c/tasks.md")
	info, _ := os.Stat(file)
	write("changes/c/tasks.md", "- [x] task")
	os.Chtimes(file, info.ModTime(), info.ModTime())
	store.Refresh()
	if store.Snapshot().Revision != first+1 {
		t.Fatal("missed same-size edit")
	}
	rev := store.Snapshot().Revision
	store.Refresh()
	if store.Snapshot().Revision != rev {
		t.Fatal("published unchanged state")
	}
	os.Rename(filepath.Join(p.OpenSpec, "changes"), filepath.Join(p.OpenSpec, "saved"))
	os.WriteFile(filepath.Join(p.OpenSpec, "changes"), []byte("not a directory"), 0644)
	if err := store.Refresh(); err == nil || !store.Snapshot().Stale {
		t.Fatal("global error not surfaced", err)
	}
	rev = store.Snapshot().Revision
	store.Refresh()
	if store.Snapshot().Revision != rev {
		t.Fatal("repeated diagnostic accumulated")
	}
	os.Remove(filepath.Join(p.OpenSpec, "changes"))
	os.Rename(filepath.Join(p.OpenSpec, "saved"), filepath.Join(p.OpenSpec, "changes"))
	store.Refresh()
	if store.Snapshot().Stale || len(store.Snapshot().Diagnostics) != 0 {
		t.Fatal("did not recover")
	}
	write("changes/c/.openspec.yaml", "schema: [broken")
	store.Refresh()
	if len(store.Snapshot().Diagnostics) != 1 {
		t.Fatal("no local error")
	}
	write("changes/c/.openspec.yaml", "schema: spec-driven\n")
	store.Refresh()
	if len(store.Snapshot().Diagnostics) != 0 {
		t.Fatal("local error persists")
	}
}

func TestThousandDocumentConvergence(t *testing.T) {
	root, p, write := fixture(t)
	for i := 0; i < 1000; i++ {
		write(fmt.Sprintf("specs/cap-%04d/spec.md", i), "# Specification\n\n## Requirements\n\nA local document.\n")
	}
	write("changes/c/tasks.md", "- [ ] task")
	store, err := New(root, p)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() { defer close(stopped); store.Watch(ctx, time.Second) }()
	defer func() { cancel(); <-stopped }()
	revision := store.Snapshot().Revision
	start := time.Now()
	write("changes/c/tasks.md", "- [x] task")
	for store.Snapshot().Revision == revision {
		if time.Since(start) > 3*time.Second {
			t.Fatal("convergence exceeded 3s")
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Logf("1,001 documents: update converged in %s", time.Since(start))
	if store.Snapshot().Documents["changes/c/tasks.md"].Source != "- [x] task" {
		t.Fatal("old content")
	}
}
