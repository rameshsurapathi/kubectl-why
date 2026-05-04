package kube

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ResourceQuotaSignals holds the hard and used resource limits
type ResourceQuotaSignals struct {
	Name      string
	Namespace string
	Hard      corev1.ResourceList
	Used      corev1.ResourceList
}

// CollectResourceQuotaSignals fetches the resource quota and its current usage
func CollectResourceQuotaSignals(
	client *kubernetes.Clientset,
	name, namespace string,
) (*ResourceQuotaSignals, error) {

	ctx := context.Background()

	rq, err := client.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"resourcequota %q not found in namespace %q: %w",
			name, namespace, err)
	}

	signals := &ResourceQuotaSignals{
		Name:      name,
		Namespace: namespace,
		Hard:      rq.Status.Hard,
		Used:      rq.Status.Used,
	}

	return signals, nil
}
