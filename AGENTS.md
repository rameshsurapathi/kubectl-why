# Repository Guidelines

## Project Structure & Module Organization

`kubectl-why` is a Go CLI and kubectl plugin organized around `Collect -> Analyze -> Render`.

- `main.go` boots the CLI.
- `cmd/` contains Cobra commands and shared flags for `pod`, `deployment`, and `job`.
- `pkg/kube/` collects Kubernetes signals only.
- `pkg/analyzer/` contains pure diagnosis logic. Keep one failure rule per `rule_*.go` file and register it in `pkg/analyzer/rules.go`.
- `pkg/render/` formats text and JSON output.
- `pkg/analyzer/testdata/` stores real pod JSON fixtures used by tests.

## Build, Test, and Development Commands

- `go build -o kubectl-why .` builds the local binary.
- `./kubectl-why pod <name> -n <namespace>` runs the CLI against a pod.
- `go test ./...` runs the full test suite.
- `go test ./pkg/analyzer/... -run TestAnalyze_` runs analyzer-focused tests.
- `go fmt ./...` formats Go source.
- `go vet ./...` checks for common mistakes.
- `golangci-lint run` runs linting when `golangci-lint` is installed.
- `go mod tidy` cleans up module dependencies.

## Coding Style & Naming Conventions

Use standard Go formatting with tabs and run `go fmt` before committing. Keep packages narrowly scoped: `pkg/kube` must not analyze, `pkg/analyzer` must not call the API, and `pkg/render` must not contain business logic. Name new rules `rule_<failure>.go`, fixtures `<failure>_pod.json`, and tests `TestAnalyze_<Failure>` or similarly specific names.

## Testing Guidelines

Tests use Go’s `testing` package plus `stretchr/testify`. Add or update fixtures in `pkg/analyzer/testdata/` for new failure modes, then assert status, severity, and fix commands in `pkg/analyzer/*_test.go`. Prefer focused fixture-backed tests over live cluster tests. Run `go test ./...` before opening a PR.

## Commit & Pull Request Guidelines

This repository currently has no shared Git history, so follow the documented contribution style in `CONTRIBUTING.md`: use concise, imperative commit subjects and prefer Conventional Commit prefixes such as `feat:` or `fix:`. For rule additions, use `feat: add <FailureType> rule`. PRs should explain the failure covered, how to reproduce it, and include sample output or a screenshot when terminal output changes.

## Security & Configuration Tips

Do not commit kubeconfig, cluster credentials, or sanitized fixtures that still contain secrets. When creating fixtures, inspect pod JSON before adding it under `pkg/analyzer/testdata/`.
