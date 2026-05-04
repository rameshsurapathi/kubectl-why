package kube

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// RoleBindingSignals holds the state of an RBAC binding
type RoleBindingSignals struct {
	Name      string
	Namespace string
	RoleRef   rbacv1.RoleRef
	Subjects  []rbacv1.Subject
	RoleFound bool
}

// CollectRoleBindingSignals fetches a RoleBinding and checks if its referenced role exists
func CollectRoleBindingSignals(
	client *kubernetes.Clientset,
	name, namespace string,
) (*RoleBindingSignals, error) {

	ctx := context.Background()

	rb, err := client.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"rolebinding %q not found in namespace %q: %w",
			name, namespace, err)
	}

	signals := &RoleBindingSignals{
		Name:      name,
		Namespace: namespace,
		RoleRef:   rb.RoleRef,
		Subjects:  rb.Subjects,
		RoleFound: true,
	}

	if rb.RoleRef.Kind == "Role" {
		_, err := client.RbacV1().Roles(namespace).Get(ctx, rb.RoleRef.Name, metav1.GetOptions{})
		if err != nil {
			signals.RoleFound = false
		}
	} else if rb.RoleRef.Kind == "ClusterRole" {
		_, err := client.RbacV1().ClusterRoles().Get(ctx, rb.RoleRef.Name, metav1.GetOptions{})
		if err != nil {
			signals.RoleFound = false
		}
	}

	return signals, nil
}
