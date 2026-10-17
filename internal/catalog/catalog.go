package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
	"go.yaml.in/yaml/v3"
	"specview/internal/project"
)

type Progress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}
type Document struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Error  string `json:"error,omitempty"`
	Source string `json:"-"`
}
type Entry struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Category   string      `json:"category"`
	Documents  []Document  `json:"documents"`
	Progress   Progress    `json:"progress"`
	NextAction *NextAction `json:"nextAction,omitempty"`
}
type Snapshot struct {
	Revision    uint64              `json:"revision"`
	Project     project.Project     `json:"project"`
	Entries     []Entry             `json:"entries"`
	Diagnostics []string            `json:"diagnostics"`
	Stale       bool                `json:"stale"`
	Documents   map[string]Document `json:"-"`
	Fingerprint string              `json:"-"`
}

func Tasks(source []byte) Progress {
	doc := goldmark.New(goldmark.WithExtensions(extension.TaskList)).Parser().Parse(text.NewReader(source))
	p := Progress{}
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if box, ok := n.(*extast.TaskCheckBox); ok && entering {
			p.Total++
			if box.IsChecked {
				p.Done++
			}
		}
		return ast.WalkContinue, nil
	})
	return p
}

// ReadFile opens every directory relative to a pinned root. Root prevents
// concurrent symlink replacement from escaping; Lstat rejects stable symlinks.
func ReadFile(root *os.Root, name string) ([]byte, error) {
	if !fs.ValidPath(name) || strings.Contains(name, "\\") || strings.Contains(name, ":") {
		return nil, fs.ErrPermission
	}
	parts := strings.Split(name, "/")
	current := root
	var opened []*os.Root
	defer func() {
		for _, r := range opened {
			r.Close()
		}
	}()
	for _, part := range parts[:len(parts)-1] {
		st, err := current.Lstat(part)
		if err != nil {
			return nil, err
		}
		if st.Mode()&os.ModeSymlink != 0 || !st.IsDir() {
			return nil, fs.ErrPermission
		}
		next, err := current.OpenRoot(part)
		if err != nil {
			return nil, err
		}
		opened = append(opened, next)
		current = next
	}
	base := parts[len(parts)-1]
	before, err := current.Lstat(base)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, fs.ErrPermission
	}
	file, err := current.Open(base)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	after, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !after.Mode().IsRegular() || !os.SameFile(before, after) {
		return nil, fs.ErrPermission
	}
	return io.ReadAll(file)
}

