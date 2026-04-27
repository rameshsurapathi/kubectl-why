package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// AnalyzePod acts as the "brain" for Pod resources.
// It takes the raw data gathered from the cluster (the PodSignals)
// and runs it through a gauntlet of heuristic rules to figure out exactly what went wrong.
func AnalyzePod(signals *kube.PodSignals) AnalysisResult {
	var matchedResults []AnalysisResult

	// 1. Evaluate against known failure patterns
	for _, rule := range registry {
		if rule.Match(signals) {
			matchedResults = append(matchedResults, rule.Analyze(signals))
		}
	}

	if len(matchedResults) > 0 {
		// Use the first matched result as the primary basis
		primary := matchedResults[0]
		primary.SchemaVersion = "v2"
		primary.Findings = nil

		// Convert all matched results to findings
		for _, res := range matchedResults {
			finding := resultToFinding(res)
			if len(res.Findings) > 0 {
				primary.Findings = append(primary.Findings, res.Findings...)
			} else {
				primary.Findings = append(primary.Findings, finding)
			}
		}

		return primary
	}

	// 2. Fallback Mechanism
	if signals.Phase == "Running" || signals.Phase == "Succeeded" {
		statusStr := "Running"
		primaryReason := "Healthy"
		summaryText := "All containers are running and ready."
		if signals.Phase == "Succeeded" {
			statusStr = "Succeeded"
			primaryReason = "Completed"
			summaryText = "All containers have completed successfully."
		}
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      "pod/" + signals.PodName,
			Namespace:     signals.Namespace,
			Status:        statusStr,
			PrimaryReason: primaryReason,
			Severity:      "healthy",
			Summary: []string{
				summaryText,
			},
			Evidence:   buildHealthyEvidence(signals),
			NextChecks: nil, // Nothing to do!
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	// True fallback if the pod is failing for an unknown reason
	res := AnalysisResult{
		SchemaVersion: "v2",
		Resource:      "pod/" + signals.PodName,
		Namespace:     signals.Namespace,
		Status:        signals.Phase,
		PrimaryReason: "Unknown",
		Severity:      "warning",
		Summary: []string{
			"Could not determine a specific failure reason using the current rules.",
			"Review the exact kubectl describe output manually.",
		},
		NextChecks: []string{
			fmt.Sprintf("kubectl describe pod %s -n %s", signals.PodName, signals.Namespace),
		},
	}
	res.Findings = append(res.Findings, resultToFinding(res))
	return res
}

func resultToFinding(res AnalysisResult) Finding {
	confidence := "high"
	if res.Severity == "warning" {
		confidence = "medium"
	}
	msg := ""
	if len(res.Summary) > 0 {
		msg = res.Summary[0]
	}
	return Finding{
		Category:       "Pod",
		ReasonCode:     res.PrimaryReason,
		Confidence:     confidence,
		AffectedObject: res.Resource,
		Message:        msg,
		Evidence:       res.Evidence,
		FixCommands:    res.FixCommands,
		NextChecks:     res.NextChecks,
	}
}

func buildHealthyEvidence(signals *kube.PodSignals) []Evidence {
	var ev []Evidence

	// Show container count
	readyCount := 0
	for _, c := range append(signals.Containers, signals.InitContainers...) {
		if c.Ready {
			readyCount++
		}
	}
	ev = append(ev, Evidence{
		Label: "Containers",
		Value: fmt.Sprintf("%d/%d ready", readyCount, len(signals.Containers)+len(signals.InitContainers)),
	})

	// Show restart count if non-zero
	totalRestarts := int32(0)
	for _, c := range append(signals.Containers, signals.InitContainers...) {
		totalRestarts += c.RestartCount
	}
	if totalRestarts > 0 {
		ev = append(ev, Evidence{
			Label: "Restarts",
			Value: fmt.Sprintf("%d (pod is healthy now but has restarted)", totalRestarts),
		})
	}

	if signals.Age != "" {
		// Show age
		ev = append(ev, Evidence{
			Label: "Age",
			Value: signals.Age,
		})
	}

	return ev
}
