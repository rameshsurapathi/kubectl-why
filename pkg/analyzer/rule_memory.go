package analyzer

import (
	"fmt"
	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// OOMKilledRule identifies containers that were destroyed by the OS
// for exceeding their requested memory limits. Exit Code 137 translates
// to "Fatal Error + Signal 9 (SIGKILL)".
type OOMKilledRule struct{}

func (r *OOMKilledRule) Name() string { return "OOMKilled" }

// Match tests if the top-level pod reason is OOM, or if ANY container
// in the pod exited with code 137 or reason 'OOMKilled'.
func (r *OOMKilledRule) Match(signals *kube.PodSignals) bool {
	if signals.PodReason == "OOMKilled" {
		return true
	}
	for _, c := range append(signals.Containers, signals.InitContainers...) {
		if c.State.IsTerminated && (c.State.ExitCode == 137 || c.State.TerminatedReason == "OOMKilled") {
			return true
		}
		if c.LastState.IsTerminated && (c.LastState.ExitCode == 137 || c.LastState.TerminatedReason == "OOMKilled") {
			return true
		}
	}
	return false
}

// Analyze compiles the OOM report, explicitly targeting the container that
// caused the failure, which is vital when a pod contains multiple sidecars.
func (r *OOMKilledRule) Analyze(signals *kube.PodSignals) AnalysisResult {
	var evidence []Evidence
	evidence = append(evidence, Evidence{Label: "Node", Value: signals.NodeName})

	for _, c := range append(signals.Containers, signals.InitContainers...) {
		isLastOOM := c.LastState.IsTerminated && (c.LastState.ExitCode == 137 || c.LastState.TerminatedReason == "OOMKilled")
		isCurrOOM := c.State.IsTerminated && (c.State.ExitCode == 137 || c.State.TerminatedReason == "OOMKilled")
		if isLastOOM || isCurrOOM {
			evidence = append(evidence, Evidence{Label: "Container", Value: c.Name, Provenance: fmt.Sprintf("pod.status.containerStatuses[name=%s]", c.Name)})
			evidence = append(evidence, Evidence{Label: "Exit Code", Value: "137", Provenance: fmt.Sprintf("pod.status.containerStatuses[name=%s].state.terminated.exitCode", c.Name)})
			evidence = append(evidence, Evidence{Label: "Reason", Value: "OOMKilled", Provenance: fmt.Sprintf("pod.status.containerStatuses[name=%s].state.terminated.reason", c.Name)})
			evidence = append(evidence, Evidence{Label: "Restarts", Value: fmt.Sprintf("%d", c.RestartCount), Provenance: fmt.Sprintf("pod.status.containerStatuses[name=%s].restartCount", c.Name)})
			break
		}
	}

	return AnalysisResult{
		Resource:      "pod/" + signals.PodName,
		Namespace:     signals.Namespace,
		Status:        "OOMKilled",
		PrimaryReason: "Container exceeded its memory limit",
		Severity:      "critical",
		Summary: []string{
			"The container was killed by the kernel because it",
			"used more memory than its configured limit.",
		},
		Evidence:   evidence,
		RecentLogs: signals.RecentLogs,
		Reasoning: "The pod container was terminated by the OOM killer. The exit code 137 indicates it was killed by a SIGKILL signal from the kernel.",
		FixCommands: []FixCommand{
			{
				Description: "Increase memory limit",
				Command: fmt.Sprintf(
					"kubectl set resources deployment/<deployment-name> --limits=memory=1Gi -n %s", signals.Namespace),
				SafetyLevel: "mutating",
			},
			{
				Description: "Check current memory usage",
				Command: fmt.Sprintf(
					"kubectl top pod %s -n %s", signals.PodName, signals.Namespace),
				SafetyLevel: "inspect",
			},
		},
		NextChecks: []string{
			"Review memory limits/requests for this specific container",
			"Inspect recent logs to check memory allocation profile right before crash",
			"Check whether workload memory requirements have increased over time",
		},
		RunbookHint: "kubernetes/oom-killed",
	}
}
