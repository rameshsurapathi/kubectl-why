package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
)

// ── Styles ────────────────────────────────────────────────

var (
	// Colors
	colorRed    = lipgloss.Color("196")
	colorYellow = lipgloss.Color("220")
	colorGreen  = lipgloss.Color("82")
	colorBlue   = lipgloss.Color("39")
	colorGray   = lipgloss.Color("241")
	colorWhite  = lipgloss.Color("255")

	// Header box
	headerBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("241")).
			Padding(0, 1).
			MarginBottom(1)

	// Resource name in header
	resourceStyle = lipgloss.NewStyle().
			Foreground(colorWhite).
			Bold(true)

	// Namespace in header
	namespaceStyle = lipgloss.NewStyle().
			Foreground(colorGray)

	// Section label (●  Why, ●  Evidence, etc.)
	sectionDot = lipgloss.NewStyle().
			Foreground(colorBlue).
			Bold(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			Width(22)

	// Status styles by severity
	statusCritical = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	statusWarning = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	statusHealthy = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	// Reason text
	reasonCritical = lipgloss.NewStyle().
			Foreground(colorRed)

	reasonWarning = lipgloss.NewStyle().
			Foreground(colorYellow)

	// Safety levels
	safetyInspect = lipgloss.NewStyle().
			Foreground(colorBlue).
			Bold(true)
	safetyLowRisk = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)
	safetyMutating = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)
	safetyDestructive = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	// Evidence value
	evidenceValue = lipgloss.NewStyle().
			Foreground(colorWhite)

	// Fix command block
	fixBlock = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("150")).
			PaddingLeft(1).
			PaddingRight(1).
			MarginLeft(4)

	// Fix description comment
	fixComment = lipgloss.NewStyle().
			Foreground(colorGray).
			MarginLeft(4)

	// Log lines
	logLine = lipgloss.NewStyle().
		Foreground(colorGray).
		MarginLeft(4)

	// Divider
	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	// Runbook hint at bottom
	runbookHint = lipgloss.NewStyle().
			Foreground(colorBlue).
			MarginLeft(2)

	// Next check items
	nextCheckStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			MarginLeft(4)

	// Tree connector styles
	treeConnector = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	// Dim text for secondary info
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))
)

// Options configures the render output.
type Options struct {
	Explain       bool
	ShowSecondary bool
}

// ── Main renderer ─────────────────────────────────────────

func Text(result analyzer.AnalysisResult, opts Options) error {
	fmt.Println()

	// ── Header ───────────────────────────────────────
	headerContent := fmt.Sprintf("%s  %s",
		resourceStyle.Render(result.Resource),
		namespaceStyle.Render("· "+result.Namespace),
	)
	fmt.Println(headerBox.Render(headerContent))

	// ── Status line ──────────────────────────────────
	printStatusLine(result)
	fmt.Println()

	// ── Why (Primary Reason) ─────────────────────────
	if result.PrimaryReason != "" {
		fmt.Printf("  %s Why\n",
			sectionDot.Render("●"))
		printReason(result)
		fmt.Println()
	}

	// ── Evidence ─────────────────────────────────────
	if len(result.Evidence) > 0 {
		fmt.Printf("  %s Evidence\n",
			sectionDot.Render("●"))
		for _, e := range result.Evidence {
			printEvidence(e)
		}
		fmt.Println()
	}

	// ── Last Logs ────────────────────────────────────
	if result.RecentLogs != "" {
		fmt.Printf("  %s Last logs\n",
			sectionDot.Render("●"))
		printLogs(result.RecentLogs)
		fmt.Println()
	}

	// ── Fix Commands ─────────────────────────────────
	if len(result.FixCommands) > 0 {
		fmt.Printf("  %s Fix\n",
			sectionDot.Render("●"))
		for _, fix := range result.FixCommands {
			printFixCommand(fix)
		}
		fmt.Println()
	}

	// ── Next Checks ──────────────────────────────────
	if len(result.NextChecks) > 0 {
		fmt.Printf("  %s Also check\n",
			sectionDot.Render("●"))
		for _, check := range result.NextChecks {
			fmt.Println(nextCheckStyle.Render("→  " + check))
		}
		fmt.Println()
	}

	// ── Secondary Findings ───────────────────────────
	if opts.ShowSecondary && len(result.Findings) > 1 {
		fmt.Printf("  %s Secondary Findings\n",
			sectionDot.Render("●"))
		for i := 1; i < len(result.Findings); i++ {
			f := result.Findings[i]
			fmt.Printf("    %s (%s)\n", f.Message, f.ReasonCode)
			if opts.Explain {
				fmt.Printf("      Confidence: %s\n", f.Confidence)
			}
			for _, e := range f.Evidence {
				fmt.Printf("      %s: %s\n", e.Label, e.Value)
			}
		}
		fmt.Println()
	}

	// ── Explain ──────────────────────────────────────
	if opts.Explain {
		hasReasoning := result.Reasoning != ""
		hasFindings := len(result.Findings) > 0

		if hasReasoning || hasFindings {
			fmt.Printf("  %s Explain\n", sectionDot.Render("●"))
			if hasReasoning {
				fmt.Printf("    Reasoning: %s\n", result.Reasoning)
			}
			if hasFindings {
				f := result.Findings[0]
				fmt.Printf("    Primary Diagnosis: %s\n", f.ReasonCode)
				fmt.Printf("    Confidence: %s\n", f.Confidence)
				if f.AffectedObject != "" {
					fmt.Printf("    Affected Object: %s\n", f.AffectedObject)
				}
				if f.Reasoning != "" {
					fmt.Printf("    Finding Reasoning: %s\n", f.Reasoning)
				}
			}
			
			// Print provenance for evidence
			hasProvenance := false
			for _, e := range result.Evidence {
				if e.Provenance != "" {
					if !hasProvenance {
						fmt.Println("\n    Evidence Provenance:")
						hasProvenance = true
					}
					fmt.Printf("      %s: %s\n", e.Label, e.Provenance)
				}
			}
			fmt.Println()
		}
	}

	return nil
}

