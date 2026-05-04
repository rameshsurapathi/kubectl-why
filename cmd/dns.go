package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/rameshsurapathi/kubectl-why/pkg/kube"
	"github.com/rameshsurapathi/kubectl-why/pkg/analyzer"
	"github.com/rameshsurapathi/kubectl-why/pkg/render"
)

var dnsCmd = &cobra.Command{
	Use:   "dns",
	Short: "Check cluster DNS health",
	Example: "  kubectl-why dns",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true

		client, err := kube.NewClient(kubeContext)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n  ✗  cannot connect to cluster: %v\n\n", err)
			os.Exit(1)
		}

		ctx := context.Background()
		signals := &kube.DNSSignals{}

		_, err = client.CoreV1().Services("kube-system").Get(ctx, "kube-dns", metav1.GetOptions{})
		if err == nil {
			signals.ServiceFound = true
			ep, err := client.CoreV1().Endpoints("kube-system").Get(ctx, "kube-dns", metav1.GetOptions{})
			if err == nil {
				for _, subset := range ep.Subsets {
					signals.EndpointsReady += len(subset.Addresses)
					signals.TotalEndpoints += len(subset.Addresses) + len(subset.NotReadyAddresses)
				}
			}
		}

		res := analyzer.AnalyzeDNS(signals)
		switch outputFmt {
		case "json":
			return render.JSON(res)
		default:
			return render.Text(res, render.Options{Explain: explainFlag, ShowSecondary: showSecondaryFlag})
		}
	},
}

func init() {
	rootCmd.AddCommand(dnsCmd)
}
