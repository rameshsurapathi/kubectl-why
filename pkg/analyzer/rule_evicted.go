package analyzer

import (
	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// EvictedRule triggers when a pod is forcefully removed from a node by kubelet.
// This is critical: it means the NODE ran out of memory or disk space,
// NOT that the container exceeded its own limits (which would be OOMKilled).
// Because this is an infrastructure-level error, it sits high in our registry priority.
type EvictedRule struct{}

func (r *EvictedRule) Name() string { return "Evicted" }

// Match checks the very top-level pod state for 'Evicted'.
func (r *EvictedRule) Match(signals *kube.PodSignals) bool {
	return signals.PodReason == "Evicted"
}

// Analyze hunts through the K8s events to find the exact notification 
// emitted by the scheduler explaining why the eviction occurred.
func (r *EvictedRule) Analyze(signals *kube.PodSignals) AnalysisResult {
	evictionMsg := "Unknown eviction reason"
	// Grab exact eviction message from conditions or events if possible
	if len(signals.Events) > 0 {
		for _, e := range signals.Events {
			if e.Reason == "Evicted" || e.Reason == "Evicting" {
				evictionMsg = e.Message
				break
			}
		}
	}

	return AnalysisResult{
		Resource:      "pod/" + signals.PodName,
		Namespace:     signals.Namespace,
		Status:        "Evicted",
		PrimaryReason: signals.PodReason,
		Severity:      "critical",
		Summary: []string{
			"Likely cause: kubelet evicted the pod because the node was under resource pressure.",
		},
		Evidence: []Evidence{
			{Label: "Pod Reason", Value: signals.PodReason},
			{Label: "Node", Value: signals.NodeName},
			{Label: "Message", Value: evictionMsg},
		},
		FixCommands: []FixCommand{
			{
				Description: "Delete the evicted pod (it will be recreated if managed by a controller)",
				Command:     "kubectl delete pod " + signals.PodName + " -n " + signals.Namespace,
				SafetyLevel: "destructive",
			},
		},
		NextChecks: []string{
			"Check node resource usage (kubectl top node " + signals.NodeName + ")",
			"Review the exact eviction message in events",
		},
		RecentEvents: extractEventStrings(signals.Events, 3),
	}
}
