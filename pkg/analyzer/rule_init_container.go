package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

type InitContainerRule struct{}

func (r *InitContainerRule) Name() string {
	return "InitContainerFailed"
}

func (r *InitContainerRule) Match(signals *kube.PodSignals) bool {
	if signals.Phase == "Pending" || signals.Phase == "Failed" || signals.Phase == "Unknown" {
		for _, c := range signals.InitContainers {
			if c.State.IsTerminated && c.State.ExitCode != 0 {
				return true
			}
			if c.State.IsWaiting {
				// e.g. CrashLoopBackOff on init container
				return true
			}
		}
	}
	return false
}

func (r *InitContainerRule) Analyze(signals *kube.PodSignals) AnalysisResult {
	var failingContainer kube.ContainerSignal
	for _, c := range signals.InitContainers {
		if (c.State.IsTerminated && c.State.ExitCode != 0) || c.State.IsWaiting {
			failingContainer = c
			break
		}
	}

	reason := "InitContainerCrash"
	if failingContainer.State.IsWaiting {
		reason = "InitContainer" + failingContainer.State.WaitingReason
	} else if failingContainer.State.IsTerminated {
		reason = "InitContainer" + failingContainer.State.TerminatedReason
	}

	msg := fmt.Sprintf("Init container %s is failing.", failingContainer.Name)
	if failingContainer.State.IsTerminated {
		msg += fmt.Sprintf(" It exited with code %d.", failingContainer.State.ExitCode)
	} else if failingContainer.State.IsWaiting {
		msg += fmt.Sprintf(" It is waiting: %s.", failingContainer.State.WaitingReason)
	}

	evidence := []Evidence{
		{Label: "Init Container", Value: failingContainer.Name},
	}
	if failingContainer.State.IsTerminated {
		evidence = append(evidence, Evidence{Label: "Exit Code", Value: fmt.Sprintf("%d", failingContainer.State.ExitCode)})
	} else if failingContainer.State.IsWaiting {
		evidence = append(evidence, Evidence{Label: "Reason", Value: failingContainer.State.WaitingReason})
	}

	return AnalysisResult{
		Resource:      "pod/" + signals.PodName,
		Namespace:     signals.Namespace,
		Status:        "Init:CrashLoopBackOff",
		PrimaryReason: reason,
		Severity:      "critical",
		Summary: []string{
			msg,
			"Pod initialization cannot proceed until this container succeeds.",
		},
		Evidence: evidence,
		FixCommands: []FixCommand{
			{
				Description: "Check init container logs",
				Command:     fmt.Sprintf("kubectl logs %s -n %s -c %s", signals.PodName, signals.Namespace, failingContainer.Name),
			},
		},
		NextChecks: []string{
			fmt.Sprintf("kubectl describe pod %s -n %s", signals.PodName, signals.Namespace),
		},
		Findings: []Finding{
			{
				Category:       "Pod",
				ReasonCode:     "INIT_CONTAINER_FAILED",
				Confidence:     "high",
				AffectedObject: "container/" + failingContainer.Name,
				Message:        msg,
				Evidence:       evidence,
				FixCommands: []FixCommand{
					{
						Description: "Check init container logs",
						Command:     fmt.Sprintf("kubectl logs %s -n %s -c %s", signals.PodName, signals.Namespace, failingContainer.Name),
					},
				},
			},
		},
	}
}
