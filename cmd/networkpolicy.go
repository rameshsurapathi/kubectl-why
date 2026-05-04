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

var networkpolicyCmd = &cobra.Command{
	Use:     "networkpolicy [name]",
	Aliases: []string{"netpol", "networkpolicies"},
	Short:   "Explain if a networkpolicy is blocking traffic",
	Example: "  kubectl-why networkpolicy default-deny -n production\n" +
		"  kubectl-why netpol default-deny -n production",
	Args: cobra.ExactArgs(1),
	RunE: runNetworkPolicyWhy,
}

func init() {
	rootCmd.AddCommand(networkpolicyCmd)
}

func runNetworkPolicyWhy(
	cmd *cobra.Command,
	args []string,
) error {
	npName := args[0]

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"\n  ✗  cannot connect to cluster: %v\n\n",
			err)
		os.Exit(1)
	}

	signals, err := kube.CollectNetworkPolicySignals(
		client, npName, namespace,
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr,
				"\n  ✗  networkpolicy/%s not found "+
					"in namespace %q\n\n"+
					"  Tip: Check available networkpolicies:\n"+
					"       kubectl get networkpolicies -n %s\n\n",
				npName, namespace, namespace)
		} else {
			fmt.Fprintf(os.Stderr, "\n  ✗  Error accessing networkpolicy: %v\n\n", err)
		}
		os.Exit(1)
	}

	result := analyzer.AnalyzeNetworkPolicy(signals)

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
