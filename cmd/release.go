package cmd

import (
	"fmt"
	"os"

	"github.com/Kashuab/claimenv/internal/lease"
	"github.com/spf13/cobra"
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Release the current claim",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		lf, err := lease.Load(eng.LeaseFile)
		if err != nil {
			return err
		}

		if err := eng.Release(cmd.Context(), lf); err != nil {
			return err
		}

		if err := lease.Delete(eng.LeaseFile); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Released slot %q from pool %q\n", lf.SlotName, lf.Pool)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(releaseCmd)
}
