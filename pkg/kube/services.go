package kube

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type ServiceSignals struct {
	Name      string
	Namespace string
	Type      string // ClusterIP, NodePort, LoadBalancer, ExternalName
	
	Selector map[string]string
	Ports    []corev1.ServicePort
	
	ExternalName string

	EndpointSlices []discoveryv1.EndpointSlice
	MatchingPods   []corev1.Pod
}

func CollectServiceSignals(client *kubernetes.Clientset, name, namespace string, maxEvents int) (*ServiceSignals, error) {
	ctx := context.Background()

	svc, err := client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("service %q not found in namespace %q: %w", name, namespace, err)
	}

	signals := &ServiceSignals{
		Name:         svc.Name,
		Namespace:    svc.Namespace,
		Type:         string(svc.Spec.Type),
		Selector:     svc.Spec.Selector,
		Ports:        svc.Spec.Ports,
		ExternalName: svc.Spec.ExternalName,
	}

	// Fetch EndpointSlices
	slices, err := client.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("kubernetes.io/service-name=%s", name),
	})
	if err == nil {
		signals.EndpointSlices = slices.Items
	}

	// Fetch Matching Pods if there's a selector
	if len(svc.Spec.Selector) > 0 {
		selector := labels.Set(svc.Spec.Selector).AsSelector()
		pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err == nil {
			signals.MatchingPods = pods.Items
		}
	}

	return signals, nil
}
