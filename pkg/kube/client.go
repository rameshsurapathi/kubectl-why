package kube

// These imports provide the Kubernetes client libraries.

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClient initializes and returns a new Kubernetes client (Clientset).
// It acts exactly like 'kubectl' does when finding your credentials.
func NewClient(context string) (*kubernetes.Clientset, error) {
	// 1. Loading Rules: Tell client-go how to find the kubeconfig file.
	// This uses the standard rules: checking the KUBECONFIG env var first.
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()



	// 2. Config Overrides: This allows us to modify the loaded config on the fly.
	// For instance, if the user ran: kubectl-why pod my-pod --context my-cluster-ctx
	configOverrides := &clientcmd.ConfigOverrides{}
	if context != "" {
		configOverrides.CurrentContext = context
	}

	// 3. Client Config: This merges the loading rules and our overrides together.
	// It's "Deferred" meaning it won't actually read the file from disk until
	// we explicitly ask for the final config below.
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	)

	// 4. Parse the file: This reads the ~/.kube/config, parses the certificates,
	// and validates the chosen context. It returns a "RESTConfig".
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	// 5. Create the Clientset: The Clientset is the actual object we use
	// to make API calls like "get pods", "get events", etc.
	return kubernetes.NewForConfig(config)
}
