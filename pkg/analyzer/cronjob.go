package analyzer

import (
	"fmt"
	"time"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
	corev1 "k8s.io/api/core/v1"
)

func AnalyzeCronJob(signals *kube.CronJobSignals) AnalysisResult {
	resource := "cronjob/" + signals.Name

	var reasonCode string
	var msg string
	severity := "warning"
	status := "Unhealthy"

	if signals.Suspend {
		reasonCode = "CRONJOB_SUSPENDED"
		msg = "The CronJob is suspended and will not schedule new jobs."
	} else if len(signals.RecentJobs) > 0 {
		lastJob := signals.RecentJobs[0]
		if lastJob.Status.Failed > 0 {
			for _, cond := range lastJob.Status.Conditions {
				if cond.Type == "Failed" && cond.Status == corev1.ConditionTrue {
					reasonCode = "LAST_JOB_FAILED"
					msg = fmt.Sprintf("The most recent job %s failed.", lastJob.Name)
					severity = "critical"
					status = "Failed"
				}
			}
		}
	}

	if reasonCode == "" && signals.LastScheduleTime != nil {
		if time.Since(signals.LastScheduleTime.Time) > 24*time.Hour && signals.LastSuccessfulTime == nil {
			reasonCode = "NO_SUCCESSFUL_JOBS_RECENTLY"
			msg = "No successful jobs in the last 24 hours."
		}
	}

	if reasonCode == "" && signals.ConcurrencyPolicy == "Forbid" && len(signals.ActiveJobs) > 0 {
		// Just a warning, might be normal but good to know
		reasonCode = "CONCURRENCY_POLICY_FORBID"
		msg = "Concurrency policy is Forbid and there are active jobs. New schedules might be skipped."
		severity = "info"
		status = "Healthy"
	}

	if reasonCode == "" {
		reasonCode = "CRONJOB_HEALTHY"
		msg = "CronJob is healthy."
		severity = "healthy"
		status = "Healthy"
	}

	res := AnalysisResult{
		SchemaVersion: "v2",
		Resource:      resource,
		Namespace:     signals.Namespace,
		Status:        status,
		PrimaryReason: reasonCode,
		Severity:      severity,
		Summary:       []string{msg},
		Evidence: []Evidence{
			{Label: "Schedule", Value: signals.Schedule},
			{Label: "Suspend", Value: fmt.Sprintf("%v", signals.Suspend)},
			{Label: "Active Jobs", Value: fmt.Sprintf("%d", len(signals.ActiveJobs))},
		},
		NextChecks: []string{
			fmt.Sprintf("kubectl describe cronjob %s -n %s", signals.Name, signals.Namespace),
		},
	}

	if signals.LastScheduleTime != nil {
		res.Evidence = append(res.Evidence, Evidence{Label: "Last Schedule", Value: signals.LastScheduleTime.Time.Format(time.RFC3339)})
	}

	res.Findings = []Finding{
		{
			Category:       "CronJob",
			ReasonCode:     reasonCode,
			Confidence:     "high",
			AffectedObject: resource,
			Message:        msg,
			Evidence:       res.Evidence,
			FixCommands:    res.FixCommands,
			NextChecks:     res.NextChecks,
		},
	}

	return res
}
