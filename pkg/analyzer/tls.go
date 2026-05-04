package analyzer

import (
	"fmt"
	"time"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// AnalyzeTLS produces a result for a TLS Secret
func AnalyzeTLS(signals *kube.TLSSignals) AnalysisResult {
	resource := "secret/" + signals.SecretName

	if signals.ParseError != "" {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Degraded",
			PrimaryReason: "Invalid TLS Secret",
			Severity:      "critical",
			Summary: []string{
				fmt.Sprintf("Failed to parse certificate: %s", signals.ParseError),
			},
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	evidence := []Evidence{
		{Label: "Subject", Value: signals.Subject},
		{Label: "Issuer", Value: signals.Issuer},
		{Label: "Valid Until", Value: signals.NotAfter.Format(time.RFC3339)},
	}

	daysRemaining := int(time.Until(signals.NotAfter).Hours() / 24)

	if !signals.IsValid {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Expired",
			PrimaryReason: "Certificate expired",
			Severity:      "critical",
			Summary: []string{
				"The TLS certificate is not valid or has expired.",
			},
			Evidence: evidence,
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	if daysRemaining < 14 {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Warning",
			PrimaryReason: "Expiring soon",
			Severity:      "warning",
			Summary: []string{
				fmt.Sprintf("Certificate expires in %d days.", daysRemaining),
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
		PrimaryReason: "Valid certificate",
		Severity:      "healthy",
		Summary: []string{
			fmt.Sprintf("Certificate is valid for %d days.", daysRemaining),
		},
		Evidence: evidence,
	}
	res.Findings = append(res.Findings, resultToFinding(res))
	return res
}