// ── Helper renderers ──────────────────────────────────────

func printStatusLine(result analyzer.AnalysisResult) {
	label := labelStyle.Render("  Status")

	var status string
	switch result.Severity {
	case "critical":
		status = statusCritical.Render(result.Status)
	case "warning":
		status = statusWarning.Render(result.Status)
	case "healthy":
		status = statusHealthy.Render(result.Status)
	default:
		status = result.Status
	}

	fmt.Printf("%s  %s\n", label, status)
	printConfidenceLine(result)
}

func printConfidenceLine(result analyzer.AnalysisResult) {
	if result.Severity == "healthy" || len(result.Findings) == 0 {
		return
	}

	finding := result.Findings[0]
	if finding.Confidence == "" {
		return
	}

	value := finding.Confidence
	if finding.ReasonCode != "" {
		value = fmt.Sprintf("%s (%s)", value, finding.ReasonCode)
	}

	fmt.Printf("%s  %s\n",
		labelStyle.Render("  Confidence"),
		evidenceValue.Render(value),
	)
}

func printReason(result analyzer.AnalysisResult) {
	// Print each summary line with appropriate color
	for _, line := range result.Summary {
		var styled string
		switch result.Severity {
		case "critical":
			styled = reasonCritical.Render(
				"    " + line)
		case "warning":
			styled = reasonWarning.Render(
				"    " + line)
		default:
			styled = "    " + line
		}
		fmt.Println(styled)
	}
}

func printEvidence(e analyzer.Evidence) {
	label := labelStyle.Render("    " + e.Label)
	value := evidenceValue.Render(e.Value)

	// If evidence has a progress bar (memory/CPU usage)
	if e.Bar != nil {
		if e.Bar.Max == 0 {
			e.Bar.Max = 1
		}
		bar := renderBar(e.Bar.Current, e.Bar.Max)
		pct := float64(e.Bar.Current) /
			float64(e.Bar.Max) * 100

		// Color the bar red if >80%
		barStyled := lipgloss.NewStyle().
			Foreground(barColor(pct)).
			Render(bar)

		fmt.Printf("%s  %s  %s  %.0f%%\n",
			label,
			value,
			barStyled,
			pct,
		)
	} else {
		fmt.Printf("%s  %s\n", label, value)
	}
}

func printFixCommand(fix analyzer.FixCommand) {
	desc := fix.Description
	if fix.SafetyLevel != "" {
		var badge string
		switch fix.SafetyLevel {
		case "inspect":
			badge = safetyInspect.Render("[inspect]")
		case "low-risk":
			badge = safetyLowRisk.Render("[low-risk]")
		case "mutating":
			badge = safetyMutating.Render("[mutating]")
		case "destructive":
			badge = safetyDestructive.Render("[destructive]")
		default:
			badge = lipgloss.NewStyle().Foreground(colorGray).Render("[" + fix.SafetyLevel + "]")
		}
		if desc != "" {
			desc = badge + " " + desc
		} else {
			desc = badge
		}
	}

	if desc != "" {
		fmt.Println(fixComment.Render("# " + desc))
	}

	// Multi-line command support
	lines := strings.Split(fix.Command, "\n")
	for i, line := range lines {
		if i == 0 {
			fmt.Println(fixBlock.Render(line))
		} else {
			// Indent continuation lines
			fmt.Println(fixBlock.Render("  " + line))
		}
	}
	fmt.Println()
}

func printLogs(logs string) {
	lines := strings.Split(strings.TrimSpace(logs), "\n")

	// Show last 5 lines only — most recent is most relevant
	start := 0
	if len(lines) > 5 {
		start = len(lines) - 5
		// Show indicator that logs were trimmed
		fmt.Println(logLine.Render(
			fmt.Sprintf("... (%d lines above hidden)",
				start)))
	}

	for _, line := range lines[start:] {
		if line == "" {
			continue
		}
		// Highlight error lines in the logs
		if isErrorLine(line) {
			fmt.Println(lipgloss.NewStyle().
				Foreground(colorRed).
				MarginLeft(4).
				Render(line))
		} else {
			fmt.Println(logLine.Render(line))
		}
	}
}

// renderBar creates a visual progress bar
// e.g. "████████████░░░░"
func renderBar(current, max int64) string {
	if max == 0 {
		return ""
	}
	total := 12
	filled := int(float64(current) /
		float64(max) * float64(total))
	if filled > total {
		filled = total
	}
	empty := total - filled
	return strings.Repeat("█", filled) +
		strings.Repeat("░", empty)
}

// barColor returns red if >80%, yellow if >60%, green otherwise
func barColor(pct float64) lipgloss.Color {
	switch {
	case pct >= 80:
		return colorRed
	case pct >= 60:
		return colorYellow
	default:
		return colorGreen
	}
}

// isErrorLine returns true if a log line looks like an error
func isErrorLine(line string) bool {
	lower := strings.ToLower(line)
	errorKeywords := []string{
		"error", "exception", "fatal", "panic",
		"failed", "failure", "critical", "err:",
		"oom", "killed", "crash",
	}
	for _, kw := range errorKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
