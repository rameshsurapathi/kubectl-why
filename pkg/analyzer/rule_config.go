package analyzer

import (
	"fmt"
	"strings"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// ConfigErrorRule detects when a pod is blocked from starting because
// of invalid configurations. The most common scenario is mounting a 
// ConfigMap or Secret as an Environment Variable or Volume, but that 
// resource actually doesn't exist in the cluster.
type ConfigErrorRule struct{}

// Name returns the identifier for this rule.
func (r *ConfigErrorRule) Name() string { return "CreateContainerConfigError" }

// Match checks if any container inside the pod has a waiting reason of CreateContainerConfigError.
func (r *ConfigErrorRule) Match(signals *kube.PodSignals) bool {
	for _, c := range append(signals.Containers, signals.InitContainers...) {
		if c.State.IsWaiting && (c.State.WaitingReason == "CreateContainerConfigError" || c.State.WaitingReason == "CreateContainerError") {
			return true
		}
	}
	return false
}

// Analyze extracts the exact error message from the Kubelet API explaining
// exactly which ConfigMap or Secret is missing, and returns it to the user.
func (r *ConfigErrorRule) Analyze(signals *kube.PodSignals) AnalysisResult {
	var reason, message string
	for _, c := range append(signals.Containers, signals.InitContainers...) {
		if c.State.IsWaiting && (c.State.WaitingReason == "CreateContainerConfigError" || c.State.WaitingReason == "CreateContainerError") {
			reason = c.State.WaitingReason
			message = c.State.WaitingMessage
			break
		}
	}

	res := AnalysisResult{
		Resource:      "pod/" + signals.PodName,
		Namespace:     signals.Namespace,
		Status:        "CreateContainerConfigError",
		PrimaryReason: "Missing ConfigMap or Secret",
		Severity:      "critical",
		Summary: []string{
			"Container cannot start because a referenced",
			"ConfigMap or Secret does not exist in this namespace.",
		},
		Evidence: []Evidence{
			{Label: "Reason", Value: reason},
			{Label: "Message", Value: message},
		},
		FixCommands: []FixCommand{
			{
				Description: "List ConfigMaps in this namespace",
				Command: fmt.Sprintf("kubectl get configmaps -n %s", signals.Namespace),
			},
			{
				Description: "List Secrets in this namespace",
				Command: fmt.Sprintf("kubectl get secrets -n %s", signals.Namespace),
			},
		},
		NextChecks: []string{
			"Check if referenced ConfigMap or Secret exists in this namespace",
		},
		RecentEvents: extractEventStrings(signals.Events, 2),
		RunbookHint:  "kubernetes/config-error",
	}

	findingReasonCode := "CONFIGMAP_NOT_FOUND"
	if strings.Contains(message, "secret") || strings.Contains(message, "Secret") {
		findingReasonCode = "SECRET_NOT_FOUND"
	} else if strings.Contains(message, "serviceaccount") {
		findingReasonCode = "SERVICE_ACCOUNT_NOT_FOUND"
	} else if reason == "CreateContainerError" {
		findingReasonCode = "CREATE_CONTAINER_ERROR"
	}

	res.Findings = []Finding{
		{
			Category:       "Pod",
			ReasonCode:     findingReasonCode,
			Confidence:     "high",
			AffectedObject: res.Resource,
			Message:        message,
			Evidence:       res.Evidence,
			FixCommands:    res.FixCommands,
			NextChecks:     res.NextChecks,
		},
	}
	return res
}

// CannotRunRule triggers when a container's entrypoint or command is impossible
// to execute. For example, trying to run a shell script that isn't copied 
// into the Docker image, or executing a file without +x execute permissions.
type CannotRunRule struct{}

func (r *CannotRunRule) Name() string { return "ContainerCannotRun" }

// Match checks all containers for the "ContainerCannotRun" waiting state.
func (r *CannotRunRule) Match(signals *kube.PodSignals) bool {
	for _, c := range append(signals.Containers, signals.InitContainers...) {
		if c.State.IsWaiting && (c.State.WaitingReason == "ContainerCannotRun" || c.State.WaitingReason == "RunContainerError") {
			return true
		}
	}
	return false
}

func (r *CannotRunRule) Analyze(signals *kube.PodSignals) AnalysisResult {
	var reason, message string
	for _, c := range append(signals.Containers, signals.InitContainers...) {
		if c.State.IsWaiting && (c.State.WaitingReason == "ContainerCannotRun" || c.State.WaitingReason == "RunContainerError") {
			reason = c.State.WaitingReason
			message = c.State.WaitingMessage
			break
		}
	}

	res := AnalysisResult{
		Resource:      "pod/" + signals.PodName,
		Namespace:     signals.Namespace,
		Status:        "ContainerCannotRun",
		PrimaryReason: "Container failed to start",
		Severity:      "critical",
		Summary: []string{
			"The container runtime could not start the container.",
			"This is often a bad entrypoint or missing binary.",
		},
		Evidence: []Evidence{
			{Label: "Reason", Value: reason},
			{Label: "Message", Value: message},
		},
		NextChecks: []string{
			"Check if entrypoint or command is valid and executable",
		},
		RecentEvents: extractEventStrings(signals.Events, 2),
	}
	
	reasonCode := "CONTAINER_CANNOT_RUN"
	if reason == "RunContainerError" {
		reasonCode = "RUN_CONTAINER_ERROR"
	}
	res.Findings = []Finding{
		{
			Category:       "Pod",
			ReasonCode:     reasonCode,
			Confidence:     "high",
			AffectedObject: res.Resource,
			Message:        message,
			Evidence:       res.Evidence,
			NextChecks:     res.NextChecks,
		},
	}
	return res
}
