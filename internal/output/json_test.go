package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/benricker/detest/internal/provider"
	"github.com/benricker/detest/internal/report"
)

func TestJSONRenderer(t *testing.T) {
	report := Report{
		Provider: "github",
		Workflows: []provider.Workflow{
			{
				Path: "wf.yml",
				Name: "Workflow",
				Jobs: []provider.Job{{Name: "build", RawID: "build"}},
			},
		},
		Steps:    []report.StepResult{{WorkflowName: "Workflow", JobName: "build", StepName: "Compile"}},
		Summary:  report.Summary{TotalWorkflows: 1, DurationMS: 10},
		Warnings: []string{"wf.yml:: note"},
	}

	buf := &bytes.Buffer{}
	renderer := NewJSON(buf)
	if err := renderer.Render(report); err != nil {
		t.Fatalf("render json: %v", err)
	}

	var decoded Report
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if decoded.Provider != report.Provider {
		t.Fatalf("provider mismatch: %s vs %s", decoded.Provider, report.Provider)
	}
	if len(decoded.Workflows) != 1 || decoded.Workflows[0].Path != "wf.yml" {
		t.Fatalf("workflow mismatch: %+v", decoded.Workflows)
	}
	if len(decoded.Warnings) != 1 {
		t.Fatalf("expected warnings serialized")
	}
}
