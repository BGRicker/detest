package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bgricker/detest/internal/config"
	"github.com/bgricker/detest/internal/discovery"
	"github.com/bgricker/detest/internal/provider"
	"github.com/bgricker/detest/internal/provider/filter"
	githubprovider "github.com/bgricker/detest/internal/provider/github"
	"github.com/bgricker/detest/internal/version"
)

// pipelineData bundles parsed workflows with warnings and metadata.
type pipelineData struct {
	provider  string
	workflows []provider.Workflow
	warnings  []provider.Warning
}

func loadPipeline(root string, cfg config.Config) (pipelineData, error) {
	providerName, err := resolveProvider(cfg.Provider)
	if err != nil {
		return pipelineData{}, err
	}

	var paths []string
	if len(cfg.Workflows) > 0 {
		paths, err = discovery.Workflows(root, cfg.Workflows)
	} else {
		paths, err = discovery.Workflows(root, nil)
	}
	if err != nil {
		if errors.Is(err, discovery.ErrNoWorkflows) {
			return pipelineData{}, fmt.Errorf("no workflows found; specify --workflow to provide files")
		}
		return pipelineData{}, err
	}

	switch providerName {
	case config.ProviderGitHub:
		parser := githubprovider.NewParser(root)
		pipeline, err := parser.Parse(paths)
		if err != nil {
			return pipelineData{}, err
		}
		versionWarnings := detectVersionWarnings(root, cfg)
		warnings := append(pipeline.Warnings, versionWarnings...)
		return pipelineData{provider: providerName, workflows: pipeline.Workflows, warnings: warnings}, nil
	default:
		return pipelineData{}, fmt.Errorf("provider %q not implemented", providerName)
	}
}

func applyFilters(data pipelineData, cfg config.Config) (pipelineData, error) {
	jobPatterns, err := filter.Compile(cfg.Jobs)
	if err != nil {
		return pipelineData{}, err
	}
	onlyPatterns, err := filter.Compile(cfg.OnlySteps)
	if err != nil {
		return pipelineData{}, err
	}
	skipPatterns, err := filter.Compile(cfg.SkipSteps)
	if err != nil {
		return pipelineData{}, err
	}

	filtered := filter.FilterWorkflows(data.workflows, jobPatterns, onlyPatterns, skipPatterns)
	return pipelineData{provider: data.provider, workflows: filtered, warnings: data.warnings}, nil
}

func detectVersionWarnings(root string, cfg config.Config) []provider.Warning {
	if !cfg.Warn.VersionMismatch {
		return nil
	}

	var warnings []provider.Warning

	rubyPath := filepath.Join(root, ".ruby-version")
	if contents, err := os.ReadFile(rubyPath); err == nil {
		required := strings.TrimSpace(string(contents))
		if required != "" {
			info, detectErr := version.DetectRuby()
			warn := buildVersionWarning("ruby", required, info.Version, detectErr)
			if warn != "" {
				warnings = append(warnings, provider.Warning{Workflow: ".ruby-version", Message: warn})
			}
		}
	}

	nodePath := filepath.Join(root, ".node-version")
	if contents, err := os.ReadFile(nodePath); err == nil {
		required := strings.TrimSpace(string(contents))
		if required != "" {
			info, detectErr := version.DetectNode()
			warn := buildVersionWarning("node", required, info.Version, detectErr)
			if warn != "" {
				warnings = append(warnings, provider.Warning{Workflow: ".node-version", Message: warn})
			}
		}
	}

	return warnings
}

func buildVersionWarning(name, required, actual string, detectErr error) string {
	if detectErr != nil {
		if version.Missing(detectErr) {
			return fmt.Sprintf("%s executable not found; required %s", name, required)
		}
		return fmt.Sprintf("unable to detect %s version: %v", name, detectErr)
	}
	if !version.CompareMajorMinor(required, actual) {
		return fmt.Sprintf("%s version mismatch: required %s (from .%s-version) but found %s", name, required, name, actual)
	}
	return ""
}
