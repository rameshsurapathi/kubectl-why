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

var pdbCmd = &cobra.Command{
	Use:     "pdb [name]",
	Aliases: []string{"poddisruptionbudget"},
	Short:   "Explain why a pod disruption budget is blocking evictions",
	Example: "  kubectl-why pdb api-server-pdb -n production",
	Args: cobra.ExactArgs(1),
	RunE: runPDBWhy,
}

func init() {
	rootCmd.AddCommand(pdbCmd)
}

func runPDBWhy(
	cmd *cobra.Command,
	args []string,
) error {
	pdbName := args[0]

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"\n  ✗  cannot connect to cluster: %v\n\n",
			err)
		os.Exit(1)
	}

	signals, err := kube.CollectPDBSignals(
		client, pdbName, namespace,
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr,
				"\n  ✗  pdb/%s not found "+
					"in namespace %q\n\n"+
					"  Tip: Check available pdbs:\n"+
					"       kubectl get pdb -n %s\n\n",
				pdbName, namespace, namespace)
		} else {
			fmt.Fprintf(os.Stderr, "\n  ✗  Error accessing pdb: %v\n\n", err)
		}
		os.Exit(1)
	}

	result := analyzer.AnalyzePDB(signals)

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
