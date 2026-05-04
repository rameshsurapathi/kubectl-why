package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// AnalyzePDB produces a result for a PodDisruptionBudget
func AnalyzePDB(signals *kube.PDBSignals) AnalysisResult {
	resource := "pdb/" + signals.Name

	evidence := []Evidence{
		{
			Label: "Disruptions Allowed",
			Value: fmt.Sprintf("%d", signals.DisruptionsAllowed),
		},
		{
			Label: "Healthy Pods",
			Value: fmt.Sprintf("%d current / %d desired", signals.CurrentHealthy, signals.DesiredHealthy),
		},
		{
			Label: "Expected Pods",
			Value: fmt.Sprintf("%d", signals.ExpectedPods),
		},
	}

	if signals.DisruptionsAllowed == 0 && signals.ExpectedPods > 0 {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Warning",
			PrimaryReason: "0 Disruptions Allowed",
			Severity:      "warning",
			Summary: []string{
				"This PDB blocks all voluntary disruptions (e.g., node drains).",
				"No pods can be evicted safely.",
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
		PrimaryReason: "Disruptions allowed",
		Severity:      "healthy",
		Summary: []string{
			"The PodDisruptionBudget allows evictions.",
		},
		Evidence: evidence,
	}
	res.Findings = append(res.Findings, resultToFinding(res))
	return res
}
