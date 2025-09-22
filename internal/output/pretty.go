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
	InitializeAllJobs(workflows []provider.Workflow) error
	StartJob(jobName string) error
	InitializeWorkflow(workflowName, jobName string, stepCount int) error
	StartStep(stepName string) error
	CompleteStep(stepName string, status string, duration time.Duration, stdout, stderr, command string) error
	CompleteJob() error
	RenderSummary(summary report.Summary) error
}

// TimerController is an optional interface for renderers that support a live timer.
type TimerController interface {
    StartTimer()
    StopTimer()
}

// PrettyRenderer renders execution results in a human-friendly format.
type PrettyRenderer struct {
	out io.Writer
}

// StreamingPrettyRenderer renders execution results with real-time updates like GitHub CI.
type StreamingPrettyRenderer struct {
	out io.Writer
	workflows []workflowInfo
	currentWorkflow int
	currentJob int
    // Timer controls for live updates
    stopTimer chan struct{}
}

type workflowInfo struct {
	name string
	jobs []jobInfo
}

type jobInfo struct {
	name string
	status string
	startTime time.Time
	duration time.Duration
	steps []stepResult
	lineNumber int // For cursor positioning
}

type stepResult struct {
	name string
	status string
	duration time.Duration
	stderr string
	stdout string
	command string
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

// InitializeAllJobs shows all jobs upfront with running indicators
func (s *StreamingPrettyRenderer) InitializeAllJobs(workflows []provider.Workflow) error {
	// Clear existing workflows
	s.workflows = []workflowInfo{}
	
	// Add all workflows and jobs
	for _, wf := range workflows {
		workflow := workflowInfo{
			name: wf.Name,
			jobs: []jobInfo{},
		}
		
		for _, job := range wf.Jobs {
			// Count run steps for this job
			stepCount := 0
			for _, step := range job.Steps {
				if step.Run != "" && step.Uses == "" {
					stepCount++
				}
			}
			
			jobInfo := jobInfo{
				name: job.Name,
				status: "pending", // Start as pending, not running
				startTime: time.Now(),
				duration: 0,
				steps: make([]stepResult, 0, stepCount),
			}
			
			workflow.jobs = append(workflow.jobs, jobInfo)
			
			// Show the job as pending initially
			fmt.Fprintf(s.out, "⏳ %s\n", job.Name)
		}
		
		s.workflows = append(s.workflows, workflow)
	}
	
	return nil
}

// StartJob marks a job as running and starts its timer
func (s *StreamingPrettyRenderer) StartJob(jobName string) error {
	// Find the job and mark it as running
	for _, workflow := range s.workflows {
		for i := range workflow.jobs {
			if workflow.jobs[i].name == jobName && workflow.jobs[i].status == "pending" {
				workflow.jobs[i].status = "running"
				workflow.jobs[i].startTime = time.Now() // Reset start time when job actually starts
				return nil
			}
		}
	}
	return nil
}

// InitializeWorkflow is kept for interface compatibility but not used in the new approach
func (s *StreamingPrettyRenderer) InitializeWorkflow(workflowName, jobName string, stepCount int) error {
	// Jobs are already initialized upfront, this method is not used
	return nil
}

// StartStep shows a step as running with a green circle.
func (s *StreamingPrettyRenderer) StartStep(stepName string) error {
	// Don't show step details during execution - wait for job completion
	return nil
}

// CompleteStep updates a step's status with checkmark or X.
func (s *StreamingPrettyRenderer) CompleteStep(stepName string, status string, duration time.Duration, stdout, stderr, command string) error {
	// Find the current job by looking for the most recent running job
	for _, workflow := range s.workflows {
		for i := range workflow.jobs {
			if workflow.jobs[i].status == "running" {
				job := &workflow.jobs[i]
				job.steps = append(job.steps, stepResult{
					name: stepName,
					status: status,
					duration: duration,
					stderr: stderr,
					stdout: stdout,
					command: command,
				})
				
				// Don't change job status here - let CompleteJob() handle it
				return nil
			}
		}
	}
	
	return nil
}

// CompleteJob shows the final job status and step details if failed.
func (s *StreamingPrettyRenderer) CompleteJob() error {
	// Find the current job by looking for the most recent running job
	for _, workflow := range s.workflows {
		for i := range workflow.jobs {
			if workflow.jobs[i].status == "running" {
				job := &workflow.jobs[i]
				job.duration = time.Since(job.startTime)
				
				// Determine final job status based on steps
				job.status = "passed" // Default to passed
				for _, step := range job.steps {
					if step.status == "failed" {
						job.status = "failed"
						break
					}
				}
				
				// If job failed, show step details and don't update running jobs
				if job.status == "failed" {
					s.showJobDetails(job)
					return nil
				}
				
				// Force an immediate display update to show the final job status
				s.updateRunningJobs()
				return nil
			}
		}
	}
	
	return nil
}

// updateJobLine updates the job status line in place
func (s *StreamingPrettyRenderer) updateJobLine(job *jobInfo) {
	var emoji string
	switch job.status {
	case "passed":
		emoji = "✅"
	case "failed":
		emoji = "❌"
	case "skipped":
		emoji = "⏭️"
	default:
		emoji = "❓"
	}
	
	// Move cursor up to the job line and overwrite it
	fmt.Fprintf(s.out, "\033[1A\033[K") // Move up, clear line
	fmt.Fprintf(s.out, "%s %s (%s)\n", emoji, job.name, formatDuration(job.duration))
}

// showJobDetails shows step details for failed jobs
func (s *StreamingPrettyRenderer) showJobDetails(job *jobInfo) {
	for _, step := range job.steps {
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
		
		// Show stderr for failed steps with better formatting
		if step.status == "failed" {
			// Show the command that failed first
			if step.command != "" {
				fmt.Fprintf(s.out, "      Command: %s\n", step.command)
			}
			
			// Combine stdout and stderr for RSpec parsing
			combinedOutput := step.stdout + "\n" + step.stderr
			cleanedOutput := cleanErrorOutput(combinedOutput)
			if cleanedOutput != "" {
				fmt.Fprintf(s.out, "%s\n", indent(cleanedOutput, "      "))
			}
		}
	}
}

// RenderSummary shows the final summary.
func (s *StreamingPrettyRenderer) RenderSummary(summary report.Summary) error {
	fmt.Fprintf(s.out, "SUMMARY: %d passed, %d failed, %d skipped (%s)\n", summary.Passed, summary.Failed, summary.Skipped, formatDuration(summary.Duration))
	return nil
}

// StartTimer starts a background timer that updates running jobs with live elapsed time
// Optional timer control interface
func (s *StreamingPrettyRenderer) StartTimer() {
    if s.stopTimer != nil {
        return
    }
    s.stopTimer = make(chan struct{})
    go func() {
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                s.updateRunningJobs()
            case <-s.stopTimer:
                return
            }
        }
    }()
}

