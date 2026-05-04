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

var hpaCmd = &cobra.Command{
	Use:     "hpa [name]",
	Aliases: []string{"hpas"},
	Short:   "Explain why a horizontal pod autoscaler is failing",
	Example: "  kubectl-why hpa api-server -n production",
	Args: cobra.ExactArgs(1),
	RunE: runHPAWhy,
}

func init() {
	rootCmd.AddCommand(hpaCmd)
}

func runHPAWhy(
	cmd *cobra.Command,
	args []string,
) error {
	hpaName := args[0]

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"\n  ✗  cannot connect to cluster: %v\n\n",
			err)
		os.Exit(1)
	}

	signals, err := kube.CollectHPASignals(
		client, hpaName, namespace, maxEvents,
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr,
				"\n  ✗  hpa/%s not found "+
					"in namespace %q\n\n"+
					"  Tip: Check available hpas:\n"+
					"       kubectl get hpa -n %s\n\n",
				hpaName, namespace, namespace)
		} else {
			fmt.Fprintf(os.Stderr, "\n  ✗  Error accessing hpa: %v\n\n", err)
		}
		os.Exit(1)
	}

	result := analyzer.AnalyzeHPA(signals)

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
