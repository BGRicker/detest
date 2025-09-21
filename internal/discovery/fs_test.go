package discovery

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkflowsAuto(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	files := []string{"b.yml", "a.yaml", "c.yml"}
	for _, name := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("name: test"), 0o644); err != nil {
			t.Fatalf("write file %s: %v", name, err)
		}
	}

	got, err := Workflows(root, nil)
	if err != nil {
		t.Fatalf("Workflows returned error: %v", err)
	}

	want := []string{
		".github/workflows/a.yaml",
		".github/workflows/b.yml",
		".github/workflows/c.yml",
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d files, got %d", len(want), len(got))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: want %q, got %q", i, want[i], got[i])
		}
	}
}

func TestWorkflowsExplicit(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "workflow.yml")
	writeFile(t, file)

	externalDir := t.TempDir()
	absOutside := filepath.Join(externalDir, "external.yml")
	writeFile(t, absOutside)

	got, err := Workflows(root, []string{"workflow.yml", absOutside, "workflow.yml"})
	if err != nil {
		t.Fatalf("Workflows returned error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(got), got)
	}
	if got[0] != "workflow.yml" {
		t.Fatalf("first path mismatch: got %q", got[0])
	}
	if got[1] != absOutside {
		t.Fatalf("second path mismatch: got %q expected %q", got[1], absOutside)
	}
}

func TestWorkflowsErrors(t *testing.T) {
	root := t.TempDir()

	if _, err := Workflows(root, nil); !errors.Is(err, ErrNoWorkflows) {
		t.Fatalf("expected ErrNoWorkflows, got %v", err)
	}

	if _, err := Workflows(root, []string{"missing.yml"}); err == nil {
		t.Fatalf("expected error for missing file")
	}

	dir := filepath.Join(root, "dir.yml")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := Workflows(root, []string{"dir.yml"}); err == nil {
		t.Fatalf("expected error for directory input")
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("name: test"), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
