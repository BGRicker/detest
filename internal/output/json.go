package output

import (
	"encoding/json"
	"io"

	"github.com/bgricker/detest/internal/provider"
	"github.com/bgricker/detest/internal/report"
)

// JSONRenderer emits structured execution data.
type JSONRenderer struct {
	out io.Writer
}

// NewJSON creates a JSON renderer writing to out.
func NewJSON(out io.Writer) *JSONRenderer {
	return &JSONRenderer{out: out}
}

// Report captures JSON output schema.
type Report struct {
	Provider  string              `json:"provider"`
	Workflows []provider.Workflow `json:"workflows"`
	Steps     []report.StepResult `json:"steps,omitempty"`
	Summary   report.Summary      `json:"summary"`
	Warnings  []string            `json:"warnings,omitempty"`
}

// Render encodes the report as JSON.
func (j *JSONRenderer) Render(report Report) error {
	enc := json.NewEncoder(j.out)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
