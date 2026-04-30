package analyzer

import (
	"fmt"
	"strings"

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

	res := AnalysisResult{
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
		FixCommands: []FixCommand{
			{
				Description: "Check recent scheduling events",
				Command:     fmt.Sprintf("kubectl get events -n %s --field-selector involvedObject.name=%s", signals.Namespace, signals.PodName),
				SafetyLevel: "inspect",
			},
		},
		NextChecks: []string{
			"Check if nodes have sufficient CPU and memory requested by this pod",
			"Check if taints, tolerations, or node selectors are preventing scheduling",
			"Check if any PVCs are failing to bind",
		},
		RecentEvents: extractEventStrings(signals.Events, 3),
	}

	reasonCode := "SCHEDULING_FAILED"
	if msg != "" {
		if strings.Contains(msg, "Insufficient cpu") {
			reasonCode = "INSUFFICIENT_CPU"
		} else if strings.Contains(msg, "Insufficient memory") {
			reasonCode = "INSUFFICIENT_MEMORY"
		} else if strings.Contains(msg, "node(s) didn't match Pod's node affinity/selector") || strings.Contains(msg, "node selector") {
			reasonCode = "NODE_SELECTOR_MISMATCH"
		} else if strings.Contains(msg, "node(s) didn't match pod affinity/anti-affinity") || strings.Contains(msg, "pod anti-affinity") {
			reasonCode = "POD_ANTI_AFFINITY_CONFLICT"
		} else if strings.Contains(msg, "node(s) had untolerated taint") {
			reasonCode = "UNtolerated_TAINT"
		} else if strings.Contains(msg, "topology spread constraints") {
			reasonCode = "TOPOLOGY_SPREAD_CONSTRAINT"
		} else if strings.Contains(msg, "0/0 nodes are available") {
			reasonCode = "NO_NODES_AVAILABLE"
		} else if strings.Contains(msg, "node(s) were unschedulable") {
			reasonCode = "NODES_UNSCHEDULABLE"
		}
	}

	res.Findings = []Finding{
		{
			Category:       "Scheduling",
			ReasonCode:     reasonCode,
			Confidence:     "high",
			AffectedObject: res.Resource,
			Message:        msg,
			Evidence:       res.Evidence,
			FixCommands:    res.FixCommands,
			NextChecks:     res.NextChecks,
		},
	}
	return res
}
