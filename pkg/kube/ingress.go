package kube

import (
	"context"
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// IngressSignals holds all data needed to diagnose an Ingress.
type IngressSignals struct {
	Name      string
	Namespace string
	Ingress   *networkingv1.Ingress
	Events    []EventSignal
}

// CollectIngressSignals fetches an Ingress and its events.
func CollectIngressSignals(
	client *kubernetes.Clientset,
	name, namespace string,
	maxEvents int,
) (*IngressSignals, error) {
	ctx := context.Background()

	ingress, err := client.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("ingress %q not found in namespace %q: %w", name, namespace, err)
	}

	signals := &IngressSignals{
		Name:      name,
		Namespace: namespace,
		Ingress:   ingress,
	}

	// Fetch Events
	events, _ := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.kind=Ingress,involvedObject.name=" + name,
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

// GetIngressBackingServices returns a unique list of Service names referenced by the Ingress.
func (s *IngressSignals) GetIngressBackingServices() []string {
	serviceMap := make(map[string]bool)

	// Check default backend
	if s.Ingress.Spec.DefaultBackend != nil && s.Ingress.Spec.DefaultBackend.Service != nil {
		serviceMap[s.Ingress.Spec.DefaultBackend.Service.Name] = true
	}

	// Check rules
	for _, rule := range s.Ingress.Spec.Rules {
		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service != nil {
					serviceMap[path.Backend.Service.Name] = true
				}
			}
		}
	}

	var services []string
	for svc := range serviceMap {
		services = append(services, svc)
	}
	return services
}
