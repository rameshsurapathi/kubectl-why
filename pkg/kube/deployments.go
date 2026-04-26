package kube

import (
	"context"
	"fmt"
	"os"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// DeploymentSignals holds deployment-level context
// plus the signals from its worst failing pod
type DeploymentSignals struct {
	DeploymentName string
	Namespace      string

	// Deployment health summary
	DesiredReplicas   int32
	ReadyReplicas     int32
	AvailableReplicas int32
	UpdatedReplicas   int32

	// All pods belonging to this deployment
	TotalPods   int
	HealthyPods int
	FailingPods int

	// The worst failing pod's signals
	// nil if all pods are healthy
	FailingPodSignals *PodSignals

	// Name of the failing pod we analyzed
	FailingPodName string

	// true if all pods are healthy
	AllHealthy bool

	// Rollout conditions
	Conditions []appsv1.DeploymentCondition
	Events     []EventSignal
}

// CollectDeploymentSignals fetches deployment status
// and finds the worst failing pod to analyze
func CollectDeploymentSignals(
	client *kubernetes.Clientset,
	name, namespace string,
	tailLines int64,
	maxEvents int,
) (*DeploymentSignals, error) {

	ctx := context.Background()

	// ── 1. Fetch the deployment ──────────────────────
	deployment, err := client.AppsV1().
		Deployments(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"deployment %q not found in namespace %q: %w",
			name, namespace, err)
	}

	signals := &DeploymentSignals{
		DeploymentName:    name,
		Namespace:         namespace,
		DesiredReplicas:   getDesiredReplicas(deployment),
		ReadyReplicas:     deployment.Status.ReadyReplicas,
		AvailableReplicas: deployment.Status.AvailableReplicas,
		UpdatedReplicas:   deployment.Status.UpdatedReplicas,
		Conditions:        deployment.Status.Conditions,
	}

	events, _ := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.kind=Deployment,involvedObject.name=" + name,
	})
	for _, e := range events.Items {
		signals.Events = append(signals.Events, EventSignal{
			Type:      e.Type,
			Reason:    e.Reason,
			Message:   e.Message,
			Count:     e.Count,
			FirstTime: e.FirstTimestamp.Time,
			LastTime:  e.LastTimestamp.Time,
			Component: e.Source.Component,
		})
	}
	sortEvents(signals.Events)
	if maxEvents > 0 && len(signals.Events) > maxEvents {
		signals.Events = signals.Events[:maxEvents]
	}

	// ── 2. Find pods belonging to this deployment ────
	pods, err := findDeploymentPods(
		client, deployment, namespace)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot find pods for deployment %q: %w",
			name, err)
	}

	signals.TotalPods = len(pods)

	// ── 3. Categorize pods ───────────────────────────
	var failingPods []corev1.Pod
	for _, pod := range pods {
		if isPodHealthy(pod) {
			signals.HealthyPods++
		} else {
			signals.FailingPods++
			failingPods = append(failingPods, pod)
		}
	}

	// ── 4. All pods healthy ──────────────────────────
	if len(failingPods) == 0 {
		signals.AllHealthy = true
		return signals, nil
	}

	// ── 5. Find the worst failing pod ───────────────
	// "Worst" = most restarts or most severe state
	worstPod := findWorstPod(failingPods)
	signals.FailingPodName = worstPod.Name

	// ── 6. Collect full pod signals for worst pod ───
	podSignals, err := CollectPodSignals(
		client,
		worstPod.Name,
		namespace,
		tailLines,
		maxEvents,
	)
	if err != nil {
		// Non-fatal — we still have deployment context
		fmt.Fprintf(os.Stderr, "  warning: could not analyze "+
			"pod %q: %v\n", worstPod.Name, err)
		return signals, nil
	}

	signals.FailingPodSignals = podSignals
	return signals, nil
}

// findDeploymentPods finds all pods owned by a deployment
// by matching the deployment's label selector
func findDeploymentPods(
	client *kubernetes.Clientset,
	deployment *appsv1.Deployment,
	namespace string,
) ([]corev1.Pod, error) {

	ctx := context.Background()

	if deployment.Spec.Selector == nil {
		return nil, fmt.Errorf("deployment has no selector")
	}

	// Use the deployment's selector to find pods
	selector := labels.Set(
		deployment.Spec.Selector.MatchLabels,
	).AsSelector()

	pods, err := client.CoreV1().Pods(namespace).List(
		ctx, metav1.ListOptions{
			LabelSelector: selector.String(),
		})
	if err != nil {
		return nil, err
	}

	return pods.Items, nil
}

// isPodHealthy returns true if a pod is running
// and all containers are ready
func isPodHealthy(pod corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}
	for _, cs := range pod.Status.ContainerStatuses {
		if !cs.Ready {
			return false
		}
	}
	return true
}

// findWorstPod returns the pod most worth analyzing —
// highest restart count or most severe failure state
func findWorstPod(pods []corev1.Pod) corev1.Pod {
	sort.Slice(pods, func(i, j int) bool {
		// Priority 1: pod in Failed phase
		if pods[i].Status.Phase == corev1.PodFailed &&
			pods[j].Status.Phase != corev1.PodFailed {
			return true
		}

		// Priority 2: highest restart count
		restartsI := maxRestarts(pods[i])
		restartsJ := maxRestarts(pods[j])
		return restartsI > restartsJ
	})
	return pods[0]
}

// maxRestarts returns the highest restart count
// across all containers in a pod
func maxRestarts(pod corev1.Pod) int32 {
	var max int32
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.RestartCount > max {
			max = cs.RestartCount
		}
	}
	return max
}

func getDesiredReplicas(
	d *appsv1.Deployment,
) int32 {
	if d.Spec.Replicas != nil {
		return *d.Spec.Replicas
	}
	return 1 // default
}
