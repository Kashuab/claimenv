package cmd

import (
	"fmt"

	"github.com/Kashuab/claimenv/internal/lease"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read <KEY>",
	Short: "Read a single env var from the claimed slot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		lf, err := lease.Load(eng.LeaseFile)
		if err != nil {
			return err
		}

		val, err := eng.ReadKey(cmd.Context(), lf, key)
		if err != nil {
			return err
		}

		fmt.Print(val)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(readCmd)
}
