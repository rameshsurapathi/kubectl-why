package analyzer

import (
	"fmt"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
)

// AnalyzeRoleBinding produces a result for a RoleBinding
func AnalyzeRoleBinding(signals *kube.RoleBindingSignals) AnalysisResult {
	resource := "rolebinding/" + signals.Name

	evidence := []Evidence{
		{
			Label: "RoleRef",
			Value: fmt.Sprintf("%s/%s", signals.RoleRef.Kind, signals.RoleRef.Name),
		},
		{
			Label: "Subjects",
			Value: fmt.Sprintf("%d bound", len(signals.Subjects)),
		},
	}

	if !signals.RoleFound {
		res := AnalysisResult{
			SchemaVersion: "v2",
			Resource:      resource,
			Namespace:     signals.Namespace,
			Status:        "Degraded",
			PrimaryReason: "Role not found",
			Severity:      "critical",
			Summary: []string{
				fmt.Sprintf("This RoleBinding references a %s named %q, but it does not exist.", signals.RoleRef.Kind, signals.RoleRef.Name),
				"The bound subjects will not receive these permissions.",
			},
			Evidence: evidence,
			NextChecks: []string{
				fmt.Sprintf("kubectl get %s %s -n %s", signals.RoleRef.Kind, signals.RoleRef.Name, signals.Namespace),
			},
		}
		res.Findings = append(res.Findings, resultToFinding(res))
		return res
	}

	res := AnalysisResult{
		SchemaVersion: "v2",
		Resource:      resource,
		Namespace:     signals.Namespace,
		Status:        "Healthy",
		PrimaryReason: "Valid RoleBinding",
		Severity:      "healthy",
		Summary: []string{
			"The RoleBinding references a valid Role/ClusterRole.",
		},
		Evidence: evidence,
	}
	res.Findings = append(res.Findings, resultToFinding(res))
	return res
}
