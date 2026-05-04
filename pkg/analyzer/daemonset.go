package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// AnalyzeDaemonSet produces a result for a daemonset
// by reusing pod analysis and adding daemonset context
func AnalyzeDaemonSet(
	signals *kube.DaemonSetSignals,
) AnalysisResult {

	resource := "daemonset/" + signals.DaemonSetName

	// ── All pods healthy ─────────────────────────────
	if signals.AllHealthy {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Healthy",
			PrimaryReason: "All pods running",
			Severity:      "healthy",
			Summary: []string{
				"All pods are running and ready.",
			},
			Evidence: []Evidence{
				{
					Label: "Pods",
					Value: fmt.Sprintf(
						"%d ready / %d desired",
						signals.NumberReady,
						signals.DesiredNumberScheduled),
				},
				{
					Label: "Misscheduled",
					Value: fmt.Sprintf(
						"%d", signals.NumberMisscheduled),
				},
			},
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	// ── Has failing pods — analyze the worst one ─────
	if signals.FailingPodSignals == nil {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Degraded",
			PrimaryReason: "Pods are failing",
			Severity:      "critical",
			Summary: []string{
				fmt.Sprintf(
					"%d of %d pods are not healthy.",
					signals.FailingPods,
					signals.TotalPods),
			},
			Evidence: []Evidence{
				{
					Label: "Healthy pods",
					Value: fmt.Sprintf("%d", signals.HealthyPods),
				},
				{
					Label: "Failing pods",
					Value: fmt.Sprintf("%d", signals.FailingPods),
				},
			},
			NextChecks: []string{
				fmt.Sprintf(
					"kubectl get pods -n %s | grep %s",
					signals.Namespace,
					signals.DaemonSetName),
			},
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	// ── Analyze the worst failing pod ────────────────
	podResult := AnalyzePod(signals.FailingPodSignals)

	// Override resource name to show daemonset context
	podResult.Resource = resource

	// Add daemonset-level context to evidence
	dsEvidence := []Evidence{
		{
			Label: "Pods",
			Value: fmt.Sprintf(
				"%d healthy  %d failing  (%d total)",
				signals.HealthyPods,
				signals.FailingPods,
				signals.TotalPods),
		},
		{
			Label: "Analyzing pod",
			Value: signals.FailingPodName,
		},
	}

	// Prepend daemonset evidence before pod evidence
	podResult.Evidence = append(
		dsEvidence,
		podResult.Evidence...,
	)

	return podResult
}
