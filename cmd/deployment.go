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

var deploymentCmd = &cobra.Command{
	Use:     "deployment [name]",
	Aliases: []string{"deploy", "deployments"},
	Short:   "Explain why a deployment is unhealthy",
	Example: "  kubectl-why deployment api -n production\n" +
		"  kubectl-why deploy api -n production",
	Args: cobra.ExactArgs(1),
	RunE: runDeploymentWhy,
}

func init() {
	rootCmd.AddCommand(deploymentCmd)
}

func runDeploymentWhy(
	cmd *cobra.Command,
	args []string,
) error {
	deployName := args[0]

	// We use SilenceUsage and SilenceErrors so Cobra doesn't print double errors
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"\n  ✗  cannot connect to cluster: %v\n\n",
			err)
		os.Exit(1)
	}

	signals, err := kube.CollectDeploymentSignals(
		client, deployName, namespace,
		tailLines, maxEvents,
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr,
				"\n  ✗  deployment/%s not found "+
					"in namespace %q\n\n"+
					"  Tip: Check available deployments:\n"+
					"       kubectl get deployments -n %s\n\n",
				deployName, namespace, namespace)
		} else {
			fmt.Fprintf(os.Stderr, "\n  ✗  Error accessing deployment: %v\n\n", err)
		}
		os.Exit(1)
	}

	result := analyzer.AnalyzeDeployment(signals)

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