// Scan never writes. Callers serialize scans and publish snapshots as immutable.
func Scan(root *os.Root, p project.Project) (*Snapshot, error) {
	if _, err := root.Stat("."); err != nil {
		return nil, err
	}
	s := &Snapshot{Project: p, Entries: []Entry{}, Diagnostics: []string{}, Documents: map[string]Document{}}
	s.Project.Diagnostics = []string{}
	if _, err := project.CheckConfig(root); err != nil {
		s.Diagnostics = append(s.Diagnostics, err.Error())
		if errors.Is(err, project.ErrStore) {
			s.Stale = true
			return seal(s), nil
		}
	}
	defaultSchema := "spec-driven"
	configUncertain := false
	if name, err := project.CheckConfig(root); err != nil {
		configUncertain = true
	} else if name != "" {
		b, err := ReadFile(root, name)
		var config changeMetadata
		if err != nil {
			configUncertain = true
		} else if err = yaml.Unmarshal(b, &config); err != nil {
			configUncertain = true
		} else if config.Schema != "" {
			defaultSchema = config.Schema
		}
	}
	metadata := map[string]changeMetadata{}
	metadataErrors := map[string]bool{}
	groups := map[string]*Entry{}
	var visit func(string) error
	visit = func(dir string) error {
		f, err := root.Open(dir)
		if err != nil {
			return err
		}
		items, err := f.ReadDir(-1)
		f.Close()
		if err != nil {
			return err
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Name() < items[j].Name() })
		for _, item := range items {
			name := path.Join(dir, item.Name())
			if item.Type()&os.ModeSymlink != 0 {
				s.Diagnostics = append(s.Diagnostics, name+": symbolic link ignored")
				continue
			}
			if item.IsDir() {
				ensureEntry(groups, name)
				if err := visit(name); err != nil {
					s.Diagnostics = append(s.Diagnostics, name+": "+err.Error())
				}
				continue
			}
			if item.Name() == ".openspec.yaml" {
				b, err := ReadFile(root, name)
				var meta changeMetadata
				if err == nil {
					err = yaml.Unmarshal(b, &meta)
				}
				owner := path.Dir(name)
				if err != nil {
					s.Diagnostics = append(s.Diagnostics, name+": "+err.Error())
					metadataErrors[owner] = true
				} else {
					metadata[owner] = meta
				}
				continue
			}
			if !strings.HasSuffix(strings.ToLower(name), ".md") {
				continue
			}
			id, category, label := groupOf(name)
			if id == "" {
				continue
			}
			e := groups[id]
			if e == nil {
				e = &Entry{ID: id, Name: label, Category: category, Documents: []Document{}}
				groups[id] = e
			}
			b, err := ReadFile(root, name)
			d := Document{Path: name, Name: strings.TrimPrefix(name, id+"/"), Kind: "document", Source: string(b)}
			switch d.Name {
			case "proposal.md":
				d.Kind = "proposal"
			case "design.md":
				d.Kind = "design"
			case "tasks.md":
				d.Kind = "tasks"
			case "spec.md":
				d.Kind = "spec"
			default:
				if strings.HasPrefix(d.Name, "specs/") {
					d.Kind = "delta"
				}
			}
			if err != nil {
				d.Error = err.Error()
				s.Diagnostics = append(s.Diagnostics, name+": "+err.Error())
			}
			if d.Kind == "tasks" && err == nil {
				e.Progress = Tasks(b)
			}
			e.Documents = append(e.Documents, d)
			s.Documents[name] = d
		}
		return nil
	}
	for _, dir := range []string{"specs", "changes"} {
		st, err := root.Lstat(dir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if st.Mode()&os.ModeSymlink != 0 {
			s.Diagnostics = append(s.Diagnostics, dir+": symbolic link ignored")
			continue
		}
		if err := visit(dir); err != nil {
			return nil, err
		}
	}
	for _, e := range groups {
		sort.Slice(e.Documents, func(i, j int) bool {
			a, b := e.Documents[i], e.Documents[j]
			rank := func(k string) int {
				switch k {
				case "proposal":
					return 0
				case "design":
					return 1
				case "tasks":
					return 2
				default:
					return 3
				}
			}
			if rank(a.Kind) != rank(b.Kind) {
				return rank(a.Kind) < rank(b.Kind)
			}
			return a.Path < b.Path
		})
		meta := metadata[e.ID]
		schema := defaultSchema
		if meta.Schema != "" {
			schema = meta.Schema
		}
		e.NextAction = suggestNext(*e, schema, meta.SkipSpecs, configUncertain || metadataErrors[e.ID])
		s.Entries = append(s.Entries, *e)
	}
	sort.Slice(s.Entries, func(i, j int) bool { return s.Entries[i].ID < s.Entries[j].ID })
	return seal(s), nil
}

func ensureEntry(groups map[string]*Entry, dir string) {
	parts := strings.Split(dir, "/")
	if (len(parts) == 2 && parts[0] == "changes" && parts[1] != "archive") || (len(parts) == 3 && parts[0] == "changes" && parts[1] == "archive") {
		_, cat, name := groupOf(dir + "/placeholder.md")
		if groups[dir] == nil {
			groups[dir] = &Entry{ID: dir, Category: cat, Name: name, Documents: []Document{}}
		}
	}
}
func groupOf(name string) (string, string, string) {
	p := strings.Split(name, "/")
	if len(p) < 2 {
		return "", "", ""
	}
	if p[0] == "specs" {
		id := path.Dir(name)
		if id == "specs" {
			return "specs", "specs", "Spécifications"
		}
		return id, "specs", strings.TrimPrefix(id, "specs/")
	}
	if p[0] == "changes" && p[1] == "archive" {
		if len(p) < 4 {
			return "", "", ""
		}
		return strings.Join(p[:3], "/"), "archive", p[2]
	}
	if p[0] == "changes" && len(p) >= 3 {
		return strings.Join(p[:2], "/"), "changes", p[1]
	}
	return "", "", ""
}
func seal(s *Snapshot) *Snapshot {
	// Include source content and metadata, not just timestamps or file sizes.
	b, _ := json.Marshal(s)
	h := sha256.New()
	h.Write(b)
	keys := make([]string, 0, len(s.Documents))
	for k := range s.Documents {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(h, "%d:%s%d:%s", len(k), k, len(s.Documents[k].Source), s.Documents[k].Source)
	}
	s.Fingerprint = hex.EncodeToString(h.Sum(nil))
	return s
}
