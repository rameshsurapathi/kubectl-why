# kubectl-why

**Explain why a Kubernetes pod, deployment, or job 
is failing — without switching between 5 commands.**

![kubectl-why demo](demo.gif)

---

## The problem

It's 3am. PagerDuty woke you up.
Your pod is in `CrashLoopBackOff`.

You run:

```bash
kubectl describe pod api-123
kubectl logs api-123 --previous
kubectl get events --field-selector \
  involvedObject.name=api-123
```

You get walls of text. You grep through them 
half-asleep. The answer was buried in the events 
section the whole time.

**There's a better way.**

---

## Install

> **Requires Go 1.25+** when building from source (driven by `k8s.io/client-go v0.35`).
> Pre-built binaries have no Go dependency.

```bash
# Homebrew (macOS / Linux)
brew tap rameshsurapathi/tap
brew install kubectl-why

# Go install  (requires Go 1.25+)
go install github.com/rameshsurapathi/kubectl-why@latest

# Download binary  (no Go needed)
# → github.com/rameshsurapathi/kubectl-why/releases
```

Works as a standalone CLI or as a kubectl plugin:

```bash
kubectl-why pod api-123          # standalone
kubectl why pod api-123          # as kubectl plugin
```

---

## Usage

```bash
# Diagnose a pod
kubectl-why pod <name> -n <namespace>

# Diagnose a deployment
kubectl-why deployment <name> -n <namespace>
kubectl-why deploy <name> -n <namespace>    # alias

# Diagnose a job
kubectl-why job <name> -n <namespace>

# Planned: explain a Kubernetes error or exit code
# kubectl-why explain 137
# kubectl-why explain OOMKilled
```

**Flags:**

```
-n, --namespace   Kubernetes namespace (default: default)
--context         Kubernetes context
--tail            Log lines to fetch (default: 20)
--events          Max Kubernetes events to show (default: 5)
-o, --output      Output format: text|json
```

---

## What it detects

| Failure | What you see | What kubectl-why tells you |
|---|---|---|
| OOMKilled | CrashLoopBackOff | Memory limit exceeded, usage bar, fix command |
| ImagePullBackOff | ImagePullBackOff | Bad tag or auth issue, exact image name |
| ErrImagePull | ErrImagePull | First pull failure before backoff |
| CreateContainerConfigError | Error | Missing ConfigMap or Secret name |
| CrashLoopBackOff | CrashLoopBackOff | Exit code, last logs, restart count |
| Pending | Pending | Insufficient CPU/memory, node details |
| ContainerCannotRun | Error | Bad entrypoint or missing binary |
| Evicted | Failed | Node resource pressure, exact eviction message, node name |
| Exit Code 139 (Segfault) | CrashLoopBackOff | Segmentation fault detected, restart count |
| Exit Code 1 (App Crash) | CrashLoopBackOff | Smart log scan for exceptions/panics, extracted error lines |
| Healthy | Running | Pod health summary |

Deployment and job support built in — analyzes
the worst failing pod automatically.

---

## Example output

**OOMKilled pod:**

```
╭─ pod/api-123 · production ──────────────────╮

  Status        OOMKilled

  ●  Why
    Container exceeded its memory limit.
    The kernel killed it to protect the node.

  ●  Evidence
    Node           ip-10-0-12-34
    Container      api
    Exit code       137
    Reason          OOMKilled
    Restarts        8

  ●  Last logs
    java.lang.OutOfMemoryError: Java heap space

  ●  Fix
    # Increase memory limit
    kubectl set resources deployment/api \
      --limits=memory=1Gi -n production
```

---

## Contributing

Adding a new failure pattern is ~20 lines of Go.
See [CONTRIBUTING.md](CONTRIBUTING.md) for a 
step-by-step guide.

**Wanted:** rules for `PostStartHookError`, 
`InvalidImageName`, node-level analysis.

---

## Roadmap

Planned next steps are split into two layers: deepen
workload debugging first, then expand into platform
debugging.

### V2

- More pod and init-container failure patterns
- Better job diagnosis (`BackoffLimitExceeded`, clearer evidence, stronger fixes)
- Service diagnosis (`kubectl-why service`) for selector mismatch and no endpoints
- `--all-namespaces` support
- `kubectl-why explain <exit-code|reason>` for beginner-friendly error lookup

### V3

- Path tracing (`kubectl-why path`) across ingress, service, and pod
- HPA and scaling diagnosis
- NetworkPolicy debugging
- Node-level diagnosis
- Provider-specific identity checks such as IRSA / workload identity

Roadmap items are directional, not fixed commitments.

---

## Why this exists

I got tired of running the same 4 kubectl commands 
every time a pod failed in production.
The information was always there — it just took 
too long to find.

kubectl-why collects it all in one command 
and tells you what it means.

---

## License

MIT
