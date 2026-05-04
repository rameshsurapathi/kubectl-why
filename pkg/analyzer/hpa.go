package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
)

// AnalyzeHPA produces a result for an HPA
func AnalyzeHPA(signals *kube.HPASignals) AnalysisResult {
	resource := "hpa/" + signals.Name

	min := int32(1)
	if signals.MinReplicas != nil {
		min = *signals.MinReplicas
	}

	evidence := []Evidence{
		{
			Label: "Replicas",
			Value: fmt.Sprintf("%d current / %d desired", signals.CurrentReplicas, signals.DesiredReplicas),
		},
		{
			Label: "Limits",
			Value: fmt.Sprintf("Min: %d / Max: %d", min, signals.MaxReplicas),
		},
	}

	var unableToScaleReason string
	var scalingActiveReason string
	var scalingLimitedReason string

	for _, cond := range signals.Conditions {
		if cond.Type == autoscalingv2.AbleToScale && cond.Status == corev1.ConditionFalse {
			unableToScaleReason = cond.Message
		}
		if cond.Type == autoscalingv2.ScalingActive && cond.Status == corev1.ConditionFalse {
			scalingActiveReason = cond.Message
		}
		if cond.Type == autoscalingv2.ScalingLimited && cond.Status == corev1.ConditionTrue {
			scalingLimitedReason = cond.Message
		}
	}

	if unableToScaleReason != "" || scalingActiveReason != "" {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Degraded",
			PrimaryReason: "HPA cannot scale",
			Severity:      "critical",
			Summary: []string{
				"The HorizontalPodAutoscaler is unable to function correctly.",
			},
			Evidence: evidence,
			NextChecks: []string{
				fmt.Sprintf("kubectl describe hpa %s -n %s", signals.Name, signals.Namespace),
			},
		}
		if unableToScaleReason != "" {
			res.Summary = append(res.Summary, unableToScaleReason)
		}
		if scalingActiveReason != "" {
			res.Summary = append(res.Summary, scalingActiveReason)
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	if scalingLimitedReason != "" {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Warning",
			PrimaryReason: "Scaling is limited",
			Severity:      "warning",
			Summary: []string{
				"The HPA is hitting its min or max bounds.",
				scalingLimitedReason,
			},
			Evidence: evidence,
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	// Check max bounds manually as a fallback
	if signals.CurrentReplicas >= signals.MaxReplicas && signals.DesiredReplicas >= signals.MaxReplicas {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Warning",
			PrimaryReason: "Max replicas reached",
			Severity:      "warning",
			Summary: []string{
				"The HPA wants to scale up but has hit the MaxReplicas limit.",
			},
			Evidence: evidence,
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	res := AnalysisResult{
		SchemaVersion: "v2",
		Resource:      resource,
		Namespace:     signals.Namespace,
		Status:        "Healthy",
		PrimaryReason: "HPA functioning normally",
		Severity:      "healthy",
		Summary: []string{
			"The HPA is active and within limits.",
		},
		Evidence: evidence,
	}
	res.Findings = append(res.Findings, resultToFinding(res))
	return res
}
