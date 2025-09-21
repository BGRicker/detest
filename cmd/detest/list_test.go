package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestListCommandBasic(t *testing.T) {
	root := projectRoot(t)
	chdir(t, root)

	cmd := newRootCmd()
	cmd.SetArgs([]string{"list", "--workflow", "testdata/workflows/ci_basic.yml"})

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execute: %v", err)
	}

	want := readGolden(t, filepath.Join(root, "testdata", "golden", "list_basic.txt"))
	if diff := diffStrings(want, buf.String()); diff != "" {
		t.Fatalf("unexpected output:\n%s", diff)
	}
}

func TestListCommandFilters(t *testing.T) {
	root := projectRoot(t)
	chdir(t, root)

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"list",
		"--workflow", "testdata/workflows/ci_envs.yml",
		"--job", "unit",
		"--only-step", "/Step/",
	})

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execute: %v", err)
	}

	want := readGolden(t, filepath.Join(root, "testdata", "golden", "list_filter.txt"))
	if diff := diffStrings(want, buf.String()); diff != "" {
		t.Fatalf("unexpected output:\n%s", diff)
	}
}

func TestListCommandJSON(t *testing.T) {
	root := projectRoot(t)
	chdir(t, root)

	cmd := newRootCmd()
	cmd.SetArgs([]string{"list", "--workflow", "testdata/workflows/ci_basic.yml", "--format", "json"})

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execute: %v", err)
	}

	want := readGolden(t, filepath.Join(root, "testdata", "golden", "list_basic.json"))
	if diff := diffStrings(want, buf.String()); diff != "" {
		t.Fatalf("unexpected output:\n%s", diff)
	}
}

func TestListCommandConfig(t *testing.T) {
	root := projectRoot(t)
	tmp := t.TempDir()
	copyDir(t, filepath.Join(root, "testdata"), filepath.Join(tmp, "testdata"))

	configYAML := []byte(`provider: github
workflows:
  - testdata/workflows/ci_envs.yml
jobs:
  - /Unit/
only_step:
  - step one
`)
	if err := os.WriteFile(filepath.Join(tmp, ".detest.yml"), configYAML, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	chdir(t, tmp)

	cmd := newRootCmd()
	cmd.SetArgs([]string{"list"})

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execute: %v", err)
	}

	want := readGolden(t, filepath.Join(root, "testdata", "golden", "list_filter.txt"))
	if diff := diffStrings(want, buf.String()); diff != "" {
		t.Fatalf("unexpected output:\n%s", diff)
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("locate project root: %v", err)
	}
	return root
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %q: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(prev); err != nil {
			t.Fatalf("restore dir: %v", err)
		}
	})
}

func readGolden(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %q: %v", path, err)
	}
	return string(data)
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("read dir %q: %v", src, err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", dst, err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			copyDir(t, srcPath, dstPath)
			continue
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("read file %q: %v", srcPath, err)
		}
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			t.Fatalf("write file %q: %v", dstPath, err)
		}
	}
}

func diffStrings(want, got string) string {
	if want == got {
		return ""
	}
	return "--- want\n" + want + "\n--- got\n" + got
}
