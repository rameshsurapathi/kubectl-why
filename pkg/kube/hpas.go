package kube

import (
	"context"
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HPASignals holds autoscaling signals and conditions
type HPASignals struct {
	Name            string
	Namespace       string
	MinReplicas     *int32
	MaxReplicas     int32
	CurrentReplicas int32
	DesiredReplicas int32
	Conditions      []autoscalingv2.HorizontalPodAutoscalerCondition
	Metrics         []autoscalingv2.MetricSpec
	Events          []EventSignal
}

// CollectHPASignals fetches HPA metrics and conditions
func CollectHPASignals(
	client *kubernetes.Clientset,
	name, namespace string,
	maxEvents int,
) (*HPASignals, error) {

	ctx := context.Background()

	hpa, err := client.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"hpa %q not found in namespace %q: %w",
			name, namespace, err)
	}

	signals := &HPASignals{
		Name:            name,
		Namespace:       namespace,
		MinReplicas:     hpa.Spec.MinReplicas,
		MaxReplicas:     hpa.Spec.MaxReplicas,
		CurrentReplicas: hpa.Status.CurrentReplicas,
		DesiredReplicas: hpa.Status.DesiredReplicas,
		Conditions:      hpa.Status.Conditions,
		Metrics:         hpa.Spec.Metrics,
	}

	events, _ := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.kind=HorizontalPodAutoscaler,involvedObject.name=" + name,
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

	return signals, nil
}
