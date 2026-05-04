package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// AnalyzeDNS produces a result for the cluster DNS health
func AnalyzeDNS(signals *kube.DNSSignals) AnalysisResult {
	resource := "cluster/dns"
	evidence := []Evidence{
		{Label: "Service Found", Value: fmt.Sprintf("%v", signals.ServiceFound)},
		{Label: "Endpoints Ready", Value: fmt.Sprintf("%d/%d", signals.EndpointsReady, signals.TotalEndpoints)},
	}

	if !signals.ServiceFound {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     "kube-system",
			Status:        "Degraded",
			PrimaryReason: "kube-dns service not found",
			Severity:      "critical",
			Summary: []string{
				"The cluster DNS service 'kube-dns' was not found in the 'kube-system' namespace.",
			},
			Evidence: evidence,
			NextChecks: []string{
				"kubectl get svc -n kube-system",
			},
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	if signals.EndpointsReady == 0 {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     "kube-system",
			Status:        "Degraded",
			PrimaryReason: "No DNS pods running",
			Severity:      "critical",
			Summary: []string{
				"The 'kube-dns' service has no ready endpoints.",
				"CoreDNS pods are likely failing.",
			},
			Evidence: evidence,
			NextChecks: []string{
				"kubectl get pods -n kube-system -l k8s-app=kube-dns",
			},
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	res := AnalysisResult{
		SchemaVersion: "v2",
		Resource:      resource,
		Namespace:     "kube-system",
		Status:        "Healthy",
		PrimaryReason: "DNS is healthy",
		Severity:      "healthy",
		Summary: []string{
			"Cluster DNS is running and has ready endpoints.",
		},
		Evidence: evidence,
	}
	res.Findings = append(res.Findings, resultToFinding(res))
	return res
}
