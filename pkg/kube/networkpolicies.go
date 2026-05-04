package kube

import (
	"context"
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NetworkPolicySignals holds network policy state and matched pod counts
type NetworkPolicySignals struct {
	Name        string
	Namespace   string
	PodSelector metav1.LabelSelector
	PolicyTypes []networkingv1.PolicyType
	HasIngress  bool
	HasEgress   bool
	MatchedPods int
}

// CollectNetworkPolicySignals fetches the network policy and checks how many pods it affects
func CollectNetworkPolicySignals(
	client *kubernetes.Clientset,
	name, namespace string,
) (*NetworkPolicySignals, error) {

	ctx := context.Background()

	np, err := client.NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"networkpolicy %q not found in namespace %q: %w",
			name, namespace, err)
	}

	signals := &NetworkPolicySignals{
		Name:        name,
		Namespace:   namespace,
		PodSelector: np.Spec.PodSelector,
		PolicyTypes: np.Spec.PolicyTypes,
		HasIngress:  len(np.Spec.Ingress) > 0,
		HasEgress:   len(np.Spec.Egress) > 0,
	}

	selector, err := metav1.LabelSelectorAsSelector(&np.Spec.PodSelector)
	if err == nil {
		pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err == nil {
			signals.MatchedPods = len(pods.Items)
		}
	}

	return signals, nil
}
