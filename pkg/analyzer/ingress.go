package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// AnalyzeIngress evaluates the health of an Ingress resource.
func AnalyzeIngress(signals *kube.IngressSignals) AnalysisResult {
	resource := "ingress/" + signals.Name
	var findings []Finding

	// Check LoadBalancer status (if it has no IPs/hostnames, it might not be processed by a controller)
	hasAddress := len(signals.Ingress.Status.LoadBalancer.Ingress) > 0
	if !hasAddress {
		findings = append(findings, Finding{
			Category:       "Ingress",
			ReasonCode:     "INGRESS_NO_ADDRESS",
			Confidence:     "high",
			AffectedObject: resource,
			Message:        "Ingress has not been assigned an address by the ingress controller.",
			FixCommands: []FixCommand{
				{
					Description: "Check ingress controller logs",
					Command:     "kubectl logs -n <ingress-controller-namespace> -l app.kubernetes.io/name=ingress-nginx",
					SafetyLevel: "inspect",
				},
			},
		})
	}

	// Check for warning events
	for _, e := range signals.Events {
		if e.Type == "Warning" {
			findings = append(findings, Finding{
				Category:       "Ingress",
				ReasonCode:     "INGRESS_WARNING_EVENT",
				Confidence:     "medium",
				AffectedObject: resource,
				Message:        fmt.Sprintf("Warning Event: %s", e.Message),
				Evidence: []Evidence{
					{Label: "Reason", Value: e.Reason},
				},
			})
		}
	}

	status := "Healthy"
	severity := "healthy"
	primaryReason := "Ingress is configured"
	summary := "The Ingress resource is well-formed."
	var fixCommands []FixCommand

	if len(findings) > 0 {
		f := findings[0]
		primaryReason = f.ReasonCode
		summary = f.Message
		fixCommands = f.FixCommands
		if f.ReasonCode == "INGRESS_NO_ADDRESS" {
			severity = "warning"
			status = "Pending"
			summary = "Ingress is waiting for an address from the controller."
		} else {
			severity = "warning"
			status = "Warning"
		}
	} else {
		findings = append(findings, Finding{
			Category:       "Ingress",
			ReasonCode:     "INGRESS_HEALTHY",
			Confidence:     "high",
			AffectedObject: resource,
			Message:        summary,
		})
	}

	// Gather services targeted
	services := signals.GetIngressBackingServices()
	svcList := fmt.Sprintf("%d backing services", len(services))

	return AnalysisResult{
		SchemaVersion: "v2",
		Resource:      resource,
		Namespace:     signals.Namespace,
		Status:        status,
		PrimaryReason: primaryReason,
		Severity:      severity,
		Summary:       []string{summary},
		Evidence: []Evidence{
			{Label: "Targets", Value: svcList},
			{Label: "Has Address", Value: fmt.Sprintf("%v", hasAddress)},
		},
		FixCommands:  fixCommands,
		NextChecks: []string{
			fmt.Sprintf("kubectl describe ingress %s -n %s", signals.Name, signals.Namespace),
		},
		RecentEvents: extractEventStrings(signals.Events, 3),
		Findings:     findings,
	}
}
