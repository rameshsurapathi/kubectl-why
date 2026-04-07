package kube

import "time"

// PodSignals holds every raw signal collected from
// the Kubernetes API needed to diagnose a pod.
// The analyzer reads this — it never calls the API directly.
type PodSignals struct {
    // Identity
    PodName   string
    Namespace string
    NodeName  string

    // Top-level pod state
    Phase     string // Pending, Running, Succeeded, Failed, Unknown
    PodReason string // e.g. "Evicted", "OOMKilled" at pod level

    // Conditions (from pod.status.conditions)
    Conditions []ConditionSignal

    // Containers (main containers)
    Containers []ContainerSignal

    // Init containers
    InitContainers []ContainerSignal

    // Events sorted by most recent first
    Events []EventSignal

    // Logs from the most relevant failing container
    RecentLogs string

    // Which container the logs came from
    LogsFromContainer string

    // When the pod was created
    CreatedAt time.Time

    // How long the pod has been in current state
    // (calculated, not from API)
    Age string
}

type ConditionSignal struct {
    Type               string // Ready, PodScheduled, etc.
    Status             string // True, False, Unknown
    Reason             string
    Message            string
    LastTransitionTime time.Time
}

type ContainerSignal struct {
    Name         string
    Image        string
    Ready        bool
    RestartCount int32
    IsInit       bool

    // Current state
    State ContainerStateDetail

    // Previous terminated state (crash history)
    LastState ContainerStateDetail
}

type ContainerStateDetail struct {
    // Waiting state
    IsWaiting      bool
    WaitingReason  string // CrashLoopBackOff, ImagePullBackOff, etc.
    WaitingMessage string

    // Terminated state
    IsTerminated       bool
    ExitCode           int32
    TerminatedReason   string // OOMKilled, Error, Completed
    TerminatedMessage  string
    FinishedAt         time.Time
    StartedAt          time.Time

    // Running state
    IsRunning bool
}

type EventSignal struct {
    Type      string // Normal, Warning
    Reason    string // FailedScheduling, BackOff, Pulled, etc.
    Message   string
    Count     int32
    FirstTime time.Time
    LastTime  time.Time
    Component string // kubelet, scheduler, etc.
}
