package main

import (
	"errors"
	"fmt"

	"github.com/benricker/detest/internal/config"
	"github.com/benricker/detest/internal/discovery"
	"github.com/benricker/detest/internal/provider"
	"github.com/benricker/detest/internal/provider/filter"
	githubprovider "github.com/benricker/detest/internal/provider/github"
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
		return pipelineData{provider: providerName, workflows: pipeline.Workflows, warnings: pipeline.Warnings}, nil
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
