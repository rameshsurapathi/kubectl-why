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

var tlsCmd = &cobra.Command{
	Use:     "tls [secret-name]",
	Aliases: []string{"cert", "certificate"},
	Short:   "Explain why a TLS certificate is invalid or expired",
	Example: "  kubectl-why tls api-server-cert -n production\n" +
		"  kubectl-why cert api-server-cert -n production",
	Args: cobra.ExactArgs(1),
	RunE: runTLSWhy,
}

func init() {
	rootCmd.AddCommand(tlsCmd)
}

func runTLSWhy(
	cmd *cobra.Command,
	args []string,
) error {
	secretName := args[0]

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	client, err := kube.NewClient(kubeContext)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"\n  ✗  cannot connect to cluster: %v\n\n",
			err)
		os.Exit(1)
	}

	signals, err := kube.CollectTLSSignals(
		client, secretName, namespace,
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(os.Stderr,
				"\n  ✗  secret/%s not found "+
					"in namespace %q\n\n"+
					"  Tip: Check available secrets:\n"+
					"       kubectl get secrets --field-selector type=kubernetes.io/tls -n %s\n\n",
				secretName, namespace, namespace)
		} else {
			fmt.Fprintf(os.Stderr, "\n  ✗  Error accessing secret: %v\n\n", err)
		}
		os.Exit(1)
	}

	result := analyzer.AnalyzeTLS(signals)

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
