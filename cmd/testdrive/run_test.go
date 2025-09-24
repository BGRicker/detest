package main

import (
	"bytes"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunCommandDryPretty(t *testing.T) {
	root := projectRoot(t)
	chdir(t, root)

	cmd := newRootCmd()
	cmd.SetArgs([]string{"run", "--workflow", "testdata/workflows/ci_basic.yml", "--dry-run"})

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execute: %v", err)
	}

	want := readGolden(t, filepath.Join(root, "testdata", "golden", "run_dry_pretty.txt"))
	if diff := diffStrings(want, buf.String()); diff != "" {
		t.Fatalf("unexpected output:\n%s", diff)
	}
}

func TestRunCommandDryJSON(t *testing.T) {
	root := projectRoot(t)
	chdir(t, root)

	cmd := newRootCmd()
	cmd.SetArgs([]string{"run", "--workflow", "testdata/workflows/ci_basic.yml", "--dry-run", "--format", "json"})

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("command execute: %v", err)
	}

	want := readGolden(t, filepath.Join(root, "testdata", "golden", "run_dry_json.json"))
	if diff := diffStrings(want, buf.String()); diff != "" {
		t.Fatalf("unexpected output:\n%s", diff)
	}
}

func TestRunCommandExecuteFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("execution test unstable on windows shells")
	}

	root := projectRoot(t)
	chdir(t, root)

	cmd := newRootCmd()
	cmd.SetArgs([]string{"run", "--workflow", "testdata/workflows/ci_run.yml"})

	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(errBuf)

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error for failing workflow")
	}

	output := out.String()
	if !strings.Contains(output, "✅ Hello Step") {
		t.Fatalf("expected success marker for first step, got %q", output)
	}
	if !strings.Contains(output, "❌ Failing Step") {
		t.Fatalf("expected failure marker, got %q", output)
	}
	if !strings.Contains(output, "SUMMARY: 1 passed, 1 failed, 0 skipped") {
		t.Fatalf("expected summary, got %q", output)
	}
	if !strings.Contains(err.Error(), "one or more steps failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
