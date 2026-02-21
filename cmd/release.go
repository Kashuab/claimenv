package cmd

import (
	"fmt"
	"os"

	"github.com/Kashuab/claimenv/internal/lease"
	"github.com/spf13/cobra"
)

var releaseCmd = &cobra.Command{
	Use:   "release [pool]",
	Short: "Release the current claim",
	Long: `Release the current claim. With no arguments, releases using the local lease file.
With a pool name argument, releases by holder identity (no lease file needed).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			// Release by holder identity — no lease file needed
			poolName := args[0]
			if err := eng.ReleaseByHolder(cmd.Context(), poolName); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Released claim in pool %q (holder: %s)\n", poolName, eng.Identity)
			return nil
		}

		// Release by lease file
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
