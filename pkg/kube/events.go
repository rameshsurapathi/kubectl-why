package kube

import (
    "context"
    "sort"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
)

// CollectEvents fetches Kubernetes events for a specific
// pod, sorted by most recent first
func CollectEvents(
    client *kubernetes.Clientset,
    podUID, namespace string,
    maxEvents int,
) ([]EventSignal, error) {

    ctx := context.Background()

    events, err := client.CoreV1().Events(namespace).List(
        ctx, metav1.ListOptions{
            FieldSelector: "involvedObject.uid=" + podUID,
        })
    if err != nil {
        return nil, err
    }

    // Sort: warnings first, then by most recent
    sort.Slice(events.Items, func(i, j int) bool {
        // Warnings bubble to top
        if events.Items[i].Type != events.Items[j].Type {
            return events.Items[i].Type == "Warning"
        }
        // Then most recent
        return events.Items[i].LastTimestamp.After(
            events.Items[j].LastTimestamp.Time)
    })

    var signals []EventSignal
    for i, e := range events.Items {
        if i >= maxEvents {
            break
        }
        signals = append(signals, EventSignal{
            Type:      e.Type,
            Reason:    e.Reason,
            Message:   e.Message,
            Count:     e.Count,
            FirstTime: e.FirstTimestamp.Time,
            LastTime:  e.LastTimestamp.Time,
            Component: e.Source.Component,
        })
    }

    return signals, nil
}
