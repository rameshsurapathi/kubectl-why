package analyzer

import (
	"fmt"
	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// CrashLoopRule is the generic catch-all for pods that continually restart.
// Because it sits lower in the registry priority list, it will only trigger
// if the crash wasn't specifically identified as an OOM or AppCrash.
// It reports the last known termination exit code to help guide debugging.
type CrashLoopRule struct{}

func (r *CrashLoopRule) Name() string { return "CrashLoopBackOff" }

// Match checks the Waiting state for the generic CrashLoopBackOff string.
func (r *CrashLoopRule) Match(signals *kube.PodSignals) bool {
	for _, c := range append(signals.Containers, signals.InitContainers...) {
		if c.State.IsWaiting && c.State.WaitingReason == "CrashLoopBackOff" {
			return true
		}
	}
	return false
}

// Analyze builds the report. It uses the `LastState` to show exactly what
// the container was doing right before it went into the BackOff state.
func (r *CrashLoopRule) Analyze(signals *kube.PodSignals) AnalysisResult {
	var reason, lastReason string
	var restarts int32

	for _, c := range append(signals.Containers, signals.InitContainers...) {
		if c.State.IsWaiting && c.State.WaitingReason == "CrashLoopBackOff" {
			reason = c.State.WaitingReason
			lastReason = c.LastState.TerminatedReason
			if lastReason == "" && c.LastState.IsTerminated && c.LastState.ExitCode != 0 {
				lastReason = fmt.Sprintf("Exit Code %d", c.LastState.ExitCode)
			}
			restarts = c.RestartCount
			break
		}
	}

	return AnalysisResult{
		Resource:      "pod/" + signals.PodName,
		Namespace:     signals.Namespace,
		Status:        "CrashLoopBackOff",
		PrimaryReason: "Container is repeatedly crashing",
		Severity:      "critical",
		Summary: []string{
			"The container starts, crashes, and Kubernetes",
			"keeps restarting it with increasing backoff delays.",
		},
		Evidence: []Evidence{
			{Label: "Reason", Value: reason},
			{Label: "Restarts", Value: fmt.Sprintf("%d", restarts)},
			{Label: "Last Crash", Value: lastReason},
		},
		RecentLogs: signals.RecentLogs,
		FixCommands: []FixCommand{
			{
				Description: "Check logs from previous crash",
				Command: fmt.Sprintf("kubectl logs %s -n %s --previous", signals.PodName, signals.Namespace),
			},
			{
				Description: "Describe pod for full event history",
				Command: fmt.Sprintf("kubectl describe pod %s -n %s", signals.PodName, signals.Namespace),
			},
		},
		NextChecks: []string{
			"Check the application logs (included above) for stack traces or fatal errors",
		},
		RecentEvents: extractEventStrings(signals.Events, 3),
		RunbookHint:  "kubernetes/crashloop",
	}
}
