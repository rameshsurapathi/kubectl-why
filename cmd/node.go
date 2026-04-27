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

var nodeCmd = &cobra.Command{
	Use:     "node [name]",
	Aliases: []string{"no", "nodes"},
	Short:   "Explain why a Node is not ready or has pressure",
	Args:    cobra.ExactArgs(1),
	RunE:    runNodeWhy,
}

func init() {
	rootCmd.AddCommand(nodeCmd)
}

func runNodeWhy(cmd *cobra.Command, args []string) error {
	nodeName := args[0]
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to cluster: %v\n", err)
		os.Exit(1)
	}

	signals, err := kube.CollectNodeSignals(client, nodeName, maxEvents)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr, "✗ node/%s not found\n", nodeName)
		} else {
			fmt.Fprintf(os.Stderr, "✗ Error accessing node: %v\n", err)
		}
		os.Exit(1)
	}

	result := analyzer.AnalyzeNode(signals)

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
