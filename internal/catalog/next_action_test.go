package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNextActionFromFiles(t *testing.T) {
	tests := []struct {
		name, config, metadata string
		files                  map[string]string
		want, target           string
	}{
		{name: "empty", want: "instructions proposal", target: "terminal"},
		{name: "proposal", files: map[string]string{"proposal.md": "# Proposal"}, want: "instructions specs", target: "terminal"},
		{name: "specs", files: map[string]string{"proposal.md": "# Proposal", "specs/a/spec.md": "# Spec"}, want: "instructions tasks", target: "terminal"},
		{name: "apply", files: map[string]string{"proposal.md": "# Proposal", "specs/a/spec.md": "# Spec", "tasks.md": "- [ ] Work"}, want: "/openspec-apply-change demo", target: "agent"},
		{name: "archive", files: map[string]string{"proposal.md": "# Proposal", "specs/a/spec.md": "# Spec", "tasks.md": "- [x] Work"}, want: "/openspec-archive-change demo", target: "agent"},
		{name: "no measurable tasks", files: map[string]string{"proposal.md": "# Proposal", "specs/a/spec.md": "# Spec", "tasks.md": "# Tasks"}, want: "openspec status", target: "terminal"},
		{name: "skip specs", metadata: "schema: spec-driven\nskip_specs: true\n", files: map[string]string{"proposal.md": "# Proposal", "tasks.md": "- [ ] Work"}, want: "/openspec-apply-change demo", target: "agent"},
		{name: "custom project schema", config: "schema: custom", want: "openspec status", target: "terminal"},
		{name: "custom change schema", metadata: "schema: custom", want: "openspec status", target: "terminal"},
		{name: "invalid metadata", metadata: "schema: [broken", want: "openspec status", target: "terminal"},
		{name: "invalid config", config: "schema: [broken", want: "openspec status", target: "terminal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, p, write := fixture(t)
			os.MkdirAll(filepath.Join(p.OpenSpec, "changes/demo"), 0755)
			for name, body := range tt.files {
				write("changes/demo/"+name, body)
			}
			if tt.config != "" {
				write("config.yaml", tt.config)
			}
			if tt.metadata != "" {
				write("changes/demo/.openspec.yaml", tt.metadata)
			}
			s, err := Scan(root, p)
			if err != nil {
				t.Fatal(err)
			}
			var action *NextAction
			for _, e := range s.Entries {
				if e.ID == "changes/demo" {
					action = e.NextAction
				}
			}
			if action == nil || !strings.Contains(action.Command, tt.want) || action.Target != tt.target {
				t.Fatalf("action=%+v want=%s", action, tt.want)
			}
		})
	}
}
func TestNextActionExcludedAndQuoting(t *testing.T) {
	for _, category := range []string{"specs", "archive"} {
		if action := suggestNext(Entry{Category: category}, "spec-driven", false, false); action != nil {
			t.Fatal(action)
		}
	}
	e := Entry{Name: "a 'quoted' name", Category: "changes"}
	if action := suggestNext(e, "custom", false, false); action.Command != "openspec status --change 'a '\"'\"'quoted'\"'\"' name'" {
		t.Fatal(action.Command)
	}
	if agentArgument("a name") != "\"a name\"" {
		t.Fatal("unsafe agent argument")
	}
	if action := suggestNext(Entry{Category: "changes", Documents: []Document{{Kind: "tasks", Error: "unreadable"}}}, "spec-driven", false, false); !strings.HasPrefix(action.Command, "openspec status") {
		t.Fatal(action)
	}
}
func TestNextActionUpdatesWithTasksAndMetadata(t *testing.T) {
	root, p, write := fixture(t)
	write("changes/demo/proposal.md", "# Proposal")
	write("changes/demo/tasks.md", "- [ ] Work")
	write("changes/demo/.openspec.yaml", "schema: spec-driven\nskip_specs: true")
	store, err := New(root, p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(store.Snapshot().Entries[0].NextAction.Command, "apply-change") {
		t.Fatal("no apply suggestion")
	}
	revision := store.Snapshot().Revision
	write("changes/demo/tasks.md", "- [x] Work")
	store.Refresh()
	if store.Snapshot().Revision <= revision || !strings.Contains(store.Snapshot().Entries[0].NextAction.Command, "archive-change") {
		t.Fatal("suggestion not updated")
	}
	write("changes/demo/.openspec.yaml", "schema: custom")
	store.Refresh()
	if !strings.Contains(store.Snapshot().Entries[0].NextAction.Command, "openspec status") {
		t.Fatal("schema change ignored")
	}
}
