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

var pvcCmd = &cobra.Command{
	Use:     "pvc [name]",
	Aliases: []string{"pvcs"},
	Short:   "Explain why a PersistentVolumeClaim is pending or lost",
	Args:    cobra.ExactArgs(1),
	RunE:    runPVCWhy,
}

func init() {
	rootCmd.AddCommand(pvcCmd)
}

func runPVCWhy(cmd *cobra.Command, args []string) error {
	pvcName := args[0]
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to cluster: %v\n", err)
		os.Exit(1)
	}

	signals, err := kube.CollectPVCSignals(client, pvcName, namespace, maxEvents)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr, "✗ pvc/%s not found in namespace %q\n", pvcName, namespace)
		} else {
			fmt.Fprintf(os.Stderr, "✗ Error accessing pvc: %v\n", err)
		}
		os.Exit(1)
	}

	result := analyzer.AnalyzePVC(signals)

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
