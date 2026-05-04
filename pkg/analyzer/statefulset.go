package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// AnalyzeStatefulSet produces a result for a statefulset
// by reusing pod analysis and adding statefulset context
func AnalyzeStatefulSet(
	signals *kube.StatefulSetSignals,
) AnalysisResult {

	resource := "statefulset/" + signals.StatefulSetName

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
						"%d/%d ready",
						signals.ReadyReplicas,
						signals.DesiredReplicas),
				},
				{
					Label: "Current",
					Value: fmt.Sprintf(
						"%d", signals.CurrentReplicas),
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
					signals.StatefulSetName),
			},
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	// ── Analyze the worst failing pod ────────────────
	podResult := AnalyzePod(signals.FailingPodSignals)

	// Override resource name to show statefulset context
	podResult.Resource = resource

	// Add statefulset-level context to evidence
	stsEvidence := []Evidence{
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

	// Prepend statefulset evidence before pod evidence
	podResult.Evidence = append(
		stsEvidence,
		podResult.Evidence...,
	)

	return podResult
}