func (s *StreamingPrettyRenderer) StopTimer() {
    if s.stopTimer != nil {
        close(s.stopTimer)
        s.stopTimer = nil
    }
}

// updateRunningJobs updates all running jobs with current elapsed time
func (s *StreamingPrettyRenderer) updateRunningJobs() {
	// Count how many lines we need to move up
	totalJobs := 0
	for _, workflow := range s.workflows {
		totalJobs += len(workflow.jobs)
	}
	
	// Move cursor up to the first job line
	for i := 0; i < totalJobs; i++ {
		fmt.Fprintf(s.out, "\033[1A") // Move up one line
	}
	
	// Update all jobs
	for _, workflow := range s.workflows {
		for _, job := range workflow.jobs {
			if job.status == "pending" {
				fmt.Fprintf(s.out, "\033[K") // Clear line
				fmt.Fprintf(s.out, "⏳ %s\n", job.name)
			} else if job.status == "running" {
				elapsed := time.Since(job.startTime)
				fmt.Fprintf(s.out, "\033[K") // Clear line
				fmt.Fprintf(s.out, "🟢 %s (%s)\n", job.name, formatDuration(elapsed))
			} else {
				// Job is complete, show final status
				var emoji string
				switch job.status {
				case "passed":
					emoji = "✅"
				case "failed":
					emoji = "❌"
				case "skipped":
					emoji = "⏭️"
				default:
					emoji = "❓"
				}
				fmt.Fprintf(s.out, "\033[K") // Clear line
				fmt.Fprintf(s.out, "%s %s (%s)\n", emoji, job.name, formatDuration(job.duration))
			}
		}
	}
}

// cleanErrorOutput removes noise and makes error output more readable
func cleanErrorOutput(stderr string) string {
	lines := strings.Split(stderr, "\n")
	
	// Check if this looks like RSpec output
	if isRSpecOutput(lines) {
		return formatRSpecFailures(lines)
	}
	
	// Otherwise, use the general cleaning logic
	var cleaned []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines
		if line == "" {
			continue
		}
		
		// Skip asdf migration warnings
		if strings.Contains(line, "Bash implementation") ||
		   strings.Contains(line, "Migration guide") ||
		   strings.Contains(line, "asdf website") ||
		   strings.Contains(line, "Source code") ||
		   strings.Contains(line, "migrate to the new version") {
			continue
		}
		
		// Skip parser warnings
		if strings.Contains(line, "parser/current is loading parser") ||
		   strings.Contains(line, "Please see https://github.com/whitequark/parser") {
			continue
		}
		
		// Skip config file warnings
		if strings.Contains(line, "config file has been renamed") ||
		   strings.Contains(line, "is deprecated") {
			continue
		}
		
		// Skip shoulda-matchers warnings
		if strings.Contains(line, "Warning from shoulda-matchers") ||
		   strings.Contains(line, "validate_inclusion_of") ||
		   strings.Contains(line, "boolean column") ||
		   strings.Contains(line, "************************************************************************") {
			continue
		}
		
		// Keep important error lines
		if strings.Contains(line, "failed") ||
		   strings.Contains(line, "error") ||
		   strings.Contains(line, "Error") ||
		   strings.Contains(line, "FAILED") ||
		   strings.Contains(line, "aborted") ||
		   strings.Contains(line, "Tasks: TOP") {
			cleaned = append(cleaned, line)
		}
	}
	
	// If we have cleaned lines, return them; otherwise return a simple message
	if len(cleaned) > 0 {
		return strings.Join(cleaned, "\n")
	}
	
	return "Step failed - see verbose output for details"
}

