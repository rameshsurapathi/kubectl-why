package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
	"github.com/rameshsurapathi/kubectl-why/pkg/render"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var traceShowHealthy bool

var traceCmd = &cobra.Command{
	Use:   "trace",
	Short: "Trace relationships between Kubernetes resources",
}

var traceServiceCmd = &cobra.Command{
	Use:     "service [name]",
	Aliases: []string{"svc"},
	Short:   "Trace a Service to endpoints and backing pods",
	Example: "  kubectl-why trace service frontend -n production\n" +
		"  kubectl-why trace svc frontend -n production --show-healthy\n" +
		"  kubectl-why trace service frontend -n production -o json",
	Args: cobra.ExactArgs(1),
	RunE: runTraceService,
}

var traceDeploymentCmd = &cobra.Command{
	Use:     "deployment [name]",
	Aliases: []string{"deploy"},
	Short:   "Trace a Deployment to ReplicaSets, Pods, and dependencies",
	Example: "  kubectl-why trace deployment api -n production\n" +
		"  kubectl-why trace deploy api -n production --show-healthy",
	Args: cobra.ExactArgs(1),
	RunE: runTraceDeployment,
}

var traceIngressCmd = &cobra.Command{
	Use:     "ingress [name]",
	Aliases: []string{"ing"},
	Short:   "Trace an Ingress to Services and backing Pods",
	Example: "  kubectl-why trace ingress api-ingress -n production\n" +
		"  kubectl-why trace ing api-ingress -n production --show-healthy",
	Args: cobra.ExactArgs(1),
	RunE: runTraceIngress,
}

func init() {
	rootCmd.AddCommand(traceCmd)
	traceCmd.AddCommand(traceServiceCmd)
	traceCmd.AddCommand(traceDeploymentCmd)
	traceCmd.AddCommand(traceIngressCmd)
	traceServiceCmd.Flags().BoolVar(&traceShowHealthy,
		"show-healthy", false, "Include healthy backing pods in trace output")
	traceDeploymentCmd.Flags().BoolVar(&traceShowHealthy,
		"show-healthy", false, "Include healthy components in trace output")
	traceIngressCmd.Flags().BoolVar(&traceShowHealthy,
		"show-healthy", false, "Include healthy services and pods in trace output")
}

func runTraceService(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"\n  ✗  cannot connect to cluster: %v\n\n",
			err)
		os.Exit(1)
	}

	result, err := collectServiceTrace(
		client, args[0], namespace, tailLines, maxEvents)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"\n  ✗  cannot trace service/%s in namespace %q: %v\n\n",
			args[0], namespace, err)
		os.Exit(1)
	}

	switch outputFmt {
	case "json":
		return render.ServiceTraceJSON(result)
	default:
		return render.ServiceTraceText(result, traceShowHealthy)
	}
}

func collectServiceTrace(
	client *kubernetes.Clientset,
	name string,
	namespace string,
	tailLines int64,
	maxEvents int,
) (analyzer.ServiceTraceResult, error) {
	signals, err := kube.CollectServiceSignals(client, name, namespace, maxEvents)
	if err != nil {
		return analyzer.ServiceTraceResult{}, err
	}

	serviceResult := analyzer.AnalyzeService(signals)
	pods := serviceBackingPods(signals)

	var podResults []analyzer.AnalysisResult
	for podName := range pods {
		podSignals, err := kube.CollectPodSignals(
			client, podName, namespace, tailLines, maxEvents)
		if err != nil {
			continue
		}
		podResults = append(podResults, analyzer.AnalyzePod(podSignals))
	}

	endpointCount, readyEndpointCount := endpointReadiness(signals)
	return analyzer.BuildServiceTraceResult(
		serviceResult,
		podResults,
		endpointCount,
		readyEndpointCount,
	), nil
}

func serviceBackingPods(signals *kube.ServiceSignals) map[string]struct{} {
	pods := map[string]struct{}{}

	for _, pod := range signals.MatchingPods {
		pods[pod.Name] = struct{}{}
	}

	for _, slice := range signals.EndpointSlices {
		for _, endpoint := range slice.Endpoints {
			if endpoint.TargetRef == nil ||
				endpoint.TargetRef.Kind != "Pod" ||
				endpoint.TargetRef.Name == "" {
				continue
			}
			pods[endpoint.TargetRef.Name] = struct{}{}
		}
	}

	return pods
}

