package analyzer

import (
	"context"

	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NamespaceReport contains both the flat analysis results and the topological map.
type NamespaceReport struct {
	Namespace string
	Results   []AnalysisResult
	Map       MapResult
}

// CollectNamespaceReport performs a full scan and builds a topological map.
func CollectNamespaceReport(
	client *kubernetes.Clientset,
	namespace string,
	tailLines int64,
	maxEvents int,
) (*NamespaceReport, error) {
	ctx := context.Background()

	// 1. Fetch raw resources first
	deployments, _ := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	statefulSets, _ := client.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	daemonSets, _ := client.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	services, _ := client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	ingresses, _ := client.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	pods, _ := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})

	// 2. Perform analysis on each resource
	var results []AnalysisResult

	for _, d := range deployments.Items {
		sig, err := kube.CollectDeploymentSignals(client, d.Name, namespace, tailLines, maxEvents)
		if err == nil {
			results = append(results, AnalyzeDeployment(sig))
		}
	}
	for _, s := range statefulSets.Items {
		sig, err := kube.CollectStatefulSetSignals(client, s.Name, namespace, tailLines, maxEvents)
		if err == nil {
			results = append(results, AnalyzeStatefulSet(sig))
		}
	}
	for _, d := range daemonSets.Items {
		sig, err := kube.CollectDaemonSetSignals(client, d.Name, namespace, tailLines, maxEvents)
		if err == nil {
			results = append(results, AnalyzeDaemonSet(sig))
		}
	}
	for _, svc := range services.Items {
		sig, err := kube.CollectServiceSignals(client, svc.Name, namespace, maxEvents)
		if err == nil {
			results = append(results, AnalyzeService(sig))
		}
	}

	// Add standalone pods
	for _, pod := range pods.Items {
		if HasControllerOwner(pod) {
			continue
		}
		sig, err := kube.CollectPodSignals(client, pod.Name, namespace, tailLines, maxEvents)
		if err == nil {
			results = append(results, AnalyzePod(sig))
		}
	}

	// 3. Build the map using the raw resources and analysis results
	mapResult := BuildNamespaceMap(
		namespace,
		deployments.Items,
		statefulSets.Items,
		daemonSets.Items,
		services.Items,
		ingresses.Items,
		pods.Items,
		results,
	)

	return &NamespaceReport{
		Namespace: namespace,
		Results:   results,
		Map:       mapResult,
	}, nil
}

// CollectNamespaceResults scans a namespace for all supported resources and returns analysis results.
func CollectNamespaceResults(
	client *kubernetes.Clientset,
	namespace string,
	tailLines int64,
	maxEvents int,
) ([]AnalysisResult, error) {
	report, err := CollectNamespaceReport(client, namespace, tailLines, maxEvents)
	if err != nil {
		return nil, err
	}
	return report.Results, nil
}

// HasControllerOwner returns true if the pod is owned by a controller.
func HasControllerOwner(pod corev1.Pod) bool {
	for _, owner := range pod.OwnerReferences {
		if owner.Controller != nil && *owner.Controller {
			return true
		}
	}
	return false
}
