package render

import (
	"encoding/json"
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
)

// ServiceTraceJSON renders Service relationship diagnosis as JSON.
func ServiceTraceJSON(result analyzer.ServiceTraceResult) error {
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// ServiceTraceText renders Service relationship diagnosis for humans.
func ServiceTraceText(result analyzer.ServiceTraceResult, showHealthy bool) error {
	fmt.Println()
	fmt.Println(headerBox.Render(fmt.Sprintf("%s  %s",
		resourceStyle.Render(result.Resource),
		namespaceStyle.Render("· trace"),
	)))

	printStatusLine(analyzer.AnalysisResult{
		Status:   result.Status,
		Severity: scanSeverity(result.Status),
	})
	fmt.Println()

	fmt.Printf("  %s Service\n", sectionDot.Render("●"))
	printTraceLine(result.Service)
	fmt.Println()

	fmt.Printf("  %s Traffic Path\n", sectionDot.Render("●"))
	fmt.Println(nextCheckStyle.Render(fmt.Sprintf(
		"→  %d/%d endpoints ready",
		result.ReadyEndpointCount,
		result.EndpointCount,
	)))
	for _, line := range result.Summary {
		fmt.Println(nextCheckStyle.Render("→  " + line))
	}
	fmt.Println()

	fmt.Printf("  %s Backing Pods\n", sectionDot.Render("●"))
	printed := 0
	for _, pod := range result.BackingPods {
		if pod.Severity == "healthy" && !showHealthy {
			continue
		}
		printTraceLine(pod)
		printed++
	}
	if printed == 0 {
		if len(result.BackingPods) == 0 {
			fmt.Println(nextCheckStyle.Render("→  No Kubernetes pod backends found for this Service."))
		} else {
			fmt.Println(nextCheckStyle.Render("→  No unhealthy backing pods found."))
		}
	}
	fmt.Println()

	return nil
}

func printTraceLine(result analyzer.AnalysisResult) {
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

// DeploymentTraceJSON renders Deployment relationship diagnosis as JSON.
func DeploymentTraceJSON(result analyzer.DeploymentTraceResult) error {
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// DeploymentTraceText renders Deployment relationship diagnosis for humans.
func DeploymentTraceText(result analyzer.DeploymentTraceResult, showHealthy bool) error {
	fmt.Println()
	fmt.Println(headerBox.Render(fmt.Sprintf("%s  %s",
		resourceStyle.Render(result.Resource),
		namespaceStyle.Render("· trace"),
	)))

	printStatusLine(analyzer.AnalysisResult{
		Status:   result.Status,
		Severity: scanSeverity(result.Status),
	})
	fmt.Println()

	fmt.Printf("  %s Deployment\n", sectionDot.Render("●"))
	printTraceLine(result.Deployment)
	fmt.Println()

	if len(result.ReplicaSets) > 0 {
		fmt.Printf("  %s ReplicaSets\n", sectionDot.Render("●"))
		printed := 0
		for _, rs := range result.ReplicaSets {
			if rs.Severity == "healthy" && !showHealthy {
				continue
			}
			printTraceLine(rs)
			printed++
		}
		if printed == 0 {
			fmt.Println(nextCheckStyle.Render("→  No unhealthy ReplicaSets found."))
		}
		fmt.Println()
	}

	fmt.Printf("  %s Pods\n", sectionDot.Render("●"))
	printedPods := 0
	for _, pod := range result.Pods {
		if pod.Severity == "healthy" && !showHealthy {
			continue
		}
		printTraceLine(pod)
		printedPods++
	}
	if printedPods == 0 {
		if len(result.Pods) == 0 {
			fmt.Println(nextCheckStyle.Render("→  No Pods found for this Deployment."))
		} else {
			fmt.Println(nextCheckStyle.Render("→  No unhealthy Pods found."))
		}
	}
	fmt.Println()

	if len(result.Dependencies) > 0 {
		fmt.Printf("  %s Dependencies\n", sectionDot.Render("●"))
		printedDeps := 0
		for _, dep := range result.Dependencies {
			if dep.Severity == "healthy" && !showHealthy {
				continue
			}
			printTraceLine(dep)
			printedDeps++
		}
		if printedDeps == 0 {
			fmt.Println(nextCheckStyle.Render("→  No missing/unhealthy dependencies found."))
		}
		fmt.Println()
	}

	return nil
}

// IngressTraceJSON renders Ingress relationship diagnosis as JSON.
func IngressTraceJSON(result analyzer.IngressTraceResult) error {
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// IngressTraceText renders Ingress relationship diagnosis for humans.
func IngressTraceText(result analyzer.IngressTraceResult, showHealthy bool) error {
	fmt.Println()
	fmt.Println(headerBox.Render(fmt.Sprintf("%s  %s",
		resourceStyle.Render(result.Resource),
		namespaceStyle.Render("· trace"),
	)))

	printStatusLine(analyzer.AnalysisResult{
		Status:   result.Status,
		Severity: scanSeverity(result.Status),
	})
	fmt.Println()

	fmt.Printf("  %s Ingress\n", sectionDot.Render("●"))
	printTraceLine(result.Ingress)
	fmt.Println()

	if len(result.ServiceTraces) > 0 {
		fmt.Printf("  %s Backend Services\n", sectionDot.Render("●"))
		for _, st := range result.ServiceTraces {
			printTraceLine(st.Service)
			
			// Show endpoints state
			fmt.Println(nextCheckStyle.Render(fmt.Sprintf(
				"  →  %d/%d endpoints ready",
				st.ReadyEndpointCount,
				st.EndpointCount,
			)))
			
			// Print backing pods for this service
			printedPods := 0
			for _, pod := range st.BackingPods {
				if pod.Severity == "healthy" && !showHealthy {
					continue
				}
				// indenting pods further
				reason := pod.PrimaryReason
				if len(pod.Findings) > 0 && pod.Findings[0].ReasonCode != "" {
					reason = pod.Findings[0].ReasonCode
				}
				confidence := ""
				if len(pod.Findings) > 0 && pod.Findings[0].Confidence != "" {
					confidence = " · confidence " + pod.Findings[0].Confidence
				}
				fmt.Printf("      %s  %s  %s%s\n",
					scanSeverityMarker(pod.Severity),
					pod.Resource,
					reason,
					confidence,
				)
				if len(pod.Summary) > 0 {
					fmt.Println(logLine.Render("  " + pod.Summary[0]))
				}
				printedPods++
			}
			
			if printedPods == 0 && len(st.BackingPods) > 0 {
				fmt.Println(nextCheckStyle.Render("      →  No unhealthy backing pods."))
			} else if len(st.BackingPods) == 0 {
				fmt.Println(nextCheckStyle.Render("      →  No backing pods found."))
			}
			fmt.Println()
		}
	}

	return nil
}