func endpointReadiness(signals *kube.ServiceSignals) (int, int) {
	total := 0
	ready := 0

	for _, slice := range signals.EndpointSlices {
		for _, endpoint := range slice.Endpoints {
			total++
			if isEndpointReady(endpoint.Conditions.Ready) {
				ready++
			}
		}
	}

	return total, ready
}

func isEndpointReady(ready *bool) bool {
	return ready == nil || *ready
}

func runTraceDeployment(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n  ✗  cannot connect to cluster: %v\n\n", err)
		os.Exit(1)
	}

	result, err := collectDeploymentTrace(client, args[0], namespace, tailLines, maxEvents)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n  ✗  cannot trace deployment/%s in namespace %q: %v\n\n", args[0], namespace, err)
		os.Exit(1)
	}

	switch outputFmt {
	case "json":
		return render.DeploymentTraceJSON(result)
	default:
		return render.DeploymentTraceText(result, traceShowHealthy)
	}
}

func collectDeploymentTrace(
	client *kubernetes.Clientset,
	name string,
	namespace string,
	tailLines int64,
	maxEvents int,
) (analyzer.DeploymentTraceResult, error) {
	// 1. Collect Deployment signals
	signals, err := kube.CollectDeploymentSignals(client, name, namespace, tailLines, maxEvents)
	if err != nil {
		return analyzer.DeploymentTraceResult{}, err
	}

	deploymentResult := analyzer.AnalyzeDeployment(signals)

	// 2. Fetch ReplicaSets
	ctx := context.Background()
	deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return analyzer.DeploymentTraceResult{}, err
	}

	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return analyzer.DeploymentTraceResult{}, err
	}

	rsList, err := client.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return analyzer.DeploymentTraceResult{}, err
	}

	var rsResults []analyzer.AnalysisResult
	for _, rs := range rsList.Items {
		// Only consider ReplicaSets owned by this Deployment
		isOwned := false
		for _, ref := range rs.OwnerReferences {
			if ref.UID == deployment.UID {
				isOwned = true
				break
			}
		}
		if !isOwned {
			continue
		}

		// Simple analysis for ReplicaSet
		status := "Healthy"
		primaryReason := "ReplicaSet healthy"
		severity := "healthy"
		summary := []string{"ReplicaSet has all replicas available."}

		if rs.Status.Replicas != rs.Status.AvailableReplicas {
			status = "Degraded"
			primaryReason = "ReplicaSet unavailable"
			severity = "warning"
			summary = []string{fmt.Sprintf("%d/%d replicas available", rs.Status.AvailableReplicas, rs.Status.Replicas)}
		}

		rsResults = append(rsResults, analyzer.AnalysisResult{
			SchemaVersion: "v2",
			Resource:      "replicaset/" + rs.Name,
			Namespace:     namespace,
			Status:        status,
			PrimaryReason: primaryReason,
			Severity:      severity,
			Summary:       summary,
			Findings: []analyzer.Finding{
				{
					Category:       "ReplicaSet",
					ReasonCode:     replicaSetReasonCode(severity),
					Confidence:     "high",
					AffectedObject: "replicaset/" + rs.Name,
					Message:        summary[0],
				},
			},
		})
	}

	// 3. Fetch Pods and collect dependent resources
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return analyzer.DeploymentTraceResult{}, err
	}

	var podResults []analyzer.AnalysisResult
	pvcs := map[string]bool{}
	configMaps := map[string]bool{}
	secrets := map[string]bool{}
	serviceAccounts := map[string]bool{}

	for _, pod := range pods.Items {
		podSignals, err := kube.CollectPodSignals(client, pod.Name, namespace, tailLines, maxEvents)
		if err != nil {
			continue
		}
		podResults = append(podResults, analyzer.AnalyzePod(podSignals))
		collectPodDependencies(pod, pvcs, configMaps, secrets, serviceAccounts)
	}

	// 4. Check dependencies
	var depResults []analyzer.AnalysisResult

	// Check PVCs
	for pvcName := range pvcs {
		pvcSignals, err := kube.CollectPVCSignals(client, pvcName, namespace, maxEvents)
		if err == nil {
			depResults = append(depResults, analyzer.AnalyzePVC(pvcSignals))
		} else {
			depResults = append(depResults, missingDependency("pvc", pvcName, namespace))
		}
	}

	// Check ServiceAccounts
	for serviceAccountName := range serviceAccounts {
		_, err := client.CoreV1().ServiceAccounts(namespace).Get(ctx, serviceAccountName, metav1.GetOptions{})
		if err != nil {
			depResults = append(depResults, missingDependency("serviceaccount", serviceAccountName, namespace))
		} else {
			depResults = append(depResults, healthyDependency("serviceaccount", serviceAccountName, namespace))
		}
	}

	// Check ConfigMaps
	for cmName := range configMaps {
		_, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, cmName, metav1.GetOptions{})
		if err != nil {
			depResults = append(depResults, missingDependency("configmap", cmName, namespace))
		} else {
			depResults = append(depResults, healthyDependency("configmap", cmName, namespace))
		}
	}

	// Check Secrets
	for secretName := range secrets {
		_, err := client.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
		if err != nil {
			depResults = append(depResults, missingDependency("secret", secretName, namespace))
		} else {
			depResults = append(depResults, healthyDependency("secret", secretName, namespace))
		}
	}

	return analyzer.BuildDeploymentTraceResult(
		deploymentResult,
		rsResults,
		podResults,
		depResults,
	), nil
}

