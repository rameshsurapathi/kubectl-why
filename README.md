# kubectl-why

**Turn Kubernetes failures into plain-English diagnosis.**

`kubectl-why` explains why a Pod, Deployment, or Job is failing by collecting the useful bits from status, events, and logs, then showing:

- **Why** it failed
- **Evidence** that supports the diagnosis
- **Fix** commands you can run next

It is built for:

- Kubernetes beginners who do not yet know what `OOMKilled`, `CrashLoopBackOff`, or `ImagePullBackOff` really mean
- Developers debugging workloads without jumping between multiple `kubectl` commands
- SRE / DevOps engineers who want fast, readable failure summaries

![kubectl-why demo](demo.gif)

---

## Why use it?

When a workload fails, Kubernetes usually gives you the answer, but it is spread across:

```bash
kubectl describe pod <name>
kubectl logs <name> --previous
kubectl get events
```

That is fine once you know what to look for. It is much harder when you are still learning Kubernetes or when the failure is buried in noisy output.

`kubectl-why` turns those raw signals into a short diagnosis you can understand quickly.

---

## Install

> **Requires Go 1.25+** when building from source.
> Pre-built binaries do not require Go.

```bash
# Homebrew (macOS / Linux)
brew tap rameshsurapathi/tap
brew install kubectl-why

# Go install
go install github.com/rameshsurapathi/kubectl-why@latest

# Download release binaries
# https://github.com/rameshsurapathi/kubectl-why/releases
```

Works as a standalone CLI or as a kubectl plugin:

```bash
kubectl-why pod api-123
kubectl why pod api-123
```

---

## Usage

```bash
# Pod diagnosis
kubectl-why pod <name> -n <namespace>

# Deployment & Rollout diagnosis
kubectl-why deployment <name> -n <namespace>
kubectl-why rollout deployment <name> -n <namespace>

# Job & CronJob diagnosis
kubectl-why job <name> -n <namespace>
kubectl-why cronjob <name> -n <namespace>

# Service & Storage diagnosis
kubectl-why service <name> -n <namespace>
kubectl-why pvc <name> -n <namespace>

# Node diagnosis
kubectl-why node <name>

# Namespace scan
kubectl-why scan -n <namespace>
kubectl-why scan -n <namespace> --show-healthy

# JSON output for automation
kubectl-why pod <name> -o json
kubectl-why scan -n <namespace> -o json
```

**Flags**

```text
-n, --namespace      Kubernetes namespace (default: default)
--context            Kubernetes context
--tail               Log lines to fetch (default: 20)
--events             Max Kubernetes events to show (default: 5)
-o, --output         Output format: text|json
--explain            Show AI/rule confidence and deep reasoning behind the diagnosis
--show-secondary     Show secondary warnings/findings (e.g. out of memory + failed mount)
--show-healthy       Include healthy resources in namespace scan output
```

---

## What it explains today

`kubectl-why` currently detects:

**Pod lifecycle:**
- `OOMKilled` and App crashes (`Exit Code X`)
- `ImagePullBackOff` / `ErrImagePull` (including auth failures)
- `CreateContainerConfigError` (Missing ConfigMaps/Secrets/ServiceAccounts)
- `CrashLoopBackOff`
- `Pending` (Scheduling failures: CPU/Memory, Taints, Affinities)
- `ContainerCannotRun` / `RunContainerError`
- `Evicted`
- Failing Readiness/Liveness/Startup Probes
- Init Container failures
- Volume Mount and Attach failures

**Workloads & Platform:**
- Stuck Deployment rollouts (`ProgressDeadlineExceeded`, Unavailable replicas)
- Failed or suspended CronJobs
- PVC provisioning and binding issues
- Service misconfigurations (no matching pods, endpoints failing)
- Node pressure and readiness issues

Deployment, Job, CronJob, Service, PVC, and Node support are included. The tool analyzes the most relevant failing components automatically.

**Namespace scan:**
- `kubectl-why scan -n <namespace>` checks common namespaced resources together
- Ranks critical issues before warnings
- Hides healthy resources by default, with `--show-healthy` for full inventory
- Emits a v3 JSON envelope with ranked resource diagnoses for automation

---

## Example diagnoses

**OOMKilled**

```text
Status        OOMKilled

Why
  Container exceeded its memory limit.
  The kernel killed it to protect the node.

Evidence
  Container      api
  Exit code      137
  Reason         OOMKilled
  Restarts       8
```

**ImagePullBackOff**

```text
Status        ImagePullBackOff

Why
  Kubernetes cannot pull the image.
  This is usually a wrong tag, deleted image, or auth failure.

Evidence
  Image         ghcr.io/example/api:doesntexist
  Error         pull access denied
```

**Pending**

```text
Status        Pending — cannot be scheduled

Why
  The scheduler cannot find a node that meets
  the pod's resource requests or constraints.

Evidence
  Scheduler Msg  0/3 nodes are available: 3 Insufficient memory
```

**Service Routing Failure**

```text
Status        Degraded

Why
  Service matches 3 pods, but none are ready.

Evidence
  Type           ClusterIP
  Selector       map[app:backend]
```

**PVC Pending**

```text
Status        Pending

Why
  The PVC will not bind until a Pod is created that uses it (WaitForFirstConsumer binding mode).

Evidence
  StorageClass   standard-rwo
  Message        waiting for first consumer to be created before binding
```

---

## How it works

The tool follows a simple flow:

1. **Collect** relevant Kubernetes signals from the API
2. **Analyze** the failure using focused rules
3. **Render** a short explanation with evidence and next steps

That makes it useful both as a fast debugging tool and as a way to learn what common Kubernetes failures actually mean.

---

## Roadmap
Planned next steps include expanding into deeper platform debugging.

- Path tracing (`kubectl-why path`) across ingress, service, and pod
- HPA and scaling diagnosis
- NetworkPolicy debugging
- Advanced provider-specific identity checks such as IRSA / workload identity

Roadmap items are directional, not fixed commitments.

---

## Contributing

Adding a new failure pattern is intentionally small and approachable. See [CONTRIBUTING.md](CONTRIBUTING.md) for fixtures, rule registration, tests, and development workflow.

Good next additions include:

- `PostStartHookError`
- `InvalidImageName`
- init-container failure improvements

---

## License

MIT
