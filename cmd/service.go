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

var serviceCmd = &cobra.Command{
	Use:     "service [name]",
	Aliases: []string{"svc", "services"},
	Short:   "Explain why a Service is failing to route traffic",
	Args:    cobra.ExactArgs(1),
	RunE:    runServiceWhy,
}

func init() {
	rootCmd.AddCommand(serviceCmd)
}

func runServiceWhy(cmd *cobra.Command, args []string) error {
	svcName := args[0]
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to cluster: %v\n", err)
		os.Exit(1)
	}

	signals, err := kube.CollectServiceSignals(client, svcName, namespace, maxEvents)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr, "✗ service/%s not found in namespace %q\n", svcName, namespace)
		} else {
			fmt.Fprintf(os.Stderr, "✗ Error accessing service: %v\n", err)
		}
		os.Exit(1)
	}

	result := analyzer.AnalyzeService(signals)

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
