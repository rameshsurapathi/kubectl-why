package kube

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CollectPVCSignals fetches PVC information
func CollectPVCSignals(client *kubernetes.Clientset, pvcName, namespace string, maxEvents int) (*PVCSignals, error) {
	ctx := context.Background()

	pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	signals := &PVCSignals{
		Name:       pvc.Name,
		Namespace:  pvc.Namespace,
		Phase:      string(pvc.Status.Phase),
		VolumeName: pvc.Spec.VolumeName,
	}

	if pvc.Spec.StorageClassName != nil {
		signals.StorageClassName = *pvc.Spec.StorageClassName
	}

	if capacity, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
		signals.Capacity = capacity.String()
	}

	// Fetch Events
	events, _ := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.kind=PersistentVolumeClaim,involvedObject.name=" + pvcName,
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

	// Sort events (newest first)
	sortEvents(signals.Events)
	if maxEvents > 0 && len(signals.Events) > maxEvents {
		signals.Events = signals.Events[:maxEvents]
	}

	return signals, nil
}
