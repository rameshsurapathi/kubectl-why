package analyzer

import (
	"fmt"
	"strings"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// SegfaultRule flags containers that crashed due to memory violations.
// This is typically seen in native languages (C/C++) when invalid 
// memory pointers are accessed, resulting in the OS killing the process
// with signal 11 (which translates to exit code 139).
type SegfaultRule struct{}

func (r *SegfaultRule) Name() string { return "ExitCode139" }

// Match checks the current and previous states for Exit Code 139.
func (r *SegfaultRule) Match(signals *kube.PodSignals) bool {
	for _, c := range append(signals.Containers, signals.InitContainers...) {
		if (c.State.IsTerminated && c.State.ExitCode == 139) || (c.LastState.IsTerminated && c.LastState.ExitCode == 139) {
			return true
		}
	}
	return false
}

func (r *SegfaultRule) Analyze(signals *kube.PodSignals) AnalysisResult {
	var restarts int32
	for _, c := range append(signals.Containers, signals.InitContainers...) {
		if (c.State.IsTerminated && c.State.ExitCode == 139) || (c.LastState.IsTerminated && c.LastState.ExitCode == 139) {
			restarts = c.RestartCount
			break
		}
	}

	return AnalysisResult{
		Resource:      "pod/" + signals.PodName,
		Namespace:     signals.Namespace,
		Status:        "Exit Code 139",
		PrimaryReason: "Segmentation Fault",
		Severity:      "critical",
		Summary: []string{
			"Likely cause: process likely crashed with segmentation fault (invalid memory access).",
		},
		Evidence: []Evidence{
			{Label: "Exit Code", Value: "139"},
			{Label: "Restarts", Value: fmt.Sprintf("%d", restarts)},
		},
		RecentLogs: signals.RecentLogs,
		NextChecks: []string{
			"Check application logs for native crashes via C/C++ libs",
		},
	}
}

// AppCrashRule detects generic application crashes resulting in Exit Code 1.
// Unlike basic diagnosis, this rule acts as a smart log parser, actively scanning
// the buffered container logs to extract the exact stack trace or fatal exception
// so the developer doesn't even have to open the logs manually.
type AppCrashRule struct{}

func (r *AppCrashRule) Name() string { return "ExitCode1" }

func (r *AppCrashRule) Match(signals *kube.PodSignals) bool {
	for _, c := range append(signals.Containers, signals.InitContainers...) {
		if (c.State.IsTerminated && c.State.ExitCode == 1) || (c.LastState.IsTerminated && c.LastState.ExitCode == 1) {
			return true
		}
	}
	return false
}

func (r *AppCrashRule) Analyze(signals *kube.PodSignals) AnalysisResult {
	var restarts int32
	var lastReason string

	for _, c := range append(signals.Containers, signals.InitContainers...) {
		if (c.State.IsTerminated && c.State.ExitCode == 1) || (c.LastState.IsTerminated && c.LastState.ExitCode == 1) {
			restarts = c.RestartCount
			if c.LastState.TerminatedReason != "" {
				lastReason = c.LastState.TerminatedReason
			} else if c.State.TerminatedReason != "" {
				lastReason = c.State.TerminatedReason
			}
			break
		}
	}

	summary := []string{
		"Likely cause: application exited with a generic failure; check logs for root cause.",
	}

	evidence := []Evidence{
		{Label: "Exit Code", Value: "1"},
		{Label: "Reason", Value: lastReason},
		{Label: "Restarts", Value: fmt.Sprintf("%d", restarts)},
	}

	// SMART LOGIC: Scan the actual logs for keywords to highlight the exact error!
	if signals.RecentLogs != "" {
		lines := strings.Split(signals.RecentLogs, "\n")
		var extractedErrors []string

		for _, line := range lines {
			l := strings.ToLower(line)
			if strings.Contains(l, "error") || strings.Contains(l, "exception") || strings.Contains(l, "panic") || strings.Contains(l, "fatal") || strings.Contains(l, "traceback") {
				extractedErrors = append(extractedErrors, line)
			}
		}

		if len(extractedErrors) > 0 {
			summary = []string{
				"Likely cause: application code threw a fatal exception or error.",
			}

			// Add the last 3 specific errors found to the UI Evidence
			start := 0
			if len(extractedErrors) > 3 {
				start = len(extractedErrors) - 3
			}
			count := 1
			for _, errLine := range extractedErrors[start:] {
				// Clean up super long stack traces for the terminal UI
				if len(errLine) > 75 {
					errLine = errLine[:72] + "..."
				}
				evidence = append(evidence, Evidence{Label: fmt.Sprintf("Log Match %d", count), Value: errLine})
				count++
			}
		}
	}

	return AnalysisResult{
		Resource:      "pod/" + signals.PodName,
		Namespace:     signals.Namespace,
		Status:        "Exit Code 1",
		PrimaryReason: "Application Crash",
		Severity:      "critical",
		Summary:       summary,
		Evidence:      evidence,
		RecentLogs:    signals.RecentLogs,
		NextChecks: []string{
			"Inspect recent logs to find the full application exception or error stack trace",
		},
	}
}
