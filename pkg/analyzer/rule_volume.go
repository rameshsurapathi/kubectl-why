package analyzer

import (
	"strings"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

type VolumeRule struct{}

func (r *VolumeRule) Name() string { return "VolumeError" }

func (r *VolumeRule) Match(signals *kube.PodSignals) bool {
	for _, e := range signals.Events {
		if strings.Contains(e.Reason, "FailedMount") || strings.Contains(e.Reason, "FailedAttachVolume") {
			return true
		}
	}
	return false
}

func (r *VolumeRule) Analyze(signals *kube.PodSignals) AnalysisResult {
	var reasonCode string
	var msg string

	for _, e := range signals.Events {
		if strings.Contains(e.Reason, "FailedMount") {
			reasonCode = "MOUNT_FAILED"
			msg = e.Message
			break
		}
		if strings.Contains(e.Reason, "FailedAttachVolume") {
			reasonCode = "ATTACH_FAILED"
			msg = e.Message
			break
		}
	}

	if strings.Contains(msg, "not found") && strings.Contains(msg, "pvc") {
		reasonCode = "PVC_NOT_FOUND"
	}

	res := AnalysisResult{
		Resource:      "pod/" + signals.PodName,
		Namespace:     signals.Namespace,
		Status:        "ContainerCreating",
		PrimaryReason: "Volume Mount/Attach Failed",
		Severity:      "critical",
		Summary: []string{
			"The pod cannot start because a volume failed to mount or attach.",
		},
		Evidence: []Evidence{
			{Label: "Error", Value: msg},
		},
		NextChecks: []string{
			"Check if the PVC exists and is bound",
			"Check if the StorageClass is valid",
		},
	}
	res.Findings = []Finding{
		{
			Category:       "Storage",
			ReasonCode:     reasonCode,
			Confidence:     "high",
			AffectedObject: res.Resource,
			Message:        msg,
			Evidence:       res.Evidence,
			NextChecks:     res.NextChecks,
		},
	}

	return res
}
