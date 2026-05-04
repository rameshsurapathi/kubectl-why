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

var rbacCmd = &cobra.Command{
	Use:     "rolebinding [name]",
	Aliases: []string{"rb"},
	Short:   "Explain why a rolebinding might be failing",
	Example: "  kubectl-why rolebinding admin-binding -n default",
	Args: cobra.ExactArgs(1),
	RunE: runRoleBindingWhy,
}

func init() {
	rootCmd.AddCommand(rbacCmd)
}

func runRoleBindingWhy(
	cmd *cobra.Command,
	args []string,
) error {
	rbName := args[0]

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"\n  ✗  cannot connect to cluster: %v\n\n",
			err)
		os.Exit(1)
	}

	signals, err := kube.CollectRoleBindingSignals(
		client, rbName, namespace,
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr,
				"\n  ✗  rolebinding/%s not found "+
					"in namespace %q\n\n",
				rbName, namespace)
		} else {
			fmt.Fprintf(os.Stderr, "\n  ✗  Error accessing rolebinding: %v\n\n", err)
		}
		os.Exit(1)
	}

	result := analyzer.AnalyzeRoleBinding(signals)

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
