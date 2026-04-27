package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
	"github.com/rameshsurapathi/kubectl-why/pkg/render"
)

var rolloutCmd = &cobra.Command{
	Use:   "rollout",
	Short: "Explain rollout issues for resources",
}

var rolloutDeploymentCmd = &cobra.Command{
	Use:     "deployment [name]",
	Aliases: []string{"deploy"},
	Short:   "Explain why a deployment rollout is stuck or failing",
	Args:    cobra.ExactArgs(1),
	RunE:    runRolloutDeploymentWhy,
}

func init() {
	rootCmd.AddCommand(rolloutCmd)
	rolloutCmd.AddCommand(rolloutDeploymentCmd)
}

func runRolloutDeploymentWhy(cmd *cobra.Command, args []string) error {
	deployName := args[0]
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to cluster: %v\n", err)
		os.Exit(1)
	}

	signals, err := kube.CollectDeploymentSignals(client, deployName, namespace, tailLines, maxEvents)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr, "✗ deployment/%s not found in namespace %q\n", deployName, namespace)
		} else {
			fmt.Fprintf(os.Stderr, "✗ Error accessing deployment: %v\n", err)
		}
		os.Exit(1)
	}

	// We can add a RolloutAnalyzer or enhance AnalyzeDeployment
	result := analyzer.AnalyzeDeploymentRollout(signals)

	switch outputFmt {
	case "json":
		return render.JSON(result)
	default:
		return render.Text(result, render.Options{
			Explain:       explainFlag,
			ShowSecondary: showSecondaryFlag,
		})
	}
}
