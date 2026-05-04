package analyzer

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

type MapNode struct {
	Kind     string
	Name     string
	Status   string
	Severity string
	Message  string
	Children []*MapNode
}

type MapResult struct {
	Namespace string
	RootNodes []*MapNode
}

func BuildNamespaceMap(
	namespace string,
	deployments []appsv1.Deployment,
	statefulSets []appsv1.StatefulSet,
	daemonSets []appsv1.DaemonSet,
	services []corev1.Service,
	ingresses []networkingv1.Ingress,
	pods []corev1.Pod,
	analyzedResults []AnalysisResult,
) MapResult {

	// Quick lookup for analysis results
	resultsByResource := make(map[string]AnalysisResult)
	for _, res := range analyzedResults {
		resultsByResource[res.Resource] = res
	}

	root := MapResult{Namespace: namespace}

	// Deployments
	if len(deployments) > 0 {
		deployGroup := &MapNode{Kind: "Group", Name: "Deployments", Status: "Healthy", Severity: "healthy"}
		for _, deploy := range deployments {
			node := buildWorkloadNode("deployment", deploy.Name, deploy.Spec.Template.Labels, resultsByResource, services, pods)
			deployGroup.Children = append(deployGroup.Children, node)
			updateGroupSeverity(deployGroup, node.Severity)
		}
		root.RootNodes = append(root.RootNodes, deployGroup)
	}

	// StatefulSets
	if len(statefulSets) > 0 {
		stsGroup := &MapNode{Kind: "Group", Name: "StatefulSets", Status: "Healthy", Severity: "healthy"}
		for _, sts := range statefulSets {
			node := buildWorkloadNode("statefulset", sts.Name, sts.Spec.Template.Labels, resultsByResource, services, pods)
			stsGroup.Children = append(stsGroup.Children, node)
			updateGroupSeverity(stsGroup, node.Severity)
		}
		root.RootNodes = append(root.RootNodes, stsGroup)
	}

	// DaemonSets
	if len(daemonSets) > 0 {
		dsGroup := &MapNode{Kind: "Group", Name: "DaemonSets", Status: "Healthy", Severity: "healthy"}
		for _, ds := range daemonSets {
			node := buildWorkloadNode("daemonset", ds.Name, ds.Spec.Template.Labels, resultsByResource, services, pods)
			dsGroup.Children = append(dsGroup.Children, node)
			updateGroupSeverity(dsGroup, node.Severity)
		}
		root.RootNodes = append(root.RootNodes, dsGroup)
	}

	return root
}

func buildWorkloadNode(
	kind, name string,
	labels map[string]string,
	results map[string]AnalysisResult,
	services []corev1.Service,
	pods []corev1.Pod,
) *MapNode {

	resourceKey := kind + "/" + name
	res, ok := results[resourceKey]

	status := "Healthy"
	severity := "healthy"
	msg := ""

	if ok {
		status = res.Status
		severity = res.Severity
		if len(res.Summary) > 0 {
			msg = strings.Join(res.Summary, " ")
		}
	}

	node := &MapNode{
		Kind:     strings.Title(kind),
		Name:     name,
		Status:   status,
		Severity: severity,
		Message:  msg,
	}

	// Find associated Services
	for _, svc := range services {
		if isSubset(svc.Spec.Selector, labels) {
			svcKey := "service/" + svc.Name
			svcRes, svcOk := results[svcKey]
			svcStatus := "Healthy"
			svcSev := "healthy"
			svcMsg := ""
			if svcOk {
				svcStatus = svcRes.Status
				svcSev = svcRes.Severity
				if len(svcRes.Summary) > 0 {
					svcMsg = strings.Join(svcRes.Summary, " ")
				}
			}
			node.Children = append(node.Children, &MapNode{
				Kind:     "Service",
				Name:     svc.Name,
				Status:   svcStatus,
				Severity: svcSev,
				Message:  svcMsg,
			})
		}
	}

	// Count ready pods manually for context
	readyCount := 0
	totalCount := 0
	for _, pod := range pods {
		if isSubset(labels, pod.Labels) && hasOwnerName(pod, name) {
			totalCount++
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					readyCount++
				}
			}
		}
	}

	// Try to extract the failing pod name from Evidence
	failingPodName := ""
	if ok {
		for _, ev := range res.Evidence {
			if ev.Label == "Analyzing pod" {
				failingPodName = ev.Value
				break
			}
		}
	}

	if failingPodName != "" {
		node.Children = append(node.Children, &MapNode{
			Kind:     "Pod",
			Name:     failingPodName,
			Status:   status,
			Severity: severity,
			Message:  msg,
		})
	} else if totalCount > 0 {
		podsStatus := "Healthy"
		podsSev := "healthy"
		if readyCount < totalCount {
			podsStatus = "Degraded"
			podsSev = "warning"
		}
		node.Children = append(node.Children, &MapNode{
			Kind:     "Pods",
			Name:     fmt.Sprintf("%d/%d Ready", readyCount, totalCount),
			Status:   podsStatus,
			Severity: podsSev,
		})
	}

	return node
}

func isSubset(subset, superset map[string]string) bool {
	if len(subset) == 0 {
		return false
	}
	for k, v := range subset {
		if superset[k] != v {
			return false
		}
	}
	return true
}

func hasOwnerName(pod corev1.Pod, name string) bool {
	for _, owner := range pod.OwnerReferences {
		if strings.Contains(owner.Name, name) {
			return true
		}
	}
	return false
}

func updateGroupSeverity(group *MapNode, childSev string) {
	if childSev == "critical" {
		group.Severity = "critical"
		group.Status = "Critical"
	} else if childSev == "warning" && group.Severity != "critical" {
		group.Severity = "warning"
		group.Status = "Warning"
	}
}
