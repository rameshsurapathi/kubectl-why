//the "cmd" package, which main.go imported

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// Global flags — shared across all subcommands
var (
	namespace   string
	kubeContext string
	outputFmt   string
	maxEvents   int
	tailLines   int64
)

// rootCmd represents the base command when called without any subcommands

var rootCmd = &cobra.Command{
	Use:   "kubectl-why",
	Short: "Explain why a Kubernetes resource is failing",
	Long: `kubectl why diagnoses failing Kubernetes pods, 
deployments, and jobs — without switching between 
kubectl describe, kubectl logs, and kubectl get events.`,
}

// To start Cobra's command handling, read the command-line argument
// figure out which command/subcommand was used, run the matching function

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

//In Go, init() is a special function. It runs automatically before main().

func init() {
	// Hide the default completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	// Global persistent flags — available on all subcommands
	rootCmd.PersistentFlags().StringVarP(&namespace,
		"namespace", "n", "default", "Kubernetes namespace")
	rootCmd.PersistentFlags().StringVar(&kubeContext,
		"context", "", "Kubernetes context")
	rootCmd.PersistentFlags().StringVarP(&outputFmt,
		"output", "o", "text", "Output format: text|json")
	rootCmd.PersistentFlags().IntVar(&maxEvents,
		"events", 5, "Max events to show")
	rootCmd.PersistentFlags().Int64Var(&tailLines,
		"tail", 20, "Number of log lines to fetch")

}
