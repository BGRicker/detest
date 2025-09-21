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

// StreamingRenderer interface for real-time step updates.
type StreamingRenderer interface {
	InitializeWorkflow(workflowName, jobName string, stepCount int) error
	StartStep(stepName string) error
	CompleteStep(stepName string, status string, duration time.Duration, stderr string) error
	CompleteJob() error
	RenderSummary(summary report.Summary) error
}

// PrettyRenderer renders execution results in a human-friendly format.
type PrettyRenderer struct {
	out io.Writer
}

// StreamingPrettyRenderer renders execution results with real-time updates like GitHub CI.
type StreamingPrettyRenderer struct {
	out io.Writer
	stepCount int
	currentStep int
	workflowName string
	jobName string
	jobStatus string
	jobStartTime time.Time
	jobSteps []stepResult
	showStepDetails bool
}

type stepResult struct {
	name string
	status string
	duration time.Duration
	stderr string
}

// NewPretty creates a PrettyRenderer writing to the provided writer.
func NewPretty(out io.Writer) *PrettyRenderer {
	return &PrettyRenderer{out: out}
}

// NewStreamingPretty creates a StreamingPrettyRenderer for real-time updates.
func NewStreamingPretty(out io.Writer) *StreamingPrettyRenderer {
	return &StreamingPrettyRenderer{out: out}
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

// InitializeWorkflow sets up the streaming renderer for a workflow.
func (s *StreamingPrettyRenderer) InitializeWorkflow(workflowName, jobName string, stepCount int) error {
	s.workflowName = workflowName
	s.jobName = jobName
	s.stepCount = stepCount
	s.currentStep = 0
	s.jobStatus = "running"
	s.jobStartTime = time.Now()
	s.jobSteps = make([]stepResult, 0, stepCount)
	s.showStepDetails = false
	
	// Don't print workflow/job headers - we'll show clean job status instead
	return nil
}

// StartStep shows a step as running with a green circle.
func (s *StreamingPrettyRenderer) StartStep(stepName string) error {
	s.currentStep++
	label := stepName
	if label == "" {
		label = "step"
	}
	
	// Don't show step details during execution - wait for job completion
	return nil
}

// CompleteStep updates a step's status with checkmark or X.
func (s *StreamingPrettyRenderer) CompleteStep(stepName string, status string, duration time.Duration, stderr string) error {
	label := stepName
	if label == "" {
		label = "step"
	}
	
	// Store step result
	s.jobSteps = append(s.jobSteps, stepResult{
		name: label,
		status: status,
		duration: duration,
		stderr: stderr,
	})
	
	// Update job status if step failed
	if status == "failed" {
		s.jobStatus = "failed"
	}
	
	// Don't show step details during execution - wait for job completion
	return nil
}

// CompleteJob shows the final job status and step details if failed.
func (s *StreamingPrettyRenderer) CompleteJob() error {
	jobDuration := time.Since(s.jobStartTime)
	
	// Determine final job status
	if s.jobStatus == "running" {
		s.jobStatus = "passed"
	}
	
	var emoji string
	switch s.jobStatus {
	case "passed":
		emoji = "✅"
	case "failed":
		emoji = "❌"
	case "skipped":
		emoji = "⏭️"
	default:
		emoji = "❓"
	}
	
	// Show clean job-level status (no workflow/job headers)
	fmt.Fprintf(s.out, "%s %s (%s)\n", emoji, s.jobName, formatDuration(jobDuration))
	
	// If job failed, show step details
	if s.jobStatus == "failed" {
		for _, step := range s.jobSteps {
			var stepEmoji string
			switch step.status {
			case "passed":
				stepEmoji = "✅"
			case "failed":
				stepEmoji = "❌"
			case "skipped":
				stepEmoji = "⏭️"
			default:
				stepEmoji = "❓"
			}
			fmt.Fprintf(s.out, "    %s %s (%s)\n", stepEmoji, step.name, formatDuration(step.duration))
			
			// Show stderr for failed steps
			if step.status == "failed" && step.stderr != "" {
				fmt.Fprintf(s.out, "%s\n", indent(step.stderr, "      "))
			}
		}
	}
	
	return nil
}

// RenderSummary shows the final summary.
func (s *StreamingPrettyRenderer) RenderSummary(summary report.Summary) error {
	fmt.Fprintf(s.out, "SUMMARY: %d passed, %d failed, %d skipped (%s)\n", summary.Passed, summary.Failed, summary.Skipped, formatDuration(summary.Duration))
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
