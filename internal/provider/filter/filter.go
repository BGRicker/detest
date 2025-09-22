package filter

import (
	"fmt"
	"regexp"
	"strings"

    "github.com/bgricker/testdrive/internal/provider"
)

// Pattern represents a compiled filter condition supporting substring and regex matching.
type Pattern struct {
	raw   string
	regex *regexp.Regexp
	lower string
}

// Compile transforms raw pattern strings into Pattern values.
func Compile(patterns []string) ([]Pattern, error) {
	result := make([]Pattern, 0, len(patterns))
	for _, raw := range patterns {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if strings.HasPrefix(raw, "/") && strings.HasSuffix(raw, "/") && len(raw) >= 2 {
			expr := raw[1 : len(raw)-1]
			re, err := regexp.Compile(expr)
			if err != nil {
				return nil, fmt.Errorf("compile regexp %q: %w", raw, err)
			}
			result = append(result, Pattern{raw: raw, regex: re})
			continue
		}
		result = append(result, Pattern{raw: raw, lower: strings.ToLower(raw)})
	}
	return result, nil
}

// Match reports whether the pattern matches the supplied string.
func (p Pattern) Match(s string) bool {
	if s == "" {
		return false
	}
	if p.regex != nil {
		return p.regex.MatchString(s)
	}
	return strings.Contains(strings.ToLower(s), p.lower)
}

// FilterWorkflows applies job and step filters to workflows, returning a new slice with matches.
func FilterWorkflows(workflows []provider.Workflow, jobPatterns, onlyPatterns, skipPatterns []Pattern) []provider.Workflow {
	if len(workflows) == 0 {
		return nil
	}

	result := make([]provider.Workflow, 0, len(workflows))
	for _, wf := range workflows {
		filteredJobs := make([]provider.Job, 0, len(wf.Jobs))
		for _, job := range wf.Jobs {
			if len(jobPatterns) > 0 && !matchesJob(job, jobPatterns) {
				continue
			}
			filteredSteps := filterSteps(job.Steps, onlyPatterns, skipPatterns)
			if len(filteredSteps) == 0 {
				continue
			}
			jobCopy := job
			jobCopy.Steps = filteredSteps
			filteredJobs = append(filteredJobs, jobCopy)
		}
		if len(filteredJobs) == 0 {
			continue
		}
		wfCopy := wf
		wfCopy.Jobs = filteredJobs
		result = append(result, wfCopy)
	}
	return result
}

func matchesJob(job provider.Job, patterns []Pattern) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		if pattern.Match(job.Name) || pattern.Match(job.RawID) {
			return true
		}
	}
	return false
}

func filterSteps(steps []provider.Step, onlyPatterns, skipPatterns []Pattern) []provider.Step {
	if len(steps) == 0 {
		return nil
	}
	result := make([]provider.Step, 0, len(steps))
	for _, step := range steps {
		if step.Run == "" {
			continue
		}
		if len(onlyPatterns) > 0 && !matchesStep(step, onlyPatterns) {
			continue
		}
		if len(skipPatterns) > 0 && matchesStep(step, skipPatterns) {
			continue
		}
		result = append(result, step)
	}
	return result
}

func matchesStep(step provider.Step, patterns []Pattern) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		if pattern.Match(step.Name) || pattern.Match(step.Run) {
			return true
		}
	}
	return false
}
