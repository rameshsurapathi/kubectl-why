package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
	"github.com/rameshsurapathi/kubectl-why/pkg/render"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var scanShowHealthy bool

var scanCmd = &cobra.Command{
	Use:     "scan",
	Aliases: []string{"ns"},
	Short:   "Scan a namespace for common Kubernetes issues",
	Example: "  kubectl-why scan -n production\n" +
		"  kubectl-why scan -n production --show-healthy\n" +
		"  kubectl-why scan -n production -o json",
	Args: cobra.NoArgs,
	RunE: runScanNamespace,
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().BoolVar(&scanShowHealthy,
		"show-healthy", false, "Include healthy resources in scan output")
}

func runScanNamespace(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"\n  ✗  cannot connect to cluster: %v\n\n",
			err)
		os.Exit(1)
	}

	results, err := collectNamespaceResults(
		client, namespace, tailLines, maxEvents)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"\n  ✗  cannot scan namespace %q: %v\n\n",
			namespace, err)
		os.Exit(1)
	}

	result := analyzer.BuildScanResult(namespace, results)

	switch outputFmt {
	case "json":
		return render.ScanJSON(result)
	default:
		return render.ScanText(result, scanShowHealthy)
	}
}

func collectNamespaceResults(
	client *kubernetes.Clientset,
	namespace string,
	tailLines int64,
	maxEvents int,
) ([]analyzer.AnalysisResult, error) {
	ctx := context.Background()
	var results []analyzer.AnalysisResult

	if _, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{}); err != nil {
		return nil, err
	}

	deployments, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, deployment := range deployments.Items {
			signals, err := kube.CollectDeploymentSignals(
				client, deployment.Name, namespace, tailLines, maxEvents)
			if err == nil {
				results = append(results, analyzer.AnalyzeDeployment(signals))
			}
		}
	}

	jobs, err := client.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, job := range jobs.Items {
			signals, err := kube.CollectJobSignals(
				client, job.Name, namespace, tailLines, maxEvents)
			if err == nil {
				results = append(results, analyzer.AnalyzeJob(signals))
			}
		}
	}

	cronJobs, err := client.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, cronJob := range cronJobs.Items {
			signals, err := kube.CollectCronJobSignals(client, cronJob.Name, namespace, maxEvents)
			if err == nil {
				results = append(results, analyzer.AnalyzeCronJob(signals))
			}
		}
	}

	services, err := client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, service := range services.Items {
			signals, err := kube.CollectServiceSignals(client, service.Name, namespace, maxEvents)
			if err == nil {
				results = append(results, analyzer.AnalyzeService(signals))
			}
		}
	}

	pvcs, err := client.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, pvc := range pvcs.Items {
			signals, err := kube.CollectPVCSignals(client, pvc.Name, namespace, maxEvents)
			if err == nil {
				results = append(results, analyzer.AnalyzePVC(signals))
			}
		}
	}

	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, pod := range pods.Items {
			if hasControllerOwner(pod) {
				continue
			}
			signals, err := kube.CollectPodSignals(
				client, pod.Name, namespace, tailLines, maxEvents)
			if err == nil {
				results = append(results, analyzer.AnalyzePod(signals))
			}
		}
	}

	return results, nil
}

func hasControllerOwner(pod corev1.Pod) bool {
	for _, owner := range pod.OwnerReferences {
		if owner.Controller != nil && *owner.Controller {
			return true
		}
	}
	return false
}
