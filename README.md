# kubectl-why

**Turn Kubernetes failures into plain-English diagnosis.**

`kubectl-why` explains why a Pod, Deployment, Job, Service, or any workload is failing by collecting the useful bits from status, events, and logs, then showing:

- **Why** it failed — with human-readable reasoning
- **Evidence** that supports the diagnosis — with JSON-path provenance
- **Fix** commands you can run next — tagged with safety levels

It is built for:

- Kubernetes beginners who do not yet know what `OOMKilled`, `CrashLoopBackOff`, or `ImagePullBackOff` really mean
- Developers debugging workloads without jumping between multiple `kubectl` commands
- SRE / DevOps engineers who want fast, readable failure summaries
- Platform teams building automation on top of structured JSON output

![kubectl-why demo](demo.gif)

---

## Why use it?

When a workload fails, Kubernetes usually gives you the answer, but it is spread across:

```bash
kubectl describe pod <name>
kubectl logs <name> --previous
kubectl get events
kubectl describe node <node>
kubectl get endpointslices
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

### Resource diagnosis (v1 + v2)

```bash
# Pod diagnosis
kubectl-why pod <name> -n <namespace>
kubectl-why pod <name> -n <namespace> --explain

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
```

### Namespace scan (v3)

```bash
# Scan all resources in a namespace
kubectl-why scan -n <namespace>

# Include healthy resources
kubectl-why scan -n <namespace> --show-healthy

# JSON output for automation
kubectl-why scan -n <namespace> -o json
```

### Relationship tracing (v3)

```bash
# Trace Ingress -> Services -> EndpointSlices -> Pods
kubectl-why trace ingress <name> -n <namespace>
kubectl-why trace ingress <name> -n <namespace> --show-healthy

# Trace Service -> EndpointSlices -> Pods
kubectl-why trace service <name> -n <namespace>
kubectl-why trace service <name> -n <namespace> --show-healthy

# Trace Deployment -> ReplicaSets -> Pods -> Dependencies (PVCs, ConfigMaps, Secrets)
kubectl-why trace deployment <name> -n <namespace>
kubectl-why trace deployment <name> -n <namespace> --show-healthy
```

### JSON output

```bash
# Any resource supports JSON output
kubectl-why pod <name> -o json
kubectl-why deployment <name> -o json
kubectl-why scan -n <namespace> -o json
kubectl-why trace service <name> -n <namespace> -o json
kubectl-why trace ingress <name> -n <namespace> -o json
```

### Flags

```text
-n, --namespace      Kubernetes namespace (default: default)
--context            Kubernetes context
--tail               Log lines to fetch (default: 20)
--events             Max Kubernetes events to show (default: 5)
-o, --output         Output format: text|json
--explain            Show deep reasoning, evidence provenance, and confidence levels
--show-secondary     Show secondary warnings/findings (e.g. out of memory + failed mount)
--show-healthy       Include healthy resources in scan/trace output
```

---

## What it explains today

`kubectl-why` currently detects:

### Pod lifecycle (11 rules)
- `OOMKilled` — Container killed by kernel for exceeding memory limit
- `CrashLoopBackOff` — Container repeatedly crashing and restarting
- `ImagePullBackOff` / `ErrImagePull` — Wrong tag, deleted image, or auth failure
- `CreateContainerConfigError` — Missing ConfigMaps, Secrets, or ServiceAccounts
- `Pending` — Scheduling failures from insufficient CPU/Memory, taints, or affinities
- `Evicted` — Node-level resource pressure forced pod removal
- `ContainerCannotRun` / `RunContainerError` — Application crash on startup
- Failing Readiness / Liveness / Startup Probes
- Init Container failures blocking pod startup
- Volume Mount and Attach failures
- Non-zero exit codes with signal interpretation

### Workloads & Platform (v2)
- **Deployments** — Stuck rollouts (`ProgressDeadlineExceeded`), unavailable replicas, ReplicaSet drift
- **Jobs** — Failed pods, backoff limits, deadline exceeded
- **CronJobs** — Suspended schedules, missed deadlines, last-run failures
- **Services** — No matching pods, selector mismatches, endpoint failures
- **PVCs** — Provisioning failures, binding issues, `WaitForFirstConsumer` stalls
- **Nodes** — `NotReady`, `MemoryPressure`, `DiskPressure`, `PIDPressure`, cordoned/unschedulable

### Namespace scan (v3)
- Scans Deployments, Jobs, CronJobs, Services, PVCs, and standalone Pods together
- Ranks results by severity: critical → warning → healthy
- Groups findings by impact category: Traffic, Rollout, Batch, Storage, Scheduling, Workload

### Relationship tracing (v3)
- **Ingress tracing** — Follows the full traffic path: Ingress → Services → EndpointSlices → Pods
- **Service tracing** — Discovers backing pods and shows endpoint readiness
- **Deployment tracing** — Traces ReplicaSets, active Pods, and dependent resources (PVCs, ConfigMaps, Secrets, ServiceAccounts, ImagePullSecrets)
- **Dependency checks** — Detects missing ConfigMaps, Secrets, and ServiceAccounts before pods fail by inspecting volume mounts, `envFrom`, `env.valueFrom`, and `imagePullSecrets`
- **Path impact analysis** — Explicitly links Service degradation to specific Pod failures (e.g., "Service is impacted because 2 backing pods are in a critical state")

### Remediation safety levels (v3)
Every suggested fix command is tagged with a safety level:
- 🔵 `[inspect]` — Read-only commands (`kubectl logs`, `kubectl get events`)
- 🟢 `[low-risk]` — Safe mutations (`kubectl label`)
- 🟡 `[mutating]` — State-changing commands (`kubectl set resources`, `kubectl uncordon`)
- 🔴 `[destructive]` — High-risk commands (`kubectl delete pod`)

### Deep reasoning & provenance (v3)
- `--explain` flag shows *why* a rule matched with human-readable reasoning
- Evidence includes JSON-path provenance (e.g., `pod.status.containerStatuses[name=api].state.terminated.exitCode`)
- Findings include confidence levels: `high`, `medium`, `low`

---

## Example diagnoses

### OOMKilled

```
$ kubectl-why pod api-server-7f8b4 -n production

  ╭─────────────────────────────────────────────╮
  │  pod/api-server-7f8b4  · production         │
  ╰─────────────────────────────────────────────╯

  Status        OOMKilled

  ● Why
    The container was killed by the kernel because it
    used more memory than its configured limit.

  ● Evidence
    Node           gke-prod-pool-abc123
    Container      api
    Exit Code      137
    Reason         OOMKilled
    Restarts       8

  ● Fix
    [mutating]    Increase memory limit
                  kubectl set resources deployment/<deployment-name> --limits=memory=1Gi -n production
    [inspect]     Check current memory usage
                  kubectl top pod api-server-7f8b4 -n production

  ● Next
    → Review memory limits/requests for this specific container
    → Inspect recent logs to check memory allocation profile right before crash
    → Check whether workload memory requirements have increased over time
