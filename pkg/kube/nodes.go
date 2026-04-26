package kube

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type NodeSignals struct {
	Name string

	Conditions    []corev1.NodeCondition
	Unschedulable bool
	Taints        []corev1.Taint

	Capacity    corev1.ResourceList
	Allocatable corev1.ResourceList

	Events []EventSignal
}

func CollectNodeSignals(client *kubernetes.Clientset, name string, maxEvents int) (*NodeSignals, error) {
	ctx := context.Background()

	node, err := client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("node %q not found: %w", name, err)
	}

	signals := &NodeSignals{
		Name:          node.Name,
		Conditions:    node.Status.Conditions,
		Unschedulable: node.Spec.Unschedulable,
		Taints:        node.Spec.Taints,
		Capacity:      node.Status.Capacity,
		Allocatable:   node.Status.Allocatable,
	}

	events, _ := client.CoreV1().Events(corev1.NamespaceAll).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.kind=Node,involvedObject.name=" + name,
	})
	for _, e := range events.Items {
		if e.Type == "Warning" {
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
	}
	sortEvents(signals.Events)
	if maxEvents > 0 && len(signals.Events) > maxEvents {
		signals.Events = signals.Events[:maxEvents]
	}

	return signals, nil
}
