package analyzer

import (
	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// PendingRule diagnoses issues where the cluster brain (kube-scheduler) refuses
// or is unable to assign the pod to any physical node. This frequently means
// there simply isn't enough raw RAM or CPU available on the cluster to host it.
type PendingRule struct{}

func (r *PendingRule) Name() string { return "Pending" }

// Match triggers only when the pod is Pending AND has a FailedScheduling event.
// This prevents false positives on pods that are simply waiting to be scheduled
// for the first time (no events yet) vs pods that are genuinely stuck.
func (r *PendingRule) Match(signals *kube.PodSignals) bool {
	if signals.Phase != "Pending" {
		return false
	}
	for _, e := range signals.Events {
		if e.Reason == "FailedScheduling" {
			return true
		}
	}
	return false
}

// Analyze parses the event log perfectly to retrieve the exact wording
// the scheduler emitted. (e.g. "0/3 nodes are available: 3 Insufficient memory")
func (r *PendingRule) Analyze(signals *kube.PodSignals) AnalysisResult {
	var msg string
	if len(signals.Events) > 0 {
		for _, e := range signals.Events {
			if e.Reason == "FailedScheduling" {
				msg = e.Message
				break
			}
		}
	}

	return AnalysisResult{
		Resource:      "pod/" + signals.PodName,
		Namespace:     signals.Namespace,
		Status:        "Pending — cannot be scheduled",
		PrimaryReason: "No nodes available to run this pod",
		Severity:      "warning",
		Summary: []string{
			"The scheduler cannot find a node that meets",
			"the pod's resource requests or constraints.",
		},
		Evidence: []Evidence{
			{Label: "Phase", Value: signals.Phase},
			{Label: "Scheduler Msg", Value: msg},
		},
		NextChecks: []string{
			"Check if nodes have sufficient CPU and memory requested by this pod",
			"Check if taints, tolerations, or node selectors are preventing scheduling",
			"Check if any PVCs are failing to bind",
		},
		RecentEvents: extractEventStrings(signals.Events, 3),
	}
}
