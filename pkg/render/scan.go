package render

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	
	groups := map[string][]analyzer.AnalysisResult{
		"Traffic Impact (Services, Ingress)":    {},
		"Rollout Impact (Deployments, Rollouts)": {},
		"Batch Impact (Jobs, CronJobs)":         {},
		"Storage Impact (PVCs)":                 {},
		"Scheduling Impact (Nodes, Pending)":    {},
		"Workload Failures (Pods)":              {},
	}

	for _, item := range result.Results {
		if item.Severity == "healthy" && !showHealthy {
			continue
		}
		
		// Categorize
		category := "Workload Failures (Pods)"
		if strings.HasPrefix(item.Resource, "service/") || strings.HasPrefix(item.Resource, "ingress/") {
			category = "Traffic Impact (Services, Ingress)"
		} else if strings.HasPrefix(item.Resource, "deployment/") || strings.HasPrefix(item.Resource, "rollout/") {
			category = "Rollout Impact (Deployments, Rollouts)"
		} else if strings.HasPrefix(item.Resource, "job/") || strings.HasPrefix(item.Resource, "cronjob/") {
			category = "Batch Impact (Jobs, CronJobs)"
		} else if strings.HasPrefix(item.Resource, "pvc/") {
			category = "Storage Impact (PVCs)"
		} else if strings.HasPrefix(item.Resource, "node/") || item.Status == "Pending — cannot be scheduled" || item.Status == "Pending" {
			category = "Scheduling Impact (Nodes, Pending)"
		}
		
		groups[category] = append(groups[category], item)
	}

	printed := 0
	orderedCategories := []string{
		"Traffic Impact (Services, Ingress)",
		"Rollout Impact (Deployments, Rollouts)",
		"Batch Impact (Jobs, CronJobs)",
		"Storage Impact (PVCs)",
		"Scheduling Impact (Nodes, Pending)",
		"Workload Failures (Pods)",
	}

	for _, cat := range orderedCategories {
		items := groups[cat]
		if len(items) == 0 {
			continue
		}
		fmt.Printf("\n    %s\n", lipgloss.NewStyle().Foreground(colorWhite).Underline(true).Render(cat))
		for _, item := range items {
			printScanItem(item)
			printed++
		}
	}

	if printed == 0 {
		fmt.Println(nextCheckStyle.Render("\n→  No unhealthy resources found."))
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
