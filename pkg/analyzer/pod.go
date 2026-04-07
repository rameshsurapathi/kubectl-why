package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// AnalyzePod acts as the "brain" for Pod resources. 
// It takes the raw data gathered from the cluster (the PodSignals)
// and runs it through a gauntlet of heuristic rules to figure out exactly what went wrong.
func AnalyzePod(signals *kube.PodSignals) AnalysisResult {
	
	// 1. Evaluate against known failure patterns (The 'Gauntlet')
	// We iterate through every registered rule (e.g. OOMKilledRule, ImagePullBackOffRule).
	// Because rules are in the 'registry' in priority order, the most critical or 
	// specific rules are evaluated first.
	for _, rule := range registry {
		
		// If the rule's specific condition matches the pod's data (e.g., ExitCode == 137)...
		if rule.Match(signals) {
			// Ask the rule to generate the final human-readable diagnosis report.
			return rule.Analyze(signals)
		}
	}

	// 2. Fallback Mechanism
	// If the pod is failing but NONE of our rules match the data, we return a 
	// generic "Unknown" result. This prevents the CLI from crashing or printing nothing,
	// No failure rules matched. Check if the pod is actually healthy!
	if signals.Phase == "Running" || signals.Phase == "Succeeded" {
		statusStr := "Running"
		primaryReason := "Healthy"
		summaryText := "All containers are running and ready."
		if signals.Phase == "Succeeded" {
			statusStr = "Succeeded"
			primaryReason = "Completed"
			summaryText = "All containers have completed successfully."
		}
		return AnalysisResult{
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
	}

	// True fallback if the pod is failing for an unknown reason
	return AnalysisResult{
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
