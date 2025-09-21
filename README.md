# Detest

Detest mirrors your GitHub Actions `run:` steps locally so you can catch failures before pushing. It discovers workflows, respects job and step filters, and outputs either a concise pretty report or machine-friendly JSON.

## Install

```bash
go install github.com/bgricker/detest/cmd/detest@latest
```

## Usage

```bash
# List the jobs and steps that would run
$ detest list

# Execute all run: steps sequentially
$ detest run

# Preview commands without executing
$ detest run --dry-run

# Filter by job/steps and switch formats
$ detest run --job test --only-step "Lint" --format json

# Stream command output as it runs
$ detest run --verbose
```

Flags such as `--workflow`, `--job`, `--only-step`, and `--skip-step` accept multiple values and support substring or `/regex/` matches. When no workflows are provided, Detest automatically loads `.github/workflows/*.yml`/`*.yaml` in lexicographic order. Execution stops with a non-zero exit code if any step fails, but all remaining steps continue to run so you see the full picture.

## Configuration

An optional `.detest.yml` can provide defaults for the CLI. Command-line flags always win over config values.

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
  version_mismatch: false  # reserved for future version checks
```

## Current Status

- ✅ GitHub Actions workflow parser (run steps only)
- ✅ Sequential execution with env/shell/working-directory resolution
- ✅ Pretty & JSON reporters with per-step duration/excerpts
- ✅ Dry-run, verbose streaming, job/step filters, repeatable `--workflow`
- 🚧 Upcoming: cross-language version warnings, additional CI providers, matrix & services support

Want to dig in? Run `go test ./...` to exercise the parser, runner, and CLI tests.