```

### ImagePullBackOff

```
$ kubectl-why pod web-frontend-abcde -n staging

  ╭─────────────────────────────────────────────╮
  │  pod/web-frontend-abcde  · staging          │
  ╰─────────────────────────────────────────────╯

  Status        ImagePullBackOff

  ● Why
    Kubernetes cannot pull the container image.
    This is usually a wrong tag, a deleted image, or an auth failure.

  ● Evidence
    Container      frontend
    Image          ghcr.io/example/frontend:v2.99
    Error          manifest unknown: manifest unknown

  ● Fix
    [inspect]     Verify image exists
                  docker manifest inspect ghcr.io/example/frontend:v2.99
    [inspect]     Check pull secret
                  kubectl get secret regcred -n staging -o yaml

  ● Next
    → Check if the image tag exists in the container registry
    → Verify that imagePullSecrets are configured and valid
```

### CrashLoopBackOff

```
$ kubectl-why pod worker-5d9f7 -n default

  ╭─────────────────────────────────────────────╮
  │  pod/worker-5d9f7  · default                │
  ╰─────────────────────────────────────────────╯

  Status        CrashLoopBackOff

  ● Why
    The container has crashed 14 times and Kubernetes is
    backing off before restarting it again.

  ● Evidence
    Container      worker
    Exit Code      1
    Restarts       14
    Back-off       5m0s

  ● Logs (last 5 lines)
    panic: runtime error: invalid memory address
    goroutine 1 [running]:
    main.main()
        /app/main.go:42 +0x1a3

  ● Fix
    [inspect]     Check container logs
                  kubectl logs worker-5d9f7 -n default --previous
```

### Pending Pod — Scheduling failure

```
$ kubectl-why pod ml-training-gpu-0 -n ml

  ╭─────────────────────────────────────────────╮
  │  pod/ml-training-gpu-0  · ml                │
  ╰─────────────────────────────────────────────╯

  Status        Pending — cannot be scheduled

  ● Why
    The scheduler cannot find a node that meets
    the pod's resource requests or constraints.

  ● Evidence
    Phase          Pending
    Scheduler Msg  0/5 nodes are available: 5 Insufficient nvidia.com/gpu

  ● Fix
    [inspect]     Check recent scheduling events
                  kubectl get events -n ml --field-selector involvedObject.name=ml-training-gpu-0

  ● Next
    → Check if nodes have sufficient GPU resources
    → Check if taints, tolerations, or node selectors are preventing scheduling
