package analyzer

import (
	"fmt"
	"strings"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// AnalyzeResourceQuota produces a result for a ResourceQuota
func AnalyzeResourceQuota(signals *kube.ResourceQuotaSignals) AnalysisResult {
	resource := "resourcequota/" + signals.Name

	var overLimit []string
	var approachingLimit []string
	var evidence []Evidence

	for k, hardVal := range signals.Hard {
		if usedVal, exists := signals.Used[k]; exists {
			evidence = append(evidence, Evidence{
				Label: string(k),
				Value: fmt.Sprintf("%v used / %v hard", usedVal.String(), hardVal.String()),
			})

			// Compare resource quantities
			if usedVal.Cmp(hardVal) >= 0 {
				overLimit = append(overLimit, string(k))
			} else if float64(usedVal.MilliValue()) >= 0.8*float64(hardVal.MilliValue()) {
				approachingLimit = append(approachingLimit, string(k))
			}
		}
	}

	if len(overLimit) > 0 {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Exceeded",
			PrimaryReason: "Quota exceeded",
			Severity:      "critical",
			Summary: []string{
				fmt.Sprintf("Resource limits exceeded for: %s", strings.Join(overLimit, ", ")),
			},
			Evidence: evidence,
			NextChecks: []string{
				fmt.Sprintf("kubectl describe resourcequota %s -n %s", signals.Name, signals.Namespace),
			},
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	if len(approachingLimit) > 0 {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Warning",
			PrimaryReason: "Approaching quota limit",
			Severity:      "warning",
			Summary: []string{
				fmt.Sprintf("High usage for: %s", strings.Join(approachingLimit, ", ")),
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
		PrimaryReason: "Usage within limits",
		Severity:      "healthy",
		Summary: []string{
			"All resources are within their hard limits.",
		},
		Evidence: evidence,
	}
	res.Findings = append(res.Findings, resultToFinding(res))
	return res
}
