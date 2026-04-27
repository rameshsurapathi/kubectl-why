package kube

import (
	"context"
	"fmt"
	"sort"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type CronJobSignals struct {
	Name      string
	Namespace string

	Schedule          string
	Suspend           bool
	ConcurrencyPolicy string

	LastScheduleTime   *metav1.Time
	LastSuccessfulTime *metav1.Time

	ActiveJobs []corev1ObjectReference

	RecentJobs []batchv1.Job
}

type corev1ObjectReference struct {
	Name      string
	Namespace string
}

func CollectCronJobSignals(client *kubernetes.Clientset, name, namespace string, maxEvents int) (*CronJobSignals, error) {
	ctx := context.Background()

	cj, err := client.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("cronjob %q not found in namespace %q: %w", name, namespace, err)
	}

	signals := &CronJobSignals{
		Name:               cj.Name,
		Namespace:          cj.Namespace,
		Schedule:           cj.Spec.Schedule,
		Suspend:            cj.Spec.Suspend != nil && *cj.Spec.Suspend,
		ConcurrencyPolicy:  string(cj.Spec.ConcurrencyPolicy),
		LastScheduleTime:   cj.Status.LastScheduleTime,
		LastSuccessfulTime: cj.Status.LastSuccessfulTime,
	}

	for _, active := range cj.Status.Active {
		signals.ActiveJobs = append(signals.ActiveJobs, corev1ObjectReference{
			Name:      active.Name,
			Namespace: active.Namespace,
		})
	}

	jobs, err := client.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, j := range jobs.Items {
			for _, ref := range j.OwnerReferences {
				if ref.Kind == "CronJob" && ref.Name == cj.Name {
					signals.RecentJobs = append(signals.RecentJobs, j)
					break
				}
			}
		}
	}

	sort.Slice(signals.RecentJobs, func(i, j int) bool {
		return signals.RecentJobs[i].CreationTimestamp.After(signals.RecentJobs[j].CreationTimestamp.Time)
	})

	return signals, nil
}
