package kube

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PDBSignals holds PodDisruptionBudget states
type PDBSignals struct {
	Name               string
	Namespace          string
	MinAvailable       string
	MaxUnavailable     string
	ExpectedPods       int32
	DesiredHealthy     int32
	CurrentHealthy     int32
	DisruptionsAllowed int32
}

// CollectPDBSignals fetches PDB disruption allowance and expectations
func CollectPDBSignals(
	client *kubernetes.Clientset,
	name, namespace string,
) (*PDBSignals, error) {

	ctx := context.Background()

	pdb, err := client.PolicyV1().PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"pdb %q not found in namespace %q: %w",
			name, namespace, err)
	}

	signals := &PDBSignals{
		Name:               name,
		Namespace:          namespace,
		ExpectedPods:       pdb.Status.ExpectedPods,
		DesiredHealthy:     pdb.Status.DesiredHealthy,
		CurrentHealthy:     pdb.Status.CurrentHealthy,
		DisruptionsAllowed: pdb.Status.DisruptionsAllowed,
	}

	if pdb.Spec.MinAvailable != nil {
		signals.MinAvailable = pdb.Spec.MinAvailable.String()
	}
	if pdb.Spec.MaxUnavailable != nil {
		signals.MaxUnavailable = pdb.Spec.MaxUnavailable.String()
	}

	return signals, nil
}
