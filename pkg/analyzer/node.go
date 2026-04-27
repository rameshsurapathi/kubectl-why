package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
	corev1 "k8s.io/api/core/v1"
)

func AnalyzeNode(signals *kube.NodeSignals) AnalysisResult {
	resource := "node/" + signals.Name

	var findings []Finding

	isReady := false
	for _, cond := range signals.Conditions {
		if cond.Type == corev1.NodeReady {
			if cond.Status == corev1.ConditionTrue {
				isReady = true
			} else {
				findings = append(findings, Finding{
					Category:       "Node",
					ReasonCode:     "NODE_NOT_READY",
					Confidence:     "high",
					AffectedObject: resource,
					Message:        fmt.Sprintf("Node is not ready: %s", cond.Reason),
					Evidence: []Evidence{
						{Label: "Condition", Value: string(cond.Type)},
						{Label: "Reason", Value: cond.Reason},
						{Label: "Message", Value: cond.Message},
					},
				})
			}
		} else if cond.Status == corev1.ConditionTrue {
			// Pressure conditions (MemoryPressure, DiskPressure, PIDPressure, NetworkUnavailable)
			findings = append(findings, Finding{
				Category:       "Node",
				ReasonCode:     "NODE_" + string(cond.Type),
				Confidence:     "high",
				AffectedObject: resource,
				Message:        fmt.Sprintf("Node has %s: %s", cond.Type, cond.Reason),
				Evidence: []Evidence{
					{Label: "Condition", Value: string(cond.Type)},
					{Label: "Reason", Value: cond.Reason},
					{Label: "Message", Value: cond.Message},
				},
			})
		}
	}

	if signals.Unschedulable {
		findings = append(findings, Finding{
			Category:       "Node",
			ReasonCode:     "NODE_UNSCHEDULABLE",
			Confidence:     "high",
			AffectedObject: resource,
			Message:        "Node is cordoned and unschedulable.",
			FixCommands: []FixCommand{
				{
					Description: "Uncordon the node",
					Command:     fmt.Sprintf("kubectl uncordon %s", signals.Name),
				},
			},
		})
	}

	for _, e := range signals.Events {
		findings = append(findings, Finding{
			Category:       "Node",
			ReasonCode:     "NODE_WARNING_EVENT",
			Confidence:     "medium",
			AffectedObject: resource,
			Message:        fmt.Sprintf("Warning Event: %s", e.Message),
			Evidence: []Evidence{
				{Label: "Reason", Value: e.Reason},
			},
		})
	}

	status := "Healthy"
	severity := "healthy"
	primaryReason := "Node is healthy"
	summary := "The node is ready and accepting pods."

	if len(findings) > 0 {
		f := findings[0]
		primaryReason = f.ReasonCode
		summary = f.Message
		if !isReady || f.ReasonCode == "NODE_DiskPressure" || f.ReasonCode == "NODE_MemoryPressure" || f.ReasonCode == "NODE_PIDPressure" {
			severity = "critical"
			status = "Degraded"
		} else {
			severity = "warning"
			status = "Warning"
		}
	} else {
		findings = append(findings, Finding{
			Category:       "Node",
			ReasonCode:     "NODE_HEALTHY",
			Confidence:     "high",
			AffectedObject: resource,
			Message:        summary,
		})
	}

	return AnalysisResult{
		SchemaVersion: "v2",
		Resource:      resource,
		Namespace:     "cluster-scoped",
		Status:        status,
		PrimaryReason: primaryReason,
		Severity:      severity,
		Summary:       []string{summary},
		Evidence: []Evidence{
			{Label: "Unschedulable", Value: fmt.Sprintf("%v", signals.Unschedulable)},
		},
		FixCommands:  findings[0].FixCommands,
		NextChecks: []string{
			fmt.Sprintf("kubectl describe node %s", signals.Name),
		},
		RecentEvents: extractEventStrings(signals.Events, 3),
		Findings:     findings,
	}
}