func collectPodDependencies(
	pod corev1.Pod,
	pvcs map[string]bool,
	configMaps map[string]bool,
	secrets map[string]bool,
	serviceAccounts map[string]bool,
) {
	if pod.Spec.ServiceAccountName != "" && pod.Spec.ServiceAccountName != "default" {
		serviceAccounts[pod.Spec.ServiceAccountName] = true
	}
	for _, secretRef := range pod.Spec.ImagePullSecrets {
		secrets[secretRef.Name] = true
	}
	for _, vol := range pod.Spec.Volumes {
		if vol.PersistentVolumeClaim != nil {
			pvcs[vol.PersistentVolumeClaim.ClaimName] = true
		}
		if vol.ConfigMap != nil && !optionalBool(vol.ConfigMap.Optional) {
			configMaps[vol.ConfigMap.Name] = true
		}
		if vol.Secret != nil && !optionalBool(vol.Secret.Optional) {
			secrets[vol.Secret.SecretName] = true
		}
		if vol.Projected != nil {
			if strings.HasPrefix(vol.Name, "kube-api-access-") {
				continue
			}
			for _, source := range vol.Projected.Sources {
				if source.ConfigMap != nil && !optionalBool(source.ConfigMap.Optional) {
					configMaps[source.ConfigMap.Name] = true
				}
				if source.Secret != nil && !optionalBool(source.Secret.Optional) {
					secrets[source.Secret.Name] = true
				}
			}
		}
	}

	containers := make([]corev1.Container, 0, len(pod.Spec.InitContainers)+len(pod.Spec.Containers))
	containers = append(containers, pod.Spec.InitContainers...)
	containers = append(containers, pod.Spec.Containers...)
	for _, container := range containers {
		for _, envFrom := range container.EnvFrom {
			if envFrom.ConfigMapRef != nil && !optionalBool(envFrom.ConfigMapRef.Optional) {
				configMaps[envFrom.ConfigMapRef.Name] = true
			}
			if envFrom.SecretRef != nil && !optionalBool(envFrom.SecretRef.Optional) {
				secrets[envFrom.SecretRef.Name] = true
			}
		}
		for _, env := range container.Env {
			if env.ValueFrom == nil {
				continue
			}
			if env.ValueFrom.ConfigMapKeyRef != nil && !optionalBool(env.ValueFrom.ConfigMapKeyRef.Optional) {
				configMaps[env.ValueFrom.ConfigMapKeyRef.Name] = true
			}
			if env.ValueFrom.SecretKeyRef != nil && !optionalBool(env.ValueFrom.SecretKeyRef.Optional) {
				secrets[env.ValueFrom.SecretKeyRef.Name] = true
			}
		}
	}
}

func optionalBool(value *bool) bool {
	return value != nil && *value
}

func replicaSetReasonCode(severity string) string {
	if severity == "healthy" {
		return "REPLICASET_HEALTHY"
	}
	return "REPLICASET_UNAVAILABLE"
}

