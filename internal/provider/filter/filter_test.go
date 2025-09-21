package filter

import (
	"testing"

	"github.com/benricker/detest/internal/provider"
)

func TestFilterWorkflowsByJob(t *testing.T) {
	wf := provider.Workflow{
		Path: "wf.yml",
		Name: "Example",
		Jobs: []provider.Job{
			{
				Name:  "Build",
				RawID: "build",
				Steps: []provider.Step{{Name: "Build", Run: "go build"}},
			},
			{
				Name:  "Test",
				RawID: "test",
				Steps: []provider.Step{{Name: "Test", Run: "go test"}},
			},
		},
	}

	patterns, err := Compile([]string{"build"})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	filtered := FilterWorkflows([]provider.Workflow{wf}, patterns, nil, nil)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 workflow, got %d", len(filtered))
	}
	if len(filtered[0].Jobs) != 1 || filtered[0].Jobs[0].RawID != "build" {
		t.Fatalf("expected only build job, got %+v", filtered[0].Jobs)
	}
}

func TestFilterWorkflowsSteps(t *testing.T) {
	wf := provider.Workflow{
		Path: "wf.yml",
		Name: "Example",
		Jobs: []provider.Job{
			{
				Name:  "Test",
				RawID: "test",
				Steps: []provider.Step{
					{Name: "Install", Uses: "actions/setup"},
					{Name: "Lint", Run: "go vet ./..."},
					{Name: "Unit", Run: "go test ./..."},
				},
			},
		},
	}

	only, err := Compile([]string{"/go/"})
	if err != nil {
		t.Fatalf("compile only: %v", err)
	}
	skip, err := Compile([]string{"unit"})
	if err != nil {
		t.Fatalf("compile skip: %v", err)
	}

	filtered := FilterWorkflows([]provider.Workflow{wf}, nil, only, skip)
	if len(filtered) != 1 {
		t.Fatalf("expected workflow retained")
	}
	steps := filtered[0].Jobs[0].Steps
	if len(steps) != 1 {
		t.Fatalf("expected 1 step after filtering, got %d", len(steps))
	}
	if steps[0].Name != "Lint" {
		t.Fatalf("expected Lint step, got %s", steps[0].Name)
	}
}

func TestCompileErrors(t *testing.T) {
	if _, err := Compile([]string{"/(/"}); err == nil {
		t.Fatalf("expected compile error")
	}
}
