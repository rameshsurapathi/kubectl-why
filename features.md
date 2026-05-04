# kubectl-why v4 Features

This document tracks all new features, enhancements, and improvements implemented in the `kubectl-why` v4 release.

## 1. StatefulSet Support

`kubectl-why` now supports diagnosing `StatefulSet` resources. It explains failures related to StatefulSet pods, giving detailed output and combining the context of the StatefulSet with the failing pod's diagnostics.

### Example Usage

```bash
kubectl-why statefulset web -n production
# or
kubectl-why sts web -n production
```

### Sample Output

```text
  ╭─────────────────────────────────────────────╮
  │  statefulset/web  · production              │
  ╰─────────────────────────────────────────────╯

  Status        Degraded

  ● Why
    Pods are failing
    3 of 5 pods are not healthy.

  ● Evidence
    Healthy pods   2
    Failing pods   3

  ● Next
    → kubectl get pods -n production -l app=web
```
*(If a pod is crashing, it will show the worst pod's diagnosis such as `CrashLoopBackOff`, `OOMKilled`, etc. within the statefulset context)*

## 2. DaemonSet Support

`kubectl-why` now supports diagnosing `DaemonSet` resources. It groups failures by the DaemonSet and provides insight into scheduling or container startup issues across nodes.

### Example Usage

```bash
kubectl-why daemonset fluentd -n kube-system
# or
kubectl-why ds fluentd -n kube-system
```

### Sample Output

```text
  ╭─────────────────────────────────────────────╮
  │  daemonset/fluentd  · kube-system           │
  ╰─────────────────────────────────────────────╯

  Status        Healthy

  ● Why
    All pods running
    All pods are running and ready.

  ● Evidence
    Pods           12 ready / 12 desired
    Misscheduled   0
```

## 3. Namespace Scan Enhancement (v4)

The `kubectl-why scan -n <namespace>` command has been updated to automatically include `StatefulSets`, `DaemonSets`, `ResourceQuotas`, and `NetworkPolicies` in its diagnostic sweep.

## 4. ResourceQuota Support

`kubectl-why` now tracks and flags `ResourceQuota` objects. If a namespace is close to or has exceeded its CPU, Memory, or Object quotas, it identifies the breached quotas and explains why subsequent deployments or pods are unable to scale.

### Example Usage

```bash
kubectl-why resourcequota default-quota -n production
# or
kubectl-why rq default-quota -n production
```

### Sample Output

```text
  ╭─────────────────────────────────────────────╮
  │  resourcequota/default-quota  · production  │
  ╰─────────────────────────────────────────────╯

  Status        Exceeded

  ● Why
    Quota exceeded
    Resource limits exceeded for: requests.cpu

  ● Evidence
    requests.cpu   10 used / 10 hard
    limits.cpu     15 used / 20 hard
    pods           8 used / 50 hard

  ● Next
    → kubectl describe resourcequota default-quota -n production
```

## 5. NetworkPolicy Support

`kubectl-why` now analyzes `NetworkPolicy` objects. It determines whether a NetworkPolicy is inadvertently blocking all traffic (default deny) or if a NetworkPolicy's pod selector isn't matching any running pods (indicating a configuration typo).

### Example Usage

```bash
kubectl-why networkpolicy backend-deny -n staging
# or
kubectl-why netpol backend-deny -n staging
```

### Sample Output

```text
  ╭─────────────────────────────────────────────╮
  │  networkpolicy/backend-deny  · staging      │
  ╰─────────────────────────────────────────────╯

  Status        Warning

  ● Why
    Default Deny All
    This NetworkPolicy blocks ALL Ingress and Egress traffic for the selected pods.

  ● Evidence
    Matched Pods       12
    Has Ingress Rules  false
    Has Egress Rules   false
```

## 6. HorizontalPodAutoscaler (HPA) Support

`kubectl-why` now tracks and flags `HorizontalPodAutoscaler` objects. It helps determine why a deployment is not scaling up or down as expected, highlighting if the metrics server is unreachable, if the scaling max limits have been hit, or if it is unable to fetch the target metrics.

### Example Usage

```bash
kubectl-why hpa api-server -n production
```

### Sample Output

```text
  ╭─────────────────────────────────────────────╮
  │  hpa/api-server  · production               │
  ╰─────────────────────────────────────────────╯

  Status        Warning

  ● Why
    Max replicas reached
    The HPA wants to scale up but has hit the MaxReplicas limit.

  ● Evidence
    Replicas       10 current / 12 desired
    Limits         Min: 2 / Max: 10

  ● Next
    → kubectl describe hpa api-server -n production
```

## 7. Pod Disruption Budget (PDB) Support

`kubectl-why` now tracks and flags `PodDisruptionBudget` objects. It warns if a PDB inadvertently blocks all evictions (preventing node drains) by allowing zero disruptions.

### Example Usage

```bash
kubectl-why pdb api-server-pdb -n production
```

### Sample Output

```text
  ╭─────────────────────────────────────────────╮
  │  pdb/api-server-pdb  · production           │
  ╰─────────────────────────────────────────────╯

  Status        Warning

  ● Why
    0 Disruptions Allowed
    This PDB blocks all voluntary disruptions (e.g., node drains).
    No pods can be evicted safely.

  ● Evidence
    Disruptions Allowed  0
    Healthy Pods         2 current / 3 desired
    Expected Pods        3
```

## 8. TLS Certificate Diagnostics

`kubectl-why` can now analyze `kubernetes.io/tls` secrets and determine if the embedded certificate is expired or expiring soon (< 14 days).

### Example Usage

```bash
kubectl-why tls api-server-cert -n production
# or
kubectl-why cert api-server-cert -n production
```

## 9. RBAC RoleBinding Analysis

`kubectl-why` now helps diagnose RBAC issues. It inspects `RoleBinding` objects and flags an error if the binding points to a `Role` or `ClusterRole` that does not exist.

### Example Usage

```bash
kubectl-why rolebinding admin-binding -n default
# or
kubectl-why rb admin-binding -n default
```

## 10. Cluster DNS Health Check

A new standalone command is added to quickly diagnose the health of the cluster's internal DNS (`CoreDNS`).

### Example Usage

```bash
kubectl-why dns
```

### Sample Output

```text
  ╭─────────────────────────────────────────────╮
  │  cluster/dns  · kube-system                 │
  ╰─────────────────────────────────────────────╯

  Status        Healthy

  ● Why
    DNS is healthy
    Cluster DNS is running and has ready endpoints.

  ● Evidence
    Service Found    true
    Endpoints Ready  2/2
```

## 11. Interactive Dashboard (View)

`kubectl-why` introduces the `view` command (aliases: `dash`, `ui`), an interactive Terminal User Interface (TUI) that unifies namespace scanning, relationship tracing, and resource deep-dives into a single screen.

*Note: The older `scan`, `map`, and `trace` commands have been removed to keep the tool extremely simple and focused. The `view` dashboard completely replaces them.*

The dashboard automatically links Workloads (Deployments, StatefulSets, DaemonSets) with their backing Services and active Pods. It features:
- A navigatable tree on the left pane to explore your cluster topology.
- A dynamic detail panel on the right pane that instantly shows logs, evidence, and fix commands when you select a failing resource.

### Example Usage

```bash
kubectl-why view -n production
```

### Key Bindings
- `Up/Down` or `k/j`: Navigate the tree
- `Space/Right/Enter`: Expand/Collapse groups and workloads
- `r`: Refresh the live scan
- `q`: Quit the dashboard
