package render

import (
	"encoding/json"
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
)

// ScanJSON renders a namespace-level scan as JSON.
func ScanJSON(result analyzer.ScanResult) error {
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// ScanText renders a compact namespace-level diagnosis.
func ScanText(result analyzer.ScanResult, showHealthy bool) error {
	fmt.Println()
	fmt.Println(headerBox.Render(fmt.Sprintf("%s  %s",
		resourceStyle.Render("namespace/"+result.Namespace),
		namespaceStyle.Render("· scan"),
	)))

	statusResult := analyzer.AnalysisResult{
		Status:   result.Status,
		Severity: scanSeverity(result.Status),
	}
	printStatusLine(statusResult)
	fmt.Println()

	if len(result.Summary) > 0 {
		fmt.Printf("  %s Summary\n", sectionDot.Render("●"))
		for _, line := range result.Summary {
			fmt.Println(nextCheckStyle.Render("→  " + line))
		}
		fmt.Println()
	}

	fmt.Printf("  %s Findings\n", sectionDot.Render("●"))
	printed := 0
	for _, item := range result.Results {
		if item.Severity == "healthy" && !showHealthy {
			continue
		}
		printScanItem(item)
		printed++
	}
	if printed == 0 {
		fmt.Println(nextCheckStyle.Render("→  No unhealthy resources found."))
	}
	fmt.Println()

	return nil
}

func printScanItem(result analyzer.AnalysisResult) {
	reason := result.PrimaryReason
	if len(result.Findings) > 0 && result.Findings[0].ReasonCode != "" {
		reason = result.Findings[0].ReasonCode
	}

	confidence := ""
	if len(result.Findings) > 0 && result.Findings[0].Confidence != "" {
		confidence = " · confidence " + result.Findings[0].Confidence
	}

	fmt.Printf("    %s  %s  %s%s\n",
		scanSeverityMarker(result.Severity),
		result.Resource,
		reason,
		confidence,
	)
	if len(result.Summary) > 0 {
		fmt.Println(logLine.Render(result.Summary[0]))
	}
}

func scanSeverity(status string) string {
	switch status {
	case "Critical":
		return "critical"
	case "Warning":
		return "warning"
	default:
		return "healthy"
	}
}

func scanSeverityMarker(severity string) string {
	switch severity {
	case "critical":
		return statusCritical.Render("critical")
	case "warning":
		return statusWarning.Render("warning ")
	case "healthy":
		return statusHealthy.Render("healthy ")
	default:
		return severity
	}
}
