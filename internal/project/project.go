// Package project resolves one local OpenSpec project without writing to it.
package project

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

type Project struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	OpenSpec    string   `json:"openspec"`
	Diagnostics []string `json:"diagnostics"`
}

// Resolve searches ancestors only when input is empty. cwd is explicit for tests.
func Resolve(input, cwd string) (Project, error) {
	target := input
	if target == "" {
		target = cwd
	} else if !filepath.IsAbs(target) {
		target = filepath.Join(cwd, target)
	}
	target, err := filepath.Abs(target)
	if err != nil {
		return Project{}, err
	}
	if input != "" && filepath.Base(target) == "openspec" {
		target = filepath.Dir(target)
	}
	for {
		spec := filepath.Join(target, "openspec")
		st, statErr := os.Stat(spec)
		if statErr == nil && st.IsDir() {
			canonical, err := filepath.EvalSymlinks(spec)
			if err != nil {
				return Project{}, err
			}
			r, err := os.OpenRoot(canonical)
			if err != nil {
				return Project{}, err
			}
			f, err := r.Open(".")
			if err == nil {
				_, err = f.ReadDir(1)
				f.Close()
				if errors.Is(err, fs.ErrNotExist) {
					err = nil
				}
			}
			r.Close()
			if err != nil && !errors.Is(err, io.EOF) {
				return Project{}, fmt.Errorf("cannot read OpenSpec root: %w", err)
			}
			p := Project{Name: filepath.Base(target), Path: filepath.Dir(canonical), OpenSpec: canonical, Diagnostics: []string{}}
			root, err := os.OpenRoot(canonical)
			if err != nil {
				return Project{}, err
			}
			_, configErr := CheckConfig(root)
			root.Close()
			if errors.Is(configErr, ErrStore) {
				return Project{}, configErr
			}
			if configErr != nil {
				p.Diagnostics = append(p.Diagnostics, configErr.Error())
			}
			return p, nil
		}
		if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
			return Project{}, statErr
		}
		parent := filepath.Dir(target)
		if input != "" || parent == target {
			return Project{}, fmt.Errorf("no local openspec/ directory found at %s", target)
		}
		target = parent
	}
}

var ErrStore = errors.New("external OpenSpec stores are not supported; select a standalone local project")

// CheckConfig prefers config.yaml, even when malformed. It never falls back on a read error.
func CheckConfig(root *os.Root) (string, error) {
	for _, name := range []string{"config.yaml", "config.yml"} {
		b, err := root.ReadFile(name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return name, fmt.Errorf("%s: %w", name, err)
		}
		var value map[string]any
		if err := yaml.Unmarshal(b, &value); err != nil {
			return name, fmt.Errorf("%s: invalid YAML: %w", name, err)
		}
		if store, ok := value["store"]; ok && store != nil && strings.TrimSpace(fmt.Sprint(store)) != "" {
			return name, ErrStore
		}
		return name, nil
	}
	return "", nil
}
