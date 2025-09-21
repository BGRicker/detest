package version

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Info captures a language version installed on the system.
type Info struct {
	Name    string
	Version string
}

var (
	rubyRegex = regexp.MustCompile(`(?i)ruby\s+(\d+\.\d+(?:\.\d+)?)`)
	nodeRegex = regexp.MustCompile(`(?i)v?(\d+\.\d+(?:\.\d+)?)`)
)

// DetectRuby returns the system Ruby version by calling `ruby -v`.
func DetectRuby() (Info, error) {
	out, err := runCommand("ruby", "-v")
	if err != nil {
		return Info{}, err
	}
	match := rubyRegex.FindStringSubmatch(out)
	if len(match) < 2 {
		return Info{}, fmt.Errorf("unable to parse ruby version from %q", out)
	}
	return Info{Name: "ruby", Version: match[1]}, nil
}

// DetectNode returns the system Node.js version by calling `node -v`.
func DetectNode() (Info, error) {
	out, err := runCommand("node", "-v")
	if err != nil {
		return Info{}, err
	}
	match := nodeRegex.FindStringSubmatch(out)
	if len(match) < 2 {
		return Info{}, fmt.Errorf("unable to parse node version from %q", out)
	}
	return Info{Name: "node", Version: match[1]}, nil
}

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = nil
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

// CompareMajorMinor compares major.minor portions of two semver-like versions.
func CompareMajorMinor(desired, actual string) bool {
	d := semverPrefix(desired)
	a := semverPrefix(actual)
	if d == "" || a == "" {
		return false
	}
	return strings.EqualFold(d, a)
}

func semverPrefix(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return ""
	}
	return fmt.Sprintf("%s.%s", parts[0], parts[1])
}

// Missing reports whether executing the command returns a not-found error.
func Missing(cmdErr error) bool {
	return errors.Is(cmdErr, exec.ErrNotFound)
}
