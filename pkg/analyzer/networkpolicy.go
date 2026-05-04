package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
	networkingv1 "k8s.io/api/networking/v1"
)

// AnalyzeNetworkPolicy produces a result for a NetworkPolicy
func AnalyzeNetworkPolicy(signals *kube.NetworkPolicySignals) AnalysisResult {
	resource := "networkpolicy/" + signals.Name

	evidence := []Evidence{
		{
			Label: "Matched Pods",
			Value: fmt.Sprintf("%d", signals.MatchedPods),
		},
		{
			Label: "Has Ingress Rules",
			Value: fmt.Sprintf("%v", signals.HasIngress),
		},
		{
			Label: "Has Egress Rules",
			Value: fmt.Sprintf("%v", signals.HasEgress),
		},
	}

	if signals.MatchedPods == 0 {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Warning",
			PrimaryReason: "No pods matched",
			Severity:      "warning",
			Summary: []string{
				"This NetworkPolicy does not select any running pods. It might have an incorrect PodSelector.",
			},
			Evidence: evidence,
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	isIngressDeny := false
	isEgressDeny := false

	for _, pt := range signals.PolicyTypes {
		if pt == networkingv1.PolicyTypeIngress && !signals.HasIngress {
			isIngressDeny = true
		}
		if pt == networkingv1.PolicyTypeEgress && !signals.HasEgress {
			isEgressDeny = true
		}
	}

	if isIngressDeny && isEgressDeny {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Warning",
			PrimaryReason: "Default Deny All",
			Severity:      "warning",
			Summary: []string{
				"This NetworkPolicy blocks ALL Ingress and Egress traffic for the selected pods.",
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
		PrimaryReason: "Valid Network Policy",
		Severity:      "healthy",
		Summary: []string{
			fmt.Sprintf("NetworkPolicy matches %d pods.", signals.MatchedPods),
		},
		Evidence: evidence,
	}
	res.Findings = append(res.Findings, resultToFinding(res))
	return res
}
