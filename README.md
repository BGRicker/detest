# Testdrive

Testdrive mirrors your GitHub Actions `run:` steps locally so you can catch failures before pushing. It discovers workflows, respects job and step filters, and outputs either a concise pretty report or machine-friendly JSON.

## Install

```bash
go install github.com/bgricker/testdrive/cmd/testdrive@latest
```

## Usage

```bash
# List the jobs and steps that would run
$ testdrive list

# Execute all run: steps sequentially
$ testdrive run

# Preview commands without executing
$ testdrive run --dry-run

# Filter by job/steps and switch formats
$ testdrive run --job test --only-step "Lint" --format json

# Stream command output as it runs
$ testdrive run --verbose

# Allow privileged commands (e.g., sudo/apt-get) when absolutely necessary
$ TESTDRIVE_ALLOW_PRIVILEGED=1 testdrive run
```

### Streaming UI (GitHub-style)

When format is `pretty` (default) and not in verbose mode, Testdrive renders a live, GitHub-style summary:

- ✅/❌ per job with individual timers
- 🟢 while a job is running, ⏳ when queued
- Failed jobs expand to show step breakdown, durations, the exact `Command:` run, and cleaned failure output (including parsed RSpec failures)
- Routine CI noise is suppressed in streaming mode to keep output focused

Example:

```
✅ lint (3.2s)
✅ scan_js (1.9s)
❌ test (31.4s)
    ⏭️ Install packages (0s)
    ✅ Install modules (247ms)
    ❌ Run tests (31.1s)
      Command: bin/rails db:setup spec
      spec/jobs/foo_spec.rb:123 expected X got Y
```

Flags such as `--workflow`, `--job`, `--only-step`, and `--skip-step` accept multiple values and support substring or `/regex/` matches. When no workflows are provided, Testdrive automatically loads `.github/workflows/*.yml`/`*.yaml` in lexicographic order. Execution stops with a non-zero exit code if any step fails, but all remaining steps continue to run so you see the full picture.

## Environment Support

Testdrive automatically inherits your shell environment and supports version managers:

- **asdf**: Automatically sources `asdf.sh` (or `asdf.fish` for fish shell) to ensure correct Ruby, Node, Python versions
- **rbenv**: Works with your existing rbenv setup
- **Shell compatibility**: Supports bash, zsh, ksh, sh, and fish shells
- **Environment variables**: Merges workflow → job → step environment variables
- **Working directories**: Respects `working-directory` settings from workflows

## Configuration

An optional `.testdrive.yml` can provide defaults for the CLI. Command-line flags always win over config values.

```yaml
provider: github          # auto|github (defaults to auto)
workflows:
  - .github/workflows/ci.yml
jobs:
  - test
only_step:
  - /lint/
skip_step:
  - "Upload artifact"
dry_run: false
verbose: false
format: pretty             # pretty|json
warn:
  version_mismatch: true   # warn when local Ruby/Node major.minor differs
privileged_command_patterns:
  - (?i)^sudo\b
  - (?i)\bapt-get\b
```

## Current Status

- ✅ GitHub Actions workflow parser (run steps only)
- ✅ Sequential execution with env/shell/working-directory resolution
- ✅ Pretty & JSON reporters; streaming GitHub-style UI with live timers
- ✅ Dry-run, verbose streaming, job/step filters, repeatable `--workflow`
- ✅ Environment inheritance with asdf/rbenv support
- ✅ Cross-shell compatibility (bash, zsh, ksh, sh, fish)
- ✅ Privileged command detection and skipping
- 🚧 Upcoming: richer runtime pre-flight checks, additional CI providers, matrix & services support
  - Version mismatch warnings are enabled by default; set `warn.version_mismatch: false` to silence them.

Want to dig in? Run `go test ./...` to exercise the parser, runner, and CLI tests.
