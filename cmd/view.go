package cmd

import (
	"github.com/rameshsurapathi/kubectl-why/pkg/tui"
	"github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
	Use:     "view",
	GroupID: "primary",
	Short:   "Launch the interactive troubleshooting dashboard",
	Aliases: []string{"ui", "dashboard", "dash", "map", "scan"}, // Kept aliases for backwards compatibility for a bit
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := tui.Config{
			Namespace:   namespace,
			KubeContext: kubeContext,
			TailLines:   tailLines,
			MaxEvents:   maxEvents,
		}
		return tui.Start(cfg)
	},
}

func init() {
	rootCmd.AddCommand(viewCmd)
}