```

### Evicted Pod — Node pressure

```
$ kubectl-why pod cache-redis-0 -n infra

  ╭─────────────────────────────────────────────╮
  │  pod/cache-redis-0  · infra                 │
  ╰─────────────────────────────────────────────╯

  Status        Evicted

  ● Why
    Likely cause: kubelet evicted the pod because
    the node was under resource pressure.

  ● Evidence
    Pod Reason     Evicted
    Node           gke-prod-pool-node3
    Message        The node was low on resource: memory.

  ● Fix
    [destructive] Delete the evicted pod (it will be recreated if managed by a controller)
                  kubectl delete pod cache-redis-0 -n infra

  ● Next
    → Check node resource usage (kubectl top node gke-prod-pool-node3)
    → Review the exact eviction message in events
```

### Deployment — Stuck rollout

```
$ kubectl-why deployment api -n production

  ╭─────────────────────────────────────────────╮
  │  deployment/api  · production               │
  ╰─────────────────────────────────────────────╯

  Status        ProgressDeadlineExceeded

  ● Why
    The deployment has not completed within its
    progress deadline (600s). 1/3 replicas are available.

  ● Evidence
    Replicas       3 desired / 1 available / 2 unavailable
    Strategy       RollingUpdate
    Condition      ProgressDeadlineExceeded

  ● Pod Diagnoses
    critical  pod/api-v2-abc12  ImagePullBackOff
    critical  pod/api-v2-def34  ImagePullBackOff

  ● Fix
    [inspect]     Check rollout status
                  kubectl rollout status deployment/api -n production
    [mutating]    Roll back to previous version
                  kubectl rollout undo deployment/api -n production
```

### Service — No healthy backends

```
$ kubectl-why service backend-api -n default

  ╭─────────────────────────────────────────────╮
  │  service/backend-api  · default             │
  ╰─────────────────────────────────────────────╯

  Status        Degraded

  ● Why
    Service matches 3 pods, but none are ready.
    All endpoints are failing.

  ● Evidence
    Type           ClusterIP
    Selector       map[app:backend]
    Endpoints      0/3 ready
```

### PVC — Waiting for consumer

```
$ kubectl-why pvc data-volume -n analytics

  ╭─────────────────────────────────────────────╮
  │  pvc/data-volume  · analytics               │
  ╰─────────────────────────────────────────────╯

  Status        Pending

  ● Why
    The PVC will not bind until a Pod is created that uses it
    (WaitForFirstConsumer binding mode).

  ● Evidence
    StorageClass   standard-rwo
    Access Mode    ReadWriteOnce
    Message        waiting for first consumer to be created before binding
```

### Node — Memory pressure

```
$ kubectl-why node gke-prod-pool-node3

  ╭─────────────────────────────────────────────╮
  │  node/gke-prod-pool-node3  · cluster-scoped │
  ╰─────────────────────────────────────────────╯

  Status        Degraded

  ● Why
    Node has MemoryPressure: KubeletHasInsufficientMemory

  ● Evidence
    Unschedulable  false
    Condition      MemoryPressure

  ● Fix
    [inspect]     Inspect the node
                  kubectl describe node gke-prod-pool-node3
```

### Namespace scan — Grouped by impact

```
$ kubectl-why scan -n production

  ╭─────────────────────────────────────────────╮
  │  namespace/production  · scan               │
  ╰─────────────────────────────────────────────╯

  Status        Critical

  ● Summary
    →  6 resources scanned, 3 critical, 1 warning

  ● Findings

    Traffic Impact (Services, Ingress)
      critical  service/api          NO_READY_ENDPOINTS
      warning   service/internal     PARTIAL_ENDPOINTS

    Rollout Impact (Deployments, Rollouts)
      critical  deployment/api       PROGRESS_DEADLINE_EXCEEDED

    Workload Failures (Pods)
      critical  pod/api-v2-abc12     IMAGE_PULL_FAILED
```

### Trace Ingress — Full traffic path

```
$ kubectl-why trace ingress api-gateway -n production

  ╭─────────────────────────────────────────────╮
  │  ingress/api-gateway  · trace               │
  ╰─────────────────────────────────────────────╯

  Status        Critical

  ● Ingress
    warning   ingress/api-gateway   INGRESS_NO_ADDRESS

  ● Backend Services
    critical  service/api           NO_READY_ENDPOINTS
      →  0/2 endpoints ready
      critical  pod/api-abc12       IMAGE_PULL_FAILED
      critical  pod/api-def34       IMAGE_PULL_FAILED

    healthy   service/docs          SERVICE_HEALTHY
      →  2/2 endpoints ready
      →  No unhealthy backing pods.
