package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

type NonZeroExitRule struct{}

func (r *NonZeroExitRule) Name() string { return "NonZeroExit" }

func (r *NonZeroExitRule) Match(signals *kube.PodSignals) bool {
	// Let's only match if there is a terminated container with non-zero exit code
	// that isn't handled by other specific rules (e.g. 137=OOM, 139=Segfault, 1=AppCrash)
	for _, c := range signals.Containers {
		if c.State.IsTerminated && c.State.ExitCode != 0 && c.State.ExitCode != 1 && c.State.ExitCode != 137 && c.State.ExitCode != 139 {
			return true
		}
		// Also handle CrashLoopBackOff where the LastState has a strange non-zero exit code
		if c.State.IsWaiting && c.State.WaitingReason == "CrashLoopBackOff" && c.LastState.IsTerminated && c.LastState.ExitCode != 0 && c.LastState.ExitCode != 1 && c.LastState.ExitCode != 137 && c.LastState.ExitCode != 139 {
			return true
		}
	}
	return false
}

func (r *NonZeroExitRule) Analyze(signals *kube.PodSignals) AnalysisResult {
	var exitCode int32
	var failingContainer string
	var message string

	for _, c := range signals.Containers {
		if c.State.IsTerminated && c.State.ExitCode != 0 && c.State.ExitCode != 1 && c.State.ExitCode != 137 && c.State.ExitCode != 139 {
			exitCode = c.State.ExitCode
			failingContainer = c.Name
			message = c.State.TerminatedMessage
			break
		}
		if c.State.IsWaiting && c.State.WaitingReason == "CrashLoopBackOff" && c.LastState.IsTerminated && c.LastState.ExitCode != 0 && c.LastState.ExitCode != 1 && c.LastState.ExitCode != 137 && c.LastState.ExitCode != 139 {
			exitCode = c.LastState.ExitCode
			failingContainer = c.Name
			message = c.LastState.TerminatedMessage
			break
		}
	}

	if message == "" {
		message = fmt.Sprintf("Process exited with code %d", exitCode)
	}

	status := "Failed"
	for _, c := range signals.Containers {
		if c.State.IsWaiting && c.State.WaitingReason == "CrashLoopBackOff" {
			status = "CrashLoopBackOff"
			break
		}
	}

	msg := fmt.Sprintf("Container %s exited with non-zero code %d", failingContainer, exitCode)
	res := AnalysisResult{
		Resource:      "pod/" + signals.PodName,
		Namespace:     signals.Namespace,
		Status:        status,
		PrimaryReason: fmt.Sprintf("Exit Code %d", exitCode),
		Severity:      "critical",
		Summary: []string{
			msg,
			"This indicates an application-specific failure.",
		},
		Evidence: []Evidence{
			{Label: "Container", Value: failingContainer},
			{Label: "Exit Code", Value: fmt.Sprintf("%d", exitCode)},
			{Label: "Message", Value: message},
		},
		FixCommands: []FixCommand{
			{
				Description: "Check container logs",
				Command:     fmt.Sprintf("kubectl logs %s -n %s -c %s", signals.PodName, signals.Namespace, failingContainer),
				SafetyLevel: "inspect",
			},
		},
		NextChecks: []string{
			fmt.Sprintf("kubectl describe pod %s -n %s", signals.PodName, signals.Namespace),
		},
	}
	
	res.Findings = []Finding{
		{
			Category:       "Pod",
			ReasonCode:     "CONTAINER_EXIT_NONZERO",
			Confidence:     "high",
			AffectedObject: "container/" + failingContainer,
			Message:        msg,
			Evidence:       res.Evidence,
			FixCommands:    res.FixCommands,
			NextChecks:     res.NextChecks,
		},
	}
	return res
}
