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

var statefulsetCmd = &cobra.Command{
	Use:     "statefulset [name]",
	Aliases: []string{"sts", "statefulsets"},
	Short:   "Explain why a statefulset is unhealthy",
	Example: "  kubectl-why statefulset web -n production\n" +
		"  kubectl-why sts web -n production",
	Args: cobra.ExactArgs(1),
	RunE: runStatefulSetWhy,
}

func init() {
	rootCmd.AddCommand(statefulsetCmd)
}

func runStatefulSetWhy(
	cmd *cobra.Command,
	args []string,
) error {
	stsName := args[0]

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"\n  ✗  cannot connect to cluster: %v\n\n",
			err)
		os.Exit(1)
	}

	signals, err := kube.CollectStatefulSetSignals(
		client, stsName, namespace,
		tailLines, maxEvents,
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr,
				"\n  ✗  statefulset/%s not found "+
					"in namespace %q\n\n"+
					"  Tip: Check available statefulsets:\n"+
					"       kubectl get statefulsets -n %s\n\n",
				stsName, namespace, namespace)
		} else {
			fmt.Fprintf(os.Stderr, "\n  ✗  Error accessing statefulset: %v\n\n", err)
		}
		os.Exit(1)
	}

	result := analyzer.AnalyzeStatefulSet(signals)

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
