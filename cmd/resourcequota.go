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

var resourcequotaCmd = &cobra.Command{
	Use:     "resourcequota [name]",
	Aliases: []string{"rq", "quota", "quotas"},
	Short:   "Explain why a resourcequota is blocking resources",
	Example: "  kubectl-why resourcequota default-quota -n production\n" +
		"  kubectl-why rq default-quota -n production",
	Args: cobra.ExactArgs(1),
	RunE: runResourceQuotaWhy,
}

func init() {
	rootCmd.AddCommand(resourcequotaCmd)
}

func runResourceQuotaWhy(
	cmd *cobra.Command,
	args []string,
) error {
	rqName := args[0]

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"\n  ✗  cannot connect to cluster: %v\n\n",
			err)
		os.Exit(1)
	}

	signals, err := kube.CollectResourceQuotaSignals(
		client, rqName, namespace,
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr,
				"\n  ✗  resourcequota/%s not found "+
					"in namespace %q\n\n"+
					"  Tip: Check available resourcequotas:\n"+
					"       kubectl get resourcequotas -n %s\n\n",
				rqName, namespace, namespace)
		} else {
			fmt.Fprintf(os.Stderr, "\n  ✗  Error accessing resourcequota: %v\n\n", err)
		}
		os.Exit(1)
	}

	result := analyzer.AnalyzeResourceQuota(signals)

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
