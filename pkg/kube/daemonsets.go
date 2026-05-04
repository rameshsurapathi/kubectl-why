package kube

import (
	"context"
	"fmt"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// DaemonSetSignals holds daemonset-level context
// plus the signals from its worst failing pod
type DaemonSetSignals struct {
	DaemonSetName string
	Namespace     string

	// DaemonSet health summary
	DesiredNumberScheduled int32
	CurrentNumberScheduled int32
	NumberReady            int32
	NumberAvailable        int32
	NumberMisscheduled     int32

	// All pods belonging to this daemonset
	TotalPods   int
	HealthyPods int
	FailingPods int

	// The worst failing pod's signals
	FailingPodSignals *PodSignals

	// Name of the failing pod we analyzed
	FailingPodName string

	// true if all pods are healthy
	AllHealthy bool

	Events []EventSignal
}

// CollectDaemonSetSignals fetches daemonset status
// and finds the worst failing pod to analyze
func CollectDaemonSetSignals(
	client *kubernetes.Clientset,
	name, namespace string,
	tailLines int64,
	maxEvents int,
) (*DaemonSetSignals, error) {

	ctx := context.Background()

	// ── 1. Fetch the daemonset ──────────────────────
	ds, err := client.AppsV1().
		DaemonSets(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"daemonset %q not found in namespace %q: %w",
			name, namespace, err)
	}

	signals := &DaemonSetSignals{
		DaemonSetName:          name,
		Namespace:              namespace,
		DesiredNumberScheduled: ds.Status.DesiredNumberScheduled,
		CurrentNumberScheduled: ds.Status.CurrentNumberScheduled,
		NumberReady:            ds.Status.NumberReady,
		NumberAvailable:        ds.Status.NumberAvailable,
		NumberMisscheduled:     ds.Status.NumberMisscheduled,
	}

	events, _ := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.kind=DaemonSet,involvedObject.name=" + name,
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

	// ── 2. Find pods belonging to this daemonset ────
	pods, err := findDaemonSetPods(
		client, ds, namespace)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot find pods for daemonset %q: %w",
			name, err)
	}

	signals.TotalPods = len(pods)

	// ── 3. Categorize pods ───────────────────────────
	var failingPods []corev1.Pod
	for _, pod := range pods {
		if isPodHealthy(pod) { // Reusing from deployments.go
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
	worstPod := findWorstPod(failingPods) // Reusing from deployments.go
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
		fmt.Fprintf(os.Stderr, "  warning: could not analyze "+
			"pod %q: %v\n", worstPod.Name, err)
		return signals, nil
	}

	signals.FailingPodSignals = podSignals
	return signals, nil
}

// findDaemonSetPods finds all pods owned by a daemonset
func findDaemonSetPods(
	client *kubernetes.Clientset,
	ds *appsv1.DaemonSet,
	namespace string,
) ([]corev1.Pod, error) {

	ctx := context.Background()

	if ds.Spec.Selector == nil {
		return nil, fmt.Errorf("daemonset has no selector")
	}

	selector := labels.Set(
		ds.Spec.Selector.MatchLabels,
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
