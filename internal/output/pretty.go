package output

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bgricker/detest/internal/provider"
	"github.com/bgricker/detest/internal/report"
)

// PrettyRenderer renders execution results in a human-friendly format.
type PrettyRenderer struct {
	out io.Writer
}

// NewPretty creates a PrettyRenderer writing to the provided writer.
func NewPretty(out io.Writer) *PrettyRenderer {
	return &PrettyRenderer{out: out}
}

// RenderList renders workflows/jobs/steps in list mode.
func (p *PrettyRenderer) RenderList(workflows []provider.Workflow) error {
	for _, wf := range workflows {
		if _, err := fmt.Fprintf(p.out, "Workflow %s\n", decorateName(wf.Name, wf.Path)); err != nil {
			return err
		}
		for _, job := range wf.Jobs {
			if _, err := fmt.Fprintf(p.out, "  Job %s\n", job.Name); err != nil {
				return err
			}
			for _, step := range job.Steps {
				label := step.Name
				if label == "" {
					label = step.Run
				}
				if step.Run == "" {
					continue
				}
				if _, err := fmt.Fprintf(p.out, "    • %s\n", label); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// RenderResults shows execution outcomes for steps with a summary.
func (p *PrettyRenderer) RenderResults(results []report.StepResult, summary report.Summary) error {
	type key struct {
		workflow string
		job      string
	}

	var current key
	var buffer bytes.Buffer

	flush := func() error {
		if buffer.Len() == 0 {
			return nil
		}
		if _, err := buffer.WriteTo(p.out); err != nil {
			return err
		}
		buffer.Reset()
		return nil
	}

	for _, res := range results {
		k := key{workflow: res.WorkflowName, job: res.JobName}
		if current != k {
			if err := flush(); err != nil {
				return err
			}
			current = k
			fmt.Fprintf(&buffer, "Workflow %s\n", decorateName(res.WorkflowName, res.WorkflowPath))
			fmt.Fprintf(&buffer, "  Job %s\n", res.JobName)
		}

		statusSymbol := statusGlyph(res.Status)
		duration := formatDuration(res.Duration)
		label := res.StepName
		if label == "" {
			label = res.StepRun
		}
		fmt.Fprintf(&buffer, "    %s %s (%s)\n", statusSymbol, label, duration)
		if res.Status == "failed" && res.Stderr != "" {
			fmt.Fprintf(&buffer, "      stderr: %s\n", indent(res.Stderr, "      "))
		}
		if res.Status == "skipped" && res.Stderr != "" {
			fmt.Fprintf(&buffer, "      note: %s\n", indent(res.Stderr, "      "))
		}
		if res.DryRun {
			fmt.Fprintf(&buffer, "      command: %s\n", res.StepRun)
		}
	}

	if err := flush(); err != nil {
		return err
	}

	fmt.Fprintf(p.out, "SUMMARY: %d passed, %d failed, %d skipped (%s)\n", summary.Passed, summary.Failed, summary.Skipped, formatDuration(summary.Duration))
	return nil
}

func decorateName(name, path string) string {
	if name == "" || name == path {
		return path
	}
	return fmt.Sprintf("%s (%s)", name, path)
}

func statusGlyph(status string) string {
	switch status {
	case "passed":
		return "✓"
	case "failed":
		return "✗"
	case "skipped":
		return "-"
	default:
		return "?"
	}
}

func indent(s, pad string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = pad + lines[i]
	}
	return strings.Join(lines, "\n")
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	return d.Truncate(time.Millisecond).String()
}
