package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// AnalyzeDeploymentRollout produces a result for deployment rollout issues
func AnalyzeDeploymentRollout(signals *kube.DeploymentSignals) AnalysisResult {
	resource := "deployment/" + signals.DeploymentName

	var reasonCode string
	var msg string

	for _, cond := range signals.Conditions {
		if cond.Type == appsv1.DeploymentProgressing && cond.Status == corev1.ConditionFalse && cond.Reason == "ProgressDeadlineExceeded" {
			reasonCode = "PROGRESS_DEADLINE_EXCEEDED"
			msg = cond.Message
		}
	}

	if reasonCode == "" && signals.UpdatedReplicas < signals.DesiredReplicas {
		reasonCode = "UPDATED_REPLICAS_LESS_THAN_DESIRED"
		msg = fmt.Sprintf("%d of %d replicas updated.", signals.UpdatedReplicas, signals.DesiredReplicas)
	}

	if reasonCode == "" && signals.AvailableReplicas < signals.DesiredReplicas {
		reasonCode = "UNAVAILABLE_REPLICAS"
		msg = fmt.Sprintf("%d of %d replicas available.", signals.AvailableReplicas, signals.DesiredReplicas)
	}

	if reasonCode == "" {
		if signals.TotalPods == 0 && signals.DesiredReplicas > 0 {
			reasonCode = "NO_PODS_MATCH_SELECTOR"
			msg = "Deployment selector matches no pods, or ReplicaSet failed to create pods."
		} else {
			reasonCode = "ROLLOUT_HEALTHY"
			msg = "Deployment rollout is healthy."
		}
	}

	res := AnalysisResult{
		SchemaVersion: "v2",
		Resource:      resource,
		Namespace:     signals.Namespace,
		Status:        "Progressing",
		PrimaryReason: "Rollout " + reasonCode,
		Severity:      "critical",
		Summary: []string{msg},
		Evidence: []Evidence{
			{Label: "Desired", Value: fmt.Sprintf("%d", signals.DesiredReplicas)},
			{Label: "Updated", Value: fmt.Sprintf("%d", signals.UpdatedReplicas)},
			{Label: "Available", Value: fmt.Sprintf("%d", signals.AvailableReplicas)},
		},
		FixCommands: []FixCommand{
			{
				Description: "Check rollout status",
				Command:     fmt.Sprintf("kubectl rollout status deployment %s -n %s", signals.DeploymentName, signals.Namespace),
			},
		},
		NextChecks: []string{
			fmt.Sprintf("kubectl describe deployment %s -n %s", signals.DeploymentName, signals.Namespace),
		},
		RecentEvents: extractEventStrings(signals.Events, 3),
	}

	if reasonCode == "ROLLOUT_HEALTHY" {
		res.Status = "Healthy"
		res.Severity = "healthy"
	}

	res.Findings = []Finding{
		{
			Category:       "Rollout",
			ReasonCode:     reasonCode,
			Confidence:     "high",
			AffectedObject: resource,
			Message:        msg,
			Evidence:       res.Evidence,
			FixCommands:    res.FixCommands,
			NextChecks:     res.NextChecks,
		},
	}

	// Also embed failing pod root cause if we have one
	if signals.FailingPodSignals != nil {
		podRes := AnalyzePod(signals.FailingPodSignals)
		res.Findings = append(res.Findings, podRes.Findings...)
		res.Evidence = append(res.Evidence, Evidence{Label: "Failing Pod", Value: signals.FailingPodName})
		if reasonCode == "PROGRESS_DEADLINE_EXCEEDED" || reasonCode == "UNAVAILABLE_REPLICAS" {
			res.PrimaryReason = podRes.PrimaryReason
			res.Summary = append(res.Summary, podRes.Summary...)
		}
	}

	return res
}
