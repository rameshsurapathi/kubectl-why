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

var cronjobCmd = &cobra.Command{
	Use:     "cronjob [name]",
	Aliases: []string{"cj", "cronjobs"},
	Short:   "Explain why a CronJob is failing or not scheduling",
	Args:    cobra.ExactArgs(1),
	RunE:    runCronJobWhy,
}

func init() {
	rootCmd.AddCommand(cronjobCmd)
}

func runCronJobWhy(cmd *cobra.Command, args []string) error {
	cjName := args[0]
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to cluster: %v\n", err)
		os.Exit(1)
	}

	signals, err := kube.CollectCronJobSignals(client, cjName, namespace, maxEvents)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr, "✗ cronjob/%s not found in namespace %q\n", cjName, namespace)
		} else {
			fmt.Fprintf(os.Stderr, "✗ Error accessing cronjob: %v\n", err)
		}
		os.Exit(1)
	}

	result := analyzer.AnalyzeCronJob(signals)

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
