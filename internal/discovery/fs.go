package discovery

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ErrNoWorkflows indicates that no workflow files were found during discovery.
var ErrNoWorkflows = errors.New("no workflows discovered")

// Workflows returns workflow file paths. If explicit paths are provided they are
// validated and returned in the order given. Otherwise the default GitHub
// Actions workflow glob is used and results are sorted lexicographically.
func Workflows(root string, explicit []string) ([]string, error) {
	if len(explicit) > 0 {
		return resolveExplicit(root, explicit)
	}

	ymlGlob := filepath.Join(root, ".github", "workflows", "*.yml")
	yamlGlob := filepath.Join(root, ".github", "workflows", "*.yaml")

	matches := make(map[string]struct{})

	addMatches := func(pattern string) error {
		found, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("glob %q: %w", pattern, err)
		}
		for _, m := range found {
			matches[m] = struct{}{}
		}
		return nil
	}

	if err := addMatches(ymlGlob); err != nil {
		return nil, err
	}
	if err := addMatches(yamlGlob); err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return nil, ErrNoWorkflows
	}

	paths := make([]string, 0, len(matches))
	for p := range matches {
		rel := mustRelOrClean(root, p)
		paths = append(paths, rel)
	}
	sort.Strings(paths)

	return paths, nil
}

func resolveExplicit(root string, explicit []string) ([]string, error) {
	seen := make(map[string]struct{})
	resolved := make([]string, 0, len(explicit))
	for _, input := range explicit {
		cleaned := input
		if !filepath.IsAbs(cleaned) {
			cleaned = filepath.Join(root, cleaned)
		}
		info, err := os.Stat(cleaned)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("workflow %q not found", input)
			}
			return nil, fmt.Errorf("stat %q: %w", input, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("workflow %q is a directory", input)
		}
		rel := mustRelOrClean(root, cleaned)
		if _, ok := seen[rel]; ok {
			continue
		}
		seen[rel] = struct{}{}
		resolved = append(resolved, rel)
	}
	if len(resolved) == 0 {
		return nil, ErrNoWorkflows
	}
	return resolved, nil
}

func mustRelOrClean(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Clean(path)
	}
	rel = filepath.Clean(rel)
	if rel == "." || strings.HasPrefix(rel, "..") {
		return filepath.Clean(path)
	}
	return rel
}
