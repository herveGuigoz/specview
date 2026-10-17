package project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolve(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "my project")
	if err := os.MkdirAll(filepath.Join(dir, "openspec"), 0755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "src", "nested")
	os.MkdirAll(sub, 0755)
	for _, input := range []string{"", dir, filepath.Join(dir, "openspec")} {
		p, err := Resolve(input, sub)
		if err != nil || p.Name != "my project" {
			t.Fatalf("%q: %+v %v", input, p, err)
		}
	}
	if _, err := Resolve(base, sub); err == nil {
		t.Fatal("accepted missing root")
	}
	entries, _ := os.ReadDir(base)
	if len(entries) != 1 {
		t.Fatal("resolver wrote files")
	}
}

func TestConfig(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "openspec")
	os.Mkdir(spec, 0755)
	os.WriteFile(filepath.Join(spec, "config.yml"), []byte("store: external\n"), 0644)
	if _, err := Resolve(dir, dir); !errors.Is(err, ErrStore) {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(spec, "config.yaml"), []byte("schema: spec-driven\n"), 0644)
	if _, err := Resolve(dir, dir); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(spec, "config.yaml"), []byte("schema: [broken"), 0644)
	p, err := Resolve(dir, dir)
	if err != nil || len(p.Diagnostics) != 1 {
		t.Fatalf("%+v %v", p, err)
	}
}
