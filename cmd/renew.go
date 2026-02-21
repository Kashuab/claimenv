package cmd

import (
	"fmt"
	"os"

	"github.com/Kashuab/claimenv/internal/lease"
	"github.com/spf13/cobra"
)

var renewCmd = &cobra.Command{
	Use:   "renew",
	Short: "Extend the TTL on the current claim",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		lf, err := lease.Load(eng.LeaseFile)
		if err != nil {
			return err
		}

		renewed, err := eng.Renew(cmd.Context(), lf)
		if err != nil {
			return err
		}

		if err := lease.Save(eng.LeaseFile, renewed); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Renewed lease for slot %q in pool %q (new expiry: %s)\n",
			renewed.SlotName, renewed.Pool, renewed.ExpiresAt.Format("2006-01-02 15:04:05"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renewCmd)
}
