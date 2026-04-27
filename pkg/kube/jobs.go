package kube

import (
	"context"
	"fmt"
	"sort"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// JobSignals holds job-level context plus
// the signals from its failed pod
type JobSignals struct {
	JobName   string
	Namespace string

	// Job status
	Active    int32
	Succeeded int32
	Failed    int32

	// Job conditions
	IsComplete bool
	IsFailed   bool
	FailReason string

	// Backoff limit info
	BackoffLimit int32
	Retries      int32

	// The failed pod's signals
	// nil if job succeeded or no failed pods
	FailedPodSignals *PodSignals
	FailedPodName    string
}

// CollectJobSignals fetches job status and
// finds the most recently failed pod
func CollectJobSignals(
	client *kubernetes.Clientset,
	name, namespace string,
	tailLines int64,
	maxEvents int,
) (*JobSignals, error) {

	ctx := context.Background()

	// ── 1. Fetch the job ─────────────────────────────
	job, err := client.BatchV1().
		Jobs(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"job %q not found in namespace %q: %w",
			name, namespace, err)
	}

	signals := &JobSignals{
		JobName:   name,
		Namespace: namespace,
		Active:    job.Status.Active,
		Succeeded: job.Status.Succeeded,
		Failed:    job.Status.Failed,
	}

	// Backoff limit
	if job.Spec.BackoffLimit != nil {
		signals.BackoffLimit = *job.Spec.BackoffLimit
	}
	signals.Retries = job.Status.Failed

	// ── 2. Check job conditions ──────────────────────
	for _, c := range job.Status.Conditions {
		switch c.Type {
		case batchv1.JobComplete:
			if c.Status == corev1.ConditionTrue {
				signals.IsComplete = true
			}
		case batchv1.JobFailed:
			if c.Status == corev1.ConditionTrue {
				signals.IsFailed = true
				signals.FailReason = c.Message
			}
		}
	}

	// ── 3. Find job pods ─────────────────────────────
	pods, err := findJobPods(client, job, namespace)
	if err != nil {
		return signals, nil // non-fatal
	}

	// ── 4. Find most recently failed pod ────────────
	failedPod := findMostRecentFailedPod(pods)
	if failedPod == nil {
		return signals, nil
	}

	signals.FailedPodName = failedPod.Name

	// ── 5. Collect full pod signals ──────────────────
	podSignals, err := CollectPodSignals(
		client,
		failedPod.Name,
		namespace,
		tailLines,
		maxEvents,
	)
	if err != nil {
		return signals, nil // non-fatal
	}

	signals.FailedPodSignals = podSignals
	return signals, nil
}

func findJobPods(
	client *kubernetes.Clientset,
	job *batchv1.Job,
	namespace string,
) ([]corev1.Pod, error) {

	ctx := context.Background()

	if job.Spec.Selector == nil {
		return nil, fmt.Errorf("job has no selector")
	}

	selector := labels.Set(
		job.Spec.Selector.MatchLabels,
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

// findMostRecentFailedPod returns the most recently
// failed pod from a list of job pods
func findMostRecentFailedPod(
	pods []corev1.Pod,
) *corev1.Pod {

	// Collect only failed pods
	var failed []corev1.Pod
	for _, pod := range pods {
		isFailed := false
		if pod.Status.Phase == corev1.PodFailed {
			isFailed = true
		} else {
			// Also check for containers in error state
			for _, cs := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
				if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
					isFailed = true
					break
				}
				if cs.State.Waiting != nil &&
					cs.State.Waiting.Reason != "" &&
					cs.State.Waiting.Reason != "ContainerCreating" &&
					cs.State.Waiting.Reason != "PodInitializing" {
					isFailed = true
					break
				}
			}
		}
		if isFailed {
			failed = append(failed, pod)
		}
	}

	if len(failed) == 0 {
		return nil
	}

	// Sort by most recent failure
	sort.Slice(failed, func(i, j int) bool {
		return failed[i].CreationTimestamp.After(
			failed[j].CreationTimestamp.Time)
	})

	return &failed[0]
}
