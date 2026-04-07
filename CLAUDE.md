# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o kubectl-why .

# Run
./kubectl-why pod <pod-name> -n <namespace>
./kubectl-why deployment <deploy-name> -n <namespace>
./kubectl-why job <job-name> -n <namespace>

# Run tests
go test ./...

# Run a single test
go test ./pkg/analyzer/... -run TestOOMKilledRule

# Lint (requires golangci-lint)
golangci-lint run

# Tidy dependencies
go mod tidy
```

## Architecture

`kubectl-why` is a kubectl plugin that diagnoses failing Kubernetes resources. The flow through the system is always: **Collect → Analyze → Render**.

### Three-package design

**`pkg/kube/`** — data collection only. Never analyzes, never renders.
- `client.go`: builds a `kubernetes.Clientset` from `~/.kube/config` (respects `--context`)
- `signals.go`: defines the `PodSignals` struct — the canonical data transfer object between kube and analyzer
- `pods.go`, `deployments.go`, `jobs.go`: each exposes a `Collect*Signals(client, name, namespace, ...)` function that fetches all relevant API objects (pod spec, container statuses, events, logs) and returns the typed signals struct
- `events.go`, `logs.go`, `debug.go`: helpers for the collectors

**`pkg/analyzer/`** — pure logic, no API calls.
- `result.go`: defines `AnalysisResult` — the single output type for all resources and all rules
- `rules.go`: defines the `Rule` interface (`Name()`, `Match(*PodSignals)`, `Analyze(*PodSignals)`) and the global `registry []Rule`. **Rule order in the registry is priority order** — the first matching rule wins.
- `rule_*.go`: one file per failure mode (OOM, crashloop, image pull, eviction, scheduling, config errors, etc.)
- `pod.go`, `deployment.go`, `job.go`: the `Analyze*()` entry points; they loop through the registry and call the first matching rule. Deployment and Job analysis reuses `AnalyzePod` internally.

**`pkg/render/`** — output only.
- `text.go`: colored terminal output via `charmbracelet/lipgloss`; severity (`"critical"`, `"warning"`, `"healthy"`) drives color selection
- `json.go`: JSON output for `--output json`

**`cmd/`** — wires flags to the three packages.
- `root.go`: global persistent flags (`--namespace`, `--context`, `--output`, `--events`, `--tail`) available on all subcommands
- `pod.go`, `deployment.go`, `job.go`: each calls `kube.Collect*Signals → analyzer.Analyze* → render.Text/JSON`

### Adding a new rule

1. Create `pkg/analyzer/rule_<name>.go` implementing `Rule` (three methods).
2. Register it in `pkg/analyzer/rules.go` `init()` at the correct priority position. More specific failure patterns should appear before generic catch-alls (e.g., `OOMKilledRule` before `CrashLoopRule`).

### Adding a new resource type

1. Add a `Collect*Signals` function in `pkg/kube/` and a signals struct in `signals.go`.
2. Add an `Analyze*` function in `pkg/analyzer/` (can delegate to `AnalyzePod` for pod-backed resources).
3. Add a cobra subcommand in `cmd/`.

### Key data types

- `kube.PodSignals` — the single source of truth passed from collection to analysis; rules only read this, never call the API
- `analyzer.AnalysisResult` — everything a renderer needs: `Status`, `Severity`, `Summary []string`, `Evidence []Evidence`, `FixCommands []FixCommand`, `NextChecks []string`, `RecentLogs`
- `analyzer.Evidence.Bar *ProgressBar` — optional; causes `render/text.go` to draw an ASCII progress bar (used for memory/CPU limits)
