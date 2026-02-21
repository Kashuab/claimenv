package cmd

import (
	"fmt"
	"os"

	"github.com/Kashuab/claimenv/internal/lease"
	"github.com/spf13/cobra"
)

var writeCmd = &cobra.Command{
	Use:   "write <KEY> <VALUE>",
	Short: "Write a single env var to the claimed slot",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		lf, err := lease.Load(eng.LeaseFile)
		if err != nil {
			return err
		}

		if err := eng.WriteKey(cmd.Context(), lf, key, value); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Wrote %s to slot %d in pool %q\n", key, lf.SlotIndex, lf.Pool)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(writeCmd)
}
