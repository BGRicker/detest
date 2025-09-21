package github

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParserParseBasic(t *testing.T) {
	root := projectRoot(t)
	parser := NewParser(root)
	paths := []string{
		"testdata/workflows/ci_basic.yml",
		"testdata/workflows/ci_envs.yml",
	}

	pipeline, err := parser.Parse(paths)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if pipeline.Provider != ProviderName {
		t.Fatalf("expected provider %q, got %q", ProviderName, pipeline.Provider)
	}

	if len(pipeline.Workflows) != len(paths) {
		t.Fatalf("expected %d workflows, got %d", len(paths), len(pipeline.Workflows))
	}

	basic := pipeline.Workflows[0]
	if basic.Name != "Basic CI" {
		t.Fatalf("expected workflow name 'Basic CI', got %q", basic.Name)
	}
	if len(basic.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(basic.Jobs))
	}
	job := basic.Jobs[0]
	if job.RawID != "build" {
		t.Fatalf("expected job raw id 'build', got %q", job.RawID)
	}
	if job.Name != "build" {
		t.Fatalf("expected job name fallback to id, got %q", job.Name)
	}
	if len(job.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(job.Steps))
	}
	if job.Steps[0].Uses == "" {
		t.Fatalf("expected first step to be uses step")
	}
	if job.Steps[1].Run != "go test ./..." {
		t.Fatalf("expected run command preserved, got %q", job.Steps[1].Run)
	}

	envWorkflow := pipeline.Workflows[1]
	if envWorkflow.Defaults.RunShell != "bash" {
		t.Fatalf("expected workflow default shell bash, got %q", envWorkflow.Defaults.RunShell)
	}
	if envWorkflow.Env["WF_VAR"] != "wf-value" {
		t.Fatalf("unexpected workflow env: %v", envWorkflow.Env)
	}
	if envWorkflow.Env["SHARED"] != "wf" {
		t.Fatalf("workflow env SHARED expected wf, got %q", envWorkflow.Env["SHARED"])
	}

	if len(envWorkflow.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(envWorkflow.Jobs))
	}
	envJob := envWorkflow.Jobs[0]
	if envJob.Env["JOB_VAR"] != "job-value" {
		t.Fatalf("unexpected job env: %v", envJob.Env)
	}
	if envJob.Env["SHARED"] != "job" {
		t.Fatalf("job env SHARED expected job, got %q", envJob.Env["SHARED"])
	}
	if envJob.Defaults.WorkingDirectory != "./app" {
		t.Fatalf("expected job working directory ./app, got %q", envJob.Defaults.WorkingDirectory)
	}
	if len(envJob.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(envJob.Steps))
	}
	step := envJob.Steps[0]
	if step.Env["STEP_VAR"] != "step-value" {
		t.Fatalf("unexpected step env: %v", step.Env)
	}
	if step.Env["SHARED"] != "step" {
		t.Fatalf("step env SHARED expected step, got %q", step.Env["SHARED"])
	}
}

func TestParserWarnings(t *testing.T) {
	root := projectRoot(t)
	parser := NewParser(root)
	pipeline, err := parser.Parse([]string{"testdata/workflows/ci_services_matrix.yml"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if len(pipeline.Warnings) != 4 {
		t.Fatalf("expected 4 warnings, got %d", len(pipeline.Warnings))
	}

	messages := make([]string, 0, len(pipeline.Warnings))
	for _, w := range pipeline.Warnings {
		messages = append(messages, w.Message)
	}

	mustContain(t, messages, "services are not supported")
	mustContain(t, messages, "strategy.matrix is not supported")
	mustContain(t, messages, "job-level if condition is ignored")
	mustContain(t, messages, "unsupported if condition")
}

func TestParserMissingFile(t *testing.T) {
	root := projectRoot(t)
	parser := NewParser(root)
	_, err := parser.Parse([]string{"testdata/workflows/missing.yml"})
	if err == nil {
		t.Fatalf("expected error for missing workflow")
	}
}

func TestStepNameFallback(t *testing.T) {
	yamlDoc := `name: unnamed
jobs:
  build:
    steps:
      - run: echo one
      - name: Explicit
        run: echo two
`
	wf, warnings, err := decodeWorkflow(strings.NewReader(yamlDoc), "temp.yml")
	if err != nil {
		t.Fatalf("decodeWorkflow error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if len(wf.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(wf.Jobs))
	}
	steps := wf.Jobs[0].Steps
	if steps[0].Name != "step 1" {
		t.Fatalf("expected first step name fallback 'step 1', got %q", steps[0].Name)
	}
	if steps[1].Name != "Explicit" {
		t.Fatalf("expected second step name preserved, got %q", steps[1].Name)
	}
}

func TestParseInvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "broken.yml")
	if err := os.WriteFile(path, []byte("::bad yaml"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, _, err := decodeWorkflow(strings.NewReader("::bad yaml"), "broken.yml"); err == nil {
		t.Fatalf("expected parse error for invalid yaml")
	}

	parser := NewParser(tmp)
	if _, err := parser.Parse([]string{"broken.yml"}); err == nil {
		t.Fatalf("expected error from parser.Parse on invalid yaml")
	}
}

func TestConvertEnv(t *testing.T) {
	env := map[string]interface{}{"B": 2, "A": "1"}
	converted := convertEnv(env)
	if len(converted) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(converted))
	}
	if converted["A"] != "1" {
		t.Fatalf("expected string conversion for key A, got %q", converted["A"])
	}
	if converted["B"] != "2" {
		t.Fatalf("expected string conversion for key B, got %q", converted["B"])
	}
}

func TestParserParseMissingJobs(t *testing.T) {
	yamlDoc := `name: Empty Jobs`
	wf, _, err := decodeWorkflow(strings.NewReader(yamlDoc), "empty.yml")
	if err != nil {
		t.Fatalf("decodeWorkflow error: %v", err)
	}
	if len(wf.Jobs) != 0 {
		t.Fatalf("expected no jobs, got %d", len(wf.Jobs))
	}
}

func TestParseWorkflowFileError(t *testing.T) {
	_, _, err := decodeWorkflow(&errorReader{}, "bad.yml")
	if err == nil {
		t.Fatalf("expected error from reader")
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	info, err := os.Stat(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("locating project root: %v", err)
	}
	if info.IsDir() {
		t.Fatalf("expected go.mod at %q to be a file", filepath.Join(root, "go.mod"))
	}
	return root
}

func mustContain(t *testing.T, list []string, target string) {
	t.Helper()
	for _, item := range list {
		if strings.Contains(item, target) {
			return
		}
	}
	t.Fatalf("expected to find %q in %v", target, list)
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("boom")
}
