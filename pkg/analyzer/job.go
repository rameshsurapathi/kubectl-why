package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// AnalyzeJob produces a result for a Kubernetes job
func AnalyzeJob(signals *kube.JobSignals) AnalysisResult {

	resource := "job/" + signals.JobName

	// ── Job completed successfully ───────────────────
	if signals.IsComplete {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Completed",
			PrimaryReason: "Job finished successfully",
			Severity:      "healthy",
			Summary: []string{
				"The job completed successfully.",
			},
			Evidence: []Evidence{
				{
					Label: "Succeeded",
					Value: fmt.Sprintf(
						"%d", signals.Succeeded),
				},
				{
					Label: "Failed attempts",
					Value: fmt.Sprintf(
						"%d", signals.Failed),
				},
			},
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	// ── Job failed ───────────────────────────────────
	if signals.IsFailed {
		result := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Failed",
			PrimaryReason: "Job exceeded backoff limit",
			Severity:      "critical",
			Summary: []string{
				fmt.Sprintf(
					"Job failed after %d attempts "+
						"(backoff limit: %d).",
					signals.Retries,
					signals.BackoffLimit),
			},
			Evidence: []Evidence{
				{
					Label: "Attempts",
					Value: fmt.Sprintf(
						"%d / %d",
						signals.Retries,
						signals.BackoffLimit),
				},
			},
		}

		// Add fail reason if available
		if signals.FailReason != "" {
			result.Evidence = append(
				result.Evidence,
				Evidence{
					Label: "Reason",
					Value: signals.FailReason,
				})
		}

		// If we have a failed pod — add pod analysis
		if signals.FailedPodSignals != nil {
			podResult := AnalyzePod(
				signals.FailedPodSignals)

			// Merge pod evidence and fix commands
			result.Evidence = append(
				result.Evidence,
				Evidence{
					Label: "Failed pod",
					Value: signals.FailedPodName,
				},
			)
			for _, e := range podResult.Evidence {
				if e.Label == "Reason" || e.Label == "Restarts" {
					continue
				}
				result.Evidence = append(result.Evidence, e)
			}

			// Add concrete log command for failed job
			result.FixCommands = []FixCommand{
				{
					Description: "Check logs from failed pod",
					Command: fmt.Sprintf(
						"kubectl logs %s -n %s",
						signals.FailedPodName,
						signals.Namespace),
				},
			}

			// Keep next checks as supplementary
			result.NextChecks = []string{
				fmt.Sprintf(
					"kubectl describe job %s -n %s",
					signals.JobName, signals.Namespace),
			}

			// Use pod's more specific reason if available
			if podResult.PrimaryReason != "Unknown" {
				result.PrimaryReason = podResult.PrimaryReason
				result.Summary = append(
					result.Summary,
					podResult.Summary...,
				)
			}

			// Append pod findings
			result.Findings = append(result.Findings, podResult.Findings...)
		}

		if len(result.Findings) == 0 {
			result.Findings = append(result.Findings, resultToFinding(result))
		}

		return result
	}

	// ── Job still running ────────────────────────────
	if signals.Active > 0 {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Running",
			PrimaryReason: "Job is in progress",
			Severity:      "healthy",
			Summary: []string{
				"The job is currently running.",
			},
			Evidence: []Evidence{
				{
					Label: "Active pods",
					Value: fmt.Sprintf(
						"%d", signals.Active),
				},
				{
					Label: "Completed",
					Value: fmt.Sprintf(
						"%d", signals.Succeeded),
				},
				{
					Label: "Failed attempts",
					Value: fmt.Sprintf(
						"%d", signals.Failed),
				},
			},
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	// ── Unknown state ────────────────────────────────
	res := AnalysisResult{
		SchemaVersion: "v2",
		Resource:      resource,
		Namespace:     signals.Namespace,
		Status:        "Unknown",
		PrimaryReason: "Cannot determine job state",
		Severity:      "warning",
		Summary: []string{
			"Job state could not be determined.",
		},
		NextChecks: []string{
			fmt.Sprintf(
				"kubectl describe job %s -n %s",
				signals.JobName, signals.Namespace),
		},
	}
	res.Findings = append(res.Findings, resultToFinding(res))
	return res
}
