package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// AnalyzeDeployment produces a result for a deployment
// by reusing pod analysis and adding deployment context
func AnalyzeDeployment(
	signals *kube.DeploymentSignals,
) AnalysisResult {

	resource := "deployment/" + signals.DeploymentName

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
					Label: "Available",
					Value: fmt.Sprintf(
						"%d", signals.AvailableReplicas),
				},
			},
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	// ── Has failing pods — analyze the worst one ─────
	if signals.FailingPodSignals == nil {
		// We know pods are failing but couldn't
		// get details — give basic info
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
					"kubectl get pods -n %s -l app=%s",
					signals.Namespace,
					signals.DeploymentName),
			},
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	// ── Analyze the worst failing pod ────────────────
	// Reuse all the pod analysis rules we already built
	podResult := AnalyzePod(signals.FailingPodSignals)

	// Override resource name to show deployment context
	podResult.Resource = resource

	// Add deployment-level context to evidence
	deployEvidence := []Evidence{
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

	// Prepend deployment evidence before pod evidence
	podResult.Evidence = append(
		deployEvidence,
		podResult.Evidence...,
	)

	return podResult
}
