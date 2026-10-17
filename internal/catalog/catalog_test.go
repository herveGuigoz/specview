package catalog

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"specview/internal/project"
)

func fixture(t *testing.T) (*os.Root, project.Project, func(string, string)) {
	t.Helper()
	dir := t.TempDir()
	spec := filepath.Join(dir, "openspec")
	os.Mkdir(spec, 0755)
	root, err := os.OpenRoot(spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	write := func(name, body string) {
		t.Helper()
		full := filepath.Join(spec, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root, project.Project{Name: "fixture", Path: dir, OpenSpec: spec, Diagnostics: []string{}}, write
}
func TestInventory(t *testing.T) {
	root, p, write := fixture(t)
	write("specs/identity/user-auth/spec.md", "# Auth")
	write("changes/example/proposal.md", "# Proposal")
	write("changes/example/research/notes.md", "# Notes")
	write("changes/archive/example/proposal.md", "# Archived")
	os.MkdirAll(filepath.Join(p.OpenSpec, "changes/empty"), 0755)
	s, err := Scan(root, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Entries) != 4 || len(s.Documents) != 4 {
		t.Fatalf("%+v", s)
	}
	for _, e := range s.Entries {
		if e.ID == "specs/identity/user-auth" && e.Name != "identity/user-auth" {
			t.Fatal(e)
		}
	}
	if s.Documents["changes/example/research/notes.md"].Kind != "document" {
		t.Fatal("missing extra artifact")
	}
	original := s.Fingerprint
	write("changes/example/proposal.md", "# Modified")
	next, _ := Scan(root, p)
	if next.Fingerprint == original || s.Documents["changes/example/proposal.md"].Source != "# Proposal" {
		t.Fatal("snapshot changed or fingerprint missed edit")
	}
}
func TestTasks(t *testing.T) {
	body := "- [x] done\n- [X] done\n- [ ] todo\n\n```md\n- [ ] example\n```\n\n~~~\n- [x] example\n~~~\n\n    - [ ] indented example\n"
	if p := Tasks([]byte(body)); p != (Progress{Done: 2, Total: 3}) {
		t.Fatal(p)
	}
	if p := Tasks([]byte("# No tasks")); p.Total != 0 {
		t.Fatal(p)
	}
}
func TestDiagnosticsAndReadOnly(t *testing.T) {
	root, p, write := fixture(t)
	write("changes/c/proposal.md", "# readable")
	write("changes/c/.openspec.yaml", "schema: [broken")
	before, _ := Scan(root, p)
	if len(before.Diagnostics) != 1 {
		t.Fatal(before.Diagnostics)
	}
	paths := []string{p.OpenSpec, filepath.Join(p.OpenSpec, "changes"), filepath.Join(p.OpenSpec, "changes/c")}
	for _, d := range paths {
		os.Chmod(d, 0555)
		defer os.Chmod(d, 0755)
	}
	file := filepath.Join(p.OpenSpec, "changes/c/proposal.md")
	os.Chmod(file, 0444)
	defer os.Chmod(file, 0644)
	after, err := Scan(root, p)
	if err != nil || after.Fingerprint != before.Fingerprint {
		t.Fatalf("read-only scan: %v", err)
	}
	os.Chmod(file, 0000)
	bad, err := Scan(root, p)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && os.Getuid() != 0 && bad.Documents["changes/c/proposal.md"].Error == "" {
		t.Fatal("missing permission diagnostic")
	}
}
func TestConfinement(t *testing.T) {
	root, p, write := fixture(t)
	write("changes/a/proposal.md", "safe")
	outside := filepath.Join(t.TempDir(), "secret.md")
	os.WriteFile(outside, []byte("SECRET"), 0644)
	for _, name := range []string{"../secret.md", outside, "changes/../../secret.md", "changes\\a\\proposal.md"} {
		if b, err := ReadFile(root, name); err == nil || len(b) > 0 {
			t.Fatalf("escaped: %q %s", name, b)
		}
	}
	link := filepath.Join(p.OpenSpec, "changes/a/link.md")
	if err := os.Symlink(outside, link); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if b, err := ReadFile(root, "changes/a/link.md"); err == nil || len(b) > 0 {
		t.Fatal("read symbolic link")
	}
	os.Symlink(filepath.Dir(outside), filepath.Join(p.OpenSpec, "changes/a/external"))
	s, _ := Scan(root, p)
	for _, d := range s.Documents {
		if strings.Contains(d.Source, "SECRET") {
			t.Fatal("secret leaked")
		}
	}
	// Repeated replacement exercises the Lstat/Open race; root confinement must
	// hold even if a symlink is installed between the validation and open.
	target := filepath.Join(p.OpenSpec, "changes/a/proposal.md")
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for {
			select {
			case <-done:
				return
			default:
				os.Remove(target)
				os.Symlink(outside, target)
				os.Remove(target)
				os.WriteFile(target, []byte("safe"), 0644)
			}
		}
	}()
	for i := 0; i < 200; i++ {
		b, _ := ReadFile(root, "changes/a/proposal.md")
		if strings.Contains(string(b), "SECRET") {
			close(done)
			<-finished
			t.Fatal("race escaped")
		}
	}
	close(done)
	<-finished
}
