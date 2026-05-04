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

var daemonsetCmd = &cobra.Command{
	Use:     "daemonset [name]",
	Aliases: []string{"ds", "daemonsets"},
	Short:   "Explain why a daemonset is unhealthy",
	Example: "  kubectl-why daemonset fluentd -n kube-system\n" +
		"  kubectl-why ds fluentd -n kube-system",
	Args: cobra.ExactArgs(1),
	RunE: runDaemonSetWhy,
}

func init() {
	rootCmd.AddCommand(daemonsetCmd)
}

func runDaemonSetWhy(
	cmd *cobra.Command,
	args []string,
) error {
	dsName := args[0]

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"\n  ✗  cannot connect to cluster: %v\n\n",
			err)
		os.Exit(1)
	}

	signals, err := kube.CollectDaemonSetSignals(
		client, dsName, namespace,
		tailLines, maxEvents,
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr,
				"\n  ✗  daemonset/%s not found "+
					"in namespace %q\n\n"+
					"  Tip: Check available daemonsets:\n"+
					"       kubectl get daemonsets -n %s\n\n",
				dsName, namespace, namespace)
		} else {
			fmt.Fprintf(os.Stderr, "\n  ✗  Error accessing daemonset: %v\n\n", err)
		}
		os.Exit(1)
	}

	result := analyzer.AnalyzeDaemonSet(signals)

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
