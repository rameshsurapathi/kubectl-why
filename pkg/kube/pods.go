package kube

import (
    "context"
    "fmt"
    "os"
    "time"

    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
)

// CollectPodSignals is the main entry point for
// collecting all diagnostic data for a pod
func CollectPodSignals(
    client *kubernetes.Clientset,
    name, namespace string,
    tailLines int64,
    maxEvents int,
) (*PodSignals, error) {

    ctx := context.Background()

    // ── 1. Fetch the pod object ──────────────────────
    pod, err := client.CoreV1().Pods(namespace).Get(
        ctx, name, metav1.GetOptions{})
    if err != nil {
        return nil, fmt.Errorf(
            "pod %q not found in namespace %q: %w",
            name, namespace, err)
    }

    // ── 2. Build base signals ────────────────────────
    signals := &PodSignals{
        PodName:   name,
        Namespace: namespace,
        NodeName:  pod.Spec.NodeName,
        Phase:     string(pod.Status.Phase),
        PodReason: pod.Status.Reason,
        CreatedAt: pod.CreationTimestamp.Time,
        Age:       age(pod.CreationTimestamp.Time),
    }

    // ── 3. Conditions ────────────────────────────────
    signals.Conditions = collectConditions(pod)

    // ── 4. Container statuses ────────────────────────
    for _, cs := range pod.Status.ContainerStatuses {
        signals.Containers = append(
            signals.Containers,
            BuildContainerSignal(cs, false),
        )
    }

    for _, cs := range pod.Status.InitContainerStatuses {
        signals.InitContainers = append(
            signals.InitContainers,
            BuildContainerSignal(cs, true),
        )
    }

    // ── 5. Events ────────────────────────────────────
    events, err := CollectEvents(client, string(pod.UID), namespace, maxEvents)
    if err != nil {
        // Non-fatal — continue without events
        fmt.Fprintf(os.Stderr, "  warning: could not fetch events: %v\n", err)
    } else {
        signals.Events = events
    }

    // ── 6. Logs ──────────────────────────────────────
    targetContainer := findFailingContainer(pod)
    if targetContainer != "" {
        logs, err := CollectLogs(
            client, name, namespace,
            targetContainer, tailLines,
        )
        if err != nil {
            // Non-fatal — continue without logs
            _ = err // visually silent, logs are best effort
        } else {
            signals.RecentLogs = logs
            signals.LogsFromContainer = targetContainer
        }
    }

    return signals, nil
}

// collectConditions extracts pod conditions into signals
func collectConditions(pod *corev1.Pod) []ConditionSignal {
    var conditions []ConditionSignal
    for _, c := range pod.Status.Conditions {
        conditions = append(conditions, ConditionSignal{
            Type:               string(c.Type),
            Status:             string(c.Status),
            Reason:             c.Reason,
            Message:            c.Message,
            LastTransitionTime: c.LastTransitionTime.Time,
        })
    }
    return conditions
}

// BuildContainerSignal converts a Kubernetes
// ContainerStatus into our internal signal format
func BuildContainerSignal(
    cs corev1.ContainerStatus,
    isInit bool,
) ContainerSignal {

    c := ContainerSignal{
        Name:         cs.Name,
        Image:        cs.Image,
        Ready:        cs.Ready,
        RestartCount: cs.RestartCount,
        IsInit:       isInit,
    }

    // ── Current state ────────────────────────────────
    switch {
    case cs.State.Waiting != nil:
        c.State = ContainerStateDetail{
            IsWaiting:      true,
            WaitingReason:  cs.State.Waiting.Reason,
            WaitingMessage: cs.State.Waiting.Message,
        }

    case cs.State.Terminated != nil:
        t := cs.State.Terminated
        c.State = ContainerStateDetail{
            IsTerminated:      true,
            ExitCode:          t.ExitCode,
            TerminatedReason:  t.Reason,
            TerminatedMessage: t.Message,
            FinishedAt:        t.FinishedAt.Time,
            StartedAt:         t.StartedAt.Time,
        }

    case cs.State.Running != nil:
        c.State = ContainerStateDetail{
            IsRunning: true,
        }
    }

    // ── Last terminated state (crash history) ────────
    if cs.LastTerminationState.Terminated != nil {
        t := cs.LastTerminationState.Terminated
        c.LastState = ContainerStateDetail{
            IsTerminated:      true,
            ExitCode:          t.ExitCode,
            TerminatedReason:  t.Reason,
            TerminatedMessage: t.Message,
            FinishedAt:        t.FinishedAt.Time,
            StartedAt:         t.StartedAt.Time,
        }
    }

    return c
}

// findFailingContainer returns the name of the container
// most likely to have useful diagnostic logs.
//
// Priority order:
// 1. Container currently in Waiting state (most likely cause)
// 2. Container currently in Terminated state
// 3. Container with highest restart count
// 4. First container in spec (fallback)
func findFailingContainer(pod *corev1.Pod) string {
    // Priority 1: waiting container
    for _, cs := range pod.Status.ContainerStatuses {
        if cs.State.Waiting != nil {
            return cs.Name
        }
    }

    // Also check init containers
    for _, cs := range pod.Status.InitContainerStatuses {
        if cs.State.Waiting != nil {
            return cs.Name
        }
    }

    // Priority 2: terminated container
    for _, cs := range pod.Status.ContainerStatuses {
        if cs.State.Terminated != nil {
            return cs.Name
        }
    }

    // Priority 3: highest restart count
    var maxRestarts int32
    var name string
    for _, cs := range pod.Status.ContainerStatuses {
        if cs.RestartCount > maxRestarts {
            maxRestarts = cs.RestartCount
            name = cs.Name
        }
    }
    if name != "" {
        return name
    }

    // Priority 4: first container in spec
    if len(pod.Spec.Containers) > 0 {
        return pod.Spec.Containers[0].Name
    }

    return ""
}

// age returns a human-readable duration since t
func age(t time.Time) string {
    d := time.Since(t)
    switch {
    case d < time.Minute:
        return fmt.Sprintf("%ds", int(d.Seconds()))
    case d < time.Hour:
        return fmt.Sprintf("%dm", int(d.Minutes()))
    case d < 24*time.Hour:
        return fmt.Sprintf("%dh", int(d.Hours()))
    default:
        return fmt.Sprintf("%dd", int(d.Hours()/24))
    }
}
