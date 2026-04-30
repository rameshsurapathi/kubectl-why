package analyzer

import (
	"sort"
	"strconv"
)

// ScanResult is the v3 namespace-level diagnosis output.
type ScanResult struct {
	SchemaVersion string           `json:"schemaVersion"`
	Namespace     string           `json:"namespace"`
	Status        string           `json:"status"`
	Summary       []string         `json:"summary"`
	Results       []AnalysisResult `json:"results"`
}

// BuildScanResult ranks individual diagnoses into one namespace-level result.
func BuildScanResult(namespace string, results []AnalysisResult) ScanResult {
	ranked := append([]AnalysisResult(nil), results...)
	sort.SliceStable(ranked, func(i, j int) bool {
		left := severityRank(ranked[i].Severity)
		right := severityRank(ranked[j].Severity)
		if left != right {
			return left > right
		}
		return ranked[i].Resource < ranked[j].Resource
	})

	critical, warning, healthy := 0, 0, 0
	for _, result := range ranked {
		switch result.Severity {
		case "critical":
			critical++
		case "warning":
			warning++
		case "healthy":
			healthy++
		}
	}

	status := "Healthy"
	switch {
	case critical > 0:
		status = "Critical"
	case warning > 0:
		status = "Warning"
	}

	return ScanResult{
		SchemaVersion: "v3",
		Namespace:     namespace,
		Status:        status,
		Summary: []string{
			pluralize(critical, "critical issue"),
			pluralize(warning, "warning"),
			pluralize(healthy, "healthy resource"),
		},
		Results: ranked,
	}
}

func severityRank(severity string) int {
	switch severity {
	case "critical":
		return 3
	case "warning":
		return 2
	case "healthy":
		return 1
	default:
		return 0
	}
}

func pluralize(count int, singular string) string {
	if count == 1 {
		return "1 " + singular
	}
	return strconv.Itoa(count) + " " + singular + "s"
}
