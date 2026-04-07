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

var jobCmd = &cobra.Command{
	Use:     "job [name]",
	Aliases: []string{"jobs"},
	Short:   "Explain why a job failed",
	Example: "  kubectl-why job nightly-sync -n batch",
	Args:    cobra.ExactArgs(1),
	RunE:    runJobWhy,
}

func init() {
	rootCmd.AddCommand(jobCmd)
}

func runJobWhy(
	cmd *cobra.Command,
	args []string,
) error {
	jobName := args[0]

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

	signals, err := kube.CollectJobSignals(
		client, jobName, namespace,
		tailLines, maxEvents,
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr,
				"\n  ✗  job/%s not found "+
					"in namespace %q\n\n"+
					"  Tip: Check available jobs:\n"+
					"       kubectl get jobs -n %s\n\n",
				jobName, namespace, namespace)
		} else {
			fmt.Fprintf(os.Stderr, "\n  ✗  Error accessing job: %v\n\n", err)
		}
		os.Exit(1)
	}

	result := analyzer.AnalyzeJob(signals)

	switch outputFmt {
	case "json":
		return render.JSON(result)
	default:
		return render.Text(result)
	}
}