// isRSpecOutput checks if the output looks like RSpec test results
func isRSpecOutput(lines []string) bool {
	for _, line := range lines {
		if strings.Contains(line, "Failures:") ||
		   strings.Contains(line, "Failed examples:") ||
		   strings.Contains(line, "rspec ./spec/") ||
		   strings.Contains(line, "Finished in") ||
		   strings.Contains(line, "examples,") ||
		   strings.Contains(line, "Failure/Error:") {
			return true
		}
	}
	return false
}

// formatRSpecFailures formats RSpec failure output in a clean, hierarchical way
func formatRSpecFailures(lines []string) string {
	var result []string
	var currentFailure []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and noise
		if line == "" || 
		   strings.Contains(line, "Bash implementation") ||
		   strings.Contains(line, "Migration guide") ||
		   strings.Contains(line, "asdf website") ||
		   strings.Contains(line, "Source code") ||
		   strings.Contains(line, "migrate to the new version") ||
		   strings.Contains(line, "parser/current is loading parser") ||
		   strings.Contains(line, "Please see https://github.com/whitequark/parser") ||
		   strings.Contains(line, "config file has been renamed") ||
		   strings.Contains(line, "is deprecated") ||
		   strings.Contains(line, "Warning from shoulda-matchers") ||
		   strings.Contains(line, "validate_inclusion_of") ||
		   strings.Contains(line, "boolean column") ||
		   strings.Contains(line, "************************************************************************") ||
		   strings.Contains(line, "Finished in") ||
		   strings.Contains(line, "examples,") ||
		   strings.Contains(line, "Randomized with seed") ||
		   strings.Contains(line, "Pending:") ||
		   strings.Contains(line, "Not yet implemented") ||
		   strings.Contains(line, "Database connection mocking") ||
		   strings.Contains(line, "# ./spec/support/database_cleaner.rb") {
			continue
		}
		
		// Start of a new failure (numbered like "2) DetectMovementsJob...")
		if strings.Contains(line, ") ") && !strings.Contains(line, "Failure/Error:") {
			if len(currentFailure) > 0 {
				result = append(result, formatSingleFailure(currentFailure)...)
			}
			currentFailure = []string{line}
		} else if len(currentFailure) > 0 {
			// Continue collecting details for current failure
			if strings.Contains(line, "Failure/Error:") ||
			   strings.Contains(line, "expected") ||
			   strings.Contains(line, "got") ||
			   strings.HasPrefix(line, "# ./spec/") {
				currentFailure = append(currentFailure, line)
			}
		}
	}
	
	// Handle the last failure
	if len(currentFailure) > 0 {
		result = append(result, formatSingleFailure(currentFailure)...)
	}
	
	if len(result) > 0 {
		return strings.Join(result, "\n")
	}
	
	return "RSpec tests failed - see verbose output for details"
}

// formatSingleFailure formats a single RSpec failure
func formatSingleFailure(failureLines []string) []string {
	var result []string
	
	for i, line := range failureLines {
		if i == 0 {
			// Extract the spec file and line number from the failure line
			// Format: "1) EspnInjuryService.get_injury_summary_for_event provides injury summary for both teams"
			// We need to extract the spec file from the stack trace later
			if strings.Contains(line, "Failure/Error:") {
				// Extract the failure message
				if idx := strings.Index(line, "Failure/Error:"); idx != -1 {
					failureMsg := strings.TrimSpace(line[idx+len("Failure/Error:"):])
					result = append(result, fmt.Sprintf("        ❌ %s", failureMsg))
				}
			}
		} else if strings.Contains(line, "expected") && strings.Contains(line, "got") {
			// This is the detailed error message
			result = append(result, fmt.Sprintf("                    %s", line))
		} else if strings.HasPrefix(line, "# ./spec/") {
			// Extract the spec file path
			specPath := strings.TrimPrefix(line, "# ./")
			result = append(result, fmt.Sprintf("        ❌ %s", specPath))
		}
	}
	
	return result
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
