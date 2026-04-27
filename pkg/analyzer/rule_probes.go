package analyzer

import (
	"fmt"
	"strings"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

type ProbesRule struct{}

func (r *ProbesRule) Name() string {
	return "Probes"
}

func (r *ProbesRule) Match(signals *kube.PodSignals) bool {
	for _, e := range signals.Events {
		if strings.Contains(e.Reason, "Unhealthy") {
			return true
		}
	}
	return false
}

func (r *ProbesRule) Analyze(signals *kube.PodSignals) AnalysisResult {
	var probeType string
	var failingContainer string
	var msg string

	for _, e := range signals.Events {
		if strings.Contains(e.Reason, "Unhealthy") {
			if strings.Contains(e.Message, "Liveness probe failed") {
				probeType = "Liveness"
			} else if strings.Contains(e.Message, "Readiness probe failed") {
				probeType = "Readiness"
			} else if strings.Contains(e.Message, "Startup probe failed") {
				probeType = "Startup"
			} else {
				probeType = "Probe"
			}
			msg = e.Message
			// Try to extract container from "Liveness probe failed: HTTP probe failed... container 'nginx'" 
			// Events typically have InvolvedObject with fieldPath=spec.containers{nginx} but we just have text.
			// Let's just find an unready container
			break
		}
	}

	for _, c := range signals.Containers {
		if !c.Ready {
			failingContainer = c.Name
			break
		}
	}
	if failingContainer == "" && len(signals.Containers) > 0 {
		failingContainer = signals.Containers[0].Name
	}

	reasonCode := "PROBE_FAILED"
	if probeType != "Probe" {
		reasonCode = strings.ToUpper(probeType) + "_PROBE_FAILED"
	}

	return AnalysisResult{
		Resource:      "pod/" + signals.PodName,
		Namespace:     signals.Namespace,
		Status:        "Running",
		PrimaryReason: probeType + " Probe Failed",
		Severity:      "warning",
		Summary: []string{
			fmt.Sprintf("The %s probe is failing for container %s.", strings.ToLower(probeType), failingContainer),
		},
		Evidence: []Evidence{
			{Label: "Container", Value: failingContainer},
			{Label: "Probe Type", Value: probeType},
			{Label: "Message", Value: msg},
		},
		FixCommands: []FixCommand{
			{
				Description: "Check container logs",
				Command:     fmt.Sprintf("kubectl logs %s -n %s -c %s", signals.PodName, signals.Namespace, failingContainer),
			},
		},
		NextChecks: []string{
			fmt.Sprintf("kubectl describe pod %s -n %s", signals.PodName, signals.Namespace),
		},
		Findings: []Finding{
			{
				Category:       "Pod",
				ReasonCode:     reasonCode,
				Confidence:     "high",
				AffectedObject: "container/" + failingContainer,
				Message:        fmt.Sprintf("The %s probe is failing: %s", strings.ToLower(probeType), msg),
				Evidence: []Evidence{
					{Label: "Message", Value: msg},
				},
				FixCommands: []FixCommand{
					{
						Description: "Check container logs",
						Command:     fmt.Sprintf("kubectl logs %s -n %s -c %s", signals.PodName, signals.Namespace, failingContainer),
					},
				},
			},
		},
	}
}
