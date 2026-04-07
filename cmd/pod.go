package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
	"github.com/rameshsurapathi/kubectl-why/pkg/render"
	"github.com/spf13/cobra"
)

var podCmd = &cobra.Command{
	Use:   "pod [name]", // Means the command is run as: kubectl-why pod my-pod-name
	Short: "Explain why a pod is failing",
	Args:  cobra.ExactArgs(1), // Ensures the user provides exactly 1 argument (the pod name)
	RunE:  runPod,             // If the command is typed correctly, run the 'runWhy' function
}

func init() {
	rootCmd.AddCommand(podCmd)
}

// This is the conductor of the orchestra.
// It ties all the other packages together in three distinct steps

func runPod(cmd *cobra.Command, args []string) error {
	podName := args[0]

	// We use SilenceUsage and SilenceErrors so Cobra doesn't print double errors
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	// Build kube client
	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to cluster: %v\n", err)
		os.Exit(1)
	}

	// Collect signals
	signals, err := kube.CollectPodSignals(
		client, podName, namespace, tailLines, maxEvents)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			// Don't print raw Go error — print clean message
			fmt.Fprintf(os.Stderr,
				"\n  ✗  pod/%s not found in namespace %q\n\n"+
					"  Tip: Check available pods with:\n"+
					"       kubectl get pods -n %s\n\n",
				podName, namespace, namespace)
		} else {
			// Print for actual RBAC / network failures so they aren't masked
			fmt.Fprintf(os.Stderr, "\n  ✗  Error accessing pod: %v\n\n", err)
		}
		os.Exit(1)
	}

	// Analyze
	result := analyzer.AnalyzePod(signals)

	// Render
	switch outputFmt {
	case "json":
		return render.JSON(result)
	default:
		return render.Text(result)
	}
}
