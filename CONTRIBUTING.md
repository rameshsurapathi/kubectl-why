# Contributing to kubectl-why

Thank you for your interest in contributing.
The most impactful contributions are new failure
pattern rules — no deep Go knowledge needed.

## How to add a new failure pattern

Adding a new rule is ~20 lines of Go.

### 1. Add a fixture

Get a real pod in the failure state you want to support,
then save its JSON:

```bash
kubectl get pod <failing-pod> -n <namespace> -o json \
  > pkg/analyzer/testdata/<failure-name>_pod.json
```

> **No cluster?** Synthetic fixtures are acceptable for rule unit tests.
> Craft a minimal JSON with just the fields your rule reads
> (e.g. `status.phase`, `status.containerStatuses[].state`).
> See `pkg/analyzer/testdata/appcrash_pod.json` as an example.

### 2. Create a rule file `pkg/analyzer/rule_<name>.go`

Each rule lives in its own file. Create
`pkg/analyzer/rule_<name>.go` and implement two methods:

```go
package analyzer

import "github.com/rameshsurapathi/kubectl-why/pkg/kube"

type MyRule struct{}

func (r *MyRule) Name() string { return "MyRule" }

// Match returns true if this rule applies to the given pod signals
func (r *MyRule) Match(s *kube.PodSignals) bool {
    for _, c := range s.Containers {
        if c.State.IsWaiting &&
            c.State.WaitingReason == "MyReason" {
            return true
        }
    }
    return false
}

// Analyze returns the full diagnosis
func (r *MyRule) Analyze(
    s *kube.PodSignals,
) AnalysisResult {
    return AnalysisResult{
        Resource:      "pod/" + s.PodName,
        Namespace:     s.Namespace,
        Status:        "MyFailureStatus",
        PrimaryReason: "Human-readable reason",
        Severity:      "critical",
        Summary: []string{
            "What this means in plain English.",
        },
        Evidence: []Evidence{
            {Label: "Reason", Value: "..."},
        },
        FixCommands: []FixCommand{
            {
                Description: "How to fix it",
                Command:     "kubectl ...",
            },
        },
    }
}
```

Then register it in `pkg/analyzer/rules.go` by adding it to the
`registry` slice inside `init()`. Order matters — first match wins:

```go
func init() {
    registry = []Rule{
        &EvictedRule{},
        &OOMKilledRule{},
        // ... existing rules ...
        &MyRule{},   // ← add yours here, position = priority
    }
}
```

> **Note:** Rules use pointer receivers (`*MyRule`, not `MyRule`).
> The registry stores `Rule` interface values as pointers (`&MyRule{}`).
> Using value receivers will cause subtle interface-satisfaction bugs.

### 3. Write a test

```go
func TestAnalyze_MyFailure(t *testing.T) {
    signals := loadPodFixture(t, "myfailure_pod.json")
    result := analyzer.AnalyzePod(signals)

    assert.Equal(t, "MyFailureStatus", result.Status)
    assert.Equal(t, "critical", result.Severity)
    assert.NotEmpty(t, result.FixCommands)
}
```

### 4. Run tests

```bash
go test ./... 
```

### 5. Open a PR

Title format: `feat: add <FailureType> rule`

Include in your PR description:
- What Kubernetes failure this covers
- How to reproduce it
- Screenshot of the output

## Current failure patterns

| Pattern | Rule file | Status |
|---|---|---|
| OOMKilled | `rule_memory.go` | ✅ |
| ImagePullBackOff / ErrImagePull | `rule_image.go` | ✅ |
| CreateContainerConfigError | `rule_config.go` | ✅ |
| CrashLoopBackOff | `rule_crashloop.go` | ✅ |
| AppCrash (exit code 1) | `rule_crash.go` | ✅ |
| Segfault (exit code 139) | `rule_crash.go` | ✅ |
| Pending (scheduling) | `rule_scheduling.go` | ✅ |
| ContainerCannotRun | `rule_config.go` | ✅ |
| Evicted | `rule_evicted.go` | ✅ |

## Wanted contributions

These failure patterns are not yet supported.
Good first issues:

- [ ] `PostStartHookError` — hook failed after start
- [ ] `PreStopHookError` — hook failed before stop  
- [ ] `InvalidImageName` — malformed image reference
- [ ] `CreateContainerError` — generic container create fail
- [ ] `OOMKilled on init container` — init OOM
- [ ] Node-level analysis (`kubectl-why node`)

## Development setup

**Prerequisites:**
- **Go 1.25+** — required by `k8s.io/client-go v0.35`. Run `go version` to check.
  Install the latest Go from [go.dev/dl](https://go.dev/dl/).
- A Kubernetes cluster for capturing real fixtures (optional — see below).

```bash
git clone https://github.com/rameshsurapathi/kubectl-why
cd kubectl-why
go mod download
go build -o kubectl-why .
go test ./...
```

[kind](https://kind.sigs.k8s.io/) works well for a local cluster:

```bash
brew install kind
kind create cluster
```

## Code style

- Run `go fmt ./...` before committing
- Run `go vet ./...` to catch common issues
- Keep rules focused — one rule per failure type
- Evidence values should be plain text, not Go internals
- Fix commands should be copy-pasteable as-is
