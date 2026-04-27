package analyzer

import (
	"fmt"
	"strings"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// AnalyzePVC provides diagnosis for PersistentVolumeClaims
func AnalyzePVC(signals *kube.PVCSignals) AnalysisResult {
	resource := "pvc/" + signals.Name

	if signals.Phase == "Bound" {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Bound",
			PrimaryReason: "PVC is healthy and bound",
			Severity:      "healthy",
			Summary: []string{
				"The PersistentVolumeClaim is successfully bound to a PersistentVolume.",
			},
			Evidence: []Evidence{
				{Label: "Volume", Value: signals.VolumeName},
				{Label: "Capacity", Value: signals.Capacity},
				{Label: "StorageClass", Value: signals.StorageClassName},
			},
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	var reasonCode string
	var msg string
	
	for _, e := range signals.Events {
		if strings.Contains(e.Reason, "ProvisioningFailed") || strings.Contains(e.Reason, "WaitForFirstConsumer") {
			reasonCode = e.Reason
			msg = e.Message
			break
		}
	}

	primaryReason := "PVC is Pending"
	summary := "The PVC is waiting to be bound to a PersistentVolume."
	if reasonCode == "ProvisioningFailed" {
		primaryReason = "Dynamic Provisioning Failed"
		summary = "The storage provider failed to provision a volume."
	} else if reasonCode == "WaitForFirstConsumer" {
		primaryReason = "Waiting for Pod"
		summary = "The PVC will not bind until a Pod is created that uses it (WaitForFirstConsumer binding mode)."
	} else if signals.Phase == "Lost" {
		primaryReason = "Volume Lost"
		summary = "The underlying PersistentVolume has been lost."
		reasonCode = "VOLUME_LOST"
	} else {
		reasonCode = "PVC_PENDING"
	}

	if msg == "" {
		msg = "No detailed events found."
	}

	res := AnalysisResult{
		SchemaVersion: "v2",
		Resource:      resource,
		Namespace:     signals.Namespace,
		Status:        signals.Phase,
		PrimaryReason: primaryReason,
		Severity:      "critical",
		Summary: []string{summary},
		Evidence: []Evidence{
			{Label: "StorageClass", Value: signals.StorageClassName},
			{Label: "Message", Value: msg},
		},
		FixCommands: []FixCommand{
			{
				Description: "Check StorageClass",
				Command:     fmt.Sprintf("kubectl get sc %s", signals.StorageClassName),
			},
		},
		NextChecks: []string{
			"Check if the requested StorageClass exists and is the default",
			"Check if you have enough quota for storage",
			fmt.Sprintf("kubectl describe pvc %s -n %s", signals.Name, signals.Namespace),
		},
		RecentEvents: extractEventStrings(signals.Events, 3),
	}
	
	// fix struct literal syntax for Severity logic:
	if signals.Phase == "Pending" && reasonCode == "WaitForFirstConsumer" {
		res.Severity = "warning"
	}

	res.Findings = []Finding{
		{
			Category:       "Storage",
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
