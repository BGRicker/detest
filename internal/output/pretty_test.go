package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bgricker/detest/internal/provider"
	"github.com/bgricker/detest/internal/report"
)

func TestPrettyRenderList(t *testing.T) {
	wf := provider.Workflow{
		Path: "wf.yml",
		Name: "Workflow",
		Jobs: []provider.Job{
			{
				Name:  "Build",
				Steps: []provider.Step{{Name: "Compile", Run: "go build"}},
			},
		},
	}

	buf := &bytes.Buffer{}
	renderer := NewPretty(buf)
	if err := renderer.RenderList([]provider.Workflow{wf}); err != nil {
		t.Fatalf("render list: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Workflow Workflow (wf.yml)") {
		t.Fatalf("expected workflow header, got %q", out)
	}
	if !strings.Contains(out, "• Compile") {
		t.Fatalf("expected step bullet, got %q", out)
	}
}

func TestPrettyRenderResults(t *testing.T) {
	results := []report.StepResult{
		{
			WorkflowPath: "wf.yml",
			WorkflowName: "Workflow",
			JobName:      "Build",
			StepName:     "Compile",
			StepRun:      "go build",
			Status:       "passed",
			Duration:     123456789,
		},
		{
			WorkflowPath: "wf.yml",
			WorkflowName: "Workflow",
			JobName:      "Build",
			StepName:     "Test",
			StepRun:      "go test",
			Status:       "failed",
			Stderr:       "boom",
		},
	}

	summary := report.Summary{Passed: 1, Failed: 1, Duration: 123456789, DurationMS: 123}

	buf := &bytes.Buffer{}
	renderer := NewPretty(buf)
	if err := renderer.RenderResults(results, summary); err != nil {
		t.Fatalf("render results: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "✓ Compile") {
		t.Fatalf("expected success glyph, got %q", out)
	}
	if !strings.Contains(out, "✗ Test") {
		t.Fatalf("expected failure glyph, got %q", out)
	}
	if !strings.Contains(out, "stderr:") || !strings.Contains(out, "boom") {
		t.Fatalf("expected stderr output, got %q", out)
	}
	if !strings.Contains(out, "SUMMARY: 1 passed, 1 failed") {
		t.Fatalf("expected summary line, got %q", out)
	}
}
