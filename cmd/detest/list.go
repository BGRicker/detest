package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bgricker/detest/internal/config"
	"github.com/bgricker/detest/internal/output"
	"github.com/bgricker/detest/internal/provider"
	"github.com/bgricker/detest/internal/report"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List workflow jobs and steps",
		RunE:  runList,
	}
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, root, err := loadConfig(cmd)
	if err != nil {
		return err
	}

	data, err := loadPipeline(root, cfg)
	if err != nil {
		return err
	}

	filtered, err := applyFilters(data, cfg)
	if err != nil {
		return err
	}

	return renderList(cmd, cfg, data.provider, filtered.workflows, filtered.warnings)
}

func renderList(cmd *cobra.Command, cfg config.Config, providerName string, workflows []provider.Workflow, warnings []provider.Warning) error {
	if len(workflows) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No matching jobs or steps")
		return nil
	}

	warningsList := collapseWarnings(warnings)

	switch strings.ToLower(cfg.Format) {
	case config.FormatPretty:
		renderer := output.NewPretty(cmd.OutOrStdout())
		if err := renderer.RenderList(workflows); err != nil {
			return err
		}
	case config.FormatJSON:
		report := output.Report{
			Provider:  providerName,
			Workflows: workflows,
			Summary:   computeListSummary(workflows),
			Warnings:  warningsList,
		}
		renderer := output.NewJSON(cmd.OutOrStdout())
		if err := renderer.Render(report); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported format %q", cfg.Format)
	}

	if len(warningsList) > 0 && cfg.Format == config.FormatPretty {
		for _, msg := range warningsList {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", msg)
		}
	}

	return nil
}

func computeListSummary(workflows []provider.Workflow) report.Summary {
	var jobs, steps int
	for _, wf := range workflows {
		jobs += len(wf.Jobs)
		for _, job := range wf.Jobs {
			steps += len(job.Steps)
		}
	}
	return report.Summary{
		TotalWorkflows: len(workflows),
		TotalJobs:      jobs,
		TotalSteps:     steps,
		Duration:       0,
		DurationMS:     0,
		ExitCode:       0,
	}
}

func collapseWarnings(warnings []provider.Warning) []string {
	if len(warnings) == 0 {
		return nil
	}
	out := make([]string, 0, len(warnings))
	for _, w := range warnings {
		out = append(out, fmt.Sprintf("%s:%s: %s", w.Workflow, w.Job, w.Message))
	}
	return out
}

func loadConfig(cmd *cobra.Command) (config.Config, string, error) {
	root, err := os.Getwd()
	if err != nil {
		return config.Config{}, "", fmt.Errorf("determine working directory: %w", err)
	}

	cfg, err := config.Load(root)
	if err != nil {
		return config.Config{}, "", err
	}

	flags, err := gatherFlags(cmd)
	if err != nil {
		return config.Config{}, "", err
	}
	config.ApplyFlags(&cfg, flags)

	return cfg, root, nil
}

func resolveProvider(input string) (string, error) {
	if input == "" || input == config.ProviderAuto {
		return config.ProviderGitHub, nil
	}
	switch input {
	case config.ProviderGitHub:
		return input, nil
	default:
		return "", fmt.Errorf("unsupported provider %q", input)
	}
}
