package report

import "time"

// StepResult captures the outcome of a single step.
type StepResult struct {
    WorkflowPath string        `json:"workflow_path"`
    WorkflowName string        `json:"workflow_name"`
    JobName      string        `json:"job_name"`
    StepName     string        `json:"step_name"`
    StepRun      string        `json:"step_run"`
    Status       string        `json:"status"`
    Duration     time.Duration `json:"-"`
    DurationMS   int64         `json:"duration_ms"`
    Stdout       string        `json:"stdout,omitempty"`
    Stderr       string        `json:"stderr,omitempty"`
    ExitCode     int           `json:"exit_code"`
    DryRun       bool          `json:"dry_run"`
}

// Summary aggregates pipeline execution results.
type Summary struct {
    TotalWorkflows int           `json:"total_workflows"`
    TotalJobs      int           `json:"total_jobs"`
    TotalSteps     int           `json:"total_steps"`
    Passed         int           `json:"passed"`
    Failed         int           `json:"failed"`
    Skipped        int           `json:"skipped"`
    Duration       time.Duration `json:"-"`
    DurationMS     int64         `json:"duration_ms"`
    ExitCode       int           `json:"exit_code"`
}