func missingDependency(kind, name, namespace string) analyzer.AnalysisResult {
	reason := dependencyReason(kind)
	message := fmt.Sprintf("The required %s does not exist.", kind)
	result := analyzer.AnalysisResult{
		SchemaVersion: "v2",
		Resource:      kind + "/" + name,
		Namespace:     namespace,
		Status:        "NotFound",
		PrimaryReason: reason,
		Severity:      "critical",
		Summary:       []string{message},
	}
	result.Findings = []analyzer.Finding{
		{
			Category:       "Dependency",
			ReasonCode:     dependencyReasonCode(kind),
			Confidence:     "high",
			AffectedObject: result.Resource,
			Message:        message,
		},
	}
	return result
}

func healthyDependency(kind, name, namespace string) analyzer.AnalysisResult {
	return analyzer.AnalysisResult{
		SchemaVersion: "v2",
		Resource:      kind + "/" + name,
		Namespace:     namespace,
		Status:        "Healthy",
		PrimaryReason: "Dependency exists",
		Severity:      "healthy",
		Summary:       []string{"The required " + kind + " exists."},
		Findings: []analyzer.Finding{
			{
				Category:       "Dependency",
				ReasonCode:     "DEPENDENCY_EXISTS",
				Confidence:     "high",
				AffectedObject: kind + "/" + name,
				Message:        "The required " + kind + " exists.",
			},
		},
	}
}

func dependencyReason(kind string) string {
	switch kind {
	case "configmap":
		return "ConfigMap Not Found"
	case "secret":
		return "Secret Not Found"
	case "serviceaccount":
		return "ServiceAccount Not Found"
	case "pvc":
		return "PVC Not Found"
	default:
		return "Dependency Not Found"
	}
}

func dependencyReasonCode(kind string) string {
	switch kind {
	case "configmap":
		return "CONFIGMAP_NOT_FOUND"
	case "secret":
		return "SECRET_NOT_FOUND"
	case "serviceaccount":
		return "SERVICE_ACCOUNT_NOT_FOUND"
	case "pvc":
		return "PVC_NOT_FOUND"
	default:
		return "DEPENDENCY_NOT_FOUND"
	}
}

func runTraceIngress(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n  ✗  cannot connect to cluster: %v\n\n", err)
		os.Exit(1)
	}

	result, err := collectIngressTrace(client, args[0], namespace, tailLines, maxEvents)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n  ✗  cannot trace ingress/%s in namespace %q: %v\n\n", args[0], namespace, err)
		os.Exit(1)
	}

	switch outputFmt {
	case "json":
		return render.IngressTraceJSON(result)
	default:
		return render.IngressTraceText(result, traceShowHealthy)
	}
}

func collectIngressTrace(
	client *kubernetes.Clientset,
	name string,
	namespace string,
	tailLines int64,
	maxEvents int,
) (analyzer.IngressTraceResult, error) {
	// 1. Collect Ingress signals
	signals, err := kube.CollectIngressSignals(client, name, namespace, maxEvents)
	if err != nil {
		return analyzer.IngressTraceResult{}, err
	}

	ingressResult := analyzer.AnalyzeIngress(signals)

	// 2. Fetch and Trace backing Services
	backingServices := signals.GetIngressBackingServices()
	var serviceTraces []analyzer.ServiceTraceResult

	for _, svcName := range backingServices {
		st, err := collectServiceTrace(client, svcName, namespace, tailLines, maxEvents)
		if err == nil {
			serviceTraces = append(serviceTraces, st)
		} else {
			// Service might be missing
			missingSvc := analyzer.AnalysisResult{
				Resource:      "service/" + svcName,
				Namespace:     namespace,
				Status:        "NotFound",
				PrimaryReason: "Service Not Found",
				Severity:      "critical",
				Summary:       []string{"The Ingress routes traffic to a Service that does not exist."},
			}
			serviceTraces = append(serviceTraces, analyzer.ServiceTraceResult{
				SchemaVersion: "v3",
				Resource:      missingSvc.Resource,
				Namespace:     missingSvc.Namespace,
				Status:        "Critical",
				Summary:       []string{"Service does not exist"},
				Service:       missingSvc,
			})
		}
	}

	return analyzer.BuildIngressTraceResult(
		ingressResult,
		serviceTraces,
	), nil
}
