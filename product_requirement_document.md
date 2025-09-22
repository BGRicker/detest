PRODUCT_REQUIREMENTS_DOCUMENT.md

Product: Testdrive

Run your CI’s tests locally, without the push.

⸻

1. Overview

Testdrive is a command-line tool that allows developers to execute the same test, lint, and audit steps defined in their CI/CD pipelines directly on their local machines.
It is written in Go, installable as a binary (eventually via Homebrew), and initially supports GitHub Actions workflows.

By parsing CI workflow definitions (.yml/.yaml), Testdrive ensures consistency between local and remote runs while eliminating the overhead of remembering commands or maintaining separate Makefiles.

⸻

2. Goals
	•	Feedback loop reduction: eliminate the “push → wait for CI” cycle.
	•	Single source of truth: CI config is the canonical definition of required checks.
	•	Noise reduction: concise, developer-friendly output that highlights only what matters.
	•	Multi-provider potential: start with GitHub Actions, but design to support GitLab CI and CircleCI.
	•	Zero infrastructure requirement: no Docker or VM emulation in v1; run commands directly on host.

⸻

3. Non-Goals (v1)
	•	No orchestration of CI services (e.g., Postgres/Redis containers).
	•	No matrix expansion (strategy.matrix).
	•	No parallel job execution.
	•	No 100% fidelity emulation of the CI runtime environment.

These may appear in the future roadmap.

⸻

4. User Stories
	1.	As a developer, I want to run all my project’s CI checks with one command (testdrive run) so I can catch issues locally before pushing.
	2.	As a developer, I want to see which jobs and steps will run (testdrive list) so I understand what CI expects.
	3.	As a developer, I want to selectively run only linting jobs (testdrive run --job lint) so I can get fast targeted feedback.
	4.	As a team lead, I want CI checks enforced consistently across contributors without needing to maintain duplicate Makefiles.
	5.	As a contributor, I want minimal, readable output that tells me what failed and why, without scrolling through verbose CI logs.
	6.	As a DevOps engineer, I want Detest to honor .ruby-version or .node-version so I can detect mismatched local environments before problems occur.

⸻

5. Key Features (v1)

5.1 Workflow Parsing
	•	Detect .github/workflows/*.yml automatically.
	•	Parse jobs, steps, run commands, env, defaults (shell, working directory).
	•	Ignore uses: steps; warn on unsupported features (services, matrix, if:).

5.2 Execution
	•	Run steps sequentially in local shell.
	•	Merge env (workflow → job → step → process env).
	•	Apply working directory rules.
	•	Capture stdout/stderr, duration, and exit codes.

5.3 Output
	•	Pretty mode (default): grouped by job, ✓/✗ markers, duration, summary line.
	•	JSON mode: machine-readable report (jobs, steps, status, durations, summary).
	•	Verbose mode: full streaming logs.

5.4 Filtering
	•	Run/list only specific jobs (--job) or steps (--only-step, --skip-step).
	•	Regex support for advanced filters.
	•	Dry run: print commands without executing.

5.5 Configuration
	•	Optional .testdrive.yml for overrides (provider, workflow paths, job/step filters, output).
	•	CLI flags override config.

5.6 Environment Warnings
	•	Compare .ruby-version vs local ruby -v.
	•	Compare .node-version vs local node -v.
	•	Warn if mismatched (non-fatal).

⸻

6. Future Features
	•	Multi-provider support: GitLab CI, CircleCI.
	•	Service orchestration: optional Docker/Compose integration for DBs and caches.
	•	Matrix expansion: run combinations of versions/environments.
	•	Parallelism: concurrent job execution.
	•	Reporters: JUnit XML, GitHub annotations.
	•	Watch mode: rerun jobs when files change.
	•	Plugin system: user-defined step handlers.

⸻

7. Competitive Landscape
	•	act: Runs GitHub Actions in Docker. Full emulation but heavy and provider-specific.
	•	Foreman/Overmind: Process managers, but not CI-aware.
	•	Make/Rake: Require duplication of CI logic.

Testdrive differentiators:
	•	Lighter, faster, no Docker dependency.
	•	CI config is the single source of truth.
	•	Friendly output designed for local iteration.
	•	Extensible to multiple providers.

⸻

8. Success Metrics
	•	Adoption: Number of GitHub repos integrating Testdrive into contribution docs.
	•	Developer efficiency: Reduction in “push → fail → fix → push again” cycles.
	•	Community engagement: GitHub stars, contributions, issues opened/closed.
	•	Reliability: Consistency of results between local Testdrive and remote CI runs.

⸻

9. Risks & Mitigations
	•	Environment drift (local vs CI container images).
	•	Mitigation: warn on runtime version mismatches, optional Docker mode in future.
	•	Unsupported CI features (services, matrix, conditions).
	•	Mitigation: explicit warnings, clear roadmap.
	•	Scope creep into full CI emulation.
	•	Mitigation: strict v1 boundaries; focus on test/lint steps.

⸻

10. Release Plan
	•	MVP (v1):
	•	GitHub Actions support.
	•	list, run, filters, dry run.
	•	Pretty + JSON output.
	•	Version mismatch warnings.
	•	Release Candidate (v1.1):
	•	Homebrew distribution.
	•	Expanded error handling and output polish.
	•	Future Releases:
	•	Multi-provider support, services, matrix/parallel, watch mode.

⸻

11. Open Questions
	1.	Should failed steps stop execution immediately or continue to run remaining steps? (Default: continue, configurable later.)
	2.	Should Testdrive auto-install missing runtimes (Ruby, Node) or only warn? (Default: warn.)
	3.	Should service helpers (Postgres, Redis) be part of core or plugins?

⸻

Do you want me to also generate a sample GitHub issue template (feature request/bug report) so contributors know how to engage with Testdrive from day one?
