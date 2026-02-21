package cmd

import (
	"fmt"
	"os"

	"github.com/Kashuab/claimenv/internal/lease"
	"github.com/spf13/cobra"
)

var claimCmd = &cobra.Command{
	Use:   "claim <pool>",
	Short: "Claim an available slot from a pool",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		poolName := args[0]

		lf, err := eng.Claim(cmd.Context(), poolName)
		if err != nil {
			return err
		}

		if err := lease.Save(eng.LeaseFile, lf); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Claimed slot %d from pool %q (lease: %s, expires: %s)\n",
			lf.SlotIndex, lf.Pool, lf.LeaseID, lf.ExpiresAt.Format("2006-01-02 15:04:05"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(claimCmd)
}
