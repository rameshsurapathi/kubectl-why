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

// Build-time version info — injected by goreleaser via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// rootCmd represents the base command when called without any subcommands

var rootCmd = &cobra.Command{
	Use:     "kubectl-why",
	Version: version,
	Short:   "Explain why a Kubernetes resource is failing",
	Long: `kubectl why diagnoses failing Kubernetes pods, 
deployments, and jobs — without switching between 
kubectl describe, kubectl logs, and kubectl get events.`,
}

// To start Cobra's command handling, read the command-line argument
// figure out which command/subcommand was used, run the matching function

func Execute() {
	setupCommandOrder()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

//In Go, init() is a special function. It runs automatically before main().

func init() {
	// Set a rich version string shown by --version
	rootCmd.SetVersionTemplate(
		"kubectl-why {{.Version}} (commit: " + commit + ", built: " + date + ")\n",
	)
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
		
	rootCmd.AddGroup(&cobra.Group{
		ID:    "primary",
		Title: "Primary Commands:",
	})
	
	rootCmd.AddGroup(&cobra.Group{
		ID:    "core",
		Title: "Core Workloads & Storage:",
	})
	
	rootCmd.AddGroup(&cobra.Group{
		ID:    "other",
		Title: "Other Resources:",
	})
}

var (
	// Defaulted to true per user request, removed as explicit flags to keep tool simple
	explainFlag       = true
	showSecondaryFlag = true
)

func setupCommandOrder() {
	cobra.EnableCommandSorting = false

	// Define the exact order of commands
	order := []string{
		"help",
		"view",
		"pod",
		"deployment",
		"statefulset",
		"daemonset",
		"service",
		"pvc",
		"job",
		"cronjob",
		"hpa",
		"networkpolicy",
		"pdb",
		"rolebinding",
		"resourcequota",
		"tls",
		"node",
		"dns",
		"rollout",
	}

	cmdMap := make(map[string]*cobra.Command)
	for _, c := range rootCmd.Commands() {
		cmdMap[c.Name()] = c
	}

	// Remove all
	for _, c := range cmdMap {
		rootCmd.RemoveCommand(c)
	}

	// Add back in specific order
	for _, name := range order {
		if c, ok := cmdMap[name]; ok {
			// Assign to groups for better categorization
			if name == "view" {
				c.GroupID = "primary"
			} else if name == "pod" || name == "deployment" || name == "statefulset" || name == "daemonset" || name == "service" || name == "pvc" {
				c.GroupID = "core"
			} else if name != "help" {
				c.GroupID = "other"
			}
			
			rootCmd.AddCommand(c)
			delete(cmdMap, name)
		}
	}

	// Add any leftovers
	for _, c := range cmdMap {
		c.GroupID = "other"
		rootCmd.AddCommand(c)
	}
}
