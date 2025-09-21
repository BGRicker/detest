IMPLEMENTATION_SPEC.md

Project: Detest

Goal:
Build a Go CLI that reads CI workflow files and runs the same run: steps locally with concise output. Start with GitHub Actions support; design for pluggable providers (GitLab, CircleCI) later.

⸻

High-Level Requirements
	•	Parse one or more GitHub Actions workflow files (.github/workflows/*.yml).
	•	Extract only shell steps (steps[].run) and execute them locally in order.
	•	Respect per-workflow/job/step env, shell, and working-directory.
	•	Ignore CI-only steps (uses: …, cache/artifact, checkout), but keep their metadata for potential future rules.
	•	Produce concise “pretty” output and optional JSON report.
	•	Provide filters for jobs/steps; support dry-run and list modes.
	•	Exit non-zero if any executed step fails.
	•	No container orchestration, service boot, matrix or parallelism in v1 (warn if encountered).

⸻

CLI Surface (v1)

Commands
	•	detest list — Discover workflows, resolve jobs/steps, print what would run.
	•	detest run — Execute resolved steps locally.

Global Flags
	•	--provider [auto|github] (default: auto)
	•	--workflow <path> (repeatable; default: discovery)
	•	--job <pattern> (repeatable; substring or /regex/)
	•	--only-step <pattern> (repeatable; substring or /regex/)
	•	--skip-step <pattern> (repeatable; substring or /regex/)
	•	--dry-run (print commands without executing)
	•	--verbose (stream full stdout/stderr)
	•	--format [pretty|json] (default: pretty)

Config File

Optional .detest.yml in repo root; merged with flags (flags win).

provider: github                  # auto|github
workflows:
  - .github/workflows/ci.yml
jobs:
  include: [lint, test]
  exclude: []
steps:
  only: []
  skip: ["Upload artifact"]
output:
  verbose: false
  format: pretty
warn:
  version_mismatch: true


⸻

Project Layout

detest/
├─ cmd/detest/main.go                # Cobra root command wiring
├─ internal/
│  ├─ discovery/fs.go                # find workflow files
│  ├─ provider/
│  │  ├─ provider.go                 # interfaces + shared models
│  │  └─ github/
│  │     ├─ parser.go                # GH Actions YAML -> model
│  │     └─ parser_test.go
│  ├─ runner/
│  │  ├─ exec.go                     # process exec, env, cwd, shell
│  │  └─ result.go                   # step/job/workflow results
│  ├─ output/
│  │  ├─ pretty.go                   # concise grouped rendering
│  │  └─ json.go                     # machine-readable report
│  ├─ config/config.go               # load/merge .detest.yml + flags
│  └─ version/warn.go                # .ruby-version / .node-version checks
├─ testdata/
│  ├─ workflows/ci_basic.yml
│  ├─ workflows/ci_envs.yml
│  ├─ workflows/ci_workdir.yml
│  └─ workflows/ci_services_matrix.yml
└─ go.mod


⸻

Core Types

// internal/provider/provider.go
type Pipeline struct {
    Provider  string
    Workflows []Workflow
}

type Workflow struct {
    Path string
    Name string
    Env  map[string]string
    Jobs []Job
    Defaults Defaults
}

type Defaults struct {
    RunShell         string
    WorkingDirectory string
}

type Job struct {
    Name     string
    RawID    string
    Env      map[string]string
    Steps    []Step
    Defaults Defaults
}

type Step struct {
    Name             string
    Run              string   // set if this is a shell step
    Uses             string   // set if this is an action step (ignored v1)
    Env              map[string]string
    Shell            string
    WorkingDirectory string
}


⸻

Discovery
	•	If --workflow not provided: glob .github/workflows/*.{yml,yaml}.
	•	Resolve to absolute paths; ensure deterministic ordering (lexicographic).

⸻

GitHub Actions Parser (v1)
	•	YAML decoding via gopkg.in/yaml.v3.
	•	Map fields:
	•	name, top-level env, defaults.run.shell, defaults.run.working-directory.
	•	jobs map (preserve order by iterating keys sorted).
	•	Each job.steps:
	•	If run present → create Step with Run.
	•	If uses present → set Uses (ignored by runner).
	•	Merge env: workflow → job → step (later overrides earlier).
	•	Carry shell, working-directory overrides if present.
	•	Warnings: detect services, strategy.matrix, and step/job if:; record on model for rendering a one-line note.
	•	Do not crash on unrecognized keys.

⸻

Filtering
	•	Job include/exclude applied to Workflow.Jobs by:
	•	substring match (case-insensitive), or
	•	regex when pattern is /…/.
	•	Step only/skip applied within each job using same match rules.
	•	list and run both reflect filters.

⸻

Execution
	•	Sequential execution by workflow → job → step.
	•	Skip steps with Uses (log only when --verbose).
	•	For each Run step:
	•	Resolve shell: step > job default > workflow default > OS default (bash -lc on *nix; cmd /C on Windows).
	•	Resolve cwd: step > job default > workflow default > repo root.
	•	Build env: process env + workflow env + job env + step env.
	•	Spawn process; stream output:
	•	--verbose: stream raw stdout/stderr with minimal prefixes.
	•	default: capture and only show last N lines on failure (configurable, default 40).
	•	Measure duration; collect exit code and tail of stderr.
	•	Stop on first failure? No — continue through job by default; summarize failures at end. Exit code non-zero if any failed.

⸻

Output

Pretty (default)
	•	Group by workflow → job.
	•	Line per step with ✓/✗, duration.
	•	On failures: show short excerpt (tail) and step name.
	•	End SUMMARY line with totals and overall duration.

JSON

Shape:

{
  "provider": "github",
  "workflows": [
    {
      "path": ".github/workflows/ci.yml",
      "name": "CI",
      "jobs": [
        {
          "name": "lint",
          "steps": [
            {"name": "yarn install", "status": "passed", "duration_ms": 2710},
            {"name": "bin/rails standard", "status": "passed", "duration_ms": 5060}
          ]
        }
      ]
    }
  ],
  "warnings": ["services declared but not executed"],
  "summary": {"passed": 9, "failed": 1, "duration_ms": 42123, "exit_code": 1}
}


⸻

Version Hints
	•	If .ruby-version exists:
	•	Read expected version; compare major.minor against ruby -v if present.
	•	Print non-fatal warning on mismatch when warn.version_mismatch: true.
	•	Same for .node-version via node -v.

⸻

Error Handling
	•	Missing workflows: exit 2 with clear message and discovery hint.
	•	YAML parse error: exit 2 with file/line context when available.
	•	Command exec error: record failure, continue to next step, final exit code 1.
	•	Unknown provider in config: exit 2.

⸻

Tests
	•	Unit tests:
	•	Parser: env inheritance, defaults, shell/cwd, non-run steps ignored, warning detection.
	•	Filters: job/step include, exclude, regex.
	•	Output: pretty and JSON snapshot/golden tests.
	•	Integration-lite:
	•	run --dry-run against fixtures (deterministic command list).
	•	Fake commands via sh -c 'echo ok' in fixtures to validate success/failure paths.
	•	CI for Detest itself (GitHub Actions):
	•	go build, go vet, golangci-lint, go test ./....

⸻

Non-Goals (v1)
	•	No orchestration for services (Postgres/Redis).
	•	No strategy.matrix expansion.
	•	No parallel jobs.
	•	No Docker requirement.

⸻

Roadmap (post-v1)
	•	Providers: GitLab CI (.gitlab-ci.yml), CircleCI (.circleci/config.yml).
	•	Matrix expansion and needs graph to retain ordering and optional parallelism.
	•	Optional service boot via Docker/Compose with minimal config.
	•	Watch mode (detest watch) to re-run on file changes.
	•	Reporters: JUnit, GitHub Annotations (via JSON).
	•	Plugin interface for custom step handlers.

⸻

Acceptance Criteria
	•	detest list prints workflows, jobs, and resolved run steps in deterministic order.
	•	detest run --dry-run prints the exact shell commands (with env/cwd) that would execute.
	•	detest run executes steps, honors env/shell/working-directory, and returns exit code 1 when any step fails.
	•	--job, --only-step, --skip-step filter correctly and are reflected in list and run.
	•	--format json outputs valid JSON matching the schema above.
	•	Parser logs warnings (but does not crash) for services, matrix, and if:.

⸻

Dependencies
	•	CLI/config: github.com/spf13/cobra, github.com/spf13/viper
	•	YAML: gopkg.in/yaml.v3
	•	(CI lint) github.com/golangci/golangci-lint/cmd/golangci-lint

⸻

Implementation Order
	1.	Scaffold Cobra CLI + config loader; implement discovery.
	2.	Implement GitHub Actions parser → internal model + tests.
	3.	list command (with filters) + golden tests.
	4.	run --dry-run printing resolved commands.
	5.	Exec runner with pretty output; exit codes.
	6.	JSON formatter.
	7.	Version-hint warnings.
	8.	Polish errors, warnings, docs, and CI for this repo.