```

### Trace Deployment — Dependency spider

```
$ kubectl-why trace deployment api -n production

  ╭─────────────────────────────────────────────╮
  │  deployment/api  · trace                    │
  ╰─────────────────────────────────────────────╯

  Status        Critical

  ● Deployment
    critical  deployment/api        PROGRESS_DEADLINE_EXCEEDED

  ● ReplicaSets
    critical  pod/api-v2-abc12      IMAGE_PULL_FAILED
    critical  pod/api-v2-def34      IMAGE_PULL_FAILED

  ● Dependencies
    critical  secret/registry-creds Secret Not Found
    →  No missing/unhealthy dependencies found.
```

### Trace Service — Endpoint readiness

```
$ kubectl-why trace service frontend -n default

  ╭─────────────────────────────────────────────╮
  │  service/frontend  · trace                  │
  ╰─────────────────────────────────────────────╯

  Status        Warning

  ● Service
    warning   service/frontend      PARTIAL_ENDPOINTS

  ● Endpoints
    2/3 endpoints ready

  ● Backing Pods
    critical  pod/frontend-abc12    OOMKilled · confidence high
    →  No unhealthy backing pods.
```

### CronJob — Suspended

```
$ kubectl-why cronjob nightly-backup -n ops

  ╭─────────────────────────────────────────────╮
  │  cronjob/nightly-backup  · ops              │
  ╰─────────────────────────────────────────────╯

  Status        Suspended

  ● Why
    The CronJob is currently suspended and will not
    create new Jobs until it is resumed.

  ● Fix
    [mutating]    Resume the CronJob
                  kubectl patch cronjob nightly-backup -n ops -p '{"spec":{"suspend":false}}'
```

### Deep reasoning with --explain

```
$ kubectl-why pod api-server-7f8b4 -n production --explain

  ...

  ● Explain
    Reasoning: The pod container was terminated by the OOM killer.
               The exit code 137 indicates it was killed by a SIGKILL
               signal from the kernel.
    Primary Diagnosis: OOM_KILLED
    Confidence: high
    Affected Object: container/api

    Evidence Provenance:
      Container: pod.status.containerStatuses[name=api]
      Exit Code: pod.status.containerStatuses[name=api].state.terminated.exitCode
      Reason: pod.status.containerStatuses[name=api].state.terminated.reason
      Restarts: pod.status.containerStatuses[name=api].restartCount
```

---

## How it works

The tool follows a simple three-stage pipeline:

```
┌──────────┐      ┌──────────┐      ┌──────────┐
│ Collect  │ ───▶ │ Analyze  │ ───▶ │  Render  │
│ (kube/)  │      │(analyzer)│      │ (render) │
└──────────┘      └──────────┘      └──────────┘
```

1. **Collect** — Gathers relevant Kubernetes signals from the API (pod status, container states, events, logs, node conditions, endpoint slices)
2. **Analyze** — Matches failure patterns using focused rules with confidence scoring
3. **Render** — Produces human-readable text or structured JSON with evidence and fix commands

This architecture makes it easy to add new failure patterns: one rule file, one fixture, one test.

---

## Version history

### v1 — Pod diagnosis
Single-pod root cause analysis with 11 failure rules, evidence collection, and fix commands.

### v2 — Multi-resource support
Expanded to Deployments, Jobs, CronJobs, Services, PVCs, and Nodes. Added `--explain` flag, secondary findings, and structured v2 JSON output.

### v3 — Namespace scan & relationship tracing
Added `kubectl-why scan` for whole-namespace diagnosis, `kubectl-why trace` for Ingress/Service/Deployment relationship tracing, impact grouping, safety-tagged remediation commands, deep reasoning, and evidence provenance.

---

## Roadmap

Planned next steps include expanding into deeper platform debugging.

- HPA and scaling diagnosis
- NetworkPolicy debugging
- StatefulSet diagnosis and ordered pod analysis
- Advanced provider-specific identity checks such as IRSA / workload identity
- Interactive TUI mode for navigating large scan results

Roadmap items are directional, not fixed commitments.

---

## Contributing

Adding a new failure pattern is intentionally small and approachable. See [CONTRIBUTING.md](CONTRIBUTING.md) for fixtures, rule registration, tests, and development workflow.

Good next additions include:

- `PostStartHookError`
- `InvalidImageName`
- Init-container failure improvements
- StatefulSet-specific diagnosis rules

---

## License

MIT
