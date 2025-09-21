package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bgricker/detest/internal/provider"
)

func TestRunnerDryRun(t *testing.T) {
	opts := Options{DryRun: true}
	r := New(opts)
	wf := sampleWorkflow("echo hi")

	results, summary, err := r.Run([]provider.Workflow{wf})
	if err != nil {
		t.Fatalf("runner Run: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "skipped" || !results[0].DryRun {
		t.Fatalf("expected skipped dry run, got %+v", results[0])
	}
	if summary.Skipped != 1 || summary.TotalSteps != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestRunnerExecSuccess(t *testing.T) {
	root := t.TempDir()
	stdout := &bytes.Buffer{}
	r := New(Options{Root: root, Stdout: stdout})
	wf := sampleWorkflow("echo hi")

	results, summary, err := r.Run([]provider.Workflow{wf})
	if err != nil {
		t.Fatalf("runner Run: %v", err)
	}
	if summary.Passed != 1 || summary.ExitCode != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if strings.TrimSpace(results[0].Stdout) != "hi" {
		t.Fatalf("expected stdout 'hi', got %q", results[0].Stdout)
	}
}

func TestRunnerExecFailure(t *testing.T) {
	root := t.TempDir()
	r := New(Options{Root: root})
	wf := sampleWorkflow("exit 3")

	results, summary, err := r.Run([]provider.Workflow{wf})
	if err != nil {
		t.Fatalf("runner Run: %v", err)
	}
	if summary.Failed != 1 || summary.ExitCode != 1 {
		t.Fatalf("expected failure summary, got %+v", summary)
	}
	if results[0].Status != "failed" || results[0].ExitCode == 0 {
		t.Fatalf("unexpected result: %+v", results[0])
	}
}

func TestRunnerEnvMerge(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("env merge test requires POSIX shell")
	}
	root := t.TempDir()
	r := New(Options{Root: root})
	wf := provider.Workflow{
		Path: "wf.yml",
		Name: "wf",
		Env:  map[string]string{"WF_VAR": "wf"},
		Jobs: []provider.Job{
			{
				Name:  "job",
				RawID: "job",
				Env:   map[string]string{"JOB_VAR": "job"},
				Steps: []provider.Step{
					{
						Name: "step",
						Run:  echoCommand(`$WF_VAR-$JOB_VAR-$STEP_VAR`),
						Env:  map[string]string{"STEP_VAR": "step"},
					},
				},
			},
		},
	}

	results, _, err := r.Run([]provider.Workflow{wf})
	if err != nil {
		t.Fatalf("runner Run: %v", err)
	}
	if want := "wf-job-step"; !strings.Contains(results[0].Stdout, want) {
		t.Fatalf("expected output %q, got %q", want, results[0].Stdout)
	}
}

func TestRunnerWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("working directory test uses POSIX commands")
	}
	root := t.TempDir()
	sub := filepath.Join(root, "subdir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}
	r := New(Options{Root: root})
	wf := provider.Workflow{
		Path: "wf.yml",
		Name: "wf",
		Jobs: []provider.Job{
			{
				Name: "job",
				Defaults: provider.Defaults{
					WorkingDirectory: "subdir",
				},
				Steps: []provider.Step{{
					Name: "pwd",
					Run:  pwdCommand(),
				}},
			},
		},
	}

	results, _, err := r.Run([]provider.Workflow{wf})
	if err != nil {
		t.Fatalf("runner Run: %v", err)
	}
	if !strings.Contains(results[0].Stdout, "subdir") {
		t.Fatalf("expected working dir output to include subdir, got %q", results[0].Stdout)
	}
}

func TestRunnerTailCapture(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tail capture test requires POSIX tools")
	}
	root := t.TempDir()
	r := New(Options{Root: root, TailLines: 2})
	wf := sampleWorkflow("printf '1\n2\n3\n'; exit 1")

	results, _, err := r.Run([]provider.Workflow{wf})
	if err != nil {
		t.Fatalf("runner Run: %v", err)
	}
	if got := strings.TrimSpace(results[0].Stdout); got != "2\n3" {
		t.Fatalf("expected tail '2\\n3', got %q", got)
	}
}

func TestRunnerSkipsPrivilegedCommands(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("skip logic focused on non-linux hosts")
	}
	root := t.TempDir()
	r := New(Options{Root: root})
	wf := sampleWorkflow("sudo apt-get update")

	results, summary, err := r.Run([]provider.Workflow{wf})
	if err != nil {
		t.Fatalf("runner Run: %v", err)
	}
	if summary.Skipped != 1 {
		t.Fatalf("expected skipped count 1, got %+v", summary)
	}
	if results[0].Status != "skipped" {
		t.Fatalf("expected step skipped, got %+v", results[0])
	}
	if results[0].Stderr == "" {
		t.Fatalf("expected skip message")
	}
}

func TestSimplifyErrorBundler(t *testing.T) {
	msg := "Could not find 'bundler' (2.6.9) required by your Gemfile.lock"
	simplified := simplifyError(msg)
	if !strings.Contains(simplified, "gem install bundler:2.6.9") {
		t.Fatalf("expected actionable bundler message, got %q", simplified)
	}
}

func sampleWorkflow(script string) provider.Workflow {
	return provider.Workflow{
		Path: "wf.yml",
		Name: "workflow",
		Jobs: []provider.Job{
			{
				Name:  "job",
				RawID: "job",
				Steps: []provider.Step{{
					Name: "step",
					Run:  script,
				}},
			},
		},
	}
}

func echoCommand(arg string) string {
	return "echo " + arg
}

func pwdCommand() string {
	if runtime.GOOS == "windows" {
		return "cd"
	}
	return "pwd"
}
