package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
	corev1 "k8s.io/api/core/v1"
)

func AnalyzeService(signals *kube.ServiceSignals) AnalysisResult {
	resource := "service/" + signals.Name

	if signals.Type == "ExternalName" {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "ExternalName",
			PrimaryReason: "Service proxies to an external DNS name",
			Severity:      "info",
			Summary: []string{
				fmt.Sprintf("This service directs traffic to external name: %s.", signals.ExternalName),
			},
			Evidence: []Evidence{
				{Label: "Type", Value: signals.Type},
				{Label: "ExternalName", Value: signals.ExternalName},
			},
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	var reasonCode string
	var msg string
	severity := "critical"
	status := "Degraded"

	if len(signals.Selector) == 0 {
		reasonCode = "NO_SELECTOR"
		msg = "Service has no selector. It requires manual Endpoint creation."
		severity = "warning"
	} else if len(signals.MatchingPods) == 0 {
		reasonCode = "NO_MATCHING_PODS"
		msg = "Service selector matches no pods in this namespace."
	} else {
		// Check if any pod is ready
		readyPods := 0
		failingPods := 0
		for _, pod := range signals.MatchingPods {
			isReady := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					isReady = true
					break
				}
			}
			if isReady {
				readyPods++
			} else {
				failingPods++
			}
		}

		if readyPods == 0 {
			reasonCode = "NO_READY_PODS"
			msg = fmt.Sprintf("Service matches %d pods, but none are ready.", len(signals.MatchingPods))
		} else if failingPods > 0 {
			reasonCode = "SOME_PODS_FAILING"
			msg = fmt.Sprintf("Service matches %d pods (%d ready, %d failing).", len(signals.MatchingPods), readyPods, failingPods)
			severity = "warning"
		} else {
			reasonCode = "SERVICE_HEALTHY"
			msg = "Service is healthy and all backing pods are ready."
			severity = "healthy"
			status = "Healthy"
		}
	}

	res := AnalysisResult{
		SchemaVersion: "v2",
		Resource:      resource,
		Namespace:     signals.Namespace,
		Status:        status,
		PrimaryReason: reasonCode,
		Severity:      severity,
		Summary:       []string{msg},
		Evidence: []Evidence{
			{Label: "Type", Value: signals.Type},
			{Label: "Selector", Value: fmt.Sprintf("%v", signals.Selector)},
		},
		FixCommands: []FixCommand{
			{
				Description: "Check endpoints",
				Command:     fmt.Sprintf("kubectl get endpoints %s -n %s", signals.Name, signals.Namespace),
			},
		},
		NextChecks: []string{
			fmt.Sprintf("kubectl describe service %s -n %s", signals.Name, signals.Namespace),
		},
	}

	res.Findings = []Finding{
		{
			Category:       "Service",
			ReasonCode:     reasonCode,
			Confidence:     "high",
			AffectedObject: resource,
			Message:        msg,
			Evidence:       res.Evidence,
			FixCommands:    res.FixCommands,
			NextChecks:     res.NextChecks,
		},
	}

	return res
}
